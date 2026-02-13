//go:build bench

package md2pdf

// Notes:
// - Benchmarks for Service.Convert pipeline performance
// - Uses mock PDF converter (benchPDFConverter) to isolate pipeline from browser overhead
// - Tests scaling with document size and concurrent access patterns
// - Also benchmarks data conversion helpers (toSignatureData, toCoverData, etc.)

import (
	"context"
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// Mock Implementations
// ---------------------------------------------------------------------------

// benchPDFConverter is a mock for benchmarking without actual browser.
type benchPDFConverter struct{}

func (m *benchPDFConverter) ToPDF(ctx context.Context, htmlContent string, opts *pdfOptions) ([]byte, error) {
	// Return a mock PDF (minimal valid PDF header)
	return []byte("%PDF-1.4\n"), nil
}

func (m *benchPDFConverter) Close() error {
	return nil
}

// newBenchService creates a Service with mock PDF converter for benchmarking.
func newBenchService() *Service {
	s := New()
	s.pdfConverter = &benchPDFConverter{}
	return s
}

// ---------------------------------------------------------------------------
// BenchmarkService_Convert - Full Pipeline Performance
// ---------------------------------------------------------------------------

func BenchmarkService_Convert(b *testing.B) {
	service := newBenchService()
	defer service.Close()

	ctx := context.Background()

	inputs := []struct {
		name  string
		input Input
	}{
		{
			name: "minimal",
			input: Input{
				Markdown: "# Hello\n\nWorld",
			},
		},
		{
			name: "with_css",
			input: Input{
				Markdown: generateBenchmarkMarkdown(10),
				CSS:      strings.Repeat(".class { color: red; }\n", 50),
			},
		},
		{
			name: "with_footer",
			input: Input{
				Markdown: generateBenchmarkMarkdown(10),
				Footer: &Footer{
					Position:       "right",
					ShowPageNumber: true,
					Date:           "2025-01-08",
					Status:         "v1.0",
				},
			},
		},
		{
			name: "with_signature",
			input: Input{
				Markdown: generateBenchmarkMarkdown(10),
				Signature: &Signature{
					Name:         "John Doe",
					Title:        "Engineer",
					Email:        "john@example.com",
					Organization: "Corp",
				},
			},
		},
		{
			name: "with_watermark",
			input: Input{
				Markdown: generateBenchmarkMarkdown(10),
				Watermark: &Watermark{
					Text:    "DRAFT",
					Color:   "#888888",
					Opacity: 0.1,
					Angle:   -45,
				},
			},
		},
		{
			name: "with_cover",
			input: Input{
				Markdown: generateBenchmarkMarkdown(10),
				Cover: &Cover{
					Title:        "Document Title",
					Subtitle:     "A Guide",
					Author:       "John Doe",
					Organization: "Corp",
					Date:         "2025-01-08",
					Version:      "1.0",
				},
			},
		},
		{
			name: "with_toc",
			input: Input{
				Markdown: generateBenchmarkMarkdown(20),
				TOC: &TOC{
					Title:    "Contents",
					MaxDepth: 3,
				},
			},
		},
		{
			name: "full_features",
			input: Input{
				Markdown: generateBenchmarkMarkdown(20),
				CSS:      strings.Repeat(".class { color: red; }\n", 20),
				Footer: &Footer{
					Position:       "center",
					ShowPageNumber: true,
					Date:           "2025-01-08",
					Status:         "v2.0",
					Text:           "Confidential",
				},
				Signature: &Signature{
					Name:         "John Doe",
					Title:        "Senior Engineer",
					Email:        "john@example.com",
					Organization: "Example Corp",
					Links: []Link{
						{Label: "LinkedIn", URL: "https://linkedin.com"},
					},
				},
				Page: &PageSettings{
					Size:        "a4",
					Orientation: "portrait",
					Margin:      0.75,
				},
				Watermark: &Watermark{
					Text:    "CONFIDENTIAL",
					Color:   "#ff0000",
					Opacity: 0.15,
					Angle:   -30,
				},
				Cover: &Cover{
					Title:        "Comprehensive Technical Guide",
					Subtitle:     "Version 2.0",
					Author:       "John Doe",
					AuthorTitle:  "Senior Engineer",
					Organization: "Example Corporation",
					Date:         "2025-01-08",
					Version:      "2.0.0",
				},
				TOC: &TOC{
					Title:    "Table of Contents",
					MaxDepth: 4,
				},
				PageBreaks: &PageBreaks{
					BeforeH1: true,
					BeforeH2: true,
					Orphans:  3,
					Widows:   3,
				},
			},
		},
	}

	for _, input := range inputs {
		b.Run(input.name, func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				result, err := service.Convert(ctx, input.input)
				if err != nil {
					b.Fatalf("Convert() unexpected error: %v", err)
				}
				_ = result
			}
		})
	}
}

