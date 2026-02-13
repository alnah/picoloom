package main

// Notes:
// - resolveCSSContent: we test CSS loading from flag, config style name, and default.
// - printResultsOutput: we test success/failure counting (actual output formatting
//   is an implementation detail).
// - convertFile: we test error paths (read failure, write failure, mkdir failure).
//   Success paths are covered by integration tests.
// - loadTemplateSetFromDir: we test directory loading with complete/incomplete templates.
// - resolveTemplateSet: we test name vs path resolution.
// These are acceptable gaps: we test observable behavior, not implementation details.

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	md2pdf "github.com/alnah/go-md2pdf"
	"github.com/alnah/go-md2pdf/internal/config"
)

// ---------------------------------------------------------------------------
// TestResolveCSSContent - CSS content resolution
// ---------------------------------------------------------------------------

func TestResolveCSSContent(t *testing.T) {
	t.Parallel()

	loader, _ := md2pdf.NewAssetLoader("")

	t.Run("empty style and no config returns default style", func(t *testing.T) {
		t.Parallel()
		got, err := resolveCSSContent("", nil, false, loader)
		if err != nil {
			t.Fatalf("resolveCSSContent(\"\", nil, false, loader) unexpected error: %v", err)
		}
		if got == "" {
			t.Errorf("resolveCSSContent(\"\", nil, false, loader) = \"\", want default CSS content")
		}
		// Verify it's the default style (contains our default.css markers)
		if !strings.Contains(got, "Default theme") {
			t.Errorf("resolveCSSContent(\"\", nil, false, loader) missing \"Default theme\" marker")
		}
	})

	t.Run("CSS file path returns file content", func(t *testing.T) {
		t.Parallel()

		tempDir := t.TempDir()
		cssPath := filepath.Join(tempDir, "style.css")
		cssContent := "body { color: red; }"
		if err := os.WriteFile(cssPath, []byte(cssContent), 0644); err != nil {
			t.Fatalf("failed to write CSS file: %v", err)
		}

		got, err := resolveCSSContent(cssPath, nil, false, loader)
		if err != nil {
			t.Fatalf("resolveCSSContent(%q, nil, false, loader) unexpected error: %v", cssPath, err)
		}
		if got != cssContent {
			t.Errorf("resolveCSSContent(%q, nil, false, loader) = %q, want %q", cssPath, got, cssContent)
		}
	})

	t.Run("error case: nonexistent file", func(t *testing.T) {
		t.Parallel()

		_, err := resolveCSSContent("/nonexistent/style.css", nil, false, loader)
		if err == nil {
			t.Errorf("resolveCSSContent(\"/nonexistent/style.css\", nil, false, loader) error = nil, want error")
		}
	})

	t.Run("config style name loads from embedded assets", func(t *testing.T) {
		t.Parallel()

		cfg := &Config{Style: "creative"}
		got, err := resolveCSSContent("", cfg, false, loader)
		if err != nil {
			t.Fatalf("resolveCSSContent(\"\", cfg, false, loader) unexpected error: %v", err)
		}
		if got == "" {
			t.Errorf("resolveCSSContent(\"\", cfg, false, loader) = \"\", want embedded CSS content")
		}
	})

	t.Run("CSS file path overrides config style", func(t *testing.T) {
		t.Parallel()

		tempDir := t.TempDir()
		cssPath := filepath.Join(tempDir, "override.css")
		cssContent := "body { color: blue; }"
		if err := os.WriteFile(cssPath, []byte(cssContent), 0644); err != nil {
			t.Fatalf("failed to write CSS file: %v", err)
		}

		cfg := &Config{Style: "creative"}
		got, err := resolveCSSContent(cssPath, cfg, false, loader)
		if err != nil {
			t.Fatalf("resolveCSSContent(%q, cfg, false, loader) unexpected error: %v", cssPath, err)
		}
		if got != cssContent {
			t.Errorf("resolveCSSContent(%q, cfg, false, loader) = %q, want %q", cssPath, got, cssContent)
		}
	})

	t.Run("error case: unknown config style", func(t *testing.T) {
		t.Parallel()

		cfg := &Config{Style: "nonexistent"}
		_, err := resolveCSSContent("", cfg, false, loader)
		if err == nil {
			t.Errorf("resolveCSSContent(\"\", cfg, false, loader) error = nil, want error")
		}
	})

	t.Run("noStyle flag returns empty with config style", func(t *testing.T) {
		t.Parallel()

		cfg := &Config{Style: "creative"}
		got, err := resolveCSSContent("", cfg, true, loader)
		if err != nil {
			t.Fatalf("resolveCSSContent(\"\", cfg, true, loader) unexpected error: %v", err)
		}
		if got != "" {
			t.Errorf("resolveCSSContent(\"\", cfg, true, loader) = %q, want \"\"", got)
		}
	})

	t.Run("noStyle flag returns empty with CSS file path", func(t *testing.T) {
		t.Parallel()

		tempDir := t.TempDir()
		cssPath := filepath.Join(tempDir, "style.css")
		if err := os.WriteFile(cssPath, []byte("body { color: red; }"), 0644); err != nil {
			t.Fatalf("failed to write CSS file: %v", err)
		}

		got, err := resolveCSSContent(cssPath, nil, true, loader)
		if err != nil {
			t.Fatalf("resolveCSSContent(%q, nil, true, loader) unexpected error: %v", cssPath, err)
		}
		if got != "" {
			t.Errorf("resolveCSSContent(%q, nil, true, loader) = %q, want \"\"", cssPath, got)
		}
	})
}

