package pipeline

import (
	"context"
	"fmt"
	"html"
	"regexp"
	"strconv"
	"strings"
)

// TOCData holds TOC configuration for injection.
type TOCData struct {
	Title    string
	MinDepth int // Minimum heading level (default: 2, skips H1)
	MaxDepth int // Maximum heading level (default: 3)
}

// TOCInjector defines the contract for TOC injection into HTML.
type TOCInjector interface {
	InjectTOC(ctx context.Context, htmlContent string, data *TOCData) (string, error)
}

// headingInfo represents an extracted heading from HTML.
type headingInfo struct {
	Level int    // 1-6
	ID    string // anchor ID
	Text  string // heading text content
}

// headingPattern matches h1-h6 tags with id attribute.
// Captures: 1=level, 2=id, 3=inner HTML (may contain inline tags)
var headingPattern = regexp.MustCompile(`(?is)<h([1-6])[^>]*\bid="([^"]*)"[^>]*>(.*?)</h[1-6]>`)

// htmlTagPattern matches HTML tags for stripping from heading text.
var htmlTagPattern = regexp.MustCompile(`<[^>]*>`)

// stripHTMLTags removes HTML tags from a string, decodes HTML entities,
// and trims whitespace. Decoding entities is essential to avoid double-encoding
// when the text is later escaped for HTML output (e.g., in TOC generation).
func stripHTMLTags(s string) string {
	s = htmlTagPattern.ReplaceAllString(s, "")
	s = html.UnescapeString(s)
	return strings.TrimSpace(s)
}

// extractHeadings parses HTML and returns headings between minDepth and maxDepth.
// Headings without IDs are skipped.
func extractHeadings(htmlContent string, minDepth, maxDepth int) []headingInfo {
	matches := headingPattern.FindAllStringSubmatch(htmlContent, -1)
	if len(matches) == 0 {
		return nil
	}

	var headings []headingInfo
	for _, m := range matches {
		level, _ := strconv.Atoi(m[1])
		if level < minDepth || level > maxDepth {
			continue
		}
		headings = append(headings, headingInfo{
			Level: level,
			ID:    m[2],
			Text:  stripHTMLTags(m[3]),
		})
	}
	return headings
}

// numberingState tracks hierarchical numbering for TOC entries.
// Supports normalization (first heading becomes level 1) and gap skipping.
type numberingState struct {
	counters     [6]int // counters[0] = level 1 count, etc.
	minLevelSeen int    // for normalization (0 = not set)
	lastLevel    int    // for tracking parent relationships
}

// newNumberingState creates a new numbering state.
func newNumberingState() *numberingState {
	return &numberingState{minLevelSeen: 0, lastLevel: 0}
}

// next returns the next number string and effective depth for the given heading level.
// Handles normalization and gap skipping.
// The effective depth is used for nesting decisions in TOC generation.
func (n *numberingState) next(level int) (numStr string, effectiveDepth int) {
	// Initialize minLevelSeen on first heading
	if n.minLevelSeen == 0 {
		n.minLevelSeen = level
	}

	// Calculate effective depth (1-based, normalized)
	effectiveDepth = level - n.minLevelSeen + 1
	if effectiveDepth < 1 {
		effectiveDepth = 1
	}

	// Handle gap skipping: if we jump levels, treat as direct child
	// E.g., H1 -> H3 becomes depth 1 -> depth 2 (not depth 3)
	if n.lastLevel > 0 && effectiveDepth > n.lastLevel+1 {
		effectiveDepth = n.lastLevel + 1
	}

	// Reset deeper level counters
	for i := effectiveDepth; i < 6; i++ {
		n.counters[i] = 0
	}

	// Increment current level
	n.counters[effectiveDepth-1]++
	n.lastLevel = effectiveDepth

	// Build number string: "1.2.3."
	var parts []string
	for i := 0; i < effectiveDepth; i++ {
		parts = append(parts, strconv.Itoa(n.counters[i]))
	}
	return strings.Join(parts, ".") + ".", effectiveDepth
}

// generateNumberedTOC creates HTML for a numbered table of contents.
// Uses <div> elements instead of <ul>/<li> to avoid CSS list-style conflicts.
func generateNumberedTOC(headings []headingInfo, title string) string {
	if len(headings) == 0 {
		return ""
	}

	var buf strings.Builder
	buf.WriteString(`<nav class="toc">`)

	if title != "" {
		buf.WriteString(`<h2 class="toc-title">`)
		buf.WriteString(html.EscapeString(title))
		buf.WriteString(`</h2>`)
	}

	buf.WriteString(`<div class="toc-list">`)

	numbering := newNumberingState()

	for _, h := range headings {
		// Get number and effective depth (handles normalization and gap skipping)
		num, effectiveDepth := numbering.next(h.Level)

		// Calculate indentation: (depth - 1) * 1.5em
		indent := float64(effectiveDepth-1) * 1.5

		// Write the TOC item
		buf.WriteString(`<div class="toc-item"`)
		if indent > 0 {
			buf.WriteString(fmt.Sprintf(` style="padding-left:%.1fem"`, indent))
		}
		buf.WriteString(`><a href="#`)
		buf.WriteString(html.EscapeString(h.ID))
		buf.WriteString(`">`)
		buf.WriteString(num)
		buf.WriteString(` `)
		buf.WriteString(html.EscapeString(h.Text))
		buf.WriteString(`</a></div>`)
	}

	buf.WriteString(`</div></nav>`)
	return buf.String()
}

// TOCInjection implements TOCInjector.
type TOCInjection struct{}

// NewTOCInjection creates a new TOC injector.
func NewTOCInjection() *TOCInjection {
	return &TOCInjection{}
}

// InjectTOC extracts headings and injects a numbered TOC after the cover page.
// If data is nil, returns htmlContent unchanged.
func (t *TOCInjection) InjectTOC(ctx context.Context, htmlContent string, data *TOCData) (string, error) {
	if data == nil {
		return htmlContent, nil
	}

	// Check for cancellation
	if ctx.Err() != nil {
		return "", ctx.Err()
	}

	// Extract headings
	headings := extractHeadings(htmlContent, data.MinDepth, data.MaxDepth)
	if len(headings) == 0 {
		return htmlContent, nil
	}

	// Generate TOC HTML
	tocHTML := generateNumberedTOC(headings, data.Title)
	if tocHTML == "" {
		return htmlContent, nil
	}

	lowerHTML := strings.ToLower(htmlContent)

	// Try inserting after cover page marker.
	// Note: We use <span data-cover-end> instead of <!-- cover-end --> comment
	// because html/template strips HTML comments for security reasons.
	coverEndPattern := regexp.MustCompile(`(?i)</div>\s*</section>\s*<span[^>]*data-cover-end[^>]*>\s*</span>`)
	if loc := coverEndPattern.FindStringIndex(htmlContent); loc != nil {
		insertPos := loc[1]
		return htmlContent[:insertPos] + tocHTML + htmlContent[insertPos:], nil
	}

	// Fallback: insert after <body> tag
	if idx := strings.Index(lowerHTML, "<body"); idx != -1 {
		closeIdx := strings.Index(htmlContent[idx:], ">")
		if closeIdx != -1 {
			insertPos := idx + closeIdx + 1
			return htmlContent[:insertPos] + tocHTML + htmlContent[insertPos:], nil
		}
	}

	// Last fallback: prepend
	return tocHTML + htmlContent, nil
}
