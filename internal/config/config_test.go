package config

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Input.DefaultDir != "" {
		t.Errorf("DefaultConfig().Input.DefaultDir = %q, want empty", cfg.Input.DefaultDir)
	}
	if cfg.Output.DefaultDir != "" {
		t.Errorf("DefaultConfig().Output.DefaultDir = %q, want empty", cfg.Output.DefaultDir)
	}
	if cfg.Style != "" {
		t.Errorf("DefaultConfig().Style = %q, want empty", cfg.Style)
	}
	if cfg.Footer.Enabled {
		t.Error("DefaultConfig().Footer.Enabled = true, want false")
	}
	if cfg.Signature.Enabled {
		t.Error("DefaultConfig().Signature.Enabled = true, want false")
	}
	if cfg.Assets.BasePath != "" {
		t.Errorf("DefaultConfig().Assets.BasePath = %q, want empty", cfg.Assets.BasePath)
	}
}

func TestValidateFieldLength(t *testing.T) {
	tests := []struct {
		name      string
		fieldName string
		value     string
		maxLength int
		wantErr   bool
	}{
		{
			name:      "empty value is valid",
			fieldName: "test",
			value:     "",
			maxLength: 10,
			wantErr:   false,
		},
		{
			name:      "value at limit is valid",
			fieldName: "test",
			value:     "1234567890",
			maxLength: 10,
			wantErr:   false,
		},
		{
			name:      "value under limit is valid",
			fieldName: "test",
			value:     "12345",
			maxLength: 10,
			wantErr:   false,
		},
		{
			name:      "value over limit returns error",
			fieldName: "test.field",
			value:     "12345678901",
			maxLength: 10,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateFieldLength(tt.fieldName, tt.value, tt.maxLength)
			if tt.wantErr {
				if err == nil {
					t.Fatal("validateFieldLength(tt.fieldName, tt.value, tt.maxLength) error = nil, want error")
				}
				if !errors.Is(err, ErrFieldTooLong) {
					t.Errorf("validateFieldLength(tt.fieldName, tt.value, tt.maxLength) error = %v, want ErrFieldTooLong", err)
				}
			} else {
				if err != nil {
					t.Errorf("validateFieldLength(tt.fieldName, tt.value, tt.maxLength) unexpected error: %v", err)
				}
			}
		})
	}
}

func TestConfig_Validate(t *testing.T) {
	t.Run("valid config passes validation", func(t *testing.T) {
		cfg := &Config{
			Author: AuthorConfig{
				Name:  "John Doe",
				Title: "Developer",
				Email: "john@example.com",
			},
			Document: DocumentConfig{
				Date:    "2025-01-15",
				Version: "FINAL",
			},
			Signature: SignatureConfig{
				Links: []Link{
					{Label: "GitHub", URL: "https://github.com/johndoe"},
				},
			},
			Footer: FooterConfig{
				Text: "Confidential",
			},
		}
		err := cfg.Validate()
		if err != nil {
			t.Errorf("Config.Validate() unexpected error: %v", err)
		}
	})

	t.Run("author.name too long returns error", func(t *testing.T) {
		cfg := &Config{
			Author: AuthorConfig{
				Name: string(make([]byte, MaxNameLength+1)),
			},
		}
		err := cfg.Validate()
		if !errors.Is(err, ErrFieldTooLong) {
			t.Errorf("Config.Validate() error = %v, want ErrFieldTooLong", err)
		}
	})

	t.Run("author.email too long returns error", func(t *testing.T) {
		cfg := &Config{
			Author: AuthorConfig{
				Email: string(make([]byte, MaxEmailLength+1)),
			},
		}
		err := cfg.Validate()
		if !errors.Is(err, ErrFieldTooLong) {
			t.Errorf("Config.Validate() error = %v, want ErrFieldTooLong", err)
		}
	})

	t.Run("signature.links[].url too long returns error", func(t *testing.T) {
		cfg := &Config{
			Signature: SignatureConfig{
				Links: []Link{
					{Label: "Valid", URL: string(make([]byte, MaxURLLength+1))},
				},
			},
		}
		err := cfg.Validate()
		if !errors.Is(err, ErrFieldTooLong) {
			t.Errorf("Config.Validate() error = %v, want ErrFieldTooLong", err)
		}
	})

	t.Run("footer.text too long returns error", func(t *testing.T) {
		cfg := &Config{
			Footer: FooterConfig{
				Text: string(make([]byte, MaxTextLength+1)),
			},
		}
		err := cfg.Validate()
		if !errors.Is(err, ErrFieldTooLong) {
			t.Errorf("Config.Validate() error = %v, want ErrFieldTooLong", err)
		}
	})

	t.Run("document.version too long returns error", func(t *testing.T) {
		cfg := &Config{
			Document: DocumentConfig{
				Version: string(make([]byte, MaxVersionLength+1)),
			},
		}
		err := cfg.Validate()
		if !errors.Is(err, ErrFieldTooLong) {
			t.Errorf("Config.Validate() error = %v, want ErrFieldTooLong", err)
		}
	})
}

