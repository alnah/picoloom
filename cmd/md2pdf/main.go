package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	md2pdf "github.com/alnah/go-md2pdf"
	"github.com/alnah/go-md2pdf/internal/config"
	"go.uber.org/automaxprocs/maxprocs"
)

// Version is set at build time via ldflags.
var Version = "dev"

func main() {
	env := DefaultEnv()
	os.Exit(runMain(os.Args, env))
}

// runMain is the main entry point, testable via dependency injection.
func runMain(args []string, env *Environment) int {
	if len(args) < 2 {
		printUsage(env.Stderr)
		return ExitUsage
	}

	cmd := args[1]
	cmdArgs := args[2:]

	// Legacy detection: if first arg looks like a markdown file, warn and run convert
	if !isCommand(cmd) && looksLikeMarkdown(cmd) {
		fmt.Fprintln(env.Stderr, "DEPRECATED: use 'md2pdf convert' instead")
		cmd = "convert"
		cmdArgs = args[1:]
	}

	switch cmd {
	case "convert":
		if err := runConvertCmd(cmdArgs, env); err != nil {
			fmt.Fprintln(env.Stderr, err)
			return exitCodeFor(err)
		}
	case "config":
		if err := runConfigCmd(cmdArgs, env); err != nil {
			fmt.Fprintln(env.Stderr, err)
			return exitCodeFor(err)
		}
	case "doctor":
		return runDoctorCmd(cmdArgs, env)
	case "version":
		fmt.Fprintf(env.Stdout, "md2pdf %s\n", Version)
	case "help":
		runHelp(cmdArgs, env)
	case "completion":
		if err := runCompletion(cmdArgs, env); err != nil {
			fmt.Fprintln(env.Stderr, err)
			return exitCodeFor(err)
		}
	default:
		fmt.Fprintf(env.Stderr, "unknown command: %s\n", cmd)
		printUsage(env.Stderr)
		return ExitUsage
	}

	return ExitSuccess
}

// isCommand checks if a string is a known command.
func isCommand(s string) bool {
	switch s {
	case "convert", "config", "doctor", "version", "help", "completion":
		return true
	}
	return false
}

// looksLikeMarkdown checks if a string looks like a markdown file.
func looksLikeMarkdown(s string) bool {
	return strings.HasSuffix(s, ".md") || strings.HasSuffix(s, ".markdown")
}

// runConvertCmd handles the convert command.
func runConvertCmd(args []string, env *Environment) error {
	flags, positionalArgs, err := parseConvertFlags(args)
	if err != nil {
		return err
	}

	envCfg := loadEnvConfig()
	warnUnknownEnvVars(env.Stderr)

	if err := resolveWorkers(flags, envCfg); err != nil {
		return err
	}
	configureMaxProcs(flags.common.verbose, env)

	if err := loadRuntimeConfig(flags, envCfg, env); err != nil {
		return err
	}
	if err := configureAssetLoader(flags, env); err != nil {
		return err
	}

	templateSet, err := resolveTemplateSetForRun(flags, env)
	if err != nil {
		return err
	}

	timeout, err := resolveTimeoutWithEnv(flags.timeout, envCfg.Timeout, env.Config.Timeout)
	if err != nil {
		return err
	}

	converterPool := createConverterPool(flags, env, templateSet, timeout)
	defer converterPool.Close()

	pool := &poolAdapter{pool: converterPool}
	ctx, stop := notifyContext(context.Background())
	defer stop()

	if flags.common.verbose {
		fmt.Fprintln(env.Stderr, "Starting conversion...")
	}

	return runConvert(ctx, positionalArgs, flags, pool, env)
}

func resolveWorkers(flags *convertFlags, envCfg *envConfig) error {
	workers := flags.workers
	if workers == 0 && envCfg.Workers > 0 {
		workers = envCfg.Workers
	}
	if err := validateWorkers(workers); err != nil {
		return err
	}
	flags.workers = workers
	return nil
}

func configureMaxProcs(verbose bool, env *Environment) {
	if verbose {
		_, _ = maxprocs.Set(maxprocs.Logger(func(format string, args ...interface{}) {
			fmt.Fprintf(env.Stderr, format+"\n", args...)
		}))
		return
	}
	_, _ = maxprocs.Set(maxprocs.Logger(func(string, ...interface{}) {}))
}

