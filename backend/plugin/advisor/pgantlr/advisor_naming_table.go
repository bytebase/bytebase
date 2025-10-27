package pgantlr

import (
	"context"
	"fmt"
	"regexp"

	"github.com/antlr4-go/antlr/v4"

	parser "github.com/bytebase/parser/postgresql"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	pgparser "github.com/bytebase/bytebase/backend/plugin/parser/pg"
)

var (
	_ advisor.Advisor = (*NamingTableConventionAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_POSTGRES, advisor.SchemaRuleTableNaming, &NamingTableConventionAdvisor{})
}

// NamingTableConventionAdvisor is the advisor checking for table naming convention.
type NamingTableConventionAdvisor struct {
}

// Check checks for table naming convention.
func (*NamingTableConventionAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	tree, err := getANTLRTree(checkCtx)
	if err != nil {
		return nil, err
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	format, maxLength, err := advisor.UnmarshalNamingRulePayloadAsRegexp(checkCtx.Rule.Payload)
	if err != nil {
		return nil, err
	}

	checker := &namingTableConventionChecker{
		BasePostgreSQLParserListener: &parser.BasePostgreSQLParserListener{},
		level:                        level,
		title:                        string(checkCtx.Rule.Type),
		format:                       format,
		maxLength:                    maxLength,
	}

	antlr.ParseTreeWalkerDefault.Walk(checker, tree.Tree)

	return checker.adviceList, nil
}

type namingTableConventionChecker struct {
	*parser.BasePostgreSQLParserListener

	adviceList []*storepb.Advice
	level      storepb.Advice_Status
	title      string
	format     *regexp.Regexp
	maxLength  int
}

// EnterCreatestmt handles CREATE TABLE
func (c *namingTableConventionChecker) EnterCreatestmt(ctx *parser.CreatestmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}

	allQualifiedNames := ctx.AllQualified_name()
	if len(allQualifiedNames) > 0 {
		tableName := extractTableName(allQualifiedNames[0])
		c.checkTableName(tableName, ctx)
	}
}

// EnterRenamestmt handles ALTER TABLE RENAME TO
func (c *namingTableConventionChecker) EnterRenamestmt(ctx *parser.RenamestmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}

	// Check for ALTER TABLE ... RENAME TO new_name
	if ctx.TABLE() != nil && ctx.TO() != nil {
		allNames := ctx.AllName()
		if len(allNames) > 0 {
			// The new table name is the last Name() in RENAME TO new_name
			newTableName := pgparser.NormalizePostgreSQLName(allNames[len(allNames)-1])
			c.checkTableName(newTableName, ctx)
		}
	}
}

func (c *namingTableConventionChecker) checkTableName(tableName string, ctx antlr.ParserRuleContext) {
	if !c.format.MatchString(tableName) {
		c.adviceList = append(c.adviceList, &storepb.Advice{
			Status:  c.level,
			Code:    advisor.NamingTableConventionMismatch.Int32(),
			Title:   c.title,
			Content: fmt.Sprintf(`"%s" mismatches table naming convention, naming format should be %q`, tableName, c.format),
			StartPosition: &storepb.Position{
				Line:   int32(ctx.GetStart().GetLine()),
				Column: 0,
			},
		})
	}
	if c.maxLength > 0 && len(tableName) > c.maxLength {
		c.adviceList = append(c.adviceList, &storepb.Advice{
			Status:  c.level,
			Code:    advisor.NamingTableConventionMismatch.Int32(),
			Title:   c.title,
			Content: fmt.Sprintf("\"%s\" mismatches table naming convention, its length should be within %d characters", tableName, c.maxLength),
			StartPosition: &storepb.Position{
				Line:   int32(ctx.GetStart().GetLine()),
				Column: 0,
			},
		})
	}
}
