// Package mssql implements the SQL advisor rules for MSSQL.
package mssql

import (
	"context"
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
	extract := func(checkCtx advisor.Context) ([]advisor.V1Stmt[ast.Node], error) {
		return advisor.ExtractNodes(checkCtx.ParsedStatements, func(a base.AST) (ast.Node, bool) {
			n, ok := tsqlparser.GetNode(a)
			return n, ok && n != nil
		}), nil
	}
	byteOffsetLine := func(sql string, offset int) int32 {
		return tsqlparser.ByteOffsetToRunePosition(sql, offset).Line
	}
	e := advisor.NewV1Engine(extract, byteOffsetLine, advisor.V1PosFinding)

	compatEntries := map[v1review.Code]advisor.V1CompatEntry{
		v1review.IncompatibleDropTable:            {Code: code.CompatibilityDropSchema, Format: "Drop table %s may cause incompatibility with the existing data and code"},
		v1review.IncompatibleDropSchema:           {Code: code.CompatibilityDropSchema, Format: "Drop schema %s may cause incompatibility with the existing data and code"},
		v1review.IncompatibleDropDatabase:         {Code: code.CompatibilityDropSchema, Format: "Drop database %s may cause incompatibility with the existing data and code"},
		v1review.IncompatibleDropColumn:           {Code: code.CompatibilityDropSchema, Format: "Drop column %s may cause incompatibility with the existing data and code"},
		v1review.IncompatibleAlterColumn:          {Code: code.CompatibilityAlterColumn, Format: "Alter COLUMN %s may cause incompatibility with the existing data and code"},
		v1review.IncompatibleAddForeignKeyNoCheck: {Code: code.CompatibilityAddForeignKey, Format: "Add FOREIGN KEY WITH NO CHECK may cause incompatibility with the existing data and code"},
		v1review.IncompatibleAddCheckNoCheck:      {Code: code.CompatibilityAddForeignKey, Format: "Add CHECK WITH NO CHECK may cause incompatibility with the existing data and code"},
		v1review.IncompatibleAddPrimaryKey:        {Code: code.CompatibilityAddPrimaryKey, Format: "Add PRIMARY KEY may cause incompatibility with the existing data and code"},
		v1review.IncompatibleAddUniqueKey:         {Code: code.CompatibilityAddUniqueKey, Format: "Add UNIQUE KEY may cause incompatibility with the existing data and code"},
		v1review.IncompatibleAddCheck:             {Code: code.CompatibilityAddCheck, Format: "Add CHECK may cause incompatibility with the existing data and code"},
		v1review.IncompatibleAddForeignKey:        {Code: code.CompatibilityAddForeignKey, Format: "Add FOREIGN KEY may cause incompatibility with the existing data and code"},
		v1review.IncompatibleRenameTable:          {Code: code.CompatibilityRenameTable, Format: "sp_rename may cause incompatibility with the existing data and code, and break scripts and stored procedures."},
	}

	advisor.Register(storepb.Engine_MSSQL, storepb.SQLReviewRule_BUILTIN_PRIOR_BACKUP_CHECK, mssqlPriorBackup(e))
	advisor.Register(storepb.Engine_MSSQL, storepb.SQLReviewRule_COLUMN_MAXIMUM_VARCHAR_LENGTH,
		e.NumberStmt(mssqlv1review.ColumnMaximumVarcharLength, true, code.VarcharLengthExceedsLimit, func(_ advisor.V1Stmt[ast.Node], _ v1review.Finding, maximum int) string {
			return fmt.Sprintf("The maximum varchar length is %d.", maximum)
		}))
	advisor.Register(storepb.Engine_MSSQL, storepb.SQLReviewRule_COLUMN_NO_NULL,
		e.SimpleStmt(mssqlv1review.ColumnNoNull, code.ColumnCannotNull, func(_ advisor.V1Stmt[ast.Node], f v1review.Finding) string {
			return fmt.Sprintf("Column [%s] is nullable, which is not allowed.", f.Name)
		}))
	advisor.Register(storepb.Engine_MSSQL, storepb.SQLReviewRule_COLUMN_REQUIRED,
		e.StringArrayStmt(mssqlv1review.ColumnRequired, "string_array_payload is required for column required rule", code.NoRequiredColumn, "", func(_ advisor.V1Stmt[ast.Node], f v1review.Finding) string {
			return fmt.Sprintf("Table %s missing required column %q", f.Table, f.Name)
		}))
	advisor.Register(storepb.Engine_MSSQL, storepb.SQLReviewRule_COLUMN_TYPE_DISALLOW_LIST,
		e.StringArrayStmt(mssqlv1review.ColumnTypeDisallowList, "string_array_payload is required for column type disallow list rule", code.DisabledColumnType, "", func(_ advisor.V1Stmt[ast.Node], f v1review.Finding) string {
			return "Column type " + f.Value + " is disallowed"
		}))
	advisor.Register(storepb.Engine_MSSQL, storepb.SQLReviewRule_SYSTEM_FUNCTION_DISALLOW_CREATE,
		e.SimpleStmt(mssqlv1review.SystemFunctionDisallowCreate, code.DisallowCreateFunction, func(advisor.V1Stmt[ast.Node], v1review.Finding) string {
			return "Creating or altering functions is prohibited"
		}))
	advisor.Register(storepb.Engine_MSSQL, storepb.SQLReviewRule_INDEX_NOT_REDUNDANT,
		e.SimpleBatch(
			func(checkCtx advisor.Context, nodes []ast.Node) []v1review.Finding {
				return mssqlv1review.IndexNotRedundant(nodes, v1review.Schema{Metadata: checkCtx.DBSchema}, checkCtx.CurrentDatabase)
			},
			code.RedundantIndex,
			"",
			func(f v1review.Finding) string {
				return fmt.Sprintf("Redundant indexes with the same prefix ('%s' and '%s') in '%s.%s' is not allowed", f.Other, f.Name, f.Schema, f.Table)
			},
		))
	advisor.Register(storepb.Engine_MSSQL, storepb.SQLReviewRule_SCHEMA_BACKWARD_COMPATIBILITY,
		e.BackwardCompatibility(
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
		))
	advisor.Register(storepb.Engine_MSSQL, storepb.SQLReviewRule_NAMING_IDENTIFIER_NO_KEYWORD,
		e.SimpleStmt(mssqlv1review.NamingIdentifierNoKeyword, code.NameIsKeywordIdentifier, func(_ advisor.V1Stmt[ast.Node], f v1review.Finding) string {
			return fmt.Sprintf("Identifier [%s] is a keyword identifier and should be avoided.", f.Name)
		}, e.PosSearchOrStart))
	advisor.Register(storepb.Engine_MSSQL, storepb.SQLReviewRule_NAMING_TABLE,
		e.NamingRegex(
			mssqlv1review.NamingTable,
			code.NamingTableConventionMismatch,
			"naming_payload is required for this rule",
			func(f v1review.Finding, format string) string {
				return fmt.Sprintf("%s mismatches table naming convention, naming format should be %q", f.Name, format)
			},
			func(f v1review.Finding, maxLength int) string {
				return fmt.Sprintf("%s mismatches table naming convention, its length should be within %d characters", f.Name, maxLength)
			},
		))
	advisor.Register(storepb.Engine_MSSQL, storepb.SQLReviewRule_NAMING_TABLE_NO_KEYWORD,
		e.SimpleStmt(mssqlv1review.NamingTableNoKeyword, code.NameIsKeywordIdentifier, func(_ advisor.V1Stmt[ast.Node], f v1review.Finding) string {
			return fmt.Sprintf("Table name [%s] is a reserved keyword and should be avoided.", f.Name)
		}))
	advisor.Register(storepb.Engine_MSSQL, storepb.SQLReviewRule_SYSTEM_PROCEDURE_DISALLOW_CREATE,
		e.SimpleStmt(mssqlv1review.SystemProcedureDisallowCreate, code.DisallowCreateProcedure, func(advisor.V1Stmt[ast.Node], v1review.Finding) string {
			return "Creating or altering procedures is prohibited"
		}))
	advisor.Register(storepb.Engine_MSSQL, storepb.SQLReviewRule_STATEMENT_SELECT_NO_SELECT_ALL,
		e.SimpleStmt(mssqlv1review.StatementSelectNoSelectAll, code.StatementSelectAll, func(advisor.V1Stmt[ast.Node], v1review.Finding) string {
			return "Avoid using SELECT *."
		}))
	advisor.Register(storepb.Engine_MSSQL, storepb.SQLReviewRule_STATEMENT_DISALLOW_CROSS_DB_QUERIES,
		e.SimpleBatch(
			func(checkCtx advisor.Context, nodes []ast.Node) []v1review.Finding {
				return mssqlv1review.StatementDisallowCrossDBQueries(nodes, checkCtx.CurrentDatabase)
			},
			code.StatementDisallowCrossDBQueries,
			"",
			func(f v1review.Finding) string {
				return fmt.Sprintf("Cross database queries (target databse: '%s', current database: '%s') are prohibited", f.Name, f.Value)
			},
		))
	advisor.Register(storepb.Engine_MSSQL, storepb.SQLReviewRule_STATEMENT_DISALLOW_TRUNCATE,
		e.SimpleStmt(mssqlv1review.StatementDisallowTruncate, code.StatementDisallowTruncate, func(_ advisor.V1Stmt[ast.Node], f v1review.Finding) string {
			return fmt.Sprintf(
				`TRUNCATE TABLE %q is not allowed: TRUNCATE is minimally logged, bypasses DELETE triggers, and requires ALTER TABLE permission. Prior-backup treats this as DDL and does not produce row-level snapshots.`,
				f.Table,
			)
		}))
	advisor.Register(storepb.Engine_MSSQL, storepb.SQLReviewRule_STATEMENT_WHERE_DISALLOW_FUNCTIONS_AND_CALCULATIONS,
		e.WhereDisallowFuncAndCalc(func(checkCtx advisor.Context) func([]ast.Node) []v1review.Finding {
			schema := e.SchemaOf(checkCtx)
			return func(nodes []ast.Node) []v1review.Finding {
				return mssqlv1review.StatementWhereDisallowFunctionsAndCalculations(nodes, schema)
			}
		}, false))
	advisor.Register(storepb.Engine_MSSQL, storepb.SQLReviewRule_STATEMENT_WHERE_REQUIRE_SELECT,
		e.SimpleStmt(mssqlv1review.StatementWhereRequireSelect, code.StatementNoWhere, func(advisor.V1Stmt[ast.Node], v1review.Finding) string {
			return "WHERE clause is required for SELETE statement."
		}))
	advisor.Register(storepb.Engine_MSSQL, storepb.SQLReviewRule_STATEMENT_WHERE_REQUIRE_UPDATE_DELETE,
		e.SimpleStmt(mssqlv1review.StatementWhereRequireUpdateDelete, code.StatementNoWhere, func(_ advisor.V1Stmt[ast.Node], f v1review.Finding) string {
			return fmt.Sprintf("WHERE clause is required for %s statement.", f.Name)
		}))
	advisor.Register(storepb.Engine_MSSQL, storepb.SQLReviewRule_TABLE_DISALLOW_DDL,
		e.StringArrayStmt(mssqlv1review.TableDisallowDDL, "string_array_payload is required for table disallow DDL rule", code.TableDisallowDDL, v1review.TableDisallowed, func(_ advisor.V1Stmt[ast.Node], f v1review.Finding) string {
			return fmt.Sprintf("DDL is disallowed on table %s.", f.Name)
		}))
	advisor.Register(storepb.Engine_MSSQL, storepb.SQLReviewRule_TABLE_DISALLOW_DML,
		e.StringArrayStmt(mssqlv1review.TableDisallowDML, "string_array_payload is required for table disallow DML rule", code.TableDisallowDML, v1review.TableDisallowed, func(_ advisor.V1Stmt[ast.Node], f v1review.Finding) string {
			return fmt.Sprintf("DML is disallowed on table %s.", f.Name)
		}))
	advisor.Register(storepb.Engine_MSSQL, storepb.SQLReviewRule_TABLE_DROP_NAMING_CONVENTION,
		e.NamingRegex(
			mssqlv1review.TableDropNaming,
			code.TableDropNamingConventionMismatch,
			"naming_payload is required for table drop naming convention rule",
			func(f v1review.Finding, format string) string {
				return fmt.Sprintf("[%s] mismatches drop table naming convention, naming format should be %q", f.Name, format)
			},
			nil,
		))
	advisor.Register(storepb.Engine_MSSQL, storepb.SQLReviewRule_TABLE_NO_FOREIGN_KEY,
		e.SimpleBatch(
			func(_ advisor.Context, nodes []ast.Node) []v1review.Finding {
				return mssqlv1review.TableNoForeignKey(nodes)
			},
			code.TableHasFK,
			v1review.TableForeignKey,
			func(f v1review.Finding) string {
				return fmt.Sprintf("FOREIGN KEY is not allowed in the table %s.", f.Name)
			},
		))
	advisor.Register(storepb.Engine_MSSQL, storepb.SQLReviewRule_TABLE_REQUIRE_PK,
		e.SimpleBatch(
			func(_ advisor.Context, nodes []ast.Node) []v1review.Finding {
				return mssqlv1review.TableRequirePK(nodes)
			},
			code.TableNoPK,
			v1review.TablePrimaryKeyMissing,
			func(f v1review.Finding) string {
				return fmt.Sprintf("Table %s requires PRIMARY KEY.", f.Name)
			},
		))
}

