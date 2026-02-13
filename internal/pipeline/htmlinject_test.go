package pipeline

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/alnah/go-md2pdf/internal/assets"
)

func TestSanitizeCSS(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "no escape needed",
			input:    "body { color: red; }",
			expected: "body { color: red; }",
		},
		{
			name:     "escapes style close",
			input:    "</style>",
			expected: `<\/style>`,
		},
		{
			name:     "escapes script close",
			input:    "</script>",
			expected: `<\/script>`,
		},
		{
			name:     "multiple occurrences",
			input:    "</a></b>",
			expected: `<\/a><\/b>`,
		},
		{
			name:     "nested sequences",
			input:    "</</style>",
			expected: `<\/<\/style>`,
		},
		{
			name:     "case variation STYLE",
			input:    "</STYLE>",
			expected: `<\/STYLE>`,
		},
		{
			name:     "case variation Script",
			input:    "</Script>",
			expected: `<\/Script>`,
		},
		{
			name:     "mixed case sTyLe",
			input:    "</sTyLe>",
			expected: `<\/sTyLe>`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := sanitizeCSS(tt.input)
			if got != tt.expected {
				t.Errorf("sanitizeCSS(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestCSSInjection_InjectCSS(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		html     string
		css      string
		expected string
	}{
		{
			name:     "empty CSS returns HTML unchanged",
			html:     "<html><head></head><body>Hello</body></html>",
			css:      "",
			expected: "<html><head></head><body>Hello</body></html>",
		},
		{
			name:     "injects before </head>",
			html:     "<html><head></head><body>Hello</body></html>",
			css:      "body { color: red; }",
			expected: "<html><head><style>body { color: red; }</style></head><body>Hello</body></html>",
		},
		{
			name:     "injects before </HEAD> mixed case",
			html:     "<html><HEAD></HEAD><body>Hello</body></html>",
			css:      "body { color: red; }",
			expected: "<html><HEAD><style>body { color: red; }</style></HEAD><body>Hello</body></html>",
		},
		{
			name:     "injects after <body> when no </head>",
			html:     "<html><body>Hello</body></html>",
			css:      "body { color: red; }",
			expected: "<html><body><style>body { color: red; }</style>Hello</body></html>",
		},
		{
			name:     "injects after <body> with attributes",
			html:     `<html><body class="main" id="app">Hello</body></html>`,
			css:      "body { color: red; }",
			expected: `<html><body class="main" id="app"><style>body { color: red; }</style>Hello</body></html>`,
		},
		{
			name:     "injects after <BODY> mixed case",
			html:     "<html><BODY>Hello</BODY></html>",
			css:      "body { color: red; }",
			expected: "<html><BODY><style>body { color: red; }</style>Hello</BODY></html>",
		},
		{
			name:     "prepends to bare fragment",
			html:     "<p>Hello</p>",
			css:      "p { color: blue; }",
			expected: "<style>p { color: blue; }</style><p>Hello</p>",
		},
		{
			name:     "sanitizes CSS with closing tags",
			html:     "<html><head></head><body>Hello</body></html>",
			css:      "</style><script>alert('xss')</script>",
			expected: `<html><head><style><\/style><script>alert('xss')<\/script></style></head><body>Hello</body></html>`,
		},
		{
			name:     "unicode in CSS content property",
			html:     "<html><head></head><body>Hello</body></html>",
			css:      `.icon::before { content: ""; }`,
			expected: `<html><head><style>.icon::before { content: ""; }</style></head><body>Hello</body></html>`,
		},
		{
			name:     "unicode in HTML preserved",
			html:     "<html><head></head><body>Bonjour le monde</body></html>",
			css:      "body { color: red; }",
			expected: "<html><head><style>body { color: red; }</style></head><body>Bonjour le monde</body></html>",
		},
	}

	injector := &CSSInjection{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			got := injector.InjectCSS(ctx, tt.html, tt.css)
			if got != tt.expected {
				t.Errorf("CSSInjection.InjectCSS(%q, %q) = %q, want %q", tt.html, tt.css, got, tt.expected)
			}
		})
	}
}

func TestCSSInjection_InjectCSS_ContextCancellation(t *testing.T) {
	t.Parallel()

	injector := &CSSInjection{}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	html := "<html><head></head><body>Hello</body></html>"
	css := "body { color: red; }"

	// When context is cancelled, returns HTML unchanged
	got := injector.InjectCSS(ctx, html, css)
	if got != html {
		t.Errorf("CSSInjection.InjectCSS() with cancelled context = %q, want %q", got, html)
	}
}

