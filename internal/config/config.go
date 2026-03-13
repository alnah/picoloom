// Package config centralizes parsing and validation so every entry point
// enforces the same safety and defaulting rules.
package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	md2pdf "github.com/alnah/go-md2pdf"
	"github.com/alnah/go-md2pdf/internal/dateutil"
	"github.com/alnah/go-md2pdf/internal/fileutil"
	"github.com/alnah/go-md2pdf/internal/hints"
	"github.com/alnah/go-md2pdf/internal/yamlutil"
)

// Sentinel errors for config operations.
var (
	ErrConfigNotFound  = errors.New("config file not found")
	ErrEmptyConfigName = errors.New("config name cannot be empty")
	ErrConfigParse     = errors.New("failed to parse config")
	ErrFieldTooLong    = errors.New("field exceeds maximum length")
)

// Field length limits for multi-tenant safety.
const (
	MaxNameLength           = 100  // Full name (generous)
	MaxTitleLength          = 100  // Professional title
	MaxEmailLength          = 254  // RFC 5321
	MaxURLLength            = 2048 // Browser limit
	MaxTextLength           = 500  // Footer/free-form text
	MaxLabelLength          = 100  // Link label
	MaxPageSizeLength       = 10   // "letter", "a4", "legal"
	MaxOrientationLength    = 10   // "portrait", "landscape"
	MaxWatermarkTextLength  = 50   // "DRAFT", "CONFIDENTIAL"
	MaxWatermarkColorLength = 20   // "#888888" or color name
	MaxDocTitleLength       = 200  // Document title
	MaxSubtitleLength       = 200  // Document subtitle
	MaxOrganizationLength   = 100  // Organization name
	MaxVersionLength        = 50   // Version string
	MaxDateLength           = 30   // "2025-12-31" or "December 31, 2025"
	MaxTOCTitleLength       = 100  // TOC title
	// Extended metadata field limits
	MaxPhoneLength        = 30  // Phone number
	MaxAddressLength      = 200 // Postal address (multiline)
	MaxDepartmentLength   = 100 // Department name
	MaxClientNameLength   = 100 // Client/customer name
	MaxProjectNameLength  = 100 // Project name
	MaxDocumentTypeLength = 50  // Document type label
	MaxDocumentIDLength   = 50  // Document reference ID
	MaxDescriptionLength  = 500 // Document summary
)

// Config holds all configuration for document generation.
type Config struct {
	Author     AuthorConfig     `yaml:"author"`
	Document   DocumentConfig   `yaml:"document"`
	Input      InputConfig      `yaml:"input"`
	Output     OutputConfig     `yaml:"output"`
	Style      string           `yaml:"style"`   // CSS style name or file path
	Timeout    string           `yaml:"timeout"` // PDF generation timeout (e.g., "30s", "2m")
	Footer     FooterConfig     `yaml:"footer"`
	Signature  SignatureConfig  `yaml:"signature"`
	Assets     AssetsConfig     `yaml:"assets"`
	Page       PageConfig       `yaml:"page"`
	Watermark  WatermarkConfig  `yaml:"watermark"`
	Cover      CoverConfig      `yaml:"cover"`
	TOC        TOCConfig        `yaml:"toc"`
	PageBreaks PageBreaksConfig `yaml:"pageBreaks"`
}

// AuthorConfig holds shared author metadata used by cover and signature.
type AuthorConfig struct {
	Name         string `yaml:"name"`
	Title        string `yaml:"title"`
	Email        string `yaml:"email"`
	Organization string `yaml:"organization"`
	// Extended metadata fields
	Phone      string `yaml:"phone"`      // Contact phone number
	Address    string `yaml:"address"`    // Postal address (use YAML literal block for multiline)
	Department string `yaml:"department"` // Department name
}

// Validate checks author field lengths.
func (a *AuthorConfig) Validate() error {
	if err := validateFieldLength("author.name", a.Name, MaxNameLength); err != nil {
		return err
	}
	if err := validateFieldLength("author.title", a.Title, MaxTitleLength); err != nil {
		return err
	}
	if err := validateFieldLength("author.email", a.Email, MaxEmailLength); err != nil {
		return err
	}
	if err := validateFieldLength("author.organization", a.Organization, MaxOrganizationLength); err != nil {
		return err
	}
	if err := validateFieldLength("author.phone", a.Phone, MaxPhoneLength); err != nil {
		return err
	}
	if err := validateFieldLength("author.address", a.Address, MaxAddressLength); err != nil {
		return err
	}
	if err := validateFieldLength("author.department", a.Department, MaxDepartmentLength); err != nil {
		return err
	}
	return nil
}

