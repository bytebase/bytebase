// Package pg provides the PostgreSQL differ plugin.
package pg

import (
	"bytes"
	"io"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/plugin/parser"
	"github.com/bytebase/bytebase/plugin/parser/ast"
	"github.com/bytebase/bytebase/plugin/parser/differ"
)

var (
	_ differ.SchemaDiffer = (*SchemaDiffer)(nil)
)

func init() {
	differ.Register(parser.Postgres, &SchemaDiffer{})
}

// SchemaDiffer it the parser for PostgreSQL dialect.
type SchemaDiffer struct {
}

type diffNode struct {
	newSchemaList   []*ast.CreateSchemaStmt
	newTableList    []*ast.CreateTableStmt
	modifyTableList []ast.Node
	dropNodeList    []ast.Node
}

// SchemaDiff computes the schema differences between old and new schema.
func (*SchemaDiffer) SchemaDiff(oldStmt, newStmt string) (string, error) {
	oldNodes, err := parser.Parse(parser.Postgres, parser.ParseContext{}, oldStmt)
	if err != nil {
		return "", errors.Wrapf(err, "failed to parse old statement %q", oldStmt)
	}
	newNodes, err := parser.Parse(parser.Postgres, parser.ParseContext{}, newStmt)
	if err != nil {
		return "", errors.Wrapf(err, "failed to parse new statement %q", newStmt)
	}

	oldTableMap := make(map[string]*ast.CreateTableStmt)
	oldSchemaMap := make(map[string]*ast.CreateSchemaStmt)
	var oldSchemaList []*ast.CreateSchemaStmt
	for _, node := range oldNodes {
		switch stmt := node.(type) {
		case *ast.CreateSchemaStmt:
			oldSchemaMap[stmt.Name] = stmt
			oldSchemaList = append(oldSchemaList, stmt)
		case *ast.CreateTableStmt:
			oldTableMap[stmt.Name.Name] = stmt
		}
	}

	diff := &diffNode{}
	for _, node := range newNodes {
		switch stmt := node.(type) {
		case *ast.CreateTableStmt:
			oldTable, hasTable := oldTableMap[stmt.Name.Name]
			// Add the new table.
			if !hasTable {
				diff.newTableList = append(diff.newTableList, stmt)
				continue
			}
			// Modify the table.
			if err := diff.modifyTable(oldTable, stmt); err != nil {
				return "", err
			}
		case *ast.CreateSchemaStmt:
			if _, hasSchema := oldSchemaMap[stmt.Name]; !hasSchema {
				diff.newSchemaList = append(diff.newSchemaList, stmt)
				continue
			}
			delete(oldSchemaMap, stmt.Name)
		default:
			return "", errors.Errorf("unsupported statement %+v", stmt)
		}
	}

	// Drop the remaining old schemata.
	if dropSchemaStmt := dropSchemata(oldSchemaMap, oldSchemaList); dropSchemaStmt != nil {
		diff.dropNodeList = append(diff.dropNodeList, dropSchemaStmt)
	}

	return diff.deparse()
}

func (diff *diffNode) modifyTable(oldTable *ast.CreateTableStmt, newTable *ast.CreateTableStmt) error {
	tableName := oldTable.Name

	// Modify table for columns.
	oldColumnMap := make(map[string]*ast.ColumnDef)
	for _, column := range oldTable.ColumnList {
		oldColumnMap[column.ColumnName] = column
	}

	alterTableStmt := &ast.AlterTableStmt{
		Table: tableName,
	}
	for _, newColumn := range newTable.ColumnList {
		oldColumn, exists := oldColumnMap[newColumn.ColumnName]
		// Add the new column.
		if !exists {
			alterTableStmt.AlterItemList = append(alterTableStmt.AlterItemList, &ast.AddColumnListStmt{
				Table:      tableName,
				ColumnList: []*ast.ColumnDef{newColumn},
			})
			continue
		}
		// Modify the column.
		if err := diff.modifyColumn(alterTableStmt, oldColumn, newColumn); err != nil {
			return err
		}
		delete(oldColumnMap, oldColumn.ColumnName)
	}

	for _, oldColumn := range oldTable.ColumnList {
		if _, exists := oldColumnMap[oldColumn.ColumnName]; exists {
			alterTableStmt.AlterItemList = append(alterTableStmt.AlterItemList, &ast.DropColumnStmt{
				Table:      alterTableStmt.Table,
				ColumnName: oldColumn.ColumnName,
			})
		}
	}

	if len(alterTableStmt.AlterItemList) > 0 {
		diff.modifyTableList = append(diff.modifyTableList, alterTableStmt)
	}

	return nil
}

