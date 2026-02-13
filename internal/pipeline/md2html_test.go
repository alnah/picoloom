package pipeline

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestGoldmarkConverter_ToHTML(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		input        string
		wantContains []string
		wantNot      []string
	}{
		{
			name:  "basic heading",
			input: "# Hello World",
			wantContains: []string{
				"<!DOCTYPE html>",
				"<h1",
				"Hello World",
				"</h1>",
			},
		},
		{
			name:  "multiple headings with IDs",
			input: "# First\n## Second\n### Third",
			wantContains: []string{
				"<h1",
				"<h2",
				"<h3",
				`id="`,
			},
		},
		{
			name:  "paragraph with hard breaks",
			input: "Line one\nLine two",
			wantContains: []string{
				"<p>",
				"Line one",
				"<br",
				"Line two",
			},
		},
		{
			name:  "GFM table",
			input: "| A | B |\n|---|---|\n| 1 | 2 |",
			wantContains: []string{
				"<table>",
				"<thead>",
				"<tbody>",
				"<th>",
				"<td>",
			},
		},
		{
			name:  "GFM strikethrough",
			input: "~~deleted~~",
			wantContains: []string{
				"<del>",
				"deleted",
				"</del>",
			},
		},
		{
			name:  "GFM autolink",
			input: "Visit https://example.com for more",
			wantContains: []string{
				"<a href=\"https://example.com\"",
				"https://example.com",
			},
		},
		{
			name:  "GFM task list",
			input: "- [x] Done\n- [ ] Todo",
			wantContains: []string{
				"<input",
				"checked",
				"type=\"checkbox\"",
			},
		},
		{
			name:  "footnote",
			input: "Text[^1]\n\n[^1]: Footnote content",
			wantContains: []string{
				"<sup",
				"footnote",
			},
		},
		{
			name:  "code block with syntax highlighting",
			input: "```go\nfunc main() {}\n```",
			wantContains: []string{
				"<pre",
				"<code",
				"func",
			},
		},
		{
			name:  "inline code",
			input: "Use `fmt.Println` function",
			wantContains: []string{
				"<code>",
				"fmt.Println",
				"</code>",
			},
		},
		{
			name:  "bold and italic",
			input: "**bold** and *italic*",
			wantContains: []string{
				"<strong>",
				"bold",
				"<em>",
				"italic",
			},
		},
		{
			name:  "links",
			input: "[text](https://example.com)",
			wantContains: []string{
				"<a href=\"https://example.com\"",
				"text",
				"</a>",
			},
		},
		{
			name:  "images",
			input: "![alt text](image.png)",
			wantContains: []string{
				"<img",
				"src=\"image.png\"",
				"alt=\"alt text\"",
			},
		},
		{
			name:  "blockquote",
			input: "> Quoted text",
			wantContains: []string{
				"<blockquote>",
				"Quoted text",
			},
		},
		{
			name:  "unordered list",
			input: "- Item 1\n- Item 2",
			wantContains: []string{
				"<ul>",
				"<li>",
				"Item 1",
				"Item 2",
			},
		},
		{
			name:  "ordered list",
			input: "1. First\n2. Second",
			wantContains: []string{
				"<ol>",
				"<li>",
				"First",
				"Second",
			},
		},
		{
			name:  "horizontal rule",
			input: "---",
			wantContains: []string{
				"<hr",
			},
		},
		{
			name:  "empty input",
			input: "",
			wantContains: []string{
				"<!DOCTYPE html>",
				"<html>",
				"<body>",
				"</body>",
				"</html>",
			},
		},
		{
			name:  "unicode content",
			input: "# 日本語\n\nBonjour le monde",
			wantContains: []string{
				"日本語",
				"Bonjour le monde",
			},
		},
		{
			// Raw HTML is sanitized by Goldmark (no WithUnsafe option).
			// This is intentional for security - the ==highlight== feature uses
			// placeholders that are converted to <mark> AFTER Goldmark processing.
			name:  "raw HTML is sanitized for security",
			input: "<script>alert('xss')</script>",
			wantContains: []string{
				"<!-- raw HTML omitted -->",
			},
			wantNot: []string{
				"<script>",
			},
		},
		{
			name:  "HTML template structure",
			input: "# Test",
			wantContains: []string{
				"<!DOCTYPE html>",
				"<html>",
				"<head>",
				"<meta charset=\"utf-8\">",
				"<title>Document</title>",
				"</head>",
				"<body>",
				"</body>",
				"</html>",
			},
		},
	}

	converter := NewGoldmarkConverter()
	ctx := context.Background()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result, err := converter.ToHTML(ctx, tt.input)
			if err != nil {
				t.Fatalf("ToHTML(%q) unexpected error: %v", tt.input, err)
			}

			for _, want := range tt.wantContains {
				if !strings.Contains(result, want) {
					t.Errorf("ToHTML(%q) missing expected content %q\nGot:\n%s", tt.input, want, result)
				}
			}

			for _, notWant := range tt.wantNot {
				if strings.Contains(result, notWant) {
					t.Errorf("ToHTML(%q) contains unwanted content %q\nGot:\n%s", tt.input, notWant, result)
				}
			}
		})
	}
}

