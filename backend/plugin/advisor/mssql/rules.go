// Package mssql implements the SQL advisor rules for MSSQL.
package mssql

import (
	"fmt"
	"strings"

	"github.com/bytebase/omni/mssql/ast"
	mssqlv1review "github.com/bytebase/omni/mssql/v1review"
	"github.com/bytebase/omni/v1review"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	tsqlparser "github.com/bytebase/bytebase/backend/plugin/parser/tsql"
)

func init() {
	extract := advisor.NodeExtractor(func(a base.AST) (ast.Node, bool) {
		n, ok := tsqlparser.GetNode(a)
		return n, ok && n != nil
	})
	e := advisor.NewV1Engine(extract, nil, advisor.V1PosFinding)

	compat := func(c code.Code, prefix string) advisor.V1CompatEntry {
		return advisor.V1CompatEntry{Code: c, Format: prefix + " may cause incompatibility with the existing data and code"}
	}
	compatEntries := map[v1review.Code]advisor.V1CompatEntry{
		v1review.IncompatibleDropTable:            compat(code.CompatibilityDropSchema, "Drop table %s"),
		v1review.IncompatibleDropSchema:           compat(code.CompatibilityDropSchema, "Drop schema %s"),
		v1review.IncompatibleDropDatabase:         compat(code.CompatibilityDropSchema, "Drop database %s"),
		v1review.IncompatibleDropColumn:           compat(code.CompatibilityDropSchema, "Drop column %s"),
		v1review.IncompatibleAlterColumn:          compat(code.CompatibilityAlterColumn, "Alter COLUMN %s"),
		v1review.IncompatibleAddForeignKeyNoCheck: compat(code.CompatibilityAddForeignKey, "Add FOREIGN KEY WITH NO CHECK"),
		v1review.IncompatibleAddCheckNoCheck:      compat(code.CompatibilityAddForeignKey, "Add CHECK WITH NO CHECK"),
		v1review.IncompatibleAddPrimaryKey:        compat(code.CompatibilityAddPrimaryKey, "Add PRIMARY KEY"),
		v1review.IncompatibleAddUniqueKey:         compat(code.CompatibilityAddUniqueKey, "Add UNIQUE KEY"),
		v1review.IncompatibleAddCheck:             compat(code.CompatibilityAddCheck, "Add CHECK"),
		v1review.IncompatibleAddForeignKey:        compat(code.CompatibilityAddForeignKey, "Add FOREIGN KEY"),
		v1review.IncompatibleRenameTable:          {Code: code.CompatibilityRenameTable, Format: "sp_rename may cause incompatibility with the existing data and code, and break scripts and stored procedures."},
	}

	line1 := &storepb.Position{Line: 1}
	advisor.RegisterRules(storepb.Engine_MSSQL, map[storepb.SQLReviewRule_Type]advisor.Advisor{
		storepb.SQLReviewRule_BUILTIN_PRIOR_BACKUP_CHECK: e.PriorBackup(advisor.V1PriorBackupConfig[ast.Node]{
			Engine:     storepb.Engine_MSSQL,
			DefaultPos: line1,
			Check: func(checkCtx advisor.Context, _ []advisor.V1Stmt[ast.Node], nodes []ast.Node, facts v1review.PriorBackupFacts) ([]v1review.Finding, error) {
				return mssqlv1review.PriorBackupCheck(nodes, e.SchemaOf(checkCtx), facts), nil
			},
			Render: func(_ []advisor.V1Stmt[ast.Node], _ []ast.Node, f v1review.Finding) (code.Code, string, *storepb.Position, bool) {
				switch f.Code {
				case v1review.PriorBackupScriptTooLarge:
					return code.BuiltinPriorBackupCheck, fmt.Sprintf("The size of statements in the sheet exceeds the limit of %s", f.Value), line1, true
				case v1review.PriorBackupMixedDMLOnTable:
					return code.BuiltinPriorBackupCheck, fmt.Sprintf("The statement type is not the same for all statements on the same table %q", f.Table), line1, true
				default:
					return 0, "", nil, false
				}
			},
		}),
		storepb.SQLReviewRule_COLUMN_MAXIMUM_VARCHAR_LENGTH: e.NumberStmt(mssqlv1review.ColumnMaximumVarcharLength, true, code.VarcharLengthExceedsLimit,
			e.FmtLimit("The maximum varchar length is %d.")),
		storepb.SQLReviewRule_COLUMN_NO_NULL: e.SimpleStmt(mssqlv1review.ColumnNoNull, code.ColumnCannotNull,
			e.FmtName("Column [%s] is nullable, which is not allowed.")),
		storepb.SQLReviewRule_COLUMN_REQUIRED: e.StringArrayStmt(mssqlv1review.ColumnRequired, "string_array_payload is required for column required rule", code.NoRequiredColumn, "",
			e.FmtTableCol("Table %s missing required column %q")),
		storepb.SQLReviewRule_COLUMN_TYPE_DISALLOW_LIST: e.StringArrayStmt(mssqlv1review.ColumnTypeDisallowList, "string_array_payload is required for column type disallow list rule", code.DisabledColumnType, "", func(_ advisor.V1Stmt[ast.Node], f v1review.Finding) string {
			return "Column type " + f.Value + " is disallowed"
		}),
		storepb.SQLReviewRule_SYSTEM_FUNCTION_DISALLOW_CREATE: e.SimpleConst(mssqlv1review.SystemFunctionDisallowCreate, code.DisallowCreateFunction,
			"Creating or altering functions is prohibited"),
		storepb.SQLReviewRule_INDEX_NOT_REDUNDANT: e.SimpleBatch(
			func(checkCtx advisor.Context, nodes []ast.Node) []v1review.Finding {
				return mssqlv1review.IndexNotRedundant(nodes, v1review.Schema{Metadata: checkCtx.DBSchema}, checkCtx.CurrentDatabase)
			},
			code.RedundantIndex, "",
			func(f v1review.Finding) string {
				return fmt.Sprintf("Redundant indexes with the same prefix ('%s' and '%s') in '%s.%s' is not allowed", f.Other, f.Name, f.Schema, f.Table)
			},
		),
		storepb.SQLReviewRule_SCHEMA_BACKWARD_COMPATIBILITY: e.BackwardCompatibility(
			func(checkCtx advisor.Context, nodes []ast.Node) []v1review.Finding {
				return mssqlv1review.SchemaBackwardCompatibility(nodes, checkCtx.CurrentDatabase)
			},
			compatEntries,
			func(_ advisor.V1Stmt[ast.Node], f v1review.Finding, format string) string {
				if strings.Contains(format, "%s") {
					return fmt.Sprintf(format, f.Name)
				}
				return format
			},
		),
		storepb.SQLReviewRule_NAMING_IDENTIFIER_NO_KEYWORD: e.SimpleStmt(mssqlv1review.NamingIdentifierNoKeyword, code.NameIsKeywordIdentifier,
			e.FmtName("Identifier [%s] is a keyword identifier and should be avoided."), e.PosSearchOrStart),
		storepb.SQLReviewRule_NAMING_TABLE: e.NamingRule(
			mssqlv1review.NamingTable, code.NamingTableConventionMismatch, "", "table", true,
			func(f v1review.Finding) string { return f.Name },
		),
		storepb.SQLReviewRule_NAMING_TABLE_NO_KEYWORD: e.SimpleStmt(mssqlv1review.NamingTableNoKeyword, code.NameIsKeywordIdentifier,
			e.FmtName("Table name [%s] is a reserved keyword and should be avoided.")),
		storepb.SQLReviewRule_SYSTEM_PROCEDURE_DISALLOW_CREATE: e.SimpleConst(mssqlv1review.SystemProcedureDisallowCreate, code.DisallowCreateProcedure,
			"Creating or altering procedures is prohibited"),
		storepb.SQLReviewRule_STATEMENT_SELECT_NO_SELECT_ALL: e.SimpleConst(mssqlv1review.StatementSelectNoSelectAll, code.StatementSelectAll,
			"Avoid using SELECT *."),
		storepb.SQLReviewRule_STATEMENT_DISALLOW_CROSS_DB_QUERIES: e.SimpleBatchDB(mssqlv1review.StatementDisallowCrossDBQueries, code.StatementDisallowCrossDBQueries, "",
			func(f v1review.Finding) string {
				return fmt.Sprintf("Cross database queries (target databse: '%s', current database: '%s') are prohibited", f.Name, f.Value)
			}),
		storepb.SQLReviewRule_STATEMENT_DISALLOW_TRUNCATE: e.SimpleStmt(mssqlv1review.StatementDisallowTruncate, code.StatementDisallowTruncate, func(_ advisor.V1Stmt[ast.Node], f v1review.Finding) string {
			return fmt.Sprintf(
				`TRUNCATE TABLE %q is not allowed: TRUNCATE is minimally logged, bypasses DELETE triggers, and requires ALTER TABLE permission. Prior-backup treats this as DDL and does not produce row-level snapshots.`,
				f.Table,
			)
		}),
		storepb.SQLReviewRule_STATEMENT_WHERE_DISALLOW_FUNCTIONS_AND_CALCULATIONS: e.WhereDisallowFuncAndCalc(func(checkCtx advisor.Context) func([]ast.Node) []v1review.Finding {
			schema := e.SchemaOf(checkCtx)
			return func(nodes []ast.Node) []v1review.Finding {
				return mssqlv1review.StatementWhereDisallowFunctionsAndCalculations(nodes, schema)
			}
		}, false),
		storepb.SQLReviewRule_STATEMENT_WHERE_REQUIRE_SELECT: e.SimpleConst(mssqlv1review.StatementWhereRequireSelect, code.StatementNoWhere,
			"WHERE clause is required for SELETE statement."),
		storepb.SQLReviewRule_STATEMENT_WHERE_REQUIRE_UPDATE_DELETE: e.SimpleStmt(mssqlv1review.StatementWhereRequireUpdateDelete, code.StatementNoWhere,
			e.FmtName("WHERE clause is required for %s statement.")),
		storepb.SQLReviewRule_TABLE_DISALLOW_DDL: e.StringArrayStmt(mssqlv1review.TableDisallowDDL, "string_array_payload is required for table disallow DDL rule", code.TableDisallowDDL, v1review.TableDisallowed,
			e.FmtName("DDL is disallowed on table %s.")),
		storepb.SQLReviewRule_TABLE_DISALLOW_DML: e.StringArrayStmt(mssqlv1review.TableDisallowDML, "string_array_payload is required for table disallow DML rule", code.TableDisallowDML, v1review.TableDisallowed,
			e.FmtName("DML is disallowed on table %s.")),
		storepb.SQLReviewRule_TABLE_DROP_NAMING_CONVENTION: e.NamingRule(
			mssqlv1review.TableDropNaming, code.TableDropNamingConventionMismatch, "naming_payload is required for table drop naming convention rule", "drop table", false,
			func(f v1review.Finding) string { return fmt.Sprintf("[%s]", f.Name) },
		),
		storepb.SQLReviewRule_TABLE_NO_FOREIGN_KEY: e.SimpleBatchNodes(mssqlv1review.TableNoForeignKey, code.TableHasFK, v1review.TableForeignKey,
			advisor.FmtFindingName("FOREIGN KEY is not allowed in the table %s.")),
		storepb.SQLReviewRule_TABLE_REQUIRE_PK: e.SimpleBatchNodes(mssqlv1review.TableRequirePK, code.TableNoPK, v1review.TablePrimaryKeyMissing,
			advisor.FmtFindingName("Table %s requires PRIMARY KEY.")),
	})
}
