package main

import (
	"errors"
	"os"

	md2pdf "github.com/alnah/go-md2pdf"
	"github.com/alnah/go-md2pdf/internal/config"
)

// Exit codes for md2pdf CLI.
// Follows Unix conventions: 0=success, 1=general, 2=usage, and custom codes < 126.
const (
	ExitSuccess = 0 // Successful conversion
	ExitGeneral = 1 // General/unexpected error
	ExitUsage   = 2 // Invalid flags, config, or validation
	ExitIO      = 3 // File not found, permission denied
	ExitBrowser = 4 // Browser/Chrome errors
)

// exitCodeFor returns the appropriate exit code for an error.
// It uses errors.Is to check wrapped errors, so callers must use fmt.Errorf("%w", err).
func exitCodeFor(err error) int {
	if err == nil {
		return ExitSuccess
	}

	// Browser errors (exit 4)
	if errors.Is(err, md2pdf.ErrBrowserConnect) ||
		errors.Is(err, md2pdf.ErrPageCreate) ||
		errors.Is(err, md2pdf.ErrPageLoad) ||
		errors.Is(err, md2pdf.ErrPDFGeneration) {
		return ExitBrowser
	}

	// I/O errors (exit 3)
	if errors.Is(err, os.ErrNotExist) ||
		errors.Is(err, os.ErrPermission) ||
		errors.Is(err, ErrReadMarkdown) ||
		errors.Is(err, ErrReadCSS) ||
		errors.Is(err, ErrWritePDF) ||
		errors.Is(err, ErrNoInput) {
		return ExitIO
	}

	// Usage/config/validation errors (exit 2)
	if errors.Is(err, config.ErrConfigNotFound) ||
		errors.Is(err, config.ErrConfigParse) ||
		errors.Is(err, config.ErrFieldTooLong) ||
		errors.Is(err, ErrConfigCommandUsage) ||
		errors.Is(err, ErrConfigInitNeedsTTY) ||
		errors.Is(err, ErrConfigInitExists) ||
		errors.Is(err, md2pdf.ErrEmptyMarkdown) ||
		errors.Is(err, md2pdf.ErrInvalidPageSize) ||
		errors.Is(err, md2pdf.ErrInvalidOrientation) ||
		errors.Is(err, md2pdf.ErrInvalidMargin) ||
		errors.Is(err, md2pdf.ErrInvalidFooterPosition) ||
		errors.Is(err, md2pdf.ErrInvalidWatermarkColor) ||
		errors.Is(err, md2pdf.ErrInvalidTOCDepth) ||
		errors.Is(err, md2pdf.ErrInvalidOrphans) ||
		errors.Is(err, md2pdf.ErrInvalidWidows) ||
		errors.Is(err, md2pdf.ErrStyleNotFound) ||
		errors.Is(err, md2pdf.ErrTemplateSetNotFound) ||
		errors.Is(err, md2pdf.ErrIncompleteTemplateSet) ||
		errors.Is(err, md2pdf.ErrInvalidAssetPath) ||
		errors.Is(err, ErrUnsupportedShell) {
		return ExitUsage
	}

	return ExitGeneral
}