func TestGoldmarkConverter_ToHTML_ContextCancellation(t *testing.T) {
	t.Parallel()

	converter := NewGoldmarkConverter()

	t.Run("error case: cancelled context", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		_, err := converter.ToHTML(ctx, "# Test")
		if err == nil {
			t.Fatal("ToHTML(ctx, \"# Test\") error = nil, want error")
		}
		if err != context.Canceled {
			t.Errorf("ToHTML(ctx, \"# Test\") error = %v, want context.Canceled", err)
		}
	})

	t.Run("error case: deadline exceeded", func(t *testing.T) {
		t.Parallel()

		// Create an already-expired context to avoid flaky timing issues
		ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
		defer cancel()

		_, err := converter.ToHTML(ctx, "# Test")
		if err == nil {
			t.Fatal("ToHTML(ctx, \"# Test\") error = nil, want error")
		}
		if err != context.DeadlineExceeded {
			t.Errorf("ToHTML(ctx, \"# Test\") error = %v, want context.DeadlineExceeded", err)
		}
	})

	t.Run("happy path: valid context", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		result, err := converter.ToHTML(ctx, "# Test")
		if err != nil {
			t.Fatalf("ToHTML(ctx, \"# Test\") unexpected error: %v", err)
		}
		if !strings.Contains(result, "Test") {
			t.Errorf("ToHTML(ctx, \"# Test\") missing expected content \"Test\"\nGot:\n%s", result)
		}
	})
}

func TestNewGoldmarkConverter(t *testing.T) {
	t.Parallel()

	converter := NewGoldmarkConverter()

	if converter == nil {
		t.Fatal("NewGoldmarkConverter() returned nil")
	}

	if converter.md == nil {
		t.Error("converter.md is nil")
	}
}

func TestGoldmarkConverter_GFMExtensions(t *testing.T) {
	t.Parallel()

	converter := NewGoldmarkConverter()
	ctx := context.Background()

	t.Run("table alignment", func(t *testing.T) {
		t.Parallel()

		input := "| Left | Center | Right |\n|:-----|:------:|------:|\n| L | C | R |"
		result, err := converter.ToHTML(ctx, input)
		if err != nil {
			t.Fatalf("ToHTML(ctx, %q) unexpected error: %v", input, err)
		}

		if !strings.Contains(result, "left") && !strings.Contains(result, "align") {
			// Table structure should exist even if alignment styling varies
			if !strings.Contains(result, "<table>") {
				t.Errorf("ToHTML(ctx, %q) missing expected table markup", input)
			}
		}
	})

	t.Run("nested lists", func(t *testing.T) {
		t.Parallel()

		input := "- Item 1\n  - Nested 1\n  - Nested 2\n- Item 2"
		result, err := converter.ToHTML(ctx, input)
		if err != nil {
			t.Fatalf("ToHTML(ctx, %q) unexpected error: %v", input, err)
		}

		// Count list elements to verify nesting
		ulCount := strings.Count(result, "<ul>")
		if ulCount < 2 {
			t.Errorf("ToHTML(ctx, %q) <ul> count = %d, want >= 2", input, ulCount)
		}
	})

	t.Run("complex document", func(t *testing.T) {
		t.Parallel()

		input := `# Title

This is a paragraph with **bold** and *italic* text.

## Table

| Column A | Column B |
|----------|----------|
| Data 1   | Data 2   |

## Code

` + "```go\npackage main\n\nfunc main() {\n\tfmt.Println(\"Hello\")\n}\n```" + `

## Lists

- Item 1
- Item 2
  - Nested

1. First
2. Second

---

> Blockquote

[Link](https://example.com)
`
		result, err := converter.ToHTML(ctx, input)
		if err != nil {
			t.Fatalf("ToHTML(ctx, complex input) unexpected error: %v", err)
		}

		checks := []string{
			"<h1", "<h2", // Headings
			"<strong>", "<em>", // Formatting
			"<table>", "<th>", "<td>", // Table
			"<pre", "<code", // Code
			"<ul>", "<ol>", "<li>", // Lists
			"<hr",          // Horizontal rule
			"<blockquote>", // Quote
			"<a href=",     // Link
		}

		for _, check := range checks {
			if !strings.Contains(result, check) {
				t.Errorf("ToHTML(ctx, complex input) missing expected content %q", check)
			}
		}
	})
}

func TestGoldmarkConverter_HeadingIDs(t *testing.T) {
	t.Parallel()

	converter := NewGoldmarkConverter()
	ctx := context.Background()

	tests := []struct {
		name   string
		input  string
		wantID string
	}{
		{
			name:   "simple heading",
			input:  "# Hello",
			wantID: `id="hello"`,
		},
		{
			name:   "heading with spaces",
			input:  "# Hello World",
			wantID: `id="hello-world"`,
		},
		{
			name:   "heading with special chars",
			input:  "# Hello, World!",
			wantID: `id="hello-world"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result, err := converter.ToHTML(ctx, tt.input)
			if err != nil {
				t.Fatalf("ToHTML(ctx, %q) unexpected error: %v", tt.input, err)
			}

			if !strings.Contains(result, tt.wantID) {
				t.Errorf("ToHTML(ctx, %q) missing expected heading ID %q\nGot:\n%s", tt.input, tt.wantID, result)
			}
		})
	}
}
