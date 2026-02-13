package assets

import (
	"errors"
	"strings"
	"testing"
)

func TestNewEmbeddedLoader(t *testing.T) {
	t.Parallel()

	loader := NewEmbeddedLoader()
	if loader == nil {
		t.Fatal("NewEmbeddedLoader() returned nil")
	}
}

func TestEmbeddedLoader_LoadStyle(t *testing.T) {
	t.Parallel()

	loader := NewEmbeddedLoader()

	tests := []struct {
		name        string
		styleName   string
		wantErr     error
		wantContain string
	}{
		{
			name:        "happy path: loads creative style",
			styleName:   "creative",
			wantErr:     nil,
			wantContain: "font-family",
		},
		{
			name:      "error case: nonexistent style",
			styleName: "nonexistent-style-xyz",
			wantErr:   ErrStyleNotFound,
		},
		{
			name:      "error case: empty name",
			styleName: "",
			wantErr:   ErrInvalidAssetName,
		},
		{
			name:      "error case: path traversal with forward slash",
			styleName: "../secret",
			wantErr:   ErrInvalidAssetName,
		},
		{
			name:      "error case: path traversal with backslash",
			styleName: "..\\secret",
			wantErr:   ErrInvalidAssetName,
		},
		{
			name:      "error case: name with dot",
			styleName: "style.name",
			wantErr:   ErrInvalidAssetName,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := loader.LoadStyle(tt.styleName)

			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("LoadStyle(%q) error = %v, want %v", tt.styleName, err, tt.wantErr)
				}
				return
			}

			if err != nil {
				t.Fatalf("LoadStyle(%q) unexpected error: %v", tt.styleName, err)
			}

			if tt.wantContain != "" && !strings.Contains(got, tt.wantContain) {
				t.Errorf("LoadStyle(%q) content should contain %q", tt.styleName, tt.wantContain)
			}
		})
	}
}

func TestEmbeddedLoader_LoadTemplateSet(t *testing.T) {
	t.Parallel()

	loader := NewEmbeddedLoader()

	tests := []struct {
		name    string
		setName string
		wantErr error
	}{
		{
			name:    "happy path: loads default template set",
			setName: "default",
			wantErr: nil,
		},
		{
			name:    "error case: nonexistent template set",
			setName: "nonexistent-template-xyz",
			wantErr: ErrTemplateSetNotFound,
		},
		{
			name:    "error case: empty name",
			setName: "",
			wantErr: ErrInvalidAssetName,
		},
		{
			name:    "error case: path traversal",
			setName: "../secret",
			wantErr: ErrInvalidAssetName,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ts, err := loader.LoadTemplateSet(tt.setName)

			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("LoadTemplateSet(%q) error = %v, want %v", tt.setName, err, tt.wantErr)
				}
				return
			}

			if err != nil {
				t.Fatalf("LoadTemplateSet(%q) unexpected error: %v", tt.setName, err)
			}

			if !strings.Contains(ts.Cover, "cover") {
				t.Errorf("LoadTemplateSet(%q) cover should contain 'cover'", tt.setName)
			}
			if !strings.Contains(ts.Signature, "signature") {
				t.Errorf("LoadTemplateSet(%q) signature should contain 'signature'", tt.setName)
			}
		})
	}
}

func TestEmbeddedLoaderImplementsAssetLoader(t *testing.T) {
	t.Parallel()

	var _ AssetLoader = (*EmbeddedLoader)(nil)
}

func TestAvailableStyles(t *testing.T) {
	t.Parallel()

	styles := AvailableStyles()

	if len(styles) == 0 {
		t.Fatal("AvailableStyles() returned empty list")
	}

	// Check that expected styles are present
	expectedStyles := []string{"default", "technical", "creative", "academic"}
	for _, expected := range expectedStyles {
		found := false
		for _, s := range styles {
			if s == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("AvailableStyles() missing expected style %q", expected)
		}
	}

	// Check that list is sorted
	for i := 1; i < len(styles); i++ {
		if styles[i] < styles[i-1] {
			t.Errorf("AvailableStyles() not sorted: %q comes after %q", styles[i], styles[i-1])
		}
	}
}

func TestEmbeddedLoader_LoadStyle_ErrorIncludesAvailableStyles(t *testing.T) {
	t.Parallel()

	loader := NewEmbeddedLoader()
	_, err := loader.LoadStyle("nonexistent-style")

	if err == nil {
		t.Fatal("LoadStyle(\"nonexistent-style\") error = nil, want error")
	}

	errMsg := err.Error()
	if !strings.Contains(errMsg, "hint:") {
		t.Error("LoadStyle(\"nonexistent-style\") error should contain hint")
	}
	if !strings.Contains(errMsg, "available:") {
		t.Error("LoadStyle(\"nonexistent-style\") error should list available styles")
	}
	if !strings.Contains(errMsg, "default") {
		t.Error("LoadStyle(\"nonexistent-style\") error hint should include 'default' style")
	}
}