// DocumentConfig holds shared document metadata used by cover and footer.
type DocumentConfig struct {
	Title    string `yaml:"title"`    // "" = auto per-file (H1 → filename)
	Subtitle string `yaml:"subtitle"` // Optional subtitle
	Version  string `yaml:"version"`  // Version string (used in cover and footer)
	Date     string `yaml:"date"`     // "auto" = YYYY-MM-DD at startup
	// Extended metadata fields
	ClientName   string `yaml:"clientName"`   // Client/customer name
	ProjectName  string `yaml:"projectName"`  // Project name
	DocumentType string `yaml:"documentType"` // e.g., "Technical Specification"
	DocumentID   string `yaml:"documentID"`   // e.g., "DOC-2024-001"
	Description  string `yaml:"description"`  // Brief document summary
}

// Validate checks document field lengths and date format.
func (d *DocumentConfig) Validate() error {
	if err := validateFieldLength("document.title", d.Title, MaxDocTitleLength); err != nil {
		return err
	}
	if err := validateFieldLength("document.subtitle", d.Subtitle, MaxSubtitleLength); err != nil {
		return err
	}
	if err := validateFieldLength("document.version", d.Version, MaxVersionLength); err != nil {
		return err
	}
	if err := validateFieldLength("document.date", d.Date, MaxDateLength); err != nil {
		return err
	}
	// Validate date format if using auto syntax
	if err := validateDateFormat(d.Date); err != nil {
		return err
	}
	if err := validateFieldLength("document.clientName", d.ClientName, MaxClientNameLength); err != nil {
		return err
	}
	if err := validateFieldLength("document.projectName", d.ProjectName, MaxProjectNameLength); err != nil {
		return err
	}
	if err := validateFieldLength("document.documentType", d.DocumentType, MaxDocumentTypeLength); err != nil {
		return err
	}
	if err := validateFieldLength("document.documentID", d.DocumentID, MaxDocumentIDLength); err != nil {
		return err
	}
	if err := validateFieldLength("document.description", d.Description, MaxDescriptionLength); err != nil {
		return err
	}
	return nil
}

// validateDateFormat checks that auto:FORMAT syntax uses valid tokens.
func validateDateFormat(date string) error {
	if date == "" {
		return nil
	}
	lower := strings.ToLower(date)
	if !strings.HasPrefix(lower, "auto:") {
		return nil // Not auto syntax, skip validation
	}
	// Use ParseDateFormat to validate (ignore result, just check error)
	formatPart := date[5:]
	if _, err := dateutil.ParseDateFormat(formatPart); err != nil {
		return fmt.Errorf("document.date: %w", err)
	}
	return nil
}

// InputConfig defines input source options.
type InputConfig struct {
	DefaultDir string `yaml:"defaultDir"` // Default input directory (empty = must specify)
}

// OutputConfig defines output destination options.
type OutputConfig struct {
	DefaultDir string `yaml:"defaultDir"` // Default output directory (empty = same as source)
}

// FooterConfig defines page footer options.
// Uses document.date and document.version for date/status display.
type FooterConfig struct {
	Enabled        bool   `yaml:"enabled"`
	Position       string `yaml:"position"`       // "left", "center", "right" (default: "right")
	ShowPageNumber bool   `yaml:"showPageNumber"` // Show page numbers
	Text           string `yaml:"text"`           // Optional free-form text
	ShowDocumentID bool   `yaml:"showDocumentID"` // Display DocumentID from document config
}

// Validate checks footer field values.
func (f *FooterConfig) Validate() error {
	if err := validateFieldLength("footer.text", f.Text, MaxTextLength); err != nil {
		return err
	}
	if f.Position != "" {
		switch strings.ToLower(f.Position) {
		case "left", "center", "right":
			// valid
		default:
			return fmt.Errorf("footer.position: invalid value %q (must be left, center, or right)", f.Position)
		}
	}
	return nil
}

