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
	format, err := api.UnmarshalNamingRulePayloadFormat(ctx.Rule.Payload)
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

	return checker.generateAdviceList(), nil
}

type namingColumnConventionChecker struct {
	adviceList []advisor.Advice
	level      advisor.Status
	format     *regexp.Regexp
	tables     tableState
}

func (v *namingColumnConventionChecker) Enter(in ast.Node) (ast.Node, bool) {
	switch node := in.(type) {
	// CREATE TABLE
	case *ast.CreateTableStmt:
		v.createTable(node)
	// ALTER TABLE
	case *ast.AlterTableStmt:
		table := v.tables.getTable(node.Table.Name.O)
		for _, spec := range node.Specs {
			switch spec.Tp {
			// RENAME COLUMN
			case ast.AlterTableRenameColumn:
				delete(table, spec.OldColumnName.Name.O)
				table[spec.NewColumnName.Name.O] = true
			// ADD COLUMNS
			case ast.AlterTableAddColumns:
				for _, column := range spec.NewColumns {
					table[column.Name.Name.O] = true
				}
			// DROP COLUMN
			case ast.AlterTableDropColumn:
				delete(table, spec.OldColumnName.Name.O)
			// CHANGE COLUMN
			case ast.AlterTableChangeColumn:
				delete(table, spec.OldColumnName.Name.O)
				table[spec.NewColumns[0].Name.Name.O] = true
			}
		}
	}
	return in, false
}

func (v *namingColumnConventionChecker) Leave(in ast.Node) (ast.Node, bool) {
	return in, true
}

func (v *namingColumnConventionChecker) createTable(node *ast.CreateTableStmt) {
	table := make(columnSet)
	for _, column := range node.Cols {
		table[column.Name.Name.O] = true
	}
	v.tables[node.Table.Name.O] = table
}

func (v *namingColumnConventionChecker) generateAdviceList() []advisor.Advice {
	tableList := v.tables.tableList()
	for _, tableName := range tableList {
		table := v.tables[tableName]
		columnList := table.columnList()
		for _, columnName := range columnList {
			if !v.format.MatchString(columnName) {
				v.adviceList = append(v.adviceList, advisor.Advice{
					Status:  v.level,
					Code:    common.NamingColumnConventionMismatch,
					Title:   "Mismatch column naming convention",
					Content: fmt.Sprintf("`%s`.`%s` mismatches column naming convention", tableName, columnName),
				})
			}
		}
	}

	if len(v.adviceList) == 0 {
		v.adviceList = append(v.adviceList, advisor.Advice{
			Status:  advisor.Success,
			Code:    common.Ok,
			Title:   "OK",
			Content: "",
		})
	}
	return v.adviceList
}