// ---------------------------------------------------------------------------
// TestPrintResultsOutput - Conversion result counting
// ---------------------------------------------------------------------------

func TestPrintResultsOutput(t *testing.T) {
	t.Parallel()

	t.Run("all successful conversions", func(t *testing.T) {
		t.Parallel()
		results := []ConversionResult{
			{InputPath: "a.md", OutputPath: "a.pdf", Err: nil},
			{InputPath: "b.md", OutputPath: "b.pdf", Err: nil},
		}
		failed := printResults(results, true, false)
		if failed != 0 {
			t.Errorf("printResults(results, true, false) = %d, want 0", failed)
		}
	})

	t.Run("mixed success and failures", func(t *testing.T) {
		t.Parallel()

		results := []ConversionResult{
			{InputPath: "a.md", OutputPath: "a.pdf", Err: nil},
			{InputPath: "b.md", OutputPath: "b.pdf", Err: ErrReadMarkdown},
			{InputPath: "c.md", OutputPath: "c.pdf", Err: ErrReadMarkdown},
		}
		failed := printResults(results, true, false)
		if failed != 2 {
			t.Errorf("printResults(results, true, false) = %d, want 2", failed)
		}
	})

	t.Run("edge case: empty results", func(t *testing.T) {
		t.Parallel()

		failed := printResults(nil, true, false)
		if failed != 0 {
			t.Errorf("printResults(nil, true, false) = %d, want 0", failed)
		}
	})
}

// ---------------------------------------------------------------------------
// TestConvertFile_ErrorPaths - File conversion error handling
// ---------------------------------------------------------------------------

