package tidb

import (
	"context"
	"fmt"
	"slices"

	"github.com/pingcap/tidb/pkg/parser/ast"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/catalog"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
)

var (
	_ advisor.Advisor = (*ColumnNoNullAdvisor)(nil)
	_ ast.Visitor     = (*columnNoNullChecker)(nil)
)

func init() {
	advisor.Register(storepb.Engine_TIDB, advisor.SchemaRuleColumnNotNull, &ColumnNoNullAdvisor{})
}

// ColumnNoNullAdvisor is the advisor checking for column no NULL value.
type ColumnNoNullAdvisor struct {
}

// Check checks for column no NULL value.
func (*ColumnNoNullAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	root, ok := checkCtx.AST.([]ast.StmtNode)
	if !ok {
		return nil, errors.Errorf("failed to convert to StmtNode")
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}
	checker := &columnNoNullChecker{
		level:        level,
		title:        string(checkCtx.Rule.Type),
		columnSet:    make(map[string]columnName),
		finalCatalog: checkCtx.FinalCatalog,
	}

	for _, stmtNode := range root {
		(stmtNode).Accept(checker)
	}

	return checker.generateAdvice(), nil
}

type columnNoNullChecker struct {
	adviceList   []*storepb.Advice
	level        storepb.Advice_Status
	title        string
	columnSet    map[string]columnName
	finalCatalog *catalog.DatabaseState
}

func (checker *columnNoNullChecker) generateAdvice() []*storepb.Advice {
	var columnList []columnName
	for _, column := range checker.columnSet {
		columnList = append(columnList, column)
	}
	slices.SortFunc(columnList, func(i, j columnName) int {
		if i.line != j.line {
			if i.line < j.line {
				return -1
			}
			return 1
		}
		if i.columnName < j.columnName {
			return -1
		}
		if i.columnName > j.columnName {
			return 1
		}
		return 0
	})

	for _, column := range columnList {
		col := checker.finalCatalog.GetColumn("", column.tableName, column.columnName)
		if col != nil && col.Nullable() {
			checker.adviceList = append(checker.adviceList, &storepb.Advice{
				Status:        checker.level,
				Code:          code.ColumnCannotNull.Int32(),
				Title:         checker.title,
				Content:       fmt.Sprintf("`%s`.`%s` cannot have NULL value", column.tableName, column.columnName),
				StartPosition: common.ConvertANTLRLineToPosition(column.line),
			})
		}
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
			default:
				// Skip other alter table specification types
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
