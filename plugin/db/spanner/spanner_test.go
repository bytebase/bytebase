package spanner

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetDatabaseFromDSN(t *testing.T) {
	tests := []struct {
		dsn   string
		match bool
		want  string
	}{
		{
			dsn:   "projects/p/instances/i/databases/d",
			match: true,
			want:  "d",
		},
		{
			dsn:   "projects/p/instances/i/databases/",
			match: false,
			want:  "",
		},
	}
	a := require.New(t)
	for i := range tests {
		test := tests[i]
		got, err := getDatabaseFromDSN(test.dsn)
		if test.match {
			a.NoError(err)
			a.Equal(test.want, got)
		} else {
			a.Error(err)
		}
	}
}
