package dateutil_test

// Notes:
// - ResolveDate: the error branch at line 114-116 (ParseDateFormat on DefaultDateFormat)
//   is not covered because DefaultDateFormat is a valid constant. This branch can only
//   fail if someone modifies the constant to an invalid value, which would be caught
//   at development time.
// These are acceptable gaps: we test observable behavior, not implementation details.

import (
	"errors"
	"testing"
	"time"

	"github.com/alnah/go-md2pdf/internal/dateutil"
)

// ---------------------------------------------------------------------------
// TestParseDateFormat - Token conversion and bracket escape syntax
// ---------------------------------------------------------------------------

func TestParseDateFormat(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		format  string
		want    string
		wantErr error
	}{
		// Valid token conversions
		{
			name:   "happy path: YYYY converts to Go year format",
			format: "YYYY",
			want:   "2006",
		},
		{
			name:   "happy path: YY converts to short year format",
			format: "YY",
			want:   "06",
		},
		{
			name:   "happy path: MMMM converts to full month name",
			format: "MMMM",
			want:   "January",
		},
		{
			name:   "happy path: MMM converts to short month name",
			format: "MMM",
			want:   "Jan",
		},
		{
			name:   "happy path: MM converts to zero-padded month",
			format: "MM",
			want:   "01",
		},
		{
			name:   "happy path: M converts to non-padded month",
			format: "M",
			want:   "1",
		},
		{
			name:   "happy path: DD converts to zero-padded day",
			format: "DD",
			want:   "02",
		},
		{
			name:   "happy path: D converts to non-padded day",
			format: "D",
			want:   "2",
		},
		// Combined formats
		{
			name:   "happy path: ISO date format YYYY-MM-DD",
			format: "YYYY-MM-DD",
			want:   "2006-01-02",
		},
		{
			name:   "happy path: European format DD/MM/YYYY",
			format: "DD/MM/YYYY",
			want:   "02/01/2006",
		},
		{
			name:   "happy path: US format MM/DD/YYYY",
			format: "MM/DD/YYYY",
			want:   "01/02/2006",
		},
		{
			name:   "happy path: long format with full month name",
			format: "MMMM D, YYYY",
			want:   "January 2, 2006",
		},
		{
			name:   "happy path: short month with year",
			format: "MMM YYYY",
			want:   "Jan 2006",
		},
		// Literal preservation
		{
			name:   "happy path: preserves literal separators",
			format: "YYYY/MM/DD",
			want:   "2006/01/02",
		},
		{
			name:   "happy path: preserves literal text without token chars",
			format: "(YYYY-MM-DD)",
			want:   "(2006-01-02)",
		},
		{
			name:   "happy path: preserves spaces",
			format: "DD MM YYYY",
			want:   "02 01 2006",
		},
		{
			name:   "happy path: D in text is matched as day token",
			format: "Date: YYYY",
			want:   "2ate: 2006", // D -> 2 (day), use [Date] to escape
		},
		// Bracket escape syntax
		{
			name:   "happy path: brackets preserve literal text",
			format: "[Date]: YYYY",
			want:   "Date: 2006",
		},
		{
			name:   "happy path: brackets preserve tokens as literals",
			format: "[YYYY]-MM-DD",
			want:   "YYYY-01-02",
		},
		{
			name:   "happy path: multiple bracket groups",
			format: "[Day]: D [Month]: M",
			want:   "Day: 2 Month: 1",
		},
		{
			name:   "happy path: empty brackets are valid",
			format: "YYYY[]MM",
			want:   "200601",
		},
		{
			name:   "happy path: brackets with special characters",
			format: "[Date/Time]: YYYY-MM-DD",
			want:   "Date/Time: 2006-01-02",
		},
		{
			name:    "error case: unclosed bracket",
			format:  "[Date YYYY",
			wantErr: dateutil.ErrInvalidDateFormat,
		},
		{
			name:   "edge case: nested-looking brackets use first close",
			format: "[a[b]c",
			want:   "a[bc", // [a[b] is the escaped part, c is literal
		},
		// Edge cases
		{
			name:    "error case: empty format",
			format:  "",
			wantErr: dateutil.ErrInvalidDateFormat,
		},
		{
			name:    "error case: format exceeding max length",
			format:  string(make([]byte, dateutil.MaxDateFormatLength+1)),
			wantErr: dateutil.ErrInvalidDateFormat,
		},
		{
			name:   "edge case: format at max length",
			format: string(make([]byte, dateutil.MaxDateFormatLength)),
			want:   string(make([]byte, dateutil.MaxDateFormatLength)),
		},
		{
			name:   "edge case: only literal characters",
			format: "---",
			want:   "---",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := dateutil.ParseDateFormat(tt.format)

			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("ParseDateFormat(%q) error = %v, wantErr %v", tt.format, err, tt.wantErr)
				}
				return
			}

			if err != nil {
				t.Fatalf("ParseDateFormat(%q) unexpected error: %v", tt.format, err)
			}

			if got != tt.want {
				t.Errorf("ParseDateFormat(%q) = %q, want %q", tt.format, got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestResolveDate - Auto date resolution with formats and presets
// ---------------------------------------------------------------------------

func TestResolveDate(t *testing.T) {
	t.Parallel()

	// Fixed time for deterministic tests: 2024-03-15
	fixedTime := time.Date(2024, 3, 15, 10, 30, 0, 0, time.UTC)

	tests := []struct {
		name    string
		value   string
		want    string
		wantErr error
	}{
		// Passthrough cases (non-auto values)
		{
			name:  "happy path: empty string passthrough",
			value: "",
			want:  "",
		},
		{
			name:  "happy path: literal date passthrough",
			value: "2024-01-01",
			want:  "2024-01-01",
		},
		{
			name:  "happy path: arbitrary text passthrough",
			value: "Q1 2024",
			want:  "Q1 2024",
		},
		// Auto with default format
		{
			name:  "happy path: auto uses default ISO format",
			value: "auto",
			want:  "2024-03-15",
		},
		{
			name:  "happy path: AUTO is case insensitive",
			value: "AUTO",
			want:  "2024-03-15",
		},
		{
			name:  "happy path: Auto mixed case works",
			value: "Auto",
			want:  "2024-03-15",
		},
		// Auto with custom format
		{
			name:  "happy path: auto:YYYY-MM-DD explicit ISO",
			value: "auto:YYYY-MM-DD",
			want:  "2024-03-15",
		},
		{
			name:  "happy path: auto:DD/MM/YYYY European format",
			value: "auto:DD/MM/YYYY",
			want:  "15/03/2024",
		},
		{
			name:  "happy path: auto:MM/DD/YYYY US format",
			value: "auto:MM/DD/YYYY",
			want:  "03/15/2024",
		},
		{
			name:  "happy path: auto:MMMM D, YYYY long format",
			value: "auto:MMMM D, YYYY",
			want:  "March 15, 2024",
		},
		{
			name:  "happy path: auto:MMM YYYY short month with year",
			value: "auto:MMM YYYY",
			want:  "Mar 2024",
		},
		// Preset formats
		{
			name:  "happy path: auto:iso preset",
			value: "auto:iso",
			want:  "2024-03-15",
		},
		{
			name:  "happy path: auto:european preset",
			value: "auto:european",
			want:  "15/03/2024",
		},
		{
			name:  "happy path: auto:us preset",
			value: "auto:us",
			want:  "03/15/2024",
		},
		{
			name:  "happy path: auto:long preset",
			value: "auto:long",
			want:  "March 15, 2024",
		},
		{
			name:  "happy path: preset is case insensitive",
			value: "auto:ISO",
			want:  "2024-03-15",
		},
		{
			name:  "happy path: preset mixed case works",
			value: "auto:European",
			want:  "15/03/2024",
		},
		// Bracket escape syntax
		{
			name:  "happy path: auto with bracket-escaped literal",
			value: "auto:[Date]: YYYY-MM-DD",
			want:  "Date: 2024-03-15",
		},
		// Error cases
		{
			name:    "error case: auto: with empty format",
			value:   "auto:",
			wantErr: dateutil.ErrInvalidDateFormat,
		},
		{
			name:    "error case: autoX invalid syntax",
			value:   "autoX",
			wantErr: dateutil.ErrInvalidDateFormat,
		},
		{
			name:    "error case: auto123 invalid syntax",
			value:   "auto123",
			wantErr: dateutil.ErrInvalidDateFormat,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := dateutil.ResolveDate(tt.value, fixedTime)

			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("ResolveDate(%q) error = %v, wantErr %v", tt.value, err, tt.wantErr)
				}
				return
			}

			if err != nil {
				t.Fatalf("ResolveDate(%q) unexpected error: %v", tt.value, err)
			}

			if got != tt.want {
				t.Errorf("ResolveDate(%q) = %q, want %q", tt.value, got, tt.want)
			}
		})
	}
}
