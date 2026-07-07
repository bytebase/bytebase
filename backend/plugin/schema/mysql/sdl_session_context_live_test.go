package mysql

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"testing"

	mysqldriver "github.com/go-sql-driver/mysql"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/schema"
	"github.com/bytebase/bytebase/backend/store/model"
)

// End-to-end proof for BYT-9832 STAGE 2 (bytebase wiring) against the LIVE 5.7 (:13307)
// and 8.0 (:13306) oracles. It drives the REAL production entry point,
// schema.SDLMigration (the function database_migrate_executor.go's diff() calls), so the
// whole path exercised is: seed object under a distinctive session context → sync to
// metadata (which captures sql_mode/charset/collation, and event time_zone) →
// schema.SDLMigration(userSDL, syncedMetadata, version) → the generated migration is
// APPLIED on a deploy session whose DEFAULT sql_mode differs → read back
// information_schema to prove the ORIGINAL context survived the recreate/ALTER.
//
// Without the STAGE 2 wiring (buildSDLSessionContextMap + ApplySessionContext on the
// source catalog) the object comes back stamped with the deploy session's mode — which is
// exactly the regression these assertions catch.

// origRoutineMode is a non-default sql_mode whose semantics visibly differ from the deploy
// default: PIPES_AS_CONCAT makes `||` string concat (not OR), NO_BACKSLASH_ESCAPES changes
// escaping. A bare recreate under the deploy default would silently lose both.
const origRoutineMode = "PIPES_AS_CONCAT,NO_BACKSLASH_ESCAPES"

// deploySessionMode is the (different) sql_mode the migration is applied under, standing in
// for a deploy connection whose session default is not the object's authoring mode.
const deploySessionMode = "STRICT_TRANS_TABLES,NO_ENGINE_SUBSTITUTION"

// openLiveDB opens a raw *sql.DB against a live oracle with multi-statement support so a
// framed migration (SET …; CREATE …; SET …) runs as one script on one session, and returns
// it with a cleanup. It connects plaintext (no tls param), the equivalent of the mysql
// CLI's --ssl-mode=DISABLED that 5.7 needs.
func openLiveDB(t *testing.T, srv liveServer, database string) *sql.DB {
	t.Helper()
	cfg := mysqldriver.NewConfig()
	cfg.User = "root"
	cfg.Passwd = liveOraclePassword
	cfg.Net = "tcp"
	cfg.Addr = fmt.Sprintf("%s:%s", srv.host, srv.port)
	cfg.DBName = database
	cfg.MultiStatements = true
	cfg.AllowNativePasswords = true
	sqlDB, err := sql.Open("mysql", cfg.FormatDSN())
	require.NoError(t, err, "[%s] sql.Open", srv.name)
	t.Cleanup(func() { _ = sqlDB.Close() })
	return sqlDB
}

// scConn grabs a dedicated single connection (so session-variable state is stable across
// statements) and registers its cleanup.
func scConn(ctx context.Context, t *testing.T, sqlDB *sql.DB, srv liveServer) *sql.Conn {
	t.Helper()
	conn, err := sqlDB.Conn(ctx)
	require.NoError(t, err, "[%s] grab conn", srv.name)
	t.Cleanup(func() { _ = conn.Close() })
	return conn
}

// newSCDatabase creates a fresh database on srv and registers cleanup, returning its name.
func newSCDatabase(ctx context.Context, t *testing.T, srv liveServer, prefix string) string {
	t.Helper()
	dbName := fmt.Sprintf("%s_%s", prefix, strings.ReplaceAll(uuid.New().String(), "-", "_"))
	admin := openLiveDB(t, srv, "")
	_, err := admin.ExecContext(ctx, fmt.Sprintf("CREATE DATABASE `%s`", dbName))
	require.NoError(t, err, "[%s] create db", srv.name)
	t.Cleanup(func() {
		c := openLiveDB(t, srv, "")
		_, _ = c.ExecContext(ctx, fmt.Sprintf("DROP DATABASE IF EXISTS `%s`", dbName))
	})
	return dbName
}

// syncDBToMetadata syncs dbName on srv to the model.DatabaseMetadata the production release
// path feeds schema.SDLMigration — the value whose per-object sql_mode/charset/collation
// (and event time_zone) STAGE 2 threads into the diff.
func syncDBToMetadata(ctx context.Context, t *testing.T, srv liveServer, dbName string) *model.DatabaseMetadata {
	t.Helper()
	driver, err := createLiveMySQLDriver(ctx, srv, dbName)
	require.NoError(t, err, "[%s] open driver", srv.name)
	defer driver.Close(ctx)
	metadata, err := driver.SyncDBSchema(ctx)
	require.NoError(t, err, "[%s] sync", srv.name)
	return model.NewDatabaseMetadata(metadata, nil, nil, storepb.Engine_MYSQL, true)
}

