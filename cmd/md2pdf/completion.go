package main

import (
	"fmt"
	"io"

	flag "github.com/spf13/pflag"
)

// Shell represents a supported shell for completion generation.
type Shell string

// Supported shells for completion.
const (
	ShellBash       Shell = "bash"
	ShellZsh        Shell = "zsh"
	ShellFish       Shell = "fish"
	ShellPowerShell Shell = "powershell"
)

// ErrUnsupportedShell is returned when an unknown shell is requested.
var ErrUnsupportedShell = fmt.Errorf("unsupported shell")

// flagType represents the completion type for a flag.
type flagType int

const (
	flagString flagType = iota // default
	flagBool
	flagInt
	flagFloat
	flagEnum // has predefined values
	flagFile // file with glob pattern
	flagDir  // directory
)

// flagDef describes a flag for completion purposes.
type flagDef struct {
	Long     string   // --output
	Short    string   // -o (empty if none)
	Type     flagType // completion type
	Desc     string   // help text
	Values   []string // for enum flags
	FileGlob string   // for file flags
}

// commandDef describes a command for completion.
type commandDef struct {
	Name        string
	Desc        string
	Flags       []flagDef
	TakesFiles  bool   // accepts file arguments
	FilePattern string // glob for file arguments (e.g., "*.md")
}

// completionMeta holds completion-specific metadata for flags.
// This is the ONLY place where completion hints are defined.
// Flag names, types, and descriptions come from the FlagSet.
type completionMeta struct {
	Values   []string // enum values
	FileGlob string   // file glob pattern
	IsDir    bool     // directory completion
}

// flagCompletionMeta maps flag names to their completion metadata.
var flagCompletionMeta = map[string]completionMeta{
	// Enum flags
	"page-size":       {Values: []string{"letter", "a4", "legal"}},
	"orientation":     {Values: []string{"portrait", "landscape"}},
	"footer-position": {Values: []string{"left", "center", "right"}},

	// File flags with glob patterns
	"config":     {FileGlob: "*.yaml,*.yml"},
	"style":      {FileGlob: "*.css"},
	"cover-logo": {FileGlob: "*.png,*.jpg,*.jpeg,*.svg"},
	"sig-image":  {FileGlob: "*.png,*.jpg,*.jpeg"},

	// Directory flags
	"output":     {IsDir: true},
	"template":   {IsDir: true},
	"asset-path": {IsDir: true},
}

// buildConvertFlagSet creates a FlagSet with all convert command flags.
// This reuses the same flag registration as parseConvertFlags.
func buildConvertFlagSet() *flag.FlagSet {
	fs := flag.NewFlagSet("convert", flag.ContinueOnError)
	f := &convertFlags{}

	// I/O flags
	fs.StringVarP(&f.output, "output", "o", "", "output file or directory")
	fs.IntVarP(&f.workers, "workers", "w", 0, "parallel workers (0 = auto)")

	// Flag groups - same as parseConvertFlags
	addCommonFlags(fs, &f.common)
	addAuthorFlags(fs, &f.author)
	addDocumentFlags(fs, &f.document)
	addPageFlags(fs, &f.page)
	addFooterFlags(fs, &f.footer)
	addCoverFlags(fs, &f.cover)
	addSignatureFlags(fs, &f.signature)
	addTOCFlags(fs, &f.toc)
	addWatermarkFlags(fs, &f.watermark)
	addPageBreakFlags(fs, &f.pageBreaks)
	addAssetFlags(fs, &f.assets)
	addOutputFlags(fs, &f.outputMode)

	return fs
}

