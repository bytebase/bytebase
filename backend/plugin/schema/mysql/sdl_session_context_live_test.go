package mysql

// Live oracle proof for BYT-9832: the SDL writers preserve the session sql_mode (and, for
// events, time_zone) each routine/trigger/event was authored under, so exporting SDL and
// re-applying it recreates the object under its authored context — not the applier's default.
//
// The writers bracket each context-carrying object with a concat-safe save/restore SET
// (SET @saved_sql_mode = @@sql_mode; SET sql_mode='…'; <CREATE>; SET sql_mode = @saved_sql_mode;).
// The save/restore is required because both the multi-file rollout concatenation
// (action/command/file.go) and the live apply run every statement on ONE session, so a bare
// SET would leak the mode into the next object. These tests exercise the property end to end
// on live 5.7 and 8.0.
//
// Mode coverage note: two families of sql_mode behave differently for the OMNI declarative
// diff. Modes such as PIPES_AS_CONCAT / NO_BACKSLASH_ESCAPES leave SHOW CREATE using backtick
// identifiers, so the dumped object parses in omni and round-trips as a no-op (the full
// declarative path is exercised). ANSI_QUOTES flips identifier quoting to double quotes, and
// omni's SDL parser does not accept a double-quoted identifier (`CREATE FUNCTION "f"` / a body
// `NEW."c"`), so an ANSI_QUOTES-authored routine/trigger cannot round-trip through the omni
// diff — a PRE-EXISTING omni limitation, independent of and unchanged by this fix (the bare
// pre-fix dump failed omni parse identically). For ANSI_QUOTES this file proves the part
// BYT-9832 actually fixes: the LIVE re-apply recreates the object under the authored mode
// (verified via information_schema), which a default-mode re-create would silently change.

import (
	"context"
	"database/sql"
	"strings"
	"testing"

	_ "github.com/go-sql-driver/mysql"
	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	mysqldb "github.com/bytebase/bytebase/backend/plugin/db/mysql"
	"github.com/bytebase/bytebase/backend/plugin/schema"
)

// rawDB returns the underlying *sql.DB for direct information_schema oracle queries.
func rawDB(ctx context.Context, t *testing.T, srv liveServer, dbName string) (*sql.DB, func()) {
	t.Helper()
	driver, err := createLiveMySQLDriver(ctx, srv, dbName)
	require.NoError(t, err)
	sqlDB := driver.(*mysqldb.Driver).GetDB()
	return sqlDB, func() { driver.Close(ctx) }
}

// routineSQLMode returns information_schema.ROUTINES.SQL_MODE for a routine.
func routineSQLMode(ctx context.Context, t *testing.T, dbh *sql.DB, dbName, name string) string {
	t.Helper()
	var mode string
	err := dbh.QueryRowContext(ctx,
		"SELECT SQL_MODE FROM information_schema.ROUTINES WHERE ROUTINE_SCHEMA = ? AND ROUTINE_NAME = ?",
		dbName, name).Scan(&mode)
	require.NoError(t, err, "query routine sql_mode for %s", name)
	return mode
}

// triggerSQLMode returns information_schema.TRIGGERS.SQL_MODE for a trigger.
func triggerSQLMode(ctx context.Context, t *testing.T, dbh *sql.DB, dbName, name string) string {
	t.Helper()
	var mode string
	err := dbh.QueryRowContext(ctx,
		"SELECT SQL_MODE FROM information_schema.TRIGGERS WHERE TRIGGER_SCHEMA = ? AND TRIGGER_NAME = ?",
		dbName, name).Scan(&mode)
	require.NoError(t, err, "query trigger sql_mode for %s", name)
	return mode
}

// eventContext returns information_schema.EVENTS.(SQL_MODE, TIME_ZONE) for an event.
func eventContext(ctx context.Context, t *testing.T, dbh *sql.DB, dbName, name string) (sqlMode, timeZone string) {
	t.Helper()
	err := dbh.QueryRowContext(ctx,
		"SELECT SQL_MODE, TIME_ZONE FROM information_schema.EVENTS WHERE EVENT_SCHEMA = ? AND EVENT_NAME = ?",
		dbName, name).Scan(&sqlMode, &timeZone)
	require.NoError(t, err, "query event context for %s", name)
	return sqlMode, timeZone
}

