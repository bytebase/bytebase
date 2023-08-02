package pg

import (
	"fmt"
	"strconv"
	"strings"

	pgquery "github.com/pganalyze/pg_query_go/v4"
	"github.com/pkg/errors"

	parser "github.com/bytebase/bytebase/backend/plugin/parser/sql"

	"github.com/bytebase/bytebase/backend/plugin/parser/sql/ast"
)

// convert converts the pg_query.Node to ast.Node.
func convert(node *pgquery.Node, statement parser.SingleSQL) (res ast.Node, err error) {
	defer func() {
		if err == nil && res != nil {
			res.SetText(strings.TrimSpace(statement.Text))
			res.SetLastLine(statement.LastLine)
			switch n := res.(type) {
			case *ast.CreateTableStmt:
				err = parser.SetLineForCreateTableStmt(parser.Postgres, n)
			case *ast.AlterTableStmt:
				for _, item := range n.AlterItemList {
					item.SetLastLine(n.LastLine())
				}
			}
		}
	}()
	switch in := node.Node.(type) {
	case *pgquery.Node_AlterTableStmt:
		alterTable := &ast.AlterTableStmt{
			Table:         convertRangeVarToTableName(in.AlterTableStmt.Relation, ast.TableTypeBaseTable),
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
						Table:       alterTable.Table,
						ColumnList:  []*ast.ColumnDef{column},
						IfNotExists: alterCmd.MissingOk,
					}

					alterTable.AlterItemList = append(alterTable.AlterItemList, addColumn)
				case pgquery.AlterTableType_AT_DropColumn:
					dropColumn := &ast.DropColumnStmt{
						Table:      alterTable.Table,
						ColumnName: alterCmd.Name,
						IfExists:   alterCmd.MissingOk,
						Behavior:   convertDropBehavior(alterCmd.Behavior),
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
						IfExists:       alterCmd.MissingOk,
					}

					alterTable.AlterItemList = append(alterTable.AlterItemList, dropConstraint)
				case pgquery.AlterTableType_AT_SetNotNull:
					setNotNull := &ast.SetNotNullStmt{
						Table:      alterTable.Table,
						ColumnName: alterCmd.Name,
					}

					alterTable.AlterItemList = append(alterTable.AlterItemList, setNotNull)
				case pgquery.AlterTableType_AT_DropNotNull:
					dropNotNull := &ast.DropNotNullStmt{
						Table:      alterTable.Table,
						ColumnName: alterCmd.Name,
					}

					alterTable.AlterItemList = append(alterTable.AlterItemList, dropNotNull)
				case pgquery.AlterTableType_AT_AlterColumnType:
					column, ok := alterCmd.Def.Node.(*pgquery.Node_ColumnDef)
					if !ok {
						return nil, parser.NewConvertErrorf("expected ColumnDef but found %t", alterCmd.Def.Node)
					}
					dataType, err := convertDataType(column.ColumnDef.TypeName)
					if err != nil {
						return nil, err
					}
					collation := (*ast.CollationNameDef)(nil)
					if column.ColumnDef.CollClause != nil {
						collation, err = convertCollationName(column.ColumnDef.CollClause)
						if err != nil {
							return nil, err
						}
					}
					alterColumType := &ast.AlterColumnTypeStmt{
						Table:      alterTable.Table,
						ColumnName: alterCmd.Name,
						Type:       dataType,
						Collation:  collation,
					}

					alterTable.AlterItemList = append(alterTable.AlterItemList, alterColumType)
				case pgquery.AlterTableType_AT_ColumnDefault:
					if alterCmd.Def == nil {
						dropDefault := &ast.DropDefaultStmt{
							Table:      alterTable.Table,
							ColumnName: alterCmd.Name,
						}

						alterTable.AlterItemList = append(alterTable.AlterItemList, dropDefault)
					} else {
						var err error
						setDefault := &ast.SetDefaultStmt{
							Table:      alterTable.Table,
							ColumnName: alterCmd.Name,
						}
						if setDefault.Expression, _, _, err = convertExpressionNode(alterCmd.Def); err != nil {
							return nil, err
						}
						text, err := pgquery.DeparseNode(pgquery.DeparseTypeExpr, alterCmd.Def)
						if err != nil {
							return nil, err
						}
						setDefault.Expression.SetText(text)

						alterTable.AlterItemList = append(alterTable.AlterItemList, setDefault)
					}
				case pgquery.AlterTableType_AT_AttachPartition:
					alterTable.AlterItemList = append(alterTable.AlterItemList, &ast.AttachPartitionStmt{
						Table: alterTable.Table,
					})
				}
			}
		}
		return alterTable, nil
	case *pgquery.Node_CreateStmt:
		return convertCreateStmt(in.CreateStmt)
	case *pgquery.Node_RenameStmt:
		switch in.RenameStmt.RenameType {
		case pgquery.ObjectType_OBJECT_COLUMN:
			tableType, err := convertToTableType(in.RenameStmt.RelationType)
			if err != nil {
				return nil, err
			}
			table := convertRangeVarToTableName(in.RenameStmt.Relation, tableType)
			return &ast.AlterTableStmt{
				Table: table,
				AlterItemList: []ast.Node{
					&ast.RenameColumnStmt{
						Table:      table,
						ColumnName: in.RenameStmt.Subname,
						NewName:    in.RenameStmt.Newname,
					},
				},
			}, nil
		case pgquery.ObjectType_OBJECT_TABLE:
			table := convertRangeVarToTableName(in.RenameStmt.Relation, ast.TableTypeBaseTable)
			return &ast.AlterTableStmt{
				Table: table,
				AlterItemList: []ast.Node{
					&ast.RenameTableStmt{
						Table:   table,
						NewName: in.RenameStmt.Newname,
					},
				},
			}, nil
		case pgquery.ObjectType_OBJECT_TABCONSTRAINT:
			table := convertRangeVarToTableName(in.RenameStmt.Relation, ast.TableTypeBaseTable)
			return &ast.AlterTableStmt{
				Table: table,
				AlterItemList: []ast.Node{
					&ast.RenameConstraintStmt{
						Table:          table,
						ConstraintName: in.RenameStmt.Subname,
						NewName:        in.RenameStmt.Newname,
					},
				},
			}, nil
		case pgquery.ObjectType_OBJECT_VIEW:
			view := convertRangeVarToTableName(in.RenameStmt.Relation, ast.TableTypeView)
			return &ast.AlterTableStmt{
				Table: view,
				AlterItemList: []ast.Node{
					&ast.RenameTableStmt{
						Table:   view,
						NewName: in.RenameStmt.Newname,
					},
				},
			}, nil
		case pgquery.ObjectType_OBJECT_INDEX:
			return &ast.RenameIndexStmt{
				Table:     convertRangeVarToIndexTableName(in.RenameStmt.Relation, ast.TableTypeUnknown),
				IndexName: in.RenameStmt.Relation.Relname,
				NewName:   in.RenameStmt.Newname,
			}, nil
		case pgquery.ObjectType_OBJECT_SCHEMA:
			return &ast.RenameSchemaStmt{
				Schema:  in.RenameStmt.Subname,
				NewName: in.RenameStmt.Newname,
			}, nil
		}
	case *pgquery.Node_IndexStmt:
		indexDef := &ast.IndexDef{
			Table:  convertRangeVarToTableName(in.IndexStmt.Relation, ast.TableTypeUnknown),
			Name:   in.IndexStmt.Idxname,
			Unique: in.IndexStmt.Unique,
		}

		method, err := convertMethodType(in.IndexStmt.AccessMethod)
		if err != nil {
			return nil, err
		}
		indexDef.Method = method

		for _, key := range in.IndexStmt.IndexParams {
			index, ok := key.Node.(*pgquery.Node_IndexElem)
			if !ok {
				return nil, parser.NewConvertErrorf("expected IndexElem but found %t", key.Node)
			}
			indexKey := &ast.IndexKeyDef{}
			if index.IndexElem.Name != "" {
				indexKey.Type = ast.IndexKeyTypeColumn
				indexKey.Key = index.IndexElem.Name
			} else {
				expression, err := pgquery.DeparseNode(pgquery.DeparseTypeExpr, index.IndexElem.Expr)
				if err != nil {
					return nil, err
				}
				indexKey.Type = ast.IndexKeyTypeExpression
				indexKey.Key = expression
			}

			order, err := convertSortOrder(index.IndexElem.Ordering)
			if err != nil {
				return nil, err
			}
			indexKey.SortOrder = order

			nullOrder, err := convertNullOrder(index.IndexElem.NullsOrdering)
			if err != nil {
				return nil, err
			}
			indexKey.NullOrder = nullOrder

			indexDef.KeyList = append(indexDef.KeyList, indexKey)
		}

		return &ast.CreateIndexStmt{Index: indexDef, IfNotExists: in.IndexStmt.IfNotExists, Concurrently: in.IndexStmt.Concurrent}, nil
	case *pgquery.Node_DropStmt:
		switch in.DropStmt.RemoveType {
		case pgquery.ObjectType_OBJECT_INDEX:
			dropIndex := &ast.DropIndexStmt{
				IfExists: in.DropStmt.MissingOk,
				Behavior: convertDropBehavior(in.DropStmt.Behavior),
			}
			for _, object := range in.DropStmt.Objects {
				list, ok := object.Node.(*pgquery.Node_List)
				if !ok {
					return nil, parser.NewConvertErrorf("expected List but found %t", object.Node)
				}
				indexDef, err := convertListToIndexDef(list, ast.TableTypeUnknown)
				if err != nil {
					return nil, err
				}
				dropIndex.IndexList = append(dropIndex.IndexList, indexDef)
			}
			return dropIndex, nil
		case pgquery.ObjectType_OBJECT_TABLE:
			dropTable := &ast.DropTableStmt{
				IfExists: in.DropStmt.MissingOk,
				Behavior: convertDropBehavior(in.DropStmt.Behavior),
			}
			for _, object := range in.DropStmt.Objects {
				list, ok := object.Node.(*pgquery.Node_List)
				if !ok {
					return nil, parser.NewConvertErrorf("expected List but found %t", object.Node)
				}
				tableDef, err := convertListToTableDef(list, ast.TableTypeBaseTable)
				if err != nil {
					return nil, err
				}
				dropTable.TableList = append(dropTable.TableList, tableDef)
			}
			return dropTable, nil
		case pgquery.ObjectType_OBJECT_VIEW:
			dropView := &ast.DropTableStmt{
				IfExists: in.DropStmt.MissingOk,
				Behavior: convertDropBehavior(in.DropStmt.Behavior),
			}
			for _, object := range in.DropStmt.Objects {
				list, ok := object.Node.(*pgquery.Node_List)
				if !ok {
					return nil, parser.NewConvertErrorf("expected List but found %t", object.Node)
				}
				viewDef, err := convertListToTableDef(list, ast.TableTypeView)
				if err != nil {
					return nil, err
				}
				dropView.TableList = append(dropView.TableList, viewDef)
			}
			return dropView, nil
		case pgquery.ObjectType_OBJECT_SCHEMA:
			dropSchema := &ast.DropSchemaStmt{
				IfExists: in.DropStmt.MissingOk,
				Behavior: convertDropBehavior(in.DropStmt.Behavior),
			}
			for _, object := range in.DropStmt.Objects {
				strNode, ok := object.Node.(*pgquery.Node_String_)
				if !ok {
					return nil, parser.NewConvertErrorf("expected String but found %t", object.Node)
				}
				dropSchema.SchemaList = append(dropSchema.SchemaList, strNode.String_.Str)
			}
			return dropSchema, nil
		case pgquery.ObjectType_OBJECT_SEQUENCE:
			dropSequence := &ast.DropSequenceStmt{
				IfExists: in.DropStmt.MissingOk,
				Behavior: convertDropBehavior(in.DropStmt.Behavior),
			}
			for _, sequence := range in.DropStmt.Objects {
				list, ok := sequence.Node.(*pgquery.Node_List)
				if !ok {
					return nil, parser.NewConvertErrorf("expected List but found %t", sequence.Node)
				}
				sequenceDef, err := convertListToSequenceNameDef(list)
				if err != nil {
					return nil, err
				}
				dropSequence.SequenceNameList = append(dropSequence.SequenceNameList, sequenceDef)
			}
			return dropSequence, nil
		case pgquery.ObjectType_OBJECT_EXTENSION:
			dropExtension := &ast.DropExtensionStmt{
				IfExists: in.DropStmt.MissingOk,
				Behavior: convertDropBehavior(in.DropStmt.Behavior),
			}
			for _, extension := range in.DropStmt.Objects {
				extensionName, ok := extension.Node.(*pgquery.Node_String_)
				if !ok {
					return nil, parser.NewConvertErrorf("expected String but found %t", extension.Node)
				}
				dropExtension.NameList = append(dropExtension.NameList, extensionName.String_.Str)
			}
			return dropExtension, nil
		case pgquery.ObjectType_OBJECT_FUNCTION:
			dropFunctionStmt := &ast.DropFunctionStmt{
				IfExists: in.DropStmt.MissingOk,
				Behavior: convertDropBehavior(in.DropStmt.Behavior),
			}
			for _, function := range in.DropStmt.Objects {
				functionNode, ok := function.Node.(*pgquery.Node_ObjectWithArgs)
				if !ok {
					return nil, parser.NewConvertErrorf("expected ObjectWithArgs but found %t", function.Node)
				}
				functionDef := &ast.FunctionDef{}
				var err error
				functionDef.Schema, functionDef.Name, err = convertObjectName(functionNode.ObjectWithArgs.Objname)
				if err != nil {
					return nil, err
				}
				functionDef.ParameterList, err = convertFunctionParameterList(functionNode.ObjectWithArgs.Objargs)
				if err != nil {
					return nil, err
				}

				dropFunctionStmt.FunctionList = append(dropFunctionStmt.FunctionList, functionDef)
			}

			return dropFunctionStmt, nil
		case pgquery.ObjectType_OBJECT_TRIGGER:
			dropTriggerStmt := &ast.DropTriggerStmt{
				IfExists: in.DropStmt.MissingOk,
				Behavior: convertDropBehavior(in.DropStmt.Behavior),
			}

			if len(in.DropStmt.Objects) != 1 {
				return nil, parser.NewConvertErrorf("expected one trigger but found %d", len(in.DropStmt.Objects))
			}
			listNode, ok := in.DropStmt.Objects[0].Node.(*pgquery.Node_List)
			if !ok {
				return nil, parser.NewConvertErrorf("expected List but found %d", in.DropStmt.Objects[0].Node)
			}

			list, err := convertListToStringList(listNode)
			if err != nil {
				return nil, err
			}
			switch len(list) {
			case 3:
				dropTriggerStmt.Trigger = &ast.TriggerDef{
					Name: list[2],
					Table: &ast.TableDef{
						Type:   ast.TableTypeUnknown,
						Schema: list[0],
						Name:   list[1],
					},
				}
			case 2:
				dropTriggerStmt.Trigger = &ast.TriggerDef{
					Name: list[1],
					Table: &ast.TableDef{
						Type:   ast.TableTypeUnknown,
						Schema: "",
						Name:   list[0],
					},
				}
			default:
				return nil, parser.NewConvertErrorf("expected one or two but found %d", len(list))
			}

			return dropTriggerStmt, nil
		case pgquery.ObjectType_OBJECT_TYPE:
			dropTypeStmt := &ast.DropTypeStmt{
				IfExists: in.DropStmt.MissingOk,
				Behavior: convertDropBehavior(in.DropStmt.Behavior),
			}

			for _, object := range in.DropStmt.Objects {
				typeName, ok := object.Node.(*pgquery.Node_TypeName)
				if !ok {
					return nil, parser.NewConvertErrorf("expected TypeName but found %t", object.Node)
				}
				schema, name, err := convertObjectName(typeName.TypeName.Names)
				if err != nil {
					return nil, err
				}
				dropTypeStmt.TypeNameList = append(dropTypeStmt.TypeNameList, &ast.TypeNameDef{
					Schema: schema,
					Name:   name,
				})
			}

			return dropTypeStmt, nil
		}
	case *pgquery.Node_DropdbStmt:
		return &ast.DropDatabaseStmt{
			DatabaseName: in.DropdbStmt.Dbname,
			IfExists:     in.DropdbStmt.MissingOk,
		}, nil
	case *pgquery.Node_SelectStmt:
		return convertSelectStmt(in.SelectStmt)
	case *pgquery.Node_UpdateStmt:
		update := &ast.UpdateStmt{
			Table: convertRangeVarToTableName(in.UpdateStmt.Relation, ast.TableTypeBaseTable),
		}
		// Convert FROM clause
		// Here we only find the SELECT stmt in FROM clause
		for _, item := range in.UpdateStmt.FromClause {
			if node, ok := item.Node.(*pgquery.Node_RangeSubselect); ok {
				subselect, err := convertRangeSubselect(node.RangeSubselect)
				if err != nil {
					return nil, err
				}
				update.SubqueryList = append(update.SubqueryList, subselect)
			}
		}
		// Convert WHERE clause
		if in.UpdateStmt.WhereClause != nil {
			var err error
			var subqueryList []*ast.SubqueryDef
			update.WhereClause, update.PatternLikeList, subqueryList, err = convertExpressionNode(in.UpdateStmt.WhereClause)
			if err != nil {
				return nil, err
			}
			update.SubqueryList = append(update.SubqueryList, subqueryList...)
		}
		return update, nil
	case *pgquery.Node_DeleteStmt:
		deleteStmt := &ast.DeleteStmt{
			Table: convertRangeVarToTableName(in.DeleteStmt.Relation, ast.TableTypeBaseTable),
		}
		if in.DeleteStmt.WhereClause != nil {
			var err error
			if deleteStmt.WhereClause, deleteStmt.PatternLikeList, deleteStmt.SubqueryList, err = convertExpressionNode(in.DeleteStmt.WhereClause); err != nil {
				return nil, err
			}
		}
		return deleteStmt, nil
	case *pgquery.Node_CreateSeqStmt:
		createSeqStmt := &ast.CreateSequenceStmt{
			IfNotExists: in.CreateSeqStmt.IfNotExists,
		}
		if in.CreateSeqStmt.Sequence == nil {
			// Unexpected case.
			return nil, parser.NewConvertErrorf("CreateSeqStmt.Sequence is nil")
		}
		createSeqStmt.SequenceDef.SequenceName = convertRangeVarToSeqName(in.CreateSeqStmt.Sequence)
		for _, option := range in.CreateSeqStmt.Options {
			defElemNode, ok := option.Node.(*pgquery.Node_DefElem)
			if !ok {
				return nil, parser.NewConvertErrorf("expected DefElem but found %t", option.Node)
			}
			switch defElemNode.DefElem.Defname {
			case "as":
				if createSeqStmt.SequenceDef.SequenceDataType, err = convertDefElemToSeqType(defElemNode.DefElem); err != nil {
					return nil, err
				}
			case "increment":
				if createSeqStmt.SequenceDef.IncrementBy, err = convertDefElemNodeIntegerToInt32(defElemNode.DefElem); err != nil {
					return nil, err
				}
			case "start":
				if createSeqStmt.SequenceDef.StartWith, err = convertDefElemNodeIntegerToInt32(defElemNode.DefElem); err != nil {
					return nil, err
				}
			case "minvalue":
				if createSeqStmt.SequenceDef.MinValue, err = convertDefElemNodeIntegerToInt32(defElemNode.DefElem); err != nil {
					return nil, err
				}
			case "maxvalue":
				if createSeqStmt.SequenceDef.MaxValue, err = convertDefElemNodeIntegerToInt32(defElemNode.DefElem); err != nil {
					return nil, err
				}
			case "cache":
				if createSeqStmt.SequenceDef.Cache, err = convertDefElemNodeIntegerToInt32(defElemNode.DefElem); err != nil {
					return nil, err
				}
			case "cycle":
				if createSeqStmt.SequenceDef.Cycle, err = convertDefElemNodeIntegerToBool(defElemNode.DefElem); err != nil {
					return nil, err
				}
			case "owned_by":
				if createSeqStmt.SequenceDef.OwnedBy, err = convertDefElemNodeListToColumnNameDef(defElemNode.DefElem); err != nil {
					return nil, err
				}
			default:
				return nil, parser.NewConvertErrorf("unsupported option %s", defElemNode.DefElem.Defname)
			}
		}
		return createSeqStmt, nil
	case *pgquery.Node_AlterSeqStmt:
		return convertAlterSequence(in.AlterSeqStmt)
	case *pgquery.Node_AlterObjectSchemaStmt:
		switch in.AlterObjectSchemaStmt.ObjectType {
		case pgquery.ObjectType_OBJECT_TABLE:
			table := convertRangeVarToTableName(in.AlterObjectSchemaStmt.Relation, ast.TableTypeBaseTable)
			return &ast.AlterTableStmt{
				Table: table,
				AlterItemList: []ast.Node{
					&ast.SetSchemaStmt{
						Table:     table,
						NewSchema: in.AlterObjectSchemaStmt.Newschema,
					},
				},
			}, nil
		case pgquery.ObjectType_OBJECT_VIEW:
			view := convertRangeVarToTableName(in.AlterObjectSchemaStmt.Relation, ast.TableTypeView)
			return &ast.AlterTableStmt{
				Table: view,
				AlterItemList: []ast.Node{
					&ast.SetSchemaStmt{
						Table:     view,
						NewSchema: in.AlterObjectSchemaStmt.Newschema,
					},
				},
			}, nil
		}
	case *pgquery.Node_ExplainStmt:
		explainStmt := &ast.ExplainStmt{}
		if query, ok := in.ExplainStmt.Query.Node.(*pgquery.Node_SelectStmt); ok {
			if explainStmt.Statement, err = convertSelectStmt(query.SelectStmt); err != nil {
				return nil, err
			}
		}
		return explainStmt, nil
	case *pgquery.Node_InsertStmt:
		insertStmt := &ast.InsertStmt{
			Table: convertRangeVarToTableName(in.InsertStmt.Relation, ast.TableTypeBaseTable),
		}

		for _, columnNode := range in.InsertStmt.Cols {
			if column, ok := columnNode.Node.(*pgquery.Node_ResTarget); ok {
				insertStmt.ColumnList = append(insertStmt.ColumnList, column.ResTarget.Name)
			} else {
				return nil, parser.NewConvertErrorf("expected ResTarget but found %t", columnNode.Node)
			}
		}

		if in.InsertStmt.SelectStmt != nil {
			if selectNode, ok := in.InsertStmt.SelectStmt.Node.(*pgquery.Node_SelectStmt); ok {
				// PG parser will parse the value list as a SELECT statement.
				if selectNode.SelectStmt.ValuesLists == nil {
					if insertStmt.Select, err = convertSelectStmt(selectNode.SelectStmt); err != nil {
						return nil, err
					}
				} else {
					for _, list := range selectNode.SelectStmt.ValuesLists {
						var valueList []ast.ExpressionNode
						listNode, ok := list.Node.(*pgquery.Node_List)
						if !ok {
							return nil, parser.NewConvertErrorf("expected Node_List but found %t", list.Node)
						}
						for _, item := range listNode.List.Items {
							value, _, _, err := convertExpressionNode(item)
							if err != nil {
								return nil, err
							}
							valueList = append(valueList, value)
						}
						insertStmt.ValueList = append(insertStmt.ValueList, valueList)
					}
				}
			} else {
				return nil, parser.NewConvertErrorf("expected SelectStmt but found %t", in.InsertStmt.SelectStmt.Node)
			}
		}
		return insertStmt, nil
	case *pgquery.Node_CopyStmt:
		copyStmt := ast.CopyStmt{
			Table:    convertRangeVarToTableName(in.CopyStmt.Relation, ast.TableTypeBaseTable),
			FilePath: in.CopyStmt.Filename,
		}

		return &copyStmt, nil
	case *pgquery.Node_CommentStmt:
		commentStmt := ast.CommentStmt{
			Comment: in.CommentStmt.Comment,
		}

		return &commentStmt, nil
	case *pgquery.Node_CreatedbStmt:
		createDatabaseStmt := ast.CreateDatabaseStmt{
			Name: in.CreatedbStmt.Dbname,
		}
		for _, option := range in.CreatedbStmt.Options {
			if item, ok := option.Node.(*pgquery.Node_DefElem); ok && item.DefElem.Defname == "encoding" {
				value, ok := item.DefElem.Arg.Node.(*pgquery.Node_String_)
				if !ok {
					return nil, parser.NewConvertErrorf("expected String but found %t", item.DefElem.Arg.Node)
				}
				createDatabaseStmt.OptionList = append(createDatabaseStmt.OptionList, &ast.DatabaseOptionDef{
					Type:  ast.DatabaseOptionEncoding,
					Value: value.String_.Str,
				})
			}
		}

		return &createDatabaseStmt, nil
	case *pgquery.Node_CreateSchemaStmt:
		createSchemaStmt := ast.CreateSchemaStmt{
			Name:        in.CreateSchemaStmt.Schemaname,
			IfNotExists: in.CreateSchemaStmt.IfNotExists,
		}
		roleSpec, err := convertRoleSpec(in.CreateSchemaStmt.Authrole)
		if err != nil {
			return nil, err
		}
		createSchemaStmt.RoleSpec = roleSpec
		for _, elt := range in.CreateSchemaStmt.SchemaElts {
			switch stmt := elt.Node.(type) {
			// Currently, only CREATE TABLE, CREATE VIEW, CREATE INDEX, CREATE SEQUENCE,
			// CREATE TRIGGER and GRANT are accepted as clauses within CREATE SCHEMA.
			// TODO(zp): support other statement list above.
			case *pgquery.Node_CreateStmt:
				createStmt, err := convertCreateStmt(stmt.CreateStmt)
				if err != nil {
					return nil, err
				}
				createSchemaStmt.SchemaElementList = append(createSchemaStmt.SchemaElementList, createStmt)
			default:
			}
		}
		return &createSchemaStmt, nil
	case *pgquery.Node_CreateExtensionStmt:
		createExtensionStmt := &ast.CreateExtensionStmt{
			Name:        in.CreateExtensionStmt.Extname,
			IfNotExists: in.CreateExtensionStmt.IfNotExists,
		}

		for _, option := range in.CreateExtensionStmt.Options {
			if item, ok := option.Node.(*pgquery.Node_DefElem); ok {
				if item.DefElem.Defname == "schema" {
					schemaName, ok := item.DefElem.Arg.Node.(*pgquery.Node_String_)
					if !ok {
						return nil, parser.NewConvertErrorf("expected String but found %t", item.DefElem.Arg.Node)
					}
					createExtensionStmt.Schema = schemaName.String_.Str
				}
			}
		}

		return createExtensionStmt, nil
	case *pgquery.Node_CreateFunctionStmt:
		var err error
		functionDef := &ast.FunctionDef{}

		functionDef.Schema, functionDef.Name, err = convertObjectName(in.CreateFunctionStmt.Funcname)
		if err != nil {
			return nil, err
		}

		functionDef.ParameterList, err = convertFunctionParameterList(in.CreateFunctionStmt.Parameters)
		if err != nil {
			return nil, err
		}

		return &ast.CreateFunctionStmt{Function: functionDef}, nil
	case *pgquery.Node_CreateTrigStmt:
		createTriggerStmt := &ast.CreateTriggerStmt{
			Trigger: &ast.TriggerDef{
				Name:  in.CreateTrigStmt.Trigname,
				Table: convertRangeVarToTableName(in.CreateTrigStmt.Relation, ast.TableTypeUnknown),
			},
		}

		return createTriggerStmt, nil
	case *pgquery.Node_CreateEnumStmt:
		var err error
		enumTypeDef := &ast.EnumTypeDef{Name: &ast.TypeNameDef{}}

		enumTypeDef.Name.Schema, enumTypeDef.Name.Name, err = convertObjectName(in.CreateEnumStmt.TypeName)
		if err != nil {
			return nil, err
		}

		enumTypeDef.LabelList, err = convertEnumLabelList(in.CreateEnumStmt.Vals)
		if err != nil {
			return nil, err
		}

		return &ast.CreateTypeStmt{Type: enumTypeDef}, nil
	case *pgquery.Node_AlterEnumStmt:
		if in.AlterEnumStmt.OldVal == "" {
			schema, name, err := convertObjectName(in.AlterEnumStmt.TypeName)
			if err != nil {
				return nil, err
			}
			typeName := &ast.TypeNameDef{
				Schema: schema,
				Name:   name,
			}
			// ADD ENUM VALUE STATEMENT
			addEnumValueStmt := &ast.AddEnumLabelStmt{
				EnumType:      typeName,
				NewLabel:      in.AlterEnumStmt.NewVal,
				NeighborLabel: in.AlterEnumStmt.NewValNeighbor,
			}
			if in.AlterEnumStmt.NewValNeighbor == "" {
				addEnumValueStmt.Position = ast.PositionTypeEnd
			} else if in.AlterEnumStmt.NewValIsAfter {
				addEnumValueStmt.Position = ast.PositionTypeAfter
			} else {
				addEnumValueStmt.Position = ast.PositionTypeBefore
			}

			return &ast.AlterTypeStmt{
				Type:          typeName,
				AlterItemList: []ast.Node{addEnumValueStmt},
			}, nil
		}
		// TODO(rebelice): support RENAME ENUM VALUE statements
	case *pgquery.Node_TransactionStmt:
		if in.TransactionStmt.Kind == pgquery.TransactionStmtKind_TRANS_STMT_COMMIT {
			return &ast.CommitStmt{}, nil
		}
	default:
		return &ast.UnconvertedStmt{}, nil
	}

	return nil, nil
}

