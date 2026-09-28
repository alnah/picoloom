package pipeline

import (
	"bytes"
	"regexp"
	"strings"

	gast "github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
)

// alertMarkerPattern matches a GitHub-style alert marker with an optional
// plain-text title on the same line.
var alertMarkerPattern = regexp.MustCompile(`^\[!([A-Za-z]+)\](?:[ \t]+(.+))?$`)

// NewAdmonitionQuoteTransformer returns an AST transformer that converts
// marked blockquotes into Admonition nodes.
func NewAdmonitionQuoteTransformer() parser.ASTTransformer {
	return &admonitionQuoteTransformer{}
}

type admonitionQuoteTransformer struct{}

// Transform implements parser.ASTTransformer.
func (t *admonitionQuoteTransformer) Transform(doc *gast.Document, reader text.Reader, pc parser.Context) {
	transformAdmonitionQuotes(doc, reader.Source())
}

// transformAdmonitionQuotes walks the tree depth-first so nested quotes are
// converted before their parent is inspected.
func transformAdmonitionQuotes(node gast.Node, source []byte) {
	for child := node.FirstChild(); child != nil; {
		next := child.NextSibling()
		transformAdmonitionQuotes(child, source)

		if quote, ok := child.(*gast.Blockquote); ok {
			if admonition := convertBlockQuote(quote, source); admonition != nil {
				parent := quote.Parent()
				parent.ReplaceChild(parent, quote, admonition)
			}
		}
		child = next
	}
}

// convertBlockQuote returns an Admonition when the first paragraph starts
// with a known alert marker, or nil to leave the quote untouched.
func convertBlockQuote(quote *gast.Blockquote, source []byte) *Admonition {
	paragraph, ok := quote.FirstChild().(*gast.Paragraph)
	if !ok {
		return nil
	}
	kind, title, ok := parseAlertMarker(paragraph, source)
	if !ok {
		return nil
	}

	stripAlertMarkerLine(paragraph)
	if paragraph.FirstChild() == nil {
		quote.RemoveChild(quote, paragraph)
	}

	admonition := NewAdmonition(kind, title)
	for child := quote.FirstChild(); child != nil; {
		next := child.NextSibling()
		quote.RemoveChild(quote, child)
		admonition.AppendChild(admonition, child)
		child = next
	}
	return admonition
}

// parseAlertMarker extracts the kind and title from the first line of the
// paragraph. The line is read from the source so the title stays plain text.
func parseAlertMarker(paragraph *gast.Paragraph, source []byte) (AdmonitionKind, string, bool) {
	line := paragraph.Lines().At(0)
	trimmed := bytes.TrimSpace(line.Value(source))
	matches := alertMarkerPattern.FindSubmatch(trimmed)
	if matches == nil {
		return AdmonitionNote, "", false
	}
	kind, ok := parseAdmonitionKind(string(matches[1]))
	if !ok {
		return AdmonitionNote, "", false
	}
	return kind, strings.TrimSpace(string(matches[2])), true
}

// stripAlertMarkerLine removes the inline nodes of the marker line,
// including the line break that ends it.
func stripAlertMarkerLine(paragraph *gast.Paragraph) {
	for child := paragraph.FirstChild(); child != nil; {
		next := child.NextSibling()
		paragraph.RemoveChild(paragraph, child)

		if textNode, ok := child.(*gast.Text); ok {
			if textNode.SoftLineBreak() || textNode.HardLineBreak() {
				return
			}
		}
		child = next
	}
}
