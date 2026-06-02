package pipeline

import (
	"context"
	"strings"
	"testing"
)

func FuzzGoldmarkConverterToHTML(f *testing.F) {
	seeds := []string{
		"# Heading\n\nParagraph with **bold** and *italic* text.",
		"| A | B |\n|---|---|\n| 1 | 2 |",
		"- [x] Done\n- [ ] Todo\n\nhttps://example.com",
		"Text[^1]\n\n[^1]: Footnote content",
		"```go\nfunc main() { println(\"hello\") }\n```",
		"<script>alert('xss')</script>",
		"[broken link](\n\n![image](../logo.png)",
		"# 日本語\n\nBonjour le monde\n\n> Citation",
	}
	for _, seed := range seeds {
		f.Add(seed)
	}

	converter := NewGoldmarkConverter()
	f.Fuzz(func(t *testing.T, markdown string) {
		if len(markdown) > 16*1024 {
			t.Skip("keep fuzz case small")
		}

		got, err := converter.ToHTML(context.Background(), markdown)
		if err != nil {
			t.Fatalf("ToHTML(ctx, markdown) error = %v", err)
		}
		for _, want := range []string{"<!DOCTYPE html>", "<html>", "<body>", "</html>"} {
			if !strings.Contains(got, want) {
				t.Fatalf("ToHTML(ctx, markdown) missing wrapper %q; got: %q", want, got)
			}
		}
	})
}
