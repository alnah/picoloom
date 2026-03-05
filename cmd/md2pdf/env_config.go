package main

import (
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/alnah/go-md2pdf/internal/config"
)

// envConfig holds configuration from environment variables.
// Provides CI/CD-friendly overrides without requiring YAML files.
type envConfig struct {
	// Tier 1 - Essential
	ConfigPath string        // MD2PDF_CONFIG: config file path
	Style      string        // MD2PDF_STYLE: CSS style name or path
	Timeout    time.Duration // MD2PDF_TIMEOUT: PDF generation timeout

	// Tier 2 - I/O and identity
	InputDir    string // MD2PDF_INPUT_DIR: default input directory
	OutputDir   string // MD2PDF_OUTPUT_DIR: default output directory
	AuthorName  string // MD2PDF_AUTHOR_NAME: author name
	AuthorOrg   string // MD2PDF_AUTHOR_ORG: organization
	AuthorEmail string // MD2PDF_AUTHOR_EMAIL: author email

	// Tier 3 - Extended
	PageSize      string // MD2PDF_PAGE_SIZE: a4, letter, legal
	WatermarkText string // MD2PDF_WATERMARK_TEXT: watermark text
	CoverLogo     string // MD2PDF_COVER_LOGO: cover logo path/URL
	DocVersion    string // MD2PDF_DOC_VERSION: document version
	DocDate       string // MD2PDF_DOC_DATE: document date
	DocID         string // MD2PDF_DOC_ID: document ID
	Workers       int    // MD2PDF_WORKERS: parallel workers
}

// knownEnvVars lists valid MD2PDF_* environment variables.
// Used to detect typos and warn users about unknown variables.
var knownEnvVars = map[string]bool{
	// Tier 1 - Essential
	"MD2PDF_CONFIG":  true,
	"MD2PDF_STYLE":   true,
	"MD2PDF_TIMEOUT": true,
	// Tier 2 - I/O and identity
	"MD2PDF_INPUT_DIR":    true,
	"MD2PDF_OUTPUT_DIR":   true,
	"MD2PDF_AUTHOR_NAME":  true,
	"MD2PDF_AUTHOR_ORG":   true,
	"MD2PDF_AUTHOR_EMAIL": true,
	// Tier 3 - Extended
	"MD2PDF_PAGE_SIZE":      true,
	"MD2PDF_WATERMARK_TEXT": true,
	"MD2PDF_COVER_LOGO":     true,
	"MD2PDF_DOC_VERSION":    true,
	"MD2PDF_DOC_DATE":       true,
	"MD2PDF_DOC_ID":         true,
	"MD2PDF_WORKERS":        true,
	"MD2PDF_CONTAINER":      true,
}

// loadEnvConfig reads configuration from environment variables.
// Returns a struct with all recognized MD2PDF_* values.
func loadEnvConfig() *envConfig {
	cfg := &envConfig{
		// Tier 1
		ConfigPath: os.Getenv("MD2PDF_CONFIG"),
		Style:      os.Getenv("MD2PDF_STYLE"),
		// Tier 2
		InputDir:    os.Getenv("MD2PDF_INPUT_DIR"),
		OutputDir:   os.Getenv("MD2PDF_OUTPUT_DIR"),
		AuthorName:  os.Getenv("MD2PDF_AUTHOR_NAME"),
		AuthorOrg:   os.Getenv("MD2PDF_AUTHOR_ORG"),
		AuthorEmail: os.Getenv("MD2PDF_AUTHOR_EMAIL"),
		// Tier 3
		PageSize:      os.Getenv("MD2PDF_PAGE_SIZE"),
		WatermarkText: os.Getenv("MD2PDF_WATERMARK_TEXT"),
		CoverLogo:     os.Getenv("MD2PDF_COVER_LOGO"),
		DocVersion:    os.Getenv("MD2PDF_DOC_VERSION"),
		DocDate:       os.Getenv("MD2PDF_DOC_DATE"),
		DocID:         os.Getenv("MD2PDF_DOC_ID"),
	}

	// Parse duration for timeout
	if timeout := os.Getenv("MD2PDF_TIMEOUT"); timeout != "" {
		if d, err := time.ParseDuration(timeout); err == nil && d > 0 {
			cfg.Timeout = d
		}
	}

	// Parse int for workers
	if workers := os.Getenv("MD2PDF_WORKERS"); workers != "" {
		if w, err := strconv.Atoi(workers); err == nil && w > 0 {
			cfg.Workers = w
		}
	}

	return cfg
}

// warnUnknownEnvVars logs warnings for unrecognized MD2PDF_* variables.
// Helps catch typos like MD2PDF_AUTOR instead of MD2PDF_AUTHOR_NAME.
func warnUnknownEnvVars(w io.Writer) {
	for _, env := range os.Environ() {
		if strings.HasPrefix(env, "MD2PDF_") {
			name := strings.SplitN(env, "=", 2)[0]
			if !knownEnvVars[name] {
				fmt.Fprintf(w, "warning: unknown environment variable %s (typo?)\n", name)
			}
		}
	}
}

// applyEnvConfig applies environment variable values to config.
// Only sets values if the env var is set AND the config value is empty/zero.
// This ensures: CLI flags > config file > env vars > defaults
// (env fills only missing values from config; CLI flags are applied later)
func applyEnvConfig(env *envConfig, cfg *config.Config) {
	// Tier 1 - Style (timeout handled separately in resolveTimeout)
	if env.Style != "" && cfg.Style == "" {
		cfg.Style = env.Style
	}

	// Tier 2 - I/O
	if env.InputDir != "" && cfg.Input.DefaultDir == "" {
		cfg.Input.DefaultDir = env.InputDir
	}
	if env.OutputDir != "" && cfg.Output.DefaultDir == "" {
		cfg.Output.DefaultDir = env.OutputDir
	}

	// Tier 2 - Author identity
	if env.AuthorName != "" && cfg.Author.Name == "" {
		cfg.Author.Name = env.AuthorName
	}
	if env.AuthorOrg != "" && cfg.Author.Organization == "" {
		cfg.Author.Organization = env.AuthorOrg
	}
	if env.AuthorEmail != "" && cfg.Author.Email == "" {
		cfg.Author.Email = env.AuthorEmail
	}

	// Tier 3 - Page
	if env.PageSize != "" && cfg.Page.Size == "" {
		cfg.Page.Size = env.PageSize
	}

	// Tier 3 - Watermark (auto-enable)
	if env.WatermarkText != "" && cfg.Watermark.Text == "" {
		cfg.Watermark.Text = env.WatermarkText
		if !cfg.Watermark.Enabled {
			cfg.Watermark.Enabled = true
		}
	}

	// Tier 3 - Cover logo (auto-enable)
	if env.CoverLogo != "" && cfg.Cover.Logo == "" {
		cfg.Cover.Logo = env.CoverLogo
		if !cfg.Cover.Enabled {
			cfg.Cover.Enabled = true
		}
	}

	// Tier 3 - Document metadata
	if env.DocVersion != "" && cfg.Document.Version == "" {
		cfg.Document.Version = env.DocVersion
	}
	if env.DocDate != "" && cfg.Document.Date == "" {
		cfg.Document.Date = env.DocDate
	}
	if env.DocID != "" && cfg.Document.DocumentID == "" {
		cfg.Document.DocumentID = env.DocID
	}
}
