package tidb

import (
	"context"
	"database/sql"
	"fmt"
	"testing"

	_ "github.com/go-sql-driver/mysql"
	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/common/testcontainer"
	"github.com/bytebase/bytebase/backend/component/sheet"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
)

// TestStatementDMLDryRunTiDBContainer exercises the dml_dry_run advisor against
// a real TiDB instance. The fixture-based TestTiDBRules path runs with a nil
// Driver, so it never reaches the EXPLAIN / BATCH DRY RUN logic — this is the
// only test that validates the live behavior after the omni migration (B1.2),
// in particular that the BATCH DRY RUN and inner-DML SQL derived by text-slicing
// (omni has no statement-level deparse) execute correctly against TiDB.
func TestStatementDMLDryRunTiDBContainer(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping TiDB testcontainer test in short mode")
	}
	ctx := context.Background()
	container := testcontainer.GetTestTiDBContainer(ctx, t)
	defer container.Close(ctx)

	const database = "dml_dry_run_test"
	if _, err := container.GetDB().ExecContext(ctx, "CREATE DATABASE IF NOT EXISTS "+database); err != nil {
		t.Fatalf("create database: %v", err)
	}

	dsn := fmt.Sprintf("root@tcp(%s:%s)/%s?multiStatements=true&tls=false", container.GetHost(), container.GetPort(), database)
	db, err := sql.Open("mysql", dsn)
	require.NoError(t, err)
	defer db.Close()
	for _, ddl := range []string{
		"CREATE TABLE t (id INT PRIMARY KEY, name VARCHAR(50))",
		"CREATE TABLE t2 (id INT PRIMARY KEY, name VARCHAR(50))",
		"INSERT INTO t VALUES (1, 'a'), (2, 'b'), (3, 'c')",
	} {
		_, err := db.ExecContext(ctx, ddl)
		require.NoError(t, err, ddl)
	}

	sm := sheet.NewManager()
	rule := &storepb.SQLReviewRule{Type: storepb.SQLReviewRule_STATEMENT_DML_DRY_RUN, Level: storepb.SQLReviewRule_WARNING}

	testCases := []struct {
		name       string
		statement  string
		wantAdvice bool
	}{
		{"plain delete valid", "DELETE FROM t WHERE id = 1", false},
		{"plain update valid", "UPDATE t SET name = 'x' WHERE id = 1", false},
		{"plain insert valid", "INSERT INTO t VALUES (4, 'd')", false},
		{"plain unknown column", "DELETE FROM t WHERE missing_col = 1", true},
		{"plain unknown table", "DELETE FROM missing_table WHERE id = 1", true},
		{"batch delete valid", "BATCH ON id LIMIT 2 DELETE FROM t WHERE id > 0", false},
		{"batch update valid", "BATCH ON id LIMIT 2 UPDATE t SET name = 'y' WHERE id > 0", false},
		{"batch insert-select valid", "BATCH ON id LIMIT 2 INSERT INTO t2 SELECT id, name FROM t", false},
		{"batch no-on valid", "BATCH LIMIT 2 DELETE FROM t WHERE id > 0", false},
		{"batch inner unknown column", "BATCH ON id LIMIT 2 DELETE FROM t WHERE missing_col > 0", true},
		{"batch unknown shard column", "BATCH ON missing_col LIMIT 2 DELETE FROM t WHERE id > 0", true},
		// TiDB rejects non-transactional REPLACE ... VALUES (only INSERT ... SELECT
		// is supported). The advisor must surface that as a dry-run failure; this
		// also exercises the REPLACE (*InsertStmt{IsReplace}) text-slice path.
		{"batch replace-values rejected by tidb", "BATCH ON id LIMIT 2 REPLACE INTO t VALUES (1, 'z')", true},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			checkCtx := advisor.Context{
				DBType:          storepb.Engine_TIDB,
				Driver:          db,
				CurrentDatabase: database,
				NoAppendBuiltin: true,
			}
			advices, err := advisor.SQLReviewCheck(ctx, sm, tc.statement, []*storepb.SQLReviewRule{rule}, checkCtx)
			require.NoError(t, err)
			if tc.wantAdvice {
				require.NotEmpty(t, advices, "expected dry-run advice for %q", tc.statement)
				for _, a := range advices {
					require.Equal(t, code.StatementDMLDryRunFailed.Int32(), a.Code)
				}
			} else {
				require.Empty(t, advices, "expected no advice for %q", tc.statement)
			}
		})
	}

	// A dry run must not mutate data: all 3 seed rows remain (no test case INSERTs
	// persist, since EXPLAIN / BATCH DRY RUN never execute the DML).
	var count int
	require.NoError(t, db.QueryRowContext(ctx, "SELECT COUNT(*) FROM t").Scan(&count))
	require.Equal(t, 3, count, "dry run must not mutate data")
}
