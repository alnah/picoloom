package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	md2pdf "github.com/alnah/go-md2pdf"
	"github.com/alnah/go-md2pdf/internal/config"
	"github.com/alnah/go-md2pdf/internal/yamlutil"
	flag "github.com/spf13/pflag"
)

const (
	// defaultConfigInitOutputPath keeps the common local config convention,
	// so generated examples and file discovery stay aligned.
	defaultConfigInitOutputPath = "./md2pdf.yaml"
	// configInitBackupSuffix marks temporary backup files used for safe overwrite recovery.
	configInitBackupSuffix = ".md2pdf-config-init.bak"
	// configInitLockSuffix marks destination-scoped lock files to prevent concurrent writes.
	configInitLockSuffix = ".md2pdf-config-init.lock"
)

var (
	// ErrConfigCommandUsage groups user-facing command-shape errors under a stable sentinel.
	ErrConfigCommandUsage = errors.New("invalid config command usage")
	// ErrConfigInitNeedsTTY guards interactive prompts from blocking in non-interactive environments.
	ErrConfigInitNeedsTTY = errors.New("interactive mode requires a TTY")
	// ErrConfigInitExists preserves existing files unless overwrite is explicit.
	ErrConfigInitExists = errors.New("config file already exists")
	// ErrConfigInitBusy rejects parallel writers targeting the same destination.
	ErrConfigInitBusy = errors.New("config init already in progress for destination")
)

// configInitFlags keeps CLI intent explicit before any file operation begins.
type configInitFlags struct {
	output  string
	force   bool
	noInput bool
}

// configInitAnswers captures wizard decisions before materializing final config.
type configInitAnswers struct {
	style              string
	authorName         string
	authorTitle        string
	authorEmail        string
	authorOrganization string
	pageSize           string
	signatureEnabled   bool
	signatureImagePath string
	watermarkEnabled   bool
	watermarkText      string
	watermarkColor     string
	coverEnabled       bool
	coverLogo          string
}

// wizardPrompt defines one UX question so validation/help stay consistent.
type wizardPrompt struct {
	title        string
	options      string
	example      string
	defaultValue string
	helpYAML     string
	validate     func(string) error
}

// wizardStyle represents a named embedded style and why a user might choose it.
type wizardStyle struct {
	name        string
	description string
}

// wizardStyles lists supported built-ins to keep prompts self-contained.
var wizardStyles = []wizardStyle{
	{name: "default", description: "minimal neutral baseline"},
	{name: "technical", description: "clean system-ui with code-friendly defaults"},
	{name: "creative", description: "more colorful headings and accents"},
	{name: "academic", description: "serif typography with high readability"},
	{name: "corporate", description: "professional business-like sans-serif"},
	{name: "legal", description: "conservative legal-style formatting"},
	{name: "invoice", description: "table-oriented business layout"},
	{name: "manuscript", description: "monospace narrative style"},
}

var (
	// Precomputed style helpers keep repeated prompt validation fast and deterministic.
	wizardStyleNames     = buildWizardStyleNames()
	wizardStyleNamesText = strings.Join(wizardStyleNames, ", ")
	wizardStyleNameSet   = buildWizardStyleNameSet(wizardStyleNames)
)

// runConfigCmd keeps "config" as a namespace so future subcommands can be added
// without breaking CLI shape.
func runConfigCmd(args []string, env *Environment) error {
	if len(args) == 0 {
		printConfigUsage(env.Stdout)
		return nil
	}

	switch args[0] {
	case "init":
		return runConfigInitCmd(args[1:], env)
	case "help", "-h", "--help":
		printConfigUsage(env.Stdout)
		return nil
	default:
		return fmt.Errorf("%w: unknown subcommand %q (run 'md2pdf help config')", ErrConfigCommandUsage, args[0])
	}
}

