package oceanbase

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/component/sheet"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	_ "github.com/bytebase/bytebase/backend/plugin/advisor/oceanbase"
	"github.com/bytebase/bytebase/backend/plugin/db"
)

// TestProbeRefusesPreExecutionsOutsideATransaction pins the contract's rule:
// Autocommit exists for a statement the engine refuses inside a transaction, so
// pre-executions, which need one, are refused rather than run beside it.
func TestProbeRefusesPreExecutionsOutsideATransaction(t *testing.T) {
	_, err := (&Driver{}).Probe(context.Background(), db.ProbeContext{
		Autocommit:    true,
		PreExecutions: []string{"SET NAMES utf8mb4"},
	}, "SELECT 1")
	require.ErrorContains(t, err, "cannot run outside a transaction")
}

// TestProbeCapabilities runs the three capabilities the advisor seam asks of
// the driver against a live server: the statement probe the two row-limit rules
// read a JSON plan from, the dry-run probe, the estimate plancheck asks for, and
// the three-value catalogue lookup the offline-DDL rule binds.
func TestProbeCapabilities(t *testing.T) {
	server := newOceanBaseTestServer(t)
	ctx := context.Background()
	database := "probe_capabilities"
	admin := server.openRaw(ctx, t, "")
	exec(t, admin, "DROP DATABASE IF EXISTS "+database, "CREATE DATABASE "+database)
	t.Cleanup(func() { exec(t, admin, "DROP DATABASE IF EXISTS "+database) })
	driver := server.open(ctx, t, database)
	raw := server.openRaw(ctx, t, database)
	exec(t, raw,
		"CREATE TABLE t (id INT PRIMARY KEY, name VARCHAR(32), generated_col INT GENERATED ALWAYS AS (id + 1) STORED)",
		"INSERT INTO t (id, name) VALUES (1, 'a'), (2, 'b')",
	)

	// The affected-row and insert-row rules issue this probe and read the plan's
	// EST.ROWS; the plan is split across rows.
	probe, err := driver.Probe(ctx, db.ProbeContext{}, "EXPLAIN format=json UPDATE t SET name = 'x'")
	require.NoError(t, err)
	require.Equal(t, []string{"Query Plan"}, probe.Columns)
	plan, err := probe.Plan()
	require.NoError(t, err)
	require.Contains(t, plan, `"EST.ROWS":2`)

	// The dry-run rule only needs the statement to plan.
	_, err = driver.Probe(ctx, db.ProbeContext{}, "EXPLAIN UPDATE t SET name = 'x'")
	require.NoError(t, err)

	// A pre-execution runs in the same transaction before the statement.
	preExecutions, err := driver.Probe(ctx, db.ProbeContext{PreExecutions: []string{"SET @probe_value = 7"}}, "SELECT @probe_value AS v")
	require.NoError(t, err)
	require.Len(t, preExecutions.Rows, 1)
	require.Equal(t, []any{int64(7)}, preExecutions.Rows[0])

	// A statement the server refuses is the probe's error, and the session is
	// left out of the transaction it opened.
	_, err = driver.Probe(ctx, db.ProbeContext{}, "EXPLAIN UPDATE no_such_table SET a = 1")
	require.ErrorContains(t, err, "doesn't exist")
	_, err = driver.Probe(ctx, db.ProbeContext{}, "EXPLAIN SELECT 1")
	require.NoError(t, err)

	// plancheck's estimate, from the same plan the probe reads.
	rows, err := driver.CountAffectedRows(ctx, "UPDATE t SET name = 'x'")
	require.NoError(t, err)
	require.Equal(t, int64(2), rows)
	rows, err = driver.CountAffectedRows(ctx, "INSERT INTO t (id, name) SELECT id + 10, name FROM t")
	require.NoError(t, err)
	require.Equal(t, int64(2), rows)
	// Only a statement that changes rows has an estimate; the driver refuses
	// the rest before it reaches the server.
	_, err = driver.CountAffectedRows(ctx, "SELECT * FROM t")
	require.ErrorContains(t, err, "expected an INSERT, UPDATE or DELETE statement")

	// The offline-DDL rule's lookup binds database, table and column.
	extra, err := driver.ColumnExtra(ctx, database, "t", "generated_col")
	require.NoError(t, err)
	require.Equal(t, "STORED GENERATED", extra)
	_, err = driver.ColumnExtra(ctx, database, "t", "no_such_column")
	require.Error(t, err)
}

