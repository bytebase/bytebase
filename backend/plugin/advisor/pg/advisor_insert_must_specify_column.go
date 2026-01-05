package pg

import (
	"context"
	"fmt"

	"github.com/antlr4-go/antlr/v4"

	parser "github.com/bytebase/parser/postgresql"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

var (
	_ advisor.Advisor = (*InsertMustSpecifyColumnAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_POSTGRES, storepb.SQLReviewRule_STATEMENT_INSERT_MUST_SPECIFY_COLUMN, &InsertMustSpecifyColumnAdvisor{})
}

// InsertMustSpecifyColumnAdvisor is the advisor checking for to enforce column specified.
type InsertMustSpecifyColumnAdvisor struct {
}

// Check checks for to enforce column specified.
func (*InsertMustSpecifyColumnAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	var adviceList []*storepb.Advice
	for _, stmtInfo := range checkCtx.ParsedStatements {
		if stmtInfo.AST == nil {
			continue
		}
		antlrAST, ok := base.GetANTLRAST(stmtInfo.AST)
		if !ok {
			continue
		}
		rule := &insertMustSpecifyColumnRule{
			BaseRule: BaseRule{
				level: level,
				title: checkCtx.Rule.Type.String(),
			},
			tokens: antlrAST.Tokens,
		}
		rule.SetBaseLine(stmtInfo.BaseLine())

		checker := NewGenericChecker([]Rule{rule})
		antlr.ParseTreeWalkerDefault.Walk(checker, antlrAST.Tree)
		adviceList = append(adviceList, checker.GetAdviceList()...)
	}

	return adviceList, nil
}

type insertMustSpecifyColumnRule struct {
	BaseRule
	tokens *antlr.CommonTokenStream
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
		r.AddAdvice(&storepb.Advice{
			Status:  r.level,
			Code:    code.InsertNotSpecifyColumn.Int32(),
			Title:   r.title,
			Content: fmt.Sprintf("The INSERT statement must specify columns but \"%s\" does not", getTextFromTokens(r.tokens, insertstmtCtx)),
			StartPosition: &storepb.Position{
				Line:   int32(insertstmtCtx.GetStart().GetLine()),
				Column: 0,
			},
		})
	}
}
