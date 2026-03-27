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

func TestTiDBVersionAtLeast(t *testing.T) {
	tests := []struct {
		version   string
		threshold string
		want      bool
	}{
		{version: "v7.1.1", threshold: "7.4.0", want: false},
		{version: "v7.2.0", threshold: "7.4.0", want: false},
		{version: "v7.3.0", threshold: "7.4.0", want: false},
		{version: "v7.4.0", threshold: "7.4.0", want: true},
		{version: "v7.5.2", threshold: "7.4.0", want: true},
		{version: "v8.0.0", threshold: "7.4.0", want: true},
		{version: "v8.5.0", threshold: "7.4.0", want: true},
	}

	a := require.New(t)
	for _, tc := range tests {
		got, err := tidbVersionAtLeast(tc.version, tc.threshold)
		a.NoError(err)
		a.Equal(tc.want, got, "version=%s threshold=%s", tc.version, tc.threshold)
	}
}
