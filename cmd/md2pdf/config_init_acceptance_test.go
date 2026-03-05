package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	md2pdf "github.com/alnah/go-md2pdf"
	"github.com/alnah/go-md2pdf/internal/config"
)

func newAcceptanceEnv(t *testing.T) (*Environment, *bytes.Buffer, *bytes.Buffer) {
	t.Helper()

	loader, err := md2pdf.NewAssetLoader("")
	if err != nil {
		t.Fatalf("md2pdf.NewAssetLoader(\"\") unexpected error: %v", err)
	}

	var stdout, stderr bytes.Buffer
	env := &Environment{
		Now:         func() time.Time { return time.Now() },
		Stdout:      &stdout,
		Stderr:      &stderr,
		AssetLoader: loader,
		Config:      config.DefaultConfig(),
	}

	return env, &stdout, &stderr
}

func runInTempDir(t *testing.T) {
	t.Helper()

	t.Chdir(t.TempDir())
}

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

	configPath := filepath.Join(".", "md2pdf.yaml")
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
