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

					column, err := convertColumnDef(def)
					if err != nil {
						return nil, err
					}

					addColumn := &ast.AddColumnListStmt{
						Table:      alterTable.Table,
						ColumnList: []*ast.ColumnDef{column},
					}

					alterTable.AlterItemList = append(alterTable.AlterItemList, addColumn)
				case pgquery.AlterTableType_AT_DropColumn:
					dropColumn := &ast.DropColumnStmt{
						Table:      alterTable.Table,
						ColumnName: alterCmd.Name,
					}

					alterTable.AlterItemList = append(alterTable.AlterItemList, dropColumn)
				case pgquery.AlterTableType_AT_AddConstraint:
					def, ok := alterCmd.Def.Node.(*pgquery.Node_Constraint)
					if !ok {
						return nil, parser.NewConvertErrorf("expected Constraint but found %t", alterCmd.Def.Node)
					}
					constraint, err := convertConstraint(def)
					if err != nil {
						return nil, err
					}

					addConstraint := &ast.AddConstraintStmt{
						Table:      alterTable.Table,
						Constraint: constraint,
					}

					alterTable.AlterItemList = append(alterTable.AlterItemList, addConstraint)
				case pgquery.AlterTableType_AT_DropConstraint:
					dropConstraint := &ast.DropConstraintStmt{
						Table:          alterTable.Table,
						ConstraintName: alterCmd.Name,
					}

					alterTable.AlterItemList = append(alterTable.AlterItemList, dropConstraint)
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
				column, err := convertColumnDef(item)
				if err != nil {
					return nil, err
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
		case pgquery.ObjectType_OBJECT_TABCONSTRAINT:
			return &ast.RenameConstraintStmt{
				Table:          convertRangeVarToTableName(in.RenameStmt.Relation),
				ConstraintName: in.RenameStmt.Subname,
				NewName:        in.RenameStmt.Newname,
			}, nil
		case pgquery.ObjectType_OBJECT_INDEX:
			return &ast.RenameIndexStmt{
				Table:     convertRangeVarToIndexTableName(in.RenameStmt.Relation),
				IndexName: in.RenameStmt.Relation.Relname,
				NewName:   in.RenameStmt.Newname,
			}, nil
		}
	case *pgquery.Node_IndexStmt:
		indexDef := &ast.IndexDef{
			Table:  convertRangeVarToTableName(in.IndexStmt.Relation),
			Name:   in.IndexStmt.Idxname,
			Unique: in.IndexStmt.Unique,
		}

		for _, key := range in.IndexStmt.IndexParams {
			index, ok := key.Node.(*pgquery.Node_IndexElem)
			if !ok {
				return nil, parser.NewConvertErrorf("expected IndexElem but found %t", key.Node)
			}
			// We only support index on columns now.
			// TODO(rebelice): support index on expressions.
			if index.IndexElem.Name != "" {
				indexDef.KeyList = append(indexDef.KeyList, &ast.IndexKeyDef{
					Type: ast.IndexKeyTypeColumn,
					Key:  index.IndexElem.Name,
				})
			} else {
				indexDef.KeyList = append(indexDef.KeyList, &ast.IndexKeyDef{
					Type: ast.IndexKeyTypeExpression,
				})
			}
		}

		return &ast.CreateIndexStmt{Index: indexDef}, nil
	case *pgquery.Node_DropStmt:
		switch in.DropStmt.RemoveType {
		case pgquery.ObjectType_OBJECT_INDEX:
			dropIndex := &ast.DropIndexStmt{}
			for _, object := range in.DropStmt.Objects {
				list, ok := object.Node.(*pgquery.Node_List)
				if !ok {
					return nil, parser.NewConvertErrorf("expected List but found %t", object.Node)
				}
				indexDef, err := convertListToIndexDef(list)
				if err != nil {
					return nil, err
				}
				dropIndex.IndexList = append(dropIndex.IndexList, indexDef)
			}
			return dropIndex, nil
		case pgquery.ObjectType_OBJECT_TABLE:
			dropTable := &ast.DropTableStmt{}
			for _, object := range in.DropStmt.Objects {
				list, ok := object.Node.(*pgquery.Node_List)
				if !ok {
					return nil, parser.NewConvertErrorf("expected List but found %t", object.Node)
				}
				tableDef, err := convertListToTableDef(list)
				if err != nil {
					return nil, err
				}
				dropTable.TableList = append(dropTable.TableList, tableDef)
			}
			return dropTable, nil
		}
	}

	return nil, nil
}