// TestLiveRulesProduceTheSameAdvice runs the four OceanBase rules that reach the
// connection through the adapter, so the advice a review reports is the advice
// the real driver's answers produce.
func TestLiveRulesProduceTheSameAdvice(t *testing.T) {
	server := newOceanBaseTestServer(t)
	ctx := context.Background()
	database := "probe_rules"
	admin := server.openRaw(ctx, t, "")
	exec(t, admin, "DROP DATABASE IF EXISTS "+database, "CREATE DATABASE "+database)
	t.Cleanup(func() { exec(t, admin, "DROP DATABASE IF EXISTS "+database) })
	driver := server.open(ctx, t, database)
	raw := server.openRaw(ctx, t, database)
	exec(t, raw,
		"CREATE TABLE t (id INT PRIMARY KEY, name VARCHAR(32), generated_col INT GENERATED ALWAYS AS (id + 1) STORED)",
		"INSERT INTO t (id, name) VALUES (1, 'a'), (2, 'b')",
	)

	const (
		// Two rows in t, so a statement without a WHERE affects both and a
		// limit of 1 is crossed.
		update = "UPDATE t SET name = 'xz';"
		dryRun = "UPDATE t SET name = 'xz' WHERE id = 1;"
		insert = "INSERT INTO t (id, name) SELECT id + 10, name FROM t;"
	)
	for _, tc := range []struct {
		name         string
		ruleType     storepb.SQLReviewRule_Type
		number       int32
		statement    string
		wantAdvice   string
		wantNoAdvice bool
	}{
		{
			name:       "affected row limit",
			ruleType:   storepb.SQLReviewRule_STATEMENT_AFFECTED_ROW_LIMIT,
			number:     1,
			statement:  update,
			wantAdvice: `"` + update + `" affected 2 rows (estimated). The count exceeds 1.`,
		},
		{
			name:       "insert row limit",
			ruleType:   storepb.SQLReviewRule_STATEMENT_INSERT_ROW_LIMIT,
			number:     1,
			statement:  insert,
			wantAdvice: `"` + insert + `" inserts 2 rows. The count exceeds 1.`,
		},
		{
			name:         "dml dry run",
			ruleType:     storepb.SQLReviewRule_STATEMENT_DML_DRY_RUN,
			statement:    dryRun,
			wantNoAdvice: true,
		},
		{
			name:       "offline ddl",
			ruleType:   storepb.SQLReviewRule_STATEMENT_DISALLOW_OFFLINE_DDL,
			statement:  "ALTER TABLE t DROP COLUMN generated_col;",
			wantAdvice: "Dropping stored generated columns is an offline DDL operation.",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rule := &storepb.SQLReviewRule{Type: tc.ruleType, Level: storepb.SQLReviewRule_WARNING, Engine: storepb.Engine_OCEANBASE}
			if tc.number > 0 {
				rule.Payload = &storepb.SQLReviewRule_NumberPayload{NumberPayload: &storepb.SQLReviewRule_NumberRulePayload{Number: tc.number}}
			}
			adviceList, err := advisor.SQLReviewCheck(context.Background(), sheet.NewManager(), tc.statement, []*storepb.SQLReviewRule{rule}, advisor.Context{
				DBType:          storepb.Engine_OCEANBASE,
				CurrentDatabase: database,
				Prober:          driver,
				NoAppendBuiltin: true,
			})
			require.NoError(t, err)
			if tc.wantNoAdvice {
				require.Empty(t, adviceList)
				return
			}
			require.Len(t, adviceList, 1)
			require.Equal(t, tc.wantAdvice, adviceList[0].Content)
		})
	}
}
