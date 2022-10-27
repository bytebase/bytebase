package server

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
	"github.com/bytebase/bytebase/plugin/db"
)

func restoreDatabaseEdit(databaseEdit *api.DatabaseEdit) (string, error) {
	var stmtList []string

	for _, createTableContext := range databaseEdit.CreateTableList {
		createTableStmt := transformCreateTableContext(createTableContext, databaseEdit.EngineType)
		stmt, err := restoreASTNode(createTableStmt)
		if err != nil {
			return "", err
		}
		stmtList = append(stmtList, stmt)
	}

	return strings.Join(stmtList, "\n"), nil
}

func transformCreateTableContext(createTableContext *api.CreateTableContext, engineType db.Type) *ast.CreateTableStmt {
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
		columnDef := transformAddColumnContext(addColumnContext, engineType)
		columnDefs = append(columnDefs, columnDef)
	}
	createTableStmt.Cols = columnDefs

	return createTableStmt
}

func transformAddColumnContext(addColumnContext *api.AddColumnContext, engineType db.Type) *ast.ColumnDef {
	colName := &ast.ColumnName{
		Name: model.NewCIStr(addColumnContext.Name),
	}
	columnDef := &ast.ColumnDef{
		Name: colName,
	}

	var colType *types.FieldType
	if engineType == db.MySQL {
		colType = types.NewFieldType(transformColumnTypeForMySQL(addColumnContext.Type))
	} else {
		types.DefaultTypeForValue(interface{}(addColumnContext.Type), colType, addColumnContext.CharacterSet, addColumnContext.Collation)
	}
	columnDef.Tp = colType

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

func transformColumnTypeForMySQL(typeStr string) byte {
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

func restoreASTNode(node ast.Node) (string, error) {
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
