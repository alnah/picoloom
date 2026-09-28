package pipeline_test

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/renderer/html"

	"github.com/alnah/picoloom/v2/internal/pipeline"
)

// renderAdmonitionNode renders a hand-built Admonition node through the
// public extension renderer. Used before the block parsers are wired.
func renderAdmonitionNode(t *testing.T, kind pipeline.AdmonitionKind, title, body string) string {
	t.Helper()

	markdown := goldmark.New(
		goldmark.WithExtensions(pipeline.AdmonitionExtension),
		goldmark.WithRendererOptions(html.WithXHTML(), html.WithHardWraps()),
	)
	doc := ast.NewDocument()
	node := pipeline.NewAdmonition(kind, title)
	if body != "" {
		paragraph := ast.NewParagraph()
		paragraph.AppendChild(paragraph, ast.NewString([]byte(body)))
		node.AppendChild(node, paragraph)
	}
	doc.AppendChild(doc, node)

	var buf bytes.Buffer
	if err := markdown.Renderer().Render(&buf, nil, doc); err != nil {
		t.Fatalf("render admonition: %v", err)
	}
	return buf.String()
}

func TestAdmonitionKind_ClassName_DefaultTitle(t *testing.T) {
	tests := []struct {
		kind       pipeline.AdmonitionKind
		wantString string
		wantClass  string
		wantTitle  string
	}{
		{pipeline.AdmonitionNote, "note", "admonition-note", "Note"},
		{pipeline.AdmonitionTip, "tip", "admonition-tip", "Tip"},
		{pipeline.AdmonitionImportant, "important", "admonition-important", "Important"},
		{pipeline.AdmonitionWarning, "warning", "admonition-warning", "Warning"},
		{pipeline.AdmonitionCaution, "caution", "admonition-caution", "Caution"},
	}

	for _, tt := range tests {
		t.Run(tt.wantString, func(t *testing.T) {
			if got := tt.kind.String(); got != tt.wantString {
				t.Errorf("String() = %q, want %q", got, tt.wantString)
			}
			if got := tt.kind.ClassName(); got != tt.wantClass {
				t.Errorf("ClassName() = %q, want %q", got, tt.wantClass)
			}
			if got := tt.kind.DefaultTitle(); got != tt.wantTitle {
				t.Errorf("DefaultTitle() = %q, want %q", got, tt.wantTitle)
			}
		})
	}
}

func TestAdmonitionKind_InvalidFallback(t *testing.T) {
	for _, kind := range []pipeline.AdmonitionKind{-1, 5, 99} {
		if got := kind.String(); got != "" {
			t.Errorf("String() = %q, want empty", got)
		}
		if got := kind.ClassName(); got != "" {
			t.Errorf("ClassName() = %q, want empty", got)
		}
		if got := kind.DefaultTitle(); got != "" {
			t.Errorf("DefaultTitle() = %q, want empty", got)
		}
	}
}

func TestAdmonition_Kind(t *testing.T) {
	node := pipeline.NewAdmonition(pipeline.AdmonitionTip, "Custom")
	if got := node.Kind(); got != pipeline.KindAdmonition {
		t.Fatalf("Kind() = %v, want %v", got, pipeline.KindAdmonition)
	}
}

func TestAdmonition_Dump(t *testing.T) {
	node := pipeline.NewAdmonition(pipeline.AdmonitionTip, "Custom")

	readEnd, writeEnd, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	oldStdout := os.Stdout
	os.Stdout = writeEnd
	t.Cleanup(func() { os.Stdout = oldStdout })

	node.Dump(nil, 0)
	if err := writeEnd.Close(); err != nil {
		t.Fatalf("close pipe: %v", err)
	}
	os.Stdout = oldStdout

	out, err := io.ReadAll(readEnd)
	if err != nil {
		t.Fatalf("read dump: %v", err)
	}
	_ = readEnd.Close()

	dump := string(out)
	for _, want := range []string{"Admonition", "Custom"} {
		if !strings.Contains(dump, want) {
			t.Fatalf("dump = %q, want it to contain %q", dump, want)
		}
	}
}

func TestNewAdmonitionHTMLRenderer_Options(t *testing.T) {
	nodeRenderer := pipeline.NewAdmonitionHTMLRenderer(html.WithXHTML())
	if nodeRenderer == nil {
		t.Fatal("NewAdmonitionHTMLRenderer() = nil")
	}
}