// SignatureConfig defines signature block options.
// Uses author.name, author.title, author.email, author.organization for display.
type SignatureConfig struct {
	Enabled   bool   `yaml:"enabled"`
	ImagePath string `yaml:"imagePath"` // Signature image path or URL
	Links     []Link `yaml:"links"`     // Additional links
}

// Validate checks signature field values.
func (s *SignatureConfig) Validate() error {
	if err := validateFieldLength("signature.imagePath", s.ImagePath, MaxURLLength); err != nil {
		return err
	}
	for i, link := range s.Links {
		if err := validateFieldLength(fmt.Sprintf("signature.links[%d].label", i), link.Label, MaxLabelLength); err != nil {
			return err
		}
		if err := validateFieldLength(fmt.Sprintf("signature.links[%d].url", i), link.URL, MaxURLLength); err != nil {
			return err
		}
	}
	return nil
}

// Link represents a clickable link in the signature.
type Link struct {
	Label string `yaml:"label"`
	URL   string `yaml:"url"`
}

// AssetsConfig defines asset loading options.
type AssetsConfig struct {
	BasePath string `yaml:"basePath"` // Empty = use embedded assets
}

// Validate checks assets configuration.
// If BasePath is set, validates that it's a readable directory.
func (a *AssetsConfig) Validate() error {
	if a.BasePath == "" {
		return nil
	}

	info, err := os.Stat(a.BasePath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("assets.basePath: directory does not exist: %s", a.BasePath)
		}
		return fmt.Errorf("assets.basePath: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("assets.basePath: not a directory: %s", a.BasePath)
	}

	return nil
}

// PageConfig defines PDF page settings.
type PageConfig struct {
	Size        string  `yaml:"size"`        // "letter", "a4", "legal" (default: "letter")
	Orientation string  `yaml:"orientation"` // "portrait", "landscape" (default: "portrait")
	Margin      float64 `yaml:"margin"`      // inches (default: 0.5)
}

// Validate checks page field values.
func (p *PageConfig) Validate() error {
	if err := validateFieldLength("page.size", p.Size, MaxPageSizeLength); err != nil {
		return err
	}
	if err := validateFieldLength("page.orientation", p.Orientation, MaxOrientationLength); err != nil {
		return err
	}
	// Validate allowed values (empty means use default)
	if p.Size != "" {
		switch strings.ToLower(p.Size) {
		case md2pdf.PageSizeLetter, md2pdf.PageSizeA4, md2pdf.PageSizeLegal:
			// valid
		default:
			return fmt.Errorf("page.size: invalid value %q (must be letter, a4, or legal)", p.Size)
		}
	}
	if p.Orientation != "" {
		switch strings.ToLower(p.Orientation) {
		case md2pdf.OrientationPortrait, md2pdf.OrientationLandscape:
			// valid
		default:
			return fmt.Errorf("page.orientation: invalid value %q (must be portrait or landscape)", p.Orientation)
		}
	}
	if p.Margin != 0 && (p.Margin < md2pdf.MinMargin || p.Margin > md2pdf.MaxMargin) {
		return fmt.Errorf("page.margin: must be between %.2f and %.2f, got %.2f", md2pdf.MinMargin, md2pdf.MaxMargin, p.Margin)
	}
	return nil
}

// WatermarkConfig defines background watermark options.
type WatermarkConfig struct {
	Enabled bool    `yaml:"enabled"`
	Text    string  `yaml:"text"`    // Text to display (e.g., "DRAFT", "CONFIDENTIAL")
	Color   string  `yaml:"color"`   // Hex color (default: "#888888")
	Opacity float64 `yaml:"opacity"` // 0.0 to 1.0 (default: 0.1)
	Angle   float64 `yaml:"angle"`   // Rotation in degrees (default: -45)
}

// Validate checks watermark field values.
func (w *WatermarkConfig) Validate() error {
	if !w.Enabled {
		return nil
	}
	if w.Text == "" {
		return fmt.Errorf("watermark.text: required when watermark is enabled")
	}
	if err := validateFieldLength("watermark.text", w.Text, MaxWatermarkTextLength); err != nil {
		return err
	}
	if err := validateFieldLength("watermark.color", w.Color, MaxWatermarkColorLength); err != nil {
		return err
	}
	if w.Opacity < md2pdf.MinWatermarkOpacity || w.Opacity > md2pdf.MaxWatermarkOpacity {
		return fmt.Errorf("watermark.opacity: must be between %.1f and %.1f, got %.2f", md2pdf.MinWatermarkOpacity, md2pdf.MaxWatermarkOpacity, w.Opacity)
	}
	if w.Angle < md2pdf.MinWatermarkAngle || w.Angle > md2pdf.MaxWatermarkAngle {
		return fmt.Errorf("watermark.angle: must be between %.0f and %.0f, got %.2f", md2pdf.MinWatermarkAngle, md2pdf.MaxWatermarkAngle, w.Angle)
	}
	return nil
}

