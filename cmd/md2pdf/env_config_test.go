package main

// Notes:
// - loadEnvConfig: we test all 16 environment variables across 3 tiers.
//   Invalid/negative values for timeout and workers are tested to verify
//   graceful handling (ignored, not errors).
// - warnUnknownEnvVars: we test typo detection and that known vars don't warn.
// - applyEnvConfig: we test priority behavior (env doesn't override config)
//   and auto-enable behavior for watermark and cover.
// - Tests use t.Setenv() which prevents t.Parallel() at parent level.
// These are acceptable gaps: we test observable behavior, not implementation details.

import (
	"bytes"
	"testing"
	"time"

	"github.com/alnah/go-md2pdf/internal/config"
)

// ---------------------------------------------------------------------------
// TestLoadEnvConfig - Environment variable loading
// ---------------------------------------------------------------------------

func TestLoadEnvConfig(t *testing.T) {
	t.Run("tier 1 essential variables", func(t *testing.T) {
		t.Setenv("PICOLOOM_CONFIG", "/path/to/config.yaml")
		t.Setenv("PICOLOOM_STYLE", "technical")
		t.Setenv("PICOLOOM_TIMEOUT", "2m")

		cfg := loadEnvConfig()

		if cfg.ConfigPath != "/path/to/config.yaml" {
			t.Errorf("loadEnvConfig() ConfigPath = %q, want /path/to/config.yaml", cfg.ConfigPath)
		}
		if cfg.Style != "technical" {
			t.Errorf("loadEnvConfig() Style = %q, want technical", cfg.Style)
		}
		if cfg.Timeout != 2*time.Minute {
			t.Errorf("loadEnvConfig() Timeout = %v, want 2m", cfg.Timeout)
		}
	})

	t.Run("tier 2 I/O and identity variables", func(t *testing.T) {
		t.Setenv("PICOLOOM_INPUT_DIR", "/input")
		t.Setenv("PICOLOOM_OUTPUT_DIR", "/output")
		t.Setenv("PICOLOOM_AUTHOR_NAME", "John Doe")
		t.Setenv("PICOLOOM_AUTHOR_ORG", "Acme Corp")
		t.Setenv("PICOLOOM_AUTHOR_EMAIL", "john@acme.com")

		cfg := loadEnvConfig()

		if cfg.InputDir != "/input" {
			t.Errorf("loadEnvConfig() InputDir = %q, want /input", cfg.InputDir)
		}
		if cfg.OutputDir != "/output" {
			t.Errorf("loadEnvConfig() OutputDir = %q, want /output", cfg.OutputDir)
		}
		if cfg.AuthorName != "John Doe" {
			t.Errorf("loadEnvConfig() AuthorName = %q, want John Doe", cfg.AuthorName)
		}
		if cfg.AuthorOrg != "Acme Corp" {
			t.Errorf("loadEnvConfig() AuthorOrg = %q, want Acme Corp", cfg.AuthorOrg)
		}
		if cfg.AuthorEmail != "john@acme.com" {
			t.Errorf("loadEnvConfig() AuthorEmail = %q, want john@acme.com", cfg.AuthorEmail)
		}
	})

	t.Run("tier 3 extended variables", func(t *testing.T) {
		t.Setenv("PICOLOOM_PAGE_SIZE", "a4")
		t.Setenv("PICOLOOM_WATERMARK_TEXT", "DRAFT")
		t.Setenv("PICOLOOM_COVER_LOGO", "https://example.com/logo.png")
		t.Setenv("PICOLOOM_DOC_VERSION", "1.0.0")
		t.Setenv("PICOLOOM_DOC_DATE", "2024-01-15")
		t.Setenv("PICOLOOM_DOC_ID", "DOC-2024-001")
		t.Setenv("PICOLOOM_WORKERS", "4")

		cfg := loadEnvConfig()

		if cfg.PageSize != "a4" {
			t.Errorf("loadEnvConfig() PageSize = %q, want a4", cfg.PageSize)
		}
		if cfg.WatermarkText != "DRAFT" {
			t.Errorf("loadEnvConfig() WatermarkText = %q, want DRAFT", cfg.WatermarkText)
		}
		if cfg.CoverLogo != "https://example.com/logo.png" {
			t.Errorf("loadEnvConfig() CoverLogo = %q, want https://example.com/logo.png", cfg.CoverLogo)
		}
		if cfg.DocVersion != "1.0.0" {
			t.Errorf("loadEnvConfig() DocVersion = %q, want 1.0.0", cfg.DocVersion)
		}
		if cfg.DocDate != "2024-01-15" {
			t.Errorf("loadEnvConfig() DocDate = %q, want 2024-01-15", cfg.DocDate)
		}
		if cfg.DocID != "DOC-2024-001" {
			t.Errorf("loadEnvConfig() DocID = %q, want DOC-2024-001", cfg.DocID)
		}
		if cfg.Workers != 4 {
			t.Errorf("loadEnvConfig() Workers = %d, want 4", cfg.Workers)
		}
	})

	t.Run("error case: invalid timeout ignored", func(t *testing.T) {
		t.Setenv("PICOLOOM_TIMEOUT", "invalid")

		cfg := loadEnvConfig()

		if cfg.Timeout != 0 {
			t.Errorf("loadEnvConfig() Timeout = %v, want 0 (invalid value ignored)", cfg.Timeout)
		}
	})

	t.Run("error case: negative timeout ignored", func(t *testing.T) {
		t.Setenv("PICOLOOM_TIMEOUT", "-5s")

		cfg := loadEnvConfig()

		if cfg.Timeout != 0 {
			t.Errorf("loadEnvConfig() Timeout = %v, want 0 (negative value ignored)", cfg.Timeout)
		}
	})

	t.Run("error case: invalid workers ignored", func(t *testing.T) {
		t.Setenv("PICOLOOM_WORKERS", "abc")

		cfg := loadEnvConfig()

		if cfg.Workers != 0 {
			t.Errorf("loadEnvConfig() Workers = %d, want 0 (invalid value ignored)", cfg.Workers)
		}
	})

	t.Run("error case: negative workers ignored", func(t *testing.T) {
		t.Setenv("PICOLOOM_WORKERS", "-2")

		cfg := loadEnvConfig()

		if cfg.Workers != 0 {
			t.Errorf("loadEnvConfig() Workers = %d, want 0 (negative value ignored)", cfg.Workers)
		}
	})

	t.Run("edge case: empty env returns zero values", func(t *testing.T) {
		// No env vars set in this subtest

		cfg := loadEnvConfig()

		if cfg.ConfigPath != "" {
			t.Errorf("loadEnvConfig() ConfigPath = %q, want empty", cfg.ConfigPath)
		}
		if cfg.Style != "" {
			t.Errorf("loadEnvConfig() Style = %q, want empty", cfg.Style)
		}
		if cfg.Timeout != 0 {
			t.Errorf("loadEnvConfig() Timeout = %v, want 0", cfg.Timeout)
		}
	})

	t.Run("legacy fallback: reads MD2PDF variables", func(t *testing.T) {
		t.Setenv("MD2PDF_STYLE", "legacy-style")
		t.Setenv("MD2PDF_TIMEOUT", "90s")

		cfg := loadEnvConfig()

		if cfg.Style != "legacy-style" {
			t.Errorf("loadEnvConfig() Style = %q, want legacy-style", cfg.Style)
		}
		if cfg.Timeout != 90*time.Second {
			t.Errorf("loadEnvConfig() Timeout = %v, want 90s", cfg.Timeout)
		}
	})

	t.Run("priority: PICOLOOM variables override MD2PDF variables", func(t *testing.T) {
		t.Setenv("PICOLOOM_STYLE", "new-style")
		t.Setenv("MD2PDF_STYLE", "legacy-style")

		cfg := loadEnvConfig()

		if cfg.Style != "new-style" {
			t.Errorf("loadEnvConfig() Style = %q, want new-style", cfg.Style)
		}
	})
}