func TestAdmonitionRenderer_DefaultTitles(t *testing.T) {
	tests := []struct {
		kind      pipeline.AdmonitionKind
		wantTitle string
	}{
		{pipeline.AdmonitionNote, "Note"},
		{pipeline.AdmonitionTip, "Tip"},
		{pipeline.AdmonitionImportant, "Important"},
		{pipeline.AdmonitionWarning, "Warning"},
		{pipeline.AdmonitionCaution, "Caution"},
	}

	for _, tt := range tests {
		t.Run(tt.wantTitle, func(t *testing.T) {
			got := renderAdmonitionNode(t, tt.kind, "", "")
			want := fmt.Sprintf(
				"<div class=\"admonition admonition-%s\" role=\"note\">\n"+
					"<p class=\"admonition-title\">%s</p>\n"+
					"</div>\n",
				tt.kind.String(), tt.wantTitle,
			)
			if got != want {
				t.Fatalf("render = %q, want %q", got, want)
			}
		})
	}
}

func TestAdmonitionRenderer_EscapesTitle(t *testing.T) {
	got := renderAdmonitionNode(t, pipeline.AdmonitionNote, `<script>alert("x") & 1</script>`, "")
	want := "<div class=\"admonition admonition-note\" role=\"note\">\n" +
		"<p class=\"admonition-title\">&lt;script&gt;alert(&quot;x&quot;) &amp; 1&lt;/script&gt;</p>\n" +
		"</div>\n"
	if got != want {
		t.Fatalf("render = %q, want %q", got, want)
	}
}

func convertAdmonitionHTML(t *testing.T, source string) string {
	t.Helper()

	html, err := pipeline.NewGoldmarkConverter().ToHTML(context.Background(), source)
	if err != nil {
		t.Fatalf("ToHTML(%q) error: %v", source, err)
	}
	return html
}

func TestAdmonitionEndToEnd_Syntaxes(t *testing.T) {
	tests := []struct {
		name   string
		source string
		want   string
	}{
		{
			name:   "quote syntax",
			source: "> [!NOTE]\n> Body.",
			want: "<div class=\"admonition admonition-note\" role=\"note\">\n" +
				"<p class=\"admonition-title\">Note</p>\n" +
				"<p>Body.</p>\n" +
				"</div>\n",
		},
		{
			name:   "fence syntax",
			source: "::: tip Custom\nBody.\n:::",
			want: "<div class=\"admonition admonition-tip\" role=\"note\">\n" +
				"<p class=\"admonition-title\">Custom</p>\n" +
				"<p>Body.</p>\n" +
				"</div>\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := convertAdmonitionHTML(t, tt.source)
			if !strings.Contains(got, tt.want) {
				t.Fatalf("ToHTML = %q, want it to contain %q", got, tt.want)
			}
		})
	}
}

func TestAdmonitionEndToEnd_Nested(t *testing.T) {
	source := ":::: note\n> [!TIP]\n> Nested.\n::::"
	want := "<div class=\"admonition admonition-note\" role=\"note\">\n" +
		"<p class=\"admonition-title\">Note</p>\n" +
		"<div class=\"admonition admonition-tip\" role=\"note\">\n" +
		"<p class=\"admonition-title\">Tip</p>\n" +
		"<p>Nested.</p>\n" +
		"</div>\n" +
		"</div>\n"

	got := convertAdmonitionHTML(t, source)
	if !strings.Contains(got, want) {
		t.Fatalf("ToHTML = %q, want it to contain %q", got, want)
	}
}

func TestAdmonitionEndToEnd_EscapesTitle(t *testing.T) {
	source := "> [!NOTE] <script>alert(\"x\")</script>\n> Body."

	got := convertAdmonitionHTML(t, source)
	if strings.Contains(got, "<script>") {
		t.Fatalf("ToHTML = %q, title must be escaped", got)
	}
	want := `<p class="admonition-title">&lt;script&gt;alert(&quot;x&quot;)&lt;/script&gt;</p>`
	if !strings.Contains(got, want) {
		t.Fatalf("ToHTML = %q, want it to contain %q", got, want)
	}
}
