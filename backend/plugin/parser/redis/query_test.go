package redis

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidateQuery(t *testing.T) {
	tests := []struct {
		statement string
		valid     bool
		allQuery  bool
		err       bool
	}{
		{
			statement: "get hello",
			valid:     true,
			allQuery:  true,
		},
		{
			statement: "set hello 1",
			valid:     false,
			allQuery:  false,
		},
		{
			statement: "memory usage hello",
			valid:     true,
			allQuery:  true,
		},
	}

	for _, test := range tests {
		gotValid, gotAllQuery, err := validateQuery(test.statement)
		if test.err {
			require.Error(t, err)
		} else {
			require.NoError(t, err)
			require.Equal(t, test.valid, gotValid)
			require.Equal(t, test.allQuery, gotAllQuery)
		}
	}
}
