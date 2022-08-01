package mysql

import (
	"fmt"
	"regexp"

	"github.com/bytebase/bytebase/plugin/advisor"
	"github.com/bytebase/bytebase/plugin/advisor/db"
	"github.com/pingcap/tidb/parser/ast"
)

var (
	_ advisor.Advisor = (*NamingColumnConventionAdvisor)(nil)
	_ ast.Visitor     = (*namingColumnConventionChecker)(nil)
)

func init() {
	advisor.Register(db.MySQL, advisor.MySQLNamingColumnConvention, &NamingColumnConventionAdvisor{})
	advisor.Register(db.TiDB, advisor.MySQLNamingColumnConvention, &NamingColumnConventionAdvisor{})
}

// NamingColumnConventionAdvisor is the advisor checking for column naming convention.
type NamingColumnConventionAdvisor struct {
}

// Check checks for column naming convention.
func (*NamingColumnConventionAdvisor) Check(ctx advisor.Context, statement string) ([]advisor.Advice, error) {
	root, errAdvice := parseStatement(statement, ctx.Charset, ctx.Collation)
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
	checker := &namingColumnConventionChecker{
		level:     level,
		title:     string(ctx.Rule.Type),
		format:    format,
		maxLength: maxLength,
		tables:    make(tableState),
	}

	for _, stmtNode := range root {
		(stmtNode).Accept(checker)
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
	maxLength  int
	tables     tableState
}

// Enter implements the ast.Visitor interface.
func (v *namingColumnConventionChecker) Enter(in ast.Node) (ast.Node, bool) {
	var columnList []string
	var tableName string
	switch node := in.(type) {
	// CREATE TABLE
	case *ast.CreateTableStmt:
		tableName = node.Table.Name.O
		for _, column := range node.Cols {
			columnList = append(columnList, column.Name.Name.O)
		}
	// ALTER TABLE
	case *ast.AlterTableStmt:
		tableName = node.Table.Name.O
		for _, spec := range node.Specs {
			switch spec.Tp {
			// RENAME COLUMN
			case ast.AlterTableRenameColumn:
				columnList = append(columnList, spec.NewColumnName.Name.O)
			// ADD COLUMNS
			case ast.AlterTableAddColumns:
				for _, column := range spec.NewColumns {
					columnList = append(columnList, column.Name.Name.O)
				}
			// CHANGE COLUMN
			case ast.AlterTableChangeColumn:
				columnList = append(columnList, spec.NewColumns[0].Name.Name.O)
			}
		}
	}

	for _, column := range columnList {
		if !v.format.MatchString(column) {
			v.adviceList = append(v.adviceList, advisor.Advice{
				Status:  v.level,
				Code:    advisor.NamingColumnConventionMismatch,
				Title:   v.title,
				Content: fmt.Sprintf("`%s`.`%s` mismatches column naming convention, naming format should be %q", tableName, column, v.format),
			})
		}
		if v.maxLength > 0 && len(column) > v.maxLength {
			v.adviceList = append(v.adviceList, advisor.Advice{
				Status:  v.level,
				Code:    advisor.NamingColumnConventionMismatch,
				Title:   v.title,
				Content: fmt.Sprintf("`%s`.`%s` mismatches column naming convention, its length should be within %d characters", tableName, column, v.maxLength),
			})
		}
	}

	return in, false
}

// Leave implements the ast.Visitor interface.
func (*namingColumnConventionChecker) Leave(in ast.Node) (ast.Node, bool) {
	return in, true
}