// ---------------------------------------------------------------------------
// BenchmarkService_ConvertBySize - Document Size Scaling
// ---------------------------------------------------------------------------

func BenchmarkService_ConvertBySize(b *testing.B) {
	service := newBenchService()
	defer service.Close()

	ctx := context.Background()
	sizes := []int{5, 10, 25, 50, 100}

	for _, size := range sizes {
		input := Input{
			Markdown: generateBenchmarkMarkdown(size),
			CSS:      strings.Repeat(".class { color: red; }\n", 20),
			Footer: &Footer{
				Position:       "right",
				ShowPageNumber: true,
			},
		}

		b.Run(sizeName(size), func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				result, err := service.Convert(ctx, input)
				if err != nil {
					b.Fatalf("Convert() unexpected error: %v", err)
				}
				_ = result
			}
		})
	}
}

func sizeName(size int) string {
	switch size {
	case 5:
		return "sections_5"
	case 10:
		return "sections_10"
	case 25:
		return "sections_25"
	case 50:
		return "sections_50"
	case 100:
		return "sections_100"
	default:
		return "sections_n"
	}
}

// ---------------------------------------------------------------------------
// BenchmarkService_ConvertParallel - Concurrent Conversions
// ---------------------------------------------------------------------------

func BenchmarkService_ConvertParallel(b *testing.B) {
	service := newBenchService()
	defer service.Close()

	ctx := context.Background()
	input := Input{
		Markdown: generateBenchmarkMarkdown(20),
		CSS:      strings.Repeat(".class { color: red; }\n", 20),
		Footer:   &Footer{Position: "right", ShowPageNumber: true},
		Watermark: &Watermark{
			Text:    "DRAFT",
			Color:   "#888888",
			Opacity: 0.1,
			Angle:   -45,
		},
	}

	b.ReportAllocs()
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			result, err := service.Convert(ctx, input)
			if err != nil {
				b.Fatalf("Convert() unexpected error: %v", err)
			}
			_ = result
		}
	})
}

// ---------------------------------------------------------------------------
// BenchmarkValidateInput - Input Validation Performance
// ---------------------------------------------------------------------------

func BenchmarkValidateInput(b *testing.B) {
	service := newBenchService()
	defer service.Close()

	inputs := []struct {
		name  string
		input Input
	}{
		{"minimal", Input{Markdown: "# Test"}},
		{"with_page", Input{
			Markdown: "# Test",
			Page:     &PageSettings{Size: "a4", Orientation: "portrait", Margin: 0.5},
		}},
		{"with_footer", Input{
			Markdown: "# Test",
			Footer:   &Footer{Position: "right", ShowPageNumber: true},
		}},
		{"with_watermark", Input{
			Markdown:  "# Test",
			Watermark: &Watermark{Text: "DRAFT", Color: "#888", Opacity: 0.1, Angle: -45},
		}},
		{"with_cover", Input{
			Markdown: "# Test",
			Cover:    &Cover{Title: "Document"},
		}},
		{"with_toc", Input{
			Markdown: "# Test",
			TOC:      &TOC{Title: "Contents", MaxDepth: 3},
		}},
		{"full", Input{
			Markdown:   "# Test",
			Page:       &PageSettings{Size: "letter", Orientation: "portrait", Margin: 0.5},
			Footer:     &Footer{Position: "right"},
			Watermark:  &Watermark{Text: "DRAFT", Color: "#888", Opacity: 0.1, Angle: -45},
			Cover:      &Cover{Title: "Doc"},
			TOC:        &TOC{MaxDepth: 3},
			PageBreaks: &PageBreaks{Orphans: 2, Widows: 2},
		}},
	}

	for _, input := range inputs {
		b.Run(input.name, func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				err := service.validateInput(input.input)
				_ = err
			}
		})
	}
}

