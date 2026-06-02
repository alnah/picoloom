package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"strings"

	picoloom "github.com/alnah/picoloom/v2"
	"github.com/alnah/picoloom/v2/internal/config"
	"github.com/alnah/picoloom/v2/internal/yamlutil"
)

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

// defaultConfigInitAnswers defines a single baseline profile so interactive and
// non-interactive generation start from the same conservative defaults.
func defaultConfigInitAnswers() configInitAnswers {
	return configInitAnswers{
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
		watermarkColor:     picoloom.DefaultWatermarkColor,
		coverEnabled:       false,
		coverLogo:          "",
	}
}

// collectConfigInitInteractiveAnswers orchestrates prompt groups so the wizard
// flow stays readable while preserving one deterministic question order.
func collectConfigInitInteractiveAnswers(reader *bufio.Reader, output io.Writer, answers configInitAnswers) (configInitAnswers, error) {
	printWizardStyleChoices(output)

	var err error
	answers, err = promptConfigInitBaseAnswers(reader, output, answers)
	if err != nil {
		return configInitAnswers{}, err
	}
	answers, err = promptConfigInitSignatureAnswers(reader, output, answers)
	if err != nil {
		return configInitAnswers{}, err
	}
	answers, err = promptConfigInitWatermarkAnswers(reader, output, answers)
	if err != nil {
		return configInitAnswers{}, err
	}
	answers, err = promptConfigInitCoverAnswers(reader, output, answers)
	if err != nil {
		return configInitAnswers{}, err
	}
	return answers, nil
}

// promptConfigInitBaseAnswers captures core identity/page fields first so users
// can establish document context before optional feature prompts.
func promptConfigInitBaseAnswers(reader *bufio.Reader, output io.Writer, answers configInitAnswers) (configInitAnswers, error) {
	style, err := promptString(reader, output, wizardPrompt{
		title:        "Style",
		options:      wizardStyleOptions(),
		example:      "technical",
		defaultValue: answers.style,
		helpYAML:     "style: technical\n# Styles: default, technical, creative, academic,\n#         corporate, legal, invoice, manuscript",
		validate:     validateWizardStyle,
	})
	if err != nil {
		return configInitAnswers{}, err
	}
	authorName, err := promptString(reader, output, wizardPrompt{
		title:        "Author name",
		options:      "free text",
		example:      "Alex Martin",
		defaultValue: answers.authorName,
		helpYAML:     "author:\n  name: Alex Martin",
	})
	if err != nil {
		return configInitAnswers{}, err
	}
	authorTitle, err := promptString(reader, output, wizardPrompt{
		title:        "Author title",
		options:      "free text or empty",
		example:      "Staff Engineer",
		defaultValue: answers.authorTitle,
		helpYAML:     "author:\n  title: Staff Engineer",
	})
	if err != nil {
		return configInitAnswers{}, err
	}
	authorEmail, err := promptString(reader, output, wizardPrompt{
		title:        "Author email",
		options:      "email or empty",
		example:      "alex@example.com",
		defaultValue: answers.authorEmail,
		helpYAML:     "author:\n  email: alex@example.com",
	})
	if err != nil {
		return configInitAnswers{}, err
	}
	authorOrganization, err := promptString(reader, output, wizardPrompt{
		title:        "Author organization",
		options:      "free text or empty",
		example:      "Acme Corp",
		defaultValue: answers.authorOrganization,
		helpYAML:     "author:\n  organization: Acme Corp",
	})
	if err != nil {
		return configInitAnswers{}, err
	}
	pageSize, err := promptString(reader, output, wizardPrompt{
		title:        "Page size",
		options:      "letter, a4, legal",
		example:      "a4",
		defaultValue: answers.pageSize,
		helpYAML:     "page:\n  size: letter",
		validate:     validatePageSize,
	})
	if err != nil {
		return configInitAnswers{}, err
	}

	answers.style = strings.ToLower(style)
	answers.authorName = authorName
	answers.authorTitle = authorTitle
	answers.authorEmail = authorEmail
	answers.authorOrganization = authorOrganization
	answers.pageSize = strings.ToLower(pageSize)
	return answers, nil
}

