package main

// Notes:
// - parseConfigInitFlags: we test defaults, explicit options, usage errors,
//   and help path behavior.
// - promptString/promptBool: we test default acceptance, example/default
//   rendering, and invalid-input retry behavior.
// - parseYesNo: we test accepted aliases and invalid input handling.
// - outputPathForExample: we test default-path normalization and passthrough.
// These are acceptable gaps: we test observable behavior, not implementation details.

import (
	"bufio"
	"bytes"
	"errors"
	"strings"
	"testing"

	flag "github.com/spf13/pflag"
)

// ---------------------------------------------------------------------------
// TestParseConfigInitFlags_* - config init flag parsing
// ---------------------------------------------------------------------------

func TestParseConfigInitFlags_Defaults(t *testing.T) {
	t.Parallel()

	flags, err := parseConfigInitFlags(nil, &bytes.Buffer{})
	if err != nil {
		t.Fatalf("parseConfigInitFlags(nil) unexpected error: %v", err)
	}
	if flags.output != defaultConfigInitOutputPath {
		t.Fatalf("flags.output = %q, want %q", flags.output, defaultConfigInitOutputPath)
	}
	if flags.force {
		t.Fatal("flags.force = true, want false")
	}
	if flags.noInput {
		t.Fatal("flags.noInput = true, want false")
	}
}

func TestParseConfigInitFlags_ParsesOptions(t *testing.T) {
	t.Parallel()

	flags, err := parseConfigInitFlags([]string{"--output", "./configs/work.yaml", "--force", "--no-input"}, &bytes.Buffer{})
	if err != nil {
		t.Fatalf("parseConfigInitFlags(...) unexpected error: %v", err)
	}
	if flags.output != "./configs/work.yaml" {
		t.Fatalf("flags.output = %q, want %q", flags.output, "./configs/work.yaml")
	}
	if !flags.force {
		t.Fatal("flags.force = false, want true")
	}
	if !flags.noInput {
		t.Fatal("flags.noInput = false, want true")
	}
}

func TestParseConfigInitFlags_Errors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		args []string
	}{
		{
			name: "empty output",
			args: []string{"--output", "   "},
		},
		{
			name: "unexpected positional args",
			args: []string{"extra"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := parseConfigInitFlags(tt.args, &bytes.Buffer{})
			if err == nil {
				t.Fatal("parseConfigInitFlags(...) error = nil, want error")
			}
			if !errors.Is(err, ErrConfigCommandUsage) {
				t.Fatalf("parseConfigInitFlags(...) error = %v, want ErrConfigCommandUsage", err)
			}
		})
	}
}

func TestParseConfigInitFlags_Help(t *testing.T) {
	t.Parallel()

	_, err := parseConfigInitFlags([]string{"--help"}, &bytes.Buffer{})
	if err == nil {
		t.Fatal("parseConfigInitFlags([--help]) error = nil, want ErrHelp")
	}
	if !errors.Is(err, flag.ErrHelp) {
		t.Fatalf("parseConfigInitFlags([--help]) error = %v, want ErrHelp", err)
	}
}

// ---------------------------------------------------------------------------
// TestPrompt* - prompt rendering and parsing behavior
// ---------------------------------------------------------------------------

func TestPromptString_UsesDefaultAndShowsFormat(t *testing.T) {
	t.Parallel()

	reader := bufio.NewReader(strings.NewReader("\n"))
	var output bytes.Buffer

	got, err := promptString(reader, &output, "Style", "technical", "technical", nil)
	if err != nil {
		t.Fatalf("promptString(...) unexpected error: %v", err)
	}
	if got != "technical" {
		t.Fatalf("promptString(...) = %q, want %q", got, "technical")
	}

	promptText := output.String()
	if !strings.Contains(promptText, "[default: technical]") {
		t.Fatalf("prompt text = %q, missing default annotation", promptText)
	}
	if !strings.Contains(promptText, "(example: technical)") {
		t.Fatalf("prompt text = %q, missing example annotation", promptText)
	}
}

func TestPromptBool_RepromptsInvalidThenParses(t *testing.T) {
	t.Parallel()

	reader := bufio.NewReader(strings.NewReader("maybe\nyes\n"))
	var output bytes.Buffer

	got, err := promptBool(reader, &output, "Show page numbers in footer", "yes/no", true)
	if err != nil {
		t.Fatalf("promptBool(...) unexpected error: %v", err)
	}
	if !got {
		t.Fatal("promptBool(...) = false, want true")
	}
	if !strings.Contains(output.String(), "Invalid value") {
		t.Fatalf("promptBool output = %q, want invalid-value guidance", output.String())
	}
}

// ---------------------------------------------------------------------------
// TestParseYesNo - yes/no parser aliases and errors
// ---------------------------------------------------------------------------

func TestParseYesNo(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input   string
		want    bool
		wantErr bool
	}{
		{input: "yes", want: true},
		{input: "Y", want: true},
		{input: "1", want: true},
		{input: "no", want: false},
		{input: "N", want: false},
		{input: "0", want: false},
		{input: "maybe", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			t.Parallel()
			got, err := parseYesNo(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("parseYesNo(%q) error = nil, want error", tt.input)
				}
				return
			}
			if err != nil {
				t.Fatalf("parseYesNo(%q) unexpected error: %v", tt.input, err)
			}
			if got != tt.want {
				t.Fatalf("parseYesNo(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestOutputPathForExample - output example path normalization
// ---------------------------------------------------------------------------

func TestOutputPathForExample(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input string
		want  string
	}{
		{input: defaultConfigInitOutputPath, want: defaultConfigInitOutputPath},
		{input: "md2pdf.yaml", want: defaultConfigInitOutputPath},
		{input: "./configs/work.yaml", want: "./configs/work.yaml"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			t.Parallel()
			got := outputPathForExample(tt.input)
			if got != tt.want {
				t.Fatalf("outputPathForExample(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
