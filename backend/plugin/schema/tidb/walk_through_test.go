package tidb

import (
	"context"
	"testing"

	metadatapb "github.com/bytebase/omni/metadata"
	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/component/sheet"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/plugin/schema"
	"github.com/bytebase/bytebase/backend/store/model"
)

func runTiDBWalkThrough(t *testing.T, statement string, caseSensitive bool) (*model.DatabaseMetadata, *storepb.Advice) {
	t.Helper()
	state := model.NewDatabaseMetadata(&metadatapb.DatabaseSchemaMetadata{Name: "test"}, nil, nil, storepb.Engine_TIDB, caseSensitive)
	sm := sheet.NewManager()
	stmts, _ := sm.GetStatementsForChecks(storepb.Engine_TIDB, statement)
	advice := schema.WalkThroughWithContext(context.Background(), storepb.Engine_TIDB, schema.WalkThroughContext{RawSQL: statement}, state, base.ExtractASTs(stmts))
	return state, advice
}

func TestWalkThroughSeam(t *testing.T) {
	t.Run("updates database metadata through registry and preserves case sensitivity", func(t *testing.T) {
		state, advice := runTiDBWalkThrough(t, "CREATE TABLE t (a INT); RENAME TABLE t TO test.t1;", false)
		require.Nil(t, advice)
		require.False(t, state.GetIsObjectCaseSensitive())
		require.Nil(t, state.GetSchemaMetadata("").GetTable("t"))
		require.NotNil(t, state.GetSchemaMetadata("").GetTable("t1"))
	})

	t.Run("cross-database rename pre-creates target database", func(t *testing.T) {
		state, advice := runTiDBWalkThrough(t, "CREATE TABLE t(a int); RENAME TABLE t to other_db.t1;", true)
		require.Nil(t, advice)
		require.Empty(t, state.GetSchemaMetadata("").GetProto().GetTables())
	})

	t.Run("CTAS IF NOT EXISTS still rejected as unimplemented in TiDB", func(t *testing.T) {
		_, advice := runTiDBWalkThrough(t, "CREATE TABLE t(a INT); CREATE TABLE IF NOT EXISTS t AS SELECT 1 AS a;", true)
		require.NotNil(t, advice)
		require.Equal(t, code.DDLSimulationFailed.Int32(), advice.Code)
		require.Equal(t, "'CREATE TABLE ... SELECT' is not implemented yet", advice.Content)
	})

	t.Run("omnicatalog prechecks and TiDB error dialect mapping", func(t *testing.T) {
		cases := []struct {
			name        string
			statement   string
			wantCode    code.Code
			wantContent string
		}{
			{
				name:        "CTAS rejected as unimplemented in TiDB",
				statement:   "CREATE TABLE t as select * from t1;",
				wantCode:    code.DDLSimulationFailed,
				wantContent: "'CREATE TABLE ... SELECT' is not implemented yet",
			},
			{
				name:        "drop current database precheck",
				statement:   "-- comment\nDROP DATABASE test;\nCREATE TABLE t(a int);",
				wantCode:    code.DatabaseIsDeleted,
				wantContent: "Database `test` is deleted",
			},
			{
				name:        "not current database precheck",
				statement:   "CREATE DATABASE other;",
				wantCode:    code.NotCurrentDatabase,
				wantContent: "Database `other` is not the current database `test`",
			},
			{
				name:        "TEXT default maps to TiDB InvalidColumnDefault",
				statement:   "CREATE TABLE t(a TEXT DEFAULT '1');",
				wantCode:    code.InvalidColumnDefault,
				wantContent: "BLOB, TEXT or JSON column `a` can't have a default value",
			},
			{
				name:        "NOT NULL DEFAULT NULL maps to InvalidColumnDefault",
				statement:   "CREATE TABLE t(a INT NOT NULL DEFAULT NULL);",
				wantCode:    code.InvalidColumnDefault,
				wantContent: "Invalid default value for column `a`",
			},
			{
				name:        "ON UPDATE non-temporal maps to TiDB dialect message",
				statement:   "CREATE TABLE t(a int ON UPDATE NOW());",
				wantCode:    code.OnUpdateColumnNotDatetimeOrTimestamp,
				wantContent: "Column `a` use ON UPDATE but is not DATETIME or TIMESTAMP",
			},
			{
				name:        "duplicate AUTO_INCREMENT maps to TiDB dialect message",
				statement:   "CREATE TABLE t(a int auto_increment, b int auto_increment);",
				wantCode:    code.AutoIncrementExists,
				wantContent: "There can be only one auto column and it must be defined as a key",
			},
			{
				name:        "missing column in index (1072)",
				statement:   "CREATE TABLE t(a int); CREATE INDEX idx_c on t(c);",
				wantCode:    code.ColumnNotExists,
				wantContent: "Column `c` does not exist",
			},
			{
				name:        "missing column in DROP COLUMN (1091 Can't DROP)",
				statement:   "CREATE TABLE t(a int); ALTER TABLE t DROP COLUMN c;",
				wantCode:    code.ColumnNotExists,
				wantContent: "Column `c` does not exist",
			},
			{
				name:        "missing index in DROP INDEX (1091 index doesn't exist)",
				statement:   "CREATE TABLE t(a int); DROP INDEX nonexistent on t;",
				wantCode:    code.IndexNotExists,
				wantContent: "Index `nonexistent` does not exist",
			},
			{
				name:        "duplicate column",
				statement:   "CREATE TABLE t(a int); ALTER TABLE t ADD COLUMN a int;",
				wantCode:    code.ColumnExists,
				wantContent: "Column `a` already exists",
			},
			{
				name:        "duplicate index",
				statement:   "CREATE TABLE t(a int); CREATE INDEX idx_a ON t(a); CREATE INDEX idx_a ON t(a);",
				wantCode:    code.IndexExists,
				wantContent: "Index `idx_a` already exists",
			},
			{
				name:        "missing table",
				statement:   "DROP TABLE t1, t2;",
				wantCode:    code.TableNotExists,
				wantContent: "Table `t1` does not exist",
			},
			{
				name:        "TiDB 8200 error maps to DDLSimulationFailed",
				statement:   "CREATE TABLE t(a varchar(10)); CREATE FULLTEXT INDEX ft ON t(a);",
				wantCode:    code.DDLSimulationFailed,
				wantContent: "FULLTEXT and SPATIAL index is not supported",
			},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				_, advice := runTiDBWalkThrough(t, tc.statement, true)
				require.NotNil(t, advice)
				require.Equal(t, tc.wantCode.Int32(), advice.Code)
				require.Equal(t, tc.wantContent, advice.Content)
			})
		}
	})
}
