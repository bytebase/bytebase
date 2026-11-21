// Package snowflake implements the SQL advisor rules for Snowflake.
package snowflake

import (
	"context"
	"fmt"
	"strings"

	"github.com/bytebase/omni/snowflake/ast"
	snowflakev1review "github.com/bytebase/omni/snowflake/v1review"
	"github.com/bytebase/omni/v1review"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	snowsqlparser "github.com/bytebase/bytebase/backend/plugin/parser/snowflake"
)

func init() {
	extract := func(checkCtx advisor.Context) ([]advisor.V1Stmt[ast.Node], error) {
		return advisor.ExtractNodes(checkCtx.ParsedStatements, snowsqlparser.GetNode), nil
	}
	byteOffsetLine := func(sql string, offset int) int32 {
		if offset <= 0 {
			return 1
		}
		if offset > len(sql) {
			offset = len(sql)
		}
		return int32(1 + strings.Count(sql[:offset], "\n"))
	}
	e := advisor.NewV1Engine(extract, byteOffsetLine, advisor.V1PosFindingOrFirstLine)

	compatEntries := map[v1review.Code]advisor.V1CompatEntry{
		v1review.IncompatibleDropTable:            {Code: code.CompatibilityDropTable, Format: "Drop table %s may cause incompatibility with the existing data and code"},
		v1review.IncompatibleDropMaybeNewTable:    {Code: code.CompatibilityDropTable, Format: "Drop table %s may cause incompatibility with the existing data and code", Warning: true},
		v1review.IncompatibleDropSchema:           {Code: code.CompatibilityDropSchema, Format: "Drop schema %s may cause incompatibility with the existing data and code"},
		v1review.IncompatibleDropMaybeNewSchema:   {Code: code.CompatibilityDropSchema, Format: "Drop schema %s may cause incompatibility with the existing data and code", Warning: true},
		v1review.IncompatibleDropDatabase:         {Code: code.CompatibilityDropDatabase, Format: "Drop database %s may cause incompatibility with the existing data and code"},
		v1review.IncompatibleDropMaybeNewDatabase: {Code: code.CompatibilityDropDatabase, Format: "Drop database %s may cause incompatibility with the existing data and code", Warning: true},
		v1review.IncompatibleDropColumn:           {Code: code.CompatibilityDropColumn, Format: "Drop column %s may cause incompatibility with the existing data and code"},
	}

	advisor.Register(storepb.Engine_SNOWFLAKE, storepb.SQLReviewRule_COLUMN_MAXIMUM_VARCHAR_LENGTH,
		e.NumberStmt(snowflakev1review.ColumnMaximumVarcharLength, true, code.VarcharLengthExceedsLimit, func(_ advisor.V1Stmt[ast.Node], _ v1review.Finding, maximum int) string {
			return fmt.Sprintf("The maximum varchar length is %d.", maximum)
		}))
	advisor.Register(storepb.Engine_SNOWFLAKE, storepb.SQLReviewRule_COLUMN_NO_NULL,
		e.SimpleStmt(snowflakev1review.ColumnNoNull, code.ColumnCannotNull, func(_ advisor.V1Stmt[ast.Node], f v1review.Finding) string {
			return fmt.Sprintf("Column %s is nullable, which is not allowed.", f.Name)
		}))
	advisor.Register(storepb.Engine_SNOWFLAKE, storepb.SQLReviewRule_COLUMN_REQUIRED,
		e.StringArrayStmt(snowflakev1review.ColumnRequired, "string_array_payload is required for column required rule", code.NoRequiredColumn, "", func(_ advisor.V1Stmt[ast.Node], f v1review.Finding) string {
			return fmt.Sprintf("Table %s missing required column %q", f.Table, f.Name)
		}))
	advisor.Register(storepb.Engine_SNOWFLAKE, storepb.SQLReviewRule_SCHEMA_BACKWARD_COMPATIBILITY,
		e.BackwardCompatibility(
			func(checkCtx advisor.Context, nodes []ast.Node) []v1review.Finding {
				return snowflakev1review.SchemaBackwardCompatibility(nodes, checkCtx.CurrentDatabase)
			},
			compatEntries,
			func(_ advisor.V1Stmt[ast.Node], f v1review.Finding, format string) string {
				return fmt.Sprintf(format, f.Name)
			},
		))
	advisor.Register(storepb.Engine_SNOWFLAKE, storepb.SQLReviewRule_NAMING_IDENTIFIER_CASE,
		advisor.Func(func(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
			stmts, nodes, status, title, err := e.Prepare(checkCtx)
			if err != nil {
				return nil, err
			}
			// The Snowflake rule requires upper case whatever the payload says, and
			// accepts a missing payload: omni ignores this config.
			cfg := v1review.CaseConfig{Upper: checkCtx.Rule.GetNamingCasePayload().GetUpper()}
			var adviceList []*storepb.Advice
			for i, stmt := range stmts {
				for _, f := range snowflakev1review.NamingIdentifierCase(nodes[i:i+1:i+1], cfg) {
					adviceList = append(adviceList, &storepb.Advice{
						Status:        status,
						Code:          code.NamingCaseMismatch.Int32(),
						Title:         title,
						Content:       fmt.Sprintf("Identifier %q should be upper case", f.Name),
						StartPosition: e.PosFindingOrFirstLine(stmt, f),
					})
				}
			}
			return adviceList, nil
		}))
	advisor.Register(storepb.Engine_SNOWFLAKE, storepb.SQLReviewRule_NAMING_IDENTIFIER_NO_KEYWORD,
		e.SimpleStmt(snowflakev1review.NamingIdentifierNoKeyword, code.NameIsKeywordIdentifier, func(_ advisor.V1Stmt[ast.Node], f v1review.Finding) string {
			return fmt.Sprintf("Identifier %s is a keyword and should be avoided", f.Name)
		}))
	advisor.Register(storepb.Engine_SNOWFLAKE, storepb.SQLReviewRule_NAMING_TABLE,
		e.NamingRegex(
			snowflakev1review.NamingTable,
			code.NamingTableConventionMismatch,
			"naming_payload is required for this rule",
			func(f v1review.Finding, format string) string {
				return fmt.Sprintf(`"%s" mismatches table naming convention, naming format should be %q`, f.Name, format)
			},
			func(f v1review.Finding, maxLength int) string {
				return fmt.Sprintf("\"%s\" mismatches table naming convention, its length should be within %d characters", f.Name, maxLength)
			},
		))
	advisor.Register(storepb.Engine_SNOWFLAKE, storepb.SQLReviewRule_NAMING_TABLE_NO_KEYWORD,
		e.SimpleStmt(snowflakev1review.NamingTableNoKeyword, code.NameIsKeywordIdentifier, func(_ advisor.V1Stmt[ast.Node], f v1review.Finding) string {
			return fmt.Sprintf("Table name %q is a keyword identifier and should be avoided.", f.Name)
		}, e.PosNil))
	advisor.Register(storepb.Engine_SNOWFLAKE, storepb.SQLReviewRule_STATEMENT_SELECT_NO_SELECT_ALL,
		e.SimpleStmt(snowflakev1review.StatementSelectNoSelectAll, code.StatementSelectAll, func(advisor.V1Stmt[ast.Node], v1review.Finding) string {
			return "Avoid using SELECT *."
		}))
	advisor.Register(storepb.Engine_SNOWFLAKE, storepb.SQLReviewRule_TABLE_DROP_NAMING_CONVENTION,
		e.NamingRegex(
			snowflakev1review.TableDropNaming,
			code.TableDropNamingConventionMismatch,
			"naming_payload is required for table drop naming convention rule",
			func(f v1review.Finding, format string) string {
				return fmt.Sprintf("%q mismatches drop table naming convention, naming format should be %q", f.Name, format)
			},
			nil,
		))
	advisor.Register(storepb.Engine_SNOWFLAKE, storepb.SQLReviewRule_TABLE_NO_FOREIGN_KEY,
		e.SimpleBatch(
			func(_ advisor.Context, nodes []ast.Node) []v1review.Finding {
				return snowflakev1review.TableNoForeignKey(nodes)
			},
			code.TableHasFK,
			v1review.TableForeignKey,
			func(f v1review.Finding) string {
				return fmt.Sprintf("FOREIGN KEY is not allowed in the table %s.", f.Name)
			},
		))
	advisor.Register(storepb.Engine_SNOWFLAKE, storepb.SQLReviewRule_TABLE_REQUIRE_PK,
		e.SimpleBatch(
			func(_ advisor.Context, nodes []ast.Node) []v1review.Finding {
				return snowflakev1review.TableRequirePK(nodes)
			},
			code.TableNoPK,
			v1review.TablePrimaryKeyMissing,
			func(f v1review.Finding) string {
				return fmt.Sprintf("Table %s requires PRIMARY KEY.", f.Name)
			},
		))
	advisor.Register(storepb.Engine_SNOWFLAKE, storepb.SQLReviewRule_STATEMENT_WHERE_REQUIRE_SELECT,
		e.Stmt(
			func(stmt advisor.V1Stmt[ast.Node]) []v1review.Finding {
				return snowflakev1review.StatementWhereRequireSelect([]ast.Node{stmt.Node}, stmt.Text)
			},
			func(_ advisor.V1Stmt[ast.Node], _ v1review.Finding) (code.Code, string, bool) {
				return code.StatementNoWhere, "WHERE clause is required for SELECT statement.", true
			},
		))
	advisor.Register(storepb.Engine_SNOWFLAKE, storepb.SQLReviewRule_STATEMENT_WHERE_REQUIRE_UPDATE_DELETE,
		e.SimpleStmt(snowflakev1review.StatementWhereRequireUpdateDelete, code.StatementNoWhere, func(_ advisor.V1Stmt[ast.Node], f v1review.Finding) string {
			return fmt.Sprintf("WHERE clause is required for %s statement.", f.Name)
		}))
}
