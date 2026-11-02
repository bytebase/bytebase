package clickhouse

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
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

func TestTranslateAggregateFunctionError(t *testing.T) {
	t.Run("nil", func(t *testing.T) {
		require.NoError(t, translateAggregateFunctionError(nil))
	})

	t.Run("non aggregate function error", func(t *testing.T) {
		err := errors.New("some other clickhouse error")
		require.Same(t, err, translateAggregateFunctionError(err))
	})

	t.Run("simple aggregate function error is not rewritten", func(t *testing.T) {
		// Defensive: SimpleAggregateFunction is supported by the driver, but if a
		// future driver bug ever surfaces an "unsupported column type" error for
		// it, our marker must not match (because SimpleAggregateFunction errors
		// do not need the -Merge / finalizeAggregation guidance).
		err := errors.New(`clickhouse: unsupported column type "SimpleAggregateFunction(sum, UInt64)"`)
		require.Same(t, err, translateAggregateFunctionError(err))
	})

	t.Run("aggregate function error", func(t *testing.T) {
		err := errors.New(`read data: failed to decode block from 127.0.0.1:9000 (conn_id=26, compression=none): clickhouse: unsupported column type "AggregateFunction(argMin, Decimal(20, 6), DateTime64(3, 'UTC'))"`)
		got := translateAggregateFunctionError(err)
		require.Error(t, got)
		require.Contains(t, got.Error(), "cannot be decoded directly")
		require.Contains(t, got.Error(), "finalizeAggregation(col)")
		require.Contains(t, got.Error(), err.Error())
	})
}

func TestIsClickHouseDate(t *testing.T) {
	for typeName, want := range map[string]bool{
		"DATE":                     true,
		"DATE32":                   true,
		"NULLABLE(DATE)":           true,
		"NULLABLE(DATE32)":         true,
		"DATETIME":                 false,
		"DATETIME64(3)":            false,
		"DATETIME('EUROPE/PARIS')": false,
		"NULLABLE(DATETIME64(3))":  false,
	} {
		require.Equal(t, want, isClickHouseDate(typeName), typeName)
	}
}

func str(v string) *v1pb.RowValue {
	return &v1pb.RowValue{Kind: &v1pb.RowValue_StringValue{StringValue: v}}
}

func stamp(t time.Time, accuracy int32) *v1pb.RowValue {
	return &v1pb.RowValue{Kind: &v1pb.RowValue_TimestampValue{TimestampValue: &v1pb.RowValue_Timestamp{
		GoogleTimestamp: timestamppb.New(t),
		Accuracy:        accuracy,
	}}}
}

// TestQueryResultRendersClickHouseTypes pins how a real ClickHouse's values
// reach the client, so a driver upgrade that changes a decoded Go type shows up
// as a failing expectation rather than as a silently reshaped result. Every
// time.Time used to be formatted as RFC3339Nano, so a Date read back as
// "2026-01-02T00:00:00Z" and a DateTime as "2026-01-02T12:34:56Z".
//
// A zoned DateTime keeps the wall clock the server prints: the expectation is
// 12:34:56 and not the 11:34:56 the instant would be in UTC.
func TestQueryResultRendersClickHouseTypes(t *testing.T) {
	ctx := context.Background()
	c := testcontainer.SharedClickHouseContainer(t)
	db, err := sql.Open("clickhouse", fmt.Sprintf("clickhouse://%s:%s@%s:%s/default",
		c.GetUsername(), c.GetPassword(), c.GetHost(), c.GetPort()))
	require.NoError(t, err)
	defer db.Close()

	noon := time.Date(2026, 1, 2, 12, 34, 56, 0, time.UTC)
	noonMillis := time.Date(2026, 1, 2, 12, 34, 56, 789000000, time.UTC)

	tests := []struct {
		name     string
		expr     string
		wantType string
		want     *v1pb.RowValue
	}{
		{"date", "toDate('2026-01-02')", "DATE", str("2026-01-02")},
		{"date32", "toDate32('2026-01-02')", "DATE32", str("2026-01-02")},
		{"nullable date", "CAST(toDate('2026-01-02') AS Nullable(Date))", "NULLABLE(DATE)", str("2026-01-02")},
		{"datetime", "toDateTime('2026-01-02 12:34:56')", "DATETIME", stamp(noon, 0)},
		{"datetime with three digits", "toDateTime64('2026-01-02 12:34:56.789', 3)", "DATETIME64(3)", stamp(noonMillis, 3)},
		{"datetime with six digits", "toDateTime64('2026-01-02 12:34:56.789', 6)", "DATETIME64(6)", stamp(noonMillis, 6)},
		{"zoned datetime", "toDateTime('2026-01-02 12:34:56', 'Europe/Paris')", "DATETIME('EUROPE/PARIS')", stamp(noon, 0)},
		{"nullable datetime", "CAST(toDateTime64('2026-01-02 12:34:56.789', 3) AS Nullable(DateTime64(3)))", "NULLABLE(DATETIME64(3))", stamp(noonMillis, 3)},
		{"timestamp names a datetime", "CAST('2026-01-02 12:34:56' AS TIMESTAMP)", "DATETIME", stamp(noon, 0)},
		// ClickHouse has no time-of-day type: TIME names an Int64, which is why
		// convertValue has no TIME case to render.
		{"time names an int64", "CAST(45296 AS TIME)", "INT64", &v1pb.RowValue{Kind: &v1pb.RowValue_Int64Value{Int64Value: 45296}}},
		{"null date", "CAST(NULL AS Nullable(Date))", "NULLABLE(DATE)", util.NullRowValue},
		{"null datetime", "CAST(NULL AS Nullable(DateTime))", "NULLABLE(DATETIME)", util.NullRowValue},
		{"string", "'hi'", "STRING", str("hi")},
		{"int64", "toInt64(42)", "INT64", &v1pb.RowValue{Kind: &v1pb.RowValue_Int64Value{Int64Value: 42}}},
		{"bool", "true", "BOOL", &v1pb.RowValue{Kind: &v1pb.RowValue_BoolValue{BoolValue: true}}},
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
