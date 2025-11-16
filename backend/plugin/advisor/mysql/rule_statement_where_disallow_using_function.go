package mysql

import (
	"context"
	"fmt"

	"github.com/bytebase/bytebase/backend/plugin/advisor/code"

	"github.com/antlr4-go/antlr/v4"
	"github.com/bytebase/parser/mysql"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	mysqlparser "github.com/bytebase/bytebase/backend/plugin/parser/mysql"
)

var (
	_ advisor.Advisor = (*StatementWhereDisallowUsingFunctionAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MYSQL, advisor.SchemaRuleStatementWhereDisallowFunctionsAndCalculations, &StatementWhereDisallowUsingFunctionAdvisor{})
}

type StatementWhereDisallowUsingFunctionAdvisor struct {
}

func (*StatementWhereDisallowUsingFunctionAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	stmtList, ok := checkCtx.AST.([]*mysqlparser.ParseResult)
	if !ok {
		return nil, errors.Errorf("failed to convert to mysql parse result")
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	// Create the rule
	rule := NewStatementWhereDisallowUsingFunctionRule(level, string(checkCtx.Rule.Type))

	// Create the generic checker with the rule
	checker := NewGenericChecker([]Rule{rule})

	for _, stmt := range stmtList {
		rule.SetBaseLine(stmt.BaseLine)
		checker.SetBaseLine(stmt.BaseLine)
		antlr.ParseTreeWalkerDefault.Walk(checker, stmt.Tree)
	}

	return checker.GetAdviceList(), nil
}

// StatementWhereDisallowUsingFunctionRule checks for functions in WHERE clause.
type StatementWhereDisallowUsingFunctionRule struct {
	BaseRule
	text          string
	isSelect      bool
	inWhereClause bool
}

// NewStatementWhereDisallowUsingFunctionRule creates a new StatementWhereDisallowUsingFunctionRule.
func NewStatementWhereDisallowUsingFunctionRule(level storepb.Advice_Status, title string) *StatementWhereDisallowUsingFunctionRule {
	return &StatementWhereDisallowUsingFunctionRule{
		BaseRule: BaseRule{
			level: level,
			title: title,
		},
	}
}

// Name returns the rule name.
func (*StatementWhereDisallowUsingFunctionRule) Name() string {
	return "StatementWhereDisallowUsingFunctionRule"
}

// OnEnter is called when entering a parse tree node.
func (r *StatementWhereDisallowUsingFunctionRule) OnEnter(ctx antlr.ParserRuleContext, nodeType string) error {
	switch nodeType {
	case NodeTypeQuery:
		queryCtx, ok := ctx.(*mysql.QueryContext)
		if !ok {
			return nil
		}
		r.text = queryCtx.GetParser().GetTokenStream().GetTextFromRuleContext(queryCtx)
	case NodeTypeSelectStatement:
		r.isSelect = true
	case NodeTypeWhereClause:
		r.inWhereClause = true
	case NodeTypeFunctionCall:
		r.checkFunctionCall(ctx.(*mysql.FunctionCallContext))
	default:
	}
	return nil
}

// OnExit is called when exiting a parse tree node.
func (r *StatementWhereDisallowUsingFunctionRule) OnExit(_ antlr.ParserRuleContext, nodeType string) error {
	switch nodeType {
	case NodeTypeSelectStatement:
		r.isSelect = false
	case NodeTypeWhereClause:
		r.inWhereClause = false
	default:
	}
	return nil
}

func (r *StatementWhereDisallowUsingFunctionRule) checkFunctionCall(ctx *mysql.FunctionCallContext) {
	if !r.isSelect || !r.inWhereClause {
		return
	}

	pi := ctx.PureIdentifier()
	if pi != nil {
		r.AddAdvice(&storepb.Advice{
			Status:        r.level,
			Code:          code.DisabledFunction.Int32(),
			Title:         r.title,
			Content:       fmt.Sprintf("Function is disallowed in where clause, but \"%s\" uses", r.text),
			StartPosition: common.ConvertANTLRLineToPosition(r.baseLine + ctx.GetStart().GetLine()),
		})
	}
}