// parseConfigInitFlags centralizes config-init argument policy so help and
// validation behavior are identical across execution paths.
func parseConfigInitFlags(args []string, stderr io.Writer) (configInitFlags, error) {
	f := configInitFlags{
		output: defaultConfigInitOutputPath,
	}

	fs := flag.NewFlagSet("config init", flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.StringVar(&f.output, "output", defaultConfigInitOutputPath, "output path for generated config")
	fs.BoolVar(&f.force, "force", false, "overwrite destination if it already exists")
	fs.BoolVar(&f.noInput, "no-input", false, "use defaults without interactive prompts")
	fs.Usage = func() {
		printConfigInitUsage(stderr)
	}

	if err := fs.Parse(args); err != nil {
		return configInitFlags{}, err
	}
	if fs.NArg() > 0 {
		return configInitFlags{}, fmt.Errorf("%w: unexpected arguments: %s", ErrConfigCommandUsage, strings.Join(fs.Args(), " "))
	}
	if strings.TrimSpace(f.output) == "" {
		return configInitFlags{}, fmt.Errorf("%w: --output cannot be empty", ErrConfigCommandUsage)
	}

	return f, nil
}

// runConfigInitCmd enforces interaction policy, builds a valid config, and
// persists it through safety-checked publishing.
func runConfigInitCmd(args []string, env *Environment) error {
	flags, err := parseConfigInitFlags(args, env.Stderr)
	if err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}

	if !flags.noInput && !stdinIsTTY(env) {
		return fmt.Errorf("%w: use --no-input for scripts or CI", ErrConfigInitNeedsTTY)
	}

	cfg, shouldWrite, err := buildConfigInitConfig(flags.noInput, env)
	if err != nil {
		return err
	}
	if !shouldWrite {
		fmt.Fprintln(env.Stdout, "Configuration generation canceled.")
		return nil
	}

	data, err := yamlutil.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("encoding generated config: %w", err)
	}
	data = formatConfigInitYAML(data)

	if err := writeConfigInitFile(flags.output, data, flags.force); err != nil {
		return err
	}

	fmt.Fprintf(env.Stdout, "Configuration file created: %s\n", flags.output)
	fmt.Fprintln(env.Stdout, "Example:")
	fmt.Fprintf(env.Stdout, "  md2pdf convert -c %s ./docs/\n", outputPathForExample(flags.output))
	return nil
}

