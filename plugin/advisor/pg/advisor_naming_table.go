package pg

import (
	"fmt"
	"regexp"

	"github.com/bytebase/bytebase/plugin/advisor"
	"github.com/bytebase/bytebase/plugin/advisor/db"
	"github.com/bytebase/bytebase/plugin/parser/ast"
)

var (
	_ advisor.Advisor = (*NamingTableConventionAdvisor)(nil)
	_ ast.Visitor     = (*namingTableConventionChecker)(nil)
)

func init() {
	advisor.Register(db.Postgres, advisor.PostgreSQLNamingTableConvention, &NamingTableConventionAdvisor{})
}

// NamingTableConventionAdvisor is the advisor checking for table naming convention.
type NamingTableConventionAdvisor struct {
}

// Check checks for table naming convention.
func (*NamingTableConventionAdvisor) Check(ctx advisor.Context, statement string) ([]advisor.Advice, error) {
	stmts, errAdvice := parseStatement(statement)
	if errAdvice != nil {
		return errAdvice, nil
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(ctx.Rule.Level)
	if err != nil {
		return nil, err
	}

	format, maxLength, err := advisor.UnamrshalNamingRulePayloadAsRegexp(ctx.Rule.Payload)
	if err != nil {
		return nil, err
	}
	checker := &namingTableConventionChecker{
		level:     level,
		title:     string(ctx.Rule.Type),
		format:    format,
		maxLength: maxLength,
	}

	for _, stmt := range stmts {
		ast.Walk(checker, stmt)
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

type namingTableConventionChecker struct {
	adviceList []advisor.Advice
	level      advisor.Status
	title      string
	format     *regexp.Regexp
	maxLength  int
}

// Visit implements the ast.Visitor interface.
func (checker *namingTableConventionChecker) Visit(node ast.Node) ast.Visitor {
	var tableNames []string

	switch n := node.(type) {
	// CREATE TABLE
	case *ast.CreateTableStmt:
		tableNames = append(tableNames, n.Name.Name)
	// ALTER TABLE RENAME TABLE
	case *ast.RenameTableStmt:
		tableNames = append(tableNames, n.NewName)
	}

	for _, tableName := range tableNames {
		if !checker.format.MatchString(tableName) {
			checker.adviceList = append(checker.adviceList, advisor.Advice{
				Status:  checker.level,
				Code:    advisor.NamingTableConventionMismatch,
				Title:   checker.title,
				Content: fmt.Sprintf(`"%s" mismatches table naming convention, naming format should be %q`, tableName, checker.format),
				Line:    node.LastLine(),
			})
		}
		if checker.maxLength > 0 && len(tableName) > checker.maxLength {
			checker.adviceList = append(checker.adviceList, advisor.Advice{
				Status:  checker.level,
				Code:    advisor.NamingTableConventionMismatch,
				Title:   checker.title,
				Content: fmt.Sprintf("\"%s\" mismatches table naming convention, its length should be within %d characters", tableName, checker.maxLength),
				Line:    node.LastLine(),
			})
		}
	}

	return checker
}
