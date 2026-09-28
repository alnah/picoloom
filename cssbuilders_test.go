package picoloom

// Notes:
// - escapeCSSString: tests CSS string escaping for quotes, backslashes, newlines
// - buildWatermarkCSS: tests watermark CSS generation with escaping
// - breakURLPattern: tests URL pattern breaking with dot leader replacement
// - buildPageBreaksCSS: tests page break CSS generation for headings and orphans/widows
// - buildAdmonitionCSS: tests structural admonition styles and type palettes
// - renderCSSTemplate: tests overlay template rendering failure handling

import (
	"strings"
	"testing"
	"text/template"
)

// ---------------------------------------------------------------------------
// TestEscapeCSSString - CSS String Escaping
// ---------------------------------------------------------------------------

func TestEscapeCSSString(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "simple text",
			input:    "DRAFT",
			expected: "DRAFT",
		},
		{
			name:     "text with spaces",
			input:    "FOR REVIEW",
			expected: "FOR REVIEW",
		},
		{
			name:     "escapes double quotes",
			input:    `DRAFT "v1"`,
			expected: `DRAFT \"v1\"`,
		},
		{
			name:     "escapes backslash",
			input:    `path\to\file`,
			expected: `path\\to\\file`,
		},
		{
			name:     "escapes newline",
			input:    "line1\nline2",
			expected: `line1\A line2`,
		},
		{
			name:     "removes carriage return",
			input:    "line1\r\nline2",
			expected: `line1\A line2`,
		},
		{
			name:     "CSS injection attempt - closing quote",
			input:    `DRAFT"; } body { display: none } .x { content: "`,
			expected: `DRAFT\"; } body { display: none } .x { content: \"`,
		},
		{
			name:     "CSS injection attempt - backslash escape",
			input:    `DRAFT\"; } body { display: none }`,
			expected: `DRAFT\\\"; } body { display: none }`,
		},
		{
			name:     "unicode preserved",
			input:    "BROUILLON",
			expected: "BROUILLON",
		},
		{
			name:     "percent preserved",
			input:    "100% done",
			expected: "100% done",
		},
		{
			name:     "mixed special characters",
			input:    "A\"B\\C\nD\rE",
			expected: `A\"B\\C\A DE`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := escapeCSSString(tt.input)
			if got != tt.expected {
				t.Errorf("escapeCSSString(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestBuildWatermarkCSS - Watermark CSS Generation
// ---------------------------------------------------------------------------

func TestBuildWatermarkCSS(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		watermark      *Watermark
		wantEmpty      bool
		wantContains   []string
		wantNotContain []string
	}{
		{
			name:      "nil watermark returns empty",
			watermark: nil,
			wantEmpty: true,
		},
		{
			name:      "empty text returns empty",
			watermark: &Watermark{Text: "", Color: "#888888", Opacity: 0.1, Angle: -45},
			wantEmpty: true,
		},
		{
			name:      "simple watermark",
			watermark: &Watermark{Text: "DRAFT", Color: "#888888", Opacity: 0.1, Angle: -45},
			wantContains: []string{
				`content: "DRAFT"`,
				"color: #888888",
				"opacity: 0.10",
				"rotate(-45.0deg)",
			},
		},
		{
			name:      "watermark with positive angle",
			watermark: &Watermark{Text: "TEST", Color: "#ff0000", Opacity: 0.5, Angle: 30},
			wantContains: []string{
				`content: "TEST"`,
				"color: #ff0000",
				"opacity: 0.50",
				"rotate(30.0deg)",
			},
		},
		{
			name:      "watermark text with quotes is escaped",
			watermark: &Watermark{Text: `DRAFT "v1"`, Color: "#888888", Opacity: 0.1, Angle: -45},
			wantContains: []string{
				`content: "DRAFT \"v1\""`,
			},
			wantNotContain: []string{
				`content: "DRAFT "v1""`, // unescaped quotes would break CSS
			},
		},
		{
			name:      "watermark text with backslash is escaped",
			watermark: &Watermark{Text: `A\B`, Color: "#888888", Opacity: 0.1, Angle: -45},
			wantContains: []string{
				`content: "A\\B"`,
			},
		},
		{
			name:      "CSS injection attempt is escaped",
			watermark: &Watermark{Text: `"; } body { display: none } .x { content: "`, Color: "#888888", Opacity: 0.1, Angle: -45},
			wantContains: []string{
				`content: "\"; } body { display: none } ` + "\u2024" + `x { content: \""`,
				"opacity: 0.10", // verify CSS structure is intact after injection attempt
			},
		},
		{
			name:      "watermark with newline in text",
			watermark: &Watermark{Text: "LINE1\nLINE2", Color: "#888888", Opacity: 0.1, Angle: -45},
			wantContains: []string{
				`content: "LINE1\A LINE2"`,
			},
		},
		{
			name:      "percent in text is preserved",
			watermark: &Watermark{Text: "50%", Color: "#888888", Opacity: 0.1, Angle: -45},
			wantContains: []string{
				`content: "50%"`,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := buildWatermarkCSS(tt.watermark)

			if tt.wantEmpty {
				if got != "" {
					t.Errorf("buildWatermarkCSS() = %q, want empty", got)
				}
				return
			}

			if got == "" {
				t.Fatal("buildWatermarkCSS() returned empty, want CSS")
			}

			for _, want := range tt.wantContains {
				if !strings.Contains(got, want) {
					t.Errorf("buildWatermarkCSS() missing %q\nGot:\n%s", want, got)
				}
			}

			for _, notWant := range tt.wantNotContain {
				if strings.Contains(got, notWant) {
					t.Errorf("buildWatermarkCSS() contains unwanted %q\nGot:\n%s", notWant, got)
				}
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestBreakURLPattern - URL Pattern Breaking
// ---------------------------------------------------------------------------

func TestBreakURLPattern(t *testing.T) {
	t.Parallel()

	// U+2024 ONE DOT LEADER - visually identical to period but not recognized as URL
	const dotLeader = "\u2024"

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "plain text unchanged",
			input:    "DRAFT",
			expected: "DRAFT",
		},
		{
			name:     "CONFIDENTIAL unchanged",
			input:    "CONFIDENTIAL",
			expected: "CONFIDENTIAL",
		},
		{
			name:     "domain.com dots replaced",
			input:    "domain.com",
			expected: "domain" + dotLeader + "com",
		},
		{
			name:     "domain.tech dots replaced",
			input:    "alnah.tech",
			expected: "alnah" + dotLeader + "tech",
		},
		{
			name:     "www.example.com all dots replaced",
			input:    "www.example.com",
			expected: "www" + dotLeader + "example" + dotLeader + "com",
		},
		{
			name:     "full URL with path",
			input:    "https://www.example.com/path",
			expected: "https://www" + dotLeader + "example" + dotLeader + "com/path",
		},
		{
			name:     "multiple dots in text",
			input:    "version 1.0.0",
			expected: "version 1" + dotLeader + "0" + dotLeader + "0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := breakURLPattern(tt.input)
			if got != tt.expected {
				t.Errorf("breakURLPattern(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestBuildPageBreaksCSS - Page Breaks CSS Generation
// ---------------------------------------------------------------------------

func TestBuildPageBreaksCSS(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		pageBreaks     *PageBreaks
		wantContains   []string
		wantNotContain []string
	}{
		{
			name:       "nil pageBreaks uses defaults",
			pageBreaks: nil,
			wantContains: []string{
				"break-after: avoid",
				"page-break-after: avoid",
				"break-inside: avoid",
				"page-break-inside: avoid",
				"orphans: 2",
				"widows: 2",
			},
			wantNotContain: []string{
				"break-before: page",
			},
		},
		{
			name:       "empty pageBreaks uses defaults",
			pageBreaks: &PageBreaks{},
			wantContains: []string{
				"orphans: 2",
				"widows: 2",
				"break-after: avoid",
			},
			wantNotContain: []string{
				"break-before: page",
			},
		},
		{
			name:       "custom orphans and widows",
			pageBreaks: &PageBreaks{Orphans: 3, Widows: 4},
			wantContains: []string{
				"orphans: 3",
				"widows: 4",
			},
		},
		{
			name:       "orphans 0 uses default",
			pageBreaks: &PageBreaks{Orphans: 0, Widows: 3},
			wantContains: []string{
				"orphans: 2",
				"widows: 3",
			},
		},
		{
			name:       "widows 0 uses default",
			pageBreaks: &PageBreaks{Orphans: 4, Widows: 0},
			wantContains: []string{
				"orphans: 4",
				"widows: 2",
			},
		},
		{
			name:       "BeforeH1 adds page break CSS",
			pageBreaks: &PageBreaks{BeforeH1: true},
			wantContains: []string{
				"/* Page breaks: before H1 */",
				"h1 {",
				"break-before: page",
				"page-break-before: always",
				"/* Exception: no break before first H1 if it's first element in body */",
				"body > h1:first-child",
			},
			wantNotContain: []string{
				"/* Page breaks: before H2 */",
				"/* Page breaks: before H3 */",
			},
		},
		{
			name:       "BeforeH2 adds page break CSS",
			pageBreaks: &PageBreaks{BeforeH2: true},
			wantContains: []string{
				"/* Page breaks: before H2 */",
				"h2 {",
				"break-before: page",
				"page-break-before: always",
			},
			wantNotContain: []string{
				"/* Page breaks: before H1 */",
				"/* Page breaks: before H3 */",
			},
		},
		{
			name:       "BeforeH3 adds page break CSS",
			pageBreaks: &PageBreaks{BeforeH3: true},
			wantContains: []string{
				"/* Page breaks: before H3 */",
				"h3 {",
				"break-before: page",
				"page-break-before: always",
			},
			wantNotContain: []string{
				"/* Page breaks: before H1 */",
				"/* Page breaks: before H2 */",
			},
		},
		{
			name:       "all heading breaks enabled",
			pageBreaks: &PageBreaks{BeforeH1: true, BeforeH2: true, BeforeH3: true},
			wantContains: []string{
				"/* Page breaks: before H1 */",
				"/* Page breaks: before H2 */",
				"/* Page breaks: before H3 */",
			},
		},
		{
			name:       "heading breaks with custom orphans widows",
			pageBreaks: &PageBreaks{BeforeH1: true, Orphans: 5, Widows: 5},
			wantContains: []string{
				"orphans: 5",
				"widows: 5",
				"/* Page breaks: before H1 */",
			},
		},
		{
			name:       "always includes hardcoded heading protection",
			pageBreaks: &PageBreaks{BeforeH2: true},
			wantContains: []string{
				"h1, h2, h3, h4, h5, h6 {",
				"break-after: avoid",
				"page-break-after: avoid",
				"break-inside: avoid",
				"page-break-inside: avoid",
			},
		},
		{
			name:       "always includes orphan widow rules for content elements",
			pageBreaks: &PageBreaks{},
			wantContains: []string{
				"p, li, dd, dt, blockquote {",
				"orphans:",
				"widows:",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := buildPageBreaksCSS(tt.pageBreaks)

			if got == "" {
				t.Fatal("buildPageBreaksCSS() returned empty, want CSS")
			}

			for _, want := range tt.wantContains {
				if !strings.Contains(got, want) {
					t.Errorf("buildPageBreaksCSS() missing %q\nGot:\n%s", want, got)
				}
			}

			for _, notWant := range tt.wantNotContain {
				if strings.Contains(got, notWant) {
					t.Errorf("buildPageBreaksCSS() contains unwanted %q\nGot:\n%s", notWant, got)
				}
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestBuildAdmonitionCSS - Admonition CSS Overlay
// ---------------------------------------------------------------------------

func TestBuildAdmonitionCSS_ContainsStructure(t *testing.T) {
	t.Parallel()

	got := buildAdmonitionCSS()

	wantContains := []string{
		"/* Admonitions: blockquote alerts and ::: fences */",
		".admonition {",
		"--admonition-accent: var(--color-accent-emphasis, #333);",
		"--admonition-bg: var(--color-canvas-subtle, #f5f5f5);",
		"border-left: 4px solid var(--admonition-accent);",
		"background: var(--admonition-bg);",
		"break-inside: auto;",
		"page-break-inside: auto;",
		".admonition-title {",
		"font-weight: 600;",
		"color: var(--admonition-accent);",
		"break-after: avoid;",
		"page-break-after: avoid;",
		".admonition > *:last-child {",
		".admonition .admonition {",
		".admonition-note {",
		".admonition-tip {",
		".admonition-important {",
		".admonition-warning {",
		".admonition-caution {",
	}

	for _, want := range wantContains {
		if !strings.Contains(got, want) {
			t.Errorf("buildAdmonitionCSS() missing %q\nGot:\n%s", want, got)
		}
	}

	if strings.Contains(got, "color-mix(") {
		t.Errorf("buildAdmonitionCSS() must not use color-mix\nGot:\n%s", got)
	}
}

func TestBuildAdmonitionCSS_PaletteVariables(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		selector string
		accent   string
		border   string
		width    bool
	}{
		{
			name:     "note",
			selector: ".admonition-note {",
			accent:   "--admonition-accent: var(--color-accent-fg, #0969da);",
			border:   "border-left-style: solid;",
		},
		{
			name:     "tip",
			selector: ".admonition-tip {",
			accent:   "--admonition-accent: var(--color-success-fg, #1a7f37);",
			border:   "border-left-style: dashed;",
		},
		{
			name:     "important",
			selector: ".admonition-important {",
			accent:   "--admonition-accent: var(--color-accent-emphasis, #0969da);",
			border:   "border-left-style: double;",
			width:    true,
		},
		{
			name:     "warning",
			selector: ".admonition-warning {",
			accent:   "--admonition-accent: var(--color-attention-fg, #9a6700);",
			border:   "border-left-style: dotted;",
		},
		{
			name:     "caution",
			selector: ".admonition-caution {",
			accent:   "--admonition-accent: var(--color-danger-fg, #cf222e);",
			border:   "border-left-style: solid;",
			width:    true,
		},
	}

	got := buildAdmonitionCSS()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			block := cssRuleBlock(t, got, tt.selector)
			if !strings.Contains(block, tt.accent) {
				t.Errorf("%s block missing %q\nGot:\n%s", tt.selector, tt.accent, block)
			}
			if !strings.Contains(block, tt.border) {
				t.Errorf("%s block missing %q\nGot:\n%s", tt.selector, tt.border, block)
			}
			if hasWidth := strings.Contains(block, "border-left-width: 6px;"); hasWidth != tt.width {
				t.Errorf("%s border-left-width present = %v, want %v\nGot:\n%s", tt.selector, hasWidth, tt.width, block)
			}
		})
	}
}

// cssRuleBlock extracts the declaration block that follows selector.
func cssRuleBlock(t *testing.T, css, selector string) string {
	t.Helper()

	start := strings.Index(css, selector)
	if start < 0 {
		t.Fatalf("selector %q not found in CSS\nGot:\n%s", selector, css)
	}
	end := strings.Index(css[start:], "}")
	if end < 0 {
		t.Fatalf("selector %q block not closed\nGot:\n%s", selector, css)
	}
	return css[start : start+end]
}

// ---------------------------------------------------------------------------
// TestBuildCombinedCSS - Stylesheet Layer Order
// ---------------------------------------------------------------------------

func TestBuildCombinedCSS_Order(t *testing.T) {
	t.Parallel()

	t.Run("overlay before base", func(t *testing.T) {
		t.Parallel()

		got := buildCombinedCSS("/* base */", Input{})
		assertOrder(t, got, "/* Page breaks", ".admonition {", "/* base */")
	})

	t.Run("full layering", func(t *testing.T) {
		t.Parallel()

		input := Input{
			CSS:        "/* user */",
			Watermark:  &Watermark{Text: "DRAFT", Color: "#888888", Opacity: 0.1, Angle: -45},
			PageBreaks: &PageBreaks{BeforeH1: true},
		}
		got := buildCombinedCSS("/* base */", input)
		assertOrder(t, got, "/* Page breaks", ".admonition {", "/* Watermark */", "/* base */", "/* user */")
	})
}

// assertOrder checks that the markers appear in the given order.
func assertOrder(t *testing.T, css string, markers ...string) {
	t.Helper()

	prev := -1
	for _, marker := range markers {
		idx := strings.Index(css, marker)
		if idx < 0 {
			t.Fatalf("marker %q not found in CSS\nGot:\n%s", marker, css)
		}
		if idx <= prev {
			t.Fatalf("marker %q out of order\nGot:\n%s", marker, css)
		}
		prev = idx
	}
}

// ---------------------------------------------------------------------------
// TestRenderCSSTemplate - Overlay Template Failure Handling
// ---------------------------------------------------------------------------

func TestRenderCSSTemplate_PanicsOnDataMismatch(t *testing.T) {
	t.Parallel()

	tmpl := template.Must(template.New("mismatch").Parse("{{.Missing}}"))

	defer func() {
		if recover() == nil {
			t.Fatal("renderCSSTemplate() did not panic on data mismatch")
		}
	}()
	renderCSSTemplate(tmpl, struct{}{})
}
