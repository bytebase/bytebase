package mysql

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"

	_ "github.com/go-sql-driver/mysql"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	"github.com/bytebase/bytebase/backend/plugin/db"
	mysqldb "github.com/bytebase/bytebase/backend/plugin/db/mysql"
	"github.com/bytebase/bytebase/backend/plugin/schema"
	"github.com/bytebase/bytebase/backend/store/model"
)

// skipUnlessLiveOracle gates every live-oracle SDL suite in this package. These
// suites need the developer's local MySQL oracles (5.7 at 127.0.0.1:13307, 8.0 at
// 127.0.0.1:13306 — see liveServers) and are opt-in via MYSQL_SDL_LIVE_ORACLE=1.
// CI runs `go test ./backend/...` without -short and has no such servers, so an
// explicit environment gate (mirroring the cosmosdb integration tests) keeps the
// suites out of CI while leaving them one env var away locally.
func skipUnlessLiveOracle(t *testing.T) {
	t.Helper()
	if testing.Short() {
		t.Skip("skipping live MySQL SDL oracle test in short mode")
	}
	if os.Getenv("MYSQL_SDL_LIVE_ORACLE") == "" {
		t.Skip("skipping live MySQL SDL oracle test: set MYSQL_SDL_LIVE_ORACLE=1 (needs local MySQL 5.7 at :13307 and 8.0 at :13306)")
	}
}

// liveServer describes a live oracle MySQL instance the SDL smoke test exercises.
type liveServer struct {
	name    string
	host    string
	port    string
	version string
}

// liveServers are the two oracle MySQL instances. Idempotence diverges by version
// (5.7 injects integer display widths and uses the utf8mb4_general_ci default; 8.0
// drops widths and uses utf8mb4_0900_ai_ci), so both are exercised.
var liveServers = []liveServer{
	{name: "mysql80", host: "127.0.0.1", port: "13306", version: "8.0"},
	{name: "mysql57", host: "127.0.0.1", port: "13307", version: "5.7"},
}

const liveOraclePassword = "010424"

// representativeDDL is the user-authored target schema D. It intentionally spans the
// constructs whose stored form MySQL rewrites: integer display widths, BOOLEAN
// (stored tinyint(1)), DECIMAL, implicit vs explicit charset/collation, defaults
// (literal, CURRENT_TIMESTAMP, ON UPDATE), a secondary index, a foreign key, a
// STORED generated column, and a view.
const representativeDDL = `
CREATE TABLE author (
	id INT PRIMARY KEY AUTO_INCREMENT,
	name VARCHAR(100) NOT NULL,
	active BOOLEAN NOT NULL DEFAULT TRUE,
	rating DECIMAL(5, 2) NOT NULL DEFAULT 0.00,
	bio TEXT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci,
	created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
	INDEX idx_author_name (name)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE book (
	id INT PRIMARY KEY AUTO_INCREMENT,
	author_id INT NOT NULL,
	title VARCHAR(200) NOT NULL,
	price DECIMAL(10, 2) NOT NULL DEFAULT 0.00,
	discount DECIMAL(4, 2) NOT NULL DEFAULT 0.00,
	final_price DECIMAL(12, 4) AS (price * (1 - discount)) STORED,
	page_count INT NOT NULL DEFAULT 0,
	CONSTRAINT fk_book_author FOREIGN KEY (author_id) REFERENCES author (id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE OR REPLACE VIEW active_books AS
SELECT b.id, b.title, b.final_price
FROM book b JOIN author a ON b.author_id = a.id
WHERE a.active = TRUE;
`

// statementCount counts the non-empty ";"-separated statements in generated DDL.
// MigrationPlan.SQL joins ops with ";\n", so this equals the number of operations.
func statementCount(sql string) int {
	n := 0
	for _, part := range strings.Split(sql, ";") {
		if strings.TrimSpace(part) != "" {
			n++
		}
	}
	return n
}

// createLiveMySQLDriver opens a driver against a live oracle MySQL instance.
func createLiveMySQLDriver(ctx context.Context, srv liveServer, database string) (db.Driver, error) {
	driver := &mysqldb.Driver{}
	config := db.ConnectionConfig{
		DataSource: &storepb.DataSource{
			Type:     storepb.DataSourceType_ADMIN,
			Username: "root",
			Host:     srv.host,
			Port:     srv.port,
			Database: database,
		},
		Password: liveOraclePassword,
		ConnectionContext: db.ConnectionContext{
			EngineVersion: srv.version,
			DatabaseName:  database,
		},
	}
	return driver.Open(ctx, storepb.Engine_MYSQL, config)
}

