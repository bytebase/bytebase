// Package mysql provides the MySQL schema edit plugin.
package mysql

import (
	"bytes"
	"strings"

	"github.com/pingcap/tidb/parser/ast"
	"github.com/pingcap/tidb/parser/format"
	"github.com/pingcap/tidb/parser/model"
	"github.com/pingcap/tidb/parser/mysql"
	"github.com/pingcap/tidb/types"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/api"
	bbparser "github.com/bytebase/bytebase/plugin/parser"

	"github.com/bytebase/bytebase/plugin/parser/edit"
	// Register pingcap parser driver.
	_ "github.com/pingcap/tidb/types/parser_driver"
)

var (
	_ edit.SchemaEditor = (*SchemaEditor)(nil)
)

func init() {
	edit.Register(bbparser.MySQL, &SchemaEditor{})
}

// SchemaEditor it the editor for MySQL dialect.
type SchemaEditor struct{}

// DeparseDatabaseEdit deparses DatabaseEdit to DDL statement.
func (*SchemaEditor) DeparseDatabaseEdit(databaseEdit *api.DatabaseEdit) (string, error) {
	var stmtList []string
	for _, createTableContext := range databaseEdit.CreateTableList {
		createTableStmt := transformCreateTableContext(createTableContext)
		stmt, err := deparseASTNode(createTableStmt)
		if err != nil {
			return "", err
		}
		stmtList = append(stmtList, stmt)
	}
	for _, alterTableContext := range databaseEdit.AlterTableList {
		alterTableStmt := transformAlterTableContext(alterTableContext)
		stmt, err := deparseASTNode(alterTableStmt)
		if err != nil {
			return "", err
		}
		stmtList = append(stmtList, stmt)
	}
	for _, dropTableContext := range databaseEdit.DropTableList {
		dropTableStmt := transformDropTableContext(dropTableContext)
		stmt, err := deparseASTNode(dropTableStmt)
		if err != nil {
			return "", err
		}
		stmtList = append(stmtList, stmt)
	}
	return strings.Join(stmtList, "\n"), nil
}

func transformCreateTableContext(createTableContext *api.CreateTableContext) *ast.CreateTableStmt {
	tableName := &ast.TableName{
		Name: model.NewCIStr(createTableContext.Name),
	}
	createTableStmt := &ast.CreateTableStmt{
		Table: tableName,
	}

	var createTableOptions []*ast.TableOption
	if createTableContext.Engine != "" {
		createTableOptions = append(createTableOptions, &ast.TableOption{
			Tp:       ast.TableOptionEngine,
			StrValue: createTableContext.Engine,
		})
	}
	if createTableContext.CharacterSet != "" {
		createTableOptions = append(createTableOptions, &ast.TableOption{
			Tp:       ast.TableOptionCharset,
			StrValue: createTableContext.CharacterSet,
		})
	}
	if createTableContext.Collation != "" {
		createTableOptions = append(createTableOptions, &ast.TableOption{
			Tp:       ast.TableOptionCollate,
			StrValue: createTableContext.Collation,
		})
	}
	if createTableContext.Comment != "" {
		createTableOptions = append(createTableOptions, &ast.TableOption{
			Tp:       ast.TableOptionComment,
			StrValue: createTableContext.Comment,
		})
	}
	createTableStmt.Options = createTableOptions

	var columnDefs []*ast.ColumnDef
	for _, addColumnContext := range createTableContext.AddColumnList {
		columnDef := transformAddColumnContext(addColumnContext)
		columnDefs = append(columnDefs, columnDef)
	}
	createTableStmt.Cols = columnDefs

	return createTableStmt
}

func transformAlterTableContext(alterTableContext *api.AlterTableContext) *ast.AlterTableStmt {
	tableName := &ast.TableName{
		Name: model.NewCIStr(alterTableContext.Name),
	}
	alterTableStmt := &ast.AlterTableStmt{
		Table: tableName,
		Specs: []*ast.AlterTableSpec{},
	}

	if len(alterTableContext.AddColumnList) > 0 {
		alterTableSpec := &ast.AlterTableSpec{
			Tp:         ast.AlterTableAddColumns,
			NewColumns: []*ast.ColumnDef{},
		}
		for _, addColumnContext := range alterTableContext.AddColumnList {
			alterTableSpec.NewColumns = append(alterTableSpec.NewColumns, transformAddColumnContext(addColumnContext))
		}
		alterTableStmt.Specs = append(alterTableStmt.Specs, alterTableSpec)
	}

	if len(alterTableContext.AlterColumnList) > 0 {
		for _, alterColumnContext := range alterTableContext.AlterColumnList {
			alterTableSpec := &ast.AlterTableSpec{
				Tp: ast.AlterTableModifyColumn,
				Position: &ast.ColumnPosition{
					Tp: ast.ColumnPositionNone,
				},
			}
			alterTableSpec.NewColumns = []*ast.ColumnDef{transformAlterColumnContext(alterColumnContext)}
			alterTableStmt.Specs = append(alterTableStmt.Specs, alterTableSpec)
		}
	}

	if len(alterTableContext.DropColumnList) > 0 {
		for _, dropColumnContext := range alterTableContext.DropColumnList {
			alterTableSpec := &ast.AlterTableSpec{
				Tp: ast.AlterTableDropColumn,
				OldColumnName: &ast.ColumnName{
					Name: model.NewCIStr(dropColumnContext.Name),
				},
			}
			alterTableStmt.Specs = append(alterTableStmt.Specs, alterTableSpec)
		}
	}

	return alterTableStmt
}

