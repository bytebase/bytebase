package trino

import (
	"context"
	"errors"
	"testing"
	"time"

	omnidriver "github.com/bytebase/omni/driver"
	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/testing/protocmp"
	"google.golang.org/protobuf/types/known/timestamppb"

	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
)

type stubQuerySession struct {
	gotScript string
	gotOpts   omnidriver.QueryOptions
	results   []*omnidriver.Result
}

func (s *stubQuerySession) Query(_ context.Context, script string, opts omnidriver.QueryOptions) ([]*omnidriver.Result, error) {
	s.gotScript = script
	s.gotOpts = opts
	return s.results, nil
}

func (*stubQuerySession) Close() error { return nil }

func str(v string) *v1pb.RowValue {
	return &v1pb.RowValue{Kind: &v1pb.RowValue_StringValue{StringValue: v}}
}

// TestQueryResultRendersTrinoTypes pins how the adapter maps omni driver
// values, query options, and columnless zero-affected-row results into v1pb.
func TestQueryResultRendersTrinoTypes(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	ts := time.Date(2026, 1, 2, 12, 34, 56, 0, time.UTC)
	tsFrac := time.Date(2026, 1, 2, 12, 34, 56, 789000000, time.UTC)
	tsTZ := time.Date(2026, 1, 2, 12, 34, 56, 0, time.FixedZone("", 7200))

	tests := []struct {
		name    string
		colType string
		value   omnidriver.Value
		want    *v1pb.RowValue
	}{
		{"date", "DATE", omnidriver.Value{Kind: omnidriver.String, String: "2026-01-02"}, str("2026-01-02")},
		{"time", "TIME", omnidriver.Value{Kind: omnidriver.String, String: "12:34:56"}, str("12:34:56")},
		{"time with precision", "TIME", omnidriver.Value{Kind: omnidriver.String, String: "12:34:56.789"}, str("12:34:56.789")},
		{"zoned time", "TIME WITH TIME ZONE", omnidriver.Value{Kind: omnidriver.String, String: "12:34:56+02:00"}, str("12:34:56+02:00")},
		{"zoned time with precision", "TIME WITH TIME ZONE", omnidriver.Value{Kind: omnidriver.String, String: "12:34:56.789+02:00"}, str("12:34:56.789+02:00")},
		{"timestamp", "TIMESTAMP", omnidriver.Value{Kind: omnidriver.Timestamp, Time: ts}, &v1pb.RowValue{
			Kind: &v1pb.RowValue_TimestampValue{TimestampValue: &v1pb.RowValue_Timestamp{
				GoogleTimestamp: timestamppb.New(ts),
			}},
		}},
		{"timestamp with a fraction", "TIMESTAMP", omnidriver.Value{Kind: omnidriver.Timestamp, Time: tsFrac, Precision: 3}, &v1pb.RowValue{
			Kind: &v1pb.RowValue_TimestampValue{TimestampValue: &v1pb.RowValue_Timestamp{
				GoogleTimestamp: timestamppb.New(tsFrac),
				Accuracy:        3,
			}},
		}},
		{"zoned timestamp", "TIMESTAMP WITH TIME ZONE", omnidriver.Value{Kind: omnidriver.TimestampTZ, Time: tsTZ}, &v1pb.RowValue{
			Kind: &v1pb.RowValue_TimestampTzValue{TimestampTzValue: &v1pb.RowValue_TimestampTZ{
				GoogleTimestamp: timestamppb.New(time.Date(2026, 1, 2, 10, 34, 56, 0, time.UTC)),
				Offset:          7200,
			}},
		}},
		{"varchar", "VARCHAR", omnidriver.Value{Kind: omnidriver.String, String: "hi"}, str("hi")},
		{"bigint", "BIGINT", omnidriver.Value{Kind: omnidriver.Int64, Int64: 42}, &v1pb.RowValue{Kind: &v1pb.RowValue_Int64Value{Int64Value: 42}}},
		{"boolean", "BOOLEAN", omnidriver.Value{Kind: omnidriver.Bool, Bool: true}, &v1pb.RowValue{Kind: &v1pb.RowValue_BoolValue{BoolValue: true}}},
		{"decimal", "DECIMAL", omnidriver.Value{Kind: omnidriver.String, String: "1.25"}, str("1.25")},
		{"null date", "DATE", omnidriver.Value{Kind: omnidriver.Null}, util.NullRowValue},
		{"null varchar", "VARCHAR", omnidriver.Value{Kind: omnidriver.Null}, util.NullRowValue},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			sess := &stubQuerySession{
				results: []*omnidriver.Result{{
					Columns: []omnidriver.Column{{Name: "v", TypeName: tc.colType}},
					Rows:    [][]omnidriver.Value{{tc.value}},
				}},
			}
			driver := &Driver{}
			driver.SetSession(sess)

			results, err := driver.QueryConn(ctx, "SELECT v", db.QueryContext{
				Schema:               "analytics",
				Limit:                50,
				Explain:              true,
				Option:               &v1pb.QueryOption{ExplainFormat: v1pb.QueryOption_JSON},
				MaximumSQLResultSize: 1 << 20,
				MaskingEnabled:       true,
			})
			require.NoError(t, err)
			require.Equal(t, omnidriver.QueryOptions{
				Schema:         "analytics",
				Limit:          50,
				Explain:        true,
				ExplainFormat:  "JSON",
				MaxResultBytes: 1 << 20,
				MaskingEnabled: true,
			}, sess.gotOpts)
			require.Len(t, results, 1)
			require.Empty(t, results[0].Error)
			require.Equal(t, []string{tc.colType}, results[0].ColumnTypeNames)
			require.Len(t, results[0].Rows, 1)
			require.Empty(t, cmp.Diff(tc.want, results[0].Rows[0].Values[0], protocmp.Transform()))
		})
	}

	t.Run("fills zero affected rows only for columnless error-free results", func(t *testing.T) {
		t.Parallel()
		affected := int64(3)
		sess := &stubQuerySession{
			results: []*omnidriver.Result{
				{Statement: "SET SESSION query_max_run_time = '1m'"},
				{Statement: "INSERT INTO t VALUES (1), (2), (3)", AffectedRows: &affected},
				{Statement: "SELECT * FROM missing", Err: errors.New("table not found")},
			},
		}
		driver := &Driver{}
		driver.SetSession(sess)

		results, err := driver.QueryConn(ctx, "script", db.QueryContext{MaximumSQLResultSize: 1024})
		require.NoError(t, err)
		require.Len(t, results, 3)

		require.Equal(t, []string{"Affected Rows"}, results[0].ColumnNames)
		require.Equal(t, int64(1), results[0].RowsCount)
		require.Equal(t, int64(0), results[0].Rows[0].Values[0].GetInt64Value())

		require.Equal(t, []string{"Affected Rows"}, results[1].ColumnNames)
		require.Equal(t, int64(3), results[1].Rows[0].Values[0].GetInt64Value())

		require.Equal(t, "table not found", results[2].Error)
		require.Empty(t, results[2].Rows)
	})
}
