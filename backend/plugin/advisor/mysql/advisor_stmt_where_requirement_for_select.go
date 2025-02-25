package mysql

import (
	"fmt"

	"github.com/antlr4-go/antlr/v4"
	mysql "github.com/bytebase/mysql-parser"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/plugin/advisor"
	mysqlparser "github.com/bytebase/bytebase/backend/plugin/parser/mysql"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

var (
	_ advisor.Advisor = (*WhereRequirementForSelectAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MYSQL, advisor.MySQLWhereRequirementForSelect, &WhereRequirementForSelectAdvisor{})
	advisor.Register(storepb.Engine_MARIADB, advisor.MySQLWhereRequirementForSelect, &WhereRequirementForSelectAdvisor{})
	advisor.Register(storepb.Engine_OCEANBASE, advisor.MySQLWhereRequirementForSelect, &WhereRequirementForSelectAdvisor{})
}

// WhereRequirementForSelectAdvisor is the advisor checking for the WHERE clause requirement for SELECT statements.
type WhereRequirementForSelectAdvisor struct {
}

// Check checks for the WHERE clause requirement.
func (*WhereRequirementForSelectAdvisor) Check(ctx advisor.Context) ([]*storepb.Advice, error) {
	root, ok := ctx.AST.([]*mysqlparser.ParseResult)
	if !ok {
		return nil, errors.Errorf("failed to convert to StmtNode")
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(ctx.Rule.Level)
	if err != nil {
		return nil, err
	}
	checker := &whereRequirementForSelectChecker{
		level: level,
		title: string(ctx.Rule.Type),
	}
	for _, stmtNode := range root {
		checker.baseLine = stmtNode.BaseLine
		antlr.ParseTreeWalkerDefault.Walk(checker, stmtNode.Tree)
	}

	return checker.adviceList, nil
}

type whereRequirementForSelectChecker struct {
	*mysql.BaseMySQLParserListener

	baseLine   int
	adviceList []*storepb.Advice
	level      storepb.Advice_Status
	title      string
	text       string
}

func (checker *whereRequirementForSelectChecker) EnterQuery(ctx *mysql.QueryContext) {
	checker.text = ctx.GetParser().GetTokenStream().GetTextFromRuleContext(ctx)
}

func (checker *whereRequirementForSelectChecker) EnterQuerySpecification(ctx *mysql.QuerySpecificationContext) {
	// Allow SELECT queries without a FROM clause to proceed, e.g. SELECT 1.
	if ctx.FromClause() == nil {
		return
	}
	if ctx.WhereClause() == nil || ctx.WhereClause().WHERE_SYMBOL() == nil {
		checker.handleWhereClause(ctx.GetStart().GetLine())
	}
}

func (checker *whereRequirementForSelectChecker) handleWhereClause(lineNumber int) {
	checker.adviceList = append(checker.adviceList, &storepb.Advice{
		Status:  checker.level,
		Code:    advisor.StatementNoWhere.Int32(),
		Title:   checker.title,
		Content: fmt.Sprintf("\"%s\" requires WHERE clause", checker.text),
		StartPosition: &storepb.Position{
			Line: int32(checker.baseLine + lineNumber),
		},
	})
}
