package pg

import (
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/plugin/parser/pg/legacy/ast"
)

func GetStatementTypes(asts any) ([]string, error) {
	nodes, ok := asts.([]ast.Node)
	if !ok {
		return nil, errors.Errorf("invalid ast type %T", asts)
	}
	sqlTypeSet := make(map[string]bool)
	for _, node := range nodes {
		types := getStatementType(node)
		for _, t := range types {
			sqlTypeSet[t] = true
		}
	}
	var sqlTypes []string
	for sqlType := range sqlTypeSet {
		sqlTypes = append(sqlTypes, sqlType)
	}
	return sqlTypes, nil
}

func getStatementType(node ast.Node) []string {
	switch node := node.(type) {
	// DDL

	// CREATE
	case *ast.CreateIndexStmt:
		return []string{"CREATE_INDEX"}
	case *ast.CreateTableStmt:
		switch node.Name.Type {
		case ast.TableTypeView, ast.TableTypeMaterializedView:
			return []string{"CREATE_VIEW"}
		case ast.TableTypeBaseTable:
			return []string{"CREATE_TABLE"}
		default:
			return []string{"CREATE_TABLE"}
		}
	case *ast.CreateViewStmt, *ast.CreateMaterializedViewStmt:
		return []string{"CREATE_VIEW"}
	case *ast.CreateSequenceStmt:
		return []string{"CREATE_SEQUENCE"}
	case *ast.CreateDatabaseStmt:
		return []string{"CREATE_DATABASE"}
	case *ast.CreateSchemaStmt:
		return []string{"CREATE_SCHEMA"}
	case *ast.CreateFunctionStmt:
		return []string{"CREATE_FUNCTION"}
	case *ast.CreateTriggerStmt:
		return []string{"CREATE_TRIGGER"}
	case *ast.CreateTypeStmt:
		return []string{"CREATE_TYPE"}
	case *ast.CreateExtensionStmt:
		return []string{"CREATE_EXTENSION"}

	// DROP
	case *ast.DropColumnStmt:
		return []string{"DROP_COLUMN"}
	case *ast.DropConstraintStmt:
		return []string{"DROP_CONSTRAINT"}
	case *ast.DropDatabaseStmt:
		return []string{"DROP_DATABASE"}
	case *ast.DropDefaultStmt:
		return []string{"DROP_DEFAULT"}
	case *ast.DropExtensionStmt:
		return []string{"DROP_EXTENSION"}
	case *ast.DropFunctionStmt:
		return []string{"DROP_FUNCTION"}
	case *ast.DropIndexStmt:
		return []string{"DROP_INDEX"}
	case *ast.DropNotNullStmt:
		return []string{"DROP_NOT_NULL"}
	case *ast.DropSchemaStmt:
		return []string{"DROP_SCHEMA"}
	case *ast.DropSequenceStmt:
		return []string{"DROP_SEQUENCE"}
	case *ast.DropTableStmt:
		var types []string
		for _, table := range node.TableList {
			switch table.Type {
			case ast.TableTypeView, ast.TableTypeMaterializedView:
				types = append(types, "DROP_VIEW")
			case ast.TableTypeBaseTable:
				types = append(types, "DROP_TABLE")
			default:
				types = append(types, "DROP_TABLE")
			}
		}
		return types
	case *ast.DropTriggerStmt:
		return []string{"DROP_TRIGGER"}
	case *ast.DropTypeStmt:
		return []string{"DROP_TYPE"}

		// ALTER
	case *ast.AlterColumnTypeStmt:
		return []string{"ALTER_COLUMN_TYPE"}
	case *ast.AlterSequenceStmt:
		return []string{"ALTER_SEQUENCE"}
	case *ast.AlterTableStmt:
		switch node.Table.Type {
		case ast.TableTypeView, ast.TableTypeMaterializedView:
			return []string{"ALTER_VIEW"}
		case ast.TableTypeBaseTable:
			return []string{"ALTER_TABLE"}
		default:
			return []string{"ALTER_TABLE"}
		}
	case *ast.AlterTypeStmt:
		return []string{"ALTER_TYPE"}

	case *ast.AddColumnListStmt:
		return []string{"ALTER_TABLE_ADD_COLUMN_LIST"}
	case *ast.AddConstraintStmt:
		return []string{"ALTER_TABLE_ADD_CONSTRAINT"}

	// RENAME
	case *ast.RenameColumnStmt:
		return []string{"RENAME_COLUMN"}
	case *ast.RenameConstraintStmt:
		return []string{"RENAME_CONSTRAINT"}
	case *ast.RenameIndexStmt:
		return []string{"RENAME_INDEX"}
	case *ast.RenameSchemaStmt:
		return []string{"RENAME_SCHEMA"}
	case *ast.RenameTableStmt:
		switch node.Table.Type {
		case ast.TableTypeView, ast.TableTypeMaterializedView:
			return []string{"RENAME_VIEW"}
		case ast.TableTypeBaseTable:
			return []string{"RENAME_TABLE"}
		default:
			return []string{"RENAME_TABLE"}
		}

	case *ast.CommentStmt:
		return []string{"COMMENT"}

	// DML

	case *ast.InsertStmt:
		return []string{"INSERT"}
	case *ast.UpdateStmt:
		return []string{"UPDATE"}
	case *ast.DeleteStmt:
		return []string{"DELETE"}
	}

	return []string{"UNKNOWN"}
}