func (*diffNode) modifyColumn(alterTableStmt *ast.AlterTableStmt, oldColumn *ast.ColumnDef, newColumn *ast.ColumnDef) error {
	columnName := oldColumn.ColumnName
	// compare the data type
	equivalent, err := equivalentType(oldColumn.Type, newColumn.Type)
	if err != nil {
		return err
	}
	if !equivalent {
		alterTableStmt.AlterItemList = append(alterTableStmt.AlterItemList, &ast.AlterColumnTypeStmt{
			Table:      alterTableStmt.Table,
			ColumnName: columnName,
			Type:       newColumn.Type,
		})
	}
	// compare the NOT NULL
	oldNotNull := hasNotNull(oldColumn)
	newNotNull := hasNotNull(newColumn)
	needSetNotNull := !oldNotNull && newNotNull
	needDropNotNull := oldNotNull && !newNotNull
	if needSetNotNull {
		alterTableStmt.AlterItemList = append(alterTableStmt.AlterItemList, &ast.SetNotNullStmt{
			Table:      alterTableStmt.Table,
			ColumnName: columnName,
		})
	} else if needDropNotNull {
		alterTableStmt.AlterItemList = append(alterTableStmt.AlterItemList, &ast.DropNotNullStmt{
			Table:      alterTableStmt.Table,
			ColumnName: columnName,
		})
	}
	// compare the DEFAULT
	oldDefault, oldHasDefault := getDefault(oldColumn)
	newDefault, newHasDefault := getDefault(newColumn)
	needSetDefault := (!oldHasDefault && newHasDefault) || (oldHasDefault && newHasDefault && oldDefault != newDefault)
	needDropDefault := oldHasDefault && !newHasDefault
	if needSetDefault {
		expression := &ast.UnconvertedExpressionDef{}
		expression.SetText(newDefault)
		alterTableStmt.AlterItemList = append(alterTableStmt.AlterItemList, &ast.SetDefaultStmt{
			Table:      alterTableStmt.Table,
			ColumnName: columnName,
			Expression: expression,
		})
	} else if needDropDefault {
		alterTableStmt.AlterItemList = append(alterTableStmt.AlterItemList, &ast.DropDefaultStmt{
			Table:      alterTableStmt.Table,
			ColumnName: columnName,
		})
	}

	// TODO(rebelice): compare other column properties
	return nil
}

func getDefault(column *ast.ColumnDef) (string, bool) {
	for _, constraint := range column.ConstraintList {
		if constraint.Type == ast.ConstraintTypeDefault {
			return constraint.Expression.Text(), true
		}
	}
	return "", false
}

func hasNotNull(column *ast.ColumnDef) bool {
	for _, constraint := range column.ConstraintList {
		if constraint.Type == ast.ConstraintTypeNotNull {
			return true
		}
	}
	return false
}

func equivalentType(typeA ast.DataType, typeB ast.DataType) (bool, error) {
	typeStringA, err := parser.Deparse(parser.Postgres, parser.DeparseContext{}, typeA)
	if err != nil {
		return false, err
	}
	typeStringB, err := parser.Deparse(parser.Postgres, parser.DeparseContext{}, typeB)
	if err != nil {
		return false, err
	}
	return typeStringA == typeStringB, nil
}

func (diff *diffNode) deparse() (string, error) {
	var buf bytes.Buffer
	for _, newSchema := range diff.newSchemaList {
		newSchema.IfNotExists = true
		sql, err := parser.Deparse(parser.Postgres, parser.DeparseContext{}, newSchema)
		if err != nil {
			return "", err
		}
		if err := writeStringWithNewLine(&buf, sql); err != nil {
			return "", err
		}
	}

	for _, newTable := range diff.newTableList {
		if err := writeStringWithNewLine(&buf, newTable.Text()); err != nil {
			return "", err
		}
	}

	for _, modifyTable := range diff.modifyTableList {
		sql, err := parser.Deparse(parser.Postgres, parser.DeparseContext{}, modifyTable)
		if err != nil {
			return "", err
		}
		if err := writeStringWithNewLine(&buf, sql); err != nil {
			return "", err
		}
	}

	// Deparse the drop node in reverse order.
	for i := len(diff.dropNodeList) - 1; i >= 0; i-- {
		sql, err := parser.Deparse(parser.Postgres, parser.DeparseContext{}, diff.dropNodeList[i])
		if err != nil {
			return "", err
		}
		if err := writeStringWithNewLine(&buf, sql); err != nil {
			return "", err
		}
	}
	return buf.String(), nil
}

func dropSchemata(m map[string]*ast.CreateSchemaStmt, l []*ast.CreateSchemaStmt) *ast.DropSchemaStmt {
	var schemaNames []string
	for _, stmt := range l {
		if _, ok := m[stmt.Name]; ok {
			schemaNames = append(schemaNames, stmt.Name)
		}
	}
	if len(schemaNames) == 0 {
		return nil
	}
	return &ast.DropSchemaStmt{
		IfExists:   true,
		SchemaList: schemaNames,
	}
}

func writeStringWithNewLine(out io.Writer, str string) error {
	if _, err := out.Write([]byte(str)); err != nil {
		return err
	}
	if _, err := out.Write([]byte("\n")); err != nil {
		return err
	}
	return nil
}
