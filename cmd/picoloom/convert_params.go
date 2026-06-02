// Package main provides the md2pdf CLI.
//
// TRUST BOUNDARY: Config is validated at load time by Config.Validate().
// The buildXxxData() functions in this file transform validated config into
// library types. They do NOT re-validate because:
//   - Config.Validate() already checked constraints at load time
//   - Library's validateInput() catches any issues for direct library users
//   - Redundant validation creates maintenance burden and can drift
//
// Validation parity tests cover config-to-library builder drift.
package main

import (
	"path/filepath"
	"regexp"
	"strings"

	picoloom "github.com/alnah/picoloom/v2"
	"github.com/alnah/picoloom/v2/internal/config"
)

// conversionParams groups parameters shared across batch/file conversion.
type conversionParams struct {
	css        string
	footer     *picoloom.Footer
	signature  *picoloom.Signature
	page       *picoloom.PageSettings
	watermark  *picoloom.Watermark
	toc        *picoloom.TOC
	pageBreaks *picoloom.PageBreaks
	cfg        *config.Config
	htmlOnly   bool // Output HTML only, skip PDF
	htmlOutput bool // Output HTML alongside PDF
}

// buildSignatureData creates picoloom.Signature from config.
// Uses cfg.Author.* for author information.
// Department is always shown if defined (signature always displays it).
//
// Note: Image path validation happens at library boundary (Signature.Validate),
// not here. This function is a pure transformation from config to library type.
func buildSignatureData(cfg *config.Config, noSignature bool) *picoloom.Signature {
	if noSignature || !cfg.Signature.Enabled {
		return nil
	}

	// Convert config links to picoloom.Link
	links := make([]picoloom.Link, len(cfg.Signature.Links))
	for i, l := range cfg.Signature.Links {
		links[i] = picoloom.Link{Label: l.Label, URL: l.URL}
	}

	return &picoloom.Signature{
		Name:         cfg.Author.Name,
		Title:        cfg.Author.Title,
		Email:        cfg.Author.Email,
		Organization: cfg.Author.Organization,
		ImagePath:    cfg.Signature.ImagePath,
		Links:        links,
		Phone:        cfg.Author.Phone,
		Address:      cfg.Author.Address,
		Department:   cfg.Author.Department,
	}
}

// buildFooterData creates picoloom.Footer from config.
// Uses cfg.Document.Date and cfg.Document.Version for date/status.
// DocumentID is only shown if cfg.Footer.ShowDocumentID is true.
func buildFooterData(cfg *config.Config, noFooter bool) *picoloom.Footer {
	if noFooter || !cfg.Footer.Enabled {
		return nil
	}

	var docID string
	if cfg.Footer.ShowDocumentID {
		docID = cfg.Document.DocumentID
	}

	return &picoloom.Footer{
		Position:       cfg.Footer.Position,
		ShowPageNumber: cfg.Footer.ShowPageNumber,
		Date:           cfg.Document.Date,
		Status:         cfg.Document.Version,
		Text:           cfg.Footer.Text,
		DocumentID:     docID,
	}
}

// buildWatermarkData creates picoloom.Watermark from config.
// Flags are merged into config by mergeFlags before this is called.
func buildWatermarkData(cfg *config.Config) *picoloom.Watermark {
	if !cfg.Watermark.Enabled {
		return nil
	}

	w := &picoloom.Watermark{
		Text:    cfg.Watermark.Text,
		Color:   cfg.Watermark.Color,
		Opacity: cfg.Watermark.Opacity,
		Angle:   cfg.Watermark.Angle,
	}

	// Apply defaults for color and opacity.
	// Angle default is handled in mergeFlags to distinguish "not set" from "0".
	if w.Color == "" {
		w.Color = picoloom.DefaultWatermarkColor
	}
	if w.Opacity == 0 {
		w.Opacity = picoloom.DefaultWatermarkOpacity
	}

	return w
}