func TestLoadConfig(t *testing.T) {
	t.Run("empty name returns ErrEmptyConfigName", func(t *testing.T) {
		_, err := LoadConfig("")
		if !errors.Is(err, ErrEmptyConfigName) {
			t.Errorf("LoadConfig(\"\") error = %v, want ErrEmptyConfigName", err)
		}
	})

	t.Run("valid file path loads config", func(t *testing.T) {
		dir := t.TempDir()
		configPath := filepath.Join(dir, "test.yaml")
		content := `style: "default"
footer:
  enabled: true
  position: "center"
`
		if err := os.WriteFile(configPath, []byte(content), 0600); err != nil {
			t.Fatalf("setup WriteFile: %v", err)
		}

		cfg, err := LoadConfig(configPath)
		if err != nil {
			t.Fatalf("LoadConfig(configPath) unexpected error: %v", err)
		}
		if cfg.Style != "default" {
			t.Errorf("LoadConfig(configPath).Style = %q, want %q", cfg.Style, "default")
		}
		if !cfg.Footer.Enabled {
			t.Error("LoadConfig(configPath).Footer.Enabled = false, want true")
		}
		if cfg.Footer.Position != "center" {
			t.Errorf("LoadConfig(configPath).Footer.Position = %q, want %q", cfg.Footer.Position, "center")
		}
	})

	t.Run("loads input and output directories", func(t *testing.T) {
		dir := t.TempDir()
		configPath := filepath.Join(dir, "test.yaml")
		content := `input:
  defaultDir: "/path/to/input"
output:
  defaultDir: "/path/to/output"
`
		if err := os.WriteFile(configPath, []byte(content), 0600); err != nil {
			t.Fatalf("setup WriteFile: %v", err)
		}

		cfg, err := LoadConfig(configPath)
		if err != nil {
			t.Fatalf("LoadConfig(configPath) unexpected error: %v", err)
		}
		if cfg.Input.DefaultDir != "/path/to/input" {
			t.Errorf("LoadConfig(configPath).Input.DefaultDir = %q, want %q", cfg.Input.DefaultDir, "/path/to/input")
		}
		if cfg.Output.DefaultDir != "/path/to/output" {
			t.Errorf("LoadConfig(configPath).Output.DefaultDir = %q, want %q", cfg.Output.DefaultDir, "/path/to/output")
		}
	})

	t.Run("nonexistent file path returns ErrConfigNotFound", func(t *testing.T) {
		_, err := LoadConfig("/nonexistent/path/config.yaml")
		if !errors.Is(err, ErrConfigNotFound) {
			t.Errorf("LoadConfig(\"/nonexistent/path/config.yaml\") error = %v, want ErrConfigNotFound", err)
		}
	})

	t.Run("invalid YAML returns ErrConfigParse", func(t *testing.T) {
		dir := t.TempDir()
		configPath := filepath.Join(dir, "invalid.yaml")
		if err := os.WriteFile(configPath, []byte("style: [unclosed"), 0600); err != nil {
			t.Fatalf("setup: %v", err)
		}

		_, err := LoadConfig(configPath)
		if !errors.Is(err, ErrConfigParse) {
			t.Errorf("LoadConfig(configPath) error = %v, want ErrConfigParse", err)
		}
	})

	t.Run("unknown field returns ErrConfigParse in strict mode", func(t *testing.T) {
		dir := t.TempDir()
		configPath := filepath.Join(dir, "unknown.yaml")
		content := `style: "default"
unknownField: "should fail"
`
		if err := os.WriteFile(configPath, []byte(content), 0600); err != nil {
			t.Fatalf("setup WriteFile: %v", err)
		}

		_, err := LoadConfig(configPath)
		if !errors.Is(err, ErrConfigParse) {
			t.Errorf("LoadConfig(configPath) error = %v, want ErrConfigParse", err)
		}
	})

	t.Run("field too long returns ErrFieldTooLong", func(t *testing.T) {
		dir := t.TempDir()
		configPath := filepath.Join(dir, "toolong.yaml")
		longName := string(make([]byte, MaxNameLength+1))
		content := "author:\n  name: \"" + longName + "\"\n"
		if err := os.WriteFile(configPath, []byte(content), 0600); err != nil {
			t.Fatalf("setup WriteFile: %v", err)
		}

		_, err := LoadConfig(configPath)
		if !errors.Is(err, ErrFieldTooLong) {
			t.Errorf("LoadConfig(configPath) error = %v, want ErrFieldTooLong", err)
		}
	})

	t.Run("unreadable file returns read error not ErrConfigNotFound", func(t *testing.T) {
		dir := t.TempDir()
		configPath := filepath.Join(dir, "unreadable.yaml")
		if err := os.WriteFile(configPath, []byte("style: test\n"), 0600); err != nil {
			t.Fatalf("setup: %v", err)
		}
		if err := os.Chmod(configPath, 0000); err != nil {
			t.Fatalf("setup chmod: %v", err)
		}
		defer os.Chmod(configPath, 0600)

		_, err := LoadConfig(configPath)
		if err == nil {
			t.Fatal("LoadConfig(configPath) error = nil, want error")
		}
		if errors.Is(err, ErrConfigNotFound) {
			t.Error("LoadConfig(configPath) error should not be ErrConfigNotFound for permission error")
		}
	})

	t.Run("config name resolves yaml in current directory", func(t *testing.T) {
		dir := t.TempDir()
		configPath := filepath.Join(dir, "myconfig.yaml")
		if err := os.WriteFile(configPath, []byte("style: fromname\n"), 0600); err != nil {
			t.Fatalf("setup: %v", err)
		}

		originalWd, err := os.Getwd()
		if err != nil {
			t.Fatalf("failed to get working directory: %v", err)
		}
		defer os.Chdir(originalWd)
		if err := os.Chdir(dir); err != nil {
			t.Fatalf("chdir: %v", err)
		}

		cfg, err := LoadConfig("myconfig")
		if err != nil {
			t.Fatalf("LoadConfig(\"myconfig\") unexpected error: %v", err)
		}
		if cfg.Style != "fromname" {
			t.Errorf("LoadConfig(\"myconfig\").Style = %q, want %q", cfg.Style, "fromname")
		}
	})

	t.Run("config name resolves yml when yaml not found", func(t *testing.T) {
		dir := t.TempDir()
		configPath := filepath.Join(dir, "myconfig.yml")
		if err := os.WriteFile(configPath, []byte("style: fromyml\n"), 0600); err != nil {
			t.Fatalf("setup: %v", err)
		}

		originalWd, err := os.Getwd()
		if err != nil {
			t.Fatalf("failed to get working directory: %v", err)
		}
		defer os.Chdir(originalWd)
		if err := os.Chdir(dir); err != nil {
			t.Fatalf("chdir: %v", err)
		}

		cfg, err := LoadConfig("myconfig")
		if err != nil {
			t.Fatalf("LoadConfig(\"myconfig\") unexpected error: %v", err)
		}
		if cfg.Style != "fromyml" {
			t.Errorf("LoadConfig(\"myconfig\").Style = %q, want %q", cfg.Style, "fromyml")
		}
	})

	t.Run("config name prefers yaml over yml", func(t *testing.T) {
		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, "myconfig.yaml"), []byte("style: yaml\n"), 0600); err != nil {
			t.Fatalf("setup yaml: %v", err)
		}
		if err := os.WriteFile(filepath.Join(dir, "myconfig.yml"), []byte("style: yml\n"), 0600); err != nil {
			t.Fatalf("setup yml: %v", err)
		}

		originalWd, err := os.Getwd()
		if err != nil {
			t.Fatalf("failed to get working directory: %v", err)
		}
		defer os.Chdir(originalWd)
		if err := os.Chdir(dir); err != nil {
			t.Fatalf("chdir: %v", err)
		}

		cfg, err := LoadConfig("myconfig")
		if err != nil {
			t.Fatalf("LoadConfig(\"myconfig\") unexpected error: %v", err)
		}
		if cfg.Style != "yaml" {
			t.Errorf("LoadConfig(\"myconfig\").Style = %q, want %q (should prefer .yaml)", cfg.Style, "yaml")
		}
	})

	t.Run("config name resolves from user config directory", func(t *testing.T) {
		userConfigDir, err := os.UserConfigDir()
		if err != nil {
			t.Skip("cannot get user config dir")
		}

		appConfigDir := filepath.Join(userConfigDir, "go-md2pdf")
		configPath := filepath.Join(appConfigDir, "testconfig.yaml")

		if err := os.MkdirAll(appConfigDir, 0755); err != nil {
			t.Fatalf("setup mkdir: %v", err)
		}
		if err := os.WriteFile(configPath, []byte("style: userdir\n"), 0600); err != nil {
			t.Fatalf("setup write: %v", err)
		}
		defer os.Remove(configPath)

		// Change to empty dir so local file isn't found
		dir := t.TempDir()
		originalWd, err := os.Getwd()
		if err != nil {
			t.Fatalf("failed to get working directory: %v", err)
		}
		defer os.Chdir(originalWd)
		if err := os.Chdir(dir); err != nil {
			t.Fatalf("chdir: %v", err)
		}

		cfg, err := LoadConfig("testconfig")
		if err != nil {
			t.Fatalf("LoadConfig(\"testconfig\") unexpected error: %v", err)
		}
		if cfg.Style != "userdir" {
			t.Errorf("LoadConfig(\"testconfig\").Style = %q, want %q", cfg.Style, "userdir")
		}
	})

	t.Run("config name not found returns ErrConfigNotFound", func(t *testing.T) {
		dir := t.TempDir()
		originalWd, err := os.Getwd()
		if err != nil {
			t.Fatalf("failed to get working directory: %v", err)
		}
		defer os.Chdir(originalWd)
		if err := os.Chdir(dir); err != nil {
			t.Fatalf("chdir: %v", err)
		}

		_, err = LoadConfig("nonexistent")
		if !errors.Is(err, ErrConfigNotFound) {
			t.Errorf("LoadConfig(\"nonexistent\") error = %v, want ErrConfigNotFound", err)
		}
	})

	t.Run("config name not found includes searched paths and hint", func(t *testing.T) {
		dir := t.TempDir()
		originalWd, err := os.Getwd()
		if err != nil {
			t.Fatalf("failed to get working directory: %v", err)
		}
		defer os.Chdir(originalWd)
		if err := os.Chdir(dir); err != nil {
			t.Fatalf("chdir: %v", err)
		}

		_, err = LoadConfig("nonexistent")
		if err == nil {
			t.Fatal("LoadConfig(\"nonexistent\") error = nil, want error")
		}
		errMsg := err.Error()
		if !strings.Contains(errMsg, "searched:") {
			t.Error("LoadConfig(\"nonexistent\") error should include searched paths")
		}
		if !strings.Contains(errMsg, "hint:") {
			t.Error("LoadConfig(\"nonexistent\") error should include hint")
		}
		if !strings.Contains(errMsg, "--config") {
			t.Error("LoadConfig(\"nonexistent\") error hint should mention --config flag")
		}
	})

	t.Run("loads page settings", func(t *testing.T) {
		dir := t.TempDir()
		configPath := filepath.Join(dir, "test.yaml")
		content := `page:
  size: "a4"
  orientation: "landscape"
  margin: 1.0
`
		if err := os.WriteFile(configPath, []byte(content), 0600); err != nil {
			t.Fatalf("setup WriteFile: %v", err)
		}

		cfg, err := LoadConfig(configPath)
		if err != nil {
			t.Fatalf("LoadConfig(configPath) unexpected error: %v", err)
		}
		if cfg.Page.Size != "a4" {
			t.Errorf("LoadConfig(configPath).Page.Size = %q, want %q", cfg.Page.Size, "a4")
		}
		if cfg.Page.Orientation != "landscape" {
			t.Errorf("LoadConfig(configPath).Page.Orientation = %q, want %q", cfg.Page.Orientation, "landscape")
		}
		if cfg.Page.Margin != 1.0 {
			t.Errorf("LoadConfig(configPath).Page.Margin = %v, want %v", cfg.Page.Margin, 1.0)
		}
	})

	t.Run("page.size too long returns ErrFieldTooLong", func(t *testing.T) {
		dir := t.TempDir()
		configPath := filepath.Join(dir, "test.yaml")
		longSize := string(make([]byte, MaxPageSizeLength+1))
		content := "page:\n  size: \"" + longSize + "\"\n"
		if err := os.WriteFile(configPath, []byte(content), 0600); err != nil {
			t.Fatalf("setup WriteFile: %v", err)
		}

		_, err := LoadConfig(configPath)
		if !errors.Is(err, ErrFieldTooLong) {
			t.Errorf("LoadConfig(configPath) error = %v, want ErrFieldTooLong", err)
		}
	})

	t.Run("page.orientation too long returns ErrFieldTooLong", func(t *testing.T) {
		dir := t.TempDir()
		configPath := filepath.Join(dir, "test.yaml")
		longOrientation := string(make([]byte, MaxOrientationLength+1))
		content := "page:\n  orientation: \"" + longOrientation + "\"\n"
		if err := os.WriteFile(configPath, []byte(content), 0600); err != nil {
			t.Fatalf("setup WriteFile: %v", err)
		}

		_, err := LoadConfig(configPath)
		if !errors.Is(err, ErrFieldTooLong) {
			t.Errorf("LoadConfig(configPath) error = %v, want ErrFieldTooLong", err)
		}
	})

	t.Run("loads TOC settings", func(t *testing.T) {
		dir := t.TempDir()
		configPath := filepath.Join(dir, "test.yaml")
		content := `toc:
  enabled: true
  title: "Table of Contents"
  maxDepth: 4
`
		if err := os.WriteFile(configPath, []byte(content), 0600); err != nil {
			t.Fatalf("setup WriteFile: %v", err)
		}

		cfg, err := LoadConfig(configPath)
		if err != nil {
			t.Fatalf("LoadConfig(configPath) unexpected error: %v", err)
		}
		if !cfg.TOC.Enabled {
			t.Error("LoadConfig(configPath).TOC.Enabled = false, want true")
		}
		if cfg.TOC.Title != "Table of Contents" {
			t.Errorf("LoadConfig(configPath).TOC.Title = %q, want %q", cfg.TOC.Title, "Table of Contents")
		}
		if cfg.TOC.MaxDepth != 4 {
			t.Errorf("LoadConfig(configPath).TOC.MaxDepth = %d, want %d", cfg.TOC.MaxDepth, 4)
		}
	})

	t.Run("toc.title too long returns ErrFieldTooLong", func(t *testing.T) {
		dir := t.TempDir()
		configPath := filepath.Join(dir, "test.yaml")
		longTitle := string(make([]byte, MaxTOCTitleLength+1))
		content := "toc:\n  title: \"" + longTitle + "\"\n"
		if err := os.WriteFile(configPath, []byte(content), 0600); err != nil {
			t.Fatalf("setup WriteFile: %v", err)
		}

		_, err := LoadConfig(configPath)
		if !errors.Is(err, ErrFieldTooLong) {
			t.Errorf("LoadConfig(configPath) error = %v, want ErrFieldTooLong", err)
		}
	})

	t.Run("toc.maxDepth invalid range returns error", func(t *testing.T) {
		dir := t.TempDir()
		configPath := filepath.Join(dir, "test.yaml")
		content := `toc:
  enabled: true
  maxDepth: 7
`
		if err := os.WriteFile(configPath, []byte(content), 0600); err != nil {
			t.Fatalf("setup WriteFile: %v", err)
		}

		_, err := LoadConfig(configPath)
		if err == nil {
			t.Fatal("LoadConfig(configPath) error = nil, want error")
		}
	})

	t.Run("toc.maxDepth 0 is valid when enabled", func(t *testing.T) {
		dir := t.TempDir()
		configPath := filepath.Join(dir, "test.yaml")
		content := `toc:
  enabled: true
  maxDepth: 0
`
		if err := os.WriteFile(configPath, []byte(content), 0600); err != nil {
			t.Fatalf("setup WriteFile: %v", err)
		}

		cfg, err := LoadConfig(configPath)
		if err != nil {
			t.Fatalf("LoadConfig(configPath) unexpected error: %v", err)
		}
		if cfg.TOC.MaxDepth != 0 {
			t.Errorf("LoadConfig(configPath).TOC.MaxDepth = %d, want 0 (will use default)", cfg.TOC.MaxDepth)
		}
	})

	t.Run("loads author and document settings", func(t *testing.T) {
		dir := t.TempDir()
		configPath := filepath.Join(dir, "test.yaml")
		content := `author:
  name: "John Doe"
  title: "Developer"
  email: "john@example.com"
  organization: "Acme Corp"
document:
  title: "My Document"
  subtitle: "A Subtitle"
  version: "1.0"
  date: "auto"
`
		if err := os.WriteFile(configPath, []byte(content), 0600); err != nil {
			t.Fatalf("setup WriteFile: %v", err)
		}

		cfg, err := LoadConfig(configPath)
		if err != nil {
			t.Fatalf("LoadConfig(configPath) unexpected error: %v", err)
		}
		if cfg.Author.Name != "John Doe" {
			t.Errorf("LoadConfig(configPath).Author.Name = %q, want %q", cfg.Author.Name, "John Doe")
		}
		if cfg.Author.Organization != "Acme Corp" {
			t.Errorf("LoadConfig(configPath).Author.Organization = %q, want %q", cfg.Author.Organization, "Acme Corp")
		}
		if cfg.Document.Title != "My Document" {
			t.Errorf("LoadConfig(configPath).Document.Title = %q, want %q", cfg.Document.Title, "My Document")
		}
		if cfg.Document.Date != "auto" {
			t.Errorf("LoadConfig(configPath).Document.Date = %q, want %q", cfg.Document.Date, "auto")
		}
	})
}

