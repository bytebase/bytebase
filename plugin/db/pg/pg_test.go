package pg

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestHasDatabaseInStatement(t *testing.T) {
	tests := []struct {
		databases               []*pgDatabaseSchema
		createDatabaseStatement string
		want                    bool
		wantErr                 bool
	}{
		{
			[]*pgDatabaseSchema{
				{name: "hello"},
			},
			`CREATE DATABASE "h1" ENCODING "UTF8";`,
			false,
			false,
		},
		{
			[]*pgDatabaseSchema{
				{name: "hello"},
			},
			`CREATE DATABASE "hello" ENCODING "UTF8";`,
			true,
			false,
		},
		{
			[]*pgDatabaseSchema{
				{name: "hello"},
			},
			`CREATE DATABASE "hello";`,
			true,
			false,
		},
		{
			[]*pgDatabaseSchema{
				{name: "hello"},
			},
			`CREATE DATABASE hello;`,
			true,
			false,
		},
		{
			[]*pgDatabaseSchema{
				{name: "hello"},
			},
			`CREATE DATABASE hello ENCODING "UTF8";`,
			true,
			false,
		},
		{
			[]*pgDatabaseSchema{
				{name: "hello"},
			},
			`CREATE DATABASE;`,
			false,
			true,
		},
	}

	for _, test := range tests {
		got, err := hasDatabaseInStatement(test.databases, test.createDatabaseStatement)
		if test.wantErr {
			require.Error(t, err)
		} else {
			require.NoError(t, err)
		}
		require.Equal(t, test.want, got)
	}
}
