package pg

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIsAtLeastPG10(t *testing.T) {
	tests := []struct {
		version string
		want    bool
	}{
		{
			"hello",
			true,
		},
		{
			"9.6.24",
			false,
		},
		{
			"9.624",
			false,
		},
		{
			"10.0.0",
			true,
		},
		{
			"16.0.0",
			true,
		},
		{
			"16.14",
			true,
		},
	}

	for _, test := range tests {
		got := isAtLeastPG10(test.version)
		require.Equal(t, test.want, got)
	}
}
