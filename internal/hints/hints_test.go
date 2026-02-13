package hints

// Notes:
// - ForBrowserConnect tests cannot use t.Parallel() because they:
//   1. Use t.Setenv() which modifies process environment
//   2. Modify the package-level IsInContainer variable
// These are acceptable gaps: we test observable behavior through environment manipulation.

import (
	"strings"
	"testing"
)

func TestForBrowserConnect(t *testing.T) {
	t.Run("in CI environment", func(t *testing.T) {
		// Save and restore IsInContainer (not parallel-safe, see package notes)
		orig := IsInContainer
		defer func() { IsInContainer = orig }()
		IsInContainer = func() bool { return false }

		t.Setenv("CI", "true")
		t.Setenv("ROD_NO_SANDBOX", "")
		t.Setenv("ROD_BROWSER_BIN", "")

		got := ForBrowserConnect()

		if !strings.Contains(got, "hint:") {
			t.Errorf("ForBrowserConnect() missing hint prefix, got %q", got)
		}
		if !strings.Contains(got, "ROD_NO_SANDBOX") {
			t.Errorf("ForBrowserConnect() missing ROD_NO_SANDBOX suggestion in CI, got %q", got)
		}
		if !strings.Contains(got, "ROD_BROWSER_BIN") {
			t.Errorf("ForBrowserConnect() missing ROD_BROWSER_BIN suggestion, got %q", got)
		}
	})

	t.Run("in Docker container", func(t *testing.T) {
		orig := IsInContainer
		defer func() { IsInContainer = orig }()
		IsInContainer = func() bool { return true }

		t.Setenv("CI", "")
		t.Setenv("ROD_NO_SANDBOX", "")
		t.Setenv("ROD_BROWSER_BIN", "")

		got := ForBrowserConnect()

		if !strings.Contains(got, "ROD_NO_SANDBOX") {
			t.Errorf("ForBrowserConnect() missing ROD_NO_SANDBOX suggestion in Docker, got %q", got)
		}
	})

	t.Run("ROD_NO_SANDBOX already set", func(t *testing.T) {
		orig := IsInContainer
		defer func() { IsInContainer = orig }()
		IsInContainer = func() bool { return true }

		t.Setenv("CI", "")
		t.Setenv("ROD_NO_SANDBOX", "1")
		t.Setenv("ROD_BROWSER_BIN", "")

		got := ForBrowserConnect()

		if strings.Contains(got, "ROD_NO_SANDBOX") {
			t.Errorf("ForBrowserConnect() should not suggest ROD_NO_SANDBOX when already set, got %q", got)
		}
	})

	t.Run("ROD_BROWSER_BIN already set", func(t *testing.T) {
		orig := IsInContainer
		defer func() { IsInContainer = orig }()
		IsInContainer = func() bool { return false }

		t.Setenv("CI", "")
		t.Setenv("ROD_NO_SANDBOX", "")
		t.Setenv("ROD_BROWSER_BIN", "/usr/bin/chrome")

		got := ForBrowserConnect()

		if strings.Contains(got, "ROD_BROWSER_BIN") {
			t.Errorf("ForBrowserConnect() should not suggest ROD_BROWSER_BIN when already set, got %q", got)
		}
	})

	t.Run("no sandbox hint needed outside CI/Docker", func(t *testing.T) {
		orig := IsInContainer
		defer func() { IsInContainer = orig }()
		IsInContainer = func() bool { return false }

		t.Setenv("CI", "")
		t.Setenv("ROD_NO_SANDBOX", "")
		t.Setenv("ROD_BROWSER_BIN", "/usr/bin/chrome")

		got := ForBrowserConnect()

		// Should have no sandbox hint (not in CI/Docker) but no browser hint
		if strings.Contains(got, "ROD_BROWSER_BIN") {
			t.Errorf("ForBrowserConnect() should not suggest ROD_BROWSER_BIN when set, got %q", got)
		}
	})

	t.Run("all environment variables configured", func(t *testing.T) {
		orig := IsInContainer
		defer func() { IsInContainer = orig }()
		IsInContainer = func() bool { return true } // In Docker

		t.Setenv("CI", "true")
		t.Setenv("ROD_NO_SANDBOX", "1")
		t.Setenv("ROD_BROWSER_BIN", "/usr/bin/chrome")

		got := ForBrowserConnect()

		// Both env vars set, should return empty hint
		if got != "" {
			t.Errorf("ForBrowserConnect() = %q, want empty string", got)
		}
	})
}

