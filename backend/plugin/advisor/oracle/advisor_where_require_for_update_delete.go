// Package oracle is the advisor for oracle database.
package oracle

import (
	"context"

	"github.com/antlr4-go/antlr/v4"
	parser "github.com/bytebase/plsql-parser"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
)

var (
	_ advisor.Advisor = (*WhereRequireForUpdateDeleteAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_ORACLE, advisor.OracleWhereRequirementForUpdateDelete, &WhereRequireForUpdateDeleteAdvisor{})
	advisor.Register(storepb.Engine_DM, advisor.OracleWhereRequirementForUpdateDelete, &WhereRequireForUpdateDeleteAdvisor{})
	advisor.Register(storepb.Engine_OCEANBASE_ORACLE, advisor.OracleWhereRequirementForUpdateDelete, &WhereRequireForUpdateDeleteAdvisor{})
}

// WhereRequireForUpdateDeleteAdvisor is the advisor checking for WHERE clause requirement.
type WhereRequireForUpdateDeleteAdvisor struct {
}

// Check checks for WHERE clause requirement.
func (*WhereRequireForUpdateDeleteAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	tree, ok := checkCtx.AST.(antlr.Tree)
	if !ok {
		return nil, errors.Errorf("failed to convert to Tree")
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	listener := &whereRequireForUpdateDeleteListener{
		level:           level,
		title:           string(checkCtx.Rule.Type),
		currentDatabase: checkCtx.CurrentDatabase,
	}

	antlr.ParseTreeWalkerDefault.Walk(listener, tree)

	return listener.generateAdvice()
}

// whereRequireForUpdateDeleteListener is the listener for WHERE clause requirement.
type whereRequireForUpdateDeleteListener struct {
	*parser.BasePlSqlParserListener

	level           storepb.Advice_Status
	title           string
	currentDatabase string
	adviceList      []*storepb.Advice
}

func (l *whereRequireForUpdateDeleteListener) generateAdvice() ([]*storepb.Advice, error) {
	return l.adviceList, nil
}

// EnterUpdate_statement is called when production update_statement is entered.
func (l *whereRequireForUpdateDeleteListener) EnterUpdate_statement(ctx *parser.Update_statementContext) {
	if ctx.Where_clause() == nil {
		l.adviceList = append(l.adviceList, &storepb.Advice{
			Status:        l.level,
			Code:          advisor.StatementNoWhere.Int32(),
			Title:         l.title,
			Content:       "WHERE clause is required for UPDATE statement.",
			StartPosition: common.ConvertANTLRLineToPosition(ctx.GetStop().GetLine()),
		})
	}
}

// EnterDelete_statement is called when production delete_statement is entered.
func (l *whereRequireForUpdateDeleteListener) EnterDelete_statement(ctx *parser.Delete_statementContext) {
	if ctx.Where_clause() == nil {
		l.adviceList = append(l.adviceList, &storepb.Advice{
			Status:        l.level,
			Code:          advisor.StatementNoWhere.Int32(),
			Title:         l.title,
			Content:       "WHERE clause is required for DELETE statement.",
			StartPosition: common.ConvertANTLRLineToPosition(ctx.GetStop().GetLine()),
		})
	}
}