func convertListToTableDef(in *pgquery.Node_List) (*ast.TableDef, error) {
	stringList, err := convertListToStringList(in)
	if err != nil {
		return nil, err
	}
	switch len(in.List.Items) {
	case 2:
		return &ast.TableDef{
			Schema: stringList[0],
			Name:   stringList[1],
		}, nil
	case 1:
		return &ast.TableDef{Name: stringList[0]}, nil
	default:
		return nil, parser.NewConvertErrorf("expected length is 1 or 2, but found %d", len(in.List.Items))
	}
}

func convertListToIndexDef(in *pgquery.Node_List) (*ast.IndexDef, error) {
	stringList, err := convertListToStringList(in)
	if err != nil {
		return nil, err
	}
	indexDef := &ast.IndexDef{}
	switch len(in.List.Items) {
	case 2:
		indexDef.Table = &ast.TableDef{Schema: stringList[0]}
		indexDef.Name = stringList[1]
	case 1:
		indexDef.Name = stringList[0]
	default:
		return nil, parser.NewConvertErrorf("expected length is 1 or 2, but found %d", len(in.List.Items))
	}
	return indexDef, nil
}

func convertListToStringList(in *pgquery.Node_List) ([]string, error) {
	var res []string
	for _, item := range in.List.Items {
		s, ok := item.Node.(*pgquery.Node_String_)
		if !ok {
			return nil, parser.NewConvertErrorf("expected String but found %t", item.Node)
		}
		res = append(res, s.String_.Str)
	}
	return res, nil
}

func convertRangeVarToTableName(in *pgquery.RangeVar) *ast.TableDef {
	return &ast.TableDef{
		Database: in.Catalogname,
		Schema:   in.Schemaname,
		Name:     in.Relname,
	}
}

func convertRangeVarToIndexTableName(in *pgquery.RangeVar) *ast.TableDef {
	if in.Schemaname == "" {
		return nil
	}
	return &ast.TableDef{
		Schema: in.Schemaname,
	}
}

func convertConstraint(in *pgquery.Node_Constraint) (*ast.ConstraintDef, error) {
	cons := &ast.ConstraintDef{
		Name: in.Constraint.Conname,
		Type: convertConstraintType(in.Constraint.Contype, in.Constraint.Indexname != ""),
	}

	switch cons.Type {
	case ast.ConstraintTypePrimary, ast.ConstraintTypeUnique:
		for _, key := range in.Constraint.Keys {
			name, ok := key.Node.(*pgquery.Node_String_)
			if !ok {
				return nil, parser.NewConvertErrorf("expected String but found %t", key.Node)
			}
			cons.KeyList = append(cons.KeyList, name.String_.Str)
		}
	case ast.ConstraintTypeForeign:
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
	case ast.ConstraintTypePrimaryUsingIndex, ast.ConstraintTypeUniqueUsingIndex:
		cons.IndexName = in.Constraint.Indexname
	}

	return cons, nil
}

func convertConstraintType(in pgquery.ConstrType, usingIndex bool) ast.ConstraintType {
	switch in {
	case pgquery.ConstrType_CONSTR_PRIMARY:
		if usingIndex {
			return ast.ConstraintTypePrimaryUsingIndex
		}
		return ast.ConstraintTypePrimary
	case pgquery.ConstrType_CONSTR_UNIQUE:
		if usingIndex {
			return ast.ConstraintTypeUniqueUsingIndex
		}
		return ast.ConstraintTypeUnique
	case pgquery.ConstrType_CONSTR_FOREIGN:
		if usingIndex {
			return ast.ConstraintTypeUndefined
		}
		return ast.ConstraintTypeForeign
	}
	return ast.ConstraintTypeUndefined
}

func convertColumnDef(in *pgquery.Node_ColumnDef) (*ast.ColumnDef, error) {
	column := &ast.ColumnDef{
		ColumnName: in.ColumnDef.Colname,
	}

	for _, cons := range in.ColumnDef.Constraints {
		constraint, ok := cons.Node.(*pgquery.Node_Constraint)
		if !ok {
			return nil, parser.NewConvertErrorf("expected Constraint but found %t", cons.Node)
		}
		columnCons, err := convertConstraint(constraint)
		if err != nil {
			return nil, err
		}
		columnCons.KeyList = append(columnCons.KeyList, in.ColumnDef.Colname)
		column.ConstraintList = append(column.ConstraintList, columnCons)
	}

	return column, nil
}