// buildPageSettings creates picoloom.PageSettings from config.
// Flags are merged into config by mergeFlags before this is called.
func buildPageSettings(cfg *config.Config) *picoloom.PageSettings {
	hasConfig := cfg.Page.Size != "" || cfg.Page.Orientation != "" || cfg.Page.Margin > 0

	if !hasConfig {
		return nil
	}

	ps := &picoloom.PageSettings{
		Size:        cfg.Page.Size,
		Orientation: cfg.Page.Orientation,
		Margin:      cfg.Page.Margin,
	}

	// Apply defaults
	if ps.Size == "" {
		ps.Size = picoloom.PageSizeLetter
	}
	if ps.Orientation == "" {
		ps.Orientation = picoloom.OrientationPortrait
	}
	if ps.Margin == 0 {
		ps.Margin = picoloom.DefaultMargin
	}

	return ps
}

// firstHeadingPattern matches the first # heading in markdown content.
var firstHeadingPattern = regexp.MustCompile(`(?m)^#\s+(.+)$`)

// extractFirstHeading extracts the first # heading from markdown content.
func extractFirstHeading(markdown string) string {
	matches := firstHeadingPattern.FindStringSubmatch(markdown)
	if len(matches) >= 2 {
		return strings.TrimSpace(matches[1])
	}
	return ""
}

// buildCoverData creates picoloom.Cover from config and markdown content.
// Uses cfg.Author.* and cfg.Document.* for metadata.
// Department is only shown if cfg.Cover.ShowDepartment is true.
func buildCoverData(cfg *config.Config, markdownContent, filename string) *picoloom.Cover {
	if !cfg.Cover.Enabled {
		return nil
	}

	c := &picoloom.Cover{
		Logo: cfg.Cover.Logo,
	}

	// Title: config -> H1 -> filename
	if cfg.Document.Title != "" {
		c.Title = cfg.Document.Title
	} else {
		c.Title = extractFirstHeading(markdownContent)
		if c.Title == "" {
			c.Title = strings.TrimSuffix(filepath.Base(filename), filepath.Ext(filename))
		}
	}

	c.Subtitle = cfg.Document.Subtitle
	c.Author = cfg.Author.Name
	c.AuthorTitle = cfg.Author.Title
	c.Organization = cfg.Author.Organization
	c.Date = cfg.Document.Date // Already resolved
	c.Version = cfg.Document.Version

	// Extended metadata fields
	c.ClientName = cfg.Document.ClientName
	c.ProjectName = cfg.Document.ProjectName
	c.DocumentType = cfg.Document.DocumentType
	c.DocumentID = cfg.Document.DocumentID
	c.Description = cfg.Document.Description

	// Department only if explicitly enabled on cover
	if cfg.Cover.ShowDepartment {
		c.Department = cfg.Author.Department
	}

	return c
}

// buildTOCData creates picoloom.TOC from config.
func buildTOCData(cfg *config.Config, tocFlags tocFlags) *picoloom.TOC {
	if tocFlags.disabled || !cfg.TOC.Enabled {
		return nil
	}

	maxDepth := cfg.TOC.MaxDepth
	if maxDepth == 0 {
		maxDepth = picoloom.DefaultTOCMaxDepth
	}

	toc := &picoloom.TOC{
		Title:    cfg.TOC.Title,
		MinDepth: cfg.TOC.MinDepth, // 0 = library defaults to 2
		MaxDepth: maxDepth,
	}

	return toc
}

// buildPageBreaksData creates picoloom.PageBreaks from config.
// Flags are merged into config by mergeFlags before this is called.
func buildPageBreaksData(cfg *config.Config) *picoloom.PageBreaks {
	if !cfg.PageBreaks.Enabled {
		return nil
	}

	pb := &picoloom.PageBreaks{
		BeforeH1: cfg.PageBreaks.BeforeH1,
		BeforeH2: cfg.PageBreaks.BeforeH2,
		BeforeH3: cfg.PageBreaks.BeforeH3,
		Orphans:  picoloom.DefaultOrphans,
		Widows:   picoloom.DefaultWidows,
	}

	if cfg.PageBreaks.Orphans > 0 {
		pb.Orphans = cfg.PageBreaks.Orphans
	}
	if cfg.PageBreaks.Widows > 0 {
		pb.Widows = cfg.PageBreaks.Widows
	}

	return pb
}
