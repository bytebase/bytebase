package mysql

import (
	"context"
	"testing"

	metadatapb "github.com/bytebase/omni/metadata"
	omniast "github.com/bytebase/omni/mysql/ast"
	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/component/sheet"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	mysqlparser "github.com/bytebase/bytebase/backend/plugin/parser/mysql"
	"github.com/bytebase/bytebase/backend/plugin/schema"
	"github.com/bytebase/bytebase/backend/store/model"
)

func runMySQLWalkThrough(t *testing.T, statement string) (*model.DatabaseMetadata, *storepb.Advice) {
	t.Helper()
	state := model.NewDatabaseMetadata(&metadatapb.DatabaseSchemaMetadata{Name: "test"}, nil, nil, storepb.Engine_MYSQL, true)
	sm := sheet.NewManager()
	stmts, _ := sm.GetStatementsForChecks(storepb.Engine_MYSQL, statement)
	advice := schema.WalkThroughWithContext(
		context.Background(),
		storepb.Engine_MYSQL,
		schema.WalkThroughContext{RawSQL: statement},
		state,
		base.ExtractASTs(stmts),
	)
	return state, advice
}

func TestWalkThroughSeamPrechecksAndErrorMapping(t *testing.T) {
	t.Run("updates database metadata through registry on valid DDL", func(t *testing.T) {
		state, advice := runMySQLWalkThrough(t, "CREATE TABLE t1 (a INT); RENAME TABLE t1 TO t2;")
		require.Nil(t, advice)
		require.Nil(t, state.GetSchemaMetadata("").GetTable("t1"))
		require.NotNil(t, state.GetSchemaMetadata("").GetTable("t2"))
	})

	t.Run("cross-database rename pre-creates target database and removes moved table", func(t *testing.T) {
		for _, stmt := range []string{
			"CREATE TABLE `test`.`t1` (a INT); RENAME TABLE `test`.`t1` TO `other`.`t1`;",
			"CREATE TABLE t1(a INT); ALTER TABLE t1 RENAME TO other.t1;",
		} {
			state, advice := runMySQLWalkThrough(t, stmt)
			require.Nil(t, advice, stmt)
			require.Empty(t, state.GetSchemaMetadata("").GetProto().GetTables(), stmt)
		}
	})

	t.Run("literal containing SELECT does not trigger CTAS fallback", func(t *testing.T) {
		state, advice := runMySQLWalkThrough(t, "CREATE TABLE t_literal (c varchar(20) DEFAULT ' SELECT ');")
		require.Nil(t, advice)
		require.NotNil(t, state.GetSchemaMetadata("").GetTable("t_literal"))
	})

	t.Run("omnicatalog prechecks and catalog error mapping", func(t *testing.T) {
		cases := []struct {
			name        string
			statement   string
			wantCode    code.Code
			wantContent string
		}{
			{
				name:        "cross-database create table and rename",
				statement:   "CREATE TABLE `test`.`t1` (a INT); CREATE TABLE `other`.`t1` (a INT); RENAME TABLE `other`.`t1` TO `other`.`t2`;",
				wantCode:    code.NotCurrentDatabase,
				wantContent: "Database `other` is not the current database `test`",
			},
			{
				name:        "create another database",
				statement:   "CREATE DATABASE another;",
				wantCode:    code.NotCurrentDatabase,
				wantContent: "Database `another` is not the current database `test`",
			},
			{
				name:        "drop current database",
				statement:   "-- this is comment\nDROP DATABASE test;\nCREATE TABLE t(a int);",
				wantCode:    code.DatabaseIsDeleted,
				wantContent: "Database `test` is deleted",
			},
			{
				name:        "earlier exec error wins over later precheck",
				statement:   "DROP TABLE missing; CREATE DATABASE other;",
				wantCode:    code.TableNotExists,
				wantContent: "Table `missing` does not exist",
			},
			{
				name:        "CTAS AST precheck",
				statement:   "CREATE TABLE t1(a int); CREATE TABLE t2 (b INT) SELECT a FROM t1;",
				wantCode:    code.StatementCreateTableAs,
				wantContent: `CREATE TABLE AS statement is used in "CREATE TABLE t2 (b INT) SELECT a FROM t1;"`,
			},
			{
				name:        "CTAS with VALUES ROW",
				statement:   "CREATE TABLE tvx SELECT x, z FROM (VALUES ROW(1,3,5), ROW(2,4,6)) AS v(x,y,z);",
				wantCode:    code.StatementCreateTableAs,
				wantContent: `CREATE TABLE AS statement is used in "CREATE TABLE tvx SELECT x, z FROM (VALUES ROW(1,3,5), ROW(2,4,6)) AS v(x,y,z);"`,
			},
			{
				name:        "CTAS over existing table reports TableExists",
				statement:   "CREATE TABLE t1(a int); CREATE TABLE t1 AS SELECT 1;",
				wantCode:    code.TableExists,
				wantContent: "Table `t1` already exists",
			},
			{
				name:        "spatial index on nullable geometry",
				statement:   "CREATE TABLE t1(a GEOMETRY); CREATE SPATIAL INDEX sk1 ON t1 (a);",
				wantCode:    code.SpatialIndexKeyNullable,
				wantContent: "All parts of a SPATIAL index must be NOT NULL",
			},
			{
				name:        "invalid NOT NULL DEFAULT NULL",
				statement:   "CREATE TABLE t1(a INT NOT NULL DEFAULT NULL);",
				wantCode:    code.InvalidColumnDefault,
				wantContent: "Invalid default value for column `a`",
			},
			{
				name:        "BLOB column default",
				statement:   "CREATE TABLE t1(a BLOB DEFAULT 'x');",
				wantCode:    code.InvalidColumnDefault,
				wantContent: "BLOB, TEXT, GEOMETRY or JSON column `a` can't have a default value",
			},
			{
				name:        "duplicate AUTO_INCREMENT",
				statement:   "CREATE TABLE t1(a INT AUTO_INCREMENT, b INT AUTO_INCREMENT);",
				wantCode:    code.AutoIncrementExists,
				wantContent: "Incorrect table definition; there can be only one auto column and it must be defined as a key",
			},
			{
				name:        "invalid ON UPDATE",
				statement:   "CREATE TABLE t1(a INT ON UPDATE NOW());",
				wantCode:    code.OnUpdateColumnNotDatetimeOrTimestamp,
				wantContent: "Invalid ON UPDATE clause for 'a' column",
			},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				_, advice := runMySQLWalkThrough(t, tc.statement)
				require.NotNil(t, advice)
				require.Equal(t, tc.wantCode.Int32(), advice.Code)
				require.Equal(t, tc.wantContent, advice.Content)
			})
		}
	})
}

