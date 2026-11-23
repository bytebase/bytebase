// Package oracle implements the SQL advisor rules for Oracle.
package oracle

import (
	"context"
	"fmt"
	"strings"

	"github.com/bytebase/omni/oracle/ast"
	oraclev1review "github.com/bytebase/omni/oracle/v1review"
	"github.com/bytebase/omni/v1review"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	plsqlparser "github.com/bytebase/bytebase/backend/plugin/parser/plsql"
)

func init() {
	e := advisor.NewV1Engine(advisor.NodeExtractor(plsqlparser.GetNode), nil, advisor.V1PosFindingOrFirstLine)

	advisor.RegisterRules(storepb.Engine_ORACLE, map[storepb.SQLReviewRule_Type]advisor.Advisor{
		storepb.SQLReviewRule_BUILTIN_PRIOR_BACKUP_CHECK: e.PriorBackup(advisor.V1PriorBackupConfig[ast.Node]{
			Engine: storepb.Engine_ORACLE,
			Check: func(checkCtx advisor.Context, _ []advisor.V1Stmt[ast.Node], nodes []ast.Node, facts v1review.PriorBackupFacts) ([]v1review.Finding, error) {
				return oraclev1review.PriorBackupCheck(nodes, e.SchemaOf(checkCtx), facts), nil
			},
			Render: func(_ []advisor.V1Stmt[ast.Node], _ []ast.Node, f v1review.Finding) (code.Code, string, *storepb.Position, bool) {
				if f.Code == v1review.PriorBackupUncopyableColumn {
					return code.BuiltinPriorBackupCheck, fmt.Sprintf("Prior backup cannot copy table %q.%q: column(s) %s use the LONG/LONG RAW type, which CREATE TABLE AS SELECT does not support (ORA-00997). Disable prior backup for this issue or migrate the column type", f.Schema, f.Table, f.Value), nil, true
				}
				return 0, "", nil, false
			},
		}),
		storepb.SQLReviewRule_COLUMN_ADD_NOT_NULL_REQUIRE_DEFAULT: e.SimpleStmt(oraclev1review.ColumnAddNotNullRequireDefault, code.NotNullColumnWithNoDefault,
			e.FmtName("Adding not null column %q requires default.")),
		storepb.SQLReviewRule_COLUMN_COMMENT: e.ColumnComment(
			func(checkCtx advisor.Context, nodes []ast.Node, required bool, maxLength int) []v1review.Finding {
				return oraclev1review.ColumnComment(nodes, checkCtx.CurrentDatabase, required, maxLength)
			},
			-1, true,
			func(f v1review.Finding) string { return normalizeIdentifierName(f.Table + "." + f.Name) },
		),
		storepb.SQLReviewRule_COLUMN_MAXIMUM_CHARACTER_LENGTH: e.NumberStmt(oraclev1review.ColumnMaximumCharacterLength, true, code.CharLengthExceedsLimit,
			e.FmtLimit("The maximum character length is %d.")),
		storepb.SQLReviewRule_COLUMN_MAXIMUM_VARCHAR_LENGTH: e.NumberStmt(oraclev1review.ColumnMaximumVarcharLength, true, code.VarcharLengthExceedsLimit,
			e.FmtLimit("The maximum varchar length is %d.")),
		storepb.SQLReviewRule_COLUMN_NO_NULL: e.SimpleBatchDB(oraclev1review.ColumnNoNull, code.ColumnCannotNull, "",
			advisor.FmtFindingName("Column %q is nullable, which is not allowed.")),
		storepb.SQLReviewRule_COLUMN_REQUIRED: e.ColumnRequired(advisor.V1ColumnRequiredConfig[ast.Node]{
			Fn:       oraclev1review.ColumnRequired,
			QuoteCol: true,
		}),
		storepb.SQLReviewRule_COLUMN_REQUIRE_DEFAULT: e.SimpleBatchDB(oraclev1review.ColumnRequireDefault, code.NoDefault, "",
			advisor.FmtFindingName("Column %q doesn't have default value")),
		storepb.SQLReviewRule_COLUMN_TYPE_DISALLOW_LIST: e.StringArrayStmt(oraclev1review.ColumnTypeDisallowList, "string_array_payload is required for column type disallow list rule", code.DisabledColumnType, "", func(_ advisor.V1Stmt[ast.Node], f v1review.Finding) string {
			return fmt.Sprintf("Disallow column type %s but column \"%s\" is", f.Value, f.Name)
		}),
		storepb.SQLReviewRule_INDEX_KEY_NUMBER_LIMIT: e.NumberStmt(oraclev1review.IndexKeyNumberLimit, true, code.IndexKeyNumberExceedsLimit,
			e.FmtLimit("Index key number should be less than or equal to %d")),
		storepb.SQLReviewRule_STATEMENT_INSERT_MUST_SPECIFY_COLUMN: e.SimpleConst(oraclev1review.StatementInsertMustSpecifyColumn, code.InsertNotSpecifyColumn,
			"INSERT statement should specify column name."),
		storepb.SQLReviewRule_NAMING_IDENTIFIER_CASE: e.NamingIdentifierCase(oraclev1review.NamingIdentifierCase, false),
		storepb.SQLReviewRule_NAMING_IDENTIFIER_NO_KEYWORD: e.SimpleStmt(oraclev1review.NamingIdentifierNoKeyword, code.NameIsKeywordIdentifier,
			e.FmtName("Identifier %q is a keyword and should be avoided")),
		storepb.SQLReviewRule_NAMING_TABLE: e.NamingRule(
			oraclev1review.NamingTable, code.NamingTableConventionMismatch, "", "table", true,
			advisor.DoubleQuotedName,
		),
		storepb.SQLReviewRule_NAMING_TABLE_NO_KEYWORD: e.SimpleStmt(oraclev1review.NamingTableNoKeyword, code.NameIsKeywordIdentifier,
			e.FmtName("Table name %q is a keyword identifier and should be avoided.")),
		storepb.SQLReviewRule_STATEMENT_SELECT_NO_SELECT_ALL: e.SimpleConst(oraclev1review.StatementSelectNoSelectAll, code.StatementSelectAll,
			"Avoid using SELECT *."),
		storepb.SQLReviewRule_STATEMENT_DISALLOW_TRUNCATE: e.Stmt(oraclev1review.StatementDisallowTruncate,
			func(_ advisor.V1Stmt[ast.Node], f v1review.Finding) (code.Code, string, bool) {
				switch f.Code {
				case v1review.Truncate:
					return code.StatementDisallowTruncate, fmt.Sprintf(
						`TRUNCATE TABLE %q is not allowed: it issues an implicit COMMIT and cannot be rolled back. Any prior uncommitted work in the same transaction is also committed. Prior-backup treats this as DDL and does not produce row-level snapshots.`,
						f.Table,
					), true
				case v1review.TruncatePartition:
					return code.StatementDisallowTruncate, fmt.Sprintf(
						`ALTER TABLE %q TRUNCATE %s %q is not allowed: partition truncate shares the implicit-commit gap of TRUNCATE TABLE on Oracle. Prior-backup treats this as DDL and does not produce row-level snapshots.`,
						f.Table, f.Value, f.Name,
					), true
				default:
					return 0, "", false
				}
			},
		),
		storepb.SQLReviewRule_STATEMENT_DML_DRY_RUN: e.DMLDryRun(advisor.V1DMLDryRunConfig[ast.Node]{
			Probes: oraclev1review.StatementDMLDryRunProbes,
			Judge:  oraclev1review.StatementDMLDryRun,
			Probe: func(ctx context.Context, prober db.Prober, stmt advisor.V1Stmt[ast.Node], _ []string) v1review.Probe {
				// BEGIN opens a PL/SQL block in Oracle, so the dry run runs alone.
				return advisor.ExplainProbe(ctx, prober, db.ProbeContext{Autocommit: true}, fmt.Sprintf("EXPLAIN PLAN FOR %s", stmt.Text))
			},
			FormatContent: func(stmt advisor.V1Stmt[ast.Node], f v1review.Finding) string {
				return fmt.Sprintf("Failed to dry run statement at line %d: %v", stmt.BaseLine+1, f.Value)
			},
			Pos: func(stmt advisor.V1Stmt[ast.Node], _ v1review.Finding) *storepb.Position {
				return base.LinePosition(stmt.BaseLine + 1)
			},
		}),
		storepb.SQLReviewRule_STATEMENT_WHERE_DISALLOW_FUNCTIONS_AND_CALCULATIONS: e.WhereDisallowFuncAndCalc(func(checkCtx advisor.Context) func([]ast.Node) []v1review.Finding {
			// Oracle sync stores only the connected owner's objects, under the anonymous schema "".
			schema := e.SchemaOf(checkCtx)
			currentDB := checkCtx.CurrentDatabase
			return func(nodes []ast.Node) []v1review.Finding {
				return oraclev1review.StatementWhereDisallowFunctionsAndCalculations(nodes, schema, "", currentDB)
			}
		}, true),
		storepb.SQLReviewRule_TABLE_COMMENT: e.TableComment(
			func(checkCtx advisor.Context, nodes []ast.Node, cfg *v1review.TableCommentConfig) []v1review.Finding {
				return oraclev1review.TableComment(nodes, checkCtx.CurrentDatabase, cfg)
			},
			func(f v1review.Finding) string { return normalizeIdentifierName(f.Name) },
		),
		storepb.SQLReviewRule_TABLE_NO_FOREIGN_KEY: e.SimpleBatchDB(oraclev1review.TableNoForeignKey, code.TableHasFK, v1review.TableForeignKey,
			func(f v1review.Finding) string {
				return fmt.Sprintf("Foreign key is not allowed in the table %s.", normalizeIdentifierName(f.Name))
			}),
		storepb.SQLReviewRule_TABLE_REQUIRE_PK: e.SimpleBatchDB(oraclev1review.TableRequirePK, code.TableNoPK, v1review.TablePrimaryKeyMissing,
			func(f v1review.Finding) string {
				return fmt.Sprintf("Table %s requires PRIMARY KEY.", normalizeIdentifierName(f.Name))
			}),
		storepb.SQLReviewRule_STATEMENT_WHERE_NO_LEADING_WILDCARD_LIKE: e.SimpleConst(oraclev1review.StatementWhereNoLeadingWildcardLike, code.StatementLeadingWildcardLike,
			"Avoid using leading wildcard LIKE."),
		storepb.SQLReviewRule_STATEMENT_WHERE_REQUIRE_SELECT: e.SimpleConst(oraclev1review.StatementWhereRequireSelect, code.StatementNoWhere,
			"WHERE clause is required for SELECT statement."),
		storepb.SQLReviewRule_STATEMENT_WHERE_REQUIRE_UPDATE_DELETE: e.SimpleStmt(oraclev1review.StatementWhereRequireUpdateDelete, code.StatementNoWhere,
			e.FmtName("WHERE clause is required for %s statement.")),
	})
}

// normalizeIdentifierName normalizes the identifier name to the format of "schema"."table"."column".
func normalizeIdentifierName(name string) string {
	list := strings.Split(name, ".")
	result := make([]string, len(list))
	for i, item := range list {
		result[i] = fmt.Sprintf("%q", item)
	}
	return strings.Join(result, ".")
}