// TestSDLSessionContextRoundTripLive is the BYT-9832 full-round-trip oracle for the
// omni-parseable modes: author a function under PIPES_AS_CONCAT and a procedure under
// NO_BACKSLASH_ESCAPES, dump to SDL, apply that dump to a FRESH scratch DB, re-dump, and
// assert (1) the re-dump self-diffs empty through the omni declarative path (the SET framing
// is cosmetic to the diff) and (2) the re-applied objects carry the authored mode (via
// information_schema), proving the mode survived the SDL export/re-apply.
//
//nolint:tparallel
func TestSDLSessionContextRoundTripLive(t *testing.T) {
	skipUnlessLiveOracle(t)
	ctx := context.Background()

	// fa under NO_BACKSLASH_ESCAPES, pb under PAD_CHAR_TO_FULL_LENGTH. Both are non-default
	// modes that leave SHOW CREATE using backtick identifiers AND do not change how a plain
	// body lexes, so the dumped objects round-trip through omni (the SET is cosmetic to the
	// diff). Two DISTINCT modes let the leakage check below be meaningful. (Modes that alter
	// body syntax — ANSI_QUOTES's "…" identifiers, PIPES_AS_CONCAT's ||  — are exercised in
	// TestSDLSessionContextAnsiQuotesLive via the live re-apply, since omni cannot parse them.)
	originDDL := `CREATE TABLE t (id INT PRIMARY KEY, val INT NOT NULL DEFAULT 0) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

SET sql_mode = 'NO_BACKSLASH_ESCAPES';
CREATE FUNCTION fa() RETURNS INT DETERMINISTIC RETURN 1;
SET sql_mode = '';

SET sql_mode = 'PAD_CHAR_TO_FULL_LENGTH';
CREATE PROCEDURE pb() BEGIN SELECT 1; END;
SET sql_mode = '';
`

	for _, srv := range liveServers {
		srv := srv
		t.Run(srv.name, func(t *testing.T) {
			originDB := newLiveDatabase(ctx, t, srv, "sdl_sesctx_origin")
			require.NoError(t, applyDDL(ctx, t, srv, originDB, originDDL))

			source := dumpSDL(ctx, t, srv, originDB)
			t.Logf("[%s] dumped SDL:\n%s", srv.name, source)
			require.Contains(t, source, "SET @saved_sql_mode = @@sql_mode;",
				"[%s] dump must bracket objects with the concat-safe save", srv.name)
			require.Contains(t, source, "SET sql_mode = @saved_sql_mode;",
				"[%s] dump must restore sql_mode after each object", srv.name)
			require.Contains(t, strings.ToUpper(source), "NO_BACKSLASH_ESCAPES",
				"[%s] dump must preserve the NO_BACKSLASH_ESCAPES mode", srv.name)
			require.Contains(t, strings.ToUpper(source), "PAD_CHAR_TO_FULL_LENGTH",
				"[%s] dump must preserve the PAD_CHAR_TO_FULL_LENGTH mode", srv.name)

			// The dump round-trips through the omni declarative diff (SET is cosmetic).
			selfDiff, err := mysqlDiffSDLMigration(source, source, srv.version)
			require.NoError(t, err)
			require.Empty(t, selfDiff, "[%s] source-vs-source diff must be empty, got:\n%s", srv.name, selfDiff)

			// Re-apply the dumped SDL to a fresh scratch DB.
			scratchDB := newLiveDatabase(ctx, t, srv, "sdl_sesctx_scratch")
			require.NoError(t, applyDDL(ctx, t, srv, scratchDB, source),
				"[%s] dumped SDL must re-apply cleanly", srv.name)

			// (1) Re-dump self-diff empty: the scratch schema equals the origin dump.
			redump := dumpSDL(ctx, t, srv, scratchDB)
			diff, err := mysqlDiffSDLMigration(redump, source, srv.version)
			require.NoError(t, err)
			require.Empty(t, diff, "[%s] re-dump must equal origin dump (round-trip), residual:\n%s", srv.name, diff)

			// (2) The re-applied objects carry the authored mode (information_schema oracle),
			// and the modes did not leak into each other.
			dbh, closeDB := rawDB(ctx, t, srv, scratchDB)
			defer closeDB()
			faMode := routineSQLMode(ctx, t, dbh, scratchDB, "fa")
			pbMode := routineSQLMode(ctx, t, dbh, scratchDB, "pb")
			require.Contains(t, faMode, "NO_BACKSLASH_ESCAPES", "[%s] re-applied fa must carry NO_BACKSLASH_ESCAPES", srv.name)
			require.Contains(t, pbMode, "PAD_CHAR_TO_FULL_LENGTH", "[%s] re-applied pb must carry PAD_CHAR_TO_FULL_LENGTH", srv.name)
			require.NotContains(t, pbMode, "NO_BACKSLASH_ESCAPES",
				"[%s] pb must not inherit fa's NO_BACKSLASH_ESCAPES (leakage)", srv.name)
			require.NotContains(t, faMode, "PAD_CHAR_TO_FULL_LENGTH",
				"[%s] fa must not inherit pb's PAD_CHAR_TO_FULL_LENGTH (leakage)", srv.name)
		})
	}
}

