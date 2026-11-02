//nolint:revive
package util

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"
)

func TestBuildTimestampOrStringRowValue(t *testing.T) {
	tests := []struct {
		name       string
		value      string
		wantString bool
	}{
		{
			name:       "valid timestamp",
			value:      "2026-05-11 12:34:56.007",
			wantString: false,
		},
		{
			name:       "zero year timestamp",
			value:      "0000-01-01 00:00:00.007",
			wantString: true,
		},
		{
			name:       "zero date timestamp",
			value:      "0000-00-00 00:00:00",
			wantString: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := BuildTimestampOrStringRowValue(tc.value, 3)
			_, err := protojson.Marshal(got)
			require.NoError(t, err)

			if tc.wantString {
				require.Equal(t, tc.value, got.GetStringValue())
				return
			}

			require.NotNil(t, got.GetTimestampValue())
			require.EqualValues(t, 3, got.GetTimestampValue().GetAccuracy())
		})
	}
}

func TestFormatTimeOnly(t *testing.T) {
	tests := []struct {
		name   string
		in     time.Time
		want   string
		wantTZ string
	}{
		{"whole second", time.Date(1, 1, 1, 12, 34, 56, 0, time.UTC), "12:34:56", "12:34:56+00:00"},
		{"milliseconds pad to six digits", time.Date(1, 1, 1, 12, 34, 56, 789000000, time.UTC), "12:34:56.789000", "12:34:56.789000+00:00"},
		{"trailing zeros are kept", time.Date(1, 1, 1, 12, 34, 56, 700000000, time.UTC), "12:34:56.700000", "12:34:56.700000+00:00"},
		{"microseconds", time.Date(1, 1, 1, 12, 34, 56, 123456000, time.UTC), "12:34:56.123456", "12:34:56.123456+00:00"},
		{"more than six digits are kept", time.Date(1, 1, 1, 12, 34, 56, 123456700, time.UTC), "12:34:56.1234567", "12:34:56.1234567+00:00"},
		{"nanoseconds", time.Date(1, 1, 1, 12, 34, 56, 123456789, time.UTC), "12:34:56.123456789", "12:34:56.123456789+00:00"},
		{"zoned", time.Date(1, 1, 1, 12, 34, 56, 700000000, time.FixedZone("", 2*60*60)), "12:34:56.700000", "12:34:56.700000+02:00"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, FormatTimeOnly(tc.in))
			require.Equal(t, tc.wantTZ, FormatTimeOnlyTZ(tc.in))
		})
	}
}