func convertEnumLabelList(list []*pgquery.Node) ([]string, error) {
	var result []string
	for _, node := range list {
		stringNode, ok := node.Node.(*pgquery.Node_String_)
		if !ok {
			return nil, parser.NewConvertErrorf("expected String but found %t", node.Node)
		}
		result = append(result, stringNode.String_.Str)
	}
	return result, nil
}

func convertFunctionParameterList(parameterList []*pgquery.Node) ([]*ast.FunctionParameterDef, error) {
	var result []*ast.FunctionParameterDef
	for _, node := range parameterList {
		var err error
		switch parameterNode := node.Node.(type) {
		case *pgquery.Node_FunctionParameter:
			parameterDef := &ast.FunctionParameterDef{
				Name: parameterNode.FunctionParameter.Name,
				Mode: convertParameterMode(parameterNode.FunctionParameter.Mode),
			}
			parameterDef.Type, err = convertDataType(parameterNode.FunctionParameter.ArgType)
			if err != nil {
				return nil, err
			}
			result = append(result, parameterDef)
		case *pgquery.Node_TypeName:
			parameterDef := &ast.FunctionParameterDef{}
			parameterDef.Type, err = convertDataType(parameterNode.TypeName)
			if err != nil {
				return nil, err
			}
			result = append(result, parameterDef)
		default:
			return nil, parser.NewConvertErrorf("expected FunctionParameter or TypeName but found %t", node.Node)
		}
	}
	return result, nil
}