func TestConvertFile_ErrorPaths(t *testing.T) {
	t.Parallel()

	// Mock converter that returns success
	mockConv := &staticMockConverter{result: []byte("%PDF-1.4 mock")}

	t.Run("error case: mkdir failure", func(t *testing.T) {
		t.Parallel()

		tempDir := t.TempDir()

		// Create a file where directory should be (blocks mkdir)
		blockingFile := filepath.Join(tempDir, "blocked")
		if err := os.WriteFile(blockingFile, []byte("blocker"), 0644); err != nil {
			t.Fatalf("failed to create blocking file: %v", err)
		}

		// Create input file
		inputPath := filepath.Join(tempDir, "doc.md")
		if err := os.WriteFile(inputPath, []byte("# Test"), 0644); err != nil {
			t.Fatalf("failed to create input: %v", err)
		}

		// Try to output to a path under the blocking file (will fail mkdir)
		f := FileToConvert{
			InputPath:  inputPath,
			OutputPath: filepath.Join(blockingFile, "subdir", "out.pdf"),
		}

		result := convertFile(context.Background(), mockConv, f, &conversionParams{cfg: config.DefaultConfig()})

		if result.Err == nil {
			t.Errorf("convertFile(ctx, mockConv, f, params) error = nil, want error")
		}
	})

	t.Run("error case: write failure returns ErrWritePDF", func(t *testing.T) {
		t.Parallel()

		tempDir := t.TempDir()

		// Create input file
		inputPath := filepath.Join(tempDir, "doc.md")
		if err := os.WriteFile(inputPath, []byte("# Test"), 0644); err != nil {
			t.Fatalf("failed to create input: %v", err)
		}

		// Create output directory as read-only
		outDir := filepath.Join(tempDir, "readonly")
		if err := os.MkdirAll(outDir, 0750); err != nil {
			t.Fatalf("failed to create output dir: %v", err)
		}
		if err := os.Chmod(outDir, 0500); err != nil {
			t.Fatalf("failed to chmod: %v", err)
		}
		t.Cleanup(func() {
			os.Chmod(outDir, 0750) // Restore for cleanup
		})

		f := FileToConvert{
			InputPath:  inputPath,
			OutputPath: filepath.Join(outDir, "out.pdf"),
		}

		result := convertFile(context.Background(), mockConv, f, &conversionParams{cfg: config.DefaultConfig()})

		if result.Err == nil {
			t.Errorf("convertFile(ctx, mockConv, f, params) error = nil, want error")
		}
		if !errors.Is(result.Err, ErrWritePDF) {
			t.Errorf("convertFile(ctx, mockConv, f, params) error = %v, want ErrWritePDF", result.Err)
		}
	})

	t.Run("error case: read failure returns ErrReadMarkdown", func(t *testing.T) {
		t.Parallel()

		f := FileToConvert{
			InputPath:  "/nonexistent/doc.md",
			OutputPath: "/tmp/out.pdf",
		}

		result := convertFile(context.Background(), mockConv, f, &conversionParams{cfg: config.DefaultConfig()})

		if result.Err == nil {
			t.Errorf("convertFile(ctx, mockConv, f, params) error = nil, want error")
		}
		if !errors.Is(result.Err, ErrReadMarkdown) {
			t.Errorf("convertFile(ctx, mockConv, f, params) error = %v, want ErrReadMarkdown", result.Err)
		}
	})

	t.Run("read failure error message includes file path", func(t *testing.T) {
		t.Parallel()

		inputPath := "/nonexistent/specific/doc.md"
		f := FileToConvert{
			InputPath:  inputPath,
			OutputPath: "/tmp/out.pdf",
		}

		result := convertFile(context.Background(), mockConv, f, &conversionParams{cfg: config.DefaultConfig()})

		if result.Err == nil {
			t.Fatalf("convertFile(ctx, mockConv, f, params) error = nil, want error")
		}
		if !strings.Contains(result.Err.Error(), inputPath) {
			t.Errorf("convertFile(ctx, mockConv, f, params) error = %v, want error containing %q", result.Err, inputPath)
		}
	})

	t.Run("mkdir failure error message includes hint", func(t *testing.T) {
		t.Parallel()

		tempDir := t.TempDir()

		// Create a file where directory should be (blocks mkdir)
		blockingFile := filepath.Join(tempDir, "blocked")
		if err := os.WriteFile(blockingFile, []byte("blocker"), 0644); err != nil {
			t.Fatalf("failed to create blocking file: %v", err)
		}

		// Create input file
		inputPath := filepath.Join(tempDir, "doc.md")
		if err := os.WriteFile(inputPath, []byte("# Test"), 0644); err != nil {
			t.Fatalf("failed to create input: %v", err)
		}

		f := FileToConvert{
			InputPath:  inputPath,
			OutputPath: filepath.Join(blockingFile, "subdir", "out.pdf"),
		}

		result := convertFile(context.Background(), mockConv, f, &conversionParams{cfg: config.DefaultConfig()})

		if result.Err == nil {
			t.Fatalf("convertFile(ctx, mockConv, f, params) error = nil, want error")
		}
		errMsg := result.Err.Error()
		if !strings.Contains(errMsg, "hint:") {
			t.Errorf("convertFile(ctx, mockConv, f, params) error = %v, want error containing \"hint:\"", result.Err)
		}
		if !strings.Contains(errMsg, "parent directory") {
			t.Errorf("convertFile(ctx, mockConv, f, params) error = %v, want error containing \"parent directory\"", result.Err)
		}
	})
}

