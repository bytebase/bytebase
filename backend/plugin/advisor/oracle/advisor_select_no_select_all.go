// Package oracle is the advisor for oracle database.
package oracle

import (
	"github.com/antlr4-go/antlr/v4"
	parser "github.com/bytebase/plsql-parser"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/plugin/advisor"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

var (
	_ advisor.Advisor = (*SelectNoSelectAllAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_ORACLE, advisor.OracleNoSelectAll, &SelectNoSelectAllAdvisor{})
	advisor.Register(storepb.Engine_DM, advisor.OracleNoSelectAll, &SelectNoSelectAllAdvisor{})
	advisor.Register(storepb.Engine_OCEANBASE_ORACLE, advisor.OracleNoSelectAll, &SelectNoSelectAllAdvisor{})
}

// SelectNoSelectAllAdvisor is the advisor checking for no select all.
type SelectNoSelectAllAdvisor struct {
}

// Check checks for no select all.
func (*SelectNoSelectAllAdvisor) Check(ctx advisor.Context, _ string) ([]advisor.Advice, error) {
	tree, ok := ctx.AST.(antlr.Tree)
	if !ok {
		return nil, errors.Errorf("failed to convert to Tree")
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(ctx.Rule.Level)
	if err != nil {
		return nil, err
	}

	listener := &selectNoSelectAllListener{
		level:         level,
		title:         string(ctx.Rule.Type),
		currentSchema: ctx.CurrentSchema,
	}

	antlr.ParseTreeWalkerDefault.Walk(listener, tree)

	return listener.generateAdvice()
}

// selectNoSelectAllListener is the listener for no select all.
type selectNoSelectAllListener struct {
	*parser.BasePlSqlParserListener

	level         advisor.Status
	title         string
	currentSchema string
	adviceList    []advisor.Advice
}

func (l *selectNoSelectAllListener) generateAdvice() ([]advisor.Advice, error) {
	if len(l.adviceList) == 0 {
		l.adviceList = append(l.adviceList, advisor.Advice{
			Status:  advisor.Success,
			Code:    advisor.Ok,
			Title:   "OK",
			Content: "",
		})
	}
	return l.adviceList, nil
}

// EnterSelected_list is called when production selected_list is entered.
func (l *selectNoSelectAllListener) EnterSelected_list(ctx *parser.Selected_listContext) {
	if ctx.ASTERISK() != nil {
		l.adviceList = append(l.adviceList, advisor.Advice{
			Status:  l.level,
			Code:    advisor.StatementSelectAll,
			Title:   l.title,
			Content: "Avoid using SELECT *.",
			Line:    ctx.GetStart().GetLine(),
		})
	}
}
