package mysql

import (
	"fmt"

	"github.com/bytebase/bytebase/common"
	"github.com/bytebase/bytebase/plugin/advisor"
	"github.com/bytebase/bytebase/plugin/db"
	"github.com/pingcap/tidb/parser/ast"
)

var (
	_ advisor.Advisor = (*ColumnNoNullAdvisor)(nil)
)

func init() {
	advisor.Register(db.MySQL, advisor.MySQLColumnNoNull, &ColumnNoNullAdvisor{})
	advisor.Register(db.TiDB, advisor.MySQLColumnNoNull, &ColumnNoNullAdvisor{})
}

// ColumnNoNullAdvisor is the advisor checking for column no NULL value.
type ColumnNoNullAdvisor struct {
}

// Check checks for column no NULL value.
func (adv *ColumnNoNullAdvisor) Check(ctx advisor.Context, statement string) ([]advisor.Advice, error) {
	root, errAdvice := parseStatement(statement, ctx.Charset, ctx.Collation)
	if errAdvice != nil {
		return errAdvice, nil
	}

	level, err := advisor.NewStatusBySchemaReviewRuleLevel(ctx.Rule.Level)
	if err != nil {
		return nil, err
	}
	checker := &columnNoNullChecker{level: level}

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

type columnNoNullChecker struct {
	adviceList []advisor.Advice
	level      advisor.Status
}

type columnName struct {
	tableName  string
	columnName string
}

// Enter implements the ast.Visitor interface
func (v *columnNoNullChecker) Enter(in ast.Node) (ast.Node, bool) {
	var columns []columnName
	switch node := in.(type) {
	// CREATE TABLE
	case *ast.CreateTableStmt:
		for _, column := range node.Cols {
			if canNull(column) {
				columns = append(columns, columnName{
					tableName:  node.Table.Name.String(),
					columnName: column.Name.Name.String(),
				})
			}
		}
	// ALTER TABLE
	case *ast.AlterTableStmt:
		for _, spec := range node.Specs {
			switch spec.Tp {
			// ADD COLUMNS
			case ast.AlterTableAddColumns:
				for _, column := range spec.NewColumns {
					if canNull(column) {
						columns = append(columns, columnName{
							tableName:  node.Table.Name.String(),
							columnName: column.Name.Name.String(),
						})
					}
				}
			// CHANGE COLUMN
			case ast.AlterTableChangeColumn:
				if canNull(spec.NewColumns[0]) {
					columns = append(columns, columnName{
						tableName:  node.Table.Name.String(),
						columnName: spec.NewColumns[0].Name.Name.String(),
					})
				}
			}
		}
	}

	for _, column := range columns {
		v.adviceList = append(v.adviceList, advisor.Advice{
			Status:  v.level,
			Code:    common.ColumnCanNull,
			Title:   "Column can have NULL value",
			Content: fmt.Sprintf("`%s`.`%s` can have NULL value", column.tableName, column.columnName),
		})
	}

	return in, false
}

// Leave implements the ast.Visitor interface
func (v *columnNoNullChecker) Leave(in ast.Node) (ast.Node, bool) {
	return in, true
}

func canNull(column *ast.ColumnDef) bool {
	for _, option := range column.Options {
		if option.Tp == ast.ColumnOptionNotNull || option.Tp == ast.ColumnOptionPrimaryKey {
			return false
		}
	}
	return true
}
