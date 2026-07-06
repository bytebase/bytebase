package v1

import (
	"testing"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/plugin/schema"

	// Blank import registers the MySQL ParseStatements func used by base.ParseStatements.
	_ "github.com/bytebase/bytebase/backend/plugin/parser/mysql"
	// Blank import registers the MySQL SDL dumper used by schema.GetDatabaseDefinition.
	_ "github.com/bytebase/bytebase/backend/plugin/schema/mysql"
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

	// SET: MySQL-only overlay (BYT-9832). The MySQL SDL dump brackets every
	// routine/event/trigger carrying a non-empty sql_mode/time_zone with a session-context
	// preamble of SET statements, so SET must pass the MySQL gate. It stays MySQL-scoped:
	// PostgreSQL's own SET dump is dropped as UNSPECIFIED by pg's extractor and never
	// reaches the allow-list, so PG must NOT admit SET here.
	require.True(t, isAllowedInSDL(storepb.Engine_MYSQL, storepb.StatementType_SET), "SET should be allowed in MySQL SDL")
	require.False(t, isAllowedInSDL(storepb.Engine_POSTGRES, storepb.StatementType_SET), "SET should be disallowed in PostgreSQL SDL")

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
// parses but the MySQL classifier does not know (GRANT, CALL, ...) surfaces as
// STATEMENT_TYPE_UNSPECIFIED and MUST reach the release gate (with its line) and be
// rejected — dropping it would let arbitrary unclassified statements bypass the SDL
// allowlist entirely. SET is deliberately NOT used here: it is now a classified type
// (StatementType_SET, allowed for MySQL SDL), so it would not exercise fail-closed.
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

// TestMySQLSetClassifiedAndAllowedInSDL proves the session-context preamble the BYT-9832
// SDL dump emits is now accepted by the gate: a `SET sql_mode=…` classifies as
// StatementType_SET (not UNSPECIFIED) and is allowed in MySQL SDL, while CALL — a
// genuinely-unknown statement — stays UNSPECIFIED and rejected (the fail-closed posture is
// preserved for everything except the deliberately-allowed SET).
func TestMySQLSetClassifiedAndAllowedInSDL(t *testing.T) {
	// The SDL gate narrows SET to the dumper's session-context framing only (BYT-9832 P2):
	// session-scope sql_mode/time_zone and user-variable saves pass; every other SET (GLOBAL,
	// PERSIST, FOREIGN_KEY_CHECKS, SET NAMES, other session vars, …) is downgraded to
	// UNSPECIFIED and rejected — the same fail-closed treatment as a truly-unknown statement.
	allowed := []string{
		"SET @saved_sql_mode = @@sql_mode;\n",
		"SET sql_mode='ANSI_QUOTES';\n",
		"SET sql_mode = @saved_sql_mode;\n",
		"SET @saved_time_zone = @@time_zone;\n",
		"SET time_zone='+05:30';\n",
		"SET time_zone = @saved_time_zone;\n",
	}
	for _, sql := range allowed {
		stmts, err := base.ParseStatements(storepb.Engine_MYSQL, sql)
		require.NoError(t, err, sql)
		got, err := getStatementTypesWithPositionsForEngine(storepb.Engine_MYSQL, base.ExtractASTs(stmts))
		require.NoError(t, err)
		require.Len(t, got, 1, sql)
		require.Equal(t, storepb.StatementType_SET, got[0].Type, "framing SET must classify as SET: %q", sql)
		require.True(t, isAllowedInSDL(storepb.Engine_MYSQL, got[0].Type), "framing SET must be allowed in MySQL SDL: %q", sql)
	}

	// Non-declarative SET and a genuinely-unknown statement (CALL) must both fail closed at
	// the gate: downgraded/left as UNSPECIFIED and rejected by isAllowedInSDL.
	rejected := []string{
		"SET GLOBAL max_connections = 1;\n",
		"SET PERSIST sql_mode = 'ANSI_QUOTES';\n",
		"SET FOREIGN_KEY_CHECKS = 0;\n",
		"SET SESSION unique_checks = 0;\n",
		"SET NAMES utf8mb4;\n",
		"SET sql_mode = 'ANSI_QUOTES', FOREIGN_KEY_CHECKS = 0;\n",
		"CALL p();\n",
	}
	for _, sql := range rejected {
		stmts, err := base.ParseStatements(storepb.Engine_MYSQL, sql)
		require.NoError(t, err, sql)
		got, err := getStatementTypesWithPositionsForEngine(storepb.Engine_MYSQL, base.ExtractASTs(stmts))
		require.NoError(t, err)
		require.Len(t, got, 1, sql)
		require.Equal(t, storepb.StatementType_STATEMENT_TYPE_UNSPECIFIED, got[0].Type, "non-framing statement must fail closed: %q", sql)
		require.False(t, isAllowedInSDL(storepb.Engine_MYSQL, got[0].Type), "non-framing statement must be rejected by the gate: %q", sql)
	}
}

// TestMySQLSDLDumpWithRoutinesPassesGate is the direct BYT-9832 P1 regression guard: it
// runs the REAL production SDL dumper (schema.GetDatabaseDefinition with SDLFormat) over a
// schema containing a function, procedure, trigger (each with a non-empty sql_mode) and an
// event (non-UTC time_zone), then feeds the dump through the REAL release-check gate
// (base.ParseStatements -> getStatementTypesWithPositionsForEngine -> isAllowedInSDL) and
// asserts EVERY statement is allowed. Before this fix the session-context SET preamble was
// rejected as "Disallowed statement in SDL file", breaking the export->commit->check
// round-trip for any MySQL DB with a routine/event/trigger.
func TestMySQLSDLDumpWithRoutinesPassesGate(t *testing.T) {
	meta := &storepb.DatabaseSchemaMetadata{
		Name: "probe",
		Schemas: []*storepb.SchemaMetadata{
			{
				Tables: []*storepb.TableMetadata{
					{
						Name: "t",
						Columns: []*storepb.ColumnMetadata{
							{Name: "id", Type: "int", Nullable: false},
							{Name: "val", Type: "int", Nullable: false, Default: "0"},
						},
						Indexes: []*storepb.IndexMetadata{{Name: "PRIMARY", Primary: true, Expressions: []string{"id"}}},
						Engine:  "InnoDB",
						Charset: "utf8mb4",
						Triggers: []*storepb.TriggerMetadata{
							{Name: "trg", Timing: "BEFORE", Event: "INSERT", Body: "SET NEW.val = NEW.val + 1", SqlMode: "NO_BACKSLASH_ESCAPES"},
						},
					},
				},
				Functions: []*storepb.FunctionMetadata{
					{Name: "fa", Definition: `CREATE FUNCTION fa() RETURNS INT DETERMINISTIC RETURN 1`, SqlMode: "NO_BACKSLASH_ESCAPES"},
				},
				Procedures: []*storepb.ProcedureMetadata{
					{Name: "pb", Definition: `CREATE PROCEDURE pb() BEGIN SELECT 1; END`, SqlMode: "PAD_CHAR_TO_FULL_LENGTH"},
				},
				Events: []*storepb.EventMetadata{
					{Name: "ev", Definition: `CREATE EVENT ev ON SCHEDULE EVERY 1 DAY DO INSERT INTO t (id, val) VALUES (1, 1)`, SqlMode: "NO_BACKSLASH_ESCAPES", TimeZone: "+05:30"},
				},
			},
		},
	}

	sdl, err := schema.GetDatabaseDefinition(storepb.Engine_MYSQL, schema.GetDefinitionContext{SDLFormat: true}, meta)
	require.NoError(t, err)
	// Guard the fixture actually exercised the preamble; otherwise the gate assertion is vacuous.
	require.Contains(t, sdl, "SET sql_mode =", "dump must contain the sql_mode session preamble")
	require.Contains(t, sdl, "SET time_zone =", "dump must contain the event time_zone preamble")

	stmts, err := base.ParseStatements(storepb.Engine_MYSQL, sdl)
	require.NoError(t, err, "the emitted SDL must parse for the gate to run")
	got, err := getStatementTypesWithPositionsForEngine(storepb.Engine_MYSQL, base.ExtractASTs(stmts))
	require.NoError(t, err)
	require.NotEmpty(t, got)

	sawSet := false
	for _, stmt := range got {
		require.True(t, isAllowedInSDL(storepb.Engine_MYSQL, stmt.Type),
			"statement type %s at line %d must be allowed in MySQL SDL (BYT-9832 regression):\n%s",
			stmt.Type, stmt.Line, stmt.Text)
		if stmt.Type == storepb.StatementType_SET {
			sawSet = true
		}
	}
	require.True(t, sawSet, "the dump must have produced at least one SET the gate had to allow")
}
