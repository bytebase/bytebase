package v1

import (
	"testing"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"

	// Blank import registers the MySQL ParseStatements func used by base.ParseStatements.
	_ "github.com/bytebase/bytebase/backend/plugin/parser/mysql"
)

// TestEngineSupportsDeclarativeRelease asserts MySQL is admitted into the declarative
// release gating alongside PostgreSQL, and that engines without SDL support are excluded.
func TestEngineSupportsDeclarativeRelease(t *testing.T) {
	require.True(t, engineSupportsDeclarativeRelease(storepb.Engine_POSTGRES), "postgres must be admitted")
	require.True(t, engineSupportsDeclarativeRelease(storepb.Engine_MYSQL), "mysql must be admitted")
	require.False(t, engineSupportsDeclarativeRelease(storepb.Engine_ORACLE), "oracle must not be admitted")
	require.False(t, engineSupportsDeclarativeRelease(storepb.Engine_ENGINE_UNSPECIFIED), "unspecified must not be admitted")
}

// TestGetStatementTypesWithPositionsForEngineMySQL proves the MySQL case of the gating
// statement-type extractor classifies SDL statements and carries position info.
func TestGetStatementTypesWithPositionsForEngineMySQL(t *testing.T) {
	sql := "CREATE TABLE t (id INT PRIMARY KEY);\nDROP TABLE old;\n"
	stmts, err := base.ParseStatements(storepb.Engine_MYSQL, sql)
	require.NoError(t, err)
	asts := base.ExtractASTs(stmts)

	got, err := getStatementTypesWithPositionsForEngine(storepb.Engine_MYSQL, asts)
	require.NoError(t, err)
	require.Len(t, got, 2)

	require.Equal(t, storepb.StatementType_CREATE_TABLE, got[0].Type)
	require.Equal(t, storepb.StatementType_DROP_TABLE, got[1].Type)
	// Positions are one-based; the DROP on line 2 must report a positive line number.
	require.Positive(t, got[1].Line, "expected a positive line number for the second statement")
}

// TestIsAllowedInSDL asserts the engine-keyed SDL allow-list: the common set admits
// the CREATE statement types both declarative dumps emit (CREATE TRIGGER is shared
// deliberately — the PostgreSQL SDL dump exports triggers and the pg omni differ
// handles OpDropTrigger, so a declared trigger is legal PG SDL too); CREATE EVENT is a
// MySQL-only overlay (no PostgreSQL analog); mutating statements and unclassified
// (UNSPECIFIED) statements are rejected for every engine — fail closed.
func TestIsAllowedInSDL(t *testing.T) {
	commonAllowed := []storepb.StatementType{
		storepb.StatementType_CREATE_TABLE,
		storepb.StatementType_CREATE_VIEW,
		storepb.StatementType_CREATE_INDEX,
		storepb.StatementType_CREATE_FUNCTION,
		storepb.StatementType_CREATE_PROCEDURE,
		storepb.StatementType_CREATE_TRIGGER,
	}
	for _, engine := range []storepb.Engine{storepb.Engine_MYSQL, storepb.Engine_POSTGRES} {
		for _, st := range commonAllowed {
			require.True(t, isAllowedInSDL(engine, st), "%s should be allowed in %s SDL", st, engine)
		}
	}

	// CREATE EVENT: MySQL-only overlay. The pg parser can never classify one, but the
	// per-engine keying documents intent and keeps the PG gate exactly its own set.
	require.True(t, isAllowedInSDL(storepb.Engine_MYSQL, storepb.StatementType_CREATE_EVENT), "CREATE EVENT should be allowed in MySQL SDL")
	require.False(t, isAllowedInSDL(storepb.Engine_POSTGRES, storepb.StatementType_CREATE_EVENT), "CREATE EVENT should be disallowed in PostgreSQL SDL")

	// PostgreSQL keeps its sequence/schema statement types; they are common (harmless
	// for MySQL, whose parser never emits them).
	require.True(t, isAllowedInSDL(storepb.Engine_POSTGRES, storepb.StatementType_CREATE_SEQUENCE))
	require.True(t, isAllowedInSDL(storepb.Engine_POSTGRES, storepb.StatementType_CREATE_SCHEMA))
	require.True(t, isAllowedInSDL(storepb.Engine_POSTGRES, storepb.StatementType_ALTER_SEQUENCE))
	require.True(t, isAllowedInSDL(storepb.Engine_POSTGRES, storepb.StatementType_COMMENT))

	disallowed := []storepb.StatementType{
		storepb.StatementType_ALTER_TABLE,
		storepb.StatementType_DROP_TABLE,
		storepb.StatementType_INSERT,
		storepb.StatementType_UPDATE,
		storepb.StatementType_DELETE,
		storepb.StatementType_CREATE_DATABASE,
		// Unclassified statements must fail closed on every engine.
		storepb.StatementType_STATEMENT_TYPE_UNSPECIFIED,
	}
	for _, engine := range []storepb.Engine{storepb.Engine_MYSQL, storepb.Engine_POSTGRES} {
		for _, st := range disallowed {
			require.False(t, isAllowedInSDL(engine, st), "%s should be disallowed in %s SDL", st, engine)
		}
	}
}

// TestMySQLSDLGateFailsClosedOnUnclassifiedStatement pins the X12 fix: a statement omni
// parses but the MySQL classifier does not know (GRANT, SET, ...) surfaces as
// STATEMENT_TYPE_UNSPECIFIED and MUST reach the release gate (with its line) and be
// rejected — dropping it would let arbitrary unclassified statements bypass the SDL
// allowlist entirely.
func TestMySQLSDLGateFailsClosedOnUnclassifiedStatement(t *testing.T) {
	sql := "CREATE TABLE t (id INT PRIMARY KEY);\nGRANT SELECT ON *.* TO 'x'@'%';\n"
	stmts, err := base.ParseStatements(storepb.Engine_MYSQL, sql)
	require.NoError(t, err, "omni must parse the GRANT for this gate to be reachable")
	asts := base.ExtractASTs(stmts)

	got, err := getStatementTypesWithPositionsForEngine(storepb.Engine_MYSQL, asts)
	require.NoError(t, err)
	require.Len(t, got, 2, "the unclassified statement must NOT be dropped from the gate input")

	require.Equal(t, storepb.StatementType_CREATE_TABLE, got[0].Type)
	require.True(t, isAllowedInSDL(storepb.Engine_MYSQL, got[0].Type))

	require.Equal(t, storepb.StatementType_STATEMENT_TYPE_UNSPECIFIED, got[1].Type)
	require.False(t, isAllowedInSDL(storepb.Engine_MYSQL, got[1].Type), "UNSPECIFIED must fail closed")
	require.Equal(t, 2, got[1].Line, "the disallowed advice must carry the GRANT's line")
	require.Contains(t, got[1].Text, "GRANT")
}
