package md2pdf

import (
	"context"
	"fmt"
	"html"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/alnah/go-md2pdf/internal/fileutil"
	"github.com/alnah/go-md2pdf/internal/hints"
	"github.com/alnah/go-md2pdf/internal/pipeline"
	"github.com/alnah/go-md2pdf/internal/process"
	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
)

// browserCloseTimeout is the maximum time to wait for browser.Close() before force-killing.
const browserCloseTimeout = 5 * time.Second

// pdfConverter abstracts HTML to PDF conversion to allow different backends.
type pdfConverter interface {
	ToPDF(ctx context.Context, htmlContent string, opts *pdfOptions) ([]byte, error)
	Close() error
}

// pdfRenderer abstracts PDF rendering from an HTML file to enable testing without a browser.
type pdfRenderer interface {
	RenderFromFile(ctx context.Context, filePath string, opts *pdfOptions) ([]byte, error)
}

// pdfOptions holds options for PDF generation.
type pdfOptions struct {
	Footer *pipeline.FooterData
	Page   *PageSettings
}

// footerMarginExtra is added to bottom margin when footer is active.
const footerMarginExtra = 0.25

// Page dimensions in inches (ISO/ANSI standards).
const (
	letterWidthInches  = 8.5
	letterHeightInches = 11.0
	a4WidthInches      = 8.27  // 210mm
	a4HeightInches     = 11.69 // 297mm
	legalWidthInches   = 8.5
	legalHeightInches  = 14.0
)

// Footer styling constants.
const (
	footerFontSize = "10px"
	footerColor    = "#aaa"
	footerPaddingH = "0.5in"
)

// pageDimensions maps page size to (width, height) in inches.
var pageDimensions = map[string]struct{ width, height float64 }{
	PageSizeLetter: {letterWidthInches, letterHeightInches},
	PageSizeA4:     {a4WidthInches, a4HeightInches},
	PageSizeLegal:  {legalWidthInches, legalHeightInches},
}

// rodRenderer implements pdfRenderer using go-rod.
// Rod automatically downloads Chromium on first run if not found.
type rodRenderer struct {
	browser   *rod.Browser
	launcher  *launcher.Launcher
	timeout   time.Duration
	closeOnce sync.Once
}

// newRodRenderer creates a rodRenderer with the given timeout.
func newRodRenderer(timeout time.Duration) *rodRenderer {
	return &rodRenderer{timeout: timeout}
}

// ensureBrowser lazily connects to the browser.
// Uses rod's managed Chromium (~/.cache/rod/browser/) for complete isolation
// from the user's Chrome installation. This prevents corruption of Chrome.app
// state that would require a system restart to fix.
func (r *rodRenderer) ensureBrowser(ctx context.Context) error {
	if r.browser != nil {
		return nil
	}
	if err := ctx.Err(); err != nil {
		return err
	}

	// Configure launcher
	// Leakless(false) prevents hanging on macOS - see github.com/go-rod/rod/issues/210
	// We compensate by explicitly calling Kill() and Cleanup() in Close().
	l := launcher.New().Context(ctx).Headless(true).Leakless(false).Set("disable-gpu")

	// Allow disabling sandbox for CI/Docker environments that lack kernel support.
	if os.Getenv("ROD_NO_SANDBOX") == "1" {
		l = l.NoSandbox(true)
	}

	// Optional: allow explicit browser override for CI/debugging.
	// DO NOT auto-detect system Chrome - it causes corruption issues.
	if bin := os.Getenv("ROD_BROWSER_BIN"); bin != "" {
		l = l.Bin(bin)
	}

	u, err := l.Launch()
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return ctxErr
		}
		return fmt.Errorf("%w: %v%s", ErrBrowserConnect, err, hints.ForBrowserConnect())
	}

	// Store launcher reference for cleanup in Close()
	r.launcher = l

	browser := rod.New().ControlURL(u).Context(ctx)
	if err := browser.Connect(); err != nil {
		r.launcher.Kill()
		r.launcher.Cleanup()
		r.browser = nil
		r.launcher = nil
		if ctxErr := ctx.Err(); ctxErr != nil {
			return ctxErr
		}
		return fmt.Errorf("%w: %v%s", ErrBrowserConnect, err, hints.ForBrowserConnect())
	}
	// Keep a neutral context on the shared browser handle; operations use per-call contexts.
	r.browser = browser.Context(context.Background())
	return nil
}

