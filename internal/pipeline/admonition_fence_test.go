package pipeline_test

import (
	"testing"
)

func TestAdmonitionFenceParser_AllTypes(t *testing.T) {
	tests := []struct {
		name      string
		label     string
		wantKind  string
		wantTitle string
	}{
		{"note", "note", "note", "Note"},
		{"tip", "tip", "tip", "Tip"},
		{"important", "important", "important", "Important"},
		{"warning", "warning", "warning", "Warning"},
		{"caution", "caution", "caution", "Caution"},
		{"uppercase", "WARNING", "warning", "Warning"},
		{"alias info", "info", "note", "Note"},
		{"alias success", "success", "tip", "Tip"},
		{"alias danger", "danger", "caution", "Caution"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			source := "::: " + tt.label + "\nBody.\n:::"
			got := renderTestMarkdown(t, newAdmonitionTestMarkdown(), source)
			want := expectedAdmonitionHTML(tt.wantKind, tt.wantTitle, "<p>Body.</p>\n")
			if got != want {
				t.Fatalf("render = %q, want %q", got, want)
			}
		})
	}
}

func TestAdmonitionFenceParser_CustomTitle(t *testing.T) {
	tests := []struct {
		name      string
		kind      string
		source    string
		wantTitle string
	}{
		{
			name:      "custom title",
			kind:      "tip",
			source:    "::: tip Custom title\nBody.\n:::",
			wantTitle: "Custom title",
		},
		{
			name:      "padded title",
			kind:      "note",
			source:    "::: note   Padded title   \nBody.\n:::",
			wantTitle: "Padded title",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := renderTestMarkdown(t, newAdmonitionTestMarkdown(), tt.source)
			want := expectedAdmonitionHTML(tt.kind, tt.wantTitle, "<p>Body.</p>\n")
			if got != want {
				t.Fatalf("render = %q, want %q", got, want)
			}
		})
	}
}

func TestAdmonitionFenceParser_Nested(t *testing.T) {
	body := "<p>Nested.</p>\n"
	inner := expectedAdmonitionHTML("tip", "Tip", body)

	tests := []struct {
		name   string
		source string
	}{
		{
			name:   "longer outer fence",
			source: ":::: note\n::: tip\nNested.\n:::\n::::",
		},
		{
			name:   "same fence length",
			source: "::: note\n::: tip\nNested.\n:::\n:::",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := renderTestMarkdown(t, newAdmonitionTestMarkdown(), tt.source)
			want := expectedAdmonitionHTML("note", "Note", inner)
			if got != want {
				t.Fatalf("render = %q, want %q", got, want)
			}
		})
	}
}

func TestAdmonitionFenceParser_UnknownTypePassthrough(t *testing.T) {
	source := "::: bogus\nBody.\n:::"

	got := renderTestMarkdown(t, newAdmonitionTestMarkdown(), source)
	want := renderTestMarkdown(t, newPlainTestMarkdown(), source)
	if got != want {
		t.Fatalf("render = %q, want passthrough %q", got, want)
	}
}

func TestAdmonitionFenceParser_MalformedPassthrough(t *testing.T) {
	tests := []struct {
		name   string
		source string
	}{
		{"bare colons", ":::"},
		{"two colons", "::"},
		{"no space", ":::note\nBody."},
		{"trailing space only", "::: \nBody."},
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

func TestAdmonitionFenceParser_Unclosed(t *testing.T) {
	source := "::: note\nBody."

	got := renderTestMarkdown(t, newAdmonitionTestMarkdown(), source)
	want := expectedAdmonitionHTML("note", "Note", "<p>Body.</p>\n")
	if got != want {
		t.Fatalf("render = %q, want %q", got, want)
	}
}

func TestAdmonitionFenceParser_CodeBlockUntouched(t *testing.T) {
	tests := []struct {
		name   string
		source string
	}{
		{"fenced code", "```\n::: note\nBody.\n:::\n```"},
		{"indented code", "    ::: note\n    Body.\n    :::"},
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

func TestAdmonitionFenceParser_Indentation(t *testing.T) {
	indents := []string{"", " ", "  ", "   "}

	for _, indent := range indents {
		t.Run("indent "+indent, func(t *testing.T) {
			source := indent + "::: note\n" + indent + "Body.\n" + indent + ":::"
			got := renderTestMarkdown(t, newAdmonitionTestMarkdown(), source)
			want := expectedAdmonitionHTML("note", "Note", "<p>Body.</p>\n")
			if got != want {
				t.Fatalf("render = %q, want %q", got, want)
			}
		})
	}

	t.Run("four spaces is indented code", func(t *testing.T) {
		source := "    ::: note\n    Body.\n    :::"
		got := renderTestMarkdown(t, newAdmonitionTestMarkdown(), source)
		want := renderTestMarkdown(t, newPlainTestMarkdown(), source)
		if got != want {
			t.Fatalf("render = %q, want passthrough %q", got, want)
		}
	})
}

func TestAdmonitionFenceParser_DoesNotInterruptParagraph(t *testing.T) {
	source := "text\n::: note\nBody.\n:::"

	got := renderTestMarkdown(t, newAdmonitionTestMarkdown(), source)
	want := renderTestMarkdown(t, newPlainTestMarkdown(), source)
	if got != want {
		t.Fatalf("render = %q, want passthrough %q", got, want)
	}
}
