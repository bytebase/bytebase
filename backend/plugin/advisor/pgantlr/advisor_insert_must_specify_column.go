package pgantlr

import (
	"context"
	"fmt"

	"github.com/antlr4-go/antlr/v4"

	parser "github.com/bytebase/parser/postgresql"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
)

var (
	_ advisor.Advisor = (*InsertMustSpecifyColumnAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_POSTGRES, advisor.SchemaRuleStatementInsertMustSpecifyColumn, &InsertMustSpecifyColumnAdvisor{})
}

// InsertMustSpecifyColumnAdvisor is the advisor checking for to enforce column specified.
type InsertMustSpecifyColumnAdvisor struct {
}

// Check checks for to enforce column specified.
func (*InsertMustSpecifyColumnAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	tree, err := getANTLRTree(checkCtx)
	if err != nil {
		return nil, err
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	checker := &insertMustSpecifyColumnChecker{
		BasePostgreSQLParserListener: &parser.BasePostgreSQLParserListener{},
		level:                        level,
		title:                        string(checkCtx.Rule.Type),
		statementsText:               checkCtx.Statements,
	}

	antlr.ParseTreeWalkerDefault.Walk(checker, tree.Tree)

	return checker.adviceList, nil
}

type insertMustSpecifyColumnChecker struct {
	*parser.BasePostgreSQLParserListener

	adviceList     []*storepb.Advice
	level          storepb.Advice_Status
	title          string
	statementsText string
}

func (c *insertMustSpecifyColumnChecker) EnterInsertstmt(ctx *parser.InsertstmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}

	// Check if column list is specified
	// In PostgreSQL, INSERT has an optional insert_column_list
	// If insert_column_list is not specified or empty, we should report it
	if ctx.Insert_rest() == nil {
		return
	}

	// Check if there's an insert_column_list
	// Insert_column_list exists, which means columns are specified
	hasColumnList := ctx.Insert_rest().Insert_column_list() != nil

	if !hasColumnList {
		// Extract the statement text from the original statements
		stmtText := extractStatementText(c.statementsText, ctx.GetStart().GetLine(), ctx.GetStop().GetLine())

		c.adviceList = append(c.adviceList, &storepb.Advice{
			Status:  c.level,
			Code:    advisor.InsertNotSpecifyColumn.Int32(),
			Title:   c.title,
			Content: fmt.Sprintf("The INSERT statement must specify columns but \"%s\" does not", stmtText),
			StartPosition: &storepb.Position{
				Line:   int32(ctx.GetStart().GetLine()),
				Column: 0,
			},
		})
	}
}
