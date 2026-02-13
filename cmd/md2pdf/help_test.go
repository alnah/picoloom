package main

// Notes:
// - printUsage/printConvertUsage: we test that required content strings are
//   present in the output. We don't test exact formatting as that's an
//   implementation detail.
// - runHelp: we test routing to the correct help topic.
// These are acceptable gaps: we test observable behavior, not implementation details.

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	md2pdf "github.com/alnah/go-md2pdf"
)

// ---------------------------------------------------------------------------
// TestPrintUsage - Main usage output
// ---------------------------------------------------------------------------

func TestPrintUsage(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	printUsage(&buf)
	output := buf.String()

	requiredStrings := []string{
		"Usage: md2pdf",
		"Commands:",
		"convert",
		"version",
		"help",
	}

	for _, s := range requiredStrings {
		if !strings.Contains(output, s) {
			t.Errorf("printUsage() output missing %q", s)
		}
	}
}

// ---------------------------------------------------------------------------
// TestPrintConvertUsage - Convert command usage output
// ---------------------------------------------------------------------------

func TestPrintConvertUsage(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	printConvertUsage(&buf)
	output := buf.String()

	// Check for flag group headers
	flagGroups := []string{
		"Author:",
		"Document:",
		"Page:",
		"Footer:",
		"Cover:",
		"Signature:",
		"Table of Contents:",
		"Watermark:",
		"Page Breaks:",
		"Styling:",
	}

	for _, group := range flagGroups {
		if !strings.Contains(output, group) {
			t.Errorf("printConvertUsage() output missing group header %q", group)
		}
	}

	// Check for new author flags
	authorFlags := []string{
		"--author-name",
		"--author-title",
		"--author-email",
		"--author-org",
	}

	for _, flag := range authorFlags {
		if !strings.Contains(output, flag) {
			t.Errorf("printConvertUsage() output missing %q", flag)
		}
	}

	// Check for new document flags
	documentFlags := []string{
		"--doc-title",
		"--doc-subtitle",
		"--doc-version",
		"--doc-date",
	}

	for _, flag := range documentFlags {
		if !strings.Contains(output, flag) {
			t.Errorf("printConvertUsage() output missing %q", flag)
		}
	}

	// Check for watermark shorthand flags
	wmFlags := []string{
		"--wm-text",
		"--wm-color",
		"--wm-opacity",
		"--wm-angle",
	}

	for _, flag := range wmFlags {
		if !strings.Contains(output, flag) {
			t.Errorf("printConvertUsage() output missing %q", flag)
		}
	}

	// Check for timeout flag (both short and long forms)
	timeoutFlags := []string{
		"-t, --timeout",
		"default: 30s",
		"30s, 2m, 1m30s",
	}

	for _, flag := range timeoutFlags {
		if !strings.Contains(output, flag) {
			t.Errorf("printConvertUsage() output missing %q", flag)
		}
	}

	// Check for exit codes section
	exitCodesSection := []string{
		"Exit Codes:",
		"0  Success",
		"1  General",
		"2  Usage",
		"3  I/O",
		"4  Browser",
	}

	for _, s := range exitCodesSection {
		if !strings.Contains(output, s) {
			t.Errorf("printConvertUsage() output missing %q", s)
		}
	}

	// Check for EXAMPLES section
	if !strings.Contains(output, "EXAMPLES") {
		t.Error("printConvertUsage() output missing EXAMPLES section")
	}

	examples := []string{
		"md2pdf convert document.md",
		"md2pdf convert -o report.pdf",
		"md2pdf convert ./docs/",
		"md2pdf convert -c work",
		"--style technical --timeout 2m",
		"-p a4 --orientation landscape --wm-text DRAFT",
	}

	for _, ex := range examples {
		if !strings.Contains(output, ex) {
			t.Errorf("printConvertUsage() output missing example: %q", ex)
		}
	}
}

// ---------------------------------------------------------------------------
// TestHelpDefaultsMatchConstants - Verify documented defaults match actual values
// ---------------------------------------------------------------------------

func TestHelpDefaultsMatchConstants(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	printConvertUsage(&buf)
	output := buf.String()

	// Map of documented defaults to actual constants
	// This ensures help stays in sync with code
	defaults := []struct {
		name     string
		expected string
	}{
		{"page-size", fmt.Sprintf("default: %s", md2pdf.PageSizeLetter)},
		{"orientation", fmt.Sprintf("default: %s", md2pdf.OrientationPortrait)},
		{"margin", fmt.Sprintf("default: %.1f", md2pdf.DefaultMargin)},
		{"toc-min-depth", fmt.Sprintf("default: %d", md2pdf.DefaultTOCMinDepth)},
		{"toc-max-depth", fmt.Sprintf("default: %d", md2pdf.DefaultTOCMaxDepth)},
		{"wm-color", fmt.Sprintf("default: %s", md2pdf.DefaultWatermarkColor)},
		{"wm-opacity", fmt.Sprintf("default: %.1f", md2pdf.DefaultWatermarkOpacity)},
		{"wm-angle", fmt.Sprintf("default: %.0f", md2pdf.DefaultWatermarkAngle)},
		{"orphans", fmt.Sprintf("default: %d", md2pdf.DefaultOrphans)},
		{"widows", fmt.Sprintf("default: %d", md2pdf.DefaultWidows)},
	}

	for _, d := range defaults {
		if !strings.Contains(output, d.expected) {
			t.Errorf("printConvertUsage() help for --%s missing %q", d.name, d.expected)
		}
	}
}

// ---------------------------------------------------------------------------
// TestRunHelp - Help command routing
// ---------------------------------------------------------------------------

func TestRunHelp(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		args         []string
		wantInStdout []string
		wantInStderr []string
	}{
		{
			name:         "no args shows main usage",
			args:         []string{},
			wantInStdout: []string{"Usage: md2pdf", "Commands:"},
		},
		{
			name:         "convert shows convert help",
			args:         []string{"convert"},
			wantInStdout: []string{"Usage: md2pdf convert", "Author:", "Document:"},
		},
		{
			name:         "version shows version help",
			args:         []string{"version"},
			wantInStdout: []string{"Usage: md2pdf version"},
		},
		{
			name:         "help shows help help",
			args:         []string{"help"},
			wantInStdout: []string{"Usage: md2pdf help"},
		},
		{
			name:         "unknown command shows error",
			args:         []string{"unknown"},
			wantInStderr: []string{"Unknown command: unknown"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			loader, _ := md2pdf.NewAssetLoader("")
			var stdout, stderr bytes.Buffer
			env := &Environment{
				Stdout:      &stdout,
				Stderr:      &stderr,
				AssetLoader: loader,
			}

			runHelp(tt.args, env)

			stdoutStr := stdout.String()
			stderrStr := stderr.String()

			for _, want := range tt.wantInStdout {
				if !strings.Contains(stdoutStr, want) {
					t.Errorf("runHelp(%v) stdout missing %q, got %q", tt.args, want, stdoutStr)
				}
			}

			for _, want := range tt.wantInStderr {
				if !strings.Contains(stderrStr, want) {
					t.Errorf("runHelp(%v) stderr missing %q, got %q", tt.args, want, stderrStr)
				}
			}
		})
	}
}
