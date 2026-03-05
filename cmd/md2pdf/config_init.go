package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"

	md2pdf "github.com/alnah/go-md2pdf"
	"github.com/alnah/go-md2pdf/internal/config"
	"github.com/alnah/go-md2pdf/internal/yamlutil"
	flag "github.com/spf13/pflag"
)

const (
	defaultConfigInitOutputPath = "./md2pdf.yaml"
	configInitBackupSuffix      = ".md2pdf-config-init.bak"
	configInitLockSuffix        = ".md2pdf-config-init.lock"
)

var (
	ErrConfigCommandUsage = errors.New("invalid config command usage")
	ErrConfigInitNeedsTTY = errors.New("interactive mode requires a TTY")
	ErrConfigInitExists   = errors.New("config file already exists")
	ErrConfigInitBusy     = errors.New("config init already in progress for destination")
)

type configInitFlags struct {
	output  string
	force   bool
	noInput bool
}

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

type wizardPrompt struct {
	title        string
	options      string
	example      string
	defaultValue string
	helpYAML     string
	validate     func(string) error
}

type wizardStyle struct {
	name        string
	description string
}

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

	if err := writeConfigInitFile(flags.output, data, flags.force); err != nil {
		return err
	}

	fmt.Fprintf(env.Stdout, "Configuration file created: %s\n", flags.output)
	fmt.Fprintln(env.Stdout, "Example:")
	fmt.Fprintf(env.Stdout, "  md2pdf convert -c %s ./docs/\n", outputPathForExample(flags.output))
	return nil
}

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

func confirmConfigInitWrite(reader *bufio.Reader, output io.Writer, cfg *config.Config) (bool, error) {
	data, err := yamlutil.Marshal(cfg)
	if err != nil {
		return false, fmt.Errorf("encoding preview config: %w", err)
	}

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

func formatPromptDefault(value string) string {
	if strings.TrimSpace(value) == "" {
		return "<empty>"
	}
	return value
}

func boolDefaultLabel(value bool) string {
	if value {
		return "yes"
	}
	return "no"
}

func printWizardStyleChoices(output io.Writer) {
	fmt.Fprintln(output, "Available styles:")
	for _, style := range wizardStyles {
		fmt.Fprintf(output, "  - %s: %s\n", style.name, style.description)
	}
}

func wizardStyleOptions() string {
	parts := make([]string, 0, len(wizardStyles))
	for _, style := range wizardStyles {
		parts = append(parts, style.name)
	}
	return strings.Join(parts, ", ")
}

func validateWizardStyle(value string) error {
	normalized := strings.ToLower(strings.TrimSpace(value))
	names := make([]string, 0, len(wizardStyles))
	for _, style := range wizardStyles {
		names = append(names, style.name)
	}
	if !slices.Contains(names, normalized) {
		return fmt.Errorf("must be one of: %s", strings.Join(names, ", "))
	}
	return nil
}

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

func validatePageSize(value string) error {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "letter", "a4", "legal":
		return nil
	default:
		return fmt.Errorf("must be one of: letter, a4, legal")
	}
}

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

func writeConfigInitFile(outputPath string, data []byte, force bool) error {
	return writeConfigInitFileWithOps(outputPath, data, force, defaultConfigInitFileOps())
}

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

func configInitBackupPath(outputPath string) string {
	return outputPath + configInitBackupSuffix
}

func configInitLockPath(outputPath string) string {
	return outputPath + configInitLockSuffix
}

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

func stdinReader(env *Environment) io.Reader {
	if env.Stdin != nil {
		return env.Stdin
	}
	return os.Stdin
}

func stdinIsTTY(env *Environment) bool {
	if env.IsStdinTTY != nil {
		return env.IsStdinTTY()
	}
	return isTerminal(os.Stdin)
}

func outputPathForExample(path string) string {
	if path == defaultConfigInitOutputPath || path == "md2pdf.yaml" {
		return defaultConfigInitOutputPath
	}
	return path
}

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