// syncToSDL applies representativeDDL to a fresh database on srv, syncs it back to
// metadata, and returns the canonical SDL dump (MetadataToSDL). The database is
// dropped on cleanup.
func syncToSDL(ctx context.Context, t *testing.T, srv liveServer) string {
	t.Helper()

	dbName := fmt.Sprintf("sdl_smoke_%s", strings.ReplaceAll(uuid.New().String(), "-", "_"))

	admin, err := createLiveMySQLDriver(ctx, srv, "")
	require.NoError(t, err)
	_, err = admin.Execute(ctx, fmt.Sprintf("CREATE DATABASE `%s`", dbName), db.ExecuteOptions{})
	require.NoError(t, err)
	admin.Close(ctx)

	t.Cleanup(func() {
		cleanup, err := createLiveMySQLDriver(ctx, srv, "")
		if err != nil {
			return
		}
		defer cleanup.Close(ctx)
		_, _ = cleanup.Execute(ctx, fmt.Sprintf("DROP DATABASE IF EXISTS `%s`", dbName), db.ExecuteOptions{})
	})

	driver, err := createLiveMySQLDriver(ctx, srv, dbName)
	require.NoError(t, err)
	defer driver.Close(ctx)

	_, err = driver.Execute(ctx, representativeDDL, db.ExecuteOptions{})
	require.NoError(t, err)

	metadata, err := driver.SyncDBSchema(ctx)
	require.NoError(t, err)

	source, err := schema.MetadataToSDL(storepb.Engine_MYSQL, model.NewDatabaseMetadata(metadata, nil, nil, storepb.Engine_MYSQL, true))
	require.NoError(t, err)
	require.NotEmpty(t, source, "MetadataToSDL produced empty SDL")

	return source
}

// syncToMetadata applies representativeDDL to a fresh database on srv, syncs it back, and
// returns the model.DatabaseMetadata — the exact value the production release path feeds to
// schema.SDLMigration — together with the database name so the caller can apply DDL back.
// The database is dropped on cleanup.
func syncToMetadata(ctx context.Context, t *testing.T, srv liveServer) (*model.DatabaseMetadata, string) {
	t.Helper()

	dbName := newLiveDatabase(ctx, t, srv, "sdl_prod")
	driver, err := createLiveMySQLDriver(ctx, srv, dbName)
	require.NoError(t, err)
	defer driver.Close(ctx)

	_, err = driver.Execute(ctx, representativeDDL, db.ExecuteOptions{})
	require.NoError(t, err)

	metadata, err := driver.SyncDBSchema(ctx)
	require.NoError(t, err)
	return model.NewDatabaseMetadata(metadata, nil, nil, storepb.Engine_MYSQL, true), dbName
}

// TestSDLProductionPathLive exercises the REAL production entry point, schema.SDLMigration
// (the function database_migrate_executor.go's diff() calls), end to end against live 5.7
// and 8.0. It proves the version threading wired through schema.SDLMigration ->
// mysqlDiffSDLMigration -> the MySQL version-aware registry reaches the omni
// normalizer:
//
//	(1) the 5.7 no-op (synced metadata diffed against its own authoring DDL) is empty —
//	    a non-empty result is the version-dispatch regression (utf8mb4 default-collation
//	    phantom) this wiring fixes; and
//	(2) a 5.7 schema change produces DDL valid on 5.7 — never names utf8mb4_0900_ai_ci
//	    (errno 1273) — and applies cleanly to the live 5.7 server.
//
//nolint:tparallel
func TestSDLProductionPathLive(t *testing.T) {
	skipUnlessLiveOracle(t)
	ctx := context.Background()

	addColumnDDL := strings.Replace(
		representativeDDL,
		"\tpage_count INT NOT NULL DEFAULT 0,",
		"\tpage_count INT NOT NULL DEFAULT 0,\n\tisbn VARCHAR(20) NULL,",
		1,
	)
	require.NotEqual(t, representativeDDL, addColumnDDL, "test setup: addColumnDDL must differ from D")

	for _, srv := range liveServers {
		srv := srv
		t.Run(srv.name, func(t *testing.T) {
			meta, dbName := syncToMetadata(ctx, t, srv)

			// (1) Production-path no-op idempotence: schema.SDLMigration converts meta to
			// SDL internally and diffs against the user DDL, threading srv.version.
			noop, err := schema.SDLMigration(storepb.Engine_MYSQL, representativeDDL, meta, srv.version)
			require.NoError(t, err)
			require.Empty(t, noop, "[%s] production-path no-op must be empty, got:\n%s", srv.name, noop)

			// (2) Production-path change: minimal, version-valid DDL.
			change, err := schema.SDLMigration(storepb.Engine_MYSQL, addColumnDDL, meta, srv.version)
			require.NoError(t, err)
			require.NotEmpty(t, change, "[%s] production-path change must be non-empty", srv.name)
			upper := strings.ToUpper(change)
			require.Contains(t, upper, "ADD COLUMN", "[%s] expected ADD COLUMN, got:\n%s", srv.name, change)
			require.Contains(t, upper, "ISBN", "[%s] expected the new column, got:\n%s", srv.name, change)
			require.NotContains(t, upper, "CREATE TABLE", "[%s] change must not rebuild the table:\n%s", srv.name, change)
			if srv.version == "5.7" {
				require.NotContains(t, change, "utf8mb4_0900_ai_ci",
					"[%s] production-path 5.7 change names an 8.0-only collation (errno 1273):\n%s", srv.name, change)
			}

			// Apply the production-path DDL back to the real server — errno 1273 would
			// surface here on 5.7 if the version were not threaded through schema.SDLMigration.
			applyDriver, err := createLiveMySQLDriver(ctx, srv, dbName)
			require.NoError(t, err)
			defer applyDriver.Close(ctx)
			_, applyErr := applyDriver.Execute(ctx, change, db.ExecuteOptions{})
			require.NoError(t, applyErr, "[%s] production-path DDL failed to apply:\n%s", srv.name, change)
		})
	}
}