// applyMigrationUnderDeployMode runs the generated migration on a session whose sql_mode is
// first set to deploySessionMode, proving the framing (not the ambient session) governs the
// stored context.
func applyMigrationUnderDeployMode(ctx context.Context, t *testing.T, conn *sql.Conn, srv, migrationSQL string) {
	t.Helper()
	_, err := conn.ExecContext(ctx, "SET SESSION sql_mode = "+quoteLiteral(deploySessionMode))
	require.NoError(t, err, "[%s] set deploy mode", srv)
	_, err = conn.ExecContext(ctx, migrationSQL)
	require.NoError(t, err, "[%s] APPLY FAILED:\n%s", srv, migrationSQL)
}

// quoteLiteral single-quotes a MySQL string literal for the test's own SET statements
// (the omni generator does its own quoting inside the migration).
func quoteLiteral(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "''") + "'"
}

func execUnderMode(ctx context.Context, t *testing.T, conn *sql.Conn, srv liveServer, sqlMode, timeZone string, stmts ...string) {
	t.Helper()
	_, err := conn.ExecContext(ctx, "SET SESSION sql_mode = "+quoteLiteral(sqlMode))
	require.NoError(t, err, "[%s] set authoring sql_mode", srv.name)
	if timeZone != "" {
		_, err = conn.ExecContext(ctx, "SET SESSION time_zone = "+quoteLiteral(timeZone))
		require.NoError(t, err, "[%s] set authoring time_zone", srv.name)
	}
	for _, s := range stmts {
		_, err = conn.ExecContext(ctx, s)
		require.NoError(t, err, "[%s] seed stmt %q", srv.name, s)
	}
}

func readRoutineSQLMode(ctx context.Context, t *testing.T, conn *sql.Conn, dbName, name string) string {
	t.Helper()
	var mode sql.NullString
	err := conn.QueryRowContext(ctx,
		"SELECT SQL_MODE FROM information_schema.ROUTINES WHERE ROUTINE_SCHEMA=? AND ROUTINE_NAME=?",
		dbName, name).Scan(&mode)
	require.NoError(t, err, "read routine sql_mode")
	return mode.String
}

func readTriggerSQLMode(ctx context.Context, t *testing.T, conn *sql.Conn, dbName, name string) string {
	t.Helper()
	var mode sql.NullString
	err := conn.QueryRowContext(ctx,
		"SELECT SQL_MODE FROM information_schema.TRIGGERS WHERE TRIGGER_SCHEMA=? AND TRIGGER_NAME=?",
		dbName, name).Scan(&mode)
	require.NoError(t, err, "read trigger sql_mode")
	return mode.String
}

func readEventModeTZ(ctx context.Context, t *testing.T, conn *sql.Conn, dbName, name string) (string, string) {
	t.Helper()
	var mode, tz sql.NullString
	err := conn.QueryRowContext(ctx,
		"SELECT SQL_MODE, TIME_ZONE FROM information_schema.EVENTS WHERE EVENT_SCHEMA=? AND EVENT_NAME=?",
		dbName, name).Scan(&mode, &tz)
	require.NoError(t, err, "read event sql_mode/time_zone")
	return mode.String, tz.String
}

// TestSDLSessionContextRoutineLive proves a routine whose BODY changes is re-emitted under
// its ORIGINAL sql_mode across the whole production wiring (schema.SDLMigration).
//
//nolint:tparallel
func TestSDLSessionContextRoutineLive(t *testing.T) {
	skipUnlessLiveOracle(t)
	ctx := context.Background()

	for _, srv := range liveServers {
		srv := srv
		t.Run(srv.name, func(t *testing.T) {
			dbName := newSCDatabase(ctx, t, srv, "sc_rtn")
			conn := scConn(ctx, t, openLiveDB(t, srv, dbName), srv)

			// Seed the function under the ORIGINAL mode.
			execUnderMode(ctx, t, conn, srv, origRoutineMode, "",
				"CREATE FUNCTION f(a INT) RETURNS INT DETERMINISTIC RETURN a + 1")

			// This is the exact value the release path hands schema.SDLMigration.
			metadata := syncDBToMetadata(ctx, t, srv, dbName)

			// Desired SDL: same routine, body edited (a + 2). Bare — carries no session framing.
			userSDL := "CREATE FUNCTION f(a INT) RETURNS INT DETERMINISTIC RETURN a + 2;"
			migrationSQL, err := schema.SDLMigration(storepb.Engine_MYSQL, userSDL, metadata, srv.version)
			require.NoError(t, err)
			require.NotEmpty(t, migrationSQL, "[%s] expected a recreate migration", srv.name)
			t.Logf("[%s] routine migration:\n%s", srv.name, migrationSQL)
			// The framing must be present (proof STAGE 2 wired the context through).
			require.Contains(t, migrationSQL, "SET sql_mode",
				"[%s] migration lacks sql_mode framing:\n%s", srv.name, migrationSQL)

			applyMigrationUnderDeployMode(ctx, t, conn, srv.name, migrationSQL)

			got := readRoutineSQLMode(ctx, t, conn, dbName, "f")
			require.Equal(t, origRoutineMode, got,
				"[%s] routine sql_mode not preserved across recreate\nmigration:\n%s", srv.name, migrationSQL)
		})
	}
}

