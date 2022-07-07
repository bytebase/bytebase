//go:build !release
// +build !release

package pg

import (
	"github.com/bytebase/bytebase/plugin/parser"
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
						return nil, parser.NewConvertErrorf("expected ColumnDef but found %t", alterCmd.Def.Node)
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
		table := &ast.CreateTableStmt{
			IfNotExists: in.CreateStmt.IfNotExists,
			Name:        convertRangeVarToTableName(in.CreateStmt.Relation),
		}

		for _, elt := range in.CreateStmt.TableElts {
			switch item := elt.Node.(type) {
			case *pgquery.Node_ColumnDef:
				column := &ast.ColumnDef{
					ColumnName: item.ColumnDef.Colname,
				}

				for _, cons := range item.ColumnDef.Constraints {
					constraint, ok := cons.Node.(*pgquery.Node_Constraint)
					if !ok {
						return nil, parser.NewConvertErrorf("expected Constraint but found %t", cons.Node)
					}
					columnCons, err := convertConstraint(constraint)
					if err != nil {
						return nil, err
					}
					columnCons.KeyList = append(columnCons.KeyList, item.ColumnDef.Colname)
					column.ConstraintList = append(column.ConstraintList, columnCons)
				}

				table.ColumnList = append(table.ColumnList, column)
			case *pgquery.Node_Constraint:
				cons, err := convertConstraint(item)
				if err != nil {
					return nil, err
				}
				table.ConstraintList = append(table.ConstraintList, cons)
			}
		}
		return table, nil
	case *pgquery.Node_RenameStmt:
		switch in.RenameStmt.RenameType {
		case pgquery.ObjectType_OBJECT_COLUMN:
			return &ast.RenameColumnStmt{
				Table:      convertRangeVarToTableName(in.RenameStmt.Relation),
				ColumnName: in.RenameStmt.Subname,
				NewName:    in.RenameStmt.Newname,
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

func convertConstraint(in *pgquery.Node_Constraint) (*ast.ConstraintDef, error) {
	cons := &ast.ConstraintDef{
		Name: in.Constraint.Conname,
		Type: convertConstraintType(in.Constraint.Contype),
	}

	for _, key := range in.Constraint.Keys {
		name, ok := key.Node.(*pgquery.Node_String_)
		if !ok {
			return nil, parser.NewConvertErrorf("expected String but found %t", key.Node)
		}
		cons.KeyList = append(cons.KeyList, name.String_.Str)
	}

	if cons.Type == ast.ConstraintTypeForeign {
		cons.Foreign = &ast.ForeignDef{
			Table: convertRangeVarToTableName(in.Constraint.Pktable),
		}

		for _, item := range in.Constraint.PkAttrs {
			name, ok := item.Node.(*pgquery.Node_String_)
			if !ok {
				return nil, parser.NewConvertErrorf("expected String but found %t", item.Node)
			}
			cons.Foreign.ColumnList = append(cons.Foreign.ColumnList, name.String_.Str)
		}

		for _, item := range in.Constraint.FkAttrs {
			name, ok := item.Node.(*pgquery.Node_String_)
			if !ok {
				return nil, parser.NewConvertErrorf("expected String but found %t", item.Node)
			}
			cons.KeyList = append(cons.KeyList, name.String_.Str)
		}
	}

	return cons, nil
}

func convertConstraintType(in pgquery.ConstrType) ast.ConstraintType {
	switch in {
	case pgquery.ConstrType_CONSTR_PRIMARY:
		return ast.ConstraintTypePrimary
	case pgquery.ConstrType_CONSTR_UNIQUE:
		return ast.ConstraintTypeUnique
	case pgquery.ConstrType_CONSTR_FOREIGN:
		return ast.ConstraintTypeForeign
	}
	return ast.ConstraintTypeUndefined
}