func TestConfig_Validate_Page(t *testing.T) {
	t.Parallel()

	t.Run("empty size and orientation passes (uses defaults)", func(t *testing.T) {
		t.Parallel()
		cfg := &Config{Page: PageConfig{}}
		err := cfg.Validate()
		if err != nil {
			t.Errorf("Config.Validate() unexpected error: %v", err)
		}
	})

	t.Run("valid size letter passes", func(t *testing.T) {
		t.Parallel()
		cfg := &Config{Page: PageConfig{Size: "letter"}}
		err := cfg.Validate()
		if err != nil {
			t.Errorf("Config.Validate() unexpected error: %v", err)
		}
	})

	t.Run("valid size a4 passes", func(t *testing.T) {
		t.Parallel()
		cfg := &Config{Page: PageConfig{Size: "a4"}}
		err := cfg.Validate()
		if err != nil {
			t.Errorf("Config.Validate() unexpected error: %v", err)
		}
	})

	t.Run("valid size legal passes", func(t *testing.T) {
		t.Parallel()
		cfg := &Config{Page: PageConfig{Size: "legal"}}
		err := cfg.Validate()
		if err != nil {
			t.Errorf("Config.Validate() unexpected error: %v", err)
		}
	})

	t.Run("size case insensitive", func(t *testing.T) {
		t.Parallel()
		cfg := &Config{Page: PageConfig{Size: "A4"}}
		err := cfg.Validate()
		if err != nil {
			t.Errorf("Config.Validate() unexpected error: %v", err)
		}
	})

	t.Run("invalid size returns error", func(t *testing.T) {
		t.Parallel()
		cfg := &Config{Page: PageConfig{Size: "tabloid"}}
		err := cfg.Validate()
		if err == nil {
			t.Fatal("Config.Validate() error = nil, want error")
		}
		if !strings.Contains(err.Error(), "page.size") {
			t.Errorf("Config.Validate() error should mention page.size, got: %v", err)
		}
	})

	t.Run("valid orientation portrait passes", func(t *testing.T) {
		t.Parallel()
		cfg := &Config{Page: PageConfig{Orientation: "portrait"}}
		err := cfg.Validate()
		if err != nil {
			t.Errorf("Config.Validate() unexpected error: %v", err)
		}
	})

	t.Run("valid orientation landscape passes", func(t *testing.T) {
		t.Parallel()
		cfg := &Config{Page: PageConfig{Orientation: "landscape"}}
		err := cfg.Validate()
		if err != nil {
			t.Errorf("Config.Validate() unexpected error: %v", err)
		}
	})

	t.Run("orientation case insensitive", func(t *testing.T) {
		t.Parallel()
		cfg := &Config{Page: PageConfig{Orientation: "LANDSCAPE"}}
		err := cfg.Validate()
		if err != nil {
			t.Errorf("Config.Validate() unexpected error: %v", err)
		}
	})

	t.Run("invalid orientation returns error", func(t *testing.T) {
		t.Parallel()
		cfg := &Config{Page: PageConfig{Orientation: "diagonal"}}
		err := cfg.Validate()
		if err == nil {
			t.Fatal("Config.Validate() error = nil, want error")
		}
		if !strings.Contains(err.Error(), "page.orientation") {
			t.Errorf("Config.Validate() error should mention page.orientation, got: %v", err)
		}
	})

	t.Run("valid size and orientation together passes", func(t *testing.T) {
		t.Parallel()
		cfg := &Config{Page: PageConfig{Size: "a4", Orientation: "landscape", Margin: 1.0}}
		err := cfg.Validate()
		if err != nil {
			t.Errorf("Config.Validate() unexpected error: %v", err)
		}
	})
}