func transformDropTableContext(dropTableContext *api.DropTableContext) *ast.DropTableStmt {
	tableName := &ast.TableName{
		Name: model.NewCIStr(dropTableContext.Name),
	}
	dropTableStmt := &ast.DropTableStmt{
		Tables:   []*ast.TableName{tableName},
		IfExists: true,
	}
	return dropTableStmt
}

func transformAddColumnContext(addColumnContext *api.AddColumnContext) *ast.ColumnDef {
	colName := &ast.ColumnName{
		Name: model.NewCIStr(addColumnContext.Name),
	}
	columnDef := &ast.ColumnDef{
		Name: colName,
		Tp:   transformColumnType(addColumnContext.Type),
	}

	var columnOptionList []*ast.ColumnOption
	if addColumnContext.Comment != "" {
		columnOptionList = append(columnOptionList, &ast.ColumnOption{
			Tp:   ast.ColumnOptionComment,
			Expr: ast.NewValueExpr(interface{}(addColumnContext.Comment), addColumnContext.CharacterSet, addColumnContext.Collation),
		})
	}
	if addColumnContext.Collation != "" {
		columnOptionList = append(columnOptionList, &ast.ColumnOption{
			Tp:       ast.ColumnOptionCollate,
			StrValue: addColumnContext.Collation,
		})
	}
	if addColumnContext.Default != nil {
		columnOptionList = append(columnOptionList, &ast.ColumnOption{
			Tp:   ast.ColumnOptionDefaultValue,
			Expr: ast.NewValueExpr(interface{}(*addColumnContext.Default), addColumnContext.CharacterSet, addColumnContext.Collation),
		})
	}
	if !addColumnContext.Nullable {
		columnOptionList = append(columnOptionList, &ast.ColumnOption{
			Tp: ast.ColumnOptionNotNull,
		})
	}
	columnDef.Options = columnOptionList

	return columnDef
}

func transformAlterColumnContext(alterColumnContext *api.AlterColumnContext) *ast.ColumnDef {
	colName := &ast.ColumnName{
		Name: model.NewCIStr(alterColumnContext.Name),
	}
	columnDef := &ast.ColumnDef{
		Name: colName,
		Tp:   transformColumnType(alterColumnContext.Type),
	}

	var columnOptionList []*ast.ColumnOption
	if alterColumnContext.Comment != "" {
		columnOptionList = append(columnOptionList, &ast.ColumnOption{
			Tp:   ast.ColumnOptionComment,
			Expr: ast.NewValueExpr(interface{}(alterColumnContext.Comment), alterColumnContext.CharacterSet, alterColumnContext.Collation),
		})
	}
	if alterColumnContext.Collation != "" {
		columnOptionList = append(columnOptionList, &ast.ColumnOption{
			Tp:       ast.ColumnOptionCollate,
			StrValue: alterColumnContext.Collation,
		})
	}
	if !alterColumnContext.Nullable {
		columnOptionList = append(columnOptionList, &ast.ColumnOption{
			Tp: ast.ColumnOptionNotNull,
		})
	}
	if alterColumnContext.Default != nil {
		columnOptionList = append(columnOptionList, &ast.ColumnOption{
			Tp:   ast.ColumnOptionDefaultValue,
			Expr: ast.NewValueExpr(interface{}(*alterColumnContext.Default), alterColumnContext.CharacterSet, alterColumnContext.Collation),
		})
	}
	columnDef.Options = columnOptionList

	return columnDef
}

func transformColumnType(typeStr string) *types.FieldType {
	colType := types.NewFieldType(getColumnType(typeStr))
	return colType
}

// TODO(steven): Refine the type conversion.
func getColumnType(typeStr string) byte {
	switch typeStr {
	// Maybe it should be a regexp to match like `varchar(%d+)`.
	case "varchar":
		return mysql.TypeVarchar
	case "int":
		return mysql.TypeLong
	case "bigint":
		return mysql.TypeLonglong
	case "json":
		return mysql.TypeJSON
	}

	return mysql.TypeUnspecified
}

func deparseASTNode(node ast.Node) (string, error) {
	var buf bytes.Buffer
	restoreFlag := format.DefaultRestoreFlags | format.RestorePrettyFormat
	if err := node.Restore(format.NewRestoreCtx(restoreFlag, &buf)); err != nil {
		return "", errors.Wrapf(err, "cannot restore node %v", node)
	}

	stmt := buf.String()
	if stmt != "" {
		stmt += ";"
	}
	return stmt, nil
}
