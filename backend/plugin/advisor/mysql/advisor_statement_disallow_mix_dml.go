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
	_ advisor.Advisor = (*StatementDisallowMixDMLAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MYSQL, advisor.MySQLStatementDisallowMixDML, &StatementDisallowMixDMLAdvisor{})
}

type StatementDisallowMixDMLAdvisor struct {
}

func (*StatementDisallowMixDMLAdvisor) Check(ctx advisor.Context, _ string) ([]advisor.Advice, error) {
	stmtList, ok := ctx.AST.([]*mysqlparser.ParseResult)
	if !ok {
		return nil, errors.Errorf("failed to convert to mysql parse result")
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(ctx.Rule.Level)
	if err != nil {
		return nil, err
	}

	checker := &statementDisallowMixDMLChecker{
		level:             level,
		title:             string(ctx.Rule.Type),
		dmlStatementCount: make(map[string]int),
	}

	for _, stmt := range stmtList {
		antlr.ParseTreeWalkerDefault.Walk(checker, stmt.Tree)
	}

	if len(checker.dmlStatementCount) > 1 {
		content := "Found"
		for t, count := range checker.dmlStatementCount {
			content += fmt.Sprintf(" %d %s,", count, t)
		}
		content += " disallow mixing different types of DML statements"
		checker.adviceList = append(checker.adviceList, advisor.Advice{
			Status:  checker.level,
			Code:    advisor.StatementDisallowMixDML,
			Title:   checker.title,
			Content: content,
		})
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

type statementDisallowMixDMLChecker struct {
	*mysql.BaseMySQLParserListener

	adviceList []advisor.Advice
	level      advisor.Status
	title      string

	dmlStatementCount map[string]int
}

func (c *statementDisallowMixDMLChecker) EnterInsertStatement(_ *mysql.InsertStatementContext) {
	c.dmlStatementCount["INSERT"]++
}

func (c *statementDisallowMixDMLChecker) EnterUpdateStatement(_ *mysql.UpdateStatementContext) {
	c.dmlStatementCount["UPDATE"]++
}

func (c *statementDisallowMixDMLChecker) EnterDeleteStatement(_ *mysql.DeleteStatementContext) {
	c.dmlStatementCount["DELETE"]++
}
