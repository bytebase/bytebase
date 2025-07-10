// Package oracle is the advisor for oracle database.
package oracle

import (
	"context"
	"fmt"

	"github.com/antlr4-go/antlr/v4"
	parser "github.com/bytebase/plsql-parser"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
)

var (
	_ advisor.Advisor = (*IndexKeyNumberLimitAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_ORACLE, advisor.OracleIndexKeyNumberLimit, &IndexKeyNumberLimitAdvisor{})
	advisor.Register(storepb.Engine_DM, advisor.OracleIndexKeyNumberLimit, &IndexKeyNumberLimitAdvisor{})
	advisor.Register(storepb.Engine_OCEANBASE_ORACLE, advisor.OracleIndexKeyNumberLimit, &IndexKeyNumberLimitAdvisor{})
}

// IndexKeyNumberLimitAdvisor is the advisor checking for index key number limit.
type IndexKeyNumberLimitAdvisor struct {
}

// Check checks for index key number limit.
func (*IndexKeyNumberLimitAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	tree, ok := checkCtx.AST.(antlr.Tree)
	if !ok {
		return nil, errors.Errorf("failed to convert to Tree")
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}
	payload, err := advisor.UnmarshalNumberTypeRulePayload(checkCtx.Rule.Payload)
	if err != nil {
		return nil, err
	}

	listener := &indexKeyNumberLimitListener{
		level:           level,
		title:           string(checkCtx.Rule.Type),
		currentDatabase: checkCtx.CurrentDatabase,
		max:             payload.Number,
	}

	if listener.max > 0 {
		antlr.ParseTreeWalkerDefault.Walk(listener, tree)
	}

	return listener.generateAdvice()
}

// indexKeyNumberLimitListener is the listener for index key number limit.
type indexKeyNumberLimitListener struct {
	*parser.BasePlSqlParserListener

	level           storepb.Advice_Status
	title           string
	currentDatabase string
	max             int
	adviceList      []*storepb.Advice
}

func (l *indexKeyNumberLimitListener) generateAdvice() ([]*storepb.Advice, error) {
	return l.adviceList, nil
}

// EnterTable_index_clause is called when production table_index_clause is entered.
func (l *indexKeyNumberLimitListener) EnterTable_index_clause(ctx *parser.Table_index_clauseContext) {
	keys := len(ctx.AllIndex_expr_option())
	if keys > l.max {
		l.adviceList = append(l.adviceList, &storepb.Advice{
			Status:        l.level,
			Code:          advisor.IndexKeyNumberExceedsLimit.Int32(),
			Title:         l.title,
			Content:       fmt.Sprintf("Index key number should be less than or equal to %d", l.max),
			StartPosition: common.ConvertANTLRLineToPosition(ctx.GetStart().GetLine()),
		})
	}
}

// EnterOut_of_line_constraint is called when production out_of_line_constraint is entered.
func (l *indexKeyNumberLimitListener) EnterOut_of_line_constraint(ctx *parser.Out_of_line_constraintContext) {
	keys := len(ctx.AllColumn_name())
	if keys > l.max {
		l.adviceList = append(l.adviceList, &storepb.Advice{
			Status:        l.level,
			Code:          advisor.IndexKeyNumberExceedsLimit.Int32(),
			Title:         l.title,
			Content:       fmt.Sprintf("Index key number should be less than or equal to %d", l.max),
			StartPosition: common.ConvertANTLRLineToPosition(ctx.GetStart().GetLine()),
		})
	}
}
