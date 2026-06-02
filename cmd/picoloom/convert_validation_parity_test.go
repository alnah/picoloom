package main

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	picoloom "github.com/alnah/picoloom/v2"
	configpkg "github.com/alnah/picoloom/v2/internal/config"
)

type publicConfigValidator interface {
	Validate() error
}

// Keep this file as the guardrail for config -> CLI builders -> public type
// validation parity. When adding a user-facing conversion option, classify the
// new Input field here and add cases if it maps to a public type with Validate
// behavior.
func TestConfigBuilderParityClassifiesInputFields(t *testing.T) {
	t.Parallel()

	type inputFieldClassification struct {
		reason                string
		requiresBuilderParity bool
	}

	classifications := map[string]inputFieldClassification{
		"Markdown":   {reason: "read per file and passed through directly"},
		"SourceDir":  {reason: "derived per file for relative path rewriting"},
		"CSS":        {reason: "resolved by resolveCSSContent, no public Validate method"},
		"HTMLOnly":   {reason: "output mode flag, no config-to-public validator"},
		"Footer":     {reason: "buildFooterData", requiresBuilderParity: true},
		"Signature":  {reason: "buildSignatureData", requiresBuilderParity: true},
		"Page":       {reason: "buildPageSettings", requiresBuilderParity: true},
		"Watermark":  {reason: "buildWatermarkData", requiresBuilderParity: true},
		"Cover":      {reason: "buildCoverData", requiresBuilderParity: true},
		"TOC":        {reason: "buildTOCData", requiresBuilderParity: true},
		"PageBreaks": {reason: "buildPageBreaksData", requiresBuilderParity: true},
	}

	inputType := reflect.TypeOf(picoloom.Input{})
	for i := 0; i < inputType.NumField(); i++ {
		field := inputType.Field(i).Name
		classification, ok := classifications[field]
		if !ok {
			t.Fatalf("picoloom.Input.%s is not classified; add config validation, CLI merge/build wiring, and parity tests if needed", field)
		}
		if classification.reason == "" {
			t.Fatalf("picoloom.Input.%s classification must explain its wiring", field)
		}
		if classification.requiresBuilderParity && !strings.HasPrefix(classification.reason, "build") {
			t.Fatalf("picoloom.Input.%s requires builder parity but reason %q does not name a builder", field, classification.reason)
		}
	}

	for field := range classifications {
		if _, ok := inputType.FieldByName(field); !ok {
			t.Fatalf("stale Input field classification %q; update this guardrail", field)
		}
	}
}

func TestConfigBuildersProduceValidPublicTypes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		cfg   *Config
		build func(*Config) publicConfigValidator
	}{
		{
			name: "footer position accepted by config and public type",
			cfg: &Config{
				Footer: FooterConfig{Enabled: true, Position: "center", ShowPageNumber: true, Text: "Footer"},
			},
			build: func(cfg *Config) publicConfigValidator {
				return buildFooterData(cfg, false)
			},
		},
		{
			name: "signature URL image accepted by config and public type",
			cfg: &Config{
				Author:    AuthorConfig{Name: "Jane", Title: "Writer", Email: "jane@example.com"},
				Signature: SignatureConfig{Enabled: true, ImagePath: "https://example.com/signature.png"},
			},
			build: func(cfg *Config) publicConfigValidator {
				return buildSignatureData(cfg, false)
			},
		},
		{
			name: "cover URL logo accepted by config and public type",
			cfg: &Config{
				Author:   AuthorConfig{Name: "Jane"},
				Document: DocumentConfig{Title: "Report"},
				Cover:    CoverConfig{Enabled: true, Logo: "https://example.com/logo.png"},
			},
			build: func(cfg *Config) publicConfigValidator {
				return buildCoverData(cfg, "# Ignored", "report.md")
			},
		},
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

func TestMergeAndValidateRuntimeConfig(t *testing.T) {
	t.Parallel()

	longAuthorName := strings.Repeat("a", configpkg.MaxNameLength+1)

	tests := []struct {
		name            string
		cfg             *Config
		flags           *convertFlags
		wantErrIs       error
		wantErrContains string
	}{
		{
			name:      "rejects CLI field length after merge",
			cfg:       configpkg.DefaultConfig(),
			flags:     &convertFlags{author: authorFlags{name: longAuthorName}},
			wantErrIs: configpkg.ErrFieldTooLong,
		},
		{
			name:            "rejects env-filled invalid page size when not overridden",
			cfg:             &Config{Page: PageConfig{Size: "invalid-env-page"}},
			flags:           &convertFlags{},
			wantErrContains: "page.size",
		},
		{
			name:  "allows CLI override to repair env-filled page size",
			cfg:   &Config{Page: PageConfig{Size: "invalid-env-page"}},
			flags: &convertFlags{page: pageFlags{size: picoloom.PageSizeA4}},
		},
		{
			name:            "rejects CLI TOC depth after merge",
			cfg:             configpkg.DefaultConfig(),
			flags:           &convertFlags{toc: tocFlags{minDepth: 7}},
			wantErrContains: "toc.minDepth",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := mergeAndValidateRuntimeConfig(tt.flags, tt.cfg)
			if tt.wantErrIs == nil && tt.wantErrContains == "" {
				if err != nil {
					t.Fatalf("mergeAndValidateRuntimeConfig() error = %v, want nil", err)
				}
				return
			}
			if err == nil {
				t.Fatal("mergeAndValidateRuntimeConfig() error = nil, want error")
			}
			if tt.wantErrIs != nil && !errors.Is(err, tt.wantErrIs) {
				t.Fatalf("mergeAndValidateRuntimeConfig() error = %v, want errors.Is(%v)", err, tt.wantErrIs)
			}
			if tt.wantErrContains != "" && !strings.Contains(err.Error(), tt.wantErrContains) {
				t.Fatalf("mergeAndValidateRuntimeConfig() error = %v, want containing %q", err, tt.wantErrContains)
			}
		})
	}
}
