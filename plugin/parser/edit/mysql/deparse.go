package mysql

import (
	"bytes"
	"fmt"
	"strings"

	tidbparser "github.com/pingcap/tidb/parser"
	tidbast "github.com/pingcap/tidb/parser/ast"
	"github.com/pingcap/tidb/parser/format"
	"github.com/pingcap/tidb/parser/model"
	"github.com/pingcap/tidb/types"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/api"
)

// DeparseDatabaseEdit deparses DatabaseEdit to DDL statement.
func (*SchemaEditor) DeparseDatabaseEdit(databaseEdit *api.DatabaseEdit) (string, error) {
	var stmtList []string
	for _, createTableContext := range databaseEdit.CreateTableList {
		createTableStmt, err := transformCreateTableContext(createTableContext)
		if err != nil {
			return "", err
		}

		stmt, err := restoreASTNode(createTableStmt)
		if err != nil {
			return "", err
		}

		stmtList = append(stmtList, stmt)
	}
	for _, alterTableContext := range databaseEdit.AlterTableList {
		alterTableStmt, err := transformAlterTableContext(alterTableContext)
		if err != nil {
			return "", err
		}

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

func transformCreateTableContext(createTableContext *api.CreateTableContext) (*tidbast.CreateTableStmt, error) {
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
		column, err := transformAddColumnContext(addColumnContext)
		if err != nil {
			return nil, err
		}
		columnDefs = append(columnDefs, column)
	}
	createTableStmt.Cols = columnDefs

	return createTableStmt, nil
}

func transformAlterTableContext(alterTableContext *api.AlterTableContext) (*tidbast.AlterTableStmt, error) {
	tableName := &tidbast.TableName{
		Name: model.NewCIStr(alterTableContext.Name),
	}
	alterTableStmt := &tidbast.AlterTableStmt{
		Table: tableName,
		Specs: []*tidbast.AlterTableSpec{},
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

	if len(alterTableContext.AddColumnList) > 0 {
		alterTableSpec := &tidbast.AlterTableSpec{
			Tp:         tidbast.AlterTableAddColumns,
			NewColumns: []*tidbast.ColumnDef{},
		}
		for _, addColumnContext := range alterTableContext.AddColumnList {
			column, err := transformAddColumnContext(addColumnContext)
			if err != nil {
				return nil, err
			}
			alterTableSpec.NewColumns = append(alterTableSpec.NewColumns, column)
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
			column, err := transformChangeColumnContext(changeColumnContext)
			if err != nil {
				return nil, err
			}
			alterTableSpec.NewColumns = []*tidbast.ColumnDef{column}
			alterTableStmt.Specs = append(alterTableStmt.Specs, alterTableSpec)
		}
	}

	return alterTableStmt, nil
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

func transformAddColumnContext(addColumnContext *api.AddColumnContext) (*tidbast.ColumnDef, error) {
	colName := &tidbast.ColumnName{
		Name: model.NewCIStr(addColumnContext.Name),
	}
	columnType, err := transformColumnType(addColumnContext.Type)
	if err != nil {
		return nil, errors.Wrap(err, "failed to transform column type")
	}

	columnDef := &tidbast.ColumnDef{
		Name: colName,
		Tp:   columnType,
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

	return columnDef, nil
}

func transformChangeColumnContext(changeColumnContext *api.ChangeColumnContext) (*tidbast.ColumnDef, error) {
	colName := &tidbast.ColumnName{
		Name: model.NewCIStr(changeColumnContext.NewName),
	}
	columnType, err := transformColumnType(changeColumnContext.Type)
	if err != nil {
		return nil, errors.Wrap(err, "failed to transform column type")
	}

	columnDef := &tidbast.ColumnDef{
		Name: colName,
		Tp:   columnType,
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

	return columnDef, nil
}

func transformColumnType(typeStr string) (*types.FieldType, error) {
	// Mock a CreateTableStmt with type string to get the actually types.FieldType.
	stmt := fmt.Sprintf("CREATE TABLE column_type(column_type %s);", typeStr)
	nodeList, _, err := tidbparser.New().Parse(stmt, "", "")
	if err != nil {
		return nil, err
	}
	if len(nodeList) != 1 {
		return nil, errors.Errorf("expect node list length to be 1, get %d", len(nodeList))
	}

	node, ok := nodeList[0].(*tidbast.CreateTableStmt)
	if !ok {
		return nil, errors.New("expect the type of the node to be CreateTableStmt")
	}
	if len(node.Cols) != 1 {
		return nil, errors.Errorf("expect node list length to be 1, get %d", len(nodeList))
	}

	col := node.Cols[0]
	return col.Tp, nil
}

func restoreASTNode(node tidbast.Node) (string, error) {
	var buf bytes.Buffer
	restoreFlag := format.DefaultRestoreFlags | format.RestorePrettyFormat
	if err := node.Restore(format.NewRestoreCtx(restoreFlag, &buf)); err != nil {
		return "", errors.Wrapf(err, "cannot restore node %v", node)
	}

	stmt := buf.String()
	if stmt != "" {
		stmt += ";\n"
	}
	return stmt, nil
}
