package main

import (
	"fmt"
	"io"
)

// printUsage prints the main usage message.
func printUsage(w io.Writer) {
	fmt.Fprintln(w, "Usage: md2pdf <command> [flags] [args]")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Commands:")
	fmt.Fprintln(w, "  convert      Convert markdown files to PDF")
	fmt.Fprintln(w, "  config       Manage configuration files")
	fmt.Fprintln(w, "  doctor       Check system configuration")
	fmt.Fprintln(w, "  completion   Generate shell completion script")
	fmt.Fprintln(w, "  version      Show version information")
	fmt.Fprintln(w, "  help         Show help for a command")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Run 'md2pdf help <command>' for details on a specific command.")
}

// printConvertUsage prints usage for the convert command.
func printConvertUsage(w io.Writer) {
	for _, line := range convertUsageLines {
		fmt.Fprintln(w, line)
	}
}

var convertUsageLines = []string{
	"Usage: md2pdf convert <input> [flags]",
	"",
	"Convert markdown files to PDF.",
	"",
	"EXAMPLES",
	"    # Convert a single file",
	"    md2pdf convert document.md",
	"",
	"    # Convert with custom output path",
	"    md2pdf convert -o report.pdf document.md",
	"",
	"    # Batch convert a directory",
	"    md2pdf convert ./docs/ -o ./pdfs/",
	"",
	"    # Use a config file",
	"    md2pdf convert -c work document.md",
	"",
	"    # Custom style and timeout",
	"    md2pdf convert --style technical --timeout 2m large.md",
	"",
	"    # A4 landscape with watermark",
	"    md2pdf convert -p a4 --orientation landscape --wm-text DRAFT doc.md",
	"",
	"Arguments:",
	"  input    Markdown file or directory (optional if config has input.defaultDir)",
	"",
	"Input/Output:",
	"  -o, --output <path>       Output file or directory",
	"  -c, --config <name>       Config file name or path",
	"  -w, --workers <n>         Parallel workers (0 = auto)",
	"  -t, --timeout <duration>  PDF generation timeout (default: 30s)",
	"                            Examples: 30s, 2m, 1m30s",
	"",
	"Author:",
	"      --author-name <s>     Author name",
	"      --author-title <s>    Author professional title",
	"      --author-email <s>    Author email",
	"      --author-org <s>      Organization name",
	"      --author-phone <s>    Author phone number",
	"      --author-address <s>  Author postal address",
	"      --author-dept <s>     Author department",
	"",
	"Document:",
	"      --doc-title <s>       Document title (\"\" = auto from H1)",
	"      --doc-subtitle <s>    Document subtitle",
	"      --doc-version <s>     Version string",
	"      --doc-date <s>        Date: \"auto\", \"auto:FORMAT\", or literal",
	"                            Tokens: YYYY, YY, MMMM, MMM, MM, M, DD, D",
	"                            Presets (case-insensitive): iso, european, us, long",
	"                            Use [text] to escape literals: [Date]: YYYY",
	"      --doc-client <s>      Client name",
	"      --doc-project <s>     Project name",
	"      --doc-type <s>        Document type",
	"      --doc-id <s>          Document ID/reference",
	"      --doc-desc <s>        Document description",
	"",
	"Page:",
	"  -p, --page-size <s>       letter, a4, legal (default: letter)",
	"      --orientation <s>     portrait, landscape (default: portrait)",
	"      --margin <f>          Margin in inches (default: 0.5)",
	"",
	"Footer:",
	"      --footer-position <s> left, center, right (default: right)",
	"      --footer-text <s>     Custom footer text",
	"      --footer-page-number  Show page numbers",
	"      --footer-doc-id       Show document ID in footer",
	"      --no-footer           Disable footer",
	"",
	"Cover:",
	"      --cover-logo <path>   Logo path or URL",
	"      --cover-dept          Show author department on cover",
	"      --no-cover            Disable cover page",
	"",
	"Signature:",
	"      --sig-image <path>    Signature image path",
	"      --no-signature        Disable signature block",
	"",
	"Table of Contents:",
	"      --toc-title <s>       TOC heading text",
	"      --toc-min-depth <n>   Min heading depth (1-6, default: 2)",
	"                            1=H1, 2=H2, etc. Use 2 to skip title",
	"      --toc-max-depth <n>   Max heading depth (1-6, default: 3)",
	"      --no-toc              Disable table of contents",
	"",
	"Watermark:",
	"      --wm-text <s>         Watermark text",
	"      --wm-color <s>        Color hex (default: #888888)",
	"      --wm-opacity <f>      Opacity 0.0-1.0 (default: 0.1)",
	"      --wm-angle <f>        Angle in degrees (default: -45)",
	"      --no-watermark        Disable watermark",
	"",
	"Page Breaks:",
	"      --break-before <s>    Break before headings: h1,h2,h3",
	"      --orphans <n>         Min lines at page bottom (default: 2)",
	"      --widows <n>          Min lines at page top (default: 2)",
	"      --no-page-breaks      Disable page break features",
	"",
	"Assets & Styling:",
	"      --style <name|path>   CSS style name or file path (default: default)",
	"                            Name: uses embedded or custom asset",
	"                            Path: reads file directly (contains / or \\)",
	"      --template <name|path> Template set name or directory path",
	"                            Name: uses embedded or custom asset",
	"                            Path: loads from directory (contains / or \\)",
	"      --asset-path <dir>    Custom asset directory (overrides config)",
	"      --no-style            Disable CSS styling",
	"",
	"Debug Output:",
	"      --html                Output HTML alongside PDF",
	"      --html-only           Output HTML only, skip PDF generation",
	"                            (if both specified, --html-only takes precedence)",
	"",
	"Output Control:",
	"  -q, --quiet               Only show errors",
	"  -v, --verbose             Show detailed timing",
	"",
	"Exit Codes:",
	"  0  Success      Conversion completed",
	"  1  General      Unexpected error",
	"  2  Usage        Invalid flags, config, or validation",
	"  3  I/O          File not found, permission denied",
	"  4  Browser      Chrome not found, connection failed",
}

