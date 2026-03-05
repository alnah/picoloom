package main

// Notes:
// - Acceptance slice: command discovery, no-input generation, overwrite policy,
//   and non-TTY guardrail through `runMain`.
// - Unit slice: flag parsing, prompt behavior, yes/no parser, and output path
//   normalization.
// - Safety slice: rollback behavior, race protection, replace semantics, and
//   interrupted-backup recovery/cleanup.
// These are acceptable gaps: we test observable behavior, not implementation details.

import (
	"bufio"
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	md2pdf "github.com/alnah/go-md2pdf"
	"github.com/alnah/go-md2pdf/internal/config"
	"github.com/alnah/go-md2pdf/internal/yamlutil"
	flag "github.com/spf13/pflag"
)

// ---------------------------------------------------------------------------
// Test Infrastructure - acceptance helpers
// ---------------------------------------------------------------------------

func newAcceptanceEnv(t *testing.T) (*Environment, *bytes.Buffer, *bytes.Buffer) {
	t.Helper()

	loader, err := md2pdf.NewAssetLoader("")
	if err != nil {
		t.Fatalf("md2pdf.NewAssetLoader(\"\") unexpected error: %v", err)
	}

	var stdout, stderr bytes.Buffer
	env := &Environment{
		Now:         func() time.Time { return time.Now() },
		Stdin:       strings.NewReader(""),
		Stdout:      &stdout,
		Stderr:      &stderr,
		IsStdinTTY:  func() bool { return false },
		AssetLoader: loader,
		Config:      config.DefaultConfig(),
	}

	return env, &stdout, &stderr
}

func runInTempDir(t *testing.T) {
	t.Helper()

	t.Chdir(t.TempDir())
}

// ---------------------------------------------------------------------------
// TestConfigInitAcceptance_* - command acceptance behavior
// ---------------------------------------------------------------------------

func TestConfigInitAcceptance_CommandDiscovery(t *testing.T) {
	env, stdout, _ := newAcceptanceEnv(t)

	code := runMain([]string{"md2pdf", "help"}, env)
	if code != ExitSuccess {
		t.Fatalf("runMain([md2pdf help]) = %d, want %d", code, ExitSuccess)
	}

	if !strings.Contains(stdout.String(), "config") {
		t.Fatalf("help output = %q, want substring %q", stdout.String(), "config")
	}

	env2, stdout2, _ := newAcceptanceEnv(t)
	code = runMain([]string{"md2pdf", "help", "config"}, env2)
	if code != ExitSuccess {
		t.Fatalf("runMain([md2pdf help config]) = %d, want %d", code, ExitSuccess)
	}
	if !strings.Contains(strings.ToLower(stdout2.String()), "init") {
		t.Fatalf("help config output = %q, want substring %q", stdout2.String(), "init")
	}

	commands := getCommands()
	hasConfig := false
	for _, cmd := range commands {
		if cmd.Name == "config" {
			hasConfig = true
			break
		}
	}
	if !hasConfig {
		t.Fatalf("getCommands() missing config command")
	}
}

func TestConfigInitAcceptance_NoInputWritesDefaultConfig(t *testing.T) {
	runInTempDir(t)
	env, stdout, stderr := newAcceptanceEnv(t)

	code := runMain([]string{"md2pdf", "config", "init", "--no-input"}, env)
	if code != ExitSuccess {
		t.Fatalf("runMain([md2pdf config init --no-input]) = %d, want %d\nstderr: %s", code, ExitSuccess, stderr.String())
	}

	configPath := "./md2pdf.yaml"
	if _, err := os.Stat(configPath); err != nil {
		t.Fatalf("os.Stat(%q) unexpected error: %v", configPath, err)
	}

	if _, err := config.LoadConfig(configPath); err != nil {
		t.Fatalf("config.LoadConfig(%q) unexpected error: %v", configPath, err)
	}

	if !strings.Contains(stdout.String(), "md2pdf convert -c ./md2pdf.yaml") {
		t.Fatalf("stdout = %q, want usage example for convert with generated config", stdout.String())
	}
	if strings.Contains(strings.ToLower(stdout.String()), "preset") {
		t.Fatalf("stdout = %q, must not mention presets", stdout.String())
	}
}