// ---------------------------------------------------------------------------
// BenchmarkToSignatureData - Signature Data Conversion Performance
// ---------------------------------------------------------------------------

func BenchmarkToSignatureData(b *testing.B) {
	sig := &Signature{
		Name:         "John Doe",
		Title:        "Senior Engineer",
		Email:        "john@example.com",
		Organization: "Example Corp",
		ImagePath:    "/path/to/image.png",
		Links: []Link{
			{Label: "LinkedIn", URL: "https://linkedin.com/in/johndoe"},
			{Label: "GitHub", URL: "https://github.com/johndoe"},
		},
	}

	b.Run("nil", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			result := toSignatureData(nil)
			_ = result
		}
	})

	b.Run("full", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			result := toSignatureData(sig)
			_ = result
		}
	})
}

// ---------------------------------------------------------------------------
// BenchmarkToCoverData - Cover Data Conversion Performance
// ---------------------------------------------------------------------------

func BenchmarkToCoverData(b *testing.B) {
	cover := &Cover{
		Title:        "Document Title",
		Subtitle:     "A Comprehensive Guide",
		Logo:         "https://example.com/logo.png",
		Author:       "John Doe",
		AuthorTitle:  "Senior Engineer",
		Organization: "Example Corp",
		Date:         "2025-01-08",
		Version:      "1.0.0",
	}

	b.Run("nil", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			result := toCoverData(nil)
			_ = result
		}
	})

	b.Run("full", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			result := toCoverData(cover)
			_ = result
		}
	})
}

// ---------------------------------------------------------------------------
// BenchmarkToFooterData - Footer Data Conversion Performance
// ---------------------------------------------------------------------------

func BenchmarkToFooterData(b *testing.B) {
	footer := &Footer{
		Position:       "center",
		ShowPageNumber: true,
		Date:           "2025-01-08",
		Status:         "v2.0",
		Text:           "Confidential",
	}

	b.Run("nil", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			result := toFooterData(nil)
			_ = result
		}
	})

	b.Run("full", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			result := toFooterData(footer)
			_ = result
		}
	})
}

// ---------------------------------------------------------------------------
// BenchmarkToTOCData - TOC Data Conversion Performance
// ---------------------------------------------------------------------------

func BenchmarkToTOCData(b *testing.B) {
	toc := &TOC{
		Title:    "Table of Contents",
		MaxDepth: 4,
	}

	b.Run("nil", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			result := toTOCData(nil)
			_ = result
		}
	})

	b.Run("full", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			result := toTOCData(toc)
			_ = result
		}
	})
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func generateBenchmarkMarkdown(sections int) string {
	var sb strings.Builder
	sb.WriteString("# Document Title\n\n")
	sb.WriteString("Introduction paragraph with **bold** and *italic* text.\n\n")

	for i := 0; i < sections; i++ {
		level := (i % 3) + 1
		sb.WriteString(strings.Repeat("#", level+1))
		sb.WriteString(" Section ")
		sb.WriteString(string(rune('A' + (i % 26))))
		sb.WriteString("\n\n")
		sb.WriteString("This is a paragraph with some content. ")
		sb.WriteString("It includes [links](https://example.com) and `inline code`.\n\n")

		sb.WriteString("- Item one\n")
		sb.WriteString("- Item two\n")
		sb.WriteString("- Item three\n\n")

		if i%3 == 0 {
			sb.WriteString("```go\nfunc main() {\n    fmt.Println(\"Hello\")\n}\n```\n\n")
		}

		if i%5 == 0 {
			sb.WriteString("| A | B | C |\n|---|---|---|\n| 1 | 2 | 3 |\n\n")
		}
	}

	return sb.String()
}
