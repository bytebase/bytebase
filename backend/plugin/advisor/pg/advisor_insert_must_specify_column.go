package pg

import (
	"context"
	"fmt"

	"github.com/antlr4-go/antlr/v4"

	parser "github.com/bytebase/parser/postgresql"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
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

	rule := &insertMustSpecifyColumnRule{
		BaseRule: BaseRule{
			level: level,
			title: string(checkCtx.Rule.Type),
		},
		statementsText: checkCtx.Statements,
	}

	checker := NewGenericChecker([]Rule{rule})
	antlr.ParseTreeWalkerDefault.Walk(checker, tree.Tree)

	return checker.GetAdviceList(), nil
}

type insertMustSpecifyColumnRule struct {
	BaseRule

	statementsText string
}

func (*insertMustSpecifyColumnRule) Name() string {
	return "insert_must_specify_column"
}

func (r *insertMustSpecifyColumnRule) OnEnter(ctx antlr.ParserRuleContext, nodeType string) error {
	switch nodeType {
	case "Insertstmt":
		r.handleInsertstmt(ctx)
	default:
		// Do nothing for other node types
	}
	return nil
}

func (*insertMustSpecifyColumnRule) OnExit(_ antlr.ParserRuleContext, _ string) error {
	return nil
}

func (r *insertMustSpecifyColumnRule) handleInsertstmt(ctx antlr.ParserRuleContext) {
	insertstmtCtx, ok := ctx.(*parser.InsertstmtContext)
	if !ok {
		return
	}

	if !isTopLevel(insertstmtCtx.GetParent()) {
		return
	}

	// Check if column list is specified
	// In PostgreSQL, INSERT has an optional insert_column_list
	// If insert_column_list is not specified or empty, we should report it
	if insertstmtCtx.Insert_rest() == nil {
		return
	}

	// Check if there's an insert_column_list
	// Insert_column_list exists, which means columns are specified
	hasColumnList := insertstmtCtx.Insert_rest().Insert_column_list() != nil

	if !hasColumnList {
		// Extract the statement text from the original statements
		stmtText := extractStatementText(r.statementsText, insertstmtCtx.GetStart().GetLine(), insertstmtCtx.GetStop().GetLine())

		r.AddAdvice(&storepb.Advice{
			Status:  r.level,
			Code:    code.InsertNotSpecifyColumn.Int32(),
			Title:   r.title,
			Content: fmt.Sprintf("The INSERT statement must specify columns but \"%s\" does not", stmtText),
			StartPosition: &storepb.Position{
				Line:   int32(insertstmtCtx.GetStart().GetLine()),
				Column: 0,
			},
		})
	}
}
