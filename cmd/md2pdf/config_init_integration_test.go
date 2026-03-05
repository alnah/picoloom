//go:build integration

package main

// Notes:
// - runMain config init no-input mode: we test end-to-end file creation for
//   default behavior with a custom output path.
// - overwrite policy: we test existing-file preservation without --force and
//   replacement with --force.
// - validation boundary: we verify generated output reloads with config.LoadConfig.
// These are acceptable gaps: we test CLI/file invariants, not prompt internals.

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/alnah/go-md2pdf/internal/config"
)

// ---------------------------------------------------------------------------
// TestIntegration_ConfigInit_NoInputCustomOutput - custom output generation
// ---------------------------------------------------------------------------

func TestIntegration_ConfigInit_NoInputCustomOutput(t *testing.T) {
	t.Chdir(t.TempDir())

	env := DefaultEnv()
	outputPath := filepath.Join(".", "configs", "work.yaml")

	code := runMain([]string{"md2pdf", "config", "init", "--no-input", "--output", outputPath}, env)
	if code != ExitSuccess {
		t.Fatalf("runMain([config init --no-input --output]) = %d, want %d", code, ExitSuccess)
	}

	if _, err := os.Stat(outputPath); err != nil {
		t.Fatalf("os.Stat(%q) unexpected error: %v", outputPath, err)
	}
	if _, err := config.LoadConfig(outputPath); err != nil {
		t.Fatalf("config.LoadConfig(%q) unexpected error: %v", outputPath, err)
	}
}

// ---------------------------------------------------------------------------
// TestIntegration_ConfigInit_NoForceKeepsExisting - no-force protection
// ---------------------------------------------------------------------------

func TestIntegration_ConfigInit_NoForceKeepsExisting(t *testing.T) {
	t.Chdir(t.TempDir())

	env := DefaultEnv()
	outputPath := filepath.Join(".", "md2pdf.yaml")
	original := []byte("document:\n  title: keep\n")
	if err := os.WriteFile(outputPath, original, 0o644); err != nil {
		t.Fatalf("os.WriteFile(%q) unexpected error: %v", outputPath, err)
	}

	code := runMain([]string{"md2pdf", "config", "init", "--no-input", "--output", outputPath}, env)
	if code != ExitUsage {
		t.Fatalf("runMain([config init --no-input --output existing]) = %d, want %d", code, ExitUsage)
	}

	got, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("os.ReadFile(%q) unexpected error: %v", outputPath, err)
	}
	if string(got) != string(original) {
		t.Fatalf("existing file was modified without --force")
	}
}

// ---------------------------------------------------------------------------
// TestIntegration_ConfigInit_ForceReplacesExisting - force overwrite path
// ---------------------------------------------------------------------------

func TestIntegration_ConfigInit_ForceReplacesExisting(t *testing.T) {
	t.Chdir(t.TempDir())

	env := DefaultEnv()
	outputPath := filepath.Join(".", "md2pdf.yaml")
	original := []byte("document:\n  title: old\n")
	if err := os.WriteFile(outputPath, original, 0o644); err != nil {
		t.Fatalf("os.WriteFile(%q) unexpected error: %v", outputPath, err)
	}

	code := runMain([]string{"md2pdf", "config", "init", "--no-input", "--output", outputPath, "--force"}, env)
	if code != ExitSuccess {
		t.Fatalf("runMain([config init --no-input --output --force]) = %d, want %d", code, ExitSuccess)
	}

	got, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("os.ReadFile(%q) unexpected error: %v", outputPath, err)
	}
	if string(got) == string(original) {
		t.Fatalf("existing file was not replaced with --force")
	}
	if _, err := config.LoadConfig("./md2pdf.yaml"); err != nil {
		t.Fatalf("config.LoadConfig(%q) unexpected error: %v", "./md2pdf.yaml", err)
	}
}
