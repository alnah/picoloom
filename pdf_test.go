package md2pdf

// Notes:
// - Tests rodConverter and rodRenderer with mock implementations
// - Tests buildFooterTemplate with various footer configurations
// - Tests resolvePageDimensions for all page sizes and orientations
// - Tests buildPDFOptions for margin calculations with footer

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/alnah/go-md2pdf/internal/fileutil"
	"github.com/alnah/go-md2pdf/internal/pipeline"
)

// ---------------------------------------------------------------------------
// Compile-Time Interface Checks
// ---------------------------------------------------------------------------

var (
	_ pdfConverter = (*rodConverter)(nil)
	_ pdfRenderer  = (*rodRenderer)(nil)
)

// ---------------------------------------------------------------------------
// Mock Implementations
// ---------------------------------------------------------------------------

type mockRenderer struct {
	Result     []byte
	Err        error
	CalledWith string
	CalledOpts *pdfOptions
}

func (m *mockRenderer) RenderFromFile(ctx context.Context, filePath string, opts *pdfOptions) ([]byte, error) {
	m.CalledWith = filePath
	m.CalledOpts = opts
	return m.Result, m.Err
}

type testableRodConverter struct {
	mock *mockRenderer
}

func (c *testableRodConverter) ToPDF(ctx context.Context, htmlContent string, opts *pdfOptions) ([]byte, error) {
	tmpPath, cleanup, err := fileutil.WriteTempFile(htmlContent, "html")
	if err != nil {
		return nil, err
	}
	defer cleanup()

	return c.mock.RenderFromFile(ctx, tmpPath, opts)
}

// ---------------------------------------------------------------------------
// TestRodConverter_ToPDF - PDF Conversion with Mock Renderer
// ---------------------------------------------------------------------------