func loadRuntimeConfig(flags *convertFlags, envCfg *envConfig, env *Environment) error {
	configPath := resolveConfigPath(flags.common.config, envCfg.ConfigPath)

	if env.Config == nil {
		env.Config = config.DefaultConfig()
	}
	if configPath != "" {
		loaded, err := config.LoadConfig(configPath)
		if err != nil {
			return fmt.Errorf("loading config: %w", err)
		}
		env.Config = loaded
	}

	// Priority: CLI flags > config file > env vars > defaults.
	// Env vars only fill missing config values here.
	applyEnvConfig(envCfg, env.Config)
	return nil
}

func resolveConfigPath(flagPath, envPath string) string {
	if flagPath != "" {
		return flagPath
	}
	return envPath
}

func configureAssetLoader(flags *convertFlags, env *Environment) error {
	assetBasePath := resolveAssetBasePath(flags, env.Config)
	if assetBasePath == "" {
		return nil
	}

	loader, err := md2pdf.NewAssetLoader(assetBasePath)
	if err != nil {
		return fmt.Errorf("initializing assets: %w", err)
	}
	env.AssetLoader = loader

	if flags.common.verbose {
		fmt.Fprintf(env.Stderr, "Using custom assets from: %s\n", assetBasePath)
	}
	return nil
}

func resolveAssetBasePath(flags *convertFlags, cfg *config.Config) string {
	if flags.assets.assetPath != "" {
		return flags.assets.assetPath
	}
	return cfg.Assets.BasePath
}

func resolveTemplateSetForRun(flags *convertFlags, env *Environment) (*md2pdf.TemplateSet, error) {
	templateSet, err := resolveTemplateSet(flags.assets.template, env.AssetLoader)
	if err != nil {
		return nil, fmt.Errorf("loading template set: %w", err)
	}
	if flags.common.verbose && flags.assets.template != "" {
		fmt.Fprintf(env.Stderr, "Using template set: %s\n", templateSet.Name)
	}
	return templateSet, nil
}

func createConverterPool(flags *convertFlags, env *Environment, templateSet *md2pdf.TemplateSet, timeout time.Duration) *md2pdf.ConverterPool {
	poolSize := md2pdf.ResolvePoolSize(flags.workers)
	if flags.common.verbose {
		fmt.Fprintf(env.Stderr, "Pool size: %d\n", poolSize)
		if timeout > 0 {
			fmt.Fprintf(env.Stderr, "Timeout: %v\n", timeout)
		}
	}
	return md2pdf.NewConverterPool(poolSize, buildPoolOptions(env.AssetLoader, templateSet, timeout)...)
}

func buildPoolOptions(loader md2pdf.AssetLoader, templateSet *md2pdf.TemplateSet, timeout time.Duration) []md2pdf.Option {
	opts := []md2pdf.Option{
		md2pdf.WithAssetLoader(loader),
		md2pdf.WithTemplateSet(templateSet),
	}
	if timeout > 0 {
		opts = append(opts, md2pdf.WithTimeout(timeout))
	}
	return opts
}

// poolAdapter adapts md2pdf.ConverterPool to the local Pool interface.
type poolAdapter struct {
	pool *md2pdf.ConverterPool
}

func (a *poolAdapter) Acquire() CLIConverter {
	return a.pool.Acquire()
}

func (a *poolAdapter) Release(c CLIConverter) {
	conv, ok := c.(*md2pdf.Converter)
	if !ok {
		// Defensive no-op: pool only manages *md2pdf.Converter instances.
		// Avoid crashing the CLI if a wrong test double/type is passed.
		return
	}
	a.pool.Release(conv)
}

func (a *poolAdapter) Size() int {
	return a.pool.Size()
}

func parsePositiveDuration(value string) (time.Duration, error) {
	d, err := time.ParseDuration(value)
	if err != nil {
		return 0, fmt.Errorf("invalid timeout %q (use format like \"30s\", \"2m\")", value)
	}
	if d <= 0 {
		return 0, fmt.Errorf("timeout must be positive, got %q", value)
	}
	return d, nil
}

// resolveTimeoutWithEnv parses timeout with priority: flag > env > config.
// Returns 0 if none is set (use library default).
func resolveTimeoutWithEnv(flagValue string, envValue time.Duration, configValue string) (time.Duration, error) {
	// Flag takes highest priority
	if flagValue != "" {
		return parsePositiveDuration(flagValue)
	}

	// Env var is second priority (already parsed as duration)
	if envValue > 0 {
		return envValue, nil
	}

	// Config file is third priority
	if configValue != "" {
		return parsePositiveDuration(configValue)
	}

	return 0, nil
}
