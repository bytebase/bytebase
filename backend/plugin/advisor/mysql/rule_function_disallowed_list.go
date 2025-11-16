package mysql

import (
	"context"
	"fmt"
	"slices"
	"strings"

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
	_ advisor.Advisor = (*FunctionDisallowedListAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MYSQL, advisor.SchemaRuleFunctionDisallowList, &FunctionDisallowedListAdvisor{})
}

// FunctionDisallowedListAdvisor is the advisor checking for disallowed function list.
type FunctionDisallowedListAdvisor struct {
}

func (*FunctionDisallowedListAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	stmtList, ok := checkCtx.AST.([]*mysqlparser.ParseResult)
	if !ok {
		return nil, errors.Errorf("failed to convert to mysql parser result")
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}
	payload, err := advisor.UnmarshalStringArrayTypeRulePayload(checkCtx.Rule.Payload)
	if err != nil {
		return nil, err
	}

	var disallowList []string
	for _, fn := range payload.List {
		disallowList = append(disallowList, strings.ToUpper(fn))
	}

	// Create the rule
	rule := NewFunctionDisallowedListRule(level, string(checkCtx.Rule.Type), disallowList)

	// Create the generic checker with the rule
	checker := NewGenericChecker([]Rule{rule})

	for _, stmt := range stmtList {
		rule.SetBaseLine(stmt.BaseLine)
		checker.SetBaseLine(stmt.BaseLine)
		antlr.ParseTreeWalkerDefault.Walk(checker, stmt.Tree)
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