// buildConfigInitConfig starts from conservative defaults so non-interactive
// generation is immediately usable and interactive mode can be safely canceled.
func buildConfigInitConfig(noInput bool, env *Environment) (*config.Config, bool, error) {
	answers := configInitAnswers{
		style:              "technical",
		authorName:         "Your Name",
		authorTitle:        "",
		authorEmail:        "",
		authorOrganization: "",
		pageSize:           "letter",
		signatureEnabled:   false,
		signatureImagePath: "",
		watermarkEnabled:   false,
		watermarkText:      "DRAFT",
		watermarkColor:     md2pdf.DefaultWatermarkColor,
		coverEnabled:       false,
		coverLogo:          "",
	}

	var reader *bufio.Reader
	if !noInput {
		reader = bufio.NewReader(stdinReader(env))
		printWizardStyleChoices(env.Stdout)

		style, err := promptString(reader, env.Stdout, wizardPrompt{
			title:        "Style",
			options:      wizardStyleOptions(),
			example:      "technical",
			defaultValue: answers.style,
			helpYAML:     "style: technical\n# Styles: default, technical, creative, academic,\n#         corporate, legal, invoice, manuscript",
			validate:     validateWizardStyle,
		})
		if err != nil {
			return nil, false, err
		}
		authorName, err := promptString(reader, env.Stdout, wizardPrompt{
			title:        "Author name",
			options:      "free text",
			example:      "Alex Martin",
			defaultValue: answers.authorName,
			helpYAML:     "author:\n  name: Alex Martin",
		})
		if err != nil {
			return nil, false, err
		}
		authorTitle, err := promptString(reader, env.Stdout, wizardPrompt{
			title:        "Author title",
			options:      "free text or empty",
			example:      "Staff Engineer",
			defaultValue: answers.authorTitle,
			helpYAML:     "author:\n  title: Staff Engineer",
		})
		if err != nil {
			return nil, false, err
		}
		authorEmail, err := promptString(reader, env.Stdout, wizardPrompt{
			title:        "Author email",
			options:      "email or empty",
			example:      "alex@example.com",
			defaultValue: answers.authorEmail,
			helpYAML:     "author:\n  email: alex@example.com",
		})
		if err != nil {
			return nil, false, err
		}
		authorOrganization, err := promptString(reader, env.Stdout, wizardPrompt{
			title:        "Author organization",
			options:      "free text or empty",
			example:      "Acme Corp",
			defaultValue: answers.authorOrganization,
			helpYAML:     "author:\n  organization: Acme Corp",
		})
		if err != nil {
			return nil, false, err
		}
		pageSize, err := promptString(reader, env.Stdout, wizardPrompt{
			title:        "Page size",
			options:      "letter, a4, legal",
			example:      "a4",
			defaultValue: answers.pageSize,
			helpYAML:     "page:\n  size: letter",
			validate:     validatePageSize,
		})
		if err != nil {
			return nil, false, err
		}
		signatureEnabled, err := promptBool(reader, env.Stdout, wizardPrompt{
			title:        "Enable signature block",
			options:      "yes, no",
			example:      "yes",
			defaultValue: boolDefaultLabel(answers.signatureEnabled),
			helpYAML:     "signature:\n  enabled: true\n  imagePath: ./assets/signature.png",
		})
		if err != nil {
			return nil, false, err
		}
		signatureImagePath := answers.signatureImagePath
		if signatureEnabled {
			signatureImagePath, err = promptString(reader, env.Stdout, wizardPrompt{
				title:        "Signature image path",
				options:      "file path or URL",
				example:      "./assets/signature.png",
				defaultValue: signatureImagePath,
				helpYAML:     "signature:\n  imagePath: ./assets/signature.png",
			})
			if err != nil {
				return nil, false, err
			}
		}
		watermarkEnabled, err := promptBool(reader, env.Stdout, wizardPrompt{
			title:        "Enable watermark",
			options:      "yes, no",
			example:      "yes",
			defaultValue: boolDefaultLabel(answers.watermarkEnabled),
			helpYAML:     "watermark:\n  enabled: true\n  text: DRAFT\n  color: #888888\n  opacity: 0.1\n  angle: -45",
		})
		if err != nil {
			return nil, false, err
		}
		watermarkText := answers.watermarkText
		watermarkColor := answers.watermarkColor
		if watermarkEnabled {
			watermarkText, err = promptString(reader, env.Stdout, wizardPrompt{
				title:        "Watermark text",
				options:      "free text (required when enabled)",
				example:      "CONFIDENTIAL",
				defaultValue: watermarkText,
				helpYAML:     "watermark:\n  text: CONFIDENTIAL",
			})
			if err != nil {
				return nil, false, err
			}
			watermarkColor, err = promptString(reader, env.Stdout, wizardPrompt{
				title:        "Watermark color",
				options:      "hex color (#RGB or #RRGGBB)",
				example:      "#888888",
				defaultValue: watermarkColor,
				helpYAML:     "watermark:\n  color: #888888",
				validate:     validateWatermarkColor,
			})
			if err != nil {
				return nil, false, err
			}
		}
		coverEnabled, err := promptBool(reader, env.Stdout, wizardPrompt{
			title:        "Enable cover page",
			options:      "yes, no",
			example:      "yes",
			defaultValue: boolDefaultLabel(answers.coverEnabled),
			helpYAML:     "cover:\n  enabled: true\n  logo: ./assets/logo.png",
		})
		if err != nil {
			return nil, false, err
		}
		coverLogo := answers.coverLogo
		if coverEnabled {
			coverLogo, err = promptString(reader, env.Stdout, wizardPrompt{
				title:        "Cover logo path",
				options:      "file path or URL",
				example:      "./assets/logo.png",
				defaultValue: coverLogo,
				helpYAML:     "cover:\n  logo: ./assets/logo.png",
			})
			if err != nil {
				return nil, false, err
			}
		}

		answers.style = strings.ToLower(style)
		answers.authorName = authorName
		answers.authorTitle = authorTitle
		answers.authorEmail = authorEmail
		answers.authorOrganization = authorOrganization
		answers.pageSize = strings.ToLower(pageSize)
		answers.signatureEnabled = signatureEnabled
		answers.signatureImagePath = signatureImagePath
		answers.watermarkEnabled = watermarkEnabled
		answers.watermarkText = watermarkText
		answers.watermarkColor = watermarkColor
		answers.coverEnabled = coverEnabled
		answers.coverLogo = coverLogo
	}

	cfg := config.DefaultConfig()
	cfg.Style = answers.style
	cfg.Author.Name = answers.authorName
	cfg.Author.Title = answers.authorTitle
	cfg.Author.Email = answers.authorEmail
	cfg.Author.Organization = answers.authorOrganization
	cfg.Page.Size = answers.pageSize
	cfg.Page.Orientation = "portrait"
	cfg.Signature.Enabled = answers.signatureEnabled
	cfg.Signature.ImagePath = answers.signatureImagePath
	cfg.Watermark.Enabled = answers.watermarkEnabled
	cfg.Watermark.Text = answers.watermarkText
	cfg.Watermark.Color = answers.watermarkColor
	cfg.Watermark.Opacity = md2pdf.DefaultWatermarkOpacity
	cfg.Watermark.Angle = md2pdf.DefaultWatermarkAngle
	cfg.Cover.Enabled = answers.coverEnabled
	cfg.Cover.Logo = answers.coverLogo

	if err := cfg.Validate(); err != nil {
		return nil, false, fmt.Errorf("validating generated config: %w", err)
	}

	if !noInput {
		ok, err := confirmConfigInitWrite(reader, env.Stdout, cfg)
		if err != nil {
			return nil, false, err
		}
		if !ok {
			return cfg, false, nil
		}
	}

	return cfg, true, nil
}

