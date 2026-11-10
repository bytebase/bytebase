package trino

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/require"
	"github.com/trinodb/trino-go-client/trino"
	"google.golang.org/protobuf/testing/protocmp"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/bytebase/bytebase/backend/common/testcontainer"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
)

func TestMakeValueByTypeNameHoldsAVariant(t *testing.T) {
	destination := makeValueByTypeName("VARIANT", nil)
	variant, ok := destination.(*trino.Variant)
	require.True(t, ok, "a VARIANT column needs a *trino.Variant destination, got %T", destination)

	// rows.Scan hands a VARIANT destination the trino.Variant the driver built,
	// which is why the default destination below cannot serve one.
	require.NoError(t, variant.Scan(trino.Variant{}))
	require.Error(t, new(sql.NullString).Scan(trino.Variant{}))
}

func TestConvertValueReadsAVariant(t *testing.T) {
	// A SQL NULL reaches Scan as nil and leaves the Variant invalid.
	//
	// This pins the NULL path only. convertValue's *trino.Variant branch is not
	// covered: a populated Variant cannot be built outside the driver, since its
	// fields are unexported, the only constructor decodes a server response, and
	// a hand-made Variant{Valid: true} panics in String. Removing that branch
	// still passes this test, so a VARIANT value rendering correctly needs a
	// Trino server to prove.
	variant := new(trino.Variant)
	require.NoError(t, variant.Scan(nil))
	require.False(t, variant.Valid)
	require.Equal(t, util.NullRowValue, convertValue("VARIANT", nil, variant))
}

func str(v string) *v1pb.RowValue {
	return &v1pb.RowValue{Kind: &v1pb.RowValue_StringValue{StringValue: v}}
}

// TestQueryResultRendersTrinoTypes pins how a real Trino's values reach the
// client, so a driver upgrade that changes a decoded Go type shows up as a
// failing expectation rather than as a silently reshaped result. The driver
// hands DATE and TIME a time.Time: before they had a destination of their own,
// Go rendered them through *sql.NullString as RFC3339Nano, so a date read back
// as "2026-01-02T00:00:00Z" and a time as "0000-01-01T12:34:56Z".
func TestQueryResultRendersTrinoTypes(t *testing.T) {
	ctx := context.Background()
	c := testcontainer.SharedTrinoContainer(t)
	db, err := sql.Open("trino", fmt.Sprintf(
		"http://%s@%s:%s?catalog=memory&schema=default",
		c.GetUsername(), c.GetHost(), c.GetPort()))
	require.NoError(t, err)
	defer db.Close()

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
		{"timestamp", "TIMESTAMP '2026-01-02 12:34:56'", "TIMESTAMP", &v1pb.RowValue{
			Kind: &v1pb.RowValue_TimestampValue{TimestampValue: &v1pb.RowValue_Timestamp{
				GoogleTimestamp: timestamppb.New(time.Date(2026, 1, 2, 12, 34, 56, 0, time.UTC)),
				Accuracy:        6,
			}},
		}},
		{"varchar", "CAST('hi' AS VARCHAR)", "VARCHAR", str("hi")},
		{"bigint", "CAST(42 AS BIGINT)", "BIGINT", &v1pb.RowValue{Kind: &v1pb.RowValue_Int64Value{Int64Value: 42}}},
		{"boolean", "true", "BOOLEAN", &v1pb.RowValue{Kind: &v1pb.RowValue_BoolValue{BoolValue: true}}},
		{"decimal", "CAST(1.25 AS DECIMAL(5,2))", "DECIMAL", &v1pb.RowValue{Kind: &v1pb.RowValue_DoubleValue{DoubleValue: 1.25}}},
		{"null date", "CAST(NULL AS DATE)", "DATE", util.NullRowValue},
		{"null varchar", "CAST(NULL AS VARCHAR)", "VARCHAR", util.NullRowValue},
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
