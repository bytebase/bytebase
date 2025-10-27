package pgantlr

import (
	"context"
	"fmt"
	"regexp"

	"github.com/antlr4-go/antlr/v4"

	parser "github.com/bytebase/parser/postgresql"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/parser/pg"
)

var (
	_ advisor.Advisor = (*TableDropNamingConventionAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_POSTGRES, advisor.SchemaRuleTableDropNamingConvention, &TableDropNamingConventionAdvisor{})
}

// TableDropNamingConventionAdvisor is the advisor checking for table drop with naming convention.
type TableDropNamingConventionAdvisor struct {
}

// Check checks for table drop with naming convention.
func (*TableDropNamingConventionAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	tree, err := getANTLRTree(checkCtx)
	if err != nil {
		return nil, err
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	format, _, err := advisor.UnmarshalNamingRulePayloadAsRegexp(checkCtx.Rule.Payload)
	if err != nil {
		return nil, err
	}

	checker := &tableDropNamingConventionChecker{
		BasePostgreSQLParserListener: &parser.BasePostgreSQLParserListener{},
		level:                        level,
		title:                        string(checkCtx.Rule.Type),
		format:                       format,
	}

	antlr.ParseTreeWalkerDefault.Walk(checker, tree.Tree)

	return checker.adviceList, nil
}

type tableDropNamingConventionChecker struct {
	*parser.BasePostgreSQLParserListener

	adviceList []*storepb.Advice
	level      storepb.Advice_Status
	title      string
	format     *regexp.Regexp
}

// EnterDropstmt handles DROP TABLE statements
func (c *tableDropNamingConventionChecker) EnterDropstmt(ctx *parser.DropstmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}

	// Check if this is a DROP TABLE statement
	if ctx.Object_type_any_name() == nil || ctx.Object_type_any_name().TABLE() == nil {
		return
	}

	// Check all tables being dropped
	if ctx.Any_name_list() != nil {
		allNames := ctx.Any_name_list().AllAny_name()
		for _, nameCtx := range allNames {
			tableName := c.extractTableNameFromAnyName(nameCtx)
			if tableName != "" && !c.format.MatchString(tableName) {
				c.adviceList = append(c.adviceList, &storepb.Advice{
					Status:  c.level,
					Code:    advisor.TableDropNamingConventionMismatch.Int32(),
					Title:   c.title,
					Content: fmt.Sprintf("`%s` mismatches drop table naming convention, naming format should be %q", tableName, c.format),
					StartPosition: &storepb.Position{
						Line:   int32(ctx.GetStart().GetLine()),
						Column: 0,
					},
				})
			}
		}
	}
}

// extractTableNameFromAnyName extracts the table name from Any_name context.
// For schema.table, returns "table". For just "table", returns "table".
func (*tableDropNamingConventionChecker) extractTableNameFromAnyName(ctx parser.IAny_nameContext) string {
	parts := pg.NormalizePostgreSQLAnyName(ctx)
	if len(parts) == 0 {
		return ""
	}
	return parts[len(parts)-1]
}