// CoverConfig defines cover page options.
// Uses author.* and document.* for author info and metadata.
type CoverConfig struct {
	Enabled        bool   `yaml:"enabled"`
	Logo           string `yaml:"logo"`           // Logo path or URL (cover-specific)
	ShowDepartment bool   `yaml:"showDepartment"` // Show author.department on cover
}

// Validate checks cover field values.
func (c *CoverConfig) Validate() error {
	if err := validateFieldLength("cover.logo", c.Logo, MaxURLLength); err != nil {
		return err
	}
	return nil
}

// TOCConfig defines table of contents options.
type TOCConfig struct {
	Enabled  bool   `yaml:"enabled"`
	Title    string `yaml:"title"`    // Empty = no title above TOC
	MinDepth int    `yaml:"minDepth"` // 1-6, default 2 (skips H1)
	MaxDepth int    `yaml:"maxDepth"` // 1-6, default 3
}

// Validate checks TOC field values.
func (t *TOCConfig) Validate() error {
	if err := validateFieldLength("toc.title", t.Title, MaxTOCTitleLength); err != nil {
		return err
	}
	if t.Enabled {
		if t.MinDepth != 0 && (t.MinDepth < 1 || t.MinDepth > 6) {
			return fmt.Errorf("toc.minDepth: must be between 1 and 6, got %d", t.MinDepth)
		}
		if t.MaxDepth != 0 && (t.MaxDepth < 1 || t.MaxDepth > 6) {
			return fmt.Errorf("toc.maxDepth: must be between 1 and 6, got %d", t.MaxDepth)
		}
		if t.MinDepth != 0 && t.MaxDepth != 0 && t.MinDepth > t.MaxDepth {
			return fmt.Errorf("toc.minDepth (%d) cannot be greater than toc.maxDepth (%d)", t.MinDepth, t.MaxDepth)
		}
	}
	return nil
}

// PageBreaksConfig defines page break options.
type PageBreaksConfig struct {
	Enabled  bool `yaml:"enabled"`  // Enable page break features (default: true for orphan/widow)
	BeforeH1 bool `yaml:"beforeH1"` // Page break before H1 headings
	BeforeH2 bool `yaml:"beforeH2"` // Page break before H2 headings
	BeforeH3 bool `yaml:"beforeH3"` // Page break before H3 headings
	Orphans  int  `yaml:"orphans"`  // Min lines at page bottom (1-5, default 2)
	Widows   int  `yaml:"widows"`   // Min lines at page top (1-5, default 2)
}

// Validate checks page breaks field values.
func (pb *PageBreaksConfig) Validate() error {
	if pb.Orphans != 0 {
		if pb.Orphans < 1 || pb.Orphans > 5 {
			return fmt.Errorf("pageBreaks.orphans: must be between 1 and 5, got %d", pb.Orphans)
		}
	}
	if pb.Widows != 0 {
		if pb.Widows < 1 || pb.Widows > 5 {
			return fmt.Errorf("pageBreaks.widows: must be between 1 and 5, got %d", pb.Widows)
		}
	}
	return nil
}

// Validate checks field lengths to prevent abuse in multi-tenant scenarios.
// Called automatically by LoadConfig, but available for consumers
// who construct Config manually (e.g., API adapters, library users).
func (c *Config) Validate() error {
	if err := c.Author.Validate(); err != nil {
		return err
	}
	if err := c.Document.Validate(); err != nil {
		return err
	}
	if err := validateTimeout(c.Timeout); err != nil {
		return err
	}
	if err := c.Footer.Validate(); err != nil {
		return err
	}
	if err := c.Signature.Validate(); err != nil {
		return err
	}
	if err := c.Assets.Validate(); err != nil {
		return err
	}
	if err := c.Page.Validate(); err != nil {
		return err
	}
	if err := c.Watermark.Validate(); err != nil {
		return err
	}
	if err := c.Cover.Validate(); err != nil {
		return err
	}
	if err := c.TOC.Validate(); err != nil {
		return err
	}
	if err := c.PageBreaks.Validate(); err != nil {
		return err
	}
	return nil
}

