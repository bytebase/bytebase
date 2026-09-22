//nolint:revive
package common

import (
	"math"
	"testing"
	"unicode/utf8"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTruncateString(t *testing.T) {
	tests := []struct {
		name      string
		str       string
		limit     int
		want      string
		truncated bool
	}{
		{
			name:      "simple truncate 0",
			str:       "0123",
			limit:     0,
			want:      "",
			truncated: true,
		},
		{
			name:      "simple truncate 2",
			str:       "0123",
			limit:     2,
			want:      "01",
			truncated: true,
		},
		{
			name:      "simple truncate 3",
			str:       "0123",
			limit:     3,
			want:      "012",
			truncated: true,
		},
		{
			name:      "simple truncate 4",
			str:       "0123",
			limit:     4,
			want:      "0123",
			truncated: false,
		},
		{
			name:      "simple truncate 20",
			str:       "0123",
			limit:     20,
			want:      "0123",
			truncated: false,
		},
		{
			name:      "unicode truncate 5",
			str:       "H㐀〾▓朗퐭텟şüöžåйкл¤",
			limit:     5,
			want:      "H㐀〾▓朗",
			truncated: true,
		},
		{
			name:      "unicode truncate 10",
			str:       "H㐀〾▓朗퐭텟şüöžåйкл¤",
			limit:     10,
			want:      "H㐀〾▓朗퐭텟şüö",
			truncated: true,
		},
		{
			name:      "unicode fit",
			str:       "H㐀〾▓朗퐭텟şüöžåйкл¤",
			limit:     16,
			want:      "H㐀〾▓朗퐭텟şüöžåйкл¤",
			truncated: false,
		},
	}
	a := assert.New(t)
	for i := range tests {
		test := tests[i]
		t.Run(test.name, func(_ *testing.T) {
			got, truncated := TruncateString(test.str, test.limit)
			a.Equal(test.want, got)
			a.Equal(test.truncated, truncated)
		})
	}
}

func TestValidatePhone(t *testing.T) {
	tests := []struct {
		phone string
		want  bool
	}{
		{
			phone: "1234567890",
			want:  false,
		},
		{
			phone: "+8615655556666",
			want:  true,
		},
	}

	for _, test := range tests {
		got := ValidatePhone(test.phone)
		isValid := got == nil
		if isValid != test.want {
			t.Errorf("validatePhone %s, err %v", test.phone, got)
		}
	}
}

// TestSanitizeUTF8String pins the replacement shape: the invalid bytes survive
// as their hex escape so a corrupted value stays diagnosable after sanitizing.
func TestSanitizeUTF8String(t *testing.T) {
	// Vietnamese text encoded in Windows-1258 and stored in an AL32UTF8
	// database by a misconfigured client: 0xe1 is a valid lead byte with no
	// continuation.
	const corrupted = "Tr\xe1ng th\xe1i"
	require.False(t, utf8.ValidString(corrupted), "test precondition")

	sanitized := SanitizeUTF8String(corrupted)

	require.True(t, utf8.ValidString(sanitized))
	require.Contains(t, sanitized, "\\xe1")
	require.Equal(t, "\u6d4b\u8bd5", SanitizeUTF8String("\u6d4b\u8bd5"), "valid UTF-8 must pass through unchanged")
}

func TestRoundRows(t *testing.T) {
	require.Equal(t, int64(3), RoundRows(2.5))
	require.Equal(t, int64(1000), RoundRows(999.6))
	require.Equal(t, int64(math.MaxInt64), RoundRows(math.MaxInt64))
	require.Equal(t, int64(math.MaxInt64), RoundRows(1e30))
}

func TestAddRows(t *testing.T) {
	require.Equal(t, int64(5), AddRows(2, 3))
	require.Equal(t, int64(math.MaxInt64), AddRows(math.MaxInt64, 1))
	require.Equal(t, int64(math.MaxInt64), AddRows(math.MaxInt64-1, math.MaxInt64-1))
}
