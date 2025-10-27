package pgantlr

import (
	"context"

	"github.com/antlr4-go/antlr/v4"

	parser "github.com/bytebase/parser/postgresql"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
)

var (
	_ advisor.Advisor = (*StatementCreateSpecifySchema)(nil)
)

func init() {
	advisor.Register(storepb.Engine_POSTGRES, advisor.SchemaRuleStatementCreateSpecifySchema, &StatementCreateSpecifySchema{})
}

type StatementCreateSpecifySchema struct {
}

func (*StatementCreateSpecifySchema) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	tree, err := getANTLRTree(checkCtx)
	if err != nil {
		return nil, err
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	checker := &statementCreateSpecifySchemaChecker{
		BasePostgreSQLParserListener: &parser.BasePostgreSQLParserListener{},
		level:                        level,
		title:                        string(checkCtx.Rule.Type),
	}

	antlr.ParseTreeWalkerDefault.Walk(checker, tree.Tree)

	return checker.adviceList, nil
}

type statementCreateSpecifySchemaChecker struct {
	*parser.BasePostgreSQLParserListener

	adviceList []*storepb.Advice
	level      storepb.Advice_Status
	title      string
}

// EnterCreatestmt handles CREATE TABLE
func (c *statementCreateSpecifySchemaChecker) EnterCreatestmt(ctx *parser.CreatestmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}

	allQualifiedNames := ctx.AllQualified_name()
	if len(allQualifiedNames) > 0 {
		schemaName := extractSchemaName(allQualifiedNames[0])
		if schemaName == "" {
			c.adviceList = append(c.adviceList, &storepb.Advice{
				Status:  c.level,
				Code:    advisor.StatementCreateWithoutSchemaName.Int32(),
				Title:   c.title,
				Content: "Table schema should be specified.",
				StartPosition: &storepb.Position{
					Line:   int32(ctx.GetStart().GetLine()),
					Column: 0,
				},
			})
		}
	}
}