func TestForTimeout(t *testing.T) {
	got := ForTimeout()

	if !strings.Contains(got, "hint:") {
		t.Errorf("ForTimeout() missing hint prefix, got %q", got)
	}
	if !strings.Contains(got, "--timeout") {
		t.Errorf("ForTimeout() missing --timeout flag mention, got %q", got)
	}
}

func TestForConfigNotFound(t *testing.T) {
	tests := []struct {
		name     string
		paths    []string
		wantHint bool
		contains string
	}{
		{
			name:     "empty paths",
			paths:    []string{},
			wantHint: true,
			contains: "--config",
		},
		{
			name:     "with paths",
			paths:    []string{"./foo.yaml", "~/.config/go-md2pdf/foo.yaml"},
			wantHint: true,
			contains: "go-md2pdf/foo.yaml",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ForConfigNotFound(tt.paths)

			if tt.wantHint && !strings.Contains(got, "hint:") {
				t.Errorf("ForConfigNotFound(%v) missing hint prefix, got %q", tt.paths, got)
			}
			if !strings.Contains(got, tt.contains) {
				t.Errorf("ForConfigNotFound(%v) = %q, want to contain %q", tt.paths, got, tt.contains)
			}
		})
	}
}

func TestForOutputDirectory(t *testing.T) {
	got := ForOutputDirectory()

	if !strings.Contains(got, "hint:") {
		t.Errorf("ForOutputDirectory() missing hint prefix, got %q", got)
	}
	if !strings.Contains(got, "parent directory") {
		t.Errorf("ForOutputDirectory() missing parent directory mention, got %q", got)
	}
}

func TestForStyleNotFound(t *testing.T) {
	tests := []struct {
		name      string
		available []string
		wantEmpty bool
		contains  string
	}{
		{
			name:      "empty available",
			available: []string{},
			wantEmpty: true,
		},
		{
			name:      "with styles",
			available: []string{"default", "technical"},
			contains:  "default, technical",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ForStyleNotFound(tt.available)

			if tt.wantEmpty && got != "" {
				t.Errorf("ForStyleNotFound(%v) = %q, want empty string", tt.available, got)
			}
			if !tt.wantEmpty && !strings.Contains(got, tt.contains) {
				t.Errorf("ForStyleNotFound(%v) = %q, want to contain %q", tt.available, got, tt.contains)
			}
		})
	}
}

func TestForSignatureImage(t *testing.T) {
	got := ForSignatureImage()

	if !strings.Contains(got, "hint:") {
		t.Errorf("ForSignatureImage() missing hint prefix, got %q", got)
	}
	if !strings.Contains(got, "PNG") {
		t.Errorf("ForSignatureImage() missing PNG format mention, got %q", got)
	}
	if !strings.Contains(got, "URL") {
		t.Errorf("ForSignatureImage() missing URL mention, got %q", got)
	}
}

func TestFormat(t *testing.T) {
	t.Run("consistency across all hint functions", func(t *testing.T) {
		// All hints should start with newline, spaces, and "hint:"
		hints := []string{
			ForTimeout(),
			ForOutputDirectory(),
			ForSignatureImage(),
		}

		for _, h := range hints {
			if !strings.HasPrefix(h, "\n  hint: ") {
				t.Errorf("hint format inconsistent: got %q, want prefix %q", h, "\n  hint: ")
			}
		}
	})
}