// promptString provides a uniform question loop so defaults, inline help, and
// validation failures behave predictably for every wizard field.
func promptString(reader *bufio.Reader, output io.Writer, prompt wizardPrompt) (string, error) {
	for {
		if prompt.options != "" {
			fmt.Fprintf(output, "Options: %s\n", prompt.options)
		}
		fmt.Fprintf(output, "%s [default: %s] (example: %s, type ? for help): ", prompt.title, formatPromptDefault(prompt.defaultValue), prompt.example)

		line, err := reader.ReadString('\n')
		isEOF := errors.Is(err, io.EOF)
		if err != nil && !isEOF {
			return "", fmt.Errorf("reading %s answer: %w", strings.ToLower(prompt.title), err)
		}
		value := strings.TrimSpace(line)
		if value == "?" {
			printPromptHelp(output, prompt)
			if isEOF {
				return "", fmt.Errorf("invalid %s value: expected answer after help", strings.ToLower(prompt.title))
			}
			continue
		}
		if value == "" {
			value = prompt.defaultValue
		}
		if prompt.validate != nil {
			if err := prompt.validate(value); err != nil {
				if isEOF {
					return "", fmt.Errorf("invalid %s value: %w", strings.ToLower(prompt.title), err)
				}
				fmt.Fprintf(output, "Invalid value: %v\n", err)
				continue
			}
		}
		return value, nil
	}
}

// promptBool reuses promptString so yes/no questions inherit the same UX and
// retry semantics as text prompts.
func promptBool(reader *bufio.Reader, output io.Writer, prompt wizardPrompt) (bool, error) {
	for {
		value, err := promptString(reader, output, prompt)
		if err != nil {
			return false, err
		}
		parsed, err := parseYesNo(value)
		if err != nil {
			fmt.Fprintf(output, "Invalid value: %v\n", err)
			continue
		}
		return parsed, nil
	}
}