func TestConfig_Validate_PageBreaks(t *testing.T) {
	t.Run("pageBreaks disabled passes validation", func(t *testing.T) {
		cfg := &Config{PageBreaks: PageBreaksConfig{Enabled: false}}
		err := cfg.Validate()
		if err != nil {
			t.Errorf("Config.Validate() unexpected error: %v", err)
		}
	})

	t.Run("pageBreaks enabled with valid orphans passes", func(t *testing.T) {
		cfg := &Config{PageBreaks: PageBreaksConfig{Enabled: true, Orphans: 3}}
		err := cfg.Validate()
		if err != nil {
			t.Errorf("Config.Validate() unexpected error: %v", err)
		}
	})

	t.Run("pageBreaks enabled with valid widows passes", func(t *testing.T) {
		cfg := &Config{PageBreaks: PageBreaksConfig{Enabled: true, Widows: 4}}
		err := cfg.Validate()
		if err != nil {
			t.Errorf("Config.Validate() unexpected error: %v", err)
		}
	})

	t.Run("pageBreaks orphans 0 passes (uses default)", func(t *testing.T) {
		cfg := &Config{PageBreaks: PageBreaksConfig{Enabled: true, Orphans: 0}}
		err := cfg.Validate()
		if err != nil {
			t.Errorf("Config.Validate() unexpected error: %v", err)
		}
	})

	t.Run("pageBreaks widows 0 passes (uses default)", func(t *testing.T) {
		cfg := &Config{PageBreaks: PageBreaksConfig{Enabled: true, Widows: 0}}
		err := cfg.Validate()
		if err != nil {
			t.Errorf("Config.Validate() unexpected error: %v", err)
		}
	})

	t.Run("pageBreaks orphans at min boundary passes", func(t *testing.T) {
		cfg := &Config{PageBreaks: PageBreaksConfig{Enabled: true, Orphans: 1}}
		err := cfg.Validate()
		if err != nil {
			t.Errorf("Config.Validate() unexpected error: %v", err)
		}
	})

	t.Run("pageBreaks orphans at max boundary passes", func(t *testing.T) {
		cfg := &Config{PageBreaks: PageBreaksConfig{Enabled: true, Orphans: 5}}
		err := cfg.Validate()
		if err != nil {
			t.Errorf("Config.Validate() unexpected error: %v", err)
		}
	})

	t.Run("pageBreaks widows at min boundary passes", func(t *testing.T) {
		cfg := &Config{PageBreaks: PageBreaksConfig{Enabled: true, Widows: 1}}
		err := cfg.Validate()
		if err != nil {
			t.Errorf("Config.Validate() unexpected error: %v", err)
		}
	})

	t.Run("pageBreaks widows at max boundary passes", func(t *testing.T) {
		cfg := &Config{PageBreaks: PageBreaksConfig{Enabled: true, Widows: 5}}
		err := cfg.Validate()
		if err != nil {
			t.Errorf("Config.Validate() unexpected error: %v", err)
		}
	})

	t.Run("pageBreaks orphans below 1 returns error", func(t *testing.T) {
		cfg := &Config{PageBreaks: PageBreaksConfig{Orphans: -1}}
		err := cfg.Validate()
		if err == nil {
			t.Fatal("Config.Validate() error = nil, want error")
		}
	})

	t.Run("pageBreaks orphans above 5 returns error", func(t *testing.T) {
		cfg := &Config{PageBreaks: PageBreaksConfig{Orphans: 6}}
		err := cfg.Validate()
		if err == nil {
			t.Fatal("Config.Validate() error = nil, want error")
		}
	})

	t.Run("pageBreaks widows below 1 returns error", func(t *testing.T) {
		cfg := &Config{PageBreaks: PageBreaksConfig{Widows: -1}}
		err := cfg.Validate()
		if err == nil {
			t.Fatal("Config.Validate() error = nil, want error")
		}
	})

	t.Run("pageBreaks widows above 5 returns error", func(t *testing.T) {
		cfg := &Config{PageBreaks: PageBreaksConfig{Widows: 6}}
		err := cfg.Validate()
		if err == nil {
			t.Fatal("Config.Validate() error = nil, want error")
		}
	})

	t.Run("pageBreaks all heading breaks valid", func(t *testing.T) {
		cfg := &Config{PageBreaks: PageBreaksConfig{
			Enabled:  true,
			BeforeH1: true,
			BeforeH2: true,
			BeforeH3: true,
			Orphans:  2,
			Widows:   2,
		}}
		err := cfg.Validate()
		if err != nil {
			t.Errorf("Config.Validate() unexpected error: %v", err)
		}
	})
}