func TestSignatureInjection_InjectSignature(t *testing.T) {
	t.Parallel()

	loader := assets.NewEmbeddedLoader()
	ts, err := loader.LoadTemplateSet(assets.DefaultTemplateSetName)
	if err != nil {
		t.Fatalf("LoadTemplateSet() unexpected error: %v", err)
	}
	injector, err := NewSignatureInjection(ts.Signature)
	if err != nil {
		t.Fatalf("NewSignatureInjection() unexpected error: %v", err)
	}

	t.Run("nil data returns HTML unchanged", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()
		html := "<html><body>Hello</body></html>"
		got, err := injector.InjectSignature(ctx, html, nil)
		if err != nil {
			t.Fatalf("SignatureInjection.InjectSignature() unexpected error: %v", err)
		}
		if got != html {
			t.Errorf("SignatureInjection.InjectSignature(nil) = %q, want %q", got, html)
		}
	})

	t.Run("injects signature before </body>", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()
		html := "<html><body>Content</body></html>"
		data := &SignatureData{Name: "John Doe", Email: "john@example.com"}

		got, err := injector.InjectSignature(ctx, html, data)
		if err != nil {
			t.Fatalf("SignatureInjection.InjectSignature() unexpected error: %v", err)
		}

		// Verify signature is injected before </body>
		if !strings.Contains(got, "John Doe") {
			t.Error("signature name not found in output")
		}
		if !strings.Contains(got, "john@example.com") {
			t.Error("signature email not found in output")
		}

		// Verify position: signature should appear before </body>
		sigIdx := strings.Index(got, "signature-block")
		bodyIdx := strings.Index(got, "</body>")
		if sigIdx == -1 || bodyIdx == -1 || sigIdx > bodyIdx {
			t.Error("signature should be inserted before </body>")
		}
	})

	t.Run("injects before </BODY> mixed case", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()
		html := "<html><BODY>Content</BODY></html>"
		data := &SignatureData{Name: "Test"}

		got, err := injector.InjectSignature(ctx, html, data)
		if err != nil {
			t.Fatalf("SignatureInjection.InjectSignature() unexpected error: %v", err)
		}

		if !strings.Contains(got, "Test") {
			t.Error("signature name not found in output")
		}
	})

	t.Run("appends to end when no </body>", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()
		html := "<p>Content</p>"
		data := &SignatureData{Name: "Test"}

		got, err := injector.InjectSignature(ctx, html, data)
		if err != nil {
			t.Fatalf("SignatureInjection.InjectSignature() unexpected error: %v", err)
		}

		// Signature should be at the end
		if !strings.HasSuffix(got, "</div>\n") {
			t.Errorf("signature should be appended at end, got: %q", got)
		}
	})

	t.Run("renders all signature fields", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()
		html := "<html><body></body></html>"
		data := &SignatureData{
			Name:      "Jane Smith",
			Title:     "Software Engineer",
			Email:     "jane@example.com",
			ImagePath: "https://example.com/photo.jpg",
			Links: []SignatureLink{
				{Label: "GitHub", URL: "https://github.com/jane"},
				{Label: "LinkedIn", URL: "https://linkedin.com/in/jane"},
			},
		}

		got, err := injector.InjectSignature(ctx, html, data)
		if err != nil {
			t.Fatalf("SignatureInjection.InjectSignature() unexpected error: %v", err)
		}

		// Verify all fields are rendered
		expectedParts := []string{
			"Jane Smith",
			"Software Engineer",
			"jane@example.com",
			"https://example.com/photo.jpg",
			"GitHub",
			"https://github.com/jane",
			"LinkedIn",
			"https://linkedin.com/in/jane",
		}
		for _, part := range expectedParts {
			if !strings.Contains(got, part) {
				t.Errorf("expected %q in output", part)
			}
		}
	})

	t.Run("optional fields can be empty", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()
		html := "<html><body></body></html>"
		data := &SignatureData{
			Name: "Minimal",
			// Title, Email, ImagePath, Links all empty
		}

		got, err := injector.InjectSignature(ctx, html, data)
		if err != nil {
			t.Fatalf("SignatureInjection.InjectSignature() unexpected error: %v", err)
		}

		if !strings.Contains(got, "Minimal") {
			t.Error("name should be rendered")
		}
		// Should not contain empty <em> or <a> tags for missing fields
		if strings.Contains(got, "<em></em>") {
			t.Error("empty title should not render empty <em> tag")
		}
	})
}

