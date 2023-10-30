package mysqlwip

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
	_ advisor.Advisor = (*WhereRequirementAdvisor)(nil)
)

func init() {
	// only for mysqlwip test.
	advisor.Register(storepb.Engine_ENGINE_UNSPECIFIED, advisor.MySQLWhereRequirement, &WhereRequirementAdvisor{})
}

// WhereRequirementAdvisor is the advisor checking for the WHERE clause requirement.
type WhereRequirementAdvisor struct {
}

// Check checks for the WHERE clause requirement.
func (*WhereRequirementAdvisor) Check(ctx advisor.Context, _ string) ([]advisor.Advice, error) {
	root, ok := ctx.AST.([]*mysqlparser.ParseResult)
	if !ok {
		return nil, errors.Errorf("failed to convert to StmtNode")
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(ctx.Rule.Level)
	if err != nil {
		return nil, err
	}
	checker := &whereRequirementChecker{
		level: level,
		title: string(ctx.Rule.Type),
	}
	for _, stmtNode := range root {
		checker.baseLine = stmtNode.BaseLine
		antlr.ParseTreeWalkerDefault.Walk(checker, stmtNode.Tree)
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

type whereRequirementChecker struct {
	*mysql.BaseMySQLParserListener

	baseLine   int
	adviceList []advisor.Advice
	level      advisor.Status
	title      string
	text       string
}

func (checker *whereRequirementChecker) EnterQuery(ctx *mysql.QueryContext) {
	checker.text = ctx.GetParser().GetTokenStream().GetTextFromRuleContext(ctx)
}

// EnterDeleteStatement is called when production deleteStatement is entered.
func (checker *whereRequirementChecker) EnterDeleteStatement(ctx *mysql.DeleteStatementContext) {
	if ctx.WhereClause() == nil || ctx.WhereClause().WHERE_SYMBOL() == nil {
		checker.handleWhereClause(ctx.GetStart().GetLine())
	}
}

// EnterUpdateStatement is called when production updateStatement is entered.
func (checker *whereRequirementChecker) EnterUpdateStatement(ctx *mysql.UpdateStatementContext) {
	if ctx.WhereClause() == nil || ctx.WhereClause().WHERE_SYMBOL() == nil {
		checker.handleWhereClause(ctx.GetStart().GetLine())
	}
}

func (checker *whereRequirementChecker) EnterQuerySpecification(ctx *mysql.QuerySpecificationContext) {
	if ctx.WhereClause() == nil || ctx.WhereClause().WHERE_SYMBOL() == nil {
		checker.handleWhereClause(ctx.GetStart().GetLine())
	}
}

func (checker *whereRequirementChecker) handleWhereClause(lineNumber int) {
	checker.adviceList = append(checker.adviceList, advisor.Advice{
		Status:  checker.level,
		Code:    advisor.StatementNoWhere,
		Title:   checker.title,
		Content: fmt.Sprintf("\"%s\" requires WHERE clause", checker.text),
		Line:    checker.baseLine + lineNumber,
	})
}