func TestLoadConfig_PageBreaks(t *testing.T) {
	t.Run("loads page breaks settings", func(t *testing.T) {
		dir := t.TempDir()
		configPath := filepath.Join(dir, "test.yaml")
		content := `pageBreaks:
  enabled: true
  beforeH1: true
  beforeH2: false
  beforeH3: true
  orphans: 3
  widows: 4
`
		if err := os.WriteFile(configPath, []byte(content), 0600); err != nil {
			t.Fatalf("setup WriteFile: %v", err)
		}

		cfg, err := LoadConfig(configPath)
		if err != nil {
			t.Fatalf("LoadConfig(configPath) unexpected error: %v", err)
		}
		if !cfg.PageBreaks.Enabled {
			t.Error("LoadConfig(configPath).PageBreaks.Enabled = false, want true")
		}
		if !cfg.PageBreaks.BeforeH1 {
			t.Error("LoadConfig(configPath).PageBreaks.BeforeH1 = false, want true")
		}
		if cfg.PageBreaks.BeforeH2 {
			t.Error("LoadConfig(configPath).PageBreaks.BeforeH2 = true, want false")
		}
		if !cfg.PageBreaks.BeforeH3 {
			t.Error("LoadConfig(configPath).PageBreaks.BeforeH3 = false, want true")
		}
		if cfg.PageBreaks.Orphans != 3 {
			t.Errorf("LoadConfig(configPath).PageBreaks.Orphans = %d, want 3", cfg.PageBreaks.Orphans)
		}
		if cfg.PageBreaks.Widows != 4 {
			t.Errorf("LoadConfig(configPath).PageBreaks.Widows = %d, want 4", cfg.PageBreaks.Widows)
		}
	})

	t.Run("pageBreaks.orphans invalid range returns error", func(t *testing.T) {
		dir := t.TempDir()
		configPath := filepath.Join(dir, "test.yaml")
		content := `pageBreaks:
  enabled: true
  orphans: 10
`
		if err := os.WriteFile(configPath, []byte(content), 0600); err != nil {
			t.Fatalf("setup WriteFile: %v", err)
		}

		_, err := LoadConfig(configPath)
		if err == nil {
			t.Fatal("LoadConfig(configPath) error = nil, want error")
		}
	})

	t.Run("pageBreaks.widows invalid range returns error", func(t *testing.T) {
		dir := t.TempDir()
		configPath := filepath.Join(dir, "test.yaml")
		content := `pageBreaks:
  enabled: true
  widows: 10
`
		if err := os.WriteFile(configPath, []byte(content), 0600); err != nil {
			t.Fatalf("setup WriteFile: %v", err)
		}

		_, err := LoadConfig(configPath)
		if err == nil {
			t.Fatal("LoadConfig(configPath) error = nil, want error")
		}
	})
}

func TestConfig_Validate_TOC(t *testing.T) {
	t.Run("toc disabled passes validation", func(t *testing.T) {
		cfg := &Config{TOC: TOCConfig{Enabled: false}}
		err := cfg.Validate()
		if err != nil {
			t.Errorf("Config.Validate() unexpected error: %v", err)
		}
	})

	t.Run("toc enabled with valid depth passes", func(t *testing.T) {
		cfg := &Config{TOC: TOCConfig{Enabled: true, MaxDepth: 3}}
		err := cfg.Validate()
		if err != nil {
			t.Errorf("Config.Validate() unexpected error: %v", err)
		}
	})

	t.Run("toc enabled with depth 0 passes (uses default)", func(t *testing.T) {
		cfg := &Config{TOC: TOCConfig{Enabled: true, MaxDepth: 0}}
		err := cfg.Validate()
		if err != nil {
			t.Errorf("Config.Validate() unexpected error: %v", err)
		}
	})

	t.Run("toc.title at max length passes", func(t *testing.T) {
		cfg := &Config{TOC: TOCConfig{Title: string(make([]byte, MaxTOCTitleLength))}}
		err := cfg.Validate()
		if err != nil {
			t.Errorf("Config.Validate() unexpected error: %v", err)
		}
	})

	t.Run("toc.title too long returns error", func(t *testing.T) {
		cfg := &Config{TOC: TOCConfig{Title: string(make([]byte, MaxTOCTitleLength+1))}}
		err := cfg.Validate()
		if !errors.Is(err, ErrFieldTooLong) {
			t.Errorf("Config.Validate() error = %v, want ErrFieldTooLong", err)
		}
	})

	t.Run("toc enabled with depth 7 returns error", func(t *testing.T) {
		cfg := &Config{TOC: TOCConfig{Enabled: true, MaxDepth: 7}}
		err := cfg.Validate()
		if err == nil {
			t.Fatal("Config.Validate() error = nil, want error")
		}
	})

	t.Run("toc enabled with negative depth returns error", func(t *testing.T) {
		cfg := &Config{TOC: TOCConfig{Enabled: true, MaxDepth: -1}}
		err := cfg.Validate()
		if err == nil {
			t.Fatal("Config.Validate() error = nil, want error")
		}
	})

	t.Run("toc enabled with valid minDepth passes", func(t *testing.T) {
		cfg := &Config{TOC: TOCConfig{Enabled: true, MinDepth: 2, MaxDepth: 4}}
		err := cfg.Validate()
		if err != nil {
			t.Errorf("Config.Validate() unexpected error: %v", err)
		}
	})

	t.Run("toc enabled with minDepth 0 passes (uses default)", func(t *testing.T) {
		cfg := &Config{TOC: TOCConfig{Enabled: true, MinDepth: 0, MaxDepth: 3}}
		err := cfg.Validate()
		if err != nil {
			t.Errorf("Config.Validate() unexpected error: %v", err)
		}
	})

	t.Run("toc enabled with minDepth 7 returns error", func(t *testing.T) {
		cfg := &Config{TOC: TOCConfig{Enabled: true, MinDepth: 7}}
		err := cfg.Validate()
		if err == nil {
			t.Fatal("Config.Validate() error = nil, want error")
		}
	})

	t.Run("toc enabled with negative minDepth returns error", func(t *testing.T) {
		cfg := &Config{TOC: TOCConfig{Enabled: true, MinDepth: -1}}
		err := cfg.Validate()
		if err == nil {
			t.Fatal("Config.Validate() error = nil, want error")
		}
	})

	t.Run("toc enabled with minDepth greater than maxDepth returns error", func(t *testing.T) {
		cfg := &Config{TOC: TOCConfig{Enabled: true, MinDepth: 4, MaxDepth: 2}}
		err := cfg.Validate()
		if err == nil {
			t.Fatal("Config.Validate() error = nil, want error")
		}
	})

	t.Run("toc enabled with minDepth equal to maxDepth passes", func(t *testing.T) {
		cfg := &Config{TOC: TOCConfig{Enabled: true, MinDepth: 3, MaxDepth: 3}}
		err := cfg.Validate()
		if err != nil {
			t.Errorf("Config.Validate() unexpected error: %v", err)
		}
	})
}

