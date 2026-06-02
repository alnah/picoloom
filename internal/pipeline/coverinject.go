package pipeline

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"html/template"
	"strings"
)

// ErrCoverRender indicates cover template rendering failed.
var ErrCoverRender = errors.New("cover template rendering failed")

// CoverData holds cover page information for injection into HTML.
type CoverData struct {
	Title        string
	Subtitle     string
	Logo         string
	Author       string
	AuthorTitle  string
	Organization string
	Date         string
	Version      string
	// Extended metadata fields
	ClientName   string
	ProjectName  string
	DocumentType string
	DocumentID   string
	Description  string
	Department   string // From author config (DRY)
}

// CoverInjector defines the contract for cover injection into HTML.
type CoverInjector interface {
	InjectCover(ctx context.Context, htmlContent string, data *CoverData) (string, error)
}

// CoverInjection renders and injects a cover page into HTML content.
type CoverInjection struct {
	tmpl *template.Template
}

// NewCoverInjection creates a CoverInjection from template content.
// Returns error if the template cannot be parsed.
func NewCoverInjection(tmplContent string) (*CoverInjection, error) {
	tmpl, err := template.New("cover").Parse(tmplContent)
	if err != nil {
		return nil, fmt.Errorf("parsing cover template: %w", err)
	}

	return &CoverInjection{tmpl: tmpl}, nil
}

// InjectCover renders the cover template and injects it after <body>.
// If data is nil, returns htmlContent unchanged.
// Returns error if template rendering fails.
func (c *CoverInjection) InjectCover(ctx context.Context, htmlContent string, data *CoverData) (string, error) {
	if data == nil {
		return htmlContent, nil
	}

	// Check for cancellation
	if ctx.Err() != nil {
		return "", ctx.Err()
	}

	var buf bytes.Buffer
	if err := c.tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("%w: %w", ErrCoverRender, err)
	}

	coverHTML := buf.String()
	lowerHTML := strings.ToLower(htmlContent)

	// Try inserting after <body>
	if idx := strings.Index(lowerHTML, "<body"); idx != -1 {
		// Find the closing > of <body...>
		closeIdx := strings.Index(htmlContent[idx:], ">")
		if closeIdx != -1 {
			insertPos := idx + closeIdx + 1
			return htmlContent[:insertPos] + coverHTML + htmlContent[insertPos:], nil
		}
	}

	// Fallback: prepend
	return coverHTML + htmlContent, nil
}