// TestSDLDeclarativePathLive exercises the full wired declarative path against live
// MySQL 5.7 and 8.0: sync -> MetadataToSDL -> DiffSDLMigration. It asserts the
// no-op idempotence property, minimal-DDL change generation, and apply-back.
//
//nolint:tparallel
func TestSDLDeclarativePathLive(t *testing.T) {
	skipUnlessLiveOracle(t)
	ctx := context.Background()

	for _, srv := range liveServers {
		srv := srv
		t.Run(srv.name, func(t *testing.T) {
			// ---- Assertion 1: no-op idempotence ----
			source := syncToSDL(ctx, t, srv)

			t.Logf("[%s] canonical SDL source:\n%s", srv.name, source)

			// source vs source must ALWAYS be empty on both versions (identical inputs;
			// version-independent). A non-empty result here would be a pure wiring/dump
			// determinism bug.
			selfDiff, err := mysqlDiffSDLMigration(source, source, srv.version)
			require.NoError(t, err)
			require.Empty(t, selfDiff, "[%s] source-vs-source diff must be empty, got:\n%s", srv.name, selfDiff)

			// source vs D is the meaningful case: the stored/dumped canonical form
			// diffed against the user's original DDL must canonicalize equal. The version
			// is threaded so a 5.7 schema is canonicalized as 5.7 (a bare CHARSET=utf8mb4
			// resolves to utf8mb4_general_ci, not the 8.0 default) — the fix for the
			// version-dispatch gap (omni catalog SessionState.Version + LoadSDLWithVersion).
			noopDiff, err := mysqlDiffSDLMigration(source, representativeDDL, srv.version)
			require.NoError(t, err)
			require.Empty(t, noopDiff, "[%s] no-op idempotence FAILED: source-vs-D diff must be empty, got:\n%s", srv.name, noopDiff)
		})
	}
}