func TestConfig_Validate_Watermark(t *testing.T) {
	t.Parallel()

	t.Run("watermark disabled skips validation", func(t *testing.T) {
		t.Parallel()
		cfg := &Config{Watermark: WatermarkConfig{Enabled: false}}
		err := cfg.Validate()
		if err != nil {
			t.Errorf("Config.Validate() unexpected error: %v", err)
		}
	})

	t.Run("watermark enabled without text returns error", func(t *testing.T) {
		t.Parallel()
		cfg := &Config{Watermark: WatermarkConfig{
			Enabled: true,
			Text:    "",
			Opacity: 0.5,
			Angle:   -45,
		}}
		err := cfg.Validate()
		if err == nil {
			t.Fatal("Config.Validate() error = nil, want error")
		}
	})

	t.Run("watermark.text too long returns error", func(t *testing.T) {
		t.Parallel()
		cfg := &Config{Watermark: WatermarkConfig{
			Enabled: true,
			Text:    string(make([]byte, MaxWatermarkTextLength+1)),
			Opacity: 0.5,
			Angle:   -45,
		}}
		err := cfg.Validate()
		if !errors.Is(err, ErrFieldTooLong) {
			t.Errorf("Config.Validate() error = %v, want ErrFieldTooLong", err)
		}
	})

	t.Run("watermark.color too long returns error", func(t *testing.T) {
		t.Parallel()
		cfg := &Config{Watermark: WatermarkConfig{
			Enabled: true,
			Text:    "DRAFT",
			Color:   string(make([]byte, MaxWatermarkColorLength+1)),
			Opacity: 0.5,
			Angle:   -45,
		}}
		err := cfg.Validate()
		if !errors.Is(err, ErrFieldTooLong) {
			t.Errorf("Config.Validate() error = %v, want ErrFieldTooLong", err)
		}
	})

	t.Run("watermark.opacity below minimum returns error", func(t *testing.T) {
		t.Parallel()
		cfg := &Config{Watermark: WatermarkConfig{
			Enabled: true,
			Text:    "DRAFT",
			Opacity: -0.1,
			Angle:   -45,
		}}
		err := cfg.Validate()
		if err == nil {
			t.Fatal("Config.Validate() error = nil, want error")
		}
	})

	t.Run("watermark.opacity above maximum returns error", func(t *testing.T) {
		t.Parallel()
		cfg := &Config{Watermark: WatermarkConfig{
			Enabled: true,
			Text:    "DRAFT",
			Opacity: 1.1,
			Angle:   -45,
		}}
		err := cfg.Validate()
		if err == nil {
			t.Fatal("Config.Validate() error = nil, want error")
		}
	})

	t.Run("watermark.angle below minimum returns error", func(t *testing.T) {
		t.Parallel()
		cfg := &Config{Watermark: WatermarkConfig{
			Enabled: true,
			Text:    "DRAFT",
			Opacity: 0.5,
			Angle:   -181,
		}}
		err := cfg.Validate()
		if err == nil {
			t.Fatal("Config.Validate() error = nil, want error")
		}
	})

	t.Run("watermark.angle above maximum returns error", func(t *testing.T) {
		t.Parallel()
		cfg := &Config{Watermark: WatermarkConfig{
			Enabled: true,
			Text:    "DRAFT",
			Opacity: 0.5,
			Angle:   181,
		}}
		err := cfg.Validate()
		if err == nil {
			t.Fatal("Config.Validate() error = nil, want error")
		}
	})

	t.Run("valid watermark config passes", func(t *testing.T) {
		t.Parallel()
		cfg := &Config{Watermark: WatermarkConfig{
			Enabled: true,
			Text:    "CONFIDENTIAL",
			Color:   "#888888",
			Opacity: 0.5,
			Angle:   -45,
		}}
		err := cfg.Validate()
		if err != nil {
			t.Errorf("Config.Validate() unexpected error: %v", err)
		}
	})
}

func TestConfig_Validate_Footer(t *testing.T) {
	t.Parallel()

	t.Run("footer.position invalid returns error", func(t *testing.T) {
		t.Parallel()
		cfg := &Config{Footer: FooterConfig{
			Enabled:  true,
			Position: "invalid",
		}}
		err := cfg.Validate()
		if err == nil {
			t.Fatal("Config.Validate() error = nil, want error")
		}
	})

	t.Run("footer.position uppercase valid", func(t *testing.T) {
		t.Parallel()
		cfg := &Config{Footer: FooterConfig{
			Enabled:  true,
			Position: "LEFT",
		}}
		err := cfg.Validate()
		if err != nil {
			t.Errorf("Config.Validate() unexpected error: %v", err)
		}
	})

	t.Run("footer.position center valid", func(t *testing.T) {
		t.Parallel()
		cfg := &Config{Footer: FooterConfig{
			Enabled:  true,
			Position: "center",
		}}
		err := cfg.Validate()
		if err != nil {
			t.Errorf("Config.Validate() unexpected error: %v", err)
		}
	})
}

func TestConfig_Validate_Author(t *testing.T) {
	t.Parallel()

	t.Run("author.title too long returns error", func(t *testing.T) {
		t.Parallel()
		cfg := &Config{Author: AuthorConfig{
			Title: string(make([]byte, MaxTitleLength+1)),
		}}
		err := cfg.Validate()
		if !errors.Is(err, ErrFieldTooLong) {
			t.Errorf("Config.Validate() error = %v, want ErrFieldTooLong", err)
		}
	})

	t.Run("author.organization too long returns error", func(t *testing.T) {
		t.Parallel()
		cfg := &Config{Author: AuthorConfig{
			Organization: string(make([]byte, MaxOrganizationLength+1)),
		}}
		err := cfg.Validate()
		if !errors.Is(err, ErrFieldTooLong) {
			t.Errorf("Config.Validate() error = %v, want ErrFieldTooLong", err)
		}
	})

	t.Run("author.phone too long returns error", func(t *testing.T) {
		t.Parallel()
		cfg := &Config{Author: AuthorConfig{
			Phone: string(make([]byte, MaxPhoneLength+1)),
		}}
		err := cfg.Validate()
		if !errors.Is(err, ErrFieldTooLong) {
			t.Errorf("Config.Validate() error = %v, want ErrFieldTooLong", err)
		}
	})

	t.Run("author.address too long returns error", func(t *testing.T) {
		t.Parallel()
		cfg := &Config{Author: AuthorConfig{
			Address: string(make([]byte, MaxAddressLength+1)),
		}}
		err := cfg.Validate()
		if !errors.Is(err, ErrFieldTooLong) {
			t.Errorf("Config.Validate() error = %v, want ErrFieldTooLong", err)
		}
	})

	t.Run("author.department too long returns error", func(t *testing.T) {
		t.Parallel()
		cfg := &Config{Author: AuthorConfig{
			Department: string(make([]byte, MaxDepartmentLength+1)),
		}}
		err := cfg.Validate()
		if !errors.Is(err, ErrFieldTooLong) {
			t.Errorf("Config.Validate() error = %v, want ErrFieldTooLong", err)
		}
	})

	t.Run("valid extended author fields pass validation", func(t *testing.T) {
		t.Parallel()
		cfg := &Config{Author: AuthorConfig{
			Name:       "John Doe",
			Phone:      "+1-555-123-4567",
			Address:    "123 Main St\nCity, State 12345",
			Department: "Engineering",
		}}
		err := cfg.Validate()
		if err != nil {
			t.Errorf("Config.Validate() unexpected error: %v", err)
		}
	})
}

