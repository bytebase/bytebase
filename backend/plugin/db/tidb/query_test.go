package tidb

import (
	"context"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/testing/protocmp"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/bytebase/bytebase/backend/common/testcontainer"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
)

func str(v string) *v1pb.RowValue {
	return &v1pb.RowValue{Kind: &v1pb.RowValue_StringValue{StringValue: v}}
}

func stamp(t time.Time, accuracy int32) *v1pb.RowValue {
	return &v1pb.RowValue{Kind: &v1pb.RowValue_TimestampValue{TimestampValue: &v1pb.RowValue_Timestamp{
		GoogleTimestamp: timestamppb.New(t),
		Accuracy:        accuracy,
	}}}
}

// TestQueryResultRendersTiDBTypes is TestQueryResultRendersMySQLTypes for TiDB,
// which shares MySQL's mapping. It is measured separately because sharing an
// implementation does not make the engine's behavior the same, and a fork that
// inherited a correct fact is still worth proving once.
func TestQueryResultRendersTiDBTypes(t *testing.T) {
	ctx := context.Background()
	db := testcontainer.SharedTiDBContainer(t).GetDB()

	noonMillis := time.Date(2026, 1, 2, 12, 34, 56, 789000000, time.UTC)

	tests := []struct {
		name     string
		expr     string
		wantType string
		want     *v1pb.RowValue
	}{
		{"date", "CAST('2026-01-02' AS DATE)", "DATE", str("2026-01-02")},
		{"time", "CAST('12:34:56' AS TIME)", "TIME", str("12:34:56")},
		{"time with three digits", "CAST('12:34:56.789' AS TIME(3))", "TIME", str("12:34:56.789")},
		{"time with six digits", "CAST('12:34:56.123456' AS TIME(6))", "TIME", str("12:34:56.123456")},
		{"datetime", "CAST('2026-01-02 12:34:56.789' AS DATETIME(3))", "DATETIME", stamp(noonMillis, 3)},
		{"timestamp names a datetime", "TIMESTAMP('2026-01-02 12:34:56')", "DATETIME", stamp(time.Date(2026, 1, 2, 12, 34, 56, 0, time.UTC), 0)},
		{"null date", "CAST(NULL AS DATE)", "DATE", util.NullRowValue},
		{"null time", "CAST(NULL AS TIME)", "TIME", util.NullRowValue},
		{"varchar", "'hi'", "VARCHAR", str("hi")},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rows, err := db.QueryContext(ctx, "SELECT "+tc.expr)
			require.NoError(t, err)
			defer rows.Close()

			result, err := util.RowsToQueryResult(rows, makeValueByTypeName, convertValue, 1<<20)
			require.NoError(t, err)
			require.NoError(t, rows.Err())
			require.Len(t, result.Rows, 1)
			require.Equal(t, tc.wantType, result.ColumnTypeNames[0])
			require.Empty(t, cmp.Diff(tc.want, result.Rows[0].Values[0], protocmp.Transform()))
		})
	}
}