func convertParameterMode(mode pgquery.FunctionParameterMode) ast.FunctionParameterMode {
	switch mode {
	case pgquery.FunctionParameterMode_FUNC_PARAM_IN:
		return ast.FunctionParameterModeIn
	case pgquery.FunctionParameterMode_FUNC_PARAM_OUT:
		return ast.FunctionParameterModeOut
	case pgquery.FunctionParameterMode_FUNC_PARAM_INOUT:
		return ast.FunctionParameterModeInOut
	case pgquery.FunctionParameterMode_FUNC_PARAM_VARIADIC:
		return ast.FunctionParameterModeVariadic
	default:
		return ast.FunctionParameterModeUndefined
	}
}

// convertObjectName requires one or two nodes, and return two strings.
func convertObjectName(list []*pgquery.Node) (string, string, error) {
	switch len(list) {
	case 2:
		schema, err := convertToString(list[0])
		if err != nil {
			return "", "", err
		}
		name, err := convertToString(list[1])
		if err != nil {
			return "", "", err
		}
		return schema, name, nil
	case 1:
		name, err := convertToString(list[0])
		if err != nil {
			return "", "", err
		}
		return "", name, nil
	default:
		return "", "", parser.NewConvertErrorf("expected 1 or 2 items but found %d", len(list))
	}
}