// confirmConfigInitWrite adds an explicit final acknowledgment to reduce
// accidental writes after interactive data entry.
func confirmConfigInitWrite(reader *bufio.Reader, output io.Writer, cfg *config.Config) (bool, error) {
	data, err := yamlutil.Marshal(cfg)
	if err != nil {
		return false, fmt.Errorf("encoding preview config: %w", err)
	}
	data = formatConfigInitYAML(data)

	fmt.Fprintln(output)
	fmt.Fprintln(output, "Configuration summary:")
	fmt.Fprintf(output, "- style: %s\n", cfg.Style)
	fmt.Fprintf(output, "- author: %s\n", cfg.Author.Name)
	fmt.Fprintf(output, "- page.size: %s\n", cfg.Page.Size)
	fmt.Fprintf(output, "- signature.enabled: %t\n", cfg.Signature.Enabled)
	fmt.Fprintf(output, "- watermark.enabled: %t\n", cfg.Watermark.Enabled)
	fmt.Fprintf(output, "- cover.enabled: %t\n", cfg.Cover.Enabled)
	fmt.Fprintln(output)
	fmt.Fprintln(output, "YAML preview:")
	fmt.Fprintln(output, string(data))

	return promptBool(reader, output, wizardPrompt{
		title:        "Write configuration file",
		options:      "yes, no",
		example:      "yes",
		defaultValue: "yes",
		helpYAML:     "# Type yes to write the file",
	})
}

// printPromptHelp keeps field-level guidance in the wizard so users do not have
// to leave the terminal to find valid examples.
func printPromptHelp(output io.Writer, prompt wizardPrompt) {
	fmt.Fprintln(output, "Help:")
	if prompt.options != "" {
		fmt.Fprintf(output, "  Options: %s\n", prompt.options)
	}
	if prompt.example != "" {
		fmt.Fprintf(output, "  Example value: %s\n", prompt.example)
	}
	if prompt.helpYAML != "" {
		fmt.Fprintln(output, "  YAML example:")
		for _, line := range strings.Split(prompt.helpYAML, "\n") {
			fmt.Fprintf(output, "    %s\n", line)
		}
	}
}

// formatPromptDefault makes empty defaults explicit in prompts to avoid
// ambiguity between "blank" and "missing".
func formatPromptDefault(value string) string {
	if strings.TrimSpace(value) == "" {
		return "<empty>"
	}
	return value
}

// boolDefaultLabel keeps boolean defaults readable in a human prompt context.
func boolDefaultLabel(value bool) string {
	if value {
		return "yes"
	}
	return "no"
}

// formatConfigInitYAML inserts spacing between top-level sections so generated
// files are easier to review and edit manually.
func formatConfigInitYAML(data []byte) []byte {
	if len(data) == 0 {
		return data
	}

	content := string(data)
	hasTrailingNewline := strings.HasSuffix(content, "\n")
	lines := strings.Split(strings.TrimRight(content, "\n"), "\n")
	if len(lines) == 0 {
		return data
	}

	var builder strings.Builder
	topLevelKeyCount := 0
	for i, line := range lines {
		if isTopLevelYAMLKeyLine(line) {
			if topLevelKeyCount > 0 {
				builder.WriteByte('\n')
			}
			topLevelKeyCount++
		}

		builder.WriteString(line)
		if i < len(lines)-1 {
			builder.WriteByte('\n')
		}
	}
	if hasTrailingNewline {
		builder.WriteByte('\n')
	}

	return []byte(builder.String())
}

// isTopLevelYAMLKeyLine scopes spacing rules to top-level keys only, preserving
// nested YAML structure untouched.
func isTopLevelYAMLKeyLine(line string) bool {
	if strings.TrimSpace(line) == "" {
		return false
	}
	if strings.HasPrefix(line, " ") || strings.HasPrefix(line, "\t") {
		return false
	}
	trimmed := strings.TrimSpace(line)
	if strings.HasPrefix(trimmed, "#") {
		return false
	}
	return strings.Contains(line, ":")
}

// printWizardStyleChoices exposes style intent up front to reduce trial-and-error
// during initial configuration.
func printWizardStyleChoices(output io.Writer) {
	fmt.Fprintln(output, "Available styles:")
	for _, style := range wizardStyles {
		fmt.Fprintf(output, "  - %s: %s\n", style.name, style.description)
	}
}

// wizardStyleOptions returns a stable, human-readable style list for prompts.
func wizardStyleOptions() string {
	return wizardStyleNamesText
}

