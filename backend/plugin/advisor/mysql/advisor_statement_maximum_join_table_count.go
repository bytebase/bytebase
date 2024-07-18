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
	_ advisor.Advisor = (*StatementMaximumJoinTableCountAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MYSQL, advisor.MySQLStatementMaximumJoinTableCount, &StatementMaximumJoinTableCountAdvisor{})
}

type StatementMaximumJoinTableCountAdvisor struct {
}

func (*StatementMaximumJoinTableCountAdvisor) Check(ctx advisor.Context, _ string) ([]*storepb.Advice, error) {
	stmtList, ok := ctx.AST.([]*mysqlparser.ParseResult)
	if !ok {
		return nil, errors.Errorf("failed to convert to mysql parse result")
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(ctx.Rule.Level)
	if err != nil {
		return nil, err
	}
	payload, err := advisor.UnmarshalNumberTypeRulePayload(ctx.Rule.Payload)
	if err != nil {
		return nil, err
	}
	checker := &statementMaximumJoinTableCountChecker{
		level:         level,
		title:         string(ctx.Rule.Type),
		limitMaxValue: payload.Number,
	}

	for _, stmt := range stmtList {
		checker.baseLine = stmt.BaseLine
		antlr.ParseTreeWalkerDefault.Walk(checker, stmt.Tree)
	}

	return checker.adviceList, nil
}

type statementMaximumJoinTableCountChecker struct {
	*mysql.BaseMySQLParserListener

	baseLine      int
	adviceList    []*storepb.Advice
	level         storepb.Advice_Status
	title         string
	text          string
	limitMaxValue int
	count         int
}

func (checker *statementMaximumJoinTableCountChecker) EnterQuery(ctx *mysql.QueryContext) {
	checker.text = ctx.GetParser().GetTokenStream().GetTextFromRuleContext(ctx)
}

func (checker *statementMaximumJoinTableCountChecker) EnterJoinedTable(ctx *mysql.JoinedTableContext) {
	checker.count++
	// The count starts from 0. We count the number of tables in the joins.
	if checker.count == checker.limitMaxValue {
		checker.adviceList = append(checker.adviceList, &storepb.Advice{
			Status:  checker.level,
			Code:    advisor.StatementMaximumJoinTableCount.Int32(),
			Title:   checker.title,
			Content: fmt.Sprintf("\"%s\" exceeds the maximum number of joins %d.", checker.text, checker.limitMaxValue),
			StartPosition: &storepb.Position{
				Line: int32(checker.baseLine + ctx.GetStart().GetLine()),
			},
		})
	}
}
