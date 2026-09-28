package pipeline_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer/html"

	"github.com/alnah/picoloom/v2/internal/pipeline"
)

// newAdmonitionTestMarkdown builds a Goldmark instance with the admonition
// extension. Replaced by NewGoldmarkConverter once it is wired in T4.
func newAdmonitionTestMarkdown() goldmark.Markdown {
	return goldmark.New(
		goldmark.WithExtensions(extension.GFM, extension.Footnote, pipeline.AdmonitionExtension),
		goldmark.WithParserOptions(parser.WithAutoHeadingID()),
		goldmark.WithRendererOptions(html.WithXHTML(), html.WithHardWraps()),
	)
}

// newPlainTestMarkdown is the reference instance without the admonition
// extension, used to assert passthrough output.
func newPlainTestMarkdown() goldmark.Markdown {
	return goldmark.New(
		goldmark.WithExtensions(extension.GFM, extension.Footnote),
		goldmark.WithParserOptions(parser.WithAutoHeadingID()),
		goldmark.WithRendererOptions(html.WithXHTML(), html.WithHardWraps()),
	)
}

func renderTestMarkdown(t *testing.T, markdown goldmark.Markdown, source string) string {
	t.Helper()

	var buf bytes.Buffer
	if err := markdown.Convert([]byte(source), &buf); err != nil {
		t.Fatalf("convert markdown: %v", err)
	}
	return buf.String()
}

func expectedAdmonitionHTML(kind, title, bodyHTML string) string {
	return "<div class=\"admonition admonition-" + kind + "\" role=\"note\">\n" +
		"<p class=\"admonition-title\">" + title + "</p>\n" +
		bodyHTML +
		"</div>\n"
}

func TestAdmonitionQuoteTransformer_DetectsAllTypes(t *testing.T) {
	tests := []struct {
		name      string
		marker    string
		wantKind  string
		wantTitle string
	}{
		{"note", "NOTE", "note", "Note"},
		{"tip", "TIP", "tip", "Tip"},
		{"important", "IMPORTANT", "important", "Important"},
		{"warning", "WARNING", "warning", "Warning"},
		{"caution", "CAUTION", "caution", "Caution"},
		{"alias info", "INFO", "note", "Note"},
		{"alias success", "SUCCESS", "tip", "Tip"},
		{"alias danger", "DANGER", "caution", "Caution"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			source := "> [!" + tt.marker + "]\n> Body."
			got := renderTestMarkdown(t, newAdmonitionTestMarkdown(), source)
			want := expectedAdmonitionHTML(tt.wantKind, tt.wantTitle, "<p>Body.</p>\n")
			if got != want {
				t.Fatalf("render = %q, want %q", got, want)
			}
		})
	}
}

func TestAdmonitionQuoteTransformer_CustomTitle(t *testing.T) {
	tests := []struct {
		name      string
		kind      string
		source    string
		wantTitle string
		wantBody  string
	}{
		{
			name:      "custom title",
			kind:      "tip",
			source:    "> [!TIP] Custom title\n> Body.",
			wantTitle: "Custom title",
			wantBody:  "<p>Body.</p>\n",
		},
		{
			name:      "padded title",
			kind:      "note",
			source:    "> [!NOTE]   Padded title   \n> Body.",
			wantTitle: "Padded title",
			wantBody:  "<p>Body.</p>\n",
		},
		{
			name:      "title limited to first line",
			kind:      "note",
			source:    "> [!NOTE] Header\n> continuation",
			wantTitle: "Header",
			wantBody:  "<p>continuation</p>\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := renderTestMarkdown(t, newAdmonitionTestMarkdown(), tt.source)
			want := expectedAdmonitionHTML(tt.kind, tt.wantTitle, tt.wantBody)
			if got != want {
				t.Fatalf("render = %q, want %q", got, want)
			}
		})
	}
}