// validateWizardStyle keeps generated config valid against known built-in styles.
func validateWizardStyle(value string) error {
	normalized := strings.ToLower(strings.TrimSpace(value))
	if _, ok := wizardStyleNameSet[normalized]; !ok {
		return fmt.Errorf("must be one of: %s", wizardStyleNamesText)
	}
	return nil
}

// buildWizardStyleNames precomputes ordered style names once to keep prompt
// rendering deterministic.
func buildWizardStyleNames() []string {
	names := make([]string, 0, len(wizardStyles))
	for _, style := range wizardStyles {
		names = append(names, style.name)
	}
	return names
}

// buildWizardStyleNameSet enables O(1) membership checks in prompt validation.
func buildWizardStyleNameSet(names []string) map[string]struct{} {
	set := make(map[string]struct{}, len(names))
	for _, name := range names {
		set[name] = struct{}{}
	}
	return set
}

// parseYesNo accepts common boolean aliases to make CLI interaction tolerant
// without sacrificing explicit intent.
func parseYesNo(value string) (bool, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "y", "yes", "true", "1":
		return true, nil
	case "n", "no", "false", "0":
		return false, nil
	default:
		return false, fmt.Errorf("expected yes/no")
	}
}

// validatePageSize constrains output layout to supported rendering sizes.
func validatePageSize(value string) error {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "letter", "a4", "legal":
		return nil
	default:
		return fmt.Errorf("must be one of: letter, a4, legal")
	}
}

// validateWatermarkColor delegates to core watermark validation so CLI and
// library paths share the same acceptance rules.
func validateWatermarkColor(value string) error {
	test := &md2pdf.Watermark{
		Color:   value,
		Opacity: md2pdf.DefaultWatermarkOpacity,
		Angle:   md2pdf.DefaultWatermarkAngle,
	}
	if err := test.Validate(); err != nil {
		return err
	}
	return nil
}

// configInitFileOps abstracts filesystem side effects so safety behavior can be
// tested deterministically across platforms and failure modes.
type configInitFileOps struct {
	stat     func(string) (os.FileInfo, error)
	mkdirAll func(string, os.FileMode) error
	create   func(string, string) (*os.File, error)
	rename   func(string, string) error
	remove   func(string) error
	link     func(string, string) error
	openFile func(string, int, os.FileMode) (*os.File, error)
	readFile func(string) ([]byte, error)
}

// defaultConfigInitFileOps binds file operations to the real OS implementation.
func defaultConfigInitFileOps() configInitFileOps {
	return configInitFileOps{
		stat:     os.Stat,
		mkdirAll: os.MkdirAll,
		create:   os.CreateTemp,
		rename:   os.Rename,
		remove:   os.Remove,
		link:     os.Link,
		openFile: os.OpenFile,
		readFile: os.ReadFile,
	}
}

// writeConfigInitFile is the production entry point for safe config publishing.
func writeConfigInitFile(outputPath string, data []byte, force bool) error {
	return writeConfigInitFileWithOps(outputPath, data, force, defaultConfigInitFileOps())
}

