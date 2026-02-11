package md2pdf

// Notes:
// - Tests Service.Convert with mocked pipeline components to isolate unit logic
// - Mock implementations (mockPreprocessor, mockHTMLConverter, etc.) allow testing
//   error handling and data flow without real browser or file system access
// - Internal test options (withPreprocessor, etc.) enable dependency injection
// - Validation tests cover all Input fields and their error conditions

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/alnah/go-md2pdf/internal/pipeline"
)

// ---------------------------------------------------------------------------
// Mock Implementations
// ---------------------------------------------------------------------------

type mockPreprocessor struct {
	called bool
	input  string
	output string
}

func (m *mockPreprocessor) PreprocessMarkdown(ctx context.Context, content string) string {
	m.called = true
	m.input = content
	if m.output != "" {
		return m.output
	}
	return content
}

type mockHTMLConverter struct {
	called bool
	input  string
	output string
	err    error
}

func (m *mockHTMLConverter) ToHTML(ctx context.Context, content string) (string, error) {
	m.called = true
	m.input = content
	if m.err != nil {
		return "", m.err
	}
	if m.output != "" {
		return m.output, nil
	}
	return "<html>" + content + "</html>", nil
}

type mockCSSInjector struct {
	called    bool
	inputHTML string
	inputCSS  string
	output    string
}

func (m *mockCSSInjector) InjectCSS(ctx context.Context, htmlContent, cssContent string) string {
	m.called = true
	m.inputHTML = htmlContent
	m.inputCSS = cssContent
	if m.output != "" {
		return m.output
	}
	return htmlContent
}

type mockPDFConverter struct {
	called    bool
	inputHTML string
	inputOpts *pdfOptions
	output    []byte
	err       error
}

func (m *mockPDFConverter) ToPDF(ctx context.Context, htmlContent string, opts *pdfOptions) ([]byte, error) {
	m.called = true
	m.inputHTML = htmlContent
	m.inputOpts = opts
	if m.err != nil {
		return nil, m.err
	}
	if m.output != nil {
		return m.output, nil
	}
	return []byte("%PDF-1.4 mock"), nil
}

func (m *mockPDFConverter) Close() error {
	return nil
}

type mockSignatureInjector struct {
	called    bool
	inputHTML string
	inputData *pipeline.SignatureData
	output    string
	err       error
}

func (m *mockSignatureInjector) InjectSignature(ctx context.Context, htmlContent string, data *pipeline.SignatureData) (string, error) {
	m.called = true
	m.inputHTML = htmlContent
	m.inputData = data
	if m.err != nil {
		return "", m.err
	}
	if m.output != "" {
		return m.output, nil
	}
	return htmlContent, nil
}

type mockCoverInjector struct {
	called    bool
	inputHTML string
	inputData *pipeline.CoverData
	output    string
	err       error
}

func (m *mockCoverInjector) InjectCover(ctx context.Context, htmlContent string, data *pipeline.CoverData) (string, error) {
	m.called = true
	m.inputHTML = htmlContent
	m.inputData = data
	if m.err != nil {
		return "", m.err
	}
	if m.output != "" {
		return m.output, nil
	}
	return htmlContent, nil
}

type mockTOCInjector struct {
	called    bool
	inputHTML string
	inputData *pipeline.TOCData
	output    string
	err       error
}

func (m *mockTOCInjector) InjectTOC(ctx context.Context, htmlContent string, data *pipeline.TOCData) (string, error) {
	m.called = true
	m.inputHTML = htmlContent
	m.inputData = data
	if m.err != nil {
		return "", m.err
	}
	if m.output != "" {
		return m.output, nil
	}
	return htmlContent, nil
}

type panicPreprocessor struct{}

func (p *panicPreprocessor) PreprocessMarkdown(ctx context.Context, content string) string {
	panic("simulated panic in preprocessor")
}

type mockAssetLoader struct {
	styleContent   string
	styleErr       error
	templateSet    *TemplateSet
	templateSetErr error
}

func (m *mockAssetLoader) LoadStyle(name string) (string, error) {
	if m.styleErr != nil {
		return "", m.styleErr
	}
	return m.styleContent, nil
}

func (m *mockAssetLoader) LoadTemplateSet(name string) (*TemplateSet, error) {
	if m.templateSetErr != nil {
		return nil, m.templateSetErr
	}
	if m.templateSet != nil {
		return m.templateSet, nil
	}
	// Return a minimal valid template set
	return &TemplateSet{
		Name:      name,
		Cover:     "<div>cover</div>",
		Signature: "<div>signature</div>",
	}, nil
}

// ---------------------------------------------------------------------------
// Test Options (Internal Dependency Injection)
// ---------------------------------------------------------------------------

func withPreprocessor(p pipeline.MarkdownPreprocessor) Option {
	return func(s *Service) {
		s.preprocessor = p
	}
}

func withHTMLConverter(c pipeline.HTMLConverter) Option {
	return func(s *Service) {
		s.htmlConverter = c
	}
}

func withCSSInjector(c pipeline.CSSInjector) Option {
	return func(s *Service) {
		s.cssInjector = c
	}
}

func withSignatureInjector(i pipeline.SignatureInjector) Option {
	return func(s *Service) {
		s.signatureInjector = i
	}
}

func withPDFConverter(c pdfConverter) Option {
	return func(s *Service) {
		s.pdfConverter = c
	}
}

func withCoverInjector(c pipeline.CoverInjector) Option {
	return func(s *Service) {
		s.coverInjector = c
	}
}

func withTOCInjector(t pipeline.TOCInjector) Option {
	return func(s *Service) {
		s.tocInjector = t
	}
}

// ---------------------------------------------------------------------------
// TestValidateInput - Input Validation
// ---------------------------------------------------------------------------

