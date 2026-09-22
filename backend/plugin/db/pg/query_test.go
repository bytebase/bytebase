package pg

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"
)

func TestBuildTimestamptzRowValue(t *testing.T) {
	tests := []struct {
		name       string
		typeName   string
		input      time.Time
		wantString string // non-empty means we expect a string fallback with this value
	}{
		{
			name:     "valid timestamp",
			typeName: "TIMESTAMP",
			input:    time.Date(2026, 5, 11, 12, 34, 56, 0, time.UTC),
		},
		{
			name:     "valid timestamptz",
			typeName: "TIMESTAMPTZ",
			input:    time.Date(2026, 5, 11, 12, 34, 56, 0, time.FixedZone("", 2*60*60)),
		},
		{
			name:       "BC timestamp falls back to PG-formatted string",
			typeName:   "TIMESTAMP",
			input:      time.Date(0, 12, 31, 15, 0, 0, 0, time.UTC), // Go year 0 == 1 BC
			wantString: "0001-12-31 15:00:00 BC",
		},
		{
			name:       "year 10000 timestamp omits zone",
			typeName:   "TIMESTAMP",
			input:      time.Date(10000, 1, 1, 0, 0, 0, 0, time.UTC),
			wantString: "10000-01-01 00:00:00",
		},
		{
			name:       "year 10000 timestamptz keeps offset",
			typeName:   "TIMESTAMPTZ",
			input:      time.Date(10000, 5, 11, 12, 34, 56, 0, time.FixedZone("", 2*60*60)),
			wantString: "10000-05-11 12:34:56+02:00",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := buildTimestamptzRowValue(tc.typeName, tc.input, 6)

			// The whole point of the fix: marshaling must never fail.
			_, err := protojson.Marshal(got)
			require.NoError(t, err)

			if tc.wantString != "" {
				require.Equal(t, tc.wantString, got.GetStringValue())
				return
			}
			if tc.typeName == "TIMESTAMP" {
				require.NotNil(t, got.GetTimestampValue())
			} else {
				require.NotNil(t, got.GetTimestampTzValue())
			}
		})
	}
}

func TestPadZeroes(t *testing.T) {
	tests := []struct {
		name     string
		rawStr   string
		acc      int
		expected string
	}{
		{
			name:     "No decimal part",
			rawStr:   "12:34:56",
			acc:      6,
			expected: "12:34:56",
		},
		{
			name:     "Already correct precision",
			rawStr:   "12:34:56.123456",
			acc:      6,
			expected: "12:34:56.123456",
		},
		{
			name:     "Less precision",
			rawStr:   "12:34:56.123",
			acc:      6,
			expected: "12:34:56.123000",
		},
		{
			name:     "More precision",
			rawStr:   "12:34:56.123456789",
			acc:      6,
			expected: "12:34:56.123456789",
		},
		{
			name:     "With timezone",
			rawStr:   "12:34:56.123+02:00",
			acc:      6,
			expected: "12:34:56.123000+02:00",
		},
		{
			name:     "With negative timezone",
			rawStr:   "12:34:56.123-02:00",
			acc:      6,
			expected: "12:34:56.123000-02:00",
		},
		{
			name:     "Invalid format: timezone before decimal (edge case fix)",
			rawStr:   "12:34-05:56.123",
			acc:      6,
			expected: "12:34-05:56.123000",
		},
		{
			name:     "Negative interval: leading minus should be ignored",
			rawStr:   "-00:04:37.530865",
			acc:      6,
			expected: "-00:04:37.530865",
		},
		{
			name:     "Negative interval with short precision",
			rawStr:   "-00:02:45.25",
			acc:      6,
			expected: "-00:02:45.250000",
		},
		{
			name:     "Negative interval without decimal",
			rawStr:   "-00:05:00",
			acc:      6,
			expected: "-00:05:00",
		},
		{
			name:     "Negative interval with timezone (minus after decimal)",
			rawStr:   "-12:34:56.123-05:00",
			acc:      6,
			expected: "-12:34:56.123-05:00",
		},
	}

	for _, test := range tests {
		got := padZeroes(test.rawStr, test.acc)
		require.Equal(t, test.expected, got)
	}
}