// validateFieldLength checks if a field exceeds its maximum allowed length.
func validateFieldLength(fieldName, value string, maxLength int) error {
	if len(value) > maxLength {
		return fmt.Errorf("%w: %s (%d chars, max %d)", ErrFieldTooLong, fieldName, len(value), maxLength)
	}
	return nil
}

// validateTimeout checks that the timeout string is a valid positive duration.
func validateTimeout(timeout string) error {
	if timeout == "" {
		return nil
	}
	d, err := time.ParseDuration(timeout)
	if err != nil {
		return fmt.Errorf("timeout: invalid duration %q (use format like \"30s\", \"2m\")", timeout)
	}
	if d <= 0 {
		return fmt.Errorf("timeout: must be positive, got %q", timeout)
	}
	return nil
}

// DefaultConfig returns a neutral configuration with all features disabled.
func DefaultConfig() *Config {
	return &Config{
		Author:     AuthorConfig{},
		Document:   DocumentConfig{},
		Input:      InputConfig{DefaultDir: ""},
		Output:     OutputConfig{DefaultDir: ""},
		Style:      "",
		Footer:     FooterConfig{Enabled: false},
		Signature:  SignatureConfig{Enabled: false},
		Assets:     AssetsConfig{BasePath: ""},
		Page:       PageConfig{},
		Watermark:  WatermarkConfig{Enabled: false},
		Cover:      CoverConfig{Enabled: false},
		TOC:        TOCConfig{Enabled: false},
		PageBreaks: PageBreaksConfig{Enabled: false},
	}
}

// LoadConfig loads configuration from a file path or config name.
// If nameOrPath contains a path separator, it's treated as a file path.
// Otherwise, it's treated as a config name and searched in standard locations.
// Returns error if the file is not found (no silent fallback).
func LoadConfig(nameOrPath string) (*Config, error) {
	if nameOrPath == "" {
		return nil, ErrEmptyConfigName
	}

	var configPath string
	var err error

	if isFilePath(nameOrPath) {
		configPath = nameOrPath
	} else {
		configPath, err = resolveConfigPath(nameOrPath)
		if err != nil {
			return nil, err
		}
	}

	data, err := os.ReadFile(configPath) // #nosec G304 -- config path is user-provided
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("%w: %s", ErrConfigNotFound, configPath)
		}
		return nil, fmt.Errorf("reading config file: %w", err)
	}

	var cfg Config
	if err := yamlutil.UnmarshalStrict(data, &cfg); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrConfigParse, err)
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// isFilePath delegates to fileutil.IsFilePath for path detection.
// See fileutil.IsFilePath for documentation and examples.
func isFilePath(s string) bool {
	return fileutil.IsFilePath(s)
}

// resolveConfigPath searches for a config file by name in standard locations.
// Tries extensions in order: .yaml, .yml
// Tries locations in order: current directory, ~/.config/picoloom/, legacy dirs.
func resolveConfigPath(name string) (string, error) {
	extensions := []string{".yaml", ".yml"}
	triedPaths := make([]string, 0, len(extensions)*3)

	// Try current directory first (both extensions)
	for _, ext := range extensions {
		localPath := name + ext
		if fileExists(localPath) {
			return localPath, nil
		}
		triedPaths = append(triedPaths, localPath)
	}

	// Try user config directory (both extensions)
	userConfigDir, err := os.UserConfigDir()
	if err == nil {
		for _, ext := range extensions {
			for _, configDirName := range []string{"picoloom", "go-md2pdf"} {
				userPath := filepath.Join(userConfigDir, configDirName, name+ext)
				if fileExists(userPath) {
					return userPath, nil
				}
				triedPaths = append(triedPaths, userPath)
			}
		}
	}

	return "", fmt.Errorf("%w\n  searched: %s%s", ErrConfigNotFound, strings.Join(triedPaths, ", "), hints.ForConfigNotFound(triedPaths))
}

// fileExists delegates to fileutil.FileExists for path existence check.
// See fileutil.FileExists for documentation.
func fileExists(path string) bool {
	return fileutil.FileExists(path)
}
