package pipeline

import (
	"net/url"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/net/html"
)

func FuzzRewriteRelativePaths(f *testing.F) {
	seeds := []struct {
		html      string
		sourceDir string
	}{
		{`<img src="./images/logo.png" src="../secret.png">`, "/docs"},
		{`<!DOCTYPE html [ <!ENTITY weird "value"> ]><html><body><a href="docs/report.html?x=1&amp;y=2">report</a></body></html>`, "/docs"},
		{`<svg><foreignObject><img src="./inside.svg.png"></foreignObject></svg><math><mi href="./math">x</mi></math>`, "/docs"},
		{`<div><p><img src="unterminated"><a href="../secret.png">secret`, "/docs"},
		{`<img src="../secret.png"><a href="images/../../../secret.md">secret</a>`, "/docs"},
		{`<img src="https://example.com/logo.png"><a href="#section">anchor</a><img src="data:image/png;base64,AAAA"><a href="file:///already/absolute"><img src="//cdn.example.com/logo.png">`, "/docs"},
		{`<p>empty source dir leaves input unchanged</p><img src="./logo.png">`, ""},
		{`plain text &amp; entities < broken >`, "/docs"},
	}
	for _, seed := range seeds {
		f.Add(seed.html, seed.sourceDir)
	}

	f.Fuzz(func(t *testing.T, htmlContent, sourceDir string) {
		if len(htmlContent) > 16*1024 || len(sourceDir) > 1024 {
			t.Skip("keep fuzz case small")
		}

		got, err := RewriteRelativePaths(htmlContent, sourceDir)
		if err != nil {
			t.Fatalf("RewriteRelativePaths returned error: %v", err)
		}

		if sourceDir == "" {
			if got != htmlContent {
				t.Fatalf("RewriteRelativePaths with empty sourceDir changed input: got %q, want %q", got, htmlContent)
			}
			return
		}

		_, _, err = parseHTML(got)
		if err != nil {
			t.Fatalf("rewritten HTML cannot be parsed again: %v; html: %q", err, got)
		}

		if strings.Contains(htmlContent, "file://") {
			return
		}

		absSourceDir, err := filepath.Abs(sourceDir)
		if err != nil {
			t.Skipf("sourceDir cannot be made absolute: %v", err)
		}
		assertGeneratedFileURLsStayUnderSourceDir(t, got, absSourceDir)
	})
}

func assertGeneratedFileURLsStayUnderSourceDir(t *testing.T, htmlContent, sourceDir string) {
	t.Helper()

	doc, _, err := parseHTML(htmlContent)
	if err != nil {
		t.Fatalf("parseHTML(%q) error = %v", htmlContent, err)
	}

	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && (n.Data == "img" || n.Data == "a") {
			for _, attr := range n.Attr {
				if attr.Key != "src" && attr.Key != "href" {
					continue
				}
				if !strings.HasPrefix(attr.Val, "file://") {
					continue
				}
				parsed, err := url.Parse(attr.Val)
				if err != nil {
					t.Fatalf("generated file URL %q is invalid: %v", attr.Val, err)
				}
				path := filepath.FromSlash(parsed.Path)
				if !isPathUnderDir(path, sourceDir) {
					t.Fatalf("generated file URL %q escaped sourceDir %q", attr.Val, sourceDir)
				}
			}
		}

		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)
}
