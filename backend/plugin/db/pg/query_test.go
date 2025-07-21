package pg

import (
	"testing"

	"github.com/stretchr/testify/require"
)

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
			expected: "12:34-05:56.123",
		},
	}

	for _, test := range tests {
		got := padZeroes(test.rawStr, test.acc)
		require.Equal(t, test.expected, got)
	}
}
