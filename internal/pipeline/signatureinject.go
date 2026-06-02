package pipeline

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"html/template"
	"strings"
)

// ErrSignatureRender indicates signature template rendering failed.
var ErrSignatureRender = errors.New("signature template rendering failed")

// SignatureData holds signature information for injection into HTML.
type SignatureData struct {
	Name         string
	Title        string
	Email        string
	Organization string
	ImagePath    string
	Links        []SignatureLink
	// Extended metadata fields
	Phone      string
	Address    string
	Department string
}

// SignatureLink represents a clickable link in the signature block.
type SignatureLink struct {
	Label string
	URL   string
}

// SignatureInjector defines the contract for signature injection into HTML.
type SignatureInjector interface {
	InjectSignature(ctx context.Context, htmlContent string, data *SignatureData) (string, error)
}

// SignatureInjection renders and injects a signature block into HTML content.
type SignatureInjection struct {
	tmpl *template.Template
}

// NewSignatureInjection creates a SignatureInjection from template content.
// Returns error if the template cannot be parsed.
func NewSignatureInjection(tmplContent string) (*SignatureInjection, error) {
	tmpl, err := template.New("signature").Parse(tmplContent)
	if err != nil {
		return nil, fmt.Errorf("parsing signature template: %w", err)
	}

	return &SignatureInjection{tmpl: tmpl}, nil
}

// InjectSignature renders the signature template and injects it before </body>.
// If data is nil, returns htmlContent unchanged.
// Returns error if template rendering fails.
func (s *SignatureInjection) InjectSignature(ctx context.Context, htmlContent string, data *SignatureData) (string, error) {
	if data == nil {
		return htmlContent, nil
	}

	// Check for cancellation
	if ctx.Err() != nil {
		return "", ctx.Err()
	}

	var buf bytes.Buffer
	if err := s.tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("%w: %w", ErrSignatureRender, err)
	}

	signatureHTML := buf.String()
	lowerHTML := strings.ToLower(htmlContent)

	// Try inserting before </body>
	if idx := strings.Index(lowerHTML, "</body>"); idx != -1 {
		return htmlContent[:idx] + signatureHTML + htmlContent[idx:], nil
	}

	// Fallback: append to end
	return htmlContent + signatureHTML, nil
}