// Close releases browser resources.
// Safe to call multiple times (idempotent via sync.Once).
// Uses a timeout to avoid hanging indefinitely if browser.Close() blocks.
func (r *rodRenderer) Close() error {
	var closeErr error
	r.closeOnce.Do(func() {
		// Get PID before any cleanup - we'll need it to kill the process group
		var pid int
		if r.launcher != nil {
			pid = r.launcher.PID()
		}

		// Try graceful close first with timeout
		if r.browser != nil {
			done := make(chan error, 1)
			go func() {
				done <- r.browser.Close()
			}()

			// Use NewTimer instead of time.After to avoid timer leak.
			// time.After creates a timer that runs until expiration even if
			// the select chooses another case, leaking memory temporarily.
			timer := time.NewTimer(browserCloseTimeout)
			select {
			case closeErr = <-done:
				// Browser closed normally - stop timer to prevent leak
				timer.Stop()
			case <-timer.C:
				// Timeout - will be force-killed below
			}
			r.browser = nil
		}

		// Force kill the Chrome process group (kills all child processes too)
		if pid > 0 {
			// Kill the entire process group to ensure GPU, renderer,
			// and other Chrome child processes are terminated.
			process.KillProcessGroup(pid)
		}

		// Also call launcher.Kill() as fallback and cleanup user-data-dir
		if r.launcher != nil {
			r.launcher.Kill()
			r.launcher.Cleanup()
			r.launcher = nil
		}
	})
	return closeErr
}

// RenderFromFile opens a local HTML file in headless Chrome and renders it to PDF.
// Returns explicit errors instead of panicking when browser operations fail.
func (r *rodRenderer) RenderFromFile(ctx context.Context, filePath string, opts *pdfOptions) ([]byte, error) {
	// Check context before starting
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	if err := r.ensureBrowser(ctx); err != nil {
		return nil, err
	}

	page, err := r.browser.Page(proto.TargetCreateTarget{URL: "file://" + filePath})
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrPageCreate, err)
	}
	defer page.Close()

	renderCtx, cancel, err := renderOperationContext(ctx, r.timeout)
	if err != nil {
		return nil, err
	}
	defer cancel()
	pageWithCtx := page.Context(renderCtx)

	if err := pageWithCtx.WaitLoad(); err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return nil, ctxErr
		}
		return nil, fmt.Errorf("%w: %v%s", ErrPageLoad, err, hints.ForTimeout())
	}

	// Check context after page load
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	// Build PDF options
	pdfOpts := r.buildPDFOptions(opts)

	// Generate PDF
	reader, err := pageWithCtx.PDF(pdfOpts)
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return nil, ctxErr
		}
		return nil, fmt.Errorf("%w: %v", ErrPDFGeneration, err)
	}
	defer reader.Close()

	pdfBuf, err := io.ReadAll(reader)
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return nil, ctxErr
		}
		return nil, fmt.Errorf("%w: reading PDF stream: %v", ErrPDFGeneration, err)
	}

	return pdfBuf, nil
}

// renderOperationContext keeps browser operations tied to caller cancellation
// so Ctrl+C can stop long-running page work instead of waiting for fallback timeouts.
func renderOperationContext(ctx context.Context, fallbackTimeout time.Duration) (context.Context, context.CancelFunc, error) {
	if err := ctx.Err(); err != nil {
		return nil, nil, err
	}
	if _, hasDeadline := ctx.Deadline(); hasDeadline || fallbackTimeout <= 0 {
		return ctx, func() {}, nil
	}
	opCtx, cancel := context.WithTimeout(ctx, fallbackTimeout)
	return opCtx, cancel, nil
}