func TestConfig_Validate_Document(t *testing.T) {
	t.Parallel()

	t.Run("document.title too long returns error", func(t *testing.T) {
		t.Parallel()
		cfg := &Config{Document: DocumentConfig{
			Title: string(make([]byte, MaxDocTitleLength+1)),
		}}
		err := cfg.Validate()
		if !errors.Is(err, ErrFieldTooLong) {
			t.Errorf("Config.Validate() error = %v, want ErrFieldTooLong", err)
		}
	})

	t.Run("document.subtitle too long returns error", func(t *testing.T) {
		t.Parallel()
		cfg := &Config{Document: DocumentConfig{
			Subtitle: string(make([]byte, MaxSubtitleLength+1)),
		}}
		err := cfg.Validate()
		if !errors.Is(err, ErrFieldTooLong) {
			t.Errorf("Config.Validate() error = %v, want ErrFieldTooLong", err)
		}
	})

	t.Run("document.date too long returns error", func(t *testing.T) {
		t.Parallel()
		cfg := &Config{Document: DocumentConfig{
			Date: string(make([]byte, MaxDateLength+1)),
		}}
		err := cfg.Validate()
		if !errors.Is(err, ErrFieldTooLong) {
			t.Errorf("Config.Validate() error = %v, want ErrFieldTooLong", err)
		}
	})

	t.Run("document.date auto passthrough is valid", func(t *testing.T) {
		t.Parallel()
		cfg := &Config{Document: DocumentConfig{
			Date: "auto",
		}}
		err := cfg.Validate()
		if err != nil {
			t.Errorf("Config.Validate() unexpected error: %v", err)
		}
	})

	t.Run("document.date auto:FORMAT with valid format passes", func(t *testing.T) {
		t.Parallel()
		cfg := &Config{Document: DocumentConfig{
			Date: "auto:DD/MM/YYYY",
		}}
		err := cfg.Validate()
		if err != nil {
			t.Errorf("Config.Validate() unexpected error: %v", err)
		}
	})

	t.Run("document.date auto:preset with valid preset passes", func(t *testing.T) {
		t.Parallel()
		cfg := &Config{Document: DocumentConfig{
			Date: "auto:european",
		}}
		err := cfg.Validate()
		if err != nil {
			t.Errorf("Config.Validate() unexpected error: %v", err)
		}
	})

	t.Run("document.date auto: with empty format returns error", func(t *testing.T) {
		t.Parallel()
		cfg := &Config{Document: DocumentConfig{
			Date: "auto:",
		}}
		err := cfg.Validate()
		if err == nil {
			t.Fatal("expected error for empty format after auto:")
		}
	})

	t.Run("document.date literal value passes", func(t *testing.T) {
		t.Parallel()
		cfg := &Config{Document: DocumentConfig{
			Date: "2024-01-15",
		}}
		err := cfg.Validate()
		if err != nil {
			t.Errorf("Config.Validate() unexpected error: %v", err)
		}
	})

	t.Run("document.clientName too long returns error", func(t *testing.T) {
		t.Parallel()
		cfg := &Config{Document: DocumentConfig{
			ClientName: string(make([]byte, MaxClientNameLength+1)),
		}}
		err := cfg.Validate()
		if !errors.Is(err, ErrFieldTooLong) {
			t.Errorf("Config.Validate() error = %v, want ErrFieldTooLong", err)
		}
	})

	t.Run("document.projectName too long returns error", func(t *testing.T) {
		t.Parallel()
		cfg := &Config{Document: DocumentConfig{
			ProjectName: string(make([]byte, MaxProjectNameLength+1)),
		}}
		err := cfg.Validate()
		if !errors.Is(err, ErrFieldTooLong) {
			t.Errorf("Config.Validate() error = %v, want ErrFieldTooLong", err)
		}
	})

	t.Run("document.documentType too long returns error", func(t *testing.T) {
		t.Parallel()
		cfg := &Config{Document: DocumentConfig{
			DocumentType: string(make([]byte, MaxDocumentTypeLength+1)),
		}}
		err := cfg.Validate()
		if !errors.Is(err, ErrFieldTooLong) {
			t.Errorf("Config.Validate() error = %v, want ErrFieldTooLong", err)
		}
	})

	t.Run("document.documentID too long returns error", func(t *testing.T) {
		t.Parallel()
		cfg := &Config{Document: DocumentConfig{
			DocumentID: string(make([]byte, MaxDocumentIDLength+1)),
		}}
		err := cfg.Validate()
		if !errors.Is(err, ErrFieldTooLong) {
			t.Errorf("Config.Validate() error = %v, want ErrFieldTooLong", err)
		}
	})

	t.Run("document.description too long returns error", func(t *testing.T) {
		t.Parallel()
		cfg := &Config{Document: DocumentConfig{
			Description: string(make([]byte, MaxDescriptionLength+1)),
		}}
		err := cfg.Validate()
		if !errors.Is(err, ErrFieldTooLong) {
			t.Errorf("Config.Validate() error = %v, want ErrFieldTooLong", err)
		}
	})

	t.Run("valid extended document fields pass validation", func(t *testing.T) {
		t.Parallel()
		cfg := &Config{Document: DocumentConfig{
			Title:        "Design Document",
			ClientName:   "Acme Corporation",
			ProjectName:  "Project Phoenix",
			DocumentType: "Technical Specification",
			DocumentID:   "DOC-2024-001",
			Description:  "Technical specification for the Phoenix system",
		}}
		err := cfg.Validate()
		if err != nil {
			t.Errorf("Config.Validate() unexpected error: %v", err)
		}
	})
}

func TestConfig_Validate_Signature(t *testing.T) {
	t.Parallel()

	t.Run("signature.imagePath too long returns error", func(t *testing.T) {
		t.Parallel()
		cfg := &Config{Signature: SignatureConfig{
			ImagePath: string(make([]byte, MaxURLLength+1)),
		}}
		err := cfg.Validate()
		if !errors.Is(err, ErrFieldTooLong) {
			t.Errorf("Config.Validate() error = %v, want ErrFieldTooLong", err)
		}
	})

	t.Run("signature.links label too long returns error", func(t *testing.T) {
		t.Parallel()
		cfg := &Config{Signature: SignatureConfig{
			Links: []Link{
				{Label: string(make([]byte, MaxLabelLength+1)), URL: "https://example.com"},
			},
		}}
		err := cfg.Validate()
		if !errors.Is(err, ErrFieldTooLong) {
			t.Errorf("Config.Validate() error = %v, want ErrFieldTooLong", err)
		}
	})
}