func mssqlPriorBackup(e *advisor.V1Engine[ast.Node]) advisor.Func {
	return func(ctx context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
		if !checkCtx.EnablePriorBackup {
			return nil, nil
		}
		_, nodes, level, title, err := e.Prepare(checkCtx)
		if err != nil {
			return nil, err
		}
		facts := advisor.NewPriorBackupFacts(ctx, checkCtx, storepb.Engine_MSSQL)
		var adviceList []*storepb.Advice
		for _, f := range mssqlv1review.PriorBackupCheck(nodes, e.SchemaOf(checkCtx), facts) {
			var adviceCode code.Code
			var content string
			switch f.Code {
			case v1review.PriorBackupScriptTooLarge:
				adviceCode, content = code.BuiltinPriorBackupCheck, fmt.Sprintf("The size of statements in the sheet exceeds the limit of %s", f.Value)
			case v1review.PriorBackupDatabaseMissing:
				adviceCode, content = code.DatabaseNotExists, fmt.Sprintf("Need database %q to do prior backup but it does not exist", f.Name)
			case v1review.PriorBackupDDLInBatch:
				adviceCode, content = code.BuiltinPriorBackupCheck, "Prior backup cannot deal with mixed DDL and DML statements"
			case v1review.PriorBackupMixedDMLOnTable:
				adviceCode, content = code.BuiltinPriorBackupCheck, fmt.Sprintf("The statement type is not the same for all statements on the same table %q", f.Table)
			default:
				continue
			}
			adviceList = append(adviceList, &storepb.Advice{
				Status:        level,
				Title:         title,
				Content:       content,
				Code:          adviceCode.Int32(),
				StartPosition: &storepb.Position{Line: 1},
			})
		}
		return adviceList, nil
	}
}
