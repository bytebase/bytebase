package pg

import (
	"context"
	"fmt"
	"slices"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/catalog"
	"github.com/bytebase/bytebase/backend/plugin/parser/pg/legacy/ast"
)

var (
	_ advisor.Advisor = (*ColumnNoNullAdvisor)(nil)
	_ ast.Visitor     = (*columnNoNullChecker)(nil)
)

func init() {
	advisor.Register(storepb.Engine_POSTGRES, advisor.SchemaRuleColumnNotNull, &ColumnNoNullAdvisor{})
}

// ColumnNoNullAdvisor is the advisor checking for column no NULL value.
type ColumnNoNullAdvisor struct {
}

// Check checks for column no NULL value.
func (*ColumnNoNullAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	stmts, ok := checkCtx.AST.([]ast.Node)
	if !ok {
		return nil, errors.Errorf("failed to convert to Node")
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	checker := &columnNoNullChecker{
		level:           level,
		title:           string(checkCtx.Rule.Type),
		catalog:         checkCtx.Catalog,
		nullableColumns: make(columnMap),
	}

	for _, stmt := range stmts {
		ast.Walk(checker, stmt)
	}

	return checker.generateAdviceList(), nil
}

type columnNoNullChecker struct {
	adviceList      []*storepb.Advice
	level           storepb.Advice_Status
	title           string
	catalog         *catalog.Finder
	nullableColumns columnMap
}

func (checker *columnNoNullChecker) generateAdviceList() []*storepb.Advice {
	var columnList []columnName
	for column := range checker.nullableColumns {
		columnList = append(columnList, column)
	}
	if len(columnList) > 0 {
		// Order it cause the random iteration order in Go, see https://go.dev/blog/maps
		slices.SortFunc(columnList, func(i, j columnName) int {
			if i.schema != j.schema {
				if i.schema < j.schema {
					return -1
				}
				return 1
			}
			if i.table != j.table {
				if i.table < j.table {
					return -1
				}
				return 1
			}
			if i.column < j.column {
				return -1
			}
			if i.column > j.column {
				return 1
			}
			return 0
		})
	}
	for _, column := range columnList {
		checker.adviceList = append(checker.adviceList, &storepb.Advice{
			Status:        checker.level,
			Code:          advisor.ColumnCannotNull.Int32(),
			Title:         checker.title,
			Content:       fmt.Sprintf(`Column "%s" in %s cannot have NULL value`, column.column, column.normalizeTableName()),
			StartPosition: common.ConvertPGParserLineToPosition(checker.nullableColumns[column]),
		})
	}

	return checker.adviceList
}

// Visit implements the ast.Visitor interface.
func (checker *columnNoNullChecker) Visit(node ast.Node) ast.Visitor {
	switch n := node.(type) {
	// CREATE TABLE
	case *ast.CreateTableStmt:
		for _, column := range n.ColumnList {
			checker.addColumn(n.Name, column.ColumnName, column.LastLine())
			checker.removeColumnByConstraintList(n.Name, column.ConstraintList)
		}
		checker.removeColumnByConstraintList(n.Name, n.ConstraintList)
	// ALTER TABLE
	case *ast.AlterTableStmt:
		for _, item := range n.AlterItemList {
			switch cmd := item.(type) {
			// ALTER TABLE ADD COLUMN
			case *ast.AddColumnListStmt:
				for _, column := range cmd.ColumnList {
					checker.addColumn(n.Table, column.ColumnName, n.LastLine())
					checker.removeColumnByConstraintList(n.Table, column.ConstraintList)
				}
			// ALTER TABLE ALTER COLUMN SET NOT NULL
			case *ast.SetNotNullStmt:
				checker.removeColumn(n.Table, cmd.ColumnName)
			// ALTER TABLE ALTER COLUMN DROP NOT NULL
			case *ast.DropNotNullStmt:
				checker.addColumn(n.Table, cmd.ColumnName, n.LastLine())
			// ALTER TABLE ADD CONSTRAINT
			case *ast.AddConstraintStmt:
				checker.removeColumnByConstraintList(n.Table, []*ast.ConstraintDef{cmd.Constraint})
			}
		}
	}

	return checker
}

func (checker *columnNoNullChecker) addColumn(table *ast.TableDef, column string, line int) {
	checker.nullableColumns[convertToColumnName(table, column)] = line
}

func (checker *columnNoNullChecker) removeColumn(table *ast.TableDef, column string) {
	delete(checker.nullableColumns, convertToColumnName(table, column))
}

func (checker *columnNoNullChecker) removeColumnByConstraintList(table *ast.TableDef, constraintList []*ast.ConstraintDef) {
	for _, constraint := range constraintList {
		switch constraint.Type {
		case ast.ConstraintTypeNotNull, ast.ConstraintTypePrimary:
			for _, column := range constraint.KeyList {
				checker.removeColumn(table, column)
			}
		case ast.ConstraintTypePrimaryUsingIndex:
			_, index := checker.catalog.Origin.FindIndex(&catalog.IndexFind{
				SchemaName: normalizeSchemaName(table.Schema),
				TableName:  table.Name,
				IndexName:  constraint.IndexName,
			})
			if index == nil {
				continue
			}
			for _, expression := range index.ExpressionList() {
				checker.removeColumn(table, expression)
			}
		default:
			// Other constraint types
		}
	}
}

func convertToColumnName(table *ast.TableDef, column string) columnName {
	colName := columnName{
		schema: table.Schema,
		table:  table.Name,
		column: column,
	}
	if colName.schema == "" {
		colName.schema = PostgreSQLPublicSchema
	}
	return colName
}