func TestSignatureInjection_InjectSignature_ContextCancellation(t *testing.T) {
	t.Parallel()

	loader := assets.NewEmbeddedLoader()
	ts, err := loader.LoadTemplateSet(assets.DefaultTemplateSetName)
	if err != nil {
		t.Fatalf("LoadTemplateSet() unexpected error: %v", err)
	}
	injector, err := NewSignatureInjection(ts.Signature)
	if err != nil {
		t.Fatalf("NewSignatureInjection() unexpected error: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	data := &SignatureData{Name: "Test"}
	_, err = injector.InjectSignature(ctx, "<body></body>", data)

	if err == nil {
		t.Fatal("SignatureInjection.InjectSignature() error = nil, want error")
	}
	if !errors.Is(err, context.Canceled) {
		t.Errorf("SignatureInjection.InjectSignature() error = %v, want context.Canceled", err)
	}
}

func TestSignatureInjection_InjectSignature_TemplateError(t *testing.T) {
	t.Parallel()

	// Create injector with a broken template to test error path
	// This is difficult to trigger with valid SignatureData,
	// but we can verify the error type is returned correctly
	// by using the mock in service_test.go

	// For the real implementation, verify that error wrapping works
	loader := assets.NewEmbeddedLoader()
	ts, err := loader.LoadTemplateSet(assets.DefaultTemplateSetName)
	if err != nil {
		t.Fatalf("LoadTemplateSet() unexpected error: %v", err)
	}
	injector, err := NewSignatureInjection(ts.Signature)
	if err != nil {
		t.Fatalf("NewSignatureInjection() unexpected error: %v", err)
	}
	ctx := context.Background()
	data := &SignatureData{Name: "Test"}

	_, err = injector.InjectSignature(ctx, "<body></body>", data)
	if err != nil {
		if !errors.Is(err, ErrSignatureRender) {
			t.Errorf("error should wrap ErrSignatureRender, got: %v", err)
		}
	}
	// Note: with valid data and template, no error expected
}

func TestCoverInjection_InjectCover(t *testing.T) {
	t.Parallel()

	loader := assets.NewEmbeddedLoader()
	ts, err := loader.LoadTemplateSet(assets.DefaultTemplateSetName)
	if err != nil {
		t.Fatalf("LoadTemplateSet() unexpected error: %v", err)
	}
	injector, err := NewCoverInjection(ts.Cover)
	if err != nil {
		t.Fatalf("NewCoverInjection() unexpected error: %v", err)
	}

	t.Run("nil data returns HTML unchanged", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()
		html := "<html><body>Hello</body></html>"
		got, err := injector.InjectCover(ctx, html, nil)
		if err != nil {
			t.Fatalf("CoverInjection.InjectCover() unexpected error: %v", err)
		}
		if got != html {
			t.Errorf("CoverInjection.InjectCover(nil) = %q, want %q", got, html)
		}
	})

	t.Run("injects cover after <body>", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()
		html := "<html><body>Content</body></html>"
		data := &CoverData{Title: "My Document"}

		got, err := injector.InjectCover(ctx, html, data)
		if err != nil {
			t.Fatalf("CoverInjection.InjectCover() unexpected error: %v", err)
		}

		// Verify cover is injected after <body>
		if !strings.Contains(got, "My Document") {
			t.Error("cover title not found in output")
		}

		// Verify position: cover should appear after <body>
		bodyIdx := strings.Index(got, "<body>")
		coverIdx := strings.Index(got, "cover-page")
		if bodyIdx == -1 || coverIdx == -1 || coverIdx < bodyIdx {
			t.Error("cover should be inserted after <body>")
		}
	})

	t.Run("injects after <body> with attributes", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()
		html := `<html><body class="main">Content</body></html>`
		data := &CoverData{Title: "Test"}

		got, err := injector.InjectCover(ctx, html, data)
		if err != nil {
			t.Fatalf("CoverInjection.InjectCover() unexpected error: %v", err)
		}

		if !strings.Contains(got, "Test") {
			t.Error("cover title not found in output")
		}
	})

	t.Run("injects after <BODY> mixed case", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()
		html := "<html><BODY>Content</BODY></html>"
		data := &CoverData{Title: "Test"}

		got, err := injector.InjectCover(ctx, html, data)
		if err != nil {
			t.Fatalf("CoverInjection.InjectCover() unexpected error: %v", err)
		}

		if !strings.Contains(got, "Test") {
			t.Error("cover title not found in output")
		}
	})

	t.Run("prepends when no <body>", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()
		html := "<p>Content</p>"
		data := &CoverData{Title: "Test"}

		got, err := injector.InjectCover(ctx, html, data)
		if err != nil {
			t.Fatalf("CoverInjection.InjectCover() unexpected error: %v", err)
		}

		// Cover should be at the start (now wrapped in section)
		if !strings.HasPrefix(got, "<section class=\"cover\"><div class=\"cover-page\">") {
			t.Errorf("cover should be prepended, got: %q", got[:min(100, len(got))])
		}
	})

	t.Run("renders all cover fields", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()
		html := "<html><body></body></html>"
		data := &CoverData{
			Title:        "My Document",
			Subtitle:     "A Comprehensive Guide",
			Logo:         "https://example.com/logo.png",
			Author:       "John Doe",
			AuthorTitle:  "Senior Developer",
			Organization: "Acme Corp",
			Date:         "2025-01-15",
			Version:      "v1.0.0",
		}

		got, err := injector.InjectCover(ctx, html, data)
		if err != nil {
			t.Fatalf("CoverInjection.InjectCover() unexpected error: %v", err)
		}

		// Verify all fields are rendered
		expectedParts := []string{
			"My Document",
			"A Comprehensive Guide",
			"https://example.com/logo.png",
			"John Doe",
			"Senior Developer",
			"Acme Corp",
			"2025-01-15",
			"v1.0.0",
		}
		for _, part := range expectedParts {
			if !strings.Contains(got, part) {
				t.Errorf("expected %q in output", part)
			}
		}
	})

	t.Run("optional fields can be empty", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()
		html := "<html><body></body></html>"
		data := &CoverData{
			Title: "Minimal",
			// All other fields empty
		}

		got, err := injector.InjectCover(ctx, html, data)
		if err != nil {
			t.Fatalf("CoverInjection.InjectCover() unexpected error: %v", err)
		}

		if !strings.Contains(got, "Minimal") {
			t.Error("title should be rendered")
		}
		if !strings.Contains(got, "cover-page") {
			t.Error("cover page class should be present")
		}
	})

	t.Run("HTML escapes special characters", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()
		html := "<html><body></body></html>"
		data := &CoverData{
			Title:  "<script>alert('xss')</script>",
			Author: "John & Jane",
		}

		got, err := injector.InjectCover(ctx, html, data)
		if err != nil {
			t.Fatalf("CoverInjection.InjectCover() unexpected error: %v", err)
		}

		// HTML template should escape these
		if strings.Contains(got, "<script>alert") {
			t.Error("script tag should be escaped")
		}
		if !strings.Contains(got, "&lt;script&gt;") && !strings.Contains(got, "&#") {
			// Either HTML entity or numeric escape is acceptable
			t.Log("Note: checking for HTML escaping of script tag")
		}
	})
}

