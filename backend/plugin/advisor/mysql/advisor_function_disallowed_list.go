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
	_ advisor.Advisor = (*FunctionDisallowedListAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MYSQL, advisor.MySQLFunctionDisallowedList, &FunctionDisallowedListAdvisor{})
}

// FunctionDisallowedListAdvisor is the advisor checking for disallowed function list.
type FunctionDisallowedListAdvisor struct {
}

func (*FunctionDisallowedListAdvisor) Check(ctx advisor.Context, _ string) ([]*storepb.Advice, error) {
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
	checker := &functionDisallowedListChecker{
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

	return checker.adviceList, nil
}

type functionDisallowedListChecker struct {
	*mysql.BaseMySQLParserListener

	baseLine     int
	adviceList   []*storepb.Advice
	level        storepb.Advice_Status
	title        string
	text         string
	disallowList []string
}

func (checker *functionDisallowedListChecker) EnterQuery(ctx *mysql.QueryContext) {
	checker.text = ctx.GetParser().GetTokenStream().GetTextFromRuleContext(ctx)
}

func (checker *functionDisallowedListChecker) EnterFunctionCall(ctx *mysql.FunctionCallContext) {
	pi := ctx.PureIdentifier()
	if pi != nil {
		functionName := mysqlparser.NormalizeMySQLPureIdentifier(pi)
		if slices.Contains(checker.disallowList, strings.ToUpper(functionName)) {
			checker.adviceList = append(checker.adviceList, &storepb.Advice{
				Status:  checker.level,
				Code:    advisor.DisabledFunction.Int32(),
				Title:   checker.title,
				Content: fmt.Sprintf("Function \"%s\" is disallowed, but \"%s\" uses", functionName, checker.text),
				StartPosition: &storepb.Position{
					Line: int32(checker.baseLine + ctx.GetStart().GetLine()),
				},
			})
		}
	}
}