// ---------------------------------------------------------------------------
// TestConvertFile_SourceDir - SourceDir auto-setting from input path
// ---------------------------------------------------------------------------

func TestConvertFile_SourceDir(t *testing.T) {
	t.Parallel()

	t.Run("set to input file parent directory with nested path", func(t *testing.T) {
		t.Parallel()

		tempDir := t.TempDir()

		// Create a nested directory structure: tempDir/docs/report.md
		docsDir := filepath.Join(tempDir, "docs")
		if err := os.MkdirAll(docsDir, 0750); err != nil {
			t.Fatalf("failed to create docs dir: %v", err)
		}

		inputPath := filepath.Join(docsDir, "report.md")
		if err := os.WriteFile(inputPath, []byte("# Report\n![image](./images/logo.png)"), 0644); err != nil {
			t.Fatalf("failed to create input: %v", err)
		}

		// Use capturing mock to verify SourceDir
		mockConv := &capturingMockConverter{result: []byte("%PDF-1.4 mock")}

		f := FileToConvert{
			InputPath:  inputPath,
			OutputPath: filepath.Join(tempDir, "output.pdf"),
		}

		_ = convertFile(context.Background(), mockConv, f, &conversionParams{cfg: config.DefaultConfig()})

		// Verify SourceDir was set to the directory containing the input file
		expectedSourceDir := docsDir
		if mockConv.capturedIn.SourceDir != expectedSourceDir {
			t.Errorf("convertFile(ctx, mockConv, f, params) SourceDir = %q, want %q", mockConv.capturedIn.SourceDir, expectedSourceDir)
		}
	})

	t.Run("set to input file parent directory at root", func(t *testing.T) {
		t.Parallel()

		tempDir := t.TempDir()

		// Create file in temp root
		inputPath := filepath.Join(tempDir, "doc.md")
		if err := os.WriteFile(inputPath, []byte("# Doc"), 0644); err != nil {
			t.Fatalf("failed to create input: %v", err)
		}

		mockConv := &capturingMockConverter{result: []byte("%PDF-1.4 mock")}

		f := FileToConvert{
			InputPath:  inputPath,
			OutputPath: filepath.Join(tempDir, "doc.pdf"),
		}

		_ = convertFile(context.Background(), mockConv, f, &conversionParams{cfg: config.DefaultConfig()})

		// SourceDir should be tempDir (the parent directory of the input file)
		if mockConv.capturedIn.SourceDir != tempDir {
			t.Errorf("convertFile(ctx, mockConv, f, params) SourceDir = %q, want %q", mockConv.capturedIn.SourceDir, tempDir)
		}
	})
}

// ---------------------------------------------------------------------------
// TestHtmlOutputPath - HTML output path generation
// ---------------------------------------------------------------------------

