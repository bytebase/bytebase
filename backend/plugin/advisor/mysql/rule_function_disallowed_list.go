package mysql

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/bytebase/bytebase/backend/plugin/advisor/code"

	"github.com/antlr4-go/antlr/v4"
	"github.com/bytebase/parser/mysql"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	mysqlparser "github.com/bytebase/bytebase/backend/plugin/parser/mysql"
)

var (
	_ advisor.Advisor = (*FunctionDisallowedListAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MYSQL, storepb.SQLReviewRule_SYSTEM_FUNCTION_DISALLOWED_LIST, &FunctionDisallowedListAdvisor{})
}

// FunctionDisallowedListAdvisor is the advisor checking for disallowed function list.
type FunctionDisallowedListAdvisor struct {
}

func (*FunctionDisallowedListAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}
	stringArrayPayload := checkCtx.Rule.GetStringArrayPayload()

	var disallowList []string
	for _, fn := range stringArrayPayload.List {
		disallowList = append(disallowList, strings.ToUpper(fn))
	}

	// Create the rule
	rule := NewFunctionDisallowedListRule(level, checkCtx.Rule.Type.String(), disallowList)

	// Create the generic checker with the rule
	checker := NewGenericChecker([]Rule{rule})

	for _, stmt := range checkCtx.ParsedStatements {
		rule.SetBaseLine(stmt.BaseLine())
		checker.SetBaseLine(stmt.BaseLine())
		if stmt.AST == nil {
			continue
		}
		antlrAST, ok := base.GetANTLRAST(stmt.AST)
		if !ok {
			continue
		}
		antlr.ParseTreeWalkerDefault.Walk(checker, antlrAST.Tree)
	}

	return checker.GetAdviceList(), nil
}

// FunctionDisallowedListRule checks for disallowed function list.
type FunctionDisallowedListRule struct {
	BaseRule
	text         string
	disallowList []string
}

// NewFunctionDisallowedListRule creates a new FunctionDisallowedListRule.
func NewFunctionDisallowedListRule(level storepb.Advice_Status, title string, disallowList []string) *FunctionDisallowedListRule {
	return &FunctionDisallowedListRule{
		BaseRule: BaseRule{
			level: level,
			title: title,
		},
		disallowList: disallowList,
	}
}

// Name returns the rule name.
func (*FunctionDisallowedListRule) Name() string {
	return "FunctionDisallowedListRule"
}

// OnEnter is called when entering a parse tree node.
func (r *FunctionDisallowedListRule) OnEnter(ctx antlr.ParserRuleContext, nodeType string) error {
	switch nodeType {
	case NodeTypeQuery:
		r.checkQuery(ctx.(*mysql.QueryContext))
	case NodeTypeFunctionCall:
		r.checkFunctionCall(ctx.(*mysql.FunctionCallContext))
	default:
	}
	return nil
}

// OnExit is called when exiting a parse tree node.
func (*FunctionDisallowedListRule) OnExit(_ antlr.ParserRuleContext, _ string) error {
	return nil
}

func (r *FunctionDisallowedListRule) checkQuery(ctx *mysql.QueryContext) {
	r.text = ctx.GetParser().GetTokenStream().GetTextFromRuleContext(ctx)
}

func (r *FunctionDisallowedListRule) checkFunctionCall(ctx *mysql.FunctionCallContext) {
	pi := ctx.PureIdentifier()
	if pi != nil {
		functionName := mysqlparser.NormalizeMySQLPureIdentifier(pi)
		if slices.Contains(r.disallowList, strings.ToUpper(functionName)) {
			r.AddAdvice(&storepb.Advice{
				Status:        r.level,
				Code:          code.DisabledFunction.Int32(),
				Title:         r.title,
				Content:       fmt.Sprintf("Function \"%s\" is disallowed, but \"%s\" uses", functionName, r.text),
				StartPosition: common.ConvertANTLRLineToPosition(r.baseLine + ctx.GetStart().GetLine()),
			})
		}
	}
}
