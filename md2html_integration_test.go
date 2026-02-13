//go:build integration

package md2pdf

// Notes:
// - Tests GoldmarkConverter HTML generation with various markdown features
// - Verifies syntax highlighting with Chroma classes
// - Tests highlight ==text== feature through full preprocessing pipeline
// - Verifies raw HTML sanitization for security (no WithUnsafe)

import (
	"context"
	"strings"
	"testing"

	"github.com/alnah/go-md2pdf/internal/pipeline"
)

// ---------------------------------------------------------------------------
// TestGoldmarkConverter_ToHTML_Integration - Goldmark HTML Conversion
// ---------------------------------------------------------------------------

func TestGoldmarkConverter_ToHTML_Integration(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	t.Run("basic markdown", func(t *testing.T) {
		t.Parallel()
		content := `# Hello

World`
		converter := pipeline.NewGoldmarkConverter()
		got, err := converter.ToHTML(ctx, content)
		if err != nil {
			t.Fatalf("ToHTML() unexpected error: %v", err)
		}

		if !strings.Contains(got, "<h1") {
			t.Errorf("ToHTML() output missing <h1>, got %q", got)
		}
		if !strings.Contains(got, "Hello") {
			t.Errorf("ToHTML() output missing 'Hello', got %q", got)
		}
		if !strings.Contains(got, "<p>World</p>") && !strings.Contains(got, "<p>World") {
			t.Errorf("ToHTML() output missing paragraph with 'World', got %q", got)
		}
	})

	t.Run("unicode content", func(t *testing.T) {
		t.Parallel()

		content := `# Bonjour le monde

Ceci est un test avec des caracteres speciaux.`

		converter := pipeline.NewGoldmarkConverter()
		got, err := converter.ToHTML(ctx, content)
		if err != nil {
			t.Fatalf("ToHTML() unexpected error: %v", err)
		}

		if !strings.Contains(got, "Bonjour") {
			t.Errorf("ToHTML() output missing unicode text, got %q", got)
		}
	})

	t.Run("code block with special chars", func(t *testing.T) {
		t.Parallel()

		content := "# Code Example\n\n```go\nfunc main() {\n\tfmt.Println(\"<hello>\")\n}\n```"

		converter := pipeline.NewGoldmarkConverter()
		got, err := converter.ToHTML(ctx, content)
		if err != nil {
			t.Fatalf("ToHTML() unexpected error: %v", err)
		}

		if !strings.Contains(got, "<code") {
			t.Errorf("ToHTML() output missing <code>, got %q", got)
		}
	})

	t.Run("code block has syntax highlighting classes", func(t *testing.T) {
		t.Parallel()

		content := "```go\nfunc main() {}\n```"

		converter := pipeline.NewGoldmarkConverter()
		got, err := converter.ToHTML(ctx, content)
		if err != nil {
			t.Fatalf("ToHTML() unexpected error: %v", err)
		}

		// Chroma adds class="chroma" to the pre element
		if !strings.Contains(got, `class="chroma"`) {
			t.Errorf("ToHTML() output missing chroma class on pre element, got %q", got)
		}
		// Chroma adds token classes like "kd" (keyword declaration) for syntax tokens
		if !strings.Contains(got, `class="kd"`) {
			t.Errorf("ToHTML() output missing syntax token classes (e.g., kd for keyword), got %q", got)
		}
	})

	t.Run("table markdown", func(t *testing.T) {
		t.Parallel()

		content := `# Table Test

| Name | Age |
|------|-----|
| Alice | 30 |
| Bob | 25 |`

		converter := pipeline.NewGoldmarkConverter()
		got, err := converter.ToHTML(ctx, content)
		if err != nil {
			t.Fatalf("ToHTML() unexpected error: %v", err)
		}

		if !strings.Contains(got, "<table") {
			t.Errorf("ToHTML() output missing <table>, got %q", got)
		}
		if !strings.Contains(got, "Alice") {
			t.Errorf("ToHTML() output missing table data, got %q", got)
		}
	})

	t.Run("nested list", func(t *testing.T) {
		t.Parallel()

		content := `# List Test

- Item 1
  - Subitem 1.1
  - Subitem 1.2
- Item 2`

		converter := pipeline.NewGoldmarkConverter()
		got, err := converter.ToHTML(ctx, content)
		if err != nil {
			t.Fatalf("ToHTML() unexpected error: %v", err)
		}

		if !strings.Contains(got, "<ul") {
			t.Errorf("ToHTML() output missing <ul>, got %q", got)
		}
		if !strings.Contains(got, "<li") {
			t.Errorf("ToHTML() output missing <li>, got %q", got)
		}
	})

	t.Run("whitespace-only content is valid", func(t *testing.T) {
		t.Parallel()

		content := "   \n\t\n   "

		converter := pipeline.NewGoldmarkConverter()
		_, err := converter.ToHTML(ctx, content)
		if err != nil {
			t.Fatalf("ToHTML() unexpected error: %v", err)
		}
	})

	t.Run("highlight placeholders pass through Goldmark unchanged", func(t *testing.T) {
		// The ==highlight== feature uses Unicode Private Use Area placeholders
		// that pass through Goldmark unchanged (no WithUnsafe needed).
		// Post-processing then converts them to <mark> tags.
		t.Parallel()

		// Simulate what convertHighlights() produces (placeholder, not <mark>)
		content := "This is " + pipeline.MarkStartPlaceholder + "highlighted" + pipeline.MarkEndPlaceholder + " text"

		converter := pipeline.NewGoldmarkConverter()
		got, err := converter.ToHTML(ctx, content)
		if err != nil {
			t.Fatalf("ToHTML() unexpected error: %v", err)
		}

		// Placeholders should survive Goldmark
		if !strings.Contains(got, pipeline.MarkStartPlaceholder) || !strings.Contains(got, pipeline.MarkEndPlaceholder) {
			t.Errorf("ToHTML() placeholders not preserved through Goldmark, got %q", got)
		}

		// After post-processing, should become <mark>
		final := pipeline.ConvertMarkPlaceholders(got)
		if !strings.Contains(final, "<mark>highlighted</mark>") {
			t.Errorf("ConvertMarkPlaceholders() did not convert to <mark> tags, got %q", final)
		}
	})

	t.Run("raw HTML is sanitized for security", func(t *testing.T) {
		// Verify that raw HTML does NOT pass through (no WithUnsafe).
		// This is critical for web API security.
		t.Parallel()

		content := "This has <script>alert('xss')</script> injection"

		converter := pipeline.NewGoldmarkConverter()
		got, err := converter.ToHTML(ctx, content)
		if err != nil {
			t.Fatalf("ToHTML() unexpected error: %v", err)
		}

		if strings.Contains(got, "<script>") {
			t.Errorf("ToHTML() did not sanitize raw HTML (should not contain <script>), got %q", got)
		}
	})
}