// TestSDLMinimalChangeLive asserts that a real change to D produces minimal DDL and
// that applying it back converges (re-diff empty). Split out so a no-op failure
// (TestSDLDeclarativePathLive) is reported independently of a change-generation
// failure.
//
//nolint:tparallel
func TestSDLMinimalChangeLive(t *testing.T) {
	skipUnlessLiveOracle(t)
	ctx := context.Background()

	// D2 = D with one added column on `book`.
	addColumnDDL := strings.Replace(
		representativeDDL,
		"\tpage_count INT NOT NULL DEFAULT 0,",
		"\tpage_count INT NOT NULL DEFAULT 0,\n\tisbn VARCHAR(20) NULL,",
		1,
	)
	require.NotEqual(t, representativeDDL, addColumnDDL, "test setup: addColumnDDL must differ from D")

	// D2' = D with a changed column type (page_count INT -> BIGINT).
	changeTypeDDL := strings.Replace(
		representativeDDL,
		"\tpage_count INT NOT NULL DEFAULT 0,",
		"\tpage_count BIGINT NOT NULL DEFAULT 0,",
		1,
	)
	require.NotEqual(t, representativeDDL, changeTypeDDL, "test setup: changeTypeDDL must differ from D")

	// D2'' = D with an added secondary index on book(title).
	addIndexDDL := strings.Replace(
		representativeDDL,
		"\tpage_count INT NOT NULL DEFAULT 0,",
		"\tpage_count INT NOT NULL DEFAULT 0,\n\tINDEX idx_book_title (title),",
		1,
	)
	require.NotEqual(t, representativeDDL, addIndexDDL, "test setup: addIndexDDL must differ from D")

	for _, srv := range liveServers {
		srv := srv
		t.Run(srv.name, func(t *testing.T) {
			t.Run("add_column", func(t *testing.T) {
				source := syncToSDL(ctx, t, srv)

				diff, err := mysqlDiffSDLMigration(source, addColumnDDL, srv.version)
				require.NoError(t, err)
				t.Logf("[%s] add-column diff:\n%s", srv.name, diff)
				require.NotEmpty(t, diff, "[%s] add-column must produce DDL", srv.name)
				upper := strings.ToUpper(diff)
				require.Contains(t, upper, "ADD COLUMN", "[%s] expected ADD COLUMN, got:\n%s", srv.name, diff)
				require.Contains(t, upper, "ISBN", "[%s] expected the new column, got:\n%s", srv.name, diff)
				require.NotContains(t, upper, "CREATE TABLE", "[%s] add-column must not rebuild the table:\n%s", srv.name, diff)
				require.NotContains(t, upper, "DROP TABLE", "[%s] add-column must not drop the table:\n%s", srv.name, diff)
				// Strict minimality: exactly the ADD COLUMN, nothing else — on BOTH versions,
				// now that the version is threaded and the 5.7 utf8mb4-default-collation
				// phantom no longer contaminates the diff.
				require.Equal(t, 1, statementCount(diff),
					"[%s] add-column must be a single minimal statement, got:\n%s", srv.name, diff)
			})

			t.Run("change_type", func(t *testing.T) {
				source := syncToSDL(ctx, t, srv)

				diff, err := mysqlDiffSDLMigration(source, changeTypeDDL, srv.version)
				require.NoError(t, err)
				t.Logf("[%s] change-type diff:\n%s", srv.name, diff)
				require.NotEmpty(t, diff, "[%s] change-type must produce DDL", srv.name)
				upper := strings.ToUpper(diff)
				require.True(t,
					strings.Contains(upper, "MODIFY") || strings.Contains(upper, "CHANGE"),
					"[%s] expected MODIFY/CHANGE COLUMN, got:\n%s", srv.name, diff)
				require.Contains(t, upper, "BIGINT", "[%s] expected the new type, got:\n%s", srv.name, diff)
				require.NotContains(t, upper, "CREATE TABLE", "[%s] change-type must not rebuild the table:\n%s", srv.name, diff)
				require.Equal(t, 1, statementCount(diff),
					"[%s] change-type must be a single minimal statement, got:\n%s", srv.name, diff)
			})

			// add_index now exercises REAL index-diff behavior: the merged omni
			// breadth engine populates SchemaDiff.Indexes and generates a minimal
			// ADD KEY. MySQL renders a secondary index as `ADD KEY` (the canonical
			// synonym of ADD INDEX), so the assertion accepts either spelling.
			t.Run("add_index", func(t *testing.T) {
				source := syncToSDL(ctx, t, srv)

				diff, err := mysqlDiffSDLMigration(source, addIndexDDL, srv.version)
				require.NoError(t, err)
				t.Logf("[%s] add-index diff:\n%s", srv.name, diff)
				require.NotEmpty(t, diff, "[%s] add-index must produce DDL", srv.name)
				upper := strings.ToUpper(diff)
				require.True(t,
					strings.Contains(upper, "ADD KEY") || strings.Contains(upper, "ADD INDEX"),
					"[%s] expected ADD KEY/INDEX, got:\n%s", srv.name, diff)
				require.Contains(t, upper, "IDX_BOOK_TITLE", "[%s] expected the new index, got:\n%s", srv.name, diff)
				require.NotContains(t, upper, "CREATE TABLE", "[%s] add-index must not rebuild the table:\n%s", srv.name, diff)
				require.NotContains(t, upper, "DROP TABLE", "[%s] add-index must not drop the table:\n%s", srv.name, diff)
				require.Equal(t, 1, statementCount(diff),
					"[%s] add-index must be a single minimal statement, got:\n%s", srv.name, diff)
			})
		})
	}
}