// TestSDLSessionContextTriggerLive proves a trigger whose BODY changes is re-emitted under
// its ORIGINAL sql_mode across schema.SDLMigration.
//
//nolint:tparallel
func TestSDLSessionContextTriggerLive(t *testing.T) {
	skipUnlessLiveOracle(t)
	ctx := context.Background()

	for _, srv := range liveServers {
		srv := srv
		t.Run(srv.name, func(t *testing.T) {
			dbName := newSCDatabase(ctx, t, srv, "sc_trg")
			conn := scConn(ctx, t, openLiveDB(t, srv, dbName), srv)

			base := []string{
				"CREATE TABLE t (id INT PRIMARY KEY, val INT NOT NULL DEFAULT 0) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4",
				"CREATE TABLE audit (id INT PRIMARY KEY AUTO_INCREMENT, tid INT, oldv INT) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4",
			}
			// Tables under the deploy mode; the TRIGGER under the ORIGINAL mode.
			execUnderMode(ctx, t, conn, srv, deploySessionMode, "", base...)
			execUnderMode(ctx, t, conn, srv, origRoutineMode, "",
				"CREATE TRIGGER t_audit AFTER UPDATE ON t FOR EACH ROW BEGIN INSERT INTO audit (tid, oldv) VALUES (NEW.id, OLD.val); END")

			metadata := syncDBToMetadata(ctx, t, srv, dbName)

			userSDL := strings.Join([]string{
				"CREATE TABLE t (id INT PRIMARY KEY, val INT NOT NULL DEFAULT 0) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;",
				"CREATE TABLE audit (id INT PRIMARY KEY AUTO_INCREMENT, tid INT, oldv INT) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;",
				"CREATE TRIGGER t_audit AFTER UPDATE ON t FOR EACH ROW BEGIN INSERT INTO audit (tid, oldv) VALUES (NEW.id, OLD.val + 1); END;",
			}, "\n")
			migrationSQL, err := schema.SDLMigration(storepb.Engine_MYSQL, userSDL, metadata, srv.version)
			require.NoError(t, err)
			require.NotEmpty(t, migrationSQL, "[%s] expected a trigger recreate migration", srv.name)
			t.Logf("[%s] trigger migration:\n%s", srv.name, migrationSQL)
			require.Contains(t, migrationSQL, "SET sql_mode",
				"[%s] trigger migration lacks sql_mode framing:\n%s", srv.name, migrationSQL)

			applyMigrationUnderDeployMode(ctx, t, conn, srv.name, migrationSQL)

			got := readTriggerSQLMode(ctx, t, conn, dbName, "t_audit")
			require.Equal(t, origRoutineMode, got,
				"[%s] trigger sql_mode not preserved across recreate\nmigration:\n%s", srv.name, migrationSQL)
		})
	}
}

// TestSDLSessionContextEventLive proves an event whose BODY changes preserves both sql_mode
// and the ORIGINAL time_zone across schema.SDLMigration. The event modify path is an ALTER
// EVENT … DO (which empirically re-stamps sql_mode from the session), so a bare apply would
// silently lose the original mode — the case the framing exists for.
//
//nolint:tparallel
func TestSDLSessionContextEventLive(t *testing.T) {
	skipUnlessLiveOracle(t)
	ctx := context.Background()

	const origEventMode = "PIPES_AS_CONCAT"
	const origTZ = "+08:00"

	for _, srv := range liveServers {
		srv := srv
		t.Run(srv.name, func(t *testing.T) {
			dbName := newSCDatabase(ctx, t, srv, "sc_evt")
			conn := scConn(ctx, t, openLiveDB(t, srv, dbName), srv)

			// event_scheduler toggle may lack privilege; ignore its error only.
			if _, err := conn.ExecContext(ctx, "SET GLOBAL event_scheduler = OFF"); err != nil {
				t.Logf("[%s] event_scheduler off (ignored): %v", srv.name, err)
			}
			execUnderMode(ctx, t, conn, srv, origEventMode, origTZ,
				"CREATE EVENT e ON SCHEDULE EVERY 1 HOUR DISABLE DO SET @x = 1")

			metadata := syncDBToMetadata(ctx, t, srv, dbName)

			userSDL := "CREATE EVENT e ON SCHEDULE EVERY 1 HOUR DISABLE DO SET @x = 2;"
			migrationSQL, err := schema.SDLMigration(storepb.Engine_MYSQL, userSDL, metadata, srv.version)
			require.NoError(t, err)
			require.NotEmpty(t, migrationSQL, "[%s] expected an event modify migration", srv.name)
			t.Logf("[%s] event migration:\n%s", srv.name, migrationSQL)
			require.Contains(t, migrationSQL, "SET sql_mode",
				"[%s] event migration lacks sql_mode framing:\n%s", srv.name, migrationSQL)

			applyMigrationUnderDeployMode(ctx, t, conn, srv.name, migrationSQL)

			gotMode, gotTZ := readEventModeTZ(ctx, t, conn, dbName, "e")
			require.Equal(t, origEventMode, gotMode,
				"[%s] event sql_mode not preserved\nmigration:\n%s", srv.name, migrationSQL)
			require.Equal(t, origTZ, gotTZ,
				"[%s] event time_zone not preserved\nmigration:\n%s", srv.name, migrationSQL)
		})
	}
}

