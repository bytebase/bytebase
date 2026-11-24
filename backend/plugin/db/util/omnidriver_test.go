//nolint:revive
package util

import (
	"errors"
	"testing"
	"time"

	omnidriver "github.com/bytebase/omni/driver"
	metadatapb "github.com/bytebase/omni/metadata"
	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func TestOmniResultsToQueryResultsWith(t *testing.T) {
	require.Nil(t, OmniResultsToQueryResultsWith(nil, 1024, QueryResultOptions{}))

	results := []*omnidriver.Result{
		{
			Statement: "SELECT 1",
			Columns:   []omnidriver.Column{{Name: "col", TypeName: "INT"}},
			Rows:      [][]omnidriver.Value{{{Kind: omnidriver.Int64, Int64: 1}}},
			Plan:      &omnidriver.Plan{Format: "JSON", Executed: true, Content: "{}"},
		},
	}

	withPlan := OmniResultsToQueryResultsWith(results, 1024, QueryResultOptions{})
	require.Len(t, withPlan, 1)
	require.NotNil(t, withPlan[0].QueryPlan)
	require.Equal(t, v1pb.QueryOption_JSON, withPlan[0].QueryPlan.Format)

	withoutPlan := OmniResultsToQueryResultsWith(results, 1024, QueryResultOptions{ClearExplainPlan: true})
	require.Len(t, withoutPlan, 1)
	require.Nil(t, withoutPlan[0].QueryPlan)

	typedWithContent := OmniResultsToQueryResultsWith([]*omnidriver.Result{
		{
			Statement: "EXPLAIN SELECT 1",
			Columns:   []omnidriver.Column{{Name: "col", TypeName: "int"}},
			Rows: [][]omnidriver.Value{
				{{Kind: omnidriver.Int64, Int64: 1}},
				{{Kind: omnidriver.Int64, Int64: 2}},
			},
			Plan:      &omnidriver.Plan{Format: "json", Executed: true, Content: `{"ok":true}`},
			Truncated: true,
			Err:       errors.New("query failed"),
		},
	}, 512, QueryResultOptions{
		TypedExplainPlan:     true,
		IncludePlanContent:   true,
		UppercaseColumnTypes: true,
		TruncatedWins:        true,
		Limit:                1,
	})
	require.Len(t, typedWithContent, 1)
	require.Equal(t, []string{"INT"}, typedWithContent[0].ColumnTypeNames)
	require.Equal(t, int64(1), typedWithContent[0].RowsCount)
	require.Equal(t, v1pb.QueryOption_JSON, typedWithContent[0].QueryPlan.Format)
	require.Equal(t, `{"ok":true}`, typedWithContent[0].QueryPlan.Content)
	require.Equal(t, FormatMaximumSQLResultSizeMessage(512), typedWithContent[0].Error)
}

func TestQueryResultOptionsExtensionsAndMySQLForkHelpers(t *testing.T) {
	t.Run("FillZeroAffectedRows and ClearExplainPlanOnError", func(t *testing.T) {
		got := OmniResultsToQueryResultsWith([]*omnidriver.Result{
			{Statement: "CREATE TABLE t (id INT)"},
			{Statement: "EXPLAIN SELECT 1", Plan: &omnidriver.Plan{Format: "text"}, Err: errors.New("boom")},
		}, 1024, QueryResultOptions{
			ClearExplainPlanOnError: true,
			TypedExplainPlan:        true,
			FillZeroAffectedRows:    true,
		})
		require.Len(t, got, 2)
		require.Equal(t, []string{"Affected Rows"}, got[0].ColumnNames)
		require.Equal(t, int64(1), got[0].RowsCount)
		require.Equal(t, int64(0), got[0].Rows[0].Values[0].GetInt64Value())
		require.Nil(t, got[1].QueryPlan)
		require.Equal(t, "boom", got[1].Error)
	})

	t.Run("FormatOutOfRangeTimestamp formats extreme timestamps without mutating input", func(t *testing.T) {
		outOfRangeNTZ := omnidriver.Value{Kind: omnidriver.Timestamp, Time: time.Date(0, 1, 1, 0, 0, 0, 7000000, time.UTC)}
		outOfRangeTZ := omnidriver.Value{Kind: omnidriver.TimestampTZ, Time: time.Date(10000, 1, 2, 3, 4, 5, 0, time.UTC)}
		validTS := omnidriver.Value{Kind: omnidriver.Timestamp, Time: time.Date(2026, 5, 11, 12, 34, 56, 0, time.UTC)}
		input := &omnidriver.Result{
			Statement: "SELECT 1",
			Columns:   []omnidriver.Column{{Name: "a"}, {Name: "b"}, {Name: "c"}},
			Rows:      [][]omnidriver.Value{{outOfRangeNTZ, outOfRangeTZ, validTS}},
		}
		got := OmniResultToQueryResultWith(input, 1024, QueryResultOptions{
			FormatOutOfRangeTimestamp: FormatDefaultTimestamp,
		})
		require.Equal(t, "0000-01-01 00:00:00.007", got.Rows[0].Values[0].GetStringValue())
		require.Equal(t, "10000-01-02T03:04:05Z", got.Rows[0].Values[1].GetStringValue())
		require.NotNil(t, got.Rows[0].Values[2].GetTimestampValue())
		// Input cells must remain untouched.
		require.Equal(t, omnidriver.Timestamp, input.Rows[0][0].Kind)
		require.Equal(t, omnidriver.TimestampTZ, input.Rows[0][1].Kind)
	})

	t.Run("OmniQueryOptions maps context fields", func(t *testing.T) {
		qCtx := db.QueryContext{
			Schema:               "app",
			Limit:                25,
			Explain:              true,
			MaximumSQLResultSize: 4096,
			MaskingEnabled:       true,
		}
		require.Equal(t, omnidriver.QueryOptions{
			Schema:         "app",
			Limit:          25,
			Explain:        true,
			ExplainFormat:  "JSON",
			MaxResultBytes: 4096,
			MaskingEnabled: true,
		}, OmniQueryOptions(qCtx, true, "JSON"))
		require.Equal(t, omnidriver.QueryOptions{
			Limit:          25,
			Explain:        true,
			ExplainFormat:  "TEXT",
			MaxResultBytes: 4096,
			MaskingEnabled: true,
		}, OmniQueryOptions(qCtx, false, "TEXT"))
	})

	t.Run("MySQLFork helpers", func(t *testing.T) {
		opts := MySQLForkQueryOptions(db.QueryContext{
			Limit:                50,
			Explain:              true,
			MaximumSQLResultSize: 2048,
			MaskingEnabled:       true,
		})
		require.Equal(t, omnidriver.QueryOptions{
			Limit:          50,
			Explain:        true,
			MaxResultBytes: 2048,
			MaskingEnabled: true,
		}, opts)

		resOpts := MySQLForkQueryResultOptions()
		require.True(t, resOpts.SkipPlan)
		require.True(t, resOpts.SkipMessages)
		require.NotNil(t, resOpts.ConvertValue)

		require.Equal(t, NullRowValue, MySQLForkRowValue(omnidriver.Value{Kind: omnidriver.Bytes, Bytes: []byte{}}))
		require.Equal(t, []byte("abc"), MySQLForkRowValue(omnidriver.Value{Kind: omnidriver.Bytes, Bytes: []byte("abc")}).GetBytesValue())
		require.Equal(t, 1.5, MySQLForkRowValue(omnidriver.Value{Kind: omnidriver.Float32, Float64: 1.5}).GetDoubleValue())
		require.Equal(t, int64(7), MySQLForkRowValue(omnidriver.Value{Kind: omnidriver.Int32, Int64: 7}).GetInt64Value())
		require.Equal(t, NullRowValue, MySQLForkRowValue(omnidriver.Value{Kind: omnidriver.Uint64, Uint64: 1}))
	})
}

func TestFillZeroAffectedRows(t *testing.T) {
	result := &omnidriver.Result{Statement: "CREATE TABLE t (id INT)"}
	qr := OmniResultToQueryResult(result, 1024)
	fillZeroAffectedRows(result, qr)
	require.Equal(t, []string{"Affected Rows"}, qr.ColumnNames)
	require.Equal(t, int64(1), qr.RowsCount)
	require.Equal(t, int64(0), qr.Rows[0].Values[0].GetInt64Value())

	errResult := &omnidriver.Result{Statement: "BAD", Err: errors.New("boom")}
	errQR := OmniResultToQueryResult(errResult, 1024)
	fillZeroAffectedRows(errResult, errQR)
	require.Empty(t, errQR.ColumnNames)
}

func TestOmniExplainFormat(t *testing.T) {
	require.Equal(t, "JSON", OmniExplainFormat(v1pb.QueryOption_JSON, "TEXT", v1pb.QueryOption_JSON))
	require.Equal(t, "TEXT", OmniExplainFormat(v1pb.QueryOption_XML, "TEXT", v1pb.QueryOption_JSON))
	require.Equal(t, "", OmniExplainFormat(v1pb.QueryOption_XML, "", v1pb.QueryOption_TEXT, v1pb.QueryOption_JSON))
}

func TestOmniInstanceToMetadata(t *testing.T) {
	require.Nil(t, omniInstanceToMetadata(nil, true))

	limit := int32(10)
	validUntil := "2030-01-01"
	attr := "SUPERUSER"
	instance := &omnidriver.Instance{
		Version:             "8.0.0",
		LowerCaseTableNames: 1,
		Databases:           []*metadatapb.DatabaseSchemaMetadata{{Name: "app"}},
		Roles: []*omnidriver.Role{
			{Name: "admin", ConnectionLimit: &limit, ValidUntil: &validUntil, Attribute: &attr},
		},
	}

	withoutMeta := omniInstanceToMetadata(instance, false)
	require.Equal(t, "8.0.0", withoutMeta.Version)
	require.Len(t, withoutMeta.Databases, 1)
	require.Nil(t, withoutMeta.Metadata)

	withMeta := omniInstanceToMetadata(instance, true)
	require.Equal(t, int32(1), withMeta.Metadata.GetMysqlLowerCaseTableNames())
	require.Len(t, withMeta.Metadata.GetRoles(), 1)
	require.Equal(t, "admin", withMeta.Metadata.GetRoles()[0].GetName())
	require.Equal(t, int32(10), withMeta.Metadata.GetRoles()[0].GetConnectionLimit())
	require.Equal(t, "2030-01-01", withMeta.Metadata.GetRoles()[0].GetValidUntil())
	require.Equal(t, "SUPERUSER", withMeta.Metadata.GetRoles()[0].GetAttribute())
}

func TestExecuteLoggerOptions(t *testing.T) {
	var logs []*storepb.TaskRunLog
	opts := db.ExecuteOptions{
		CreateTaskRunLog: func(_ time.Time, entry *storepb.TaskRunLog) error {
			logs = append(logs, entry)
			return nil
		},
	}

	omitLogger := NewExecuteLogger(opts, WithOmitPerResultAffectedRows(), WithCommandRange(func(cmd omnidriver.Command) *storepb.Range {
		return &storepb.Range{Start: int32(cmd.ByteStart + 1), End: int32(cmd.ByteEnd + 1)}
	}))
	omitLogger.CommandExecute(omnidriver.Command{ByteStart: 0, ByteEnd: 8, Text: "SELECT 1"})
	omitLogger.CommandResponse(3, []int64{1, 2}, nil)
	omitLogger.TransactionControl(omnidriver.Begin, nil)
	omitLogger.Retry(errors.New("retry"), 1, 2)

	require.Len(t, logs, 4)
	require.Equal(t, &storepb.Range{Start: 1, End: 9}, logs[0].GetCommandExecute().GetRange())
	require.Equal(t, int64(3), logs[1].GetCommandResponse().GetAffectedRows())
	require.Nil(t, logs[1].GetCommandResponse().GetAllAffectedRows())
	require.Equal(t, storepb.TaskRunLog_TransactionControl_BEGIN, logs[2].GetTransactionControl().GetType())
	require.Equal(t, int32(1), logs[3].GetRetryInfo().GetRetryCount())

	logs = nil
	skipLogger := NewExecuteLogger(opts, WithSkipCommandLogs())
	skipLogger.CommandExecute(omnidriver.Command{Text: "SELECT 1"})
	skipLogger.CommandResponse(1, []int64{1}, nil)
	skipLogger.TransactionControl(omnidriver.Commit, nil)
	require.Len(t, logs, 1)
	require.Equal(t, storepb.TaskRunLog_TransactionControl_COMMIT, logs[0].GetTransactionControl().GetType())
}

func TestDirectiveShift(t *testing.T) {
	for _, tc := range []struct {
		script string
		want   string
	}{
		{"-- txn-mode=off\nINSERT INTO t VALUES (1)", "INSERT INTO t VALUES (1)"},
		{"-- keep\n-- txn-mode=off\nINSERT INTO t VALUES (1)", "INSERT INTO t VALUES (1)"},
		{"-- txn-mode=off\n-- keep\nINSERT INTO t VALUES (1)", "-- keep\nINSERT INTO t VALUES (1)"},
		{"-- keep\n\n-- txn-isolation=serializable\n\nINSERT INTO t VALUES (1)", "\nINSERT INTO t VALUES (1)"},
		{"INSERT INTO t VALUES (1)", "INSERT INTO t VALUES (1)"},
		{"-- txn-mode=off\nSELECT 1;\n  SELECT 2", "SELECT 1;\n  SELECT 2"},
		{"-- txn-mode=off\nSELECT 1;\n  SELECT 2  ", "SELECT 1;\n  SELECT 2"},
		{"-- txn-mode=off\n  SELECT 1", "SELECT 1"},
		{"-- txn-mode=off\n\u00a0SELECT 1", "SELECT 1"},
	} {
		t.Run(tc.script, func(t *testing.T) {
			_, cleaned := base.ParseTransactionConfig(tc.script)
			start, end := DirectiveShift(tc.script, cleaned)(0, len(cleaned))
			require.Equal(t, tc.want, tc.script[start:end])
		})
	}
}