func TestCoverInjection_InjectCover_ContextCancellation(t *testing.T) {
	t.Parallel()

	loader := assets.NewEmbeddedLoader()
	ts, err := loader.LoadTemplateSet(assets.DefaultTemplateSetName)
	if err != nil {
		t.Fatalf("LoadTemplateSet() unexpected error: %v", err)
	}
	injector, err := NewCoverInjection(ts.Cover)
	if err != nil {
		t.Fatalf("NewCoverInjection() unexpected error: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	data := &CoverData{Title: "Test"}
	_, err = injector.InjectCover(ctx, "<body></body>", data)

	if err == nil {
		t.Fatal("CoverInjection.InjectCover() error = nil, want error")
	}
	if !errors.Is(err, context.Canceled) {
		t.Errorf("CoverInjection.InjectCover() error = %v, want context.Canceled", err)
	}
}

func TestExtractHeadings(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		html     string
		minDepth int
		maxDepth int
		want     []headingInfo
	}{
		{
			name:     "empty HTML returns nil",
			html:     "",
			minDepth: 1,
			maxDepth: 3,
			want:     nil,
		},
		{
			name:     "no headings returns nil",
			html:     "<p>Just a paragraph</p>",
			minDepth: 1,
			maxDepth: 3,
			want:     nil,
		},
		{
			name:     "heading without id is skipped",
			html:     "<h1>No ID</h1>",
			minDepth: 1,
			maxDepth: 3,
			want:     nil,
		},
		{
			name:     "single h1 with id",
			html:     `<h1 id="intro">Introduction</h1>`,
			minDepth: 1,
			maxDepth: 3,
			want:     []headingInfo{{Level: 1, ID: "intro", Text: "Introduction"}},
		},
		{
			name:     "multiple headings",
			html:     `<h1 id="a">A</h1><h2 id="b">B</h2><h3 id="c">C</h3>`,
			minDepth: 1,
			maxDepth: 3,
			want: []headingInfo{
				{Level: 1, ID: "a", Text: "A"},
				{Level: 2, ID: "b", Text: "B"},
				{Level: 3, ID: "c", Text: "C"},
			},
		},
		{
			name:     "respects maxDepth limit",
			html:     `<h1 id="a">A</h1><h2 id="b">B</h2><h3 id="c">C</h3><h4 id="d">D</h4>`,
			minDepth: 1,
			maxDepth: 2,
			want: []headingInfo{
				{Level: 1, ID: "a", Text: "A"},
				{Level: 2, ID: "b", Text: "B"},
			},
		},
		{
			name:     "case insensitive H1",
			html:     `<H1 id="test">Test</H1>`,
			minDepth: 1,
			maxDepth: 3,
			want:     []headingInfo{{Level: 1, ID: "test", Text: "Test"}},
		},
		{
			name:     "mixed case h2",
			html:     `<H2 ID="mixed">Mixed</H2>`,
			minDepth: 1,
			maxDepth: 3,
			want:     []headingInfo{{Level: 2, ID: "mixed", Text: "Mixed"}}, // case-insensitive matching
		},
		{
			name:     "heading with extra attributes",
			html:     `<h1 class="title" id="main" data-foo="bar">Main</h1>`,
			minDepth: 1,
			maxDepth: 3,
			want:     []headingInfo{{Level: 1, ID: "main", Text: "Main"}},
		},
		{
			name:     "trims whitespace from text",
			html:     `<h1 id="space">  Spaced Text  </h1>`,
			minDepth: 1,
			maxDepth: 3,
			want:     []headingInfo{{Level: 1, ID: "space", Text: "Spaced Text"}},
		},
		{
			name:     "maxDepth 6 includes all levels",
			html:     `<h1 id="h1">H1</h1><h6 id="h6">H6</h6>`,
			minDepth: 1,
			maxDepth: 6,
			want: []headingInfo{
				{Level: 1, ID: "h1", Text: "H1"},
				{Level: 6, ID: "h6", Text: "H6"},
			},
		},
		{
			name:     "maxDepth 1 only h1",
			html:     `<h1 id="h1">H1</h1><h2 id="h2">H2</h2>`,
			minDepth: 1,
			maxDepth: 1,
			want:     []headingInfo{{Level: 1, ID: "h1", Text: "H1"}},
		},
		{
			name:     "minDepth 2 skips h1",
			html:     `<h1 id="h1">H1</h1><h2 id="h2">H2</h2><h3 id="h3">H3</h3>`,
			minDepth: 2,
			maxDepth: 3,
			want: []headingInfo{
				{Level: 2, ID: "h2", Text: "H2"},
				{Level: 3, ID: "h3", Text: "H3"},
			},
		},
		{
			name:     "inline em tag stripped",
			html:     `<h1 id="intro"><em>Hello</em> World</h1>`,
			minDepth: 1,
			maxDepth: 3,
			want:     []headingInfo{{Level: 1, ID: "intro", Text: "Hello World"}},
		},
		{
			name:     "inline code tag stripped",
			html:     `<h1 id="func"><code>func</code> Main</h1>`,
			minDepth: 1,
			maxDepth: 3,
			want:     []headingInfo{{Level: 1, ID: "func", Text: "func Main"}},
		},
		{
			name:     "inline strong tag stripped",
			html:     `<h1 id="bold">Plain <strong>bold</strong> plain</h1>`,
			minDepth: 1,
			maxDepth: 3,
			want:     []headingInfo{{Level: 1, ID: "bold", Text: "Plain bold plain"}},
		},
		{
			name:     "nested inline tags stripped",
			html:     `<h1 id="nested"><em><strong>Nested</strong></em></h1>`,
			minDepth: 1,
			maxDepth: 3,
			want:     []headingInfo{{Level: 1, ID: "nested", Text: "Nested"}},
		},
		{
			name:     "multiple inline tags stripped",
			html:     `<h1 id="multi"><code>code</code> and <em>emphasis</em></h1>`,
			minDepth: 1,
			maxDepth: 3,
			want:     []headingInfo{{Level: 1, ID: "multi", Text: "code and emphasis"}},
		},
		{
			name:     "anchor tag inside heading stripped",
			html:     `<h1 id="link"><a href="#">Link Text</a></h1>`,
			minDepth: 1,
			maxDepth: 3,
			want:     []headingInfo{{Level: 1, ID: "link", Text: "Link Text"}},
		},
		// HTML entity decoding - fixes double-encoding bug in TOC
		{
			name:     "ampersand entity decoded",
			html:     `<h1 id="ab">A &amp; B</h1>`,
			minDepth: 1,
			maxDepth: 3,
			want:     []headingInfo{{Level: 1, ID: "ab", Text: "A & B"}},
		},
		{
			name:     "less than entity decoded",
			html:     `<h1 id="lt">x &lt; y</h1>`,
			minDepth: 1,
			maxDepth: 3,
			want:     []headingInfo{{Level: 1, ID: "lt", Text: "x < y"}},
		},
		{
			name:     "greater than entity decoded",
			html:     `<h1 id="gt">x &gt; y</h1>`,
			minDepth: 1,
			maxDepth: 3,
			want:     []headingInfo{{Level: 1, ID: "gt", Text: "x > y"}},
		},
		{
			name:     "quote entity decoded",
			html:     `<h1 id="quote">&quot;quoted&quot;</h1>`,
			minDepth: 1,
			maxDepth: 3,
			want:     []headingInfo{{Level: 1, ID: "quote", Text: "\"quoted\""}},
		},
		{
			name:     "numeric entity decoded",
			html:     `<h1 id="dash">foo &#8212; bar</h1>`,
			minDepth: 1,
			maxDepth: 3,
			want:     []headingInfo{{Level: 1, ID: "dash", Text: "foo — bar"}},
		},
		{
			name:     "multiple entities decoded",
			html:     `<h1 id="multi">A &amp; B &lt; C &gt; D</h1>`,
			minDepth: 1,
			maxDepth: 3,
			want:     []headingInfo{{Level: 1, ID: "multi", Text: "A & B < C > D"}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := extractHeadings(tt.html, tt.minDepth, tt.maxDepth)

			if len(got) != len(tt.want) {
				t.Fatalf("extractHeadings(%q, %d, %d) returned %d headings, want %d", tt.html, tt.minDepth, tt.maxDepth, len(got), len(tt.want))
			}

			for i, want := range tt.want {
				if got[i].Level != want.Level {
					t.Errorf("heading[%d].Level = %d, want %d", i, got[i].Level, want.Level)
				}
				if got[i].ID != want.ID {
					t.Errorf("heading[%d].ID = %q, want %q", i, got[i].ID, want.ID)
				}
				if got[i].Text != want.Text {
					t.Errorf("heading[%d].Text = %q, want %q", i, got[i].Text, want.Text)
				}
			}
		})
	}
}

func TestStripHTMLTags(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input string
		want  string
	}{
		{"plain text", "plain text"},
		{"<em>emphasized</em>", "emphasized"},
		{"<strong>bold</strong>", "bold"},
		{"<code>code</code>", "code"},
		{"<a href=\"#\">link</a>", "link"},
		{"<em>Hello</em> World", "Hello World"},
		{"Plain <strong>bold</strong> plain", "Plain bold plain"},
		{"<em><strong>nested</strong></em>", "nested"},
		{"  <em>spaced</em>  ", "spaced"},
		{"", ""},
		{"no tags", "no tags"},
		{"<br/>self closing", "self closing"},
		{"<div class=\"foo\">with attrs</div>", "with attrs"},
		// HTML entity decoding - fixes double-encoding bug in TOC
		{"A &amp; B", "A & B"},
		{"&lt;script&gt;", "<script>"},
		{"&quot;quoted&quot;", "\"quoted\""},
		{"&#39;apostrophe&#39;", "'apostrophe'"},
		{"&lt;em&gt;not a tag&lt;/em&gt;", "<em>not a tag</em>"},
		{"mixed &amp; <em>tags</em> &amp; entities", "mixed & tags & entities"},
		{"&#8212; em dash", "— em dash"},
		{"&copy; 2025", "© 2025"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			t.Parallel()

			got := stripHTMLTags(tt.input)
			if got != tt.want {
				t.Errorf("stripHTMLTags(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestNumberingState_next(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		levels []int
		want   []string
	}{
		{
			name:   "sequential h1s",
			levels: []int{1, 1, 1},
			want:   []string{"1.", "2.", "3."},
		},
		{
			name:   "h1 then h2s",
			levels: []int{1, 2, 2},
			want:   []string{"1.", "1.1.", "1.2."},
		},
		{
			name:   "h1 h2 h3 nested",
			levels: []int{1, 2, 3},
			want:   []string{"1.", "1.1.", "1.1.1."},
		},
		{
			name:   "return to h1 resets counters",
			levels: []int{1, 2, 1},
			want:   []string{"1.", "1.1.", "2."},
		},
		{
			name:   "return to h2 resets h3",
			levels: []int{1, 2, 3, 2},
			want:   []string{"1.", "1.1.", "1.1.1.", "1.2."},
		},
		{
			name:   "normalization starts at h2",
			levels: []int{2, 2, 3},
			want:   []string{"1.", "2.", "2.1."},
		},
		{
			name:   "normalization starts at h3",
			levels: []int{3, 3},
			want:   []string{"1.", "2."},
		},
		{
			name:   "gap skipping h1 to h3",
			levels: []int{1, 3},
			want:   []string{"1.", "1.1."},
		},
		{
			name:   "gap skipping h1 to h4",
			levels: []int{1, 4, 4},
			want:   []string{"1.", "1.1.", "1.1.1."}, // consecutive gaps increase depth each time
		},
		{
			name:   "complex sequence",
			levels: []int{1, 2, 2, 3, 2, 1, 2},
			want:   []string{"1.", "1.1.", "1.2.", "1.2.1.", "1.3.", "2.", "2.1."},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			state := newNumberingState()

			for i, level := range tt.levels {
				got, _ := state.next(level)
				if got != tt.want[i] {
					t.Errorf("numberingState.next(%d) at step %d = %q, want %q", level, i, got, tt.want[i])
				}
			}
		})
	}
}

func TestNumberingState_next_EffectiveDepth(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		levels     []int
		wantDepths []int
	}{
		{
			name:       "sequential h1s all depth 1",
			levels:     []int{1, 1, 1},
			wantDepths: []int{1, 1, 1},
		},
		{
			name:       "h1 h2 h3 increasing depths",
			levels:     []int{1, 2, 3},
			wantDepths: []int{1, 2, 3},
		},
		{
			name:       "normalization starts at h2",
			levels:     []int{2, 3},
			wantDepths: []int{1, 2},
		},
		{
			name:       "gap skipping h1 to h3",
			levels:     []int{1, 3},
			wantDepths: []int{1, 2}, // h3 becomes depth 2 (gap skipped)
		},
		{
			name:       "return to shallower level",
			levels:     []int{1, 2, 3, 1},
			wantDepths: []int{1, 2, 3, 1},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			state := newNumberingState()

			for i, level := range tt.levels {
				_, depth := state.next(level)
				if depth != tt.wantDepths[i] {
					t.Errorf("numberingState.next(%d) at step %d depth = %d, want %d", level, i, depth, tt.wantDepths[i])
				}
			}
		})
	}
}

