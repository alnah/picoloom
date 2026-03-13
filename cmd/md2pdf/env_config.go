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
	ConfigPath string        // PICOLOOM_CONFIG / MD2PDF_CONFIG: config file path
	Style      string        // PICOLOOM_STYLE / MD2PDF_STYLE: CSS style name or path
	Timeout    time.Duration // PICOLOOM_TIMEOUT / MD2PDF_TIMEOUT: PDF generation timeout

	// Tier 2 - I/O and identity
	InputDir    string // PICOLOOM_INPUT_DIR / MD2PDF_INPUT_DIR: default input directory
	OutputDir   string // PICOLOOM_OUTPUT_DIR / MD2PDF_OUTPUT_DIR: default output directory
	AuthorName  string // PICOLOOM_AUTHOR_NAME / MD2PDF_AUTHOR_NAME: author name
	AuthorOrg   string // PICOLOOM_AUTHOR_ORG / MD2PDF_AUTHOR_ORG: organization
	AuthorEmail string // PICOLOOM_AUTHOR_EMAIL / MD2PDF_AUTHOR_EMAIL: author email

	// Tier 3 - Extended
	PageSize      string // PICOLOOM_PAGE_SIZE / MD2PDF_PAGE_SIZE: a4, letter, legal
	WatermarkText string // PICOLOOM_WATERMARK_TEXT / MD2PDF_WATERMARK_TEXT: watermark text
	CoverLogo     string // PICOLOOM_COVER_LOGO / MD2PDF_COVER_LOGO: cover logo path/URL
	DocVersion    string // PICOLOOM_DOC_VERSION / MD2PDF_DOC_VERSION: document version
	DocDate       string // PICOLOOM_DOC_DATE / MD2PDF_DOC_DATE: document date
	DocID         string // PICOLOOM_DOC_ID / MD2PDF_DOC_ID: document ID
	Workers       int    // PICOLOOM_WORKERS / MD2PDF_WORKERS: parallel workers
}

const (
	canonicalEnvPrefix = "PICOLOOM_"
	legacyEnvPrefix    = "MD2PDF_"
)

var envVarSuffixes = []string{
	"CONFIG",
	"STYLE",
	"TIMEOUT",
	"INPUT_DIR",
	"OUTPUT_DIR",
	"AUTHOR_NAME",
	"AUTHOR_ORG",
	"AUTHOR_EMAIL",
	"PAGE_SIZE",
	"WATERMARK_TEXT",
	"COVER_LOGO",
	"DOC_VERSION",
	"DOC_DATE",
	"DOC_ID",
	"WORKERS",
	"CONTAINER",
}

// knownEnvVars lists valid PICOLOOM_* and MD2PDF_* environment variables.
// Used to detect typos and warn users about unknown variables.
var knownEnvVars = buildKnownEnvVars()

func buildKnownEnvVars() map[string]bool {
	known := make(map[string]bool, len(envVarSuffixes)*2)
	for _, suffix := range envVarSuffixes {
		known[canonicalEnvPrefix+suffix] = true
		known[legacyEnvPrefix+suffix] = true
	}
	return known
}

func lookupEnv(suffix string) string {
	if value := os.Getenv(canonicalEnvPrefix + suffix); value != "" {
		return value
	}
	return os.Getenv(legacyEnvPrefix + suffix)
}

// loadEnvConfig reads configuration from environment variables.
// Returns a struct with all recognized PICOLOOM_* values and legacy MD2PDF_* fallbacks.
func loadEnvConfig() *envConfig {
	cfg := &envConfig{
		// Tier 1
		ConfigPath: lookupEnv("CONFIG"),
		Style:      lookupEnv("STYLE"),
		// Tier 2
		InputDir:    lookupEnv("INPUT_DIR"),
		OutputDir:   lookupEnv("OUTPUT_DIR"),
		AuthorName:  lookupEnv("AUTHOR_NAME"),
		AuthorOrg:   lookupEnv("AUTHOR_ORG"),
		AuthorEmail: lookupEnv("AUTHOR_EMAIL"),
		// Tier 3
		PageSize:      lookupEnv("PAGE_SIZE"),
		WatermarkText: lookupEnv("WATERMARK_TEXT"),
		CoverLogo:     lookupEnv("COVER_LOGO"),
		DocVersion:    lookupEnv("DOC_VERSION"),
		DocDate:       lookupEnv("DOC_DATE"),
		DocID:         lookupEnv("DOC_ID"),
	}

	// Parse duration for timeout
	if timeout := lookupEnv("TIMEOUT"); timeout != "" {
		if d, err := time.ParseDuration(timeout); err == nil && d > 0 {
			cfg.Timeout = d
		}
	}

	// Parse int for workers
	if workers := lookupEnv("WORKERS"); workers != "" {
		if w, err := strconv.Atoi(workers); err == nil && w > 0 {
			cfg.Workers = w
		}
	}

	return cfg
}

// warnUnknownEnvVars logs warnings for unrecognized PICOLOOM_* and MD2PDF_* variables.
// Helps catch typos like PICOLOOM_AHTOR_NAME or MD2PDF_AHTOR_NAME.
func warnUnknownEnvVars(w io.Writer) {
	for _, env := range os.Environ() {
		if strings.HasPrefix(env, canonicalEnvPrefix) || strings.HasPrefix(env, legacyEnvPrefix) {
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
