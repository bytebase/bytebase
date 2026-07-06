package mysql

// Offline unit coverage for the BYT-9832 SDL session-context framing (the live oracle proofs
// live in sdl_session_context_live_test.go). These assert the exact bytes the writers emit and
// prove — without a server — that the emitted SET is cosmetic to the omni declarative diff.

import (
	"strings"
	"testing"

	"github.com/bytebase/omni/mysql/catalog"
	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

// sessionContextMeta builds a schema with a function, procedure, trigger, and event each
// carrying an explicit sql_mode (and, for the event, a time_zone). The bodies are mode-neutral
// so the emitted SDL also parses in omni (ANSI_QUOTES bodies are a separate, omni-unsupported
// case covered by the live suite).
func sessionContextMeta() *storepb.DatabaseSchemaMetadata {
	return &storepb.DatabaseSchemaMetadata{
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
}

// TestSDLSessionContextWriterOutput asserts the concat-safe save/restore bracketing is emitted
// for every context-carrying object and that the whole dump round-trips through omni with an
// empty self-diff (the SET is cosmetic to the routine/trigger/event diff).
func TestSDLSessionContextWriterOutput(t *testing.T) {
	sdl, err := getSDLFormat(sessionContextMeta())
	require.NoError(t, err)
	t.Logf("emitted SDL:\n%s", sdl)

	for _, want := range []string{
		"SET @saved_sql_mode = @@sql_mode;",
		"SET sql_mode = 'NO_BACKSLASH_ESCAPES';",
		"SET sql_mode = 'PAD_CHAR_TO_FULL_LENGTH';",
		"SET sql_mode = @saved_sql_mode;",
		"SET @saved_time_zone = @@time_zone;",
		"SET time_zone = '+05:30';",
		"SET time_zone = @saved_time_zone;",
	} {
		require.Contains(t, sdl, want, "emitted SDL missing %q", want)
	}

	// The event brackets BOTH time_zone and sql_mode. Nesting is LIFO: save time_zone (outer)
	// then sql_mode (inner); restore sql_mode (inner) then time_zone (outer). Scope to the
	// event's block (from its time_zone save to its time_zone restore) and verify the four
	// framing lines appear in that exact order.
	tzSave := strings.Index(sdl, "SET @saved_time_zone = @@time_zone;")
	tzRestore := strings.Index(sdl, "SET time_zone = @saved_time_zone;")
	require.Positive(t, tzSave, "event must save time_zone")
	require.Greater(t, tzRestore, tzSave, "event must restore time_zone after saving it")
	block := sdl[tzSave : tzRestore+len("SET time_zone = @saved_time_zone;")]
	modeSave := strings.Index(block, "SET @saved_sql_mode = @@sql_mode;")
	modeRestore := strings.Index(block, "SET sql_mode = @saved_sql_mode;")
	require.Positive(t, modeSave, "event block must save sql_mode after time_zone (inner bracket)")
	require.Positive(t, modeRestore, "event block must restore sql_mode")
	// Within the block: time_zone-save (0) < sql_mode-save < sql_mode-restore < time_zone-restore (end).
	require.Less(t, modeRestore, strings.Index(block, "SET time_zone = @saved_time_zone;"),
		"event must restore sql_mode (inner) before time_zone (outer) — LIFO")

	_, err = catalog.LoadSDLWithVersion(withDatabaseContext(sdl), catalog.MySQL80)
	require.NoError(t, err, "emitted SDL must LoadSDL cleanly")

	selfDiff, err := mysqlDiffSDLMigration(sdl, sdl, "8.0")
	require.NoError(t, err)
	require.Empty(t, selfDiff, "self-diff must be empty, got:\n%s", selfDiff)
}

// TestSDLSessionContextDefaultModeNoBrackets confirms an object with an empty SqlMode emits NO
// save/restore bracket (bare CREATE), so default-mode dumps are unchanged.
func TestSDLSessionContextDefaultModeNoBrackets(t *testing.T) {
	meta := &storepb.DatabaseSchemaMetadata{
		Name: "probe2",
		Schemas: []*storepb.SchemaMetadata{
			{
				Functions: []*storepb.FunctionMetadata{
					{Name: "f0", Definition: `CREATE FUNCTION f0() RETURNS INT DETERMINISTIC RETURN 1`, SqlMode: ""},
				},
			},
		},
	}
	sdl, err := getSDLFormat(meta)
	require.NoError(t, err)
	require.NotContains(t, sdl, "SET sql_mode", "empty-sql_mode routine must not emit any SET")
	require.NotContains(t, sdl, "@saved_sql_mode", "empty-sql_mode routine must not save/restore")
}

// TestSDLSessionContextCosmeticToDiff proves the emitted SET framing does not perturb the omni
// declarative diff: a mode-bracketed routine diffs empty against the bare form of the same
// routine, because omni compares routine bodies opaquely and does not fold session sql_mode
// into routine identity. This is the property that makes the fix bytebase-only (no omni change).
func TestSDLSessionContextCosmeticToDiff(t *testing.T) {
	bracketed := "SET @saved_sql_mode = @@sql_mode;\nSET sql_mode = 'NO_BACKSLASH_ESCAPES';\n" +
		"CREATE FUNCTION f() RETURNS INT DETERMINISTIC RETURN 1;\nSET sql_mode = @saved_sql_mode;\n"
	bare := "CREATE FUNCTION f() RETURNS INT DETERMINISTIC RETURN 1;\n"

	diff, err := mysqlDiffSDLMigration(bracketed, bare, "8.0")
	require.NoError(t, err)
	require.Empty(t, diff, "bracketed vs bare must diff empty (SET is cosmetic), got:\n%s", diff)
}
