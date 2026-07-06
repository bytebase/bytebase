package mysql

import (
	"strings"

	"github.com/bytebase/omni/mysql/ast"
	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func GetStatementTypes(asts []base.AST) ([]storepb.StatementType, error) {
	sqlTypeSet := make(map[storepb.StatementType]bool)
	for _, a := range asts {
		node, ok := GetOmniNode(a)
		if !ok {
			return nil, errors.New("expected OmniAST for MySQL")
		}
		t := classifyStatementType(node)
		sqlTypeSet[t] = true
	}
	var sqlTypes []storepb.StatementType
	for sqlType := range sqlTypeSet {
		sqlTypes = append(sqlTypes, sqlType)
	}
	return sqlTypes, nil
}

// StatementTypeWithPosition contains a statement type and its position information.
// It mirrors the PostgreSQL equivalent so the SDL statement-type gating in the release
// service can report disallowed statements with line numbers.
type StatementTypeWithPosition struct {
	Type storepb.StatementType
	// Line is the one-based line number where the statement ends.
	Line int
	Text string
}

// GetStatementTypesWithPositions returns statement types with position information,
// preserving statement order. Line numbers are one-based. It mirrors
// pg.GetStatementTypes so the declarative-release gating can surface disallowed SDL
// statements with positions for MySQL.
func GetStatementTypesWithPositions(asts []base.AST) ([]StatementTypeWithPosition, error) {
	if len(asts) == 0 {
		return []StatementTypeWithPosition{}, nil
	}

	var results []StatementTypeWithPosition
	for _, a := range asts {
		omniAST, ok := a.(*OmniAST)
		if !ok {
			return nil, errors.New("expected OmniAST for MySQL")
		}
		if omniAST.Node == nil {
			continue
		}

		// STATEMENT_TYPE_UNSPECIFIED entries (statements omni parses but the classifier
		// does not know — GRANT, CALL, …) are INCLUDED, with their positions and text, so
		// the SDL statement-type gate fails closed: dropping them here would let an
		// unclassified statement bypass the release-check allowlist entirely. SET is
		// deliberately NOT in that unknown set — it is classified as StatementType_SET so
		// the gate can allow the session-context preamble the MySQL SDL dump emits, while
		// genuinely-unknown statements stay UNSPECIFIED and rejected.
		stmtType := classifyStatementType(omniAST.Node)

		line := 0
		if omniAST.StartPosition != nil {
			line = int(omniAST.StartPosition.Line)
		}
		// End line = start line + embedded newlines in the statement text.
		endLine := line
		if omniAST.Text != "" {
			endLine += strings.Count(omniAST.Text, "\n")
		}

		results = append(results, StatementTypeWithPosition{
			Type: stmtType,
			Line: endLine,
			Text: omniAST.Text,
		})
	}

	return results, nil
}

func classifyStatementType(node ast.Node) storepb.StatementType {
	switch n := node.(type) {
	// CREATE
	case *ast.CreateDatabaseStmt:
		return storepb.StatementType_CREATE_DATABASE
	case *ast.CreateTableStmt:
		return storepb.StatementType_CREATE_TABLE
	case *ast.CreateIndexStmt:
		return storepb.StatementType_CREATE_INDEX
	case *ast.CreateViewStmt:
		return storepb.StatementType_CREATE_VIEW
	case *ast.CreateEventStmt:
		return storepb.StatementType_CREATE_EVENT
	case *ast.CreateTriggerStmt:
		return storepb.StatementType_CREATE_TRIGGER
	case *ast.CreateFunctionStmt:
		if n.IsProcedure {
			return storepb.StatementType_CREATE_PROCEDURE
		}
		return storepb.StatementType_CREATE_FUNCTION

	// DROP
	case *ast.DropDatabaseStmt:
		return storepb.StatementType_DROP_DATABASE
	case *ast.DropTableStmt:
		return storepb.StatementType_DROP_TABLE
	case *ast.DropIndexStmt:
		return storepb.StatementType_DROP_INDEX
	case *ast.DropViewStmt:
		return storepb.StatementType_DROP_VIEW
	case *ast.DropEventStmt:
		return storepb.StatementType_DROP_EVENT
	case *ast.DropTriggerStmt:
		return storepb.StatementType_DROP_TRIGGER
	case *ast.DropRoutineStmt:
		if n.IsProcedure {
			return storepb.StatementType_DROP_PROCEDURE
		}
		return storepb.StatementType_DROP_FUNCTION

	// ALTER
	case *ast.AlterTableStmt:
		return storepb.StatementType_ALTER_TABLE
	case *ast.AlterDatabaseStmt:
		return storepb.StatementType_ALTER_DATABASE
	case *ast.AlterViewStmt:
		return storepb.StatementType_ALTER_VIEW
	case *ast.AlterEventStmt:
		return storepb.StatementType_ALTER_EVENT

	// OTHER DDL
	case *ast.TruncateStmt:
		return storepb.StatementType_TRUNCATE
	case *ast.RenameTableStmt:
		return storepb.StatementType_RENAME

	// DML
	case *ast.InsertStmt:
		return storepb.StatementType_INSERT
	case *ast.UpdateStmt:
		return storepb.StatementType_UPDATE
	case *ast.DeleteStmt:
		return storepb.StatementType_DELETE

	// SESSION / UTILITY
	// SET is classified explicitly (rather than left UNSPECIFIED) because the MySQL SDL
	// dump brackets each routine/event/trigger with a `SET @saved_… ; SET sql_mode=… ; …;
	// SET … = @saved_…` session-context preamble, and the declarative-release gate must be
	// able to allow those SET statements by type (see extraAllowedSDLStatementTypesByEngine
	// in release_service_check.go). Every SET form omni parses — user vars, system vars,
	// SET NAMES, SET CHARACTER SET — is a single *ast.SetStmt.
	case *ast.SetStmt:
		return storepb.StatementType_SET

	default:
		return storepb.StatementType_STATEMENT_TYPE_UNSPECIFIED
	}
}
