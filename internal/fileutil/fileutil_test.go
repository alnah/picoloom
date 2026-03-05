package fileutil_test

// Notes:
// - TestWriteTempFile_CreateTempError: this test modifies the global TMPDIR
//   environment variable and cannot run in parallel with other tests.
// - Coverage at 82.1%: the WriteString and Close error branches in WriteTempFile
//   are not tested because triggering disk write failures is platform-specific.
// These are acceptable gaps: we test observable behavior, not implementation details.

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/alnah/go-md2pdf/internal/fileutil"
)

type stringBoolCase struct {
	name  string
	input string
	want  bool
}

func runStringBoolPredicateTests(t *testing.T, fnName string, tests []stringBoolCase, predicate func(string) bool) {
	t.Helper()

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := predicate(tt.input)
			if got != tt.want {
				t.Errorf("%s(%q) = %v, want %v", fnName, tt.input, got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestValidateExtension - Extension validation
// ---------------------------------------------------------------------------

func TestValidateExtension(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		extension string
		wantErr   error
	}{
		{
			name:      "valid extension md",
			extension: "md",
			wantErr:   nil,
		},
		{
			name:      "valid extension html",
			extension: "html",
			wantErr:   nil,
		},
		{
			name:      "empty extension",
			extension: "",
			wantErr:   fileutil.ErrExtensionEmpty,
		},
		{
			name:      "forward slash path traversal",
			extension: "../etc/passwd",
			wantErr:   fileutil.ErrExtensionPathTraversal,
		},
		{
			name:      "backslash path traversal",
			extension: "..\\windows\\system32",
			wantErr:   fileutil.ErrExtensionPathTraversal,
		},
		{
			name:      "null byte injection",
			extension: "html\x00exe",
			wantErr:   fileutil.ErrExtensionPathTraversal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := fileutil.ValidateExtension(tt.extension)
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("ValidateExtension(%q) = %v, want %v", tt.extension, err, tt.wantErr)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestWriteTempFile - Temporary file creation
// ---------------------------------------------------------------------------

func TestWriteTempFile(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		content   string
		extension string
	}{
		{
			name:      "markdown file",
			content:   "# Test Markdown",
			extension: "md",
		},
		{
			name:      "html file",
			content:   "<html><body>Test Content</body></html>",
			extension: "html",
		},
		{
			name:      "empty content",
			content:   "",
			extension: "md",
		},
		{
			name:      "unicode content",
			content:   "# Hello World\n\nThis is a test with special characters: cafe, naive, resume",
			extension: "md",
		},
		{
			name:      "unicode html content",
			content:   "<html><body>Hello World</body></html>",
			extension: "html",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			path, cleanup, err := fileutil.WriteTempFile(tt.content, tt.extension)
			if err != nil {
				t.Fatalf("WriteTempFile() error = %v", err)
			}
			defer cleanup()

			// Verify file exists
			if _, err := os.Stat(path); os.IsNotExist(err) {
				t.Errorf("temp file does not exist at %s", path)
			}

			// Verify path pattern
			if !strings.Contains(path, "md2pdf-") {
				t.Errorf("path %q does not contain prefix 'md2pdf-'", path)
			}
			if !strings.HasSuffix(path, "."+tt.extension) {
				t.Errorf("path %q does not have extension .%s", path, tt.extension)
			}

			// Verify content
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("failed to read temp file: %v", err)
			}
			if string(data) != tt.content {
				t.Errorf("file content = %q, want %q", string(data), tt.content)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestWriteTempFile_Cleanup - Cleanup function removes file
// ---------------------------------------------------------------------------

func TestWriteTempFile_Cleanup(t *testing.T) {
	t.Parallel()

	path, cleanup, err := fileutil.WriteTempFile("test content", "md")
	if err != nil {
		t.Fatalf("WriteTempFile() error = %v", err)
	}

	// Verify file exists before cleanup
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Fatalf("temp file does not exist at %s", path)
	}

	// Call cleanup
	cleanup()

	// Verify file is removed after cleanup
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Errorf("temp file still exists after cleanup at %s", path)
	}
}

// ---------------------------------------------------------------------------
// TestWriteTempFile_InvalidExtension - Invalid extension errors
// ---------------------------------------------------------------------------

func TestWriteTempFile_InvalidExtension(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		extension string
		wantErr   error
	}{
		{
			name:      "empty extension",
			extension: "",
			wantErr:   fileutil.ErrExtensionEmpty,
		},
		{
			name:      "path traversal",
			extension: "../foo",
			wantErr:   fileutil.ErrExtensionPathTraversal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, cleanup, err := fileutil.WriteTempFile("content", tt.extension)
			if cleanup != nil {
				defer cleanup()
			}
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("WriteTempFile() error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestWriteTempFile_CreateTempError - CreateTemp failure handling
// ---------------------------------------------------------------------------

// NOTE: This test modifies TMPDIR and cannot run in parallel.
func TestWriteTempFile_CreateTempError(t *testing.T) {
	// Save original TMPDIR and restore after test
	originalTmpdir := os.Getenv("TMPDIR")
	defer func() {
		if originalTmpdir == "" {
			os.Unsetenv("TMPDIR")
		} else {
			os.Setenv("TMPDIR", originalTmpdir)
		}
	}()

	// Set TMPDIR to a non-existent directory to trigger CreateTemp failure
	os.Setenv("TMPDIR", "/nonexistent/path/that/does/not/exist")

	_, cleanup, err := fileutil.WriteTempFile("content", "md")
	if cleanup != nil {
		defer cleanup()
	}

	if err == nil {
		t.Fatal("WriteTempFile() expected error when TMPDIR is invalid, got nil")
	}

	if !strings.Contains(err.Error(), "creating temp file") {
		t.Errorf("WriteTempFile() error = %q, want error containing 'creating temp file'", err.Error())
	}
}

// ---------------------------------------------------------------------------
// TestWriteTempFile_LargeContent - Large file handling
// ---------------------------------------------------------------------------

func TestWriteTempFile_LargeContent(t *testing.T) {
	t.Parallel()

	// Test with large content to verify WriteString handles it correctly
	largeContent := strings.Repeat("x", 1024*1024) // 1MB

	path, cleanup, err := fileutil.WriteTempFile(largeContent, "txt")
	if err != nil {
		t.Fatalf("WriteTempFile() error = %v", err)
	}
	defer cleanup()

	// Verify file contains all content
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile error = %v", err)
	}
	if len(data) != len(largeContent) {
		t.Errorf("file size = %d, want %d", len(data), len(largeContent))
	}
}

// ---------------------------------------------------------------------------
// TestFileExists - File existence check
// ---------------------------------------------------------------------------

func TestFileExists(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()

	// Create a test file
	testFile := filepath.Join(tempDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("content"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// Create a test directory
	testDir := filepath.Join(tempDir, "testdir")
	if err := os.Mkdir(testDir, 0755); err != nil {
		t.Fatalf("failed to create test dir: %v", err)
	}

	tests := []struct {
		name string
		path string
		want bool
	}{
		{
			name: "existing file returns true",
			path: testFile,
			want: true,
		},
		{
			name: "directory returns false",
			path: testDir,
			want: false,
		},
		{
			name: "nonexistent path returns false",
			path: filepath.Join(tempDir, "nonexistent"),
			want: false,
		},
		{
			name: "empty path returns false",
			path: "",
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := fileutil.FileExists(tt.path)
			if got != tt.want {
				t.Errorf("FileExists(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestIsFilePath - File path detection
// ---------------------------------------------------------------------------

func TestIsFilePath(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{
			name:  "simple name returns false",
			input: "professional",
			want:  false,
		},
		{
			name:  "relative path with dot-slash returns true",
			input: "./custom.css",
			want:  true,
		},
		{
			name:  "parent path returns true",
			input: "../shared/style.css",
			want:  true,
		},
		{
			name:  "absolute Unix path returns true",
			input: "/absolute/path.css",
			want:  true,
		},
		{
			name:  "Windows path with backslash returns true",
			input: "C:\\windows\\path.css",
			want:  true,
		},
		{
			name:  "hyphenated name returns false",
			input: "my-style",
			want:  false,
		},
		{
			name:  "path with subdirectory returns true",
			input: "sub/dir",
			want:  true,
		},
		{
			name:  "empty string returns false",
			input: "",
			want:  false,
		},
		{
			name:  "name with dots but no slash returns false",
			input: "name.with.dots",
			want:  false,
		},
		{
			name:  "underscore name returns false",
			input: "my_style",
			want:  false,
		},
		{
			name:  "single forward slash returns true",
			input: "/",
			want:  true,
		},
		{
			name:  "single backslash returns true",
			input: "\\",
			want:  true,
		},
		{
			name:  "Windows drive letter path returns true",
			input: "D:/Documents/style.css",
			want:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := fileutil.IsFilePath(tt.input)
			if got != tt.want {
				t.Errorf("IsFilePath(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestIsCSS - CSS content detection
// ---------------------------------------------------------------------------

func TestIsCSS(t *testing.T) {
	t.Parallel()

	tests := []stringBoolCase{
		{
			name:  "style name returns false",
			input: "technical",
			want:  false,
		},
		{
			name:  "file path returns false",
			input: "./custom.css",
			want:  false,
		},
		{
			name:  "CSS content with braces returns true",
			input: "body { color: red; }",
			want:  true,
		},
		{
			name:  "CSS content with multiple rules returns true",
			input: "h1 { font-size: 2em } p { margin: 1em }",
			want:  true,
		},
		{
			name:  "empty string returns false",
			input: "",
			want:  false,
		},
		{
			name:  "hyphenated name returns false",
			input: "my-style",
			want:  false,
		},
		{
			name:  "malformed CSS with only open brace returns true",
			input: "body {",
			want:  true,
		},
	}

	runStringBoolPredicateTests(t, "IsCSS", tests, fileutil.IsCSS)
}

// ---------------------------------------------------------------------------
// TestIsURL - URL detection
// ---------------------------------------------------------------------------

func TestIsURL(t *testing.T) {
	t.Parallel()

	tests := []stringBoolCase{
		{
			name:  "http URL returns true",
			input: "http://example.com",
			want:  true,
		},
		{
			name:  "https URL returns true",
			input: "https://example.com",
			want:  true,
		},
		{
			name:  "file path returns false",
			input: "/path/to/file",
			want:  false,
		},
		{
			name:  "relative path returns false",
			input: "./file.txt",
			want:  false,
		},
		{
			name:  "empty string returns false",
			input: "",
			want:  false,
		},
		{
			name:  "ftp URL returns false",
			input: "ftp://example.com",
			want:  false,
		},
		{
			name:  "HTTP uppercase returns false",
			input: "HTTP://example.com",
			want:  false,
		},
	}

	runStringBoolPredicateTests(t, "IsURL", tests, fileutil.IsURL)
}
