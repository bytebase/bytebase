//go:build !release
// +build !release

package pg

import (
	"fmt"
	"regexp"

	"github.com/bytebase/bytebase/plugin/advisor"
	"github.com/bytebase/bytebase/plugin/parser/ast"
)

var (
	_ advisor.Advisor = (*NamingTableConventionAdvisor)(nil)
)

func init() {
	advisor.Register(advisor.Postgres, advisor.PostgreSQLNamingTableConvention, &NamingTableConventionAdvisor{})
}

// NamingTableConventionAdvisor is the advisor checking for table naming convention.
type NamingTableConventionAdvisor struct {
}

// Check checks for table naming convention.
func (adv *NamingTableConventionAdvisor) Check(ctx advisor.Context, statement string) ([]advisor.Advice, error) {
	stmts, errAdvice := parseStatement(statement)
	if errAdvice != nil {
		return errAdvice, nil
	}

	level, err := advisor.NewStatusBySchemaReviewRuleLevel(ctx.Rule.Level)
	if err != nil {
		return nil, err
	}
	format, err := advisor.UnamrshalNamingRulePayloadAsRegexp(ctx.Rule.Payload)
	if err != nil {
		return nil, err
	}
	checker := &namingTableConventionChecker{
		level:  level,
		title:  string(ctx.Rule.Type),
		format: format,
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
			})
		}
	}

	return checker
}
