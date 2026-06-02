package picoloom_test

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/alnah/picoloom/v2"
)

// Example demonstrates basic markdown to HTML conversion.
// For PDF output, set HTMLOnly to false (requires Chrome).
func Example() {
	conv, err := picoloom.NewConverter()
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	defer conv.Close()

	result, err := conv.Convert(context.Background(), picoloom.Input{
		Markdown: "# Hello World\n\nThis is a test.",
		HTMLOnly: true, // Skip PDF generation for this example
	})
	if err != nil {
		fmt.Println("error:", err)
		return
	}

	// Check that HTML was generated
	if strings.Contains(string(result.HTML), "<h1") {
		fmt.Println("HTML generated successfully")
	}
	// Output: HTML generated successfully
}

// Example_withCover demonstrates adding a cover page.
func Example_withCover() {
	conv, err := picoloom.NewConverter()
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	defer conv.Close()

	result, err := conv.Convert(context.Background(), picoloom.Input{
		Markdown: "# Introduction\n\nDocument content here.",
		Cover: &picoloom.Cover{
			Title:        "Project Report",
			Subtitle:     "Q4 2025 Analysis",
			Author:       "John Doe",
			Organization: "Acme Corp",
			Date:         "2025-12-15",
			Version:      "v1.0",
		},
		HTMLOnly: true,
	})
	if err != nil {
		fmt.Println("error:", err)
		return
	}

	if strings.Contains(string(result.HTML), "Project Report") {
		fmt.Println("Cover page generated")
	}
	// Output: Cover page generated
}

// Example_withTOC demonstrates adding a table of contents.
func Example_withTOC() {
	conv, err := picoloom.NewConverter()
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	defer conv.Close()

	markdown := `# Document Title

## Chapter 1

Content for chapter 1.

## Chapter 2

Content for chapter 2.

### Section 2.1

Subsection content.
`

	result, err := conv.Convert(context.Background(), picoloom.Input{
		Markdown: markdown,
		TOC: &picoloom.TOC{
			Title:    "Contents",
			MinDepth: 2, // Start at h2 (skip document title)
			MaxDepth: 3, // Include up to h3
		},
		HTMLOnly: true,
	})
	if err != nil {
		fmt.Println("error:", err)
		return
	}

	if strings.Contains(string(result.HTML), "toc") {
		fmt.Println("TOC generated")
	}
	// Output: TOC generated
}

// Example_withCustomCSS demonstrates injecting custom CSS.
func Example_withCustomCSS() {
	conv, err := picoloom.NewConverter()
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	defer conv.Close()

	result, err := conv.Convert(context.Background(), picoloom.Input{
		Markdown: "# Styled Document\n\nCustom styling applied.",
		CSS: `
			body { font-family: Georgia, serif; }
			h1 { color: #2c3e50; border-bottom: 2px solid #3498db; }
		`,
		HTMLOnly: true,
	})
	if err != nil {
		fmt.Println("error:", err)
		return
	}

	if strings.Contains(string(result.HTML), "Georgia") {
		fmt.Println("Custom CSS injected")
	}
	// Output: Custom CSS injected
}

// Example_withSignature demonstrates adding a signature block.
func Example_withSignature() {
	conv, err := picoloom.NewConverter()
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	defer conv.Close()

	result, err := conv.Convert(context.Background(), picoloom.Input{
		Markdown: "# Report\n\nDocument content.",
		Signature: &picoloom.Signature{
			Name:         "Jane Smith",
			Title:        "Senior Engineer",
			Email:        "jane@example.com",
			Organization: "Tech Corp",
		},
		HTMLOnly: true,
	})
	if err != nil {
		fmt.Println("error:", err)
		return
	}

	if strings.Contains(string(result.HTML), "Jane Smith") {
		fmt.Println("Signature block added")
	}
	// Output: Signature block added
}

// Example_withWatermark demonstrates adding a watermark.
func Example_withWatermark() {
	conv, err := picoloom.NewConverter()
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	defer conv.Close()

	result, err := conv.Convert(context.Background(), picoloom.Input{
		Markdown: "# Draft Document\n\nThis is a draft.",
		Watermark: &picoloom.Watermark{
			Text:    "DRAFT",
			Color:   "#888888",
			Opacity: 0.1,
			Angle:   -45,
		},
		HTMLOnly: true,
	})
	if err != nil {
		fmt.Println("error:", err)
		return
	}

	if strings.Contains(string(result.HTML), "DRAFT") {
		fmt.Println("Watermark CSS generated")
	}
	// Output: Watermark CSS generated
}