func TestValidateInput(t *testing.T) {
	t.Parallel()

	service, err := New()
	if err != nil {
		t.Fatalf("failed to create service: %v", err)
	}
	defer service.Close()

	tests := []struct {
		name    string
		input   Input
		wantErr error
	}{
		{
			name:    "valid input",
			input:   Input{Markdown: "# Hello"},
			wantErr: nil,
		},
		{
			name:    "empty markdown",
			input:   Input{Markdown: ""},
			wantErr: ErrEmptyMarkdown,
		},
		{
			name:    "with CSS",
			input:   Input{Markdown: "# Hello", CSS: "body { color: red; }"},
			wantErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := service.validateInput(tt.input)
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("validateInput() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestConvert_Success - Successful Conversion Pipeline
// ---------------------------------------------------------------------------

func TestConvert_Success(t *testing.T) {
	t.Parallel()

	preprocessor := &mockPreprocessor{output: "preprocessed"}
	htmlConv := &mockHTMLConverter{output: "<html>converted</html>"}
	cssInj := &mockCSSInjector{output: "<html>with-css</html>"}
	sigInjector := &mockSignatureInjector{output: "<html>with-sig</html>"}
	pdfConv := &mockPDFConverter{output: []byte("%PDF-1.4 test")}

	service, err := New(
		withPreprocessor(preprocessor),
		withHTMLConverter(htmlConv),
		withCSSInjector(cssInj),
		withSignatureInjector(sigInjector),
		withPDFConverter(pdfConv),
	)
	if err != nil {
		t.Fatalf("failed to create service: %v", err)
	}
	defer service.Close()

	input := Input{
		Markdown: "# Hello",
		CSS:      "body {}",
	}

	ctx := context.Background()
	result, err := service.Convert(ctx, input)
	if err != nil {
		t.Fatalf("Convert() unexpected error: %v", err)
	}

	if string(result.PDF) != "%PDF-1.4 test" {
		t.Errorf("Convert() result.PDF = %q, want %q", result.PDF, "%PDF-1.4 test")
	}

	// Verify pipeline was called in order with correct inputs
	if !preprocessor.called {
		t.Error("preprocessor was not called")
	}
	if preprocessor.input != "# Hello" {
		t.Errorf("preprocessor input = %q, want %q", preprocessor.input, "# Hello")
	}

	if !htmlConv.called {
		t.Error("htmlConverter was not called")
	}
	if htmlConv.input != "preprocessed" {
		t.Errorf("htmlConverter input = %q, want %q", htmlConv.input, "preprocessed")
	}

	if !cssInj.called {
		t.Error("pipeline.CSSInjector was not called")
	}
	if cssInj.inputHTML != "<html>converted</html>" {
		t.Errorf("pipeline.CSSInjector inputHTML = %q, want %q", cssInj.inputHTML, "<html>converted</html>")
	}
	// Page breaks CSS is always prepended, user CSS should be at the end
	if !strings.HasSuffix(cssInj.inputCSS, "body {}") {
		t.Errorf("pipeline.CSSInjector inputCSS should end with user CSS %q, got %q", "body {}", cssInj.inputCSS)
	}
	// Verify page breaks CSS is present
	if !strings.Contains(cssInj.inputCSS, "break-after: avoid") {
		t.Errorf("pipeline.CSSInjector inputCSS should contain page breaks CSS, got %q", cssInj.inputCSS)
	}

	if !sigInjector.called {
		t.Error("pipeline.SignatureInjector was not called")
	}
	if sigInjector.inputHTML != "<html>with-css</html>" {
		t.Errorf("pipeline.SignatureInjector inputHTML = %q, want %q", sigInjector.inputHTML, "<html>with-css</html>")
	}

	if !pdfConv.called {
		t.Error("pdfConverter was not called")
	}
	if pdfConv.inputHTML != "<html>with-sig</html>" {
		t.Errorf("pdfConverter inputHTML = %q, want %q", pdfConv.inputHTML, "<html>with-sig</html>")
	}
}

// ---------------------------------------------------------------------------
// TestConvert_ValidationError - Validation Error Handling
// ---------------------------------------------------------------------------

func TestConvert_ValidationError(t *testing.T) {
	t.Parallel()

	service, err := New()
	if err != nil {
		t.Fatalf("failed to create service: %v", err)
	}
	defer service.Close()

	ctx := context.Background()
	_, err = service.Convert(ctx, Input{Markdown: ""})

	if !errors.Is(err, ErrEmptyMarkdown) {
		t.Errorf("Convert() error = %v, want %v", err, ErrEmptyMarkdown)
	}
}

// ---------------------------------------------------------------------------
// TestConvert_HTMLConverterError - HTML Converter Error Handling
// ---------------------------------------------------------------------------

func TestConvert_HTMLConverterError(t *testing.T) {
	t.Parallel()

	htmlErr := errors.New("pandoc failed")

	service, err := New(
		withPreprocessor(&mockPreprocessor{}),
		withHTMLConverter(&mockHTMLConverter{err: htmlErr}),
		withCSSInjector(&mockCSSInjector{}),
		withSignatureInjector(&mockSignatureInjector{}),
		withPDFConverter(&mockPDFConverter{}),
	)
	if err != nil {
		t.Fatalf("failed to create service: %v", err)
	}
	defer service.Close()

	ctx := context.Background()
	_, err = service.Convert(ctx, Input{Markdown: "# Hello"})

	if err == nil {
		t.Fatal("Convert() expected error, got nil")
	}
	if !errors.Is(err, htmlErr) {
		t.Errorf("Convert() error should wrap %v, got %v", htmlErr, err)
	}
}

// ---------------------------------------------------------------------------
// TestConvert_PDFConverterError - PDF Converter Error Handling
// ---------------------------------------------------------------------------

func TestConvert_PDFConverterError(t *testing.T) {
	t.Parallel()

	pdfErr := errors.New("chrome failed")

	service, err := New(
		withPreprocessor(&mockPreprocessor{}),
		withHTMLConverter(&mockHTMLConverter{}),
		withCSSInjector(&mockCSSInjector{}),
		withSignatureInjector(&mockSignatureInjector{}),
		withPDFConverter(&mockPDFConverter{err: pdfErr}),
	)
	if err != nil {
		t.Fatalf("failed to create service: %v", err)
	}
	defer service.Close()

	ctx := context.Background()
	_, err = service.Convert(ctx, Input{Markdown: "# Hello"})

	if err == nil {
		t.Fatal("Convert() expected error, got nil")
	}
	if !errors.Is(err, pdfErr) {
		t.Errorf("Convert() error should wrap %v, got %v", pdfErr, err)
	}
}

// ---------------------------------------------------------------------------
// TestConvert_SignatureInjectorError - Signature Injector Error Handling
// ---------------------------------------------------------------------------

func TestConvert_SignatureInjectorError(t *testing.T) {
	t.Parallel()

	sigErr := errors.New("signature template failed")

	service, err := New(
		withPreprocessor(&mockPreprocessor{}),
		withHTMLConverter(&mockHTMLConverter{}),
		withCSSInjector(&mockCSSInjector{}),
		withSignatureInjector(&mockSignatureInjector{err: sigErr}),
		withPDFConverter(&mockPDFConverter{}),
	)
	if err != nil {
		t.Fatalf("failed to create service: %v", err)
	}
	defer service.Close()

	ctx := context.Background()
	_, err = service.Convert(ctx, Input{Markdown: "# Hello"})

	if err == nil {
		t.Fatal("Convert() expected error, got nil")
	}
	if !errors.Is(err, sigErr) {
		t.Errorf("Convert() error should wrap %v, got %v", sigErr, err)
	}
}

// ---------------------------------------------------------------------------
// TestConvert_NoCSSByDefault - Default CSS Behavior
// ---------------------------------------------------------------------------

func TestConvert_NoCSSByDefault(t *testing.T) {
	t.Parallel()

	cssInj := &mockCSSInjector{}

	service, err := New(
		withPreprocessor(&mockPreprocessor{}),
		withHTMLConverter(&mockHTMLConverter{}),
		withCSSInjector(cssInj),
		withSignatureInjector(&mockSignatureInjector{}),
		withPDFConverter(&mockPDFConverter{}),
	)
	if err != nil {
		t.Fatalf("failed to create service: %v", err)
	}
	defer service.Close()

	ctx := context.Background()
	_, err = service.Convert(ctx, Input{Markdown: "# Hello"})

	if err != nil {
		t.Fatalf("Convert() unexpected error: %v", err)
	}

	// Page breaks CSS is always generated, so we should get at least that
	if !strings.Contains(cssInj.inputCSS, "break-after: avoid") {
		t.Errorf("pipeline.CSSInjector should receive page breaks CSS by default, got %q", cssInj.inputCSS)
	}
	// But no user CSS should be appended
	if strings.Contains(cssInj.inputCSS, "body") {
		t.Errorf("pipeline.CSSInjector should not contain user CSS rules, got %q", cssInj.inputCSS)
	}
}

// ---------------------------------------------------------------------------
// TestNew - Service Factory
// ---------------------------------------------------------------------------

func TestNew(t *testing.T) {
	t.Parallel()

	service, err := New()
	if err != nil {
		t.Fatalf("failed to create service: %v", err)
	}
	defer service.Close()

	if service.preprocessor == nil {
		t.Error("preprocessor is nil")
	}
	if service.htmlConverter == nil {
		t.Error("htmlConverter is nil")
	}
	if service.cssInjector == nil {
		t.Error("cssInjector is nil")
	}
	if service.signatureInjector == nil {
		t.Error("signatureInjector is nil")
	}
	if service.pdfConverter == nil {
		t.Error("pdfConverter is nil")
	}
}

// ---------------------------------------------------------------------------
// TestWithTimeout - Timeout Option
// ---------------------------------------------------------------------------

func TestWithTimeout(t *testing.T) {
	t.Parallel()

	service, err := New(WithTimeout(60 * defaultTimeout))
	if err != nil {
		t.Fatalf("failed to create service: %v", err)
	}
	defer service.Close()

	if service.cfg.timeout != 60*defaultTimeout {
		t.Errorf("timeout = %v, want %v", service.cfg.timeout, 60*defaultTimeout)
	}
}

// ---------------------------------------------------------------------------
// TestWithAssetLoader - Asset Loader Option
// ---------------------------------------------------------------------------

func TestWithAssetLoader(t *testing.T) {
	t.Parallel()

	customLoader := &mockAssetLoader{
		styleContent: "/* custom */",
		templateSet: &TemplateSet{
			Name:      "custom",
			Cover:     "<div>custom cover</div>",
			Signature: "<div>custom signature</div>",
		},
	}

	service, err := New(WithAssetLoader(customLoader))
	if err != nil {
		t.Fatalf("failed to create service: %v", err)
	}
	defer service.Close()

	if service.publicAssetLoader != customLoader {
		t.Error("publicAssetLoader should be the custom loader")
	}
}

// ---------------------------------------------------------------------------
// TestWithAssetLoader_UsedByInjectors - Asset Loader Injector Integration
// ---------------------------------------------------------------------------

func TestWithAssetLoader_UsedByInjectors(t *testing.T) {
	t.Parallel()

	// Test that the asset loader is used when creating cover and signature injectors.
	// We use a mock loader that returns valid templates.
	loader := &mockAssetLoader{
		templateSet: NewTemplateSet("test", "<div>cover</div>", "<div>sig</div>"),
	}

	service, err := New(WithAssetLoader(loader))
	if err != nil {
		t.Fatalf("failed to create service: %v", err)
	}
	defer service.Close()

	// Service should have initialized its injectors using the loader
	if service.coverInjector == nil {
		t.Error("coverInjector should not be nil")
	}
	if service.signatureInjector == nil {
		t.Error("signatureInjector should not be nil")
	}
}

// ---------------------------------------------------------------------------
// TestService_Close - Service Cleanup
// ---------------------------------------------------------------------------

func TestService_Close(t *testing.T) {
	t.Parallel()

	service, err := New()
	if err != nil {
		t.Fatalf("failed to create service: %v", err)
	}

	// Close should not error
	if err := service.Close(); err != nil {
		t.Errorf("Close() error = %v", err)
	}

	// Double close should also not error
	if err := service.Close(); err != nil {
		t.Errorf("Close() second call error = %v", err)
	}
}

// ---------------------------------------------------------------------------
// TestToSignatureData - Signature Data Conversion
// ---------------------------------------------------------------------------

func TestToSignatureData(t *testing.T) {
	t.Parallel()

	t.Run("nil returns nil", func(t *testing.T) {
		t.Parallel()
		result := toSignatureData(nil)
		if result != nil {
			t.Error("expected nil for nil input")
		}
	})

	t.Run("converts all fields", func(t *testing.T) {
		t.Parallel()

		sig := &Signature{
			Name:      "John Doe",
			Title:     "Developer",
			Email:     "john@example.com",
			ImagePath: "/path/to/image.png",
			Links: []Link{
				{Label: "GitHub", URL: "https://github.com/john"},
			},
		}

		result := toSignatureData(sig)

		if result.Name != sig.Name {
			t.Errorf("Name = %q, want %q", result.Name, sig.Name)
		}
		if result.Title != sig.Title {
			t.Errorf("Title = %q, want %q", result.Title, sig.Title)
		}
		if result.Email != sig.Email {
			t.Errorf("Email = %q, want %q", result.Email, sig.Email)
		}
		if result.ImagePath != sig.ImagePath {
			t.Errorf("ImagePath = %q, want %q", result.ImagePath, sig.ImagePath)
		}
		if len(result.Links) != 1 {
			t.Fatalf("Links count = %d, want 1", len(result.Links))
		}
		if result.Links[0].Label != "GitHub" || result.Links[0].URL != "https://github.com/john" {
			t.Errorf("Links[0] = %+v, want {GitHub, https://github.com/john}", result.Links[0])
		}
	})

	t.Run("converts extended metadata fields", func(t *testing.T) {
		t.Parallel()

		sig := &Signature{
			Name:       "Jane Smith",
			Phone:      "+1-555-123-4567",
			Address:    "123 Main St\nCity, State 12345",
			Department: "Engineering",
		}

		result := toSignatureData(sig)

		if result.Phone != sig.Phone {
			t.Errorf("Phone = %q, want %q", result.Phone, sig.Phone)
		}
		if result.Address != sig.Address {
			t.Errorf("Address = %q, want %q", result.Address, sig.Address)
		}
		if result.Department != sig.Department {
			t.Errorf("Department = %q, want %q", result.Department, sig.Department)
		}
	})
}

// ---------------------------------------------------------------------------
// TestToFooterData - Footer Data Conversion
// ---------------------------------------------------------------------------

func TestToFooterData(t *testing.T) {
	t.Parallel()

	t.Run("nil returns nil", func(t *testing.T) {
		t.Parallel()
		result := toFooterData(nil)
		if result != nil {
			t.Error("expected nil for nil input")
		}
	})

	t.Run("converts all fields", func(t *testing.T) {
		t.Parallel()

		footer := &Footer{
			Position:       "center",
			ShowPageNumber: true,
			Date:           "2025-01-15",
			Status:         "DRAFT",
			Text:           "Footer",
		}

		result := toFooterData(footer)

		if result.Position != footer.Position {
			t.Errorf("Position = %q, want %q", result.Position, footer.Position)
		}
		if result.ShowPageNumber != footer.ShowPageNumber {
			t.Errorf("ShowPageNumber = %v, want %v", result.ShowPageNumber, footer.ShowPageNumber)
		}
		if result.Date != footer.Date {
			t.Errorf("Date = %q, want %q", result.Date, footer.Date)
		}
		if result.Status != footer.Status {
			t.Errorf("Status = %q, want %q", result.Status, footer.Status)
		}
		if result.Text != footer.Text {
			t.Errorf("Text = %q, want %q", result.Text, footer.Text)
		}
	})

	t.Run("converts DocumentID field", func(t *testing.T) {
		t.Parallel()

		footer := &Footer{
			Position:   "right",
			DocumentID: "DOC-2024-001",
		}

		result := toFooterData(footer)

		if result.DocumentID != footer.DocumentID {
			t.Errorf("DocumentID = %q, want %q", result.DocumentID, footer.DocumentID)
		}
	})
}

// ---------------------------------------------------------------------------
// TestToCoverData - Cover Data Conversion
// ---------------------------------------------------------------------------

func TestToCoverData(t *testing.T) {
	t.Parallel()

	t.Run("nil returns nil", func(t *testing.T) {
		t.Parallel()
		result := toCoverData(nil)
		if result != nil {
			t.Error("expected nil for nil input")
		}
	})

	t.Run("converts all fields", func(t *testing.T) {
		t.Parallel()

		cover := &Cover{
			Title:        "My Document",
			Subtitle:     "A Comprehensive Guide",
			Logo:         "/path/to/logo.png",
			Author:       "John Doe",
			AuthorTitle:  "Senior Developer",
			Organization: "Acme Corp",
			Date:         "2025-01-15",
			Version:      "v1.0.0",
		}

		result := toCoverData(cover)

		if result.Title != cover.Title {
			t.Errorf("Title = %q, want %q", result.Title, cover.Title)
		}
		if result.Subtitle != cover.Subtitle {
			t.Errorf("Subtitle = %q, want %q", result.Subtitle, cover.Subtitle)
		}
		if result.Logo != cover.Logo {
			t.Errorf("Logo = %q, want %q", result.Logo, cover.Logo)
		}
		if result.Author != cover.Author {
			t.Errorf("Author = %q, want %q", result.Author, cover.Author)
		}
		if result.AuthorTitle != cover.AuthorTitle {
			t.Errorf("AuthorTitle = %q, want %q", result.AuthorTitle, cover.AuthorTitle)
		}
		if result.Organization != cover.Organization {
			t.Errorf("Organization = %q, want %q", result.Organization, cover.Organization)
		}
		if result.Date != cover.Date {
			t.Errorf("Date = %q, want %q", result.Date, cover.Date)
		}
		if result.Version != cover.Version {
			t.Errorf("Version = %q, want %q", result.Version, cover.Version)
		}
	})

	t.Run("empty fields preserved", func(t *testing.T) {
		t.Parallel()

		cover := &Cover{
			Title: "Only Title",
			// All other fields empty
		}

		result := toCoverData(cover)

		if result.Title != "Only Title" {
			t.Errorf("Title = %q, want %q", result.Title, "Only Title")
		}
		if result.Subtitle != "" {
			t.Errorf("Subtitle = %q, want empty", result.Subtitle)
		}
		if result.Logo != "" {
			t.Errorf("Logo = %q, want empty", result.Logo)
		}
		if result.Author != "" {
			t.Errorf("Author = %q, want empty", result.Author)
		}
	})

	t.Run("converts extended metadata fields", func(t *testing.T) {
		t.Parallel()

		cover := &Cover{
			Title:        "Project Spec",
			ClientName:   "Acme Corporation",
			ProjectName:  "Project Phoenix",
			DocumentType: "Technical Specification",
			DocumentID:   "DOC-2024-001",
			Description:  "System design document",
			Department:   "Engineering",
		}

		result := toCoverData(cover)

		if result.ClientName != cover.ClientName {
			t.Errorf("ClientName = %q, want %q", result.ClientName, cover.ClientName)
		}
		if result.ProjectName != cover.ProjectName {
			t.Errorf("ProjectName = %q, want %q", result.ProjectName, cover.ProjectName)
		}
		if result.DocumentType != cover.DocumentType {
			t.Errorf("DocumentType = %q, want %q", result.DocumentType, cover.DocumentType)
		}
		if result.DocumentID != cover.DocumentID {
			t.Errorf("DocumentID = %q, want %q", result.DocumentID, cover.DocumentID)
		}
		if result.Description != cover.Description {
			t.Errorf("Description = %q, want %q", result.Description, cover.Description)
		}
		if result.Department != cover.Department {
			t.Errorf("Department = %q, want %q", result.Department, cover.Department)
		}
	})
}

// ---------------------------------------------------------------------------
// TestToTOCData - TOC Data Conversion
// ---------------------------------------------------------------------------

func TestToTOCData(t *testing.T) {
	t.Parallel()

	t.Run("nil returns nil", func(t *testing.T) {
		t.Parallel()
		result := toTOCData(nil)
		if result != nil {
			t.Error("expected nil for nil input")
		}
	})

	t.Run("converts all fields", func(t *testing.T) {
		t.Parallel()

		toc := &TOC{
			Title:    "Table of Contents",
			MinDepth: 2,
			MaxDepth: 4,
		}

		result := toTOCData(toc)

		if result.Title != toc.Title {
			t.Errorf("Title = %q, want %q", result.Title, toc.Title)
		}
		if result.MinDepth != toc.MinDepth {
			t.Errorf("MinDepth = %d, want %d", result.MinDepth, toc.MinDepth)
		}
		if result.MaxDepth != toc.MaxDepth {
			t.Errorf("MaxDepth = %d, want %d", result.MaxDepth, toc.MaxDepth)
		}
	})

	t.Run("zero MinDepth gets default", func(t *testing.T) {
		t.Parallel()

		toc := &TOC{
			Title:    "Contents",
			MinDepth: 0,
			MaxDepth: 3,
		}

		result := toTOCData(toc)

		if result.MinDepth != DefaultTOCMinDepth {
			t.Errorf("MinDepth = %d, want %d (default)", result.MinDepth, DefaultTOCMinDepth)
		}
	})

	t.Run("zero MaxDepth gets default", func(t *testing.T) {
		t.Parallel()

		toc := &TOC{
			Title:    "Contents",
			MaxDepth: 0,
		}

		result := toTOCData(toc)

		if result.MaxDepth != DefaultTOCMaxDepth {
			t.Errorf("MaxDepth = %d, want %d (default)", result.MaxDepth, DefaultTOCMaxDepth)
		}
	})

	t.Run("empty title preserved", func(t *testing.T) {
		t.Parallel()

		toc := &TOC{
			Title:    "",
			MaxDepth: 3,
		}

		result := toTOCData(toc)

		if result.Title != "" {
			t.Errorf("Title = %q, want empty", result.Title)
		}
	})
}

// ---------------------------------------------------------------------------
// TestValidateInput_TOC - TOC Validation
// ---------------------------------------------------------------------------

func TestValidateInput_TOC(t *testing.T) {
	t.Parallel()

	service, err := New()
	if err != nil {
		t.Fatalf("failed to create service: %v", err)
	}
	defer service.Close()

	t.Run("nil TOC is valid", func(t *testing.T) {
		t.Parallel()
		input := Input{Markdown: "# Hello", TOC: nil}
		err := service.validateInput(input)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("valid TOC passes", func(t *testing.T) {
		t.Parallel()

		input := Input{
			Markdown: "# Hello",
			TOC:      &TOC{Title: "Contents", MaxDepth: 3},
		}
		err := service.validateInput(input)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("invalid TOC depth fails", func(t *testing.T) {
		t.Parallel()

		input := Input{
			Markdown: "# Hello",
			TOC:      &TOC{MaxDepth: 7},
		}
		err := service.validateInput(input)
		if err == nil {
			t.Fatal("expected error for invalid TOC depth")
		}
		if !errors.Is(err, ErrInvalidTOCDepth) {
			t.Errorf("error = %v, want ErrInvalidTOCDepth", err)
		}
	})
}

// ---------------------------------------------------------------------------
// TestConvert_RecoversPanic - Panic Recovery
// ---------------------------------------------------------------------------

func TestConvert_RecoversPanic(t *testing.T) {
	t.Parallel()

	service, err := New(
		withPreprocessor(&panicPreprocessor{}),
		withHTMLConverter(&mockHTMLConverter{}),
		withCSSInjector(&mockCSSInjector{}),
		withSignatureInjector(&mockSignatureInjector{}),
		withPDFConverter(&mockPDFConverter{}),
	)
	if err != nil {
		t.Fatalf("failed to create service: %v", err)
	}
	defer service.Close()

	ctx := context.Background()
	_, err = service.Convert(ctx, Input{Markdown: "# Test"})

	if err == nil {
		t.Fatal("expected error from panic recovery, got nil")
	}
	if !strings.Contains(err.Error(), "internal error") {
		t.Errorf("expected 'internal error' in message, got %q", err.Error())
	}
}

// ---------------------------------------------------------------------------
// TestConvert_ContextCancellation - Context Cancellation Handling
// ---------------------------------------------------------------------------

func TestConvert_ContextCancellation(t *testing.T) {
	t.Parallel()

	service, err := New(
		withPreprocessor(&mockPreprocessor{}),
		withHTMLConverter(&mockHTMLConverter{output: "<html></html>"}),
		withCSSInjector(&mockCSSInjector{}),
		withSignatureInjector(&mockSignatureInjector{}),
		withPDFConverter(&mockPDFConverter{}),
	)
	if err != nil {
		t.Fatalf("failed to create service: %v", err)
	}
	defer service.Close()

	// Cancel context before calling Convert
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err = service.Convert(ctx, Input{Markdown: "# Test"})

	if err == nil {
		t.Fatal("expected context error, got nil")
	}
	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled, got %v", err)
	}
}

// ---------------------------------------------------------------------------
// TestValidateInput_InvalidWatermark - Watermark Validation
// ---------------------------------------------------------------------------

func TestValidateInput_InvalidWatermark(t *testing.T) {
	t.Parallel()

	service, err := New()
	if err != nil {
		t.Fatalf("failed to create service: %v", err)
	}
	defer service.Close()

	tests := []struct {
		name      string
		watermark *Watermark
		wantErr   bool
	}{
		{
			name:      "opacity too high",
			watermark: &Watermark{Text: "DRAFT", Opacity: 1.5},
			wantErr:   true,
		},
		{
			name:      "opacity negative",
			watermark: &Watermark{Text: "DRAFT", Opacity: -0.1},
			wantErr:   true,
		},
		{
			name:      "angle too high",
			watermark: &Watermark{Text: "DRAFT", Opacity: 0.5, Angle: 100},
			wantErr:   true,
		},
		{
			name:      "angle too low",
			watermark: &Watermark{Text: "DRAFT", Opacity: 0.5, Angle: -100},
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			input := Input{Markdown: "# Test", Watermark: tt.watermark}
			err := service.validateInput(input)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateInput() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestValidateInput_InvalidPageBreaks - Page Breaks Validation
// ---------------------------------------------------------------------------

func TestValidateInput_InvalidPageBreaks(t *testing.T) {
	t.Parallel()

	service, err := New()
	if err != nil {
		t.Fatalf("failed to create service: %v", err)
	}
	defer service.Close()

	tests := []struct {
		name       string
		pageBreaks *PageBreaks
		wantErr    error
	}{
		{
			name:       "orphans too high",
			pageBreaks: &PageBreaks{Orphans: MaxOrphans + 1},
			wantErr:    ErrInvalidOrphans,
		},
		{
			name:       "widows too high",
			pageBreaks: &PageBreaks{Widows: MaxWidows + 1},
			wantErr:    ErrInvalidWidows,
		},
		{
			name:       "orphans negative",
			pageBreaks: &PageBreaks{Orphans: -1},
			wantErr:    ErrInvalidOrphans,
		},
		{
			name:       "widows negative",
			pageBreaks: &PageBreaks{Widows: -1},
			wantErr:    ErrInvalidWidows,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			input := Input{Markdown: "# Test", PageBreaks: tt.pageBreaks}
			err := service.validateInput(input)
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("validateInput() error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestService_CloseNilConverter - Close with Nil Converter
// ---------------------------------------------------------------------------

func TestService_CloseNilConverter(t *testing.T) {
	t.Parallel()

	service := &Service{
		pdfConverter: nil,
	}

	err := service.Close()
	if err != nil {
		t.Errorf("Close() with nil pdfConverter should not error, got %v", err)
	}
}

// ---------------------------------------------------------------------------
// TestConvert_WatermarkCSSOrder - CSS Ordering with Watermark
// ---------------------------------------------------------------------------

func TestConvert_WatermarkCSSOrder(t *testing.T) {
	t.Parallel()

	cssInj := &mockCSSInjector{}

	service, err := New(
		withPreprocessor(&mockPreprocessor{}),
		withHTMLConverter(&mockHTMLConverter{}),
		withCSSInjector(cssInj),
		withCoverInjector(&mockCoverInjector{}),
		withTOCInjector(&mockTOCInjector{}),
		withSignatureInjector(&mockSignatureInjector{}),
		withPDFConverter(&mockPDFConverter{}),
	)
	if err != nil {
		t.Fatalf("failed to create service: %v", err)
	}
	defer service.Close()

	ctx := context.Background()
	input := Input{
		Markdown: "# Test",
		CSS:      "body { color: blue; }",
		Watermark: &Watermark{
			Text:    "DRAFT",
			Color:   "#888888",
			Opacity: 0.1,
			Angle:   -45,
		},
	}

	_, err = service.Convert(ctx, input)
	if err != nil {
		t.Fatalf("Convert() unexpected error: %v", err)
	}

	css := cssInj.inputCSS

	// User CSS should be at the end
	if !strings.HasSuffix(css, "body { color: blue; }") {
		t.Errorf("user CSS should be at end, got %q", css)
	}

	// Watermark CSS should contain the watermark text
	if !strings.Contains(css, "DRAFT") {
		t.Errorf("CSS should contain watermark text 'DRAFT', got %q", css)
	}

	// Page breaks CSS should be present
	if !strings.Contains(css, "break-after: avoid") {
		t.Errorf("CSS should contain page breaks rules, got %q", css)
	}

	// Verify order: page breaks before watermark before user CSS
	pageBreaksIdx := strings.Index(css, "break-after")
	watermarkIdx := strings.Index(css, "DRAFT")
	userCSSIdx := strings.Index(css, "body { color: blue; }")

	if pageBreaksIdx > watermarkIdx {
		t.Errorf("page breaks CSS should come before watermark CSS")
	}
	if watermarkIdx > userCSSIdx {
		t.Errorf("watermark CSS should come before user CSS")
	}
}

// ---------------------------------------------------------------------------
// TestConvert_CoverInjectorError - Cover Injector Error Handling
// ---------------------------------------------------------------------------

func TestConvert_CoverInjectorError(t *testing.T) {
	t.Parallel()

	coverErr := errors.New("cover template failed")

	service, err := New(
		withPreprocessor(&mockPreprocessor{}),
		withHTMLConverter(&mockHTMLConverter{}),
		withCSSInjector(&mockCSSInjector{}),
		withCoverInjector(&mockCoverInjector{err: coverErr}),
		withTOCInjector(&mockTOCInjector{}),
		withSignatureInjector(&mockSignatureInjector{}),
		withPDFConverter(&mockPDFConverter{}),
	)
	if err != nil {
		t.Fatalf("failed to create service: %v", err)
	}
	defer service.Close()

	ctx := context.Background()
	_, err = service.Convert(ctx, Input{Markdown: "# Hello"})

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, coverErr) {
		t.Errorf("error should wrap %v, got %v", coverErr, err)
	}
	if !strings.Contains(err.Error(), "injecting cover") {
		t.Errorf("error should mention 'injecting cover', got %q", err.Error())
	}
}

// ---------------------------------------------------------------------------
// TestConvert_TOCInjectorError - TOC Injector Error Handling
// ---------------------------------------------------------------------------

func TestConvert_TOCInjectorError(t *testing.T) {
	t.Parallel()

	tocErr := errors.New("TOC generation failed")

	service, err := New(
		withPreprocessor(&mockPreprocessor{}),
		withHTMLConverter(&mockHTMLConverter{}),
		withCSSInjector(&mockCSSInjector{}),
		withCoverInjector(&mockCoverInjector{}),
		withTOCInjector(&mockTOCInjector{err: tocErr}),
		withSignatureInjector(&mockSignatureInjector{}),
		withPDFConverter(&mockPDFConverter{}),
	)
	if err != nil {
		t.Fatalf("failed to create service: %v", err)
	}
	defer service.Close()

	ctx := context.Background()
	_, err = service.Convert(ctx, Input{Markdown: "# Hello"})

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, tocErr) {
		t.Errorf("error should wrap %v, got %v", tocErr, err)
	}
	if !strings.Contains(err.Error(), "injecting TOC") {
		t.Errorf("error should mention 'injecting TOC', got %q", err.Error())
	}
}

// ---------------------------------------------------------------------------
// TestConvert_PDFOptionsTransmission - PDF Options Passing
// ---------------------------------------------------------------------------

func TestConvert_PDFOptionsTransmission(t *testing.T) {
	t.Parallel()

	pdfConv := &mockPDFConverter{}

	service, err := New(
		withPreprocessor(&mockPreprocessor{}),
		withHTMLConverter(&mockHTMLConverter{}),
		withCSSInjector(&mockCSSInjector{}),
		withCoverInjector(&mockCoverInjector{}),
		withTOCInjector(&mockTOCInjector{}),
		withSignatureInjector(&mockSignatureInjector{}),
		withPDFConverter(pdfConv),
	)
	if err != nil {
		t.Fatalf("failed to create service: %v", err)
	}
	defer service.Close()

	ctx := context.Background()
	input := Input{
		Markdown: "# Test",
		Page: &PageSettings{
			Size:        PageSizeA4,
			Orientation: OrientationLandscape,
			Margin:      1.5,
		},
		Footer: &Footer{
			Position:       "center",
			ShowPageNumber: true,
			Date:           "2025-01-15",
			Status:         "FINAL",
			Text:           "Confidential",
		},
	}

	_, err = service.Convert(ctx, input)
	if err != nil {
		t.Fatalf("Convert() unexpected error: %v", err)
	}

	if pdfConv.inputOpts == nil {
		t.Fatal("PDF options not passed to converter")
	}

	// Verify page settings
	if pdfConv.inputOpts.Page == nil {
		t.Fatal("Page settings not passed to PDF converter")
	}
	if pdfConv.inputOpts.Page.Size != PageSizeA4 {
		t.Errorf("Page.Size = %q, want %q", pdfConv.inputOpts.Page.Size, PageSizeA4)
	}
	if pdfConv.inputOpts.Page.Orientation != OrientationLandscape {
		t.Errorf("Page.Orientation = %q, want %q", pdfConv.inputOpts.Page.Orientation, OrientationLandscape)
	}

	// Verify footer data
	if pdfConv.inputOpts.Footer == nil {
		t.Fatal("Footer not passed to PDF converter")
	}
	if pdfConv.inputOpts.Footer.Position != "center" {
		t.Errorf("Footer.Position = %q, want %q", pdfConv.inputOpts.Footer.Position, "center")
	}
	if !pdfConv.inputOpts.Footer.ShowPageNumber {
		t.Error("Footer.ShowPageNumber = false, want true")
	}
}

// ---------------------------------------------------------------------------
// TestConvert_CoverDataTransmission - Cover Data Passing
// ---------------------------------------------------------------------------

func TestConvert_CoverDataTransmission(t *testing.T) {
	t.Parallel()

	coverInj := &mockCoverInjector{}

	service, err := New(
		withPreprocessor(&mockPreprocessor{}),
		withHTMLConverter(&mockHTMLConverter{}),
		withCSSInjector(&mockCSSInjector{}),
		withCoverInjector(coverInj),
		withTOCInjector(&mockTOCInjector{}),
		withSignatureInjector(&mockSignatureInjector{}),
		withPDFConverter(&mockPDFConverter{}),
	)
	if err != nil {
		t.Fatalf("failed to create service: %v", err)
	}
	defer service.Close()

	ctx := context.Background()
	input := Input{
		Markdown: "# Test",
		Cover: &Cover{
			Title:        "My Document",
			Subtitle:     "A Guide",
			Author:       "John Doe",
			AuthorTitle:  "Engineer",
			Organization: "Corp",
			Date:         "2025-01-15",
			Version:      "v1.0",
		},
	}

	_, err = service.Convert(ctx, input)
	if err != nil {
		t.Fatalf("Convert() unexpected error: %v", err)
	}

	if !coverInj.called {
		t.Fatal("cover injector was not called")
	}
	if coverInj.inputData == nil {
		t.Fatal("cover data not passed to injector")
	}
	if coverInj.inputData.Title != "My Document" {
		t.Errorf("Cover.Title = %q, want %q", coverInj.inputData.Title, "My Document")
	}
	if coverInj.inputData.Author != "John Doe" {
		t.Errorf("Cover.Author = %q, want %q", coverInj.inputData.Author, "John Doe")
	}
}

// ---------------------------------------------------------------------------
// TestConvert_TOCDataTransmission - TOC Data Passing
// ---------------------------------------------------------------------------

func TestConvert_TOCDataTransmission(t *testing.T) {
	t.Parallel()

	tocInj := &mockTOCInjector{}

	service, err := New(
		withPreprocessor(&mockPreprocessor{}),
		withHTMLConverter(&mockHTMLConverter{}),
		withCSSInjector(&mockCSSInjector{}),
		withCoverInjector(&mockCoverInjector{}),
		withTOCInjector(tocInj),
		withSignatureInjector(&mockSignatureInjector{}),
		withPDFConverter(&mockPDFConverter{}),
	)
	if err != nil {
		t.Fatalf("failed to create service: %v", err)
	}
	defer service.Close()

	ctx := context.Background()
	input := Input{
		Markdown: "# Test",
		TOC: &TOC{
			Title:    "Table of Contents",
			MaxDepth: 4,
		},
	}

	_, err = service.Convert(ctx, input)
	if err != nil {
		t.Fatalf("Convert() unexpected error: %v", err)
	}

	if !tocInj.called {
		t.Fatal("TOC injector was not called")
	}
	if tocInj.inputData == nil {
		t.Fatal("TOC data not passed to injector")
	}
	if tocInj.inputData.Title != "Table of Contents" {
		t.Errorf("TOC.Title = %q, want %q", tocInj.inputData.Title, "Table of Contents")
	}
	if tocInj.inputData.MaxDepth != 4 {
		t.Errorf("TOC.MaxDepth = %d, want %d", tocInj.inputData.MaxDepth, 4)
	}
}

// ---------------------------------------------------------------------------
// TestConvert_NilOptionalFieldsNotPassed - Nil Optional Fields Handling
// ---------------------------------------------------------------------------

func TestConvert_NilOptionalFieldsNotPassed(t *testing.T) {
	t.Parallel()

	coverInj := &mockCoverInjector{}
	tocInj := &mockTOCInjector{}

	service, err := New(
		withPreprocessor(&mockPreprocessor{}),
		withHTMLConverter(&mockHTMLConverter{}),
		withCSSInjector(&mockCSSInjector{}),
		withCoverInjector(coverInj),
		withTOCInjector(tocInj),
		withSignatureInjector(&mockSignatureInjector{}),
		withPDFConverter(&mockPDFConverter{}),
	)
	if err != nil {
		t.Fatalf("failed to create service: %v", err)
	}
	defer service.Close()

	ctx := context.Background()
	input := Input{
		Markdown: "# Test",
		Cover:    nil,
		TOC:      nil,
	}

	_, err = service.Convert(ctx, input)
	if err != nil {
		t.Fatalf("Convert() unexpected error: %v", err)
	}

	// Injectors should be called but with nil data
	if !coverInj.called {
		t.Fatal("cover injector should be called")
	}
	if coverInj.inputData != nil {
		t.Error("cover data should be nil when no cover provided")
	}

	if !tocInj.called {
		t.Fatal("TOC injector should be called")
	}
	if tocInj.inputData != nil {
		t.Error("TOC data should be nil when no TOC provided")
	}
}

// ---------------------------------------------------------------------------
// TestValidateInput_InvalidPage - Page Settings Validation
// ---------------------------------------------------------------------------

func TestValidateInput_InvalidPage(t *testing.T) {
	t.Parallel()

	service, err := New()
	if err != nil {
		t.Fatalf("failed to create service: %v", err)
	}
	defer service.Close()

	tests := []struct {
		name    string
		page    *PageSettings
		wantErr error
	}{
		{
			name:    "invalid size",
			page:    &PageSettings{Size: "invalid", Orientation: "portrait", Margin: 0.5},
			wantErr: ErrInvalidPageSize,
		},
		{
			name:    "invalid orientation",
			page:    &PageSettings{Size: "letter", Orientation: "diagonal", Margin: 0.5},
			wantErr: ErrInvalidOrientation,
		},
		{
			name:    "margin too small",
			page:    &PageSettings{Size: "letter", Orientation: "portrait", Margin: 0.1},
			wantErr: ErrInvalidMargin,
		},
		{
			name:    "margin too large",
			page:    &PageSettings{Size: "letter", Orientation: "portrait", Margin: 5.0},
			wantErr: ErrInvalidMargin,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			input := Input{Markdown: "# Test", Page: tt.page}
			err := service.validateInput(input)
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("validateInput() error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestValidateInput_InvalidFooter - Footer Validation
// ---------------------------------------------------------------------------

func TestValidateInput_InvalidFooter(t *testing.T) {
	t.Parallel()

	service, err := New()
	if err != nil {
		t.Fatalf("failed to create service: %v", err)
	}
	defer service.Close()

	input := Input{
		Markdown: "# Test",
		Footer:   &Footer{Position: "top"},
	}

	err = service.validateInput(input)
	if !errors.Is(err, ErrInvalidFooterPosition) {
		t.Errorf("validateInput() error = %v, want ErrInvalidFooterPosition", err)
	}
}

// ---------------------------------------------------------------------------
// TestValidateInput_InvalidWatermarkColor - Watermark Color Validation
// ---------------------------------------------------------------------------

func TestValidateInput_InvalidWatermarkColor(t *testing.T) {
	t.Parallel()

	service, err := New()
	if err != nil {
		t.Fatalf("failed to create service: %v", err)
	}
	defer service.Close()

	input := Input{
		Markdown:  "# Test",
		Watermark: &Watermark{Text: "DRAFT", Color: "red", Opacity: 0.1, Angle: -45},
	}

	err = service.validateInput(input)
	if !errors.Is(err, ErrInvalidWatermarkColor) {
		t.Errorf("validateInput() error = %v, want ErrInvalidWatermarkColor", err)
	}
}

// ---------------------------------------------------------------------------
// TestValidateInput_InvalidSignature - Signature Validation
// ---------------------------------------------------------------------------

func TestValidateInput_InvalidSignature(t *testing.T) {
	t.Parallel()

	service, err := New()
	if err != nil {
		t.Fatalf("failed to create service: %v", err)
	}
	defer service.Close()

	t.Run("nil signature is valid", func(t *testing.T) {
		t.Parallel()
		input := Input{Markdown: "# Hello", Signature: nil}
		err := service.validateInput(input)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("valid signature passes", func(t *testing.T) {
		t.Parallel()
		input := Input{
			Markdown:  "# Hello",
			Signature: &Signature{Name: "John Doe", Email: "john@example.com"},
		}
		err := service.validateInput(input)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("nonexistent image path fails", func(t *testing.T) {
		t.Parallel()
		input := Input{
			Markdown:  "# Hello",
			Signature: &Signature{ImagePath: "/nonexistent/path/to/signature.png"},
		}
		err := service.validateInput(input)
		if err == nil {
			t.Fatal("expected error for nonexistent image path")
		}
		if !errors.Is(err, ErrSignatureImageNotFound) {
			t.Errorf("error = %v, want ErrSignatureImageNotFound", err)
		}
	})

	t.Run("URL image path passes", func(t *testing.T) {
		t.Parallel()
		input := Input{
			Markdown:  "# Hello",
			Signature: &Signature{ImagePath: "https://example.com/signature.png"},
		}
		err := service.validateInput(input)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})
}

// ---------------------------------------------------------------------------
// TestConvert_ReturnsConvertResult - ConvertResult Structure
// ---------------------------------------------------------------------------

func TestConvert_ReturnsConvertResult(t *testing.T) {
	t.Parallel()

	mockPDF := &mockPDFConverter{output: []byte("%PDF-1.4 test")}
	mockHTML := &mockHTMLConverter{output: "<html><body>Test</body></html>"}

	service, err := New(
		withHTMLConverter(mockHTML),
		withPDFConverter(mockPDF),
	)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	result, err := service.Convert(context.Background(), Input{Markdown: "# Test"})
	if err != nil {
		t.Fatalf("Convert() error = %v", err)
	}

	// Verify ConvertResult contains both HTML and PDF
	if result == nil {
		t.Fatal("Convert() returned nil result")
	}
	if len(result.HTML) == 0 {
		t.Error("Convert() result.HTML is empty")
	}
	if len(result.PDF) == 0 {
		t.Error("Convert() result.PDF is empty")
	}
	if string(result.PDF) != "%PDF-1.4 test" {
		t.Errorf("Convert() result.PDF = %q, want %q", result.PDF, "%PDF-1.4 test")
	}
}

// ---------------------------------------------------------------------------
// TestConvert_HTMLOnlySkipsPDF - HTML Only Mode
// ---------------------------------------------------------------------------

func TestConvert_HTMLOnlySkipsPDF(t *testing.T) {
	t.Parallel()

	mockPDF := &mockPDFConverter{output: []byte("%PDF-1.4 test")}
	mockHTML := &mockHTMLConverter{output: "<html><body>HTMLOnly Test</body></html>"}

	service, err := New(
		withHTMLConverter(mockHTML),
		withPDFConverter(mockPDF),
	)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	result, err := service.Convert(context.Background(), Input{
		Markdown: "# Test",
		HTMLOnly: true,
	})
	if err != nil {
		t.Fatalf("Convert() error = %v", err)
	}

	// Verify HTML is populated but PDF is empty
	if len(result.HTML) == 0 {
		t.Error("Convert() result.HTML should not be empty in HTMLOnly mode")
	}
	if len(result.PDF) != 0 {
		t.Errorf("Convert() result.PDF should be empty in HTMLOnly mode, got %d bytes", len(result.PDF))
	}

	// Verify PDF converter was NOT called
	if mockPDF.called {
		t.Error("PDF converter should not be called in HTMLOnly mode")
	}
}

// ---------------------------------------------------------------------------
// TestConvert_HTMLOnlyStillProcessesInjections - HTML Only with Injections
// ---------------------------------------------------------------------------

func TestConvert_HTMLOnlyStillProcessesInjections(t *testing.T) {
	t.Parallel()

	mockPDF := &mockPDFConverter{}
	mockHTML := &mockHTMLConverter{output: "<html><body>Content</body></html>"}
	mockCSS := &mockCSSInjector{output: "<html><style>css</style><body>Content</body></html>"}

	service, err := New(
		withHTMLConverter(mockHTML),
		withPDFConverter(mockPDF),
		withCSSInjector(mockCSS),
	)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	result, err := service.Convert(context.Background(), Input{
		Markdown: "# Test",
		CSS:      "body { color: red; }",
		HTMLOnly: true,
	})
	if err != nil {
		t.Fatalf("Convert() error = %v", err)
	}

	// Verify CSS injection was called
	if !mockCSS.called {
		t.Error("CSS injector should still be called in HTMLOnly mode")
	}

	// Verify HTML contains injected CSS
	if !strings.Contains(string(result.HTML), "css") {
		t.Error("result.HTML should contain injected CSS")
	}
}

// ---------------------------------------------------------------------------
// TestWithTemplateSet - Template Set Option
// ---------------------------------------------------------------------------

func TestWithTemplateSet(t *testing.T) {
	t.Parallel()

	customTemplateSet := NewTemplateSet(
		"custom",
		"<div class=\"custom-cover\">{{.Title}}</div>",
		"<div class=\"custom-sig\">{{.Name}}</div>",
	)

	service, err := New(WithTemplateSet(customTemplateSet))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer service.Close()

	// Verify service was created with custom template set
	// The template set is used internally by injectors, so we verify
	// the service was created successfully
	if service == nil {
		t.Fatal("New(WithTemplateSet()) returned nil service")
	}
}

// ---------------------------------------------------------------------------
// TestWithTemplateSet_UsedByInjectors - Template Set Injector Integration
// ---------------------------------------------------------------------------

func TestWithTemplateSet_UsedByInjectors(t *testing.T) {
	t.Parallel()

	customTemplateSet := NewTemplateSet(
		"test-templates",
		"<section class=\"cover\"><div class=\"cover-page\"><p class=\"cover-title\">{{.Title}}</p></div></section><span data-cover-end></span>",
		"<div class=\"signature-block\"><div class=\"sig-person\"><strong>{{.Name}}</strong></div></div>",
	)

	mockPDF := &mockPDFConverter{output: []byte("%PDF-1.4")}

	service, err := New(
		WithTemplateSet(customTemplateSet),
		withPDFConverter(mockPDF),
	)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	// Test that cover injection uses the custom template
	result, err := service.Convert(context.Background(), Input{
		Markdown: "# Test",
		Cover: &Cover{
			Title: "Test Cover",
		},
	})
	if err != nil {
		t.Fatalf("Convert() error = %v", err)
	}

	// Verify the HTML contains content from our custom template
	htmlStr := string(result.HTML)
	if !strings.Contains(htmlStr, "Test Cover") {
		t.Error("result.HTML should contain cover title from custom template")
	}
}

// ---------------------------------------------------------------------------
// TestNew_WithoutTemplateSet_LoadsDefault - Default Template Set Loading
// ---------------------------------------------------------------------------

func TestNew_WithoutTemplateSet_LoadsDefault(t *testing.T) {
	t.Parallel()

	// When no WithTemplateSet is provided, Service should load the default template set
	service, err := New()
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer service.Close()

	if service == nil {
		t.Fatal("New() returned nil service")
	}
}

// ---------------------------------------------------------------------------
// TestWithAssetPath - Asset Path Option
// ---------------------------------------------------------------------------

func TestWithAssetPath(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	service, err := New(WithAssetPath(tmpDir))
	if err != nil {
		t.Fatalf("New(WithAssetPath) error = %v", err)
	}
	defer service.Close()

	if service.cfg.assetPath != tmpDir {
		t.Errorf("cfg.assetPath = %q, want %q", service.cfg.assetPath, tmpDir)
	}
}

// ---------------------------------------------------------------------------
// TestWithAssetPath_InvalidPath - Invalid Asset Path Handling
// ---------------------------------------------------------------------------

func TestWithAssetPath_InvalidPath(t *testing.T) {
	t.Parallel()

	_, err := New(WithAssetPath("/nonexistent/path/to/assets"))
	if err == nil {
		t.Fatal("New() expected error for invalid asset path, got nil")
	}
	if !errors.Is(err, ErrInvalidAssetPath) {
		t.Errorf("New() error = %v, want ErrInvalidAssetPath", err)
	}
}

// ---------------------------------------------------------------------------
// TestWithAssetPath_LoadsFromFilesystem - Filesystem Asset Loading
// ---------------------------------------------------------------------------

func TestWithAssetPath_LoadsFromFilesystem(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	// Service with custom path (even if empty dir) should still work via fallback
	service, err := New(WithAssetPath(tmpDir))
	if err != nil {
		t.Fatalf("New(WithAssetPath) error = %v", err)
	}
	defer service.Close()

	// Service should be usable - the test verifies no panic/error during creation
	if service == nil {
		t.Fatal("New() returned nil service")
	}
}

// ---------------------------------------------------------------------------
// TestWithStyle - Style Option
// ---------------------------------------------------------------------------

func TestWithStyle(t *testing.T) {
	t.Parallel()

	t.Run("CSS content", func(t *testing.T) {
		t.Parallel()
		customCSS := "body { font-family: monospace; }"

		service, err := New(WithStyle(customCSS))
		if err != nil {
			t.Fatalf("New(WithStyle) error = %v", err)
		}
		defer service.Close()

		// CSS content is detected by presence of '{' and stored in resolvedStyle
		if service.cfg.resolvedStyle != customCSS {
			t.Errorf("cfg.resolvedStyle = %q, want %q", service.cfg.resolvedStyle, customCSS)
		}
	})

	t.Run("style name", func(t *testing.T) {
		t.Parallel()
		service, err := New(WithStyle("technical"))
		if err != nil {
			t.Fatalf("New(WithStyle) error = %v", err)
		}
		defer service.Close()

		// Should have loaded the technical style from embedded assets
		if service.cfg.resolvedStyle == "" {
			t.Error("cfg.resolvedStyle is empty, expected technical.css content")
		}
		// Verify it contains something from technical.css (system-ui is distinctive)
		if !strings.Contains(service.cfg.resolvedStyle, "system-ui") {
			t.Error("cfg.resolvedStyle doesn't contain expected 'system-ui' from technical.css")
		}
	})

	t.Run("file path", func(t *testing.T) {
		t.Parallel()
		// Create a temp CSS file
		tmpDir := t.TempDir()
		cssPath := filepath.Join(tmpDir, "custom.css")
		cssContent := "h1 { color: red; }"
		if err := os.WriteFile(cssPath, []byte(cssContent), 0644); err != nil {
			t.Fatalf("WriteFile error = %v", err)
		}

		service, err := New(WithStyle(cssPath))
		if err != nil {
			t.Fatalf("New(WithStyle) error = %v", err)
		}
		defer service.Close()

		if service.cfg.resolvedStyle != cssContent {
			t.Errorf("cfg.resolvedStyle = %q, want %q", service.cfg.resolvedStyle, cssContent)
		}
	})

	t.Run("unknown style name", func(t *testing.T) {
		t.Parallel()
		_, err := New(WithStyle("nonexistent"))
		if err == nil {
			t.Error("expected error for unknown style name, got nil")
		}
	})

	t.Run("missing file", func(t *testing.T) {
		t.Parallel()
		_, err := New(WithStyle("./nonexistent.css"))
		if err == nil {
			t.Error("expected error for missing file, got nil")
		}
	})

	t.Run("empty string", func(t *testing.T) {
		t.Parallel()
		service, err := New(WithStyle(""))
		if err != nil {
			t.Fatalf("New(WithStyle) error = %v", err)
		}
		defer service.Close()

		// Empty string should leave resolvedStyle empty
		if service.cfg.resolvedStyle != "" {
			t.Errorf("cfg.resolvedStyle = %q, want empty", service.cfg.resolvedStyle)
		}
	})

	t.Run("CSS injected into HTML", func(t *testing.T) {
		t.Parallel()
		customCSS := "body { background-color: #ff0000; }"

		service := &Service{
			cfg:               converterConfig{resolvedStyle: customCSS},
			preprocessor:      &mockPreprocessor{},
			htmlConverter:     &mockHTMLConverter{output: "<html><body>test</body></html>"},
			cssInjector:       &pipeline.CSSInjection{},
			coverInjector:     &mockCoverInjector{},
			tocInjector:       &mockTOCInjector{},
			signatureInjector: &mockSignatureInjector{},
			pdfConverter:      &mockPDFConverter{},
		}

		result, err := service.Convert(context.Background(), Input{
			Markdown: "# Test",
			HTMLOnly: true,
		})
		if err != nil {
			t.Fatalf("Convert error = %v", err)
		}

		// Verify CSS is injected into the HTML output
		html := string(result.HTML)
		if !strings.Contains(html, "background-color: #ff0000") {
			t.Errorf("HTML does not contain injected CSS.\nHTML: %s", html)
		}
	})

	t.Run("Input.CSS overrides service style", func(t *testing.T) {
		t.Parallel()
		serviceCSS := "body { color: blue; }"
		inputCSS := "body { color: red; }"

		service := &Service{
			cfg:               converterConfig{resolvedStyle: serviceCSS},
			preprocessor:      &mockPreprocessor{},
			htmlConverter:     &mockHTMLConverter{output: "<html><body>test</body></html>"},
			cssInjector:       &pipeline.CSSInjection{},
			coverInjector:     &mockCoverInjector{},
			tocInjector:       &mockTOCInjector{},
			signatureInjector: &mockSignatureInjector{},
			pdfConverter:      &mockPDFConverter{},
		}

		result, err := service.Convert(context.Background(), Input{
			Markdown: "# Test",
			CSS:      inputCSS,
			HTMLOnly: true,
		})
		if err != nil {
			t.Fatalf("Convert error = %v", err)
		}

		// Both CSS should be present, with Input.CSS appearing after service CSS
		html := string(result.HTML)
		if !strings.Contains(html, "color: blue") {
			t.Error("HTML does not contain service CSS")
		}
		if !strings.Contains(html, "color: red") {
			t.Error("HTML does not contain Input.CSS")
		}
		// Input.CSS should come after service CSS (so it overrides in cascade)
		blueIdx := strings.Index(html, "color: blue")
		redIdx := strings.Index(html, "color: red")
		if blueIdx > redIdx {
			t.Error("Input.CSS should appear after service CSS for proper cascade override")
		}
	})
}

// ---------------------------------------------------------------------------
// TestConvert_SourceDir - Relative Path Rewriting
// ---------------------------------------------------------------------------

func TestConvert_SourceDir_RewritesRelativePaths(t *testing.T) {
	t.Parallel()

	// Mock HTML converter that produces HTML with relative image
	mockHTML := &mockHTMLConverter{
		output: `<html><body><img src="./images/logo.png"></body></html>`,
	}

	service := &Service{
		preprocessor:      &mockPreprocessor{},
		htmlConverter:     mockHTML,
		cssInjector:       &mockCSSInjector{},
		coverInjector:     &mockCoverInjector{},
		tocInjector:       &mockTOCInjector{},
		signatureInjector: &mockSignatureInjector{},
		pdfConverter:      &mockPDFConverter{},
	}

	result, err := service.Convert(context.Background(), Input{
		Markdown:  "# Test\n![logo](./images/logo.png)",
		SourceDir: "/docs",
		HTMLOnly:  true,
	})
	if err != nil {
		t.Fatalf("Convert() error = %v", err)
	}

	html := string(result.HTML)

	// Relative path should be rewritten to absolute file:// URL
	if !strings.Contains(html, "file://") {
		t.Errorf("Expected relative path to be rewritten to file://, got: %s", html)
	}

	// Original relative path should NOT be present
	if strings.Contains(html, `src="./images/logo.png"`) {
		t.Error("Original relative path should be rewritten")
	}
}

func TestConvert_SourceDir_EmptySourceDirNoRewrite(t *testing.T) {
	t.Parallel()

	mockHTML := &mockHTMLConverter{
		output: `<html><body><img src="./images/logo.png"></body></html>`,
	}

	service := &Service{
		preprocessor:      &mockPreprocessor{},
		htmlConverter:     mockHTML,
		cssInjector:       &mockCSSInjector{},
		coverInjector:     &mockCoverInjector{},
		tocInjector:       &mockTOCInjector{},
		signatureInjector: &mockSignatureInjector{},
		pdfConverter:      &mockPDFConverter{},
	}

	result, err := service.Convert(context.Background(), Input{
		Markdown:  "# Test",
		SourceDir: "", // Empty - no rewriting
		HTMLOnly:  true,
	})
	if err != nil {
		t.Fatalf("Convert() error = %v", err)
	}

	html := string(result.HTML)

	// Relative path should remain unchanged when SourceDir is empty
	if !strings.Contains(html, `src="./images/logo.png"`) {
		t.Errorf("Relative path should remain unchanged when SourceDir is empty, got: %s", html)
	}
}

func TestConvert_SourceDir_AbsolutePathsUnchanged(t *testing.T) {
	t.Parallel()

	mockHTML := &mockHTMLConverter{
		output: `<html><body><img src="https://example.com/logo.png"></body></html>`,
	}

	service := &Service{
		preprocessor:      &mockPreprocessor{},
		htmlConverter:     mockHTML,
		cssInjector:       &mockCSSInjector{},
		coverInjector:     &mockCoverInjector{},
		tocInjector:       &mockTOCInjector{},
		signatureInjector: &mockSignatureInjector{},
		pdfConverter:      &mockPDFConverter{},
	}

	result, err := service.Convert(context.Background(), Input{
		Markdown:  "# Test",
		SourceDir: "/docs",
		HTMLOnly:  true,
	})
	if err != nil {
		t.Fatalf("Convert() error = %v", err)
	}

	html := string(result.HTML)

	// HTTPS URLs should remain unchanged
	if !strings.Contains(html, `src="https://example.com/logo.png"`) {
		t.Errorf("HTTPS URL should remain unchanged, got: %s", html)
	}
}

func TestConvert_SourceDir_MultipleImages(t *testing.T) {
	t.Parallel()

	mockHTML := &mockHTMLConverter{
		output: `<html><body><img src="./a.png"><img src="./b.png"><img src="https://x.com/c.png"></body></html>`,
	}

	service := &Service{
		preprocessor:      &mockPreprocessor{},
		htmlConverter:     mockHTML,
		cssInjector:       &mockCSSInjector{},
		coverInjector:     &mockCoverInjector{},
		tocInjector:       &mockTOCInjector{},
		signatureInjector: &mockSignatureInjector{},
		pdfConverter:      &mockPDFConverter{},
	}

	result, err := service.Convert(context.Background(), Input{
		Markdown:  "# Test",
		SourceDir: "/docs",
		HTMLOnly:  true,
	})
	if err != nil {
		t.Fatalf("Convert() error = %v", err)
	}

	html := string(result.HTML)

	// Both relative paths should be rewritten
	if strings.Contains(html, `src="./a.png"`) {
		t.Error("./a.png should be rewritten")
	}
	if strings.Contains(html, `src="./b.png"`) {
		t.Error("./b.png should be rewritten")
	}

	// HTTPS URL should remain unchanged
	if !strings.Contains(html, `src="https://x.com/c.png"`) {
		t.Error("HTTPS URL should remain unchanged")
	}

	// Should have two file:// URLs
	if strings.Count(html, "file://") != 2 {
		t.Errorf("Expected 2 file:// URLs, got %d in: %s", strings.Count(html, "file://"), html)
	}
}

// ---------------------------------------------------------------------------
// TestConvert_FrontmatterStripped - YAML Frontmatter Removal
// ---------------------------------------------------------------------------

func TestConvert_FrontmatterStripped(t *testing.T) {
	t.Parallel()

	service, err := New()
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer service.Close()

	markdown := `---
title: Test Document
author: John Doe
date: 2024-01-15
tags: [test, example]
---

# Introduction

This document has YAML frontmatter that should not appear in the output.

## Content

The frontmatter above contains metadata.`

	result, err := service.Convert(context.Background(), Input{
		Markdown: markdown,
		HTMLOnly: true, // Skip PDF for faster test
	})
	if err != nil {
		t.Fatalf("Convert() error = %v", err)
	}

	html := string(result.HTML)

	// Frontmatter metadata should NOT appear in HTML
	if strings.Contains(html, "title: Test Document") {
		t.Error("HTML should not contain frontmatter key 'title: Test Document'")
	}
	if strings.Contains(html, "author: John Doe") {
		t.Error("HTML should not contain frontmatter key 'author: John Doe'")
	}
	if strings.Contains(html, "date: 2024-01-15") {
		t.Error("HTML should not contain frontmatter key 'date: 2024-01-15'")
	}
	if strings.Contains(html, "tags: [test, example]") {
		t.Error("HTML should not contain frontmatter key 'tags: [test, example]'")
	}

	// Frontmatter delimiters should NOT appear
	if strings.Contains(html, "---") {
		t.Error("HTML should not contain frontmatter delimiters '---'")
	}

	// Content should be present
	if !strings.Contains(html, "<h1") {
		t.Error("HTML should contain <h1> heading")
	}
	if !strings.Contains(html, "Introduction") {
		t.Error("HTML should contain 'Introduction' heading text")
	}
	if !strings.Contains(html, "This document has YAML frontmatter") {
		t.Error("HTML should contain paragraph content")
	}
	if !strings.Contains(html, "Content") {
		t.Error("HTML should contain 'Content' heading text")
	}
}

// ---------------------------------------------------------------------------
// TestConvert_MalformedFrontmatterPreserved - Malformed Frontmatter Safety
// ---------------------------------------------------------------------------

func TestConvert_MalformedFrontmatterPreserved(t *testing.T) {
	t.Parallel()

	service, err := New()
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer service.Close()

	tests := []struct {
		name     string
		markdown string
		wantText string // Text that SHOULD appear (malformed frontmatter preserved)
	}{
		{
			name: "missing closing delimiter",
			markdown: `---
title: Test
# Content`,
			wantText: "title: Test",
		},
		{
			name: "missing opening delimiter",
			markdown: `title: Test
---
# Content`,
			wantText: "title: Test",
		},
		{
			name: "single delimiter only becomes horizontal rule",
			markdown: `---
# Content`,
			wantText: "<hr", // Single --- becomes <hr /> (horizontal rule in markdown)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result, err := service.Convert(context.Background(), Input{
				Markdown: tt.markdown,
				HTMLOnly: true,
			})
			if err != nil {
				t.Fatalf("Convert() error = %v", err)
			}

			html := string(result.HTML)

			// Malformed frontmatter should appear in HTML (preserved as-is)
			if !strings.Contains(html, tt.wantText) {
				t.Errorf("HTML should contain malformed frontmatter text %q (preserved for safety)", tt.wantText)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestConvert_FrontmatterWithCodeBlocks - Code Blocks Not Stripped
// ---------------------------------------------------------------------------

func TestConvert_FrontmatterWithCodeBlocks(t *testing.T) {
	t.Parallel()

	service, err := New()
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer service.Close()

	markdown := "```yaml\n---\ntitle: Config\n---\n```\n\n# Content"

	result, err := service.Convert(context.Background(), Input{
		Markdown: markdown,
		HTMLOnly: true,
	})
	if err != nil {
		t.Fatalf("Convert() error = %v", err)
	}

	html := string(result.HTML)

	// Code block content should be preserved (not stripped as frontmatter)
	// Syntax highlighting wraps text in spans, so search for fragments
	if !strings.Contains(html, "title") || !strings.Contains(html, "Config") {
		t.Error("HTML should contain code block with 'title' and 'Config'")
	}
	if !strings.Contains(html, "<code") || !strings.Contains(html, "</code>") {
		t.Error("HTML should contain code block tags")
	}
}
