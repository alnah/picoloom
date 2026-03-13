package main

import (
	"path/filepath"
	"testing"
)

func TestShouldProcessPath(t *testing.T) {
	t.Parallel()

	tests := []struct {
		path string
		want bool
	}{
		{path: "main.go", want: true},
		{path: "go.mod", want: true},
		{path: "README.md", want: true},
		{path: "Dockerfile", want: true},
		{path: "Makefile", want: true},
		{path: "image.png", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			t.Parallel()
			if got := shouldProcessPath(tt.path); got != tt.want {
				t.Fatalf("shouldProcessPath(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}

func TestRewriteContent_GoFile(t *testing.T) {
	t.Parallel()

	input := []byte(`package md2pdf_test

import md2pdf "github.com/alnah/go-md2pdf"

func example() {
	_ = md2pdf.PageSizeA4
}
`)

	got, replacements := rewriteContent(filepath.Join("pkg", "example.go"), input)
	want := `package picoloom_test

import picoloom "github.com/alnah/picoloom/v2"

func example() {
	_ = picoloom.PageSizeA4
}
`

	if string(got) != want {
		t.Fatalf("rewriteContent() mismatch:\n--- got ---\n%s\n--- want ---\n%s", got, want)
	}
	if replacements != 4 {
		t.Fatalf("rewriteContent() replacements = %d, want 4", replacements)
	}
}

func TestRewriteContent_TextFile(t *testing.T) {
	t.Parallel()

	input := []byte("MD2PDF_STYLE=technical\nconfig=md2pdf.yaml\nimage=ghcr.io/alnah/go-md2pdf\n")
	got, replacements := rewriteContent("README.md", input)
	want := "PICOLOOM_STYLE=technical\nconfig=picoloom.yaml\nimage=ghcr.io/alnah/picoloom\n"

	if string(got) != want {
		t.Fatalf("rewriteContent() mismatch:\n--- got ---\n%s\n--- want ---\n%s", got, want)
	}
	if replacements != 3 {
		t.Fatalf("rewriteContent() replacements = %d, want 3", replacements)
	}
}
