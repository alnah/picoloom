package picoloom

import (
	"fmt"
	"strings"
	"text/template"

	"github.com/alnah/picoloom/v2/internal/assets"
)

// defaultFontFamily is the standard font stack for PDF footers and generated content.
const defaultFontFamily = "sans-serif"

// watermarkFontSize is the font size for watermark text overlay.
const watermarkFontSize = "8rem"

// CSS overlay templates are embedded source files, parsed once at startup.
var watermarkCSSTemplate = mustParseCSSTemplate("watermark", assets.WatermarkOverlayTemplate())
var pageBreaksCSSTemplate = mustParseCSSTemplate("pagebreaks", assets.PageBreaksOverlayTemplate())

// watermarkCSSData holds the values substituted into watermarkCSSTemplate.
type watermarkCSSData struct {
	Content    string
	Angle      string
	FontSize   string
	Color      string
	Opacity    string
	FontFamily string
}

// pageBreaksCSSData holds the values substituted into pageBreaksCSSTemplate.
type pageBreaksCSSData struct {
	Orphans  int
	Widows   int
	BeforeH1 bool
	BeforeH2 bool
	BeforeH3 bool
}

// buildWatermarkCSS generates CSS for a diagonal background watermark.
// The watermark uses position:fixed to appear on all pages when printed.
func buildWatermarkCSS(w *Watermark) string {
	if w == nil || w.Text == "" {
		return ""
	}

	return renderCSSTemplate(watermarkCSSTemplate, watermarkCSSData{
		Content:    escapeCSSString(breakURLPattern(w.Text)),
		Angle:      fmt.Sprintf("%.1f", w.Angle),
		FontSize:   watermarkFontSize,
		Color:      w.Color,
		Opacity:    fmt.Sprintf("%.2f", w.Opacity),
		FontFamily: defaultFontFamily,
	})
}

// escapeCSSString escapes a string for safe use in CSS content property.
// Prevents CSS injection by escaping backslashes, quotes, and newlines.
func escapeCSSString(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `"`, `\"`)
	s = strings.ReplaceAll(s, "\n", `\A `)
	s = strings.ReplaceAll(s, "\r", "")
	return s
}

// breakURLPattern replaces ALL dots with a Unicode lookalike (ONE DOT LEADER U+2024)
// to prevent PDF viewers from auto-detecting URLs and making them clickable.
// The character ․ looks identical to . but is not recognized as a URL separator.
//
// Note: This affects all dots unconditionally, including version numbers (1.0.0),
// abbreviations (e.g.), and decimal numbers. This is intentional - the U+2024
// character is visually indistinguishable from a period in rendered output.
func breakURLPattern(text string) string {
	return strings.ReplaceAll(text, ".", "\u2024")
}

// buildPageBreaksCSS generates CSS for page break control.
// Always includes hardcoded rules for heading protection (break-after/inside: avoid).
// Configurable rules for page breaks before h1/h2/h3 and orphan/widow control.
func buildPageBreaksCSS(pb *PageBreaks) string {
	data := pageBreaksCSSData{
		Orphans: DefaultOrphans,
		Widows:  DefaultWidows,
	}
	if pb != nil {
		if pb.Orphans > 0 {
			data.Orphans = pb.Orphans
		}
		if pb.Widows > 0 {
			data.Widows = pb.Widows
		}
		data.BeforeH1 = pb.BeforeH1
		data.BeforeH2 = pb.BeforeH2
		data.BeforeH3 = pb.BeforeH3
	}
	return renderCSSTemplate(pageBreaksCSSTemplate, data)
}

// buildAdmonitionCSS returns the embedded structural admonition stylesheet.
func buildAdmonitionCSS() string {
	return assets.AdmonitionOverlayCSS()
}

// mustParseCSSTemplate parses an embedded overlay template, a startup failure.
func mustParseCSSTemplate(name, source string) *template.Template {
	return template.Must(template.New(name).Parse(source))
}

// renderCSSTemplate renders an overlay template. Execution only fails on
// programmer errors such as a template/data field mismatch.
func renderCSSTemplate(tmpl *template.Template, data any) string {
	var buf strings.Builder
	if err := tmpl.Execute(&buf, data); err != nil {
		panic(fmt.Sprintf("rendering CSS overlay: %v", err))
	}
	return buf.String()
}