// TestSDLApplyBackLive exercises assertion 3: take D, sync to source, compute the
// minimal change DDL against D2, apply that DDL to the real DB, re-sync, and confirm
// the schema now matches D2 (re-running the diff yields empty).
//
//nolint:tparallel
func TestSDLApplyBackLive(t *testing.T) {
	skipUnlessLiveOracle(t)
	ctx := context.Background()

	addColumnDDL := strings.Replace(
		representativeDDL,
		"\tpage_count INT NOT NULL DEFAULT 0,",
		"\tpage_count INT NOT NULL DEFAULT 0,\n\tisbn VARCHAR(20) NULL,",
		1,
	)

	for _, srv := range liveServers {
		srv := srv
		t.Run(srv.name, func(t *testing.T) {
			// Build the database, capture its name, sync to source.
			dbName := fmt.Sprintf("sdl_apply_%s", strings.ReplaceAll(uuid.New().String(), "-", "_"))

			admin, err := createLiveMySQLDriver(ctx, srv, "")
			require.NoError(t, err)
			_, err = admin.Execute(ctx, fmt.Sprintf("CREATE DATABASE `%s`", dbName), db.ExecuteOptions{})
			require.NoError(t, err)
			admin.Close(ctx)
			t.Cleanup(func() {
				cleanup, err := createLiveMySQLDriver(ctx, srv, "")
				if err != nil {
					return
				}
				defer cleanup.Close(ctx)
				_, _ = cleanup.Execute(ctx, fmt.Sprintf("DROP DATABASE IF EXISTS `%s`", dbName), db.ExecuteOptions{})
			})

			driver, err := createLiveMySQLDriver(ctx, srv, dbName)
			require.NoError(t, err)
			defer driver.Close(ctx)

			_, err = driver.Execute(ctx, representativeDDL, db.ExecuteOptions{})
			require.NoError(t, err)
			metadata, err := driver.SyncDBSchema(ctx)
			require.NoError(t, err)
			source, err := schema.MetadataToSDL(storepb.Engine_MYSQL, model.NewDatabaseMetadata(metadata, nil, nil, storepb.Engine_MYSQL, true))
			require.NoError(t, err)

			// Compute minimal change DDL for D2 (add column), version-aware so the change
			// DDL is valid on the target server (no utf8mb4_0900_ai_ci on 5.7).
			changeDDL, err := mysqlDiffSDLMigration(source, addColumnDDL, srv.version)
			require.NoError(t, err)
			require.NotEmpty(t, changeDDL, "[%s] expected non-empty change DDL", srv.name)
			t.Logf("[%s] apply-back change DDL:\n%s", srv.name, changeDDL)

			// The change DDL must never name a collation the target server lacks.
			if srv.version == "5.7" {
				require.NotContains(t, changeDDL, "utf8mb4_0900_ai_ci",
					"[%s] change DDL names a collation that does not exist on 5.7:\n%s", srv.name, changeDDL)
			}

			// Apply it back to the real DB — this is where Error 1273 would surface on 5.7
			// if the version were not threaded.
			applyDriver, err := createLiveMySQLDriver(ctx, srv, dbName)
			require.NoError(t, err)
			defer applyDriver.Close(ctx)
			_, applyErr := applyDriver.Execute(ctx, changeDDL, db.ExecuteOptions{})
			require.NoError(t, applyErr, "[%s] apply-back DDL failed to apply", srv.name)

			newMetadata, err := applyDriver.SyncDBSchema(ctx)
			require.NoError(t, err)
			newSource, err := schema.MetadataToSDL(storepb.Engine_MYSQL, model.NewDatabaseMetadata(newMetadata, nil, nil, storepb.Engine_MYSQL, true))
			require.NoError(t, err)

			// The schema must now match D2: re-diff yields empty.
			converge, err := mysqlDiffSDLMigration(newSource, addColumnDDL, srv.version)
			require.NoError(t, err)
			require.Empty(t, converge, "[%s] apply-back did not converge to D2, residual diff:\n%s", srv.name, converge)
		})
	}
}

// breadthCase exercises one MySQL breadth object type end to end through the wired
// declarative path. baseDDL builds the starting schema; targetDDL is the desired SDL.
// noop holds true when baseDDL already equals the target's object (idempotence is then
// asserted against targetDDL directly). When noop is false, baseDDL omits/differs the
// object so the diff produces a real change whose generated DDL must contain changeWant
// (case-insensitive) and, when applied back, must converge.
type breadthCase struct {
	name       string
	baseDDL    string
	targetDDL  string
	changeWant string // uppercase substring the minimal change DDL must contain
	skip57     bool   // CHECK constraints are 8.0-only
}

// newLiveDatabase creates a fresh database on srv and registers cleanup. It returns the
// database name so the caller can apply DDL and sync it repeatedly within one test.
func newLiveDatabase(ctx context.Context, t *testing.T, srv liveServer, prefix string) string {
	t.Helper()
	dbName := fmt.Sprintf("%s_%s", prefix, strings.ReplaceAll(uuid.New().String(), "-", "_"))
	admin, err := createLiveMySQLDriver(ctx, srv, "")
	require.NoError(t, err)
	_, err = admin.Execute(ctx, fmt.Sprintf("CREATE DATABASE `%s`", dbName), db.ExecuteOptions{})
	require.NoError(t, err)
	admin.Close(ctx)
	t.Cleanup(func() {
		c, err := createLiveMySQLDriver(ctx, srv, "")
		if err != nil {
			return
		}
		defer c.Close(ctx)
		_, _ = c.Execute(ctx, fmt.Sprintf("DROP DATABASE IF EXISTS `%s`", dbName), db.ExecuteOptions{})
	})
	return dbName
}

