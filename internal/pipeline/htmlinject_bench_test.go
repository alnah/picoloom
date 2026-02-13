//go:build bench

package pipeline

import (
	"context"
	"fmt"
	"strings"
	"testing"
)

// BenchmarkCSSInjection_InjectCSS benchmarks CSS injection into HTML.
// Critical for styling as it's called on every conversion.
func BenchmarkCSSInjection_InjectCSS(b *testing.B) {
	injector := &CSSInjection{}
	ctx := context.Background()

	smallHTML := `<!DOCTYPE html>
<html>
<head><title>Test</title></head>
<body><h1>Hello</h1></body>
</html>`

	largeHTML := `<!DOCTYPE html>
<html>
<head><title>Test</title></head>
<body>` + strings.Repeat("<p>Paragraph content here.</p>\n", 500) + `</body>
</html>`

	smallCSS := "body { margin: 0; }"
	largeCSS := strings.Repeat(".class-name { color: red; font-size: 14px; margin: 10px; }\n", 100)

	inputs := []struct {
		name string
		html string
		css  string
	}{
		{"small_html_small_css", smallHTML, smallCSS},
		{"small_html_large_css", smallHTML, largeCSS},
		{"large_html_small_css", largeHTML, smallCSS},
		{"large_html_large_css", largeHTML, largeCSS},
		{"no_head_tag", "<body><p>Content</p></body>", smallCSS},
		{"empty_css", smallHTML, ""},
	}

	for _, input := range inputs {
		b.Run(input.name, func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				result := injector.InjectCSS(ctx, input.html, input.css)
				_ = result
			}
		})
	}
}

// BenchmarkSanitizeCSS benchmarks CSS sanitization.
// Tests escaping of potentially dangerous sequences.
func BenchmarkSanitizeCSS(b *testing.B) {
	inputs := []struct {
		name string
		css  string
	}{
		{"clean", strings.Repeat(".class { color: red; }\n", 50)},
		{"with_escapes", strings.Repeat(".class { content: '</style>'; }\n", 50)},
		{"large_clean", strings.Repeat(".class { color: red; font-size: 14px; }\n", 500)},
	}

	for _, input := range inputs {
		b.Run(input.name, func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				result := sanitizeCSS(input.css)
				_ = result
			}
		})
	}
}

// BenchmarkSignatureInjection_InjectSignature benchmarks signature block injection.
func BenchmarkSignatureInjection_InjectSignature(b *testing.B) {
	injector := NewSignatureInjection()
	ctx := context.Background()

	html := generateTestHTML(100)

	signatures := []struct {
		name string
		data *SignatureData
	}{
		{"nil", nil},
		{"minimal", &SignatureData{Name: "John Doe"}},
		{"full", &SignatureData{
			Name:         "John Doe",
			Title:        "Senior Engineer",
			Email:        "john@example.com",
			Organization: "Example Corp",
			ImagePath:    "/path/to/signature.png",
			Links: []SignatureLink{
				{Label: "LinkedIn", URL: "https://linkedin.com/in/johndoe"},
				{Label: "GitHub", URL: "https://github.com/johndoe"},
			},
		}},
	}

	for _, sig := range signatures {
		b.Run(sig.name, func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				result, err := injector.InjectSignature(ctx, html, sig.data)
				if err != nil {
					b.Fatal(err)
				}
				_ = result
			}
		})
	}
}

// BenchmarkCoverInjection_InjectCover benchmarks cover page injection.
func BenchmarkCoverInjection_InjectCover(b *testing.B) {
	injector := NewCoverInjection()
	ctx := context.Background()

	html := generateTestHTML(100)

	covers := []struct {
		name string
		data *CoverData
	}{
		{"nil", nil},
		{"minimal", &CoverData{Title: "Document Title"}},
		{"full", &CoverData{
			Title:        "Comprehensive Guide",
			Subtitle:     "A Deep Dive into Topics",
			Logo:         "https://example.com/logo.png",
			Author:       "John Doe",
			AuthorTitle:  "Senior Engineer",
			Organization: "Example Corporation",
			Date:         "2025-01-08",
			Version:      "1.0.0",
		}},
	}

	for _, cover := range covers {
		b.Run(cover.name, func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				result, err := injector.InjectCover(ctx, html, cover.data)
				if err != nil {
					b.Fatal(err)
				}
				_ = result
			}
		})
	}
}

