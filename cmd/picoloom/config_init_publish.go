package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/alnah/picoloom/v2/internal/config"
)

const (
	// configInitBackupSuffix marks temporary backup files used for safe overwrite recovery.
	configInitBackupSuffix = ".picoloom-config-init.bak"
	// configInitLockSuffix marks destination-scoped lock files to prevent concurrent writes.
	configInitLockSuffix = ".picoloom-config-init.lock"
)

// configInitFileOps abstracts filesystem side effects so safety behavior can be
// tested deterministically across platforms and failure modes.
type configInitFileOps struct {
	stat     func(string) (os.FileInfo, error)
	mkdirAll func(string, os.FileMode) error
	create   func(string, string) (*os.File, error)
	rename   func(string, string) error
	remove   func(string) error
	link     func(string, string) error
	openFile func(string, int, os.FileMode) (*os.File, error)
	readFile func(string) ([]byte, error)
}

// defaultConfigInitFileOps binds file operations to the real OS implementation.
func defaultConfigInitFileOps() configInitFileOps {
	return configInitFileOps{
		stat:     os.Stat,
		mkdirAll: os.MkdirAll,
		create:   os.CreateTemp,
		rename:   os.Rename,
		remove:   os.Remove,
		link:     os.Link,
		openFile: os.OpenFile,
		readFile: os.ReadFile,
	}
}

// writeConfigInitFile is the production entry point for safe config publishing.
func writeConfigInitFile(outputPath string, data []byte, force bool) error {
	return writeConfigInitFileWithOps(outputPath, data, force, defaultConfigInitFileOps())
}

// writeConfigInitFileWithOps enforces atomic-ish write invariants (lock, temp
// file, validation, publish strategy) to prevent partial or conflicting writes.
func writeConfigInitFileWithOps(outputPath string, data []byte, force bool, ops configInitFileOps) (retErr error) {
	if strings.TrimSpace(outputPath) == "" {
		return fmt.Errorf("%w: output path cannot be empty", ErrConfigCommandUsage)
	}

	dir := filepath.Dir(outputPath)
	if err := ops.mkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("creating destination directory: %w", err)
	}
	lockPath := configInitLockPath(outputPath)
	lockFile, err := ops.openFile(lockPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		if os.IsExist(err) {
			return fmt.Errorf("%w: %s (remove stale lock if needed: %s)", ErrConfigInitBusy, outputPath, lockPath)
		}
		return fmt.Errorf("acquiring destination lock: %w", err)
	}
	if err := lockFile.Close(); err != nil {
		_ = ops.remove(lockPath)
		return fmt.Errorf("closing destination lock: %w", err)
	}
	defer func() {
		if err := ops.remove(lockPath); err != nil && !os.IsNotExist(err) && retErr == nil {
			retErr = fmt.Errorf("releasing destination lock: %w", err)
		}
	}()

	if err := recoverConfigInitBackup(outputPath, ops); err != nil {
		return err
	}

	tmpFile, err := ops.create(dir, ".picoloom-config-init-*.yaml")
	if err != nil {
		return fmt.Errorf("creating temp config file: %w", err)
	}

	tmpPath := tmpFile.Name()
	defer func() {
		_ = tmpFile.Close()
		if retErr != nil {
			_ = ops.remove(tmpPath)
		}
	}()

	if _, err := tmpFile.Write(data); err != nil {
		return fmt.Errorf("writing temp config file: %w", err)
	}
	if err := tmpFile.Sync(); err != nil {
		return fmt.Errorf("syncing temp config file: %w", err)
	}
	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("closing temp config file: %w", err)
	}

	if _, err := config.LoadConfig(tmpPath); err != nil {
		return fmt.Errorf("validating generated config: %w", err)
	}

	if force {
		return publishConfigForce(tmpPath, outputPath, ops)
	}

	return publishConfigNoForce(tmpPath, outputPath, ops)
}

// configInitBackupPath keeps backup naming deterministic for recovery logic.
func configInitBackupPath(outputPath string) string {
	return outputPath + configInitBackupSuffix
}

// configInitLockPath scopes lock files to destination path to serialize writers.
func configInitLockPath(outputPath string) string {
	return outputPath + configInitLockSuffix
}

// recoverConfigInitBackup repairs interrupted overwrite states before any new
// write attempt, so destination semantics remain predictable.
func recoverConfigInitBackup(outputPath string, ops configInitFileOps) error {
	_, outputErr := ops.stat(outputPath)
	if outputErr != nil && !os.IsNotExist(outputErr) {
		return fmt.Errorf("checking destination config file: %w", outputErr)
	}

	backupPath := configInitBackupPath(outputPath)
	_, backupErr := ops.stat(backupPath)
	if backupErr != nil && !os.IsNotExist(backupErr) {
		return fmt.Errorf("checking backup config file: %w", backupErr)
	}
	if os.IsNotExist(backupErr) {
		return nil
	}

	if os.IsNotExist(outputErr) {
		if err := ops.rename(backupPath, outputPath); err != nil {
			return fmt.Errorf("restoring interrupted overwrite: %w", err)
		}
		return nil
	}

	if err := ops.remove(backupPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("cleaning stale backup file: %w", err)
	}
	return nil
}