func TestWalkThroughCreateTableIfNotExistsCTASExistingTable(t *testing.T) {
	originDatabase := &metadatapb.DatabaseSchemaMetadata{
		Name: "test",
		Schemas: []*metadatapb.SchemaMetadata{
			{
				Tables: []*metadatapb.TableMetadata{
					{
						Name: "t1",
						Columns: []*metadatapb.ColumnMetadata{
							{
								Name:     "a",
								Position: 1,
								Nullable: true,
								Type:     "int",
							},
						},
					},
				},
			},
		},
	}

	state := model.NewDatabaseMetadata(originDatabase, nil, nil, storepb.Engine_MYSQL, true)
	require.NotNil(t, state.GetSchemaMetadata("").GetTable("t1"))
	statement := "CREATE TABLE IF NOT EXISTS t1 AS SELECT 1;"
	sm := sheet.NewManager()
	stmts, _ := sm.GetStatementsForChecks(storepb.Engine_MYSQL, statement)
	asts := base.ExtractASTs(stmts)
	node, ok := asts[0].(*mysqlparser.AST)
	require.True(t, ok)
	createTable, ok := node.Node.(*omniast.CreateTableStmt)
	require.True(t, ok)
	require.True(t, createTable.IfNotExists)

	advice := WalkThroughWithContext(context.Background(), schema.WalkThroughContext{RawSQL: statement}, state, asts)
	require.Nil(t, advice)
	require.NotNil(t, state.GetSchemaMetadata("").GetTable("t1"))
}