// applyAndDump applies ddl to dbName on srv, syncs the schema, and returns the canonical
// SDL dump (MetadataToSDL of the synced metadata) — i.e. the declarative-path source.
func applyAndDump(ctx context.Context, t *testing.T, srv liveServer, dbName, ddl string) string {
	t.Helper()
	driver, err := createLiveMySQLDriver(ctx, srv, dbName)
	require.NoError(t, err)
	defer driver.Close(ctx)
	_, err = driver.Execute(ctx, ddl, db.ExecuteOptions{})
	require.NoError(t, err, "apply DDL")
	metadata, err := driver.SyncDBSchema(ctx)
	require.NoError(t, err)
	source, err := schema.MetadataToSDL(storepb.Engine_MYSQL, model.NewDatabaseMetadata(metadata, nil, nil, storepb.Engine_MYSQL, true))
	require.NoError(t, err)
	return source
}

// breadthCases covers the eight MySQL breadth object types the merged omni engine
// supports: secondary/unique index, foreign key, CHECK (8.0), partition, view, stored
// routine (function + procedure), trigger, and event. Each carries a base→target delta
// that names exactly one object so the generated change is minimal.
func breadthCases() []breadthCase {
	return []breadthCase{
		{
			name:    "secondary_index",
			baseDDL: `CREATE TABLE t (id INT PRIMARY KEY, name VARCHAR(100) NOT NULL) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;`,
			targetDDL: `CREATE TABLE t (id INT PRIMARY KEY, name VARCHAR(100) NOT NULL,
	INDEX idx_name (name)) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;`,
			changeWant: "ADD KEY",
		},
		{
			name:    "unique_index",
			baseDDL: `CREATE TABLE t (id INT PRIMARY KEY, email VARCHAR(100) NOT NULL) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;`,
			targetDDL: `CREATE TABLE t (id INT PRIMARY KEY, email VARCHAR(100) NOT NULL,
	UNIQUE KEY uk_email (email)) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;`,
			changeWant: "UNIQUE",
		},
		{
			name: "foreign_key",
			baseDDL: `CREATE TABLE parent (id INT PRIMARY KEY) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
CREATE TABLE child (id INT PRIMARY KEY, pid INT NOT NULL) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;`,
			targetDDL: `CREATE TABLE parent (id INT PRIMARY KEY) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
CREATE TABLE child (id INT PRIMARY KEY, pid INT NOT NULL,
	CONSTRAINT fk_child_parent FOREIGN KEY (pid) REFERENCES parent (id) ON DELETE CASCADE) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;`,
			changeWant: "FOREIGN KEY",
		},
		{
			name:    "check_constraint",
			baseDDL: `CREATE TABLE t (id INT PRIMARY KEY, age INT NOT NULL) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;`,
			targetDDL: `CREATE TABLE t (id INT PRIMARY KEY, age INT NOT NULL,
	CONSTRAINT chk_age CHECK (age >= 0)) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;`,
			changeWant: "CHECK",
			skip57:     true,
		},
		{
			name:    "partition",
			baseDDL: `CREATE TABLE sales (id INT NOT NULL AUTO_INCREMENT, sale_date DATE NOT NULL, PRIMARY KEY (id, sale_date)) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;`,
			targetDDL: `CREATE TABLE sales (id INT NOT NULL AUTO_INCREMENT, sale_date DATE NOT NULL, PRIMARY KEY (id, sale_date)) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4
PARTITION BY RANGE (YEAR(sale_date)) (PARTITION p2023 VALUES LESS THAN (2024), PARTITION p_future VALUES LESS THAN MAXVALUE);`,
			changeWant: "PARTITION BY",
		},
		{
			name: "view",
			baseDDL: `CREATE TABLE t (id INT PRIMARY KEY, active BOOLEAN NOT NULL DEFAULT TRUE, title VARCHAR(100) NOT NULL) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
CREATE VIEW v AS SELECT id FROM t WHERE active = TRUE;`,
			targetDDL: `CREATE TABLE t (id INT PRIMARY KEY, active BOOLEAN NOT NULL DEFAULT TRUE, title VARCHAR(100) NOT NULL) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
CREATE VIEW v AS SELECT id, title FROM t WHERE active = TRUE;`,
			changeWant: "VIEW",
		},
		{
			name: "function",
			baseDDL: `CREATE TABLE t (id INT PRIMARY KEY) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
CREATE FUNCTION add_n(a INT) RETURNS INT DETERMINISTIC RETURN a + 1;`,
			targetDDL: `CREATE TABLE t (id INT PRIMARY KEY) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
CREATE FUNCTION add_n(a INT) RETURNS INT DETERMINISTIC RETURN a + 2;`,
			changeWant: "FUNCTION",
		},
		{
			name:    "procedure",
			baseDDL: `CREATE TABLE t (id INT PRIMARY KEY, cnt INT NOT NULL DEFAULT 0) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;`,
			targetDDL: `CREATE TABLE t (id INT PRIMARY KEY, cnt INT NOT NULL DEFAULT 0) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
CREATE PROCEDURE bump(IN tid INT) BEGIN UPDATE t SET cnt = cnt + 1 WHERE id = tid; END;`,
			changeWant: "PROCEDURE",
		},
		{
			name: "trigger",
			baseDDL: `CREATE TABLE t (id INT PRIMARY KEY, val INT NOT NULL DEFAULT 0) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
CREATE TABLE audit (id INT PRIMARY KEY AUTO_INCREMENT, tid INT, oldv INT) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
CREATE TRIGGER t_audit AFTER UPDATE ON t FOR EACH ROW BEGIN INSERT INTO audit (tid, oldv) VALUES (NEW.id, OLD.val); END;`,
			targetDDL: `CREATE TABLE t (id INT PRIMARY KEY, val INT NOT NULL DEFAULT 0) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
CREATE TABLE audit (id INT PRIMARY KEY AUTO_INCREMENT, tid INT, oldv INT) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
CREATE TRIGGER t_audit AFTER UPDATE ON t FOR EACH ROW BEGIN INSERT INTO audit (tid, oldv) VALUES (NEW.id, OLD.val + 1); END;`,
			changeWant: "TRIGGER",
		},
		{
			name:    "event",
			baseDDL: `CREATE TABLE log (id INT PRIMARY KEY AUTO_INCREMENT, d DATE) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;`,
			targetDDL: `CREATE TABLE log (id INT PRIMARY KEY AUTO_INCREMENT, d DATE) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
CREATE EVENT IF NOT EXISTS daily_ev ON SCHEDULE EVERY 1 DAY DO INSERT INTO log (d) VALUES (CURDATE());`,
			changeWant: "EVENT",
		},
	}
}

