package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/alnah/picoloom/v2/internal/config"
	"github.com/alnah/picoloom/v2/internal/yamlutil"
	flag "github.com/spf13/pflag"
)

const (
	// defaultConfigInitOutputPath keeps the common local config convention,
	// so generated examples and file discovery stay aligned.
	defaultConfigInitOutputPath = "./picoloom.yaml"
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
		return fmt.Errorf("%w: unknown subcommand %q (run '%s help config')", ErrConfigCommandUsage, args[0], envCLIName(env))
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
	fmt.Fprintf(env.Stdout, "  %s convert -c %s ./docs/\n", envCLIName(env), outputPathForExample(flags.output))
	return nil
}

// buildConfigInitConfig starts from conservative defaults so non-interactive
// generation is immediately usable and interactive mode can be safely canceled.
func buildConfigInitConfig(noInput bool, env *Environment) (*config.Config, bool, error) {
	answers := defaultConfigInitAnswers()

	var reader *bufio.Reader
	if !noInput {
		reader = bufio.NewReader(stdinReader(env))
		interactiveAnswers, err := collectConfigInitInteractiveAnswers(reader, env.Stdout, answers)
		if err != nil {
			return nil, false, err
		}
		answers = interactiveAnswers
	}

	cfg := buildConfigInitConfigFromAnswers(answers)

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
	if path == defaultConfigInitOutputPath || path == "picoloom.yaml" || path == "md2pdf.yaml" {
		return defaultConfigInitOutputPath
	}
	return path
}

// printConfigUsage documents the config command namespace entry point.
func printConfigUsage(w io.Writer) {
	printConfigUsageFor(w, canonicalCLIName)
}

func printConfigUsageFor(w io.Writer, cliName string) {
	fmt.Fprintf(w, "Usage: %s config <subcommand> [flags]\n", cliName)
	fmt.Fprintln(w)
	fmt.Fprintf(w, "Manage %s configuration files.\n", cliName)
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Subcommands:")
	fmt.Fprintln(w, "  init        Create a config file with an interactive wizard")
	fmt.Fprintln(w)
	fmt.Fprintf(w, "Run '%s help config init' for command details.\n", cliName)
}

// printConfigInitUsage documents config-init flags and canonical usage examples.
func printConfigInitUsage(w io.Writer) {
	printConfigInitUsageFor(w, canonicalCLIName)
}

func printConfigInitUsageFor(w io.Writer, cliName string) {
	fmt.Fprintf(w, "Usage: %s config init [flags]\n", cliName)
	fmt.Fprintln(w)
	fmt.Fprintf(w, "Create a new %s configuration file.\n", cliName)
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Flags:")
	fmt.Fprintln(w, "      --output <path>   Output path for generated config (default: ./picoloom.yaml)")
	fmt.Fprintln(w, "      --force           Overwrite destination if it exists")
	fmt.Fprintln(w, "      --no-input        Use defaults without interactive prompts")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Examples:")
	fmt.Fprintf(w, "  %s config init\n", cliName)
	fmt.Fprintf(w, "  %s config init --output ./configs/work.yaml\n", cliName)
	fmt.Fprintf(w, "  %s config init --no-input --force\n", cliName)
}
