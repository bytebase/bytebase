package util

import (
	"testing"

	"github.com/stretchr/testify/require"

	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
)

func TestPrefixExplain(t *testing.T) {
	explain := PrefixExplain(v1pb.QueryOption_TEXT)
	for _, statement := range []string{"SELECT 1", "SELECT 'EXPLAIN'", "SELECT explained FROM t"} {
		got, err := explain.Statement(statement, v1pb.QueryOption_EXPLAIN_FORMAT_UNSPECIFIED)
		require.NoError(t, err, statement)
		require.Equal(t, "EXPLAIN "+statement, got)
	}
	for _, statement := range []string{
		"EXPLAIN SELECT 1",
		"explain analyze select 1",
		"/* plan */ explain\nSELECT 1",
		"-- plan\nEXPLAIN FORMAT=JSON SELECT 1",
	} {
		_, err := explain.Statement(statement, v1pb.QueryOption_EXPLAIN_FORMAT_UNSPECIFIED)
		require.Error(t, err, statement)
	}
}
