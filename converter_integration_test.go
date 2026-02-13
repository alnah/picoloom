//go:build integration

package md2pdf

// Notes:
// - Tests NewConversionService with proper pipeline component initialization
// - Tests Convert with various page settings and page breaks configurations
// - Tests file output writing and PDF validity
// - Uses acquireService helper from integration_setup_test.go for pooled services

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/alnah/go-md2pdf/internal/pipeline"
)

// ---------------------------------------------------------------------------
// TestNewConverter - Service Initialization
// ---------------------------------------------------------------------------

func TestNewConverter(t *testing.T) {
	t.Parallel()

	service := acquireService(t)

	if service.preprocessor == nil {
		t.Errorf("NewConverter() preprocessor = nil, want non-nil")
	}
	if _, ok := service.preprocessor.(*pipeline.CommonMarkPreprocessor); !ok {
		t.Errorf("NewConverter() preprocessor type = %T, want *pipeline.CommonMarkPreprocessor", service.preprocessor)
	}

	if service.htmlConverter == nil {
		t.Errorf("NewConverter() htmlConverter = nil, want non-nil")
	}
	if _, ok := service.htmlConverter.(*pipeline.GoldmarkConverter); !ok {
		t.Errorf("NewConverter() htmlConverter type = %T, want *pipeline.GoldmarkConverter", service.htmlConverter)
	}

	if service.cssInjector == nil {
		t.Errorf("NewConverter() cssInjector = nil, want non-nil")
	}
	if _, ok := service.cssInjector.(*pipeline.CSSInjection); !ok {
		t.Errorf("NewConverter() cssInjector type = %T, want *pipeline.CSSInjection", service.cssInjector)
	}

	if service.pdfConverter == nil {
		t.Errorf("NewConverter() pdfConverter = nil, want non-nil")
	}
	// pdfConverter is already *rodConverter (concrete type), type assertion not needed
}

// ---------------------------------------------------------------------------
// TestConverter_Convert - Basic Conversion
// ---------------------------------------------------------------------------

func TestConverter_Convert(t *testing.T) {
	t.Parallel()

	service := acquireService(t)

	ctx := context.Background()
	input := Input{
		Markdown: "# Hello\n\nWorld",
	}

	data, err := service.Convert(ctx, input)
	if err != nil {
		t.Fatalf("Convert() unexpected error: %v", err)
	}

	// Verify PDF bytes
	if !bytes.HasPrefix(data.PDF, []byte("%PDF-")) {
		t.Errorf("Convert() PDF missing magic bytes, got prefix %q", data.PDF[:5])
	}

	if len(data.PDF) < 100 {
		t.Errorf("Convert() PDF size = %d bytes, want >= 100", len(data.PDF))
	}
}

// ---------------------------------------------------------------------------
// TestConverter_Convert_FileOutput - File Output
// ---------------------------------------------------------------------------

func TestConverter_Convert_FileOutput(t *testing.T) {
	t.Parallel()

	service := acquireService(t)

	ctx := context.Background()
	input := Input{
		Markdown: "# Hello\n\nWorld",
	}

	data, err := service.Convert(ctx, input)
	if err != nil {
		t.Fatalf("Convert() unexpected error: %v", err)
	}

	outputPath := filepath.Join(t.TempDir(), "out.pdf")
	err = os.WriteFile(outputPath, data.PDF, 0644)
	if err != nil {
		t.Fatalf("WriteFile() unexpected error: %v", err)
	}

	info, err := os.Stat(outputPath)
	if err != nil {
		t.Fatalf("Stat() unexpected error: %v", err)
	}
	if info.Size() == 0 {
		t.Errorf("PDF file size = 0, want > 0")
	}
}

// ---------------------------------------------------------------------------
// TestConverter_Convert_PageSettings - Page Settings Variations
// ---------------------------------------------------------------------------

