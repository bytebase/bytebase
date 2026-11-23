// Package snowflake implements the SQL advisor rules for Snowflake.
package snowflake

import (
	"fmt"

	"github.com/bytebase/omni/snowflake/ast"
	snowflakev1review "github.com/bytebase/omni/snowflake/v1review"
	"github.com/bytebase/omni/v1review"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	snowsqlparser "github.com/bytebase/bytebase/backend/plugin/parser/snowflake"
)

func init() {
	e := advisor.NewV1Engine(advisor.NodeExtractor(snowsqlparser.GetNode), nil, advisor.V1PosFindingOrFirstLine)

	compat := func(c code.Code, obj string, warning bool) advisor.V1CompatEntry {
		return advisor.V1CompatEntry{Code: c, Format: "Drop " + obj + " %s may cause incompatibility with the existing data and code", Warning: warning}
	}
	compatEntries := map[v1review.Code]advisor.V1CompatEntry{
		v1review.IncompatibleDropTable:            compat(code.CompatibilityDropTable, "table", false),
		v1review.IncompatibleDropMaybeNewTable:    compat(code.CompatibilityDropTable, "table", true),
		v1review.IncompatibleDropSchema:           compat(code.CompatibilityDropSchema, "schema", false),
		v1review.IncompatibleDropMaybeNewSchema:   compat(code.CompatibilityDropSchema, "schema", true),
		v1review.IncompatibleDropDatabase:         compat(code.CompatibilityDropDatabase, "database", false),
		v1review.IncompatibleDropMaybeNewDatabase: compat(code.CompatibilityDropDatabase, "database", true),
		v1review.IncompatibleDropColumn:           compat(code.CompatibilityDropColumn, "column", false),
	}

	advisor.RegisterRules(storepb.Engine_SNOWFLAKE, map[storepb.SQLReviewRule_Type]advisor.Advisor{
		storepb.SQLReviewRule_COLUMN_MAXIMUM_VARCHAR_LENGTH: e.NumberStmt(snowflakev1review.ColumnMaximumVarcharLength, true, code.VarcharLengthExceedsLimit,
			e.FmtLimit("The maximum varchar length is %d.")),
		storepb.SQLReviewRule_COLUMN_NO_NULL: e.SimpleStmt(snowflakev1review.ColumnNoNull, code.ColumnCannotNull,
			e.FmtName("Column %s is nullable, which is not allowed.")),
		storepb.SQLReviewRule_COLUMN_REQUIRED: e.StringArrayStmt(snowflakev1review.ColumnRequired, "string_array_payload is required for column required rule", code.NoRequiredColumn, "",
			e.FmtTableCol("Table %s missing required column %q")),
		storepb.SQLReviewRule_SCHEMA_BACKWARD_COMPATIBILITY: e.BackwardCompatibility(
			func(checkCtx advisor.Context, nodes []ast.Node) []v1review.Finding {
				return snowflakev1review.SchemaBackwardCompatibility(nodes, checkCtx.CurrentDatabase)
			},
			compatEntries,
			func(_ advisor.V1Stmt[ast.Node], f v1review.Finding, format string) string {
				return fmt.Sprintf(format, f.Name)
			},
		),
		storepb.SQLReviewRule_NAMING_IDENTIFIER_CASE: e.NamingIdentifierCase(snowflakev1review.NamingIdentifierCase, true),
		storepb.SQLReviewRule_NAMING_IDENTIFIER_NO_KEYWORD: e.SimpleStmt(snowflakev1review.NamingIdentifierNoKeyword, code.NameIsKeywordIdentifier,
			e.FmtName("Identifier %s is a keyword and should be avoided")),
		storepb.SQLReviewRule_NAMING_TABLE: e.NamingRule(
			snowflakev1review.NamingTable, code.NamingTableConventionMismatch, "", "table", true,
			advisor.DoubleQuotedName,
		),
		storepb.SQLReviewRule_NAMING_TABLE_NO_KEYWORD: e.SimpleStmt(snowflakev1review.NamingTableNoKeyword, code.NameIsKeywordIdentifier,
			e.FmtName("Table name %q is a keyword identifier and should be avoided."), e.PosNil),
		storepb.SQLReviewRule_STATEMENT_SELECT_NO_SELECT_ALL: e.SimpleConst(snowflakev1review.StatementSelectNoSelectAll, code.StatementSelectAll,
			"Avoid using SELECT *."),
		storepb.SQLReviewRule_TABLE_DROP_NAMING_CONVENTION: e.NamingRule(
			snowflakev1review.TableDropNaming, code.TableDropNamingConventionMismatch, "naming_payload is required for table drop naming convention rule", "drop table", false,
			func(f v1review.Finding) string { return fmt.Sprintf("%q", f.Name) },
		),
		storepb.SQLReviewRule_TABLE_NO_FOREIGN_KEY: e.SimpleBatchNodes(snowflakev1review.TableNoForeignKey, code.TableHasFK, v1review.TableForeignKey,
			advisor.FmtFindingName("FOREIGN KEY is not allowed in the table %s.")),
		storepb.SQLReviewRule_TABLE_REQUIRE_PK: e.SimpleBatchNodes(snowflakev1review.TableRequirePK, code.TableNoPK, v1review.TablePrimaryKeyMissing,
			advisor.FmtFindingName("Table %s requires PRIMARY KEY.")),
		storepb.SQLReviewRule_STATEMENT_WHERE_REQUIRE_SELECT: e.Stmt(snowflakev1review.StatementWhereRequireSelect,
			func(_ advisor.V1Stmt[ast.Node], _ v1review.Finding) (code.Code, string, bool) {
				return code.StatementNoWhere, "WHERE clause is required for SELECT statement.", true
			},
		),
		storepb.SQLReviewRule_STATEMENT_WHERE_REQUIRE_UPDATE_DELETE: e.SimpleStmt(snowflakev1review.StatementWhereRequireUpdateDelete, code.StatementNoWhere,
			e.FmtName("WHERE clause is required for %s statement.")),
	})
}
