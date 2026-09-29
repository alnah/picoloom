package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func benchmarkCopyTempToExclusiveFile(b *testing.B, size int) {
	b.Helper()

	dir := b.TempDir()
	payload := bytes.Repeat([]byte("a"), size)
	ops := defaultConfigInitFileOps()

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		tmpPath := filepath.Join(dir, fmt.Sprintf("tmp-%d.yaml", i))
		outputPath := filepath.Join(dir, fmt.Sprintf("out-%d.yaml", i))

		b.StopTimer()
		if err := os.WriteFile(tmpPath, payload, 0o644); err != nil {
			b.Fatalf("os.WriteFile(%q) unexpected error: %v", tmpPath, err)
		}
		b.StartTimer()

		if err := copyTempToExclusiveFile(tmpPath, outputPath, ops); err != nil {
			b.Fatalf("copyTempToExclusiveFile(%q, %q) unexpected error: %v", tmpPath, outputPath, err)
		}

		b.StopTimer()
		if err := os.Remove(outputPath); err != nil {
			b.Fatalf("os.Remove(%q) unexpected error: %v", outputPath, err)
		}
	}
}

func BenchmarkCopyTempToExclusiveFile_4KB(b *testing.B) {
	benchmarkCopyTempToExclusiveFile(b, 4*1024)
}

func BenchmarkCopyTempToExclusiveFile_1MB(b *testing.B) {
	benchmarkCopyTempToExclusiveFile(b, 1024*1024)
}

func BenchmarkValidateWizardStyle_Valid(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if err := validateWizardStyle("technical"); err != nil {
			b.Fatalf("validateWizardStyle(\"technical\") unexpected error: %v", err)
		}
	}
}

func BenchmarkValidateWizardStyle_Invalid(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = validateWizardStyle("unknown-style")
	}
}

func BenchmarkWizardStyleOptions(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = wizardStyleOptions()
	}
}
