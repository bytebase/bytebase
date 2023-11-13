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
	_ advisor.Advisor = (*TableDisallowPartitionAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MYSQL, advisor.MySQLTableDisallowPartition, &TableDisallowPartitionAdvisor{})
	advisor.Register(storepb.Engine_MARIADB, advisor.MySQLTableDisallowPartition, &TableDisallowPartitionAdvisor{})
	advisor.Register(storepb.Engine_OCEANBASE, advisor.MySQLTableDisallowPartition, &TableDisallowPartitionAdvisor{})
}

// TableDisallowPartitionAdvisor is the advisor checking for disallow table partition.
type TableDisallowPartitionAdvisor struct {
}

// Check checks for disallow table partition.
func (*TableDisallowPartitionAdvisor) Check(ctx advisor.Context, _ string) ([]advisor.Advice, error) {
	stmtList, ok := ctx.AST.([]*mysqlparser.ParseResult)
	if !ok {
		return nil, errors.Errorf("failed to convert to mysql parser result")
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(ctx.Rule.Level)
	if err != nil {
		return nil, err
	}
	checker := &tableDisallowPartitionChecker{
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

type tableDisallowPartitionChecker struct {
	*mysql.BaseMySQLParserListener

	baseLine   int
	adviceList []advisor.Advice
	level      advisor.Status
	title      string
	text       string
}

func (checker *tableDisallowPartitionChecker) EnterQuery(ctx *mysql.QueryContext) {
	checker.text = ctx.GetParser().GetTokenStream().GetTextFromRuleContext(ctx)
}

// EnterCreateDatabase is called when production createDatabase is entered.
func (checker *tableDisallowPartitionChecker) EnterCreateTable(ctx *mysql.CreateTableContext) {
	code := advisor.Ok
	if ctx.PartitionClause() != nil && ctx.PartitionClause().PartitionTypeDef() != nil {
		code = advisor.CreateTablePartition
	}

	if code != advisor.Ok {
		checker.adviceList = append(checker.adviceList, advisor.Advice{
			Status:  checker.level,
			Code:    code,
			Title:   checker.title,
			Content: fmt.Sprintf("Table partition is forbidden, but \"%s\" creates", checker.text),
			Line:    checker.baseLine + ctx.GetStart().GetLine(),
		})
	}
}

// EnterAlterTable is called when production alterTable is entered.
func (checker *tableDisallowPartitionChecker) EnterAlterTable(ctx *mysql.AlterTableContext) {
	code := advisor.Ok
	if ctx.AlterTableActions().PartitionClause() != nil && ctx.AlterTableActions().PartitionClause().PartitionTypeDef() != nil {
		code = advisor.CreateTablePartition
	}
	if code != advisor.Ok {
		checker.adviceList = append(checker.adviceList, advisor.Advice{
			Status:  checker.level,
			Code:    code,
			Title:   checker.title,
			Content: fmt.Sprintf("Table partition is forbidden, but \"%s\" creates", checker.text),
			Line:    checker.baseLine + ctx.GetStart().GetLine(),
		})
	}
}
