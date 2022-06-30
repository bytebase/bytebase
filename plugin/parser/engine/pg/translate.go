package pg

import (
	"fmt"

	"github.com/bytebase/bytebase/plugin/parser/ast"
	pg_query "github.com/pganalyze/pg_query_go/v2"
)

// translate translates the pg_query.Node to ast.Node.
func translate(node *pg_query.Node) (ast.Node, error) {
	switch in := node.Node.(type) {
	case *pg_query.Node_AlterTableStmt:
		alterTable := &ast.AlterTableStmt{
			Table: convertRangeVarToTableName(in.AlterTableStmt.Relation),
			Cmds:  []ast.Node{},
		}
		for _, cmd := range in.AlterTableStmt.Cmds {
			if cmdNode, ok := cmd.Node.(*pg_query.Node_AlterTableCmd); ok {
				alterCmd := cmdNode.AlterTableCmd

				switch alterCmd.Subtype {
				case pg_query.AlterTableType_AT_AddColumn:
					def, ok := alterCmd.Def.Node.(*pg_query.Node_ColumnDef)
					if !ok {
						return nil, fmt.Errorf("expected ColumnDef but found %t", alterCmd.Def.Node)
					}

					addColumn := &ast.AddColumnsStmt{
						Table: alterTable.Table,
						Columns: []*ast.ColumnDef{
							{
								ColumnName: def.ColumnDef.Colname,
							},
						},
					}

					alterTable.Cmds = append(alterTable.Cmds, addColumn)
				case pg_query.AlterTableType_AT_AddIndex:
					// TODO.
					// Trick linter to use switch...case...
				}
			}
		}
		return alterTable, nil
	case *pg_query.Node_CreateStmt:
		stmt := &ast.CreateTableStmt{
			IfNotExists: in.CreateStmt.IfNotExists,
			Name:        convertRangeVarToTableName(in.CreateStmt.Relation),
		}

		for _, elt := range in.CreateStmt.TableElts {
			if colDef, ok := elt.Node.(*pg_query.Node_ColumnDef); ok {
				stmt.Cols = append(stmt.Cols, &ast.ColumnDef{
					ColumnName: colDef.ColumnDef.Colname,
				})
			}
		}
		return stmt, nil
	case *pg_query.Node_RenameStmt:
		switch in.RenameStmt.RenameType {
		case pg_query.ObjectType_OBJECT_COLUMN:
			return &ast.RenameColumnStmt{
				Table:   convertRangeVarToTableName(in.RenameStmt.Relation),
				Column:  &ast.ColumnDef{ColumnName: in.RenameStmt.Subname},
				NewName: in.RenameStmt.Newname,
			}, nil
		case pg_query.ObjectType_OBJECT_TABLE:
			return &ast.RenameTableStmt{
				Table:   convertRangeVarToTableName(in.RenameStmt.Relation),
				NewName: in.RenameStmt.Newname,
			}, nil
		}
	}

	return nil, nil
}

func convertRangeVarToTableName(in *pg_query.RangeVar) *ast.TableName {
	return &ast.TableName{
		Catalog: in.Catalogname,
		Schema:  in.Schemaname,
		Name:    in.Relname,
	}
}
