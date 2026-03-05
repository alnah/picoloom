package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/alnah/go-md2pdf/internal/config"
	"github.com/alnah/go-md2pdf/internal/yamlutil"
	flag "github.com/spf13/pflag"
)

const defaultConfigInitOutputPath = "./md2pdf.yaml"

var (
	ErrConfigCommandUsage = errors.New("invalid config command usage")
	ErrConfigInitNeedsTTY = errors.New("interactive mode requires a TTY")
	ErrConfigInitExists   = errors.New("config file already exists")
)

type configInitFlags struct {
	output  string
	force   bool
	noInput bool
}

type configInitAnswers struct {
	style          string
	pageSize       string
	documentDate   string
	showPageNumber bool
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

	cfg, err := buildConfigInitConfig(flags.noInput, env)
	if err != nil {
		return err
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

func buildConfigInitConfig(noInput bool, env *Environment) (*config.Config, error) {
	answers := configInitAnswers{
		style:          "technical",
		pageSize:       "letter",
		documentDate:   "auto",
		showPageNumber: true,
	}

	if !noInput {
		reader := bufio.NewReader(stdinReader(env))
		style, err := promptString(reader, env.Stdout,
			"Style",
			"technical or ./assets/styles/corporate.css",
			answers.style,
			nil,
		)
		if err != nil {
			return nil, err
		}
		pageSize, err := promptString(reader, env.Stdout,
			"Page size",
			"letter, a4, legal",
			answers.pageSize,
			validatePageSize,
		)
		if err != nil {
			return nil, err
		}
		documentDate, err := promptString(reader, env.Stdout,
			"Document date",
			"auto or auto:DD/MM/YYYY",
			answers.documentDate,
			validateDocumentDate,
		)
		if err != nil {
			return nil, err
		}
		showPageNumber, err := promptBool(reader, env.Stdout,
			"Show page numbers in footer",
			"yes/no",
			answers.showPageNumber,
		)
		if err != nil {
			return nil, err
		}
		answers.style = style
		answers.pageSize = pageSize
		answers.documentDate = documentDate
		answers.showPageNumber = showPageNumber
	}

	cfg := config.DefaultConfig()
	cfg.Style = answers.style
	cfg.Page.Size = answers.pageSize
	cfg.Page.Orientation = "portrait"
	cfg.Document.Date = answers.documentDate
	cfg.Footer.Enabled = answers.showPageNumber
	cfg.Footer.Position = "right"
	cfg.Footer.ShowPageNumber = answers.showPageNumber

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("validating generated config: %w", err)
	}

	return cfg, nil
}

func promptString(reader *bufio.Reader, output io.Writer, title, example, defaultValue string, validate func(string) error) (string, error) {
	for {
		fmt.Fprintf(output, "%s [default: %s] (example: %s): ", title, defaultValue, example)

		line, err := reader.ReadString('\n')
		isEOF := errors.Is(err, io.EOF)
		if err != nil && !isEOF {
			return "", fmt.Errorf("reading %s answer: %w", strings.ToLower(title), err)
		}
		value := strings.TrimSpace(line)
		if value == "" {
			value = defaultValue
		}
		if validate != nil {
			if err := validate(value); err != nil {
				if isEOF {
					return "", fmt.Errorf("invalid %s value: %w", strings.ToLower(title), err)
				}
				fmt.Fprintf(output, "Invalid value: %v\n", err)
				continue
			}
		}
		return value, nil
	}
}

func promptBool(reader *bufio.Reader, output io.Writer, title, example string, defaultValue bool) (bool, error) {
	defaultLabel := "no"
	if defaultValue {
		defaultLabel = "yes"
	}

	for {
		value, err := promptString(reader, output, title, example, defaultLabel, nil)
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

func validateDocumentDate(value string) error {
	test := config.DefaultConfig()
	test.Document.Date = value
	if err := test.Document.Validate(); err != nil {
		return err
	}
	return nil
}

func writeConfigInitFile(outputPath string, data []byte, force bool) (retErr error) {
	if !force {
		if _, err := os.Stat(outputPath); err == nil {
			return fmt.Errorf("%w: %s (use --force)", ErrConfigInitExists, outputPath)
		} else if err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("checking destination %s: %w", outputPath, err)
		}
	}

	dir := filepath.Dir(outputPath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("creating destination directory: %w", err)
	}

	tmpFile, err := os.CreateTemp(dir, ".md2pdf-config-init-*.yaml")
	if err != nil {
		return fmt.Errorf("creating temp config file: %w", err)
	}

	tmpPath := tmpFile.Name()
	defer func() {
		_ = tmpFile.Close()
		if retErr != nil {
			_ = os.Remove(tmpPath)
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
		if err := os.Remove(outputPath); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("removing existing config file: %w", err)
		}
	}

	if err := os.Rename(tmpPath, outputPath); err != nil {
		return fmt.Errorf("moving generated config into place: %w", err)
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
