// Package picoloom converts Markdown documents to PDF using headless Chrome.
//
// # Quick Start
//
// Create a converter, convert markdown, and close when done:
//
//	conv, err := picoloom.NewConverter()
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer conv.Close()
//
//	result, err := conv.Convert(ctx, picoloom.Input{
//	    Markdown: "# Hello\n\nWorld",
//	})
//	if err != nil {
//	    log.Fatal(err)
//	}
//	os.WriteFile("output.pdf", result.PDF, 0644)
//
// The result contains both the PDF bytes (result.PDF) and the intermediate
// HTML (result.HTML) for debugging. Use Input.HTMLOnly to skip PDF generation.
//
// # Conversion Pipeline
//
// The conversion process follows these stages:
//
//  1. Markdown preprocessing (line normalization, ==highlight== syntax)
//  2. Markdown to HTML conversion via Goldmark (GFM, syntax highlighting)
//  3. HTML injection (CSS, cover page, TOC, signature block)
//  4. PDF rendering via headless Chrome (go-rod)
//
// # Configuration
//
// Use functional options to customize the converter:
//
//	conv, err := picoloom.NewConverter(
//	    picoloom.WithTimeout(2 * time.Minute),
//	    picoloom.WithStyle("technical"),
//	    picoloom.WithAssetPath("/path/to/custom/assets"),
//	)
//
// Per-conversion options are passed via Input:
//
//	result, err := conv.Convert(ctx, picoloom.Input{
//	    Markdown:  content,
//	    SourceDir: "/path/to/markdown",  // for relative image paths
//	    CSS:       "body { font-size: 14px; }",
//	    Page:      &picoloom.PageSettings{Size: "a4"},
//	    Footer:    &picoloom.Footer{ShowPageNumber: true},
//	    Cover:     &picoloom.Cover{Title: "Report"},
//	    TOC:       &picoloom.TOC{Title: "Contents"},
//	    Watermark: &picoloom.Watermark{Text: "DRAFT"},
//	    Signature: &picoloom.Signature{Name: "John Doe"},
//	})
//
// # Server and Multi-Tenant Safety
//
// For services, APIs, queues, or multi-tenant applications, configure an
// explicit Markdown size limit:
//
//	conv, err := picoloom.NewConverter(
//	    picoloom.WithMaxMarkdownBytes(1 << 20), // 1 MiB
//	)
//
// The default limit is 0 to preserve compatibility with large local documents.
// Markdown parsing is CPU-bound, and Goldmark does not accept a standard
// context.Context cancellation signal while parsing. Picoloom checks
// cancellation around parsing, but size limits are the reliable guard that
// rejects oversized input before preprocessing and parsing begin.
//
// # Parallel Processing
//
// For batch conversion, use ConverterPool to manage multiple browser instances:
//
//	pool := picoloom.NewConverterPool(4)
//	defer pool.Close()
//
//	conv := pool.Acquire()
//	defer pool.Release(conv)
//	result, err := conv.Convert(ctx, input)
//
// Each converter owns a browser process. ResolvePoolSize(0) returns a
// conservative automatic size clamped to MaxPoolSize. Explicit library pool
// sizes are not capped, so only request values above MaxPoolSize when the
// environment has enough memory and process capacity.
//
// # Custom Assets
//
// Override built-in styles and templates using AssetLoader:
//
//	loader, err := picoloom.NewAssetLoader("/path/to/assets")
//	conv, err := picoloom.NewConverter(picoloom.WithAssetLoader(loader))
//
// Asset directory structure:
//
//	assets/
//	├── styles/
//	│   └── custom.css
//	└── templates/
//	    └── custom/
//	        ├── cover.html
//	        └── signature.html
//
// # Browser Requirements
//
// PDF generation requires Chrome/Chromium. The go-rod library automatically
// downloads a managed Chromium instance on first run (~/.cache/rod/browser/).
//
// For containers and CI environments, set ROD_NO_SANDBOX=1 to disable the
// Chrome sandbox. Use ROD_BROWSER_BIN to specify a custom Chrome binary.
package picoloom