// ---------------------------------------------------------------------------
// TestHighlightFullPipeline - Highlight Feature End-to-End
// ---------------------------------------------------------------------------

func TestHighlightFullPipeline(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	tests := []struct {
		name     string
		markdown string
		wantMark string
	}{
		{
			name:     "single highlight",
			markdown: "This is ==important== text",
			wantMark: "<mark>important</mark>",
		},
		{
			name:     "multiple highlights",
			markdown: "Both ==one== and ==two== work",
			wantMark: "<mark>one</mark>",
		},
		{
			name:     "highlight in heading",
			markdown: "# Title with ==emphasis==",
			wantMark: "<mark>emphasis</mark>",
		},
		{
			name:     "highlight with unicode",
			markdown: "Text ==日本語== here",
			wantMark: "<mark>日本語</mark>",
		},
	}

	preprocessor := &pipeline.CommonMarkPreprocessor{}
	converter := pipeline.NewGoldmarkConverter()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Step 1: Preprocessing converts ==text== to placeholders
			preprocessed := preprocessor.PreprocessMarkdown(ctx, tt.markdown)

			// Step 2: Goldmark converts to HTML (placeholders pass through)
			html, err := converter.ToHTML(ctx, preprocessed)
			if err != nil {
				t.Fatalf("ToHTML() error: %v", err)
			}

			// Step 3: Post-processing converts placeholders to <mark> tags
			final := pipeline.ConvertMarkPlaceholders(html)

			// Verify the <mark> tag is in the final output
			if !strings.Contains(final, tt.wantMark) {
				t.Errorf("highlight pipeline output missing %q\n"+
					"Input:          %q\n"+
					"Preprocessed:   %q\n"+
					"After Goldmark: %q\n"+
					"Final HTML:     %q",
					tt.wantMark, tt.markdown, preprocessed, html, final)
			}
		})
	}
}
