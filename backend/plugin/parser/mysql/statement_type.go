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
		// unclassified statement bypass the release-check allowlist entirely.
		stmtType := classifyStatementType(omniAST.Node)

		// SDL-gate narrowing (this function feeds ONLY the declarative-release gate; the
		// general GetStatementTypes/classifyStatementType path — ContainsDDL, plancheck —
		// is untouched and still maps every SET to StatementType_SET). Allowing SET by type
		// would admit ANY user-authored SET (SET GLOBAL …, SET PERSIST …, SET
		// FOREIGN_KEY_CHECKS=0, SET NAMES …), which is not declarative SDL. Only the
		// session-context framing the MySQL SDL dumper emits around routines/events/triggers
		// is legitimate here, so downgrade every other SET back to UNSPECIFIED and let the
		// gate reject it fail-closed.
		if stmtType == storepb.StatementType_SET {
			if set, ok := omniAST.Node.(*ast.SetStmt); !ok || !isSDLSessionContextSet(set) {
				stmtType = storepb.StatementType_STATEMENT_TYPE_UNSPECIFIED
			}
		}

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

// isSDLSessionContextSet reports whether a SET statement is one of the session-context
// framing forms the MySQL SDL dumper emits around a routine/event/trigger
// (writeSDLSessionContextPrefix in backend/plugin/schema/mysql/get_database_definition.go):
//
//	SET @saved_sql_mode = @@sql_mode;      SET @saved_time_zone = @@time_zone;
//	SET sql_mode = 'ANSI_QUOTES';          SET time_zone = '+05:30';
//	SET sql_mode = @saved_sql_mode;        SET time_zone = @saved_time_zone;
//
// It exists solely to keep the declarative-release SDL gate fail-closed: a SET is allowed
// only when EVERY assignment is either a user variable (`@name`, the save/restore temp) or
// a plain SESSION-scoped `sql_mode`/`time_zone`. Anything else — GLOBAL/PERSIST/PERSIST_ONLY/
// LOCAL scope, any other system variable (foreign_key_checks, unique_checks, autocommit, …),
// an explicit `@@`-qualified LHS, SET NAMES / SET CHARACTER SET — is rejected. The omni parser
// encodes the LHS kind in Assignment.Column.Column via prefix: `@name` (user var), `@@name` /
// `@@SCOPE.name` (system var), a bare name (session/`Scope`-scoped var), or the literals
// "NAMES" / "CHARACTER SET".
func isSDLSessionContextSet(set *ast.SetStmt) bool {
	// Statement-level scope: only the default (session) form is emitted. GLOBAL / PERSIST /
	// PERSIST_ONLY / LOCAL are never part of the framing and must fail closed.
	if scope := strings.ToUpper(set.Scope); scope != "" && scope != "SESSION" {
		return false
	}
	if len(set.Assignments) == 0 {
		return false
	}
	for _, a := range set.Assignments {
		if a == nil || a.Column == nil {
			return false
		}
		// A qualified LHS (SET NEW.col / OLD.col) is never a top-level framing SET.
		if a.Column.Table != "" {
			return false
		}
		name := a.Column.Column
		switch {
		case strings.HasPrefix(name, "@@"):
			// Explicit system-variable LHS (@@x, @@GLOBAL.x) — the dumper never writes one.
			return false
		case strings.HasPrefix(name, "@"):
			// User variable (@saved_sql_mode, @saved_time_zone): the save/restore temp. OK.
			continue
		default:
			// Bare (session-scoped) variable: only sql_mode / time_zone qualify. This also
			// rejects the SET NAMES / SET CHARACTER SET forms (Column "NAMES" / "CHARACTER SET").
			switch strings.ToLower(name) {
			case "sql_mode", "time_zone":
				continue
			default:
				return false
			}
		}
	}
	return true
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
	// A SET is a SET: classified uniformly here (every SET form omni parses — user vars,
	// system vars, SET NAMES, SET CHARACTER SET — is a single *ast.SetStmt), which keeps
	// ContainsDDL and the plancheck report correct. The declarative-release SDL gate needs
	// only the narrow session-context framing the MySQL SDL dump emits, so it narrows this
	// type further on its own path (isSDLSessionContextSet in GetStatementTypesWithPositions
	// downgrades every other SET back to UNSPECIFIED). classifyStatementType itself stays
	// content-agnostic so non-gate consumers are unaffected.
	case *ast.SetStmt:
		return storepb.StatementType_SET

	default:
		return storepb.StatementType_STATEMENT_TYPE_UNSPECIFIED
	}
}