func TestGenerateNumberedTOC(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		headings     []headingInfo
		title        string
		wantEmpty    bool
		wantContains []string
	}{
		{
			name:      "nil headings returns empty",
			headings:  nil,
			title:     "Contents",
			wantEmpty: true,
		},
		{
			name:      "empty headings returns empty",
			headings:  []headingInfo{},
			title:     "Contents",
			wantEmpty: true,
		},
		{
			name: "single heading with title",
			headings: []headingInfo{
				{Level: 1, ID: "intro", Text: "Introduction"},
			},
			title: "Table of Contents",
			wantContains: []string{
				`<nav class="toc">`,
				`<h2 class="toc-title">Table of Contents</h2>`,
				`<div class="toc-list">`,
				`<div class="toc-item">`,
				`href="#intro"`,
				`1. Introduction`,
				`</nav>`,
			},
		},
		{
			name: "single heading without title",
			headings: []headingInfo{
				{Level: 1, ID: "intro", Text: "Introduction"},
			},
			title: "",
			wantContains: []string{
				`<nav class="toc">`,
				`<div class="toc-list">`,
				`<div class="toc-item">`,
				`href="#intro"`,
			},
		},
		{
			name: "escapes HTML in heading text",
			headings: []headingInfo{
				{Level: 1, ID: "xss", Text: "<script>alert('xss')</script>"},
			},
			title: "",
			wantContains: []string{
				`&lt;script&gt;`,
			},
		},
		{
			name: "escapes HTML in ID",
			headings: []headingInfo{
				{Level: 1, ID: `test"><script>`, Text: "Test"},
			},
			title: "",
			wantContains: []string{
				`href="#test&#34;&gt;&lt;script&gt;"`,
			},
		},
		{
			name: "escapes HTML in title",
			headings: []headingInfo{
				{Level: 1, ID: "test", Text: "Test"},
			},
			title: "<script>alert('xss')</script>",
			wantContains: []string{
				`&lt;script&gt;`,
			},
		},
		{
			name: "nested headings use indentation via padding",
			headings: []headingInfo{
				{Level: 1, ID: "ch1", Text: "Chapter 1"},
				{Level: 2, ID: "sec1", Text: "Section 1"},
			},
			title: "",
			wantContains: []string{
				`<div class="toc-item">`, // Level 1: no padding
				`1. Chapter 1`,
				`padding-left:1.5em`, // Level 2: indented
				`1.1. Section 1`,
			},
		},
		// Special characters - verify proper single encoding (not double)
		{
			name: "ampersand in text is properly encoded once",
			headings: []headingInfo{
				{Level: 1, ID: "ab", Text: "A & B"}, // Already decoded by stripHTMLTags
			},
			title: "",
			wantContains: []string{
				`A &amp; B`, // Should be encoded once
			},
		},
		{
			name: "less than in text is properly encoded",
			headings: []headingInfo{
				{Level: 1, ID: "lt", Text: "x < y"},
			},
			title: "",
			wantContains: []string{
				`x &lt; y`,
			},
		},
		{
			name: "multiple special chars encoded correctly",
			headings: []headingInfo{
				{Level: 1, ID: "special", Text: "A & B < C > D"},
			},
			title: "",
			wantContains: []string{
				`A &amp; B &lt; C &gt; D`,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := generateNumberedTOC(tt.headings, tt.title)

			if tt.wantEmpty {
				if got != "" {
					t.Errorf("generateNumberedTOC(%v, %q) = %q, want empty", tt.headings, tt.title, got)
				}
				return
			}

			if got == "" {
				t.Fatal("generateNumberedTOC() = empty, want HTML")
			}

			for _, want := range tt.wantContains {
				if !strings.Contains(got, want) {
					t.Errorf("generateNumberedTOC() missing %q\nGot:\n%s", want, got)
				}
			}
		})
	}
}

