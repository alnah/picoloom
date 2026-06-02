package pipeline

import (
	"bytes"
	"context"
	"errors"
	"fmt"

	chromahtml "github.com/alecthomas/chroma/v2/formatters/html"
	"github.com/yuin/goldmark"
	highlighting "github.com/yuin/goldmark-highlighting/v2"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer/html"
)

// ErrHTMLConversion indicates HTML conversion failed.
var ErrHTMLConversion = errors.New("HTML conversion failed")

// htmlTemplate wraps Goldmark's fragment output in a complete HTML5 document.
const htmlTemplate = `<!DOCTYPE html>
<html>
<head>
<meta charset="utf-8">
<title>Document</title>
</head>
<body>
%s
</body>
</html>`

// HTMLConverter abstracts Markdown to HTML conversion.
type HTMLConverter interface {
	ToHTML(ctx context.Context, content string) (string, error)
}

// GoldmarkConverter converts Markdown to HTML using goldmark (pure Go).
type GoldmarkConverter struct {
	md goldmark.Markdown
}

// NewGoldmarkConverter creates a GoldmarkConverter with GFM extensions and syntax highlighting.
func NewGoldmarkConverter() *GoldmarkConverter {
	md := goldmark.New(
		goldmark.WithExtensions(
			extension.GFM,      // Tables, strikethrough, autolinks, task lists
			extension.Footnote, // [^1] footnotes
			highlighting.NewHighlighting(
				highlighting.WithFormatOptions(
					chromahtml.WithClasses(true), // CSS classes for smaller HTML and external stylesheet control
				),
			),
		),
		goldmark.WithParserOptions(
			parser.WithAutoHeadingID(), // Generate IDs for headings (required for TOC)
		),
		goldmark.WithRendererOptions(
			html.WithHardWraps(), // Treat newlines as <br>
			html.WithXHTML(),     // Self-closing tags
			// Note: WithUnsafe() intentionally NOT used for security.
			// The ==highlight== feature uses placeholders converted after Goldmark.
		),
	)
	return &GoldmarkConverter{md: md}
}

// ToHTML converts Markdown content to a standalone HTML5 document.
//
// Cancellation is checked before starting conversion and again while waiting
// for conversion to finish. Goldmark does not accept context.Context and is not
// internally interrupted after md.Convert starts; if ctx is canceled mid-run,
// this method returns ctx.Err() while the in-flight conversion finishes in its
// worker goroutine.
func (c *GoldmarkConverter) ToHTML(ctx context.Context, content string) (string, error) {
	// Fast path: check context before starting.
	if err := ctx.Err(); err != nil {
		return "", err
	}

	type result struct {
		html string
		err  error
	}

	done := make(chan result, 1)

	go func() {
		var buf bytes.Buffer
		if err := c.md.Convert([]byte(content), &buf); err != nil {
			done <- result{err: fmt.Errorf("%w: %w", ErrHTMLConversion, err)}
			return
		}
		done <- result{html: fmt.Sprintf(htmlTemplate, buf.String())}
	}()

	select {
	case <-ctx.Done():
		return "", ctx.Err()
	case r := <-done:
		return r.html, r.err
	}
}
