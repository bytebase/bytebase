package oracle

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseVersion(t *testing.T) {
	type testData struct {
		Banner string
		First  int
		Second int
	}
	tests := []testData{
		{
			Banner: "12.1.0.2.0",
			First:  12,
			Second: 1,
		},
		{
			Banner: "12.1.0.",
			First:  12,
			Second: 1,
		},
	}

	for _, test := range tests {
		v, err := parseVersion(test.Banner)
		require.NoError(t, err)
		require.Equal(t, test.First, v.first)
		require.Equal(t, test.Second, v.second)
	}
}
