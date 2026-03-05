package main

// Notes:
// - GenerateCompletion: we test that shell scripts are generated with expected
//   content markers. We do not test that the scripts actually work in the
//   target shell (that would require integration tests with actual shells).
// - getCommands: we test the command definitions are complete and correct.
// These are acceptable gaps: we test observable behavior, not runtime shell behavior.

import (
	"bytes"
	"errors"
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// TestGenerateCompletion - Shell completion script generation
// ---------------------------------------------------------------------------

func TestGenerateCompletion(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		shell          Shell
		wantContains   []string
		wantNotContain []string
	}{
		{
			name:  "bash generates valid script",
			shell: ShellBash,
			wantContains: []string{
				"_md2pdf_completions",
				"complete -F",
				"compgen",
				"convert",
				"--output",
				"--page-size",
			},
		},
		{
			name:  "zsh generates valid script",
			shell: ShellZsh,
			wantContains: []string{
				"#compdef md2pdf",
				"_md2pdf",
				"_arguments",
				"_describe",
				"convert",
				"--output",
			},
		},
		{
			name:  "fish generates valid script",
			shell: ShellFish,
			wantContains: []string{
				"complete -c md2pdf",
				"__fish_md2pdf_needs_command",
				"__fish_md2pdf_using_command",
				"convert",
				"-l output", // fish uses -l for long flags
			},
		},
		{
			name:  "powershell generates valid script",
			shell: ShellPowerShell,
			wantContains: []string{
				"Register-ArgumentCompleter",
				"-CommandName md2pdf",
				"CompletionResult",
				"convert",
				"--output",
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var buf bytes.Buffer
			err := GenerateCompletion(&buf, tt.shell)

			if err != nil {
				t.Fatalf("GenerateCompletion(%q) unexpected error: %v", tt.shell, err)
			}

			output := buf.String()
			if output == "" {
				t.Fatalf("GenerateCompletion(%q) produced empty output", tt.shell)
			}

			for _, want := range tt.wantContains {
				if !strings.Contains(output, want) {
					t.Errorf("output missing expected content %q", want)
				}
			}

			for _, notWant := range tt.wantNotContain {
				if strings.Contains(output, notWant) {
					t.Errorf("output contains unexpected content %q", notWant)
				}
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestGenerateCompletion - Error handling for unknown shells
// ---------------------------------------------------------------------------

func TestGenerateCompletion_UnsupportedShell(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		shell Shell
	}{
		{name: "empty shell", shell: ""},
		{name: "unknown shell", shell: "unknown"},
		{name: "sh is not supported", shell: "sh"},
		{name: "ksh is not supported", shell: "ksh"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var buf bytes.Buffer
			err := GenerateCompletion(&buf, tt.shell)

			if err == nil {
				t.Fatalf("GenerateCompletion(%q) error = nil, want error", tt.shell)
			}

			if !errors.Is(err, ErrUnsupportedShell) {
				t.Errorf("error should wrap ErrUnsupportedShell, got: %v", err)
			}

			if !strings.Contains(err.Error(), string(tt.shell)) {
				t.Errorf("error message should contain shell name %q, got: %v", tt.shell, err)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestRunCompletion - Usage message when no shell specified
// ---------------------------------------------------------------------------

func TestRunCompletion_NoArgs(t *testing.T) {
	t.Parallel()

	var stdout, stderr bytes.Buffer
	env := &Environment{
		Stdout: &stdout,
		Stderr: &stderr,
	}

	err := runCompletion([]string{}, env)

	if err != nil {
		t.Fatalf("runCompletion([]) unexpected error: %v", err)
	}

	output := stdout.String()
	if !strings.Contains(output, "Usage: md2pdf completion") {
		t.Error("expected usage message when no args provided")
	}
	if !strings.Contains(output, "bash") {
		t.Error("usage should mention bash shell")
	}
	if !strings.Contains(output, "zsh") {
		t.Error("usage should mention zsh shell")
	}
}

// ---------------------------------------------------------------------------
// TestRunCompletion - Successful completion for supported shells
// ---------------------------------------------------------------------------

func TestRunCompletion_ValidShell(t *testing.T) {
	t.Parallel()

	tests := []struct {
		shell        string
		wantContains string
	}{
		{"bash", "_md2pdf_completions"},
		{"zsh", "#compdef md2pdf"},
		{"fish", "complete -c md2pdf"},
		{"powershell", "Register-ArgumentCompleter"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.shell, func(t *testing.T) {
			t.Parallel()

			var stdout, stderr bytes.Buffer
			env := &Environment{
				Stdout: &stdout,
				Stderr: &stderr,
			}

			err := runCompletion([]string{tt.shell}, env)

			if err != nil {
				t.Fatalf("runCompletion([%q]) unexpected error: %v", tt.shell, err)
			}

			if !strings.Contains(stdout.String(), tt.wantContains) {
				t.Errorf("output missing expected %q", tt.wantContains)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestRunCompletion - Error handling for invalid shell
// ---------------------------------------------------------------------------

func TestRunCompletion_InvalidShell(t *testing.T) {
	t.Parallel()

	var stdout, stderr bytes.Buffer
	env := &Environment{
		Stdout: &stdout,
		Stderr: &stderr,
	}

	err := runCompletion([]string{"invalid"}, env)

	if err == nil {
		t.Fatal("runCompletion([\"invalid\"]) error = nil, want error")
	}

	if !errors.Is(err, ErrUnsupportedShell) {
		t.Errorf("error should wrap ErrUnsupportedShell, got: %v", err)
	}
}

// ---------------------------------------------------------------------------
// TestGetCommands - Command definitions
// ---------------------------------------------------------------------------

func TestGetCommands(t *testing.T) {
	t.Parallel()

	commands := getCommands()

	expectedCommands := []string{"convert", "config", "version", "help", "completion"}
	if len(commands) != len(expectedCommands) {
		t.Fatalf("getCommands() = %d commands, want %d", len(commands), len(expectedCommands))
	}

	commandNames := make(map[string]bool)
	for _, cmd := range commands {
		commandNames[cmd.Name] = true
	}

	for _, expected := range expectedCommands {
		if !commandNames[expected] {
			t.Errorf("missing expected command %q", expected)
		}
	}
}

// ---------------------------------------------------------------------------
// TestGetCommands - Convert command flag definitions
// ---------------------------------------------------------------------------

func TestGetCommands_ConvertHasFlags(t *testing.T) {
	t.Parallel()

	commands := getCommands()

	var convertCmd *commandDef
	for i := range commands {
		if commands[i].Name == "convert" {
			convertCmd = &commands[i]
			break
		}
	}

	if convertCmd == nil {
		t.Fatal("convert command not found")
	}

	if len(convertCmd.Flags) == 0 {
		t.Error("convert command should have flags")
	}

	if !convertCmd.TakesFiles {
		t.Error("convert command should accept files")
	}

	if convertCmd.FilePattern == "" {
		t.Error("convert command should have file pattern")
	}

	// Check for expected flags
	flagNames := make(map[string]flagDef)
	for _, f := range convertCmd.Flags {
		flagNames[f.Long] = f
	}

	expectedFlags := []struct {
		name      string
		wantShort string
		wantType  flagType
	}{
		{"output", "o", flagDir},
		{"config", "c", flagFile},
		{"page-size", "p", flagEnum},
		{"quiet", "q", flagBool},
		{"verbose", "v", flagBool},
		{"workers", "w", flagInt},
	}

	for _, expected := range expectedFlags {
		f, ok := flagNames[expected.name]
		if !ok {
			t.Errorf("missing expected flag --%s", expected.name)
			continue
		}
		if f.Short != expected.wantShort {
			t.Errorf("flag --%s: short = %q, want %q", expected.name, f.Short, expected.wantShort)
		}
		if f.Type != expected.wantType {
			t.Errorf("flag --%s: type = %v, want %v", expected.name, f.Type, expected.wantType)
		}
	}
}

// ---------------------------------------------------------------------------
// TestGetCommands - Enum flag value definitions
// ---------------------------------------------------------------------------

func TestGetCommands_EnumFlagsHaveValues(t *testing.T) {
	t.Parallel()

	commands := getCommands()

	var convertCmd *commandDef
	for i := range commands {
		if commands[i].Name == "convert" {
			convertCmd = &commands[i]
			break
		}
	}

	if convertCmd == nil {
		t.Fatal("convert command not found")
	}

	enumFlags := map[string][]string{
		"page-size":       {"letter", "a4", "legal"},
		"orientation":     {"portrait", "landscape"},
		"footer-position": {"left", "center", "right"},
	}

	for _, f := range convertCmd.Flags {
		if expectedValues, isEnum := enumFlags[f.Long]; isEnum {
			if f.Type != flagEnum {
				t.Errorf("flag --%s should be flagEnum, got %v", f.Long, f.Type)
			}
			if len(f.Values) != len(expectedValues) {
				t.Errorf("flag --%s: got %d values, want %d", f.Long, len(f.Values), len(expectedValues))
			}
			for i, v := range expectedValues {
				if i < len(f.Values) && f.Values[i] != v {
					t.Errorf("flag --%s: value[%d] = %q, want %q", f.Long, i, f.Values[i], v)
				}
			}
		}
	}
}

// ---------------------------------------------------------------------------
// TestGetCommands - File flag glob pattern definitions
// ---------------------------------------------------------------------------

func TestGetCommands_FileFlagsHaveGlobs(t *testing.T) {
	t.Parallel()

	commands := getCommands()

	var convertCmd *commandDef
	for i := range commands {
		if commands[i].Name == "convert" {
			convertCmd = &commands[i]
			break
		}
	}

	if convertCmd == nil {
		t.Fatal("convert command not found")
	}

	fileFlags := map[string]string{
		"config":     "*.yaml,*.yml",
		"style":      "*.css",
		"cover-logo": "*.png,*.jpg,*.jpeg,*.svg",
		"sig-image":  "*.png,*.jpg,*.jpeg",
	}

	for _, f := range convertCmd.Flags {
		if expectedGlob, isFile := fileFlags[f.Long]; isFile {
			if f.Type != flagFile {
				t.Errorf("flag --%s should be flagFile, got %v", f.Long, f.Type)
			}
			if f.FileGlob != expectedGlob {
				t.Errorf("flag --%s: glob = %q, want %q", f.Long, f.FileGlob, expectedGlob)
			}
		}
	}
}

// ---------------------------------------------------------------------------
// TestGetCommands - Directory flag type definitions
// ---------------------------------------------------------------------------

func TestGetCommands_DirFlagsAreMarked(t *testing.T) {
	t.Parallel()

	commands := getCommands()

	var convertCmd *commandDef
	for i := range commands {
		if commands[i].Name == "convert" {
			convertCmd = &commands[i]
			break
		}
	}

	if convertCmd == nil {
		t.Fatal("convert command not found")
	}

	dirFlags := []string{"output", "template", "asset-path"}

	for _, f := range convertCmd.Flags {
		for _, dirFlag := range dirFlags {
			if f.Long == dirFlag {
				if f.Type != flagDir {
					t.Errorf("flag --%s should be flagDir, got %v", f.Long, f.Type)
				}
			}
		}
	}
}

// ---------------------------------------------------------------------------
// TestGenerateCompletion - Bash script completeness
// ---------------------------------------------------------------------------

func TestGenerateCompletion_BashContainsAllCommands(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	err := GenerateCompletion(&buf, ShellBash)
	if err != nil {
		t.Fatalf("GenerateCompletion(&buf, ShellBash) unexpected error: %v", err)
	}

	output := buf.String()
	commands := getCommands()
	for _, cmd := range commands {
		if !strings.Contains(output, cmd.Name) {
			t.Errorf("bash completion missing command %q", cmd.Name)
		}
	}
}

// ---------------------------------------------------------------------------
// TestGenerateCompletion - Zsh script completeness
// ---------------------------------------------------------------------------

func TestGenerateCompletion_ZshContainsAllCommands(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	err := GenerateCompletion(&buf, ShellZsh)
	if err != nil {
		t.Fatalf("GenerateCompletion(&buf, ShellZsh) unexpected error: %v", err)
	}

	output := buf.String()
	commands := getCommands()
	for _, cmd := range commands {
		if !strings.Contains(output, cmd.Name) {
			t.Errorf("zsh completion missing command %q", cmd.Name)
		}
	}
}

// ---------------------------------------------------------------------------
// TestGenerateCompletion - Fish script completeness
// ---------------------------------------------------------------------------

func TestGenerateCompletion_FishContainsAllCommands(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	err := GenerateCompletion(&buf, ShellFish)
	if err != nil {
		t.Fatalf("GenerateCompletion(&buf, ShellFish) unexpected error: %v", err)
	}

	output := buf.String()
	commands := getCommands()
	for _, cmd := range commands {
		if !strings.Contains(output, cmd.Name) {
			t.Errorf("fish completion missing command %q", cmd.Name)
		}
	}
}

// ---------------------------------------------------------------------------
// TestGenerateCompletion - PowerShell completeness
// ---------------------------------------------------------------------------

func TestGenerateCompletion_PowerShellContainsAllCommands(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	err := GenerateCompletion(&buf, ShellPowerShell)
	if err != nil {
		t.Fatalf("GenerateCompletion(&buf, ShellPowerShell) unexpected error: %v", err)
	}

	output := buf.String()
	commands := getCommands()
	for _, cmd := range commands {
		if !strings.Contains(output, cmd.Name) {
			t.Errorf("powershell completion missing command %q", cmd.Name)
		}
	}
}

// ---------------------------------------------------------------------------
// TestGenerateCompletion - Zsh enum value completion
// ---------------------------------------------------------------------------

func TestGenerateCompletion_ZshEnumCompletion(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	err := GenerateCompletion(&buf, ShellZsh)
	if err != nil {
		t.Fatalf("GenerateCompletion(&buf, ShellZsh) unexpected error: %v", err)
	}

	output := buf.String()

	// Check enum values are present in completion
	enumValues := []string{"letter", "a4", "legal", "portrait", "landscape", "left", "center", "right"}
	for _, v := range enumValues {
		if !strings.Contains(output, v) {
			t.Errorf("zsh completion missing enum value %q", v)
		}
	}
}

// ---------------------------------------------------------------------------
// TestGenerateCompletion - Bash enum value completion
// ---------------------------------------------------------------------------

func TestGenerateCompletion_BashEnumCompletion(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	err := GenerateCompletion(&buf, ShellBash)
	if err != nil {
		t.Fatalf("GenerateCompletion(&buf, ShellBash) unexpected error: %v", err)
	}

	output := buf.String()

	// Check enum values are present in completion
	enumValues := []string{"letter", "a4", "legal", "portrait", "landscape", "left", "center", "right"}
	for _, v := range enumValues {
		if !strings.Contains(output, v) {
			t.Errorf("bash completion missing enum value %q", v)
		}
	}
}

// ---------------------------------------------------------------------------
// TestShellConstants - Shell type constants
// ---------------------------------------------------------------------------

func TestShellConstants(t *testing.T) {
	t.Parallel()

	// Verify shell constants have expected values
	tests := []struct {
		shell Shell
		want  string
	}{
		{ShellBash, "bash"},
		{ShellZsh, "zsh"},
		{ShellFish, "fish"},
		{ShellPowerShell, "powershell"},
	}

	for _, tt := range tests {
		if string(tt.shell) != tt.want {
			t.Errorf("Shell constant %v = %q, want %q", tt.shell, string(tt.shell), tt.want)
		}
	}
}

// ---------------------------------------------------------------------------
// TestPrintCompletionUsage - Completion usage help output
// ---------------------------------------------------------------------------

func TestPrintCompletionUsage(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	printCompletionUsage(&buf)

	output := buf.String()

	expectedContent := []string{
		"Usage: md2pdf completion",
		"bash",
		"zsh",
		"fish",
		"powershell",
		"Installation",
	}

	for _, expected := range expectedContent {
		if !strings.Contains(output, expected) {
			t.Errorf("completion usage missing %q", expected)
		}
	}
}