func TestConfig_Validate_Cover(t *testing.T) {
	t.Parallel()

	t.Run("cover.logo too long returns error", func(t *testing.T) {
		t.Parallel()
		cfg := &Config{Cover: CoverConfig{
			Logo: string(make([]byte, MaxURLLength+1)),
		}}
		err := cfg.Validate()
		if !errors.Is(err, ErrFieldTooLong) {
			t.Errorf("Config.Validate() error = %v, want ErrFieldTooLong", err)
		}
	})

	t.Run("valid cover config passes", func(t *testing.T) {
		t.Parallel()
		cfg := &Config{Cover: CoverConfig{
			Enabled: true,
			Logo:    "https://example.com/logo.png",
		}}
		err := cfg.Validate()
		if err != nil {
			t.Errorf("Config.Validate() unexpected error: %v", err)
		}
	})
}

func TestConfig_Validate_Assets(t *testing.T) {
	t.Parallel()

	t.Run("empty basePath is valid", func(t *testing.T) {
		t.Parallel()
		cfg := &Config{Assets: AssetsConfig{BasePath: ""}}
		err := cfg.Validate()
		if err != nil {
			t.Errorf("Config.Validate() unexpected error: %v", err)
		}
	})

	t.Run("valid directory basePath is valid", func(t *testing.T) {
		t.Parallel()
		tmpDir := t.TempDir()
		cfg := &Config{Assets: AssetsConfig{BasePath: tmpDir}}
		err := cfg.Validate()
		if err != nil {
			t.Errorf("Config.Validate() unexpected error: %v", err)
		}
	})

	t.Run("nonexistent basePath returns error", func(t *testing.T) {
		t.Parallel()
		cfg := &Config{Assets: AssetsConfig{BasePath: "/nonexistent/path/xyz123"}}
		err := cfg.Validate()
		if err == nil {
			t.Error("Config.Validate() error = nil, want error")
		}
		if !strings.Contains(err.Error(), "does not exist") {
			t.Errorf("Config.Validate() error should mention 'does not exist', got: %v", err)
		}
	})

	t.Run("file instead of directory returns error", func(t *testing.T) {
		t.Parallel()
		tmpDir := t.TempDir()
		filePath := filepath.Join(tmpDir, "notadir.txt")
		if err := os.WriteFile(filePath, []byte("test"), 0644); err != nil {
			t.Fatalf("failed to create test file: %v", err)
		}

		cfg := &Config{Assets: AssetsConfig{BasePath: filePath}}
		err := cfg.Validate()
		if err == nil {
			t.Error("Config.Validate() error = nil, want error")
		}
		if !strings.Contains(err.Error(), "not a directory") {
			t.Errorf("Config.Validate() error should mention 'not a directory', got: %v", err)
		}
	})
}

func TestConfig_Validate_Timeout(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		timeout   string
		wantErr   bool
		errSubstr string
	}{
		{
			name:    "empty timeout is valid",
			timeout: "",
			wantErr: false,
		},
		{
			name:    "valid seconds",
			timeout: "30s",
			wantErr: false,
		},
		{
			name:    "valid minutes",
			timeout: "2m",
			wantErr: false,
		},
		{
			name:    "valid combined",
			timeout: "1m30s",
			wantErr: false,
		},
		{
			name:    "valid fractional",
			timeout: "0.5s",
			wantErr: false,
		},
		{
			name:    "valid long timeout",
			timeout: "30m",
			wantErr: false,
		},
		{
			name:      "invalid format",
			timeout:   "abc",
			wantErr:   true,
			errSubstr: "invalid duration",
		},
		{
			name:      "missing unit",
			timeout:   "30",
			wantErr:   true,
			errSubstr: "invalid duration",
		},
		{
			name:      "negative duration",
			timeout:   "-5s",
			wantErr:   true,
			errSubstr: "must be positive",
		},
		{
			name:      "zero duration",
			timeout:   "0s",
			wantErr:   true,
			errSubstr: "must be positive",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			cfg := &Config{Timeout: tt.timeout}
			err := cfg.Validate()
			if tt.wantErr {
				if err == nil {
					t.Fatalf("Config.Validate() error = nil, want error")
					return
				}
				if tt.errSubstr != "" && !strings.Contains(err.Error(), tt.errSubstr) {
					t.Errorf("Config.Validate() error should contain %q, got: %v", tt.errSubstr, err)
				}
			} else {
				if err != nil {
					t.Errorf("Config.Validate() unexpected error: %v", err)
				}
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestLoadConfig_Timeout - YAML loading with timeout field
// ---------------------------------------------------------------------------

func TestLoadConfig_Timeout(t *testing.T) {
	t.Parallel()

	t.Run("loads valid timeout from YAML", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		configPath := filepath.Join(dir, "test.yaml")
		content := `timeout: "2m"
style: "default"
`
		if err := os.WriteFile(configPath, []byte(content), 0600); err != nil {
			t.Fatalf("setup WriteFile: %v", err)
		}

		cfg, err := LoadConfig(configPath)
		if err != nil {
			t.Fatalf("LoadConfig(configPath) unexpected error: %v", err)
		}
		if cfg.Timeout != "2m" {
			t.Errorf("LoadConfig(configPath).Timeout = %q, want %q", cfg.Timeout, "2m")
		}
	})

	t.Run("loads combined duration from YAML", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		configPath := filepath.Join(dir, "test.yaml")
		content := `timeout: "1m30s"
`
		if err := os.WriteFile(configPath, []byte(content), 0600); err != nil {
			t.Fatalf("setup WriteFile: %v", err)
		}

		cfg, err := LoadConfig(configPath)
		if err != nil {
			t.Fatalf("LoadConfig(configPath) unexpected error: %v", err)
		}
		if cfg.Timeout != "1m30s" {
			t.Errorf("LoadConfig(configPath).Timeout = %q, want %q", cfg.Timeout, "1m30s")
		}
	})

	t.Run("rejects invalid timeout in YAML", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		configPath := filepath.Join(dir, "test.yaml")
		content := `timeout: "invalid"
`
		if err := os.WriteFile(configPath, []byte(content), 0600); err != nil {
			t.Fatalf("setup WriteFile: %v", err)
		}

		_, err := LoadConfig(configPath)
		if err == nil {
			t.Error("LoadConfig(configPath) error = nil, want error")
		}
		if !strings.Contains(err.Error(), "invalid duration") {
			t.Errorf("LoadConfig(configPath) error should mention 'invalid duration', got: %v", err)
		}
	})

	t.Run("rejects negative timeout in YAML", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		configPath := filepath.Join(dir, "test.yaml")
		content := `timeout: "-5s"
`
		if err := os.WriteFile(configPath, []byte(content), 0600); err != nil {
			t.Fatalf("setup WriteFile: %v", err)
		}

		_, err := LoadConfig(configPath)
		if err == nil {
			t.Error("LoadConfig(configPath) error = nil, want error")
		}
		if !strings.Contains(err.Error(), "must be positive") {
			t.Errorf("LoadConfig(configPath) error should mention 'must be positive', got: %v", err)
		}
	})

	t.Run("empty timeout is valid", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		configPath := filepath.Join(dir, "test.yaml")
		content := `style: "default"
`
		if err := os.WriteFile(configPath, []byte(content), 0600); err != nil {
			t.Fatalf("setup WriteFile: %v", err)
		}

		cfg, err := LoadConfig(configPath)
		if err != nil {
			t.Fatalf("LoadConfig(configPath) unexpected error: %v", err)
		}
		if cfg.Timeout != "" {
			t.Errorf("LoadConfig(configPath).Timeout = %q, want empty string", cfg.Timeout)
		}
	})
}
