package yamlutil_test

// Notes:
// - Marshal error branch (line 46-48 in yaml.go): not tested because yaml.Marshal
//   only fails with unmarshalable types (channels, functions) which are compile-time
//   detectable and not realistic in production usage.
// These are acceptable gaps: we test observable behavior, not implementation details.

import (
	"errors"
	"strings"
	"testing"

	"github.com/alnah/go-md2pdf/internal/yamlutil"
)

type testConfig struct {
	Name    string `yaml:"name"`
	Count   int    `yaml:"count"`
	Enabled bool   `yaml:"enabled"`
}

// ---------------------------------------------------------------------------
// TestUnmarshal - Parses YAML into Go structs
// ---------------------------------------------------------------------------

func TestUnmarshal(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		data    []byte
		dest    any
		wantErr error
		check   func(t *testing.T, v any)
	}{
		{
			name: "happy path",
			data: []byte("name: test\ncount: 42\nenabled: true"),
			dest: &testConfig{},
			check: func(t *testing.T, v any) {
				cfg := v.(*testConfig)
				if cfg.Name != "test" {
					t.Errorf("Name = %q, want %q", cfg.Name, "test")
				}
				if cfg.Count != 42 {
					t.Errorf("Count = %d, want %d", cfg.Count, 42)
				}
				if !cfg.Enabled {
					t.Error("Enabled = false, want true")
				}
			},
		},
		{
			name:    "error case: nil data",
			data:    nil,
			dest:    &testConfig{},
			wantErr: yamlutil.ErrNilData,
		},
		{
			name:    "error case: empty data",
			data:    []byte{},
			dest:    &testConfig{},
			wantErr: yamlutil.ErrNilData,
		},
		{
			name:    "error case: nil destination",
			data:    []byte("name: test"),
			dest:    nil,
			wantErr: yamlutil.ErrNilDestination,
		},
		{
			name:    "error case: invalid YAML syntax",
			data:    []byte("name: [unclosed"),
			dest:    &testConfig{},
			wantErr: errors.New("yamlutil:"), // partial match
		},
		{
			name: "edge case: unicode content",
			data: []byte("name: 日本語テスト"),
			dest: &testConfig{},
			check: func(t *testing.T, v any) {
				cfg := v.(*testConfig)
				if cfg.Name != "日本語テスト" {
					t.Errorf("Name = %q, want %q", cfg.Name, "日本語テスト")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := yamlutil.Unmarshal(tt.data, tt.dest)
			if tt.wantErr != nil {
				if err == nil {
					t.Fatalf("Unmarshal(data, dest) error = nil, want error")
				}
				if errors.Is(err, tt.wantErr) {
					return // exact match via errors.Is
				}
				if !strings.Contains(err.Error(), tt.wantErr.Error()) {
					t.Fatalf("Unmarshal(data, dest) error = %q, want containing %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("Unmarshal(data, dest) unexpected error: %v", err)
			}
			if tt.check != nil {
				tt.check(t, tt.dest)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestUnmarshalStrict - Parses YAML and rejects unknown fields
// ---------------------------------------------------------------------------

func TestUnmarshalStrict(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		data    []byte
		dest    any
		wantErr error
		check   func(t *testing.T, v any)
	}{
		{
			name: "happy path",
			data: []byte("name: strict\ncount: 10"),
			dest: &testConfig{},
			check: func(t *testing.T, v any) {
				cfg := v.(*testConfig)
				if cfg.Name != "strict" {
					t.Errorf("Name = %q, want %q", cfg.Name, "strict")
				}
				if cfg.Count != 10 {
					t.Errorf("Count = %d, want %d", cfg.Count, 10)
				}
			},
		},
		{
			name:    "error case: unknown field",
			data:    []byte("name: test\nunknown_field: value"),
			dest:    &testConfig{},
			wantErr: errors.New("yamlutil:"), // should error on unknown field
		},
		{
			name:    "error case: nil data",
			data:    nil,
			dest:    &testConfig{},
			wantErr: yamlutil.ErrNilData,
		},
		{
			name:    "error case: empty data",
			data:    []byte{},
			dest:    &testConfig{},
			wantErr: yamlutil.ErrNilData,
		},
		{
			name:    "error case: nil destination",
			data:    []byte("name: test"),
			dest:    nil,
			wantErr: yamlutil.ErrNilDestination,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := yamlutil.UnmarshalStrict(tt.data, tt.dest)
			if tt.wantErr != nil {
				if err == nil {
					t.Fatalf("UnmarshalStrict(data, dest) error = nil, want error")
				}
				if errors.Is(err, tt.wantErr) {
					return
				}
				if !strings.Contains(err.Error(), tt.wantErr.Error()) {
					t.Fatalf("UnmarshalStrict(data, dest) error = %q, want containing %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("UnmarshalStrict(data, dest) unexpected error: %v", err)
			}
			if tt.check != nil {
				tt.check(t, tt.dest)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestMarshal - Serializes Go structs to YAML
// ---------------------------------------------------------------------------

func TestMarshal(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   any
		wantErr bool
		check   func(t *testing.T, data []byte)
	}{
		{
			name:  "happy path",
			input: &testConfig{Name: "marshal", Count: 5, Enabled: true},
			check: func(t *testing.T, data []byte) {
				s := string(data)
				if !strings.Contains(s, "name: marshal") {
					t.Errorf("output missing 'name: marshal', got: %s", s)
				}
				if !strings.Contains(s, "count: 5") {
					t.Errorf("output missing 'count: 5', got: %s", s)
				}
				if !strings.Contains(s, "enabled: true") {
					t.Errorf("output missing 'enabled: true', got: %s", s)
				}
			},
		},
		{
			name:  "edge case: nil value produces null",
			input: nil,
			check: func(t *testing.T, data []byte) {
				s := strings.TrimSpace(string(data))
				if s != "null" {
					t.Errorf("output = %q, want %q", s, "null")
				}
			},
		},
		{
			name:  "edge case: unicode content",
			input: &testConfig{Name: "日本語"},
			check: func(t *testing.T, data []byte) {
				if !strings.Contains(string(data), "日本語") {
					t.Errorf("output missing unicode content, got: %s", data)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			data, err := yamlutil.Marshal(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatal("Marshal(input) error = nil, want error")
				}
				return
			}
			if err != nil {
				t.Fatalf("Marshal(input) unexpected error: %v", err)
			}
			if tt.check != nil {
				tt.check(t, data)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestRoundTrip - Verifies Marshal/Unmarshal symmetry
// ---------------------------------------------------------------------------

func TestRoundTrip(t *testing.T) {
	t.Parallel()

	original := testConfig{
		Name:    "roundtrip",
		Count:   99,
		Enabled: true,
	}

	data, err := yamlutil.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal(original) unexpected error: %v", err)
	}

	var decoded testConfig
	if err := yamlutil.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal(data, &decoded) unexpected error: %v", err)
	}

	if decoded.Name != original.Name {
		t.Errorf("Name = %q, want %q", decoded.Name, original.Name)
	}
	if decoded.Count != original.Count {
		t.Errorf("Count = %d, want %d", decoded.Count, original.Count)
	}
	if decoded.Enabled != original.Enabled {
		t.Errorf("Enabled = %v, want %v", decoded.Enabled, original.Enabled)
	}
}

// ---------------------------------------------------------------------------
// TestErrorWrapping - Verifies error types are detectable via errors.Is
// ---------------------------------------------------------------------------

func TestErrorWrapping(t *testing.T) {
	t.Parallel()

	t.Run("ErrNilData is detectable via errors.Is", func(t *testing.T) {
		t.Parallel()

		err := yamlutil.Unmarshal(nil, &testConfig{})
		if !errors.Is(err, yamlutil.ErrNilData) {
			t.Errorf("errors.Is(err, ErrNilData) = false, want true")
		}
	})

	t.Run("ErrNilDestination is detectable via errors.Is", func(t *testing.T) {
		t.Parallel()

		err := yamlutil.Unmarshal([]byte("name: test"), nil)
		if !errors.Is(err, yamlutil.ErrNilDestination) {
			t.Errorf("errors.Is(err, ErrNilDestination) = false, want true")
		}
	})

	t.Run("wrapped errors have yamlutil prefix", func(t *testing.T) {
		t.Parallel()

		err := yamlutil.Unmarshal([]byte("invalid: [unclosed"), &testConfig{})
		if err == nil {
			t.Fatal("Unmarshal(invalid) error = nil, want error")
		}
		if !strings.HasPrefix(err.Error(), "yamlutil:") {
			t.Errorf("error = %q, want prefix 'yamlutil:'", err)
		}
	})
}

// ---------------------------------------------------------------------------
// TestInputSizeLimit - Verifies MaxInputSize enforcement
// ---------------------------------------------------------------------------

// Note: This test modifies the global MaxInputSize variable, so it cannot
// run in parallel with other tests to avoid data races.

func TestInputSizeLimit(t *testing.T) {
	// Save and restore original MaxInputSize
	originalMax := yamlutil.MaxInputSize
	t.Cleanup(func() { yamlutil.MaxInputSize = originalMax })

	t.Run("edge case: input at limit succeeds", func(t *testing.T) {
		yamlutil.MaxInputSize = 100
		data := make([]byte, 100)
		copy(data, []byte("name: x"))
		var cfg testConfig
		err := yamlutil.Unmarshal(data, &cfg)
		if err != nil {
			t.Errorf("Unmarshal(data, &cfg) unexpected error: %v", err)
		}
	})

	t.Run("error case: input exceeding limit", func(t *testing.T) {
		yamlutil.MaxInputSize = 100
		data := make([]byte, 101)
		copy(data, []byte("name: x"))
		var cfg testConfig
		err := yamlutil.Unmarshal(data, &cfg)
		if !errors.Is(err, yamlutil.ErrInputTooLarge) {
			t.Errorf("Unmarshal(oversized) = %v, want ErrInputTooLarge", err)
		}
	})

	t.Run("error message includes sizes", func(t *testing.T) {
		yamlutil.MaxInputSize = 50
		data := make([]byte, 100)
		var cfg testConfig
		err := yamlutil.Unmarshal(data, &cfg)
		if err == nil {
			t.Fatal("Unmarshal(oversized) error = nil, want error")
		}
		msg := err.Error()
		if !strings.Contains(msg, "100 bytes") {
			t.Errorf("error should contain actual size, got: %s", msg)
		}
		if !strings.Contains(msg, "max 50") {
			t.Errorf("error should contain max size, got: %s", msg)
		}
	})

	t.Run("UnmarshalStrict also enforces limit", func(t *testing.T) {
		yamlutil.MaxInputSize = 100
		data := make([]byte, 101)
		copy(data, []byte("name: x"))
		var cfg testConfig
		err := yamlutil.UnmarshalStrict(data, &cfg)
		if !errors.Is(err, yamlutil.ErrInputTooLarge) {
			t.Errorf("UnmarshalStrict(oversized) = %v, want ErrInputTooLarge", err)
		}
	})
}
