// Package oracle is the advisor for oracle database.
package oracle

import (
	"strings"

	"github.com/antlr4-go/antlr/v4"
	parser "github.com/bytebase/plsql-parser"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/plugin/advisor"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

var (
	_ advisor.Advisor = (*WhereRequireForSelectAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_ORACLE, advisor.OracleWhereRequirementForSelect, &WhereRequireForSelectAdvisor{})
	advisor.Register(storepb.Engine_DM, advisor.OracleWhereRequirementForSelect, &WhereRequireForSelectAdvisor{})
	advisor.Register(storepb.Engine_OCEANBASE_ORACLE, advisor.OracleWhereRequirementForSelect, &WhereRequireForSelectAdvisor{})
}

// WhereRequireForSelectAdvisor is the advisor checking for WHERE clause requirement.
type WhereRequireForSelectAdvisor struct {
}

// Check checks for WHERE clause requirement.
func (*WhereRequireForSelectAdvisor) Check(ctx advisor.Context, _ string) ([]*storepb.Advice, error) {
	tree, ok := ctx.AST.(antlr.Tree)
	if !ok {
		return nil, errors.Errorf("failed to convert to Tree")
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(ctx.Rule.Level)
	if err != nil {
		return nil, err
	}

	listener := &whereRequireForSelectListener{
		level:           level,
		title:           string(ctx.Rule.Type),
		currentDatabase: ctx.CurrentDatabase,
	}

	antlr.ParseTreeWalkerDefault.Walk(listener, tree)

	return listener.generateAdvice()
}

// whereRequireForSelectListener is the listener for WHERE clause requirement.
type whereRequireForSelectListener struct {
	*parser.BasePlSqlParserListener

	level           storepb.Advice_Status
	title           string
	currentDatabase string
	adviceList      []*storepb.Advice
}

func (l *whereRequireForSelectListener) generateAdvice() ([]*storepb.Advice, error) {
	return l.adviceList, nil
}

// EnterQuery_block is called when production query_block is entered.
func (l *whereRequireForSelectListener) EnterQuery_block(ctx *parser.Query_blockContext) {
	// Allow SELECT queries without a FROM clause to proceed, e.g. SELECT 1.
	if ctx.From_clause() == nil || ctx.From_clause().Table_ref_list() == nil {
		return
	}
	if strings.ToLower(ctx.From_clause().Table_ref_list().GetText()) == "dual" {
		return
	}
	if ctx.Where_clause() == nil {
		l.adviceList = append(l.adviceList, &storepb.Advice{
			Status:  l.level,
			Code:    advisor.StatementNoWhere.Int32(),
			Title:   l.title,
			Content: "WHERE clause is required for SELECT statement.",
			StartPosition: &storepb.Position{
				Line: int32(ctx.GetStop().GetLine()),
			},
		})
	}
}
