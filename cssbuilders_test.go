package md2pdf

// Notes:
// - escapeCSSString: tests CSS string escaping for quotes, backslashes, newlines
// - buildWatermarkCSS: tests watermark CSS generation with escaping
// - breakURLPattern: tests URL pattern breaking with dot leader replacement
// - buildPageBreaksCSS: tests page break CSS generation for headings and orphans/widows

import (
	"strings"
	"testing"
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
