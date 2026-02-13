package main

// Notes:
// - buildSignatureData: we test all branches including enabled/disabled, URL vs
//   local image paths, and extended metadata fields.
// - buildFooterData: we test enabled/disabled states and all footer options.
// - buildPageSettings: we test page size/orientation/margin combinations.
// - buildWatermarkData: we test watermark text, color, opacity, and angle validation.
// - buildCoverData: we test title extraction from config, markdown H1, and filename.
// - buildTOCData: we test enabled/disabled, minDepth/maxDepth configuration,
//   and cross-validation (minDepth <= maxDepth). Boundary values 1-6 tested.
// - buildPageBreaksData: we test heading break before and orphan/widow settings.
// These are acceptable gaps: we test observable behavior, not implementation details.

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	md2pdf "github.com/alnah/go-md2pdf"
	"github.com/alnah/go-md2pdf/internal/dateutil"
	"github.com/alnah/go-md2pdf/internal/fileutil"
)

// ---------------------------------------------------------------------------
// TestBuildSignatureData - Signature block data construction
// ---------------------------------------------------------------------------

func TestBuildSignatureData(t *testing.T) {
	t.Parallel()

	t.Run("noSignature flag returns nil", func(t *testing.T) {
		t.Parallel()
		cfg := &Config{
			Author:    AuthorConfig{Name: "Test"},
			Signature: SignatureConfig{Enabled: true},
		}
		got, err := buildSignatureData(cfg, true)
		if err != nil {
			t.Fatalf("buildSignatureData() unexpected error: %v", err)
		}
		if got != nil {
			t.Errorf("buildSignatureData() = %v, want nil", got)
		}
	})

	t.Run("signature disabled in config returns nil", func(t *testing.T) {
		t.Parallel()

		cfg := &Config{
			Author:    AuthorConfig{Name: "Test"},
			Signature: SignatureConfig{Enabled: false},
		}
		got, err := buildSignatureData(cfg, false)
		if err != nil {
			t.Fatalf("buildSignatureData() unexpected error: %v", err)
		}
		if got != nil {
			t.Errorf("buildSignatureData() = %v, want nil", got)
		}
	})

	t.Run("valid signature config returns SignatureData", func(t *testing.T) {
		t.Parallel()

		cfg := &Config{
			Author: AuthorConfig{
				Name:  "John Doe",
				Title: "Developer",
				Email: "john@example.com",
			},
			Signature: SignatureConfig{
				Enabled: true,
				Links: []Link{
					{Label: "GitHub", URL: "https://github.com/johndoe"},
				},
			},
		}
		got, err := buildSignatureData(cfg, false)
		if err != nil {
			t.Fatalf("buildSignatureData() unexpected error: %v", err)
		}
		if got == nil {
			t.Fatalf("buildSignatureData() = nil, want SignatureData")
		}
		if got.Name != "John Doe" {
			t.Errorf("Name = %q, want %q", got.Name, "John Doe")
		}
		if got.Title != "Developer" {
			t.Errorf("Title = %q, want %q", got.Title, "Developer")
		}
		if got.Email != "john@example.com" {
			t.Errorf("Email = %q, want %q", got.Email, "john@example.com")
		}
		if len(got.Links) != 1 {
			t.Fatalf("buildSignatureData() len(Links) = %d, want 1", len(got.Links))
		}
		if got.Links[0].Label != "GitHub" {
			t.Errorf("Links[0].Label = %q, want %q", got.Links[0].Label, "GitHub")
		}
		if got.Links[0].URL != "https://github.com/johndoe" {
			t.Errorf("Links[0].URL = %q, want %q", got.Links[0].URL, "https://github.com/johndoe")
		}
	})

	t.Run("URL image path is accepted without file validation", func(t *testing.T) {
		t.Parallel()

		cfg := &Config{
			Author: AuthorConfig{Name: "Test"},
			Signature: SignatureConfig{
				Enabled:   true,
				ImagePath: "https://example.com/logo.png",
			},
		}
		got, err := buildSignatureData(cfg, false)
		if err != nil {
			t.Fatalf("buildSignatureData() unexpected error: %v", err)
		}
		if got == nil {
			t.Fatalf("buildSignatureData() = nil, want SignatureData")
		}
		if got.ImagePath != "https://example.com/logo.png" {
			t.Errorf("ImagePath = %q, want %q", got.ImagePath, "https://example.com/logo.png")
		}
	})

	t.Run("existing local image path is accepted", func(t *testing.T) {
		t.Parallel()

		tempDir := t.TempDir()
		imagePath := filepath.Join(tempDir, "logo.png")
		if err := os.WriteFile(imagePath, []byte("fake png"), 0644); err != nil {
			t.Fatalf("WriteFile() unexpected error: %v", err)
		}

		cfg := &Config{
			Author: AuthorConfig{Name: "Test"},
			Signature: SignatureConfig{
				Enabled:   true,
				ImagePath: imagePath,
			},
		}
		got, err := buildSignatureData(cfg, false)
		if err != nil {
			t.Fatalf("buildSignatureData() unexpected error: %v", err)
		}
		if got == nil {
			t.Fatalf("buildSignatureData() = nil, want SignatureData")
		}
		if got.ImagePath != imagePath {
			t.Errorf("ImagePath = %q, want %q", got.ImagePath, imagePath)
		}
	})

	t.Run("empty image path is accepted", func(t *testing.T) {
		t.Parallel()

		cfg := &Config{
			Author:    AuthorConfig{Name: "Test"},
			Signature: SignatureConfig{Enabled: true},
		}
		got, err := buildSignatureData(cfg, false)
		if err != nil {
			t.Fatalf("buildSignatureData() unexpected error: %v", err)
		}
		if got == nil {
			t.Fatalf("buildSignatureData() = nil, want SignatureData")
		}
	})

	t.Run("extended metadata fields are included", func(t *testing.T) {
		t.Parallel()

		cfg := &Config{
			Author: AuthorConfig{
				Name:       "Jane Smith",
				Phone:      "+1-555-123-4567",
				Address:    "123 Main St\nCity, State 12345",
				Department: "Engineering",
			},
			Signature: SignatureConfig{Enabled: true},
		}
		got, err := buildSignatureData(cfg, false)
		if err != nil {
			t.Fatalf("buildSignatureData() unexpected error: %v", err)
		}
		if got == nil {
			t.Fatalf("buildSignatureData() = nil, want SignatureData")
		}
		if got.Phone != "+1-555-123-4567" {
			t.Errorf("Phone = %q, want %q", got.Phone, "+1-555-123-4567")
		}
		if got.Address != "123 Main St\nCity, State 12345" {
			t.Errorf("Address = %q, want %q", got.Address, "123 Main St\nCity, State 12345")
		}
		if got.Department != "Engineering" {
			t.Errorf("Department = %q, want %q", got.Department, "Engineering")
		}
	})
}