func convertToString(in *pgquery.Node) (string, error) {
	stringNode, ok := in.Node.(*pgquery.Node_String_)
	if !ok {
		return "", parser.NewConvertErrorf("expected String but found %t", in.Node)
	}
	return stringNode.String_.Str, nil
}

func convertAlterSequence(in *pgquery.AlterSeqStmt) (*ast.AlterSequenceStmt, error) {
	alterSequenceStmt := &ast.AlterSequenceStmt{
		IfExists: in.MissingOk,
	}

	if in.Sequence == nil {
		return nil, parser.NewConvertErrorf("AlterSeqStmt.Sequence is nil")
	}

	alterSequenceStmt.Name = convertRangeVarToSeqName(in.Sequence)

	for _, option := range in.Options {
		defElemNode, ok := option.Node.(*pgquery.Node_DefElem)
		if !ok {
			return nil, parser.NewConvertErrorf("expected DefElem but found %t", option.Node)
		}
		var err error
		switch defElemNode.DefElem.Defname {
		case "as":
			if alterSequenceStmt.Type, err = convertDefElemToSeqType(defElemNode.DefElem); err != nil {
				return nil, err
			}
		case "increment":
			if alterSequenceStmt.IncrementBy, err = convertDefElemNodeIntegerToInt32(defElemNode.DefElem); err != nil {
				return nil, err
			}
		case "start":
			if alterSequenceStmt.StartWith, err = convertDefElemNodeIntegerToInt32(defElemNode.DefElem); err != nil {
				return nil, err
			}
		case "restart":
			if alterSequenceStmt.RestartWith, err = convertDefElemNodeIntegerToInt32(defElemNode.DefElem); err != nil {
				return nil, err
			}
		case "minvalue":
			if alterSequenceStmt.MinValue, err = convertDefElemNodeIntegerToInt32(defElemNode.DefElem); err != nil {
				return nil, err
			}
			if alterSequenceStmt.MinValue == nil {
				alterSequenceStmt.NoMinValue = true
			}
		case "maxvalue":
			if alterSequenceStmt.MaxValue, err = convertDefElemNodeIntegerToInt32(defElemNode.DefElem); err != nil {
				return nil, err
			}
			if alterSequenceStmt.MaxValue == nil {
				alterSequenceStmt.NoMaxValue = true
			}
		case "cache":
			if alterSequenceStmt.Cache, err = convertDefElemNodeIntegerToInt32(defElemNode.DefElem); err != nil {
				return nil, err
			}
		case "cycle":
			var cycle bool
			if cycle, err = convertDefElemNodeIntegerToBool(defElemNode.DefElem); err != nil {
				return nil, err
			}
			alterSequenceStmt.Cycle = &cycle
		case "owned_by":
			owner, err := convertDefElemNodeListToColumnNameDef(defElemNode.DefElem)
			if err != nil {
				return nil, err
			}
			if owner.Table.Database == "" &&
				owner.Table.Schema == "" &&
				owner.Table.Name == "" &&
				owner.ColumnName == "none" {
				alterSequenceStmt.OwnedByNone = true
			} else {
				alterSequenceStmt.OwnedBy = owner
			}
		default:
			return nil, parser.NewConvertErrorf("unsupported option %s", defElemNode.DefElem.Defname)
		}
	}

	return alterSequenceStmt, nil
}

