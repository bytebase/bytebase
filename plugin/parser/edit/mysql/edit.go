// Package mysql provides the MySQL schema edit plugin.
package mysql

import (
	"bytes"
	"strings"

	tidbast "github.com/pingcap/tidb/parser/ast"
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
		stmt, err := restoreASTNode(createTableStmt)
		if err != nil {
			return "", err
		}
		stmtList = append(stmtList, stmt)
	}
	for _, alterTableContext := range databaseEdit.AlterTableList {
		alterTableStmt := transformAlterTableContext(alterTableContext)
		stmt, err := restoreASTNode(alterTableStmt)
		if err != nil {
			return "", err
		}
		stmtList = append(stmtList, stmt)
	}
	for _, renameTableContext := range databaseEdit.RenameTableList {
		renameTableStmt := transformRenameTableContext(renameTableContext)
		stmt, err := restoreASTNode(renameTableStmt)
		if err != nil {
			return "", err
		}
		stmtList = append(stmtList, stmt)
	}
	for _, dropTableContext := range databaseEdit.DropTableList {
		dropTableStmt := transformDropTableContext(dropTableContext)
		stmt, err := restoreASTNode(dropTableStmt)
		if err != nil {
			return "", err
		}
		stmtList = append(stmtList, stmt)
	}
	return strings.Join(stmtList, "\n"), nil
}

func transformCreateTableContext(createTableContext *api.CreateTableContext) *tidbast.CreateTableStmt {
	tableName := &tidbast.TableName{
		Name: model.NewCIStr(createTableContext.Name),
	}
	createTableStmt := &tidbast.CreateTableStmt{
		Table: tableName,
	}

	var createTableOptions []*tidbast.TableOption
	if createTableContext.Engine != "" {
		createTableOptions = append(createTableOptions, &tidbast.TableOption{
			Tp:       tidbast.TableOptionEngine,
			StrValue: createTableContext.Engine,
		})
	}
	if createTableContext.CharacterSet != "" {
		createTableOptions = append(createTableOptions, &tidbast.TableOption{
			Tp:       tidbast.TableOptionCharset,
			StrValue: createTableContext.CharacterSet,
		})
	}
	if createTableContext.Collation != "" {
		createTableOptions = append(createTableOptions, &tidbast.TableOption{
			Tp:       tidbast.TableOptionCollate,
			StrValue: createTableContext.Collation,
		})
	}
	if createTableContext.Comment != "" {
		createTableOptions = append(createTableOptions, &tidbast.TableOption{
			Tp:       tidbast.TableOptionComment,
			StrValue: createTableContext.Comment,
		})
	}
	createTableStmt.Options = createTableOptions

	var columnDefs []*tidbast.ColumnDef
	for _, addColumnContext := range createTableContext.AddColumnList {
		columnDef := transformAddColumnContext(addColumnContext)
		columnDefs = append(columnDefs, columnDef)
	}
	createTableStmt.Cols = columnDefs

	return createTableStmt
}

func transformAlterTableContext(alterTableContext *api.AlterTableContext) *tidbast.AlterTableStmt {
	tableName := &tidbast.TableName{
		Name: model.NewCIStr(alterTableContext.Name),
	}
	alterTableStmt := &tidbast.AlterTableStmt{
		Table: tableName,
		Specs: []*tidbast.AlterTableSpec{},
	}

	if len(alterTableContext.AddColumnList) > 0 {
		alterTableSpec := &tidbast.AlterTableSpec{
			Tp:         tidbast.AlterTableAddColumns,
			NewColumns: []*tidbast.ColumnDef{},
		}
		for _, addColumnContext := range alterTableContext.AddColumnList {
			alterTableSpec.NewColumns = append(alterTableSpec.NewColumns, transformAddColumnContext(addColumnContext))
		}
		alterTableStmt.Specs = append(alterTableStmt.Specs, alterTableSpec)
	}

	if len(alterTableContext.ChangeColumnList) > 0 {
		for _, changeColumnContext := range alterTableContext.ChangeColumnList {
			oldColumnName := &tidbast.ColumnName{
				Name: model.NewCIStr(changeColumnContext.OldName),
			}
			newColumnName := &tidbast.ColumnName{
				Name: model.NewCIStr(changeColumnContext.NewName),
			}
			alterTableSpec := &tidbast.AlterTableSpec{
				Tp:            tidbast.AlterTableChangeColumn,
				OldColumnName: oldColumnName,
				NewColumnName: newColumnName,
				// TODO(steven): support modify the column position.
				Position: &tidbast.ColumnPosition{
					Tp: tidbast.ColumnPositionNone,
				},
			}
			alterTableSpec.NewColumns = []*tidbast.ColumnDef{transformChangeColumnContext(changeColumnContext)}
			alterTableStmt.Specs = append(alterTableStmt.Specs, alterTableSpec)
		}
	}

	if len(alterTableContext.DropColumnList) > 0 {
		for _, dropColumnContext := range alterTableContext.DropColumnList {
			alterTableSpec := &tidbast.AlterTableSpec{
				Tp: tidbast.AlterTableDropColumn,
				OldColumnName: &tidbast.ColumnName{
					Name: model.NewCIStr(dropColumnContext.Name),
				},
			}
			alterTableStmt.Specs = append(alterTableStmt.Specs, alterTableSpec)
		}
	}

	return alterTableStmt
}