// TestSDLSessionContextNewObjectDefaultLive proves a routine ABSENT from the source (a
// first-time create) is emitted BARE — no session framing — so it adopts the server default
// mode. STAGE 2 must only preserve context for objects that already exist in the source.
//
//nolint:tparallel
func TestSDLSessionContextNewObjectDefaultLive(t *testing.T) {
	skipUnlessLiveOracle(t)
	ctx := context.Background()

	for _, srv := range liveServers {
		srv := srv
		t.Run(srv.name, func(t *testing.T) {
			dbName := newSCDatabase(ctx, t, srv, "sc_new")
			conn := scConn(ctx, t, openLiveDB(t, srv, dbName), srv)

			// Source has only a table; the routine is brand new in the desired SDL.
			execUnderMode(ctx, t, conn, srv, deploySessionMode, "",
				"CREATE TABLE t (id INT PRIMARY KEY) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4")

			metadata := syncDBToMetadata(ctx, t, srv, dbName)

			userSDL := strings.Join([]string{
				"CREATE TABLE t (id INT PRIMARY KEY) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;",
				"CREATE FUNCTION g(a INT) RETURNS INT DETERMINISTIC RETURN a + 5;",
			}, "\n")
			migrationSQL, err := schema.SDLMigration(storepb.Engine_MYSQL, userSDL, metadata, srv.version)
			require.NoError(t, err)
			require.NotEmpty(t, migrationSQL, "[%s] expected a create migration", srv.name)
			t.Logf("[%s] new-object migration:\n%s", srv.name, migrationSQL)
			// A first-time create must NOT be wrapped in session framing.
			require.NotContains(t, migrationSQL, "SET sql_mode",
				"[%s] first-time create should be bare (no session framing):\n%s", srv.name, migrationSQL)

			// applyMigrationUnderDeployMode runs it on a session whose sql_mode is
			// deploySessionMode; a bare CREATE therefore captures that mode. This proves a
			// brand-new object adopts the deploy session default (no preserved context).
			applyMigrationUnderDeployMode(ctx, t, conn, srv.name, migrationSQL)

			got := readRoutineSQLMode(ctx, t, conn, dbName, "g")
			require.Equal(t, deploySessionMode, got,
				"[%s] new routine should adopt the deploy session mode, got %q", srv.name, got)
		})
	}
}

// TestSDLSessionContextNoChurnLive proves that an identical source and desired routine —
// same body, with the source's session context carried in from synced metadata — produces
// NO migration. The session context is deliberately excluded from declarative identity, so
// a mode-only carry must never manufacture a phantom recreate.
//
//nolint:tparallel
func TestSDLSessionContextNoChurnLive(t *testing.T) {
	skipUnlessLiveOracle(t)
	ctx := context.Background()

	for _, srv := range liveServers {
		srv := srv
		t.Run(srv.name, func(t *testing.T) {
			dbName := newSCDatabase(ctx, t, srv, "sc_noop")
			conn := scConn(ctx, t, openLiveDB(t, srv, dbName), srv)

			execUnderMode(ctx, t, conn, srv, origRoutineMode, "",
				"CREATE FUNCTION f(a INT) RETURNS INT DETERMINISTIC RETURN a + 1")

			metadata := syncDBToMetadata(ctx, t, srv, dbName)

			// Desired SDL == the dumped current object (same body). No change intended.
			userSDL, err := schema.MetadataToSDL(storepb.Engine_MYSQL, metadata)
			require.NoError(t, err)

			migrationSQL, err := schema.SDLMigration(storepb.Engine_MYSQL, userSDL, metadata, srv.version)
			require.NoError(t, err)
			require.Empty(t, migrationSQL,
				"[%s] identical schema (context carried) must produce no migration, got:\n%s", srv.name, migrationSQL)
		})
	}
}
