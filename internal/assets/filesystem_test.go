package assets

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestNewFilesystemLoader(t *testing.T) {
	t.Parallel()

	t.Run("happy path", func(t *testing.T) {
		t.Parallel()

		tmpDir := t.TempDir()

		loader, err := NewFilesystemLoader(tmpDir)
		if err != nil {
			t.Fatalf("NewFilesystemLoader(%q) unexpected error: %v", tmpDir, err)
		}
		if loader == nil {
			t.Fatal("NewFilesystemLoader() = nil, want non-nil")
		}
	})

	t.Run("error case: empty path", func(t *testing.T) {
		t.Parallel()

		_, err := NewFilesystemLoader("")
		if !errors.Is(err, ErrInvalidBasePath) {
			t.Errorf("NewFilesystemLoader(\"\") = %v, want ErrInvalidBasePath", err)
		}
	})

	t.Run("error case: nonexistent directory", func(t *testing.T) {
		t.Parallel()

		_, err := NewFilesystemLoader("/nonexistent/path/abc123xyz")
		if !errors.Is(err, ErrInvalidBasePath) {
			t.Errorf("NewFilesystemLoader(\"/nonexistent/path/abc123xyz\") = %v, want ErrInvalidBasePath", err)
		}
	})

	t.Run("error case: file instead of directory", func(t *testing.T) {
		t.Parallel()

		tmpDir := t.TempDir()
		filePath := filepath.Join(tmpDir, "file.txt")
		if err := os.WriteFile(filePath, []byte("test"), 0644); err != nil {
			t.Fatalf("failed to create test file: %v", err)
		}

		_, err := NewFilesystemLoader(filePath)
		if !errors.Is(err, ErrInvalidBasePath) {
			t.Errorf("NewFilesystemLoader(%q) = %v, want ErrInvalidBasePath", filePath, err)
		}
	})
}

func TestFilesystemLoader_LoadStyle(t *testing.T) {
	t.Parallel()

	t.Run("happy path", func(t *testing.T) {
		t.Parallel()

		tmpDir := t.TempDir()
		stylesDir := filepath.Join(tmpDir, "styles")
		if err := os.MkdirAll(stylesDir, 0755); err != nil {
			t.Fatalf("failed to create styles dir: %v", err)
		}

		cssContent := "body { color: red; }"
		if err := os.WriteFile(filepath.Join(stylesDir, "custom.css"), []byte(cssContent), 0644); err != nil {
			t.Fatalf("failed to write CSS file: %v", err)
		}

		loader, err := NewFilesystemLoader(tmpDir)
		if err != nil {
			t.Fatalf("NewFilesystemLoader(%q) unexpected error: %v", tmpDir, err)
		}

		got, err := loader.LoadStyle("custom")
		if err != nil {
			t.Fatalf("LoadStyle(\"custom\") unexpected error: %v", err)
		}
		if got != cssContent {
			t.Errorf("LoadStyle(\"custom\") = %q, want %q", got, cssContent)
		}
	})

	t.Run("error case: nonexistent style", func(t *testing.T) {
		t.Parallel()

		tmpDir := t.TempDir()
		stylesDir := filepath.Join(tmpDir, "styles")
		if err := os.MkdirAll(stylesDir, 0755); err != nil {
			t.Fatalf("failed to create styles dir: %v", err)
		}

		loader, err := NewFilesystemLoader(tmpDir)
		if err != nil {
			t.Fatalf("NewFilesystemLoader(%q) unexpected error: %v", tmpDir, err)
		}

		_, err = loader.LoadStyle("nonexistent")
		if !errors.Is(err, ErrStyleNotFound) {
			t.Errorf("LoadStyle(\"nonexistent\") = %v, want ErrStyleNotFound", err)
		}
	})

	t.Run("error case: invalid asset name", func(t *testing.T) {
		t.Parallel()

		tmpDir := t.TempDir()
		loader, err := NewFilesystemLoader(tmpDir)
		if err != nil {
			t.Fatalf("NewFilesystemLoader(%q) unexpected error: %v", tmpDir, err)
		}

		tests := []string{"", "../secret", "..\\secret", "style.evil"}
		for _, name := range tests {
			_, err := loader.LoadStyle(name)
			if !errors.Is(err, ErrInvalidAssetName) {
				t.Errorf("LoadStyle(%q) = %v, want ErrInvalidAssetName", name, err)
			}
		}
	})
}

