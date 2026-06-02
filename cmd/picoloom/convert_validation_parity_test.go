package main

import (
	"testing"

	picoloom "github.com/alnah/picoloom/v2"
)

type publicConfigValidator interface {
	Validate() error
}

func TestConfigBuildersProduceValidPublicTypes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		cfg   *Config
		build func(*Config) publicConfigValidator
	}{
		{
			name: "page settings at minimum margin",
			cfg:  &Config{Page: PageConfig{Size: picoloom.PageSizeLetter, Orientation: picoloom.OrientationPortrait, Margin: picoloom.MinMargin}},
			build: func(cfg *Config) publicConfigValidator {
				return buildPageSettings(cfg)
			},
		},
		{
			name: "page settings at maximum margin",
			cfg:  &Config{Page: PageConfig{Size: picoloom.PageSizeA4, Orientation: picoloom.OrientationLandscape, Margin: picoloom.MaxMargin}},
			build: func(cfg *Config) publicConfigValidator {
				return buildPageSettings(cfg)
			},
		},
		{
			name: "watermark at minimum opacity and angle",
			cfg:  &Config{Watermark: WatermarkConfig{Enabled: true, Text: "DRAFT", Color: "#888", Opacity: picoloom.MinWatermarkOpacity, Angle: picoloom.MinWatermarkAngle}},
			build: func(cfg *Config) publicConfigValidator {
				return buildWatermarkData(cfg)
			},
		},
		{
			name: "watermark at maximum opacity and angle",
			cfg:  &Config{Watermark: WatermarkConfig{Enabled: true, Text: "DRAFT", Color: "#888888", Opacity: picoloom.MaxWatermarkOpacity, Angle: picoloom.MaxWatermarkAngle}},
			build: func(cfg *Config) publicConfigValidator {
				return buildWatermarkData(cfg)
			},
		},
		{
			name: "toc at minimum depths",
			cfg:  &Config{TOC: TOCConfig{Enabled: true, MinDepth: 1, MaxDepth: 1}},
			build: func(cfg *Config) publicConfigValidator {
				return buildTOCData(cfg, tocFlags{})
			},
		},
		{
			name: "toc at maximum depths",
			cfg:  &Config{TOC: TOCConfig{Enabled: true, MinDepth: 6, MaxDepth: 6}},
			build: func(cfg *Config) publicConfigValidator {
				return buildTOCData(cfg, tocFlags{})
			},
		},
		{
			name: "page breaks at minimum orphans and widows",
			cfg:  &Config{PageBreaks: PageBreaksConfig{Enabled: true, Orphans: picoloom.MinOrphans, Widows: picoloom.MinWidows}},
			build: func(cfg *Config) publicConfigValidator {
				return buildPageBreaksData(cfg)
			},
		},
		{
			name: "page breaks at maximum orphans and widows",
			cfg:  &Config{PageBreaks: PageBreaksConfig{Enabled: true, Orphans: picoloom.MaxOrphans, Widows: picoloom.MaxWidows}},
			build: func(cfg *Config) publicConfigValidator {
				return buildPageBreaksData(cfg)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if err := tt.cfg.Validate(); err != nil {
				t.Fatalf("Config.Validate() unexpected error: %v", err)
			}

			got := tt.build(tt.cfg)
			if got == nil {
				t.Fatalf("builder returned nil for validated config %+v", tt.cfg)
			}
			if err := got.Validate(); err != nil {
				t.Fatalf("builder returned invalid public type: %T.Validate() error = %v", got, err)
			}
		})
	}
}

func TestConfigValidationRejectsPublicTypeBounds(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		cfg  *Config
	}{
		{name: "page margin below public minimum", cfg: &Config{Page: PageConfig{Margin: picoloom.MinMargin - 0.01}}},
		{name: "page margin above public maximum", cfg: &Config{Page: PageConfig{Margin: picoloom.MaxMargin + 0.01}}},
		{name: "watermark opacity below public minimum", cfg: &Config{Watermark: WatermarkConfig{Enabled: true, Text: "DRAFT", Opacity: picoloom.MinWatermarkOpacity - 0.01, Angle: picoloom.DefaultWatermarkAngle}}},
		{name: "watermark opacity above public maximum", cfg: &Config{Watermark: WatermarkConfig{Enabled: true, Text: "DRAFT", Opacity: picoloom.MaxWatermarkOpacity + 0.01, Angle: picoloom.DefaultWatermarkAngle}}},
		{name: "watermark angle below public minimum", cfg: &Config{Watermark: WatermarkConfig{Enabled: true, Text: "DRAFT", Opacity: picoloom.DefaultWatermarkOpacity, Angle: picoloom.MinWatermarkAngle - 1}}},
		{name: "watermark angle above public maximum", cfg: &Config{Watermark: WatermarkConfig{Enabled: true, Text: "DRAFT", Opacity: picoloom.DefaultWatermarkOpacity, Angle: picoloom.MaxWatermarkAngle + 1}}},
		{name: "page breaks orphans below public minimum", cfg: &Config{PageBreaks: PageBreaksConfig{Enabled: true, Orphans: -1}}},
		{name: "page breaks orphans above public maximum", cfg: &Config{PageBreaks: PageBreaksConfig{Enabled: true, Orphans: picoloom.MaxOrphans + 1}}},
		{name: "page breaks widows below public minimum", cfg: &Config{PageBreaks: PageBreaksConfig{Enabled: true, Widows: -1}}},
		{name: "page breaks widows above public maximum", cfg: &Config{PageBreaks: PageBreaksConfig{Enabled: true, Widows: picoloom.MaxWidows + 1}}},
		{name: "watermark color rejects non-hex value", cfg: &Config{Watermark: WatermarkConfig{Enabled: true, Text: "DRAFT", Color: "red", Opacity: picoloom.DefaultWatermarkOpacity, Angle: picoloom.DefaultWatermarkAngle}}},
		{name: "toc min depth below public minimum", cfg: &Config{TOC: TOCConfig{Enabled: true, MinDepth: -1, MaxDepth: 1}}},
		{name: "toc min depth above public maximum", cfg: &Config{TOC: TOCConfig{Enabled: true, MinDepth: 7}}},
		{name: "toc max depth below public minimum", cfg: &Config{TOC: TOCConfig{Enabled: true, MinDepth: 1, MaxDepth: -1}}},
		{name: "toc max depth above public maximum", cfg: &Config{TOC: TOCConfig{Enabled: true, MinDepth: 1, MaxDepth: 7}}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if err := tt.cfg.Validate(); err == nil {
				t.Fatalf("Config.Validate() error = nil, want error for config %+v", tt.cfg)
			}
		})
	}
}