// resolvePageDimensions returns width, height, margin, and bottom margin.
// Applies defaults for nil/zero values, swaps for landscape, adds footer space.
func resolvePageDimensions(page *PageSettings, hasFooter bool) (w, h, margin, bottomMargin float64) {
	// Apply defaults
	size := PageSizeLetter
	orientation := OrientationPortrait
	margin = DefaultMargin

	if page != nil {
		if page.Size != "" {
			size = strings.ToLower(page.Size)
		}
		if page.Orientation != "" {
			orientation = strings.ToLower(page.Orientation)
		}
		if page.Margin > 0 {
			margin = page.Margin
		}
	}

	// Get dimensions for size
	dims, ok := pageDimensions[size]
	if !ok {
		dims = pageDimensions[PageSizeLetter] // fallback
	}
	w, h = dims.width, dims.height

	// Swap for landscape
	if orientation == OrientationLandscape {
		w, h = h, w
	}

	// Bottom margin: add extra space for footer
	bottomMargin = margin
	if hasFooter {
		bottomMargin = margin + footerMarginExtra
	}

	return w, h, margin, bottomMargin
}

// buildPDFOptions constructs proto.PagePrintToPDF with page settings and optional footer.
func (r *rodRenderer) buildPDFOptions(opts *pdfOptions) *proto.PagePrintToPDF {
	hasFooter := opts != nil && opts.Footer != nil
	var page *PageSettings
	if opts != nil {
		page = opts.Page
	}

	w, h, margin, bottomMargin := resolvePageDimensions(page, hasFooter)

	pdfOpts := &proto.PagePrintToPDF{
		PaperWidth:      toFloatPtr(w),
		PaperHeight:     toFloatPtr(h),
		MarginTop:       toFloatPtr(margin),
		MarginBottom:    toFloatPtr(bottomMargin),
		MarginLeft:      toFloatPtr(margin),
		MarginRight:     toFloatPtr(margin),
		PrintBackground: true,
	}

	if hasFooter {
		pdfOpts.DisplayHeaderFooter = true
		pdfOpts.HeaderTemplate = "<span></span>" // Empty header
		pdfOpts.FooterTemplate = buildFooterTemplate(opts.Footer)
	}

	return pdfOpts
}

// buildFooterTemplate generates an HTML template for Chrome's native footer.
// Supports pageNumber, totalPages, date placeholders via CSS classes.
func buildFooterTemplate(data *pipeline.FooterData) string {
	if data == nil {
		return "<span></span>"
	}

	var parts []string

	if data.ShowPageNumber {
		parts = append(parts, `<span class="pageNumber"></span>/<span class="totalPages"></span>`)
	}
	if data.Date != "" {
		parts = append(parts, html.EscapeString(data.Date))
	}
	if data.Status != "" {
		parts = append(parts, html.EscapeString(data.Status))
	}
	if data.DocumentID != "" {
		parts = append(parts, html.EscapeString(data.DocumentID))
	}
	if data.Text != "" {
		parts = append(parts, html.EscapeString(data.Text))
	}

	if len(parts) == 0 {
		return "<span></span>"
	}

	content := strings.Join(parts, " - ")

	// Position: left, center, or right (default)
	textAlign := "right"
	switch data.Position {
	case "left":
		textAlign = "left"
	case "center":
		textAlign = "center"
	}

	return fmt.Sprintf(`<div style="font-size: %s; font-family: %s; color: %s; width: 100%%; text-align: %s; padding: 0 %s;">%s</div>`,
		footerFontSize, defaultFontFamily, footerColor, textAlign, footerPaddingH, content)
}

// toFloatPtr returns a pointer to a float64 value.
func toFloatPtr(v float64) *float64 {
	return &v
}

// rodConverter converts HTML to PDF using headless Chrome via go-rod.
type rodConverter struct {
	renderer *rodRenderer
}

// newRodConverter creates a rodConverter with production renderer.
func newRodConverter(timeout time.Duration) *rodConverter {
	return &rodConverter{
		renderer: newRodRenderer(timeout),
	}
}

// ToPDF converts HTML content to PDF bytes using headless Chrome.
// Page dimensions are configured via opts.Page (defaults to US Letter, portrait, 0.5in margins).
func (c *rodConverter) ToPDF(ctx context.Context, htmlContent string, opts *pdfOptions) ([]byte, error) {
	tmpPath, cleanup, err := fileutil.WriteTempFile(htmlContent, "html")
	if err != nil {
		return nil, err
	}
	defer cleanup()

	return c.renderer.RenderFromFile(ctx, tmpPath, opts)
}

// Close releases browser resources.
func (c *rodConverter) Close() error {
	if c.renderer != nil {
		return c.renderer.Close()
	}
	return nil
}