// printDoctorUsage prints usage for the doctor command.
func printDoctorUsage(w io.Writer) {
	fmt.Fprintln(w, "Usage: md2pdf doctor [flags]")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Check system configuration for PDF generation.")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Flags:")
	fmt.Fprintln(w, "  --json    Output in JSON format (for CI/scripts)")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Checks performed:")
	fmt.Fprintln(w, "  - Chrome/Chromium: binary exists, version, sandbox status")
	fmt.Fprintln(w, "  - Environment: container detection (Docker, Podman, Kubernetes)")
	fmt.Fprintln(w, "  - System: temp directory writability")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Environment variables:")
	fmt.Fprintln(w, "  MD2PDF_CONTAINER=1    Force container detection")
	fmt.Fprintln(w, "  ROD_BROWSER_BIN       Explicit path to Chrome binary")
	fmt.Fprintln(w, "  ROD_NO_SANDBOX=1      Disable Chrome sandbox (required in containers)")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Exit codes:")
	fmt.Fprintln(w, "  0  All checks passed (including warnings)")
	fmt.Fprintln(w, "  1  Errors found (conversion will likely fail)")
}

// runHelp prints help for a specific command.
func runHelp(args []string, env *Environment) {
	if len(args) == 0 {
		printUsage(env.Stdout)
		return
	}

	switch args[0] {
	case "convert":
		printConvertUsage(env.Stdout)
	case "config":
		if len(args) > 1 && args[1] == "init" {
			printConfigInitUsage(env.Stdout)
			return
		}
		printConfigUsage(env.Stdout)
	case "doctor":
		printDoctorUsage(env.Stdout)
	case "completion":
		printCompletionUsage(env.Stdout)
	case "version":
		fmt.Fprintln(env.Stdout, "Usage: md2pdf version")
		fmt.Fprintln(env.Stdout)
		fmt.Fprintln(env.Stdout, "Show version information.")
	case "help":
		fmt.Fprintln(env.Stdout, "Usage: md2pdf help [command]")
		fmt.Fprintln(env.Stdout)
		fmt.Fprintln(env.Stdout, "Show help for a command.")
	default:
		fmt.Fprintf(env.Stderr, "Unknown command: %s\n", args[0])
		printUsage(env.Stderr)
	}
}
