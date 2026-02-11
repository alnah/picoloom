package pipeline

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestNormalizeLineEndings(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "LF unchanged",
			input:    "line1\nline2\nline3",
			expected: "line1\nline2\nline3",
		},
		{
			name:     "CRLF to LF",
			input:    "line1\r\nline2\r\nline3",
			expected: "line1\nline2\nline3",
		},
		{
			name:     "CR to LF",
			input:    "line1\rline2\rline3",
			expected: "line1\nline2\nline3",
		},
		{
			name:     "mixed line endings",
			input:    "line1\r\nline2\rline3\nline4",
			expected: "line1\nline2\nline3\nline4",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := normalizeLineEndings(tt.input)
			if got != tt.expected {
				t.Errorf("normalizeLineEndings() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestCompressBlankLines(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "single blank line unchanged",
			input:    "line1\n\nline2",
			expected: "line1\n\nline2",
		},
		{
			name:     "two blank lines compressed to two newlines",
			input:    "line1\n\n\nline2",
			expected: "line1\n\nline2",
		},
		{
			name:     "three blank lines compressed to two",
			input:    "line1\n\n\n\nline2",
			expected: "line1\n\nline2",
		},
		{
			name:     "five blank lines compressed to two",
			input:    "line1\n\n\n\n\n\nline2",
			expected: "line1\n\nline2",
		},
		{
			name:     "multiple groups compressed",
			input:    "a\n\n\n\nb\n\n\n\n\nc",
			expected: "a\n\nb\n\nc",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := compressBlankLines(tt.input)
			if got != tt.expected {
				t.Errorf("compressBlankLines() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestConvertHighlights(t *testing.T) {
	t.Parallel()

	// Helper to build expected output with placeholders
	mark := func(s string) string {
		return MarkStartPlaceholder + s + MarkEndPlaceholder
	}

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "single highlight",
			input:    "This is ==highlighted== text",
			expected: "This is " + mark("highlighted") + " text",
		},
		{
			name:     "multiple highlights",
			input:    "==one== and ==two==",
			expected: mark("one") + " and " + mark("two"),
		},
		{
			name:     "empty highlight",
			input:    "empty ==== here",
			expected: "empty " + mark("") + " here",
		},
		{
			name:     "highlight with spaces",
			input:    "==hello world==",
			expected: mark("hello world"),
		},
		{
			name:     "no highlights",
			input:    "plain text",
			expected: "plain text",
		},
		{
			name:     "unclosed highlight unchanged",
			input:    "==unclosed",
			expected: "==unclosed",
		},
		{
			name:     "unicode highlight",
			input:    "This is ==日本語== text",
			expected: "This is " + mark("日本語") + " text",
		},
		{
			name:     "triple equals captures inner equals with trailing",
			input:    "===not highlight===",
			expected: mark("=not highlight") + "=",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := convertHighlights(tt.input)
			if got != tt.expected {
				t.Errorf("convertHighlights() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestConvertMarkPlaceholders(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "single placeholder",
			input:    "text " + MarkStartPlaceholder + "highlighted" + MarkEndPlaceholder + " more",
			expected: "text <mark>highlighted</mark> more",
		},
		{
			name:     "multiple placeholders",
			input:    MarkStartPlaceholder + "one" + MarkEndPlaceholder + " and " + MarkStartPlaceholder + "two" + MarkEndPlaceholder,
			expected: "<mark>one</mark> and <mark>two</mark>",
		},
		{
			name:     "no placeholders",
			input:    "plain text without markers",
			expected: "plain text without markers",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "nested in HTML",
			input:    "<p>" + MarkStartPlaceholder + "important" + MarkEndPlaceholder + "</p>",
			expected: "<p><mark>important</mark></p>",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := ConvertMarkPlaceholders(tt.input)
			if got != tt.expected {
				t.Errorf("ConvertMarkPlaceholders() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestStripFrontmatter(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		// Well-formed frontmatter (should be stripped)
		{
			name:     "basic frontmatter",
			input:    "---\ntitle: Test\n---\n# Content",
			expected: "# Content",
		},
		{
			name:     "frontmatter with multiple keys",
			input:    "---\ntitle: Test\nauthor: John\ndate: 2024-01-15\n---\nContent",
			expected: "Content",
		},
		{
			name:     "empty frontmatter with blank line",
			input:    "---\n\n---\nContent",
			expected: "Content",
		},
		{
			name:     "frontmatter after blank line is preserved",
			input:    "  \n---\ntitle: Test\n---\nContent",
			expected: "  \n---\ntitle: Test\n---\nContent",
		},
		{
			name:     "frontmatter with leading spaces before delimiter",
			input:    " ---\ntitle: Test\n---\nContent",
			expected: "Content",
		},
		{
			name:     "frontmatter with leading tab before delimiter",
			input:    "\t---\ntitle: Test\n---\nContent",
			expected: "Content",
		},
		{
			name:     "frontmatter with trailing spaces after opening delimiter",
			input:    "---  \ntitle: Test\n---\nContent",
			expected: "Content",
		},
		{
			name:     "frontmatter with multi-line values",
			input:    "---\ndescription: |\n  Line 1\n  Line 2\n---\nContent",
			expected: "Content",
		},
		{
			name:     "frontmatter with dashes inside",
			input:    "---\ncode: some---value\n---\nContent",
			expected: "Content",
		},

		// Malformed frontmatter (should NOT be stripped, left intact)
		{
			name:     "missing closing delimiter",
			input:    "---\ntitle: Test\nContent",
			expected: "---\ntitle: Test\nContent",
		},
		{
			name:     "missing opening delimiter",
			input:    "title: Test\n---\nContent",
			expected: "title: Test\n---\nContent",
		},
		{
			name:     "single delimiter only",
			input:    "---\nContent",
			expected: "---\nContent",
		},
		{
			name:     "closing delimiter without newline before",
			input:    "---\ntitle: Test---\nContent",
			expected: "---\ntitle: Test---\nContent",
		},

		// No frontmatter (should be unchanged)
		{
			name:     "plain markdown",
			input:    "# Heading\nContent",
			expected: "# Heading\nContent",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "whitespace only",
			input:    "  \n\n  ",
			expected: "  \n\n  ",
		},

		// Edge cases
		{
			name:     "multiple frontmatter blocks only strips first",
			input:    "---\na: 1\n---\nText\n---\nb: 2\n---\nMore",
			expected: "Text\n---\nb: 2\n---\nMore",
		},
		{
			name:     "frontmatter not at start is preserved",
			input:    "Text\n---\ntitle: Test\n---\nMore",
			expected: "Text\n---\ntitle: Test\n---\nMore",
		},
		{
			name:     "frontmatter in code block is preserved",
			input:    "```\n---\ncode\n---\n```",
			expected: "```\n---\ncode\n---\n```",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := stripFrontmatter(tt.input)
			if got != tt.expected {
				t.Errorf("stripFrontmatter():\ngot:  %q\nwant: %q", got, tt.expected)
			}
		})
	}
}

func TestCommonMarkPreprocessor_PreprocessMarkdown_CancelledContext(t *testing.T) {
	t.Parallel()

	preprocessor := &CommonMarkPreprocessor{}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	input := "---\ntitle: Test\n---\n==highlight==\r\n\r\n\r\nEnd"
	got := preprocessor.PreprocessMarkdown(ctx, input)
	if got != input {
		t.Errorf("PreprocessMarkdown() with cancelled context should return input unchanged\ngot:  %q\nwant: %q", got, input)
	}
}

func TestCommonMarkPreprocessor_PreprocessMarkdown(t *testing.T) {
	t.Parallel()

	// Helper to build expected output with placeholders
	mark := func(s string) string {
		return MarkStartPlaceholder + s + MarkEndPlaceholder
	}

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
			name:     "plain text unchanged",
			input:    "Hello world",
			expected: "Hello world",
		},
		{
			name:     "CRLF normalized to LF",
			input:    "line1\r\nline2\r\nline3",
			expected: "line1\nline2\nline3",
		},
		{
			name:     "CR normalized to LF",
			input:    "line1\rline2\rline3",
			expected: "line1\nline2\nline3",
		},
		{
			name:     "highlights converted to placeholders",
			input:    "This is ==important== text",
			expected: "This is " + mark("important") + " text",
		},
		{
			name:     "multiple highlights converted to placeholders",
			input:    "==one== and ==two==",
			expected: mark("one") + " and " + mark("two"),
		},
		{
			name:     "multiple blank lines compressed to two",
			input:    "a\n\n\n\n\nb",
			expected: "a\n\nb",
		},
		{
			name:     "frontmatter stripped from pipeline",
			input:    "---\ntitle: Test\n---\n# Content",
			expected: "# Content",
		},
		{
			name:     "full pipeline: normalize, strip, highlight, compress",
			input:    "---\r\ntitle: Test\r\n---\r\nTitle\r\n\r\n\r\n\r\nText with ==highlight==\r\n\r\n\r\nEnd",
			expected: "Title\n\nText with " + mark("highlight") + "\n\nEnd",
		},
		{
			name:     "full pipeline: normalize, highlight, compress",
			input:    "Title\r\n\r\n\r\n\r\nText with ==highlight==\r\n\r\n\r\nEnd",
			expected: "Title\n\nText with " + mark("highlight") + "\n\nEnd",
		},
	}

	preprocessor := &CommonMarkPreprocessor{}
	ctx := context.Background()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := preprocessor.PreprocessMarkdown(ctx, tt.input)
			if got != tt.expected {
				t.Errorf("PreprocessMarkdown():\ngot:  %q\nwant: %q", got, tt.expected)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Benchmark Tests
// ---------------------------------------------------------------------------

func BenchmarkStripFrontmatter(b *testing.B) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "1KB with frontmatter",
			input: "---\ntitle: Test\nauthor: John\n---\n" + strings.Repeat("# Heading\n\nParagraph text.\n\n", 20),
		},
		{
			name:  "10KB with frontmatter",
			input: "---\ntitle: Test\nauthor: John\n---\n" + strings.Repeat("# Heading\n\nParagraph text.\n\n", 200),
		},
		{
			name:  "100KB with frontmatter",
			input: "---\ntitle: Test\nauthor: John\n---\n" + strings.Repeat("# Heading\n\nParagraph text.\n\n", 2000),
		},
		{
			name:  "no frontmatter",
			input: strings.Repeat("# Heading\n\nParagraph text.\n\n", 200),
		},
	}

	for _, tt := range tests {
		b.Run(tt.name, func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				_ = stripFrontmatter(tt.input)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Fuzz Tests
// ---------------------------------------------------------------------------

func FuzzStripFrontmatter(f *testing.F) {
	// Seed corpus with various input patterns
	f.Add("---\ntitle: Test\n---\nContent")
	f.Add("---\n---\nContent")
	f.Add("---\ntitle: Test\nContent")
	f.Add(strings.Repeat("---\n", 100))
	f.Add(strings.Repeat("a", 10000))
	f.Add("# Heading\nContent")
	f.Add("")

	f.Fuzz(func(t *testing.T, input string) {
		// Should complete in reasonable time (ReDoS protection)
		start := time.Now()
		result := stripFrontmatter(input)
		duration := time.Since(start)

		if duration > 100*time.Millisecond {
			t.Errorf("stripFrontmatter too slow: %v for input length %d", duration, len(input))
		}

		// Non-frontmatter input should never be erased.
		// The regex allows optional horizontal whitespace before ---,
		// so trim that before checking the prefix.
		trimmed := strings.TrimLeft(input, " \t")
		if result == "" && input != "" && !strings.HasPrefix(trimmed, "---\n") {
			t.Errorf("stripFrontmatter erased non-frontmatter input of length %d", len(input))
		}
	})
}
