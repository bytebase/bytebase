package mssql

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/common/testcontainer"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/plugin/db"
)

func TestShowplanStatistic(t *testing.T) {
	for _, tc := range []struct {
		name   string
		option *v1pb.QueryOption
		want   string
	}{
		{name: "no option", option: nil, want: "SHOWPLAN_ALL"},
		{name: "empty option", option: &v1pb.QueryOption{}, want: "SHOWPLAN_ALL"},
		{
			name:   "text",
			option: &v1pb.QueryOption{ExplainFormat: v1pb.QueryOption_TEXT},
			want:   "SHOWPLAN_ALL",
		},
		{
			name:   "xml",
			option: &v1pb.QueryOption{ExplainFormat: v1pb.QueryOption_XML},
			want:   "SHOWPLAN_XML",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, showplanStatistic(tc.option))
		})
	}
}

func TestQueryConnRecordsLimitedStatement(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	driver := newSyncTestDatabase(ctx, t, testcontainer.SharedMSSQLContainer(t))
	executeBatches(ctx, t, driver, "CREATE TABLE dbo.t (id INT PRIMARY KEY);\nINSERT INTO dbo.t VALUES (1), (2), (3);")

	conn, err := driver.GetDB().Conn(ctx)
	require.NoError(t, err)
	defer conn.Close()

	// SET returns nothing, so the driver pads an empty result in for it.
	results, err := driver.QueryConn(ctx, conn, "SET NOCOUNT OFF;\nSELECT TOP 3 id FROM dbo.t ORDER BY id;", db.QueryContext{
		Limit:                2,
		MaximumSQLResultSize: 1 << 30,
	})
	require.NoError(t, err)
	require.Len(t, results, 2)
	require.Equal(t, "SET NOCOUNT OFF;", results[0].GetStatement())
	require.Equal(t, "\nSELECT TOP 2 id FROM dbo.t ORDER BY id;", results[1].GetStatement())
	require.Len(t, results[1].GetRows(), 2)
}