// ---------------------------------------------------------------------------
// TestWarnUnknownEnvVars - Unknown variable detection
// ---------------------------------------------------------------------------

func TestWarnUnknownEnvVars(t *testing.T) {
	t.Run("warns on unknown PICOLOOM and MD2PDF variables", func(t *testing.T) {
		t.Setenv("PICOLOOM_TYPO", "value")
		t.Setenv("MD2PDF_AHTOR_NAME", "typo")

		var buf bytes.Buffer
		warnUnknownEnvVars(&buf)

		output := buf.String()
		if !bytes.Contains(buf.Bytes(), []byte("PICOLOOM_TYPO")) {
			t.Errorf("warnUnknownEnvVars() should warn about PICOLOOM_TYPO, got: %s", output)
		}
		if !bytes.Contains(buf.Bytes(), []byte("MD2PDF_AHTOR_NAME")) {
			t.Errorf("warnUnknownEnvVars() should warn about MD2PDF_AHTOR_NAME, got: %s", output)
		}
		if !bytes.Contains(buf.Bytes(), []byte("typo?")) {
			t.Errorf("warnUnknownEnvVars() should suggest typo, got: %s", output)
		}
	})

	t.Run("happy path: no warning for known variables", func(t *testing.T) {
		t.Setenv("PICOLOOM_CONFIG", "/path")
		t.Setenv("PICOLOOM_STYLE", "technical")
		t.Setenv("PICOLOOM_TIMEOUT", "2m")
		t.Setenv("PICOLOOM_INPUT_DIR", "/input")
		t.Setenv("PICOLOOM_OUTPUT_DIR", "/output")
		t.Setenv("PICOLOOM_AUTHOR_NAME", "John")
		t.Setenv("PICOLOOM_AUTHOR_ORG", "Acme")
		t.Setenv("PICOLOOM_AUTHOR_EMAIL", "john@acme.com")
		t.Setenv("PICOLOOM_PAGE_SIZE", "a4")
		t.Setenv("PICOLOOM_WATERMARK_TEXT", "DRAFT")
		t.Setenv("PICOLOOM_COVER_LOGO", "/logo.png")
		t.Setenv("PICOLOOM_DOC_VERSION", "1.0")
		t.Setenv("PICOLOOM_DOC_DATE", "auto")
		t.Setenv("PICOLOOM_DOC_ID", "DOC-001")
		t.Setenv("PICOLOOM_WORKERS", "4")

		var buf bytes.Buffer
		warnUnknownEnvVars(&buf)

		if buf.Len() > 0 {
			t.Errorf("warnUnknownEnvVars() should not warn for known vars, got: %s", buf.String())
		}
	})

	t.Run("edge case: ignores non PICOLOOM and non MD2PDF variables", func(t *testing.T) {
		t.Setenv("PATH", "/usr/bin")
		t.Setenv("HOME", "/home/user")
		t.Setenv("SOME_OTHER_VAR", "value")

		var buf bytes.Buffer
		warnUnknownEnvVars(&buf)

		// Should not warn about unrelated env vars
		if bytes.Contains(buf.Bytes(), []byte("PATH")) {
			t.Errorf("warnUnknownEnvVars() should not warn about PATH")
		}
	})
}

