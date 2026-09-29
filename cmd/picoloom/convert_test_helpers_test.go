package main

// Notes:
// - This file contains test helpers and type aliases used across convert tests.
// - These are not functions under test themselves, but supporting infrastructure.
// No coverage gaps: this is test infrastructure, not production code.

import (
	"context"
	"fmt"

	picoloom "github.com/alnah/picoloom/v2"
	"github.com/alnah/picoloom/v2/internal/config"
)

// Type aliases for cleaner test code.
type (
	Config           = config.Config
	InputConfig      = config.InputConfig
	OutputConfig     = config.OutputConfig
	SignatureConfig  = config.SignatureConfig
	FooterConfig     = config.FooterConfig
	AuthorConfig     = config.AuthorConfig
	DocumentConfig   = config.DocumentConfig
	PageConfig       = config.PageConfig
	WatermarkConfig  = config.WatermarkConfig
	CoverConfig      = config.CoverConfig
	TOCConfig        = config.TOCConfig
	PageBreaksConfig = config.PageBreaksConfig
	Link             = config.Link
)

// cliFlags is an alias for convertFlags (backward compatibility for tests).
type cliFlags = convertFlags

// parseFlags is a compatibility wrapper that simulates CLI invocation.
// Unlike parseConvertFlags, it expects args[0] to be the program name (e.g., "md2pdf")
// and skips it before parsing. This matches how os.Args works in production.
//
// Example: parseFlags([]string{"md2pdf", "--verbose", "doc.md"})
// is equivalent to: parseConvertFlags([]string{"--verbose", "doc.md"})
func parseFlags(args []string) (*convertFlags, []string, error) {
	if len(args) > 0 {
		return parseConvertFlags(args[1:])
	}
	return parseConvertFlags(args)
}

// printResults is a compatibility wrapper for tests.
func printResults(results []ConversionResult, quiet, verbose bool) int {
	env := DefaultEnv()
	return printResultsWithWriter(results, quiet, verbose, env)
}

// staticMockConverter is a simple mock that returns a fixed result.
type staticMockConverter struct {
	result []byte
	err    error
}

func (m *staticMockConverter) Convert(_ context.Context, _ picoloom.Input) (*picoloom.ConvertResult, error) {
	if m.err != nil {
		return nil, m.err
	}
	return &picoloom.ConvertResult{PDF: m.result}, nil
}

// capturingMockConverter captures the Input for inspection in tests.
type capturingMockConverter struct {
	result      []byte
	err         error
	capturedIn  picoloom.Input
	convertFunc func(picoloom.Input) (*picoloom.ConvertResult, error)
}

func (m *capturingMockConverter) Convert(_ context.Context, input picoloom.Input) (*picoloom.ConvertResult, error) {
	m.capturedIn = input
	if m.convertFunc != nil {
		return m.convertFunc(input)
	}
	if m.err != nil {
		return nil, m.err
	}
	return &picoloom.ConvertResult{PDF: m.result}, nil
}

// mockTemplateLoader implements picoloom.AssetLoader for testing resolveTemplateSet.
type mockTemplateLoader struct {
	templateSets map[string]*picoloom.TemplateSet
	err          error
}

func (m *mockTemplateLoader) LoadStyle(_ string) (string, error) {
	return "", nil
}

func (m *mockTemplateLoader) LoadTemplateSet(name string) (*picoloom.TemplateSet, error) {
	if m.err != nil {
		return nil, m.err
	}
	if ts, ok := m.templateSets[name]; ok {
		return ts, nil
	}
	return nil, fmt.Errorf("%w: %q", picoloom.ErrTemplateSetNotFound, name)
}
