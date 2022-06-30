package pg

import (
	"fmt"

	"github.com/bytebase/bytebase/plugin/parser/ast"
	pgquery "github.com/pganalyze/pg_query_go/v2"
)

// convert converts the pg_query.Node to ast.Node.
func convert(node *pgquery.Node) (ast.Node, error) {
	switch in := node.Node.(type) {
	case *pgquery.Node_AlterTableStmt:
		alterTable := &ast.AlterTableStmt{
			Table:         convertRangeVarToTableName(in.AlterTableStmt.Relation),
			AlterItemList: []ast.Node{},
		}
		for _, cmd := range in.AlterTableStmt.Cmds {
			if cmdNode, ok := cmd.Node.(*pgquery.Node_AlterTableCmd); ok {
				alterCmd := cmdNode.AlterTableCmd

				switch alterCmd.Subtype {
				case pgquery.AlterTableType_AT_AddColumn:
					def, ok := alterCmd.Def.Node.(*pgquery.Node_ColumnDef)
					if !ok {
						return nil, fmt.Errorf("expected ColumnDef but found %t", alterCmd.Def.Node)
					}

					addColumn := &ast.AddColumnListStmt{
						Table: alterTable.Table,
						ColumnList: []*ast.ColumnDef{
							{
								ColumnName: def.ColumnDef.Colname,
							},
						},
					}

					alterTable.AlterItemList = append(alterTable.AlterItemList, addColumn)
				case pgquery.AlterTableType_AT_AddIndex:
					// TODO.
					// Trick linter to use switch...case...
				}
			}
		}
		return alterTable, nil
	case *pgquery.Node_CreateStmt:
		stmt := &ast.CreateTableStmt{
			IfNotExists: in.CreateStmt.IfNotExists,
			Name:        convertRangeVarToTableName(in.CreateStmt.Relation),
		}

		for _, elt := range in.CreateStmt.TableElts {
			if colDef, ok := elt.Node.(*pgquery.Node_ColumnDef); ok {
				stmt.ColumnList = append(stmt.ColumnList, &ast.ColumnDef{
					ColumnName: colDef.ColumnDef.Colname,
				})
			}
		}
		return stmt, nil
	case *pgquery.Node_RenameStmt:
		switch in.RenameStmt.RenameType {
		case pgquery.ObjectType_OBJECT_COLUMN:
			return &ast.RenameColumnStmt{
				Table:   convertRangeVarToTableName(in.RenameStmt.Relation),
				Column:  &ast.ColumnDef{ColumnName: in.RenameStmt.Subname},
				NewName: in.RenameStmt.Newname,
			}, nil
		case pgquery.ObjectType_OBJECT_TABLE:
			return &ast.RenameTableStmt{
				Table:   convertRangeVarToTableName(in.RenameStmt.Relation),
				NewName: in.RenameStmt.Newname,
			}, nil
		}
	}

	return nil, nil
}

func convertRangeVarToTableName(in *pgquery.RangeVar) *ast.TableDef {
	return &ast.TableDef{
		Database: in.Catalogname,
		Schema:   in.Schemaname,
		Name:     in.Relname,
	}
}