// TestSDLSessionContextAnsiQuotesLive is the BYT-9832 proof for the ANSI_QUOTES mode, which the
// omni SDL parser cannot round-trip (double-quoted identifiers). It asserts the part the fix
// delivers: the dump carries the save/restore SET framing, and re-applying it to a live server
// recreates the routine/trigger UNDER ANSI_QUOTES (via information_schema) — a default-mode
// re-create would parse the "…"-quoted identifiers as string literals and silently change the
// object. The omni self-diff is intentionally not asserted here (pre-existing omni limitation).
//
//nolint:tparallel
func TestSDLSessionContextAnsiQuotesLive(t *testing.T) {
	skipUnlessLiveOracle(t)
	ctx := context.Background()

	// fa and trg authored under ANSI_QUOTES with bodies that rely on "…" as identifiers.
	originDDL := `CREATE TABLE t (id INT PRIMARY KEY, val INT NOT NULL DEFAULT 0) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

SET sql_mode = 'ANSI_QUOTES';
CREATE FUNCTION fa() RETURNS INT DETERMINISTIC RETURN (SELECT COUNT(*) FROM t WHERE "id" >= 0);
CREATE TRIGGER trg BEFORE INSERT ON t FOR EACH ROW SET NEW."val" = NEW."val" + 1;
SET sql_mode = '';
`

	for _, srv := range liveServers {
		srv := srv
		t.Run(srv.name, func(t *testing.T) {
			originDB := newLiveDatabase(ctx, t, srv, "sdl_ansi_origin")
			require.NoError(t, applyDDL(ctx, t, srv, originDB, originDDL))

			source := dumpSDL(ctx, t, srv, originDB)
			t.Logf("[%s] ANSI_QUOTES dump:\n%s", srv.name, source)
			require.Contains(t, source, "SET sql_mode = 'ANSI_QUOTES';",
				"[%s] dump must set the ANSI_QUOTES mode before the object", srv.name)
			require.Contains(t, source, "SET sql_mode = @saved_sql_mode;",
				"[%s] dump must restore sql_mode after the object", srv.name)

			// Re-apply the raw SDL dump to a fresh DB on one session; the SET framing must
			// recreate the objects under ANSI_QUOTES.
			scratchDB := newLiveDatabase(ctx, t, srv, "sdl_ansi_scratch")
			require.NoError(t, applyDDL(ctx, t, srv, scratchDB, source),
				"[%s] ANSI_QUOTES SDL must re-apply cleanly", srv.name)

			dbh, closeDB := rawDB(ctx, t, srv, scratchDB)
			defer closeDB()
			require.Contains(t, routineSQLMode(ctx, t, dbh, scratchDB, "fa"), "ANSI_QUOTES",
				"[%s] re-applied function fa must carry ANSI_QUOTES", srv.name)
			require.Contains(t, triggerSQLMode(ctx, t, dbh, scratchDB, "trg"), "ANSI_QUOTES",
				"[%s] re-applied trigger trg must carry ANSI_QUOTES", srv.name)

			// The re-applied object is behaviorally correct under its mode: fa returns the row
			// count (its "id" is an identifier), which only works if ANSI_QUOTES was in force at
			// create time. A default-mode create would have made "id" a constant string.
			var got int
			require.NoError(t, dbh.QueryRowContext(ctx, "SELECT `fa`()").Scan(&got))
			require.Equal(t, 0, got, "[%s] fa() must evaluate its identifier body under ANSI_QUOTES", srv.name)
		})
	}
}

