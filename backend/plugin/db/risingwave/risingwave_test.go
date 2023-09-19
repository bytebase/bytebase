package risingwave

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetDatabaseInCreateDatabaseStatement(t *testing.T) {
	tests := []struct {
		createDatabaseStatement string
		want                    string
		wantErr                 bool
	}{
		{
			`CREATE DATABASE "hello" ENCODING "UTF8";`,
			"hello",
			false,
		},
		{
			`CREATE DATABASE "hello";`,
			"hello",
			false,
		},
		{
			`CREATE DATABASE hello;`,
			"hello",
			false,
		},
		{
			`CREATE DATABASE hello ENCODING "UTF8";`,
			"hello",
			false,
		},
		{
			`CREATE DATABASE;`,
			"",
			true,
		},
	}

	for _, test := range tests {
		got, err := getDatabaseInCreateDatabaseStatement(test.createDatabaseStatement)
		if test.wantErr {
			require.Error(t, err)
		} else {
			require.NoError(t, err)
		}
		require.Equal(t, test.want, got)
	}
}