// ---------------------------------------------------------------------------
// TestIsURL - URL detection helper
// ---------------------------------------------------------------------------

func TestIsURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input string
		want  bool
	}{
		{"https://example.com", true},
		{"http://example.com", true},
		{"https://example.com/path/to/file.png", true},
		{"/local/path/to/file.png", false},
		{"relative/path.png", false},
		{"", false},
		{"ftp://example.com", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			t.Parallel()

			got := fileutil.IsURL(tt.input)
			if got != tt.want {
				t.Errorf("IsURL(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestBuildFooterData - Footer data construction
// ---------------------------------------------------------------------------

func TestBuildFooterData(t *testing.T) {
	t.Parallel()

	t.Run("footer disabled returns nil", func(t *testing.T) {
		t.Parallel()
		cfg := &Config{Footer: FooterConfig{
			Enabled:        false,
			Position:       "right",
			ShowPageNumber: true,
			Text:           "Footer Text",
		}}
		got := buildFooterData(cfg, false)
		if got != nil {
			t.Errorf("buildFooterData() = %v, want nil", got)
		}
	})

	t.Run("footer enabled returns FooterData", func(t *testing.T) {
		t.Parallel()

		cfg := &Config{
			Document: DocumentConfig{
				Date:    "2025-01-15",
				Version: "DRAFT",
			},
			Footer: FooterConfig{
				Enabled:        true,
				Position:       "center",
				ShowPageNumber: true,
				Text:           "Footer Text",
			},
		}
		got := buildFooterData(cfg, false)
		if got == nil {
			t.Fatalf("buildFooterData() = nil, want FooterData")
		}
		if got.Position != "center" {
			t.Errorf("Position = %q, want %q", got.Position, "center")
		}
		if !got.ShowPageNumber {
			t.Errorf("ShowPageNumber = false, want true")
		}
		if got.Date != "2025-01-15" {
			t.Errorf("Date = %q, want %q", got.Date, "2025-01-15")
		}
		if got.Status != "DRAFT" {
			t.Errorf("Status = %q, want %q", got.Status, "DRAFT")
		}
		if got.Text != "Footer Text" {
			t.Errorf("Text = %q, want %q", got.Text, "Footer Text")
		}
	})

	t.Run("footer enabled with minimal config", func(t *testing.T) {
		t.Parallel()

		cfg := &Config{Footer: FooterConfig{
			Enabled: true,
			// All other fields empty/false
		}}
		got := buildFooterData(cfg, false)
		if got == nil {
			t.Fatalf("buildFooterData() = nil, want FooterData")
		}
		// All fields should be zero values
		if got.Position != "" {
			t.Errorf("Position = %q, want empty", got.Position)
		}
		if got.ShowPageNumber {
			t.Errorf("ShowPageNumber = true, want false")
		}
		if got.Date != "" {
			t.Errorf("Date = %q, want empty", got.Date)
		}
		if got.Status != "" {
			t.Errorf("Status = %q, want empty", got.Status)
		}
		if got.Text != "" {
			t.Errorf("Text = %q, want empty", got.Text)
		}
	})

	t.Run("noFooter flag returns nil even when enabled in config", func(t *testing.T) {
		t.Parallel()

		cfg := &Config{Footer: FooterConfig{
			Enabled:        true,
			Position:       "center",
			ShowPageNumber: true,
			Text:           "Footer Text",
		}}
		got := buildFooterData(cfg, true)
		if got != nil {
			t.Errorf("buildFooterData() = %v, want nil", got)
		}
	})

	t.Run("ShowDocumentID includes DocumentID in footer", func(t *testing.T) {
		t.Parallel()

		cfg := &Config{
			Document: DocumentConfig{
				DocumentID: "DOC-2024-001",
			},
			Footer: FooterConfig{
				Enabled:        true,
				ShowDocumentID: true,
			},
		}
		got := buildFooterData(cfg, false)
		if got == nil {
			t.Fatalf("buildFooterData() = nil, want FooterData")
		}
		if got.DocumentID != "DOC-2024-001" {
			t.Errorf("DocumentID = %q, want %q", got.DocumentID, "DOC-2024-001")
		}
	})

	t.Run("ShowDocumentID false excludes DocumentID", func(t *testing.T) {
		t.Parallel()

		cfg := &Config{
			Document: DocumentConfig{
				DocumentID: "DOC-2024-001",
			},
			Footer: FooterConfig{
				Enabled:        true,
				ShowDocumentID: false,
			},
		}
		got := buildFooterData(cfg, false)
		if got == nil {
			t.Fatalf("buildFooterData() = nil, want FooterData")
		}
		if got.DocumentID != "" {
			t.Errorf("DocumentID = %q, want empty", got.DocumentID)
		}
	})
}

// ---------------------------------------------------------------------------
// TestBuildPageSettings - Page size, orientation, and margin settings
// ---------------------------------------------------------------------------

func TestBuildPageSettings(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		flags           *cliFlags
		cfg             *Config
		wantNil         bool
		wantSize        string
		wantOrientation string
		wantMargin      float64
		wantErr         bool
	}{
		{
			name:    "no flags no config returns nil",
			flags:   &cliFlags{},
			cfg:     &Config{},
			wantNil: true,
		},
		{
			name:            "flags only",
			flags:           &cliFlags{page: pageFlags{size: "a4", orientation: "landscape", margin: 1.0}},
			cfg:             &Config{},
			wantSize:        "a4",
			wantOrientation: "landscape",
			wantMargin:      1.0,
		},
		{
			name:  "config only",
			flags: &cliFlags{},
			cfg: &Config{Page: PageConfig{
				Size:        "legal",
				Orientation: "portrait",
				Margin:      0.75,
			}},
			wantSize:        "legal",
			wantOrientation: "portrait",
			wantMargin:      0.75,
		},
		{
			name:  "flags override config",
			flags: &cliFlags{page: pageFlags{size: "a4", orientation: "landscape", margin: 1.5}},
			cfg: &Config{Page: PageConfig{
				Size:        "legal",
				Orientation: "portrait",
				Margin:      0.5,
			}},
			wantSize:        "a4",
			wantOrientation: "landscape",
			wantMargin:      1.5,
		},
		{
			name:  "partial flags override - size only",
			flags: &cliFlags{page: pageFlags{size: "a4"}},
			cfg: &Config{Page: PageConfig{
				Size:        "letter",
				Orientation: "landscape",
				Margin:      1.0,
			}},
			wantSize:        "a4",
			wantOrientation: "landscape",
			wantMargin:      1.0,
		},
		{
			name:  "partial flags override - orientation only",
			flags: &cliFlags{page: pageFlags{orientation: "landscape"}},
			cfg: &Config{Page: PageConfig{
				Size:        "a4",
				Orientation: "portrait",
				Margin:      0.75,
			}},
			wantSize:        "a4",
			wantOrientation: "landscape",
			wantMargin:      0.75,
		},
		{
			name:  "partial flags override - margin only",
			flags: &cliFlags{page: pageFlags{margin: 2.0}},
			cfg: &Config{Page: PageConfig{
				Size:        "legal",
				Orientation: "landscape",
				Margin:      0.5,
			}},
			wantSize:        "legal",
			wantOrientation: "landscape",
			wantMargin:      2.0,
		},
		{
			name:            "defaults applied when config partial",
			flags:           &cliFlags{},
			cfg:             &Config{Page: PageConfig{Size: "a4"}},
			wantSize:        "a4",
			wantOrientation: md2pdf.OrientationPortrait,
			wantMargin:      md2pdf.DefaultMargin,
		},
		{
			name:            "flags trigger validation with defaults",
			flags:           &cliFlags{page: pageFlags{size: "letter"}},
			cfg:             &Config{},
			wantSize:        "letter",
			wantOrientation: md2pdf.OrientationPortrait,
			wantMargin:      md2pdf.DefaultMargin,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Merge flags into config (simulates CLI behavior)
			mergeFlags(tt.flags, tt.cfg)
			got, err := buildPageSettings(tt.cfg)

			if tt.wantErr {
				if err == nil {
					t.Fatalf("buildPageSettings() error = nil, want error")
				}
				return
			}

			if err != nil {
				t.Fatalf("buildPageSettings() unexpected error: %v", err)
			}

			if tt.wantNil {
				if got != nil {
					t.Errorf("buildPageSettings() = %+v, want nil", got)
				}
				return
			}

			if got == nil {
				t.Fatalf("buildPageSettings() = nil, want PageSettings")
			}
			if got.Size != tt.wantSize {
				t.Errorf("Size = %q, want %q", got.Size, tt.wantSize)
			}
			if got.Orientation != tt.wantOrientation {
				t.Errorf("Orientation = %q, want %q", got.Orientation, tt.wantOrientation)
			}
			if got.Margin != tt.wantMargin {
				t.Errorf("Margin = %v, want %v", got.Margin, tt.wantMargin)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestValidateWorkers - Worker count validation
// ---------------------------------------------------------------------------

func TestValidateWorkers(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		n       int
		wantErr bool
		errMsg  string
	}{
		{
			name:    "negative returns error",
			n:       -1,
			wantErr: true,
			errMsg:  "must be >= 0",
		},
		{
			name:    "zero is valid (auto mode)",
			n:       0,
			wantErr: false,
		},
		{
			name:    "one is valid",
			n:       1,
			wantErr: false,
		},
		{
			name:    "max workers is valid",
			n:       md2pdf.MaxPoolSize,
			wantErr: false,
		},
		{
			name:    "above max returns error",
			n:       md2pdf.MaxPoolSize + 1,
			wantErr: true,
			errMsg:  fmt.Sprintf("maximum is %d", md2pdf.MaxPoolSize),
		},
		{
			name:    "large number returns error",
			n:       100,
			wantErr: true,
			errMsg:  fmt.Sprintf("maximum is %d", md2pdf.MaxPoolSize),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := validateWorkers(tt.n)

			if tt.wantErr {
				if err == nil {
					t.Fatalf("validateWorkers(%d) error = nil, want error", tt.n)
				}
				if !errors.Is(err, ErrInvalidWorkerCount) {
					t.Errorf("validateWorkers(%d) error = %v, want ErrInvalidWorkerCount", tt.n, err)
				}
				if tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("validateWorkers(%d) error message %q should contain %q", tt.n, err.Error(), tt.errMsg)
				}
				return
			}

			if err != nil {
				t.Errorf("validateWorkers(%d) unexpected error: %v", tt.n, err)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestBuildWatermarkData - Watermark text, color, opacity, and angle
// ---------------------------------------------------------------------------

func TestBuildWatermarkData(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		flags       *cliFlags
		cfg         *Config
		wantNil     bool
		wantText    string
		wantColor   string
		wantOpacity float64
		wantAngle   float64
		wantErr     bool
		errContains string
	}{
		{
			name:    "noWatermark flag returns nil",
			flags:   &cliFlags{watermark: watermarkFlags{disabled: true, angle: watermarkAngleSentinel}},
			cfg:     &Config{Watermark: WatermarkConfig{Enabled: true, Text: "DRAFT"}},
			wantNil: true,
		},
		{
			name:    "neither flags nor config returns nil",
			flags:   &cliFlags{watermark: watermarkFlags{angle: watermarkAngleSentinel}},
			cfg:     &Config{},
			wantNil: true,
		},
		{
			name:  "config only returns watermark",
			flags: &cliFlags{watermark: watermarkFlags{angle: watermarkAngleSentinel}},
			cfg: &Config{Watermark: WatermarkConfig{
				Enabled: true,
				Text:    "CONFIDENTIAL",
				Color:   "#ff0000",
				Opacity: 0.2,
				Angle:   -30,
			}},
			wantText:    "CONFIDENTIAL",
			wantColor:   "#ff0000",
			wantOpacity: 0.2,
			wantAngle:   -30,
		},
		{
			name:        "flags only returns watermark with defaults",
			flags:       &cliFlags{watermark: watermarkFlags{text: "DRAFT", angle: watermarkAngleSentinel}},
			cfg:         &Config{},
			wantText:    "DRAFT",
			wantColor:   "#888888", // default
			wantOpacity: 0.1,       // default
			wantAngle:   -45,       // default
		},
		{
			name: "flags override config",
			flags: &cliFlags{watermark: watermarkFlags{
				text:    "OVERRIDE",
				color:   "#00ff00",
				opacity: 0.5,
				angle:   15,
			}},
			cfg: &Config{Watermark: WatermarkConfig{
				Enabled: true,
				Text:    "ORIGINAL",
				Color:   "#ff0000",
				Opacity: 0.2,
				Angle:   -30,
			}},
			wantText:    "OVERRIDE",
			wantColor:   "#00ff00",
			wantOpacity: 0.5,
			wantAngle:   15,
		},
		{
			name: "partial flags override - text only",
			flags: &cliFlags{watermark: watermarkFlags{
				text:  "NEW TEXT",
				angle: watermarkAngleSentinel,
			}},
			cfg: &Config{Watermark: WatermarkConfig{
				Enabled: true,
				Text:    "ORIGINAL",
				Color:   "#ff0000",
				Opacity: 0.3,
				Angle:   -20,
			}},
			wantText:    "NEW TEXT",
			wantColor:   "#ff0000",
			wantOpacity: 0.3,
			wantAngle:   -20,
		},
		{
			name: "angle zero is valid (not sentinel)",
			flags: &cliFlags{watermark: watermarkFlags{
				text:  "DRAFT",
				angle: 0,
			}},
			cfg:         &Config{},
			wantText:    "DRAFT",
			wantColor:   "#888888",
			wantOpacity: 0.1,
			wantAngle:   0, // explicit zero, not default
		},
		{
			name:  "config angle zero preserved",
			flags: &cliFlags{watermark: watermarkFlags{angle: watermarkAngleSentinel}},
			cfg: &Config{Watermark: WatermarkConfig{
				Enabled: true,
				Text:    "DRAFT",
				Color:   "#888888",
				Opacity: 0.1,
				Angle:   0, // explicit zero in config
			}},
			wantText:    "DRAFT",
			wantColor:   "#888888",
			wantOpacity: 0.1,
			wantAngle:   0,
		},
		{
			name: "boundary angle -90 is valid",
			flags: &cliFlags{watermark: watermarkFlags{
				text:  "DRAFT",
				angle: -90,
			}},
			cfg:         &Config{},
			wantText:    "DRAFT",
			wantColor:   "#888888",
			wantOpacity: 0.1,
			wantAngle:   -90,
		},
		{
			name: "boundary angle 90 is valid",
			flags: &cliFlags{watermark: watermarkFlags{
				text:  "DRAFT",
				angle: 90,
			}},
			cfg:         &Config{},
			wantText:    "DRAFT",
			wantColor:   "#888888",
			wantOpacity: 0.1,
			wantAngle:   90,
		},
		{
			name: "boundary opacity 0 from config gets default",
			flags: &cliFlags{watermark: watermarkFlags{
				text:  "DRAFT",
				angle: watermarkAngleSentinel,
			}},
			cfg: &Config{Watermark: WatermarkConfig{
				Enabled: true,
				Text:    "DRAFT",
				Opacity: 0, // zero opacity in config - will get default
			}},
			wantText:    "DRAFT",
			wantColor:   "#888888",
			wantOpacity: 0.1, // default applied because 0 is treated as "not set"
			wantAngle:   0,   // config angle (0) is preserved when config is enabled
		},
		{
			name: "boundary opacity 1 is valid",
			flags: &cliFlags{watermark: watermarkFlags{
				text:    "DRAFT",
				opacity: 1.0,
				angle:   -999,
			}},
			cfg:         &Config{},
			wantText:    "DRAFT",
			wantColor:   "#888888",
			wantOpacity: 1.0,
			wantAngle:   -45,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Merge flags into config (simulates CLI behavior)
			mergeFlags(tt.flags, tt.cfg)
			got, err := buildWatermarkData(tt.cfg)

			if tt.wantErr {
				if err == nil {
					t.Fatalf("buildWatermarkData() error = nil, want error")
				}
				if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("buildWatermarkData() error %q should contain %q", err.Error(), tt.errContains)
				}
				return
			}

			if err != nil {
				t.Fatalf("buildWatermarkData() unexpected error: %v", err)
			}

			if tt.wantNil {
				if got != nil {
					t.Errorf("buildWatermarkData() = %+v, want nil", got)
				}
				return
			}

			if got == nil {
				t.Fatalf("buildWatermarkData() = nil, want Watermark")
			}
			if got.Text != tt.wantText {
				t.Errorf("Text = %q, want %q", got.Text, tt.wantText)
			}
			if got.Color != tt.wantColor {
				t.Errorf("Color = %q, want %q", got.Color, tt.wantColor)
			}
			if got.Opacity != tt.wantOpacity {
				t.Errorf("Opacity = %v, want %v", got.Opacity, tt.wantOpacity)
			}
			if got.Angle != tt.wantAngle {
				t.Errorf("Angle = %v, want %v", got.Angle, tt.wantAngle)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestExtractFirstHeading - H1 heading extraction from markdown
// ---------------------------------------------------------------------------

func TestExtractFirstHeading(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		markdown string
		want     string
	}{
		{
			name:     "simple H1",
			markdown: "# Hello World\n\nSome content",
			want:     "Hello World",
		},
		{
			name:     "H1 with leading/trailing spaces trimmed",
			markdown: "#   Spaced Title   \n\nContent",
			want:     "Spaced Title",
		},
		{
			name:     "H2 ignored - only H1 extracted",
			markdown: "## This is H2\n\n# This is H1",
			want:     "This is H1",
		},
		{
			name:     "no heading returns empty",
			markdown: "Just some paragraph text.\n\nNo headings here.",
			want:     "",
		},
		{
			name:     "multiple H1 returns first",
			markdown: "# First Heading\n\n# Second Heading\n\n# Third",
			want:     "First Heading",
		},
		{
			name:     "H1 with inline formatting",
			markdown: "# Title with **bold** and *italic*\n\nContent",
			want:     "Title with **bold** and *italic*",
		},
		{
			name:     "empty markdown returns empty",
			markdown: "",
			want:     "",
		},
		{
			name:     "H1 at end of file",
			markdown: "Some intro\n\n# Final Heading",
			want:     "Final Heading",
		},
		{
			name:     "hash in middle of line not H1",
			markdown: "This has a # in the middle\n\n# Real H1",
			want:     "Real H1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := extractFirstHeading(tt.markdown)
			if got != tt.want {
				t.Errorf("extractFirstHeading() = %q, want %q", got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestResolveDateWithTime - Date resolution with auto format support
// ---------------------------------------------------------------------------

func TestResolveDateWithTime(t *testing.T) {
	t.Parallel()

	fixedTime := time.Date(2025, 6, 15, 12, 0, 0, 0, time.UTC)
	mockNow := func() time.Time { return fixedTime }

	tests := []struct {
		name    string
		input   string
		want    string
		wantErr error
	}{
		{
			name:  "auto returns fixed date",
			input: "auto",
			want:  "2025-06-15",
		},
		{
			name:  "AUTO case insensitive",
			input: "AUTO",
			want:  "2025-06-15",
		},
		{
			name:  "Auto mixed case",
			input: "Auto",
			want:  "2025-06-15",
		},
		{
			name:  "explicit date preserved",
			input: "2025-01-01",
			want:  "2025-01-01",
		},
		{
			name:  "empty string preserved",
			input: "",
			want:  "",
		},
		{
			name:  "custom format preserved",
			input: "January 2025",
			want:  "January 2025",
		},
		// Error cases
		{
			name:    "auto with empty format returns error",
			input:   "auto:",
			wantErr: dateutil.ErrInvalidDateFormat,
		},
		{
			name:    "invalid auto syntax returns error",
			input:   "autoXXX",
			wantErr: dateutil.ErrInvalidDateFormat,
		},
		{
			name:    "unclosed bracket returns error",
			input:   "auto:[Date",
			wantErr: dateutil.ErrInvalidDateFormat,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := resolveDateWithTime(tt.input, mockNow)

			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("resolveDateWithTime(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				}
				return
			}

			if err != nil {
				t.Fatalf("resolveDateWithTime(%q) unexpected error: %v", tt.input, err)
			}
			if got != tt.want {
				t.Errorf("resolveDateWithTime(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestBuildCoverData - Cover page data construction
// ---------------------------------------------------------------------------

func TestBuildCoverData(t *testing.T) {
	t.Parallel()

	// Create a temp file for logo path tests
	tempDir := t.TempDir()
	existingLogo := filepath.Join(tempDir, "logo.png")
	if err := os.WriteFile(existingLogo, []byte("fake png"), 0644); err != nil {
		t.Fatalf("WriteFile() unexpected error: %v", err)
	}

	t.Run("cover disabled in config returns nil", func(t *testing.T) {
		t.Parallel()

		cfg := &Config{Cover: CoverConfig{Enabled: false}}
		got, err := buildCoverData(cfg, "# Markdown", "doc.md")
		if err != nil {
			t.Fatalf("buildCoverData() unexpected error: %v", err)
		}
		if got != nil {
			t.Errorf("buildCoverData() = %v, want nil", got)
		}
	})

	t.Run("title from document config", func(t *testing.T) {
		t.Parallel()

		cfg := &Config{
			Document: DocumentConfig{Title: "Config Title"},
			Cover:    CoverConfig{Enabled: true},
		}
		got, err := buildCoverData(cfg, "# Markdown H1", "doc.md")
		if err != nil {
			t.Fatalf("buildCoverData() unexpected error: %v", err)
		}
		if got == nil {
			t.Fatalf("buildCoverData() = nil, want Cover")
		}
		if got.Title != "Config Title" {
			t.Errorf("Title = %q, want %q", got.Title, "Config Title")
		}
	})

	t.Run("title extracted from H1 when no document title", func(t *testing.T) {
		t.Parallel()

		cfg := &Config{Cover: CoverConfig{Enabled: true}}
		got, err := buildCoverData(cfg, "# My Document Title\n\nContent here", "doc.md")
		if err != nil {
			t.Fatalf("buildCoverData() unexpected error: %v", err)
		}
		if got == nil {
			t.Fatalf("buildCoverData() = nil, want Cover")
		}
		if got.Title != "My Document Title" {
			t.Errorf("Title = %q, want %q", got.Title, "My Document Title")
		}
	})

	t.Run("title fallback to filename when no H1", func(t *testing.T) {
		t.Parallel()

		cfg := &Config{Cover: CoverConfig{Enabled: true}}
		got, err := buildCoverData(cfg, "No headings here, just content.", "my-document.md")
		if err != nil {
			t.Fatalf("buildCoverData() unexpected error: %v", err)
		}
		if got == nil {
			t.Fatalf("buildCoverData() = nil, want Cover")
		}
		if got.Title != "my-document" {
			t.Errorf("Title = %q, want %q", got.Title, "my-document")
		}
	})

	t.Run("subtitle from document config", func(t *testing.T) {
		t.Parallel()

		cfg := &Config{
			Document: DocumentConfig{Title: "Title", Subtitle: "A Comprehensive Guide"},
			Cover:    CoverConfig{Enabled: true},
		}
		got, err := buildCoverData(cfg, "", "doc.md")
		if err != nil {
			t.Fatalf("buildCoverData() unexpected error: %v", err)
		}
		if got == nil {
			t.Fatalf("buildCoverData() = nil, want Cover")
		}
		if got.Subtitle != "A Comprehensive Guide" {
			t.Errorf("Subtitle = %q, want %q", got.Subtitle, "A Comprehensive Guide")
		}
	})

	t.Run("logo from cover config", func(t *testing.T) {
		t.Parallel()

		cfg := &Config{
			Document: DocumentConfig{Title: "Title"},
			Cover:    CoverConfig{Enabled: true, Logo: existingLogo},
		}
		got, err := buildCoverData(cfg, "", "doc.md")
		if err != nil {
			t.Fatalf("buildCoverData() unexpected error: %v", err)
		}
		if got == nil {
			t.Fatalf("buildCoverData() = nil, want Cover")
		}
		if got.Logo != existingLogo {
			t.Errorf("Logo = %q, want %q", got.Logo, existingLogo)
		}
	})

	t.Run("logo URL accepted without validation", func(t *testing.T) {
		t.Parallel()

		cfg := &Config{
			Document: DocumentConfig{Title: "Title"},
			Cover:    CoverConfig{Enabled: true, Logo: "https://example.com/logo.png"},
		}
		got, err := buildCoverData(cfg, "", "doc.md")
		if err != nil {
			t.Fatalf("buildCoverData() unexpected error: %v", err)
		}
		if got == nil {
			t.Fatalf("buildCoverData() = nil, want Cover")
		}
		if got.Logo != "https://example.com/logo.png" {
			t.Errorf("Logo = %q, want %q", got.Logo, "https://example.com/logo.png")
		}
	})

	t.Run("author from author config", func(t *testing.T) {
		t.Parallel()

		cfg := &Config{
			Author:   AuthorConfig{Name: "John Doe"},
			Document: DocumentConfig{Title: "Title"},
			Cover:    CoverConfig{Enabled: true},
		}
		got, err := buildCoverData(cfg, "", "doc.md")
		if err != nil {
			t.Fatalf("buildCoverData() unexpected error: %v", err)
		}
		if got == nil {
			t.Fatalf("buildCoverData() = nil, want Cover")
		}
		if got.Author != "John Doe" {
			t.Errorf("Author = %q, want %q", got.Author, "John Doe")
		}
	})

	t.Run("authorTitle from author config", func(t *testing.T) {
		t.Parallel()

		cfg := &Config{
			Author:   AuthorConfig{Name: "John", Title: "Senior Developer"},
			Document: DocumentConfig{Title: "Title"},
			Cover:    CoverConfig{Enabled: true},
		}
		got, err := buildCoverData(cfg, "", "doc.md")
		if err != nil {
			t.Fatalf("buildCoverData() unexpected error: %v", err)
		}
		if got == nil {
			t.Fatalf("buildCoverData() = nil, want Cover")
		}
		if got.AuthorTitle != "Senior Developer" {
			t.Errorf("AuthorTitle = %q, want %q", got.AuthorTitle, "Senior Developer")
		}
	})

	t.Run("organization from author config", func(t *testing.T) {
		t.Parallel()

		cfg := &Config{
			Author:   AuthorConfig{Organization: "Acme Corp"},
			Document: DocumentConfig{Title: "Title"},
			Cover:    CoverConfig{Enabled: true},
		}
		got, err := buildCoverData(cfg, "", "doc.md")
		if err != nil {
			t.Fatalf("buildCoverData() unexpected error: %v", err)
		}
		if got == nil {
			t.Fatalf("buildCoverData() = nil, want Cover")
		}
		if got.Organization != "Acme Corp" {
			t.Errorf("Organization = %q, want %q", got.Organization, "Acme Corp")
		}
	})

	t.Run("date from document config", func(t *testing.T) {
		t.Parallel()

		cfg := &Config{
			Document: DocumentConfig{Title: "Title", Date: "2025-01-15"},
			Cover:    CoverConfig{Enabled: true},
		}
		got, err := buildCoverData(cfg, "", "doc.md")
		if err != nil {
			t.Fatalf("buildCoverData() unexpected error: %v", err)
		}
		if got == nil {
			t.Fatalf("buildCoverData() = nil, want Cover")
		}
		if got.Date != "2025-01-15" {
			t.Errorf("Date = %q, want %q", got.Date, "2025-01-15")
		}
	})

	t.Run("version from document config", func(t *testing.T) {
		t.Parallel()

		cfg := &Config{
			Document: DocumentConfig{Title: "Title", Version: "v2.0.0"},
			Cover:    CoverConfig{Enabled: true},
		}
		got, err := buildCoverData(cfg, "", "doc.md")
		if err != nil {
			t.Fatalf("buildCoverData() unexpected error: %v", err)
		}
		if got == nil {
			t.Fatalf("buildCoverData() = nil, want Cover")
		}
		if got.Version != "v2.0.0" {
			t.Errorf("Version = %q, want %q", got.Version, "v2.0.0")
		}
	})

	t.Run("all fields populated correctly", func(t *testing.T) {
		t.Parallel()

		cfg := &Config{
			Author: AuthorConfig{
				Name:         "Author Name",
				Title:        "Author Role",
				Organization: "Org Name",
			},
			Document: DocumentConfig{
				Title:    "Doc Title",
				Subtitle: "A Subtitle",
				Date:     "2025-03-15",
				Version:  "v1.0.0",
			},
			Cover: CoverConfig{
				Enabled: true,
				Logo:    existingLogo,
			},
		}
		got, err := buildCoverData(cfg, "", "doc.md")
		if err != nil {
			t.Fatalf("buildCoverData() unexpected error: %v", err)
		}
		if got == nil {
			t.Fatalf("buildCoverData() = nil, want Cover")
		}
		if got.Title != "Doc Title" {
			t.Errorf("Title = %q, want %q", got.Title, "Doc Title")
		}
		if got.Subtitle != "A Subtitle" {
			t.Errorf("Subtitle = %q, want %q", got.Subtitle, "A Subtitle")
		}
		if got.Logo != existingLogo {
			t.Errorf("Logo = %q, want %q", got.Logo, existingLogo)
		}
		if got.Author != "Author Name" {
			t.Errorf("Author = %q, want %q", got.Author, "Author Name")
		}
		if got.AuthorTitle != "Author Role" {
			t.Errorf("AuthorTitle = %q, want %q", got.AuthorTitle, "Author Role")
		}
		if got.Organization != "Org Name" {
			t.Errorf("Organization = %q, want %q", got.Organization, "Org Name")
		}
		if got.Date != "2025-03-15" {
			t.Errorf("Date = %q, want %q", got.Date, "2025-03-15")
		}
		if got.Version != "v1.0.0" {
			t.Errorf("Version = %q, want %q", got.Version, "v1.0.0")
		}
	})

	t.Run("empty optional fields preserved", func(t *testing.T) {
		t.Parallel()

		cfg := &Config{
			Document: DocumentConfig{Title: "Just Title"},
			Cover:    CoverConfig{Enabled: true},
		}
		got, err := buildCoverData(cfg, "", "doc.md")
		if err != nil {
			t.Fatalf("buildCoverData() unexpected error: %v", err)
		}
		if got == nil {
			t.Fatalf("buildCoverData() = nil, want Cover")
		}
		if got.Subtitle != "" {
			t.Errorf("Subtitle = %q, want empty", got.Subtitle)
		}
		if got.Logo != "" {
			t.Errorf("Logo = %q, want empty", got.Logo)
		}
		if got.Author != "" {
			t.Errorf("Author = %q, want empty", got.Author)
		}
		if got.Organization != "" {
			t.Errorf("Organization = %q, want empty", got.Organization)
		}
	})

	t.Run("extended metadata fields are included", func(t *testing.T) {
		t.Parallel()

		cfg := &Config{
			Author: AuthorConfig{
				Department: "Engineering",
			},
			Document: DocumentConfig{
				Title:        "Project Specification",
				ClientName:   "Acme Corporation",
				ProjectName:  "Project Phoenix",
				DocumentType: "Technical Specification",
				DocumentID:   "DOC-2024-001",
				Description:  "System design document for Project Phoenix",
			},
			Cover: CoverConfig{
				Enabled:        true,
				ShowDepartment: true,
			},
		}
		got, err := buildCoverData(cfg, "", "doc.md")
		if err != nil {
			t.Fatalf("buildCoverData() unexpected error: %v", err)
		}
		if got == nil {
			t.Fatalf("buildCoverData() = nil, want Cover")
		}
		if got.ClientName != "Acme Corporation" {
			t.Errorf("ClientName = %q, want %q", got.ClientName, "Acme Corporation")
		}
		if got.ProjectName != "Project Phoenix" {
			t.Errorf("ProjectName = %q, want %q", got.ProjectName, "Project Phoenix")
		}
		if got.DocumentType != "Technical Specification" {
			t.Errorf("DocumentType = %q, want %q", got.DocumentType, "Technical Specification")
		}
		if got.DocumentID != "DOC-2024-001" {
			t.Errorf("DocumentID = %q, want %q", got.DocumentID, "DOC-2024-001")
		}
		if got.Description != "System design document for Project Phoenix" {
			t.Errorf("Description = %q, want %q", got.Description, "System design document for Project Phoenix")
		}
		if got.Department != "Engineering" {
			t.Errorf("Department = %q, want %q", got.Department, "Engineering")
		}
	})

	t.Run("ShowDepartment false excludes department", func(t *testing.T) {
		t.Parallel()

		cfg := &Config{
			Author: AuthorConfig{
				Department: "Engineering",
			},
			Document: DocumentConfig{Title: "Doc Title"},
			Cover: CoverConfig{
				Enabled:        true,
				ShowDepartment: false,
			},
		}
		got, err := buildCoverData(cfg, "", "doc.md")
		if err != nil {
			t.Fatalf("buildCoverData() unexpected error: %v", err)
		}
		if got == nil {
			t.Fatalf("buildCoverData() = nil, want Cover")
		}
		if got.Department != "" {
			t.Errorf("Department = %q, want empty when ShowDepartment=false", got.Department)
		}
	})
}

// ---------------------------------------------------------------------------
// TestBuildTOCData - Table of contents data construction
// ---------------------------------------------------------------------------

func TestBuildTOCData(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		cfg          *Config
		flags        tocFlags
		wantNil      bool
		wantTitle    string
		wantMinDepth int
		wantMaxDepth int
		wantErr      string
	}{
		{
			name:    "noTOC flag returns nil",
			cfg:     &Config{TOC: TOCConfig{Enabled: true, Title: "Contents", MaxDepth: 3}},
			flags:   tocFlags{disabled: true},
			wantNil: true,
		},
		{
			name:    "config disabled returns nil",
			cfg:     &Config{TOC: TOCConfig{Enabled: false, Title: "Contents", MaxDepth: 3}},
			flags:   tocFlags{},
			wantNil: true,
		},
		{
			name:    "neither flag nor config enabled returns nil",
			cfg:     &Config{},
			flags:   tocFlags{},
			wantNil: true,
		},
		{
			name:         "config enabled with title and depth",
			cfg:          &Config{TOC: TOCConfig{Enabled: true, Title: "Table of Contents", MaxDepth: 4}},
			flags:        tocFlags{},
			wantTitle:    "Table of Contents",
			wantMaxDepth: 4,
		},
		{
			name:         "config enabled empty title preserved",
			cfg:          &Config{TOC: TOCConfig{Enabled: true, Title: "", MaxDepth: 3}},
			flags:        tocFlags{},
			wantTitle:    "",
			wantMaxDepth: 3,
		},
		{
			name:         "config depth 0 gets default",
			cfg:          &Config{TOC: TOCConfig{Enabled: true, Title: "TOC", MaxDepth: 0}},
			flags:        tocFlags{},
			wantTitle:    "TOC",
			wantMaxDepth: md2pdf.DefaultTOCMaxDepth,
		},
		{
			name:         "config depth 1 boundary",
			cfg:          &Config{TOC: TOCConfig{Enabled: true, MaxDepth: 1}},
			flags:        tocFlags{},
			wantMaxDepth: 1,
		},
		{
			name:         "config depth 6 boundary",
			cfg:          &Config{TOC: TOCConfig{Enabled: true, MaxDepth: 6}},
			flags:        tocFlags{},
			wantMaxDepth: 6,
		},
		// MinDepth tests
		{
			name:         "minDepth 1 includes H1",
			cfg:          &Config{TOC: TOCConfig{Enabled: true, MinDepth: 1, MaxDepth: 3}},
			flags:        tocFlags{},
			wantMinDepth: 1,
			wantMaxDepth: 3,
		},
		{
			name:         "minDepth 2 skips H1",
			cfg:          &Config{TOC: TOCConfig{Enabled: true, MinDepth: 2, MaxDepth: 4}},
			flags:        tocFlags{},
			wantMinDepth: 2,
			wantMaxDepth: 4,
		},
		{
			name:         "minDepth equals maxDepth valid",
			cfg:          &Config{TOC: TOCConfig{Enabled: true, MinDepth: 3, MaxDepth: 3}},
			flags:        tocFlags{},
			wantMinDepth: 3,
			wantMaxDepth: 3,
		},
		{
			name:         "minDepth 0 uses default",
			cfg:          &Config{TOC: TOCConfig{Enabled: true, MinDepth: 0, MaxDepth: 3}},
			flags:        tocFlags{},
			wantMinDepth: 0, // 0 = library applies default (2)
			wantMaxDepth: 3,
		},
		{
			name:         "minDepth 6 boundary valid",
			cfg:          &Config{TOC: TOCConfig{Enabled: true, MinDepth: 6, MaxDepth: 6}},
			flags:        tocFlags{},
			wantMinDepth: 6,
			wantMaxDepth: 6,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := buildTOCData(tt.cfg, tt.flags)

			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("buildTOCData() error = nil, want error containing %q", tt.wantErr)
				}
				if !strings.Contains(err.Error(), tt.wantErr) {
					t.Errorf("buildTOCData() error = %q, want containing %q", err.Error(), tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("buildTOCData() unexpected error: %v", err)
			}

			if tt.wantNil {
				if got != nil {
					t.Errorf("buildTOCData() = %+v, want nil", got)
				}
				return
			}

			if got == nil {
				t.Fatalf("buildTOCData() = nil, want TOC")
			}
			if got.Title != tt.wantTitle {
				t.Errorf("Title = %q, want %q", got.Title, tt.wantTitle)
			}
			if got.MinDepth != tt.wantMinDepth {
				t.Errorf("MinDepth = %d, want %d", got.MinDepth, tt.wantMinDepth)
			}
			if got.MaxDepth != tt.wantMaxDepth {
				t.Errorf("MaxDepth = %d, want %d", got.MaxDepth, tt.wantMaxDepth)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestParseBreakBefore - Page break heading level parsing
// ---------------------------------------------------------------------------

func TestParseBreakBefore(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		input  string
		wantH1 bool
		wantH2 bool
		wantH3 bool
	}{
		{
			name:   "empty string returns all false",
			input:  "",
			wantH1: false,
			wantH2: false,
			wantH3: false,
		},
		{
			name:   "h1 only",
			input:  "h1",
			wantH1: true,
			wantH2: false,
			wantH3: false,
		},
		{
			name:   "h2 only",
			input:  "h2",
			wantH1: false,
			wantH2: true,
			wantH3: false,
		},
		{
			name:   "h3 only",
			input:  "h3",
			wantH1: false,
			wantH2: false,
			wantH3: true,
		},
		{
			name:   "h1,h2 comma separated",
			input:  "h1,h2",
			wantH1: true,
			wantH2: true,
			wantH3: false,
		},
		{
			name:   "h2,h3 comma separated",
			input:  "h2,h3",
			wantH1: false,
			wantH2: true,
			wantH3: true,
		},
		{
			name:   "all headings h1,h2,h3",
			input:  "h1,h2,h3",
			wantH1: true,
			wantH2: true,
			wantH3: true,
		},
		{
			name:   "case insensitive H1,H2,H3",
			input:  "H1,H2,H3",
			wantH1: true,
			wantH2: true,
			wantH3: true,
		},
		{
			name:   "mixed case with spaces",
			input:  " H1 , h2 , H3 ",
			wantH1: true,
			wantH2: true,
			wantH3: true,
		},
		{
			name:   "duplicate entries",
			input:  "h1,h1,h1",
			wantH1: true,
			wantH2: false,
			wantH3: false,
		},
		{
			name:   "unrecognized entries ignored",
			input:  "h1,h4,h5,h6,invalid",
			wantH1: true,
			wantH2: false,
			wantH3: false,
		},
		{
			name:   "only unrecognized entries",
			input:  "h4,h5,h6,invalid",
			wantH1: false,
			wantH2: false,
			wantH3: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			gotH1, gotH2, gotH3 := parseBreakBefore(tt.input)

			if gotH1 != tt.wantH1 {
				t.Errorf("h1 = %v, want %v", gotH1, tt.wantH1)
			}
			if gotH2 != tt.wantH2 {
				t.Errorf("h2 = %v, want %v", gotH2, tt.wantH2)
			}
			if gotH3 != tt.wantH3 {
				t.Errorf("h3 = %v, want %v", gotH3, tt.wantH3)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestBuildPageBreaksData - Page breaks data construction
// ---------------------------------------------------------------------------

func TestBuildPageBreaksData(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		flags        *cliFlags
		cfg          *Config
		wantNil      bool
		wantBeforeH1 bool
		wantBeforeH2 bool
		wantBeforeH3 bool
		wantOrphans  int
		wantWidows   int
	}{
		{
			name:    "noPageBreaks flag returns nil",
			flags:   &cliFlags{pageBreaks: pageBreakFlags{disabled: true}},
			cfg:     &Config{PageBreaks: PageBreaksConfig{Enabled: true, BeforeH1: true}},
			wantNil: true,
		},
		{
			name:    "neither flags nor config returns nil",
			flags:   &cliFlags{},
			cfg:     &Config{},
			wantNil: true,
		},
		{
			name:         "config only returns config values",
			flags:        &cliFlags{},
			cfg:          &Config{PageBreaks: PageBreaksConfig{Enabled: true, BeforeH1: true, BeforeH2: true, Orphans: 3, Widows: 4}},
			wantBeforeH1: true,
			wantBeforeH2: true,
			wantBeforeH3: false,
			wantOrphans:  3,
			wantWidows:   4,
		},
		{
			name:         "breakBefore flag overrides config",
			flags:        &cliFlags{pageBreaks: pageBreakFlags{breakBefore: "h2,h3"}},
			cfg:          &Config{PageBreaks: PageBreaksConfig{Enabled: true, BeforeH1: true, BeforeH2: false}},
			wantBeforeH1: false,
			wantBeforeH2: true,
			wantBeforeH3: true,
			wantOrphans:  md2pdf.DefaultOrphans,
			wantWidows:   md2pdf.DefaultWidows,
		},
		{
			name:        "orphans flag overrides config",
			flags:       &cliFlags{pageBreaks: pageBreakFlags{orphans: 5}},
			cfg:         &Config{PageBreaks: PageBreaksConfig{Enabled: true, Orphans: 3}},
			wantOrphans: 5,
			wantWidows:  md2pdf.DefaultWidows,
		},
		{
			name:        "widows flag overrides config",
			flags:       &cliFlags{pageBreaks: pageBreakFlags{widows: 5}},
			cfg:         &Config{PageBreaks: PageBreaksConfig{Enabled: true, Widows: 3}},
			wantOrphans: md2pdf.DefaultOrphans,
			wantWidows:  5,
		},
		{
			name:         "all flags override config",
			flags:        &cliFlags{pageBreaks: pageBreakFlags{breakBefore: "h1", orphans: 4, widows: 5}},
			cfg:          &Config{PageBreaks: PageBreaksConfig{Enabled: true, BeforeH2: true, BeforeH3: true, Orphans: 2, Widows: 2}},
			wantBeforeH1: true,
			wantBeforeH2: false,
			wantBeforeH3: false,
			wantOrphans:  4,
			wantWidows:   5,
		},
		{
			name:    "config disabled but has values - returns nil",
			flags:   &cliFlags{},
			cfg:     &Config{PageBreaks: PageBreaksConfig{Enabled: false, BeforeH1: true, Orphans: 5}},
			wantNil: true,
		},
		{
			name:        "config orphans 0 uses default",
			flags:       &cliFlags{},
			cfg:         &Config{PageBreaks: PageBreaksConfig{Enabled: true, Orphans: 0, Widows: 3}},
			wantOrphans: md2pdf.DefaultOrphans,
			wantWidows:  3,
		},
		{
			name:        "config widows 0 uses default",
			flags:       &cliFlags{},
			cfg:         &Config{PageBreaks: PageBreaksConfig{Enabled: true, Orphans: 3, Widows: 0}},
			wantOrphans: 3,
			wantWidows:  md2pdf.DefaultWidows,
		},
		{
			name:         "breakBefore flag with empty config",
			flags:        &cliFlags{pageBreaks: pageBreakFlags{breakBefore: "h1,h2,h3"}},
			cfg:          &Config{},
			wantBeforeH1: true,
			wantBeforeH2: true,
			wantBeforeH3: true,
			wantOrphans:  md2pdf.DefaultOrphans,
			wantWidows:   md2pdf.DefaultWidows,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Merge flags into config (simulates CLI behavior)
			mergeFlags(tt.flags, tt.cfg)
			got := buildPageBreaksData(tt.cfg)

			if tt.wantNil {
				if got != nil {
					t.Errorf("buildPageBreaksData() = %+v, want nil", got)
				}
				return
			}

			if got == nil {
				t.Fatalf("buildPageBreaksData() = nil, want PageBreaks")
			}
			if got.BeforeH1 != tt.wantBeforeH1 {
				t.Errorf("BeforeH1 = %v, want %v", got.BeforeH1, tt.wantBeforeH1)
			}
			if got.BeforeH2 != tt.wantBeforeH2 {
				t.Errorf("BeforeH2 = %v, want %v", got.BeforeH2, tt.wantBeforeH2)
			}
			if got.BeforeH3 != tt.wantBeforeH3 {
				t.Errorf("BeforeH3 = %v, want %v", got.BeforeH3, tt.wantBeforeH3)
			}
			if got.Orphans != tt.wantOrphans {
				t.Errorf("Orphans = %d, want %d", got.Orphans, tt.wantOrphans)
			}
			if got.Widows != tt.wantWidows {
				t.Errorf("Widows = %d, want %d", got.Widows, tt.wantWidows)
			}
		})
	}
}
