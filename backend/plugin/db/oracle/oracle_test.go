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
			Banner: "Oracle Database 12c Enterprise Edition Release 12.1.0.2.0 - 64bit Production",
			First:  12,
			Second: 1,
		},
		{
			Banner: "Oracle Database 12.1.0.",
			First:  12,
			Second: 1,
		},
	}

	for _, test := range tests {
		first, second, err := parseVersion(test.Banner)
		require.NoError(t, err)
		require.Equal(t, test.First, first)
		require.Equal(t, test.Second, second)
	}
}
