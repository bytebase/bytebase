package pg

import (
	"github.com/bytebase/omni/pg/ast"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

// classifyStatementType returns the statement type for an omni AST node.
func classifyStatementType(node ast.Node) storepb.StatementType {
	switch n := node.(type) {
	// DDL - CREATE
	case *ast.CreateStmt:
		return storepb.StatementType_CREATE_TABLE
	case *ast.ViewStmt:
		return storepb.StatementType_CREATE_VIEW
	case *ast.IndexStmt:
		return storepb.StatementType_CREATE_INDEX
	case *ast.CreateSeqStmt:
		return storepb.StatementType_CREATE_SEQUENCE
	case *ast.CreateSchemaStmt:
		return storepb.StatementType_CREATE_SCHEMA
	case *ast.CreateFunctionStmt:
		return storepb.StatementType_CREATE_FUNCTION
	case *ast.CreateTrigStmt:
		return storepb.StatementType_CREATE_TRIGGER
	case *ast.CreateExtensionStmt:
		return storepb.StatementType_CREATE_EXTENSION
	case *ast.CreatedbStmt:
		return storepb.StatementType_CREATE_DATABASE
	case *ast.DefineStmt:
		if n.Kind == ast.OBJECT_TYPE {
			return storepb.StatementType_CREATE_TYPE
		}
		return storepb.StatementType_STATEMENT_TYPE_UNSPECIFIED
	case *ast.CreateEnumStmt:
		return storepb.StatementType_CREATE_TYPE

	// DDL - DROP
	case *ast.DropStmt:
		return getDropStatementTypeFromOmni(n)
	case *ast.DropdbStmt:
		return storepb.StatementType_DROP_DATABASE
	case *ast.TruncateStmt:
		return storepb.StatementType_TRUNCATE

	// DDL - ALTER
	case *ast.AlterTableStmt:
		if ast.ObjectType(n.ObjType) == ast.OBJECT_VIEW {
			return storepb.StatementType_ALTER_VIEW
		}
		return storepb.StatementType_ALTER_TABLE
	case *ast.AlterSeqStmt:
		return storepb.StatementType_ALTER_SEQUENCE
	case *ast.AlterEnumStmt:
		return storepb.StatementType_ALTER_TYPE

	// DDL - RENAME
	case *ast.RenameStmt:
		return getRenameStatementType(n)

	// DDL - COMMENT
	case *ast.CommentStmt:
		return storepb.StatementType_COMMENT

	// DDL - DROP FUNCTION (AlterFunctionStmt with drop action, or separate node)
	case *ast.AlterFunctionStmt:
		// This is ALTER FUNCTION, not DROP FUNCTION. Not classified in original code.
		return storepb.StatementType_STATEMENT_TYPE_UNSPECIFIED

	// DML
	case *ast.InsertStmt:
		return storepb.StatementType_INSERT
	case *ast.UpdateStmt:
		return storepb.StatementType_UPDATE
	case *ast.DeleteStmt:
		return storepb.StatementType_DELETE

	default:
		return storepb.StatementType_STATEMENT_TYPE_UNSPECIFIED
	}
}

// getDropStatementTypeFromOmni determines the specific DROP statement type from omni AST.
func getDropStatementTypeFromOmni(n *ast.DropStmt) storepb.StatementType {
	objType := ast.ObjectType(n.RemoveType)
	switch objType {
	case ast.OBJECT_TABLE:
		return storepb.StatementType_DROP_TABLE
	case ast.OBJECT_VIEW:
		return storepb.StatementType_DROP_VIEW
	case ast.OBJECT_MATVIEW:
		// DROP MATERIALIZED VIEW holds data — treat as DROP_TABLE for risk assessment.
		return storepb.StatementType_DROP_TABLE
	case ast.OBJECT_INDEX:
		return storepb.StatementType_DROP_INDEX
	case ast.OBJECT_SEQUENCE:
		return storepb.StatementType_DROP_SEQUENCE
	case ast.OBJECT_SCHEMA:
		return storepb.StatementType_DROP_SCHEMA
	case ast.OBJECT_EXTENSION:
		return storepb.StatementType_DROP_EXTENSION
	case ast.OBJECT_TYPE:
		return storepb.StatementType_DROP_TYPE
	case ast.OBJECT_TRIGGER:
		return storepb.StatementType_DROP_TRIGGER
	case ast.OBJECT_FUNCTION:
		return storepb.StatementType_DROP_FUNCTION
	default:
		return storepb.StatementType_STATEMENT_TYPE_UNSPECIFIED
	}
}

// getRenameStatementType determines the statement type for a RenameStmt.
func getRenameStatementType(n *ast.RenameStmt) storepb.StatementType {
	switch n.RenameType {
	case ast.OBJECT_INDEX:
		return storepb.StatementType_RENAME_INDEX
	case ast.OBJECT_SCHEMA:
		return storepb.StatementType_RENAME_SCHEMA
	case ast.OBJECT_SEQUENCE:
		return storepb.StatementType_RENAME_SEQUENCE
	case ast.OBJECT_VIEW:
		return storepb.StatementType_ALTER_VIEW
	case ast.OBJECT_TABLE, ast.OBJECT_COLUMN, ast.OBJECT_TABCONSTRAINT:
		// RENAME TABLE, RENAME COLUMN, RENAME CONSTRAINT all return ALTER_TABLE
		// Check RelationType for VIEW
		if n.RelationType == ast.OBJECT_VIEW {
			return storepb.StatementType_ALTER_VIEW
		}
		return storepb.StatementType_ALTER_TABLE
	default:
		return storepb.StatementType_STATEMENT_TYPE_UNSPECIFIED
	}
}
