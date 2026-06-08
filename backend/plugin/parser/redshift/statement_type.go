package redshift

import (
	"strings"

	redshiftast "github.com/bytebase/omni/redshift/ast"
	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

// StatementTypeWithPosition contains statement type and its position information.
type StatementTypeWithPosition struct {
	Type storepb.StatementType
	// Line is the one-based line number where the statement ends.
	Line int
	Text string
}

// GetStatementTypesWithPosition returns statement types with position information.
// The line numbers are one-based.
func GetStatementTypesWithPosition(asts []base.AST) ([]StatementTypeWithPosition, error) {
	if len(asts) == 0 {
		return []StatementTypeWithPosition{}, nil
	}

	var allResults []StatementTypeWithPosition
	for _, unifiedAST := range asts {
		omniAST, ok := unifiedAST.(*OmniAST)
		if !ok {
			return nil, errors.New("expected Redshift omni AST")
		}
		if omniAST.Node == nil {
			continue
		}

		stmtType := classifyOmniStatementType(omniAST.Node)
		if stmtType == storepb.StatementType_STATEMENT_TYPE_UNSPECIFIED {
			continue
		}

		line := 0
		if omniAST.StartPosition != nil {
			line = int(omniAST.StartPosition.Line)
		}
		line += strings.Count(omniAST.Text, "\n")

		allResults = append(allResults, StatementTypeWithPosition{
			Type: stmtType,
			Line: line,
			Text: omniAST.Text,
		})
	}

	return allResults, nil
}

// GetStatementTypes returns only the statement types as strings.
// This is used for registration with base.RegisterGetStatementTypes.
func GetStatementTypes(asts []base.AST) ([]storepb.StatementType, error) {
	results, err := GetStatementTypesWithPosition(asts)
	if err != nil {
		return nil, err
	}
	types := make([]storepb.StatementType, len(results))
	for i, r := range results {
		types[i] = r.Type
	}
	return types, nil
}

func classifyOmniStatementType(node redshiftast.Node) storepb.StatementType {
	switch n := node.(type) {
	case *redshiftast.CreateStmt:
		return storepb.StatementType_CREATE_TABLE
	case *redshiftast.CreateTableAsStmt:
		if n.Objtype == redshiftast.OBJECT_MATVIEW {
			return storepb.StatementType_CREATE_VIEW
		}
		return storepb.StatementType_CREATE_TABLE
	case *redshiftast.ViewStmt:
		return storepb.StatementType_CREATE_VIEW
	case *redshiftast.IndexStmt:
		return storepb.StatementType_CREATE_INDEX
	case *redshiftast.CreateSeqStmt:
		return storepb.StatementType_CREATE_SEQUENCE
	case *redshiftast.CreateSchemaStmt:
		return storepb.StatementType_CREATE_SCHEMA
	case *redshiftast.CreateFunctionStmt:
		return storepb.StatementType_CREATE_FUNCTION
	case *redshiftast.CreatedbStmt:
		return storepb.StatementType_CREATE_DATABASE
	case *redshiftast.DropdbStmt:
		return storepb.StatementType_DROP_DATABASE
	case *redshiftast.DropStmt:
		return classifyOmniDropStatementType(redshiftast.ObjectType(n.RemoveType))
	case *redshiftast.AlterTableStmt:
		if redshiftast.ObjectType(n.ObjType) == redshiftast.OBJECT_VIEW || redshiftast.ObjectType(n.ObjType) == redshiftast.OBJECT_MATVIEW {
			return storepb.StatementType_ALTER_VIEW
		}
		return storepb.StatementType_ALTER_TABLE
	case *redshiftast.AlterSeqStmt:
		return storepb.StatementType_ALTER_SEQUENCE
	case *redshiftast.RenameStmt:
		return classifyOmniRenameStatementType(n.RenameType)
	case *redshiftast.CommentStmt:
		return storepb.StatementType_COMMENT
	case *redshiftast.TruncateStmt:
		return storepb.StatementType_TRUNCATE
	case *redshiftast.InsertStmt:
		return storepb.StatementType_INSERT
	case *redshiftast.UpdateStmt:
		return storepb.StatementType_UPDATE
	case *redshiftast.DeleteStmt:
		return storepb.StatementType_DELETE
	case *redshiftast.RedshiftObjectStmt:
		return classifyOmniRedshiftObjectStatementType(n.Command, n.ObjectType)
	default:
		return storepb.StatementType_STATEMENT_TYPE_UNSPECIFIED
	}
}

func classifyOmniDropStatementType(objectType redshiftast.ObjectType) storepb.StatementType {
	switch objectType {
	case redshiftast.OBJECT_TABLE:
		return storepb.StatementType_DROP_TABLE
	case redshiftast.OBJECT_VIEW:
		return storepb.StatementType_DROP_VIEW
	case redshiftast.OBJECT_MATVIEW:
		return storepb.StatementType_DROP_TABLE
	case redshiftast.OBJECT_SCHEMA:
		return storepb.StatementType_DROP_SCHEMA
	case redshiftast.OBJECT_INDEX:
		return storepb.StatementType_DROP_INDEX
	case redshiftast.OBJECT_SEQUENCE:
		return storepb.StatementType_DROP_SEQUENCE
	case redshiftast.OBJECT_FUNCTION:
		return storepb.StatementType_DROP_FUNCTION
	default:
		return storepb.StatementType_STATEMENT_TYPE_UNSPECIFIED
	}
}

func classifyOmniRenameStatementType(objectType redshiftast.ObjectType) storepb.StatementType {
	switch objectType {
	case redshiftast.OBJECT_INDEX:
		return storepb.StatementType_RENAME_INDEX
	case redshiftast.OBJECT_SCHEMA:
		return storepb.StatementType_RENAME_SCHEMA
	case redshiftast.OBJECT_SEQUENCE:
		return storepb.StatementType_RENAME_SEQUENCE
	case redshiftast.OBJECT_VIEW, redshiftast.OBJECT_MATVIEW:
		return storepb.StatementType_ALTER_VIEW
	case redshiftast.OBJECT_TABLE:
		return storepb.StatementType_ALTER_TABLE
	default:
		return storepb.StatementType_STATEMENT_TYPE_UNSPECIFIED
	}
}

func classifyOmniRedshiftObjectStatementType(command, objectType string) storepb.StatementType {
	switch strings.ToLower(command) {
	case "create":
		switch strings.ToLower(objectType) {
		case "database":
			return storepb.StatementType_CREATE_DATABASE
		case "schema", "external schema":
			return storepb.StatementType_CREATE_SCHEMA
		default:
			return storepb.StatementType_STATEMENT_TYPE_UNSPECIFIED
		}
	case "alter":
		switch strings.ToLower(objectType) {
		case "database":
			return storepb.StatementType_ALTER_DATABASE
		case "index":
			return storepb.StatementType_ALTER_INDEX
		case "sequence":
			return storepb.StatementType_ALTER_SEQUENCE
		case "table":
			return storepb.StatementType_ALTER_TABLE
		case "view", "external view", "materialized view":
			return storepb.StatementType_ALTER_VIEW
		default:
			return storepb.StatementType_STATEMENT_TYPE_UNSPECIFIED
		}
	case "drop":
		switch strings.ToLower(objectType) {
		case "database":
			return storepb.StatementType_DROP_DATABASE
		case "schema", "external schema":
			return storepb.StatementType_DROP_SCHEMA
		default:
			return storepb.StatementType_STATEMENT_TYPE_UNSPECIFIED
		}
	default:
		return storepb.StatementType_STATEMENT_TYPE_UNSPECIFIED
	}
}