// promptConfigInitSignatureAnswers isolates signature decisions so optional
// follow-up questions are only asked when the feature is enabled.
func promptConfigInitSignatureAnswers(reader *bufio.Reader, output io.Writer, answers configInitAnswers) (configInitAnswers, error) {
	signatureEnabled, err := promptBool(reader, output, wizardPrompt{
		title:        "Enable signature block",
		options:      "yes, no",
		example:      "yes",
		defaultValue: boolDefaultLabel(answers.signatureEnabled),
		helpYAML:     "signature:\n  enabled: true\n  imagePath: ./assets/signature.png",
	})
	if err != nil {
		return configInitAnswers{}, err
	}
	signatureImagePath := answers.signatureImagePath
	if signatureEnabled {
		signatureImagePath, err = promptString(reader, output, wizardPrompt{
			title:        "Signature image path",
			options:      "file path or URL",
			example:      "./assets/signature.png",
			defaultValue: signatureImagePath,
			helpYAML:     "signature:\n  imagePath: ./assets/signature.png",
		})
		if err != nil {
			return configInitAnswers{}, err
		}
	}
	answers.signatureEnabled = signatureEnabled
	answers.signatureImagePath = signatureImagePath
	return answers, nil
}

// promptConfigInitWatermarkAnswers keeps watermark branching localized to avoid
// spreading conditional prompt logic across the main wizard flow.
func promptConfigInitWatermarkAnswers(reader *bufio.Reader, output io.Writer, answers configInitAnswers) (configInitAnswers, error) {
	watermarkEnabled, err := promptBool(reader, output, wizardPrompt{
		title:        "Enable watermark",
		options:      "yes, no",
		example:      "yes",
		defaultValue: boolDefaultLabel(answers.watermarkEnabled),
		helpYAML:     "watermark:\n  enabled: true\n  text: DRAFT\n  color: #888888\n  opacity: 0.1\n  angle: -45",
	})
	if err != nil {
		return configInitAnswers{}, err
	}
	watermarkText := answers.watermarkText
	watermarkColor := answers.watermarkColor
	if watermarkEnabled {
		watermarkText, err = promptString(reader, output, wizardPrompt{
			title:        "Watermark text",
			options:      "free text (required when enabled)",
			example:      "CONFIDENTIAL",
			defaultValue: watermarkText,
			helpYAML:     "watermark:\n  text: CONFIDENTIAL",
		})
		if err != nil {
			return configInitAnswers{}, err
		}
		watermarkColor, err = promptString(reader, output, wizardPrompt{
			title:        "Watermark color",
			options:      "hex color (#RGB or #RRGGBB)",
			example:      "#888888",
			defaultValue: watermarkColor,
			helpYAML:     "watermark:\n  color: #888888",
			validate:     validateWatermarkColor,
		})
		if err != nil {
			return configInitAnswers{}, err
		}
	}
	answers.watermarkEnabled = watermarkEnabled
	answers.watermarkText = watermarkText
	answers.watermarkColor = watermarkColor
	return answers, nil
}

// promptConfigInitCoverAnswers keeps cover-specific branching local so optional
// logo handling remains easy to evolve without touching unrelated prompts.
func promptConfigInitCoverAnswers(reader *bufio.Reader, output io.Writer, answers configInitAnswers) (configInitAnswers, error) {
	coverEnabled, err := promptBool(reader, output, wizardPrompt{
		title:        "Enable cover page",
		options:      "yes, no",
		example:      "yes",
		defaultValue: boolDefaultLabel(answers.coverEnabled),
		helpYAML:     "cover:\n  enabled: true\n  logo: ./assets/logo.png",
	})
	if err != nil {
		return configInitAnswers{}, err
	}
	coverLogo := answers.coverLogo
	if coverEnabled {
		coverLogo, err = promptString(reader, output, wizardPrompt{
			title:        "Cover logo path",
			options:      "file path or URL",
			example:      "./assets/logo.png",
			defaultValue: coverLogo,
			helpYAML:     "cover:\n  logo: ./assets/logo.png",
		})
		if err != nil {
			return configInitAnswers{}, err
		}
	}
	answers.coverEnabled = coverEnabled
	answers.coverLogo = coverLogo
	return answers, nil
}

// buildConfigInitConfigFromAnswers materializes validated answers into config in
// one place so field mapping stays consistent as wizard prompts evolve.
func buildConfigInitConfigFromAnswers(answers configInitAnswers) *config.Config {
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
	cfg.Watermark.Opacity = picoloom.DefaultWatermarkOpacity
	cfg.Watermark.Angle = picoloom.DefaultWatermarkAngle
	cfg.Cover.Enabled = answers.coverEnabled
	cfg.Cover.Logo = answers.coverLogo
	return cfg
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
	test := &picoloom.Watermark{
		Color:   value,
		Opacity: picoloom.DefaultWatermarkOpacity,
		Angle:   picoloom.DefaultWatermarkAngle,
	}
	if err := test.Validate(); err != nil {
		return err
	}
	return nil
}