// writeConfigInitFileWithOps enforces atomic-ish write invariants (lock, temp
// file, validation, publish strategy) to prevent partial or conflicting writes.
func writeConfigInitFileWithOps(outputPath string, data []byte, force bool, ops configInitFileOps) (retErr error) {
	if strings.TrimSpace(outputPath) == "" {
		return fmt.Errorf("%w: output path cannot be empty", ErrConfigCommandUsage)
	}

	dir := filepath.Dir(outputPath)
	if err := ops.mkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("creating destination directory: %w", err)
	}
	lockPath := configInitLockPath(outputPath)
	lockFile, err := ops.openFile(lockPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		if os.IsExist(err) {
			return fmt.Errorf("%w: %s (remove stale lock if needed: %s)", ErrConfigInitBusy, outputPath, lockPath)
		}
		return fmt.Errorf("acquiring destination lock: %w", err)
	}
	if err := lockFile.Close(); err != nil {
		_ = ops.remove(lockPath)
		return fmt.Errorf("closing destination lock: %w", err)
	}
	defer func() {
		if err := ops.remove(lockPath); err != nil && !os.IsNotExist(err) && retErr == nil {
			retErr = fmt.Errorf("releasing destination lock: %w", err)
		}
	}()

	if err := recoverConfigInitBackup(outputPath, ops); err != nil {
		return err
	}

	tmpFile, err := ops.create(dir, ".md2pdf-config-init-*.yaml")
	if err != nil {
		return fmt.Errorf("creating temp config file: %w", err)
	}

	tmpPath := tmpFile.Name()
	defer func() {
		_ = tmpFile.Close()
		if retErr != nil {
			_ = ops.remove(tmpPath)
		}
	}()

	if _, err := tmpFile.Write(data); err != nil {
		return fmt.Errorf("writing temp config file: %w", err)
	}
	if err := tmpFile.Sync(); err != nil {
		return fmt.Errorf("syncing temp config file: %w", err)
	}
	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("closing temp config file: %w", err)
	}

	if _, err := config.LoadConfig(tmpPath); err != nil {
		return fmt.Errorf("validating generated config: %w", err)
	}

	if force {
		return publishConfigForce(tmpPath, outputPath, ops)
	}

	return publishConfigNoForce(tmpPath, outputPath, ops)
}

// configInitBackupPath keeps backup naming deterministic for recovery logic.
func configInitBackupPath(outputPath string) string {
	return outputPath + configInitBackupSuffix
}

// configInitLockPath scopes lock files to destination path to serialize writers.
func configInitLockPath(outputPath string) string {
	return outputPath + configInitLockSuffix
}

// recoverConfigInitBackup repairs interrupted overwrite states before any new
// write attempt, so destination semantics remain predictable.
func recoverConfigInitBackup(outputPath string, ops configInitFileOps) error {
	_, outputErr := ops.stat(outputPath)
	if outputErr != nil && !os.IsNotExist(outputErr) {
		return fmt.Errorf("checking destination config file: %w", outputErr)
	}

	backupPath := configInitBackupPath(outputPath)
	_, backupErr := ops.stat(backupPath)
	if backupErr != nil && !os.IsNotExist(backupErr) {
		return fmt.Errorf("checking backup config file: %w", backupErr)
	}
	if os.IsNotExist(backupErr) {
		return nil
	}

	if os.IsNotExist(outputErr) {
		if err := ops.rename(backupPath, outputPath); err != nil {
			return fmt.Errorf("restoring interrupted overwrite: %w", err)
		}
		return nil
	}

	if err := ops.remove(backupPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("cleaning stale backup file: %w", err)
	}
	return nil
}

// publishConfigNoForce guarantees no-clobber semantics unless user explicitly
// opted into overwrite via --force.
func publishConfigNoForce(tmpPath, outputPath string, ops configInitFileOps) error {
	if err := ops.link(tmpPath, outputPath); err == nil {
		if err := ops.remove(tmpPath); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("removing temporary config after publish: %w", err)
		}
		return nil
	} else if os.IsExist(err) {
		return fmt.Errorf("%w: %s (use --force)", ErrConfigInitExists, outputPath)
	}

	return copyTempToExclusiveFile(tmpPath, outputPath, ops)
}

