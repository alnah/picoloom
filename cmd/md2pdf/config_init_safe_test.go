package main

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/alnah/go-md2pdf/internal/config"
	"github.com/alnah/go-md2pdf/internal/yamlutil"
)

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

	if backups, err := filepath.Glob(outputPath + ".bak.*"); err != nil {
		t.Fatalf("filepath.Glob backup pattern unexpected error: %v", err)
	} else if len(backups) != 0 {
		t.Fatalf("backup files remain after rollback: %v", backups)
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