func convertDropBehavior(behavior pgquery.DropBehavior) ast.DropBehavior {
	switch behavior {
	case pgquery.DropBehavior_DROP_CASCADE:
		return ast.DropBehaviorCascade
	case pgquery.DropBehavior_DROP_RESTRICT:
		return ast.DropBehaviorRestrict
	default:
		return ast.DropBehaviorNone
	}
}

func convertRoleSpec(in *pgquery.RoleSpec) (*ast.RoleSpec, error) {
	if in == nil {
		return nil, nil
	}
	switch in.Roletype {
	case pgquery.RoleSpecType_ROLE_SPEC_TYPE_UNDEFINED:
		return &ast.RoleSpec{
			Type:  ast.RoleSpecTypeNone,
			Value: "",
		}, nil
	case pgquery.RoleSpecType_ROLESPEC_CSTRING:
		return &ast.RoleSpec{
			Type:  ast.RoleSpecTypeUser,
			Value: in.Rolename,
		}, nil
	case pgquery.RoleSpecType_ROLESPEC_CURRENT_USER:
		return &ast.RoleSpec{
			Type:  ast.RoleSpecTypeCurrentUser,
			Value: "",
		}, nil
	case pgquery.RoleSpecType_ROLESPEC_SESSION_USER:
		return &ast.RoleSpec{
			Type:  ast.RoleSpecTypeSessionUser,
			Value: "",
		}, nil
	}
	return nil, parser.NewConvertErrorf("unexpected role spec type: %q", in.Roletype.String())
}

func convertNullOrder(order pgquery.SortByNulls) (ast.NullOrderType, error) {
	switch order {
	case pgquery.SortByNulls_SORTBY_NULLS_DEFAULT:
		return ast.NullOrderTypeDefault, nil
	case pgquery.SortByNulls_SORTBY_NULLS_FIRST:
		return ast.NullOrderTypeFirst, nil
	case pgquery.SortByNulls_SORTBY_NULLS_LAST:
		return ast.NullOrderTypeLast, nil
	default:
		return ast.NullOrderTypeDefault, parser.NewConvertErrorf("unsupported null sort order: %d", order)
	}
}

func convertSortOrder(order pgquery.SortByDir) (ast.SortOrderType, error) {
	switch order {
	case pgquery.SortByDir_SORTBY_DEFAULT:
		return ast.SortOrderTypeDefault, nil
	case pgquery.SortByDir_SORTBY_ASC:
		return ast.SortOrderTypeAscending, nil
	case pgquery.SortByDir_SORTBY_DESC:
		return ast.SortOrderTypeDescending, nil
	default:
		return ast.NullOrderTypeDefault, parser.NewConvertErrorf("unsupported sort order: %d", order)
	}
}

func convertMethodType(method string) (ast.IndexMethodType, error) {
	switch method {
	case "btree":
		return ast.IndexMethodTypeBTree, nil
	case "hash":
		return ast.IndexMethodTypeHash, nil
	case "gist":
		return ast.IndexMethodTypeGiST, nil
	case "spgist":
		return ast.IndexMethodTypeSpGiST, nil
	case "gin":
		return ast.IndexMethodTypeGin, nil
	case "brin":
		return ast.IndexMethodTypeBrin, nil
	case "ivfflat":
		return ast.IndexMethodTypeIvfflat, nil
	default:
		// Fallback to btree for index from plugins.
		return ast.IndexMethodTypeBTree, nil
	}
}

