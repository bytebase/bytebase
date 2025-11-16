package mysql

import (
	"context"
	"fmt"

	"github.com/antlr4-go/antlr/v4"
	"github.com/pkg/errors"

	"github.com/bytebase/parser/mysql"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	advisorcode "github.com/bytebase/bytebase/backend/plugin/advisor/code"
	mysqlparser "github.com/bytebase/bytebase/backend/plugin/parser/mysql"
)

var (
	_ advisor.Advisor = (*ProcedureDisallowCreateAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MYSQL, advisor.SchemaRuleProcedureDisallowCreate, &ProcedureDisallowCreateAdvisor{})
}

// ProcedureDisallowCreateAdvisor is the advisor checking for disallow create procedure.
type ProcedureDisallowCreateAdvisor struct {
}

// Check checks for disallow create procedure.
func (*ProcedureDisallowCreateAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	stmtList, ok := checkCtx.AST.([]*mysqlparser.ParseResult)
	if !ok {
		return nil, errors.Errorf("failed to convert to mysql parser result")
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	// Create the rule
	rule := NewProcedureDisallowCreateRule(level, string(checkCtx.Rule.Type))

	// Create the generic checker with the rule
	checker := NewGenericChecker([]Rule{rule})

	for _, stmt := range stmtList {
		rule.SetBaseLine(stmt.BaseLine)
		checker.SetBaseLine(stmt.BaseLine)
		antlr.ParseTreeWalkerDefault.Walk(checker, stmt.Tree)
	}

	return checker.GetAdviceList(), nil
}

// ProcedureDisallowCreateRule checks for disallow create procedure.
type ProcedureDisallowCreateRule struct {
	BaseRule
	text string
}

// NewProcedureDisallowCreateRule creates a new ProcedureDisallowCreateRule.
func NewProcedureDisallowCreateRule(level storepb.Advice_Status, title string) *ProcedureDisallowCreateRule {
	return &ProcedureDisallowCreateRule{
		BaseRule: BaseRule{
			level: level,
			title: title,
		},
	}
}

// Name returns the rule name.
func (*ProcedureDisallowCreateRule) Name() string {
	return "ProcedureDisallowCreateRule"
}

// OnEnter is called when entering a parse tree node.
func (r *ProcedureDisallowCreateRule) OnEnter(ctx antlr.ParserRuleContext, nodeType string) error {
	switch nodeType {
	case NodeTypeQuery:
		if queryCtx, ok := ctx.(*mysql.QueryContext); ok {
			r.text = queryCtx.GetParser().GetTokenStream().GetTextFromRuleContext(queryCtx)
		}
	case NodeTypeCreateProcedure:
		r.checkCreateProcedure(ctx.(*mysql.CreateProcedureContext))
	default:
		// Ignore other node types
	}
	return nil
}

// OnExit is called when exiting a parse tree node.
func (*ProcedureDisallowCreateRule) OnExit(_ antlr.ParserRuleContext, _ string) error {
	return nil
}

func (r *ProcedureDisallowCreateRule) checkCreateProcedure(ctx *mysql.CreateProcedureContext) {
	if !mysqlparser.IsTopMySQLRule(&ctx.BaseParserRuleContext) {
		return
	}
	code := advisorcode.Ok
	if ctx.ProcedureName() != nil {
		code = advisorcode.DisallowCreateProcedure
	}

	if code != advisorcode.Ok {
		r.AddAdvice(&storepb.Advice{
			Status:        r.level,
			Code:          code.Int32(),
			Title:         r.title,
			Content:       fmt.Sprintf("Procedure is forbidden, but \"%s\" creates", r.text),
			StartPosition: common.ConvertANTLRLineToPosition(r.baseLine + ctx.GetStart().GetLine()),
		})
	}
}
