package picoloom

import (
	"fmt"
	"strings"
	"time"

	"github.com/alnah/picoloom/v2/internal/assets"
	"github.com/alnah/picoloom/v2/internal/fileutil"
)

// Page size constants.
const (
	PageSizeLetter = "letter"
	PageSizeA4     = "a4"
	PageSizeLegal  = "legal"
)

// Orientation constants.
const (
	OrientationPortrait  = "portrait"
	OrientationLandscape = "landscape"
)

// Margin bounds in inches.
const (
	MinMargin     = 0.25
	MaxMargin     = 3.0
	DefaultMargin = 0.5
)

// Orphan/widow bounds for page break control.
const (
	MinOrphans     = 1
	MaxOrphans     = 5
	DefaultOrphans = 2
	MinWidows      = 1
	MaxWidows      = 5
	DefaultWidows  = 2
)

// PageSettings configures PDF page dimensions.
type PageSettings struct {
	Size        string  // "letter", "a4", "legal"
	Orientation string  // "portrait", "landscape"
	Margin      float64 // inches, applied to all sides
}

// DefaultPageSettings returns page settings with default values.
func DefaultPageSettings() *PageSettings {
	return &PageSettings{
		Size:        PageSizeLetter,
		Orientation: OrientationPortrait,
		Margin:      DefaultMargin,
	}
}

// Validate checks that page settings are valid.
// Returns nil if p is nil (nil means use defaults).
// Empty values are allowed and will use defaults at runtime.
// Does not mutate - uses case-insensitive comparison.
func (p *PageSettings) Validate() error {
	if p == nil {
		return nil
	}

	// Empty means use default; only validate if explicitly set
	if p.Size != "" && !isValidPageSize(p.Size) {
		return fmt.Errorf("%w: %q", ErrInvalidPageSize, p.Size)
	}

	if p.Orientation != "" && !isValidOrientation(p.Orientation) {
		return fmt.Errorf("%w: %q", ErrInvalidOrientation, p.Orientation)
	}

	// Margin: 0 means use default; only validate if explicitly set
	if p.Margin != 0 && (p.Margin < MinMargin || p.Margin > MaxMargin) {
		return fmt.Errorf("%w: %.2f (must be between %.2f and %.2f)", ErrInvalidMargin, p.Margin, MinMargin, MaxMargin)
	}

	return nil
}

// isValidPageSize checks if size is a known page size (case-insensitive).
func isValidPageSize(size string) bool {
	switch strings.ToLower(size) {
	case PageSizeLetter, PageSizeA4, PageSizeLegal:
		return true
	}
	return false
}

// isValidOrientation checks if orientation is valid (case-insensitive).
func isValidOrientation(orientation string) bool {
	switch strings.ToLower(orientation) {
	case OrientationPortrait, OrientationLandscape:
		return true
	}
	return false
}

// PageBreaks configures page break behavior for PDF output.
type PageBreaks struct {
	BeforeH1 bool // Page break before H1 headings (default: false)
	BeforeH2 bool // Page break before H2 headings (default: false)
	BeforeH3 bool // Page break before H3 headings (default: false)
	Orphans  int  // Min lines at page bottom (default: 2, range: 1-5)
	Widows   int  // Min lines at page top (default: 2, range: 1-5)
}

// Validate checks that page break settings are valid.
// Returns nil if pb is nil (nil means use defaults).
func (pb *PageBreaks) Validate() error {
	if pb == nil {
		return nil
	}
	// Orphans: 0 means use default, otherwise must be in range
	if pb.Orphans != 0 && (pb.Orphans < MinOrphans || pb.Orphans > MaxOrphans) {
		return fmt.Errorf("%w: %d (must be %d-%d)", ErrInvalidOrphans, pb.Orphans, MinOrphans, MaxOrphans)
	}
	// Widows: 0 means use default, otherwise must be in range
	if pb.Widows != 0 && (pb.Widows < MinWidows || pb.Widows > MaxWidows) {
		return fmt.Errorf("%w: %d (must be %d-%d)", ErrInvalidWidows, pb.Widows, MinWidows, MaxWidows)
	}
	return nil
}

// Input contains conversion parameters.
type Input struct {
	Markdown   string        // Markdown content (required)
	SourceDir  string        // Base directory for resolving relative paths (optional)
	CSS        string        // Custom CSS (optional)
	Footer     *Footer       // Footer config (optional)
	Signature  *Signature    // Signature config (optional)
	Page       *PageSettings // Page settings (optional, nil = defaults)
	Watermark  *Watermark    // Watermark config (optional)
	Cover      *Cover        // Cover page config (optional)
	TOC        *TOC          // Table of contents config (optional)
	PageBreaks *PageBreaks   // Page break config (optional)
	HTMLOnly   bool          // If true, skip PDF generation (for debugging)
}

