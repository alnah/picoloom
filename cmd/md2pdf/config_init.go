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
