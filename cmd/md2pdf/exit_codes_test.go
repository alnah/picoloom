package main

// Notes:
// - exitCodeFor: we test all sentinel errors from md2pdf and config packages,
//   plus wrapped errors to verify errors.Is() chain works correctly.
// - Exit code constants: we verify Unix conventions (0=success, 1=general, 2=usage)
//   and custom codes are below 126.
// These are acceptable gaps: we test observable behavior, not implementation details.

import (
	"errors"
	"fmt"
	"os"
	"testing"

	md2pdf "github.com/alnah/go-md2pdf"
	"github.com/alnah/go-md2pdf/internal/config"
)

// ---------------------------------------------------------------------------
// TestExitCodeFor - Error to exit code mapping
// ---------------------------------------------------------------------------

func TestExitCodeFor(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		err  error
		want int
	}{
		// Success
		{"returns success for nil error", nil, ExitSuccess},

		// Browser errors (exit 4)
		{"returns browser exit code for browser connect error", md2pdf.ErrBrowserConnect, ExitBrowser},
		{"returns browser exit code for page create error", md2pdf.ErrPageCreate, ExitBrowser},
		{"returns browser exit code for page load error", md2pdf.ErrPageLoad, ExitBrowser},
		{"returns browser exit code for pdf generation error", md2pdf.ErrPDFGeneration, ExitBrowser},
		{"returns browser exit code for wrapped browser connect error", fmt.Errorf("failed: %w", md2pdf.ErrBrowserConnect), ExitBrowser},

		// I/O errors (exit 3)
		{"returns io exit code for file not exist error", os.ErrNotExist, ExitIO},
		{"returns io exit code for permission denied error", os.ErrPermission, ExitIO},
		{"returns io exit code for read markdown error", ErrReadMarkdown, ExitIO},
		{"returns io exit code for read css error", ErrReadCSS, ExitIO},
		{"returns io exit code for write pdf error", ErrWritePDF, ExitIO},
		{"returns io exit code for no input error", ErrNoInput, ExitIO},
		{"returns io exit code for wrapped file not exist error", fmt.Errorf("reading: %w", os.ErrNotExist), ExitIO},

		// Usage/config/validation errors (exit 2)
		{"returns usage exit code for config not found error", config.ErrConfigNotFound, ExitUsage},
		{"returns usage exit code for config parse error", config.ErrConfigParse, ExitUsage},
		{"returns usage exit code for field too long error", config.ErrFieldTooLong, ExitUsage},
		{"returns usage exit code for empty markdown error", md2pdf.ErrEmptyMarkdown, ExitUsage},
		{"returns usage exit code for invalid page size error", md2pdf.ErrInvalidPageSize, ExitUsage},
		{"returns usage exit code for invalid orientation error", md2pdf.ErrInvalidOrientation, ExitUsage},
		{"returns usage exit code for invalid margin error", md2pdf.ErrInvalidMargin, ExitUsage},
		{"returns usage exit code for invalid footer position error", md2pdf.ErrInvalidFooterPosition, ExitUsage},
		{"returns usage exit code for invalid watermark color error", md2pdf.ErrInvalidWatermarkColor, ExitUsage},
		{"returns usage exit code for invalid toc depth error", md2pdf.ErrInvalidTOCDepth, ExitUsage},
		{"returns usage exit code for invalid orphans error", md2pdf.ErrInvalidOrphans, ExitUsage},
		{"returns usage exit code for invalid widows error", md2pdf.ErrInvalidWidows, ExitUsage},
		{"returns usage exit code for style not found error", md2pdf.ErrStyleNotFound, ExitUsage},
		{"returns usage exit code for template set not found error", md2pdf.ErrTemplateSetNotFound, ExitUsage},
		{"returns usage exit code for incomplete template set error", md2pdf.ErrIncompleteTemplateSet, ExitUsage},
		{"returns usage exit code for invalid asset path error", md2pdf.ErrInvalidAssetPath, ExitUsage},
		{"returns usage exit code for unsupported shell error", ErrUnsupportedShell, ExitUsage},
		{"returns usage exit code for wrapped config parse error", fmt.Errorf("loading: %w", config.ErrConfigParse), ExitUsage},

		// General errors (exit 1)
		{"returns general exit code for unknown error", errors.New("something unexpected"), ExitGeneral},
		{"returns general exit code for wrapped unknown error", fmt.Errorf("context: %w", errors.New("unknown")), ExitGeneral},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := exitCodeFor(tt.err)
			if got != tt.want {
				t.Errorf("exitCodeFor(%v) = %d, want %d", tt.err, got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestExitCodeConstants - Unix convention compliance
// ---------------------------------------------------------------------------

func TestExitCodeConstants(t *testing.T) {
	t.Parallel()
	// Verify exit codes follow Unix conventions
	if ExitSuccess != 0 {
		t.Errorf("ExitSuccess = %d, want 0", ExitSuccess)
	}
	if ExitGeneral != 1 {
		t.Errorf("ExitGeneral = %d, want 1", ExitGeneral)
	}
	if ExitUsage != 2 {
		t.Errorf("ExitUsage = %d, want 2", ExitUsage)
	}

	// Verify custom codes are below 126 (Unix convention)
	if ExitIO >= 126 {
		t.Errorf("ExitIO = %d, should be < 126", ExitIO)
	}
	if ExitBrowser >= 126 {
		t.Errorf("ExitBrowser = %d, should be < 126", ExitBrowser)
	}
}