// TestSDLBreadthLive is the breadth deliverable: every MySQL breadth object type is
// round-tripped end to end through the wired declarative path (sync → MetadataToSDL →
// mysqlDiffSDLMigration) on live MySQL 5.7 and 8.0, asserting three properties per
// type:
//
//	(1) no-op idempotence — the target schema synced, dumped, and diffed against its own
//	    user DDL produces an empty migration (and source-vs-source is empty);
//	(2) minimal change — building the base schema and diffing the target yields exactly
//	    the minimal ALTER/CREATE/DROP for that object (no table rebuild);
//	(3) apply-back convergence — applying that generated DDL to the real database and
//	    re-diffing yields empty.
//
//nolint:tparallel
func TestSDLBreadthLive(t *testing.T) {
	skipUnlessLiveOracle(t)
	ctx := context.Background()

	for _, srv := range liveServers {
		srv := srv
		t.Run(srv.name, func(t *testing.T) {
			for _, bc := range breadthCases() {
				bc := bc
				if srv.version == "5.7" && bc.skip57 {
					continue
				}
				t.Run(bc.name, func(t *testing.T) {
					// (1) No-op idempotence: build the TARGET, dump, diff against the
					// target user DDL — must be empty on both versions.
					noopDB := newLiveDatabase(ctx, t, srv, "sdl_breadth_noop")
					targetSource := applyAndDump(ctx, t, srv, noopDB, bc.targetDDL)
					t.Logf("[%s/%s] target dump:\n%s", srv.name, bc.name, targetSource)

					selfDiff, err := mysqlDiffSDLMigration(targetSource, targetSource, srv.version)
					require.NoError(t, err)
					require.Empty(t, selfDiff, "[%s/%s] source-vs-source must be empty:\n%s", srv.name, bc.name, selfDiff)

					noop, err := mysqlDiffSDLMigration(targetSource, bc.targetDDL, srv.version)
					require.NoError(t, err)
					require.Empty(t, noop, "[%s/%s] no-op idempotence FAILED, residual:\n%s", srv.name, bc.name, noop)

					// (2) Minimal change: build the BASE, diff the target.
					changeDB := newLiveDatabase(ctx, t, srv, "sdl_breadth_chg")
					baseSource := applyAndDump(ctx, t, srv, changeDB, bc.baseDDL)

					change, err := mysqlDiffSDLMigration(baseSource, bc.targetDDL, srv.version)
					require.NoError(t, err)
					t.Logf("[%s/%s] change DDL:\n%s", srv.name, bc.name, change)
					require.NotEmpty(t, change, "[%s/%s] change must be non-empty", srv.name, bc.name)
					require.Contains(t, strings.ToUpper(change), bc.changeWant,
						"[%s/%s] change missing %q:\n%s", srv.name, bc.name, bc.changeWant, change)
					require.NotContains(t, strings.ToUpper(change), "DROP TABLE",
						"[%s/%s] change must not drop the table:\n%s", srv.name, bc.name, change)
					require.NotContains(t, strings.ToUpper(change), "CREATE TABLE",
						"[%s/%s] change must not rebuild the table:\n%s", srv.name, bc.name, change)

					// (3) Apply-back convergence: apply the generated DDL to the real DB,
					// re-sync, re-diff must be empty. The omni plan is op-by-op; the joined
					// blob applies cleanly because the MySQL driver's SplitSQL is BEGIN…END
					// aware, so trigger/routine bodies with internal ";" stay intact.
					applyDriver, err := createLiveMySQLDriver(ctx, srv, changeDB)
					require.NoError(t, err)
					defer applyDriver.Close(ctx)
					_, applyErr := applyDriver.Execute(ctx, change, db.ExecuteOptions{})
					require.NoError(t, applyErr, "[%s/%s] apply-back failed to apply:\n%s", srv.name, bc.name, change)

					newMetadata, err := applyDriver.SyncDBSchema(ctx)
					require.NoError(t, err)
					newSource, err := schema.MetadataToSDL(storepb.Engine_MYSQL, model.NewDatabaseMetadata(newMetadata, nil, nil, storepb.Engine_MYSQL, true))
					require.NoError(t, err)

					converge, err := mysqlDiffSDLMigration(newSource, bc.targetDDL, srv.version)
					require.NoError(t, err)
					require.Empty(t, converge, "[%s/%s] apply-back did not converge, residual:\n%s", srv.name, bc.name, converge)
				})
			}
		})
	}
}

