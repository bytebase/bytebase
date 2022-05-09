package mysql

import (
	"fmt"
	"regexp"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/common"
	"github.com/bytebase/bytebase/plugin/advisor"
	"github.com/bytebase/bytebase/plugin/db"
	"github.com/pingcap/tidb/parser/ast"
)

var (
	_ advisor.Advisor = (*NamingColumnConventionAdvisor)(nil)
)

func init() {
	advisor.Register(db.MySQL, advisor.MySQLNamingColumnConvention, &NamingColumnConventionAdvisor{})
	advisor.Register(db.TiDB, advisor.MySQLNamingColumnConvention, &NamingColumnConventionAdvisor{})
}

// NamingColumnConventionAdvisor is the advisor checking for column naming convention.
type NamingColumnConventionAdvisor struct {
}

// Check checks for column naming convention.
func (adv *NamingColumnConventionAdvisor) Check(ctx advisor.Context, statement string) ([]advisor.Advice, error) {
	root, errAdvice := parseStatement(statement, ctx.Charset, ctx.Collation)
	if errAdvice != nil {
		return errAdvice, nil
	}

	level, err := advisor.NewStatusBySchemaReviewRuleLevel(ctx.Rule.Level)
	if err != nil {
		return nil, err
	}
	format, err := api.UnamrshalNamingRulePayloadAsRegexp(ctx.Rule.Payload)
	if err != nil {
		return nil, err
	}
	checker := &namingColumnConventionChecker{
		level:  level,
		format: format,
		tables: make(tableState),
	}

	for _, stmtNode := range root {
		(stmtNode).Accept(checker)
	}

	if len(checker.adviceList) == 0 {
		checker.adviceList = append(checker.adviceList, advisor.Advice{
			Status:  advisor.Success,
			Code:    common.Ok,
			Title:   "OK",
			Content: "",
		})
	}

	return checker.adviceList, nil
}

type namingColumnConventionChecker struct {
	adviceList []advisor.Advice
	level      advisor.Status
	format     *regexp.Regexp
	tables     tableState
}

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
				Status: v.level,
				Code:   common.NamingColumnConventionMismatch,
				Title:  "Mismatch column naming convention",
				// TODO: give a more explicit message about the valid naming format
				Content: fmt.Sprintf("`%s`.`%s` mismatches column naming convention", tableName, column),
			})
		}
	}

	return in, false
}

func (v *namingColumnConventionChecker) Leave(in ast.Node) (ast.Node, bool) {
	return in, true
}
