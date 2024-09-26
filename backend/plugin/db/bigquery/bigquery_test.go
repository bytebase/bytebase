package bigquery

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEncodeOperatorEmail(t *testing.T) {
	tests := []struct {
		email string
		want  string
	}{
		{
			"hello@bb.com",
			"hello_bb_com",
		},
		{
			"Hello123@BB.com",
			"hello123_bb_com",
		},
		{
			"0123456789012345678901234567890123456789012345678901234567890123456789",
			"012345678901234567890123456789012345678901234567890123456789012",
		},
		{
			"012345678901234567890123456789012345678901234567890123456789012",
			"012345678901234567890123456789012345678901234567890123456789012",
		},
	}

	for _, test := range tests {
		got := encodeOperatorEmail(test.email)
		require.Equal(t, test.want, got)
	}
}
