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
	_ advisor.Advisor = (*NamingColumnConventionAdvisor)(nil)
)

func init() {
	advisor.Register(advisor.Postgres, advisor.PostgreSQLNamingColumnConvention, &NamingColumnConventionAdvisor{})
}

// NamingColumnConventionAdvisor is the advisor checking for column convention.
type NamingColumnConventionAdvisor struct {
}

// Check checks for column naming convention.
func (adv *NamingColumnConventionAdvisor) Check(ctx advisor.Context, statement string) ([]advisor.Advice, error) {
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
	checker := &namingColumnConventionChecker{
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

type namingColumnConventionChecker struct {
	adviceList []advisor.Advice
	level      advisor.Status
	title      string
	format     *regexp.Regexp
}

// Visit implements the ast.Visitor interface.
func (checker *namingColumnConventionChecker) Visit(node ast.Node) ast.Visitor {
	var columnList []string
	var tableName string

	switch n := node.(type) {
	// CREATE TABLE
	case *ast.CreateTableStmt:
		tableName = n.Name.Name
		for _, col := range n.ColumnList {
			columnList = append(columnList, col.ColumnName)
		}
	// ALTER TABLE ADD COLUMN
	case *ast.AddColumnListStmt:
		tableName = n.Table.Name
		for _, col := range n.ColumnList {
			columnList = append(columnList, col.ColumnName)
		}
	// ALTER TABLE RENAME COLUMN
	case *ast.RenameColumnStmt:
		tableName = n.Table.Name
		columnList = append(columnList, n.NewName)
	}

	for _, column := range columnList {
		if !checker.format.MatchString(column) {
			checker.adviceList = append(checker.adviceList, advisor.Advice{
				Status:  checker.level,
				Code:    advisor.NamingColumnConventionMismatch,
				Title:   checker.title,
				Content: fmt.Sprintf("\"%s\".\"%s\" mismatches column naming convention, naming format should be %q", tableName, column, checker.format),
			})
		}
	}

	return checker
}
