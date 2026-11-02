package mssql

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

// TestQueryResultRendersMSSQLTypes pins how a real SQL Server's values reach the
// client, so a driver upgrade that changes a decoded Go type shows up as a
// failing expectation rather than as a silently reshaped result. TIME used to be
// formatted with time.TimeOnly, so a TIME(3) of 12:34:56.789 read back as
// 12:34:56 with the fraction gone.
//
// The expected fractions are the ones PostgreSQL produces for the same values:
// none for a whole second, and at least six digits otherwise.
func TestQueryResultRendersMSSQLTypes(t *testing.T) {
	ctx := context.Background()
	db := testcontainer.SharedMSSQLContainer(t).GetDB()

	noon := time.Date(2026, 1, 2, 12, 34, 56, 0, time.UTC)
	noonMillis := time.Date(2026, 1, 2, 12, 34, 56, 789000000, time.UTC)

	tests := []struct {
		name     string
		expr     string
		wantType string
		want     *v1pb.RowValue
	}{
		{"date", "CAST('2026-01-02' AS DATE)", "DATE", str("2026-01-02")},
		{"time", "CAST('12:34:56' AS TIME(0))", "TIME", str("12:34:56")},
		{"time with three digits", "CAST('12:34:56.789' AS TIME(3))", "TIME", str("12:34:56.789000")},
		{"time with trailing zeros", "CAST('12:34:56.700' AS TIME(3))", "TIME", str("12:34:56.700000")},
		{"time whose scale outruns its value", "CAST('12:34:56' AS TIME(3))", "TIME", str("12:34:56")},
		{"time with seven digits", "CAST('12:34:56.1234567' AS TIME(7))", "TIME", str("12:34:56.1234567")},
		{"datetime2", "CAST('2026-01-02 12:34:56.789' AS DATETIME2(3))", "DATETIME2", stamp(noonMillis, 3)},
		{"datetime", "CAST('2026-01-02 12:34:56' AS DATETIME)", "DATETIME", stamp(noon, 0)},
		{"smalldatetime", "CAST('2026-01-02 12:34:00' AS SMALLDATETIME)", "SMALLDATETIME", stamp(time.Date(2026, 1, 2, 12, 34, 0, 0, time.UTC), 0)},
		{"datetimeoffset", "CAST('2026-01-02 12:34:56.789 +02:00' AS DATETIMEOFFSET(3))", "DATETIMEOFFSET", &v1pb.RowValue{
			Kind: &v1pb.RowValue_TimestampTzValue{TimestampTzValue: &v1pb.RowValue_TimestampTZ{
				GoogleTimestamp: timestamppb.New(time.Date(2026, 1, 2, 10, 34, 56, 789000000, time.UTC)),
				Offset:          2 * 60 * 60,
				Accuracy:        3,
			}},
		}},
		{"null date", "CAST(NULL AS DATE)", "DATE", util.NullRowValue},
		{"null time", "CAST(NULL AS TIME(3))", "TIME", util.NullRowValue},
		{"varchar", "CAST('hi' AS VARCHAR(10))", "VARCHAR", str("hi")},
		{"int", "CAST(42 AS INT)", "INT", &v1pb.RowValue{Kind: &v1pb.RowValue_Int64Value{Int64Value: 42}}},
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
