package mysql

import (
	"github.com/bytebase/omni/mysql/analysis"
	"strings"

	"github.com/bytebase/omni/mysql/ast"
	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func GetStatementTypes(asts []base.AST) ([]storepb.StatementType, error) {
	sqlTypeSet := make(map[storepb.StatementType]bool)
	for _, a := range asts {
		node, ok := GetNode(a)
		if !ok {
			return nil, errors.New("expected AST for MySQL")
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
		node, ok := a.(*AST)
		if !ok {
			return nil, errors.New("expected AST for MySQL")
		}
		if node.Node == nil {
			continue
		}

		// STATEMENT_TYPE_UNSPECIFIED entries (statements omni parses but the classifier
		// does not know — GRANT, SET, PREPARE, …) are INCLUDED, with their positions and
		// text, so the SDL statement-type gate fails closed: dropping them here would
		// let an unclassified statement bypass the release-check allowlist entirely.
		stmtType := classifyStatementType(node.Node)

		line := 0
		if node.StartPosition != nil {
			line = int(node.StartPosition.Line)
		}
		// End line = start line + embedded newlines in the statement text.
		endLine := line
		if node.Text != "" {
			endLine += strings.Count(node.Text, "\n")
		}

		results = append(results, StatementTypeWithPosition{
			Type: stmtType,
			Line: endLine,
			Text: node.Text,
		})
	}

	return results, nil
}

func classifyStatementType(node ast.Node) storepb.StatementType {
	switch analysis.StatementTypeOf(node) {
	case analysis.CreateDatabase:
		return storepb.StatementType_CREATE_DATABASE
	case analysis.CreateTable:
		return storepb.StatementType_CREATE_TABLE
	case analysis.CreateView:
		return storepb.StatementType_CREATE_VIEW
	case analysis.CreateIndex:
		return storepb.StatementType_CREATE_INDEX
	case analysis.CreateEvent:
		return storepb.StatementType_CREATE_EVENT
	case analysis.CreateTrigger:
		return storepb.StatementType_CREATE_TRIGGER
	case analysis.CreateFunction:
		return storepb.StatementType_CREATE_FUNCTION
	case analysis.CreateProcedure:
		return storepb.StatementType_CREATE_PROCEDURE
	case analysis.DropDatabase:
		return storepb.StatementType_DROP_DATABASE
	case analysis.DropTable:
		return storepb.StatementType_DROP_TABLE
	case analysis.DropView:
		return storepb.StatementType_DROP_VIEW
	case analysis.DropIndex:
		return storepb.StatementType_DROP_INDEX
	case analysis.DropEvent:
		return storepb.StatementType_DROP_EVENT
	case analysis.DropTrigger:
		return storepb.StatementType_DROP_TRIGGER
	case analysis.DropFunction:
		return storepb.StatementType_DROP_FUNCTION
	case analysis.DropProcedure:
		return storepb.StatementType_DROP_PROCEDURE
	case analysis.AlterDatabase:
		return storepb.StatementType_ALTER_DATABASE
	case analysis.AlterTable:
		return storepb.StatementType_ALTER_TABLE
	case analysis.AlterView:
		return storepb.StatementType_ALTER_VIEW
	case analysis.AlterEvent:
		return storepb.StatementType_ALTER_EVENT
	case analysis.Truncate:
		return storepb.StatementType_TRUNCATE
	case analysis.Rename:
		return storepb.StatementType_RENAME
	case analysis.Insert:
		return storepb.StatementType_INSERT
	case analysis.Update:
		return storepb.StatementType_UPDATE
	case analysis.Delete:
		return storepb.StatementType_DELETE
	case analysis.Execute:
		return storepb.StatementType_EXECUTE
	default:
		return storepb.StatementType_STATEMENT_TYPE_UNSPECIFIED
	}
}
