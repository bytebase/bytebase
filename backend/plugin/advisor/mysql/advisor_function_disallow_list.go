package mysql

import (
	"fmt"
	"slices"
	"strings"

	"github.com/antlr4-go/antlr/v4"
	"github.com/pkg/errors"

	mysql "github.com/bytebase/mysql-parser"

	"github.com/bytebase/bytebase/backend/plugin/advisor"
	mysqlparser "github.com/bytebase/bytebase/backend/plugin/parser/mysql"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

var (
	_ advisor.Advisor = (*FunctionDisallowListAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MYSQL, advisor.MySQLFunctionDisallowList, &FunctionDisallowListAdvisor{})
}

// FunctionDisallowListAdvisor is the advisor checking for disallow function list.
type FunctionDisallowListAdvisor struct {
}

func (*FunctionDisallowListAdvisor) Check(ctx advisor.Context, _ string) ([]advisor.Advice, error) {
	stmtList, ok := ctx.AST.([]*mysqlparser.ParseResult)
	if !ok {
		return nil, errors.Errorf("failed to convert to mysql parser result")
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(ctx.Rule.Level)
	if err != nil {
		return nil, err
	}
	paylaod, err := advisor.UnmarshalStringArrayTypeRulePayload(ctx.Rule.Payload)
	if err != nil {
		return nil, err
	}
	checker := &functionDisallowListChecker{
		level: level,
		title: string(ctx.Rule.Type),
	}
	for _, fn := range paylaod.List {
		checker.disallowList = append(checker.disallowList, strings.ToUpper(fn))
	}

	for _, stmt := range stmtList {
		checker.baseLine = stmt.BaseLine
		antlr.ParseTreeWalkerDefault.Walk(checker, stmt.Tree)
	}

	if len(checker.adviceList) == 0 {
		checker.adviceList = append(checker.adviceList, advisor.Advice{
			Status:  advisor.Success,
			Code:    advisor.Ok,
			Title:   "OK",
			Content: "",
		})
	}
	return checker.adviceList, nil
}

type functionDisallowListChecker struct {
	*mysql.BaseMySQLParserListener

	baseLine     int
	adviceList   []advisor.Advice
	level        advisor.Status
	title        string
	text         string
	disallowList []string
}

func (checker *functionDisallowListChecker) EnterQuery(ctx *mysql.QueryContext) {
	checker.text = ctx.GetParser().GetTokenStream().GetTextFromRuleContext(ctx)
}

func (checker *functionDisallowListChecker) EnterFunctionCall(ctx *mysql.FunctionCallContext) {
	pi := ctx.PureIdentifier()
	if pi != nil {
		functionName := pi.IDENTIFIER().GetText()
		if slices.Contains(checker.disallowList, strings.ToUpper(functionName)) {
			checker.adviceList = append(checker.adviceList, advisor.Advice{
				Status:  checker.level,
				Code:    advisor.DisabledFunction,
				Title:   checker.title,
				Content: fmt.Sprintf("Function \"%s\" is disallowed, but \"%s\" uses", functionName, checker.text),
				Line:    checker.baseLine + ctx.GetStart().GetLine(),
			})
		}
	}
}