// copyTempToExclusiveFile is the portability fallback when hard-link publish is
// unavailable, while preserving exclusive-create guarantees.
func copyTempToExclusiveFile(tmpPath, outputPath string, ops configInitFileOps) (retErr error) {
	out, err := ops.openFile(outputPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		if os.IsExist(err) {
			return fmt.Errorf("%w: %s (use --force)", ErrConfigInitExists, outputPath)
		}
		return fmt.Errorf("creating destination config file: %w", err)
	}

	defer func() {
		_ = out.Close()
		if retErr != nil {
			_ = ops.remove(outputPath)
		}
	}()

	const smallPayloadThreshold = 64 * 1024
	if info, statErr := ops.stat(tmpPath); statErr == nil && info.Size() <= smallPayloadThreshold {
		content, readErr := ops.readFile(tmpPath)
		if readErr != nil {
			return fmt.Errorf("reading temp config file for exclusive publish: %w", readErr)
		}
		if _, writeErr := out.Write(content); writeErr != nil {
			return fmt.Errorf("writing destination config file: %w", writeErr)
		}
	} else {
		in, openErr := ops.openFile(tmpPath, os.O_RDONLY, 0)
		if openErr != nil {
			return fmt.Errorf("opening temp config file for exclusive publish: %w", openErr)
		}
		defer func() {
			_ = in.Close()
		}()
		if _, copyErr := io.Copy(out, in); copyErr != nil {
			return fmt.Errorf("writing destination config file: %w", copyErr)
		}
	}
	if err := out.Sync(); err != nil {
		return fmt.Errorf("syncing destination config file: %w", err)
	}
	if err := out.Close(); err != nil {
		return fmt.Errorf("closing destination config file: %w", err)
	}

	if err := ops.remove(tmpPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("removing temporary config after publish: %w", err)
	}

	return nil
}

// publishConfigForce implements explicit overwrite with rollback protection so
// failures do not destroy the previous config.
func publishConfigForce(tmpPath, outputPath string, ops configInitFileOps) error {
	if _, err := ops.stat(outputPath); err != nil {
		if !os.IsNotExist(err) {
			return fmt.Errorf("checking destination config file: %w", err)
		}
		if err := ops.rename(tmpPath, outputPath); err != nil {
			return fmt.Errorf("moving generated config into place: %w", err)
		}
		return nil
	}

	backupPath := configInitBackupPath(outputPath)
	if err := ops.remove(backupPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("cleaning stale backup file: %w", err)
	}
	if err := ops.rename(outputPath, backupPath); err != nil {
		return fmt.Errorf("preparing safe overwrite: %w", err)
	}

	if err := ops.rename(tmpPath, outputPath); err != nil {
		restoreErr := ops.rename(backupPath, outputPath)
		if restoreErr != nil {
			return fmt.Errorf("overwriting config failed: %w; rollback failed: %v", err, restoreErr)
		}
		return fmt.Errorf("overwriting config failed, restored previous file: %w", err)
	}

	if err := ops.remove(backupPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("cleaning backup file: %w", err)
	}

	return nil
}

// stdinReader keeps input source injectable to support tests and scripted flows.
func stdinReader(env *Environment) io.Reader {
	if env.Stdin != nil {
		return env.Stdin
	}
	return os.Stdin
}

// stdinIsTTY centralizes TTY detection so interaction policy is testable.
func stdinIsTTY(env *Environment) bool {
	if env.IsStdinTTY != nil {
		return env.IsStdinTTY()
	}
	return isTerminal(os.Stdin)
}

// outputPathForExample normalizes default path rendering so success output stays
// stable and copy-paste friendly.
func outputPathForExample(path string) string {
	if path == defaultConfigInitOutputPath || path == "md2pdf.yaml" {
		return defaultConfigInitOutputPath
	}
	return path
}

// printConfigUsage documents the config command namespace entry point.
func printConfigUsage(w io.Writer) {
	fmt.Fprintln(w, "Usage: md2pdf config <subcommand> [flags]")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Manage md2pdf configuration files.")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Subcommands:")
	fmt.Fprintln(w, "  init        Create a config file with an interactive wizard")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Run 'md2pdf help config init' for command details.")
}

// printConfigInitUsage documents config-init flags and canonical usage examples.
func printConfigInitUsage(w io.Writer) {
	fmt.Fprintln(w, "Usage: md2pdf config init [flags]")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Create a new md2pdf configuration file.")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Flags:")
	fmt.Fprintln(w, "      --output <path>   Output path for generated config (default: ./md2pdf.yaml)")
	fmt.Fprintln(w, "      --force           Overwrite destination if it exists")
	fmt.Fprintln(w, "      --no-input        Use defaults without interactive prompts")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Examples:")
	fmt.Fprintln(w, "  md2pdf config init")
	fmt.Fprintln(w, "  md2pdf config init --output ./configs/work.yaml")
	fmt.Fprintln(w, "  md2pdf config init --no-input --force")
}