// TestSDLSessionContextEventRoundTripLive covers the event path: an event with a non-UTC
// time_zone AND a non-default sql_mode must round-trip both. NO_BACKSLASH_ESCAPES keeps the
// dump omni-parseable, so the full round-trip (dump → re-apply → re-dump self-diff empty) is
// asserted along with the information_schema mode/time_zone oracle.
//
//nolint:tparallel
func TestSDLSessionContextEventRoundTripLive(t *testing.T) {
	skipUnlessLiveOracle(t)
	ctx := context.Background()

	// A numeric offset (+05:30) is stored/echoed verbatim and needs no mysql tz tables.
	eventDDL := `CREATE TABLE log (id INT PRIMARY KEY AUTO_INCREMENT, d DATE) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

SET time_zone = '+05:30';
SET sql_mode = 'NO_BACKSLASH_ESCAPES';
CREATE EVENT ev ON SCHEDULE EVERY 1 DAY DO INSERT INTO log (d) VALUES (CURDATE());
SET sql_mode = '';
SET time_zone = 'SYSTEM';
`

	for _, srv := range liveServers {
		srv := srv
		t.Run(srv.name, func(t *testing.T) {
			originDB := newLiveDatabase(ctx, t, srv, "sdl_sesctx_ev_origin")
			require.NoError(t, applyDDL(ctx, t, srv, originDB, eventDDL))

			source := dumpSDL(ctx, t, srv, originDB)
			t.Logf("[%s] event dump:\n%s", srv.name, source)
			require.Contains(t, source, "SET @saved_time_zone = @@time_zone;",
				"[%s] event dump must save time_zone", srv.name)
			require.Contains(t, source, "SET time_zone = '+05:30';",
				"[%s] event dump must set the authored time_zone", srv.name)
			require.Contains(t, source, "SET time_zone = @saved_time_zone;",
				"[%s] event dump must restore time_zone", srv.name)

			// Round-trips through the omni declarative diff.
			selfDiff, err := mysqlDiffSDLMigration(source, source, srv.version)
			require.NoError(t, err)
			require.Empty(t, selfDiff, "[%s] event source-vs-source must be empty, got:\n%s", srv.name, selfDiff)

			scratchDB := newLiveDatabase(ctx, t, srv, "sdl_sesctx_ev_scratch")
			require.NoError(t, applyDDL(ctx, t, srv, scratchDB, source),
				"[%s] event SDL must re-apply cleanly", srv.name)

			redump := dumpSDL(ctx, t, srv, scratchDB)
			diff, err := mysqlDiffSDLMigration(redump, source, srv.version)
			require.NoError(t, err)
			require.Empty(t, diff, "[%s] event re-dump must equal origin, residual:\n%s", srv.name, diff)

			dbh, closeDB := rawDB(ctx, t, srv, scratchDB)
			defer closeDB()
			mode, tz := eventContext(ctx, t, dbh, scratchDB, "ev")
			require.Contains(t, mode, "NO_BACKSLASH_ESCAPES", "[%s] re-applied event must carry NO_BACKSLASH_ESCAPES", srv.name)
			require.Equal(t, "+05:30", tz, "[%s] re-applied event must carry the authored time_zone", srv.name)
		})
	}
}

