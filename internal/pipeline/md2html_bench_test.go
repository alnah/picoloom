package pipeline

import (
	"context"
	"fmt"
	"strings"
	"testing"
)

// BenchmarkGoldmarkToHTML benchmarks markdown to HTML conversion.
// This is the core conversion step in the pipeline.
func BenchmarkGoldmarkToHTML(b *testing.B) {
	converter := NewGoldmarkConverter()
	ctx := context.Background()

	inputs := []struct {
		name    string
		content string
	}{
		{"minimal", "# Hello\n\nWorld"},
		{"paragraph", strings.Repeat("This is a paragraph with some text.\n\n", 10)},
		{"headings", generateHeadingsMarkdown(20)},
		{"code_blocks", generateCodeBlocksMarkdown(10)},
		{"tables", generateTablesMarkdown(5)},
		{"mixed_small", generateMixedMarkdown(10)},
		{"mixed_medium", generateMixedMarkdown(50)},
		{"mixed_large", generateMixedMarkdown(200)},
	}

	for _, input := range inputs {
		b.Run(input.name, func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				result, err := converter.ToHTML(ctx, input.content)
				if err != nil {
					b.Fatal(err)
				}
				_ = result
			}
		})
	}
}

// BenchmarkGoldmarkToHTMLBySize benchmarks conversion scaling with input size.
func BenchmarkGoldmarkToHTMLBySize(b *testing.B) {
	converter := NewGoldmarkConverter()
	ctx := context.Background()

	sizes := []int{1, 10, 50, 100, 500, 1000}

	for _, size := range sizes {
		content := generateMixedMarkdown(size)
		b.Run(fmt.Sprintf("sections_%d", size), func(b *testing.B) {
			b.ReportAllocs()
			b.SetBytes(int64(len(content)))
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				result, err := converter.ToHTML(ctx, content)
				if err != nil {
					b.Fatal(err)
				}
				_ = result
			}
		})
	}
}

// BenchmarkGoldmarkToHTMLParallel benchmarks concurrent HTML conversion.
func BenchmarkGoldmarkToHTMLParallel(b *testing.B) {
	converter := NewGoldmarkConverter()
	ctx := context.Background()
	content := generateMixedMarkdown(20)

	b.ReportAllocs()
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			result, err := converter.ToHTML(ctx, content)
			if err != nil {
				b.Fatal(err)
			}
			_ = result
		}
	})
}

// BenchmarkGoldmarkSyntaxHighlighting benchmarks code block highlighting.
// Tests chroma syntax highlighting performance.
func BenchmarkGoldmarkSyntaxHighlighting(b *testing.B) {
	converter := NewGoldmarkConverter()
	ctx := context.Background()

	languages := []string{"go", "python", "javascript", "rust", "sql"}

	for _, lang := range languages {
		content := generateCodeBlockWithLanguage(lang, 50)
		b.Run(lang, func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				result, err := converter.ToHTML(ctx, content)
				if err != nil {
					b.Fatal(err)
				}
				_ = result
			}
		})
	}
}

// Helper functions for generating benchmark input

func generateHeadingsMarkdown(count int) string {
	var sb strings.Builder
	for i := 0; i < count; i++ {
		level := (i % 6) + 1
		sb.WriteString(strings.Repeat("#", level))
		sb.WriteString(fmt.Sprintf(" Heading %d\n\n", i+1))
		sb.WriteString("Some content under this heading.\n\n")
	}
	return sb.String()
}

func generateCodeBlocksMarkdown(count int) string {
	var sb strings.Builder
	code := `func example() {
    fmt.Println("Hello, World!")
    for i := 0; i < 10; i++ {
        process(i)
    }
}`
	for i := 0; i < count; i++ {
		sb.WriteString("## Code Example\n\n")
		sb.WriteString("```go\n")
		sb.WriteString(code)
		sb.WriteString("\n```\n\n")
	}
	return sb.String()
}

func generateCodeBlockWithLanguage(lang string, lines int) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("```%s\n", lang))
	for i := 0; i < lines; i++ {
		sb.WriteString(fmt.Sprintf("// Line %d of code\n", i+1))
		sb.WriteString("func example() { return nil }\n")
	}
	sb.WriteString("```\n")
	return sb.String()
}

func generateTablesMarkdown(count int) string {
	var sb strings.Builder
	for i := 0; i < count; i++ {
		sb.WriteString("## Table Section\n\n")
		sb.WriteString("| Column 1 | Column 2 | Column 3 | Column 4 |\n")
		sb.WriteString("|----------|----------|----------|----------|\n")
		for j := 0; j < 10; j++ {
			sb.WriteString(fmt.Sprintf("| Cell %d-1 | Cell %d-2 | Cell %d-3 | Cell %d-4 |\n", j, j, j, j))
		}
		sb.WriteString("\n")
	}
	return sb.String()
}

func generateMixedMarkdown(sections int) string {
	var sb strings.Builder
	sb.WriteString("# Document Title\n\n")
	sb.WriteString("Introduction paragraph with **bold** and *italic* text.\n\n")

	for i := 0; i < sections; i++ {
		sb.WriteString(fmt.Sprintf("## Section %d\n\n", i+1))
		sb.WriteString("This is a paragraph with some content. ")
		sb.WriteString("It includes [links](https://example.com) and `inline code`.\n\n")

		// Add a list
		sb.WriteString("- Item one\n")
		sb.WriteString("- Item two\n")
		sb.WriteString("- Item three\n\n")

		// Add code block every 3rd section
		if i%3 == 0 {
			sb.WriteString("```go\nfunc main() {\n    fmt.Println(\"Hello\")\n}\n```\n\n")
		}

		// Add table every 5th section
		if i%5 == 0 {
			sb.WriteString("| A | B | C |\n|---|---|---|\n| 1 | 2 | 3 |\n\n")
		}
	}

	return sb.String()
}
