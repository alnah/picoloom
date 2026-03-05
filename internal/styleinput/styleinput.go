// Package styleinput classifies user-provided style values so callers can
// resolve style names, files, and inline CSS through one decision path.
package styleinput

import "github.com/alnah/go-md2pdf/internal/fileutil"

// Source identifies how a style input should be resolved.
type Source int

const (
	// SourceNone indicates that no explicit style input or default was provided.
	SourceNone Source = iota
	// SourceFile indicates a style file path.
	SourceFile
	// SourceRawCSS indicates inline CSS content.
	SourceRawCSS
	// SourceName indicates a named built-in style.
	SourceName
)

// Classify returns the style source and normalized value.
// Empty input falls back to defaultValue.
func Classify(input, defaultValue string, allowRawCSS bool) (Source, string) {
	value := input
	if value == "" {
		value = defaultValue
	}
	if value == "" {
		return SourceNone, ""
	}
	if fileutil.IsFilePath(value) {
		return SourceFile, value
	}
	if allowRawCSS && fileutil.IsCSS(value) {
		return SourceRawCSS, value
	}
	return SourceName, value
}