func transformRenameTableContext(renameTableContext *api.RenameTableContext) *tidbast.RenameTableStmt {
	oldTableName := &tidbast.TableName{
		Name: model.NewCIStr(renameTableContext.OldName),
	}
	newTableName := &tidbast.TableName{
		Name: model.NewCIStr(renameTableContext.NewName),
	}
	renameTableStmt := &tidbast.RenameTableStmt{
		TableToTables: []*tidbast.TableToTable{
			{
				OldTable: oldTableName,
				NewTable: newTableName,
			},
		},
	}
	return renameTableStmt
}

func transformDropTableContext(dropTableContext *api.DropTableContext) *tidbast.DropTableStmt {
	tableName := &tidbast.TableName{
		Name: model.NewCIStr(dropTableContext.Name),
	}
	dropTableStmt := &tidbast.DropTableStmt{
		Tables:   []*tidbast.TableName{tableName},
		IfExists: true,
	}
	return dropTableStmt
}

func transformAddColumnContext(addColumnContext *api.AddColumnContext) *tidbast.ColumnDef {
	colName := &tidbast.ColumnName{
		Name: model.NewCIStr(addColumnContext.Name),
	}
	columnDef := &tidbast.ColumnDef{
		Name: colName,
		Tp:   transformColumnType(addColumnContext.Type),
	}

	var columnOptionList []*tidbast.ColumnOption
	if addColumnContext.Comment != "" {
		columnOptionList = append(columnOptionList, &tidbast.ColumnOption{
			Tp:   tidbast.ColumnOptionComment,
			Expr: tidbast.NewValueExpr(interface{}(addColumnContext.Comment), addColumnContext.CharacterSet, addColumnContext.Collation),
		})
	}
	if addColumnContext.Collation != "" {
		columnOptionList = append(columnOptionList, &tidbast.ColumnOption{
			Tp:       tidbast.ColumnOptionCollate,
			StrValue: addColumnContext.Collation,
		})
	}
	if addColumnContext.Default != nil {
		columnOptionList = append(columnOptionList, &tidbast.ColumnOption{
			Tp:   tidbast.ColumnOptionDefaultValue,
			Expr: tidbast.NewValueExpr(interface{}(*addColumnContext.Default), addColumnContext.CharacterSet, addColumnContext.Collation),
		})
	}
	if !addColumnContext.Nullable {
		columnOptionList = append(columnOptionList, &tidbast.ColumnOption{
			Tp: tidbast.ColumnOptionNotNull,
		})
	}
	columnDef.Options = columnOptionList

	return columnDef
}

func transformChangeColumnContext(changeColumnContext *api.ChangeColumnContext) *tidbast.ColumnDef {
	colName := &tidbast.ColumnName{
		Name: model.NewCIStr(changeColumnContext.NewName),
	}
	columnDef := &tidbast.ColumnDef{
		Name: colName,
		Tp:   transformColumnType(changeColumnContext.Type),
	}

	var columnOptionList []*tidbast.ColumnOption
	if changeColumnContext.Comment != "" {
		columnOptionList = append(columnOptionList, &tidbast.ColumnOption{
			Tp:   tidbast.ColumnOptionComment,
			Expr: tidbast.NewValueExpr(interface{}(changeColumnContext.Comment), changeColumnContext.CharacterSet, changeColumnContext.Collation),
		})
	}
	if changeColumnContext.Collation != "" {
		columnOptionList = append(columnOptionList, &tidbast.ColumnOption{
			Tp:       tidbast.ColumnOptionCollate,
			StrValue: changeColumnContext.Collation,
		})
	}
	if !changeColumnContext.Nullable {
		columnOptionList = append(columnOptionList, &tidbast.ColumnOption{
			Tp: tidbast.ColumnOptionNotNull,
		})
	}
	if changeColumnContext.Default != nil {
		columnOptionList = append(columnOptionList, &tidbast.ColumnOption{
			Tp:   tidbast.ColumnOptionDefaultValue,
			Expr: tidbast.NewValueExpr(interface{}(*changeColumnContext.Default), changeColumnContext.CharacterSet, changeColumnContext.Collation),
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

func restoreASTNode(node tidbast.Node) (string, error) {
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
