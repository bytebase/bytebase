package oracle

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/plugin/parser/plsql"
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
		v, err := plsql.ParseVersion(test.Banner)
		require.NoError(t, err)
		require.Equal(t, test.First, v.First)
		require.Equal(t, test.Second, v.Second)
	}
}