func convertExpressionNode(node *pgquery.Node) (ast.ExpressionNode, []*ast.PatternLikeDef, []*ast.SubqueryDef, error) {
	if node == nil || node.Node == nil {
		return &ast.UnconvertedExpressionDef{}, nil, nil, nil
	}
	switch in := node.Node.(type) {
	case *pgquery.Node_AConst:
		return convertExpressionNode(in.AConst.Val)
	case *pgquery.Node_String_:
		return &ast.StringDef{Value: in.String_.Str}, nil, nil, nil
	case *pgquery.Node_ResTarget:
		return convertExpressionNode(in.ResTarget.Val)
	case *pgquery.Node_TypeCast:
		_, likeList, subqueryList, err := convertExpressionNode(in.TypeCast.Arg)
		if err != nil {
			return nil, nil, nil, err
		}
		return &ast.UnconvertedExpressionDef{}, likeList, subqueryList, nil
	case *pgquery.Node_ColumnRef:
		columnDef, err := ConvertNodeListToColumnNameDef(in.ColumnRef.Fields)
		return columnDef, nil, nil, err
	case *pgquery.Node_FuncCall:
		var likeList []*ast.PatternLikeDef
		var subqueryList []*ast.SubqueryDef
		for _, arg := range in.FuncCall.Args {
			_, interLike, interSubquery, err := convertExpressionNode(arg)
			if err != nil {
				return nil, nil, nil, err
			}
			likeList = append(likeList, interLike...)
			subqueryList = append(subqueryList, interSubquery...)
		}
		return &ast.UnconvertedExpressionDef{}, likeList, subqueryList, nil
	case *pgquery.Node_AExpr:
		var likeList, interLike []*ast.PatternLikeDef
		var subqueryList, interSubquery []*ast.SubqueryDef
		var lExpr, rExpr ast.ExpressionNode
		var err error
		if in.AExpr.Lexpr != nil {
			if lExpr, interLike, interSubquery, err = convertExpressionNode(in.AExpr.Lexpr); err != nil {
				return nil, nil, nil, err
			}
			likeList = append(likeList, interLike...)
			subqueryList = append(subqueryList, interSubquery...)
		}
		if in.AExpr.Rexpr != nil {
			if rExpr, interLike, interSubquery, err = convertExpressionNode(in.AExpr.Rexpr); err != nil {
				return nil, nil, nil, err
			}
			likeList = append(likeList, interLike...)
			subqueryList = append(subqueryList, interSubquery...)
		}
		if len(in.AExpr.Name) == 1 {
			name, ok := in.AExpr.Name[0].Node.(*pgquery.Node_String_)
			if !ok {
				return nil, nil, nil, parser.NewConvertErrorf("expected String but found %t", in.AExpr.Name[0].Node)
			}
			switch name.String_.Str {
			// LIKE
			case operatorLike, operatorNotLike:
				like := &ast.PatternLikeDef{
					Not:        (name.String_.Str == operatorNotLike),
					Expression: lExpr,
					Pattern:    rExpr,
				}
				likeList = append(likeList, like)
				return like, likeList, interSubquery, nil
			}
		}
		return &ast.UnconvertedExpressionDef{}, likeList, subqueryList, nil
	case *pgquery.Node_BoolExpr:
		var likeList []*ast.PatternLikeDef
		var subqueryList []*ast.SubqueryDef
		for _, arg := range in.BoolExpr.Args {
			_, interLike, interSubquery, err := convertExpressionNode(arg)
			if err != nil {
				return nil, nil, nil, err
			}
			likeList = append(likeList, interLike...)
			subqueryList = append(subqueryList, interSubquery...)
		}
		return &ast.UnconvertedExpressionDef{}, likeList, subqueryList, nil
	case *pgquery.Node_SubLink:
		if subselectNode, ok := in.SubLink.Subselect.Node.(*pgquery.Node_SelectStmt); ok {
			subselect, err := convertSelectStmt(subselectNode.SelectStmt)
			if err != nil {
				return nil, nil, nil, err
			}
			subQuery := &ast.SubqueryDef{Select: subselect}
			return subQuery, nil, []*ast.SubqueryDef{subQuery}, nil
		}
	}
	return &ast.UnconvertedExpressionDef{}, nil, nil, nil
}