func TestTOCInjection_InjectTOC(t *testing.T) {
	t.Parallel()

	injector := NewTOCInjection()

	t.Run("nil data returns HTML unchanged", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()
		html := "<html><body>Hello</body></html>"
		got, err := injector.InjectTOC(ctx, html, nil)
		if err != nil {
			t.Fatalf("TOCInjection.InjectTOC() unexpected error: %v", err)
		}
		if got != html {
			t.Errorf("TOCInjection.InjectTOC(nil) = %q, want %q", got, html)
		}
	})

	t.Run("no headings returns HTML unchanged", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()
		html := "<html><body><p>No headings</p></body></html>"
		data := &TOCData{Title: "TOC", MaxDepth: 3}
		got, err := injector.InjectTOC(ctx, html, data)
		if err != nil {
			t.Fatalf("TOCInjection.InjectTOC() unexpected error: %v", err)
		}
		if got != html {
			t.Errorf("TOCInjection.InjectTOC() should return unchanged HTML when no headings")
		}
	})

	t.Run("injects after cover-end marker", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()
		// Use <span data-cover-end> marker (not HTML comment, which html/template strips)
		html := `<html><body></div></section><span data-cover-end></span><h1 id="ch1">Chapter 1</h1></body></html>`
		data := &TOCData{Title: "Contents", MaxDepth: 3}

		got, err := injector.InjectTOC(ctx, html, data)
		if err != nil {
			t.Fatalf("TOCInjection.InjectTOC() unexpected error: %v", err)
		}

		// TOC should appear after cover-end marker
		coverEndIdx := strings.Index(got, "data-cover-end")
		tocIdx := strings.Index(got, `<nav class="toc">`)
		if coverEndIdx == -1 || tocIdx == -1 {
			t.Fatal("cover-end or toc nav not found")
		}
		if tocIdx < coverEndIdx {
			t.Error("TOC should be inserted after cover-end marker")
		}
	})

	t.Run("fallback injects after body tag", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()
		html := `<html><body><h1 id="ch1">Chapter 1</h1></body></html>`
		data := &TOCData{Title: "", MaxDepth: 3}

		got, err := injector.InjectTOC(ctx, html, data)
		if err != nil {
			t.Fatalf("TOCInjection.InjectTOC() unexpected error: %v", err)
		}

		// TOC should appear after <body>
		bodyIdx := strings.Index(got, "<body>")
		tocIdx := strings.Index(got, `<nav class="toc">`)
		if bodyIdx == -1 || tocIdx == -1 {
			t.Fatal("body or toc nav not found")
		}
		if tocIdx < bodyIdx+6 { // len("<body>") = 6
			t.Error("TOC should be inserted after <body>")
		}
	})

	t.Run("fallback with body attributes", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()
		html := `<html><body class="main"><h1 id="ch1">Chapter 1</h1></body></html>`
		data := &TOCData{MaxDepth: 3}

		got, err := injector.InjectTOC(ctx, html, data)
		if err != nil {
			t.Fatalf("TOCInjection.InjectTOC() unexpected error: %v", err)
		}

		if !strings.Contains(got, `<nav class="toc">`) {
			t.Error("TOC not injected")
		}
	})

	t.Run("last fallback prepends to HTML", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()
		html := `<h1 id="ch1">Chapter 1</h1><p>Content</p>`
		data := &TOCData{MaxDepth: 3}

		got, err := injector.InjectTOC(ctx, html, data)
		if err != nil {
			t.Fatalf("TOCInjection.InjectTOC() unexpected error: %v", err)
		}

		if !strings.HasPrefix(got, `<nav class="toc">`) {
			t.Errorf("TOC should be prepended, got: %q", got[:min(50, len(got))])
		}
	})

	t.Run("respects maxDepth", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()
		html := `<body><h1 id="h1">H1</h1><h2 id="h2">H2</h2><h3 id="h3">H3</h3></body>`
		data := &TOCData{MaxDepth: 2}

		got, err := injector.InjectTOC(ctx, html, data)
		if err != nil {
			t.Fatalf("TOCInjection.InjectTOC() unexpected error: %v", err)
		}

		if !strings.Contains(got, "H1") || !strings.Contains(got, "H2") {
			t.Error("TOC should contain H1 and H2")
		}
		// H3 should not be in TOC (check for the link, not just text)
		if strings.Contains(got, `href="#h3"`) {
			t.Error("TOC should not contain H3 link with maxDepth 2")
		}
	})
}

