// Package oracle is the advisor for oracle database.
package oracle

import (
	"context"

	"github.com/antlr4-go/antlr/v4"
	parser "github.com/bytebase/plsql-parser"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

var (
	_ advisor.Advisor = (*InsertMustSpecifyColumnAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_ORACLE, advisor.OracleInsertMustSpecifyColumn, &InsertMustSpecifyColumnAdvisor{})
	advisor.Register(storepb.Engine_DM, advisor.OracleInsertMustSpecifyColumn, &InsertMustSpecifyColumnAdvisor{})
	advisor.Register(storepb.Engine_OCEANBASE_ORACLE, advisor.OracleInsertMustSpecifyColumn, &InsertMustSpecifyColumnAdvisor{})
}

// InsertMustSpecifyColumnAdvisor is the advisor checking for to enforce column specified.
type InsertMustSpecifyColumnAdvisor struct {
}

// Check checks for to enforce column specified.
func (*InsertMustSpecifyColumnAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	tree, ok := checkCtx.AST.(antlr.Tree)
	if !ok {
		return nil, errors.Errorf("failed to convert to Tree")
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	listener := &insertMustSpecifyColumnListener{
		level:           level,
		title:           string(checkCtx.Rule.Type),
		currentDatabase: checkCtx.CurrentDatabase,
	}

	antlr.ParseTreeWalkerDefault.Walk(listener, tree)

	return listener.generateAdvice()
}

// insertMustSpecifyColumnListener is the listener for to enforce column specified.
type insertMustSpecifyColumnListener struct {
	*parser.BasePlSqlParserListener

	level           storepb.Advice_Status
	title           string
	currentDatabase string
	adviceList      []*storepb.Advice
}

func (l *insertMustSpecifyColumnListener) generateAdvice() ([]*storepb.Advice, error) {
	return l.adviceList, nil
}

// EnterInsert_into_clause is called when production insert_into_clause is entered.
func (l *insertMustSpecifyColumnListener) EnterInsert_into_clause(ctx *parser.Insert_into_clauseContext) {
	if ctx.Paren_column_list() == nil {
		l.adviceList = append(l.adviceList, &storepb.Advice{
			Status:        l.level,
			Code:          advisor.InsertNotSpecifyColumn.Int32(),
			Title:         l.title,
			Content:       "INSERT statement should specify column name.",
			StartPosition: common.ConvertANTLRLineToPosition(ctx.GetStart().GetLine()),
		})
	}
}