// convertCreateStmt convert pgquery create stmt to Bytebase create table stmt node.
func convertCreateStmt(in *pgquery.CreateStmt) (*ast.CreateTableStmt, error) {
	table := &ast.CreateTableStmt{
		IfNotExists: in.IfNotExists,
		Name:        convertRangeVarToTableName(in.Relation, ast.TableTypeBaseTable),
	}

	for _, elt := range in.TableElts {
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

	if in.Partspec != nil {
		// TODO(rebelice): convert the partition definition.
		table.PartitionDef = &ast.UnconvertedStmt{}
	}
	return table, nil
}

func convertSelectStmt(in *pgquery.SelectStmt) (*ast.SelectStmt, error) {
	selectStmt := &ast.SelectStmt{}

	setOperation, err := convertSetOperation(in.Op)
	if err != nil {
		return nil, err
	}

	selectStmt.SetOperation = setOperation
	if setOperation != ast.SetOperationTypeNone {
		lQuery, err := convertSelectStmt(in.Larg)
		if err != nil {
			return nil, err
		}
		rQuery, err := convertSelectStmt(in.Rarg)
		if err != nil {
			return nil, err
		}
		selectStmt.LQuery = lQuery
		selectStmt.RQuery = rQuery
		return selectStmt, nil
	}

	// Convert target list
	for _, node := range in.TargetList {
		convertedNode, _, _, err := convertExpressionNode(node)
		if err != nil {
			return nil, err
		}
		selectStmt.FieldList = append(selectStmt.FieldList, convertedNode)
	}
	// Convert FROM clause
	// Here we only find the SELECT stmt in FROM clause
	for _, item := range in.FromClause {
		if node, ok := item.Node.(*pgquery.Node_RangeSubselect); ok {
			subselect, err := convertRangeSubselect(node.RangeSubselect)
			if err != nil {
				return nil, err
			}
			selectStmt.SubqueryList = append(selectStmt.SubqueryList, subselect)
		}
	}
	// Convert WHERE clause
	if in.WhereClause != nil {
		var err error
		var subqueryList []*ast.SubqueryDef
		selectStmt.WhereClause, selectStmt.PatternLikeList, subqueryList, err = convertExpressionNode(in.WhereClause)
		if err != nil {
			return nil, err
		}
		selectStmt.SubqueryList = append(selectStmt.SubqueryList, subqueryList...)
	}

	// Convert ORDER BY clause.
	for _, itemNode := range in.SortClause {
		item, ok := itemNode.Node.(*pgquery.Node_SortBy)
		if !ok {
			return nil, parser.NewConvertErrorf("expected SortBy but found %t", itemNode.Node)
		}
		expression, _, _, err := convertExpressionNode(item.SortBy.Node)
		if err != nil {
			return nil, err
		}

		text, err := pgquery.DeparseNode(pgquery.DeparseTypeExpr, item.SortBy.Node)
		if err != nil {
			fmt.Printf("itemNode %t %v", itemNode.Node, item)
			return nil, err
		}
		expression.SetText(text)
		selectStmt.OrderByClause = append(selectStmt.OrderByClause, &ast.ByItemDef{Expression: expression})
	}
	return selectStmt, nil
}

func convertRangeSubselect(node *pgquery.RangeSubselect) (*ast.SubqueryDef, error) {
	subselect, ok := node.Subquery.Node.(*pgquery.Node_SelectStmt)
	if !ok {
		return nil, parser.NewConvertErrorf("expected SELECT but found %t", node.Subquery.Node)
	}
	res, err := convertSelectStmt(subselect.SelectStmt)
	if err != nil {
		return nil, err
	}
	return &ast.SubqueryDef{Select: res}, nil
}

func convertSetOperation(t pgquery.SetOperation) (ast.SetOperationType, error) {
	switch t {
	case pgquery.SetOperation_SETOP_NONE:
		return ast.SetOperationTypeNone, nil
	case pgquery.SetOperation_SETOP_UNION:
		return ast.SetOperationTypeUnion, nil
	case pgquery.SetOperation_SETOP_INTERSECT:
		return ast.SetOperationTypeIntersect, nil
	case pgquery.SetOperation_SETOP_EXCEPT:
		return ast.SetOperationTypeExcept, nil
	default:
		return 0, errors.Errorf("failed to parse set operation: unknown type %s", t)
	}
}

func convertListToSequenceNameDef(in *pgquery.Node_List) (*ast.SequenceNameDef, error) {
	stringList, err := convertListToStringList(in)
	if err != nil {
		return &ast.SequenceNameDef{}, err
	}
	switch len(in.List.Items) {
	case 2:
		return &ast.SequenceNameDef{
			Schema: stringList[0],
			Name:   stringList[1],
		}, nil
	case 1:
		return &ast.SequenceNameDef{
			Name: stringList[0],
		}, nil
	default:
		return &ast.SequenceNameDef{}, parser.NewConvertErrorf("expected length is 1 or 2, but found %d", len(in.List.Items))
	}
}

func convertListToTableDef(in *pgquery.Node_List, tableType ast.TableType) (*ast.TableDef, error) {
	stringList, err := convertListToStringList(in)
	if err != nil {
		return nil, err
	}
	switch len(in.List.Items) {
	case 2:
		return &ast.TableDef{
			Type:   tableType,
			Schema: stringList[0],
			Name:   stringList[1],
		}, nil
	case 1:
		return &ast.TableDef{
			Type: tableType,
			Name: stringList[0],
		}, nil
	default:
		return nil, parser.NewConvertErrorf("expected length is 1 or 2, but found %d", len(in.List.Items))
	}
}

func convertListToIndexDef(in *pgquery.Node_List, tableType ast.TableType) (*ast.IndexDef, error) {
	stringList, err := convertListToStringList(in)
	if err != nil {
		return nil, err
	}
	indexDef := &ast.IndexDef{}
	switch len(in.List.Items) {
	case 2:
		indexDef.Table = &ast.TableDef{
			Type:   tableType,
			Schema: stringList[0],
		}
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

func convertRangeVarToTableName(in *pgquery.RangeVar, tableType ast.TableType) *ast.TableDef {
	return &ast.TableDef{
		Type:     tableType,
		Database: in.Catalogname,
		Schema:   in.Schemaname,
		Name:     in.Relname,
	}
}

func convertRangeVarToSeqName(in *pgquery.RangeVar) *ast.SequenceNameDef {
	return &ast.SequenceNameDef{
		Schema: in.Schemaname,
		Name:   in.Relname,
	}
}

func convertRangeVarToIndexTableName(in *pgquery.RangeVar, tableType ast.TableType) *ast.TableDef {
	return &ast.TableDef{
		Type:   tableType,
		Schema: in.Schemaname,
	}
}

func convertConstraint(in *pgquery.Node_Constraint) (*ast.ConstraintDef, error) {
	cons := &ast.ConstraintDef{
		Name:           in.Constraint.Conname,
		Type:           convertConstraintType(in.Constraint.Contype, in.Constraint.Indexname != ""),
		SkipValidation: in.Constraint.SkipValidation,
		Deferrable:     in.Constraint.Deferrable,
		Initdeferred:   in.Constraint.Initdeferred,
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
		for _, col := range in.Constraint.Including {
			name, ok := col.Node.(*pgquery.Node_String_)
			if !ok {
				return nil, parser.NewConvertErrorf("expected String but found %t", col.Node)
			}
			cons.Including = append(cons.Including, name.String_.Str)
		}
		cons.IndexTableSpace = in.Constraint.Indexspace
	case ast.ConstraintTypeForeign:
		cons.Foreign = &ast.ForeignDef{
			Table: convertRangeVarToTableName(in.Constraint.Pktable, ast.TableTypeBaseTable),
		}

		var err error
		if cons.Foreign.MatchType, err = convertForeignMatchType(in.Constraint.FkMatchtype); err != nil {
			return nil, err
		}
		if cons.Foreign.OnUpdate, err = convertReferentialAction(in.Constraint.FkUpdAction); err != nil {
			return nil, err
		}
		if cons.Foreign.OnDelete, err = convertReferentialAction(in.Constraint.FkDelAction); err != nil {
			return nil, err
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
	case ast.ConstraintTypeCheck, ast.ConstraintTypeDefault:
		expression, _, _, err := convertExpressionNode(in.Constraint.RawExpr)
		if err != nil {
			return nil, err
		}
		text, err := pgquery.DeparseNode(pgquery.DeparseTypeExpr, in.Constraint.RawExpr)
		if err != nil {
			return nil, err
		}
		expression.SetText(text)
		cons.Expression = expression
	case ast.ConstraintTypeExclusion:
		if len(in.Constraint.Exclusions) >= 1 {
			exclusion, err := pgquery.DeparseNodes(pgquery.DeparseTypeExclusion, in.Constraint.Exclusions)
			if err != nil {
				return nil, err
			}
			cons.Exclusions = exclusion
		}
		var err error
		if cons.AccessMethod, err = convertMethodType(in.Constraint.AccessMethod); err != nil {
			return nil, err
		}
		if in.Constraint.WhereClause != nil {
			whereClause, err := pgquery.DeparseNode(pgquery.DeparseTypeExpr, in.Constraint.WhereClause)
			if err != nil {
				return nil, err
			}
			cons.WhereClause = whereClause
		}
	}

	return cons, nil
}

func convertForeignMatchType(tp string) (ast.ForeignMatchType, error) {
	switch tp {
	case "s":
		return ast.ForeignMatchTypeSimple, nil
	case "f":
		return ast.ForeignMatchTypeFull, nil
	case "p":
		return ast.ForeignMatchTypePartial, nil
	default:
		return ast.ForeignMatchTypeSimple, parser.NewConvertErrorf("unsupported foreign match type: %s", tp)
	}
}

func convertReferentialAction(action string) (*ast.ReferentialActionDef, error) {
	switch action {
	case "a":
		return &ast.ReferentialActionDef{Type: ast.ReferentialActionTypeNoAction}, nil
	case "r":
		return &ast.ReferentialActionDef{Type: ast.ReferentialActionTypeRestrict}, nil
	case "c":
		return &ast.ReferentialActionDef{Type: ast.ReferentialActionTypeCascade}, nil
	case "n":
		return &ast.ReferentialActionDef{Type: ast.ReferentialActionTypeSetNull}, nil
	case "d":
		return &ast.ReferentialActionDef{Type: ast.ReferentialActionTypeSetDefault}, nil
	default:
		return nil, parser.NewConvertErrorf("unsupported referential action: %s", action)
	}
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
	case pgquery.ConstrType_CONSTR_NOTNULL:
		return ast.ConstraintTypeNotNull
	case pgquery.ConstrType_CONSTR_CHECK:
		return ast.ConstraintTypeCheck
	case pgquery.ConstrType_CONSTR_DEFAULT:
		return ast.ConstraintTypeDefault
	case pgquery.ConstrType_CONSTR_EXCLUSION:
		return ast.ConstraintTypeExclusion
	}
	return ast.ConstraintTypeUndefined
}

func convertColumnDef(in *pgquery.Node_ColumnDef) (*ast.ColumnDef, error) {
	column := &ast.ColumnDef{
		ColumnName: in.ColumnDef.Colname,
	}
	columnType, err := convertDataType(in.ColumnDef.TypeName)
	if err != nil {
		return nil, err
	}
	column.Type = columnType

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

	if in.ColumnDef.CollClause != nil {
		collation, err := convertCollationName(in.ColumnDef.CollClause)
		if err != nil {
			return nil, err
		}
		column.Collation = collation
	}

	return column, nil
}

func convertCollationName(collation *pgquery.CollateClause) (*ast.CollationNameDef, error) {
	result := &ast.CollationNameDef{}
	var err error
	switch len(collation.Collname) {
	case 1:
		result.Name, err = convertToString(collation.Collname[0])
		if err != nil {
			return nil, err
		}
	case 2:
		result.Schema, err = convertToString(collation.Collname[0])
		if err != nil {
			return nil, err
		}
		result.Name, err = convertToString(collation.Collname[1])
		if err != nil {
			return nil, err
		}
	default:
		return nil, parser.NewConvertErrorf("expected one or two length but found %d", len(collation.Collname))
	}
	return result, nil
}

func convertToTableType(relationType pgquery.ObjectType) (ast.TableType, error) {
	switch relationType {
	case pgquery.ObjectType_OBJECT_TABLE:
		return ast.TableTypeBaseTable, nil
	case pgquery.ObjectType_OBJECT_VIEW:
		return ast.TableTypeView, nil
	default:
		return ast.TableTypeUnknown, parser.NewConvertErrorf("expected TABLE or VIEW but found %s", relationType)
	}
}

func stripPgCatalogPrefix(tp *pgquery.TypeName) *pgquery.TypeName {
	// The built-in data type may have the "pg_catalog" prefix.
	if len(tp.Names) > 0 {
		if first, ok := tp.Names[0].Node.(*pgquery.Node_String_); ok && first.String_.Str == "pg_catalog" {
			tp.Names = tp.Names[1:]
		}
	}
	return tp
}

func convertDataType(tp *pgquery.TypeName) (ast.DataType, error) {
	text, err := pgquery.DeparseNode(pgquery.DeparseTypeDataType, &pgquery.Node{Node: &pgquery.Node_TypeName{TypeName: tp}})
	if err != nil {
		return nil, err
	}

	dataType := func() ast.DataType {
		tp = stripPgCatalogPrefix(tp)
		if len(tp.Names) == 1 {
			name, ok := tp.Names[0].Node.(*pgquery.Node_String_)
			if !ok {
				return &ast.UnconvertedDataType{}
			}
			s := name.String_.Str
			switch {
			case strings.HasPrefix(s, "int"):
				size, err := strconv.Atoi(s[3:])
				if err != nil {
					return convertToUnconvertedDataType(tp)
				}
				return &ast.Integer{Size: size}
			case strings.HasPrefix(s, "float"):
				size, err := strconv.Atoi(s[5:])
				if err != nil {
					return convertToUnconvertedDataType(tp)
				}
				return &ast.Float{Size: size}
			case s == "serial":
				return &ast.Serial{Size: 4}
			case s == "smallserial":
				return &ast.Serial{Size: 2}
			case s == "bigserial":
				return &ast.Serial{Size: 8}
			case strings.HasPrefix(s, "serial"):
				size, err := strconv.Atoi(s[6:])
				if err != nil {
					return convertToUnconvertedDataType(tp)
				}
				return &ast.Serial{Size: size}
			case s == "numeric":
				return convertToDecimal(tp.Typmods)
			case s == "bpchar":
				return convertToCharacter(tp.Typmods)
			case s == "varchar":
				return convertToVarchar(tp.Typmods)
			case s == "text":
				return &ast.Text{}
			}
		}
		return convertToUnconvertedDataType(tp)
	}()

	// For UnconvertedDataType, we use the text deparsed by pg_query_go
	if _, ok := dataType.(*ast.UnconvertedDataType); ok {
		dataType.SetText(text)
	}
	return dataType, nil
}

func convertToUnconvertedDataType(tp *pgquery.TypeName) ast.DataType {
	res := &ast.UnconvertedDataType{}
	for _, name := range tp.Names {
		s, ok := name.Node.(*pgquery.Node_String_)
		if !ok {
			return &ast.UnconvertedDataType{}
		}
		res.Name = append(res.Name, s.String_.Str)
	}
	return res
}

func convertToDecimal(typmods []*pgquery.Node) ast.DataType {
	ok := false
	decimal := &ast.Decimal{}
	switch len(typmods) {
	case 0:
		return decimal
	case 1:
		if decimal.Precision, ok = convertToInteger(typmods[0]); !ok {
			return &ast.UnconvertedDataType{}
		}
		return decimal
	case 2:
		if decimal.Precision, ok = convertToInteger(typmods[0]); !ok {
			return &ast.UnconvertedDataType{}
		}
		if decimal.Scale, ok = convertToInteger(typmods[1]); !ok {
			return &ast.UnconvertedDataType{}
		}
		return decimal
	default:
		return &ast.UnconvertedDataType{}
	}
}

func convertToVarchar(typmods []*pgquery.Node) ast.DataType {
	if len(typmods) != 1 {
		return &ast.UnconvertedDataType{}
	}
	size, ok := convertToInteger(typmods[0])
	if !ok {
		return &ast.UnconvertedDataType{}
	}
	return &ast.CharacterVarying{Size: size}
}

func convertToCharacter(typmods []*pgquery.Node) ast.DataType {
	if len(typmods) != 1 {
		return &ast.UnconvertedDataType{}
	}
	size, ok := convertToInteger(typmods[0])
	if !ok {
		return &ast.UnconvertedDataType{}
	}
	return &ast.Character{Size: size}
}

func convertToInteger(in *pgquery.Node) (int, bool) {
	aConst, ok := in.Node.(*pgquery.Node_AConst)
	if !ok {
		return 0, false
	}
	integer, ok := aConst.AConst.Val.Node.(*pgquery.Node_Integer)
	if !ok {
		return 0, false
	}
	return int(integer.Integer.Ival), true
}

func convertDefElemNodeListToColumnNameDef(defElem *pgquery.DefElem) (*ast.ColumnNameDef, error) {
	listNode, ok := defElem.Arg.Node.(*pgquery.Node_List)
	if !ok {
		return nil, parser.NewConvertErrorf("expected List but found %T", defElem.Arg.Node)
	}
	return ConvertNodeListToColumnNameDef(listNode.List.Items)
}

func convertDefElemNodeIntegerToBool(defElem *pgquery.DefElem) (bool, error) {
	if defElem.Arg == nil {
		return false, nil
	}
	interger, ok := defElem.Arg.Node.(*pgquery.Node_Integer)
	if !ok {
		return false, parser.NewConvertErrorf("expected integer but found %T", defElem.Arg.Node)
	}
	return interger.Integer.Ival == 1, nil
}

func convertDefElemNodeIntegerToInt32(defElem *pgquery.DefElem) (*int32, error) {
	if defElem.Arg == nil {
		return nil, nil
	}
	interger, ok := defElem.Arg.Node.(*pgquery.Node_Integer)
	if !ok {
		return nil, parser.NewConvertErrorf("expected integer but found %T", defElem.Arg.Node)
	}
	val := interger.Integer.Ival
	return &val, nil
}

func convertDefElemToSeqType(defElem *pgquery.DefElem) (*ast.Integer, error) {
	typeNameNode, ok := defElem.Arg.Node.(*pgquery.Node_TypeName)
	if !ok {
		return nil, parser.NewConvertErrorf("expected TypeName but found %T", defElem.Arg.Node)
	}
	if len(typeNameNode.TypeName.Names) != 2 {
		return nil, parser.NewConvertErrorf("expected TypeName with 2 names but found %d", len(typeNameNode.TypeName.Names))
	}
	dataType, err := convertDataType(typeNameNode.TypeName)
	if err != nil {
		return nil, err
	}
	// Sequence type should be int2(smallint), int4(integer), or int8(bigint)
	intType, ok := dataType.(*ast.Integer)
	if !ok {
		return nil, parser.NewConvertErrorf("expected Integer but found %T", dataType)
	}
	if intType.Size != 2 && intType.Size != 4 && intType.Size != 8 {
		return nil, parser.NewConvertErrorf("expected Integer with size 2, 4, or 8 but found %d", intType.Size)
	}
	return intType, nil
}

// ConvertNodeListToColumnNameDef converts the node list to ColumnNameDef.
func ConvertNodeListToColumnNameDef(in []*pgquery.Node) (*ast.ColumnNameDef, error) {
	columnName := &ast.ColumnNameDef{Table: &ast.TableDef{}}
	// There are three cases for column name:
	//   1. schemaName.tableName.columnName
	//   2. tableName.columnName
	//   3. columnName
	// The pg parser will split them by ".", and use a list to define it.
	// So we need to consider this three cases.
	switch len(in) {
	// schemaName.tableName.columName
	case 3:
		schema, ok := in[0].Node.(*pgquery.Node_String_)
		if !ok {
			return nil, parser.NewConvertErrorf("expected String but found %t", in[2].Node)
		}
		columnName.Table.Schema = schema.String_.Str
		// need to convert tableName.columnName
		in = in[1:]
		fallthrough
	// tableName.columnName
	case 2:
		table, ok := in[0].Node.(*pgquery.Node_String_)
		if !ok {
			return nil, parser.NewConvertErrorf("expected String but found %t", in[1].Node)
		}
		columnName.Table.Name = table.String_.Str
		// need to convert columnName
		in = in[1:]
		fallthrough
	// columnName
	case 1:
		switch column := in[0].Node.(type) {
		// column name
		case *pgquery.Node_String_:
			columnName.ColumnName = column.String_.Str
		// e.g. SELECT * FROM t;
		case *pgquery.Node_AStar:
			columnName.ColumnName = "*"
		default:
			return nil, parser.NewConvertErrorf("expected String or AStar but found %t", in[0].Node)
		}
	default:
		return nil, parser.NewConvertErrorf("failed to convert ColumnRef, column name contains unexpected components: %v", in)
	}
	return columnName, nil
}
