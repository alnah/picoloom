package main

// Notes:
// - testConfigInitYAML: we build a valid generated payload used by safety tests.
// - force rollback: we inject a replace failure after backup move and verify
//   previous content restoration and backup cleanup.
// - no-force race safety: we simulate a concurrent writer and assert no overwrite.
// - cross-platform replace semantics: we assert force replace yields valid config.
// - backup recovery: we test interrupted force-overwrite backup restoration.
// These are acceptable gaps: we assert safety invariants, not syscall order.

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/alnah/go-md2pdf/internal/config"
	"github.com/alnah/go-md2pdf/internal/yamlutil"
)

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
// TestConfigInit_ForceRollbackOnReplaceFailure - rollback safety on failure
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

// ---------------------------------------------------------------------------
// TestConfigInit_NoForceRaceSafety - no-force TOCTOU protection
// ---------------------------------------------------------------------------

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

// ---------------------------------------------------------------------------
// TestConfigInit_CrossPlatformReplaceSemantics - replace guarantees
// ---------------------------------------------------------------------------

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

// ---------------------------------------------------------------------------
// TestConfigInit_RecoverInterruptedBackup - restore backup-only state
// ---------------------------------------------------------------------------

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

// ---------------------------------------------------------------------------
// TestConfigInit_CleanupStaleBackup - cleanup backup when output exists
// ---------------------------------------------------------------------------

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
