//go:build bench

package picoloom

// Notes:
// - Benchmarks CSS building functions with various input configurations
// - Tests performance of watermark, page breaks, and string escaping
// - Uses ReportAllocs to track memory allocations

import (
	"testing"
)

func BenchmarkBuildWatermarkCSS(b *testing.B) {
	watermarks := []struct {
		name string
		data *Watermark
	}{
		{"nil", nil},
		{"simple", &Watermark{Text: "DRAFT", Color: "#888888", Opacity: 0.1, Angle: -45}},
		{"long_text", &Watermark{Text: "CONFIDENTIAL DOCUMENT", Color: "#ff0000", Opacity: 0.2, Angle: -30}},
	}

	for _, wm := range watermarks {
		b.Run(wm.name, func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				result := buildWatermarkCSS(wm.data)
				_ = result
			}
		})
	}
}

func BenchmarkBuildPageBreaksCSS(b *testing.B) {
	configs := []struct {
		name string
		data *PageBreaks
	}{
		{"nil", nil},
		{"defaults", &PageBreaks{Orphans: 2, Widows: 2}},
		{"all_breaks", &PageBreaks{BeforeH1: true, BeforeH2: true, BeforeH3: true, Orphans: 3, Widows: 3}},
	}

	for _, cfg := range configs {
		b.Run(cfg.name, func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				result := buildPageBreaksCSS(cfg.data)
				_ = result
			}
		})
	}
}

func BenchmarkEscapeCSSString(b *testing.B) {
	inputs := []struct {
		name  string
		value string
	}{
		{"clean", "DRAFT"},
		{"with_quotes", `Text with "quotes"`},
		{"with_backslash", `Path\to\file`},
		{"with_newlines", "Line1\nLine2\r\nLine3"},
		{"with_percent", "100% Complete"},
		{"complex", `"Complex\nString" with 100% escapes`},
	}

	for _, input := range inputs {
		b.Run(input.name, func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				result := escapeCSSString(input.value)
				_ = result
			}
		})
	}
}
