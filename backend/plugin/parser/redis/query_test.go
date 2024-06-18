package redis

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidateQuery(t *testing.T) {
	tests := []struct {
		statement string
		validate  bool
		err       bool
	}{
		{
			statement: "get hello",
			validate:  true,
		},
		{
			statement: "set hello 1",
			validate:  false,
		},
	}

	for _, test := range tests {
		got, err := validateQuery(test.statement)
		if test.err {
			require.Error(t, err)
		} else {
			require.NoError(t, err)
			require.Equal(t, test.validate, got)
		}
	}
}
