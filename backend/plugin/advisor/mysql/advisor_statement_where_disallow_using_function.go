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
	_ advisor.Advisor = (*StatementWhereDisallowUsingFunctionAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MYSQL, advisor.MySQLStatementWhereDisallowUsingFunction, &StatementWhereDisallowUsingFunctionAdvisor{})
}

type StatementWhereDisallowUsingFunctionAdvisor struct {
}

func (*StatementWhereDisallowUsingFunctionAdvisor) Check(ctx advisor.Context, _ string) ([]advisor.Advice, error) {
	stmtList, ok := ctx.AST.([]*mysqlparser.ParseResult)
	if !ok {
		return nil, errors.Errorf("failed to convert to mysql parse result")
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(ctx.Rule.Level)
	if err != nil {
		return nil, err
	}
	checker := &statementWhereDisallowUsingFunctionChecker{
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

type statementWhereDisallowUsingFunctionChecker struct {
	*mysql.BaseMySQLParserListener

	baseLine      int
	adviceList    []advisor.Advice
	level         advisor.Status
	title         string
	text          string
	isSelect      bool
	inWhereClause bool
}

func (checker *statementWhereDisallowUsingFunctionChecker) EnterQuery(ctx *mysql.QueryContext) {
	checker.text = ctx.GetParser().GetTokenStream().GetTextFromRuleContext(ctx)
}

func (checker *statementWhereDisallowUsingFunctionChecker) EnterSelectStatement(*mysql.SelectStatementContext) {
	checker.isSelect = true
}

func (checker *statementWhereDisallowUsingFunctionChecker) ExitSelectStatement(*mysql.SelectStatementContext) {
	checker.isSelect = false
}

func (checker *statementWhereDisallowUsingFunctionChecker) EnterWhereClause(*mysql.WhereClauseContext) {
	checker.inWhereClause = true
}

func (checker *statementWhereDisallowUsingFunctionChecker) ExitWhereClause(*mysql.WhereClauseContext) {
	checker.inWhereClause = false
}

func (checker *statementWhereDisallowUsingFunctionChecker) EnterFunctionCall(ctx *mysql.FunctionCallContext) {
	if !checker.isSelect || !checker.inWhereClause {
		return
	}

	pi := ctx.PureIdentifier()
	if pi != nil {
		checker.adviceList = append(checker.adviceList, advisor.Advice{
			Status:  checker.level,
			Code:    advisor.DisabledFunction,
			Title:   checker.title,
			Content: fmt.Sprintf("Function is disallowed in where clause, but \"%s\" uses", checker.text),
			Line:    checker.baseLine + ctx.GetStart().GetLine(),
		})
	}
}
