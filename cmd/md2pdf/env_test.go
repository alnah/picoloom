package main

// Notes:
// - DefaultEnv: we test that it returns expected real implementations
//   (os.Stdout, os.Stderr, real time). We cannot test actual I/O behavior
//   without affecting the test process itself.
// - Environment injection: we test the DI pattern works correctly with mocks.
// These are acceptable gaps: we test observable behavior, not implementation details.

import (
	"bytes"
	"os"
	"testing"
	"time"

	md2pdf "github.com/alnah/go-md2pdf"
)

// ---------------------------------------------------------------------------
// TestDefaultEnv - Default environment factory
// ---------------------------------------------------------------------------

func TestDefaultEnv(t *testing.T) {
	t.Parallel()

	env := DefaultEnv()

	t.Run("returns real time from Now", func(t *testing.T) {
		before := time.Now()
		got := env.Now()
		after := time.Now()

		if got.Before(before) || got.After(after) {
			t.Errorf("Now() = %v, want time between %v and %v", got, before, after)
		}
	})

	t.Run("returns os.Stdout for Stdout", func(t *testing.T) {
		if env.Stdout != os.Stdout {
			t.Errorf("Stdout = %v, want os.Stdout", env.Stdout)
		}
	})

	t.Run("returns os.Stderr for Stderr", func(t *testing.T) {
		if env.Stderr != os.Stderr {
			t.Errorf("Stderr = %v, want os.Stderr", env.Stderr)
		}
	})

	t.Run("returns non-nil AssetLoader", func(t *testing.T) {
		if env.AssetLoader == nil {
			t.Errorf("AssetLoader = nil, want non-nil")
		}
	})
}

// ---------------------------------------------------------------------------
// TestEnvironmentInjection - Dependency injection pattern
// ---------------------------------------------------------------------------

func TestEnvironmentInjection(t *testing.T) {
	t.Parallel()

	loader, _ := md2pdf.NewAssetLoader("")

	t.Run("uses mock time function", func(t *testing.T) {
		t.Parallel()

		fixedTime := time.Date(2025, 6, 15, 12, 0, 0, 0, time.UTC)
		env := &Environment{
			Now:         func() time.Time { return fixedTime },
			Stdout:      &bytes.Buffer{},
			Stderr:      &bytes.Buffer{},
			AssetLoader: loader,
		}

		got := env.Now()
		if !got.Equal(fixedTime) {
			t.Errorf("Now() = %v, want %v", got, fixedTime)
		}
	})

	t.Run("captures output in mock stdout", func(t *testing.T) {
		t.Parallel()

		var stdout bytes.Buffer
		env := &Environment{
			Now:         time.Now,
			Stdout:      &stdout,
			Stderr:      &bytes.Buffer{},
			AssetLoader: loader,
		}

		// Simulate writing to stdout
		env.Stdout.Write([]byte("test output"))

		got := stdout.String()
		want := "test output"
		if got != want {
			t.Errorf("stdout = %q, want %q", got, want)
		}
	})

	t.Run("captures errors in mock stderr", func(t *testing.T) {
		t.Parallel()

		var stderr bytes.Buffer
		env := &Environment{
			Now:         time.Now,
			Stdout:      &bytes.Buffer{},
			Stderr:      &stderr,
			AssetLoader: loader,
		}

		// Simulate writing to stderr
		env.Stderr.Write([]byte("error output"))

		got := stderr.String()
		want := "error output"
		if got != want {
			t.Errorf("stderr = %q, want %q", got, want)
		}
	})
}
