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

	compatEntries := map[v1review.Code]advisor.V1CompatEntry{
		v1review.IncompatibleDropTable:            {Code: code.CompatibilityDropTable, Format: "Drop table %s may cause incompatibility with the existing data and code"},
		v1review.IncompatibleDropMaybeNewTable:    {Code: code.CompatibilityDropTable, Format: "Drop table %s may cause incompatibility with the existing data and code", Warning: true},
		v1review.IncompatibleDropSchema:           {Code: code.CompatibilityDropSchema, Format: "Drop schema %s may cause incompatibility with the existing data and code"},
		v1review.IncompatibleDropMaybeNewSchema:   {Code: code.CompatibilityDropSchema, Format: "Drop schema %s may cause incompatibility with the existing data and code", Warning: true},
		v1review.IncompatibleDropDatabase:         {Code: code.CompatibilityDropDatabase, Format: "Drop database %s may cause incompatibility with the existing data and code"},
		v1review.IncompatibleDropMaybeNewDatabase: {Code: code.CompatibilityDropDatabase, Format: "Drop database %s may cause incompatibility with the existing data and code", Warning: true},
		v1review.IncompatibleDropColumn:           {Code: code.CompatibilityDropColumn, Format: "Drop column %s may cause incompatibility with the existing data and code"},
	}

	advisor.RegisterRules(storepb.Engine_SNOWFLAKE, map[storepb.SQLReviewRule_Type]advisor.Advisor{
		storepb.SQLReviewRule_COLUMN_MAXIMUM_VARCHAR_LENGTH: e.NumberStmt(snowflakev1review.ColumnMaximumVarcharLength, true, code.VarcharLengthExceedsLimit, func(_ advisor.V1Stmt[ast.Node], _ v1review.Finding, maximum int) string {
			return fmt.Sprintf("The maximum varchar length is %d.", maximum)
		}),
		storepb.SQLReviewRule_COLUMN_NO_NULL: e.SimpleStmt(snowflakev1review.ColumnNoNull, code.ColumnCannotNull, func(_ advisor.V1Stmt[ast.Node], f v1review.Finding) string {
			return fmt.Sprintf("Column %s is nullable, which is not allowed.", f.Name)
		}),
		storepb.SQLReviewRule_COLUMN_REQUIRED: e.StringArrayStmt(snowflakev1review.ColumnRequired, "string_array_payload is required for column required rule", code.NoRequiredColumn, "", func(_ advisor.V1Stmt[ast.Node], f v1review.Finding) string {
			return fmt.Sprintf("Table %s missing required column %q", f.Table, f.Name)
		}),
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
		storepb.SQLReviewRule_NAMING_IDENTIFIER_NO_KEYWORD: e.SimpleStmt(snowflakev1review.NamingIdentifierNoKeyword, code.NameIsKeywordIdentifier, func(_ advisor.V1Stmt[ast.Node], f v1review.Finding) string {
			return fmt.Sprintf("Identifier %s is a keyword and should be avoided", f.Name)
		}),
		storepb.SQLReviewRule_NAMING_TABLE: e.NamingRule(
			snowflakev1review.NamingTable, code.NamingTableConventionMismatch, "naming_payload is required for this rule", "table", true,
			func(f v1review.Finding) string { return fmt.Sprintf(`"%s"`, f.Name) },
		),
		storepb.SQLReviewRule_NAMING_TABLE_NO_KEYWORD: e.SimpleStmt(snowflakev1review.NamingTableNoKeyword, code.NameIsKeywordIdentifier, func(_ advisor.V1Stmt[ast.Node], f v1review.Finding) string {
			return fmt.Sprintf("Table name %q is a keyword identifier and should be avoided.", f.Name)
		}, e.PosNil),
		storepb.SQLReviewRule_STATEMENT_SELECT_NO_SELECT_ALL: e.SimpleStmt(snowflakev1review.StatementSelectNoSelectAll, code.StatementSelectAll, func(advisor.V1Stmt[ast.Node], v1review.Finding) string {
			return "Avoid using SELECT *."
		}),
		storepb.SQLReviewRule_TABLE_DROP_NAMING_CONVENTION: e.NamingRule(
			snowflakev1review.TableDropNaming, code.TableDropNamingConventionMismatch, "naming_payload is required for table drop naming convention rule", "drop table", false,
			func(f v1review.Finding) string { return fmt.Sprintf("%q", f.Name) },
		),
		storepb.SQLReviewRule_TABLE_NO_FOREIGN_KEY: e.SimpleBatch(
			func(_ advisor.Context, nodes []ast.Node) []v1review.Finding {
				return snowflakev1review.TableNoForeignKey(nodes)
			},
			code.TableHasFK, v1review.TableForeignKey,
			func(f v1review.Finding) string {
				return fmt.Sprintf("FOREIGN KEY is not allowed in the table %s.", f.Name)
			},
		),
		storepb.SQLReviewRule_TABLE_REQUIRE_PK: e.SimpleBatch(
			func(_ advisor.Context, nodes []ast.Node) []v1review.Finding {
				return snowflakev1review.TableRequirePK(nodes)
			},
			code.TableNoPK, v1review.TablePrimaryKeyMissing,
			func(f v1review.Finding) string { return fmt.Sprintf("Table %s requires PRIMARY KEY.", f.Name) },
		),
		storepb.SQLReviewRule_STATEMENT_WHERE_REQUIRE_SELECT: e.Stmt(
			func(stmt advisor.V1Stmt[ast.Node]) []v1review.Finding {
				return snowflakev1review.StatementWhereRequireSelect([]ast.Node{stmt.Node}, stmt.Text)
			},
			func(_ advisor.V1Stmt[ast.Node], _ v1review.Finding) (code.Code, string, bool) {
				return code.StatementNoWhere, "WHERE clause is required for SELECT statement.", true
			},
		),
		storepb.SQLReviewRule_STATEMENT_WHERE_REQUIRE_UPDATE_DELETE: e.SimpleStmt(snowflakev1review.StatementWhereRequireUpdateDelete, code.StatementNoWhere, func(_ advisor.V1Stmt[ast.Node], f v1review.Finding) string {
			return fmt.Sprintf("WHERE clause is required for %s statement.", f.Name)
		}),
	})
}