// ---------------------------------------------------------------------------
// TestApplyEnvConfig - Config application with priority
// ---------------------------------------------------------------------------

func TestApplyEnvConfig(t *testing.T) {
	t.Run("happy path: applies env to empty config", func(t *testing.T) {
		env := &envConfig{
			Style:         "technical",
			InputDir:      "/input",
			OutputDir:     "/output",
			AuthorName:    "John Doe",
			AuthorOrg:     "Acme",
			AuthorEmail:   "john@acme.com",
			PageSize:      "a4",
			WatermarkText: "DRAFT",
			CoverLogo:     "/logo.png",
			DocVersion:    "1.0",
			DocDate:       "auto",
			DocID:         "DOC-001",
		}
		cfg := config.DefaultConfig()

		applyEnvConfig(env, cfg)

		if cfg.Style != "technical" {
			t.Errorf("applyEnvConfig() Style = %q, want technical", cfg.Style)
		}
		if cfg.Input.DefaultDir != "/input" {
			t.Errorf("applyEnvConfig() Input.DefaultDir = %q, want /input", cfg.Input.DefaultDir)
		}
		if cfg.Output.DefaultDir != "/output" {
			t.Errorf("applyEnvConfig() Output.DefaultDir = %q, want /output", cfg.Output.DefaultDir)
		}
		if cfg.Author.Name != "John Doe" {
			t.Errorf("applyEnvConfig() Author.Name = %q, want John Doe", cfg.Author.Name)
		}
		if cfg.Author.Organization != "Acme" {
			t.Errorf("applyEnvConfig() Author.Organization = %q, want Acme", cfg.Author.Organization)
		}
		if cfg.Author.Email != "john@acme.com" {
			t.Errorf("applyEnvConfig() Author.Email = %q, want john@acme.com", cfg.Author.Email)
		}
		if cfg.Page.Size != "a4" {
			t.Errorf("applyEnvConfig() Page.Size = %q, want a4", cfg.Page.Size)
		}
		if cfg.Watermark.Text != "DRAFT" {
			t.Errorf("applyEnvConfig() Watermark.Text = %q, want DRAFT", cfg.Watermark.Text)
		}
		if !cfg.Watermark.Enabled {
			t.Error("applyEnvConfig() Watermark.Enabled = false, want true (auto-enabled)")
		}
		if cfg.Cover.Logo != "/logo.png" {
			t.Errorf("applyEnvConfig() Cover.Logo = %q, want /logo.png", cfg.Cover.Logo)
		}
		if !cfg.Cover.Enabled {
			t.Error("applyEnvConfig() Cover.Enabled = false, want true (auto-enabled)")
		}
		if cfg.Document.Version != "1.0" {
			t.Errorf("applyEnvConfig() Document.Version = %q, want 1.0", cfg.Document.Version)
		}
		if cfg.Document.Date != "auto" {
			t.Errorf("applyEnvConfig() Document.Date = %q, want auto", cfg.Document.Date)
		}
		if cfg.Document.DocumentID != "DOC-001" {
			t.Errorf("applyEnvConfig() Document.DocumentID = %q, want DOC-001", cfg.Document.DocumentID)
		}
	})

	t.Run("priority: does not override existing config values", func(t *testing.T) {
		env := &envConfig{
			Style:      "env-style",
			AuthorName: "Env Author",
			PageSize:   "a4",
		}
		cfg := config.DefaultConfig()
		cfg.Style = "config-style"
		cfg.Author.Name = "Config Author"
		cfg.Page.Size = "letter"

		applyEnvConfig(env, cfg)

		// Config values should be preserved (env only fills empty values)
		if cfg.Style != "config-style" {
			t.Errorf("applyEnvConfig() Style = %q, want config-style (should not override)", cfg.Style)
		}
		if cfg.Author.Name != "Config Author" {
			t.Errorf("applyEnvConfig() Author.Name = %q, want Config Author (should not override)", cfg.Author.Name)
		}
		if cfg.Page.Size != "letter" {
			t.Errorf("applyEnvConfig() Page.Size = %q, want letter (should not override)", cfg.Page.Size)
		}
	})

	t.Run("watermark auto-enable: preserves existing enabled state", func(t *testing.T) {
		env := &envConfig{
			WatermarkText: "DRAFT",
		}
		cfg := config.DefaultConfig()
		cfg.Watermark.Enabled = true
		cfg.Watermark.Text = "CONFIDENTIAL"

		applyEnvConfig(env, cfg)

		// Existing text should be preserved
		if cfg.Watermark.Text != "CONFIDENTIAL" {
			t.Errorf("applyEnvConfig() Watermark.Text = %q, want CONFIDENTIAL", cfg.Watermark.Text)
		}
		// Enabled should still be true
		if !cfg.Watermark.Enabled {
			t.Error("applyEnvConfig() Watermark.Enabled = false, want true")
		}
	})

	t.Run("watermark auto-enable: applies env when config enabled but text empty", func(t *testing.T) {
		env := &envConfig{
			WatermarkText: "DRAFT",
		}
		cfg := config.DefaultConfig()
		cfg.Watermark.Enabled = true // Enabled but no text
		cfg.Watermark.Text = ""

		applyEnvConfig(env, cfg)

		// Env text should be applied
		if cfg.Watermark.Text != "DRAFT" {
			t.Errorf("applyEnvConfig() Watermark.Text = %q, want DRAFT", cfg.Watermark.Text)
		}
		// Enabled should still be true
		if !cfg.Watermark.Enabled {
			t.Error("applyEnvConfig() Watermark.Enabled = false, want true")
		}
	})

	t.Run("cover logo auto-enable: preserves existing enabled state", func(t *testing.T) {
		env := &envConfig{
			CoverLogo: "/env-logo.png",
		}
		cfg := config.DefaultConfig()
		cfg.Cover.Enabled = true
		cfg.Cover.Logo = "/config-logo.png"

		applyEnvConfig(env, cfg)

		// Existing logo should be preserved
		if cfg.Cover.Logo != "/config-logo.png" {
			t.Errorf("applyEnvConfig() Cover.Logo = %q, want /config-logo.png", cfg.Cover.Logo)
		}
		// Enabled should still be true
		if !cfg.Cover.Enabled {
			t.Error("applyEnvConfig() Cover.Enabled = false, want true")
		}
	})

	t.Run("cover logo auto-enable: applies env when config enabled but logo empty", func(t *testing.T) {
		env := &envConfig{
			CoverLogo: "/env-logo.png",
		}
		cfg := config.DefaultConfig()
		cfg.Cover.Enabled = true // Enabled but no logo
		cfg.Cover.Logo = ""

		applyEnvConfig(env, cfg)

		// Env logo should be applied
		if cfg.Cover.Logo != "/env-logo.png" {
			t.Errorf("applyEnvConfig() Cover.Logo = %q, want /env-logo.png", cfg.Cover.Logo)
		}
		// Enabled should still be true
		if !cfg.Cover.Enabled {
			t.Error("applyEnvConfig() Cover.Enabled = false, want true")
		}
	})

	t.Run("edge case: empty env values do not affect config", func(t *testing.T) {
		env := &envConfig{} // All empty
		cfg := config.DefaultConfig()
		cfg.Style = "existing"
		cfg.Author.Name = "Existing Author"

		applyEnvConfig(env, cfg)

		if cfg.Style != "existing" {
			t.Errorf("applyEnvConfig() Style = %q, want existing", cfg.Style)
		}
		if cfg.Author.Name != "Existing Author" {
			t.Errorf("applyEnvConfig() Author.Name = %q, want Existing Author", cfg.Author.Name)
		}
	})
}