// ConvertResult holds both HTML and PDF output from conversion.
// HTML is always populated; PDF is empty when Input.HTMLOnly is true.
type ConvertResult struct {
	HTML []byte // Final HTML after all injections
	PDF  []byte // Generated PDF (empty if HTMLOnly)
}

// Watermark bounds.
const (
	MinWatermarkOpacity     = 0.0
	MaxWatermarkOpacity     = 1.0
	DefaultWatermarkOpacity = 0.1
	MinWatermarkAngle       = -90.0
	MaxWatermarkAngle       = 90.0
	DefaultWatermarkAngle   = -45.0
	DefaultWatermarkColor   = "#888888"
)

// Watermark configures a background text watermark.
type Watermark struct {
	Text    string  // Text to display (e.g., "DRAFT", "CONFIDENTIAL")
	Color   string  // Hex color (default: "#888888")
	Opacity float64 // 0.0 to 1.0 (default: 0.1)
	Angle   float64 // Rotation in degrees (default: -45)
}

// Validate checks that watermark settings are valid.
// Returns nil if w is nil (nil means no watermark).
func (w *Watermark) Validate() error {
	if w == nil {
		return nil
	}
	if w.Color != "" && !isValidHexColor(w.Color) {
		return fmt.Errorf("%w: %q (must be hex format like #RGB or #RRGGBB)", ErrInvalidWatermarkColor, w.Color)
	}
	if w.Opacity < MinWatermarkOpacity || w.Opacity > MaxWatermarkOpacity {
		return fmt.Errorf("watermark opacity must be between %.1f and %.1f, got %.2f", MinWatermarkOpacity, MaxWatermarkOpacity, w.Opacity)
	}
	if w.Angle < MinWatermarkAngle || w.Angle > MaxWatermarkAngle {
		return fmt.Errorf("watermark angle must be between %.0f and %.0f, got %.2f", MinWatermarkAngle, MaxWatermarkAngle, w.Angle)
	}
	return nil
}

// Cover configures the cover page.
type Cover struct {
	Title        string // Document title (required)
	Subtitle     string // Optional subtitle
	Logo         string // Logo path or URL (optional)
	Author       string // Author name (optional)
	AuthorTitle  string // Author's professional title (optional)
	Organization string // Organization name (optional)
	Date         string // Date string (optional)
	Version      string // Version string (optional)
	// Extended metadata fields
	ClientName   string // Client/customer name (optional)
	ProjectName  string // Project name (optional)
	DocumentType string // Document type, e.g., "Technical Specification" (optional)
	DocumentID   string // Document reference, e.g., "DOC-2024-001" (optional)
	Description  string // Brief document summary (optional)
	Department   string // Author's department (optional, shared with Signature via config)
}

// Validate checks that cover settings are valid.
// Returns nil if c is nil (nil means no cover).
func (c *Cover) Validate() error {
	if c == nil {
		return nil
	}
	// Semantic validation: logo path exists if not URL
	if c.Logo != "" && !fileutil.IsURL(c.Logo) && !fileutil.FileExists(c.Logo) {
		return fmt.Errorf("%w: %q", ErrCoverLogoNotFound, c.Logo)
	}
	return nil
}

// TOC depth bounds.
const (
	minTOCDepth        = 1
	maxTOCDepth        = 6
	DefaultTOCMinDepth = 2 // Skip H1 by default (document title)
	DefaultTOCMaxDepth = 3
)

// TOC configures the table of contents.
type TOC struct {
	Title    string // Title above TOC (empty = no title)
	MinDepth int    // 1-6, minimum heading level to include (default: 2, skips H1)
	MaxDepth int    // 1-6, maximum heading level to include (default: 3)
}

// Validate checks that TOC settings are valid.
// Returns nil if t is nil (nil means no TOC).
func (t *TOC) Validate() error {
	if t == nil {
		return nil
	}
	// MinDepth: 0 means use default, otherwise must be in range
	if t.MinDepth != 0 && (t.MinDepth < minTOCDepth || t.MinDepth > maxTOCDepth) {
		return fmt.Errorf("%w: MinDepth %d (must be %d-%d)", ErrInvalidTOCDepth, t.MinDepth, minTOCDepth, maxTOCDepth)
	}
	// MaxDepth: 0 means use default, otherwise must be in range
	if t.MaxDepth != 0 && (t.MaxDepth < minTOCDepth || t.MaxDepth > maxTOCDepth) {
		return fmt.Errorf("%w: MaxDepth %d (must be %d-%d)", ErrInvalidTOCDepth, t.MaxDepth, minTOCDepth, maxTOCDepth)
	}
	// MinDepth must be <= MaxDepth (when both are set)
	if t.MinDepth != 0 && t.MaxDepth != 0 && t.MinDepth > t.MaxDepth {
		return fmt.Errorf("%w: MinDepth %d > MaxDepth %d", ErrInvalidTOCDepth, t.MinDepth, t.MaxDepth)
	}
	return nil
}