func TestConfigInitAcceptance_OutputAndForce(t *testing.T) {
	runInTempDir(t)
	env, _, stderr := newAcceptanceEnv(t)

	outputPath := filepath.Join(".", "configs", "work.yaml")
	if err := os.MkdirAll(filepath.Dir(outputPath), 0o755); err != nil {
		t.Fatalf("os.MkdirAll(%q) unexpected error: %v", filepath.Dir(outputPath), err)
	}
	originalContent := []byte("document:\n  title: existing\n")
	if err := os.WriteFile(outputPath, originalContent, 0o644); err != nil {
		t.Fatalf("os.WriteFile(%q) unexpected error: %v", outputPath, err)
	}

	code := runMain([]string{"md2pdf", "config", "init", "--no-input", "--output", outputPath}, env)
	if code != ExitUsage {
		t.Fatalf("runMain([config init --no-input --output existing]) = %d, want %d", code, ExitUsage)
	}
	if !strings.Contains(stderr.String(), "--force") {
		t.Fatalf("stderr = %q, want overwrite guidance containing %q", stderr.String(), "--force")
	}

	gotContent, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("os.ReadFile(%q) unexpected error: %v", outputPath, err)
	}
	if string(gotContent) != string(originalContent) {
		t.Fatalf("existing file content changed without --force")
	}

	env2, _, stderr2 := newAcceptanceEnv(t)
	code = runMain([]string{"md2pdf", "config", "init", "--no-input", "--output", outputPath, "--force"}, env2)
	if code != ExitSuccess {
		t.Fatalf("runMain([config init --no-input --output --force]) = %d, want %d\nstderr: %s", code, ExitSuccess, stderr2.String())
	}
	if _, err := config.LoadConfig(outputPath); err != nil {
		t.Fatalf("config.LoadConfig(%q) unexpected error after --force overwrite: %v", outputPath, err)
	}
}

func TestConfigInitAcceptance_NonTTYGuardrail(t *testing.T) {
	runInTempDir(t)
	env, _, stderr := newAcceptanceEnv(t)

	code := runMain([]string{"md2pdf", "config", "init"}, env)
	if code != ExitUsage {
		t.Fatalf("runMain([md2pdf config init]) = %d, want %d", code, ExitUsage)
	}
	if !strings.Contains(stderr.String(), "--no-input") {
		t.Fatalf("stderr = %q, want guidance containing %q", stderr.String(), "--no-input")
	}
}

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

// ---------------------------------------------------------------------------
// testConfigInitYAML - valid generated YAML fixture
// ---------------------------------------------------------------------------

func testConfigInitYAML(t *testing.T) []byte {
	t.Helper()

	cfg := config.DefaultConfig()
	cfg.Style = "technical"
	cfg.Page.Size = "letter"
	cfg.Document.Date = "auto"

	data, err := yamlutil.Marshal(cfg)
	if err != nil {
		t.Fatalf("yamlutil.Marshal(default config) unexpected error: %v", err)
	}
	return data
}

// ---------------------------------------------------------------------------
// TestConfigInit_* - config init safety behavior
// ---------------------------------------------------------------------------

func TestConfigInit_ForceRollbackOnReplaceFailure(t *testing.T) {
	t.Chdir(t.TempDir())

	outputPath := filepath.Join(".", "md2pdf.yaml")
	original := []byte("document:\n  title: keep-me\n")
	if err := os.WriteFile(outputPath, original, 0o644); err != nil {
		t.Fatalf("os.WriteFile(%q) unexpected error: %v", outputPath, err)
	}

	ops := defaultConfigInitFileOps()
	realRename := ops.rename
	backupMoved := false
	replaceFailed := false
	ops.rename = func(oldPath, newPath string) error {
		if oldPath == outputPath && newPath != outputPath {
			backupMoved = true
		}
		if backupMoved && !replaceFailed && oldPath != outputPath && newPath == outputPath {
			replaceFailed = true
			return errors.New("simulated replace failure")
		}
		return realRename(oldPath, newPath)
	}

	err := writeConfigInitFileWithOps(outputPath, testConfigInitYAML(t), true, ops)
	if err == nil {
		t.Fatal("writeConfigInitFileWithOps(..., force=true) error = nil, want error")
	}

	got, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("os.ReadFile(%q) unexpected error: %v", outputPath, err)
	}
	if string(got) != string(original) {
		t.Fatalf("destination content changed despite rollback")
	}

	if _, err := os.Stat(configInitBackupPath(outputPath)); !os.IsNotExist(err) {
		t.Fatalf("backup file remains after rollback, stat error: %v", err)
	}
}

