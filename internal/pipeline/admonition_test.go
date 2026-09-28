package pipeline

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/renderer/html"
)

// renderAdmonitionNode renders a hand-built Admonition node through the
// extension renderer. Used before the block parsers are wired.
func renderAdmonitionNode(t *testing.T, kind AdmonitionKind, title, body string) string {
	t.Helper()

	markdown := goldmark.New(
		goldmark.WithExtensions(AdmonitionExtension),
		goldmark.WithRendererOptions(html.WithXHTML(), html.WithHardWraps()),
	)
	doc := ast.NewDocument()
	node := NewAdmonition(kind, title)
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

func TestParseAdmonitionKind_Canonical(t *testing.T) {
	tests := []struct {
		label string
		want  AdmonitionKind
	}{
		{"note", AdmonitionNote},
		{"NOTE", AdmonitionNote},
		{"Tip", AdmonitionTip},
		{"important", AdmonitionImportant},
		{"WaRnInG", AdmonitionWarning},
		{"caution", AdmonitionCaution},
	}

	for _, tt := range tests {
		t.Run(tt.label, func(t *testing.T) {
			got, ok := parseAdmonitionKind(tt.label)
			if !ok {
				t.Fatalf("parseAdmonitionKind(%q) ok = false, want true", tt.label)
			}
			if got != tt.want {
				t.Fatalf("parseAdmonitionKind(%q) = %v, want %v", tt.label, got, tt.want)
			}
		})
	}
}

func TestParseAdmonitionKind_Aliases(t *testing.T) {
	tests := []struct {
		label string
		want  AdmonitionKind
	}{
		{"info", AdmonitionNote},
		{"INFO", AdmonitionNote},
		{"success", AdmonitionTip},
		{"dAngEr", AdmonitionCaution},
	}

	for _, tt := range tests {
		t.Run(tt.label, func(t *testing.T) {
			got, ok := parseAdmonitionKind(tt.label)
			if !ok {
				t.Fatalf("parseAdmonitionKind(%q) ok = false, want true", tt.label)
			}
			if got != tt.want {
				t.Fatalf("parseAdmonitionKind(%q) = %v, want %v", tt.label, got, tt.want)
			}
		})
	}
}

func TestParseAdmonitionKind_Unknown(t *testing.T) {
	labels := []string{
		"", "bogus", "notes", "note!", "note extra",
		"not", "noted", "informational", "successful", "dangerous",
		" info", "note ", "[]", "[!NOTE]", "noté",
	}
	for _, label := range labels {
		if _, ok := parseAdmonitionKind(label); ok {
			t.Fatalf("parseAdmonitionKind(%q) ok = true, want false", label)
		}
	}
}

func TestAdmonitionKind_ClassName_DefaultTitle(t *testing.T) {
	tests := []struct {
		kind       AdmonitionKind
		wantString string
		wantClass  string
		wantTitle  string
	}{
		{AdmonitionNote, "note", "admonition-note", "Note"},
		{AdmonitionTip, "tip", "admonition-tip", "Tip"},
		{AdmonitionImportant, "important", "admonition-important", "Important"},
		{AdmonitionWarning, "warning", "admonition-warning", "Warning"},
		{AdmonitionCaution, "caution", "admonition-caution", "Caution"},
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
	for _, kind := range []AdmonitionKind{-1, 5, 99} {
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
	node := NewAdmonition(AdmonitionTip, "Custom")
	if got := node.Kind(); got != KindAdmonition {
		t.Fatalf("Kind() = %v, want %v", got, KindAdmonition)
	}
}

func TestAdmonition_Dump(t *testing.T) {
	node := NewAdmonition(AdmonitionTip, "Custom")

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

func TestAdmonitionRenderer_DefaultTitles(t *testing.T) {
	tests := []struct {
		kind      AdmonitionKind
		wantTitle string
	}{
		{AdmonitionNote, "Note"},
		{AdmonitionTip, "Tip"},
		{AdmonitionImportant, "Important"},
		{AdmonitionWarning, "Warning"},
		{AdmonitionCaution, "Caution"},
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
	got := renderAdmonitionNode(t, AdmonitionNote, `<script>alert("x") & 1</script>`, "")
	want := "<div class=\"admonition admonition-note\" role=\"note\">\n" +
		"<p class=\"admonition-title\">&lt;script&gt;alert(&quot;x&quot;) &amp; 1&lt;/script&gt;</p>\n" +
		"</div>\n"
	if got != want {
		t.Fatalf("render = %q, want %q", got, want)
	}
}