// isValidHexColor checks if color is a valid hex color (#RGB or #RRGGBB).
func isValidHexColor(color string) bool {
	if len(color) == 0 || color[0] != '#' {
		return false
	}
	hex := color[1:]
	if len(hex) != 3 && len(hex) != 6 {
		return false
	}
	for _, c := range hex {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			return false
		}
	}
	return true
}

// Footer configures the PDF footer.
type Footer struct {
	Position       string // "left", "center", "right" (default: "right")
	ShowPageNumber bool
	Date           string
	Status         string
	Text           string
	DocumentID     string // Document reference number (optional)
}

// Validate checks that footer settings are valid.
// Returns nil if f is nil (nil means no footer).
func (f *Footer) Validate() error {
	if f == nil {
		return nil
	}
	switch strings.ToLower(f.Position) {
	case "", "left", "center", "right":
		return nil
	default:
		return fmt.Errorf("%w: %q (must be left, center, or right)", ErrInvalidFooterPosition, f.Position)
	}
}

// Signature configures the signature block.
type Signature struct {
	Name         string
	Title        string
	Email        string
	Organization string
	ImagePath    string
	Links        []Link
	// Extended metadata fields
	Phone      string // Contact phone number (optional)
	Address    string // Postal address (optional, use YAML literal block for multiline)
	Department string // Department name (optional)
}

// Validate checks that signature settings are valid.
// Returns nil if s is nil (nil means no signature).
//
// Note: Only ImagePath is validated (file existence). Other fields like Email
// and Links are pure content that renders as-is - this is a PDF rendering tool,
// not a data validation tool. Users control their content.
func (s *Signature) Validate() error {
	if s == nil {
		return nil
	}
	// Validate image path exists if set (and not a URL)
	if s.ImagePath != "" && !fileutil.IsURL(s.ImagePath) && !fileutil.FileExists(s.ImagePath) {
		return fmt.Errorf("%w: %q", ErrSignatureImageNotFound, s.ImagePath)
	}
	return nil
}

// Link represents a clickable link.
type Link struct {
	Label string
	URL   string
}

// Option configures a Converter.
type Option func(*Converter)

// converterConfig holds internal configuration for Converter.
type converterConfig struct {
	timeout       time.Duration
	templateSet   *assets.TemplateSet
	assetPath     string // Path for WithAssetPath, resolved in New()
	styleInput    string // Raw input for WithStyle (name, path, or CSS content)
	resolvedStyle string // CSS content after resolution in New()
}

// defaultTimeout is used when no timeout is specified.
const defaultTimeout = 30 * time.Second

// WithTimeout sets the conversion timeout.
// Panics if d <= 0 (programmer error, similar to time.NewTicker).
func WithTimeout(d time.Duration) Option {
	if d <= 0 {
		panic("md2pdf: WithTimeout duration must be positive")
	}
	return func(c *Converter) {
		c.cfg.timeout = d
	}
}

// WithAssetLoader sets a custom asset loader for CSS styles and HTML templates.
// Use NewAssetLoader(basePath) to load from a custom directory with
// fallback to embedded assets, or implement AssetLoader for custom backends.
//
// Example:
//
//	loader, err := picoloom.NewAssetLoader("/path/to/assets")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	conv, err := picoloom.NewConverter(picoloom.WithAssetLoader(loader))
func WithAssetLoader(loader AssetLoader) Option {
	return func(c *Converter) {
		c.publicAssetLoader = loader
	}
}

// WithAssetPath configures asset loading from a filesystem directory.
// Custom assets take precedence; missing assets fall back to embedded defaults.
//
// The directory should contain:
//   - styles/{name}.css for CSS styles
//   - templates/{name}/cover.html and signature.html for template sets
//
// This is equivalent to calling NewAssetLoader(path) and WithAssetLoader().
// Returns error from NewConverter() if the path is invalid.
func WithAssetPath(path string) Option {
	return func(c *Converter) {
		c.cfg.assetPath = path
	}
}

// WithStyle sets the CSS style for all conversions.
// Accepts:
//   - Style name: "technical", "default", "corporate"
//   - File path: "./custom.css", "/path/to/style.css"
//   - CSS content: "body { font-size: 14px; }"
//
// Detection: paths contain / or \, CSS content contains {,
// otherwise treated as a style name.
func WithStyle(style string) Option {
	return func(c *Converter) {
		c.cfg.styleInput = style
	}
}

// WithTemplateSet sets a custom template set for cover and signature.
// Use this to override the default templates loaded from embedded assets.
//
// Example:
//
//	ts := picoloom.NewTemplateSet("custom", coverHTML, signatureHTML)
//	conv, err := picoloom.NewConverter(picoloom.WithTemplateSet(ts))
func WithTemplateSet(ts *TemplateSet) Option {
	return func(c *Converter) {
		if ts != nil {
			c.cfg.templateSet = &assets.TemplateSet{
				Name:      ts.Name,
				Cover:     ts.Cover,
				Signature: ts.Signature,
			}
		}
	}
}