// ---------------------------------------------------------------------------
// TestKnownEnvVars - Known variable list completeness
// ---------------------------------------------------------------------------

func TestKnownEnvVars(t *testing.T) {
	expected := []string{
		"PICOLOOM_CONFIG",
		"PICOLOOM_STYLE",
		"PICOLOOM_TIMEOUT",
		"PICOLOOM_INPUT_DIR",
		"PICOLOOM_OUTPUT_DIR",
		"PICOLOOM_AUTHOR_NAME",
		"PICOLOOM_AUTHOR_ORG",
		"PICOLOOM_AUTHOR_EMAIL",
		"PICOLOOM_PAGE_SIZE",
		"PICOLOOM_WATERMARK_TEXT",
		"PICOLOOM_COVER_LOGO",
		"PICOLOOM_DOC_VERSION",
		"PICOLOOM_DOC_DATE",
		"PICOLOOM_DOC_ID",
		"PICOLOOM_WORKERS",
		"PICOLOOM_CONTAINER",
		"MD2PDF_CONFIG",
		"MD2PDF_STYLE",
		"MD2PDF_TIMEOUT",
		"MD2PDF_INPUT_DIR",
		"MD2PDF_OUTPUT_DIR",
		"MD2PDF_AUTHOR_NAME",
		"MD2PDF_AUTHOR_ORG",
		"MD2PDF_AUTHOR_EMAIL",
		"MD2PDF_PAGE_SIZE",
		"MD2PDF_WATERMARK_TEXT",
		"MD2PDF_COVER_LOGO",
		"MD2PDF_DOC_VERSION",
		"MD2PDF_DOC_DATE",
		"MD2PDF_DOC_ID",
		"MD2PDF_WORKERS",
		"MD2PDF_CONTAINER",
	}

	for _, name := range expected {
		if !knownEnvVars[name] {
			t.Errorf("knownEnvVars[%s] = false, want true", name)
		}
	}

	if len(knownEnvVars) != len(expected) {
		t.Errorf("len(knownEnvVars) = %d, want %d", len(knownEnvVars), len(expected))
	}
}