func TestConfigInit_NoForceRaceSafety(t *testing.T) {
	t.Chdir(t.TempDir())

	outputPath := filepath.Join(".", "md2pdf.yaml")
	concurrent := []byte("document:\n  title: concurrent-writer\n")

	ops := defaultConfigInitFileOps()
	ops.link = func(_, newPath string) error {
		if err := os.WriteFile(newPath, concurrent, 0o644); err != nil {
			return err
		}
		return os.ErrExist
	}

	err := writeConfigInitFileWithOps(outputPath, testConfigInitYAML(t), false, ops)
	if !errors.Is(err, ErrConfigInitExists) {
		t.Fatalf("writeConfigInitFileWithOps(..., force=false) error = %v, want ErrConfigInitExists", err)
	}

	got, readErr := os.ReadFile(outputPath)
	if readErr != nil {
		t.Fatalf("os.ReadFile(%q) unexpected error: %v", outputPath, readErr)
	}
	if string(got) != string(concurrent) {
		t.Fatalf("destination content overwritten in race path")
	}
}

func TestConfigInit_CrossPlatformReplaceSemantics(t *testing.T) {
	t.Chdir(t.TempDir())

	outputPath := filepath.Join(".", "md2pdf.yaml")
	original := []byte("document:\n  title: old-content\n")
	if err := os.WriteFile(outputPath, original, 0o644); err != nil {
		t.Fatalf("os.WriteFile(%q) unexpected error: %v", outputPath, err)
	}

	generated := testConfigInitYAML(t)
	if err := writeConfigInitFile(outputPath, generated, true); err != nil {
		t.Fatalf("writeConfigInitFile(..., force=true) unexpected error: %v", err)
	}

	got, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("os.ReadFile(%q) unexpected error: %v", outputPath, err)
	}
	if string(got) != string(generated) {
		t.Fatalf("destination content not replaced")
	}

	if _, err := config.LoadConfig("./md2pdf.yaml"); err != nil {
		t.Fatalf("config.LoadConfig(%q) unexpected error: %v", "./md2pdf.yaml", err)
	}
}

func TestConfigInit_RecoverInterruptedBackup(t *testing.T) {
	t.Chdir(t.TempDir())

	outputPath := filepath.Join(".", "md2pdf.yaml")
	backupPath := configInitBackupPath(outputPath)
	original := []byte("document:\n  title: recover\n")
	if err := os.WriteFile(backupPath, original, 0o644); err != nil {
		t.Fatalf("os.WriteFile(%q) unexpected error: %v", backupPath, err)
	}

	if err := recoverConfigInitBackup(outputPath, defaultConfigInitFileOps()); err != nil {
		t.Fatalf("recoverConfigInitBackup(...) unexpected error: %v", err)
	}

	got, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("os.ReadFile(%q) unexpected error: %v", outputPath, err)
	}
	if string(got) != string(original) {
		t.Fatalf("recovered output content mismatch")
	}
	if _, err := os.Stat(backupPath); !os.IsNotExist(err) {
		t.Fatalf("backup should not remain after recovery, stat error: %v", err)
	}
}

func TestConfigInit_CleanupStaleBackup(t *testing.T) {
	t.Chdir(t.TempDir())

	outputPath := filepath.Join(".", "md2pdf.yaml")
	backupPath := configInitBackupPath(outputPath)
	if err := os.WriteFile(outputPath, []byte("document:\n  title: active\n"), 0o644); err != nil {
		t.Fatalf("os.WriteFile(%q) unexpected error: %v", outputPath, err)
	}
	if err := os.WriteFile(backupPath, []byte("document:\n  title: stale\n"), 0o644); err != nil {
		t.Fatalf("os.WriteFile(%q) unexpected error: %v", backupPath, err)
	}

	if err := recoverConfigInitBackup(outputPath, defaultConfigInitFileOps()); err != nil {
		t.Fatalf("recoverConfigInitBackup(...) unexpected error: %v", err)
	}
	if _, err := os.Stat(backupPath); !os.IsNotExist(err) {
		t.Fatalf("stale backup should be removed, stat error: %v", err)
	}
}