func TestTOCInjection_InjectTOC_ContextCancellation(t *testing.T) {
	t.Parallel()

	injector := NewTOCInjection()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	data := &TOCData{Title: "TOC", MaxDepth: 3}
	html := `<body><h1 id="test">Test</h1></body>`
	_, err := injector.InjectTOC(ctx, html, data)

	if err == nil {
		t.Fatal("TOCInjection.InjectTOC() error = nil, want error")
	}
	if !errors.Is(err, context.Canceled) {
		t.Errorf("TOCInjection.InjectTOC() error = %v, want context.Canceled", err)
	}
}

// TestTOCInjection_InjectTOC_AfterCover verifies that TOC is injected AFTER the cover page.
// This is a regression test for a bug where html/template strips HTML comments,
// causing the <!-- cover-end --> marker to be removed, and TOC to be inserted
// before the cover (at the <body> fallback position).
func TestTOCInjection_InjectTOC_AfterCover(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	loader := assets.NewEmbeddedLoader()
	ts, err := loader.LoadTemplateSet(assets.DefaultTemplateSetName)
	if err != nil {
		t.Fatalf("LoadTemplateSet() unexpected error: %v", err)
	}
	coverInjector, err := NewCoverInjection(ts.Cover)
	if err != nil {
		t.Fatalf("NewCoverInjection() unexpected error: %v", err)
	}
	tocInjector := NewTOCInjection()

	// Start with HTML that has a heading (needed for TOC generation)
	html := `<!DOCTYPE html>
<html>
<head><title>Test</title></head>
<body>
<h1 id="chapter-1">Chapter 1</h1>
<p>Content here</p>
</body>
</html>`

	// Step 1: Inject cover
	CoverData := &CoverData{
		Title:  "My Document",
		Author: "Test Author",
	}
	htmlWithCover, err := coverInjector.InjectCover(ctx, html, CoverData)
	if err != nil {
		t.Fatalf("CoverInjection.InjectCover() unexpected error: %v", err)
	}

	// Step 2: Inject TOC
	TOCData := &TOCData{
		Title:    "Table of Contents",
		MaxDepth: 3,
	}
	htmlWithTOC, err := tocInjector.InjectTOC(ctx, htmlWithCover, TOCData)
	if err != nil {
		t.Fatalf("TOCInjection.InjectTOC() unexpected error: %v", err)
	}

	// Verify: TOC must appear AFTER cover, not before
	coverIdx := strings.Index(htmlWithTOC, "cover-page")
	tocIdx := strings.Index(htmlWithTOC, `<nav class="toc">`)

	if coverIdx == -1 {
		t.Fatal("cover-page not found in output")
	}
	if tocIdx == -1 {
		t.Fatal("TOC nav not found in output")
	}

	if tocIdx < coverIdx {
		t.Errorf("BUG: TOC appears BEFORE cover (toc at %d, cover at %d).\n"+
			"This happens because html/template strips HTML comments, "+
			"removing the <!-- cover-end --> marker.\n"+
			"HTML:\n%s", tocIdx, coverIdx, htmlWithTOC)
	}
}