// extractFlagsFromFlagSet extracts flag definitions from a pflag.FlagSet.
// Enriches with completion metadata from flagCompletionMeta.
func extractFlagsFromFlagSet(fs *flag.FlagSet) []flagDef {
	var flags []flagDef

	fs.VisitAll(func(f *flag.Flag) {
		fd := flagDef{
			Long:  f.Name,
			Short: f.Shorthand,
			Desc:  f.Usage,
		}

		// Determine base type from pflag type
		switch f.Value.Type() {
		case "bool":
			fd.Type = flagBool
		case "int", "int8", "int16", "int32", "int64", "uint", "uint8", "uint16", "uint32", "uint64":
			fd.Type = flagInt
		case "float32", "float64":
			fd.Type = flagFloat
		default:
			fd.Type = flagString
		}

		// Override type based on completion metadata
		if meta, ok := flagCompletionMeta[f.Name]; ok {
			if len(meta.Values) > 0 {
				fd.Type = flagEnum
				fd.Values = meta.Values
			} else if meta.FileGlob != "" {
				fd.Type = flagFile
				fd.FileGlob = meta.FileGlob
			} else if meta.IsDir {
				fd.Type = flagDir
			}
		}

		flags = append(flags, fd)
	})

	return flags
}

// getCommands returns the command registry for completion.
// Flags are extracted from the actual FlagSet - single source of truth.
func getCommands() []commandDef {
	convertFlags := extractFlagsFromFlagSet(buildConvertFlagSet())

	return []commandDef{
		{
			Name:        "convert",
			Desc:        "Convert markdown files to PDF",
			Flags:       convertFlags,
			TakesFiles:  true,
			FilePattern: "*.md,*.markdown",
		},
		{
			Name:  "config",
			Desc:  "Manage configuration files",
			Flags: nil,
		},
		{
			Name:  "version",
			Desc:  "Show version information",
			Flags: nil,
		},
		{
			Name:  "help",
			Desc:  "Show help for a command",
			Flags: nil,
		},
		{
			Name:  "completion",
			Desc:  "Generate shell completion script",
			Flags: nil,
		},
	}
}

// GenerateCompletion writes shell completion script to w.
// Returns error if shell is unsupported or write fails.
func GenerateCompletion(w io.Writer, shell Shell) error {
	switch shell {
	case ShellBash:
		return generateBash(w)
	case ShellZsh:
		return generateZsh(w)
	case ShellFish:
		return generateFish(w)
	case ShellPowerShell:
		return generatePowerShell(w)
	default:
		return fmt.Errorf("%w: %q (supported: bash, zsh, fish, powershell)", ErrUnsupportedShell, shell)
	}
}

// runCompletion handles the completion command.
func runCompletion(args []string, env *Environment) error {
	if len(args) == 0 {
		printCompletionUsage(env.Stdout)
		return nil
	}

	shell := Shell(args[0])
	return GenerateCompletion(env.Stdout, shell)
}

// printCompletionUsage prints help for the completion command.
func printCompletionUsage(w io.Writer) {
	fmt.Fprintln(w, "Usage: md2pdf completion <shell>")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Generate shell completion script for the specified shell.")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Supported shells:")
	fmt.Fprintln(w, "  bash        Bash completion script")
	fmt.Fprintln(w, "  zsh         Zsh completion script")
	fmt.Fprintln(w, "  fish        Fish completion script")
	fmt.Fprintln(w, "  powershell  PowerShell completion script")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Installation:")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "  Bash:")
	fmt.Fprintln(w, "    # Add to ~/.bashrc:")
	fmt.Fprintln(w, "    eval \"$(md2pdf completion bash)\"")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "  Zsh:")
	fmt.Fprintln(w, "    # Add to ~/.zshrc (before compinit):")
	fmt.Fprintln(w, "    eval \"$(md2pdf completion zsh)\"")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "  Fish:")
	fmt.Fprintln(w, "    md2pdf completion fish > ~/.config/fish/completions/md2pdf.fish")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "  PowerShell:")
	fmt.Fprintln(w, "    # Add to $PROFILE:")
	fmt.Fprintln(w, "    md2pdf completion powershell | Out-String | Invoke-Expression")
}