func TestRodConverter_ToPDF(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		html       string
		mock       *mockRenderer
		wantErr    error
		wantAnyErr bool
	}{
		{
			name: "successful render returns PDF bytes",
			html: "<html><body>Test</body></html>",
			mock: &mockRenderer{
				Result: []byte("%PDF-1.4 fake pdf content"),
			},
		},
		{
			name: "renderer error propagates",
			html: "<html></html>",
			mock: &mockRenderer{
				Err: errors.New("browser crashed"),
			},
			wantAnyErr: true,
		},
		{
			name: "empty HTML is valid",
			html: "",
			mock: &mockRenderer{
				Result: []byte("%PDF-1.4"),
			},
		},
		{
			name: "unicode content succeeds",
			html: "<html><body>Hello World</body></html>",
			mock: &mockRenderer{
				Result: []byte("%PDF-1.4 unicode"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			converter := &testableRodConverter{mock: tt.mock}
			ctx := context.Background()

			result, err := converter.ToPDF(ctx, tt.html, nil)

			if tt.wantAnyErr || tt.wantErr != nil {
				if err == nil {
					t.Fatalf("ToPDF(ctx, %q, nil) error = nil, want error", tt.html)
				}
				if tt.wantErr != nil && !errors.Is(err, tt.wantErr) {
					t.Errorf("ToPDF(ctx, %q, nil) error = %v, want %v", tt.html, err, tt.wantErr)
				}
				return
			}

			if err != nil {
				t.Fatalf("ToPDF(ctx, %q, nil) unexpected error: %v", tt.html, err)
			}

			// Verify PDF bytes returned
			if string(result) != string(tt.mock.Result) {
				t.Errorf("ToPDF(ctx, %q, nil) = %q, want %q", tt.html, result, tt.mock.Result)
			}

			// Verify renderer was called with temp file
			if !strings.Contains(tt.mock.CalledWith, "md2pdf-") {
				t.Errorf("ToPDF(ctx, %q, nil) temp file path = %q, want path containing 'md2pdf-'", tt.html, tt.mock.CalledWith)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestRodConverter_ToPDF_ContextCancellation - Context Handling
// ---------------------------------------------------------------------------

func TestRodConverter_ToPDF_ContextCancellation(t *testing.T) {
	t.Parallel()

	mock := &mockRenderer{
		Result: []byte("%PDF-1.4"),
	}
	converter := &testableRodConverter{mock: mock}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	// The mock doesn't check context, but real renderer would
	// This test verifies the converter accepts context parameter
	_, err := converter.ToPDF(ctx, "<html></html>", nil)
	// Mock doesn't check context, so it succeeds
	if err != nil {
		t.Fatalf("ToPDF(ctx, \"<html></html>\", nil) unexpected error: %v", err)
	}
}

// ---------------------------------------------------------------------------
// TestNewRodConverter - Converter Creation
// ---------------------------------------------------------------------------

func TestNewRodConverter(t *testing.T) {
	t.Parallel()

	converter := newRodConverter(defaultTimeout)

	if converter.renderer == nil {
		t.Fatalf("newRodConverter(%v).renderer = nil, want non-nil", defaultTimeout)
	}

	if converter.renderer.timeout != defaultTimeout {
		t.Errorf("newRodConverter(%v).renderer.timeout = %v, want %v", defaultTimeout, converter.renderer.timeout, defaultTimeout)
	}
}

func TestRenderOperationContext(t *testing.T) {
	t.Parallel()

	t.Run("uses caller context when it already has deadline", func(t *testing.T) {
		t.Parallel()

		parent, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		gotCtx, gotCancel, err := renderOperationContext(parent, 30*time.Second)
		defer gotCancel()

		if err != nil {
			t.Fatalf("renderOperationContext(parent, 30s) unexpected error: %v", err)
		}

		parentDeadline, parentOK := parent.Deadline()
		gotDeadline, gotOK := gotCtx.Deadline()
		if !parentOK || !gotOK {
			t.Fatalf("expected deadline on both contexts")
		}
		if !parentDeadline.Equal(gotDeadline) {
			t.Errorf("renderOperationContext(parent, 30s) deadline = %v, want %v", gotDeadline, parentDeadline)
		}
	})

	t.Run("adds fallback timeout when caller has no deadline", func(t *testing.T) {
		t.Parallel()

		start := time.Now()
		gotCtx, gotCancel, err := renderOperationContext(context.Background(), 500*time.Millisecond)
		defer gotCancel()

		if err != nil {
			t.Fatalf("renderOperationContext(background, 500ms) unexpected error: %v", err)
		}

		gotDeadline, ok := gotCtx.Deadline()
		if !ok {
			t.Fatal("renderOperationContext(background, 500ms) missing deadline")
		}
		remaining := time.Until(gotDeadline)
		if remaining > 500*time.Millisecond || remaining < 300*time.Millisecond {
			t.Errorf("renderOperationContext(background, 500ms) remaining timeout = %v, want around 500ms", remaining)
		}
		if time.Since(start) > 250*time.Millisecond {
			t.Fatalf("test setup took too long; timeout assertion would be unreliable")
		}
	})

	t.Run("returns cancellation error when caller is already canceled", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		gotCtx, gotCancel, err := renderOperationContext(ctx, time.Second)
		if err == nil {
			t.Fatal("renderOperationContext(canceled, 1s) error = nil, want error")
		}
		if !errors.Is(err, context.Canceled) {
			t.Errorf("renderOperationContext(canceled, 1s) error = %v, want context.Canceled", err)
		}
		if gotCtx != nil {
			t.Errorf("renderOperationContext(canceled, 1s) context = %v, want nil", gotCtx)
		}
		if gotCancel != nil {
			t.Errorf("renderOperationContext(canceled, 1s) cancel = %v, want nil", gotCancel)
		}
	})
}

func TestRodRenderer_EnsureBrowser_ContextCanceled(t *testing.T) {
	t.Parallel()

	renderer := newRodRenderer(defaultTimeout)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := renderer.ensureBrowser(ctx)
	if err == nil {
		t.Fatal("ensureBrowser(canceledCtx) error = nil, want context.Canceled")
	}
	if !errors.Is(err, context.Canceled) {
		t.Errorf("ensureBrowser(canceledCtx) error = %v, want context.Canceled", err)
	}
}

// ---------------------------------------------------------------------------
// TestBuildFooterTemplate - Footer Template Generation
// ---------------------------------------------------------------------------

func TestBuildFooterTemplate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		data     *pipeline.FooterData
		wantPart string // Substring that should appear
		wantNot  string // Substring that should NOT appear
	}{
		{
			name:     "nil data returns empty span",
			data:     nil,
			wantPart: "<span></span>",
		},
		{
			name:     "page number only",
			data:     &pipeline.FooterData{ShowPageNumber: true},
			wantPart: `class="pageNumber"`,
		},
		{
			name:     "date only",
			data:     &pipeline.FooterData{Date: "2025-01-15"},
			wantPart: "2025-01-15",
		},
		{
			name:     "status only",
			data:     &pipeline.FooterData{Status: "DRAFT"},
			wantPart: "DRAFT",
		},
		{
			name:     "text only",
			data:     &pipeline.FooterData{Text: "Footer Text"},
			wantPart: "Footer Text",
		},
		{
			name: "all fields",
			data: &pipeline.FooterData{
				ShowPageNumber: true,
				Date:           "2025-01-15",
				Status:         "DRAFT",
				Text:           "Custom",
			},
			wantPart: "pageNumber",
		},
		{
			name:     "left position",
			data:     &pipeline.FooterData{Text: "Test", Position: "left"},
			wantPart: "text-align: left",
		},
		{
			name:     "center position",
			data:     &pipeline.FooterData{Text: "Test", Position: "center"},
			wantPart: "text-align: center",
		},
		{
			name:     "right position (default)",
			data:     &pipeline.FooterData{Text: "Test", Position: "right"},
			wantPart: "text-align: right",
		},
		{
			name:     "empty position defaults to right",
			data:     &pipeline.FooterData{Text: "Test"},
			wantPart: "text-align: right",
		},
		{
			name:    "HTML escapes special chars",
			data:    &pipeline.FooterData{Text: "<script>alert('xss')</script>"},
			wantNot: "<script>",
		},
		{
			name:     "DocumentID only",
			data:     &pipeline.FooterData{DocumentID: "DOC-2024-001"},
			wantPart: "DOC-2024-001",
		},
		{
			name: "DocumentID with other fields",
			data: &pipeline.FooterData{
				Date:       "2025-01-15",
				Status:     "FINAL",
				DocumentID: "REF-001",
			},
			wantPart: "REF-001",
		},
		{
			name:    "DocumentID HTML escapes special chars",
			data:    &pipeline.FooterData{DocumentID: "<doc>&test</doc>"},
			wantNot: "<doc>",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := buildFooterTemplate(tt.data)

			if tt.wantPart != "" && !strings.Contains(result, tt.wantPart) {
				t.Errorf("buildFooterTemplate(%v) missing substring %q, got: %s", tt.data, tt.wantPart, result)
			}
			if tt.wantNot != "" && strings.Contains(result, tt.wantNot) {
				t.Errorf("buildFooterTemplate(%v) contains forbidden substring %q, got: %s", tt.data, tt.wantNot, result)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestResolvePageDimensions - Page Dimension Calculation
// ---------------------------------------------------------------------------

func TestResolvePageDimensions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name             string
		page             *PageSettings
		hasFooter        bool
		wantW            float64
		wantH            float64
		wantMargin       float64
		wantBottomMargin float64
	}{
		{
			name:             "nil uses defaults (letter portrait)",
			page:             nil,
			hasFooter:        false,
			wantW:            8.5,
			wantH:            11.0,
			wantMargin:       DefaultMargin,
			wantBottomMargin: DefaultMargin,
		},
		{
			name:             "nil with footer adds extra bottom margin",
			page:             nil,
			hasFooter:        true,
			wantW:            8.5,
			wantH:            11.0,
			wantMargin:       DefaultMargin,
			wantBottomMargin: DefaultMargin + footerMarginExtra,
		},
		{
			name:             "letter portrait explicit",
			page:             &PageSettings{Size: "letter", Orientation: "portrait", Margin: 0.5},
			hasFooter:        false,
			wantW:            8.5,
			wantH:            11.0,
			wantMargin:       0.5,
			wantBottomMargin: 0.5,
		},
		{
			name:             "letter landscape swaps dimensions",
			page:             &PageSettings{Size: "letter", Orientation: "landscape", Margin: 0.5},
			hasFooter:        false,
			wantW:            11.0,
			wantH:            8.5,
			wantMargin:       0.5,
			wantBottomMargin: 0.5,
		},
		{
			name:             "a4 portrait",
			page:             &PageSettings{Size: "a4", Orientation: "portrait", Margin: 0.5},
			hasFooter:        false,
			wantW:            8.27,
			wantH:            11.69,
			wantMargin:       0.5,
			wantBottomMargin: 0.5,
		},
		{
			name:             "a4 landscape",
			page:             &PageSettings{Size: "a4", Orientation: "landscape", Margin: 0.5},
			hasFooter:        false,
			wantW:            11.69,
			wantH:            8.27,
			wantMargin:       0.5,
			wantBottomMargin: 0.5,
		},
		{
			name:             "legal portrait",
			page:             &PageSettings{Size: "legal", Orientation: "portrait", Margin: 0.5},
			hasFooter:        false,
			wantW:            8.5,
			wantH:            14.0,
			wantMargin:       0.5,
			wantBottomMargin: 0.5,
		},
		{
			name:             "legal landscape",
			page:             &PageSettings{Size: "legal", Orientation: "landscape", Margin: 0.5},
			hasFooter:        false,
			wantW:            14.0,
			wantH:            8.5,
			wantMargin:       0.5,
			wantBottomMargin: 0.5,
		},
		{
			name:             "custom margin",
			page:             &PageSettings{Size: "letter", Orientation: "portrait", Margin: 1.0},
			hasFooter:        false,
			wantW:            8.5,
			wantH:            11.0,
			wantMargin:       1.0,
			wantBottomMargin: 1.0,
		},
		{
			name:             "custom margin with footer",
			page:             &PageSettings{Size: "letter", Orientation: "portrait", Margin: 1.0},
			hasFooter:        true,
			wantW:            8.5,
			wantH:            11.0,
			wantMargin:       1.0,
			wantBottomMargin: 1.0 + footerMarginExtra,
		},
		{
			name:             "case insensitive size",
			page:             &PageSettings{Size: "A4", Orientation: "portrait", Margin: 0.5},
			hasFooter:        false,
			wantW:            8.27,
			wantH:            11.69,
			wantMargin:       0.5,
			wantBottomMargin: 0.5,
		},
		{
			name:             "case insensitive orientation",
			page:             &PageSettings{Size: "letter", Orientation: "LANDSCAPE", Margin: 0.5},
			hasFooter:        false,
			wantW:            11.0,
			wantH:            8.5,
			wantMargin:       0.5,
			wantBottomMargin: 0.5,
		},
		{
			name:             "unknown size falls back to letter",
			page:             &PageSettings{Size: "tabloid", Orientation: "portrait", Margin: 0.5},
			hasFooter:        false,
			wantW:            8.5,
			wantH:            11.0,
			wantMargin:       0.5,
			wantBottomMargin: 0.5,
		},
		{
			name:             "empty size uses default letter",
			page:             &PageSettings{Size: "", Orientation: "portrait", Margin: 0.5},
			hasFooter:        false,
			wantW:            8.5,
			wantH:            11.0,
			wantMargin:       0.5,
			wantBottomMargin: 0.5,
		},
		{
			name:             "empty orientation uses portrait",
			page:             &PageSettings{Size: "letter", Orientation: "", Margin: 0.5},
			hasFooter:        false,
			wantW:            8.5,
			wantH:            11.0,
			wantMargin:       0.5,
			wantBottomMargin: 0.5,
		},
		{
			name:             "zero margin uses default",
			page:             &PageSettings{Size: "letter", Orientation: "portrait", Margin: 0},
			hasFooter:        false,
			wantW:            8.5,
			wantH:            11.0,
			wantMargin:       DefaultMargin,
			wantBottomMargin: DefaultMargin,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			w, h, margin, bottomMargin := resolvePageDimensions(tt.page, tt.hasFooter)

			if w != tt.wantW {
				t.Errorf("resolvePageDimensions(%v, %v) width = %v, want %v", tt.page, tt.hasFooter, w, tt.wantW)
			}
			if h != tt.wantH {
				t.Errorf("resolvePageDimensions(%v, %v) height = %v, want %v", tt.page, tt.hasFooter, h, tt.wantH)
			}
			if margin != tt.wantMargin {
				t.Errorf("resolvePageDimensions(%v, %v) margin = %v, want %v", tt.page, tt.hasFooter, margin, tt.wantMargin)
			}
			if bottomMargin != tt.wantBottomMargin {
				t.Errorf("resolvePageDimensions(%v, %v) bottomMargin = %v, want %v", tt.page, tt.hasFooter, bottomMargin, tt.wantBottomMargin)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestRodRenderer_Close_Idempotent - Close Idempotency
// ---------------------------------------------------------------------------

func TestRodRenderer_Close_Idempotent(t *testing.T) {
	t.Parallel()

	renderer := newRodRenderer(defaultTimeout)

	// Multiple calls should not panic and all should succeed
	err1 := renderer.Close()
	err2 := renderer.Close()
	err3 := renderer.Close()

	if err1 != nil {
		t.Errorf("renderer.Close() first call error = %v, want nil", err1)
	}
	if err2 != nil {
		t.Errorf("renderer.Close() second call error = %v, want nil", err2)
	}
	if err3 != nil {
		t.Errorf("renderer.Close() third call error = %v, want nil", err3)
	}
}

// ---------------------------------------------------------------------------
// TestRodConverter_Close_NilRenderer - Close with Nil Renderer
// ---------------------------------------------------------------------------

func TestRodConverter_Close_NilRenderer(t *testing.T) {
	t.Parallel()

	converter := &rodConverter{renderer: nil}

	err := converter.Close()
	if err != nil {
		t.Errorf("converter.Close() with nil renderer = %v, want nil", err)
	}
}

// ---------------------------------------------------------------------------
// TestBuildPDFOptions - PDF Options Construction
// ---------------------------------------------------------------------------

func TestBuildPDFOptions(t *testing.T) {
	t.Parallel()

	renderer := &rodRenderer{timeout: defaultTimeout}

	t.Run("nil opts uses default margins", func(t *testing.T) {
		t.Parallel()

		pdfOpts := renderer.buildPDFOptions(nil)

		if *pdfOpts.MarginBottom != DefaultMargin {
			t.Errorf("buildPDFOptions(nil).MarginBottom = %v, want %v", *pdfOpts.MarginBottom, DefaultMargin)
		}
		if pdfOpts.DisplayHeaderFooter {
			t.Errorf("buildPDFOptions(nil).DisplayHeaderFooter = true, want false")
		}
	})

	t.Run("with footer increases bottom margin", func(t *testing.T) {
		t.Parallel()

		opts := &pdfOptions{Footer: &pipeline.FooterData{Text: "Footer"}}
		pdfOpts := renderer.buildPDFOptions(opts)

		expectedMargin := DefaultMargin + footerMarginExtra
		if *pdfOpts.MarginBottom != expectedMargin {
			t.Errorf("buildPDFOptions(opts).MarginBottom = %v, want %v", *pdfOpts.MarginBottom, expectedMargin)
		}
		if !pdfOpts.DisplayHeaderFooter {
			t.Errorf("buildPDFOptions(opts).DisplayHeaderFooter = false, want true")
		}
	})

	t.Run("with page settings uses custom dimensions", func(t *testing.T) {
		t.Parallel()

		opts := &pdfOptions{
			Page: &PageSettings{Size: "a4", Orientation: "landscape", Margin: 1.0},
		}
		pdfOpts := renderer.buildPDFOptions(opts)

		if *pdfOpts.PaperWidth != 11.69 {
			t.Errorf("buildPDFOptions(opts).PaperWidth = %v, want 11.69", *pdfOpts.PaperWidth)
		}
		if *pdfOpts.PaperHeight != 8.27 {
			t.Errorf("buildPDFOptions(opts).PaperHeight = %v, want 8.27", *pdfOpts.PaperHeight)
		}
		if *pdfOpts.MarginTop != 1.0 {
			t.Errorf("buildPDFOptions(opts).MarginTop = %v, want 1.0", *pdfOpts.MarginTop)
		}
		if *pdfOpts.MarginBottom != 1.0 {
			t.Errorf("buildPDFOptions(opts).MarginBottom = %v, want 1.0", *pdfOpts.MarginBottom)
		}
	})

	t.Run("with page settings and footer", func(t *testing.T) {
		t.Parallel()

		opts := &pdfOptions{
			Page:   &PageSettings{Size: "letter", Orientation: "portrait", Margin: 0.75},
			Footer: &pipeline.FooterData{Text: "Footer"},
		}
		pdfOpts := renderer.buildPDFOptions(opts)

		if *pdfOpts.MarginTop != 0.75 {
			t.Errorf("buildPDFOptions(opts).MarginTop = %v, want 0.75", *pdfOpts.MarginTop)
		}
		expectedBottom := 0.75 + footerMarginExtra
		if *pdfOpts.MarginBottom != expectedBottom {
			t.Errorf("buildPDFOptions(opts).MarginBottom = %v, want %v", *pdfOpts.MarginBottom, expectedBottom)
		}
		if !pdfOpts.DisplayHeaderFooter {
			t.Errorf("buildPDFOptions(opts).DisplayHeaderFooter = false, want true")
		}
	})
}

// ---------------------------------------------------------------------------
// TestPageDimensions_AllSizesPresent - Page Dimensions Map Completeness
// ---------------------------------------------------------------------------

func TestPageDimensions_AllSizesPresent(t *testing.T) {
	t.Parallel()

	requiredSizes := []string{PageSizeLetter, PageSizeA4, PageSizeLegal}

	for _, size := range requiredSizes {
		if _, ok := pageDimensions[size]; !ok {
			t.Errorf("pageDimensions[%q] missing entry, want present", size)
		}
	}
}

// ---------------------------------------------------------------------------
// TestPageDimensions_ValidValues - Page Dimensions Value Validity
// ---------------------------------------------------------------------------

func TestPageDimensions_ValidValues(t *testing.T) {
	t.Parallel()

	for size, dims := range pageDimensions {
		t.Run(size, func(t *testing.T) {
			t.Parallel()

			if dims.width <= 0 {
				t.Errorf("pageDimensions[%q].width = %v, want > 0", size, dims.width)
			}
			if dims.height <= 0 {
				t.Errorf("pageDimensions[%q].height = %v, want > 0", size, dims.height)
			}
			// Portrait dimensions: height > width
			if dims.height <= dims.width {
				t.Errorf("pageDimensions[%q] = (%v x %v), want height > width", size, dims.width, dims.height)
			}
		})
	}
}
