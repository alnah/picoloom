package assets

import (
	"embed"
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/alnah/picoloom/v2/internal/hints"
)

//go:embed styles/*
var styles embed.FS

//go:embed templates/*/*
var templates embed.FS

//go:embed overlays/*
var overlays embed.FS

// AvailableStyles returns the names of all embedded styles (without .css extension).
// The list is sorted alphabetically.
func AvailableStyles() []string {
	entries, err := styles.ReadDir("styles")
	if err != nil {
		return nil
	}

	var names []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.HasSuffix(name, ".css") {
			names = append(names, strings.TrimSuffix(name, filepath.Ext(name)))
		}
	}
	sort.Strings(names)
	return names
}

// EmbeddedLoader loads assets from embedded filesystem.
// Implements AssetLoader interface.
type EmbeddedLoader struct{}

// NewEmbeddedLoader creates an EmbeddedLoader.
func NewEmbeddedLoader() *EmbeddedLoader {
	return &EmbeddedLoader{}
}

// LoadStyle loads a CSS style from embedded assets by name.
// The name should not include the .css extension.
func (e *EmbeddedLoader) LoadStyle(name string) (string, error) {
	if err := ValidateAssetName(name); err != nil {
		return "", err
	}

	content, err := styles.ReadFile("styles/" + name + ".css")
	if err != nil {
		return "", fmt.Errorf("%w: %q%s", ErrStyleNotFound, name, hints.ForStyleNotFound(AvailableStyles()))
	}

	return string(content), nil
}

// LoadTemplateSet loads a set of HTML templates from embedded assets.
// The name identifies a directory under templates/ containing cover.html and signature.html.
func (e *EmbeddedLoader) LoadTemplateSet(name string) (*TemplateSet, error) {
	if err := ValidateAssetName(name); err != nil {
		return nil, err
	}

	basePath := "templates/" + name + "/"

	cover, coverErr := templates.ReadFile(basePath + "cover.html")
	signature, sigErr := templates.ReadFile(basePath + "signature.html")

	// If both files are missing, the template set doesn't exist
	if coverErr != nil && sigErr != nil {
		return nil, fmt.Errorf("%w: %q", ErrTemplateSetNotFound, name)
	}

	// If only one file is missing, the template set is incomplete
	if coverErr != nil {
		return nil, fmt.Errorf("%w: %q missing cover.html", ErrIncompleteTemplateSet, name)
	}
	if sigErr != nil {
		return nil, fmt.Errorf("%w: %q missing signature.html", ErrIncompleteTemplateSet, name)
	}

	return &TemplateSet{
		Name:      name,
		Cover:     string(cover),
		Signature: string(signature),
	}, nil
}

// AdmonitionOverlayCSS returns the embedded structural admonition stylesheet.
func AdmonitionOverlayCSS() string {
	return string(mustReadOverlay("overlays/admonition.css"))
}

// WatermarkOverlayTemplate returns the embedded watermark CSS template.
func WatermarkOverlayTemplate() string {
	return string(mustReadOverlay("overlays/watermark.css.tmpl"))
}

// PageBreaksOverlayTemplate returns the embedded page-break CSS template.
func PageBreaksOverlayTemplate() string {
	return string(mustReadOverlay("overlays/pagebreaks.css.tmpl"))
}

// mustReadOverlay reads an embedded overlay file, which is a programmer error
// to miss because the files ship with the binary.
func mustReadOverlay(name string) []byte {
	content, err := overlays.ReadFile(name)
	if err != nil {
		panic(fmt.Sprintf("embedded overlay %s: %v", name, err))
	}
	return content
}

// Compile-time interface check.
var _ AssetLoader = (*EmbeddedLoader)(nil)