// Example_withPageSettings demonstrates configuring page settings.
func Example_withPageSettings() {
	conv, err := picoloom.NewConverter()
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	defer conv.Close()

	result, err := conv.Convert(context.Background(), picoloom.Input{
		Markdown: "# A4 Document\n\nConfigured for A4 paper.",
		Page: &picoloom.PageSettings{
			Size:        picoloom.PageSizeA4,
			Orientation: picoloom.OrientationPortrait,
			Margin:      1.0, // inches
		},
		HTMLOnly: true,
	})
	if err != nil {
		fmt.Println("error:", err)
		return
	}

	if len(result.HTML) > 0 {
		fmt.Println("Page settings configured")
	}
	// Output: Page settings configured
}

// Example_withFooter demonstrates adding a page footer.
func Example_withFooter() {
	conv, err := picoloom.NewConverter()
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	defer conv.Close()

	result, err := conv.Convert(context.Background(), picoloom.Input{
		Markdown: "# Document with Footer\n\nContent here.",
		Footer: &picoloom.Footer{
			Position:       "center",
			ShowPageNumber: true,
			Date:           "2025-01-15",
			Text:           "Confidential",
		},
		HTMLOnly: true,
	})
	if err != nil {
		fmt.Println("error:", err)
		return
	}

	if len(result.HTML) > 0 {
		fmt.Println("Footer configured")
	}
	// Output: Footer configured
}

// ExampleNewConverter_withStyle demonstrates using a built-in style.
func ExampleNewConverter_withStyle() {
	conv, err := picoloom.NewConverter(picoloom.WithStyle("technical"))
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	defer conv.Close()

	result, err := conv.Convert(context.Background(), picoloom.Input{
		Markdown: "# Technical Document\n\nUsing the technical style.",
		HTMLOnly: true,
	})
	if err != nil {
		fmt.Println("error:", err)
		return
	}

	// Technical style uses system-ui font
	if strings.Contains(string(result.HTML), "system-ui") {
		fmt.Println("Technical style applied")
	}
	// Output: Technical style applied
}

// ExampleConverterPool demonstrates parallel batch processing.
func ExampleConverterPool() {
	pool := picoloom.NewConverterPool(2)

	// Process two documents in parallel
	docs := []string{
		"# Document 1\n\nFirst document.",
		"# Document 2\n\nSecond document.",
	}

	// Channel to collect results, WaitGroup to synchronize goroutines
	results := make(chan bool, len(docs))
	var wg sync.WaitGroup

	for _, doc := range docs {
		wg.Add(1)
		go func(markdown string) {
			defer wg.Done()

			conv := pool.Acquire()
			if conv == nil {
				results <- false
				return
			}
			defer pool.Release(conv)

			result, err := conv.Convert(context.Background(), picoloom.Input{
				Markdown: markdown,
				HTMLOnly: true,
			})
			results <- err == nil && strings.Contains(string(result.HTML), "Document")
		}(doc)
	}

	// Wait for all goroutines to finish before closing pool
	wg.Wait()
	pool.Close()

	// Collect results
	success := 0
	for range docs {
		if <-results {
			success++
		}
	}
	fmt.Printf("Processed %d documents\n", success)
	// Output: Processed 2 documents
}

// ExampleNewAssetLoader demonstrates loading custom assets.
func ExampleNewAssetLoader() {
	// NewAssetLoader with empty path uses embedded assets only
	loader, err := picoloom.NewAssetLoader("")
	if err != nil {
		fmt.Println("error:", err)
		return
	}

	conv, err := picoloom.NewConverter(picoloom.WithAssetLoader(loader))
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	defer conv.Close()

	result, err := conv.Convert(context.Background(), picoloom.Input{
		Markdown: "# Custom Assets\n\nUsing asset loader.",
		HTMLOnly: true,
	})
	if err != nil {
		fmt.Println("error:", err)
		return
	}

	if len(result.HTML) > 0 {
		fmt.Println("Asset loader configured")
	}
	// Output: Asset loader configured
}