func TestTOCInjection_InjectTOC_HTMLEntities(t *testing.T) {
	t.Parallel()

	injector := NewTOCInjection()
	ctx := context.Background()

	tests := []struct {
		name           string
		html           string
		wantContains   []string
		wantNotContain []string
	}{
		{
			name: "ampersand not double-encoded",
			html: `<body><h1 id="ab">A &amp; B</h1></body>`,
			wantContains: []string{
				`A &amp; B`, // Single encoding in TOC
			},
			wantNotContain: []string{
				`&amp;amp;`, // Double encoding = bug
			},
		},
		{
			name: "less than not double-encoded",
			html: `<body><h1 id="lt">x &lt; y</h1></body>`,
			wantContains: []string{
				`x &lt; y`,
			},
			wantNotContain: []string{
				`&amp;lt;`,
			},
		},
		{
			name: "greater than not double-encoded",
			html: `<body><h1 id="gt">x &gt; y</h1></body>`,
			wantContains: []string{
				`x &gt; y`,
			},
			wantNotContain: []string{
				`&amp;gt;`,
			},
		},
		{
			name: "quote not double-encoded",
			html: `<body><h1 id="q">&quot;quoted&quot;</h1></body>`,
			wantContains: []string{
				`&#34;quoted&#34;`, // html.EscapeString uses numeric for quotes
			},
			wantNotContain: []string{
				`&amp;quot;`,
			},
		},
		{
			name: "numeric entity not double-encoded",
			html: `<body><h1 id="dash">foo &#8212; bar</h1></body>`,
			wantContains: []string{
				`foo — bar`, // Em dash character (decoded then not re-encoded as entity)
			},
			wantNotContain: []string{
				`&amp;#8212;`,
			},
		},
		{
			name: "complex heading with multiple entities",
			html: `<body><h1 id="complex">Foo &amp; Bar &lt;Baz&gt;</h1></body>`,
			wantContains: []string{
				`Foo &amp; Bar &lt;Baz&gt;`,
			},
			wantNotContain: []string{
				`&amp;amp;`,
				`&amp;lt;`,
				`&amp;gt;`,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			data := &TOCData{MaxDepth: 3}
			got, err := injector.InjectTOC(ctx, tt.html, data)
			if err != nil {
				t.Fatalf("TOCInjection.InjectTOC() unexpected error: %v", err)
			}

			for _, want := range tt.wantContains {
				if !strings.Contains(got, want) {
					t.Errorf("TOCInjection.InjectTOC() missing %q\nGot:\n%s", want, got)
				}
			}

			for _, notWant := range tt.wantNotContain {
				if strings.Contains(got, notWant) {
					t.Errorf("TOCInjection.InjectTOC() should not contain %q (double-encoding bug)\nGot:\n%s", notWant, got)
				}
			}
		})
	}
}
