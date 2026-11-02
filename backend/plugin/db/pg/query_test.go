package pg

import (
	"context"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/testing/protocmp"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/bytebase/bytebase/backend/common/testcontainer"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
)

func TestBuildTimestamptzRowValue(t *testing.T) {
	tests := []struct {
		name       string
		typeName   string
		input      time.Time
		wantString string // non-empty means we expect a string fallback with this value
	}{
		{
			name:     "valid timestamp",
			typeName: "TIMESTAMP",
			input:    time.Date(2026, 5, 11, 12, 34, 56, 0, time.UTC),
		},
		{
			name:     "valid timestamptz",
			typeName: "TIMESTAMPTZ",
			input:    time.Date(2026, 5, 11, 12, 34, 56, 0, time.FixedZone("", 2*60*60)),
		},
		{
			name:       "BC timestamp falls back to PG-formatted string",
			typeName:   "TIMESTAMP",
			input:      time.Date(0, 12, 31, 15, 0, 0, 0, time.UTC), // Go year 0 == 1 BC
			wantString: "0001-12-31 15:00:00 BC",
		},
		{
			name:       "year 10000 timestamp omits zone",
			typeName:   "TIMESTAMP",
			input:      time.Date(10000, 1, 1, 0, 0, 0, 0, time.UTC),
			wantString: "10000-01-01 00:00:00",
		},
		{
			name:       "year 10000 timestamptz keeps offset",
			typeName:   "TIMESTAMPTZ",
			input:      time.Date(10000, 5, 11, 12, 34, 56, 0, time.FixedZone("", 2*60*60)),
			wantString: "10000-05-11 12:34:56+02:00",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := buildTimestamptzRowValue(tc.typeName, tc.input, 6)

			// The whole point of the fix: marshaling must never fail.
			_, err := protojson.Marshal(got)
			require.NoError(t, err)

			if tc.wantString != "" {
				require.Equal(t, tc.wantString, got.GetStringValue())
				return
			}
			if tc.typeName == "TIMESTAMP" {
				require.NotNil(t, got.GetTimestampValue())
			} else {
				require.NotNil(t, got.GetTimestampTzValue())
			}
		})
	}
}

func TestPadZeroes(t *testing.T) {
	tests := []struct {
		name     string
		rawStr   string
		acc      int
		expected string
	}{
		{
			name:     "No decimal part",
			rawStr:   "12:34:56",
			acc:      6,
			expected: "12:34:56",
		},
		{
			name:     "Already correct precision",
			rawStr:   "12:34:56.123456",
			acc:      6,
			expected: "12:34:56.123456",
		},
		{
			name:     "Less precision",
			rawStr:   "12:34:56.123",
			acc:      6,
			expected: "12:34:56.123000",
		},
		{
			name:     "More precision",
			rawStr:   "12:34:56.123456789",
			acc:      6,
			expected: "12:34:56.123456789",
		},
		{
			name:     "With timezone",
			rawStr:   "12:34:56.123+02:00",
			acc:      6,
			expected: "12:34:56.123000+02:00",
		},
		{
			name:     "With negative timezone",
			rawStr:   "12:34:56.123-02:00",
			acc:      6,
			expected: "12:34:56.123000-02:00",
		},
		{
			name:     "Invalid format: timezone before decimal (edge case fix)",
			rawStr:   "12:34-05:56.123",
			acc:      6,
			expected: "12:34-05:56.123000",
		},
		{
			name:     "Negative interval: leading minus should be ignored",
			rawStr:   "-00:04:37.530865",
			acc:      6,
			expected: "-00:04:37.530865",
		},
		{
			name:     "Negative interval with short precision",
			rawStr:   "-00:02:45.25",
			acc:      6,
			expected: "-00:02:45.250000",
		},
		{
			name:     "Negative interval without decimal",
			rawStr:   "-00:05:00",
			acc:      6,
			expected: "-00:05:00",
		},
		{
			name:     "Negative interval with timezone (minus after decimal)",
			rawStr:   "-12:34:56.123-05:00",
			acc:      6,
			expected: "-12:34:56.123-05:00",
		},
	}

	for _, test := range tests {
		got := padZeroes(test.rawStr, test.acc)
		require.Equal(t, test.expected, got)
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

// TestQueryResultRendersPGTypes pins how a real PostgreSQL renders a DATE, a
// TIME and a TIMESTAMP. This is the standard the other drivers are held to, so
// it is pinned here rather than only described: a date is date-only, a TIME is
// time-only and keeps its fraction, and a TIMESTAMP is a timestamp_value, never
// an RFC3339 string with a T and a Z.
func TestQueryResultRendersPGTypes(t *testing.T) {
	ctx := context.Background()
	_, db := testcontainer.NewPgDatabase(t)

	noonMillis := time.Date(2026, 1, 2, 12, 34, 56, 789000000, time.UTC)

	tests := []struct {
		name     string
		expr     string
		wantType string
		want     *v1pb.RowValue
	}{
		{"date", "DATE '2026-01-02'", "DATE", str("2026-01-02")},
		{"time", "TIME '12:34:56'", "TIME", str("12:34:56")},
		{"time with three digits", "TIME '12:34:56.789'", "TIME", str("12:34:56.789000")},
		{"time with trailing zeros", "CAST('12:34:56.700' AS TIME(3))", "TIME", str("12:34:56.700000")},
		{"time whose scale outruns its value", "CAST('12:34:56' AS TIME(3))", "TIME", str("12:34:56")},
		{"timestamp", "TIMESTAMP '2026-01-02 12:34:56.789'", "TIMESTAMP", stamp(noonMillis, 6)},
		{"timestamptz", "TIMESTAMPTZ '2026-01-02 12:34:56.789+02:00'", "TIMESTAMPTZ", &v1pb.RowValue{
			Kind: &v1pb.RowValue_TimestampTzValue{TimestampTzValue: &v1pb.RowValue_TimestampTZ{
				GoogleTimestamp: timestamppb.New(time.Date(2026, 1, 2, 10, 34, 56, 789000000, time.UTC)),
				Zone:            "UTC",
				Accuracy:        6,
			}},
		}},
		{"null date", "CAST(NULL AS DATE)", "DATE", util.NullRowValue},
		{"null time", "CAST(NULL AS TIME)", "TIME", util.NullRowValue},
		{"text", "'hi'::text", "TEXT", str("hi")},
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
