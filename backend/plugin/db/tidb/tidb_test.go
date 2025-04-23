package tidb

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseVersion(t *testing.T) {
	tests := []struct {
		version string
		want    string
	}{
		{
			version: "8.0.11-TiDB-v8.5.0",
			want:    "v8.5.0",
		},
		{
			version: "8.0.11-TiDB-v7.5.2-serverless",
			want:    "v7.5.2",
		},
	}

	a := require.New(t)
	for _, tc := range tests {
		version, err := parseVersion(tc.version)
		a.NoError(err)
		a.Equal(tc.want, version)
	}
}

func TestIsNonTransactionStatement(t *testing.T) {
	tests := []struct {
		stmt string
		want bool
	}{
		{
			`CREATE DATABASE "hello" ENCODING "UTF8";`,
			false,
		},
		{
			`CREATE table hello(id integer);`,
			false,
		},
		{
			`CREATE INDEX c1 ON t1 (c1);`,
			true,
		},
		{
			`CREATE UNIQUE INDEX c1 ON t1 (c1);`,
			true,
		},
		{
			`CREATE UNIQUE INDEX c1 ON t1 (c1) INVISIBLE;`,
			true,
		},
	}

	for _, test := range tests {
		got := isNonTransactionStatement(test.stmt)
		require.Equal(t, test.want, got)
	}
}