func TestConverter_Convert_PageSettings(t *testing.T) {
	t.Parallel()

	// Test various page settings combinations to ensure they don't crash
	// and produce valid PDF output
	tests := []struct {
		name string
		page *PageSettings
	}{
		{
			name: "nil uses defaults",
			page: nil,
		},
		{
			name: "letter portrait",
			page: &PageSettings{Size: PageSizeLetter, Orientation: OrientationPortrait, Margin: DefaultMargin},
		},
		{
			name: "a4 portrait",
			page: &PageSettings{Size: PageSizeA4, Orientation: OrientationPortrait, Margin: 0.5},
		},
		{
			name: "a4 landscape",
			page: &PageSettings{Size: PageSizeA4, Orientation: OrientationLandscape, Margin: 0.5},
		},
		{
			name: "legal portrait",
			page: &PageSettings{Size: PageSizeLegal, Orientation: OrientationPortrait, Margin: 0.5},
		},
		{
			name: "legal landscape",
			page: &PageSettings{Size: PageSizeLegal, Orientation: OrientationLandscape, Margin: 1.0},
		},
		{
			name: "letter landscape custom margin",
			page: &PageSettings{Size: PageSizeLetter, Orientation: OrientationLandscape, Margin: 1.5},
		},
		{
			name: "minimum margin",
			page: &PageSettings{Size: PageSizeLetter, Orientation: OrientationPortrait, Margin: MinMargin},
		},
		{
			name: "maximum margin",
			page: &PageSettings{Size: PageSizeLetter, Orientation: OrientationPortrait, Margin: MaxMargin},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			service := acquireService(t)
			ctx := context.Background()
			input := Input{
				Markdown: "# Page Settings Test\n\nThis is a test document.",
				Page:     tt.page,
			}

			data, err := service.Convert(ctx, input)
			if err != nil {
				t.Fatalf("Convert() unexpected error: %v", err)
			}

			// Verify PDF magic bytes
			if !bytes.HasPrefix(data.PDF, []byte("%PDF-")) {
				t.Errorf("Convert() PDF missing magic bytes, got prefix %q", data.PDF[:5])
			}

			// Ensure PDF is not suspiciously small
			if len(data.PDF) < 100 {
				t.Errorf("Convert() PDF size = %d bytes, want >= 100", len(data.PDF))
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestConverter_Convert_PageSettingsWithFooter - Page Settings with Footer
// ---------------------------------------------------------------------------

func TestConverter_Convert_PageSettingsWithFooter(t *testing.T) {
	t.Parallel()

	service := acquireService(t)

	ctx := context.Background()
	input := Input{
		Markdown: "# Test with Footer\n\nContent here.",
		Page:     &PageSettings{Size: PageSizeA4, Orientation: OrientationLandscape, Margin: 1.0},
		Footer: &Footer{
			Position:       "center",
			ShowPageNumber: true,
			Text:           "Footer Text",
		},
	}

	data, err := service.Convert(ctx, input)
	if err != nil {
		t.Fatalf("Convert() unexpected error: %v", err)
	}

	if !bytes.HasPrefix(data.PDF, []byte("%PDF-")) {
		t.Errorf("Convert() PDF missing magic bytes, got prefix %q", data.PDF[:5])
	}
}

// ---------------------------------------------------------------------------
// TestConverter_Convert_PageBreaks - Page Breaks Configurations
// ---------------------------------------------------------------------------

func TestConverter_Convert_PageBreaks(t *testing.T) {
	t.Parallel()

	// Test various page break configurations to ensure they produce valid PDF output
	tests := []struct {
		name       string
		pageBreaks *PageBreaks
	}{
		{
			name:       "nil uses defaults",
			pageBreaks: nil,
		},
		{
			name:       "empty struct uses defaults",
			pageBreaks: &PageBreaks{},
		},
		{
			name:       "custom orphans and widows",
			pageBreaks: &PageBreaks{Orphans: 3, Widows: 4},
		},
		{
			name:       "break before H1",
			pageBreaks: &PageBreaks{BeforeH1: true},
		},
		{
			name:       "break before H2",
			pageBreaks: &PageBreaks{BeforeH2: true},
		},
		{
			name:       "break before H3",
			pageBreaks: &PageBreaks{BeforeH3: true},
		},
		{
			name:       "all heading breaks enabled",
			pageBreaks: &PageBreaks{BeforeH1: true, BeforeH2: true, BeforeH3: true},
		},
		{
			name:       "full configuration",
			pageBreaks: &PageBreaks{BeforeH1: true, BeforeH2: true, BeforeH3: true, Orphans: 5, Widows: 5},
		},
		{
			name:       "minimum orphans and widows",
			pageBreaks: &PageBreaks{Orphans: MinOrphans, Widows: MinWidows},
		},
		{
			name:       "maximum orphans and widows",
			pageBreaks: &PageBreaks{Orphans: MaxOrphans, Widows: MaxWidows},
		},
	}

	// Markdown with multiple headings to test page breaks
	markdown := `# Chapter 1

This is the first chapter with some content.

## Section 1.1

Some content in section 1.1.

### Subsection 1.1.1

Details in subsection.

# Chapter 2

This is the second chapter.

## Section 2.1

More content here.
`

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			service := acquireService(t)
			ctx := context.Background()
			input := Input{
				Markdown:   markdown,
				PageBreaks: tt.pageBreaks,
			}

			data, err := service.Convert(ctx, input)
			if err != nil {
				t.Fatalf("Convert() unexpected error: %v", err)
			}

			// Verify PDF magic bytes
			if !bytes.HasPrefix(data.PDF, []byte("%PDF-")) {
				t.Errorf("Convert() PDF missing magic bytes, got prefix %q", data.PDF[:5])
			}

			// Ensure PDF is not suspiciously small
			if len(data.PDF) < 100 {
				t.Errorf("Convert() PDF size = %d bytes, want >= 100", len(data.PDF))
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestConverter_Convert_PageBreaksWithOtherFeatures - Combined Features
// ---------------------------------------------------------------------------

func TestConverter_Convert_PageBreaksWithOtherFeatures(t *testing.T) {
	t.Parallel()

	service := acquireService(t)

	ctx := context.Background()
	input := Input{
		Markdown: "# Test with Page Breaks\n\n## Section One\n\nContent here.\n\n## Section Two\n\nMore content.",
		CSS:      "body { font-family: sans-serif; }",
		Page:     &PageSettings{Size: PageSizeA4, Orientation: OrientationPortrait, Margin: 1.0},
		PageBreaks: &PageBreaks{
			BeforeH1: true,
			BeforeH2: true,
			Orphans:  3,
			Widows:   3,
		},
		Footer: &Footer{
			Position:       "center",
			ShowPageNumber: true,
		},
	}

	data, err := service.Convert(ctx, input)
	if err != nil {
		t.Fatalf("Convert() unexpected error: %v", err)
	}

	if !bytes.HasPrefix(data.PDF, []byte("%PDF-")) {
		t.Errorf("Convert() PDF missing magic bytes, got prefix %q", data.PDF[:5])
	}

	if len(data.PDF) < 100 {
		t.Errorf("Convert() PDF size = %d bytes, want >= 100", len(data.PDF))
	}
}