func TestAdmonitionQuoteTransformer_PreservesUnknownType(t *testing.T) {
	source := "> [!BOGUS]\n> Body."

	got := renderTestMarkdown(t, newAdmonitionTestMarkdown(), source)
	want := renderTestMarkdown(t, newPlainTestMarkdown(), source)
	if got != want {
		t.Fatalf("render = %q, want passthrough %q", got, want)
	}
}

func TestAdmonitionQuoteTransformer_PreservesRegularBlockquote(t *testing.T) {
	source := "> Quote.\n> More."

	got := renderTestMarkdown(t, newAdmonitionTestMarkdown(), source)
	want := renderTestMarkdown(t, newPlainTestMarkdown(), source)
	if got != want {
		t.Fatalf("render = %q, want passthrough %q", got, want)
	}
}

func TestAdmonitionQuoteTransformer_NestedQuotes(t *testing.T) {
	tests := []struct {
		name   string
		source string
		want   string
	}{
		{
			name:   "quoted admonition",
			source: "> > [!NOTE]\n> > Inner body.",
			want: "<blockquote>\n" +
				expectedAdmonitionHTML("note", "Note", "<p>Inner body.</p>\n") +
				"</blockquote>\n",
		},
		{
			name:   "admonition containing nested admonition",
			source: "> [!NOTE]\n> > [!TIP]\n> > Nested.",
			want: expectedAdmonitionHTML("note", "Note",
				expectedAdmonitionHTML("tip", "Tip", "<p>Nested.</p>\n")),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := renderTestMarkdown(t, newAdmonitionTestMarkdown(), tt.source)
			if got != tt.want {
				t.Fatalf("render = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestAdmonitionQuoteTransformer_MarkerInCode(t *testing.T) {
	tests := []struct {
		name   string
		source string
	}{
		{"fenced code", "```\n> [!NOTE]\n> Body.\n```"},
		{"indented code", "    > [!NOTE]\n    > Body."},
		{"inline code", "> `[!NOTE]`\n> Body."},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := renderTestMarkdown(t, newAdmonitionTestMarkdown(), tt.source)
			want := renderTestMarkdown(t, newPlainTestMarkdown(), tt.source)
			if got != want {
				t.Fatalf("render = %q, want passthrough %q", got, want)
			}
		})
	}
}

func TestAdmonitionQuoteTransformer_RichBody(t *testing.T) {
	source := "> [!NOTE]\n" +
		"> - item one\n" +
		"> - item two\n" +
		">\n" +
		"> | A | B |\n" +
		"> |---|---|\n" +
		"> | 1 | 2 |\n" +
		">\n" +
		"> ```go\n" +
		"> code\n" +
		"> ```\n" +
		">\n" +
		"> Link [example](https://example.com) and footnote[^1].\n" +
		">\n" +
		"> [^1]: Footnote text."

	got := renderTestMarkdown(t, newAdmonitionTestMarkdown(), source)

	wants := []string{
		"<p class=\"admonition-title\">Note</p>",
		"<ul>",
		"<li>item one</li>",
		"<li>item two</li>",
		"<table>",
		"<code class=\"language-go\">code",
		"<a href=\"https://example.com\">example</a>",
		"fnref:1",
		"class=\"footnotes\"",
	}
	for _, want := range wants {
		if !strings.Contains(got, want) {
			t.Fatalf("render = %q, want it to contain %q", got, want)
		}
	}
	if strings.Contains(got, "[!NOTE]") {
		t.Fatalf("render = %q, marker must be removed", got)
	}
}

func TestAdmonitionQuoteTransformer_EscapesTitle(t *testing.T) {
	source := "> [!NOTE] <script>alert(\"x\")</script>\n> Body."

	got := renderTestMarkdown(t, newAdmonitionTestMarkdown(), source)

	if strings.Contains(got, "<script>") {
		t.Fatalf("render = %q, title must be escaped", got)
	}
	want := "<p class=\"admonition-title\">&lt;script&gt;alert(&quot;x&quot;)&lt;/script&gt;</p>"
	if !strings.Contains(got, want) {
		t.Fatalf("render = %q, want it to contain %q", got, want)
	}
}
