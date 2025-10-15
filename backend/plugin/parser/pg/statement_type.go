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
		t := getStatementType(node)
		sqlTypeSet[t] = true
	}
	var sqlTypes []string
	for sqlType := range sqlTypeSet {
		sqlTypes = append(sqlTypes, sqlType)
	}
	return sqlTypes, nil
}

func getStatementType(node ast.Node) string {
	switch node := node.(type) {
	// DDL

	// CREATE
	case *ast.CreateIndexStmt:
		return "CREATE_INDEX"
	case *ast.CreateTableStmt:
		switch node.Name.Type {
		case ast.TableTypeView:
			return "CREATE_VIEW"
		case ast.TableTypeBaseTable:
			return "CREATE_TABLE"
		default:
			return "CREATE_TABLE"
		}
	case *ast.CreateSequenceStmt:
		return "CREATE_SEQUENCE"
	case *ast.CreateDatabaseStmt:
		return "CREATE_DATABASE"
	case *ast.CreateSchemaStmt:
		return "CREATE_SCHEMA"
	case *ast.CreateFunctionStmt:
		return "CREATE_FUNCTION"
	case *ast.CreateTriggerStmt:
		return "CREATE_TRIGGER"
	case *ast.CreateTypeStmt:
		return "CREATE_TYPE"
	case *ast.CreateExtensionStmt:
		return "CREATE_EXTENSION"

	// DROP
	case *ast.DropColumnStmt:
		return "DROP_COLUMN"
	case *ast.DropConstraintStmt:
		return "DROP_CONSTRAINT"
	case *ast.DropDatabaseStmt:
		return "DROP_DATABASE"
	case *ast.DropDefaultStmt:
		return "DROP_DEFAULT"
	case *ast.DropExtensionStmt:
		return "DROP_EXTENSION"
	case *ast.DropFunctionStmt:
		return "DROP_FUNCTION"
	case *ast.DropIndexStmt:
		return "DROP_INDEX"
	case *ast.DropNotNullStmt:
		return "DROP_NOT_NULL"
	case *ast.DropSchemaStmt:
		return "DROP_SCHEMA"
	case *ast.DropSequenceStmt:
		return "DROP_SEQUENCE"
	case *ast.DropTableStmt:
		return "DROP_TABLE"
	case *ast.DropTriggerStmt:
		return "DROP_TRIGGER"
	case *ast.DropTypeStmt:
		return "DROP_TYPE"

		// ALTER
	case *ast.AlterColumnTypeStmt:
		return "ALTER_COLUMN_TYPE"
	case *ast.AlterSequenceStmt:
		return "ALTER_SEQUENCE"
	case *ast.AlterTableStmt:
		switch node.Table.Type {
		case ast.TableTypeView:
			return "ALTER_VIEW"
		case ast.TableTypeBaseTable:
			return "ALTER_TABLE"
		default:
			return "ALTER_TABLE"
		}
	case *ast.AlterTypeStmt:
		return "ALTER_TYPE"

	case *ast.AddColumnListStmt:
		return "ALTER_TABLE_ADD_COLUMN_LIST"
	case *ast.AddConstraintStmt:
		return "ALTER_TABLE_ADD_CONSTRAINT"

	// RENAME
	case *ast.RenameColumnStmt:
		return "RENAME_COLUMN"
	case *ast.RenameConstraintStmt:
		return "RENAME_CONSTRAINT"
	case *ast.RenameIndexStmt:
		return "RENAME_INDEX"
	case *ast.RenameSchemaStmt:
		return "RENAME_SCHEMA"
	case *ast.RenameTableStmt:
		switch node.Table.Type {
		case ast.TableTypeView:
			return "RENAME_VIEW"
		case ast.TableTypeBaseTable:
			return "RENAME_TABLE"
		default:
			return "RENAME_TABLE"
		}

	case *ast.CommentStmt:
		return "COMMENT"

	// DML

	case *ast.InsertStmt:
		return "INSERT"
	case *ast.UpdateStmt:
		return "UPDATE"
	case *ast.DeleteStmt:
		return "DELETE"
	}

	return "UNKNOWN"
}
