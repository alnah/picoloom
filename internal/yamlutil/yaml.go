// Package yamlutil wraps YAML parsing to isolate the external dependency.
// This allows swapping the underlying YAML library without modifying callers.
package yamlutil

import (
	"errors"
	"fmt"

	"github.com/goccy/go-yaml"
)

// MaxInputSize limits YAML input to prevent memory exhaustion (default 1MB).
var MaxInputSize = 1 << 20

var (
	// ErrNilData indicates an empty or missing input payload.
	ErrNilData = errors.New("yamlutil: nil or empty data")
	// ErrNilDestination indicates a nil destination pointer passed to decode.
	ErrNilDestination = errors.New("yamlutil: nil destination pointer")
	// ErrInputTooLarge indicates the payload exceeds MaxInputSize.
	ErrInputTooLarge = errors.New("yamlutil: input exceeds maximum size")
)

func validateInput(data []byte, v any) error {
	if len(data) == 0 {
		return ErrNilData
	}
	if len(data) > MaxInputSize {
		return fmt.Errorf("%w: %d bytes (max %d)", ErrInputTooLarge, len(data), MaxInputSize)
	}
	if v == nil {
		return ErrNilDestination
	}
	return nil
}

// Unmarshal decodes YAML content into v after input validation.
func Unmarshal(data []byte, v any) error {
	if err := validateInput(data, v); err != nil {
		return err
	}
	if err := yaml.Unmarshal(data, v); err != nil {
		return fmt.Errorf("yamlutil: %w", err)
	}
	return nil
}

// Marshal encodes v as YAML.
func Marshal(v any) ([]byte, error) {
	result, err := yaml.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("yamlutil: %w", err)
	}
	return result, nil
}

// UnmarshalStrict rejects unknown fields in the input.
func UnmarshalStrict(data []byte, v any) error {
	if err := validateInput(data, v); err != nil {
		return err
	}
	if err := yaml.UnmarshalWithOptions(data, v, yaml.Strict()); err != nil {
		return fmt.Errorf("yamlutil: %w", err)
	}
	return nil
}
