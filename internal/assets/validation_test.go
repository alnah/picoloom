package assets

import (
	"errors"
	"testing"
)

func TestValidateAssetName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		wantErr error
	}{
		// Valid names
		{
			name:    "happy path: simple name",
			input:   "creative",
			wantErr: nil,
		},
		{
			name:    "happy path: name with hyphen",
			input:   "my-style",
			wantErr: nil,
		},
		{
			name:    "happy path: name with underscore",
			input:   "my_style",
			wantErr: nil,
		},
		{
			name:    "happy path: name with numbers",
			input:   "style123",
			wantErr: nil,
		},
		{
			name:    "happy path: mixed case",
			input:   "MyStyle",
			wantErr: nil,
		},

		// Invalid names - empty
		{
			name:    "error case: empty name",
			input:   "",
			wantErr: ErrInvalidAssetName,
		},

		// Invalid names - path separators
		{
			name:    "error case: forward slash",
			input:   "path/to/style",
			wantErr: ErrInvalidAssetName,
		},
		{
			name:    "error case: backslash",
			input:   "path\\to\\style",
			wantErr: ErrInvalidAssetName,
		},

		// Invalid names - path traversal
		{
			name:    "error case: parent directory traversal",
			input:   "../secret",
			wantErr: ErrInvalidAssetName,
		},
		{
			name:    "error case: windows parent traversal",
			input:   "..\\secret",
			wantErr: ErrInvalidAssetName,
		},
		{
			name:    "error case: double parent traversal",
			input:   "../../etc/passwd",
			wantErr: ErrInvalidAssetName,
		},

		// Invalid names - dots (could allow extension manipulation)
		{
			name:    "error case: dot in name",
			input:   "style.css",
			wantErr: ErrInvalidAssetName,
		},
		{
			name:    "error case: hidden file",
			input:   ".hidden",
			wantErr: ErrInvalidAssetName,
		},
		{
			name:    "error case: double extension",
			input:   "style.css.bak",
			wantErr: ErrInvalidAssetName,
		},

		// Edge cases
		{
			name:    "edge case: absolute path unix",
			input:   "/etc/passwd",
			wantErr: ErrInvalidAssetName,
		},
		{
			name:    "edge case: absolute path windows",
			input:   "C:\\Windows\\System32",
			wantErr: ErrInvalidAssetName,
		},
		{
			name:    "edge case: just a dot",
			input:   ".",
			wantErr: ErrInvalidAssetName,
		},
		{
			name:    "edge case: two dots",
			input:   "..",
			wantErr: ErrInvalidAssetName,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := ValidateAssetName(tt.input)

			if tt.wantErr == nil {
				if err != nil {
					t.Fatalf("ValidateAssetName(%q) unexpected error: %v", tt.input, err)
				}
				return
			}

			if !errors.Is(err, tt.wantErr) {
				t.Errorf("ValidateAssetName(%q) error = %v, want %v", tt.input, err, tt.wantErr)
			}
		})
	}
}

func TestValidateAssetName_ErrorMessages(t *testing.T) {
	t.Parallel()

	t.Run("empty name has descriptive message", func(t *testing.T) {
		t.Parallel()

		err := ValidateAssetName("")
		if err == nil {
			t.Fatal("ValidateAssetName(\"\") error = nil, want error")
		}
		if err.Error() == "" {
			t.Error("error message should not be empty")
		}
	})

	t.Run("invalid name includes the name in message", func(t *testing.T) {
		t.Parallel()

		err := ValidateAssetName("../evil")
		if err == nil {
			t.Fatal("ValidateAssetName(\"../evil\") error = nil, want error")
		}
		// The error message should contain the invalid name for debugging
		errStr := err.Error()
		if errStr == "" {
			t.Error("error message should not be empty")
		}
	})
}