// BenchmarkExtractHeadings benchmarks heading extraction from HTML.
// Critical for TOC generation.
func BenchmarkExtractHeadings(b *testing.B) {
	htmls := []struct {
		name    string
		content string
		depth   int
	}{
		{"few_headings", generateHTMLWithHeadings(10), 3},
		{"many_headings", generateHTMLWithHeadings(100), 3},
		{"deep_headings", generateHTMLWithHeadings(50), 6},
		{"shallow_headings", generateHTMLWithHeadings(50), 1},
	}

	for _, h := range htmls {
		b.Run(h.name, func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				result := extractHeadings(h.content, 1, h.depth)
				_ = result
			}
		})
	}
}

// BenchmarkGenerateNumberedTOC benchmarks TOC HTML generation.
func BenchmarkGenerateNumberedTOC(b *testing.B) {
	headingCounts := []int{5, 20, 50, 100}

	for _, count := range headingCounts {
		headings := generateHeadingInfos(count)
		b.Run(fmt.Sprintf("headings_%d", count), func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				result := generateNumberedTOC(headings, "Table of Contents")
				_ = result
			}
		})
	}
}

// BenchmarkTOCInjection_InjectTOC benchmarks full TOC injection.
func BenchmarkTOCInjection_InjectTOC(b *testing.B) {
	injector := NewTOCInjection()
	ctx := context.Background()

	htmls := []struct {
		name string
		html string
		data *TOCData
	}{
		{"nil_data", generateHTMLWithHeadings(20), nil},
		{"shallow", generateHTMLWithHeadings(20), &TOCData{Title: "Contents", MaxDepth: 2}},
		{"deep", generateHTMLWithHeadings(50), &TOCData{Title: "Table of Contents", MaxDepth: 6}},
		{"no_title", generateHTMLWithHeadings(20), &TOCData{Title: "", MaxDepth: 3}},
	}

	for _, h := range htmls {
		b.Run(h.name, func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				result, err := injector.InjectTOC(ctx, h.html, h.data)
				if err != nil {
					b.Fatal(err)
				}
				_ = result
			}
		})
	}
}

// BenchmarkStripHTMLTags benchmarks HTML tag stripping.
func BenchmarkStripHTMLTags(b *testing.B) {
	inputs := []struct {
		name  string
		value string
	}{
		{"no_tags", "Plain text content"},
		{"simple_tags", "<strong>Bold</strong> and <em>italic</em>"},
		{"nested_tags", "<div><span><a href='#'>Link</a></span></div>"},
		{"many_tags", strings.Repeat("<span>text</span>", 50)},
	}

	for _, input := range inputs {
		b.Run(input.name, func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				result := stripHTMLTags(input.value)
				_ = result
			}
		})
	}
}

// Helper functions

func generateTestHTML(paragraphs int) string {
	var sb strings.Builder
	sb.WriteString(`<!DOCTYPE html>
<html>
<head><title>Test</title></head>
<body>
<h1>Document Title</h1>
`)
	for i := 0; i < paragraphs; i++ {
		sb.WriteString(fmt.Sprintf("<p>Paragraph %d with some content.</p>\n", i+1))
	}
	sb.WriteString("</body>\n</html>")
	return sb.String()
}

func generateHTMLWithHeadings(count int) string {
	var sb strings.Builder
	sb.WriteString(`<!DOCTYPE html>
<html>
<head><title>Test</title></head>
<body>
`)
	for i := 0; i < count; i++ {
		level := (i % 6) + 1
		id := fmt.Sprintf("heading-%d", i)
		sb.WriteString(fmt.Sprintf(`<h%d id="%s">Heading %d</h%d>`, level, id, i+1, level))
		sb.WriteString("\n<p>Some content under this heading.</p>\n")
	}
	sb.WriteString("</body>\n</html>")
	return sb.String()
}

func generateHeadingInfos(count int) []headingInfo {
	headings := make([]headingInfo, count)
	for i := 0; i < count; i++ {
		headings[i] = headingInfo{
			Level: (i % 3) + 1, // Levels 1-3
			ID:    fmt.Sprintf("heading-%d", i),
			Text:  fmt.Sprintf("Heading Number %d", i+1),
		}
	}
	return headings
}
