package mysql

import (
	"fmt"

	"github.com/antlr4-go/antlr/v4"
	"github.com/pkg/errors"

	mysql "github.com/bytebase/mysql-parser"

	"github.com/bytebase/bytebase/backend/plugin/advisor"
	mysqlparser "github.com/bytebase/bytebase/backend/plugin/parser/mysql"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

var (
	_ advisor.Advisor = (*NamingIdentifierNoKeywordAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MYSQL, advisor.MySQLIdentifierNamingNoKeyword, &NamingIdentifierNoKeywordAdvisor{})
}

// NamingIdentifierNoKeywordAdvisor is the advisor checking for identifier naming convention without keyword.
type NamingIdentifierNoKeywordAdvisor struct {
}

// Check checks for identifier naming convention without keyword.
func (*NamingIdentifierNoKeywordAdvisor) Check(ctx advisor.Context, _ string) ([]advisor.Advice, error) {
	stmtList, ok := ctx.AST.([]*mysqlparser.ParseResult)
	if !ok {
		return nil, errors.Errorf("failed to convert to mysql parser result")
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(ctx.Rule.Level)
	if err != nil {
		return nil, err
	}

	checker := &namingIdentifierNoKeywordChecker{
		level: level,
		title: string(ctx.Rule.Type),
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

type namingIdentifierNoKeywordChecker struct {
	*mysql.BaseMySQLParserListener

	baseLine   int
	adviceList []advisor.Advice
	level      advisor.Status
	title      string
}

// EnterPureIdentifier is called when entering the pureIdentifier production.
func (checker *namingIdentifierNoKeywordChecker) EnterPureIdentifier(ctx *mysql.PureIdentifierContext) {
	// The suspect identifier should be always wrapped in backquotes, otherwise a syntax error will be thrown before entering this checker.
	textNode := ctx.BACK_TICK_QUOTED_ID()
	if textNode == nil {
		return
	}

	// Remove backticks as possible.
	identifier := trimBackTicks(textNode.GetText())
	advice := checker.checkIdentifier(identifier)
	if advice != nil {
		advice.Line = ctx.GetStart().GetLine()
		checker.adviceList = append(checker.adviceList, *advice)
	}
}

// EnterIdentifierKeyword is called when entering the identifierKeyword production.
func (checker *namingIdentifierNoKeywordChecker) EnterIdentifierKeyword(ctx *mysql.IdentifierKeywordContext) {
	identifier := ctx.GetText()
	advice := checker.checkIdentifier(identifier)
	if advice != nil {
		advice.Line = ctx.GetStart().GetLine()
		checker.adviceList = append(checker.adviceList, *advice)
	}
}

func (checker *namingIdentifierNoKeywordChecker) checkIdentifier(identifier string) *advisor.Advice {
	if isKeyword(identifier) {
		return &advisor.Advice{
			Status:  checker.level,
			Code:    advisor.NameIsKeywordIdentifier,
			Title:   checker.title,
			Content: fmt.Sprintf("Identifier %q is a keyword and should be avoided", identifier),
		}
	}

	return nil
}

func trimBackTicks(s string) string {
	if len(s) < 2 {
		return s
	}
	return s[1 : len(s)-1]
}
