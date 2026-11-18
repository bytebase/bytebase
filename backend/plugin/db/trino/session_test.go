package trino

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/common/testcontainer"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/db"
)

// TestQueryConnKeepsTheSessionForTheDriver pins that statements of one request
// share a session even though the API calls QueryConn once per statement: a SET
// SESSION in the first call is visible to the SHOW SESSION in the second.
func TestQueryConnKeepsTheSessionForTheDriver(t *testing.T) {
	ctx := context.Background()
	c := testcontainer.SharedTrinoContainer(t)
	driver := openTestDriver(t, &storepb.DataSource{
		Host:     c.GetHost(),
		Port:     c.GetPort(),
		Username: c.GetUsername(),
		Database: "memory",
	})
	queryContext := db.QueryContext{MaximumSQLResultSize: 1 << 20}

	results, err := driver.QueryConn(ctx, nil, "SET SESSION query_max_run_time = '1m'", queryContext)
	require.NoError(t, err)
	require.Len(t, results, 1)
	require.Empty(t, results[0].Error)

	results, err = driver.QueryConn(ctx, nil, "SHOW SESSION LIKE 'query_max_run_time'", queryContext)
	require.NoError(t, err)
	require.Len(t, results, 1)
	require.Empty(t, results[0].Error)
	require.Len(t, results[0].Rows, 1)
	require.Equal(t, "1m", results[0].Rows[0].Values[1].GetStringValue())
}
