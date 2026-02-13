package pipeline

// Notes:
// - Tests RewriteRelativePaths through its public API only
// - Coverage gaps on error branches in parseHTML/renderHTML are acceptable:
//   the html package rarely fails on valid input and these paths are defensive
// - isRelativePath http:// branch tested via integration; we don't test all URL schemes exhaustively
// - Path traversal security tests verify the observable behavior (path not rewritten)
//   rather than internal isPathUnderDir implementation

import (
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// TestRewriteRelativePaths - Main Function Tests
// ---------------------------------------------------------------------------

func TestRewriteRelativePaths(t *testing.T) {
	t.Parallel()

	// Use a consistent test directory based on OS
	sourceDir := "/docs"
	if runtime.GOOS == "windows" {
		sourceDir = `C:\docs`
	}

	tests := []struct {
		name         string
		html         string
		sourceDir    string
		wantContains []string
		wantExcludes []string
	}{
		{
			name:         "relative image with dot slash",
			html:         `<img src="./images/logo.png">`,
			sourceDir:    sourceDir,
			wantContains: []string{`src="file://`},
		},
		{
			name:         "relative image without dot slash",
			html:         `<img src="images/logo.png">`,
			sourceDir:    sourceDir,
			wantContains: []string{`src="file://`},
		},
		{
			name:         "absolute path unchanged",
			html:         `<img src="/abs/logo.png">`,
			sourceDir:    sourceDir,
			wantContains: []string{`src="/abs/logo.png"`},
		},
		{
			name:         "http URL unchanged",
			html:         `<img src="https://example.com/logo.png">`,
			sourceDir:    sourceDir,
			wantContains: []string{`src="https://example.com/logo.png"`},
		},
		{
			name:         "data URI unchanged",
			html:         `<img src="data:image/png;base64,ABC123">`,
			sourceDir:    sourceDir,
			wantContains: []string{`src="data:image/png;base64,ABC123"`},
		},
		{
			name:         "file URL unchanged",
			html:         `<img src="file:///already/absolute.png">`,
			sourceDir:    sourceDir,
			wantContains: []string{`src="file:///already/absolute.png"`},
		},
		{
			name:         "empty sourceDir returns unchanged",
			html:         `<img src="./logo.png">`,
			sourceDir:    "",
			wantContains: []string{`src="./logo.png"`},
		},
		{
			name:         "anchor link unchanged",
			html:         `<a href="#section">Link</a>`,
			sourceDir:    sourceDir,
			wantContains: []string{`href="#section"`},
		},
		{
			name:         "relative link rewritten",
			html:         `<a href="./other.md">Link</a>`,
			sourceDir:    sourceDir,
			wantContains: []string{`href="file://`},
		},
		{
			name:         "external link unchanged",
			html:         `<a href="https://example.com">External</a>`,
			sourceDir:    sourceDir,
			wantContains: []string{`href="https://example.com"`},
		},
		{
			name:         "protocol-relative URL unchanged",
			html:         `<img src="//cdn.example.com/logo.png">`,
			sourceDir:    sourceDir,
			wantContains: []string{`src="//cdn.example.com/logo.png"`},
		},
		{
			name:         "video source NOT rewritten (PDFs don't support media)",
			html:         `<video src="./video.mp4"></video>`,
			sourceDir:    sourceDir,
			wantContains: []string{`src="./video.mp4"`},
		},
		{
			name:         "audio source NOT rewritten (PDFs don't support media)",
			html:         `<audio src="./audio.mp3"></audio>`,
			sourceDir:    sourceDir,
			wantContains: []string{`src="./audio.mp3"`},
		},
		{
			name:         "multiple images rewritten",
			html:         `<img src="./a.png"><img src="./b.png">`,
			sourceDir:    sourceDir,
			wantContains: []string{`file://`},
		},
		{
			name:         "nested elements rewritten",
			html:         `<div><p><img src="./nested.png"></p></div>`,
			sourceDir:    sourceDir,
			wantContains: []string{`src="file://`},
		},
		{
			name:         "script src NOT rewritten (security)",
			html:         `<script src="./script.js"></script>`,
			sourceDir:    sourceDir,
			wantContains: []string{`src="./script.js"`},
		},
		{
			name:         "empty src attribute unchanged",
			html:         `<img src="">`,
			sourceDir:    sourceDir,
			wantContains: []string{`src=""`},
		},
		{
			name:         "image without src unchanged",
			html:         `<img alt="no src">`,
			sourceDir:    sourceDir,
			wantContains: []string{`alt="no src"`},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := RewriteRelativePaths(tt.html, tt.sourceDir)
			if err != nil {
				t.Fatalf("RewriteRelativePaths(%q, %q) unexpected error: %v", tt.html, tt.sourceDir, err)
			}

			for _, want := range tt.wantContains {
				if !strings.Contains(got, want) {
					t.Errorf("RewriteRelativePaths(%q, %q) = %q, want to contain %q", tt.html, tt.sourceDir, got, want)
				}
			}

			for _, exclude := range tt.wantExcludes {
				if strings.Contains(got, exclude) {
					t.Errorf("RewriteRelativePaths(%q, %q) = %q, should not contain %q", tt.html, tt.sourceDir, got, exclude)
				}
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestRewriteRelativePaths_PathTraversal - Security Tests
// ---------------------------------------------------------------------------

func TestRewriteRelativePaths_PathTraversal(t *testing.T) {
	t.Parallel()

	sourceDir := "/docs"
	if runtime.GOOS == "windows" {
		sourceDir = `C:\docs`
	}

	tests := []struct {
		name         string
		html         string
		wantContains string
	}{
		{
			name:         "parent directory traversal blocked",
			html:         `<img src="../../../etc/passwd">`,
			wantContains: `src="../../../etc/passwd"`,
		},
		{
			name:         "double dot in middle blocked",
			html:         `<img src="images/../../../etc/passwd">`,
			wantContains: `src="images/../../../etc/passwd"`,
		},
		{
			name:         "valid subdirectory allowed",
			html:         `<img src="./images/logo.png">`,
			wantContains: `src="file://`,
		},
		{
			name:         "nested valid path allowed",
			html:         `<img src="images/sub/deep/file.png">`,
			wantContains: `src="file://`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := RewriteRelativePaths(tt.html, sourceDir)
			if err != nil {
				t.Fatalf("RewriteRelativePaths(%q, %q) unexpected error: %v", tt.html, sourceDir, err)
			}

			if !strings.Contains(got, tt.wantContains) {
				t.Errorf("RewriteRelativePaths(%q, %q) = %q, want to contain %q", tt.html, sourceDir, got, tt.wantContains)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestRewriteRelativePaths_DocumentTypes - Full Document vs Fragment
// ---------------------------------------------------------------------------

func TestRewriteRelativePaths_FullDocument(t *testing.T) {
	t.Parallel()

	sourceDir := "/docs"
	if runtime.GOOS == "windows" {
		sourceDir = `C:\docs`
	}

	html := `<!DOCTYPE html>
<html>
<head><title>Test</title></head>
<body><img src="./logo.png"></body>
</html>`

	got, err := RewriteRelativePaths(html, sourceDir)
	if err != nil {
		t.Fatalf("RewriteRelativePaths(%q, %q) unexpected error: %v", html, sourceDir, err)
	}

	// Should preserve document structure (html.Render may lowercase DOCTYPE)
	if !strings.Contains(strings.ToLower(got), "doctype") {
		t.Errorf("RewriteRelativePaths(%q, %q) should preserve DOCTYPE in output", html, sourceDir)
	}
	if !strings.Contains(got, "<html") {
		t.Errorf("RewriteRelativePaths(%q, %q) should preserve <html> in output", html, sourceDir)
	}
	if !strings.Contains(got, `src="file://`) {
		t.Errorf("RewriteRelativePaths(%q, %q) should rewrite image path to file:// URL", html, sourceDir)
	}
}

func TestRewriteRelativePaths_FullDocumentWithHtmlTag(t *testing.T) {
	t.Parallel()

	sourceDir := "/docs"
	if runtime.GOOS == "windows" {
		sourceDir = `C:\docs`
	}

	// Document starting with <html> (no DOCTYPE)
	html := `<html><body><img src="./logo.png"></body></html>`

	got, err := RewriteRelativePaths(html, sourceDir)
	if err != nil {
		t.Fatalf("RewriteRelativePaths(%q, %q) unexpected error: %v", html, sourceDir, err)
	}

	if !strings.Contains(got, "<html") {
		t.Errorf("RewriteRelativePaths(%q, %q) should preserve <html> structure", html, sourceDir)
	}
	if !strings.Contains(got, `src="file://`) {
		t.Errorf("RewriteRelativePaths(%q, %q) should rewrite image path to file:// URL", html, sourceDir)
	}
}

func TestRewriteRelativePaths_Fragment(t *testing.T) {
	t.Parallel()

	sourceDir := "/docs"
	if runtime.GOOS == "windows" {
		sourceDir = `C:\docs`
	}

	html := `<p>Hello</p><img src="./logo.png"><p>World</p>`

	got, err := RewriteRelativePaths(html, sourceDir)
	if err != nil {
		t.Fatalf("RewriteRelativePaths(%q, %q) unexpected error: %v", html, sourceDir, err)
	}

	// Fragment should NOT be wrapped in <html><body>
	if strings.Contains(got, "<html>") {
		t.Errorf("RewriteRelativePaths(%q, %q) should not wrap fragment in <html>", html, sourceDir)
	}

	// Original structure preserved
	if !strings.Contains(got, "<p>Hello</p>") {
		t.Errorf("RewriteRelativePaths(%q, %q) should preserve fragment content", html, sourceDir)
	}

	// Image rewritten
	if !strings.Contains(got, `src="file://`) {
		t.Errorf("RewriteRelativePaths(%q, %q) should rewrite image path to file:// URL", html, sourceDir)
	}
}

// ---------------------------------------------------------------------------
// TestRewriteRelativePaths_AttributePreservation - Attribute Handling
// ---------------------------------------------------------------------------

func TestRewriteRelativePaths_PreservesAttributes(t *testing.T) {
	t.Parallel()

	sourceDir := "/docs"
	if runtime.GOOS == "windows" {
		sourceDir = `C:\docs`
	}

	html := `<img src="./logo.png" alt="Logo" class="logo" width="100">`

	got, err := RewriteRelativePaths(html, sourceDir)
	if err != nil {
		t.Fatalf("RewriteRelativePaths(%q, %q) unexpected error: %v", html, sourceDir, err)
	}

	// All attributes should be preserved
	checks := []string{`alt="Logo"`, `class="logo"`, `width="100"`, `src="file://`}
	for _, check := range checks {
		if !strings.Contains(got, check) {
			t.Errorf("RewriteRelativePaths(%q, %q) = %q, want to contain %q", html, sourceDir, got, check)
		}
	}
}

// ---------------------------------------------------------------------------
// TestRewriteRelativePaths_URLEncoding - Special Characters
// ---------------------------------------------------------------------------

func TestRewriteRelativePaths_URLEncoding(t *testing.T) {
	t.Parallel()

	sourceDir := "/docs"
	if runtime.GOOS == "windows" {
		sourceDir = `C:\docs`
	}

	tests := []struct {
		name         string
		html         string
		wantContains string
	}{
		{
			name:         "path with spaces encoded",
			html:         `<img src="./my images/logo.png">`,
			wantContains: `my%20images`,
		},
		{
			name:         "path with special chars encoded",
			html:         `<img src="./docs/file#1.png">`,
			wantContains: `file%231.png`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := RewriteRelativePaths(tt.html, sourceDir)
			if err != nil {
				t.Fatalf("RewriteRelativePaths(%q, %q) unexpected error: %v", tt.html, sourceDir, err)
			}

			if !strings.Contains(got, tt.wantContains) {
				t.Errorf("RewriteRelativePaths(%q, %q) = %q, want to contain %q", tt.html, sourceDir, got, tt.wantContains)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestIsRelativePath - Helper Function Tests
// ---------------------------------------------------------------------------

func TestIsRelativePath(t *testing.T) {
	t.Parallel()

	tests := []struct {
		path string
		want bool
	}{
		// Relative paths (should return true)
		{"./image.png", true},
		{"images/logo.png", true},
		{"../parent.png", true},
		{"file.png", true},
		{"sub/dir/file.png", true},

		// Non-relative paths (should return false)
		{"", false},
		{"http://example.com/img.png", false},
		{"https://example.com/img.png", false},
		{"file:///abs/path.png", false},
		{"data:image/png;base64,ABC", false},
		{"//cdn.example.com/img.png", false},
		{"#anchor", false},
		{"/absolute/path.png", false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			t.Parallel()

			if got := isRelativePath(tt.path); got != tt.want {
				t.Errorf("isRelativePath(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestIsPathUnderDir - Security Helper Tests
// ---------------------------------------------------------------------------

func TestIsPathUnderDir(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		absPath string
		dir     string
		want    bool
	}{
		{
			name:    "direct child",
			absPath: "/docs/image.png",
			dir:     "/docs",
			want:    true,
		},
		{
			name:    "nested child",
			absPath: "/docs/images/logo.png",
			dir:     "/docs",
			want:    true,
		},
		{
			name:    "parent directory",
			absPath: "/etc/passwd",
			dir:     "/docs",
			want:    false,
		},
		{
			name:    "sibling directory",
			absPath: "/other/file.png",
			dir:     "/docs",
			want:    false,
		},
		{
			name:    "dir with trailing slash",
			absPath: "/docs/image.png",
			dir:     "/docs/",
			want:    true,
		},
		{
			name:    "similar prefix but different dir",
			absPath: "/docs-other/image.png",
			dir:     "/docs",
			want:    false,
		},
		{
			name:    "exact match",
			absPath: "/docs",
			dir:     "/docs",
			want:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Normalize paths for the current OS
			absPath := filepath.FromSlash(tt.absPath)
			dir := filepath.FromSlash(tt.dir)

			if got := isPathUnderDir(absPath, dir); got != tt.want {
				t.Errorf("isPathUnderDir(%q, %q) = %v, want %v", absPath, dir, got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestPathToFileURL - URL Generation Tests
// ---------------------------------------------------------------------------

func TestPathToFileURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		absPath string
		want    string
	}{
		{
			name:    "unix path",
			absPath: "/docs/images/logo.png",
			want:    "file:///docs/images/logo.png",
		},
		{
			name:    "path with spaces",
			absPath: "/docs/my images/logo.png",
			want:    "file:///docs/my%20images/logo.png",
		},
		{
			name:    "path with unicode",
			absPath: "/docs/日本語/logo.png",
			want:    "file:///docs/%E6%97%A5%E6%9C%AC%E8%AA%9E/logo.png",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Skip Windows-specific path tests on Unix
			if runtime.GOOS == "windows" && !strings.Contains(tt.absPath, ":") {
				// On Windows, we need drive letters, skip Unix-style tests
				t.Skip("Unix path test skipped on Windows")
			}

			got := pathToFileURL(tt.absPath)
			if got != tt.want {
				t.Errorf("pathToFileURL(%q) = %q, want %q", tt.absPath, got, tt.want)
			}
		})
	}
}