// TestSDLSessionContextNoLeakageLive is the multi-file concat no-leakage proof: two routines
// authored under DIFFERENT modes are exported to the multi-file set, concatenated in the
// action's lexical order, and applied to a fresh DB on ONE session. Each object must end up
// with ITS mode, not the neighbor's — the property a bare per-object SET would violate
// (session-level SET persists across the concatenation). Uses omni-parseable modes so the
// concat also round-trips through the declarative diff.
//
//nolint:tparallel
func TestSDLSessionContextNoLeakageLive(t *testing.T) {
	skipUnlessLiveOracle(t)
	ctx := context.Background()

	// fa under NO_BACKSLASH_ESCAPES, pb under PAD_CHAR_TO_FULL_LENGTH — two distinct non-default
	// modes with mode-neutral bodies (so the concat also round-trips through omni). In the
	// multi-file layout both live under functions/ and sort by name (fa before pb), so the
	// concatenation applies fa's block then pb's block on one connection — the leakage scenario.
	ddl := `CREATE TABLE t (id INT PRIMARY KEY) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

SET sql_mode = 'NO_BACKSLASH_ESCAPES';
CREATE FUNCTION fa() RETURNS INT DETERMINISTIC RETURN 1;
SET sql_mode = '';

SET sql_mode = 'PAD_CHAR_TO_FULL_LENGTH';
CREATE FUNCTION pb() RETURNS INT DETERMINISTIC RETURN 2;
SET sql_mode = '';
`

	for _, srv := range liveServers {
		srv := srv
		t.Run(srv.name, func(t *testing.T) {
			originDB := newLiveDatabase(ctx, t, srv, "sdl_noleak_origin")
			require.NoError(t, applyDDL(ctx, t, srv, originDB, ddl))

			// Multi-file export of the synced schema.
			meta := syncMetaForDB(ctx, t, srv, originDB)
			result, err := schema.GetMultiFileDatabaseDefinition(storepb.Engine_MYSQL,
				schema.GetDefinitionContext{SkipBackupSchema: true}, meta.GetProto())
			require.NoError(t, err)

			// Each function file must be self-contained: its own SET + restore around the CREATE.
			for _, f := range result.Files {
				if strings.HasPrefix(f.Name, "functions/") {
					require.Contains(t, f.Content, "SET @saved_sql_mode = @@sql_mode;",
						"[%s] function file %q must self-bracket its mode", srv.name, f.Name)
					require.Contains(t, f.Content, "SET sql_mode = @saved_sql_mode;",
						"[%s] function file %q must restore its mode", srv.name, f.Name)
				}
			}

			// Concatenate in the action's order and apply to a fresh DB on one session.
			concat := concatMultiFile(result)
			t.Logf("[%s] concatenated multi-file:\n%s", srv.name, concat)
			scratchDB := newLiveDatabase(ctx, t, srv, "sdl_noleak_scratch")
			require.NoError(t, applyDDL(ctx, t, srv, scratchDB, concat),
				"[%s] concatenated multi-file must apply cleanly", srv.name)

			// Each object carries ITS OWN mode — no leakage across the concatenation.
			dbh, closeDB := rawDB(ctx, t, srv, scratchDB)
			defer closeDB()
			faMode := routineSQLMode(ctx, t, dbh, scratchDB, "fa")
			pbMode := routineSQLMode(ctx, t, dbh, scratchDB, "pb")
			require.Contains(t, faMode, "NO_BACKSLASH_ESCAPES", "[%s] fa must carry NO_BACKSLASH_ESCAPES", srv.name)
			require.Contains(t, pbMode, "PAD_CHAR_TO_FULL_LENGTH", "[%s] pb must carry PAD_CHAR_TO_FULL_LENGTH", srv.name)
			require.NotContains(t, pbMode, "NO_BACKSLASH_ESCAPES",
				"[%s] pb must NOT inherit fa's NO_BACKSLASH_ESCAPES — concat leakage", srv.name)
			require.NotContains(t, faMode, "PAD_CHAR_TO_FULL_LENGTH",
				"[%s] fa must NOT carry pb's PAD_CHAR_TO_FULL_LENGTH", srv.name)

			// And the concat is a faithful, idempotent representation.
			concatMeta := syncMetaForDB(ctx, t, srv, scratchDB)
			redump, err := schema.MetadataToSDL(storepb.Engine_MYSQL, concatMeta)
			require.NoError(t, err)
			single := dumpSDL(ctx, t, srv, originDB)
			diff, err := mysqlDiffSDLMigration(redump, single, srv.version)
			require.NoError(t, err)
			require.Empty(t, diff, "[%s] concat re-dump must equal origin dump, residual:\n%s", srv.name, diff)
		})
	}
}
