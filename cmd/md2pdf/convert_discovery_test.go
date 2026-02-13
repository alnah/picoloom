package main

// Notes:
// - resolveInputPath: we test precedence (args > config > error).
// - resolveOutputDir: we test precedence (flag > config > empty).
// - resolveOutputPath: we test path resolution including directory mirroring.
// - discoverFiles: we test file discovery with temp directories. We don't test
//   symlink edge cases as they are rare and platform-specific.
// These are acceptable gaps: we test observable behavior, not implementation details.

import (
	"os"
	"path/filepath"
	"testing"
)

// ---------------------------------------------------------------------------
// TestResolveInputPath - Input path resolution precedence
// ---------------------------------------------------------------------------

func TestResolveInputPath(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		args    []string
		cfg     *Config
		want    string
		wantErr error
	}{
		{
			name: "args takes precedence over config",
			args: []string{"doc.md"},
			cfg:  &Config{Input: InputConfig{DefaultDir: "./default/"}},
			want: "doc.md",
		},
		{
			name: "config fallback when no args",
			args: []string{},
			cfg:  &Config{Input: InputConfig{DefaultDir: "./default/"}},
			want: "./default/",
		},
		{
			name:    "error when no args and no config",
			args:    []string{},
			cfg:     &Config{Input: InputConfig{DefaultDir: ""}},
			wantErr: ErrNoInput,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := resolveInputPath(tt.args, tt.cfg)

			if tt.wantErr != nil {
				if err != tt.wantErr {
					t.Errorf("resolveInputPath(%v, cfg) error = %v, want %v", tt.args, err, tt.wantErr)
				}
				return
			}

			if err != nil {
				t.Fatalf("resolveInputPath(%v, cfg) unexpected error: %v", tt.args, err)
			}

			if got != tt.want {
				t.Errorf("resolveInputPath(%v, cfg) = %q, want %q", tt.args, got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestResolveOutputDir - Output directory resolution precedence
// ---------------------------------------------------------------------------

func TestResolveOutputDir(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		flagOutput string
		cfg        *Config
		want       string
	}{
		{
			name:       "flag takes precedence over config",
			flagOutput: "./out/",
			cfg:        &Config{Output: OutputConfig{DefaultDir: "./default/"}},
			want:       "./out/",
		},
		{
			name:       "config fallback when no flag",
			flagOutput: "",
			cfg:        &Config{Output: OutputConfig{DefaultDir: "./default/"}},
			want:       "./default/",
		},
		{
			name:       "empty when no flag and no config",
			flagOutput: "",
			cfg:        &Config{Output: OutputConfig{DefaultDir: ""}},
			want:       "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := resolveOutputDir(tt.flagOutput, tt.cfg)
			if got != tt.want {
				t.Errorf("resolveOutputDir(%q, cfg) = %q, want %q", tt.flagOutput, got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestResolveOutputPath - Output path resolution with directory mirroring
// ---------------------------------------------------------------------------

func TestResolveOutputPath(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		inputPath    string
		outputDir    string
		baseInputDir string
		want         string
	}{
		{
			name:      "no output dir - PDF next to source",
			inputPath: "/docs/file.md",
			outputDir: "",
			want:      "/docs/file.pdf",
		},
		{
			name:      "output is PDF file",
			inputPath: "/docs/file.md",
			outputDir: "/out/result.pdf",
			want:      "/out/result.pdf",
		},
		{
			name:      "output is directory - single file",
			inputPath: "/docs/file.md",
			outputDir: "/out/",
			want:      "/out/file.pdf",
		},
		{
			name:         "output is directory - mirror structure",
			inputPath:    "/docs/subdir/file.md",
			outputDir:    "/out",
			baseInputDir: "/docs",
			want:         "/out/subdir/file.pdf",
		},
		{
			name:         "mirror structure with nested dirs",
			inputPath:    "/docs/a/b/c/file.md",
			outputDir:    "/out",
			baseInputDir: "/docs",
			want:         "/out/a/b/c/file.pdf",
		},
		{
			name:      "markdown extension",
			inputPath: "/docs/file.markdown",
			outputDir: "",
			want:      "/docs/file.pdf",
		},
		{
			// When filepath.Rel fails (e.g., different drives on Windows),
			// falls back to flat output in outputDir.
			name:         "filepath.Rel fallback - unrelated paths",
			inputPath:    "relative/file.md",
			outputDir:    "/out",
			baseInputDir: "/absolute/base",
			want:         "/out/file.pdf",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := resolveOutputPath(tt.inputPath, tt.outputDir, tt.baseInputDir)
			if got != tt.want {
				t.Errorf("resolveOutputPath(%q, %q, %q) = %q, want %q", tt.inputPath, tt.outputDir, tt.baseInputDir, got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestValidateMarkdownExtension - Markdown file extension validation
// ---------------------------------------------------------------------------

func TestValidateMarkdownExtension(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{
			name:    "valid .md extension",
			path:    "doc.md",
			wantErr: false,
		},
		{
			name:    "valid .markdown extension",
			path:    "doc.markdown",
			wantErr: false,
		},
		{
			name:    "invalid .txt extension",
			path:    "doc.txt",
			wantErr: true,
		},
		{
			name:    "invalid .pdf extension",
			path:    "doc.pdf",
			wantErr: true,
		},
		{
			name:    "no extension",
			path:    "doc",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := validateMarkdownExtension(tt.path)
			if (err != nil) != tt.wantErr {
				if tt.wantErr {
					t.Errorf("validateMarkdownExtension(%q) error = nil, want error", tt.path)
				} else {
					t.Errorf("validateMarkdownExtension(%q) unexpected error: %v", tt.path, err)
				}
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestDiscoverFiles - File discovery with recursive directory traversal
// ---------------------------------------------------------------------------

func TestDiscoverFiles(t *testing.T) {
	t.Parallel()

	// Create temp directory structure
	tempDir := t.TempDir()

	// Create files
	files := map[string]string{
		"doc1.md":              "# Doc 1",
		"doc2.markdown":        "# Doc 2",
		"subdir/doc3.md":       "# Doc 3",
		"subdir/deep/doc4.md":  "# Doc 4",
		"ignored.txt":          "ignored",
		"subdir/ignored2.html": "ignored",
	}

	for path, content := range files {
		fullPath := filepath.Join(tempDir, path)
		if err := os.MkdirAll(filepath.Dir(fullPath), 0750); err != nil {
			t.Fatalf("failed to create dir: %v", err)
		}
		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			t.Fatalf("failed to write file: %v", err)
		}
	}

	t.Run("single file", func(t *testing.T) {
		t.Parallel()

		inputPath := filepath.Join(tempDir, "doc1.md")
		got, err := discoverFiles(inputPath, "")
		if err != nil {
			t.Fatalf("discoverFiles(%q, \"\") unexpected error: %v", inputPath, err)
		}
		if len(got) != 1 {
			t.Errorf("discoverFiles(%q, \"\") returned %d files, want 1", inputPath, len(got))
		}
		if got[0].InputPath != inputPath {
			t.Errorf("discoverFiles(%q, \"\")[0].InputPath = %q, want %q", inputPath, got[0].InputPath, inputPath)
		}
	})

	t.Run("directory recursive", func(t *testing.T) {
		t.Parallel()

		got, err := discoverFiles(tempDir, "")
		if err != nil {
			t.Fatalf("discoverFiles(%q, \"\") unexpected error: %v", tempDir, err)
		}
		if len(got) != 4 {
			t.Errorf("discoverFiles(%q, \"\") returned %d files, want 4 (doc1.md, doc2.markdown, subdir/doc3.md, subdir/deep/doc4.md)", tempDir, len(got))
		}
	})

	t.Run("directory with output dir mirrors structure", func(t *testing.T) {
		t.Parallel()

		outputDir := filepath.Join(tempDir, "output")
		got, err := discoverFiles(tempDir, outputDir)
		if err != nil {
			t.Fatalf("discoverFiles(%q, %q) unexpected error: %v", tempDir, outputDir, err)
		}

		// Check that subdir structure is mirrored
		foundMirrored := false
		for _, f := range got {
			if filepath.Base(f.InputPath) == "doc3.md" {
				expectedOutput := filepath.Join(outputDir, "subdir", "doc3.pdf")
				if f.OutputPath != expectedOutput {
					t.Errorf("discoverFiles(%q, %q) doc3.md OutputPath = %q, want %q", tempDir, outputDir, f.OutputPath, expectedOutput)
				}
				foundMirrored = true
			}
		}
		if !foundMirrored {
			t.Errorf("discoverFiles(%q, %q) did not find doc3.md in results", tempDir, outputDir)
		}
	})

	t.Run("invalid extension returns error", func(t *testing.T) {
		t.Parallel()

		inputPath := filepath.Join(tempDir, "ignored.txt")
		_, err := discoverFiles(inputPath, "")
		if err == nil {
			t.Errorf("discoverFiles(%q, \"\") error = nil, want error", inputPath)
		}
	})

	t.Run("nonexistent path returns error", func(t *testing.T) {
		t.Parallel()

		_, err := discoverFiles("/nonexistent/path", "")
		if err == nil {
			t.Errorf("discoverFiles(\"/nonexistent/path\", \"\") error = nil, want error")
		}
	})
}
