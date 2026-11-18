package trino

import (
	"context"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/testing/protocmp"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/bytebase/bytebase/backend/common/testcontainer"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
)

func str(v string) *v1pb.RowValue {
	return &v1pb.RowValue{Kind: &v1pb.RowValue_StringValue{StringValue: v}}
}

func openTestDriver(t *testing.T, dataSource *storepb.DataSource) db.Driver {
	t.Helper()
	driver, err := (&Driver{}).Open(context.Background(), storepb.Engine_TRINO, db.ConnectionConfig{
		DataSource: dataSource,
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = driver.Close(context.Background()) })
	return driver
}

// TestQueryResultRendersTrinoTypes pins how a real Trino's values reach the
// client, so a driver upgrade that changes a decoded Go type shows up as a
// failing expectation rather than as a silently reshaped result.
func TestQueryResultRendersTrinoTypes(t *testing.T) {
	ctx := context.Background()
	c := testcontainer.SharedTrinoContainer(t)
	driver := openTestDriver(t, &storepb.DataSource{
		Host:     c.GetHost(),
		Port:     c.GetPort(),
		Username: c.GetUsername(),
		Database: "memory",
	})

	tests := []struct {
		name     string
		expr     string
		wantType string
		want     *v1pb.RowValue
	}{
		{"date", "DATE '2026-01-02'", "DATE", str("2026-01-02")},
		{"time", "TIME '12:34:56'", "TIME", str("12:34:56")},
		{"time with precision", "CAST('12:34:56.789' AS TIME(3))", "TIME", str("12:34:56.789")},
		{"zoned time", "TIME '12:34:56 +02:00'", "TIME WITH TIME ZONE", str("12:34:56+02:00")},
		{"zoned time with precision", "TIME '12:34:56.789 +02:00'", "TIME WITH TIME ZONE", str("12:34:56.789+02:00")},
		// The contract reports the value's own fractional digits as Accuracy;
		// Bytebase's own conversion reported 6 for every TIMESTAMP.
		{"timestamp", "TIMESTAMP '2026-01-02 12:34:56'", "TIMESTAMP", &v1pb.RowValue{
			Kind: &v1pb.RowValue_TimestampValue{TimestampValue: &v1pb.RowValue_Timestamp{
				GoogleTimestamp: timestamppb.New(time.Date(2026, 1, 2, 12, 34, 56, 0, time.UTC)),
			}},
		}},
		{"timestamp with a fraction", "TIMESTAMP '2026-01-02 12:34:56.789'", "TIMESTAMP", &v1pb.RowValue{
			Kind: &v1pb.RowValue_TimestampValue{TimestampValue: &v1pb.RowValue_Timestamp{
				GoogleTimestamp: timestamppb.New(time.Date(2026, 1, 2, 12, 34, 56, 789000000, time.UTC)),
				Accuracy:        3,
			}},
		}},
		// A zoned timestamp keeps its zone: the contract carries it as a
		// TimestampTZ, where Bytebase's own conversion sent every TIMESTAMP
		// through the zone-agnostic Timestamp.
		{"zoned timestamp", "TIMESTAMP '2026-01-02 12:34:56 +02:00'", "TIMESTAMP WITH TIME ZONE", &v1pb.RowValue{
			Kind: &v1pb.RowValue_TimestampTzValue{TimestampTzValue: &v1pb.RowValue_TimestampTZ{
				GoogleTimestamp: timestamppb.New(time.Date(2026, 1, 2, 10, 34, 56, 0, time.UTC)),
				Offset:          7200,
			}},
		}},
		{"varchar", "CAST('hi' AS VARCHAR)", "VARCHAR", str("hi")},
		{"bigint", "CAST(42 AS BIGINT)", "BIGINT", &v1pb.RowValue{Kind: &v1pb.RowValue_Int64Value{Int64Value: 42}}},
		{"boolean", "true", "BOOLEAN", &v1pb.RowValue{Kind: &v1pb.RowValue_BoolValue{BoolValue: true}}},
		// The contract keeps a DECIMAL's exact text, because a float64 cannot
		// hold every decimal Trino returns; Bytebase sent a double.
		{"decimal", "CAST(1.25 AS DECIMAL(5,2))", "DECIMAL", str("1.25")},
		{"null date", "CAST(NULL AS DATE)", "DATE", util.NullRowValue},
		{"null varchar", "CAST(NULL AS VARCHAR)", "VARCHAR", util.NullRowValue},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			results, err := driver.QueryConn(ctx, nil, "SELECT "+tc.expr, db.QueryContext{
				MaximumSQLResultSize: 1 << 20,
			})
			require.NoError(t, err)
			require.Len(t, results, 1)
			result := results[0]
			require.Empty(t, result.Error)
			require.Len(t, result.Rows, 1)
			require.Equal(t, tc.wantType, result.ColumnTypeNames[0])
			require.Empty(t, cmp.Diff(tc.want, result.Rows[0].Values[0], protocmp.Transform()))
		})
	}
}