func TestFilesystemLoader_LoadTemplateSet(t *testing.T) {
	t.Parallel()

	t.Run("happy path", func(t *testing.T) {
		t.Parallel()

		tmpDir := t.TempDir()
		setDir := filepath.Join(tmpDir, "templates", "custom")
		if err := os.MkdirAll(setDir, 0755); err != nil {
			t.Fatalf("failed to create template set dir: %v", err)
		}

		coverContent := "<div>custom cover</div>"
		sigContent := "<div>custom signature</div>"
		if err := os.WriteFile(filepath.Join(setDir, "cover.html"), []byte(coverContent), 0644); err != nil {
			t.Fatalf("failed to write cover file: %v", err)
		}
		if err := os.WriteFile(filepath.Join(setDir, "signature.html"), []byte(sigContent), 0644); err != nil {
			t.Fatalf("failed to write signature file: %v", err)
		}

		loader, err := NewFilesystemLoader(tmpDir)
		if err != nil {
			t.Fatalf("NewFilesystemLoader(%q) unexpected error: %v", tmpDir, err)
		}

		ts, err := loader.LoadTemplateSet("custom")
		if err != nil {
			t.Fatalf("LoadTemplateSet(\"custom\") unexpected error: %v", err)
		}
		if ts.Cover != coverContent {
			t.Errorf("LoadTemplateSet(\"custom\").Cover = %q, want %q", ts.Cover, coverContent)
		}
		if ts.Signature != sigContent {
			t.Errorf("LoadTemplateSet(\"custom\").Signature = %q, want %q", ts.Signature, sigContent)
		}
	})

	t.Run("error case: nonexistent template set", func(t *testing.T) {
		t.Parallel()

		tmpDir := t.TempDir()
		templatesDir := filepath.Join(tmpDir, "templates")
		if err := os.MkdirAll(templatesDir, 0755); err != nil {
			t.Fatalf("failed to create templates dir: %v", err)
		}

		loader, err := NewFilesystemLoader(tmpDir)
		if err != nil {
			t.Fatalf("NewFilesystemLoader(%q) unexpected error: %v", tmpDir, err)
		}

		_, err = loader.LoadTemplateSet("nonexistent")
		if !errors.Is(err, ErrTemplateSetNotFound) {
			t.Errorf("LoadTemplateSet(\"nonexistent\") = %v, want ErrTemplateSetNotFound", err)
		}
	})

	t.Run("error case: invalid asset name", func(t *testing.T) {
		t.Parallel()

		tmpDir := t.TempDir()
		loader, err := NewFilesystemLoader(tmpDir)
		if err != nil {
			t.Fatalf("NewFilesystemLoader(%q) unexpected error: %v", tmpDir, err)
		}

		tests := []string{"", "../secret", "..\\secret", "template.evil"}
		for _, name := range tests {
			_, err := loader.LoadTemplateSet(name)
			if !errors.Is(err, ErrInvalidAssetName) {
				t.Errorf("LoadTemplateSet(%q) = %v, want ErrInvalidAssetName", name, err)
			}
		}
	})

	t.Run("error case: incomplete template set missing cover", func(t *testing.T) {
		t.Parallel()

		tmpDir := t.TempDir()
		setDir := filepath.Join(tmpDir, "templates", "incomplete")
		if err := os.MkdirAll(setDir, 0755); err != nil {
			t.Fatalf("failed to create template set dir: %v", err)
		}

		// Only create signature, not cover
		if err := os.WriteFile(filepath.Join(setDir, "signature.html"), []byte("<div>sig</div>"), 0644); err != nil {
			t.Fatalf("failed to write signature file: %v", err)
		}

		loader, err := NewFilesystemLoader(tmpDir)
		if err != nil {
			t.Fatalf("NewFilesystemLoader(%q) unexpected error: %v", tmpDir, err)
		}

		_, err = loader.LoadTemplateSet("incomplete")
		if !errors.Is(err, ErrIncompleteTemplateSet) {
			t.Errorf("LoadTemplateSet(\"incomplete\") = %v, want ErrIncompleteTemplateSet", err)
		}
	})
}

func TestFilesystemLoader_PathContainment(t *testing.T) {
	t.Parallel()

	t.Run("error case: symlink escape attempt", func(t *testing.T) {
		t.Parallel()

		tmpDir := t.TempDir()
		stylesDir := filepath.Join(tmpDir, "styles")
		if err := os.MkdirAll(stylesDir, 0755); err != nil {
			t.Fatalf("failed to create styles dir: %v", err)
		}

		// Create a secret file outside the base path
		secretDir := t.TempDir()
		secretFile := filepath.Join(secretDir, "secret.css")
		if err := os.WriteFile(secretFile, []byte("secret content"), 0644); err != nil {
			t.Fatalf("failed to write secret file: %v", err)
		}

		// Create symlink inside styles pointing outside
		symlinkPath := filepath.Join(stylesDir, "evil.css")
		if err := os.Symlink(secretFile, symlinkPath); err != nil {
			t.Skipf("symlink creation not supported: %v", err)
		}

		loader, err := NewFilesystemLoader(tmpDir)
		if err != nil {
			t.Fatalf("NewFilesystemLoader(%q) unexpected error: %v", tmpDir, err)
		}

		// The symlink resolves to a path outside basePath
		// verifyPathContainment uses EvalSymlinks to detect this
		_, err = loader.LoadStyle("evil")
		if !errors.Is(err, ErrPathTraversal) {
			t.Errorf("LoadStyle(\"evil\") = %v, want ErrPathTraversal", err)
		}
	})
}

func TestFilesystemLoader_ImplementsAssetLoader(t *testing.T) {
	t.Parallel()

	var _ AssetLoader = (*FilesystemLoader)(nil)
}
