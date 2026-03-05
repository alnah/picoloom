package assets

import (
	"errors"
)

// AssetResolver combines custom and embedded loaders with fallback logic.
// When a custom loader is configured, it tries custom first, then falls back
// to embedded if the asset is not found in the custom location.
type AssetResolver struct {
	custom   AssetLoader // nil if no custom path configured
	embedded AssetLoader
}

// NewAssetResolver creates an AssetResolver.
// If customBasePath is empty, only embedded assets are used.
// If customBasePath is set, custom assets take precedence with fallback to embedded.
// Returns error if customBasePath is set but invalid.
func NewAssetResolver(customBasePath string) (*AssetResolver, error) {
	resolver := &AssetResolver{
		embedded: NewEmbeddedLoader(),
	}

	if customBasePath != "" {
		fsLoader, err := NewFilesystemLoader(customBasePath)
		if err != nil {
			return nil, err
		}
		resolver.custom = fsLoader
	}

	return resolver, nil
}

// LoadStyle loads a CSS style, trying custom loader first if available.
// Returns the style content and whether it came from the custom loader.
func (r *AssetResolver) LoadStyle(name string) (string, error) {
	return r.loadWithFallback(name, func(loader AssetLoader) (string, error) {
		return loader.LoadStyle(name)
	})
}

// LoadTemplateSet loads a template set, trying custom loader first if available.
func (r *AssetResolver) LoadTemplateSet(name string) (*TemplateSet, error) {
	// If no custom loader, use embedded directly
	if r.custom == nil {
		return r.embedded.LoadTemplateSet(name)
	}

	// Try custom loader first
	ts, err := r.custom.LoadTemplateSet(name)
	if err == nil {
		return ts, nil
	}

	// Only fall back for "not found" errors, not validation or I/O errors
	if !isNotFoundError(err) {
		return nil, err
	}

	// Fall back to embedded
	return r.embedded.LoadTemplateSet(name)
}

// loadWithFallback implements the custom-first, fallback-to-embedded logic.
func (r *AssetResolver) loadWithFallback(_ string, loadFn func(AssetLoader) (string, error)) (string, error) {
	// If no custom loader, use embedded directly
	if r.custom == nil {
		return loadFn(r.embedded)
	}

	// Try custom loader first
	content, err := loadFn(r.custom)
	if err == nil {
		return content, nil
	}

	// Only fall back for "not found" errors, not validation or I/O errors
	if !isNotFoundError(err) {
		return "", err
	}

	// Fall back to embedded
	return loadFn(r.embedded)
}

// isNotFoundError checks if the error indicates the asset was not found.
func isNotFoundError(err error) bool {
	return errors.Is(err, ErrStyleNotFound) ||
		errors.Is(err, ErrTemplateNotFound) ||
		errors.Is(err, ErrTemplateSetNotFound)
}

// HasCustomLoader returns true if a custom asset loader is configured.
func (r *AssetResolver) HasCustomLoader() bool {
	return r.custom != nil
}

// Compile-time interface check.
var _ AssetLoader = (*AssetResolver)(nil)
