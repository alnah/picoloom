package picoloom

import "errors"

// Sentinel errors for library operations.
var (
	ErrEmptyMarkdown   = errors.New("markdown content cannot be empty")
	ErrHTMLConversion  = errors.New("HTML conversion failed")
	ErrPDFGeneration   = errors.New("PDF generation failed")
	ErrBrowserConnect  = errors.New("failed to connect to browser")
	ErrPageCreate      = errors.New("failed to create browser page")
	ErrPageLoad        = errors.New("failed to load page")
	ErrSignatureRender = errors.New("signature template rendering failed")

	// Page settings validation errors.
	ErrInvalidPageSize    = errors.New("invalid page size")
	ErrInvalidOrientation = errors.New("invalid orientation")
	ErrInvalidMargin      = errors.New("invalid margin")

	// Footer validation errors.
	ErrInvalidFooterPosition = errors.New("invalid footer position")

	// Watermark validation errors.
	ErrInvalidWatermarkColor = errors.New("invalid watermark color")

	// Cover validation errors.
	ErrCoverLogoNotFound = errors.New("cover logo file not found")
	ErrCoverRender       = errors.New("cover template rendering failed")

	// Signature validation errors.
	ErrSignatureImageNotFound = errors.New("signature image file not found")

	// TOC validation errors.
	ErrInvalidTOCDepth = errors.New("invalid TOC depth")

	// Page breaks validation errors.
	ErrInvalidOrphans = errors.New("invalid orphans value")
	ErrInvalidWidows  = errors.New("invalid widows value")

	// Asset loading errors.
	ErrStyleNotFound         = errors.New("style not found")
	ErrTemplateSetNotFound   = errors.New("template set not found")
	ErrIncompleteTemplateSet = errors.New("template set missing required template")
	ErrInvalidAssetPath      = errors.New("invalid asset path")
)