// countAdviceCode returns how many advices carry the given code.
func countAdviceCode(advices []*storepb.Advice, c int32) int {
	n := 0
	for _, a := range advices {
		if a.Code == c {
			n++
		}
	}
	return n
}

// TestSDLDropAdvicesLive exercises schema.SDLDropAdvices (the registered MySQL drop-advice
// analyzer, reached the same way the release-check gating reaches it) end to end against
// live 5.7 and 8.0: sync the representative schema, then point the user SDL at a target that
// drops a whole table and a column, and assert WARNING advices are produced with the
// SDLDropOperation code. A no-op target must yield zero advices.
//
//nolint:tparallel
func TestSDLDropAdvicesLive(t *testing.T) {
	skipUnlessLiveOracle(t)
	ctx := context.Background()

	// Target SDL drops the `book` table entirely and drops author.bio (a column).
	dropTargetDDL := `
CREATE TABLE author (
	id INT PRIMARY KEY AUTO_INCREMENT,
	name VARCHAR(100) NOT NULL,
	active BOOLEAN NOT NULL DEFAULT TRUE,
	rating DECIMAL(5, 2) NOT NULL DEFAULT 0.00,
	created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
	INDEX idx_author_name (name)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
`

	for _, srv := range liveServers {
		srv := srv
		t.Run(srv.name, func(t *testing.T) {
			meta, _ := syncToMetadata(ctx, t, srv)

			// No-op target: representativeDDL itself must produce zero drop advices.
			noopAdvices, err := schema.SDLDropAdvices(storepb.Engine_MYSQL, representativeDDL, meta, srv.version)
			require.NoError(t, err)
			require.Empty(t, noopAdvices, "[%s] no-op target must yield no drop advices, got: %+v", srv.name, noopAdvices)

			// Destructive target: book table dropped (active_books view depends on it, so the
			// view is dropped too) and author.bio column dropped.
			advices, err := schema.SDLDropAdvices(storepb.Engine_MYSQL, dropTargetDDL, meta, srv.version)
			require.NoError(t, err)
			require.NotEmpty(t, advices, "[%s] destructive target must yield drop advices", srv.name)

			for _, a := range advices {
				require.Equal(t, storepb.Advice_WARNING, a.Status, "[%s] drop advice must be WARNING: %+v", srv.name, a)
			}

			dropCount := countAdviceCode(advices, code.SDLDropOperation.Int32())
			require.GreaterOrEqual(t, dropCount, 2,
				"[%s] expected at least DROP TABLE book + DROP COLUMN bio warnings, got: %+v", srv.name, advices)

			// The dropped table and column must be named somewhere in the advice content.
			joined := ""
			for _, a := range advices {
				joined += a.Content + "\n"
			}
			require.Contains(t, joined, "book", "[%s] expected the dropped table named, got:\n%s", srv.name, joined)
			require.Contains(t, joined, "bio", "[%s] expected the dropped column named, got:\n%s", srv.name, joined)
		})
	}
}
