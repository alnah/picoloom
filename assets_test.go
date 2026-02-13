package md2pdf

// Notes:
// - Tests NewAssetLoader with various path configurations (empty, valid, invalid)
// - Verifies style and template set loading with custom overrides and fallbacks
// - Error wrapping behavior is tested for proper sentinel error matching

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// TestNewTemplateSet - Template Set Construction
// ---------------------------------------------------------------------------

func TestNewTemplateSet(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		tsName    string
		cover     string
		signature string
	}{
		{
			name:      "basic template set",
			tsName:    "test",
			cover:     "<div>cover</div>",
			signature: "<div>signature</div>",
		},
		{
			name:      "empty strings",
			tsName:    "",
			cover:     "",
			signature: "",
		},
		{
			name:      "with HTML content",
			tsName:    "full",
			cover:     "<!DOCTYPE html><html><body>Cover Page</body></html>",
			signature: "<div class=\"sig\">{{.Name}}</div>",
		},
		{
			name:      "with template variables",
			tsName:    "templated",
			cover:     "<h1>{{.Title}}</h1><p>{{.Author}}</p>",
			signature: "<p>Signed by {{.Name}} on {{.Date}}</p>",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ts := NewTemplateSet(tt.tsName, tt.cover, tt.signature)

			if ts.Name != tt.tsName {
				t.Errorf("Name = %q, want %q", ts.Name, tt.tsName)
			}
			if ts.Cover != tt.cover {
				t.Errorf("Cover = %q, want %q", ts.Cover, tt.cover)
			}
			if ts.Signature != tt.signature {
				t.Errorf("Signature = %q, want %q", ts.Signature, tt.signature)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestNewAssetLoader_EmptyPath - Embedded Assets Fallback
// ---------------------------------------------------------------------------

func TestNewAssetLoader_EmptyPath(t *testing.T) {
	t.Parallel()

	loader, err := NewAssetLoader("")
	if err != nil {
		t.Fatalf("NewAssetLoader(\"\") unexpected error: %v", err)
	}

	// Verify it can load default style
	css, err := loader.LoadStyle(DefaultStyle)
	if err != nil {
		t.Fatalf("LoadStyle(%q) unexpected error: %v", DefaultStyle, err)
	}
	if css == "" {
		t.Error("LoadStyle returned empty CSS for default style")
	}

	// Verify it can load default template set
	ts, err := loader.LoadTemplateSet(DefaultTemplateSet)
	if err != nil {
		t.Fatalf("LoadTemplateSet(%q) unexpected error: %v", DefaultTemplateSet, err)
	}
	if ts == nil {
		t.Fatal("LoadTemplateSet returned nil")
	}
	if ts.Cover == "" {
		t.Error("TemplateSet.Cover is empty")
	}
	if ts.Signature == "" {
		t.Error("TemplateSet.Signature is empty")
	}
}

// ---------------------------------------------------------------------------
// TestNewAssetLoader_InvalidPath - Invalid Path Error
// ---------------------------------------------------------------------------

func TestNewAssetLoader_InvalidPath(t *testing.T) {
	t.Parallel()

	_, err := NewAssetLoader("/nonexistent/path/to/assets")
	if err == nil {
		t.Fatal("NewAssetLoader(\"/nonexistent/path/to/assets\") error = nil, want error")
	}
	if !errors.Is(err, ErrInvalidAssetPath) {
		t.Errorf("NewAssetLoader(\"/nonexistent/path/to/assets\") error = %v, want ErrInvalidAssetPath", err)
	}
}

// ---------------------------------------------------------------------------
// TestNewAssetLoader_ValidPath - Valid Path with Fallback
// ---------------------------------------------------------------------------

func TestNewAssetLoader_ValidPath(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	loader, err := NewAssetLoader(tmpDir)
	if err != nil {
		t.Fatalf("NewAssetLoader(%q) unexpected error: %v", tmpDir, err)
	}

	// Empty directory should fall back to embedded assets
	css, err := loader.LoadStyle(DefaultStyle)
	if err != nil {
		t.Fatalf("LoadStyle(%q) unexpected error: %v", DefaultStyle, err)
	}
	if css == "" {
		t.Error("Fallback to embedded style failed")
	}
}

// ---------------------------------------------------------------------------
// TestNewAssetLoader_CustomStyleOverride - Custom Style Override
// ---------------------------------------------------------------------------

func TestNewAssetLoader_CustomStyleOverride(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	// Create custom style directory and file
	stylesDir := filepath.Join(tmpDir, "styles")
	if err := os.MkdirAll(stylesDir, 0755); err != nil {
		t.Fatalf("setup: failed to create styles dir: %v", err)
	}

	customCSS := "/* custom override */ body { color: red; }"
	if err := os.WriteFile(filepath.Join(stylesDir, "default.css"), []byte(customCSS), 0644); err != nil {
		t.Fatalf("setup: failed to write custom CSS: %v", err)
	}

	loader, err := NewAssetLoader(tmpDir)
	if err != nil {
		t.Fatalf("NewAssetLoader(%q) unexpected error: %v", tmpDir, err)
	}

	// Should load custom style instead of embedded
	css, err := loader.LoadStyle(DefaultStyle)
	if err != nil {
		t.Fatalf("LoadStyle(%q) unexpected error: %v", DefaultStyle, err)
	}
	if css != customCSS {
		t.Errorf("LoadStyle(%q) = %q, want %q", DefaultStyle, css, customCSS)
	}
}

// ---------------------------------------------------------------------------
// TestAssetLoader_LoadStyle_NotFound - Style Not Found Error
// ---------------------------------------------------------------------------

func TestAssetLoader_LoadStyle_NotFound(t *testing.T) {
	t.Parallel()

	loader, err := NewAssetLoader("")
	if err != nil {
		t.Fatalf("NewAssetLoader(\"\") unexpected error: %v", err)
	}

	_, err = loader.LoadStyle("nonexistent-style")
	if err == nil {
		t.Fatal("LoadStyle(\"nonexistent-style\") error = nil, want error")
	}
	if !errors.Is(err, ErrStyleNotFound) {
		t.Errorf("LoadStyle(\"nonexistent-style\") error = %v, want ErrStyleNotFound", err)
	}
}

// ---------------------------------------------------------------------------
// TestAssetLoader_LoadTemplateSet_NotFound - Template Set Not Found Error
// ---------------------------------------------------------------------------

func TestAssetLoader_LoadTemplateSet_NotFound(t *testing.T) {
	t.Parallel()

	loader, err := NewAssetLoader("")
	if err != nil {
		t.Fatalf("NewAssetLoader(\"\") unexpected error: %v", err)
	}

	_, err = loader.LoadTemplateSet("nonexistent-templates")
	if err == nil {
		t.Fatal("LoadTemplateSet(\"nonexistent-templates\") error = nil, want error")
	}
	if !errors.Is(err, ErrTemplateSetNotFound) {
		t.Errorf("LoadTemplateSet(\"nonexistent-templates\") error = %v, want ErrTemplateSetNotFound", err)
	}
}

// ---------------------------------------------------------------------------
// TestDefaultConstants - Default Constant Values
// ---------------------------------------------------------------------------

func TestDefaultConstants(t *testing.T) {
	t.Parallel()

	if DefaultStyle != "default" {
		t.Errorf("DefaultStyle = %q, want \"default\"", DefaultStyle)
	}
	if DefaultTemplateSet != "default" {
		t.Errorf("DefaultTemplateSet = %q, want \"default\"", DefaultTemplateSet)
	}
}

// ---------------------------------------------------------------------------
// TestErrorWrapping_PreservesMessage - Error Message Preservation
// ---------------------------------------------------------------------------

func TestErrorWrapping_PreservesMessage(t *testing.T) {
	t.Parallel()

	loader, err := NewAssetLoader("")
	if err != nil {
		t.Fatalf("NewAssetLoader(\"\") unexpected error: %v", err)
	}

	_, err = loader.LoadStyle("custom-style")

	// Error message should contain the style name
	if err == nil {
		t.Fatal("LoadStyle(\"custom-style\") error = nil, want error")
	}
	errMsg := err.Error()
	if errMsg == "" {
		t.Error("error message should not be empty")
	}
	// The message should mention the style name
	if !strings.Contains(errMsg, "custom-style") {
		t.Errorf("error message %q should contain style name", errMsg)
	}
}

// ---------------------------------------------------------------------------
// TestErrorWrapping_UnwrapsToSentinel - Error Sentinel Unwrapping
// ---------------------------------------------------------------------------

func TestErrorWrapping_UnwrapsToSentinel(t *testing.T) {
	t.Parallel()

	loader, err := NewAssetLoader("")
	if err != nil {
		t.Fatalf("NewAssetLoader(\"\") unexpected error: %v", err)
	}

	// Test ErrStyleNotFound
	_, styleErr := loader.LoadStyle("nonexistent")
	if !errors.Is(styleErr, ErrStyleNotFound) {
		t.Errorf("LoadStyle(\"nonexistent\") error should unwrap to ErrStyleNotFound, got %v", styleErr)
	}

	// Test ErrTemplateSetNotFound
	_, tsErr := loader.LoadTemplateSet("nonexistent")
	if !errors.Is(tsErr, ErrTemplateSetNotFound) {
		t.Errorf("LoadTemplateSet(\"nonexistent\") error should unwrap to ErrTemplateSetNotFound, got %v", tsErr)
	}
}

// ---------------------------------------------------------------------------
// TestNewAssetLoader_CustomTemplateOverride - Custom Template Override
// ---------------------------------------------------------------------------

func TestNewAssetLoader_CustomTemplateOverride(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	// Create custom template directory and files
	templatesDir := filepath.Join(tmpDir, "templates", "default")
	if err := os.MkdirAll(templatesDir, 0755); err != nil {
		t.Fatalf("setup: failed to create templates dir: %v", err)
	}

	customCover := "<div>Custom Cover</div>"
	customSig := "<div>Custom Signature</div>"
	if err := os.WriteFile(filepath.Join(templatesDir, "cover.html"), []byte(customCover), 0644); err != nil {
		t.Fatalf("setup: failed to write cover.html: %v", err)
	}
	if err := os.WriteFile(filepath.Join(templatesDir, "signature.html"), []byte(customSig), 0644); err != nil {
		t.Fatalf("setup: failed to write signature.html: %v", err)
	}

	loader, err := NewAssetLoader(tmpDir)
	if err != nil {
		t.Fatalf("NewAssetLoader(%q) unexpected error: %v", tmpDir, err)
	}

	// Should load custom templates instead of embedded
	ts, err := loader.LoadTemplateSet(DefaultTemplateSet)
	if err != nil {
		t.Fatalf("LoadTemplateSet(%q) unexpected error: %v", DefaultTemplateSet, err)
	}
	if ts.Cover != customCover {
		t.Errorf("LoadTemplateSet(%q).Cover = %q, want %q", DefaultTemplateSet, ts.Cover, customCover)
	}
	if ts.Signature != customSig {
		t.Errorf("LoadTemplateSet(%q).Signature = %q, want %q", DefaultTemplateSet, ts.Signature, customSig)
	}
}

// ---------------------------------------------------------------------------
// TestWrappedAssetError_Error - Wrapped Error Message
// ---------------------------------------------------------------------------

func TestWrappedAssetError_Error(t *testing.T) {
	t.Parallel()

	original := errors.New("original error message")
	sentinel := errors.New("sentinel")

	wrapped := wrapError(sentinel, original)

	// Error() should return original message
	if wrapped.Error() != original.Error() {
		t.Errorf("wrapError(sentinel, original).Error() = %q, want %q", wrapped.Error(), original.Error())
	}
}

// ---------------------------------------------------------------------------
// TestWrappedAssetError_Unwrap - Wrapped Error Unwrapping
// ---------------------------------------------------------------------------

func TestWrappedAssetError_Unwrap(t *testing.T) {
	t.Parallel()

	original := errors.New("original error message")
	sentinel := errors.New("sentinel")

	wrapped := wrapError(sentinel, original)

	// Unwrap should return sentinel (for errors.Is)
	var unwrapped interface{ Unwrap() error }
	if errors.As(wrapped, &unwrapped) {
		if unwrapped.Unwrap() != sentinel {
			t.Errorf("wrapError(sentinel, original).Unwrap() = %v, want %v", unwrapped.Unwrap(), sentinel)
		}
	} else {
		t.Error("wrapped error should implement Unwrap()")
	}

	// errors.Is should match sentinel
	if !errors.Is(wrapped, sentinel) {
		t.Error("errors.Is(wrapError(sentinel, original), sentinel) = false, want true")
	}

	// errors.Is should NOT match original
	if errors.Is(wrapped, original) {
		t.Error("errors.Is(wrapError(sentinel, original), original) = true, want false")
	}
}

// ---------------------------------------------------------------------------
// TestConvertAssetError_NilError - Nil Error Conversion
// ---------------------------------------------------------------------------

func TestConvertAssetError_NilError(t *testing.T) {
	t.Parallel()

	result := convertAssetError(nil)
	if result != nil {
		t.Errorf("convertAssetError(nil) = %v, want nil", result)
	}
}

// ---------------------------------------------------------------------------
// TestIsError - Error Comparison
// ---------------------------------------------------------------------------

func TestIsError(t *testing.T) {
	t.Parallel()

	sentinel := errors.New("sentinel")

	tests := []struct {
		name   string
		err    error
		target error
		want   bool
	}{
		{
			name:   "matching error",
			err:    sentinel,
			target: sentinel,
			want:   true,
		},
		{
			name:   "both nil",
			err:    nil,
			target: nil,
			want:   true,
		},
		{
			name:   "nil error with non-nil target",
			err:    nil,
			target: sentinel,
			want:   false,
		},
		{
			name:   "non-nil error with nil target",
			err:    sentinel,
			target: nil,
			want:   false,
		},
		{
			name:   "different errors",
			err:    errors.New("other"),
			target: sentinel,
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := isError(tt.err, tt.target)
			if got != tt.want {
				t.Errorf("isError() = %v, want %v", got, tt.want)
			}
		})
	}
}