// publishConfigNoForce guarantees no-clobber semantics unless user explicitly
// opted into overwrite via --force.
func publishConfigNoForce(tmpPath, outputPath string, ops configInitFileOps) error {
	if err := ops.link(tmpPath, outputPath); err == nil {
		if err := ops.remove(tmpPath); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("removing temporary config after publish: %w", err)
		}
		return nil
	} else if os.IsExist(err) {
		return fmt.Errorf("%w: %s (use --force)", ErrConfigInitExists, outputPath)
	}

	return copyTempToExclusiveFile(tmpPath, outputPath, ops)
}

// copyTempToExclusiveFile is the portability fallback when hard-link publish is
// unavailable, while preserving exclusive-create guarantees.
func copyTempToExclusiveFile(tmpPath, outputPath string, ops configInitFileOps) (retErr error) {
	out, err := ops.openFile(outputPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		if os.IsExist(err) {
			return fmt.Errorf("%w: %s (use --force)", ErrConfigInitExists, outputPath)
		}
		return fmt.Errorf("creating destination config file: %w", err)
	}

	defer func() {
		_ = out.Close()
		if retErr != nil {
			_ = ops.remove(outputPath)
		}
	}()

	if err := writeTempConfigPayload(tmpPath, out, ops); err != nil {
		return err
	}
	if err := out.Sync(); err != nil {
		return fmt.Errorf("syncing destination config file: %w", err)
	}
	if err := out.Close(); err != nil {
		return fmt.Errorf("closing destination config file: %w", err)
	}

	if err := ops.remove(tmpPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("removing temporary config after publish: %w", err)
	}

	return nil
}

// writeTempConfigPayload picks the lowest-overhead transfer path while keeping
// fallback copy semantics identical across filesystems.
func writeTempConfigPayload(tmpPath string, out *os.File, ops configInitFileOps) error {
	const smallPayloadThreshold = 64 * 1024

	info, statErr := ops.stat(tmpPath)
	if statErr == nil && info.Size() <= smallPayloadThreshold {
		return writeTempConfigPayloadFromMemory(tmpPath, out, ops)
	}

	return writeTempConfigPayloadByStream(tmpPath, out, ops)
}

// writeTempConfigPayloadFromMemory avoids second file descriptor churn for very
// small payloads where a single read+write is simpler.
func writeTempConfigPayloadFromMemory(tmpPath string, out *os.File, ops configInitFileOps) error {
	content, readErr := ops.readFile(tmpPath)
	if readErr != nil {
		return fmt.Errorf("reading temp config file for exclusive publish: %w", readErr)
	}
	if _, writeErr := out.Write(content); writeErr != nil {
		return fmt.Errorf("writing destination config file: %w", writeErr)
	}
	return nil
}

// writeTempConfigPayloadByStream handles larger files and stat fallbacks without
// buffering entire payloads into memory.
func writeTempConfigPayloadByStream(tmpPath string, out *os.File, ops configInitFileOps) error {
	in, openErr := ops.openFile(tmpPath, os.O_RDONLY, 0)
	if openErr != nil {
		return fmt.Errorf("opening temp config file for exclusive publish: %w", openErr)
	}
	defer func() {
		_ = in.Close()
	}()

	if _, copyErr := io.Copy(out, in); copyErr != nil {
		return fmt.Errorf("writing destination config file: %w", copyErr)
	}

	return nil
}

// publishConfigForce implements explicit overwrite with rollback protection so
// failures do not destroy the previous config.
func publishConfigForce(tmpPath, outputPath string, ops configInitFileOps) error {
	if _, err := ops.stat(outputPath); err != nil {
		if !os.IsNotExist(err) {
			return fmt.Errorf("checking destination config file: %w", err)
		}
		if err := ops.rename(tmpPath, outputPath); err != nil {
			return fmt.Errorf("moving generated config into place: %w", err)
		}
		return nil
	}

	backupPath := configInitBackupPath(outputPath)
	if err := ops.remove(backupPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("cleaning stale backup file: %w", err)
	}
	if err := ops.rename(outputPath, backupPath); err != nil {
		return fmt.Errorf("preparing safe overwrite: %w", err)
	}

	if err := ops.rename(tmpPath, outputPath); err != nil {
		restoreErr := ops.rename(backupPath, outputPath)
		if restoreErr != nil {
			return fmt.Errorf("overwriting config failed: %w; rollback failed: %w", err, restoreErr)
		}
		return fmt.Errorf("overwriting config failed, restored previous file: %w", err)
	}

	if err := ops.remove(backupPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("cleaning backup file: %w", err)
	}

	return nil
}
