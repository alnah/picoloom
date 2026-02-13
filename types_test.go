package md2pdf

// Notes:
// - PageSettings: tests validation for size, orientation, and margin boundaries
// - Footer: tests position validation (left, center, right)
// - Cover: tests logo path validation (URL vs file path)
// - Watermark: tests hex color validation
// - PageBreaks: tests orphans/widows range validation
// - TOC: tests depth range validation
// - Signature: tests image path validation (URL vs file path)

import (
	"errors"
	"os"
	"strings"
	"testing"
	"time"
)

// ---------------------------------------------------------------------------
// TestPageSettings_Validate - PageSettings Validation
// ---------------------------------------------------------------------------

func TestPageSettings_Validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		ps      *PageSettings
		wantErr error
	}{
		{
			name:    "nil is valid (use defaults)",
			ps:      nil,
			wantErr: nil,
		},
		{
			name: "valid letter portrait",
			ps: &PageSettings{
				Size:        PageSizeLetter,
				Orientation: OrientationPortrait,
				Margin:      DefaultMargin,
			},
			wantErr: nil,
		},
		{
			name: "valid a4 landscape",
			ps: &PageSettings{
				Size:        PageSizeA4,
				Orientation: OrientationLandscape,
				Margin:      1.0,
			},
			wantErr: nil,
		},
		{
			name: "valid legal portrait",
			ps: &PageSettings{
				Size:        PageSizeLegal,
				Orientation: OrientationPortrait,
				Margin:      MinMargin,
			},
			wantErr: nil,
		},
		{
			name: "case insensitive size",
			ps: &PageSettings{
				Size:        "A4",
				Orientation: OrientationPortrait,
				Margin:      DefaultMargin,
			},
			wantErr: nil,
		},
		{
			name: "case insensitive orientation",
			ps: &PageSettings{
				Size:        PageSizeLetter,
				Orientation: "LANDSCAPE",
				Margin:      DefaultMargin,
			},
			wantErr: nil,
		},
		{
			name: "margin at minimum",
			ps: &PageSettings{
				Size:        PageSizeLetter,
				Orientation: OrientationPortrait,
				Margin:      MinMargin,
			},
			wantErr: nil,
		},
		{
			name: "margin at maximum",
			ps: &PageSettings{
				Size:        PageSizeLetter,
				Orientation: OrientationPortrait,
				Margin:      MaxMargin,
			},
			wantErr: nil,
		},
		{
			name: "invalid page size",
			ps: &PageSettings{
				Size:        "tabloid",
				Orientation: OrientationPortrait,
				Margin:      DefaultMargin,
			},
			wantErr: ErrInvalidPageSize,
		},
		{
			name: "empty page size valid (uses default)",
			ps: &PageSettings{
				Size:        "",
				Orientation: OrientationPortrait,
				Margin:      DefaultMargin,
			},
			wantErr: nil,
		},
		{
			name: "invalid orientation",
			ps: &PageSettings{
				Size:        PageSizeLetter,
				Orientation: "diagonal",
				Margin:      DefaultMargin,
			},
			wantErr: ErrInvalidOrientation,
		},
		{
			name: "empty orientation valid (uses default)",
			ps: &PageSettings{
				Size:        PageSizeLetter,
				Orientation: "",
				Margin:      DefaultMargin,
			},
			wantErr: nil,
		},
		{
			name: "margin below minimum",
			ps: &PageSettings{
				Size:        PageSizeLetter,
				Orientation: OrientationPortrait,
				Margin:      0.1,
			},
			wantErr: ErrInvalidMargin,
		},
		{
			name: "margin above maximum",
			ps: &PageSettings{
				Size:        PageSizeLetter,
				Orientation: OrientationPortrait,
				Margin:      5.0,
			},
			wantErr: ErrInvalidMargin,
		},
		{
			name: "margin zero valid (uses default)",
			ps: &PageSettings{
				Size:        PageSizeLetter,
				Orientation: OrientationPortrait,
				Margin:      0,
			},
			wantErr: nil,
		},
		{
			name: "margin negative",
			ps: &PageSettings{
				Size:        PageSizeLetter,
				Orientation: OrientationPortrait,
				Margin:      -1.0,
			},
			wantErr: ErrInvalidMargin,
		},
		{
			name: "all empty values valid (all use defaults)",
			ps: &PageSettings{
				Size:        "",
				Orientation: "",
				Margin:      0,
			},
			wantErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := tt.ps.Validate()

			if tt.wantErr != nil {
				if err == nil {
					t.Fatalf("Validate() error = nil, want error")
				}
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("Validate() error = %v, want %v", err, tt.wantErr)
				}
				return
			}

			if err != nil {
				t.Errorf("Validate() unexpected error: %v", err)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestDefaultPageSettings - Default PageSettings Values
// ---------------------------------------------------------------------------

func TestDefaultPageSettings(t *testing.T) {
	t.Parallel()

	ps := DefaultPageSettings()

	if ps.Size != PageSizeLetter {
		t.Errorf("Size = %q, want %q", ps.Size, PageSizeLetter)
	}
	if ps.Orientation != OrientationPortrait {
		t.Errorf("Orientation = %q, want %q", ps.Orientation, OrientationPortrait)
	}
	if ps.Margin != DefaultMargin {
		t.Errorf("Margin = %v, want %v", ps.Margin, DefaultMargin)
	}

	// Ensure defaults are valid
	if err := ps.Validate(); err != nil {
		t.Errorf("DefaultPageSettings() not valid: %v", err)
	}
}

// ---------------------------------------------------------------------------
// TestIsValidPageSize - Page Size Validation
// ---------------------------------------------------------------------------

func TestIsValidPageSize(t *testing.T) {
	t.Parallel()

	tests := []struct {
		size string
		want bool
	}{
		{"letter", true},
		{"a4", true},
		{"legal", true},
		{"LETTER", true},
		{"A4", true},
		{"Letter", true},
		{"tabloid", false},
		{"", false},
		{"a5", false},
	}

	for _, tt := range tests {
		t.Run(tt.size, func(t *testing.T) {
			t.Parallel()

			got := isValidPageSize(tt.size)
			if got != tt.want {
				t.Errorf("isValidPageSize(%q) = %v, want %v", tt.size, got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestIsValidOrientation - Orientation Validation
// ---------------------------------------------------------------------------

func TestIsValidOrientation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		orientation string
		want        bool
	}{
		{"portrait", true},
		{"landscape", true},
		{"PORTRAIT", true},
		{"LANDSCAPE", true},
		{"Portrait", true},
		{"diagonal", false},
		{"", false},
		{"auto", false},
	}

	for _, tt := range tests {
		t.Run(tt.orientation, func(t *testing.T) {
			t.Parallel()

			got := isValidOrientation(tt.orientation)
			if got != tt.want {
				t.Errorf("isValidOrientation(%q) = %v, want %v", tt.orientation, got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestFooter_Validate - Footer Position Validation
// ---------------------------------------------------------------------------

func TestFooter_Validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		footer  *Footer
		wantErr error
	}{
		{
			name:    "nil is valid",
			footer:  nil,
			wantErr: nil,
		},
		{
			name:    "empty position is valid",
			footer:  &Footer{Position: ""},
			wantErr: nil,
		},
		{
			name:    "left position is valid",
			footer:  &Footer{Position: "left"},
			wantErr: nil,
		},
		{
			name:    "center position is valid",
			footer:  &Footer{Position: "center"},
			wantErr: nil,
		},
		{
			name:    "right position is valid",
			footer:  &Footer{Position: "right"},
			wantErr: nil,
		},
		{
			name:    "case insensitive LEFT",
			footer:  &Footer{Position: "LEFT"},
			wantErr: nil,
		},
		{
			name:    "case insensitive Center",
			footer:  &Footer{Position: "Center"},
			wantErr: nil,
		},
		{
			name:    "invalid position returns error",
			footer:  &Footer{Position: "top"},
			wantErr: ErrInvalidFooterPosition,
		},
		{
			name:    "invalid position middle",
			footer:  &Footer{Position: "middle"},
			wantErr: ErrInvalidFooterPosition,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := tt.footer.Validate()

			if tt.wantErr != nil {
				if err == nil {
					t.Fatalf("Validate() error = nil, want error")
				}
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("Validate() error = %v, want %v", err, tt.wantErr)
				}
				return
			}

			if err != nil {
				t.Errorf("Validate() unexpected error: %v", err)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestWithTimeout_panic - WithTimeout Panic Behavior
// ---------------------------------------------------------------------------

func TestWithTimeout_panic(t *testing.T) {
	t.Parallel()

	t.Run("zero duration", func(t *testing.T) {
		t.Parallel()
		defer func() {
			if r := recover(); r == nil {
				t.Error("WithTimeout(0) did not panic, want panic")
			}
		}()
		WithTimeout(0)
	})

	t.Run("negative duration", func(t *testing.T) {
		t.Parallel()

		defer func() {
			if r := recover(); r == nil {
				t.Error("WithTimeout(-1 * time.Second) did not panic, want panic")
			}
		}()
		WithTimeout(-1 * time.Second)
	})
}

// ---------------------------------------------------------------------------
// TestIsValidHexColor - Hex Color Validation
// ---------------------------------------------------------------------------

func TestIsValidHexColor(t *testing.T) {
	t.Parallel()

	tests := []struct {
		color string
		want  bool
	}{
		// Valid colors
		{"#fff", true},
		{"#FFF", true},
		{"#000", true},
		{"#abc", true},
		{"#ABC", true},
		{"#123", true},
		{"#ffffff", true},
		{"#FFFFFF", true},
		{"#000000", true},
		{"#abcdef", true},
		{"#ABCDEF", true},
		{"#123456", true},
		{"#aAbBcC", true},
		{"#888888", true},
		{"#ff0000", true},

		// Invalid colors
		{"", false},
		{"fff", false},          // missing #
		{"#ff", false},          // too short
		{"#ffff", false},        // wrong length (4)
		{"#fffff", false},       // wrong length (5)
		{"#fffffff", false},     // too long (7)
		{"#ggg", false},         // invalid hex char
		{"#xyz", false},         // invalid hex char
		{"#12345g", false},      // invalid hex char
		{"red", false},          // color name not supported
		{"rgb(255,0,0)", false}, // rgb not supported
		{"#", false},            // just hash
		{" #fff", false},        // leading space
		{"#fff ", false},        // trailing space
	}

	for _, tt := range tests {
		t.Run(tt.color, func(t *testing.T) {
			t.Parallel()

			got := isValidHexColor(tt.color)
			if got != tt.want {
				t.Errorf("isValidHexColor(%q) = %v, want %v", tt.color, got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestCover_Validate - Cover Logo Path Validation
// ---------------------------------------------------------------------------

func TestCover_Validate(t *testing.T) {
	t.Parallel()

	// Create a temp file for logo path tests
	tempDir := t.TempDir()
	existingLogo := tempDir + "/logo.png"
	if err := os.WriteFile(existingLogo, []byte("fake png"), 0644); err != nil {
		t.Fatalf("failed to create test logo: %v", err)
	}

	tests := []struct {
		name    string
		cover   *Cover
		wantErr error
	}{
		{
			name:    "nil is valid",
			cover:   nil,
			wantErr: nil,
		},
		{
			name:    "empty cover is valid",
			cover:   &Cover{},
			wantErr: nil,
		},
		{
			name: "all fields populated is valid",
			cover: &Cover{
				Title:        "My Document",
				Subtitle:     "A Comprehensive Guide",
				Logo:         existingLogo,
				Author:       "John Doe",
				AuthorTitle:  "Senior Developer",
				Organization: "Acme Corp",
				Date:         "2025-01-01",
				Version:      "v1.0.0",
			},
			wantErr: nil,
		},
		{
			name:    "logo URL accepted without file validation",
			cover:   &Cover{Logo: "https://example.com/logo.png"},
			wantErr: nil,
		},
		{
			name:    "logo http URL accepted",
			cover:   &Cover{Logo: "http://example.com/logo.png"},
			wantErr: nil,
		},
		{
			name:    "logo empty is valid",
			cover:   &Cover{Logo: ""},
			wantErr: nil,
		},
		{
			name:    "existing logo path is valid",
			cover:   &Cover{Logo: existingLogo},
			wantErr: nil,
		},
		{
			name:    "nonexistent logo path returns error",
			cover:   &Cover{Logo: "/nonexistent/path/to/logo.png"},
			wantErr: ErrCoverLogoNotFound,
		},
		{
			name:    "relative nonexistent logo returns error",
			cover:   &Cover{Logo: "nonexistent.png"},
			wantErr: ErrCoverLogoNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := tt.cover.Validate()

			if tt.wantErr != nil {
				if err == nil {
					t.Fatalf("Validate() error = nil, want error")
				}
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("Validate() error = %v, want %v", err, tt.wantErr)
				}
				return
			}

			if err != nil {
				t.Errorf("Validate() unexpected error: %v", err)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestWatermark_Validate - Watermark Color Validation
// ---------------------------------------------------------------------------

func TestWatermark_Validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		watermark *Watermark
		wantErr   error
	}{
		{
			name:      "nil is valid",
			watermark: nil,
			wantErr:   nil,
		},
		{
			name:      "empty color is valid (uses default)",
			watermark: &Watermark{Text: "DRAFT", Color: ""},
			wantErr:   nil,
		},
		{
			name:      "valid 3-char hex color",
			watermark: &Watermark{Text: "DRAFT", Color: "#fff"},
			wantErr:   nil,
		},
		{
			name:      "valid 6-char hex color",
			watermark: &Watermark{Text: "DRAFT", Color: "#888888"},
			wantErr:   nil,
		},
		{
			name:      "valid uppercase hex color",
			watermark: &Watermark{Text: "DRAFT", Color: "#AABBCC"},
			wantErr:   nil,
		},
		{
			name:      "valid mixed case hex color",
			watermark: &Watermark{Text: "DRAFT", Color: "#aAbBcC"},
			wantErr:   nil,
		},
		{
			name:      "invalid color - missing hash",
			watermark: &Watermark{Text: "DRAFT", Color: "888888"},
			wantErr:   ErrInvalidWatermarkColor,
		},
		{
			name:      "invalid color - wrong length",
			watermark: &Watermark{Text: "DRAFT", Color: "#8888"},
			wantErr:   ErrInvalidWatermarkColor,
		},
		{
			name:      "invalid color - invalid hex char",
			watermark: &Watermark{Text: "DRAFT", Color: "#gggggg"},
			wantErr:   ErrInvalidWatermarkColor,
		},
		{
			name:      "invalid color - color name",
			watermark: &Watermark{Text: "DRAFT", Color: "red"},
			wantErr:   ErrInvalidWatermarkColor,
		},
		{
			name:      "invalid color - rgb format",
			watermark: &Watermark{Text: "DRAFT", Color: "rgb(255,0,0)"},
			wantErr:   ErrInvalidWatermarkColor,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := tt.watermark.Validate()

			if tt.wantErr != nil {
				if err == nil {
					t.Fatalf("Validate() error = nil, want error")
				}
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("Validate() error = %v, want %v", err, tt.wantErr)
				}
				return
			}

			if err != nil {
				t.Errorf("Validate() unexpected error: %v", err)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestPageBreaks_Validate - PageBreaks Orphans/Widows Validation
// ---------------------------------------------------------------------------

func TestPageBreaks_Validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		pageBreaks *PageBreaks
		wantErr    error
	}{
		{
			name:       "nil is valid",
			pageBreaks: nil,
			wantErr:    nil,
		},
		{
			name:       "empty struct is valid (uses defaults)",
			pageBreaks: &PageBreaks{},
			wantErr:    nil,
		},
		{
			name:       "orphans 0 is valid (means use default)",
			pageBreaks: &PageBreaks{Orphans: 0},
			wantErr:    nil,
		},
		{
			name:       "widows 0 is valid (means use default)",
			pageBreaks: &PageBreaks{Widows: 0},
			wantErr:    nil,
		},
		{
			name:       "valid orphans at minimum",
			pageBreaks: &PageBreaks{Orphans: MinOrphans},
			wantErr:    nil,
		},
		{
			name:       "valid orphans at maximum",
			pageBreaks: &PageBreaks{Orphans: MaxOrphans},
			wantErr:    nil,
		},
		{
			name:       "valid widows at minimum",
			pageBreaks: &PageBreaks{Widows: MinWidows},
			wantErr:    nil,
		},
		{
			name:       "valid widows at maximum",
			pageBreaks: &PageBreaks{Widows: MaxWidows},
			wantErr:    nil,
		},
		{
			name:       "valid orphans and widows mid range",
			pageBreaks: &PageBreaks{Orphans: 3, Widows: 3},
			wantErr:    nil,
		},
		{
			name:       "valid with all heading breaks enabled",
			pageBreaks: &PageBreaks{BeforeH1: true, BeforeH2: true, BeforeH3: true, Orphans: 2, Widows: 2},
			wantErr:    nil,
		},
		{
			name:       "invalid orphans below minimum",
			pageBreaks: &PageBreaks{Orphans: -1},
			wantErr:    ErrInvalidOrphans,
		},
		{
			name:       "invalid orphans above maximum",
			pageBreaks: &PageBreaks{Orphans: MaxOrphans + 1},
			wantErr:    ErrInvalidOrphans,
		},
		{
			name:       "invalid orphans large value",
			pageBreaks: &PageBreaks{Orphans: 100},
			wantErr:    ErrInvalidOrphans,
		},
		{
			name:       "invalid widows below minimum",
			pageBreaks: &PageBreaks{Widows: -1},
			wantErr:    ErrInvalidWidows,
		},
		{
			name:       "invalid widows above maximum",
			pageBreaks: &PageBreaks{Widows: MaxWidows + 1},
			wantErr:    ErrInvalidWidows,
		},
		{
			name:       "invalid widows large value",
			pageBreaks: &PageBreaks{Widows: 100},
			wantErr:    ErrInvalidWidows,
		},
		{
			name:       "orphans validated before widows",
			pageBreaks: &PageBreaks{Orphans: -1, Widows: -1},
			wantErr:    ErrInvalidOrphans,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := tt.pageBreaks.Validate()

			if tt.wantErr != nil {
				if err == nil {
					t.Fatalf("Validate() error = nil, want error")
				}
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("Validate() error = %v, want %v", err, tt.wantErr)
				}
				return
			}

			if err != nil {
				t.Errorf("Validate() unexpected error: %v", err)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestTOC_Validate - TOC Depth Validation
// ---------------------------------------------------------------------------

func TestTOC_Validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		toc     *TOC
		wantErr error
	}{
		{
			name:    "nil is valid",
			toc:     nil,
			wantErr: nil,
		},
		{
			name:    "valid depth 1",
			toc:     &TOC{MaxDepth: 1},
			wantErr: nil,
		},
		{
			name:    "valid depth 3",
			toc:     &TOC{MaxDepth: 3},
			wantErr: nil,
		},
		{
			name:    "valid depth 6",
			toc:     &TOC{MaxDepth: 6},
			wantErr: nil,
		},
		{
			name:    "with title",
			toc:     &TOC{Title: "Table of Contents", MaxDepth: 3},
			wantErr: nil,
		},
		{
			name:    "min depth boundary",
			toc:     &TOC{MaxDepth: 1}, // minTOCDepth
			wantErr: nil,
		},
		{
			name:    "max depth boundary",
			toc:     &TOC{MaxDepth: 6}, // maxTOCDepth
			wantErr: nil,
		},
		{
			name:    "depth 0 valid (means use default)",
			toc:     &TOC{MaxDepth: 0},
			wantErr: nil,
		},
		{
			name:    "maxDepth 7 invalid",
			toc:     &TOC{MaxDepth: 7},
			wantErr: ErrInvalidTOCDepth,
		},
		{
			name:    "maxDepth negative invalid",
			toc:     &TOC{MaxDepth: -1},
			wantErr: ErrInvalidTOCDepth,
		},
		{
			name:    "maxDepth large negative invalid",
			toc:     &TOC{MaxDepth: -100},
			wantErr: ErrInvalidTOCDepth,
		},
		{
			name:    "maxDepth large positive invalid",
			toc:     &TOC{MaxDepth: 100},
			wantErr: ErrInvalidTOCDepth,
		},
		{
			name:    "minDepth valid",
			toc:     &TOC{MinDepth: 2, MaxDepth: 3},
			wantErr: nil,
		},
		{
			name:    "minDepth 0 valid (means use default)",
			toc:     &TOC{MinDepth: 0, MaxDepth: 3},
			wantErr: nil,
		},
		{
			name:    "minDepth 7 invalid",
			toc:     &TOC{MinDepth: 7},
			wantErr: ErrInvalidTOCDepth,
		},
		{
			name:    "minDepth negative invalid",
			toc:     &TOC{MinDepth: -1},
			wantErr: ErrInvalidTOCDepth,
		},
		{
			name:    "minDepth greater than maxDepth invalid",
			toc:     &TOC{MinDepth: 4, MaxDepth: 2},
			wantErr: ErrInvalidTOCDepth,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := tt.toc.Validate()

			if tt.wantErr != nil {
				if err == nil {
					t.Fatalf("Validate() error = nil, want error")
				}
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("Validate() error = %v, want %v", err, tt.wantErr)
				}
				return
			}

			if err != nil {
				t.Errorf("Validate() unexpected error: %v", err)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestSignature_Validate - Signature ImagePath Validation
// ---------------------------------------------------------------------------

func TestSignature_Validate(t *testing.T) {
	t.Parallel()

	// Create a temp file for image path tests
	tempDir := t.TempDir()
	existingImage := tempDir + "/signature.png"
	if err := os.WriteFile(existingImage, []byte("fake png"), 0644); err != nil {
		t.Fatalf("failed to create test image: %v", err)
	}

	// Create a file with space in name
	imageWithSpace := tempDir + "/my signature.png"
	if err := os.WriteFile(imageWithSpace, []byte("fake png"), 0644); err != nil {
		t.Fatalf("failed to create test image with space: %v", err)
	}

	tests := []struct {
		name    string
		sig     *Signature
		wantErr error
	}{
		{
			name:    "nil is valid",
			sig:     nil,
			wantErr: nil,
		},
		{
			name:    "empty signature is valid",
			sig:     &Signature{},
			wantErr: nil,
		},
		{
			name:    "empty ImagePath is valid",
			sig:     &Signature{ImagePath: ""},
			wantErr: nil,
		},
		{
			name:    "HTTPS URL bypasses file check",
			sig:     &Signature{ImagePath: "https://example.com/signature.png"},
			wantErr: nil,
		},
		{
			name:    "HTTP URL bypasses file check",
			sig:     &Signature{ImagePath: "http://example.com/signature.png"},
			wantErr: nil,
		},
		{
			name:    "existing local file path is valid",
			sig:     &Signature{ImagePath: existingImage},
			wantErr: nil,
		},
		{
			name:    "nonexistent local file path returns error",
			sig:     &Signature{ImagePath: "/nonexistent/path/to/signature.png"},
			wantErr: ErrSignatureImageNotFound,
		},
		{
			name:    "relative nonexistent path returns error",
			sig:     &Signature{ImagePath: "nonexistent.png"},
			wantErr: ErrSignatureImageNotFound,
		},
		{
			name: "all fields populated is valid",
			sig: &Signature{
				Name:         "John Doe",
				Title:        "Senior Developer",
				Email:        "john@example.com",
				Organization: "Acme Corp",
				ImagePath:    existingImage,
				Links:        []Link{{Label: "GitHub", URL: "https://github.com/johndoe"}},
				Phone:        "+1-555-123-4567",
				Address:      "123 Main St",
				Department:   "Engineering",
			},
			wantErr: nil,
		},
		// Edge cases from SWOT analysis
		{
			name:    "whitespace-only path treated as nonexistent file",
			sig:     &Signature{ImagePath: "   "},
			wantErr: ErrSignatureImageNotFound,
		},
		{
			name:    "uppercase HTTP not recognized as URL",
			sig:     &Signature{ImagePath: "HTTP://example.com/img.png"},
			wantErr: ErrSignatureImageNotFound,
		},
		{
			name:    "ftp protocol not recognized as URL",
			sig:     &Signature{ImagePath: "ftp://example.com/img.png"},
			wantErr: ErrSignatureImageNotFound,
		},
		{
			name:    "path with spaces is valid if file exists",
			sig:     &Signature{ImagePath: imageWithSpace},
			wantErr: nil,
		},
		{
			name:    "other fields not validated - invalid email accepted",
			sig:     &Signature{Email: "not-an-email"},
			wantErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := tt.sig.Validate()

			if tt.wantErr != nil {
				if err == nil {
					t.Fatalf("Validate() error = nil, want error")
				}
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("Validate() error = %v, want %v", err, tt.wantErr)
				}
				return
			}

			if err != nil {
				t.Errorf("Validate() unexpected error: %v", err)
			}
		})
	}

	t.Run("error message includes path", func(t *testing.T) {
		badPath := "/nonexistent/path/to/signature.png"
		sig := &Signature{ImagePath: badPath}

		err := sig.Validate()
		if err == nil {
			t.Fatalf("Validate() error = nil, want error")
		}

		if !strings.Contains(err.Error(), badPath) {
			t.Errorf("Validate() error message = %q, want path %q included", err.Error(), badPath)
		}
	})
}