func TestHtmlOutputPath(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		pdfPath string
		want    string
	}{
		{
			name:    "simple pdf extension",
			pdfPath: "output.pdf",
			want:    "output.html",
		},
		{
			name:    "absolute path with pdf extension",
			pdfPath: "/path/to/doc.pdf",
			want:    "/path/to/doc.html",
		},
		{
			name:    "no pdf extension",
			pdfPath: "file",
			want:    "file.html",
		},
		{
			name:    "uppercase PDF not replaced (case-sensitive)",
			pdfPath: "doc.PDF",
			want:    "doc.PDF.html",
		},
		{
			name:    "multiple dots in filename",
			pdfPath: "my.report.v2.pdf",
			want:    "my.report.v2.html",
		},
		{
			name:    "Windows path",
			pdfPath: "C:\\Documents\\report.pdf",
			want:    "C:\\Documents\\report.html",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := htmlOutputPath(tt.pdfPath)
			if got != tt.want {
				t.Errorf("htmlOutputPath(%q) = %q, want %q", tt.pdfPath, got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestLoadTemplateSetFromDir - Template set loading from filesystem
// ---------------------------------------------------------------------------

func TestLoadTemplateSetFromDir(t *testing.T) {
	t.Parallel()

	t.Run("happy path: loads both templates", func(t *testing.T) {
		t.Parallel()

		tmpDir := t.TempDir()
		coverContent := "<div class=\"cover\">Cover Page</div>"
		sigContent := "<div class=\"signature\">Signature Block</div>"

		if err := os.WriteFile(filepath.Join(tmpDir, "cover.html"), []byte(coverContent), 0644); err != nil {
			t.Fatalf("failed to write cover.html: %v", err)
		}
		if err := os.WriteFile(filepath.Join(tmpDir, "signature.html"), []byte(sigContent), 0644); err != nil {
			t.Fatalf("failed to write signature.html: %v", err)
		}

		ts, err := loadTemplateSetFromDir(tmpDir)
		if err != nil {
			t.Fatalf("loadTemplateSetFromDir(%q) unexpected error: %v", tmpDir, err)
		}

		if ts.Cover != coverContent {
			t.Errorf("loadTemplateSetFromDir(%q) Cover = %q, want %q", tmpDir, ts.Cover, coverContent)
		}
		if ts.Signature != sigContent {
			t.Errorf("loadTemplateSetFromDir(%q) Signature = %q, want %q", tmpDir, ts.Signature, sigContent)
		}
		if ts.Name != tmpDir {
			t.Errorf("loadTemplateSetFromDir(%q) Name = %q, want %q", tmpDir, ts.Name, tmpDir)
		}
	})

	t.Run("error case: missing cover.html", func(t *testing.T) {
		t.Parallel()

		tmpDir := t.TempDir()
		if err := os.WriteFile(filepath.Join(tmpDir, "signature.html"), []byte("<sig/>"), 0644); err != nil {
			t.Fatalf("failed to write signature.html: %v", err)
		}

		_, err := loadTemplateSetFromDir(tmpDir)
		if !errors.Is(err, md2pdf.ErrIncompleteTemplateSet) {
			t.Errorf("loadTemplateSetFromDir(%q) error = %v, want ErrIncompleteTemplateSet", tmpDir, err)
		}
	})

	t.Run("error case: missing signature.html", func(t *testing.T) {
		t.Parallel()

		tmpDir := t.TempDir()
		if err := os.WriteFile(filepath.Join(tmpDir, "cover.html"), []byte("<cover/>"), 0644); err != nil {
			t.Fatalf("failed to write cover.html: %v", err)
		}

		_, err := loadTemplateSetFromDir(tmpDir)
		if !errors.Is(err, md2pdf.ErrIncompleteTemplateSet) {
			t.Errorf("loadTemplateSetFromDir(%q) error = %v, want ErrIncompleteTemplateSet", tmpDir, err)
		}
	})

	t.Run("error case: empty directory", func(t *testing.T) {
		t.Parallel()

		tmpDir := t.TempDir()

		_, err := loadTemplateSetFromDir(tmpDir)
		if !errors.Is(err, md2pdf.ErrTemplateSetNotFound) {
			t.Errorf("loadTemplateSetFromDir(%q) error = %v, want ErrTemplateSetNotFound", tmpDir, err)
		}
	})

	t.Run("error case: nonexistent directory", func(t *testing.T) {
		t.Parallel()

		_, err := loadTemplateSetFromDir("/nonexistent/path/to/templates")
		if !errors.Is(err, md2pdf.ErrTemplateSetNotFound) {
			t.Errorf("loadTemplateSetFromDir(\"/nonexistent/path/to/templates\") error = %v, want ErrTemplateSetNotFound", err)
		}
	})
}

// ---------------------------------------------------------------------------
// TestResolveTemplateSet - Template set resolution by name or path
// ---------------------------------------------------------------------------

func TestResolveTemplateSet(t *testing.T) {
	t.Parallel()

	t.Run("empty value loads default template set", func(t *testing.T) {
		t.Parallel()

		loader := &mockTemplateLoader{
			templateSets: map[string]*md2pdf.TemplateSet{
				"default": {Name: "default", Cover: "<cover/>", Signature: "<sig/>"},
			},
		}

		ts, err := resolveTemplateSet("", loader)
		if err != nil {
			t.Fatalf("resolveTemplateSet(\"\", loader) unexpected error: %v", err)
		}
		if ts.Name != "default" {
			t.Errorf("resolveTemplateSet(\"\", loader) Name = %q, want %q", ts.Name, "default")
		}
	})

	t.Run("template set name loads from loader", func(t *testing.T) {
		t.Parallel()

		loader := &mockTemplateLoader{
			templateSets: map[string]*md2pdf.TemplateSet{
				"corporate": {Name: "corporate", Cover: "<corp-cover/>", Signature: "<corp-sig/>"},
			},
		}

		ts, err := resolveTemplateSet("corporate", loader)
		if err != nil {
			t.Fatalf("resolveTemplateSet(\"corporate\", loader) unexpected error: %v", err)
		}
		if ts.Name != "corporate" {
			t.Errorf("resolveTemplateSet(\"corporate\", loader) Name = %q, want %q", ts.Name, "corporate")
		}
	})

	t.Run("directory path loads from filesystem", func(t *testing.T) {
		t.Parallel()

		tmpDir := t.TempDir()
		if err := os.WriteFile(filepath.Join(tmpDir, "cover.html"), []byte("<cover/>"), 0644); err != nil {
			t.Fatalf("failed to write cover.html: %v", err)
		}
		if err := os.WriteFile(filepath.Join(tmpDir, "signature.html"), []byte("<sig/>"), 0644); err != nil {
			t.Fatalf("failed to write signature.html: %v", err)
		}

		// Use path-like value (contains /)
		pathValue := tmpDir + "/"

		loader := &mockTemplateLoader{} // Should not be called for paths

		ts, err := resolveTemplateSet(pathValue, loader)
		if err != nil {
			t.Fatalf("resolveTemplateSet(%q, loader) unexpected error: %v", pathValue, err)
		}
		if ts.Cover != "<cover/>" {
			t.Errorf("resolveTemplateSet(%q, loader) Cover = %q, want %q", pathValue, ts.Cover, "<cover/>")
		}
	})

	t.Run("error case: nonexistent template set name", func(t *testing.T) {
		t.Parallel()

		loader := &mockTemplateLoader{
			templateSets: map[string]*md2pdf.TemplateSet{},
		}

		_, err := resolveTemplateSet("nonexistent", loader)
		if !errors.Is(err, md2pdf.ErrTemplateSetNotFound) {
			t.Errorf("resolveTemplateSet(\"nonexistent\", loader) error = %v, want ErrTemplateSetNotFound", err)
		}
	})
}
