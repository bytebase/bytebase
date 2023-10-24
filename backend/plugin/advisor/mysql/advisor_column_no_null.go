package mysql

import (
	"fmt"
	"sort"

	"github.com/pingcap/tidb/parser/ast"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/catalog"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

var (
	_ advisor.Advisor = (*ColumnNoNullAdvisor)(nil)
	_ ast.Visitor     = (*columnNoNullChecker)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MYSQL, advisor.MySQLColumnNoNull, &ColumnNoNullAdvisor{})
	advisor.Register(storepb.Engine_MARIADB, advisor.MySQLColumnNoNull, &ColumnNoNullAdvisor{})
	advisor.Register(storepb.Engine_OCEANBASE, advisor.MySQLColumnNoNull, &ColumnNoNullAdvisor{})
}

// ColumnNoNullAdvisor is the advisor checking for column no NULL value.
type ColumnNoNullAdvisor struct {
}

// Check checks for column no NULL value.
func (*ColumnNoNullAdvisor) Check(ctx advisor.Context, _ string) ([]advisor.Advice, error) {
	root, ok := ctx.AST.([]ast.StmtNode)
	if !ok {
		return nil, errors.Errorf("failed to convert to StmtNode")
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(ctx.Rule.Level)
	if err != nil {
		return nil, err
	}
	checker := &columnNoNullChecker{
		level:     level,
		title:     string(ctx.Rule.Type),
		columnSet: make(map[string]columnName),
		catalog:   ctx.Catalog,
	}

	for _, stmtNode := range root {
		(stmtNode).Accept(checker)
	}

	return checker.generateAdvice(), nil
}

type columnNoNullChecker struct {
	adviceList []advisor.Advice
	level      advisor.Status
	title      string
	columnSet  map[string]columnName
	catalog    *catalog.Finder
}

func (checker columnNoNullChecker) generateAdvice() []advisor.Advice {
	var columnList []columnName
	for _, column := range checker.columnSet {
		columnList = append(columnList, column)
	}
	sort.Slice(columnList, func(i, j int) bool {
		if columnList[i].line != columnList[j].line {
			return columnList[i].line < columnList[j].line
		}
		return columnList[i].columnName < columnList[j].columnName
	})

	for _, column := range columnList {
		col := checker.catalog.Final.FindColumn(&catalog.ColumnFind{
			TableName:  column.tableName,
			ColumnName: column.columnName,
		})
		if col != nil && col.Nullable() {
			checker.adviceList = append(checker.adviceList, advisor.Advice{
				Status:  checker.level,
				Code:    advisor.ColumnCannotNull,
				Title:   checker.title,
				Content: fmt.Sprintf("`%s`.`%s` cannot have NULL value", column.tableName, column.columnName),
				Line:    column.line,
			})
		}
	}

	if len(checker.adviceList) == 0 {
		checker.adviceList = append(checker.adviceList, advisor.Advice{
			Status:  advisor.Success,
			Code:    advisor.Ok,
			Title:   "OK",
			Content: "",
		})
	}
	return checker.adviceList
}

type columnName struct {
	tableName  string
	columnName string
	line       int
}

func (c columnName) name() string {
	return fmt.Sprintf("%s.%s", c.tableName, c.columnName)
}

// Enter implements the ast.Visitor interface.
func (checker *columnNoNullChecker) Enter(in ast.Node) (ast.Node, bool) {
	switch node := in.(type) {
	// CREATE TABLE
	case *ast.CreateTableStmt:
		for _, column := range node.Cols {
			col := columnName{
				tableName:  node.Table.Name.O,
				columnName: column.Name.Name.O,
				line:       column.OriginTextPosition(),
			}
			if _, exists := checker.columnSet[col.name()]; !exists {
				checker.columnSet[col.name()] = col
			}
		}
	// ALTER TABLE
	case *ast.AlterTableStmt:
		for _, spec := range node.Specs {
			switch spec.Tp {
			// ADD COLUMNS
			case ast.AlterTableAddColumns:
				for _, column := range spec.NewColumns {
					col := columnName{
						tableName:  node.Table.Name.O,
						columnName: column.Name.Name.O,
						line:       node.OriginTextPosition(),
					}
					if _, exists := checker.columnSet[col.name()]; !exists {
						checker.columnSet[col.name()] = col
					}
				}
			// CHANGE COLUMN
			case ast.AlterTableChangeColumn:
				col := columnName{
					tableName:  node.Table.Name.O,
					columnName: spec.NewColumns[0].Name.Name.O,
					line:       node.OriginTextPosition(),
				}
				if _, exists := checker.columnSet[col.name()]; !exists {
					checker.columnSet[col.name()] = col
				}
			}
		}
	}

	return in, false
}

// Leave implements the ast.Visitor interface.
func (*columnNoNullChecker) Leave(in ast.Node) (ast.Node, bool) {
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
