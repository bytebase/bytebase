package cockroachdb

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

func TestGetRoutingIDFromCockroachCloudURL(t *testing.T) {
	tests := []struct {
		host     string
		expected string
	}{
		{
			host:     "routing-id.cockroachlabs.cloud",
			expected: "routing-id",
		},
		{
			host:     "bytebase-cluster-7749.6xw.aws-ap-southeast-1.cockroachlabs.cloud",
			expected: "bytebase-cluster-7749",
		},
		{
			host:     "subdomain.routing-id.cockroachlabs.cloud",
			expected: "subdomain",
		},
		{
			host:     "subdomain.routing-id.cockroachlabs.cloud",
			expected: "subdomain",
		},
		{
			host:     "cockroachlabs.cloud",
			expected: "",
		},
		{
			host:     "example.com",
			expected: "",
		},
	}

	for _, test := range tests {
		got := getRoutingIDFromCockroachCloudURL(test.host)
		require.Equal(t, test.expected, got, "host: %s", test.host)
	}
}
