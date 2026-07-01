//nolint:revive
package util

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIsAWSDSQLHost(t *testing.T) {
	testCases := []struct {
		host string
		want bool
	}{
		{host: "abcdefgh12345678.dsql.us-east-1.on.aws", want: true},
		{host: "cluster.dsql.ap-northeast-1.on.aws", want: true},
		{host: "mydb.123456789012.us-east-1.rds.amazonaws.com", want: false},
		{host: "localhost", want: false},
		{host: "10.0.0.1", want: false},
		{host: "example.dsql.com", want: false},
		{host: "", want: false},
	}
	for _, tc := range testCases {
		require.Equal(t, tc.want, IsAWSDSQLHost(tc.host), "host=%q", tc.host)
	}
}
