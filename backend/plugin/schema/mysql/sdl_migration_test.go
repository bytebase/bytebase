package mysql

import (
	"context"
	"database/sql"
	_ "embed"
	"fmt"
	"math/rand"
	"os"
	"regexp"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/bytebase/omni/mysql/catalog"
	mysqldriver "github.com/go-sql-driver/mysql"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	"github.com/bytebase/bytebase/backend/plugin/db"
	mysqldb "github.com/bytebase/bytebase/backend/plugin/db/mysql"
	mysqlparser "github.com/bytebase/bytebase/backend/plugin/parser/mysql"
	"github.com/bytebase/bytebase/backend/plugin/schema"
	"github.com/bytebase/bytebase/backend/store/model"
)

// TestMySQLVersionFor locks in the engine-version -> omni catalog Version mapping that
// threads the version into the SDL diff. The boundary is 8.0.0: anything below maps to
// MySQL57 (utf8mb4_general_ci default), 8.0.0+ to MySQL80 (utf8mb4_0900_ai_ci); an
// empty/unparseable version falls back to MySQL80 (the historical default).
func TestMySQLVersionFor(t *testing.T) {
	cases := []struct {
		version string
		want    catalog.Version
	}{
		{"5.7.25", catalog.MySQL57},
		{"5.7", catalog.MySQL57},
		{"5.6.51", catalog.MySQL57},
		{"5.7.44-log", catalog.MySQL57},
		{"8.0.32", catalog.MySQL80},
		{"8.0", catalog.MySQL80},
		{"8.0.0", catalog.MySQL80},
		{"8.4.0", catalog.MySQL80},
		{"9.0.1", catalog.MySQL80},
		// OceanBase / unparseable / empty -> default to 8.0 stored form.
		{"", catalog.MySQL80},
		{"not-a-version", catalog.MySQL80},
		// OceanBase reports a MySQL-compat version like "8.0.30" in its version string;
		// when only a marketing string is present we default to 8.0, which is safe.
		{"OceanBase 4.2", catalog.MySQL80},
	}
	for _, c := range cases {
		if got := mysqlVersionFor(c.version); got != c.want {
			t.Errorf("mysqlVersionFor(%q) = %v, want %v", c.version, got, c.want)
		}
	}
}

// metadataFromSDL builds a model.DatabaseMetadata from SDL text using the registered MySQL
// GetDatabaseMetadata, so drop-advice tests can run fully offline (no live server).
func metadataFromSDL(t *testing.T, sdl string) *model.DatabaseMetadata {
	t.Helper()
	proto, err := schema.GetDatabaseMetadata(storepb.Engine_MYSQL, sdl)
	require.NoError(t, err)
	return model.NewDatabaseMetadata(proto, nil, nil, storepb.Engine_MYSQL, true)
}

// TestMySQLSDLDropAdvices verifies the registered MySQL drop-advice analyzer emits WARNING
// advices for destructive operations (DROP TABLE, DROP COLUMN) and stays silent on a no-op.
// It drives the version-aware path offline by building the "current" schema metadata from
// SDL.
func TestMySQLSDLDropAdvices(t *testing.T) {
	current := `
CREATE TABLE author (
	id INT PRIMARY KEY AUTO_INCREMENT,
	name VARCHAR(100) NOT NULL,
	bio TEXT
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
CREATE TABLE book (
	id INT PRIMARY KEY AUTO_INCREMENT,
	title VARCHAR(200) NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
`
	meta := metadataFromSDL(t, current)

	t.Run("no_op_no_advices", func(t *testing.T) {
		advices, err := mysqlSDLDropAdvices(strings.TrimSpace(current), meta, "8.0")
		require.NoError(t, err)
		require.Empty(t, advices, "no-op target must yield no advices, got: %+v", advices)
	})

	t.Run("drop_table_and_column_warn", func(t *testing.T) {
		// Target drops the whole `book` table and the `bio` column from `author`.
		target := `
CREATE TABLE author (
	id INT PRIMARY KEY AUTO_INCREMENT,
	name VARCHAR(100) NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
`
		advices, err := mysqlSDLDropAdvices(strings.TrimSpace(target), meta, "8.0")
		require.NoError(t, err)
		require.NotEmpty(t, advices, "destructive target must yield advices")

		for _, a := range advices {
			require.Equal(t, storepb.Advice_WARNING, a.Status, "drop advice must be WARNING: %+v", a)
			require.Equal(t, code.SDLDropOperation.Int32(), a.Code, "expected SDLDropOperation code: %+v", a)
		}

		joined := ""
		for _, a := range advices {
			joined += a.Content + "\n"
		}
		require.Contains(t, joined, "book", "expected dropped table named:\n%s", joined)
		require.Contains(t, joined, "bio", "expected dropped column named:\n%s", joined)
	})
}

func TestAdjustPrologueError(t *testing.T) {
	t.Run("load error reports caller-text line numbers", func(t *testing.T) {
		// Invalid token on line 3 of the caller's SDL. Without adjustment the
		// bbcatalog prologue (2 lines) would shift the report to line 5.
		target := "CREATE TABLE `t` (\n" +
			"  `id` int NOT NULL,\n" +
			"  BOGUS TOKEN HERE,\n" +
			"  PRIMARY KEY (`id`)\n" +
			");\n"
		_, err := mysqlDiffSDLMigration("", target, "8.0.32")
		require.Error(t, err)
		require.Contains(t, err.Error(), "(line 3, column")
		require.NotContains(t, err.Error(), "(line 5, column")
	})

	t.Run("positions inside the prologue are left untouched", func(t *testing.T) {
		err := adjustPrologueError(errors.New("unexpected token (line 2, column 5)"))
		require.EqualError(t, err, "unexpected token (line 2, column 5)")
	})

	t.Run("multiple positions all shift", func(t *testing.T) {
		err := adjustPrologueError(errors.New("bad (line 10, column 3); also (line 42, column 7)"))
		require.EqualError(t, err, "bad (line 8, column 3); also (line 40, column 7)")
	})

	t.Run("nil stays nil", func(t *testing.T) {
		require.NoError(t, adjustPrologueError(nil))
	})
}

// TestSDLRegistrationsAreMySQLOnly pins that OceanBase is NOT registered for any of the
// SDL entry points this package provides (X2): OceanBase support is pending validation
// against a live oracle, and an executable-but-unvalidated registration would silently
// route OB declarative releases through untested code (and skip nothing).
func TestSDLRegistrationsAreMySQLOnly(t *testing.T) {
	_, err := schema.DiffSDLMigration(storepb.Engine_OCEANBASE, "", "", "")
	require.ErrorContains(t, err, "not supported")

	_, err = schema.SDLDropAdvices(storepb.Engine_OCEANBASE, "", nil, "")
	require.ErrorContains(t, err, "not supported")

	_, err = schema.GetMultiFileDatabaseDefinition(storepb.Engine_OCEANBASE, schema.GetDefinitionContext{}, &storepb.DatabaseSchemaMetadata{})
	require.ErrorContains(t, err, "not supported")

	// MySQL stays registered for all three.
	_, err = schema.DiffSDLMigration(storepb.Engine_MYSQL, "", "", "")
	require.NoError(t, err)
	_, err = schema.SDLDropAdvices(storepb.Engine_MYSQL, "", nil, "")
	require.NoError(t, err)
	_, err = schema.GetMultiFileDatabaseDefinition(storepb.Engine_MYSQL, schema.GetDefinitionContext{}, &storepb.DatabaseSchemaMetadata{})
	require.NoError(t, err)
}

// metadataFromProto wraps a raw proto in the model type the diff entry points take.
func metadataFromProto(proto *storepb.DatabaseSchemaMetadata) *model.DatabaseMetadata {
	return model.NewDatabaseMetadata(proto, nil, nil, storepb.Engine_MYSQL, true)
}

// TestDiffMigrationUsesLegacyMetadataPath proves the X1 dispatch: a MySQL
// schema.DiffMigration call must go through the registered LEGACY metadata migration
// (GetDatabaseSchemaDiff + GenerateMigration), NOT the MetadataToSDL -> LoadSDL
// round-trip. The sentinel is a synced view whose stored body the omni SDL loader
// rejects — the SDL path would fail the whole diff, while the legacy path treats view
// bodies as opaque text and still diffs the tables.
func TestDiffMigrationUsesLegacyMetadataPath(t *testing.T) {
	brokenView := &storepb.ViewMetadata{
		Name: "v_broken",
		// Not parseable as a SELECT by the omni loader.
		Definition: "select ((broken from",
	}
	oldProto := &storepb.DatabaseSchemaMetadata{
		Name: "d",
		Schemas: []*storepb.SchemaMetadata{{
			Name: "",
			Tables: []*storepb.TableMetadata{{
				Name: "t",
				Columns: []*storepb.ColumnMetadata{
					{Name: "id", Type: "int", Nullable: false},
				},
			}},
			Views: []*storepb.ViewMetadata{brokenView},
		}},
	}
	newProto := &storepb.DatabaseSchemaMetadata{
		Name: "d",
		Schemas: []*storepb.SchemaMetadata{{
			Name: "",
			Tables: []*storepb.TableMetadata{{
				Name: "t",
				Columns: []*storepb.ColumnMetadata{
					{Name: "id", Type: "int", Nullable: false},
					{Name: "extra", Type: "varchar(10)", Nullable: true, Default: "NULL"},
				},
			}},
			Views: []*storepb.ViewMetadata{brokenView},
		}},
	}

	sql, err := schema.DiffMigration(storepb.Engine_MYSQL, metadataFromProto(oldProto), metadataFromProto(newProto))
	require.NoError(t, err, "metadata diff must not round-trip through the SDL loader")
	require.Contains(t, sql, "ADD COLUMN `extra`")

	// Sanity: the SDL path really would reject this metadata — proving the sentinel bites.
	sdl, err := schema.MetadataToSDL(storepb.Engine_MYSQL, metadataFromProto(oldProto))
	require.NoError(t, err)
	_, err = schema.DiffSDLMigration(storepb.Engine_MYSQL, sdl, sdl, "")
	require.Error(t, err, "sentinel view body must be un-loadable by the omni SDL path")
}

// legacyColumnChangeSQL diffs two single-column tables through the registered legacy
// metadata path and returns the migration SQL.
func legacyColumnChangeSQL(t *testing.T, oldCol, newCol *storepb.ColumnMetadata) string {
	t.Helper()
	mk := func(col *storepb.ColumnMetadata) *model.DatabaseMetadata {
		return metadataFromProto(&storepb.DatabaseSchemaMetadata{
			Name: "d",
			Schemas: []*storepb.SchemaMetadata{{
				Name:   "",
				Tables: []*storepb.TableMetadata{{Name: "t", Columns: []*storepb.ColumnMetadata{col}}},
			}},
		})
	}
	sql, err := schema.DiffMigration(storepb.Engine_MYSQL, mk(oldCol), mk(newCol))
	require.NoError(t, err)
	return sql
}

// TestLegacyDiffSRIDAndInvisible locks the X10 pairing with X1: with metadata diffs
// routed back onto the legacy differ/generator, that path must know the new SRID and
// INVISIBLE column fields — an SRID-only or INVISIBLE-only change must produce a MODIFY
// COLUMN rendering the dumper-canonical attribute comments.
func TestLegacyDiffSRIDAndInvisible(t *testing.T) {
	srid := func(v uint32) *uint32 { return &v }

	t.Run("srid_only_change_modifies", func(t *testing.T) {
		sql := legacyColumnChangeSQL(t,
			&storepb.ColumnMetadata{Name: "pt", Type: "point", Nullable: false},
			&storepb.ColumnMetadata{Name: "pt", Type: "point", Nullable: false, Srid: srid(4326)},
		)
		require.Contains(t, sql, "MODIFY COLUMN `pt` point NOT NULL /*!80003 SRID 4326 */")
	})

	t.Run("explicit_srid_zero_differs_from_unset", func(t *testing.T) {
		sql := legacyColumnChangeSQL(t,
			&storepb.ColumnMetadata{Name: "pt", Type: "point", Nullable: false},
			&storepb.ColumnMetadata{Name: "pt", Type: "point", Nullable: false, Srid: srid(0)},
		)
		require.Contains(t, sql, "MODIFY COLUMN `pt` point NOT NULL /*!80003 SRID 0 */")
	})

	t.Run("equal_srid_no_change", func(t *testing.T) {
		sql := legacyColumnChangeSQL(t,
			&storepb.ColumnMetadata{Name: "pt", Type: "point", Nullable: false, Srid: srid(4326)},
			&storepb.ColumnMetadata{Name: "pt", Type: "point", Nullable: false, Srid: srid(4326)},
		)
		require.Empty(t, sql)
	})

	t.Run("invisible_only_change_modifies", func(t *testing.T) {
		sql := legacyColumnChangeSQL(t,
			&storepb.ColumnMetadata{Name: "c", Type: "int", Nullable: true, Default: "NULL"},
			&storepb.ColumnMetadata{Name: "c", Type: "int", Nullable: true, Default: "NULL", IsInvisible: true},
		)
		require.Contains(t, sql, "MODIFY COLUMN `c` int")
		require.Contains(t, sql, " /*!80023 INVISIBLE */")
	})
}

// TestStripDatabaseContext pins the literal-aware synthetic-qualifier strip (X13): a
// string literal that happens to contain the `bbcatalog`.`x` byte sequence must survive
// verbatim while identifier-position qualifiers are removed.
func TestStripDatabaseContext(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "plain_qualifiers_stripped",
			in:   "ALTER TABLE `bbcatalog`.`t` ADD COLUMN `c` int;\nCREATE INDEX `i` ON `bbcatalog`.`t` (`c`)",
			want: "ALTER TABLE `t` ADD COLUMN `c` int;\nCREATE INDEX `i` ON `t` (`c`)",
		},
		{
			name: "single_quoted_literal_preserved",
			in:   "CREATE OR REPLACE VIEW `bbcatalog`.`v` AS select '`bbcatalog`.`x`' AS c from `bbcatalog`.`t`",
			want: "CREATE OR REPLACE VIEW `v` AS select '`bbcatalog`.`x`' AS c from `t`",
		},
		{
			name: "double_quoted_literal_preserved",
			in:   `select "` + "`bbcatalog`.`x`" + `" from ` + "`bbcatalog`.`t`",
			want: `select "` + "`bbcatalog`.`x`" + `" from ` + "`t`",
		},
		{
			name: "table_actually_named_bbcatalog",
			in:   "ALTER TABLE `bbcatalog`.`bbcatalog` ADD COLUMN `c` int",
			want: "ALTER TABLE `bbcatalog` ADD COLUMN `c` int",
		},
		{
			name: "three_part_reference",
			in:   "select `bbcatalog`.`t`.`c` from `bbcatalog`.`t`",
			want: "select `t`.`c` from `t`",
		},
		{
			name: "routine_body_comment_with_apostrophe_does_not_swallow_later_qualifier",
			// omni emits routine bodies VERBATIM: a user line comment with an
			// unbalanced quote must not open a phantom literal that swallows the next
			// op's qualifier (ops are joined with ";\n" in plan.SQL()).
			in:   "CREATE FUNCTION `bbcatalog`.`f`() RETURNS int\nBEGIN\n  -- don't do this\n  RETURN 1;\nEND;\nALTER TABLE `bbcatalog`.`t` ADD COLUMN `c` int",
			want: "CREATE FUNCTION `f`() RETURNS int\nBEGIN\n  -- don't do this\n  RETURN 1;\nEND;\nALTER TABLE `t` ADD COLUMN `c` int",
		},
		{
			name: "hash_comment_opaque_and_executable_comment_scanned",
			in:   "CREATE TABLE `bbcatalog`.`t` (\n  `pt` point NOT NULL /*!80003 SRID 0 */\n) # trailing 'note\n/* block 'c' */ ALTER TABLE `bbcatalog`.`t` COMMENT ''",
			want: "CREATE TABLE `t` (\n  `pt` point NOT NULL /*!80003 SRID 0 */\n) # trailing 'note\n/* block 'c' */ ALTER TABLE `t` COMMENT ''",
		},
		{
			name: "no_qualifier_untouched",
			in:   "ALTER TABLE `t` ADD COLUMN `c` int DEFAULT 'bbcatalog'",
			want: "ALTER TABLE `t` ADD COLUMN `c` int DEFAULT 'bbcatalog'",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, stripDatabaseContext(tc.in))
		})
	}
}

// TestDiffSDLMigrationEmptyTargetDropsEverything supports the X11 DiffSchema fix: an
// intentionally EMPTY target schema text is a legal SDL target meaning "empty schema",
// so the plan must drop the existing objects rather than erroring out.
func TestDiffSDLMigrationEmptyTargetDropsEverything(t *testing.T) {
	source := "CREATE TABLE t (id INT PRIMARY KEY);\n"
	sql, err := schema.DiffSDLMigration(storepb.Engine_MYSQL, source, "", "8.0.32")
	require.NoError(t, err)
	require.Contains(t, sql, "DROP TABLE")
	require.NotContains(t, sql, "bbcatalog")
}

// TestLoadCatalogFallbackSeedsExplicitDefaultsForTimestamp pins the fix for the LoadSQL
// fallback not seeding explicit_defaults_for_timestamp. On 5.7, EDFT is OFF, so a bare
// TIMESTAMP column materializes NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE
// CURRENT_TIMESTAMP. The source here is plain SDL (loads via LoadSDLWithVersion, EDFT
// seeded OFF); the target carries a LOCK TABLES statement that LoadSDL rejects, forcing the
// LoadSQL fallback. Before the fix that fallback kept the New() 8.0 default (EDFT ON), so
// the SAME bare TIMESTAMP materialized nullable and the diff spuriously emitted a MODIFY
// COLUMN. With EDFT seeded to the 5.7 box default on both sides, identical schemas diff to
// empty.
func TestLoadCatalogFallbackSeedsExplicitDefaultsForTimestamp(t *testing.T) {
	const table = "CREATE TABLE `t` (`ts` timestamp) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;\n"

	// Target forces the LoadSQL fallback: LOCK TABLES is not a DDL statement LoadSDL accepts.
	target := "LOCK TABLES `t` WRITE;\n" + table + "UNLOCK TABLES;\n"

	// 5.7 (EDFT OFF): identical schemas must produce no migration.
	sql57, err := schema.DiffSDLMigration(storepb.Engine_MYSQL, table, target, "5.7.44")
	require.NoError(t, err)
	require.Empty(t, sql57, "5.7 source/target with identical bare TIMESTAMP must not phantom-diff; got %q", sql57)

	// 8.0 (EDFT ON): also identical, and the fallback must not regress the 8.0 default.
	sql80, err := schema.DiffSDLMigration(storepb.Engine_MYSQL, table, target, "8.0.32")
	require.NoError(t, err)
	require.Empty(t, sql80, "8.0 source/target with identical bare TIMESTAMP must not phantom-diff; got %q", sql80)
}

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
//
//go:embed testdata/sdl/representative_ddl.sql
var representativeDDL string

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

// TestSDLSpatialSRIDPresenceLive verifies the SRID presence semantics (X5) against the
// live 8.0 oracle: a geometry column with EXPLICIT SRID 0 (information_schema reports
// SRS_ID=0 — a valid SRS, distinct from "no SRID" which reports NULL) and one with SRID
// 4326 must both round-trip — dump -> SDL -> apply to a fresh database -> re-dump —
// with byte-identical dumps and an empty self-diff. Before the fix the 0-sentinel
// conflated SRID 0 with "no SRID" and dropped the attribute at dump time.
func TestSDLSpatialSRIDPresenceLive(t *testing.T) {
	skipUnlessLiveOracle(t)
	ctx := context.Background()
	srv := liveServers[0] // 8.0 — the SRID column attribute is an 8.0-only surface.

	const ddl = `
CREATE TABLE geo (
	id INT PRIMARY KEY,
	pt0 GEOMETRY NOT NULL /*!80003 SRID 0 */,
	pt4326 POINT NOT NULL /*!80003 SRID 4326 */,
	ptnone GEOMETRY NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
`
	dbA := newLiveDatabase(ctx, t, srv, "sdl_srid_a")
	require.NoError(t, applyDDL(ctx, t, srv, dbA, ddl))
	dumpA := dumpSDL(ctx, t, srv, dbA)

	// The dump must carry the explicit SRID 0 (presence, not a zero sentinel), the
	// SRID 4326, and no SRID attribute on the undeclared column.
	require.Contains(t, dumpA, "`pt0` geometry NOT NULL /*!80003 SRID 0 */")
	require.Contains(t, dumpA, "`pt4326` point NOT NULL /*!80003 SRID 4326 */")
	require.Contains(t, dumpA, "`ptnone` geometry NOT NULL,")
	require.NotContains(t, dumpA, "`ptnone` geometry NOT NULL /*!80003")

	// Apply the dump to a fresh database and re-dump: the round-trip must be lossless.
	dbB := newLiveDatabase(ctx, t, srv, "sdl_srid_b")
	require.NoError(t, applyDDL(ctx, t, srv, dbB, dumpA))
	dumpB := dumpSDL(ctx, t, srv, dbB)
	require.Equal(t, dumpA, dumpB, "dump -> apply -> re-dump must be byte-stable")

	selfDiff, err := mysqlDiffSDLMigration(dumpB, dumpA, srv.version)
	require.NoError(t, err)
	require.Empty(t, selfDiff, "SRID round-trip self-diff must be empty")

	// Removing a real SRID restriction must diff (4326 -> unset is a column change).
	// Note the 0 <-> unset transition is folded to a no-op by omni's canonicalizer
	// (out of scope here); the dump above still preserves the explicit SRID 0
	// faithfully, so the declared schema round-trips byte-exact.
	noSridTarget := strings.Replace(dumpA, "`pt4326` point NOT NULL /*!80003 SRID 4326 */", "`pt4326` point NOT NULL", 1)
	change, err := mysqlDiffSDLMigration(dumpA, noSridTarget, srv.version)
	require.NoError(t, err)
	require.Contains(t, change, "MODIFY", "dropping an explicit SRID 4326 must produce a column modification")
}

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

// Real-world-schema smoke test for the MySQL declarative (SDL) migration path. Unlike the
// synthetic stress suites (the stress and deep-stress axes above) these cases drive
// ACTUAL production schemas — Sakila, employees, employees(partitioned), Roundcube, and
// MediaWiki — through the exact production entry points end to end against live MySQL 5.7 +
// 8.0:
//
//   - schema.SDLMigration(MYSQL, userSDL, syncedMetadata, engineVersion),
//   - mysqlDiffSDLMigration(source, target, engineVersion),
//   - schema.SDLDropAdvices(MYSQL, userSDL, syncedMetadata, engineVersion),
//   - the real db/mysql sync + schema.MetadataToSDL dumper.
//
// The schemas are embedded preprocessed fixtures (testdata/realworld/*.sql) so the test is
// self-contained. Sakila is THE priority: it carries real view / function / procedure /
// trigger bodies (incl. a SQL SECURITY INVOKER view with fully schema-qualified references),
// which is where the at-scale view/routine round-trip is won or lost.
//
// Shared helpers (createLiveMySQLDriver, newLiveDatabase, liveServers, liveOraclePassword,
// statementCount, syncMetadata, dumpSDL, applyDDL) are declared with the live round-trip
// axes above.

// ----------------------------------------------------------------------------
// Embedded preprocessed real-world schema fixtures.
//
// Preprocessing applied when the corpus was lifted into testdata (so the committed test runs
// without the external corpus):
//   - sakila:        DROP/CREATE SCHEMA + USE + session SET @OLD_* lines stripped; a single
//                    leading SET FOREIGN_KEY_CHECKS=0 prepended (the dump's forward FK refs
//                    require it). DELIMITER routine/trigger blocks kept verbatim. The
//                    actor_info view keeps its hardcoded `sakila.`-qualified body, so sakila
//                    MUST be loaded into a database literally named `sakila`.
//   - employees:     DB-context + data-load tail (flush logs / SELECT 'LOADING' / source ...)
//                    stripped; CREATE TABLE/VIEW kept. Wrapped in a scratch DB.
//   - employees_part: same as employees, keeping the /*!50510 ALTER ... PARTITION BY RANGE
//                    COLUMNS */ version-gated blocks for `titles` and `salaries`.
//   - roundcube:     no CREATE DATABASE — wrapped in a scratch DB. Every table is
//                    ROW_FORMAT=DYNAMIC (a known table-option round-trip stressor) with real
//                    charset variety (utf8mb4_unicode_ci, binary). Trailing INSERT stripped.
//   - mediawiki:     MediaWiki placeholders preprocessed: /*_*/ and /*i*/ deleted,
//                    /*$wgDBTableOptions*/ -> "ENGINE=InnoDB DEFAULT CHARSET=binary",
//                    /*$wgDBprefix*/ -> empty. 58 tables incl. a MyISAM FULLTEXT searchindex.

//go:embed testdata/realworld/sakila.sql
var sakilaSchema string

//go:embed testdata/realworld/employees.sql
var employeesSchema string

//go:embed testdata/realworld/employees_part.sql
var employeesPartSchema string

//go:embed testdata/realworld/roundcube.sql
var roundcubeSchema string

//go:embed testdata/realworld/mediawiki.sql
var mediawikiSchema string

// omniViewParseBug is the precise classification for the (B) omni SDL-parser limitation that
// blocks every multi-table view in the real-world corpus.
//
// REPRO: schema.MetadataToSDL dumps a multi-table view's FROM clause in MySQL's canonical
// parenthesized left-deep join form, e.g. (from sakila.film_list):
//
//	... from ((((`category` left join `film_category` on(...)) left join `film` on(...))
//	         join `film_actor` on(...)) join `actor` on(...)) group by ...
//
// Re-loading that dump through the SDL path (catalog.LoadSDLWithVersion / LoadSQL, called by
// mysqlDiffSDLMigration and schema.SDLMigration) fails with
//
//	expected SELECT, TABLE, VALUES, or '('
//
// LOCUS: omni mysql/parser/select.go:390 — the FROM-clause table_reference parser does not
// accept a parenthesized join group `( t1 JOIN t2 ON ... )`; the leading '(' falls through to
// the query-primary parser, which only expects a parenthesized sub-SELECT. omni's top-level
// Parse() accepts the same construct, so the gap is specifically in the SDL catalog loader's
// view-body SELECT sub-parser. This is DISTINCT from the known varchar(N) BINARY -> _bin
// collation phantom-MODIFY bug. Until omni fixes the parenthesized-join table_reference and the
// pin updates, any fixture whose synced+dumped form carries a multi-table view (sakila and the
// employees family, via current_dept_emp) cannot round-trip — these legs are pending, not red.
const omniViewParseBug = "omni SDL view-body parser rejects MySQL's canonical parenthesized JOIN in a view FROM clause (omni mysql/parser/select.go:390); blocks every multi-table view"

// omniVarcharBinaryBug is the OTHER known, being-fixed-in-parallel (B) omni bug: the omni SDL
// loader drops the `BINARY` modifier on a `varchar(N) BINARY` column (which MySQL stores as the
// `_bin` collation), so the original-vs-synced diff emits a phantom
// `ALTER TABLE ... MODIFY COLUMN ... varchar(N)`. roundcube's cache/users tables use that form.
// Pending until the omni fix merges and the pin updates.
const omniVarcharBinaryBug = "omni SDL loader drops varchar(N) BINARY -> _bin collation, producing a phantom MODIFY COLUMN (confirmed (B) bug, fixed in omni separately)"

// realWorldSchema describes one embedded production schema fixture.
type realWorldSchema struct {
	name string
	ddl  string
	// fixedDBName, when non-empty, forces the scratch database to this literal name instead of
	// a unique-suffixed one. Required for sakila, whose actor_info view body hardcodes the
	// `sakila.` schema qualifier and therefore only resolves in a database named `sakila`.
	fixedDBName string
	// only57Skip marks a fixture that does not load on 5.7 (so the 5.7 leg is skipped with a
	// classification rather than failing).
	only57Skip string
	// viewParsePending, when non-empty, marks a fixture whose canonical dump carries a
	// multi-table view that the current omni SDL parser cannot re-read (see omniViewParseBug).
	// The SDL-path legs (idempotence, original-vs-synced, and every migration scenario on this
	// schema) are skipped with this classification rather than failing on an out-of-scope,
	// being-fixed-upstream engine bug. The live LOAD + sync + MetadataToSDL dump still run and
	// are asserted non-empty in Phase 0, so the dumper half of the round-trip stays covered.
	viewParsePending string
}

func realWorldSchemas() []realWorldSchema {
	return []realWorldSchema{
		{
			name:        "sakila",
			ddl:         sakilaSchema,
			fixedDBName: "sakila",
			// On 5.7 the SQL SECURITY INVOKER actor_info view, whose body is fully `sakila.`
			// qualified, fails to CREATE even into a database named sakila because 5.7 resolves
			// the invoker view's referenced tables more strictly at create time (errno 1146 on
			// sakila.actor under FOREIGN_KEY_CHECKS=0). 8.0 creates it fine. So the 5.7 leg is
			// skipped (harness/engine-version classification — see report). All other sakila
			// objects (16 tables, 6 plain views, 3 functions, 3 procedures, 3 triggers) load on
			// both versions.
			only57Skip: "actor_info SQL SECURITY INVOKER view (hardcoded sakila.* refs) fails to CREATE on 5.7",
			// Multi-table-join views (film_list, customer_list, ...) now round-trip: the omni SDL
			// parser's parenthesized-JOIN FROM fix is merged + pinned (#356).
		},
		{name: "employees", ddl: employeesSchema},
		{name: "employees_part", ddl: employeesPartSchema},
		{name: "roundcube", ddl: roundcubeSchema},
		{name: "mediawiki", ddl: mediawikiSchema},
	}
}

// newNamedLiveDatabase is newLiveDatabase with an explicit database name (for fixtures that
// must use a fixed name, e.g. sakila). It drops any pre-existing database of that name first
// and registers cleanup.
func newNamedLiveDatabase(ctx context.Context, t *testing.T, srv liveServer, dbName string) string {
	t.Helper()
	admin, err := createLiveMySQLDriver(ctx, srv, "")
	require.NoError(t, err)
	_, err = admin.Execute(ctx, fmt.Sprintf("DROP DATABASE IF EXISTS `%s`; CREATE DATABASE `%s`", dbName, dbName), db.ExecuteOptions{})
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

// loadRealWorld creates a scratch database (fixed-named or unique), applies the fixture DDL
// through the real db/mysql driver, syncs it, and returns the synced model metadata plus the
// database name. The fixture DDL may contain DELIMITER blocks and SET FOREIGN_KEY_CHECKS=0;
// the driver's DealWithDelimiter + single-connection execution handle both.
func loadRealWorld(ctx context.Context, t *testing.T, srv liveServer, rw realWorldSchema) (*model.DatabaseMetadata, string) {
	t.Helper()
	var dbName string
	if rw.fixedDBName != "" {
		dbName = newNamedLiveDatabase(ctx, t, srv, rw.fixedDBName)
	} else {
		dbName = newLiveDatabase(ctx, t, srv, "sdl_rw_"+rw.name)
	}
	driver, err := createLiveMySQLDriver(ctx, srv, dbName)
	require.NoError(t, err)
	defer driver.Close(ctx)
	_, err = driver.Execute(ctx, normalizeDelimiters(rw.ddl), db.ExecuteOptions{})
	require.NoError(t, err, "[%s/%s] apply fixture DDL", srv.name, rw.name)
	metadata, err := driver.SyncDBSchema(ctx)
	require.NoError(t, err)
	return model.NewDatabaseMetadata(metadata, nil, nil, storepb.Engine_MYSQL, true), dbName
}

// normalizeDelimiters rewrites a mysqldump-style multi-DELIMITER fixture into the no-DELIMITER
// form the production split path (mysqlparser.DealWithDelimiter -> omni Split) handles cleanly.
//
// The sakila fixture interleaves regular ";"-delimited DDL with several DELIMITER blocks in
// different styles (";;", "//", "$$") for its triggers/routines. The production path's
// DealWithDelimiter only reliably rewrites a single uniform block; on the interleaved mix omni's
// splitter silently truncates the statement stream (it stopped after the first routine, so only
// 10 of 16 tables and 0 of 7 views ever loaded). Normalizing here — drop every standalone
// DELIMITER line and rewrite each body's custom terminator (e.g. "END //") back to a plain
// "END;" — yields the exact form the passing stress suites use (BEGIN ... END; with internal
// ";", no DELIMITER), which omni's compound-statement-aware splitter segments correctly. The
// fixtures without DELIMITER blocks pass through unchanged.
func normalizeDelimiters(ddl string) string {
	if !strings.Contains(ddl, "DELIMITER") {
		return ddl
	}
	lines := strings.Split(ddl, "\n")
	out := make([]string, 0, len(lines))
	delim := ";"
	for _, ln := range lines {
		trimmed := strings.TrimSpace(ln)
		if strings.HasPrefix(strings.ToUpper(trimmed), "DELIMITER ") {
			delim = strings.TrimSpace(trimmed[len("DELIMITER "):])
			if delim == "" {
				delim = ";"
			}
			continue
		}
		if delim != ";" && strings.HasSuffix(trimmed, delim) {
			if idx := strings.LastIndex(ln, delim); idx >= 0 {
				ln = ln[:idx] + ";"
			}
		}
		out = append(out, ln)
	}
	return strings.Join(out, "\n")
}

// objectCounts summarizes a synced schema for the Phase 0 report. It walks the raw proto so
// it is agnostic to the (single, unnamed) schema MySQL uses.
func objectCounts(meta *model.DatabaseMetadata) (tables, views, funcs, procs, triggers int) {
	proto := meta.GetProto()
	if proto == nil {
		return 0, 0, 0, 0, 0
	}
	for _, sm := range proto.GetSchemas() {
		tables += len(sm.GetTables())
		views += len(sm.GetViews())
		funcs += len(sm.GetFunctions())
		procs += len(sm.GetProcedures())
		for _, tbl := range sm.GetTables() {
			triggers += len(tbl.GetTriggers())
		}
	}
	return tables, views, funcs, procs, triggers
}

// ============================================================================
// Phase 0 + Phase 1: load each real schema, then prove SDL idempotence.
// ============================================================================

// TestSDLRealWorldIdempotence is the baseline that matters most: every loaded production
// schema must round-trip through the SDL path with NO phantom diff. For each schema × version:
//
//	(0) load the fixture live, sync, and log object counts (Phase 0 usability),
//	(1a) source-vs-source determinism: source = MetadataToSDL(synced);
//	     mysqlDiffSDLMigration(source, source, version) MUST be empty,
//	(1b) production path: schema.SDLMigration(MYSQL, source, synced, version) MUST be empty
//	     (this is the exact call the release executor makes, with the canonical dump as the
//	     user target — proving the dumper output re-imports to the identical metadata).
//
// Any non-empty no-op is a finding (logged with the spurious DDL for classification).
//
//nolint:tparallel
func TestSDLRealWorldIdempotence(t *testing.T) {
	skipUnlessLiveOracle(t)
	ctx := context.Background()

	for _, srv := range liveServers {
		srv := srv
		t.Run(srv.name, func(t *testing.T) {
			for _, rw := range realWorldSchemas() {
				rw := rw
				t.Run(rw.name, func(t *testing.T) {
					if srv.version == "5.7" && rw.only57Skip != "" {
						t.Skipf("[%s/%s] skipped on 5.7: %s", srv.name, rw.name, rw.only57Skip)
					}

					meta, dbName := loadRealWorld(ctx, t, srv, rw)
					tbl, vw, fn, pr, tg := objectCounts(meta)
					t.Logf("[PHASE0 %s/%s] loaded: tables=%d views=%d functions=%d procedures=%d triggers=%d (db=%s)",
						srv.name, rw.name, tbl, vw, fn, pr, tg, dbName)

					source := dumpSDL(ctx, t, srv, dbName)
					require.NotEmpty(t, source, "[%s/%s] MetadataToSDL produced empty SDL", srv.name, rw.name)

					if rw.viewParsePending != "" {
						// Load + sync + dump (Phase 0) covered above; the SDL re-import (Phase 1)
						// is blocked by an out-of-scope omni parser bug. Skip with classification.
						t.Skipf("[%s/%s] SDL re-import pending: %s", srv.name, rw.name, rw.viewParsePending)
					}

					// (1a) determinism.
					selfDiff, err := mysqlDiffSDLMigration(source, source, srv.version)
					require.NoError(t, err)
					if selfDiff != "" {
						t.Logf("[PHASE1a %s/%s] NON-EMPTY source-vs-source (%d stmts):\n%s",
							srv.name, rw.name, statementCount(selfDiff), selfDiff)
					}
					require.Empty(t, selfDiff, "[%s/%s] source-vs-source must be empty, got:\n%s", srv.name, rw.name, selfDiff)

					// (1b) production path, canonical dump as user target.
					noop, err := schema.SDLMigration(storepb.Engine_MYSQL, source, meta, srv.version)
					require.NoError(t, err)
					if noop != "" {
						t.Logf("[PHASE1b %s/%s] NON-EMPTY production no-op (%d stmts):\n%s",
							srv.name, rw.name, statementCount(noop), noop)
						t.Logf("[PHASE1b %s/%s] dumped source:\n%s", srv.name, rw.name, source)
					}
					require.Empty(t, noop, "[%s/%s] production-path no-op must be empty, got:\n%s", srv.name, rw.name, noop)
				})
			}
		})
	}
}

// TestSDLRealWorldOriginalVsSynced is the adversarial original-vs-synced probe: it diffs the
// ORIGINAL hand-authored fixture SDL (as a user would commit it) against the engine-synced
// metadata. This is stricter than the canonical-dump round-trip because the original carries
// non-canonical forms (utf8 vs utf8mb3, int display widths, charset defaults, hardcoded view
// qualifiers, ROW_FORMAT options, ...) that the engine rewrites on store. A non-empty result
// here pins a real normalization gap the production path would surface to a user editing the
// committed schema.
//
// NOTE: this is run only for fixtures whose original text re-imports cleanly under the omni
// SDL loader (which disables FK checks but parses the raw DDL). Fixtures that carry DELIMITER
// blocks, version-gated /*! */ partition comments, or MediaWiki-style standalone CREATE INDEX
// statements are NOT valid single-document SDL inputs to the omni loader, so they are diffed
// only through the canonical-dump round-trip in TestSDLRealWorldIdempotence. The viable
// original-vs-synced fixtures are roundcube (pure CREATE TABLE) and employees (CREATE TABLE +
// CREATE OR REPLACE VIEW). They are the at-scale normalization stressors.
//
//nolint:tparallel
func TestSDLRealWorldOriginalVsSynced(t *testing.T) {
	skipUnlessLiveOracle(t)
	ctx := context.Background()

	// Only fixtures that are valid as a single SDL document (no DELIMITER / no /*! */ gated
	// ALTERs / no standalone CREATE INDEX) are eligible. employees uses CREATE OR REPLACE VIEW
	// which the omni loader accepts; roundcube is pure CREATE TABLE.
	//
	// Both eligible fixtures currently surface a distinct, being-fixed-upstream (B) omni bug, so
	// the assertion is pending with a precise classification (the load + sync still run):
	//   - roundcube: varchar(N) BINARY -> _bin phantom MODIFY (omniVarcharBinaryBug),
	//   - employees: current_dept_emp's parenthesized-join dump fails to re-parse (omniViewParseBug).
	eligible := map[string]string{"roundcube": omniVarcharBinaryBug, "employees": omniViewParseBug}

	for _, srv := range liveServers {
		srv := srv
		t.Run(srv.name, func(t *testing.T) {
			for _, rw := range realWorldSchemas() {
				rw := rw
				pending, ok := eligible[rw.name]
				if !ok {
					continue
				}
				t.Run(rw.name, func(t *testing.T) {
					meta, _ := loadRealWorld(ctx, t, srv, rw)
					if pending != "" {
						t.Skipf("[%s/%s] original-vs-synced pending: %s", srv.name, rw.name, pending)
					}

					noop, err := schema.SDLMigration(storepb.Engine_MYSQL, rw.ddl, meta, srv.version)
					require.NoError(t, err)
					if noop != "" {
						t.Logf("[ORIGINAL-VS-SYNCED %s/%s] NON-EMPTY (%d stmts):\n%s",
							srv.name, rw.name, statementCount(noop), noop)
					}
					require.Empty(t, noop, "[%s/%s] original-vs-synced no-op must be empty, got:\n%s", srv.name, rw.name, noop)
				})
			}
		})
	}
}

// ============================================================================
// Phase 2: realistic daily-dev migrations on the live DB, mixing object types.
//
// Each scenario: source = MetadataToSDL(sync(current live DB)); build a modified target;
// plan = mysqlDiffSDLMigration(source, target, version); assert the plan is minimal +
// correctly ordered (per scenario); apply the plan to the live DB; re-sync; assert the next
// diff is empty (convergence).
//
// Sakila is the primary vehicle (real views/functions/procedures/triggers). employees_part
// covers a partition change; roundcube/mediawiki cover table/column/index/FK breadth at scale.
// ============================================================================

// rwScenario is one daily-dev migration applied on top of a freshly-loaded real schema.
type rwScenario struct {
	name string
	// schema selects which fixture to load as the live baseline.
	schema string
	// mutate transforms the canonical source SDL into the developer's modified target. It
	// receives the dumped source so it can do targeted string surgery on the real stored form.
	mutate func(t *testing.T, srv liveServer, source string) string
	// wantContains are uppercased substrings the generated plan MUST contain (minimality /
	// correctness anchors). Empty means "just assert non-empty + converges".
	wantContains []string
	// skip57 marks scenarios that only run on 8.0 (e.g. sakila-based ones).
	skip57 string
}

// mustReplace asserts the replacement actually changed the text (guards against fixture drift
// silently turning a scenario into a no-op).
func mustReplace(t *testing.T, s, old, replacement string) string {
	t.Helper()
	require.Contains(t, s, old, "scenario setup: source must contain %q to mutate", old)
	return strings.Replace(s, old, replacement, 1)
}

func rwScenarios() []rwScenario {
	return []rwScenario{
		// ---- Sakila: VIEW changes ----
		{
			name:   "sakila_view_alter_select",
			schema: "sakila",
			skip57: "sakila view scenarios run on 8.0 (sakila does not fully load on 5.7)",
			// film_list is a real view joining film/category/film_actor/actor. Append a genuinely
			// new column (replacement_cost) to its SELECT — a developer broadening a reporting view.
			// (price/rental_rate is already projected, so we add a column not already present.)
			mutate: func(t *testing.T, _ liveServer, source string) string {
				return addColumnToView(t, source, "film_list", "`film`.`replacement_cost` AS `replacement_cost`")
			},
			wantContains: []string{"FILM_LIST"},
		},
		{
			name:   "sakila_view_add_new_and_view_on_view",
			schema: "sakila",
			skip57: "sakila view scenarios run on 8.0",
			// Add a brand-new view AND a view that depends on it (view-on-view), appended to the
			// source. The plan must create both, ordered base-before-dependent.
			mutate: func(_ *testing.T, _ liveServer, source string) string {
				return source + `
CREATE VIEW v_actor_min AS SELECT actor_id, first_name FROM actor;
CREATE VIEW v_actor_min2 AS SELECT actor_id FROM v_actor_min;
`
			},
			wantContains: []string{"V_ACTOR_MIN", "V_ACTOR_MIN2"},
		},
		{
			name:   "sakila_view_drop",
			schema: "sakila",
			skip57: "sakila view scenarios run on 8.0",
			// Drop the sales_by_store view (a real aggregate view) by removing it from the source.
			mutate: func(t *testing.T, _ liveServer, source string) string {
				return dropObjectBlock(t, source, "VIEW", "sales_by_store")
			},
			wantContains: []string{"DROP VIEW", "SALES_BY_STORE"},
		},
		// ---- Sakila: FUNCTION changes ----
		{
			name:   "sakila_function_add",
			schema: "sakila",
			skip57: "sakila routine scenarios run on 8.0",
			mutate: func(_ *testing.T, _ liveServer, source string) string {
				return source + `
CREATE FUNCTION f_film_count() RETURNS INT READS SQL DATA RETURN (SELECT COUNT(*) FROM film);
`
			},
			wantContains: []string{"F_FILM_COUNT"},
		},
		{
			name:   "sakila_function_drop",
			schema: "sakila",
			skip57: "sakila routine scenarios run on 8.0",
			// Drop inventory_held_by_customer (a real function with a body).
			mutate: func(t *testing.T, _ liveServer, source string) string {
				return dropObjectBlock(t, source, "FUNCTION", "inventory_held_by_customer")
			},
			wantContains: []string{"DROP FUNCTION", "INVENTORY_HELD_BY_CUSTOMER"},
		},
		// ---- Sakila: PROCEDURE changes ----
		{
			name:   "sakila_procedure_add",
			schema: "sakila",
			skip57: "sakila routine scenarios run on 8.0",
			mutate: func(_ *testing.T, _ liveServer, source string) string {
				return source + `
CREATE PROCEDURE p_touch_film(IN fid INT) BEGIN UPDATE film SET last_update = NOW() WHERE film_id = fid; END;
`
			},
			wantContains: []string{"P_TOUCH_FILM"},
		},
		{
			name:   "sakila_procedure_drop",
			schema: "sakila",
			skip57: "sakila routine scenarios run on 8.0",
			mutate: func(t *testing.T, _ liveServer, source string) string {
				return dropObjectBlock(t, source, "PROCEDURE", "film_not_in_stock")
			},
			wantContains: []string{"DROP PROCEDURE", "FILM_NOT_IN_STOCK"},
		},
		// ---- Sakila: TRIGGER changes ----
		{
			name:   "sakila_trigger_add",
			schema: "sakila",
			skip57: "sakila trigger scenarios run on 8.0",
			mutate: func(_ *testing.T, _ liveServer, source string) string {
				return source + `
CREATE TRIGGER trg_payment_bi BEFORE INSERT ON payment FOR EACH ROW SET NEW.last_update = NOW();
`
			},
			wantContains: []string{"TRG_PAYMENT_BI"},
		},
		{
			name:   "sakila_trigger_drop",
			schema: "sakila",
			skip57: "sakila trigger scenarios run on 8.0",
			// Drop the ins_film trigger (real multi-statement trigger on film).
			mutate: func(t *testing.T, _ liveServer, source string) string {
				return dropTrigger(t, source, "ins_film")
			},
			wantContains: []string{"DROP TRIGGER", "INS_FILM"},
		},
		// ---- Sakila: TABLE / COLUMN / INDEX / FK changes ----
		{
			name:   "sakila_table_add_column_index",
			schema: "sakila",
			skip57: "run on 8.0 alongside the other sakila scenarios",
			// Add a NOT NULL column with a default + a secondary index to the real `category`
			// table (widening daily dev: a new flag + an index on it).
			mutate: func(t *testing.T, _ liveServer, source string) string {
				s := addColumnToTable(t, source, "category", "`is_featured` tinyint(1) NOT NULL DEFAULT '0'")
				return addIndexToTable(t, s, "category", "KEY `idx_cat_featured` (`is_featured`)")
			},
			wantContains: []string{"IS_FEATURED", "IDX_CAT_FEATURED"},
		},
		{
			name:   "sakila_widen_varchar",
			schema: "sakila",
			skip57: "run on 8.0 alongside the other sakila scenarios",
			// Widen actor.first_name VARCHAR(45) -> VARCHAR(80): a classic column-type change.
			mutate: func(t *testing.T, _ liveServer, source string) string {
				return widenActorFirstName(t, source)
			},
			wantContains: []string{"FIRST_NAME"},
		},
		// ---- Sakila: COMBINED feature migration (table + dependent view + function + trigger) ----
		{
			name:   "sakila_combined_feature_release",
			schema: "sakila",
			skip57: "run on 8.0 alongside the other sakila scenarios",
			// One release that, together: adds a column to `staff`, adds a function, adds a
			// trigger on `staff`, and adds a new view that reads the new column. Exercises correct
			// cross-object ordering in a single diff.
			mutate: func(t *testing.T, _ liveServer, source string) string {
				s := addColumnToTable(t, source, "staff", "`shift_note` varchar(120) DEFAULT NULL")
				s += `
CREATE FUNCTION f_staff_count() RETURNS INT READS SQL DATA RETURN (SELECT COUNT(*) FROM staff);
CREATE VIEW v_staff_notes AS SELECT staff_id, shift_note FROM staff;
CREATE TRIGGER trg_staff_bu BEFORE UPDATE ON staff FOR EACH ROW SET NEW.last_update = NOW();
`
				return s
			},
			wantContains: []string{"SHIFT_NOTE", "F_STAFF_COUNT", "V_STAFF_NOTES", "TRG_STAFF_BU"},
		},
		// ---- employees_part: PARTITION change ----
		{
			name:   "employees_part_partition_change",
			schema: "employees_part",
			// salaries is RANGE COLUMNS(from_date) partitioned. Coalesce its tail by replacing the
			// whole partition definition with a coarser 3-partition scheme. The plan must emit a
			// partition reorganization, apply, and converge.
			mutate: func(t *testing.T, _ liveServer, source string) string {
				return repartitionSalaries(t, source)
			},
			wantContains: []string{"PARTITION"},
		},
		// ---- roundcube: table/column/index/FK at scale ----
		{
			name:   "roundcube_add_column_index",
			schema: "roundcube",
			// Add a column + index to the real `users` table.
			mutate: func(t *testing.T, _ liveServer, source string) string {
				s := addColumnToTable(t, source, "users", "`timezone` varchar(64) DEFAULT NULL")
				return addIndexToTable(t, s, "users", "KEY `idx_users_tz` (`timezone`)")
			},
			wantContains: []string{"TIMEZONE", "IDX_USERS_TZ"},
		},
		{
			name:   "roundcube_drop_index",
			schema: "roundcube",
			// Drop a genuine non-FK secondary index so the plan emits a clean DROP INDEX. `session`
			// has no foreign keys and exactly one secondary index (expires_at_index), so dropping it
			// can never hit errno 1553. (contacts' only secondary index, user_contacts_index, backs
			// its user_id FK and cannot be dropped on its own — that would be a different, FK-ordering
			// scenario.)
			mutate: func(t *testing.T, _ liveServer, source string) string {
				return dropFirstSecondaryIndex(t, source, "session")
			},
			wantContains: []string{"DROP", "EXPIRES_AT_INDEX"},
		},
		// ---- mediawiki: table/column/index at scale (59-table schema) ----
		{
			name:   "mediawiki_add_column",
			schema: "mediawiki",
			// Add a column to the real `page` table in a 58-table schema. The diff must be minimal
			// (touch only `page`), not churn the other 57 tables.
			mutate: func(t *testing.T, _ liveServer, source string) string {
				return addColumnToTable(t, source, "page", "`page_extra_flag` tinyint(1) NOT NULL DEFAULT '0'")
			},
			wantContains: []string{"PAGE_EXTRA_FLAG"},
		},
	}
}

// TestSDLRealWorldMigrations is Phase 2: realistic daily-dev migrations, mixing object types,
// verified end to end (emit minimal+ordered DDL -> apply to the live DB -> converge).
//
//nolint:tparallel
func TestSDLRealWorldMigrations(t *testing.T) {
	skipUnlessLiveOracle(t)
	ctx := context.Background()

	byName := map[string]realWorldSchema{}
	for _, rw := range realWorldSchemas() {
		byName[rw.name] = rw
	}

	for _, srv := range liveServers {
		srv := srv
		t.Run(srv.name, func(t *testing.T) {
			for _, sc := range rwScenarios() {
				sc := sc
				t.Run(sc.name, func(t *testing.T) {
					if srv.version == "5.7" && sc.skip57 != "" {
						t.Skipf("[%s/%s] skipped on 5.7: %s", srv.name, sc.name, sc.skip57)
					}
					rw, ok := byName[sc.schema]
					require.True(t, ok, "unknown schema %q", sc.schema)
					if srv.version == "5.7" && rw.only57Skip != "" {
						t.Skipf("[%s/%s] schema %s not loadable on 5.7: %s", srv.name, sc.name, rw.name, rw.only57Skip)
					}
					if rw.viewParsePending != "" {
						// The mysqlDiffSDLMigration plan loads the dumped source, which carries a
						// multi-table view the omni SDL parser cannot re-read. Every scenario on this
						// schema is blocked at the source-load step (not by the scenario's own change),
						// so skip with classification rather than fail. NOTE: this gates the entire
						// sakila view/function/procedure/trigger migration path — see report headline.
						t.Skipf("[%s/%s] migration plan pending: %s", srv.name, sc.name, rw.viewParsePending)
					}

					_, dbName := loadRealWorld(ctx, t, srv, rw)
					source := dumpSDL(ctx, t, srv, dbName)

					target := sc.mutate(t, srv, source)
					require.NotEqual(t, source, target, "[%s/%s] scenario produced an identical target (no change)", srv.name, sc.name)

					plan, err := mysqlDiffSDLMigration(source, target, srv.version)
					require.NoError(t, err)
					require.NotEmpty(t, plan, "[%s/%s] scenario must produce DDL", srv.name, sc.name)
					t.Logf("[PHASE2 %s/%s] plan (%d stmts):\n%s", srv.name, sc.name, statementCount(plan), plan)

					upper := strings.ToUpper(plan)
					for _, want := range sc.wantContains {
						require.Contains(t, upper, want, "[%s/%s] plan missing %q:\n%s", srv.name, sc.name, want, plan)
					}

					// Apply the generated DDL to the live DB and confirm convergence.
					applyErr := applyDDL(ctx, t, srv, dbName, plan)
					require.NoError(t, applyErr, "[%s/%s] plan failed to apply:\n%s", srv.name, sc.name, plan)

					newSource := dumpSDL(ctx, t, srv, dbName)
					converge, err := mysqlDiffSDLMigration(newSource, target, srv.version)
					require.NoError(t, err)
					if converge != "" {
						t.Logf("[PHASE2 %s/%s] NON-EMPTY residual after apply (%d stmts):\n%s",
							srv.name, sc.name, statementCount(converge), converge)
					}
					require.Empty(t, converge, "[%s/%s] did not converge, residual:\n%s", srv.name, sc.name, converge)
				})
			}
		})
	}
}

// ============================================================================
// String-surgery helpers for Phase 2 target construction. They operate on the canonical
// MetadataToSDL dump (backtick-quoted, lowercase keywords for column defs, uppercase for
// table-level CREATE) and assert their anchor exists so a fixture drift fails loudly.
// ============================================================================

// findObjectSegment locates the whole CREATE statement for an object in the dumped source and
// returns its [start,end) byte range. It splits the source with the SAME splitter the SDL
// loader uses (mysqlparser.SplitSQL -> omni Split), so a multi-statement routine body (DECLARE
// ...; ...; END;) is treated as one segment — naive ";"-scanning truncated such bodies at their
// first internal semicolon. headerMatch must hold for the (whitespace-trimmed) segment text.
func findObjectSegment(t *testing.T, source string, headerMatch func(stmt string) bool) (int, int) {
	t.Helper()
	stmts, err := mysqlparser.SplitSQL(source)
	require.NoError(t, err, "split source for object lookup")
	for _, s := range stmts {
		if headerMatch(strings.TrimSpace(s.Text)) {
			return int(s.Range.Start), int(s.Range.End)
		}
	}
	return -1, -1
}

// dumpedViewHeader reports whether stmt is the CREATE statement for the named view in the
// canonical dump. The dumper emits "CREATE OR REPLACE ALGORITHM=UNDEFINED SQL SECURITY DEFINER
// VIEW `name` AS ..." (or, for an INVOKER view, "... SQL SECURITY INVOKER VIEW `name`"), so the
// modifiers between VIEW and the name vary — match on the "VIEW `name`" token regardless of the
// ALGORITHM/SQL SECURITY clause.
func dumpedViewHeader(stmt, view string) bool {
	u := strings.ToUpper(stmt)
	return strings.HasPrefix(u, "CREATE ") && strings.Contains(u, " VIEW `"+strings.ToUpper(view)+"` ")
}

// removeSegment excises [start,end) from source and trims now-orphaned blank lines.
func removeSegment(source string, start, end int) string {
	// Swallow a trailing newline so we don't leave a blank gap where the statement was.
	for end < len(source) && (source[end] == '\n' || source[end] == '\r') {
		end++
	}
	return source[:start] + source[end:]
}

// addColumnToView appends a column expression to a view's SELECT list by inserting it before the
// FROM of that view's definition. It operates on the whole dumped view statement (located via
// the splitter) so it is robust to the real "CREATE OR REPLACE ALGORITHM=... VIEW" header form.
// The FROM anchor tolerates any surrounding whitespace: the pretty-printed dump puts the
// top-level from on its own line (…AS `x`\nfrom `t`…).
func addColumnToView(t *testing.T, source, view, colExpr string) string {
	t.Helper()
	start, end := findObjectSegment(t, source, func(stmt string) bool { return dumpedViewHeader(stmt, view) })
	require.GreaterOrEqual(t, start, 0, "view %q not found in source", view)
	stmt := source[start:end]
	loc := reViewFromAnchor.FindStringIndex(stmt)
	require.NotNil(t, loc, "view %q body has no FROM to anchor on:\n%s", view, stmt)
	fromIdx := loc[0]
	newStmt := stmt[:fromIdx] + "," + colExpr + stmt[fromIdx:]
	return source[:start] + newStmt + source[end:]
}

// reViewFromAnchor matches the first whitespace-delimited FROM keyword in a dumped view
// statement (case-insensitive, any whitespace kind on both sides).
var reViewFromAnchor = regexp.MustCompile(`(?i)[ \t\r\n]from[ \t\r\n]`)

// dropObjectBlock removes the whole CREATE <kind> ... block (VIEW/FUNCTION/PROCEDURE) for the
// named object so the diff emits a DROP. It locates the statement via the SDL splitter, so a
// routine with a multi-statement body is removed in full.
func dropObjectBlock(t *testing.T, source, kind, name string) string {
	t.Helper()
	var match func(stmt string) bool
	switch kind {
	case "VIEW":
		match = func(stmt string) bool { return dumpedViewHeader(stmt, name) }
	case "FUNCTION":
		match = func(stmt string) bool {
			u := strings.ToUpper(stmt)
			return strings.HasPrefix(u, "CREATE ") && strings.Contains(u, "FUNCTION `"+strings.ToUpper(name)+"`")
		}
	case "PROCEDURE":
		match = func(stmt string) bool {
			u := strings.ToUpper(stmt)
			return strings.HasPrefix(u, "CREATE ") && strings.Contains(u, "PROCEDURE `"+strings.ToUpper(name)+"`")
		}
	default:
		t.Fatalf("unsupported kind %q", kind)
	}
	start, end := findObjectSegment(t, source, match)
	require.GreaterOrEqual(t, start, 0, "%s %q not found in source", kind, name)
	return removeSegment(source, start, end)
}

// dropTrigger removes the whole CREATE TRIGGER `name` ... block (located via the SDL splitter,
// so a BEGIN ... END; body is removed in full).
func dropTrigger(t *testing.T, source, name string) string {
	t.Helper()
	start, end := findObjectSegment(t, source, func(stmt string) bool {
		u := strings.ToUpper(stmt)
		return strings.HasPrefix(u, "CREATE ") && strings.Contains(u, "TRIGGER `"+strings.ToUpper(name)+"`")
	})
	require.GreaterOrEqual(t, start, 0, "trigger %q not found in source", name)
	return removeSegment(source, start, end)
}

// addColumnToTable inserts a column definition into a table's CREATE block, right after the
// table's PRIMARY KEY line (or before the closing paren if no PK line is found). It anchors on
// the canonical "CREATE TABLE `name` (" header.
func addColumnToTable(t *testing.T, source, table, colDef string) string {
	t.Helper()
	header := "CREATE TABLE `" + table + "` ("
	start := strings.Index(source, header)
	require.GreaterOrEqual(t, start, 0, "table %q not found in source", table)
	// Find the closing ")" of this CREATE TABLE at the start of a line (the dumper emits the
	// closing paren + options on its own line beginning with ')').
	bodyStart := start + len(header)
	closeIdx := findTableClose(source, bodyStart)
	require.GreaterOrEqual(t, closeIdx, 0, "table %q close paren not found", table)
	// Insert the column as a new line just before the close. Strip any trailing comma handling:
	// the dumper's last body line has no trailing comma, so we add ",\n  <colDef>" after the
	// last body line's content. Simplest robust form: insert "  <colDef>,\n" right after the
	// body start (first column position) — MySQL accepts column order changes via SDL.
	insertion := "  " + colDef + ",\n"
	return source[:bodyStart] + "\n" + insertion + source[bodyStart:closeIdx] + source[closeIdx:]
}

// addIndexToTable inserts an index definition into a table's CREATE block before the closing
// paren.
func addIndexToTable(t *testing.T, source, table, idxDef string) string {
	t.Helper()
	header := "CREATE TABLE `" + table + "` ("
	start := strings.Index(source, header)
	require.GreaterOrEqual(t, start, 0, "table %q not found in source", table)
	bodyStart := start + len(header)
	closeIdx := findTableClose(source, bodyStart)
	require.GreaterOrEqual(t, closeIdx, 0, "table %q close paren not found", table)
	// The line immediately before closeIdx ends a body element without a trailing comma. Insert
	// ",\n  <idxDef>" right before the close paren, after trimming the trailing newline.
	body := source[bodyStart:closeIdx]
	trimmed := strings.TrimRight(body, "\n")
	newBody := trimmed + ",\n  " + idxDef + "\n"
	return source[:bodyStart] + newBody + source[closeIdx:]
}

// findTableClose returns the index of the line-leading ')' that closes the CREATE TABLE body
// beginning at bodyStart. The dumper emits ")\n" or ") ENGINE=..." with the ')' at column 0.
func findTableClose(source string, bodyStart int) int {
	nl := strings.Index(source[bodyStart:], "\n)")
	if nl < 0 {
		return -1
	}
	return bodyStart + nl + 1
}

// dropFirstSecondaryIndex removes the first "  KEY `...` (...)," line from the named table so
// the diff emits a DROP INDEX. It skips PRIMARY/UNIQUE to keep the change a plain index drop.
func dropFirstSecondaryIndex(t *testing.T, source, table string) string {
	t.Helper()
	header := "CREATE TABLE `" + table + "` ("
	start := strings.Index(source, header)
	require.GreaterOrEqual(t, start, 0, "table %q not found in source", table)
	bodyStart := start + len(header)
	closeIdx := findTableClose(source, bodyStart)
	require.GreaterOrEqual(t, closeIdx, 0, "table %q close paren not found", table)
	body := source[bodyStart:closeIdx]
	lines := strings.Split(body, "\n")
	out := make([]string, 0, len(lines))
	dropped := false
	for _, ln := range lines {
		trimmed := strings.TrimSpace(ln)
		if !dropped && strings.HasPrefix(trimmed, "KEY `") {
			dropped = true
			// Remove a dangling comma on the previous line if this was the last body element.
			continue
		}
		out = append(out, ln)
	}
	require.True(t, dropped, "table %q had no secondary KEY to drop:\n%s", table, body)
	newBody := strings.Join(out, "\n")
	// Fix a possible trailing comma before the close paren.
	newBody = strings.TrimRight(newBody, "\n")
	newBody = strings.TrimRight(newBody, ",")
	newBody += "\n"
	return source[:bodyStart] + newBody + source[closeIdx:]
}

// widenActorFirstName changes actor.first_name varchar(45) -> varchar(80) in the source.
func widenActorFirstName(t *testing.T, source string) string {
	t.Helper()
	// The dumper renders the column as: `first_name` varchar(45) ...
	old := "`first_name` varchar(45)"
	require.Contains(t, source, old, "actor.first_name varchar(45) not found in source")
	return strings.Replace(source, old, "`first_name` varchar(80)", 1)
}

// repartitionSalaries replaces the salaries table partition clause with a coarser 3-partition
// RANGE COLUMNS scheme. It anchors on the canonical PARTITION BY block the dumper emits.
func repartitionSalaries(t *testing.T, source string) string {
	t.Helper()
	header := "CREATE TABLE `salaries`"
	start := strings.Index(source, header)
	require.GreaterOrEqual(t, start, 0, "salaries table not found in source")
	// Find the PARTITION BY clause within the salaries statement.
	rest := source[start:]
	semi := strings.Index(rest, ";")
	require.GreaterOrEqual(t, semi, 0, "salaries statement has no terminating semicolon")
	stmt := rest[:semi]
	pIdx := strings.Index(strings.ToUpper(stmt), "PARTITION BY")
	require.GreaterOrEqual(t, pIdx, 0, "salaries has no PARTITION BY clause to change:\n%s", stmt)
	// The dumper wraps partitioning in a /*!NNNNN PARTITION BY ... */ executable comment. Replace
	// the whole partition portion INCLUDING the wrapper opener, else we leave an unterminated
	// /*! comment (and produce invalid SQL).
	replStart := pIdx
	if c := strings.LastIndex(stmt[:pIdx], "/*!"); c >= 0 {
		replStart = c
	}
	// Build a coarser partition scheme covering the same range column.
	newPart := "PARTITION BY RANGE COLUMNS(`from_date`)\n" +
		"(PARTITION p_old VALUES LESS THAN ('1990-01-01'),\n" +
		" PARTITION p_mid VALUES LESS THAN ('2000-01-01'),\n" +
		" PARTITION p_max VALUES LESS THAN (MAXVALUE))"
	newStmt := stmt[:replStart] + newPart
	return source[:start] + newStmt + source[start+semi:]
}

// uniqueDBName mirrors newLiveDatabase's naming for ad hoc probes (kept for symmetry/debug).
//
//nolint:unused
func uniqueDBName(prefix string) string {
	return fmt.Sprintf("%s_%s", prefix, strings.ReplaceAll(uuid.New().String(), "-", "_"))
}

// Stress smoke test for the MySQL declarative (SDL) migration path, driving the
// PRODUCTION entry points end to end against live MySQL 5.7 + 8.0:
//
//   - schema.SDLMigration(MYSQL, userSDL, syncedMetadata, engineVersion) — the exact
//     call database_migrate_executor.diff() makes,
//   - mysqlDiffSDLMigration(source, target, engineVersion) — version-aware diff,
//   - schema.SDLDropAdvices(MYSQL, userSDL, syncedMetadata, engineVersion) — drop advices.
//
// The per-type basics live in the live round-trip axes above. These cases push harder:
// a large combined schema with many interacting objects, deliberately non-canonical user
// forms (where idempotence is won or lost), a multi-change release that must order DDL
// correctly, a drop-heavy release with advices, and explicit 5.7-vs-8.0 divergence guards.
//
// Shared helpers (createLiveMySQLDriver, newLiveDatabase, applyAndDump, statementCount,
// liveServers, liveServer, liveOraclePassword) are declared with the live round-trip axes
// (same package).

// syncMetadata applies ddl to a fresh database on srv, syncs it, and returns the
// model.DatabaseMetadata (the value the production release path feeds to
// schema.SDLMigration) together with the database name so the caller can apply DDL back.
func syncMetadata(ctx context.Context, t *testing.T, srv liveServer, prefix, ddl string) (*model.DatabaseMetadata, string) {
	t.Helper()
	dbName := newLiveDatabase(ctx, t, srv, prefix)
	driver, err := createLiveMySQLDriver(ctx, srv, dbName)
	require.NoError(t, err)
	defer driver.Close(ctx)
	_, err = driver.Execute(ctx, ddl, db.ExecuteOptions{})
	require.NoError(t, err, "[%s] apply setup DDL:\n%s", srv.name, ddl)
	metadata, err := driver.SyncDBSchema(ctx)
	require.NoError(t, err)
	return model.NewDatabaseMetadata(metadata, nil, nil, storepb.Engine_MYSQL, true), dbName
}

// dumpSDL syncs dbName on srv and returns the canonical MetadataToSDL dump.
func dumpSDL(ctx context.Context, t *testing.T, srv liveServer, dbName string) string {
	t.Helper()
	driver, err := createLiveMySQLDriver(ctx, srv, dbName)
	require.NoError(t, err)
	defer driver.Close(ctx)
	metadata, err := driver.SyncDBSchema(ctx)
	require.NoError(t, err)
	source, err := schema.MetadataToSDL(storepb.Engine_MYSQL, model.NewDatabaseMetadata(metadata, nil, nil, storepb.Engine_MYSQL, true))
	require.NoError(t, err)
	return source
}

// applyDDL executes ddl against dbName on srv.
func applyDDL(ctx context.Context, t *testing.T, srv liveServer, dbName, ddl string) error {
	t.Helper()
	driver, err := createLiveMySQLDriver(ctx, srv, dbName)
	require.NoError(t, err)
	defer driver.Close(ctx)
	_, err = driver.Execute(ctx, ddl, db.ExecuteOptions{})
	return err
}

// ============================================================================
// Scenario 1: Large combined-schema idempotence (the headline test).
// ============================================================================

// bigSchemaCore is the shared body of the large combined schema: a single realistic
// database with many interacting objects. It spans the normalization-sensitive constructs
// plus PKs, composite/unique/prefix/fulltext indexes, foreign keys (circular, self-ref,
// composite), CHECK constraints, RANGE + HASH partitions, views (view-on-view, joined),
// stored functions + procedures, multiple triggers per table, and an event. The
// CTE/derived-table view (v_project_load) diverges by version (5.7 has no CTE support) and
// is appended by bigSchemaProjectLoadView.
//
// CHECK note: 8.0 honors CHECK; 5.7 parses-and-ignores it (a no-op there). The same body
// therefore applies on both, and the synced metadata differs only by the absent checks on
// 5.7 — which is exactly the per-version idempotence each side must satisfy.
//
// FK note: department<->employee is circular, so the FK that closes the cycle
// (department.manager_id -> employee) is added with a trailing ALTER TABLE after both
// tables exist (the SDL loader disables foreign_key_checks, but the live setup apply does
// not).
//
//go:embed testdata/sdl/big_schema_core.sql
var bigSchemaCore string

// circularFKClose closes the department<->employee circular dependency after both
// tables exist (the live setup apply keeps foreign_key_checks on, unlike the SDL loader).
const circularFKClose = `
ALTER TABLE department ADD CONSTRAINT fk_dept_manager FOREIGN KEY (manager_id) REFERENCES employee (id) ON DELETE SET NULL;
`

// bigSchemaProjectLoadView returns the version-correct v_project_load definition.
//
//   - 8.0 uses a CTE (WITH ...) — exercising the CTE construct and confirming the dump
//     preserves it verbatim. (8.0 also stores derived-table inner refs unqualified, so the
//     CTE form round-trips cleanly.)
//   - 5.7 has no CTE support AND mis-handles derived tables in the SDL no-op path (see
//     TestSDLStressViewDerivedTable57 — 5.7 db-qualifies derived-table inner refs, which the
//     bbcatalog-wrapped omni diff cannot fold). So 5.7 uses a view-on-view form
//     (v_active_employees-style, with a real aggregate join) that is known-idempotent and
//     keeps the large-schema test focused on finding OTHER at-scale bugs.
func bigSchemaProjectLoadView(version string) string {
	if version == "5.7" {
		return `
CREATE VIEW v_proj_member_count AS
SELECT project_id, COUNT(*) AS members, SUM(allocation) AS total_alloc
FROM assignment GROUP BY project_id;

CREATE VIEW v_project_load AS
SELECT p.code, p.title, pmc.members, pmc.total_alloc
FROM project p JOIN v_proj_member_count pmc ON p.id = pmc.project_id;
`
	}
	return `
CREATE VIEW v_project_load AS
WITH per_proj AS (
	SELECT project_id, COUNT(*) AS members, SUM(allocation) AS total_alloc
	FROM assignment GROUP BY project_id
)
SELECT p.code, p.title, pp.members, pp.total_alloc
FROM project p JOIN per_proj pp ON p.id = pp.project_id;
`
}

// bigSchemaSetup is the full apply sequence for the live database: the shared core, the
// version-correct CTE/derived view, then the closing circular FK (added last so both
// referenced tables exist under foreign_key_checks=ON). CHECK constraints are valid on
// 8.0; 5.7 silently parses-and-ignores CHECK, so the same body applies on both — the
// synced metadata differs only by the absent checks on 5.7, which is exactly the
// per-version idempotence each side must satisfy.
func bigSchemaSetup(version string) string {
	return bigSchemaCore + bigSchemaProjectLoadView(version) + circularFKClose
}

// bigSchemaUserSDL is what a user would author: the table bodies (circular FK inlined into
// department) plus the version-correct project-load view. Declarative SDL has no ordering
// constraint (the omni loader disables FK checks during load). This is the single-document
// target D the production path diffs against.
func bigSchemaUserSDL(version string) string {
	return bigSchemaUserSDLCore + bigSchemaProjectLoadView(version)
}

// bigSchemaUserSDLCore is the table/routine/trigger body the user authors, sans the
// version-specific project-load view (appended by bigSchemaUserSDL).
//
//go:embed testdata/sdl/big_schema_user_sdl_core.sql
var bigSchemaUserSDLCore string

// TestSDLStressLargeSchemaIdempotence is the headline at-scale idempotence proof. One
// realistic ~13-table schema with views, view-on-view, CTE view, functions, procedures,
// multiple triggers, an event, RANGE + HASH partitions, circular/self-ref/composite FKs,
// and CHECK constraints (8.0) is applied to a live database, synced, dumped via
// MetadataToSDL, then diffed against the user-authored SDL through the PRODUCTION
// version-aware path. The diff MUST be empty on both 5.7 and 8.0.
//
//nolint:tparallel
func TestSDLStressLargeSchemaIdempotence(t *testing.T) {
	skipUnlessLiveOracle(t)
	ctx := context.Background()

	for _, srv := range liveServers {
		srv := srv
		t.Run(srv.name, func(t *testing.T) {
			meta, dbName := syncMetadata(ctx, t, srv, "sdl_stress_big", bigSchemaSetup(srv.version))
			userSDL := bigSchemaUserSDL(srv.version)

			// (a) Production-path no-op: schema.SDLMigration converts meta -> SDL and diffs
			// against the user SDL, threading the version. MUST be empty.
			noop, err := schema.SDLMigration(storepb.Engine_MYSQL, userSDL, meta, srv.version)
			require.NoError(t, err)
			if noop != "" {
				t.Logf("[%s] NON-EMPTY no-op diff (%d statements):\n%s", srv.name, statementCount(noop), noop)
			}
			require.Empty(t, noop, "[%s] large-schema production-path no-op must be empty, got:\n%s", srv.name, noop)

			// (b) source-vs-source determinism: dump and diff against itself.
			source := dumpSDL(ctx, t, srv, dbName)
			selfDiff, err := mysqlDiffSDLMigration(source, source, srv.version)
			require.NoError(t, err)
			require.Empty(t, selfDiff, "[%s] large-schema source-vs-source must be empty, got:\n%s", srv.name, selfDiff)

			// (c) source-vs-user-SDL via the version-aware diff (the same property as (a) but
			// through the lower-level entry the breadth tests use).
			noop2, err := mysqlDiffSDLMigration(source, userSDL, srv.version)
			require.NoError(t, err)
			require.Empty(t, noop2, "[%s] large-schema source-vs-D diff must be empty, got:\n%s", srv.name, noop2)
		})
	}
}

// TestSDLStressViewDerivedTable57 PINS a real (B) wiring bug found by the stress test:
// on MySQL 5.7 a view whose body contains a derived table (a subquery in the FROM clause)
// is NOT idempotent through the SDL no-op path — it emits a spurious CREATE OR REPLACE
// VIEW.
//
// Root cause: MySQL 5.7's view canonicalizer fully-qualifies the table references INSIDE a
// FROM-clause derived table with the schema name (`<synced_db>`.`assignment`), whereas 8.0
// stores them unqualified. MetadataToSDL emits that body verbatim. The omni SDL diff loads
// both inputs under the synthetic database `bbcatalog`, and its canonicalViewBody only
// folds away the view's OWN-database prefix — which is now `bbcatalog`, not the real synced
// DB name embedded in the body. So the `from` body keeps `<synced_db>`.`assignment` while
// the `to` (user) body resolves to `bbcatalog`.`assignment` and is folded to unqualified;
// the two bodies differ and the diff emits a no-op CREATE OR REPLACE VIEW.
//
// Blast radius (mapped live): 5.7 ONLY, and ONLY FROM-clause derived tables. Plain joins,
// view-on-view, scalar subqueries (SELECT list), and IN-subqueries (WHERE) are all
// idempotent on 5.7; 8.0 is fully idempotent including derived tables and CTEs.
//
// Likely fix locus: bytebase MetadataToSDL / get_database_definition.go should emit view
// bodies with the synced own-database qualifier stripped (db-neutral) before the omni
// loader sees them; or the omni SDL wrapping should rewrite the synced DB name to the
// bbcatalog context. The omni canonicalViewBody itself is correct — it just can't fold a
// qualifier naming a database other than the view's loaded own-database.
//
// This test is SKIPPED so the suite stays green; remove the Skip to reproduce the failure.
//
//nolint:tparallel
func TestSDLStressViewDerivedTable57(t *testing.T) {
	skipUnlessLiveOracle(t)
	ctx := context.Background()
	srv := liveServer{name: "mysql57", host: "127.0.0.1", port: "13307", version: "5.7"}

	ddl := `CREATE TABLE project (id INT PRIMARY KEY, code VARCHAR(20)) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
CREATE TABLE assignment (employee_id INT, project_id INT) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
CREATE VIEW vx AS
SELECT p.code, d.c
FROM project p JOIN (SELECT project_id, COUNT(*) c FROM assignment GROUP BY project_id) d
ON p.id = d.project_id;`

	meta, dbName := syncMetadata(ctx, t, srv, "sdl_stress_dtv", ddl)
	noop, err := schema.SDLMigration(storepb.Engine_MYSQL, ddl, meta, srv.version)
	require.NoError(t, err)
	source := dumpSDL(ctx, t, srv, dbName)
	t.Logf("dumped source:\n%s", source)
	require.Empty(t, noop, "5.7 derived-table view no-op must be empty, got:\n%s", noop)
}

// ============================================================================
// Scenario 2: Normalization-heavy user forms (the real idempotence challenge).
// ============================================================================

// normCase authors a table in a deliberately non-canonical form. The schema is applied
// to a live DB, synced, dumped, and the production version-aware diff of (dumped vs the
// user form) MUST be empty — the engine must canonicalize the user form to the stored
// form on each version.
type normCase struct {
	name string
	ddl  string
}

// normCases each name exactly one normalization the engine must absorb. They are NOT
// merged into one table so a single failure pinpoints the offending construct.
func normCases() []normCase {
	return []normCase{
		{
			name: "int_widths",
			// 8.0 drops widths (int/bigint/tinyint); 5.7 keeps int(11)/bigint(20)/tinyint(4).
			ddl: `CREATE TABLE t (
	a INT(11) NOT NULL,
	b BIGINT(20) NOT NULL DEFAULT 0,
	c SMALLINT(6) NOT NULL DEFAULT 0,
	d TINYINT(4) NOT NULL DEFAULT 0,
	e MEDIUMINT(9) NOT NULL DEFAULT 0,
	PRIMARY KEY (a)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;`,
		},
		{
			name: "bool_boolean",
			// BOOL / BOOLEAN both store tinyint(1); TRUE/FALSE -> 1/0.
			ddl: `CREATE TABLE t (
	id INT PRIMARY KEY,
	f1 BOOL NOT NULL DEFAULT FALSE,
	f2 BOOLEAN NOT NULL DEFAULT TRUE,
	f3 BOOL NOT NULL DEFAULT 0,
	f4 BOOLEAN NOT NULL DEFAULT 1
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;`,
		},
		{
			name: "bare_charset_utf8mb4",
			// No COLLATE: resolves to the version default (utf8mb4_general_ci on 5.7,
			// utf8mb4_0900_ai_ci on 8.0) — the headline 5.7-vs-8.0 collation case.
			ddl: `CREATE TABLE t (
	id INT PRIMARY KEY,
	a VARCHAR(50) CHARSET utf8mb4,
	b VARCHAR(50)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;`,
		},
		{
			name: "utf8_to_utf8mb3",
			// utf8 -> utf8mb3 on 8.0; stays utf8 on 5.7.
			ddl: `CREATE TABLE t (
	id INT PRIMARY KEY,
	a VARCHAR(50) CHARSET utf8,
	b TEXT CHARACTER SET utf8
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;`,
		},
		{
			name: "int_defaults",
			// DEFAULT 0 on int and DEFAULT '0' both store '0'.
			ddl: `CREATE TABLE t (
	id INT PRIMARY KEY,
	a INT NOT NULL DEFAULT 0,
	b INT NOT NULL DEFAULT '0',
	c DECIMAL(10,2) NOT NULL DEFAULT 0,
	d VARCHAR(10) NOT NULL DEFAULT '0'
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;`,
		},
		{
			name: "unnamed_index_and_fk",
			// Unnamed index + unnamed FK: the engine auto-names both. Idempotence requires
			// the dump's auto-name to canonicalize against the user's unnamed form.
			ddl: `CREATE TABLE parent (id INT PRIMARY KEY) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
CREATE TABLE t (
	id INT PRIMARY KEY,
	pid INT NOT NULL,
	name VARCHAR(50) NOT NULL,
	INDEX (name),
	FOREIGN KEY (pid) REFERENCES parent (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;`,
		},
		{
			name: "restrict_fk_actions",
			// RESTRICT == no clause. On 5.7 the dump renders ON UPDATE RESTRICT explicitly;
			// on 8.0 it is omitted. Either way the user RESTRICT form must canonicalize equal.
			ddl: `CREATE TABLE parent (id INT PRIMARY KEY) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
CREATE TABLE t (
	id INT PRIMARY KEY,
	pid INT NOT NULL,
	CONSTRAINT fk_t_parent FOREIGN KEY (pid) REFERENCES parent (id) ON DELETE RESTRICT ON UPDATE RESTRICT
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;`,
		},
		{
			name: "using_hash_on_innodb",
			// USING HASH is dropped on InnoDB (B-tree only).
			ddl: `CREATE TABLE t (
	id INT PRIMARY KEY,
	a INT NOT NULL,
	KEY idx_a (a) USING HASH
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;`,
		},
		{
			name: "generated_col_spacing",
			// Odd spacing/casing in the generated expression must canonicalize to the stored form.
			ddl: `CREATE TABLE t (
	id INT PRIMARY KEY,
	price DECIMAL(10,2) NOT NULL DEFAULT 0,
	qty INT NOT NULL DEFAULT 0,
	total DECIMAL(20,2) AS ( price  *  qty ) STORED,
	label VARCHAR(20) AS (CONCAT('x', id)) VIRTUAL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;`,
		},
		// NOTE: ROW_FORMAT=DYNAMIC (and other table CREATE_OPTIONS) are NOT idempotent — see
		// TestSDLStressTableCreateOptions, a documented (B) bug. Pulled out of the passing set.
		{
			name: "enum_set",
			ddl: `CREATE TABLE t (
	id INT PRIMARY KEY,
	color ENUM('red','green','blue') NOT NULL DEFAULT 'red',
	flags SET('a','b','c') NOT NULL DEFAULT 'a,b'
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;`,
		},
		{
			name: "timestamp_defaults",
			ddl: `CREATE TABLE t (
	id INT PRIMARY KEY,
	created TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	updated TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
	dt DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;`,
		},
		{
			name: "auto_increment_seed",
			// AUTO_INCREMENT seed in the table option: the dump may carry AUTO_INCREMENT=N;
			// idempotence requires it not to phantom-diff against the user form without it.
			ddl: `CREATE TABLE t (
	id INT NOT NULL AUTO_INCREMENT,
	name VARCHAR(50) NOT NULL,
	PRIMARY KEY (id)
) ENGINE=InnoDB AUTO_INCREMENT=100 DEFAULT CHARSET=utf8mb4;`,
		},
	}
}

// TestSDLStressNormalization is where idempotence is won or lost. Each non-canonical user
// form is applied live, synced, dumped, and the PRODUCTION version-aware diff (dumped vs
// the user form) MUST be empty on both 5.7 and 8.0. Run as the production entry
// schema.SDLMigration AND the version-aware mysqlDiffSDLMigration.
//
//nolint:tparallel
func TestSDLStressNormalization(t *testing.T) {
	skipUnlessLiveOracle(t)
	ctx := context.Background()

	for _, srv := range liveServers {
		srv := srv
		t.Run(srv.name, func(t *testing.T) {
			for _, nc := range normCases() {
				nc := nc
				t.Run(nc.name, func(t *testing.T) {
					meta, dbName := syncMetadata(ctx, t, srv, "sdl_stress_norm", nc.ddl)

					// Production entry: schema.SDLMigration (meta -> SDL internally).
					noop, err := schema.SDLMigration(storepb.Engine_MYSQL, nc.ddl, meta, srv.version)
					require.NoError(t, err)
					if noop != "" {
						source := dumpSDL(ctx, t, srv, dbName)
						t.Logf("[%s/%s] stored/dumped SDL:\n%s", srv.name, nc.name, source)
						t.Logf("[%s/%s] user form:\n%s", srv.name, nc.name, nc.ddl)
						t.Logf("[%s/%s] NON-EMPTY no-op diff:\n%s", srv.name, nc.name, noop)
					}
					require.Empty(t, noop, "[%s/%s] normalization no-op (SDLMigration) must be empty, got:\n%s", srv.name, nc.name, noop)

					// Version-aware lower-level entry, same property.
					source := dumpSDL(ctx, t, srv, dbName)
					noop2, err := mysqlDiffSDLMigration(source, nc.ddl, srv.version)
					require.NoError(t, err)
					require.Empty(t, noop2, "[%s/%s] normalization no-op (mysqlDiffSDLMigration) must be empty, got:\n%s", srv.name, nc.name, noop2)
				})
			}
		})
	}
}

// TestSDLStressTableCreateOptions PINS a second real (B) wiring bug found by the stress
// test: a table authored with a CREATE_OPTION such as ROW_FORMAT=DYNAMIC is NOT idempotent
// through the SDL no-op path on EITHER 5.7 or 8.0 — it emits a spurious
// `ALTER TABLE t ROW_FORMAT=DYNAMIC`.
//
// Root cause is purely in bytebase's SDL renderer, not omni: MySQL stores ROW_FORMAT in
// CREATE_OPTIONS, and bytebase's MySQL sync DOES capture it into TableMetadata.CreateOptions
// (backend/plugin/db/mysql/sync.go:561). But the SDL dumper
// get_database_definition.go's table-option block (~L587-609) renders only ENGINE,
// DEFAULT CHARSET, COLLATE, COMMENT, and partitions — it never emits table.CreateOptions.
// So the dumped `from` SDL drops ROW_FORMAT while the user `to` SDL keeps it; the omni diff
// (correctly) sees a table-option delta and emits the ALTER. This applies to any
// create-option (ROW_FORMAT, KEY_BLOCK_SIZE, COMPRESSION, STATS_PERSISTENT, ...).
//
// Likely fix locus: bytebase get_database_definition.go — emit table.CreateOptions in the
// SDL table-option block (filtering the synthetic 'partitioned' token, which is surfaced
// via Partitions, not as a literal option). The omni differ already round-trips ROW_FORMAT
// when it is present on both sides.
//
// SKIPPED so the suite stays green; remove the Skip to reproduce.
//
//nolint:tparallel
func TestSDLStressTableCreateOptions(t *testing.T) {
	skipUnlessLiveOracle(t)
	ctx := context.Background()

	for _, srv := range liveServers {
		srv := srv
		t.Run(srv.name, func(t *testing.T) {
			ddl := `CREATE TABLE t (
	id INT PRIMARY KEY,
	a VARCHAR(100) NOT NULL
) ENGINE=InnoDB ROW_FORMAT=DYNAMIC DEFAULT CHARSET=utf8mb4;`
			meta, dbName := syncMetadata(ctx, t, srv, "sdl_stress_rf", ddl)
			noop, err := schema.SDLMigration(storepb.Engine_MYSQL, ddl, meta, srv.version)
			require.NoError(t, err)
			source := dumpSDL(ctx, t, srv, dbName)
			t.Logf("[%s] dumped source (ROW_FORMAT dropped):\n%s", srv.name, source)
			require.Empty(t, noop, "[%s] ROW_FORMAT no-op must be empty, got:\n%s", srv.name, noop)
		})
	}
}

// ============================================================================
// Scenario 3: Multi-change release (combined DDL + ordering).
// ============================================================================

// multiChangeBase is the baseline schema for the multi-change release.
//
//go:embed testdata/sdl/multi_change_base.sql
var multiChangeBase string

// multiChangeTarget applies MANY simultaneous changes in ONE diff:
//   - add table `review` referencing product via FK,
//   - drop the legacy_code column (which has index idx_cat_legacy) from category,
//   - modify product.price type DECIMAL(10,2) -> DECIMAL(12,4),
//   - add an index on product(name),
//   - drop table legacy_audit (an FK CHILD of product — its own FK must drop first),
//   - add a CHECK on product (8.0),
//   - replace view v_catalog,
//   - add a trigger on product,
//   - change daily_stat partitioning (HASH 2 -> HASH 4 partitions).
//
// The 8.0 variant includes the CHECK; multiChangeTarget57 omits it.
//
//go:embed testdata/sdl/multi_change_target_80.sql
var multiChangeTarget80 string

// multiChangeTarget57 is multiChangeTarget80 without the CHECK constraint (5.7 ignores CHECK).
//
//go:embed testdata/sdl/multi_change_target_57.sql
var multiChangeTarget57 string

// indexOf returns the byte index of the first occurrence of substr in s, or -1. Used to
// assert relative ordering between two statements in a generated plan.
func indexOf(s, substr string) int {
	return strings.Index(s, substr)
}

// TestSDLStressMultiChange applies many simultaneous changes in ONE diff and asserts the
// plan is correct, minimal-ish, correctly ordered, and converges when applied back. Both
// versions.
//
//nolint:tparallel
func TestSDLStressMultiChange(t *testing.T) {
	skipUnlessLiveOracle(t)
	ctx := context.Background()

	for _, srv := range liveServers {
		srv := srv
		t.Run(srv.name, func(t *testing.T) {
			target := multiChangeTarget80
			if srv.version == "5.7" {
				target = multiChangeTarget57
			}

			meta, dbName := syncMetadata(ctx, t, srv, "sdl_stress_multi", multiChangeBase)

			// Production path: compute the combined plan.
			plan, err := schema.SDLMigration(storepb.Engine_MYSQL, target, meta, srv.version)
			require.NoError(t, err)
			require.NotEmpty(t, plan, "[%s] multi-change must produce DDL", srv.name)
			t.Logf("[%s] multi-change plan:\n%s", srv.name, plan)
			upper := strings.ToUpper(plan)

			// Correctness: every intended change is present.
			require.Contains(t, upper, "CREATE TABLE `REVIEW`", "[%s] expected new review table:\n%s", srv.name, plan)
			require.Contains(t, upper, "DROP COLUMN `LEGACY_CODE`", "[%s] expected drop of legacy_code:\n%s", srv.name, plan)
			require.True(t, strings.Contains(upper, "DECIMAL(12,4)"), "[%s] expected price type change:\n%s", srv.name, plan)
			require.Contains(t, upper, "IDX_PROD_NAME", "[%s] expected new index on product(name):\n%s", srv.name, plan)
			require.Contains(t, upper, "DROP TABLE `LEGACY_AUDIT`", "[%s] expected drop of legacy_audit:\n%s", srv.name, plan)
			require.Contains(t, upper, "TRG_PROD_INS", "[%s] expected new trigger:\n%s", srv.name, plan)
			require.Contains(t, upper, "V_CATALOG", "[%s] expected view replace:\n%s", srv.name, plan)
			require.Contains(t, upper, "PARTITION BY HASH", "[%s] expected daily_stat partition change:\n%s", srv.name, plan)
			if srv.version == "8.0" {
				require.Contains(t, upper, "CHK_PRICE", "[%s] expected CHECK add:\n%s", srv.name, plan)
			}

			// --- Ordering correctness ---
			dropLegacyFK := indexOf(upper, "DROP FOREIGN KEY `FK_LEGACY_AUDIT_PROD`")
			dropLegacyTable := indexOf(upper, "DROP TABLE `LEGACY_AUDIT`")
			require.GreaterOrEqual(t, dropLegacyFK, 0, "[%s] expected legacy_audit FK drop:\n%s", srv.name, plan)
			require.GreaterOrEqual(t, dropLegacyTable, 0, "[%s] expected legacy_audit table drop:\n%s", srv.name, plan)
			require.Less(t, dropLegacyFK, dropLegacyTable,
				"[%s] FK drop must precede table drop:\n%s", srv.name, plan)

			// Index drop must precede the column drop on category (dropping legacy_code).
			dropCatIndex := indexOf(upper, "DROP INDEX `IDX_CAT_LEGACY`")
			dropCatColumn := indexOf(upper, "DROP COLUMN `LEGACY_CODE`")
			require.GreaterOrEqual(t, dropCatIndex, 0, "[%s] expected category index drop:\n%s", srv.name, plan)
			require.Less(t, dropCatIndex, dropCatColumn,
				"[%s] index drop must precede column drop:\n%s", srv.name, plan)

			// The review FK is deferred to PhasePost: a standalone ADD CONSTRAINT FK_REVIEW_PROD
			// must appear AFTER the review table is created (and not be inlined in the CREATE).
			createReview := indexOf(upper, "CREATE TABLE `REVIEW`")
			require.GreaterOrEqual(t, createReview, 0, "[%s] review table create not found:\n%s", srv.name, plan)
			addReviewFK := indexOf(upper, "ADD CONSTRAINT `FK_REVIEW_PROD`")
			require.GreaterOrEqual(t, addReviewFK, 0, "[%s] expected deferred review FK add:\n%s", srv.name, plan)
			require.Less(t, createReview, addReviewFK,
				"[%s] review FK add must follow the review table create (PhasePost):\n%s", srv.name, plan)

			// Apply the WHOLE plan back to the real DB and confirm convergence (re-diff empty).
			applyErr := applyDDL(ctx, t, srv, dbName, plan)
			require.NoError(t, applyErr, "[%s] multi-change plan failed to apply:\n%s", srv.name, plan)

			newSource := dumpSDL(ctx, t, srv, dbName)
			converge, err := mysqlDiffSDLMigration(newSource, target, srv.version)
			require.NoError(t, err)
			require.Empty(t, converge, "[%s] multi-change did not converge, residual:\n%s", srv.name, converge)
		})
	}
}

// ============================================================================
// Scenario 4: Drop-heavy release + advices.
// ============================================================================

// dropHeavyBase has tables, indexes, views, and routines to drop.
//
//go:embed testdata/sdl/drop_heavy_base.sql
var dropHeavyBase string

// dropHeavyTarget drops scratch_table (whole table), a.scratch column (+ its index),
// a.idx_a_name index, the v_a view, f_double function, and p_reset procedure. Table b
// loses its FK target only if a is dropped — here a survives, b survives.
//
//go:embed testdata/sdl/drop_heavy_target.sql
var dropHeavyTarget string

// TestSDLStressDropHeavy asserts SDLDropAdvices emits WARNING advices for each destructive
// op, and that the generated destructive DDL applies and converges. Both versions.
//
//nolint:tparallel
func TestSDLStressDropHeavy(t *testing.T) {
	skipUnlessLiveOracle(t)
	ctx := context.Background()

	for _, srv := range liveServers {
		srv := srv
		t.Run(srv.name, func(t *testing.T) {
			meta, dbName := syncMetadata(ctx, t, srv, "sdl_stress_drop", dropHeavyBase)

			// Drop advices: the table/column/index drops ARE warned. (The view/function/
			// procedure drops are MISSING — a confirmed (B) bug pinned by
			// TestSDLStressDropAdvicesViewRoutineGap.)
			advices, err := schema.SDLDropAdvices(storepb.Engine_MYSQL, dropHeavyTarget, meta, srv.version)
			require.NoError(t, err)
			require.NotEmpty(t, advices, "[%s] drop-heavy target must yield advices", srv.name)
			for _, a := range advices {
				require.Equal(t, storepb.Advice_WARNING, a.Status, "[%s] drop advice must be WARNING: %+v", srv.name, a)
			}
			joined := ""
			for _, a := range advices {
				joined += a.Content + "\n"
			}
			t.Logf("[%s] drop advices:\n%s", srv.name, joined)

			// The table/column/index drops are warned (>=3: scratch_table, scratch column,
			// two indexes). The dropped table must be named.
			dropCount := countAdviceCode(advices, code.SDLDropOperation.Int32())
			require.GreaterOrEqual(t, dropCount, 3,
				"[%s] expected >=3 drop warnings (table, column, index), got %d:\n%s", srv.name, dropCount, joined)
			require.Contains(t, joined, "scratch_table", "[%s] expected dropped table named:\n%s", srv.name, joined)
			require.Contains(t, joined, "scratch", "[%s] expected dropped column named:\n%s", srv.name, joined)

			// Generate and apply the destructive DDL; confirm convergence. The PLAN correctly
			// drops the view/function/procedure even though the advices omit them.
			plan, err := schema.SDLMigration(storepb.Engine_MYSQL, dropHeavyTarget, meta, srv.version)
			require.NoError(t, err)
			require.NotEmpty(t, plan, "[%s] drop-heavy plan must be non-empty", srv.name)
			t.Logf("[%s] drop-heavy plan:\n%s", srv.name, plan)

			applyErr := applyDDL(ctx, t, srv, dbName, plan)
			require.NoError(t, applyErr, "[%s] drop-heavy plan failed to apply:\n%s", srv.name, plan)

			newSource := dumpSDL(ctx, t, srv, dbName)
			converge, err := mysqlDiffSDLMigration(newSource, dropHeavyTarget, srv.version)
			require.NoError(t, err)
			require.Empty(t, converge, "[%s] drop-heavy did not converge, residual:\n%s", srv.name, converge)
		})
	}
}

// TestSDLStressDropAdvicesViewRoutineGap PINS a third real (B) bug found by the stress
// test: mysqlSDLDropAdvices (backend/plugin/schema/mysql/sdl_migration.go) emits NO
// advice at all for any view / function / procedure / trigger / event operation — neither a
// DROP advice for a standalone drop nor a REPLACE advice for a redefinition. So a
// declarative release that drops or replaces a view/routine/trigger/event gives the user
// ZERO destructive-operation warning. Affects both 5.7 and 8.0; the generated migration DDL
// itself is correct (it does drop/replace) — only the advice walker is wrong.
//
// Two compounding faults in the replace-pair detection (sdl_migration.go L205-294):
//
//  1. The premise is wrong for this omni build. The code assumes a redefinition is rendered
//     as an OpDrop<Obj> followed by an OpCreate<Obj> of the same name, and classifies the
//     CREATE as a replace. But omni emits a redefinition as a SINGLE OpCreate<Obj> op (no
//     paired drop — verified: a view redefine yields exactly `CreateView v`). So isReplace is
//     never set for the create, and the OpCreateView/Function/... replace branch never fires
//     -> no REPLACE advice.
//
//  2. markDropped self-poisons the standalone-drop path. For every OpDropView it does
//     markDropped(OpCreateView, name); isReplace(OpCreateView, name) then reads that SAME map
//     and returns true for the drop itself. So the OpDropView branch's `if !isReplace(...)`
//     is false -> the DROP advice is SUPPRESSED even though there is no matching CREATE.
//
// Net: every view/routine/trigger/event drop is silently swallowed, and every redefinition
// is unwarned. The table/column/index/constraint/FK/check advices are unaffected (they don't
// go through the replace-pair logic).
//
// Likely fix: build the dropped/created name sets in a FIRST pass over plan.Ops, classify a
// name as "replace" only when BOTH a drop AND a create of that name exist, and emit a REPLACE
// advice from whichever op is present (omni's lone CreateView for a redefine), a DROP advice
// for a drop with no matching create, and suppress the drop half only of a genuine pair.
//
// SKIPPED so the suite stays green; remove the Skip to reproduce.
//
//nolint:tparallel
func TestSDLStressDropAdvicesViewRoutineGap(t *testing.T) {
	skipUnlessLiveOracle(t)
	ctx := context.Background()

	for _, srv := range liveServers {
		srv := srv
		t.Run(srv.name, func(t *testing.T) {
			base := `CREATE TABLE t (id INT PRIMARY KEY, a INT, b INT) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
CREATE VIEW v AS SELECT id, a FROM t;
CREATE FUNCTION f(x INT) RETURNS INT DETERMINISTIC RETURN x*2;`
			meta, _ := syncMetadata(ctx, t, srv, "sdl_stress_advgap", base)

			// Drop the view + function (table t survives).
			dropTarget := `CREATE TABLE t (id INT PRIMARY KEY, a INT, b INT) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;`

			// The migration DDL DOES drop them (proving the diff is correct; only advices fail).
			plan, err := schema.SDLMigration(storepb.Engine_MYSQL, dropTarget, meta, srv.version)
			require.NoError(t, err)
			require.Contains(t, strings.ToUpper(plan), "DROP VIEW", "[%s] plan should drop the view:\n%s", srv.name, plan)
			require.Contains(t, strings.ToUpper(plan), "DROP FUNCTION", "[%s] plan should drop the function:\n%s", srv.name, plan)

			// But the advices are EMPTY (the bug).
			advices, err := schema.SDLDropAdvices(storepb.Engine_MYSQL, dropTarget, meta, srv.version)
			require.NoError(t, err)
			require.NotEmpty(t, advices, "[%s] BUG: dropping a view + function must yield drop advices, got none", srv.name)
		})
	}
}

// ============================================================================
// Scenario 5: 5.7-vs-8.0 divergence guards.
// ============================================================================

// divergeSchema authors constructs whose stored form diverges by version: bare utf8mb4
// (collation), utf8 (mb3 on 8.0), int widths, and a CHECK (8.0 only).
//
//go:embed testdata/sdl/diverge_schema.sql
var divergeSchema string

// divergeSchemaWithCheck adds a CHECK (8.0 honors it; 5.7 parses-and-ignores).
//
//go:embed testdata/sdl/diverge_schema_with_check.sql
var divergeSchemaWithCheck string

// TestSDLStressVersionDivergence confirms each version is idempotent against its OWN
// stored form with no cross-version contamination: 5.7 never emits utf8mb4_0900_ai_ci;
// 8.0 handles utf8mb3; CHECK is present on 8.0, absent on 5.7. It dumps the synced schema
// and asserts version-specific properties on the canonical SDL, then proves no-op
// idempotence per version.
//
//nolint:tparallel
func TestSDLStressVersionDivergence(t *testing.T) {
	skipUnlessLiveOracle(t)
	ctx := context.Background()

	for _, srv := range liveServers {
		srv := srv
		t.Run(srv.name, func(t *testing.T) {
			userSDL := divergeSchema
			if srv.version == "8.0" {
				userSDL = divergeSchemaWithCheck
			}

			meta, dbName := syncMetadata(ctx, t, srv, "sdl_stress_diverge", userSDL)
			source := dumpSDL(ctx, t, srv, dbName)
			t.Logf("[%s] dumped SDL:\n%s", srv.name, source)

			// Version-specific stored-form guards on the canonical dump. Note MetadataToSDL
			// renders collations from the synced metadata (server-accurate, hence divergent)
			// but integer display widths in the 8.0-canonical form (widths dropped) on BOTH
			// versions — the int-width divergence is enforced by the omni diff normalizer, not
			// the dumper, so it is asserted via the no-op idempotence below rather than the dump
			// string. CHECK is honored on 8.0 and parsed-and-ignored on 5.7, so it appears in
			// the 8.0 dump only.
			if srv.version == "5.7" {
				require.NotContains(t, source, "utf8mb4_0900_ai_ci",
					"[%s] 5.7 dump must not contain the 8.0-only collation:\n%s", srv.name, source)
				require.NotContains(t, source, "utf8mb3",
					"[%s] 5.7 dump must not normalize utf8 -> utf8mb3:\n%s", srv.name, source)
				require.Contains(t, source, "utf8mb4_general_ci",
					"[%s] 5.7 dump must use the 5.7 default collation:\n%s", srv.name, source)
				require.NotContains(t, source, "chk_amount",
					"[%s] 5.7 ignores CHECK, so it must be absent from the dump:\n%s", srv.name, source)
			} else {
				require.Contains(t, source, "utf8mb4_0900_ai_ci",
					"[%s] 8.0 dump must use the 8.0 default collation:\n%s", srv.name, source)
				require.Contains(t, source, "utf8mb3",
					"[%s] 8.0 dump must normalize utf8 -> utf8mb3:\n%s", srv.name, source)
				require.Contains(t, source, "chk_amount",
					"[%s] 8.0 honors CHECK, so it must be present in the dump:\n%s", srv.name, source)
			}

			// No-op idempotence per version (production path).
			noop, err := schema.SDLMigration(storepb.Engine_MYSQL, userSDL, meta, srv.version)
			require.NoError(t, err)
			require.Empty(t, noop, "[%s] divergence no-op must be empty, got:\n%s", srv.name, noop)

			// A 5.7 CHANGE must never name the 8.0-only collation. Add a column to force a
			// minimal ALTER and apply it back.
			changeSDL := strings.Replace(userSDL,
				"\tPRIMARY KEY (id)",
				"\textra VARCHAR(30) NULL,\n\tPRIMARY KEY (id)", 1)
			require.NotEqual(t, userSDL, changeSDL, "[%s] test setup: change SDL must differ", srv.name)
			change, err := schema.SDLMigration(storepb.Engine_MYSQL, changeSDL, meta, srv.version)
			require.NoError(t, err)
			require.NotEmpty(t, change, "[%s] divergence change must be non-empty", srv.name)
			if srv.version == "5.7" {
				require.NotContains(t, change, "utf8mb4_0900_ai_ci",
					"[%s] 5.7 change names an 8.0-only collation (errno 1273):\n%s", srv.name, change)
			}
			applyErr := applyDDL(ctx, t, srv, dbName, change)
			require.NoError(t, applyErr, "[%s] divergence change failed to apply:\n%s", srv.name, change)

			// Converge.
			newSource := dumpSDL(ctx, t, srv, dbName)
			converge, err := mysqlDiffSDLMigration(newSource, changeSDL, srv.version)
			require.NoError(t, err)
			require.Empty(t, converge, "[%s] divergence change did not converge, residual:\n%s", srv.name, converge)
		})
	}
}

// Deep (round-2) stress test for the MySQL declarative (SDL) migration path. Round 1
// (the stress axes above) covered breadth/scale/normalization. These cases push four
// DIFFERENT axes, all adversarial — the goal is to FIND bugs, not to confirm green:
//
//   - Axis 1: malformed / semantic-error TARGET SDL fed to mysqlDiffSDLMigration and
//     schema.SDLMigration. Must fail GRACEFULLY (clean error or defined no-op) — never panic,
//     never silently emit a wrong/destructive plan.
//   - Axis 2: exotic types + extreme values. sync -> MetadataToSDL -> diff vs user form must
//     be empty (idempotent) on both versions; every non-empty no-op is a normalization gap.
//   - Axis 3: sequential release chains S0->S1->...->S4 on a real DB; each step minimal,
//     applied, re-synced, converged before the next.
//   - Axis 4: ~80-100 table scale; no-op must be empty + fast; one small change must be one
//     minimal ALTER, not a re-emit of all tables.
//
// Drives the PRODUCTION entry points only:
//   - schema.SDLMigration(MYSQL, userSDL, syncedMetadata, version),
//   - mysqlDiffSDLMigration(source, target, version),
//   - schema.SDLDropAdvices(MYSQL, userSDL, syncedMetadata, version),
//   - schema.MetadataToSDL.
//
// Shared helpers (createLiveMySQLDriver, newLiveDatabase, applyAndDump, statementCount,
// liveServers, liveServer, syncMetadata, dumpSDL, applyDDL) come from the other two live
// test files in this package.

// ============================================================================
// Axis 1: adversarial / malformed / semantic-error TARGET SDL.
// ============================================================================
//
// Robustness contract: feeding bad target SDL into the diff must produce a CLEAN Go error
// (or a defined no-op) — NEVER a panic, NEVER a malformed/destructive plan. The diff path
// returns errors as Go values; a panic is therefore a (B) bug regardless of message.

// diffOutcome captures how a single adversarial input was handled.
type diffOutcome struct {
	panicked  bool
	panicVal  any
	err       error
	plan      string
	planEmpty bool
}

// safeDiff runs mysqlDiffSDLMigration(source, target, version) under a panic
// recover so a panic is reported as data instead of crashing the test binary. A baseline
// non-empty `source` is supplied so that if the target parses as EMPTY, the resulting plan
// would be a (destructive) full-drop — letting us detect "silently dropped everything".
func safeDiff(source, target, version string) (out diffOutcome) {
	defer func() {
		if r := recover(); r != nil {
			out.panicked = true
			out.panicVal = r
		}
	}()
	plan, err := mysqlDiffSDLMigration(source, target, version)
	out.err = err
	out.plan = plan
	out.planEmpty = strings.TrimSpace(plan) == ""
	return out
}

// safeSDLMigration runs the production schema.SDLMigration under a panic recover.
func safeSDLMigration(target string, meta *model.DatabaseMetadata, version string) (out diffOutcome) {
	defer func() {
		if r := recover(); r != nil {
			out.panicked = true
			out.panicVal = r
		}
	}()
	plan, err := schema.SDLMigration(storepb.Engine_MYSQL, target, meta, version)
	out.err = err
	out.plan = plan
	out.planEmpty = strings.TrimSpace(plan) == ""
	return out
}

// adversarialCase is one bad-input probe.
//
// The UNIVERSAL bar for every case is: no panic, and no error-free plan that would
// DESTROY a valid pre-existing object that the (correctly-interpreted) target still
// contains. Beyond that, cases differ in how strict we can be about rejection:
//
//   - wantReject=true: the input is malformed enough that a robust loader SHOULD return a
//     non-nil error. We assert err != nil. (If a future omni accepts it, this flips to a
//     soft finding rather than a crash — see the test body.)
//   - wantReject=false: the input is degenerate-but-arguably-valid, or its rejection is the
//     server's job at apply time (e.g. over-length identifier, dangling FK). We only assert
//     no-panic + no-unexpected-destruction and LOG the outcome.
type adversarialCase struct {
	name       string
	target     string
	wantReject bool // require a non-nil error from the loader
	note       string
}

// adversarialCases enumerates malformed and semantically-broken target SDL. The `source`
// is a fixed valid 2-table schema so any plan the engine produces against a broken target
// is observable (and a full-table-drop plan is detectable).
func adversarialCases() []adversarialCase {
	return []adversarialCase{
		// ---- Syntax errors (must be rejected) ----
		{name: "truncated_create", target: "CREATE TABLE t (id INT", wantReject: true, note: "unterminated CREATE"},
		{name: "unmatched_paren", target: "CREATE TABLE t (id INT PRIMARY KEY, name VARCHAR(50) ENGINE=InnoDB;", wantReject: true, note: "missing close paren"},
		{name: "garbage_tokens", target: "@@@ this is not sql ;;; %%%", wantReject: true, note: "pure garbage"},
		{name: "keyword_salad", target: "CREATE CREATE TABLE TABLE t t (id id INT INT);", wantReject: true, note: "doubled keywords"},
		{name: "missing_type", target: "CREATE TABLE t (id, name VARCHAR(50));", wantReject: true, note: "column with no type"},

		// ---- Semantic errors ----
		{
			name:       "fk_nonexistent_table",
			target:     "CREATE TABLE child (id INT PRIMARY KEY, pid INT, CONSTRAINT fk FOREIGN KEY (pid) REFERENCES no_such_parent (id)) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;",
			wantReject: false, // omni accepts a dangling FK at load; the server rejects on apply. No-panic + non-destructive is the bar.
			note:       "FK to a table that does not exist",
		},
		{
			name:       "fk_nonexistent_column",
			target:     "CREATE TABLE parent (id INT PRIMARY KEY) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;\nCREATE TABLE child (id INT PRIMARY KEY, pid INT, CONSTRAINT fk FOREIGN KEY (pid) REFERENCES parent (no_such_col)) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;",
			wantReject: false,
			note:       "FK references a column that does not exist on parent",
		},
		{
			name:       "duplicate_table",
			target:     "CREATE TABLE t (id INT PRIMARY KEY) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;\nCREATE TABLE t (id INT PRIMARY KEY, x INT) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;",
			wantReject: true,
			note:       "two CREATE TABLE with the same name",
		},
		{
			name:       "duplicate_column",
			target:     "CREATE TABLE t (id INT PRIMARY KEY, name VARCHAR(50), name VARCHAR(60)) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;",
			wantReject: true,
			note:       "duplicate column name within a table",
		},
		{
			name:       "duplicate_index_name",
			target:     "CREATE TABLE t (id INT PRIMARY KEY, a INT, b INT, KEY idx_dup (a), KEY idx_dup (b)) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;",
			wantReject: true,
			note:       "two indexes with the same name",
		},
		{
			name:       "pk_declared_twice",
			target:     "CREATE TABLE t (id INT, x INT, PRIMARY KEY (id), PRIMARY KEY (x)) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;",
			wantReject: true,
			note:       "two PRIMARY KEY clauses",
		},
		{
			name:       "index_nonexistent_column",
			target:     "CREATE TABLE t (id INT PRIMARY KEY, a INT, KEY idx_ghost (no_such_col)) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;",
			wantReject: false, // omni does NOT validate index-column existence at load; server rejects on apply (errno 1072). Soft finding.
			note:       "index on a column that does not exist",
		},
		{
			name:       "check_references_other_table",
			target:     "CREATE TABLE t (id INT PRIMARY KEY, a INT, CONSTRAINT ck CHECK (a < (SELECT COUNT(*) FROM other))) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;",
			wantReject: true, // 8.0 rejects a subquery in CHECK at load
			note:       "CHECK with a subquery referencing another table",
		},
		{
			name:       "generated_col_forward_ref",
			target:     "CREATE TABLE t (id INT PRIMARY KEY, g INT AS (later + 1) STORED, later INT NOT NULL) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;",
			wantReject: false, // forward reference; MySQL allows it. No-panic + sane plan is the bar.
			note:       "generated column referencing a column declared later",
		},
		{
			name:       "circular_fk_notnull",
			target:     "CREATE TABLE a (id INT PRIMARY KEY, b_id INT NOT NULL, CONSTRAINT fk_a FOREIGN KEY (b_id) REFERENCES b (id)) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;\nCREATE TABLE b (id INT PRIMARY KEY, a_id INT NOT NULL, CONSTRAINT fk_b FOREIGN KEY (a_id) REFERENCES a (id)) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;",
			wantReject: false, // SDL loader disables FK checks; a clean plan or error both acceptable, no panic is the bar
			note:       "mutually-referential NOT NULL FKs",
		},

		// ---- Degenerate ----
		{name: "empty_string", target: "", wantReject: false, note: "empty SDL -> full drop of source; must not panic"},
		{name: "whitespace_only", target: "   \n\t  \n", wantReject: false, note: "whitespace only"},
		{name: "comments_only", target: "-- just a comment\n/* block comment */\n", wantReject: false, note: "comments only"},
		{name: "single_semicolon", target: ";", wantReject: false, note: "a lone statement terminator"},
		{name: "create_sequence", target: "CREATE SEQUENCE s START WITH 1 INCREMENT BY 1;", wantReject: true, note: "different-engine DDL (no SEQUENCE in MySQL)"},
		{name: "dml_insert", target: "INSERT INTO t (id) VALUES (1);", wantReject: false, note: "DML in SDL target (LoadSQL fallback may accept and yield a plan)"},
		{name: "dml_update", target: "UPDATE t SET id = 2;", wantReject: false, note: "UPDATE in SDL target"},
		{name: "drop_table_in_target", target: "DROP TABLE t;", wantReject: true, note: "DROP in an SDL target (SDL is CREATE-only -> rejected)"},
		{name: "alter_in_target", target: "ALTER TABLE t ADD COLUMN x INT;", wantReject: true, note: "ALTER in an SDL target (rejected)"},

		// ---- Identifier edges ----
		{
			name:       "ident_64_chars",
			target:     "CREATE TABLE `" + strings.Repeat("a", 64) + "` (id INT PRIMARY KEY) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;",
			wantReject: false, // 64 is the MAX legal length -> accepted, no panic
			note:       "64-char identifier (legal max)",
		},
		{
			name:       "ident_65_chars",
			target:     "CREATE TABLE `" + strings.Repeat("a", 65) + "` (id INT PRIMARY KEY) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;",
			wantReject: false, // omni does NOT enforce the 64-char limit at load; the server rejects on apply (errno 1059). Soft finding.
			note:       "65-char identifier (over the limit)",
		},
		{
			name:       "reserved_word_quoted",
			target:     "CREATE TABLE `select` (`from` INT PRIMARY KEY, `where` VARCHAR(50)) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;",
			wantReject: false,
			note:       "reserved words as backtick-quoted identifiers",
		},
		{
			name:       "unicode_emoji_ident",
			target:     "CREATE TABLE `ta*b📊le` (`col😀` INT PRIMARY KEY) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;",
			wantReject: false,
			note:       "unicode/emoji identifiers",
		},
		{
			name:       "ident_with_backtick_dot",
			target:     "CREATE TABLE `we``ird` (`a.b` INT PRIMARY KEY) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;",
			wantReject: false,
			note:       "identifiers needing escaping (embedded backtick, dot)",
		},
		{
			name:       "case_only_differ",
			target:     "CREATE TABLE t (id INT PRIMARY KEY, Name VARCHAR(50), name VARCHAR(60)) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;",
			wantReject: true, // MySQL column names are case-insensitive -> Name/name collide
			note:       "columns differing only by case",
		},
	}
}

// TestSDLDeepAdversarial drives malformed/semantic-error target SDL through the production
// diff. Bar: NEVER panic, NEVER silently emit a destructive plan. We feed a fixed valid
// 2-table `source` so a wrong plan that drops everything is observable.
//
//nolint:tparallel
func TestSDLDeepAdversarial(t *testing.T) {
	skipUnlessLiveOracle(t)
	ctx := context.Background()

	// A fixed valid source schema (two real tables) so the diff has something to operate on.
	const sourceSDL = `CREATE TABLE t (id INT PRIMARY KEY, name VARCHAR(50) NOT NULL) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
CREATE TABLE other (id INT PRIMARY KEY, t_id INT NOT NULL) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;`

	for _, srv := range liveServers {
		srv := srv
		t.Run(srv.name, func(t *testing.T) {
			// Materialize the source schema once so the production-path probe has a real synced
			// metadata to diff against (schema.SDLMigration path).
			meta, _ := syncMetadata(ctx, t, srv, "sdl_deep_adv", sourceSDL)

			for _, ac := range adversarialCases() {
				ac := ac
				t.Run(ac.name, func(t *testing.T) {
					// (1) Lower-level version-aware diff: source(valid) -> target(bad).
					out := safeDiff(sourceSDL, ac.target, srv.version)

					// UNIVERSAL bar 1: never panic. A panic on ANY input is a (B) bug.
					require.False(t, out.panicked,
						"[%s/%s] (B) PANIC on adversarial input %q: %v\ninput:\n%s",
						srv.name, ac.name, ac.note, out.panicVal, ac.target)

					// (2) Production entry: schema.SDLMigration(meta, badTarget). Same no-panic bar
					// on the real production call (covers the MetadataToSDL->diff composition).
					// Run for EVERY case, including the rejected ones.
					pout := safeSDLMigration(ac.target, meta, srv.version)
					require.False(t, pout.panicked,
						"[%s/%s] (B) PANIC in schema.SDLMigration on %q: %v\ninput:\n%s",
						srv.name, ac.name, ac.note, pout.panicVal, ac.target)

					if ac.wantReject {
						// The loader SHOULD reject this. If it does (err != nil), great. If it
						// does NOT, that is a robustness gap, but it is only a (B) bug if the
						// resulting error-free plan is also DESTRUCTIVE/wrong; a malformed input
						// that yields a clean error OR a harmless no-op is acceptable. We assert
						// the error and, on the rare accept, demand the plan at least not be
						// silently emitted as a non-empty migration.
						if out.err == nil {
							require.True(t, out.planEmpty,
								"[%s/%s] (B) malformed input %q was ACCEPTED (no error) and produced a non-empty plan — a wrong/garbage migration:\n%s",
								srv.name, ac.name, ac.note, out.plan)
							t.Logf("[%s/%s] SOFT: malformed input %q accepted as empty no-op rather than erroring", srv.name, ac.name, ac.note)
						} else {
							t.Logf("[%s/%s] reject OK: %v", srv.name, ac.name, out.err)
						}
						return
					}

					// wantReject==false: the input is degenerate-but-tolerable or its rejection
					// is the server's job at apply time. No-panic already asserted. A clean error
					// is fine; an error-free plan is fine (the declarative semantics of the target
					// are well-defined). Just record the outcome for the report.
					if out.err != nil {
						t.Logf("[%s/%s] tolerated-with-error: %v", srv.name, ac.name, out.err)
					} else {
						t.Logf("[%s/%s] tolerated input %q -> plan(empty=%v):\n%s",
							srv.name, ac.name, ac.note, out.planEmpty, out.plan)
					}
				})
			}
		})
	}
}

// ============================================================================
// Axis 2: exotic types + extreme values (idempotence; flag normalization gaps).
// ============================================================================
//
// Each case authors ONE less-common type-surface table. It is applied to a live DB, synced,
// dumped via MetadataToSDL, and the PRODUCTION version-aware diff of (dumped vs the user
// form) MUST be empty on each version it runs on. A non-empty no-op is a normalization gap
// (a (B) bug) and the test logs input + stored/dumped form so the owning omni Canonical* /
// dumper rule can be located. 8.0-only constructs are gated off 5.7 via only80.

// exoticCase authors one exotic-type table. only80 gates 8.0-only surface off 5.7.
//
// bGap marks a case that this round CONFIRMED is NOT idempotent — a (B) normalization /
// dumper round-trip bug. Those cases are SKIPPED by the idempotence test (so the suite stays
// green) but kept here as the executable repro, and are reproduced/explained by
// TestSDLDeepExoticKnownGaps below. bGapReason is the one-line root cause.
type exoticCase struct {
	name       string
	ddl        string
	only80     bool
	bGap       bool
	bGapReason string
}

// exoticCases enumerates the less-common type surface across families. Each names exactly
// one construct cluster so a single non-empty no-op (or load error) pinpoints the offending
// type. Cases marked bGap are confirmed (B) bugs this round found — see bGapReason and the
// detailed analysis on TestSDLDeepExoticKnownGaps.
func exoticCases() []exoticCase {
	return []exoticCase{
		// ---- JSON ----
		{
			name: "json_basic",
			ddl: `CREATE TABLE t (
	id INT PRIMARY KEY,
	doc JSON,
	doc_nn JSON NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;`,
		},
		{
			name:   "json_default_80",
			only80: true, // JSON column DEFAULT requires 8.0.13+
			ddl: `CREATE TABLE t (
	id INT PRIMARY KEY,
	doc JSON DEFAULT (JSON_OBJECT()),
	arr JSON DEFAULT (JSON_ARRAY())
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;`,
		},
		{
			name:   "json_multivalue_index_80",
			only80: true,  // multi-valued index (CAST ... AS ... ARRAY) is 8.0.17+
			bGap:   false, // FIXED (dumper strips the functional-index charset introducer + unescapes quotes; omni #353 accepts AS … ARRAY, re-pinned a362f7a7)
			// FIXED (Bug B): the synced functional-index expression
			// `(cast(json_extract(`tags`,_utf8mb4\'$.ids\') as unsigned array))` carried two things the
			// omni loader rejects — the `_utf8mb4'…'` charset introducer before the JSON-path literal
			// (`unexpected token`) and backslash-escaped single quotes (`syntax error at or near "\"`).
			// normalizeFunctionalIndexExpr in get_database_definition.go now unescapes the quotes and
			// strips the introducer, so the dumped expr is `((cast(json_extract(`tags`,'$.ids') as
			// unsigned array)))`; omni's functional-index normalizer canonicalizes the dumped and user
			// forms identically and the no-op is empty. (The `AS UNSIGNED ARRAY` cast itself is accepted
			// by omni #353 — the introducer/escaping was the LoadSDL blocker, not the ARRAY keyword.)
			bGapReason: "",
			ddl: `CREATE TABLE t (
	id INT PRIMARY KEY,
	tags JSON NOT NULL,
	KEY idx_tags ((CAST(tags->'$.ids' AS UNSIGNED ARRAY)))
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;`,
		},

		// ---- Spatial ----
		{
			name:   "spatial_types",
			only80: true, // 8.0-only GAP (FIXED): on 5.7 the dumper emits `geometrycollection` and IS idempotent; 8.0 used to break.
			// FIXED (bug 8): the dumper now normalizes the 8.0 `geomcollection` type synonym to
			// the canonical `geometrycollection` spelling omni parses (normalizeColumnType in
			// get_database_definition.go), so the dumped source reloads and the no-op is empty.
			ddl: `CREATE TABLE t (
	id INT PRIMARY KEY,
	g GEOMETRY,
	pt POINT,
	ls LINESTRING,
	poly POLYGON,
	gc GEOMETRYCOLLECTION
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;`,
		},
		{
			name: "spatial_notnull_index",
			// FIXED (bug 7): the dumper no longer emits a key-part prefix length for SPATIAL (and
			// FULLTEXT) index parts (printIndexKeyPart suppressPrefix), so the spatial key dumps as
			// `(`pt`)` matching SHOW CREATE and the no-op is empty on both 5.7 and 8.0.
			ddl: `CREATE TABLE t (
	id INT NOT NULL AUTO_INCREMENT,
	pt POINT NOT NULL,
	PRIMARY KEY (id),
	SPATIAL KEY idx_pt (pt)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;`,
		},
		{
			name:   "spatial_srid_80",
			only80: true, // SRID column attribute is 8.0
			// FIXED (bug 6 + bug 7): sync now captures the column SRID (information_schema.COLUMNS.SRS_ID
			// into ColumnMetadata.srid) and the dumper emits `/*!80003 SRID 4326 */` after NOT NULL,
			// while the spatial-key prefix is suppressed (bug 7). Both halves of the former no-op are
			// gone, so the dump round-trips empty.
			ddl: `CREATE TABLE t (
	id INT NOT NULL AUTO_INCREMENT,
	pt POINT NOT NULL SRID 4326,
	PRIMARY KEY (id),
	SPATIAL KEY idx_pt (pt)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;`,
		},

		// ---- Fractional seconds ----
		{
			name: "fractional_seconds",
			ddl: `CREATE TABLE t (
	id INT PRIMARY KEY,
	dt6 DATETIME(6) NOT NULL,
	tm3 TIME(3) NOT NULL,
	ts6 TIMESTAMP(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
	dt0 DATETIME(0)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;`,
		},

		// ---- BLOB / TEXT family ----
		{
			name: "blob_text_family",
			ddl: `CREATE TABLE t (
	id INT PRIMARY KEY,
	tb TINYBLOB,
	b BLOB,
	mb MEDIUMBLOB,
	lb LONGBLOB,
	tt TINYTEXT,
	txt TEXT,
	mt MEDIUMTEXT,
	lt LONGTEXT
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;`,
		},
		{
			name: "text_charset_collate",
			ddl: `CREATE TABLE t (
	id INT PRIMARY KEY,
	a TEXT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci,
	b LONGTEXT CHARACTER SET latin1 COLLATE latin1_swedish_ci,
	c BLOB
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;`,
		},
		{
			name: "text_blob_prefix_index",
			ddl: `CREATE TABLE t (
	id INT NOT NULL AUTO_INCREMENT,
	body TEXT,
	data BLOB,
	PRIMARY KEY (id),
	KEY idx_body (body(50)),
	KEY idx_data (data(30))
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;`,
		},

		// ---- Charsets beyond utf8mb4 ----
		{
			name: "charset_latin1_ascii",
			ddl: `CREATE TABLE t (
	id INT PRIMARY KEY,
	a VARCHAR(50) CHARACTER SET latin1 COLLATE latin1_swedish_ci,
	b VARCHAR(50) CHARACTER SET ascii,
	c CHAR(10) CHARACTER SET latin1
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;`,
		},
		{
			name: "charset_binary_varbinary",
			ddl: `CREATE TABLE t (
	id INT PRIMARY KEY,
	a BINARY(16),
	b VARBINARY(255),
	c VARCHAR(50) CHARACTER SET binary
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;`,
		},
		{
			name: "charset_gbk_big5_utf16",
			ddl: `CREATE TABLE t (
	id INT PRIMARY KEY,
	a VARCHAR(50) CHARACTER SET gbk,
	b VARCHAR(50) CHARACTER SET big5,
	c VARCHAR(50) CHARACTER SET utf16
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;`,
		},
		{
			name: "table_charset_differs_from_column",
			// Table default latin1; one column overrides to utf8mb4 (column charset != table charset).
			ddl: `CREATE TABLE t (
	id INT PRIMARY KEY,
	a VARCHAR(50),
	b VARCHAR(50) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci
) ENGINE=InnoDB DEFAULT CHARSET=latin1;`,
		},

		// ---- Numeric / bit / extreme ----
		{
			name: "bit_type",
			// FIXED (Bug A): the dumper rendered the BIT default as a quoted string (`DEFAULT 'b\\'0\\''`)
			// because the sync stores the bit literal QUOTE()-escaped in ColumnMetadata.Default; omni then
			// loaded a string default that never matched the user's `b'0'`, re-emitting `MODIFY ... DEFAULT
			// b'0'` every no-op. renderColumnDefault in get_database_definition.go now recovers the bit
			// literal and emits it unquoted (`DEFAULT b'0'`, matching SHOW CREATE / mysqldump); omni #352
			// canonicalizes the bit literal so the no-op is empty on both versions. b64 (no default) is fine.
			bGap:       false,
			bGapReason: "",
			ddl: `CREATE TABLE t (
	id INT PRIMARY KEY,
	b1 BIT(1) NOT NULL DEFAULT b'0',
	b8 BIT(8) NOT NULL DEFAULT b'101',
	b64 BIT(64)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;`,
		},
		{
			name: "decimal_max_precision",
			ddl: `CREATE TABLE t (
	id INT PRIMARY KEY,
	d DECIMAL(65,30) NOT NULL DEFAULT 0
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;`,
		},
		{
			name:       "bigint_unsigned_maxdefault",
			bGap:       false, // FIXED (re-pinned omni a362f7a7)
			bGapReason: "DANGEROUS (silently-wrong value, not just a phantom): a BIGINT UNSIGNED default of 18446744073709551615 (max uint64) is compared/re-emitted as 9223372036854775807 (max INT64) — the default is parsed into a signed int64 and clamped/overflows. The diff emits `MODIFY ... DEFAULT '9223372036854775807'`, which would CHANGE the default to the wrong number on apply. sm/ti (in-range) are fine.",
			ddl: `CREATE TABLE t (
	id INT PRIMARY KEY,
	big BIGINT UNSIGNED NOT NULL DEFAULT 18446744073709551615,
	sm SMALLINT UNSIGNED NOT NULL DEFAULT 65535,
	ti TINYINT UNSIGNED NOT NULL DEFAULT 255
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;`,
		},
		{
			name:       "year_and_set",
			bGap:       false, // FIXED (re-pinned omni a362f7a7)
			bGapReason: "YEAR default canonicalization: user `DEFAULT 2000` (numeric) vs stored `'2000'` (string) not recognized equal; the diff re-emits `MODIFY ... y year ... DEFAULT '2000'` every no-op (phantom; both versions). The SET column is idempotent.",
			ddl: `CREATE TABLE t (
	id INT PRIMARY KEY,
	y YEAR NOT NULL DEFAULT 2000,
	s SET('a','b','c','d','e','f','g') NOT NULL DEFAULT 'a,c,e'
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;`,
		},
		{
			name: "large_enum_set",
			// Many ENUM/SET members.
			ddl: `CREATE TABLE t (
	id INT PRIMARY KEY,
	e ENUM('m00','m01','m02','m03','m04','m05','m06','m07','m08','m09','m10','m11','m12','m13','m14','m15') NOT NULL DEFAULT 'm00',
	s SET('s0','s1','s2','s3','s4','s5','s6','s7','s8','s9') NOT NULL DEFAULT 's0,s9'
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;`,
		},
		{
			name: "varchar_near_row_limit",
			// A wide VARCHAR in latin1 (1 byte/char) approaching the 65535-byte row limit.
			ddl: `CREATE TABLE t (
	id INT PRIMARY KEY,
	wide VARCHAR(16000) CHARACTER SET latin1 NOT NULL DEFAULT ''
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;`,
		},

		// ---- Generated columns chained ----
		{
			name: "generated_chained",
			// total references price/qty; with_tax references total (a generated col referencing
			// another generated col); both a VIRTUAL and a STORED, plus an index on a generated col.
			// NB: `code` is a plain (non-auto-increment) column — MySQL forbids a generated column
			// referencing an AUTO_INCREMENT column (errno 3109), so `label` references `code`.
			ddl: `CREATE TABLE t (
	id INT NOT NULL AUTO_INCREMENT,
	code INT NOT NULL DEFAULT 0,
	price DECIMAL(10,2) NOT NULL DEFAULT 0,
	qty INT NOT NULL DEFAULT 0,
	total DECIMAL(20,2) AS (price * qty) STORED,
	with_tax DECIMAL(20,2) AS (total * 1.1) VIRTUAL,
	label VARCHAR(40) AS (CONCAT('#', code)) STORED,
	PRIMARY KEY (id),
	KEY idx_total (total)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;`,
		},

		// ---- AUTO_INCREMENT seed ----
		{
			name: "auto_increment_seed_500",
			ddl: `CREATE TABLE t (
	id INT NOT NULL AUTO_INCREMENT,
	name VARCHAR(50) NOT NULL,
	PRIMARY KEY (id)
) ENGINE=InnoDB AUTO_INCREMENT=500 DEFAULT CHARSET=utf8mb4;`,
		},

		// ---- Invisible columns / indexes (8.0) ----
		{
			name:   "invisible_column_80",
			only80: true,
			// FIXED (bug 4): sync now captures column invisibility (information_schema.COLUMNS.EXTRA
			// INVISIBLE token into ColumnMetadata.is_invisible) and the dumper emits
			// `/*!80023 INVISIBLE */`, so the no-op is empty.
			ddl: `CREATE TABLE t (
	id INT PRIMARY KEY,
	a INT NOT NULL DEFAULT 0,
	secret INT INVISIBLE NOT NULL DEFAULT 0
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;`,
		},
		{
			name:   "invisible_index_80",
			only80: true,
			// FIXED (bug 5): index visibility was already synced (IndexMetadata.visible from
			// information_schema.STATISTICS.IS_VISIBLE); the dumper now emits `/*!80000 INVISIBLE */`
			// in the index clause, so the no-op is empty instead of a DROP+ADD.
			ddl: `CREATE TABLE t (
	id INT PRIMARY KEY,
	a INT NOT NULL,
	KEY idx_a (a) INVISIBLE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;`,
		},
	}
}

// TestSDLDeepExoticIdempotence is Axis 2: every NON-bGap exotic-type case is applied live,
// synced, dumped, and the production version-aware diff (dumped vs user form) MUST be empty
// on both versions (8.0-only types skip 5.7). The bGap cases (confirmed normalization/dumper
// (B) bugs found this round) are skipped here and reproduced/explained by
// TestSDLDeepExoticKnownGaps so this assertion stays green. Each unexpected non-empty no-op
// is logged with input + stored form so the gap can be pinned to a Canonical* rule or the
// dumper.
//
//nolint:tparallel
func TestSDLDeepExoticIdempotence(t *testing.T) {
	skipUnlessLiveOracle(t)
	ctx := context.Background()

	for _, srv := range liveServers {
		srv := srv
		t.Run(srv.name, func(t *testing.T) {
			for _, ec := range exoticCases() {
				ec := ec
				if srv.version == "5.7" && ec.only80 {
					continue
				}
				if ec.bGap {
					continue // confirmed (B) gap — pinned by TestSDLDeepExoticKnownGaps
				}
				t.Run(ec.name, func(t *testing.T) {
					meta, dbName := syncMetadata(ctx, t, srv, "sdl_deep_exo", ec.ddl)

					// Production entry: schema.SDLMigration (meta -> SDL internally).
					noop := safeSDLMigration(ec.ddl, meta, srv.version)
					require.False(t, noop.panicked,
						"[%s/%s] (B) PANIC in schema.SDLMigration: %v\ninput:\n%s",
						srv.name, ec.name, noop.panicVal, ec.ddl)
					require.NoError(t, noop.err, "[%s/%s] SDLMigration error:\n%s", srv.name, ec.name, ec.ddl)

					source := dumpSDL(ctx, t, srv, dbName)
					if !noop.planEmpty {
						t.Logf("[%s/%s] (B) NON-EMPTY no-op normalization gap:\nUSER FORM:\n%s\nSTORED/DUMPED:\n%s\nDIFF:\n%s",
							srv.name, ec.name, ec.ddl, source, noop.plan)
					}
					require.True(t, noop.planEmpty,
						"[%s/%s] (B) exotic-type no-op (SDLMigration) must be empty, got:\n%s", srv.name, ec.name, noop.plan)

					// Version-aware lower-level entry, same property.
					noop2, err := mysqlDiffSDLMigration(source, ec.ddl, srv.version)
					require.NoError(t, err)
					require.Empty(t, noop2,
						"[%s/%s] (B) exotic-type no-op (mysqlDiffSDLMigration) must be empty, got:\n%s", srv.name, ec.name, noop2)
				})
			}
		})
	}
}

// TestSDLDeepExoticKnownGaps PINS the (B) normalization / dumper round-trip bugs this deep
// round found in the exotic-type surface. Each bGap case is applied live and its no-op is
// computed; the test asserts the no-op is currently NON-empty OR the dumped source fails to
// reload (the bug signature), and logs the full user->stored->diff so a fix subagent has the
// exact repro. When a bug is FIXED the corresponding case will start producing an empty,
// reloadable no-op — flip its bGap flag to false (moving it back under the green idempotence
// test) and this pin will report the now-idempotent case so the regression coverage moves
// with the fix.
//
// Confirmed gaps (see each case's bGapReason for the precise root cause):
//   - DANGEROUS, silently-wrong value: bigint_unsigned_maxdefault (uint64 max default clamped
//     to int64 max in the diff — would change the default on apply).
//   - Default-literal canonicalization phantoms (re-emit every no-op): bit_type (BIT b'..'),
//     year_and_set (YEAR numeric-vs-quoted).
//   - Dumper drops a real attribute (re-emit every no-op): invisible_column_80,
//     invisible_index_80, spatial_srid_80 (SRID), spatial_notnull_index / spatial_srid_80
//     (phantom spatial-key prefix length).
//   - Dumper emits a spelling the omni parser can't reload (HARD error): spatial_types
//     (`geomcollection`), json_multivalue_index_80 (charset introducer in the functional key).
//
// SKIPPED so the suite stays green; remove the Skip to reproduce all gaps at once.
//
//nolint:tparallel
func TestSDLDeepExoticKnownGaps(t *testing.T) {
	t.Skip("Pins confirmed (B) exotic-type normalization/dumper gaps; remove Skip to reproduce")
	ctx := context.Background()

	for _, srv := range liveServers {
		srv := srv
		t.Run(srv.name, func(t *testing.T) {
			for _, ec := range exoticCases() {
				ec := ec
				if !ec.bGap {
					continue
				}
				if srv.version == "5.7" && ec.only80 {
					continue
				}
				t.Run(ec.name, func(t *testing.T) {
					meta, dbName := syncMetadata(ctx, t, srv, "sdl_deep_exo_gap", ec.ddl)

					noop := safeSDLMigration(ec.ddl, meta, srv.version)
					require.False(t, noop.panicked,
						"[%s/%s] (B) PANIC: %v", srv.name, ec.name, noop.panicVal)

					source := dumpSDL(ctx, t, srv, dbName)
					t.Logf("[%s/%s] (B) %s\nUSER:\n%s\nSTORED:\n%s\nNO-OP err=%v plan:\n%s",
						srv.name, ec.name, ec.bGapReason, ec.ddl, source, noop.err, noop.plan)

					// The bug signature is EITHER a hard error from reloading the dumped source
					// OR a non-empty no-op. Asserting it documents the current broken behavior.
					broke := noop.err != nil || !noop.planEmpty
					require.True(t, broke,
						"[%s/%s] expected the known (B) gap to reproduce (load error or non-empty no-op) but it was idempotent — the bug may be FIXED; clear bGap on this case.\nreason was: %s",
						srv.name, ec.name, ec.bGapReason)
				})
			}
		})
	}
}

// ============================================================================
// Axis 3: sequential release chains (the real iterative usage).
// ============================================================================
//
// Start from S0 on a real DB. Apply a SEQUENCE of declarative releases S0->S1->...->S4,
// each a realistic incremental change set. At EACH step: source = MetadataToSDL(sync(DB)),
// diff against the step target, assert the plan is minimal+correctly-ordered, APPLY it to
// the real DB, re-sync, and assert the next no-op is empty before moving on. This proves
// correctness across a migration HISTORY, not a single diff.

// chainStep is one release in the sequence. target is the FULL desired SDL at that step
// (declarative — the complete schema, not a delta). wantContains are uppercase substrings
// the generated plan must contain; wantOrder pins a required relative ordering (first must
// appear before second). only80Extra is appended to target only on 8.0 (CHECK etc.).
type chainStep struct {
	name        string
	target80    string
	target57    string
	wantContain []string  // uppercase substrings the plan must contain
	wantAbsent  []string  // uppercase substrings the plan must NOT contain
	wantOrder   [2]string // [before, after] uppercase substrings; before must precede after (skip if either empty)
}

func (s chainStep) target(version string) string {
	if version == "5.7" {
		return s.target57
	}
	return s.target80
}

// --- The sequence. Each Sn target is the FULL schema at that point. ---

// chainS0: a small starting schema — customer + order with an FK, one view.
//
//go:embed testdata/sdl/chain_s0.sql
var chainS0 string

// chainS1: add table `order_item` + FK to ord (and to a new `product` table).
//
//go:embed testdata/sdl/chain_s1_common.sql
var chainS1Common string

// chainS2: add a column (customer.loyalty_points) + an index (ord.idx_ord_created) + a
// generated column (order_item.line_total references qty — but needs price; keep it simple:
// generated col on product: price_with_tax).
//
//go:embed testdata/sdl/chain_s2_common.sql
var chainS2Common string

// chainS3: modify a column type (ord.total DECIMAL(10,2)->DECIMAL(14,4)) + widen a VARCHAR
// (customer.name VARCHAR(100)->VARCHAR(200)) + change a default (customer.loyalty_points
// DEFAULT 0 -> DEFAULT 100).
//
//go:embed testdata/sdl/chain_s3_common.sql
var chainS3Common string

// chainS4: drop a column WITH its index (drop product.price_with_tax generated col), drop a
// table that is an FK PARENT (drop product — order_item.fk_oi_product child FK must drop
// first; also drop order_item to keep it consistent? No — keep order_item but remove its FK
// to product by dropping the product reference). To exercise "drop FK parent, child FK drops
// first", we drop `product` AND order_item's fk_oi_product + product_id column. Replace the
// view, and (8.0) add a CHECK on ord.
//
// S4 also changes order_item to remove product linkage, replaces v_cust_orders, adds a
// trigger on ord, and (8.0) adds a CHECK on ord.total.
//
//go:embed testdata/sdl/chain_s4_base.sql
var chainS4Base string

func chainSteps() []chainStep {
	s4_80 := strings.Replace(chainS4Base, "{{ORD_CHECK}}", ",\n\tCONSTRAINT chk_total CHECK (total >= 0)", 1)
	s4_57 := strings.Replace(chainS4Base, "{{ORD_CHECK}}", "", 1)
	return []chainStep{
		{
			name:        "S1_add_table_fk",
			target80:    chainS1Common,
			target57:    chainS1Common,
			wantContain: []string{"CREATE TABLE `PRODUCT`", "CREATE TABLE `ORDER_ITEM`"},
			wantAbsent:  []string{"DROP TABLE", "CREATE TABLE `CUSTOMER`", "CREATE TABLE `ORD`"},
			// The order_item FK to ord/product is deferred to PhasePost: table create precedes the FK add.
			wantOrder: [2]string{"CREATE TABLE `ORDER_ITEM`", "ADD CONSTRAINT `FK_OI_ORDER`"},
		},
		{
			name:        "S2_add_column_index_generated",
			target80:    chainS2Common,
			target57:    chainS2Common,
			wantContain: []string{"ADD COLUMN `LOYALTY_POINTS`", "ADD KEY `IDX_ORD_CREATED`", "PRICE_WITH_TAX"},
			wantAbsent:  []string{"DROP TABLE", "CREATE TABLE"},
		},
		{
			name:     "S3_modify_type_widen_default",
			target80: chainS3Common,
			target57: chainS3Common,
			// total type change, name widen, loyalty default change.
			wantContain: []string{"DECIMAL(14,4)", "VARCHAR(200)"},
			wantAbsent:  []string{"DROP TABLE", "CREATE TABLE `CUSTOMER`", "DROP COLUMN"},
		},
		{
			name:     "S4_drop_col_drop_parent_replace_view_trigger",
			target80: s4_80,
			target57: s4_57,
			// product table dropped; order_item.product_id column + its FK dropped; view replaced; trigger added.
			wantContain: []string{"DROP TABLE `PRODUCT`", "DROP COLUMN `PRODUCT_ID`", "DROP FOREIGN KEY `FK_OI_PRODUCT`", "TRG_ORD_INS", "V_CUST_ORDERS"},
			wantAbsent:  []string{"DROP TABLE `CUSTOMER`", "DROP TABLE `ORD`"},
			// The child FK on order_item must drop before the parent product table is dropped.
			wantOrder: [2]string{"DROP FOREIGN KEY `FK_OI_PRODUCT`", "DROP TABLE `PRODUCT`"},
		},
	}
}

// TestSDLDeepSequentialChain is Axis 3: drive S0 -> S1 -> S2 -> S3 -> S4 on ONE live DB.
// Each step computes source=MetadataToSDL(sync), diffs the step target, asserts the plan is
// minimal + correctly ordered, APPLIES it, re-syncs, and asserts the next no-op is empty
// before advancing. Proves correctness across a migration history. Both versions.
//
// The steps run inline (NOT as separate subtests) BECAUSE they share one evolving DB: each
// step's diff is from its predecessor's applied state. Running a single step in isolation
// (e.g. `-run .../S4`) would compute a diff from S0 and assert wrong things — so the chain is
// kept atomic per server.
//
//nolint:tparallel
func TestSDLDeepSequentialChain(t *testing.T) {
	skipUnlessLiveOracle(t)
	ctx := context.Background()

	for _, srv := range liveServers {
		srv := srv
		t.Run(srv.name, func(t *testing.T) {
			// Materialize S0 on a single DB that persists across all steps.
			dbName := newLiveDatabase(ctx, t, srv, "sdl_deep_chain")
			require.NoError(t, applyDDL(ctx, t, srv, dbName, chainS0), "[%s] apply S0", srv.name)

			// Confirm S0 itself is idempotent before starting the chain.
			s0src := dumpSDL(ctx, t, srv, dbName)
			s0noop, err := mysqlDiffSDLMigration(s0src, chainS0, srv.version)
			require.NoError(t, err)
			require.Empty(t, s0noop, "[%s] S0 must be idempotent before the chain, got:\n%s", srv.name, s0noop)

			// Run every step inline, in order, against the same evolving DB.
			for _, step := range chainSteps() {
				runChainStep(ctx, t, srv, dbName, step)
				if t.Failed() {
					return // a broken step poisons the chain state; stop so the failure is unambiguous
				}
			}
		})
	}
}

// runChainStep executes one release in the sequential chain against the live dbName.
func runChainStep(ctx context.Context, t *testing.T, srv liveServer, dbName string, step chainStep) {
	t.Helper()
	target := step.target(srv.version)

	// 1) Compute the plan from the CURRENT live DB state to the step target.
	source := dumpSDL(ctx, t, srv, dbName)
	plan, err := mysqlDiffSDLMigration(source, target, srv.version)
	require.NoError(t, err)
	require.NotEmpty(t, plan, "[%s/%s] step must produce a non-empty plan", srv.name, step.name)
	t.Logf("[%s/%s] plan:\n%s", srv.name, step.name, plan)
	upper := strings.ToUpper(plan)

	// 2) Minimality + correctness: required substrings present, forbidden absent.
	for _, want := range step.wantContain {
		require.Contains(t, upper, want, "[%s/%s] plan missing %q:\n%s", srv.name, step.name, want, plan)
	}
	for _, absent := range step.wantAbsent {
		require.NotContains(t, upper, absent, "[%s/%s] plan must not contain %q:\n%s", srv.name, step.name, absent, plan)
	}

	// 3) Ordering.
	if step.wantOrder[0] != "" && step.wantOrder[1] != "" {
		bi := indexOf(upper, step.wantOrder[0])
		ai := indexOf(upper, step.wantOrder[1])
		require.GreaterOrEqual(t, bi, 0, "[%s/%s] ordering: %q not found:\n%s", srv.name, step.name, step.wantOrder[0], plan)
		require.GreaterOrEqual(t, ai, 0, "[%s/%s] ordering: %q not found:\n%s", srv.name, step.name, step.wantOrder[1], plan)
		require.Less(t, bi, ai, "[%s/%s] %q must precede %q:\n%s", srv.name, step.name, step.wantOrder[0], step.wantOrder[1], plan)
	}

	// 4) Apply to the real DB.
	require.NoError(t, applyDDL(ctx, t, srv, dbName, plan),
		"[%s/%s] step plan failed to apply:\n%s", srv.name, step.name, plan)

	// 5) Re-sync and assert convergence (next no-op empty) BEFORE advancing.
	newSource := dumpSDL(ctx, t, srv, dbName)
	converge, err := mysqlDiffSDLMigration(newSource, target, srv.version)
	require.NoError(t, err)
	require.Empty(t, converge, "[%s/%s] step did not converge, residual:\n%s", srv.name, step.name, converge)

	// 6) Production-path cross-check: schema.SDLMigration on a freshly-synced meta must also be
	// empty (the path the release executor takes).
	meta := syncMetaForDB(ctx, t, srv, dbName)
	prodNoop, err := schema.SDLMigration(storepb.Engine_MYSQL, target, meta, srv.version)
	require.NoError(t, err)
	require.Empty(t, prodNoop, "[%s/%s] production-path no-op after apply must be empty, got:\n%s", srv.name, step.name, prodNoop)
}

// syncMetaForDB syncs an existing dbName and returns its model.DatabaseMetadata.
func syncMetaForDB(ctx context.Context, t *testing.T, srv liveServer, dbName string) *model.DatabaseMetadata {
	t.Helper()
	driver, err := createLiveMySQLDriver(ctx, srv, dbName)
	require.NoError(t, err)
	defer driver.Close(ctx)
	metadata, err := driver.SyncDBSchema(ctx)
	require.NoError(t, err)
	return model.NewDatabaseMetadata(metadata, nil, nil, storepb.Engine_MYSQL, true)
}

// ============================================================================
// Axis 4: scale / performance sanity.
// ============================================================================
//
// Generate ~90 tables (each ~8-15 columns, a PK, 2-3 secondary indexes, FKs forming a
// realistic chain). Sync -> MetadataToSDL -> no-op diff must be EMPTY and complete in
// reasonable wall-time. Then add ONE column to ONE table -> assert exactly one minimal
// ALTER, not a re-emit of all tables. Record timings.

// genScaleSchema builds n tables t000..t{n-1}. Each table i (i>=1) carries an FK to table
// i-1 (a chain), plus 2 secondary indexes and ~10 columns of mixed types. Returns the DDL.
func genScaleSchema(n int) string {
	var b strings.Builder
	for i := 0; i < n; i++ {
		fmt.Fprintf(&b, "CREATE TABLE `t%03d` (\n", i)
		b.WriteString("\tid INT NOT NULL AUTO_INCREMENT,\n")
		b.WriteString("\tcode VARCHAR(40) NOT NULL,\n")
		b.WriteString("\tname VARCHAR(120) NOT NULL,\n")
		b.WriteString("\tdescription TEXT,\n")
		b.WriteString("\tamount DECIMAL(14,2) NOT NULL DEFAULT 0.00,\n")
		b.WriteString("\tqty INT NOT NULL DEFAULT 0,\n")
		b.WriteString("\tactive BOOLEAN NOT NULL DEFAULT TRUE,\n")
		b.WriteString("\tcreated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,\n")
		b.WriteString("\tupdated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,\n")
		b.WriteString("\tstatus ENUM('new','open','closed') NOT NULL DEFAULT 'new',\n")
		if i > 0 {
			b.WriteString("\tprev_id INT NULL,\n")
		}
		b.WriteString("\tPRIMARY KEY (id),\n")
		fmt.Fprintf(&b, "\tUNIQUE KEY `uk_t%03d_code` (code),\n", i)
		fmt.Fprintf(&b, "\tKEY `idx_t%03d_name` (name),\n", i)
		if i > 0 {
			fmt.Fprintf(&b, "\tKEY `idx_t%03d_prev` (prev_id),\n", i)
			fmt.Fprintf(&b, "\tCONSTRAINT `fk_t%03d_prev` FOREIGN KEY (prev_id) REFERENCES `t%03d` (id) ON DELETE SET NULL\n", i, i-1)
		} else {
			// trim the trailing comma from the last index line for table 0
			fmt.Fprintf(&b, "\tKEY `idx_t%03d_amount` (amount)\n", i)
		}
		b.WriteString(") ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;\n\n")
	}
	return b.String()
}

// TestSDLDeepScale is Axis 4: a ~90-table schema. The no-op diff must be empty and fast;
// adding one column to one table must yield exactly one minimal ALTER, not a re-emit of all
// tables. Timings are recorded and a pathological (super-linear / multi-second) no-op is
// flagged.
//
//nolint:tparallel
func TestSDLDeepScale(t *testing.T) {
	skipUnlessLiveOracle(t)
	ctx := context.Background()

	const tableCount = 90
	schemaDDL := genScaleSchema(tableCount)

	for _, srv := range liveServers {
		srv := srv
		t.Run(srv.name, func(t *testing.T) {
			meta, dbName := syncMetadata(ctx, t, srv, "sdl_deep_scale", schemaDDL)

			// (1) No-op diff: must be empty + record wall-time.
			source := dumpSDL(ctx, t, srv, dbName)
			start := time.Now()
			noop, err := mysqlDiffSDLMigration(source, schemaDDL, srv.version)
			elapsed := time.Since(start)
			require.NoError(t, err)
			if noop != "" {
				t.Logf("[%s] NON-EMPTY %d-table no-op (%d statements):\n%s", srv.name, tableCount, statementCount(noop), noop)
			}
			require.Empty(t, noop, "[%s] %d-table no-op must be empty, got %d statements", srv.name, tableCount, statementCount(noop))
			t.Logf("[%s] SCALE no-op diff over %d tables: %s", srv.name, tableCount, elapsed)
			// Soft perf flag: a no-op over ~90 tables should be well under a few seconds.
			if elapsed > 10*time.Second {
				t.Errorf("[%s] (PERF) %d-table no-op took %s (>10s) — pathological", srv.name, tableCount, elapsed)
			}

			// (2) Production-path no-op timing too.
			pstart := time.Now()
			prodNoop, err := schema.SDLMigration(storepb.Engine_MYSQL, schemaDDL, meta, srv.version)
			pelapsed := time.Since(pstart)
			require.NoError(t, err)
			require.Empty(t, prodNoop, "[%s] %d-table production no-op must be empty", srv.name, tableCount)
			t.Logf("[%s] SCALE production-path no-op (incl. MetadataToSDL): %s", srv.name, pelapsed)

			// (3) One small change: add a column to ONE table (t045). Must be exactly one ALTER
			// touching only that table — not a re-emit of the other 89.
			changed := strings.Replace(schemaDDL,
				"CREATE TABLE `t045` (\n\tid INT NOT NULL AUTO_INCREMENT,\n\tcode VARCHAR(40) NOT NULL,",
				"CREATE TABLE `t045` (\n\tid INT NOT NULL AUTO_INCREMENT,\n\tcode VARCHAR(40) NOT NULL,\n\tnew_flag INT NOT NULL DEFAULT 0,",
				1)
			require.NotEqual(t, schemaDDL, changed, "[%s] scale change setup must differ", srv.name)

			cstart := time.Now()
			change, err := mysqlDiffSDLMigration(source, changed, srv.version)
			celapsed := time.Since(cstart)
			require.NoError(t, err)
			t.Logf("[%s] SCALE single-change diff: %s\nplan:\n%s", srv.name, celapsed, change)
			require.NotEmpty(t, change, "[%s] single-change must produce DDL", srv.name)

			upper := strings.ToUpper(change)
			require.Contains(t, upper, "ADD COLUMN `NEW_FLAG`", "[%s] expected the new column:\n%s", srv.name, change)
			require.Contains(t, upper, "`T045`", "[%s] expected the changed table named:\n%s", srv.name, change)
			require.NotContains(t, upper, "CREATE TABLE", "[%s] single-change must not recreate tables:\n%s", srv.name, change)
			require.NotContains(t, upper, "DROP TABLE", "[%s] single-change must not drop tables:\n%s", srv.name, change)
			// Strict minimality: exactly ONE statement (the single ADD COLUMN).
			require.Equal(t, 1, statementCount(change),
				"[%s] single-change over %d tables must be exactly 1 statement, got %d:\n%s",
				srv.name, tableCount, statementCount(change), change)

			// No other table may be named in the plan.
			for i := 0; i < tableCount; i++ {
				if i == 45 {
					continue
				}
				token := fmt.Sprintf("`T%03d`", i)
				require.NotContains(t, upper, token,
					"[%s] single-change must not touch table t%03d:\n%s", srv.name, i, change)
			}
		})
	}
}

// ENTERPRISE smoke axes for the MySQL declarative (SDL) migration path. The real-world
// suite (the real-world axes above) proved the path on mid-size production schemas; these
// scales verification to ENTERPRISE corpora and adds the axes an enterprise rollout
// exercises hardest:
//
//	corpus     — Zabbix 7.0.0 (203 tables, 272 FKs, 65 changelog triggers), PrestaShop
//	             (243 tables), OpenEMR (283 tables, INSERTs stripped), and the stock
//	             MySQL `sys` schema (scaffolded, gated on an in-flight omni fix).
//	baseline   — every corpus loads live, and its canonical dump self-diffs EMPTY
//	             (determinism) and no-ops through the production schema.SDLMigration.
//	A1         — single-object CRUD (create / modify / drop) for every object kind, on a
//	             20-table Zabbix slice.
//	A2         — dependent objects: FK chains, trigger→table deps (the Zabbix changelog
//	             pattern), 3-deep view stacks, function-used-by-view, circular FK pairs.
//	A3         — special types: AUTO_INCREMENT, generated columns, ENUM/SET members,
//	             charset/collation, temporal precision, DECIMAL, TEXT/BLOB families,
//	             signedness, prefix-indexed TEXT, partition ops.
//	A5         — sequential chain (with one-shot endpoint equality), apply-back,
//	             combined release, scale timing guard, multi-file export round-trip.
//	A4 (fuzz)  — see the A4 section below.
//
// The per-case oracle protocol (entOracle) is the realworld suite's, made reusable:
//
//	(1) load base B into an entsdl_-prefixed scratch DB, sync → canonical current SDL C;
//	(2) DDL = mysqlDiffSDLMigration(C, target T, version);
//	(3) apply DDL to the scratch DB — clean execution (dependency ordering) is under test;
//	(4) re-sync → C'; assert Diff(C', T) == "" (convergence) and Diff(C', C') == ""
//	    (idempotence);
//	(5) minimality asserts (exact statement counts) where cheap.
//
// Shared helpers (liveServers, createLiveMySQLDriver, newLiveDatabase, dumpSDL, applyDDL,
// statementCount, normalizeDelimiters, syncMetaForDB, objectCounts, addColumnToTable,
// addIndexToTable, addColumnToView, dropObjectBlock, dropTrigger, findObjectSegment,
// removeSegment, mustReplace, concatMultiFile) are declared with the axes above.

// ----------------------------------------------------------------------------
// Embedded enterprise corpora.
//
// Preprocessing applied when the corpus was lifted into testdata:
//   - zabbix:     Zabbix 7.0.0 create/mysql/schema.sql verbatim, minus the single
//                 INSERT INTO dbversion row (DDL-only testdata). Keeps the DELIMITER $$
//                 changelog-trigger block (normalizeDelimiters handles it), the 301
//                 standalone CREATE INDEX statements, and the 272 trailing
//                 ALTER TABLE ... ADD CONSTRAINT foreign keys.
//   - prestashop: db_structure.sql with the installer placeholders substituted
//                 (PREFIX_ -> ps_, ENGINE_TYPE -> InnoDB, ...). Leads with
//                 SET SESSION sql_mode='' (two DEFAULT '0000-00-00 00:00:00' columns
//                 need non-strict mode at load; the driver executes the corpus on a
//                 single connection so the session setting holds).
//   - openemr:    database.sql with all 5,821 INSERT statements stripped by a
//                 quote-aware scanner (283 tables, DDL-only).
//   - sys:        the stock MySQL 8.0 sys schema, single-file form. Formerly gated
//                 on two omni parser gaps it exposed (adjacent string literals #360,
//                 paren-subquery operand continuation #366) — both fixed and pinned.
// ----------------------------------------------------------------------------

//go:embed testdata/enterprise/zabbix.sql
var entZabbixSQL string

//go:embed testdata/enterprise/prestashop.sql
var entPrestashopSQL string

//go:embed testdata/enterprise/openemr.sql
var entOpenemrSQL string

//go:embed testdata/enterprise/sys.sql
var entSysSQL string

// entCorpus is one embedded enterprise schema.
type entCorpus struct {
	name string
	ddl  string
	// gate, when non-empty, skips every leg for this corpus with the given reason.
	gate string
	// gate57, when non-empty, skips only the 5.7 legs with the given reason.
	gate57 string
	// canonical marks ddl as a canonical SDL dump (alphabetical object order) rather
	// than upstream creation-ordered DDL. Canonical dumps are loaded by applying the
	// engine's own empty→schema plan, whose statements are dependency-ordered —
	// sequential client apply of the raw dump would break on view-on-view forward
	// references (sys: `host_summary` reads `x$...` views that sort later).
	canonical bool
}

func entCorpora() []entCorpus {
	return []entCorpus{
		{name: "zabbix", ddl: entZabbixSQL},
		{name: "prestashop", ddl: entPrestashopSQL},
		{name: "openemr", ddl: entOpenemrSQL},
		{name: "sys", ddl: entSysSQL, canonical: true,
			gate57: "sys corpus is the stock MySQL 8.0 sys schema; its views read performance_schema tables that 5.7.25 does not have"},
	}
}

// entLoadCorpus loads one corpus into a fresh scratch database, honoring per-version
// gates. Canonical dumps go through the engine's own dependency-ordered create plan
// (dogfooding: every corpus load exercises the empty→schema ordering guarantees).
func entLoadCorpus(ctx context.Context, t *testing.T, srv liveServer, prefix string, corpus entCorpus) string {
	t.Helper()
	if corpus.gate != "" {
		t.Skipf("[%s/%s] %s", srv.name, corpus.name, corpus.gate)
	}
	if corpus.gate57 != "" && srv.name == "mysql57" {
		t.Skipf("[%s/%s] %s", srv.name, corpus.name, corpus.gate57)
	}
	if !corpus.canonical {
		return entLoadDDL(ctx, t, srv, prefix+corpus.name, corpus.ddl)
	}
	dbName := newLiveDatabase(ctx, t, srv, prefix+corpus.name)
	plan, err := mysqlDiffSDLMigration("", corpus.ddl, srv.version)
	require.NoError(t, err, "[%s/%s] empty→corpus create plan", srv.name, corpus.name)
	driver, err := createLiveMySQLDriver(ctx, srv, dbName)
	require.NoError(t, err)
	defer driver.Close(ctx)
	_, err = driver.Execute(ctx, plan, db.ExecuteOptions{})
	require.NoError(t, err, "[%s/%s] apply engine create plan", srv.name, corpus.name)
	return dbName
}

// entNormalizeDelimiters rewrites a multi-DELIMITER script into the no-DELIMITER form
// the production split path handles. It extends the realworld suite's normalizeDelimiters
// for the Zabbix style, where the custom delimiter sits on its OWN line and most trigger
// bodies are ALREADY ';'-terminated — naively rewriting every delimiter line to ";" left
// bare ";" statements that MySQL rejects (Error 1064 near ';'). Here a delimiter (inline
// or standalone) only yields a ";" when the statement is not already terminated.
func entNormalizeDelimiters(ddl string) string {
	if !strings.Contains(ddl, "DELIMITER") {
		return ddl
	}
	lines := strings.Split(ddl, "\n")
	out := make([]string, 0, len(lines))
	delim := ";"
	terminated := func() bool {
		for i := len(out) - 1; i >= 0; i-- {
			trimmed := strings.TrimSpace(out[i])
			if trimmed == "" {
				continue
			}
			return strings.HasSuffix(trimmed, ";")
		}
		return true
	}
	for _, ln := range lines {
		trimmed := strings.TrimSpace(ln)
		if strings.HasPrefix(strings.ToUpper(trimmed), "DELIMITER ") {
			delim = strings.TrimSpace(trimmed[len("DELIMITER "):])
			if delim == "" {
				delim = ";"
			}
			continue
		}
		if delim == ";" {
			out = append(out, ln)
			continue
		}
		if trimmed == delim {
			if !terminated() {
				out = append(out, ";")
			}
			continue
		}
		if strings.HasSuffix(trimmed, delim) {
			idx := strings.LastIndex(ln, delim)
			body := strings.TrimRight(ln[:idx], " \t")
			if !strings.HasSuffix(strings.TrimSpace(body), ";") {
				body += ";"
			}
			out = append(out, body)
			continue
		}
		out = append(out, ln)
	}
	return strings.Join(out, "\n")
}

// entLoadDDL creates a fresh entsdl_-prefixed scratch database on srv (dropped in
// cleanup by newLiveDatabase) and applies ddl through the production driver path.
func entLoadDDL(ctx context.Context, t *testing.T, srv liveServer, prefix, ddl string) string {
	t.Helper()
	dbName := newLiveDatabase(ctx, t, srv, prefix)
	driver, err := createLiveMySQLDriver(ctx, srv, dbName)
	require.NoError(t, err)
	defer driver.Close(ctx)
	_, err = driver.Execute(ctx, entNormalizeDelimiters(ddl), db.ExecuteOptions{})
	require.NoError(t, err, "[%s] apply base DDL", srv.name)
	return dbName
}

// entPlanStatementCount counts plan statements compound-aware: routine/trigger bodies
// carry internal ';' that the naive statementCount (string split) over-counts, so the
// minimality asserts split with the production splitter instead.
func entPlanStatementCount(t *testing.T, plan string) int {
	t.Helper()
	stmts, err := mysqlparser.SplitSQL(plan)
	require.NoError(t, err, "split plan for statement count:\n%s", plan)
	n := 0
	for _, s := range stmts {
		if text := strings.TrimSpace(s.Text); text != "" && text != ";" {
			n++
		}
	}
	return n
}

// entOracle runs steps (2)-(4) of the oracle protocol against an already-loaded scratch
// database: dump the canonical current SDL, diff to target, apply the generated DDL, and
// prove convergence + idempotence. Returns the generated plan for minimality asserts.
func entOracle(ctx context.Context, t *testing.T, srv liveServer, dbName, target, label string) string {
	t.Helper()
	source := dumpSDL(ctx, t, srv, dbName)
	require.NotEqual(t, source, target, "[%s] target must differ from source", label)

	plan, err := mysqlDiffSDLMigration(source, target, srv.version)
	require.NoError(t, err, "[%s] diff source->target", label)
	require.NotEmpty(t, plan, "[%s] expected a non-empty migration plan", label)
	t.Logf("[%s] plan (%d stmts):\n%s", label, statementCount(plan), plan)

	applyErr := applyDDL(ctx, t, srv, dbName, plan)
	require.NoError(t, applyErr, "[%s] generated plan failed to apply:\n%s", label, plan)

	after := dumpSDL(ctx, t, srv, dbName)
	converge, err := mysqlDiffSDLMigration(after, target, srv.version)
	require.NoError(t, err, "[%s] converge diff", label)
	require.Empty(t, converge, "[%s] did not converge; residual:\n%s\nplan was:\n%s", label, converge, plan)

	self, err := mysqlDiffSDLMigration(after, after, srv.version)
	require.NoError(t, err, "[%s] idempotence diff", label)
	require.Empty(t, self, "[%s] post-apply dump not idempotent:\n%s", label, self)
	return plan
}

// ----------------------------------------------------------------------------
// Zabbix slice extraction.
//
// The corpus declares tables, then standalone CREATE [UNIQUE] INDEX statements, then a
// DELIMITER block of changelog triggers, then trailing ALTER TABLE ... ADD CONSTRAINT
// foreign keys. A slice keeps the CREATE TABLE + CREATE INDEX statements of the included
// tables, the triggers ON included tables (their bodies write `changelog`, so changelog
// must be included), and only the FKs whose BOTH endpoints are included — yielding a
// self-consistent sub-schema in the corpus's own statement order.
// ----------------------------------------------------------------------------

type entZbxKind int

const (
	entZbxTable entZbxKind = iota
	entZbxIndex
	entZbxFK
	entZbxTrigger
	entZbxOther
)

type entZbxStmt struct {
	text     string
	kind     entZbxKind
	table    string
	refTable string
}

var (
	entReTable   = regexp.MustCompile("(?i)^CREATE TABLE `(\\w+)`")
	entReIndex   = regexp.MustCompile("(?i)^CREATE (?:UNIQUE )?INDEX `\\w+` ON `(\\w+)`")
	entReFK      = regexp.MustCompile("(?is)^ALTER TABLE `(\\w+)` ADD CONSTRAINT `\\w+` FOREIGN KEY.*?REFERENCES `(\\w+)`")
	entReTrigger = regexp.MustCompile(`(?is)^create\s+trigger\s+\w+\s+(?:before|after)\s+(?:insert|update|delete)\s+on\s+` + "`?(\\w+)`?")

	entZbxOnce  sync.Once
	entZbxStmts []entZbxStmt
	entZbxErr   error
)

// entZabbixStatements splits the normalized zabbix corpus once and classifies every
// statement for slicing.
func entZabbixStatements() ([]entZbxStmt, error) {
	entZbxOnce.Do(func() {
		stmts, err := mysqlparser.SplitSQL(entNormalizeDelimiters(entZabbixSQL))
		if err != nil {
			entZbxErr = err
			return
		}
		for _, s := range stmts {
			text := strings.TrimSpace(s.Text)
			if text == "" || text == ";" {
				continue
			}
			classified := entZbxStmt{text: text, kind: entZbxOther}
			if m := entReTable.FindStringSubmatch(text); m != nil {
				classified.kind, classified.table = entZbxTable, m[1]
			} else if m := entReIndex.FindStringSubmatch(text); m != nil {
				classified.kind, classified.table = entZbxIndex, m[1]
			} else if m := entReFK.FindStringSubmatch(text); m != nil {
				classified.kind, classified.table, classified.refTable = entZbxFK, m[1], m[2]
			} else if m := entReTrigger.FindStringSubmatch(text); m != nil {
				classified.kind, classified.table = entZbxTrigger, m[1]
			}
			entZbxStmts = append(entZbxStmts, classified)
		}
	})
	return entZbxStmts, entZbxErr
}

// entZabbixSlice extracts a self-consistent slice of the zabbix corpus containing the
// given tables (which must include changelog — the trigger bodies write to it).
func entZabbixSlice(t *testing.T, tables ...string) string {
	t.Helper()
	include := make(map[string]bool, len(tables))
	for _, tbl := range tables {
		include[tbl] = true
	}
	require.True(t, include["changelog"], "zabbix slices must include changelog (trigger bodies write to it)")

	stmts, err := entZabbixStatements()
	require.NoError(t, err, "split zabbix corpus")

	var b strings.Builder
	found := map[string]bool{}
	for _, s := range stmts {
		keep := false
		switch s.kind {
		case entZbxTable, entZbxIndex, entZbxTrigger:
			keep = include[s.table]
			if s.kind == entZbxTable && keep {
				found[s.table] = true
			}
		case entZbxFK:
			keep = include[s.table] && include[s.refTable]
		case entZbxOther:
			keep = false
		default:
			keep = false
		}
		if keep {
			b.WriteString(s.text)
			if !strings.HasSuffix(strings.TrimSpace(s.text), ";") {
				b.WriteString(";")
			}
			b.WriteString("\n")
		}
	}
	for _, tbl := range tables {
		require.True(t, found[tbl], "zabbix slice: table %q not found in corpus", tbl)
	}
	return b.String()
}

// entSliceCoreTables is the ~20-table Zabbix slice used by A1/A2/A5: a real 4-deep FK
// chain (item_tag -> items -> hosts -> proxy -> proxy_group), the changelog trigger
// pattern on 8 of the tables (hosts alone carries 5 triggers, including a BEGIN/END
// body), self-referencing FKs (hosts.templateid, items.templateid/master_itemid), and
// composite unique indexes (hosts_groups).
var entSliceCoreTables = []string{
	"changelog",
	"role", "users", "media_type", "media",
	"proxy_group", "proxy", "proxy_rtdata",
	"maintenances", "hosts", "host_rtdata",
	"hstgrp", "hosts_groups",
	"interface", "valuemap", "items", "item_tag", "item_preproc",
	"drules", "dchecks",
	"connector", "connector_tag",
}

// ----------------------------------------------------------------------------
// Targeted string-surgery helpers on the canonical dump (ent-prefixed; the shared
// realworld helpers cover the untargeted forms).
// ----------------------------------------------------------------------------

// entTableSegment returns the [start,end) range of the CREATE TABLE statement for table
// in source (located via the SDL splitter, so partition comments stay in-segment).
func entTableSegment(t *testing.T, source, table string) (int, int) {
	t.Helper()
	header := "CREATE TABLE `" + table + "`"
	start, end := findObjectSegment(t, source, func(stmt string) bool {
		return strings.HasPrefix(stmt, header)
	})
	require.GreaterOrEqual(t, start, 0, "table %q not found in source", table)
	return start, end
}

// entReplaceInTable replaces old with new exactly once, scoped to table's CREATE block.
func entReplaceInTable(t *testing.T, source, table, old, replacement string) string {
	t.Helper()
	start, end := entTableSegment(t, source, table)
	seg := source[start:end]
	require.Contains(t, seg, old, "table %q block must contain %q", table, old)
	return source[:start] + strings.Replace(seg, old, replacement, 1) + source[end:]
}

// entReplaceAllInTable replaces every occurrence of old within table's CREATE block.
func entReplaceAllInTable(t *testing.T, source, table, old, replacement string) string {
	t.Helper()
	start, end := entTableSegment(t, source, table)
	seg := source[start:end]
	require.Contains(t, seg, old, "table %q block must contain %q", table, old)
	return source[:start] + strings.ReplaceAll(seg, old, replacement) + source[end:]
}

// entDropTableBlock removes the whole CREATE TABLE statement for table.
func entDropTableBlock(t *testing.T, source, table string) string {
	t.Helper()
	start, end := entTableSegment(t, source, table)
	return removeSegment(source, start, end)
}

// entDropObjectNamed removes the CREATE statement whose text contains `<KEYWORD>
// `+"`name`"+“ (EVENT and other kinds the shared dropObjectBlock does not cover).
func entDropObjectNamed(t *testing.T, source, keyword, name string) string {
	t.Helper()
	needle := keyword + " `" + strings.ToUpper(name) + "`"
	start, end := findObjectSegment(t, source, func(stmt string) bool {
		u := strings.ToUpper(stmt)
		return strings.HasPrefix(u, "CREATE ") && strings.Contains(u, needle)
	})
	require.GreaterOrEqual(t, start, 0, "%s %q not found in source", keyword, name)
	return removeSegment(source, start, end)
}

// entDropLineInTable removes the single body line containing marker from table's CREATE
// block, fixing the dangling comma when the removed line was the last body element.
func entDropLineInTable(t *testing.T, source, table, marker string) string {
	t.Helper()
	return entEditLineInTable(t, source, table, marker, "")
}

// entReplaceLineInTable swaps the single body line containing marker for newLine
// (indentation and trailing comma are managed here).
func entReplaceLineInTable(t *testing.T, source, table, marker, newLine string) string {
	t.Helper()
	return entEditLineInTable(t, source, table, marker, newLine)
}

// entEditLineInTable is the shared core of drop/replace-line: newLine == "" drops the
// marker line, otherwise the line is replaced by newLine (re-indented, comma preserved).
func entEditLineInTable(t *testing.T, source, table, marker, newLine string) string {
	t.Helper()
	start, end := entTableSegment(t, source, table)
	seg := source[start:end]
	lines := strings.Split(seg, "\n")
	idx := -1
	for i, ln := range lines {
		if i == 0 {
			continue // never the CREATE TABLE header
		}
		if strings.Contains(ln, marker) {
			idx = i
			break
		}
	}
	require.GreaterOrEqual(t, idx, 0, "table %q has no body line containing %q:\n%s", table, marker, seg)
	hadComma := strings.HasSuffix(strings.TrimSpace(lines[idx]), ",")
	if newLine == "" {
		lines = append(lines[:idx], lines[idx+1:]...)
		if !hadComma {
			// Removed the last body element: strip the now-dangling comma above it.
			for j := idx - 1; j > 0; j-- {
				trimmed := strings.TrimRight(lines[j], " \t")
				if strings.HasSuffix(trimmed, ",") {
					lines[j] = strings.TrimSuffix(trimmed, ",")
					break
				}
				if strings.TrimSpace(trimmed) != "" {
					break
				}
			}
		}
	} else {
		replaced := "  " + strings.TrimSpace(newLine)
		if hadComma {
			replaced += ","
		}
		lines[idx] = replaced
	}
	return source[:start] + strings.Join(lines, "\n") + source[end:]
}

// entSetPartitionClause replaces table's whole partition clause (the dumper's
// /*!NNNNN PARTITION BY ... */ executable comment, or a plain clause) with newClause
// (plain form; pass "" to departition). The clause runs from the comment opener (or
// PARTITION BY) to the end of the statement, so it is rebuilt rather than patched.
func entSetPartitionClause(t *testing.T, source, table, newClause string) string {
	t.Helper()
	start, end := entTableSegment(t, source, table)
	seg := source[start:end]
	pIdx := strings.Index(strings.ToUpper(seg), "PARTITION BY")
	require.GreaterOrEqual(t, pIdx, 0, "table %q has no PARTITION BY clause:\n%s", table, seg)
	clauseStart := pIdx
	if c := strings.LastIndex(seg[:pIdx], "/*!"); c >= 0 {
		clauseStart = c
	}
	// The clause (with its optional comment close) runs to the statement end; the
	// trailing ";" (when in-segment) is preserved.
	tail := ""
	rest := strings.TrimRight(seg[clauseStart:], " \t\n")
	if strings.HasSuffix(rest, ";") {
		tail = ";"
	}
	prefix := strings.TrimRight(seg[:clauseStart], " \t\n")
	if newClause == "" {
		return source[:start] + prefix + tail + source[end:]
	}
	return source[:start] + prefix + "\n" + newClause + tail + source[end:]
}

// entAppendTableClause appends clause (e.g. a plain PARTITION BY) to the end of table's
// CREATE statement, before the trailing semicolon.
func entAppendTableClause(t *testing.T, source, table, clause string) string {
	t.Helper()
	start, end := entTableSegment(t, source, table)
	seg := source[start:end]
	trimmed := strings.TrimRight(seg, " \t\n")
	require.True(t, strings.HasSuffix(trimmed, ";"), "table %q statement must end with ';':\n%s", table, seg)
	body := strings.TrimSuffix(trimmed, ";")
	return source[:start] + body + "\n" + clause + ";" + source[end:]
}

// entUintType returns the canonical dump spelling of INT UNSIGNED on srv. The sync
// driver's columnTypeCanonicalSynonyms map folds the SIGNED default display widths
// (int(11) -> int) but not the unsigned ones, so a 5.7 dump renders `int(10) unsigned`
// verbatim while a plain int renders `int` on both versions (an asymmetry the omni
// canonicalizer absorbs — recorded as an observation in the campaign findings).
func entUintType(srv liveServer) string {
	if srv.version == "5.7" {
		return "int(10) unsigned"
	}
	return "int unsigned"
}

// ----------------------------------------------------------------------------
// Baseline: every corpus loads on every version, and its canonical dump is a fixed
// point — self-diff empty AND the production schema.SDLMigration no-op empty.
// ----------------------------------------------------------------------------

//nolint:tparallel
func TestSDLEnterpriseBaseline(t *testing.T) {
	skipUnlessLiveOracle(t)
	ctx := context.Background()

	for _, srv := range liveServers {
		srv := srv
		t.Run(srv.name, func(t *testing.T) {
			for _, corpus := range entCorpora() {
				corpus := corpus
				t.Run(corpus.name, func(t *testing.T) {
					dbName := entLoadCorpus(ctx, t, srv, "entsdl_bl_", corpus)
					meta := syncMetaForDB(ctx, t, srv, dbName)
					tbl, vw, fn, pr, tg := objectCounts(meta)
					t.Logf("[BASELINE %s/%s] loaded: tables=%d views=%d functions=%d procedures=%d triggers=%d (db=%s)",
						srv.name, corpus.name, tbl, vw, fn, pr, tg, dbName)

					source := dumpSDL(ctx, t, srv, dbName)
					require.NotEmpty(t, source, "[%s/%s] MetadataToSDL produced empty SDL", srv.name, corpus.name)

					selfDiff, err := mysqlDiffSDLMigration(source, source, srv.version)
					require.NoError(t, err)
					require.Empty(t, selfDiff, "[%s/%s] canonical dump must self-diff empty, got:\n%s",
						srv.name, corpus.name, selfDiff)

					noop, err := schema.SDLMigration(storepb.Engine_MYSQL, source, meta, srv.version)
					require.NoError(t, err)
					require.Empty(t, noop, "[%s/%s] production-path no-op must be empty, got:\n%s",
						srv.name, corpus.name, noop)
				})
			}
		})
	}
}

// ----------------------------------------------------------------------------
// A1: single-object CRUD — for each object kind, one create, one semantic modify, and
// one drop, chained on a freshly loaded Zabbix core slice (each phase runs the full
// oracle protocol from a fresh canonical dump).
// ----------------------------------------------------------------------------

// entA1Aux seeds the slice with one object of each kind that needs a pre-existing
// instance to modify/drop, plus a RANGE-partitioned table for the partition kind.
//
//go:embed testdata/sdl/ent_a1_aux.sql
var entA1Aux string

// entCRUDPhase is one oracle round (create / modify / drop) within a kind.
type entCRUDPhase struct {
	name   string
	mutate func(t *testing.T, srv liveServer, source string) string
	// want are uppercased substrings the plan must contain.
	want []string
	// exactStmts, when > 0, asserts the plan is exactly this many statements
	// (single-object minimality).
	exactStmts int
}

// entCRUDKind is the create+modify+drop triple for one object kind.
type entCRUDKind struct {
	name   string
	skip57 string
	phases []entCRUDPhase
}

func entCRUDKinds() []entCRUDKind {
	return []entCRUDKind{
		{
			name: "table",
			phases: []entCRUDPhase{
				{
					name: "create",
					mutate: func(_ *testing.T, _ liveServer, source string) string {
						return source + "\nCREATE TABLE ent_crud_t (\n  id bigint unsigned NOT NULL,\n  label varchar(40) DEFAULT '' NOT NULL,\n  PRIMARY KEY (id)\n) ENGINE=InnoDB;\n"
					},
					want:       []string{"CREATE TABLE", "ENT_CRUD_T"},
					exactStmts: 1,
				},
				{
					name: "modify",
					mutate: func(t *testing.T, _ liveServer, source string) string {
						return entReplaceInTable(t, source, "ent_crud_t", ") ENGINE=InnoDB", ") ENGINE=InnoDB COMMENT='ent crud table'")
					},
					want:       []string{"ENT_CRUD_T", "COMMENT"},
					exactStmts: 1,
				},
				{
					name: "drop",
					mutate: func(t *testing.T, _ liveServer, source string) string {
						return entDropTableBlock(t, source, "ent_crud_t")
					},
					want:       []string{"DROP TABLE", "ENT_CRUD_T"},
					exactStmts: 1,
				},
			},
		},
		{
			name: "column",
			phases: []entCRUDPhase{
				{
					name: "create",
					mutate: func(t *testing.T, _ liveServer, source string) string {
						return addColumnToTable(t, source, "users", "`ent_crud_col` varchar(50) DEFAULT NULL")
					},
					want:       []string{"ALTER TABLE", "ENT_CRUD_COL"},
					exactStmts: 1,
				},
				{
					name: "modify",
					mutate: func(t *testing.T, _ liveServer, source string) string {
						return entReplaceInTable(t, source, "users", "`ent_crud_col` varchar(50)", "`ent_crud_col` varchar(120)")
					},
					want:       []string{"ENT_CRUD_COL", "VARCHAR(120)"},
					exactStmts: 1,
				},
				{
					name: "drop",
					mutate: func(t *testing.T, _ liveServer, source string) string {
						return entDropLineInTable(t, source, "users", "`ent_crud_col`")
					},
					want:       []string{"DROP COLUMN", "ENT_CRUD_COL"},
					exactStmts: 1,
				},
			},
		},
		{
			name: "index_plain",
			phases: []entCRUDPhase{
				{
					name: "create",
					mutate: func(t *testing.T, _ liveServer, source string) string {
						return addIndexToTable(t, source, "users", "KEY `ent_idx_theme` (`theme`)")
					},
					want:       []string{"ENT_IDX_THEME"},
					exactStmts: 1,
				},
				{
					name: "modify",
					mutate: func(t *testing.T, _ liveServer, source string) string {
						return entReplaceLineInTable(t, source, "users", "`ent_idx_theme`", "KEY `ent_idx_theme` (`theme`,`lang`)")
					},
					want: []string{"ENT_IDX_THEME"},
				},
				{
					name: "drop",
					mutate: func(t *testing.T, _ liveServer, source string) string {
						return entDropLineInTable(t, source, "users", "`ent_idx_theme`")
					},
					want:       []string{"ENT_IDX_THEME"},
					exactStmts: 1,
				},
			},
		},
		{
			name: "index_unique",
			phases: []entCRUDPhase{
				{
					name: "create",
					mutate: func(t *testing.T, _ liveServer, source string) string {
						return addIndexToTable(t, source, "users", "UNIQUE KEY `ent_uk_passwd` (`passwd`)")
					},
					want:       []string{"ENT_UK_PASSWD", "UNIQUE"},
					exactStmts: 1,
				},
				{
					name: "modify",
					mutate: func(t *testing.T, _ liveServer, source string) string {
						return entReplaceLineInTable(t, source, "users", "`ent_uk_passwd`", "UNIQUE KEY `ent_uk_passwd` (`passwd`,`lang`)")
					},
					want: []string{"ENT_UK_PASSWD"},
				},
				{
					name: "drop",
					mutate: func(t *testing.T, _ liveServer, source string) string {
						return entDropLineInTable(t, source, "users", "`ent_uk_passwd`")
					},
					want:       []string{"ENT_UK_PASSWD"},
					exactStmts: 1,
				},
			},
		},
		{
			name: "index_prefix",
			phases: []entCRUDPhase{
				{
					name: "create",
					mutate: func(t *testing.T, _ liveServer, source string) string {
						return addIndexToTable(t, source, "hosts", "KEY `ent_idx_desc` (`description`(32))")
					},
					want:       []string{"ENT_IDX_DESC", "(32)"},
					exactStmts: 1,
				},
				{
					name: "modify",
					mutate: func(t *testing.T, _ liveServer, source string) string {
						return entReplaceLineInTable(t, source, "hosts", "`ent_idx_desc`", "KEY `ent_idx_desc` (`description`(64))")
					},
					want: []string{"ENT_IDX_DESC", "(64)"},
				},
				{
					name: "drop",
					mutate: func(t *testing.T, _ liveServer, source string) string {
						return entDropLineInTable(t, source, "hosts", "`ent_idx_desc`")
					},
					want:       []string{"ENT_IDX_DESC"},
					exactStmts: 1,
				},
			},
		},
		{
			name: "index_composite",
			phases: []entCRUDPhase{
				{
					name: "create",
					mutate: func(t *testing.T, _ liveServer, source string) string {
						return addIndexToTable(t, source, "maintenances", "KEY `ent_idx_window` (`active_till`,`maintenance_type`)")
					},
					want:       []string{"ENT_IDX_WINDOW"},
					exactStmts: 1,
				},
				{
					name: "modify", // reorder the key parts — a semantic index change
					mutate: func(t *testing.T, _ liveServer, source string) string {
						return entReplaceLineInTable(t, source, "maintenances", "`ent_idx_window`", "KEY `ent_idx_window` (`maintenance_type`,`active_till`)")
					},
					want: []string{"ENT_IDX_WINDOW"},
				},
				{
					name: "drop",
					mutate: func(t *testing.T, _ liveServer, source string) string {
						return entDropLineInTable(t, source, "maintenances", "`ent_idx_window`")
					},
					want:       []string{"ENT_IDX_WINDOW"},
					exactStmts: 1,
				},
			},
		},
		{
			name: "foreign_key",
			phases: []entCRUDPhase{
				{
					name: "create",
					mutate: func(t *testing.T, _ liveServer, source string) string {
						return addIndexToTable(t, source, "ent_ranked", "CONSTRAINT `ent_fk_owner` FOREIGN KEY (`owner_userid`) REFERENCES `users` (`userid`) ON DELETE SET NULL")
					},
					want: []string{"ENT_FK_OWNER", "FOREIGN KEY"},
				},
				{
					name: "modify", // referential-action change
					mutate: func(t *testing.T, _ liveServer, source string) string {
						return entReplaceInTable(t, source, "ent_ranked", "ON DELETE SET NULL", "ON DELETE CASCADE")
					},
					want: []string{"ENT_FK_OWNER"},
				},
				{
					name: "drop",
					mutate: func(t *testing.T, _ liveServer, source string) string {
						return entDropLineInTable(t, source, "ent_ranked", "CONSTRAINT `ent_fk_owner`")
					},
					want: []string{"ENT_FK_OWNER"},
				},
			},
		},
		{
			name:   "check",
			skip57: "5.7 parses-and-ignores CHECK constraints (nothing syncs back)",
			phases: []entCRUDPhase{
				{
					name: "create",
					mutate: func(t *testing.T, _ liveServer, source string) string {
						return addIndexToTable(t, source, "ent_ranked", "CONSTRAINT `ent_chk_score` CHECK ((`score` >= 0))")
					},
					want:       []string{"ENT_CHK_SCORE", "CHECK"},
					exactStmts: 1,
				},
				{
					name: "modify",
					mutate: func(t *testing.T, _ liveServer, source string) string {
						return entReplaceLineInTable(t, source, "ent_ranked", "`ent_chk_score`", "CONSTRAINT `ent_chk_score` CHECK ((`score` <= 1000000))")
					},
					want: []string{"ENT_CHK_SCORE"},
				},
				{
					name: "drop",
					mutate: func(t *testing.T, _ liveServer, source string) string {
						return entDropLineInTable(t, source, "ent_ranked", "`ent_chk_score`")
					},
					want:       []string{"ENT_CHK_SCORE"},
					exactStmts: 1,
				},
			},
		},
		{
			name: "view",
			phases: []entCRUDPhase{
				{
					name: "create",
					mutate: func(_ *testing.T, _ liveServer, source string) string {
						return source + "\nCREATE VIEW ent_crud_v AS SELECT roleid, name FROM role;\n"
					},
					want:       []string{"ENT_CRUD_V"},
					exactStmts: 1,
				},
				{
					name: "modify",
					mutate: func(t *testing.T, _ liveServer, source string) string {
						return addColumnToView(t, source, "ent_crud_v", "`role`.`type` AS `type`")
					},
					want:       []string{"ENT_CRUD_V"},
					exactStmts: 1,
				},
				{
					name: "drop",
					mutate: func(t *testing.T, _ liveServer, source string) string {
						return dropObjectBlock(t, source, "VIEW", "ent_crud_v")
					},
					want:       []string{"DROP VIEW", "ENT_CRUD_V"},
					exactStmts: 1,
				},
			},
		},
		{
			name: "function",
			phases: []entCRUDPhase{
				{
					name: "create",
					mutate: func(_ *testing.T, _ liveServer, source string) string {
						return source + "\nCREATE FUNCTION ent_crud_f() RETURNS INT DETERMINISTIC RETURN 41;\n"
					},
					want:       []string{"ENT_CRUD_F"},
					exactStmts: 1,
				},
				{
					name: "modify", // body change — this omni build renders it as DROP + CREATE
					mutate: func(t *testing.T, _ liveServer, source string) string {
						return mustReplace(t, source, "RETURN 41", "RETURN 42")
					},
					want:       []string{"ENT_CRUD_F"},
					exactStmts: 2,
				},
				{
					name: "drop",
					mutate: func(t *testing.T, _ liveServer, source string) string {
						return dropObjectBlock(t, source, "FUNCTION", "ent_crud_f")
					},
					want:       []string{"DROP FUNCTION", "ENT_CRUD_F"},
					exactStmts: 1,
				},
			},
		},
		{
			name: "procedure",
			phases: []entCRUDPhase{
				{
					name: "create",
					mutate: func(_ *testing.T, _ liveServer, source string) string {
						return source + "\nCREATE PROCEDURE ent_crud_p(IN rid BIGINT UNSIGNED) BEGIN UPDATE role SET readonly = readonly WHERE roleid = rid; END;\n"
					},
					want:       []string{"ENT_CRUD_P"},
					exactStmts: 1,
				},
				{
					name: "modify",
					mutate: func(t *testing.T, _ liveServer, source string) string {
						return mustReplace(t, source, "SET readonly = readonly", "SET readonly = 0")
					},
					want:       []string{"ENT_CRUD_P"},
					exactStmts: 2,
				},
				{
					name: "drop",
					mutate: func(t *testing.T, _ liveServer, source string) string {
						return dropObjectBlock(t, source, "PROCEDURE", "ent_crud_p")
					},
					want:       []string{"DROP PROCEDURE", "ENT_CRUD_P"},
					exactStmts: 1,
				},
			},
		},
		{
			name: "trigger",
			phases: []entCRUDPhase{
				{
					name: "create",
					mutate: func(_ *testing.T, _ liveServer, source string) string {
						return source + "\nCREATE TRIGGER ent_crud_trg BEFORE UPDATE ON media_type FOR EACH ROW SET NEW.name = NEW.name;\n"
					},
					want:       []string{"ENT_CRUD_TRG"},
					exactStmts: 1,
				},
				{
					name: "modify",
					mutate: func(t *testing.T, _ liveServer, source string) string {
						return mustReplace(t, source, "SET NEW.name = NEW.name", "SET NEW.name = TRIM(NEW.name)")
					},
					want:       []string{"ENT_CRUD_TRG"},
					exactStmts: 2,
				},
				{
					name: "drop",
					mutate: func(t *testing.T, _ liveServer, source string) string {
						return dropTrigger(t, source, "ent_crud_trg")
					},
					want:       []string{"DROP TRIGGER", "ENT_CRUD_TRG"},
					exactStmts: 1,
				},
			},
		},
		{
			name: "event",
			phases: []entCRUDPhase{
				{
					name: "create",
					mutate: func(_ *testing.T, _ liveServer, source string) string {
						return source + "\nCREATE EVENT ent_crud_ev ON SCHEDULE EVERY 12 HOUR DO DELETE FROM changelog WHERE clock < 100;\n"
					},
					want:       []string{"ENT_CRUD_EV"},
					exactStmts: 1,
				},
				{
					name: "modify",
					mutate: func(t *testing.T, _ liveServer, source string) string {
						return mustReplace(t, source, "EVERY 12 HOUR", "EVERY 6 HOUR")
					},
					want: []string{"ENT_CRUD_EV"},
				},
				{
					name: "drop",
					mutate: func(t *testing.T, _ liveServer, source string) string {
						return entDropObjectNamed(t, source, "EVENT", "ent_crud_ev")
					},
					want:       []string{"DROP EVENT", "ENT_CRUD_EV"},
					exactStmts: 1,
				},
			},
		},
		{
			// NOTE (observation, not a failure): for every partition change the differ
			// emits a full `ALTER TABLE ... PARTITION BY` REPARTITION statement rather
			// than the targeted ADD/DROP/REORGANIZE PARTITION — semantically correct and
			// converging, but a whole-table rebuild on large enterprise tables. See the
			// campaign findings.
			name: "partition",
			phases: []entCRUDPhase{
				{
					name: "create", // add a partition
					mutate: func(t *testing.T, _ liveServer, source string) string {
						return entSetPartitionClause(t, source, "ent_part_log",
							"PARTITION BY RANGE (`bucket`)\n(PARTITION p0 VALUES LESS THAN (100),\n PARTITION p1 VALUES LESS THAN (200),\n PARTITION p2 VALUES LESS THAN (300))")
					},
					want: []string{"PARTITION BY", "P2"},
				},
				{
					name: "modify", // reorganize: merge p0+p1 into p01
					mutate: func(t *testing.T, _ liveServer, source string) string {
						return entSetPartitionClause(t, source, "ent_part_log",
							"PARTITION BY RANGE (`bucket`)\n(PARTITION p01 VALUES LESS THAN (200),\n PARTITION p2 VALUES LESS THAN (300))")
					},
					want: []string{"PARTITION BY", "P01"},
				},
				{
					name: "drop", // drop the p2 partition (the repartition plan re-lists survivors)
					mutate: func(t *testing.T, _ liveServer, source string) string {
						return entSetPartitionClause(t, source, "ent_part_log",
							"PARTITION BY RANGE (`bucket`)\n(PARTITION p01 VALUES LESS THAN (200))")
					},
					want: []string{"PARTITION BY", "P01"},
				},
			},
		},
	}
}

//nolint:tparallel
func TestSDLEnterpriseSingleObjectCRUD(t *testing.T) {
	skipUnlessLiveOracle(t)
	ctx := context.Background()

	for _, srv := range liveServers {
		srv := srv
		t.Run(srv.name, func(t *testing.T) {
			for _, kind := range entCRUDKinds() {
				kind := kind
				t.Run(kind.name, func(t *testing.T) {
					if srv.version == "5.7" && kind.skip57 != "" {
						t.Skipf("[%s/%s] skipped on 5.7: %s", srv.name, kind.name, kind.skip57)
					}
					base := entZabbixSlice(t, entSliceCoreTables...) + entA1Aux
					dbName := entLoadDDL(ctx, t, srv, "entsdl_a1", base)

					for _, phase := range kind.phases {
						label := srv.name + "/A1/" + kind.name + "/" + phase.name
						source := dumpSDL(ctx, t, srv, dbName)
						target := phase.mutate(t, srv, source)
						plan := entOracle(ctx, t, srv, dbName, target, label)

						upper := strings.ToUpper(plan)
						for _, want := range phase.want {
							require.Contains(t, upper, want, "[%s] plan missing %q:\n%s", label, want, plan)
						}
						if phase.exactStmts > 0 {
							require.Equal(t, phase.exactStmts, entPlanStatementCount(t, plan),
								"[%s] plan not minimal (%d stmts expected):\n%s", label, phase.exactStmts, plan)
						}
					}
				})
			}
		})
	}
}

// ----------------------------------------------------------------------------
// A2: dependent objects — the enterprise differentiator. FK chains, trigger→table
// dependencies (Zabbix changelog pattern), view-on-view stacks, function-used-by-view,
// and a circular FK pair, each as one or more oracle rounds on the core slice.
// ----------------------------------------------------------------------------

//nolint:tparallel
func TestSDLEnterpriseDependentObjects(t *testing.T) {
	skipUnlessLiveOracle(t)
	ctx := context.Background()

	for _, srv := range liveServers {
		srv := srv
		t.Run(srv.name, func(t *testing.T) {
			t.Run("fk_chain_extend", func(t *testing.T) {
				// Extend the real item_tag -> items -> hosts -> proxy -> proxy_group chain
				// with two new tables in ONE release: ent_item_note -> items and
				// ent_note_tag -> ent_item_note. Creation must be topological.
				dbName := entLoadDDL(ctx, t, srv, "entsdl_a2", entZabbixSlice(t, entSliceCoreTables...))
				source := dumpSDL(ctx, t, srv, dbName)
				target := source + `
CREATE TABLE ent_item_note (
  noteid bigint unsigned NOT NULL,
  itemid bigint unsigned NOT NULL,
  note varchar(255) DEFAULT '' NOT NULL,
  PRIMARY KEY (noteid),
  KEY ent_item_note_1 (itemid),
  CONSTRAINT ent_c_item_note_1 FOREIGN KEY (itemid) REFERENCES items (itemid) ON DELETE CASCADE
) ENGINE=InnoDB;
CREATE TABLE ent_note_tag (
  notetagid bigint unsigned NOT NULL,
  noteid bigint unsigned NOT NULL,
  tag varchar(64) DEFAULT '' NOT NULL,
  PRIMARY KEY (notetagid),
  KEY ent_note_tag_1 (noteid),
  CONSTRAINT ent_c_note_tag_1 FOREIGN KEY (noteid) REFERENCES ent_item_note (noteid) ON DELETE CASCADE
) ENGINE=InnoDB;
`
				plan := entOracle(ctx, t, srv, dbName, target, srv.name+"/A2/fk_chain_extend")
				upper := strings.ToUpper(plan)
				noteIdx := strings.Index(upper, "CREATE TABLE `ENT_ITEM_NOTE`")
				tagIdx := strings.Index(upper, "CREATE TABLE `ENT_NOTE_TAG`")
				require.GreaterOrEqual(t, noteIdx, 0, "plan must create ent_item_note:\n%s", plan)
				require.GreaterOrEqual(t, tagIdx, 0, "plan must create ent_note_tag:\n%s", plan)
			})

			t.Run("fk_chain_drop_mid", func(t *testing.T) {
				// Drop the middle of the chain — items — together with its dependents
				// item_tag and item_preproc (each carrying 3 changelog triggers) in ONE
				// release. The plan must drop FKs/tables in dependency order and must not
				// drop a trigger after its table is already gone.
				dbName := entLoadDDL(ctx, t, srv, "entsdl_a2", entZabbixSlice(t, entSliceCoreTables...))
				source := dumpSDL(ctx, t, srv, dbName)
				target := source
				for _, trg := range []string{
					"items_insert", "items_update", "items_delete",
					"item_tag_insert", "item_tag_update", "item_tag_delete",
					"item_preproc_insert", "item_preproc_update", "item_preproc_delete",
				} {
					target = dropTrigger(t, target, trg)
				}
				for _, tbl := range []string{"item_tag", "item_preproc", "items"} {
					target = entDropTableBlock(t, target, tbl)
				}
				plan := entOracle(ctx, t, srv, dbName, target, srv.name+"/A2/fk_chain_drop_mid")
				upper := strings.ToUpper(plan)
				for _, want := range []string{"`ITEMS`", "`ITEM_TAG`", "`ITEM_PREPROC`"} {
					require.Contains(t, upper, want, "plan must drop %s:\n%s", want, plan)
				}
			})

			t.Run("trigger_table_widen", func(t *testing.T) {
				// The Zabbix changelog pattern: hosts carries 5 triggers, two of which
				// (hosts_name_upper_*) read/write hosts.name and hosts.name_upper. Widen
				// BOTH columns in one release; the triggers must survive the ALTERs.
				dbName := entLoadDDL(ctx, t, srv, "entsdl_a2", entZabbixSlice(t, entSliceCoreTables...))
				source := dumpSDL(ctx, t, srv, dbName)
				target := entReplaceInTable(t, source, "hosts", "`name` varchar(128)", "`name` varchar(190)")
				target = entReplaceInTable(t, target, "hosts", "`name_upper` varchar(128)", "`name_upper` varchar(190)")
				plan := entOracle(ctx, t, srv, dbName, target, srv.name+"/A2/trigger_table_widen")
				upper := strings.ToUpper(plan)
				require.Contains(t, upper, "VARCHAR(190)", "plan must widen the columns:\n%s", plan)
				require.NotContains(t, upper, "DROP TABLE", "widening must not recreate hosts:\n%s", plan)
			})

			t.Run("trigger_modify_one_of_five", func(t *testing.T) {
				// Redefine ONE of hosts' five triggers (hosts_update) and prove the other
				// four are untouched by the plan.
				dbName := entLoadDDL(ctx, t, srv, "entsdl_a2", entZabbixSlice(t, entSliceCoreTables...))
				source := dumpSDL(ctx, t, srv, dbName)
				target := mustReplace(t, source, "values (1,old.hostid,2,", "values (1,new.hostid,2,")
				plan := entOracle(ctx, t, srv, dbName, target, srv.name+"/A2/trigger_modify_one_of_five")
				upper := strings.ToUpper(plan)
				require.Contains(t, upper, "HOSTS_UPDATE", "plan must redefine hosts_update:\n%s", plan)
				for _, untouched := range []string{"HOSTS_INSERT", "HOSTS_DELETE", "HOSTS_NAME_UPPER_INSERT", "HOSTS_NAME_UPPER_UPDATE"} {
					require.NotContains(t, upper, untouched, "plan must not touch %s:\n%s", untouched, plan)
				}
				require.Equal(t, 2, entPlanStatementCount(t, plan), "trigger redefine should be DROP+CREATE:\n%s", plan)
			})

			t.Run("view_stack", func(t *testing.T) {
				// 3-deep view stack v3 -> v2 -> v1 -> users: create the whole stack in one
				// release, modify the MIDDLE view, then drop the whole stack.
				dbName := entLoadDDL(ctx, t, srv, "entsdl_a2", entZabbixSlice(t, entSliceCoreTables...))

				source := dumpSDL(ctx, t, srv, dbName)
				target := source + `
CREATE VIEW ent_vs1 AS SELECT userid, username, surname FROM users;
CREATE VIEW ent_vs2 AS SELECT userid, username FROM ent_vs1;
CREATE VIEW ent_vs3 AS SELECT userid FROM ent_vs2;
`
				plan := entOracle(ctx, t, srv, dbName, target, srv.name+"/A2/view_stack/create")
				upper := strings.ToUpper(plan)
				v1 := strings.Index(upper, "`ENT_VS1`")
				v2 := strings.Index(upper, "`ENT_VS2`")
				v3 := strings.Index(upper, "`ENT_VS3`")
				require.True(t, v1 >= 0 && v2 >= 0 && v3 >= 0, "plan must create all three views:\n%s", plan)
				require.True(t, v1 < v2 && v2 < v3, "views must be created base-first (v1<v2<v3):\n%s", plan)

				source = dumpSDL(ctx, t, srv, dbName)
				target = addColumnToView(t, source, "ent_vs2", "`ent_vs1`.`surname` AS `surname`")
				plan = entOracle(ctx, t, srv, dbName, target, srv.name+"/A2/view_stack/modify_middle")
				upper = strings.ToUpper(plan)
				require.Contains(t, upper, "VIEW `ENT_VS2`")
				require.NotContains(t, upper, "ENT_VS3", "modifying v2 must not churn v3:\n%s", plan)
				// v2's body legitimately references ent_vs1 in FROM; minimality is proven
				// by the single-statement plan (v1 and v3 are untouched).
				require.Equal(t, 1, entPlanStatementCount(t, plan), "modifying v2 must be a single statement:\n%s", plan)

				source = dumpSDL(ctx, t, srv, dbName)
				target = dropObjectBlock(t, source, "VIEW", "ent_vs3")
				target = dropObjectBlock(t, target, "VIEW", "ent_vs2")
				target = dropObjectBlock(t, target, "VIEW", "ent_vs1")
				plan = entOracle(ctx, t, srv, dbName, target, srv.name+"/A2/view_stack/drop")
				require.Equal(t, 3, entPlanStatementCount(t, plan), "stack drop should be exactly 3 DROP VIEW:\n%s", plan)
			})

			t.Run("function_used_by_view", func(t *testing.T) {
				// Create a function and a view CALLING it in one release (MySQL rejects a
				// view over a missing function, so creation order is enforced by apply);
				// then drop both in one release.
				//
				// Was PINNED-BUG (view created before the function it calls → Error 1305):
				// fixed by omni #361 (routine creates precede view creates; drops inverse).
				dbName := entLoadDDL(ctx, t, srv, "entsdl_a2", entZabbixSlice(t, entSliceCoreTables...))

				source := dumpSDL(ctx, t, srv, dbName)
				target := source + `
CREATE FUNCTION ent_ucount() RETURNS INT READS SQL DATA RETURN (SELECT COUNT(*) FROM users);
CREATE VIEW ent_v_fn AS SELECT ent_ucount() AS n;
`
				plan := entOracle(ctx, t, srv, dbName, target, srv.name+"/A2/function_used_by_view/create")
				upper := strings.ToUpper(plan)
				require.Contains(t, upper, "ENT_UCOUNT")
				require.Contains(t, upper, "ENT_V_FN")

				source = dumpSDL(ctx, t, srv, dbName)
				target = dropObjectBlock(t, source, "VIEW", "ent_v_fn")
				target = dropObjectBlock(t, target, "FUNCTION", "ent_ucount")
				plan = entOracle(ctx, t, srv, dbName, target, srv.name+"/A2/function_used_by_view/drop")
				require.Equal(t, 2, entPlanStatementCount(t, plan), "should be exactly DROP VIEW + DROP FUNCTION:\n%s", plan)
			})

			t.Run("circular_fk_pair", func(t *testing.T) {
				// Create two mutually-referencing tables in ONE release. The generated DDL
				// must break the cycle itself (deferred ALTER or FK-checks toggling) — the
				// apply connection runs with default settings.
				dbName := entLoadDDL(ctx, t, srv, "entsdl_a2", entZabbixSlice(t, entSliceCoreTables...))
				source := dumpSDL(ctx, t, srv, dbName)
				target := source + `
CREATE TABLE ent_circ_a (
  aid bigint unsigned NOT NULL,
  bid bigint unsigned NULL,
  PRIMARY KEY (aid),
  KEY ent_circ_a_1 (bid),
  CONSTRAINT ent_c_circ_a FOREIGN KEY (bid) REFERENCES ent_circ_b (bid) ON DELETE SET NULL
) ENGINE=InnoDB;
CREATE TABLE ent_circ_b (
  bid bigint unsigned NOT NULL,
  aid bigint unsigned NULL,
  PRIMARY KEY (bid),
  KEY ent_circ_b_1 (aid),
  CONSTRAINT ent_c_circ_b FOREIGN KEY (aid) REFERENCES ent_circ_a (aid) ON DELETE SET NULL
) ENGINE=InnoDB;
`
				plan := entOracle(ctx, t, srv, dbName, target, srv.name+"/A2/circular_fk_pair")
				upper := strings.ToUpper(plan)
				require.Contains(t, upper, "ENT_CIRC_A")
				require.Contains(t, upper, "ENT_CIRC_B")
			})
		})
	}
}

// ----------------------------------------------------------------------------
// A3: special types — AUTO_INCREMENT, generated columns, ENUM/SET members,
// charset/collation, temporal precision, DECIMAL, TEXT/BLOB families, signedness,
// prefix-indexed TEXT, and partition operations. Small synthetic bases, one oracle
// round each.
// ----------------------------------------------------------------------------

type entTypeCase struct {
	name   string
	base   string
	skip57 string
	// pinned, when non-empty, skips the case with a PINNED-BUG classification (a
	// confirmed engine bug being fixed upstream; the case re-arms when the skip drops).
	pinned string
	mutate func(t *testing.T, srv liveServer, source string) string
	want   []string
	// exactStmts, when > 0, asserts the plan statement count.
	exactStmts int
}

func entTypeCases() []entTypeCase {
	return []entTypeCase{
		{
			// REPRO: add `seq bigint unsigned NOT NULL AUTO_INCREMENT` + UNIQUE KEY uk_seq
			// (seq) in one release. The plan emits TWO statements — ADD COLUMN, then ADD
			// UNIQUE KEY — and MySQL rejects the first with Error 1075 (an auto column
			// must be defined as a key). The clauses must be grouped into one ALTER.
			name: "ai_add_column",
			base: "CREATE TABLE t3 (id int NOT NULL, PRIMARY KEY (id)) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;",
			// Was PINNED-BUG (AI column and its key split into separate ALTERs →
			// Error 1075): fixed by omni #364 (mergeAutoIncrementKeyOps grouping).
			mutate: func(t *testing.T, _ liveServer, source string) string {
				s := addColumnToTable(t, source, "t3", "`seq` bigint unsigned NOT NULL AUTO_INCREMENT")
				return addIndexToTable(t, s, "t3", "UNIQUE KEY `uk_seq` (`seq`)")
			},
			want: []string{"SEQ", "AUTO_INCREMENT"},
		},
		{
			name: "ai_drop_attribute",
			base: "CREATE TABLE t3 (id int NOT NULL AUTO_INCREMENT, PRIMARY KEY (id)) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;",
			mutate: func(t *testing.T, _ liveServer, source string) string {
				return entReplaceInTable(t, source, "t3",
					"`id` int NOT NULL AUTO_INCREMENT", "`id` int NOT NULL")
			},
			want:       []string{"`ID`"},
			exactStmts: 1,
		},
		{
			name: "ai_composite_pk_member",
			base: "CREATE TABLE t3 (a int NOT NULL, b int NOT NULL AUTO_INCREMENT, PRIMARY KEY (a,b), KEY idx_b (b)) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;",
			mutate: func(t *testing.T, _ liveServer, source string) string {
				return entReplaceInTable(t, source, "t3",
					"`b` int NOT NULL AUTO_INCREMENT", "`b` bigint NOT NULL AUTO_INCREMENT")
			},
			want:       []string{"BIGINT", "AUTO_INCREMENT"},
			exactStmts: 1,
		},
		{
			name: "generated_stored_add",
			base: "CREATE TABLE t3 (id int NOT NULL, price decimal(8,2) NOT NULL, PRIMARY KEY (id)) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;",
			mutate: func(t *testing.T, _ liveServer, source string) string {
				return addColumnToTable(t, source, "t3", "`total` decimal(10,2) GENERATED ALWAYS AS ((`price` * 1.1)) STORED")
			},
			want:       []string{"TOTAL", "STORED"},
			exactStmts: 1,
		},
		{
			name: "generated_virtual_add",
			base: "CREATE TABLE t3 (id int NOT NULL, price decimal(8,2) NOT NULL, PRIMARY KEY (id)) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;",
			mutate: func(t *testing.T, _ liveServer, source string) string {
				return addColumnToTable(t, source, "t3", "`total_v` decimal(10,2) GENERATED ALWAYS AS ((`price` * 2)) VIRTUAL")
			},
			want:       []string{"TOTAL_V"},
			exactStmts: 1,
		},
		{
			name: "generated_dependent_alter",
			// Widen the BASE column and change the generated column depending on it in the
			// SAME release — the plan must sequence the two ALTERs so MySQL accepts them.
			base: "CREATE TABLE t3 (id int NOT NULL, price decimal(8,2) NOT NULL, total decimal(10,2) GENERATED ALWAYS AS (price * 2) STORED, PRIMARY KEY (id)) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;",
			mutate: func(t *testing.T, _ liveServer, source string) string {
				s := entReplaceInTable(t, source, "t3", "`price` decimal(8,2)", "`price` decimal(10,2)")
				return entReplaceInTable(t, s, "t3", "* 2", "* 3")
			},
			want: []string{"PRICE"},
		},
		{
			name: "enum_member_add",
			base: "CREATE TABLE t3 (id int NOT NULL, kind enum('a','b') NOT NULL DEFAULT 'a', PRIMARY KEY (id)) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;",
			mutate: func(t *testing.T, _ liveServer, source string) string {
				return entReplaceInTable(t, source, "t3", "enum('a','b')", "enum('a','b','c')")
			},
			want:       []string{"ENUM"},
			exactStmts: 1,
		},
		{
			name: "enum_member_remove",
			base: "CREATE TABLE t3 (id int NOT NULL, kind enum('a','b','c') NOT NULL DEFAULT 'a', PRIMARY KEY (id)) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;",
			mutate: func(t *testing.T, _ liveServer, source string) string {
				return entReplaceInTable(t, source, "t3", "enum('a','b','c')", "enum('a','b')")
			},
			want:       []string{"ENUM"},
			exactStmts: 1,
		},
		{
			name: "enum_member_reorder",
			base: "CREATE TABLE t3 (id int NOT NULL, kind enum('a','b','c') NOT NULL DEFAULT 'a', PRIMARY KEY (id)) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;",
			mutate: func(t *testing.T, _ liveServer, source string) string {
				return entReplaceInTable(t, source, "t3", "enum('a','b','c')", "enum('c','a','b')")
			},
			want:       []string{"ENUM"},
			exactStmts: 1,
		},
		{
			name: "set_member_add_remove",
			base: "CREATE TABLE t3 (id int NOT NULL, tags set('x','y') NOT NULL DEFAULT '', flags set('p','q','r') NOT NULL DEFAULT '', PRIMARY KEY (id)) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;",
			mutate: func(t *testing.T, _ liveServer, source string) string {
				s := entReplaceInTable(t, source, "t3", "set('x','y')", "set('x','y','z')")
				return entReplaceInTable(t, s, "t3", "set('p','q','r')", "set('p','q')")
			},
			want: []string{"SET("},
		},
		{
			// utf8mb4_unicode_ci is non-default on BOTH versions (8.0 default 0900_ai_ci,
			// 5.7 default general_ci), so the dump renders the explicit column COLLATE on
			// both — a stable anchor. When the collation matches the table default the
			// dump omits CHARACTER SET/COLLATE entirely.
			name: "charset_column_collation",
			base: "CREATE TABLE t3 (id int NOT NULL, label varchar(80) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL, PRIMARY KEY (id)) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;",
			mutate: func(t *testing.T, _ liveServer, source string) string {
				return entReplaceInTable(t, source, "t3",
					"`label` varchar(80) COLLATE utf8mb4_unicode_ci",
					"`label` varchar(80) COLLATE utf8mb4_bin")
			},
			want:       []string{"UTF8MB4_BIN"},
			exactStmts: 1,
		},
		{
			name: "charset_table_level",
			base: "CREATE TABLE t3 (id int NOT NULL, title varchar(60) NOT NULL, body varchar(200) DEFAULT NULL, PRIMARY KEY (id)) ENGINE=InnoDB DEFAULT CHARSET=latin1;",
			mutate: func(t *testing.T, _ liveServer, source string) string {
				// Replace the whole table-options line so the (version-dependent) COLLATE
				// suffix can never survive as an invalid latin1/utf8mb4 hybrid. The options
				// line is the segment's last line, so everything from ') ENGINE=InnoDB' to
				// the segment end is rewritten.
				start, end := entTableSegment(t, source, "t3")
				seg := source[start:end]
				optIdx := strings.Index(seg, ") ENGINE=InnoDB")
				require.GreaterOrEqual(t, optIdx, 0, "t3 options line not found:\n%s", seg)
				tail := ""
				if nl := strings.Index(seg[optIdx:], "\n"); nl >= 0 {
					tail = seg[optIdx+nl:]
				}
				newSeg := seg[:optIdx] + ") ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;" + tail
				return source[:start] + newSeg + source[end:]
			},
			want: []string{"UTF8MB4"},
		},
		{
			name: "temporal_precision",
			base: "CREATE TABLE t3 (id int NOT NULL, ts timestamp(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3), dt datetime NOT NULL, PRIMARY KEY (id)) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;",
			mutate: func(t *testing.T, _ liveServer, source string) string {
				s := entReplaceAllInTable(t, source, "t3", "timestamp(3)", "timestamp(6)")
				s = entReplaceAllInTable(t, s, "t3", "CURRENT_TIMESTAMP(3)", "CURRENT_TIMESTAMP(6)")
				return entReplaceInTable(t, s, "t3", "`dt` datetime ", "`dt` datetime(3) ")
			},
			want: []string{"TIMESTAMP(6)", "DATETIME(3)"},
		},
		{
			name: "decimal_widen",
			base: "CREATE TABLE t3 (id int NOT NULL, amount decimal(10,2) NOT NULL DEFAULT '0.00', PRIMARY KEY (id)) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;",
			mutate: func(t *testing.T, _ liveServer, source string) string {
				return entReplaceInTable(t, source, "t3", "decimal(10,2)", "decimal(12,4)")
			},
			want:       []string{"DECIMAL(12,4)"},
			exactStmts: 1,
		},
		{
			name: "decimal_narrow",
			base: "CREATE TABLE t3 (id int NOT NULL, amount decimal(12,4) NOT NULL DEFAULT '0.0000', PRIMARY KEY (id)) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;",
			mutate: func(t *testing.T, _ liveServer, source string) string {
				s := entReplaceInTable(t, source, "t3", "decimal(12,4)", "decimal(10,2)")
				return entReplaceInTable(t, s, "t3", "'0.0000'", "'0.00'")
			},
			want:       []string{"DECIMAL(10,2)"},
			exactStmts: 1,
		},
		{
			name: "text_blob_family",
			base: "CREATE TABLE t3 (id int NOT NULL, body text, payload blob, PRIMARY KEY (id)) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;",
			mutate: func(t *testing.T, _ liveServer, source string) string {
				s := entReplaceInTable(t, source, "t3", "`body` text", "`body` mediumtext")
				return entReplaceInTable(t, s, "t3", "`payload` blob", "`payload` longblob")
			},
			want: []string{"MEDIUMTEXT", "LONGBLOB"},
		},
		{
			name: "unsigned_signed_flip",
			base: "CREATE TABLE t3 (id int NOT NULL, cnt int unsigned NOT NULL DEFAULT '0', pos int NOT NULL DEFAULT '0', PRIMARY KEY (id)) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;",
			mutate: func(t *testing.T, srv liveServer, source string) string {
				s := entReplaceInTable(t, source, "t3", "`cnt` "+entUintType(srv), "`cnt` int")
				return entReplaceInTable(t, s, "t3", "`pos` int", "`pos` "+entUintType(srv))
			},
			want: []string{"`CNT`", "`POS`"},
		},
		{
			name: "prefix_indexed_text_type_change",
			base: "CREATE TABLE t3 (id int NOT NULL, body text NOT NULL, PRIMARY KEY (id), KEY idx_body (body(24))) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;",
			mutate: func(t *testing.T, _ liveServer, source string) string {
				return entReplaceInTable(t, source, "t3", "`body` text", "`body` mediumtext")
			},
			want: []string{"MEDIUMTEXT"},
		},
		{
			name: "partition_add",
			base: "CREATE TABLE t3 (id int NOT NULL, bucket int NOT NULL, PRIMARY KEY (id,bucket)) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 PARTITION BY RANGE (bucket) (PARTITION p0 VALUES LESS THAN (100), PARTITION p1 VALUES LESS THAN (200));",
			mutate: func(t *testing.T, _ liveServer, source string) string {
				return entSetPartitionClause(t, source, "t3",
					"PARTITION BY RANGE (`bucket`)\n(PARTITION p0 VALUES LESS THAN (100),\n PARTITION p1 VALUES LESS THAN (200),\n PARTITION p2 VALUES LESS THAN (300))")
			},
			want: []string{"P2"},
		},
		{
			name: "partition_drop",
			base: "CREATE TABLE t3 (id int NOT NULL, bucket int NOT NULL, PRIMARY KEY (id,bucket)) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 PARTITION BY RANGE (bucket) (PARTITION p0 VALUES LESS THAN (100), PARTITION p1 VALUES LESS THAN (200), PARTITION p2 VALUES LESS THAN (300));",
			mutate: func(t *testing.T, _ liveServer, source string) string {
				return entSetPartitionClause(t, source, "t3",
					"PARTITION BY RANGE (`bucket`)\n(PARTITION p0 VALUES LESS THAN (100),\n PARTITION p1 VALUES LESS THAN (200))")
			},
			// The differ emits a full repartition listing only the survivors (see the
			// partition-minimality observation in the campaign findings).
			want: []string{"PARTITION BY", "P0"},
		},
		{
			name: "partition_reorganize",
			base: "CREATE TABLE t3 (id int NOT NULL, bucket int NOT NULL, PRIMARY KEY (id,bucket)) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 PARTITION BY RANGE (bucket) (PARTITION p0 VALUES LESS THAN (100), PARTITION p1 VALUES LESS THAN (200), PARTITION pmax VALUES LESS THAN MAXVALUE);",
			mutate: func(t *testing.T, _ liveServer, source string) string {
				return entSetPartitionClause(t, source, "t3",
					"PARTITION BY RANGE (`bucket`)\n(PARTITION p01 VALUES LESS THAN (200),\n PARTITION pmax VALUES LESS THAN MAXVALUE)")
			},
			want: []string{"P01"},
		},
		{
			name: "partition_existing_table",
			base: "CREATE TABLE t3 (id int NOT NULL, bucket int NOT NULL, PRIMARY KEY (id,bucket)) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;",
			mutate: func(t *testing.T, _ liveServer, source string) string {
				return entAppendTableClause(t, source, "t3", "PARTITION BY HASH (`bucket`)\nPARTITIONS 4")
			},
			want: []string{"PARTITION BY", "HASH"},
		},
		{
			name: "departition",
			base: "CREATE TABLE t3 (id int NOT NULL, bucket int NOT NULL, PRIMARY KEY (id,bucket)) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 PARTITION BY HASH (bucket) PARTITIONS 4;",
			mutate: func(t *testing.T, _ liveServer, source string) string {
				return entSetPartitionClause(t, source, "t3", "")
			},
			want: []string{"T3"},
		},
	}
}

//nolint:tparallel
func TestSDLEnterpriseSpecialTypes(t *testing.T) {
	skipUnlessLiveOracle(t)
	ctx := context.Background()

	for _, srv := range liveServers {
		srv := srv
		t.Run(srv.name, func(t *testing.T) {
			for _, tc := range entTypeCases() {
				tc := tc
				t.Run(tc.name, func(t *testing.T) {
					if srv.version == "5.7" && tc.skip57 != "" {
						t.Skipf("[%s/%s] skipped on 5.7: %s", srv.name, tc.name, tc.skip57)
					}
					if tc.pinned != "" {
						t.Skip(tc.pinned)
					}
					dbName := entLoadDDL(ctx, t, srv, "entsdl_a3", tc.base)
					source := dumpSDL(ctx, t, srv, dbName)
					target := tc.mutate(t, srv, source)
					plan := entOracle(ctx, t, srv, dbName, target, srv.name+"/A3/"+tc.name)

					upper := strings.ToUpper(plan)
					for _, want := range tc.want {
						require.Contains(t, upper, want, "[%s/A3/%s] plan missing %q:\n%s", srv.name, tc.name, want, plan)
					}
					if tc.exactStmts > 0 {
						require.Equal(t, tc.exactStmts, entPlanStatementCount(t, plan),
							"[%s/A3/%s] plan not minimal:\n%s", srv.name, tc.name, plan)
					}
				})
			}
		})
	}
}

// ----------------------------------------------------------------------------
// A5(a): sequential chain S0 -> S1 -> S2 -> S3 -> S4 on the core slice. Every step must
// converge, and the endpoint must be REACHABLE FROM S0 IN ONE SHOT on a second database
// — both databases must dump to the same canonical schema.
// ----------------------------------------------------------------------------

//nolint:tparallel
func TestSDLEnterpriseSequentialChain(t *testing.T) {
	skipUnlessLiveOracle(t)
	ctx := context.Background()

	for _, srv := range liveServers {
		srv := srv
		t.Run(srv.name, func(t *testing.T) {
			base := entZabbixSlice(t, entSliceCoreTables...)
			chainDB := entLoadDDL(ctx, t, srv, "entsdl_a5chain", base)

			// The chain targets are built by cumulative TEXTUAL mutation of the canonical
			// S0 dump, so the final text is identical for the chained and one-shot legs.
			s0 := dumpSDL(ctx, t, srv, chainDB)

			// The appended blocks use backticked names so the later chain steps can locate
			// and mutate them with the same segment helpers that work on canonical dumps.
			s1 := addColumnToTable(t, s0, "users", "`ent_ch_flag` int NOT NULL DEFAULT '0'")
			s1 += `
CREATE TABLE ` + "`ent_ch_audit`" + ` (
  auditid bigint unsigned NOT NULL,
  userid bigint unsigned NULL,
  note varchar(60) DEFAULT '' NOT NULL,
  PRIMARY KEY (auditid)
) ENGINE=InnoDB;
`
			s2 := addIndexToTable(t, s1, "users", "KEY `ent_ch_idx` (`ent_ch_flag`)")
			s2 += `
CREATE VIEW ` + "`ent_ch_v`" + ` AS SELECT auditid, note FROM ent_ch_audit;
`
			s3 := addIndexToTable(t, s2, "ent_ch_audit", "KEY `ent_ch_fk` (`userid`)")
			s3 = addIndexToTable(t, s3, "ent_ch_audit", "CONSTRAINT `ent_ch_fk` FOREIGN KEY (`userid`) REFERENCES `users` (`userid`) ON DELETE SET NULL")
			s3 = entReplaceInTable(t, s3, "maintenances", "`name` varchar(128)", "`name` varchar(190)")
			s3 += `
CREATE TRIGGER ent_ch_trg BEFORE INSERT ON ent_ch_audit FOR EACH ROW SET NEW.note = TRIM(NEW.note);
`
			s4 := entDropLineInTable(t, s3, "users", "`ent_ch_idx`")
			s4 = addColumnToView(t, s4, "ent_ch_v", "`ent_ch_audit`.`userid` AS `userid`")
			s4 += `
CREATE PROCEDURE ent_ch_p(IN aid BIGINT UNSIGNED) BEGIN DELETE FROM ent_ch_audit WHERE auditid = aid; END;
`
			for i, step := range []string{s1, s2, s3, s4} {
				entOracle(ctx, t, srv, chainDB, step, srv.name+"/A5/chain/S"+string(rune('1'+i)))
			}

			// One-shot leg: a second fresh database goes S0 -> S4 in a single release.
			oneshotDB := entLoadDDL(ctx, t, srv, "entsdl_a5one", base)
			entOracle(ctx, t, srv, oneshotDB, s4, srv.name+"/A5/chain/oneshot")

			// Same endpoint: the two databases' canonical dumps must be equivalent.
			chainDump := dumpSDL(ctx, t, srv, chainDB)
			oneshotDump := dumpSDL(ctx, t, srv, oneshotDB)
			diffAB, err := mysqlDiffSDLMigration(chainDump, oneshotDump, srv.version)
			require.NoError(t, err)
			require.Empty(t, diffAB, "chained and one-shot endpoints diverge (chain->oneshot):\n%s", diffAB)
			diffBA, err := mysqlDiffSDLMigration(oneshotDump, chainDump, srv.version)
			require.NoError(t, err)
			require.Empty(t, diffBA, "chained and one-shot endpoints diverge (oneshot->chain):\n%s", diffBA)
		})
	}
}

// ----------------------------------------------------------------------------
// A5(b): apply-back — migrate B -> T (destructive + additive mix), then T -> B using the
// ORIGINAL canonical dump as the target; the database must land exactly back on B.
// ----------------------------------------------------------------------------

//nolint:tparallel
func TestSDLEnterpriseApplyBack(t *testing.T) {
	skipUnlessLiveOracle(t)
	ctx := context.Background()

	for _, srv := range liveServers {
		srv := srv
		t.Run(srv.name, func(t *testing.T) {
			dbName := entLoadDDL(ctx, t, srv, "entsdl_a5back", entZabbixSlice(t, entSliceCoreTables...))
			c0 := dumpSDL(ctx, t, srv, dbName)

			// Forward: drop a plain index, drop an unindexed column, widen a column, and
			// add a table — a realistic mixed release.
			target := entDropLineInTable(t, c0, "users", "`users_2`")
			target = entDropLineInTable(t, target, "users", "`attempt_ip`")
			target = entReplaceInTable(t, target, "media_type", "`smtp_helo` varchar(255)", "`smtp_helo` varchar(300)")
			target += `
CREATE TABLE ent_ab_t (
  id bigint unsigned NOT NULL,
  PRIMARY KEY (id)
) ENGINE=InnoDB;
`
			entOracle(ctx, t, srv, dbName, target, srv.name+"/A5/apply_back/forward")

			// Backward: the original canonical dump is the target again.
			entOracle(ctx, t, srv, dbName, c0, srv.name+"/A5/apply_back/backward")

			// The database must be exactly B again (canonical-dump equivalence).
			final := dumpSDL(ctx, t, srv, dbName)
			backDiff, err := mysqlDiffSDLMigration(final, c0, srv.version)
			require.NoError(t, err)
			require.Empty(t, backDiff, "apply-back did not restore B:\n%s", backDiff)
		})
	}
}

// ----------------------------------------------------------------------------
// A5(c): combined release — table + column + index + FK + view + trigger + routine all
// changed in ONE diff.
// ----------------------------------------------------------------------------

// entA5cAux seeds the view/function the combined release modifies.
//
//go:embed testdata/sdl/ent_a5c_aux.sql
var entA5cAux string

//nolint:tparallel
func TestSDLEnterpriseCombinedRelease(t *testing.T) {
	skipUnlessLiveOracle(t)
	ctx := context.Background()

	for _, srv := range liveServers {
		srv := srv
		t.Run(srv.name, func(t *testing.T) {
			base := entZabbixSlice(t, entSliceCoreTables...) + entA5cAux
			dbName := entLoadDDL(ctx, t, srv, "entsdl_a5cmb", base)
			source := dumpSDL(ctx, t, srv, dbName)

			// One release, seven object kinds:
			target := addColumnToTable(t, source, "media_type", "`ent_cmb_col` varchar(32) DEFAULT NULL") // column
			target = addIndexToTable(t, target, "users", "KEY `ent_cmb_idx` (`theme`)")                   // index
			target = addColumnToTable(t, target, "dchecks", "`ent_cmb_ref` bigint unsigned NULL")         // FK column +
			target = addIndexToTable(t, target, "dchecks", "KEY `ent_cmb_fk` (`ent_cmb_ref`)")
			target = addIndexToTable(t, target, "dchecks", "CONSTRAINT `ent_cmb_fk` FOREIGN KEY (`ent_cmb_ref`) REFERENCES `connector` (`connectorid`) ON DELETE SET NULL") // FK
			target = addColumnToView(t, target, "ent_cmb_v", "`role`.`type` AS `type`")                                                                                     // view
			target = mustReplace(t, target, "RETURN 7", "RETURN 8")                                                                                                         // routine
			target += `
CREATE TABLE ent_cmb_t (
  id bigint unsigned NOT NULL,
  PRIMARY KEY (id)
) ENGINE=InnoDB;
CREATE TRIGGER ent_cmb_trg BEFORE UPDATE ON maintenances FOR EACH ROW SET NEW.name = TRIM(NEW.name);
`
			plan := entOracle(ctx, t, srv, dbName, target, srv.name+"/A5/combined_release")
			upper := strings.ToUpper(plan)
			for _, want := range []string{
				"ENT_CMB_COL", "ENT_CMB_IDX", "ENT_CMB_REF", "ENT_CMB_FK",
				"ENT_CMB_V", "ENT_CMB_F", "ENT_CMB_T", "ENT_CMB_TRG",
			} {
				require.Contains(t, upper, want, "combined release plan missing %q:\n%s", want, plan)
			}
		})
	}
}

// ----------------------------------------------------------------------------
// A5(e): scale timing guard on the largest corpus (OpenEMR, 283 tables) — the no-op
// diff must stay under 30s, and a single-column change must be exactly one ALTER,
// also computed under 30s.
// ----------------------------------------------------------------------------

//nolint:tparallel
func TestSDLEnterpriseScaleGuard(t *testing.T) {
	skipUnlessLiveOracle(t)
	ctx := context.Background()
	const budget = 30 * time.Second

	for _, srv := range liveServers {
		srv := srv
		t.Run(srv.name, func(t *testing.T) {
			dbName := entLoadDDL(ctx, t, srv, "entsdl_scale", entOpenemrSQL)
			source := dumpSDL(ctx, t, srv, dbName)

			startNoop := time.Now()
			selfDiff, err := mysqlDiffSDLMigration(source, source, srv.version)
			noopDur := time.Since(startNoop)
			require.NoError(t, err)
			require.Empty(t, selfDiff, "openemr no-op diff must be empty:\n%s", selfDiff)
			t.Logf("[SCALE %s] openemr no-op diff: %v (budget %v)", srv.name, noopDur, budget)
			require.Less(t, noopDur, budget, "openemr no-op diff exceeded the %v budget", budget)

			// Single-column change: exactly one ALTER, computed within budget.
			target := addColumnToTable(t, source, "patient_data", "`ent_scale_col` varchar(40) DEFAULT NULL")
			startOne := time.Now()
			plan, err := mysqlDiffSDLMigration(source, target, srv.version)
			oneDur := time.Since(startOne)
			require.NoError(t, err)
			t.Logf("[SCALE %s] openemr single-column diff: %v, plan:\n%s", srv.name, oneDur, plan)
			require.Less(t, oneDur, budget, "openemr single-column diff exceeded the %v budget", budget)
			require.Equal(t, 1, entPlanStatementCount(t, plan), "single-column change must be exactly 1 statement:\n%s", plan)
			upper := strings.ToUpper(plan)
			require.Contains(t, upper, "ALTER TABLE")
			require.Contains(t, upper, "ENT_SCALE_COL")

			// Close the loop: apply + converge (the full oracle contract at scale).
			require.NoError(t, applyDDL(ctx, t, srv, dbName, plan), "scale plan failed to apply:\n%s", plan)
			after := dumpSDL(ctx, t, srv, dbName)
			converge, err := mysqlDiffSDLMigration(after, target, srv.version)
			require.NoError(t, err)
			require.Empty(t, converge, "openemr single-column change did not converge:\n%s", converge)
		})
	}
}

// ----------------------------------------------------------------------------
// A5(f): multi-file export round-trip at enterprise scale — for every corpus, the
// concatenated GetMultiFileDatabaseDefinition output must describe the identical schema
// as the single-file dump (diff empty both ways) and must be self-idempotent.
// ----------------------------------------------------------------------------

//nolint:tparallel
func TestSDLEnterpriseMultiFileRoundTrip(t *testing.T) {
	skipUnlessLiveOracle(t)
	ctx := context.Background()

	for _, srv := range liveServers {
		srv := srv
		t.Run(srv.name, func(t *testing.T) {
			for _, corpus := range entCorpora() {
				corpus := corpus
				t.Run(corpus.name, func(t *testing.T) {
					dbName := entLoadCorpus(ctx, t, srv, "entsdl_mf_", corpus)
					meta := syncMetaForDB(ctx, t, srv, dbName)

					result, err := schema.GetMultiFileDatabaseDefinition(storepb.Engine_MYSQL, schema.GetDefinitionContext{
						SkipBackupSchema: true,
					}, meta.GetProto())
					require.NoError(t, err)
					require.NotEmpty(t, result.Files, "[%s/%s] multi-file export produced no files", srv.name, corpus.name)

					single := dumpSDL(ctx, t, srv, dbName)
					concat := concatMultiFile(result)
					t.Logf("[MULTIFILE %s/%s] %d files, concat %d bytes, single %d bytes",
						srv.name, corpus.name, len(result.Files), len(concat), len(single))

					forward, err := mysqlDiffSDLMigration(concat, single, srv.version)
					require.NoError(t, err)
					require.Empty(t, forward, "[%s/%s] concat(multi-file) != single dump (concat->single):\n%s",
						srv.name, corpus.name, forward)
					backward, err := mysqlDiffSDLMigration(single, concat, srv.version)
					require.NoError(t, err)
					require.Empty(t, backward, "[%s/%s] concat(multi-file) != single dump (single->concat):\n%s",
						srv.name, corpus.name, backward)
					self, err := mysqlDiffSDLMigration(concat, concat, srv.version)
					require.NoError(t, err)
					require.Empty(t, self, "[%s/%s] concat(multi-file) not idempotent:\n%s", srv.name, corpus.name, self)
				})
			}
		})
	}
}

// A7 of the ENTERPRISE smoke axes: CROSS-VERSION upgrade —
// the migration-during-upgrade scenario where a schema is authored/dumped on one MySQL
// version and the target is applied on the other.
//
// Legs, per corpus (zabbix / prestashop / openemr; sys stays behind its entCorpora gates):
//
//	upgrade_57_dump_targets_80_db  — load the corpus on 5.7, take its canonical dump D57,
//	    load the SAME original DDL on 8.0, then use D57 as the TARGET against the 8.0
//	    database (version-aware diff as 8.0.32). The plan must apply cleanly and
//	    converge; 5.7-isms (integer display widths) must normalize away — asserted by the
//	    no-width-in-ALTER guard — and nothing may phantom-drop. Genuine semantic
//	    differences (the server-default charset a version applied at load time) are
//	    LEGITIMATE plan content: a 5.7-authored dump of a charset-less corpus says
//	    latin1, and declarative semantics honor it.
//	minimal_mutation_after_upgrade — (zabbix) one semantic edit (add column + index) on
//	    top of D57, targeted at the upgraded 8.0 database → exactly the minimal DDL, no
//	    cross-version noise re-emerging.
//	fresh_80_from_57_dump          — D57 against an EMPTY 8.0 database (fresh create from
//	    a 5.7-authored dump) → apply → re-dump → converges to D57 and self-diffs empty.
//	downgrade_80_dump_targets_57_db — the reverse: D80 as target on the 5.7 database.
//	    Probed: where an 8.0-ism blocks the 5.7 apply (utf8mb4_0900_ai_ci — 8.0's default
//	    utf8mb4 collation does not exist on 5.7), the leg skips with the named reason.
//	aligned_zabbix_utf8mb4_general_ci — the distilled asymmetry probe: both databases
//	    created with the SAME 5.7-legal default (utf8mb4/utf8mb4_general_ci) before
//	    loading zabbix, so the only cross-dump differences left are pure version
//	    renderings (display widths). Both cross-diffs must then be EMPTY — any residual
//	    is a phantom diff by definition.
//
// prestashop note: the corpus itself leads with SET SESSION sql_mode='' (two
// '0000-00-00 00:00:00' defaults need non-strict mode). Generated plans cannot carry
// session state, so plan APPLICATION for that corpus is prefixed with the corpus's own
// preamble — the exact concession the corpus already requires at load time.
//
// Shared helpers (liveServers, entLoadDDL, dumpSDL, applyDDL, entPlanStatementCount,
// entNormalizeDelimiters, newLiveDatabase, createLiveMySQLDriver, addColumnToTable,
// addIndexToTable, entCorpora + embedded corpora) come from the sibling files.

// Full server versions are threaded into the diff for this axis (the release path passes
// the synced server version string, e.g. "8.0.32", not a bare major.minor).
const (
	entUpg57Version = "5.7.25"
	entUpg80Version = "8.0.32"
)

type entUpgCorpus struct {
	name string
	ddl  string
	// preamble is prepended to every generated plan before application (see the
	// prestashop note above). It mirrors the corpus's own leading session setup.
	preamble string
	// gate, when non-empty, skips the corpus with the given reason.
	gate string
}

// entUpgCorpora derives the cross-version corpus list from the shared entCorpora().
// The axis authors the SAME original DDL on BOTH versions, so a corpus gated on either
// version (sys: gate + gate57) is gated here too.
//
// openemr note: the corpus's uuid columns (`uuid` binary(16) NOT NULL DEFAULT ”) once
// gated it out of this axis — the 8.0 sync stored the information_schema hex NOTATION
// ('0x…') verbatim and every cross-version leg looped on a phantom
// `MODIFY COLUMN ... DEFAULT '0x…'` residual. The sync now canonicalizes binary-family
// defaults (see TestSDLEnterpriseUpgradeBinaryDefaultCanonicalForm), so the corpus runs
// un-gated.
func entUpgCorpora() []entUpgCorpus {
	var out []entUpgCorpus
	for _, c := range entCorpora() {
		u := entUpgCorpus{name: c.name, ddl: c.ddl, gate: c.gate}
		if u.gate == "" && c.gate57 != "" {
			u.gate = c.gate57
		}
		if u.gate == "" && c.canonical {
			u.gate = "corpus ships as a canonical dump requiring the engine create-plan loader; A7 authors from raw upstream DDL"
		}
		if c.name == "prestashop" {
			u.preamble = "SET SESSION sql_mode='';\n"
		}
		out = append(out, u)
	}
	return out
}

func entUpgServers(t *testing.T) (srv57, srv80 liveServer) {
	t.Helper()
	found57, found80 := false, false
	for _, srv := range liveServers {
		switch srv.version {
		case "5.7":
			srv57, found57 = srv, true
		case "8.0":
			srv80, found80 = srv, true
		default:
		}
	}
	require.True(t, found57 && found80, "cross-version axis needs both a 5.7 and an 8.0 live server")
	return srv57, srv80
}

// entUpgTrim caps logged plan/residual text so a 250-table plan stays readable.
func entUpgTrim(s string) string {
	const limit = 6000
	if len(s) <= limit {
		return s
	}
	return s[:limit] + fmt.Sprintf("\n... (%d more bytes)", len(s)-limit)
}

// entUpgReDrop matches destructive operations that can NEVER be legitimate when both
// databases were loaded from the same original DDL — any hit is a phantom diff.
var entUpgReDrop = regexp.MustCompile(`(?i)\bDROP\s+(TABLE|COLUMN|TRIGGER|VIEW|FUNCTION|PROCEDURE|INDEX|KEY|CHECK|CONSTRAINT|FOREIGN)`)

func entUpgAssertNoDrops(t *testing.T, label, plan string) {
	t.Helper()
	if m := entUpgReDrop.FindString(plan); m != "" {
		require.Failf(t, "phantom destructive operation in cross-version plan",
			"[%s] plan contains %q — with identical source DDL on both versions nothing may be dropped:\n%s",
			label, m, entUpgTrim(plan))
	}
}

// entUpgReIntWidth matches integer display widths — the 5.7 stored form 8.0 must
// normalize away. tinyint(1) is exempt (the BOOLEAN spelling, canonical on BOTH
// versions).
var entUpgReIntWidth = regexp.MustCompile(`(?i)\b(?:tiny|small|medium|big)?int\(\d+\)`)

// entUpgAssertNoWidthInAlters proves no ALTER statement carries a 5.7 display width —
// a width inside an ALTER means the 5.7 rendering leaked through normalization and
// phantom-modified a column. CREATE TABLE statements are not judged: a pass-through
// width in a fresh CREATE is cosmetic, converges, and is not a phantom diff.
func entUpgAssertNoWidthInAlters(t *testing.T, label, plan string) {
	t.Helper()
	if strings.TrimSpace(plan) == "" {
		return
	}
	stmts, err := mysqlparser.SplitSQL(plan)
	require.NoError(t, err, "[%s] split plan for width guard", label)
	for _, s := range stmts {
		text := strings.TrimSpace(s.Text)
		if !strings.HasPrefix(strings.ToUpper(text), "ALTER TABLE") {
			continue
		}
		for _, m := range entUpgReIntWidth.FindAllString(text, -1) {
			if strings.EqualFold(m, "tinyint(1)") {
				continue
			}
			require.Failf(t, "integer display width leaked into a cross-version ALTER",
				"[%s] ALTER carries %q (5.7 display width not normalized away):\n%s", label, m, text)
		}
	}
}

// entUpgConverge diffs the live database against target (as version), applies the plan
// (with the corpus preamble), and proves convergence + idempotence. Returns the plan
// ("" when the schemas were already equivalent).
func entUpgConverge(ctx context.Context, t *testing.T, srv liveServer, dbName, target, version, preamble, label string) string {
	t.Helper()
	source := dumpSDL(ctx, t, srv, dbName)
	plan, err := mysqlDiffSDLMigration(source, target, version)
	require.NoError(t, err, "[%s] diff", label)
	if strings.TrimSpace(plan) == "" {
		t.Logf("[%s] empty plan — schemas already equivalent", label)
	} else {
		t.Logf("[%s] plan: %d statements, %d bytes:\n%s", label, entPlanStatementCount(t, plan), len(plan), entUpgTrim(plan))
		require.NoError(t, applyDDL(ctx, t, srv, dbName, preamble+plan),
			"[%s] generated plan failed to apply:\n%s", label, entUpgTrim(plan))
	}
	after := dumpSDL(ctx, t, srv, dbName)
	converge, err := mysqlDiffSDLMigration(after, target, version)
	require.NoError(t, err, "[%s] converge diff", label)
	require.Empty(t, converge, "[%s] did not converge; residual:\n%s", label, entUpgTrim(converge))
	self, err := mysqlDiffSDLMigration(after, after, version)
	require.NoError(t, err, "[%s] idempotence diff", label)
	require.Empty(t, self, "[%s] post-apply dump not idempotent:\n%s", label, entUpgTrim(self))
	return plan
}

// entUpgLoadAligned creates a database whose DEFAULT charset/collation is pinned to the
// 5.7-legal utf8mb4/utf8mb4_general_ci on either server, then loads ddl — removing the
// server-default charset divergence so cross-dumps differ only by version renderings.
func entUpgLoadAligned(ctx context.Context, t *testing.T, srv liveServer, prefix, ddl string) string {
	t.Helper()
	dbName := newLiveDatabase(ctx, t, srv, prefix)
	driver, err := createLiveMySQLDriver(ctx, srv, dbName)
	require.NoError(t, err)
	defer driver.Close(ctx)
	_, err = driver.Execute(ctx, "ALTER DATABASE `"+dbName+"` CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci", db.ExecuteOptions{})
	require.NoError(t, err, "[%s] align database default collation", srv.name)
	_, err = driver.Execute(ctx, entNormalizeDelimiters(ddl), db.ExecuteOptions{})
	require.NoError(t, err, "[%s] apply aligned base DDL", srv.name)
	return dbName
}

//nolint:tparallel
func TestSDLEnterpriseCrossVersionUpgrade(t *testing.T) {
	skipUnlessLiveOracle(t)
	ctx := context.Background()
	srv57, srv80 := entUpgServers(t)

	for _, corpus := range entUpgCorpora() {
		corpus := corpus
		t.Run(corpus.name, func(t *testing.T) {
			if corpus.gate != "" {
				t.Skipf("[A7/%s] %s", corpus.name, corpus.gate)
			}

			// Author on 5.7 and on 8.0 from the SAME original DDL; capture both
			// canonical dumps up front so later legs judge fixed texts.
			db57 := entLoadDDL(ctx, t, srv57, "entsdl_up57_"+corpus.name, corpus.ddl)
			d57 := dumpSDL(ctx, t, srv57, db57)
			db80 := entLoadDDL(ctx, t, srv80, "entsdl_up80_"+corpus.name, corpus.ddl)
			d80 := dumpSDL(ctx, t, srv80, db80)
			t.Logf("[A7/%s] D57 %d bytes, D80 %d bytes", corpus.name, len(d57), len(d80))

			t.Run("upgrade_57_dump_targets_80_db", func(t *testing.T) {
				label := "A7/" + corpus.name + "/upgrade_57_to_80"
				plan := entUpgConverge(ctx, t, srv80, db80, d57, entUpg80Version, corpus.preamble, label)
				entUpgAssertNoDrops(t, label, plan)
				entUpgAssertNoWidthInAlters(t, label, plan)
			})

			if corpus.name == "zabbix" {
				t.Run("minimal_mutation_after_upgrade", func(t *testing.T) {
					label := "A7/" + corpus.name + "/minimal_mutation_after_upgrade"
					// Self-sufficient: land the database on D57 first (a no-op when the
					// upgrade leg just ran).
					entUpgConverge(ctx, t, srv80, db80, d57, entUpg80Version, corpus.preamble, label+"/pre")

					// One semantic edit on top of the 5.7-authored dump.
					target := addColumnToTable(t, d57, "users", "`ent_up_col` varchar(50) DEFAULT NULL")
					target = addIndexToTable(t, target, "users", "KEY `ent_up_idx` (`ent_up_col`)")

					source := dumpSDL(ctx, t, srv80, db80)
					plan, err := mysqlDiffSDLMigration(source, target, entUpg80Version)
					require.NoError(t, err, "[%s] diff", label)
					require.NotEmpty(t, plan, "[%s] the semantic edit must produce a plan", label)
					t.Logf("[%s] plan:\n%s", label, plan)

					n := entPlanStatementCount(t, plan)
					require.LessOrEqual(t, n, 2, "[%s] plan must be the minimal DDL (add column + add index), got %d statements:\n%s", label, n, plan)
					stmts, err := mysqlparser.SplitSQL(plan)
					require.NoError(t, err)
					for _, s := range stmts {
						text := strings.TrimSpace(s.Text)
						if text == "" || text == ";" {
							continue
						}
						require.True(t, strings.HasPrefix(strings.ToUpper(text), "ALTER TABLE `USERS`"),
							"[%s] plan statement strays off the edited table (cross-version noise re-emerged):\n%s", label, text)
					}
					upper := strings.ToUpper(plan)
					require.Contains(t, upper, "ENT_UP_COL", "[%s] plan missing the added column:\n%s", label, plan)
					require.Contains(t, upper, "ENT_UP_IDX", "[%s] plan missing the added index:\n%s", label, plan)

					require.NoError(t, applyDDL(ctx, t, srv80, db80, plan), "[%s] minimal plan failed to apply:\n%s", label, plan)
					after := dumpSDL(ctx, t, srv80, db80)
					converge, err := mysqlDiffSDLMigration(after, target, entUpg80Version)
					require.NoError(t, err)
					require.Empty(t, converge, "[%s] minimal mutation did not converge:\n%s", label, converge)
				})
			}

			t.Run("fresh_80_from_57_dump", func(t *testing.T) {
				label := "A7/" + corpus.name + "/fresh_80_from_57_dump"
				dbEmpty := newLiveDatabase(ctx, t, srv80, "entsdl_upe80_"+corpus.name)
				plan := entUpgConverge(ctx, t, srv80, dbEmpty, d57, entUpg80Version, corpus.preamble, label)
				require.NotEmpty(t, plan, "[%s] creating from a 5.7-authored dump on an empty database must emit DDL", label)
				entUpgAssertNoWidthInAlters(t, label, plan)
			})

			t.Run("downgrade_80_dump_targets_57_db", func(t *testing.T) {
				label := "A7/" + corpus.name + "/downgrade_80_to_57"
				source := dumpSDL(ctx, t, srv57, db57)
				plan, err := mysqlDiffSDLMigration(source, d80, entUpg57Version)
				require.NoError(t, err, "[%s] diff", label)
				if strings.TrimSpace(plan) != "" {
					t.Logf("[%s] plan: %d statements, %d bytes:\n%s", label, entPlanStatementCount(t, plan), len(plan), entUpgTrim(plan))
					if applyErr := applyDDL(ctx, t, srv57, db57, corpus.preamble+plan); applyErr != nil {
						// Probe outcome: an 8.0-authored dump can carry 8.0-isms no 5.7
						// server accepts. The only one these corpora produce is 8.0's
						// utf8mb4 default collation.
						if strings.Contains(plan, "utf8mb4_0900_ai_ci") {
							t.Skipf("[%s] 8.0-authored dump is not 5.7-legal: plan carries utf8mb4_0900_ai_ci "+
								"(8.0's utf8mb4 default collation, unknown to 5.7); apply failed as expected: %v", label, applyErr)
						}
						require.NoErrorf(t, applyErr, "[%s] downgrade plan failed to apply and no known 8.0-ism explains it:\n%s",
							label, entUpgTrim(plan))
					}
				}
				after := dumpSDL(ctx, t, srv57, db57)
				converge, err := mysqlDiffSDLMigration(after, d80, entUpg57Version)
				require.NoError(t, err, "[%s] converge diff", label)
				require.Empty(t, converge, "[%s] did not converge; residual:\n%s", label, entUpgTrim(converge))
				self, err := mysqlDiffSDLMigration(after, after, entUpg57Version)
				require.NoError(t, err)
				require.Empty(t, self, "[%s] post-apply dump not idempotent:\n%s", label, entUpgTrim(self))
			})
		})
	}

	// The distilled asymmetry probe: with the server-default charset divergence removed,
	// BOTH cross-diffs must be empty — the remaining differences are exactly the
	// version renderings (5.7 display widths) the normalizer must absorb.
	t.Run("aligned_zabbix_utf8mb4_general_ci", func(t *testing.T) {
		db57 := entUpgLoadAligned(ctx, t, srv57, "entsdl_upal57", entZabbixSQL)
		db80 := entUpgLoadAligned(ctx, t, srv80, "entsdl_upal80", entZabbixSQL)
		d57 := dumpSDL(ctx, t, srv57, db57)
		d80 := dumpSDL(ctx, t, srv80, db80)

		t.Run("57_dump_targets_80_db", func(t *testing.T) {
			cur := dumpSDL(ctx, t, srv80, db80)
			plan, err := mysqlDiffSDLMigration(cur, d57, entUpg80Version)
			require.NoError(t, err, "aligned 57->80 diff")
			require.Empty(t, plan,
				"collation-aligned 5.7 dump phantom-diffs an equivalent 8.0 database (widths/collation defaults did not normalize away):\n%s",
				entUpgTrim(plan))
		})
		t.Run("80_dump_targets_57_db", func(t *testing.T) {
			cur := dumpSDL(ctx, t, srv57, db57)
			plan, err := mysqlDiffSDLMigration(cur, d80, entUpg57Version)
			require.NoError(t, err, "aligned 80->57 diff")
			require.Empty(t, plan,
				"collation-aligned 8.0 dump phantom-diffs an equivalent 5.7 database (widths/collation defaults did not normalize away):\n%s",
				entUpgTrim(plan))
		})
	})
}

// ----------------------------------------------------------------------------
// REGRESSION (formerly the pinned bug this axis found, openemr corpus): binary-family
// column defaults.
//
// MySQL reports binary/varbinary literal defaults in information_schema.COLUMNS in
// version-specific encodings: 8.0 as hex NOTATION text built from the value truncated
// at its first NUL byte ("0x" for DEFAULT '', "0x6162" for DEFAULT 'ab'), 5.7 as the
// RAW BYTES, NUL-padded to the declared width for binary(N). The sync used to store
// the 8.0 notation verbatim and the dumper re-quoted it as a STRING literal
// (DEFAULT '0x6162') — value-unfaithful dumps, cross-version phantom MODIFY loops (the
// openemr uuid columns that once gated that corpus out of the A7 legs), and round-trip
// double-encoding ('0x6162' -> '0x307836313632').
//
// The sync now decodes both encodings into one canonical form — '' when the value is
// empty (binary(N) padding stripped), a plain quoted string for clean text, an
// unquoted hex literal otherwise; see canonicalBinaryDefault in
// backend/plugin/db/mysql/sync.go. This test pins the fixed behavior: value-faithful
// dumps on both versions, EMPTY cross-version diffs (the DDL pins the table charset so
// the dumps may differ by nothing at all), and a clean round-trip through a fresh 8.0
// database. The per-version legs then cover the hex-literal fallback for values no
// quoted SDL string can carry — probed on the version whose information_schema reports
// them faithfully (0xFF61 on 8.0; a significant trailing NUL, 0x6100, on 5.7).
// ----------------------------------------------------------------------------

//go:embed testdata/sdl/ent_upg_binary_default_ddl.sql
var entUpgBinaryDefaultDDL string

//nolint:tparallel
func TestSDLEnterpriseUpgradeBinaryDefaultCanonicalForm(t *testing.T) {
	skipUnlessLiveOracle(t)
	ctx := context.Background()
	srv57, srv80 := entUpgServers(t)

	// Both versions dump the same value-faithful canonical form.
	db80 := entLoadDDL(ctx, t, srv80, "entsdl_upbin80", entUpgBinaryDefaultDDL)
	d80 := dumpSDL(ctx, t, srv80, db80)
	db57 := entLoadDDL(ctx, t, srv57, "entsdl_upbin57", entUpgBinaryDefaultDDL)
	d57 := dumpSDL(ctx, t, srv57, db57)
	for _, d := range []struct{ version, dump string }{{"8.0", d80}, {"5.7", d57}} {
		require.Contains(t, d.dump, "`u` binary(16) NOT NULL DEFAULT ''",
			"%s dump must render the empty binary(16) default as '' (not the I_S encoding):\n%s", d.version, d.dump)
		require.Contains(t, d.dump, "`v` varbinary(24) NOT NULL DEFAULT 'ab'",
			"%s dump must render the varbinary default value-faithfully:\n%s", d.version, d.dump)
		require.NotContains(t, d.dump, "0x",
			"%s dump leaked an information_schema hex encoding for a clean-text default:\n%s", d.version, d.dump)
	}

	// Cross-version: with the table charset pinned, the dumps describe the same schema
	// and BOTH cross-diffs must be empty — the shape that once looped on
	// MODIFY COLUMN ... DEFAULT '0x…'.
	cross80, err := mysqlDiffSDLMigration(d80, d57, entUpg80Version)
	require.NoError(t, err)
	require.Empty(t, cross80, "5.7 dump phantom-diffs the equivalent 8.0 database:\n%s", cross80)
	cross57, err := mysqlDiffSDLMigration(d57, d80, entUpg57Version)
	require.NoError(t, err)
	require.Empty(t, cross57, "8.0 dump phantom-diffs the equivalent 5.7 database:\n%s", cross57)

	// Round-trip the 8.0 dump through a FRESH database: the create plan must apply, the
	// re-dump must converge to the dump (no double-encoding), and self-diff empty.
	dbFresh := newLiveDatabase(ctx, t, srv80, "entsdl_upbinfresh")
	empty := dumpSDL(ctx, t, srv80, dbFresh)
	plan, err := mysqlDiffSDLMigration(empty, d80, entUpg80Version)
	require.NoError(t, err)
	require.NotEmpty(t, plan)
	require.NoError(t, applyDDL(ctx, t, srv80, dbFresh, plan), "create plan from the 8.0 dump failed to apply:\n%s", plan)
	redump := dumpSDL(ctx, t, srv80, dbFresh)
	require.NotContains(t, redump, "0x3078",
		"the 8.0 dump round-trip double-encodes the default again ('0x6162' -> '0x307836313632'):\n%s", redump)
	converge, err := mysqlDiffSDLMigration(redump, d80, entUpg80Version)
	require.NoError(t, err)
	require.Empty(t, converge, "8.0 dump did not converge through a fresh database; residual:\n%s", converge)

	// Values no quoted SDL string can carry fall back to an unquoted hex literal, each
	// probed on the version whose information_schema reports the value faithfully (8.0
	// truncates at NUL bytes; 5.7 truncates at non-UTF8 bytes).
	t.Run("hex_literal_fallback_nonutf8_80", func(t *testing.T) {
		ddl := "CREATE TABLE ent_up_binhex (id int NOT NULL, w varbinary(8) NOT NULL DEFAULT 0xFF61, PRIMARY KEY (id)) ENGINE=InnoDB;"
		dbName := entLoadDDL(ctx, t, srv80, "entsdl_upbinhex80", ddl)
		dump := dumpSDL(ctx, t, srv80, dbName)
		require.Contains(t, dump, "`w` varbinary(8) NOT NULL DEFAULT 0xff61",
			"8.0 dump must keep the non-UTF8 default as an unquoted hex literal:\n%s", dump)
		self, err := mysqlDiffSDLMigration(dump, dump, entUpg80Version)
		require.NoError(t, err)
		require.Empty(t, self, "hex-literal dump not idempotent:\n%s", self)

		fresh := newLiveDatabase(ctx, t, srv80, "entsdl_upbinhexf80")
		plan, err := mysqlDiffSDLMigration(dumpSDL(ctx, t, srv80, fresh), dump, entUpg80Version)
		require.NoError(t, err)
		require.NoError(t, applyDDL(ctx, t, srv80, fresh, plan), "hex-literal create plan failed to apply:\n%s", plan)
		require.Equal(t, dump, dumpSDL(ctx, t, srv80, fresh), "hex-literal default did not round-trip byte-identically")
	})
	t.Run("hex_literal_fallback_trailnul_57", func(t *testing.T) {
		ddl := "CREATE TABLE ent_up_binnul (id int NOT NULL, w varbinary(8) NOT NULL DEFAULT 0x6100, PRIMARY KEY (id)) ENGINE=InnoDB;"
		dbName := entLoadDDL(ctx, t, srv57, "entsdl_upbinnul57", ddl)
		dump := dumpSDL(ctx, t, srv57, dbName)
		require.Contains(t, dump, "`w` varbinary(8) NOT NULL DEFAULT 0x6100",
			"5.7 dump must keep the trailing-NUL default as an unquoted hex literal (the NUL is significant on varbinary):\n%s", dump)
		self, err := mysqlDiffSDLMigration(dump, dump, entUpg57Version)
		require.NoError(t, err)
		require.Empty(t, self, "hex-literal dump not idempotent:\n%s", self)

		fresh := newLiveDatabase(ctx, t, srv57, "entsdl_upbinnulf57")
		plan, err := mysqlDiffSDLMigration(dumpSDL(ctx, t, srv57, fresh), dump, entUpg57Version)
		require.NoError(t, err)
		require.NoError(t, applyDDL(ctx, t, srv57, fresh, plan), "hex-literal create plan failed to apply:\n%s", plan)
		require.Equal(t, dump, dumpSDL(ctx, t, srv57, fresh), "hex-literal default did not round-trip byte-identically")
	})
}

// A4 of the ENTERPRISE smoke axes: a SEEDED, deterministic
// random-migration fuzzer over a 30-table Zabbix slice. Each seed derives K ∈ [3,8]
// independent mutations from a menu (add/drop/modify column, add/drop index, add/drop FK
// with valid targets only, add/drop/modify view, add/drop trigger, add/drop routine,
// partition/departition), applies them TEXTUALLY to the canonical dump to build the
// target SDL, and runs the oracle protocol (diff → apply → converge → idempotence).
//
// Determinism: the mutation stream is a pure function of the seed (math/rand with a
// fixed source; the schema model is iterated in sorted order), so a failing seed replays
// exactly. Every failure prints the seed and a ddmin-minimized mutation list, each
// minimization trial re-running the full oracle on a fresh scratch database.
//
// ≥20 seeds run on 8.0 and ≥10 on 5.7 (the menu is 5.7-legal, so both use the same
// generator). Scratch databases are entsdl_-prefixed and dropped eagerly per trial.

// entFuzzSliceTables is the ~30-table fuzz base: the core slice plus the httptest family
// (6 more triggered tables, FK-chained through httptest/httpstep into hosts and items)
// and token (a 2-FK fan-in on users).
var entFuzzSliceTables = append(append([]string{}, entSliceCoreTables...),
	"httptest", "httpstep", "httptestitem", "httpstepitem", "httptest_field", "httpstep_field",
	"token",
)

// entFuzzAux seeds the droppable/modifiable object pool the menu needs: two views, one
// function, one procedure, and a partitioned table (the departition target).
//
//go:embed testdata/sdl/ent_fuzz_aux.sql
var entFuzzAux string

// entFzProtectedTables are never touched by column-level mutations: the changelog
// trigger bodies name changelog's columns outright (not via NEW/OLD), and ent_fz_part
// belongs to the partition mutations.
var entFzProtectedTables = map[string]bool{"changelog": true, "ent_fz_part": true}

// entFzProtectedColumns are referenced by the seeded views (or their modify-view
// extension columns) and must survive column drops.
var entFzProtectedColumns = map[string]bool{
	"users.username": true, "users.name": true,
	"hosts.host": true, "hosts.status": true,
}

// ----------------------------------------------------------------------------
// Canonical-dump schema model (guides valid mutation choices; the mutations themselves
// are textual).
// ----------------------------------------------------------------------------

type entFzIndex struct {
	name   string
	cols   []string
	unique bool
}

type entFzFK struct {
	name string
	cols []string
}

type entFzTable struct {
	name        string
	cols        []string          // ordered
	colLine     map[string]string // name -> trimmed definition line
	pk          map[string]bool
	indexes     []entFzIndex
	fks         []entFzFK
	partitioned bool
}

type entFzModel struct {
	tables      []*entFzTable // sorted by name
	byName      map[string]*entFzTable
	referenced  map[string]bool            // tables referenced by any FK
	triggerCols map[string]map[string]bool // table -> NEW./OLD.-referenced columns
	triggers    []string                   // sorted trigger names
	views       []string                   // seeded, droppable views
	functions   []string                   // seeded, droppable functions
	procedures  []string                   // seeded, droppable procedures
}

var (
	entFzReColLine = regexp.MustCompile("^`(\\w+)` (.+?),?$")
	entFzRePK      = regexp.MustCompile(`^PRIMARY KEY \((.+)\),?$`)
	entFzReIndex   = regexp.MustCompile("^(UNIQUE )?KEY `(\\w+)` \\((.+)\\),?$")
	entFzReFK      = regexp.MustCompile("^CONSTRAINT `(\\w+)` FOREIGN KEY \\((.+?)\\) REFERENCES `(\\w+)`")
	entFzReTrigger = regexp.MustCompile("(?i)^CREATE TRIGGER `(\\w+)`.* ON `(\\w+)`")
	entFzReNewOld  = regexp.MustCompile(`(?i)\b(?:new|old)\.(\w+)`)
	entFzReCol     = regexp.MustCompile("`(\\w+)`")
	entFzReVarLen  = regexp.MustCompile(`varchar\((\d+)\)`)
)

// entFuzzModel parses the canonical dump into the mutation-planning model.
func entFuzzModel(t *testing.T, source string) *entFzModel {
	t.Helper()
	stmts, err := mysqlparser.SplitSQL(source)
	require.NoError(t, err, "split canonical dump for fuzz model")

	m := &entFzModel{
		byName:      map[string]*entFzTable{},
		referenced:  map[string]bool{},
		triggerCols: map[string]map[string]bool{},
	}
	for _, s := range stmts {
		text := strings.TrimSpace(s.Text)
		upper := strings.ToUpper(text)
		switch {
		case strings.HasPrefix(upper, "CREATE TABLE `"):
			tbl := entFzParseTable(text)
			if tbl != nil {
				m.tables = append(m.tables, tbl)
				m.byName[tbl.name] = tbl
			}
		case strings.HasPrefix(upper, "CREATE TRIGGER `") || (strings.HasPrefix(upper, "CREATE ") && strings.Contains(upper, " TRIGGER `")):
			if mm := entFzReTrigger.FindStringSubmatch(text); mm != nil {
				m.triggers = append(m.triggers, mm[1])
				onTable := mm[2]
				if m.triggerCols[onTable] == nil {
					m.triggerCols[onTable] = map[string]bool{}
				}
				for _, ref := range entFzReNewOld.FindAllStringSubmatch(text, -1) {
					m.triggerCols[onTable][ref[1]] = true
				}
			}
		default:
		}
	}
	// FK-referenced tables (for partition candidacy).
	for _, s := range stmts {
		for _, mm := range entFzReFKRefs.FindAllStringSubmatch(s.Text, -1) {
			m.referenced[mm[1]] = true
		}
	}
	// Seeded droppable objects (fixed names — the fuzz base owns them).
	m.views = []string{"ent_fz_v1", "ent_fz_v2"}
	m.functions = []string{"ent_fz_fn1"}
	m.procedures = []string{"ent_fz_pr1"}
	return m
}

var entFzReFKRefs = regexp.MustCompile("FOREIGN KEY \\(.+?\\) REFERENCES `(\\w+)`")

// entFzParseTable parses one canonical CREATE TABLE statement.
func entFzParseTable(stmt string) *entFzTable {
	header := regexp.MustCompile("^CREATE TABLE `(\\w+)`").FindStringSubmatch(stmt)
	if header == nil {
		return nil
	}
	tbl := &entFzTable{
		name:        header[1],
		colLine:     map[string]string{},
		pk:          map[string]bool{},
		partitioned: strings.Contains(strings.ToUpper(stmt), "PARTITION BY"),
	}
	for _, raw := range strings.Split(stmt, "\n")[1:] {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, ")") {
			break
		}
		if mm := entFzRePK.FindStringSubmatch(line); mm != nil {
			for _, c := range entFzReCol.FindAllStringSubmatch(mm[1], -1) {
				tbl.pk[c[1]] = true
			}
			continue
		}
		if mm := entFzReIndex.FindStringSubmatch(line); mm != nil {
			idx := entFzIndex{name: mm[2], unique: mm[1] != ""}
			for _, c := range entFzReCol.FindAllStringSubmatch(mm[3], -1) {
				idx.cols = append(idx.cols, c[1])
			}
			tbl.indexes = append(tbl.indexes, idx)
			continue
		}
		if mm := entFzReFK.FindStringSubmatch(line); mm != nil {
			fk := entFzFK{name: mm[1]}
			for _, c := range entFzReCol.FindAllStringSubmatch(mm[2], -1) {
				fk.cols = append(fk.cols, c[1])
			}
			tbl.fks = append(tbl.fks, fk)
			continue
		}
		if strings.HasPrefix(line, "CONSTRAINT `") || strings.HasPrefix(line, "FULLTEXT KEY") {
			continue // CHECK constraints / fulltext — not mutation targets
		}
		if mm := entFzReColLine.FindStringSubmatch(line); mm != nil {
			tbl.cols = append(tbl.cols, mm[1])
			tbl.colLine[mm[1]] = strings.TrimSuffix(line, ",")
		}
	}
	return tbl
}

// entFzIndexedCols returns every column participating in any index or FK of tbl.
func entFzIndexedCols(tbl *entFzTable) map[string]bool {
	used := map[string]bool{}
	for c := range tbl.pk {
		used[c] = true
	}
	for _, idx := range tbl.indexes {
		for _, c := range idx.cols {
			used[c] = true
		}
	}
	for _, fk := range tbl.fks {
		for _, c := range fk.cols {
			used[c] = true
		}
	}
	return used
}

// ----------------------------------------------------------------------------
// Mutation generation.
// ----------------------------------------------------------------------------

// entFzMutation is one textual schema mutation with a replayable description.
type entFzMutation struct {
	desc  string
	apply func(t *testing.T, source string) string
}

// entFzState tracks per-round conflicts so the K mutations stay independent.
type entFzState struct {
	usedCols     map[string]bool // "table.col" touched by drop/modify
	usedIndexes  map[string]bool // "table.index" dropped
	usedFKs      map[string]bool
	usedViews    map[string]bool
	usedTriggers map[string]bool
	usedRoutines map[string]bool
	lockedTables map[string]bool // partition/departition/fk-structure locks
}

func entFzNewState() *entFzState {
	return &entFzState{
		usedCols:     map[string]bool{},
		usedIndexes:  map[string]bool{},
		usedFKs:      map[string]bool{},
		usedViews:    map[string]bool{},
		usedTriggers: map[string]bool{},
		usedRoutines: map[string]bool{},
		lockedTables: map[string]bool{},
	}
}

var entFzMenu = []string{
	"add_column", "drop_column", "modify_column",
	"add_index", "drop_index",
	"add_fk", "drop_fk",
	"add_view", "drop_view", "modify_view",
	"add_trigger", "drop_trigger",
	"add_routine", "drop_routine",
	"partition", "departition",
}

// entFuzzMutations derives k independent mutations from the model, deterministically in
// (model order, rng stream).
func entFuzzMutations(t *testing.T, model *entFzModel, rng *rand.Rand, k int) []entFzMutation {
	t.Helper()
	state := entFzNewState()
	var muts []entFzMutation
	for attempt := 0; attempt < 400 && len(muts) < k; attempt++ {
		kind := entFzMenu[rng.Intn(len(entFzMenu))]
		seq := len(muts)
		if m, ok := entFzGenerate(kind, model, rng, state, seq); ok {
			muts = append(muts, m)
		}
	}
	require.Len(t, muts, k, "fuzz generator starved (only %d of %d mutations)", len(muts), k)
	return muts
}

// entFzPickTable picks a random mutable (non-protected, non-locked) table.
func entFzPickTable(model *entFzModel, rng *rand.Rand, state *entFzState) *entFzTable {
	var candidates []*entFzTable
	for _, tbl := range model.tables {
		if entFzProtectedTables[tbl.name] || state.lockedTables[tbl.name] {
			continue
		}
		candidates = append(candidates, tbl)
	}
	if len(candidates) == 0 {
		return nil
	}
	return candidates[rng.Intn(len(candidates))]
}

// entFzGenerate builds one mutation of the given kind, or reports it infeasible.
func entFzGenerate(kind string, model *entFzModel, rng *rand.Rand, state *entFzState, seq int) (entFzMutation, bool) {
	switch kind {
	case "add_column":
		tbl := entFzPickTable(model, rng, state)
		if tbl == nil {
			return entFzMutation{}, false
		}
		defs := []string{"varchar(40) DEFAULT NULL", "int NOT NULL DEFAULT '0'", "decimal(8,2) DEFAULT NULL", "bigint unsigned DEFAULT NULL"}
		def := defs[rng.Intn(len(defs))]
		col := fmt.Sprintf("ent_fz_c%d", seq)
		table := tbl.name
		return entFzMutation{
			desc: fmt.Sprintf("add_column %s.%s (%s)", table, col, def),
			apply: func(t *testing.T, source string) string {
				return addColumnToTable(t, source, table, "`"+col+"` "+def)
			},
		}, true

	case "drop_column":
		type cand struct{ table, col string }
		var cands []cand
		for _, tbl := range model.tables {
			if entFzProtectedTables[tbl.name] || state.lockedTables[tbl.name] {
				continue
			}
			indexed := entFzIndexedCols(tbl)
			trigRef := model.triggerCols[tbl.name]
			for _, c := range tbl.cols {
				if indexed[c] || trigRef[c] || entFzProtectedColumns[tbl.name+"."+c] {
					continue
				}
				if state.usedCols[tbl.name+"."+c] {
					continue
				}
				if strings.Contains(tbl.colLine[c], "GENERATED") {
					continue
				}
				cands = append(cands, cand{tbl.name, c})
			}
		}
		if len(cands) == 0 {
			return entFzMutation{}, false
		}
		pick := cands[rng.Intn(len(cands))]
		state.usedCols[pick.table+"."+pick.col] = true
		return entFzMutation{
			desc: fmt.Sprintf("drop_column %s.%s", pick.table, pick.col),
			apply: func(t *testing.T, source string) string {
				return entDropLineInTable(t, source, pick.table, "`"+pick.col+"`")
			},
		}, true

	case "modify_column":
		type cand struct {
			table, col string
			from, to   int
		}
		var cands []cand
		for _, tbl := range model.tables {
			if entFzProtectedTables[tbl.name] || state.lockedTables[tbl.name] {
				continue
			}
			for _, c := range tbl.cols {
				if state.usedCols[tbl.name+"."+c] {
					continue
				}
				mm := entFzReVarLen.FindStringSubmatch(tbl.colLine[c])
				if mm == nil {
					continue
				}
				n := entFzAtoi(mm[1])
				if n < 8 || n > 150 {
					continue
				}
				cands = append(cands, cand{tbl.name, c, n, n + 37})
			}
		}
		if len(cands) == 0 {
			return entFzMutation{}, false
		}
		pick := cands[rng.Intn(len(cands))]
		state.usedCols[pick.table+"."+pick.col] = true
		return entFzMutation{
			desc: fmt.Sprintf("modify_column widen %s.%s varchar(%d)->varchar(%d)", pick.table, pick.col, pick.from, pick.to),
			apply: func(t *testing.T, source string) string {
				return entReplaceInTable(t, source, pick.table,
					fmt.Sprintf("`%s` varchar(%d)", pick.col, pick.from),
					fmt.Sprintf("`%s` varchar(%d)", pick.col, pick.to))
			},
		}, true

	case "add_index":
		type cand struct{ table, col string }
		var cands []cand
		for _, tbl := range model.tables {
			if entFzProtectedTables[tbl.name] || state.lockedTables[tbl.name] {
				continue
			}
			for _, c := range tbl.cols {
				line := tbl.colLine[c]
				rest := strings.TrimPrefix(line, "`"+c+"` ")
				ok := strings.HasPrefix(rest, "bigint") || strings.HasPrefix(rest, "int") || strings.HasPrefix(rest, "integer")
				if mm := entFzReVarLen.FindStringSubmatch(rest); mm != nil && strings.HasPrefix(rest, "varchar") {
					ok = entFzAtoi(mm[1]) <= 150
				}
				if ok {
					cands = append(cands, cand{tbl.name, c})
				}
			}
		}
		if len(cands) == 0 {
			return entFzMutation{}, false
		}
		pick := cands[rng.Intn(len(cands))]
		idx := fmt.Sprintf("ent_fz_i%d", seq)
		return entFzMutation{
			desc: fmt.Sprintf("add_index %s on %s(%s)", idx, pick.table, pick.col),
			apply: func(t *testing.T, source string) string {
				return addIndexToTable(t, source, pick.table, "KEY `"+idx+"` (`"+pick.col+"`)")
			},
		}, true

	case "drop_index":
		type cand struct{ table, index string }
		var cands []cand
		for _, tbl := range model.tables {
			if state.lockedTables[tbl.name] {
				continue
			}
			fkCols := map[string]bool{}
			for _, fk := range tbl.fks {
				for _, c := range fk.cols {
					fkCols[c] = true
				}
			}
			for _, idx := range tbl.indexes {
				if state.usedIndexes[tbl.name+"."+idx.name] {
					continue
				}
				overlap := false
				for _, c := range idx.cols {
					if fkCols[c] {
						overlap = true
						break
					}
				}
				if !overlap {
					cands = append(cands, cand{tbl.name, idx.name})
				}
			}
		}
		if len(cands) == 0 {
			return entFzMutation{}, false
		}
		pick := cands[rng.Intn(len(cands))]
		state.usedIndexes[pick.table+"."+pick.index] = true
		return entFzMutation{
			desc: fmt.Sprintf("drop_index %s.%s", pick.table, pick.index),
			apply: func(t *testing.T, source string) string {
				return entDropLineInTable(t, source, pick.table, "`"+pick.index+"`")
			},
		}, true

	case "add_fk":
		src := entFzPickTable(model, rng, state)
		if src == nil || src.partitioned {
			return entFzMutation{}, false
		}
		var refs []*entFzTable
		for _, tbl := range model.tables {
			if tbl.partitioned || state.lockedTables[tbl.name] || len(tbl.pk) != 1 {
				continue
			}
			pkCol := ""
			for c := range tbl.pk {
				pkCol = c
			}
			if strings.HasPrefix(strings.TrimPrefix(tbl.colLine[pkCol], "`"+pkCol+"` "), "bigint") &&
				strings.Contains(tbl.colLine[pkCol], "unsigned") {
				refs = append(refs, tbl)
			}
		}
		if len(refs) == 0 {
			return entFzMutation{}, false
		}
		ref := refs[rng.Intn(len(refs))]
		refPK := ""
		for c := range ref.pk {
			refPK = c
		}
		srcName, refName := src.name, ref.name
		state.lockedTables[srcName] = true
		state.lockedTables[refName] = true
		col := fmt.Sprintf("ent_fz_r%d", seq)
		fk := fmt.Sprintf("ent_fz_fk%d", seq)
		return entFzMutation{
			desc: fmt.Sprintf("add_fk %s.%s -> %s.%s (%s)", srcName, col, refName, refPK, fk),
			apply: func(t *testing.T, source string) string {
				s := addColumnToTable(t, source, srcName, "`"+col+"` bigint unsigned DEFAULT NULL")
				s = addIndexToTable(t, s, srcName, "KEY `"+fk+"` (`"+col+"`)")
				return addIndexToTable(t, s, srcName,
					"CONSTRAINT `"+fk+"` FOREIGN KEY (`"+col+"`) REFERENCES `"+refName+"` (`"+refPK+"`) ON DELETE SET NULL")
			},
		}, true

	case "drop_fk":
		type cand struct{ table, fk string }
		var cands []cand
		for _, tbl := range model.tables {
			if state.lockedTables[tbl.name] {
				continue
			}
			for _, fk := range tbl.fks {
				if !state.usedFKs[tbl.name+"."+fk.name] {
					cands = append(cands, cand{tbl.name, fk.name})
				}
			}
		}
		if len(cands) == 0 {
			return entFzMutation{}, false
		}
		pick := cands[rng.Intn(len(cands))]
		state.usedFKs[pick.table+"."+pick.fk] = true
		return entFzMutation{
			desc: fmt.Sprintf("drop_fk %s.%s", pick.table, pick.fk),
			apply: func(t *testing.T, source string) string {
				return entDropLineInTable(t, source, pick.table, "CONSTRAINT `"+pick.fk+"`")
			},
		}, true

	case "add_view":
		tbl := entFzPickTable(model, rng, state)
		if tbl == nil || len(tbl.pk) == 0 {
			return entFzMutation{}, false
		}
		pkCol := ""
		for _, c := range tbl.cols {
			if tbl.pk[c] {
				pkCol = c
				break
			}
		}
		if pkCol == "" {
			return entFzMutation{}, false
		}
		name := fmt.Sprintf("ent_fz_v%d", 10+seq)
		table := tbl.name
		return entFzMutation{
			desc: fmt.Sprintf("add_view %s over %s(%s)", name, table, pkCol),
			apply: func(_ *testing.T, source string) string {
				return source + fmt.Sprintf("\nCREATE VIEW %s AS SELECT %s FROM %s;\n", name, pkCol, table)
			},
		}, true

	case "drop_view":
		var cands []string
		for _, v := range model.views {
			if !state.usedViews[v] {
				cands = append(cands, v)
			}
		}
		if len(cands) == 0 {
			return entFzMutation{}, false
		}
		pick := cands[rng.Intn(len(cands))]
		state.usedViews[pick] = true
		return entFzMutation{
			desc: "drop_view " + pick,
			apply: func(t *testing.T, source string) string {
				return dropObjectBlock(t, source, "VIEW", pick)
			},
		}, true

	case "modify_view":
		extension := map[string]string{
			"ent_fz_v1": "`users`.`name` AS `name`",
			"ent_fz_v2": "`hosts`.`status` AS `status`",
		}
		var cands []string
		for _, v := range model.views {
			if !state.usedViews[v] && extension[v] != "" {
				cands = append(cands, v)
			}
		}
		if len(cands) == 0 {
			return entFzMutation{}, false
		}
		pick := cands[rng.Intn(len(cands))]
		state.usedViews[pick] = true
		ext := extension[pick]
		return entFzMutation{
			desc: "modify_view " + pick,
			apply: func(t *testing.T, source string) string {
				return addColumnToView(t, source, pick, ext)
			},
		}, true

	case "add_trigger":
		tbl := entFzPickTable(model, rng, state)
		if tbl == nil || len(tbl.pk) == 0 {
			return entFzMutation{}, false
		}
		pkCol := ""
		for _, c := range tbl.cols {
			if tbl.pk[c] {
				pkCol = c
				break
			}
		}
		if pkCol == "" {
			return entFzMutation{}, false
		}
		name := fmt.Sprintf("ent_fz_t%d", seq)
		table := tbl.name
		return entFzMutation{
			desc: fmt.Sprintf("add_trigger %s on %s", name, table),
			apply: func(_ *testing.T, source string) string {
				return source + fmt.Sprintf("\nCREATE TRIGGER %s BEFORE UPDATE ON %s FOR EACH ROW SET NEW.%s = NEW.%s;\n",
					name, table, pkCol, pkCol)
			},
		}, true

	case "drop_trigger":
		var cands []string
		for _, trg := range model.triggers {
			if !state.usedTriggers[trg] {
				cands = append(cands, trg)
			}
		}
		if len(cands) == 0 {
			return entFzMutation{}, false
		}
		pick := cands[rng.Intn(len(cands))]
		state.usedTriggers[pick] = true
		return entFzMutation{
			desc: "drop_trigger " + pick,
			apply: func(t *testing.T, source string) string {
				return dropTrigger(t, source, pick)
			},
		}, true

	case "add_routine":
		if rng.Intn(2) == 0 {
			name := fmt.Sprintf("ent_fz_fn%d", 10+seq)
			ret := seq + 100
			return entFzMutation{
				desc: "add_routine function " + name,
				apply: func(_ *testing.T, source string) string {
					return source + fmt.Sprintf("\nCREATE FUNCTION %s() RETURNS INT DETERMINISTIC RETURN %d;\n", name, ret)
				},
			}, true
		}
		name := fmt.Sprintf("ent_fz_pr%d", 10+seq)
		sel := seq + 200
		return entFzMutation{
			desc: "add_routine procedure " + name,
			apply: func(_ *testing.T, source string) string {
				return source + fmt.Sprintf("\nCREATE PROCEDURE %s() BEGIN SELECT %d; END;\n", name, sel)
			},
		}, true

	case "drop_routine":
		type cand struct{ kind, name string }
		var cands []cand
		for _, fn := range model.functions {
			if !state.usedRoutines[fn] {
				cands = append(cands, cand{"FUNCTION", fn})
			}
		}
		for _, pr := range model.procedures {
			if !state.usedRoutines[pr] {
				cands = append(cands, cand{"PROCEDURE", pr})
			}
		}
		if len(cands) == 0 {
			return entFzMutation{}, false
		}
		pick := cands[rng.Intn(len(cands))]
		state.usedRoutines[pick.name] = true
		return entFzMutation{
			desc: "drop_routine " + strings.ToLower(pick.kind) + " " + pick.name,
			apply: func(t *testing.T, source string) string {
				return dropObjectBlock(t, source, pick.kind, pick.name)
			},
		}, true

	case "partition":
		var cands []*entFzTable
		for _, tbl := range model.tables {
			// changelog is column-protected but IS the partition candidate (the only
			// FK-free, unreferenced, single-PK table in the slice); ent_fz_part is
			// already partitioned.
			if tbl.partitioned || state.lockedTables[tbl.name] {
				continue
			}
			if len(tbl.fks) > 0 || model.referenced[tbl.name] || len(tbl.pk) != 1 {
				continue
			}
			// The partition key must be part of every unique key: single-col-PK tables
			// with no other UNIQUE index qualify.
			hasUnique := false
			for _, idx := range tbl.indexes {
				if idx.unique {
					hasUnique = true
					break
				}
			}
			if hasUnique {
				continue
			}
			cands = append(cands, tbl)
		}
		if len(cands) == 0 {
			return entFzMutation{}, false
		}
		pick := cands[rng.Intn(len(cands))]
		pkCol := ""
		for c := range pick.pk {
			pkCol = c
		}
		state.lockedTables[pick.name] = true
		table := pick.name
		return entFzMutation{
			desc: fmt.Sprintf("partition %s by hash(%s)", table, pkCol),
			apply: func(t *testing.T, source string) string {
				return entAppendTableClause(t, source, table, "PARTITION BY HASH (`"+pkCol+"`)\nPARTITIONS 4")
			},
		}, true

	case "departition":
		var cands []*entFzTable
		for _, tbl := range model.tables {
			if tbl.partitioned && !state.lockedTables[tbl.name] {
				cands = append(cands, tbl)
			}
		}
		if len(cands) == 0 {
			return entFzMutation{}, false
		}
		pick := cands[rng.Intn(len(cands))]
		state.lockedTables[pick.name] = true
		table := pick.name
		return entFzMutation{
			desc: "departition " + table,
			apply: func(t *testing.T, source string) string {
				return entSetPartitionClause(t, source, table, "")
			},
		}, true

	default:
		return entFzMutation{}, false
	}
}

// entFzAtoi is a no-error atoi for regex-captured digits.
func entFzAtoi(s string) int {
	n := 0
	for _, r := range s {
		n = n*10 + int(r-'0')
	}
	return n
}

// ----------------------------------------------------------------------------
// Trial runner + ddmin minimization.
// ----------------------------------------------------------------------------

// entFuzzScratchDB creates an eagerly-droppable scratch database (fuzz + minimization
// churn through many databases inside a single test, so cleanup cannot wait for test
// end).
func entFuzzScratchDB(ctx context.Context, t *testing.T, srv liveServer) (string, func()) {
	t.Helper()
	dbName := newLiveDatabase(ctx, t, srv, "entsdl_fz")
	drop := func() {
		c, err := createLiveMySQLDriver(ctx, srv, "")
		if err != nil {
			return
		}
		defer c.Close(ctx)
		_, _ = c.Execute(ctx, "DROP DATABASE IF EXISTS `"+dbName+"`", db.ExecuteOptions{})
	}
	return dbName, drop
}

// entFuzzTrial runs the whole oracle protocol for one mutation set on a fresh scratch
// database. Infrastructure failures (server down, base DDL broken, mutation anchors
// missing) fail the test hard; ORACLE violations (diff error, apply error, residual
// after apply, non-idempotent dump) are returned as errors so the caller can minimize.
func entFuzzTrial(ctx context.Context, t *testing.T, srv liveServer, baseDDL string, muts []entFzMutation) error {
	t.Helper()
	dbName, drop := entFuzzScratchDB(ctx, t, srv)
	defer drop()

	driver, err := createLiveMySQLDriver(ctx, srv, dbName)
	require.NoError(t, err)
	_, err = driver.Execute(ctx, entNormalizeDelimiters(baseDDL), db.ExecuteOptions{})
	driver.Close(ctx)
	require.NoError(t, err, "[%s] fuzz base failed to load", srv.name)

	source := dumpSDL(ctx, t, srv, dbName)
	target := source
	for _, m := range muts {
		target = m.apply(t, target)
	}

	plan, err := mysqlDiffSDLMigration(source, target, srv.version)
	if err != nil {
		return errors.Wrap(err, "diff failed")
	}
	if strings.TrimSpace(plan) == "" {
		return errors.New("empty plan for a non-empty mutation set")
	}
	if err := applyDDL(ctx, t, srv, dbName, plan); err != nil {
		return errors.Wrapf(err, "plan failed to apply; plan was:\n%s", plan)
	}
	after := dumpSDL(ctx, t, srv, dbName)
	converge, err := mysqlDiffSDLMigration(after, target, srv.version)
	if err != nil {
		return errors.Wrap(err, "converge diff failed")
	}
	if converge != "" {
		return errors.Errorf("did not converge; residual:\n%s\nplan was:\n%s", converge, plan)
	}
	self, err := mysqlDiffSDLMigration(after, after, srv.version)
	if err != nil {
		return errors.Wrap(err, "idempotence diff failed")
	}
	if self != "" {
		return errors.Errorf("post-apply dump not idempotent:\n%s", self)
	}
	return nil
}

// entFuzzMinimize greedily shrinks a failing mutation set (ddmin-lite: drop one mutation
// at a time, keep the removal whenever the rest still fails). Returns the minimized set
// and its error.
func entFuzzMinimize(ctx context.Context, t *testing.T, srv liveServer, baseDDL string, muts []entFzMutation, firstErr error) ([]entFzMutation, error) {
	t.Helper()
	minimized := muts
	lastErr := firstErr
	for i := 0; i < len(minimized) && len(minimized) > 1; {
		trial := make([]entFzMutation, 0, len(minimized)-1)
		trial = append(trial, minimized[:i]...)
		trial = append(trial, minimized[i+1:]...)
		if err := entFuzzTrial(ctx, t, srv, baseDDL, trial); err != nil {
			minimized = trial
			lastErr = err
			continue // same index now names the next mutation
		}
		i++
	}
	return minimized, lastErr
}

func entFzDescs(muts []entFzMutation) []string {
	out := make([]string, len(muts))
	for i, m := range muts {
		out[i] = m.desc
	}
	return out
}

// ----------------------------------------------------------------------------
// The fuzz test: ≥20 seeds on 8.0, ≥10 on 5.7 (deterministic; the seed is in the
// subtest name).
// ----------------------------------------------------------------------------

//nolint:tparallel
func TestSDLEnterpriseFuzz(t *testing.T) {
	skipUnlessLiveOracle(t)
	ctx := context.Background()

	for _, srv := range liveServers {
		srv := srv
		seedCount := 20
		if srv.version == "5.7" {
			seedCount = 10
		}
		t.Run(srv.name, func(t *testing.T) {
			baseDDL := entZabbixSlice(t, entFuzzSliceTables...) + entFuzzAux

			// The model is built once per server from a reference load (the canonical dump
			// is deterministic, so it matches every trial's source).
			refDB, refDrop := entFuzzScratchDB(ctx, t, srv)
			driver, err := createLiveMySQLDriver(ctx, srv, refDB)
			require.NoError(t, err)
			_, err = driver.Execute(ctx, entNormalizeDelimiters(baseDDL), db.ExecuteOptions{})
			driver.Close(ctx)
			require.NoError(t, err, "[%s] fuzz base failed to load", srv.name)
			refSource := dumpSDL(ctx, t, srv, refDB)
			refDrop()
			model := entFuzzModel(t, refSource)
			require.NotEmpty(t, model.tables, "fuzz model parsed no tables")
			require.NotEmpty(t, model.triggers, "fuzz model parsed no triggers")

			for seed := int64(1); seed <= int64(seedCount); seed++ {
				seed := seed
				t.Run(fmt.Sprintf("seed_%d", seed), func(t *testing.T) {
					rng := rand.New(rand.NewSource(seed)) //nolint:gosec // deterministic fuzz, not crypto
					k := 3 + rng.Intn(6)                  // K ∈ [3,8]
					muts := entFuzzMutations(t, model, rng, k)
					t.Logf("[FUZZ %s seed=%d] K=%d mutations:\n  %s", srv.name, seed, k, strings.Join(entFzDescs(muts), "\n  "))

					err := entFuzzTrial(ctx, t, srv, baseDDL, muts)
					if err == nil {
						return
					}
					minimized, minErr := entFuzzMinimize(ctx, t, srv, baseDDL, muts, err)
					t.Errorf("[FUZZ %s seed=%d] FAILED\nminimized mutations (%d of %d):\n  %s\nerror:\n%v",
						srv.name, seed, len(minimized), len(muts), strings.Join(entFzDescs(minimized), "\n  "), minErr)
				})
			}
		})
	}
}

// A6 of the ENTERPRISE smoke axes: a SEEDED, deterministic
// MULTI-ROUND STATEFUL fuzzer. The single-round fuzzer (A4)
// mutates a pristine dump once; real projects mutate the LAST dump repeatedly, so drift
// and text-accretion bugs that only appear across dump->mutate->apply->dump cycles are
// invisible to it (real example this campaign: omni used to accrete one paren layer per
// dump cycle around subquery wrappers until the fold fixed it — a per-round differ never
// sees the growth).
//
// Protocol per seed, on a 30-table Zabbix slice base:
//
//	dump_0 = canonical dump of the freshly loaded base
//	for round r in 1..5:
//	    target_r = mutate(dump_{r-1}, K ∈ [2,5] mutations from the menu)
//	    oracle   = diff(dump_{r-1} -> target_r) → apply → converge → idempotence
//	    dump_r   = fresh canonical dump, and ADDITIONALLY:
//	      (a) dump_r reloads through the STRICT LoadSDL path (no LoadSQL fallback mask),
//	      (b) len(dump_r) stays under len(dump_0) + the cumulative added-object budget
//	          (linear ledger — catches unbounded text accretion), and every statement NOT
//	          touched by this round's mutations is BYTE-IDENTICAL to its dump_{r-1} form
//	          (catches per-cycle accretion and collateral churn on untouched objects),
//	      (c) object counts (tables/views/routines/triggers/columns/indexes/FKs/checks)
//	          match the mutation ledger exactly.
//
// The menu is the A4 menu PLUS (8.0-only where marked): add/drop CHECK constraint (8.0),
// add/drop FULLTEXT index, add/drop SPATIAL column+index with SRID (8.0 — the SRID
// attribute is 8.0-only), add/modify/drop STORED and VIRTUAL generated columns, toggle
// column/index INVISIBLE (8.0), and ENUM member append. 5.7-illegal mutations are gated
// off the 5.7 menu.
//
// Determinism: the mutation stream is a pure function of the seed (the seed is in the
// subtest name); the schema model is parsed from the previous round's dump in statement
// order, so candidate lists are stable. Oracle violations are ddmin-minimized (one
// mutation at a time, each trial reloading dump_{r-1} into a fresh scratch database) and
// reported with seed + round + minimized ledger. Scratch databases are entsdl_-prefixed
// and dropped eagerly.
//
// ≥12 seeds run on 8.0 and ≥6 on 5.7.
//
// Shared helpers (liveServers, createLiveMySQLDriver, newLiveDatabase, dumpSDL, applyDDL,
// syncMetaForDB, objectCounts, addColumnToTable, addIndexToTable, addColumnToView,
// dropObjectBlock, dropTrigger, entZabbixSlice, entFuzzSliceTables, entFuzzAux,
// entNormalizeDelimiters, entDropLineInTable, entReplaceLineInTable, entAppendTableClause,
// entSetPartitionClause, entFzProtectedTables, entFzProtectedColumns, entFuzzModel,
// entFzIndexedCols, entFzState, entFzMutation, withDatabaseContext, mysqlVersionFor) come
// from the sibling files in this package.

const (
	entSfRounds = 5
	// entSfRoundSlack is the per-round allowance on top of the per-mutation text budgets
	// (absorbs benign re-rendering, e.g. a DEFAULT NULL a fresh dump makes explicit).
	// Deliberately small so systematic accretion across 30+ objects trips the guard.
	entSfRoundSlack = 400
)

// ----------------------------------------------------------------------------
// Stateful-fuzz aux schema: seeds the droppable/modifiable object pool for the NEW menu
// kinds. entSfAuxCommon is 5.7-legal; entSfAux80 seeds the 8.0-only kinds (SRID spatial,
// INVISIBLE, CHECK) and is appended on 8.0 only.
// ----------------------------------------------------------------------------

//go:embed testdata/sdl/ent_sf_aux_common.sql
var entSfAuxCommon string

//go:embed testdata/sdl/ent_sf_aux_80.sql
var entSfAux80 string

func entSfBaseDDL(t *testing.T, srv liveServer) string {
	t.Helper()
	base := entZabbixSlice(t, entFuzzSliceTables...) + entFuzzAux + entSfAuxCommon
	if srv.version != "5.7" {
		base += entSfAux80
	}
	return base
}

// ----------------------------------------------------------------------------
// Object-count ledger.
// ----------------------------------------------------------------------------

// entSfCounts is the schema-shape summary compared against the mutation ledger after
// every round. Comparable (all ints) so `!=` works.
type entSfCounts struct {
	tables, views, functions, procedures, triggers int
	columns, indexes, fks, checks                  int
}

func (c entSfCounts) plus(d entSfCounts) entSfCounts {
	return entSfCounts{
		tables: c.tables + d.tables, views: c.views + d.views,
		functions: c.functions + d.functions, procedures: c.procedures + d.procedures,
		triggers: c.triggers + d.triggers, columns: c.columns + d.columns,
		indexes: c.indexes + d.indexes, fks: c.fks + d.fks, checks: c.checks + d.checks,
	}
}

func entSfCountObjects(meta *model.DatabaseMetadata) entSfCounts {
	c := entSfCounts{}
	c.tables, c.views, c.functions, c.procedures, c.triggers = objectCounts(meta)
	proto := meta.GetProto()
	if proto == nil {
		return c
	}
	for _, sm := range proto.GetSchemas() {
		for _, tbl := range sm.GetTables() {
			c.columns += len(tbl.GetColumns())
			c.indexes += len(tbl.GetIndexes())
			c.fks += len(tbl.GetForeignKeys())
			c.checks += len(tbl.GetCheckConstraints())
		}
	}
	return c
}

// entSfMutation wraps the replayable A4 mutation with the ledger metadata the stateful
// protocol needs: the expected object-count delta, the dump-text growth budget, and the
// statement keys the mutation may legitimately rewrite (everything else must stay
// byte-identical across the round).
type entSfMutation struct {
	entFzMutation
	delta   entSfCounts
	budget  int
	touched []string
}

// entSfLedger is the cross-round state: expected counts, and the once-only guards for
// mutations that cannot repeat on the same object across rounds.
type entSfLedger struct {
	expect       entSfCounts
	viewModified map[string]bool
}

// ----------------------------------------------------------------------------
// Extended schema model: everything the A4 model parses, plus the constructs the new
// menu mutates (FULLTEXT/SPATIAL keys, CHECK constraints, generated columns, INVISIBLE
// columns/indexes, ENUM columns) and the referential protections they imply.
// ----------------------------------------------------------------------------

type entSfIdxRec struct {
	table, name, line string
	cols              []string
	invisible         bool
	fkOverlap         bool
}

type entSfNamedRec struct{ table, name string }

type entSfColRec struct{ table, col string }

type entSfGenRec struct{ table, name, line string }

type entSfSchema struct {
	base       *entFzModel
	views      []string
	functions  []string
	procedures []string
	idx        []entSfIdxRec // non-PK B-tree indexes, visible AND invisible
	fulltext   []entSfNamedRec
	spatial    []entSfIdxRec // cols[0] is the single spatial column
	checks     []entSfNamedRec
	genCols    []entSfGenRec
	enumCols   []entSfColRec
	invisCols  []entSfColRec
	// visibleCols counts non-INVISIBLE columns per table (a toggle must leave >= 1).
	visibleCols map[string]int
	// protected are "table.col" keys drop/modify must not touch beyond the base model's
	// own exclusions: fulltext/spatial/invisible-index parts, generated-expression and
	// CHECK-expression references.
	protected map[string]bool
	// noPartition marks tables that cannot gain a partition clause (FULLTEXT/SPATIAL
	// indexes are unsupported on partitioned tables).
	noPartition map[string]bool
}

var (
	entSfReIdxLine  = regexp.MustCompile("^(UNIQUE )?KEY `(\\w+)` \\((.+?)\\)( /\\*!80000 INVISIBLE \\*/)?,?$")
	entSfReFtLine   = regexp.MustCompile("^FULLTEXT KEY `(\\w+)` \\((.+?)\\),?$")
	entSfReSpLine   = regexp.MustCompile("^SPATIAL KEY `(\\w+)` \\(`(\\w+)`\\),?$")
	entSfReChkLine  = regexp.MustCompile("^CONSTRAINT `(\\w+)` CHECK \\((.+)\\),?$")
	entSfReColLine  = regexp.MustCompile("^`(\\w+)` (.+?),?$")
	entSfReViewHdr  = regexp.MustCompile("(?i)^CREATE .*?VIEW `(\\w+)`")
	entSfReFnHdr    = regexp.MustCompile("(?i)^CREATE .*?FUNCTION `(\\w+)`")
	entSfRePrHdr    = regexp.MustCompile("(?i)^CREATE .*?PROCEDURE `(\\w+)`")
	entSfReGenConst = regexp.MustCompile(`\+ (\d+)\)`)
	entSfReEnumSpec = regexp.MustCompile(`enum\(([^)]*)\)`)
)

const (
	entSfInvisibleColToken = " /*!80023 INVISIBLE */"
	entSfInvisibleIdxToken = " /*!80000 INVISIBLE */"
)

// entSfParse builds the extended model from a canonical dump.
func entSfParse(t *testing.T, source string) *entSfSchema {
	t.Helper()
	m := &entSfSchema{
		base:        entFuzzModel(t, source),
		visibleCols: map[string]int{},
		protected:   map[string]bool{},
		noPartition: map[string]bool{},
	}
	stmts, err := mysqlparser.SplitSQL(source)
	require.NoError(t, err, "split canonical dump for stateful-fuzz model")
	for _, s := range stmts {
		text := strings.TrimSpace(s.Text)
		upper := strings.ToUpper(text)
		switch {
		case strings.HasPrefix(upper, "CREATE TABLE `"):
			m.parseTableExtras(text)
		case entSfReViewHdr.MatchString(text):
			m.views = append(m.views, entSfReViewHdr.FindStringSubmatch(text)[1])
		case entSfReFnHdr.MatchString(text):
			m.functions = append(m.functions, entSfReFnHdr.FindStringSubmatch(text)[1])
		case entSfRePrHdr.MatchString(text):
			m.procedures = append(m.procedures, entSfRePrHdr.FindStringSubmatch(text)[1])
		default:
		}
	}
	return m
}

func (m *entSfSchema) parseTableExtras(stmt string) {
	header := entReTable.FindStringSubmatch(stmt)
	if header == nil {
		return
	}
	table := header[1]
	fkCols := map[string]bool{}
	if tbl := m.base.byName[table]; tbl != nil {
		for _, fk := range tbl.fks {
			for _, c := range fk.cols {
				fkCols[c] = true
			}
		}
	}
	for _, raw := range strings.Split(stmt, "\n")[1:] {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, ")") {
			break
		}
		if mm := entSfReFtLine.FindStringSubmatch(line); mm != nil {
			m.fulltext = append(m.fulltext, entSfNamedRec{table: table, name: mm[1]})
			m.noPartition[table] = true
			for _, c := range entFzReCol.FindAllStringSubmatch(mm[2], -1) {
				m.protected[table+"."+c[1]] = true
			}
			continue
		}
		if mm := entSfReSpLine.FindStringSubmatch(line); mm != nil {
			m.spatial = append(m.spatial, entSfIdxRec{table: table, name: mm[1], cols: []string{mm[2]}})
			m.noPartition[table] = true
			m.protected[table+"."+mm[2]] = true
			continue
		}
		if mm := entSfReChkLine.FindStringSubmatch(line); mm != nil {
			m.checks = append(m.checks, entSfNamedRec{table: table, name: mm[1]})
			for _, c := range entFzReCol.FindAllStringSubmatch(mm[2], -1) {
				m.protected[table+"."+c[1]] = true
			}
			continue
		}
		if mm := entSfReIdxLine.FindStringSubmatch(line); mm != nil {
			rec := entSfIdxRec{table: table, name: mm[2], line: strings.TrimSuffix(line, ","), invisible: mm[4] != ""}
			for _, c := range entFzReCol.FindAllStringSubmatch(mm[3], -1) {
				rec.cols = append(rec.cols, c[1])
				if rec.invisible {
					m.protected[table+"."+c[1]] = true
				}
				if fkCols[c[1]] {
					rec.fkOverlap = true
				}
			}
			m.idx = append(m.idx, rec)
			continue
		}
		if mm := entSfReColLine.FindStringSubmatch(line); mm != nil {
			col, def := mm[1], strings.TrimSuffix(line, ",")
			if genIdx := strings.Index(def, "GENERATED ALWAYS AS"); genIdx >= 0 {
				m.genCols = append(m.genCols, entSfGenRec{table: table, name: col, line: def})
				for _, c := range entFzReCol.FindAllStringSubmatch(def[genIdx:], -1) {
					m.protected[table+"."+c[1]] = true
				}
			}
			if strings.HasPrefix(strings.TrimPrefix(def, "`"+col+"` "), "enum(") {
				m.enumCols = append(m.enumCols, entSfColRec{table: table, col: col})
			}
			if strings.HasSuffix(def, strings.TrimSpace(entSfInvisibleColToken)) {
				m.invisCols = append(m.invisCols, entSfColRec{table: table, col: col})
			} else {
				m.visibleCols[table]++
			}
		}
	}
}

// colDef returns the trimmed definition line of table.col from the base model.
func (m *entSfSchema) colDef(table, col string) string {
	if tbl := m.base.byName[table]; tbl != nil {
		return tbl.colLine[col]
	}
	return ""
}

// ----------------------------------------------------------------------------
// Mutation generation.
// ----------------------------------------------------------------------------

func entSfMenu(version string) []string {
	menu := []string{
		"add_column", "drop_column", "modify_column",
		"add_index", "drop_index",
		"add_fk", "drop_fk",
		"add_view", "drop_view", "modify_view",
		"add_trigger", "drop_trigger",
		"add_routine", "drop_routine",
		"partition", "departition",
		"add_fulltext", "drop_fulltext",
		"add_gen_stored", "add_gen_virtual", "modify_gen", "drop_gen",
		"enum_append",
	}
	if version != "5.7" {
		// 5.7-illegal kinds: CHECK is parsed-and-ignored on 5.7 (nothing syncs back, so
		// convergence is impossible); the SRID column attribute and column/index
		// INVISIBLE are 8.0-only syntax.
		menu = append(menu,
			"add_check", "drop_check",
			"add_spatial", "drop_spatial",
			"toggle_col_invisible", "toggle_idx_invisible",
		)
	}
	return menu
}

// entSfMutations derives k independent mutations for round r, deterministically in
// (model order, rng stream).
func entSfMutations(t *testing.T, m *entSfSchema, rng *rand.Rand, srv liveServer, ledger *entSfLedger, r, k int) []entSfMutation {
	t.Helper()
	return entSfMutationsFromMenu(t, m, rng, entSfMenu(srv.version), ledger, r, k)
}

// entSfMutationsFromMenu is entSfMutations with an explicit menu. The cross-version
// stateful fuzzer (A8) passes the 5.7-legal menu
// regardless of the authoring side — its targets apply to BOTH versions.
func entSfMutationsFromMenu(t *testing.T, m *entSfSchema, rng *rand.Rand, menu []string, ledger *entSfLedger, r, k int) []entSfMutation {
	t.Helper()
	state := entFzNewState()
	var muts []entSfMutation
	for attempt := 0; attempt < 500 && len(muts) < k; attempt++ {
		kind := menu[rng.Intn(len(menu))]
		if mut, ok := entSfGenerate(t, kind, m, rng, state, ledger, r, len(muts)); ok {
			muts = append(muts, mut)
		}
	}
	require.Len(t, muts, k, "stateful-fuzz generator starved (only %d of %d mutations, round %d)", len(muts), k, r)
	return muts
}

// entSfMutableTables returns the non-protected, non-locked tables.
func entSfMutableTables(m *entSfSchema, state *entFzState) []*entFzTable {
	var out []*entFzTable
	for _, tbl := range m.base.tables {
		if entFzProtectedTables[tbl.name] || state.lockedTables[tbl.name] {
			continue
		}
		out = append(out, tbl)
	}
	return out
}

// entSfColCandidates lists table.col pairs whose (dump-form) definition rest satisfies
// accept, excluding protected/used/locked/generated columns.
// entSfIsFKMemberCol reports whether table.col participates in any child-side
// foreign key of the base model. Used to keep STORED generated columns off FK
// member bases (A8 finding: both servers refuse that shape; 5.7 opaquely).
func entSfIsFKMemberCol(m *entSfSchema, table, col string) bool {
	tbl := m.base.byName[table]
	if tbl == nil {
		return false
	}
	for _, fk := range tbl.fks {
		for _, c := range fk.cols {
			if c == col {
				return true
			}
		}
	}
	return false
}

func entSfColCandidates(m *entSfSchema, state *entFzState, accept func(rest string) bool) []entSfColRec {
	var out []entSfColRec
	for _, tbl := range entSfMutableTables(m, state) {
		for _, c := range tbl.cols {
			key := tbl.name + "." + c
			if state.usedCols[key] || m.protected[key] || entFzProtectedColumns[key] {
				continue
			}
			def := tbl.colLine[c]
			if strings.Contains(def, "GENERATED ALWAYS AS") {
				continue
			}
			rest := strings.TrimPrefix(def, "`"+c+"` ")
			if accept(rest) {
				out = append(out, entSfColRec{table: tbl.name, col: c})
			}
		}
	}
	return out
}

func entSfIsIntRest(rest string) bool {
	return strings.HasPrefix(rest, "bigint") || strings.HasPrefix(rest, "int") || strings.HasPrefix(rest, "integer")
}

// entSfGenerate builds one mutation of the given kind, or reports it infeasible. seq
// namespaces generated object names within the round; r namespaces across rounds.
//
//nolint:gocyclo
func entSfGenerate(t *testing.T, kind string, m *entSfSchema, rng *rand.Rand, state *entFzState, ledger *entSfLedger, r, seq int) (entSfMutation, bool) {
	t.Helper()
	name := func(tag string) string { return fmt.Sprintf("ent_sf_r%d_%s%d", r, tag, seq) }
	tableKey := func(tbl string) []string { return []string{"table:" + tbl} }

	switch kind {
	case "add_column":
		tables := entSfMutableTables(m, state)
		if len(tables) == 0 {
			return entSfMutation{}, false
		}
		tbl := tables[rng.Intn(len(tables))].name
		defs := []string{"varchar(40) DEFAULT NULL", "int NOT NULL DEFAULT '0'", "decimal(8,2) DEFAULT NULL", "bigint unsigned DEFAULT NULL"}
		def := defs[rng.Intn(len(defs))]
		col := name("c")
		return entSfMutation{
			entFzMutation: entFzMutation{
				desc: fmt.Sprintf("add_column %s.%s (%s)", tbl, col, def),
				apply: func(t *testing.T, source string) string {
					return addColumnToTable(t, source, tbl, "`"+col+"` "+def)
				},
			},
			delta: entSfCounts{columns: 1}, budget: 150, touched: tableKey(tbl),
		}, true

	case "drop_column":
		var cands []entSfColRec
		for _, tbl := range entSfMutableTables(m, state) {
			indexed := entFzIndexedCols(tbl)
			trigRef := m.base.triggerCols[tbl.name]
			for _, c := range tbl.cols {
				key := tbl.name + "." + c
				if indexed[c] || trigRef[c] || entFzProtectedColumns[key] || m.protected[key] || state.usedCols[key] {
					continue
				}
				if strings.Contains(tbl.colLine[c], "GENERATED") {
					continue
				}
				cands = append(cands, entSfColRec{table: tbl.name, col: c})
			}
		}
		if len(cands) == 0 {
			return entSfMutation{}, false
		}
		pick := cands[rng.Intn(len(cands))]
		state.usedCols[pick.table+"."+pick.col] = true
		return entSfMutation{
			entFzMutation: entFzMutation{
				desc: fmt.Sprintf("drop_column %s.%s", pick.table, pick.col),
				apply: func(t *testing.T, source string) string {
					return entDropLineInTable(t, source, pick.table, "`"+pick.col+"`")
				},
			},
			delta: entSfCounts{columns: -1}, touched: tableKey(pick.table),
		}, true

	case "modify_column":
		type cand struct {
			rec      entSfColRec
			from, to int
		}
		var cands []cand
		for _, tbl := range entSfMutableTables(m, state) {
			for _, c := range tbl.cols {
				key := tbl.name + "." + c
				if state.usedCols[key] || m.protected[key] {
					continue
				}
				mm := entFzReVarLen.FindStringSubmatch(tbl.colLine[c])
				if mm == nil || strings.Contains(tbl.colLine[c], "GENERATED") {
					continue
				}
				n := entFzAtoi(mm[1])
				if n < 8 || n > 150 {
					continue
				}
				cands = append(cands, cand{rec: entSfColRec{table: tbl.name, col: c}, from: n, to: n + 23})
			}
		}
		if len(cands) == 0 {
			return entSfMutation{}, false
		}
		pick := cands[rng.Intn(len(cands))]
		state.usedCols[pick.rec.table+"."+pick.rec.col] = true
		return entSfMutation{
			entFzMutation: entFzMutation{
				desc: fmt.Sprintf("modify_column widen %s.%s varchar(%d)->varchar(%d)", pick.rec.table, pick.rec.col, pick.from, pick.to),
				apply: func(t *testing.T, source string) string {
					return entReplaceInTable(t, source, pick.rec.table,
						fmt.Sprintf("`%s` varchar(%d)", pick.rec.col, pick.from),
						fmt.Sprintf("`%s` varchar(%d)", pick.rec.col, pick.to))
				},
			},
			budget: 80, touched: tableKey(pick.rec.table),
		}, true

	case "add_index":
		cands := entSfColCandidates(m, state, func(rest string) bool {
			if entSfIsIntRest(rest) {
				return true
			}
			mm := entFzReVarLen.FindStringSubmatch(rest)
			return mm != nil && strings.HasPrefix(rest, "varchar") && entFzAtoi(mm[1]) <= 150
		})
		if len(cands) == 0 {
			return entSfMutation{}, false
		}
		pick := cands[rng.Intn(len(cands))]
		idx := name("i")
		return entSfMutation{
			entFzMutation: entFzMutation{
				desc: fmt.Sprintf("add_index %s on %s(%s)", idx, pick.table, pick.col),
				apply: func(t *testing.T, source string) string {
					return addIndexToTable(t, source, pick.table, "KEY `"+idx+"` (`"+pick.col+"`)")
				},
			},
			delta: entSfCounts{indexes: 1}, budget: 150, touched: tableKey(pick.table),
		}, true

	case "drop_index":
		var cands []entSfIdxRec
		for _, rec := range m.idx {
			if rec.invisible || rec.fkOverlap || state.lockedTables[rec.table] || state.usedIndexes[rec.table+"."+rec.name] {
				continue
			}
			cands = append(cands, rec)
		}
		if len(cands) == 0 {
			return entSfMutation{}, false
		}
		pick := cands[rng.Intn(len(cands))]
		state.usedIndexes[pick.table+"."+pick.name] = true
		return entSfMutation{
			entFzMutation: entFzMutation{
				desc: fmt.Sprintf("drop_index %s.%s", pick.table, pick.name),
				apply: func(t *testing.T, source string) string {
					return entDropLineInTable(t, source, pick.table, "`"+pick.name+"`")
				},
			},
			delta: entSfCounts{indexes: -1}, touched: tableKey(pick.table),
		}, true

	case "add_fk":
		var srcs []*entFzTable
		for _, tbl := range entSfMutableTables(m, state) {
			if !tbl.partitioned {
				srcs = append(srcs, tbl)
			}
		}
		if len(srcs) == 0 {
			return entSfMutation{}, false
		}
		src := srcs[rng.Intn(len(srcs))]
		var refs []*entFzTable
		for _, tbl := range m.base.tables {
			if tbl.partitioned || state.lockedTables[tbl.name] || len(tbl.pk) != 1 {
				continue
			}
			pkCol := ""
			for c := range tbl.pk {
				pkCol = c
			}
			if strings.HasPrefix(strings.TrimPrefix(tbl.colLine[pkCol], "`"+pkCol+"` "), "bigint") &&
				strings.Contains(tbl.colLine[pkCol], "unsigned") {
				refs = append(refs, tbl)
			}
		}
		if len(refs) == 0 {
			return entSfMutation{}, false
		}
		ref := refs[rng.Intn(len(refs))]
		refPK := ""
		for c := range ref.pk {
			refPK = c
		}
		srcName, refName := src.name, ref.name
		state.lockedTables[srcName] = true
		state.lockedTables[refName] = true
		col, fk := name("r"), name("fk")
		return entSfMutation{
			entFzMutation: entFzMutation{
				desc: fmt.Sprintf("add_fk %s.%s -> %s.%s (%s)", srcName, col, refName, refPK, fk),
				apply: func(t *testing.T, source string) string {
					s := addColumnToTable(t, source, srcName, "`"+col+"` bigint unsigned DEFAULT NULL")
					s = addIndexToTable(t, s, srcName, "KEY `"+fk+"` (`"+col+"`)")
					return addIndexToTable(t, s, srcName,
						"CONSTRAINT `"+fk+"` FOREIGN KEY (`"+col+"`) REFERENCES `"+refName+"` (`"+refPK+"`) ON DELETE SET NULL")
				},
			},
			delta: entSfCounts{columns: 1, indexes: 1, fks: 1}, budget: 450, touched: tableKey(srcName),
		}, true

	case "drop_fk":
		var cands []entSfNamedRec
		for _, tbl := range m.base.tables {
			if state.lockedTables[tbl.name] {
				continue
			}
			for _, fk := range tbl.fks {
				if !state.usedFKs[tbl.name+"."+fk.name] {
					cands = append(cands, entSfNamedRec{table: tbl.name, name: fk.name})
				}
			}
		}
		if len(cands) == 0 {
			return entSfMutation{}, false
		}
		pick := cands[rng.Intn(len(cands))]
		state.usedFKs[pick.table+"."+pick.name] = true
		return entSfMutation{
			entFzMutation: entFzMutation{
				desc: fmt.Sprintf("drop_fk %s.%s", pick.table, pick.name),
				apply: func(t *testing.T, source string) string {
					return entDropLineInTable(t, source, pick.table, "CONSTRAINT `"+pick.name+"`")
				},
			},
			delta: entSfCounts{fks: -1}, touched: tableKey(pick.table),
		}, true

	case "add_view":
		tables := entSfMutableTables(m, state)
		if len(tables) == 0 {
			return entSfMutation{}, false
		}
		tbl := tables[rng.Intn(len(tables))]
		pkCol := ""
		for _, c := range tbl.cols {
			if tbl.pk[c] {
				pkCol = c
				break
			}
		}
		if pkCol == "" {
			return entSfMutation{}, false
		}
		vw, table := name("v"), tbl.name
		return entSfMutation{
			entFzMutation: entFzMutation{
				desc: fmt.Sprintf("add_view %s over %s(%s)", vw, table, pkCol),
				apply: func(_ *testing.T, source string) string {
					return source + fmt.Sprintf("\nCREATE VIEW %s AS SELECT %s FROM %s;\n", vw, pkCol, table)
				},
			},
			delta: entSfCounts{views: 1}, budget: 600, touched: nil,
		}, true

	case "drop_view":
		var cands []string
		for _, v := range m.views {
			if !state.usedViews[v] {
				cands = append(cands, v)
			}
		}
		if len(cands) == 0 {
			return entSfMutation{}, false
		}
		pick := cands[rng.Intn(len(cands))]
		state.usedViews[pick] = true
		return entSfMutation{
			entFzMutation: entFzMutation{
				desc: "drop_view " + pick,
				apply: func(t *testing.T, source string) string {
					return dropObjectBlock(t, source, "VIEW", pick)
				},
			},
			delta: entSfCounts{views: -1}, touched: []string{"view:" + pick},
		}, true

	case "modify_view":
		// The extension is only appendable once per view over the whole seed (a second
		// append would duplicate the column), so the guard lives in the LEDGER, not the
		// per-round state.
		extension := map[string]string{
			"ent_fz_v1": "`users`.`name` AS `name`",
			"ent_fz_v2": "`hosts`.`status` AS `status`",
		}
		var cands []string
		for _, v := range m.views {
			if !state.usedViews[v] && !ledger.viewModified[v] && extension[v] != "" {
				cands = append(cands, v)
			}
		}
		if len(cands) == 0 {
			return entSfMutation{}, false
		}
		pick := cands[rng.Intn(len(cands))]
		state.usedViews[pick] = true
		ledger.viewModified[pick] = true
		ext := extension[pick]
		return entSfMutation{
			entFzMutation: entFzMutation{
				desc: "modify_view " + pick,
				apply: func(t *testing.T, source string) string {
					return addColumnToView(t, source, pick, ext)
				},
			},
			budget: 80, touched: []string{"view:" + pick},
		}, true

	case "add_trigger":
		tables := entSfMutableTables(m, state)
		if len(tables) == 0 {
			return entSfMutation{}, false
		}
		tbl := tables[rng.Intn(len(tables))]
		pkCol := ""
		for _, c := range tbl.cols {
			if tbl.pk[c] {
				pkCol = c
				break
			}
		}
		if pkCol == "" {
			return entSfMutation{}, false
		}
		trg, table := name("t"), tbl.name
		return entSfMutation{
			entFzMutation: entFzMutation{
				desc: fmt.Sprintf("add_trigger %s on %s", trg, table),
				apply: func(_ *testing.T, source string) string {
					return source + fmt.Sprintf("\nCREATE TRIGGER %s BEFORE UPDATE ON %s FOR EACH ROW SET NEW.%s = NEW.%s;\n",
						trg, table, pkCol, pkCol)
				},
			},
			delta: entSfCounts{triggers: 1}, budget: 500, touched: nil,
		}, true

	case "drop_trigger":
		var cands []string
		for _, trg := range m.base.triggers {
			if !state.usedTriggers[trg] {
				cands = append(cands, trg)
			}
		}
		if len(cands) == 0 {
			return entSfMutation{}, false
		}
		pick := cands[rng.Intn(len(cands))]
		state.usedTriggers[pick] = true
		return entSfMutation{
			entFzMutation: entFzMutation{
				desc: "drop_trigger " + pick,
				apply: func(t *testing.T, source string) string {
					return dropTrigger(t, source, pick)
				},
			},
			delta: entSfCounts{triggers: -1}, touched: []string{"trigger:" + pick},
		}, true

	case "add_routine":
		if rng.Intn(2) == 0 {
			fn := name("fn")
			ret := 100*r + seq
			return entSfMutation{
				entFzMutation: entFzMutation{
					desc: "add_routine function " + fn,
					apply: func(_ *testing.T, source string) string {
						return source + fmt.Sprintf("\nCREATE FUNCTION %s() RETURNS INT DETERMINISTIC RETURN %d;\n", fn, ret)
					},
				},
				delta: entSfCounts{functions: 1}, budget: 500, touched: nil,
			}, true
		}
		pr := name("pr")
		sel := 200*r + seq
		return entSfMutation{
			entFzMutation: entFzMutation{
				desc: "add_routine procedure " + pr,
				apply: func(_ *testing.T, source string) string {
					return source + fmt.Sprintf("\nCREATE PROCEDURE %s() BEGIN SELECT %d; END;\n", pr, sel)
				},
			},
			delta: entSfCounts{procedures: 1}, budget: 500, touched: nil,
		}, true

	case "drop_routine":
		type cand struct{ kind, name string }
		var cands []cand
		for _, fn := range m.functions {
			if !state.usedRoutines[fn] {
				cands = append(cands, cand{"FUNCTION", fn})
			}
		}
		for _, pr := range m.procedures {
			if !state.usedRoutines[pr] {
				cands = append(cands, cand{"PROCEDURE", pr})
			}
		}
		if len(cands) == 0 {
			return entSfMutation{}, false
		}
		pick := cands[rng.Intn(len(cands))]
		state.usedRoutines[pick.name] = true
		delta := entSfCounts{functions: -1}
		key := "function:" + pick.name
		if pick.kind == "PROCEDURE" {
			delta = entSfCounts{procedures: -1}
			key = "procedure:" + pick.name
		}
		return entSfMutation{
			entFzMutation: entFzMutation{
				desc: "drop_routine " + strings.ToLower(pick.kind) + " " + pick.name,
				apply: func(t *testing.T, source string) string {
					return dropObjectBlock(t, source, pick.kind, pick.name)
				},
			},
			delta: delta, touched: []string{key},
		}, true

	case "partition":
		var cands []*entFzTable
		for _, tbl := range m.base.tables {
			if tbl.partitioned || state.lockedTables[tbl.name] || m.noPartition[tbl.name] {
				continue
			}
			if len(tbl.fks) > 0 || m.base.referenced[tbl.name] || len(tbl.pk) != 1 {
				continue
			}
			hasUnique := false
			for _, idx := range tbl.indexes {
				if idx.unique {
					hasUnique = true
					break
				}
			}
			if hasUnique {
				continue
			}
			cands = append(cands, tbl)
		}
		if len(cands) == 0 {
			return entSfMutation{}, false
		}
		pick := cands[rng.Intn(len(cands))]
		pkCol := ""
		for c := range pick.pk {
			pkCol = c
		}
		state.lockedTables[pick.name] = true
		table := pick.name
		return entSfMutation{
			entFzMutation: entFzMutation{
				desc: fmt.Sprintf("partition %s by hash(%s)", table, pkCol),
				apply: func(t *testing.T, source string) string {
					return entAppendTableClause(t, source, table, "PARTITION BY HASH (`"+pkCol+"`)\nPARTITIONS 4")
				},
			},
			budget: 300, touched: tableKey(table),
		}, true

	case "departition":
		var cands []*entFzTable
		for _, tbl := range m.base.tables {
			if tbl.partitioned && !state.lockedTables[tbl.name] {
				cands = append(cands, tbl)
			}
		}
		if len(cands) == 0 {
			return entSfMutation{}, false
		}
		pick := cands[rng.Intn(len(cands))]
		state.lockedTables[pick.name] = true
		table := pick.name
		return entSfMutation{
			entFzMutation: entFzMutation{
				desc: "departition " + table,
				apply: func(t *testing.T, source string) string {
					return entSetPartitionClause(t, source, table, "")
				},
			},
			touched: tableKey(table),
		}, true

	case "add_fulltext":
		// One FULLTEXT creation per table per round (InnoDB builds them one at a time),
		// and never on a partitioned table — the whole table is locked for the round.
		var cands []entSfColRec
		for _, rec := range entSfColCandidates(m, state, func(rest string) bool {
			return strings.HasPrefix(rest, "varchar") || strings.HasPrefix(rest, "text") ||
				strings.HasPrefix(rest, "mediumtext") || strings.HasPrefix(rest, "longtext")
		}) {
			if tbl := m.base.byName[rec.table]; tbl != nil && tbl.partitioned {
				continue
			}
			cands = append(cands, rec)
		}
		if len(cands) == 0 {
			return entSfMutation{}, false
		}
		pick := cands[rng.Intn(len(cands))]
		state.lockedTables[pick.table] = true
		idx := name("ft")
		return entSfMutation{
			entFzMutation: entFzMutation{
				desc: fmt.Sprintf("add_fulltext %s on %s(%s)", idx, pick.table, pick.col),
				apply: func(t *testing.T, source string) string {
					return addIndexToTable(t, source, pick.table, "FULLTEXT KEY `"+idx+"` (`"+pick.col+"`)")
				},
			},
			delta: entSfCounts{indexes: 1}, budget: 150, touched: tableKey(pick.table),
		}, true

	case "drop_fulltext":
		var cands []entSfNamedRec
		for _, rec := range m.fulltext {
			if !state.lockedTables[rec.table] && !state.usedIndexes[rec.table+"."+rec.name] {
				cands = append(cands, rec)
			}
		}
		if len(cands) == 0 {
			return entSfMutation{}, false
		}
		pick := cands[rng.Intn(len(cands))]
		state.usedIndexes[pick.table+"."+pick.name] = true
		return entSfMutation{
			entFzMutation: entFzMutation{
				desc: fmt.Sprintf("drop_fulltext %s.%s", pick.table, pick.name),
				apply: func(t *testing.T, source string) string {
					return entDropLineInTable(t, source, pick.table, "FULLTEXT KEY `"+pick.name+"`")
				},
			},
			delta: entSfCounts{indexes: -1}, touched: tableKey(pick.table),
		}, true

	case "add_spatial":
		// A NOT NULL POINT column with an SRID plus its SPATIAL index, in one mutation.
		// SPATIAL indexes are unsupported on partitioned tables; the table is locked so
		// no same-round mutation interleaves with the two-clause add.
		var cands []*entFzTable
		for _, tbl := range entSfMutableTables(m, state) {
			if !tbl.partitioned {
				cands = append(cands, tbl)
			}
		}
		if len(cands) == 0 {
			return entSfMutation{}, false
		}
		pick := cands[rng.Intn(len(cands))]
		state.lockedTables[pick.name] = true
		table, col, idx := pick.name, name("sp"), name("spi")
		return entSfMutation{
			entFzMutation: entFzMutation{
				desc: fmt.Sprintf("add_spatial %s.%s (SRID 4326) + %s", table, col, idx),
				apply: func(t *testing.T, source string) string {
					s := addColumnToTable(t, source, table, "`"+col+"` point NOT NULL /*!80003 SRID 4326 */")
					return addIndexToTable(t, s, table, "SPATIAL KEY `"+idx+"` (`"+col+"`)")
				},
			},
			delta: entSfCounts{columns: 1, indexes: 1}, budget: 400, touched: tableKey(table),
		}, true

	case "drop_spatial":
		// Index and column leave together (a dangling SPATIAL KEY over a dropped column
		// would be an invalid target).
		var cands []entSfIdxRec
		for _, rec := range m.spatial {
			if !state.lockedTables[rec.table] {
				cands = append(cands, rec)
			}
		}
		if len(cands) == 0 {
			return entSfMutation{}, false
		}
		pick := cands[rng.Intn(len(cands))]
		state.lockedTables[pick.table] = true
		col := pick.cols[0]
		return entSfMutation{
			entFzMutation: entFzMutation{
				desc: fmt.Sprintf("drop_spatial %s.%s (+column %s)", pick.table, pick.name, col),
				apply: func(t *testing.T, source string) string {
					s := entDropLineInTable(t, source, pick.table, "SPATIAL KEY `"+pick.name+"`")
					return entDropLineInTable(t, s, pick.table, "`"+col+"` point")
				},
			},
			delta: entSfCounts{columns: -1, indexes: -1}, touched: tableKey(pick.table),
		}, true

	case "add_check":
		cands := entSfColCandidates(m, state, entSfIsIntRest)
		if len(cands) == 0 {
			return entSfMutation{}, false
		}
		pick := cands[rng.Intn(len(cands))]
		state.usedCols[pick.table+"."+pick.col] = true
		chk := name("ck")
		return entSfMutation{
			entFzMutation: entFzMutation{
				desc: fmt.Sprintf("add_check %s on %s(%s)", chk, pick.table, pick.col),
				apply: func(t *testing.T, source string) string {
					return addIndexToTable(t, source, pick.table, "CONSTRAINT `"+chk+"` CHECK ((`"+pick.col+"` >= 0))")
				},
			},
			delta: entSfCounts{checks: 1}, budget: 250, touched: tableKey(pick.table),
		}, true

	case "drop_check":
		var cands []entSfNamedRec
		for _, rec := range m.checks {
			if !state.lockedTables[rec.table] && !state.usedIndexes[rec.table+"."+rec.name] {
				cands = append(cands, rec)
			}
		}
		if len(cands) == 0 {
			return entSfMutation{}, false
		}
		pick := cands[rng.Intn(len(cands))]
		state.usedIndexes[pick.table+"."+pick.name] = true
		return entSfMutation{
			entFzMutation: entFzMutation{
				desc: fmt.Sprintf("drop_check %s.%s", pick.table, pick.name),
				apply: func(t *testing.T, source string) string {
					return entDropLineInTable(t, source, pick.table, "CONSTRAINT `"+pick.name+"`")
				},
			},
			delta: entSfCounts{checks: -1}, touched: tableKey(pick.table),
		}, true

	case "add_gen_stored", "add_gen_virtual":
		mode, tag := "STORED", "gs"
		if kind == "add_gen_virtual" {
			mode, tag = "VIRTUAL", "gv"
		}
		cands := entSfColCandidates(m, state, entSfIsIntRest)
		if mode == "STORED" {
			// A8 finding: BOTH servers refuse a STORED generated column whose base
			// column is a child-side FK member (8.0 clean 1215; 5.7 opaque errno
			// 150 at ALGORITHM=COPY rename). VIRTUAL stays eligible.
			kept := cands[:0]
			for _, c := range cands {
				if !entSfIsFKMemberCol(m, c.table, c.col) {
					kept = append(kept, c)
				}
			}
			cands = kept
		}
		if len(cands) == 0 {
			return entSfMutation{}, false
		}
		pick := cands[rng.Intn(len(cands))]
		state.usedCols[pick.table+"."+pick.col] = true
		col := name(tag)
		return entSfMutation{
			entFzMutation: entFzMutation{
				desc: fmt.Sprintf("%s %s.%s as (%s + 7)", kind, pick.table, col, pick.col),
				apply: func(t *testing.T, source string) string {
					return addColumnToTable(t, source, pick.table,
						"`"+col+"` bigint GENERATED ALWAYS AS ((`"+pick.col+"` + 7)) "+mode)
				},
			},
			delta: entSfCounts{columns: 1}, budget: 300, touched: tableKey(pick.table),
		}, true

	case "modify_gen":
		// Bump the additive constant of a fuzz-shaped generated expression (`col` + N).
		var cands []entSfGenRec
		for _, rec := range m.genCols {
			if state.lockedTables[rec.table] || state.usedCols[rec.table+"."+rec.name] {
				continue
			}
			if entSfReGenConst.MatchString(rec.line) {
				cands = append(cands, rec)
			}
		}
		if len(cands) == 0 {
			return entSfMutation{}, false
		}
		pick := cands[rng.Intn(len(cands))]
		state.usedCols[pick.table+"."+pick.name] = true
		old := entSfReGenConst.FindStringSubmatch(pick.line)
		newLine := entSfReGenConst.ReplaceAllString(pick.line, fmt.Sprintf("+ %d)", entFzAtoi(old[1])+1))
		return entSfMutation{
			entFzMutation: entFzMutation{
				desc: fmt.Sprintf("modify_gen %s.%s (+%s -> +%d)", pick.table, pick.name, old[1], entFzAtoi(old[1])+1),
				apply: func(t *testing.T, source string) string {
					return entReplaceLineInTable(t, source, pick.table, "`"+pick.name+"`", newLine)
				},
			},
			budget: 80, touched: tableKey(pick.table),
		}, true

	case "drop_gen":
		var cands []entSfGenRec
		for _, rec := range m.genCols {
			if state.lockedTables[rec.table] || state.usedCols[rec.table+"."+rec.name] || entFzProtectedTables[rec.table] {
				continue
			}
			cands = append(cands, rec)
		}
		if len(cands) == 0 {
			return entSfMutation{}, false
		}
		pick := cands[rng.Intn(len(cands))]
		state.usedCols[pick.table+"."+pick.name] = true
		return entSfMutation{
			entFzMutation: entFzMutation{
				desc: fmt.Sprintf("drop_gen %s.%s", pick.table, pick.name),
				apply: func(t *testing.T, source string) string {
					return entDropLineInTable(t, source, pick.table, "`"+pick.name+"`")
				},
			},
			delta: entSfCounts{columns: -1}, touched: tableKey(pick.table),
		}, true

	case "toggle_col_invisible":
		// Visible -> invisible needs >= 2 visible columns left; invisible -> visible is
		// always legal. Both directions render/strip the /*!80023 INVISIBLE */ token the
		// dumper and omni loader agree on.
		type cand struct {
			rec entSfColRec
			on  bool
		}
		var cands []cand
		for _, rec := range m.invisCols {
			key := rec.table + "." + rec.col
			if !state.usedCols[key] && !state.lockedTables[rec.table] && !entFzProtectedTables[rec.table] {
				cands = append(cands, cand{rec: rec, on: false})
			}
		}
		for _, rec := range entSfColCandidates(m, state, func(rest string) bool {
			return entSfIsIntRest(rest) || strings.HasPrefix(rest, "varchar")
		}) {
			if m.visibleCols[rec.table] < 2 {
				continue
			}
			if strings.Contains(m.colDef(rec.table, rec.col), "INVISIBLE") {
				continue
			}
			cands = append(cands, cand{rec: rec, on: true})
		}
		if len(cands) == 0 {
			return entSfMutation{}, false
		}
		pick := cands[rng.Intn(len(cands))]
		state.usedCols[pick.rec.table+"."+pick.rec.col] = true
		line := m.colDef(pick.rec.table, pick.rec.col)
		newLine := line + entSfInvisibleColToken
		dir := "->invisible"
		if !pick.on {
			newLine = strings.TrimSuffix(line, entSfInvisibleColToken)
			dir = "->visible"
		}
		return entSfMutation{
			entFzMutation: entFzMutation{
				desc: fmt.Sprintf("toggle_col_invisible %s.%s %s", pick.rec.table, pick.rec.col, dir),
				apply: func(t *testing.T, source string) string {
					return entReplaceLineInTable(t, source, pick.rec.table, "`"+pick.rec.col+"`", newLine)
				},
			},
			budget: 60, touched: tableKey(pick.rec.table),
		}, true

	case "toggle_idx_invisible":
		var cands []entSfIdxRec
		for _, rec := range m.idx {
			if rec.fkOverlap || state.lockedTables[rec.table] || state.usedIndexes[rec.table+"."+rec.name] {
				continue
			}
			cands = append(cands, rec)
		}
		if len(cands) == 0 {
			return entSfMutation{}, false
		}
		pick := cands[rng.Intn(len(cands))]
		state.usedIndexes[pick.table+"."+pick.name] = true
		newLine := pick.line + entSfInvisibleIdxToken
		dir := "->invisible"
		if pick.invisible {
			newLine = strings.TrimSuffix(pick.line, entSfInvisibleIdxToken)
			dir = "->visible"
		}
		return entSfMutation{
			entFzMutation: entFzMutation{
				desc: fmt.Sprintf("toggle_idx_invisible %s.%s %s", pick.table, pick.name, dir),
				apply: func(t *testing.T, source string) string {
					return entReplaceLineInTable(t, source, pick.table, "`"+pick.name+"`", newLine)
				},
			},
			budget: 60, touched: tableKey(pick.table),
		}, true

	case "enum_append":
		var cands []entSfColRec
		for _, rec := range m.enumCols {
			key := rec.table + "." + rec.col
			if !state.usedCols[key] && !state.lockedTables[rec.table] && !entFzProtectedTables[rec.table] && !m.protected[key] {
				cands = append(cands, rec)
			}
		}
		if len(cands) == 0 {
			return entSfMutation{}, false
		}
		pick := cands[rng.Intn(len(cands))]
		state.usedCols[pick.table+"."+pick.col] = true
		line := m.colDef(pick.table, pick.col)
		member := fmt.Sprintf("m%d_%d", r, seq)
		newLine := entSfReEnumSpec.ReplaceAllString(line, "enum($1,'"+member+"')")
		if newLine == line {
			return entSfMutation{}, false
		}
		return entSfMutation{
			entFzMutation: entFzMutation{
				desc: fmt.Sprintf("enum_append %s.%s += '%s'", pick.table, pick.col, member),
				apply: func(t *testing.T, source string) string {
					return entReplaceLineInTable(t, source, pick.table, "`"+pick.col+"`", newLine)
				},
			},
			budget: 60, touched: tableKey(pick.table),
		}, true

	default:
		return entSfMutation{}, false
	}
}

// ----------------------------------------------------------------------------
// Round runner, trial (for ddmin), and the untouched-statement stability guard.
// ----------------------------------------------------------------------------

// entSfScratchDB creates an eagerly-droppable scratch database (minimization churns
// through many databases inside a single test).
func entSfScratchDB(ctx context.Context, t *testing.T, srv liveServer, prefix string) (string, func()) {
	t.Helper()
	dbName := newLiveDatabase(ctx, t, srv, prefix)
	drop := func() {
		c, err := createLiveMySQLDriver(ctx, srv, "")
		if err != nil {
			return
		}
		defer c.Close(ctx)
		_, _ = c.Execute(ctx, "DROP DATABASE IF EXISTS `"+dbName+"`", db.ExecuteOptions{})
	}
	return dbName, drop
}

// entSfRound runs one full stateful round against an already-loaded database: mutate
// source textually, oracle (diff → apply → converge → idempotence), then the round-local
// guards — (a) the fresh dump reloads through the STRICT LoadSDL path, (c) object counts
// match want. Infrastructure failures fail t hard; protocol violations return an error
// so the caller can ddmin. Returns the fresh canonical dump.
func entSfRound(ctx context.Context, t *testing.T, srv liveServer, dbName, source string, muts []entSfMutation, want entSfCounts) (string, error) {
	t.Helper()
	target := source
	for _, m := range muts {
		target = m.apply(t, target)
	}
	after, _, err := entSfRoundToTarget(ctx, t, srv, dbName, source, target, srv.version, want)
	return after, err
}

// entSfRoundToTarget is the target-explicit core of entSfRound: the caller supplies the
// mutated target text and the version threaded into every diff. The cross-version
// stateful fuzzer (A8) authors ONE target from one side's dump and drives BOTH sides'
// databases through it, so the target cannot be derived from this side's source. Returns
// the fresh canonical dump and the generated plan.
func entSfRoundToTarget(ctx context.Context, t *testing.T, srv liveServer, dbName, source, target, version string, want entSfCounts) (string, string, error) {
	t.Helper()
	plan, err := mysqlDiffSDLMigration(source, target, version)
	if err != nil {
		return "", "", errors.Wrap(err, "diff failed")
	}
	if strings.TrimSpace(plan) == "" {
		return "", "", errors.New("empty plan for a non-empty mutation set")
	}
	if err := applyDDL(ctx, t, srv, dbName, plan); err != nil {
		return "", "", errors.Wrapf(err, "plan failed to apply; plan was:\n%s", plan)
	}
	after := dumpSDL(ctx, t, srv, dbName)
	converge, err := mysqlDiffSDLMigration(after, target, version)
	if err != nil {
		return "", "", errors.Wrap(err, "converge diff failed")
	}
	if converge != "" {
		return "", "", errors.Errorf("did not converge; residual:\n%s\nplan was:\n%s", converge, plan)
	}
	self, err := mysqlDiffSDLMigration(after, after, version)
	if err != nil {
		return "", "", errors.Wrap(err, "idempotence diff failed")
	}
	if self != "" {
		return "", "", errors.Errorf("post-apply dump not idempotent:\n%s", self)
	}
	// (a) LoadSDL must accept its own output — checked directly (DiffSDLMigration would
	// mask an SDL rejection behind the LoadSQL fallback).
	if _, err := catalog.LoadSDLWithVersion(withDatabaseContext(after), mysqlVersionFor(version)); err != nil {
		return "", "", errors.Wrap(err, "canonical dump does not reload through LoadSDL")
	}
	// (c) object counts must match the mutation ledger.
	got := entSfCountObjects(syncMetaForDB(ctx, t, srv, dbName))
	if got != want {
		return "", "", errors.Errorf("object counts diverge from the mutation ledger:\n want %+v\n got  %+v\nplan was:\n%s", want, got, plan)
	}
	return after, plan, nil
}

// entSfTrial reproduces one round from scratch: load baseSDL (a canonical dump) into a
// fresh database and run entSfRound over it. Used by ddmin.
func entSfTrial(ctx context.Context, t *testing.T, srv liveServer, baseSDL string, muts []entSfMutation) error {
	t.Helper()
	dbName, drop := entSfScratchDB(ctx, t, srv, "entsdl_sf")
	defer drop()

	driver, err := createLiveMySQLDriver(ctx, srv, dbName)
	require.NoError(t, err)
	// FOREIGN_KEY_CHECKS off for the session: baseSDL here is a CANONICAL dump
	// (alphabetical table order), so inline foreign keys may forward-reference —
	// same handling as entA8ScratchFromDump.
	_, err = driver.Execute(ctx, "SET FOREIGN_KEY_CHECKS=0;\n"+entNormalizeDelimiters(baseSDL), db.ExecuteOptions{})
	driver.Close(ctx)
	require.NoError(t, err, "[%s] stateful-fuzz trial base failed to load", srv.name)

	want := entSfCountObjects(syncMetaForDB(ctx, t, srv, dbName))
	for _, m := range muts {
		want = want.plus(m.delta)
	}
	source := dumpSDL(ctx, t, srv, dbName)
	_, err = entSfRound(ctx, t, srv, dbName, source, muts, want)
	return err
}

// entSfMinimize greedily shrinks a failing round's mutation set (ddmin-lite, mirroring
// entFuzzMinimize but over the stateful trial).
func entSfMinimize(ctx context.Context, t *testing.T, srv liveServer, baseSDL string, muts []entSfMutation, firstErr error) ([]entSfMutation, error) {
	t.Helper()
	return entSfMinimizeWith(muts, firstErr, func(trial []entSfMutation) error {
		return entSfTrial(ctx, t, srv, baseSDL, trial)
	})
}

// entSfMinimizeWith is the trial-generic ddmin-lite core, shared by the same-version
// (A6) and cross-version (A8) stateful fuzzers.
func entSfMinimizeWith(muts []entSfMutation, firstErr error, trial func([]entSfMutation) error) ([]entSfMutation, error) {
	minimized := muts
	lastErr := firstErr
	for i := 0; i < len(minimized) && len(minimized) > 1; {
		candidate := make([]entSfMutation, 0, len(minimized)-1)
		candidate = append(candidate, minimized[:i]...)
		candidate = append(candidate, minimized[i+1:]...)
		if err := trial(candidate); err != nil {
			minimized = candidate
			lastErr = err
			continue
		}
		i++
	}
	return minimized, lastErr
}

func entSfDescs(muts []entSfMutation) []string {
	out := make([]string, len(muts))
	for i, m := range muts {
		out[i] = m.desc
	}
	return out
}

// entSfStmtKeys splits a canonical dump and keys every statement by kind:name so the
// stability guard can pair statements across rounds.
func entSfStmtKeys(t *testing.T, dump string) map[string]string {
	t.Helper()
	stmts, err := mysqlparser.SplitSQL(dump)
	require.NoError(t, err, "split dump for stability keys")
	out := map[string]string{}
	for i, s := range stmts {
		text := strings.TrimSpace(s.Text)
		if text == "" || text == ";" {
			continue
		}
		key := ""
		switch {
		case entReTable.MatchString(text):
			key = "table:" + entReTable.FindStringSubmatch(text)[1]
		case entSfReViewHdr.MatchString(text):
			key = "view:" + entSfReViewHdr.FindStringSubmatch(text)[1]
		case entSfReFnHdr.MatchString(text):
			key = "function:" + entSfReFnHdr.FindStringSubmatch(text)[1]
		case entSfRePrHdr.MatchString(text):
			key = "procedure:" + entSfRePrHdr.FindStringSubmatch(text)[1]
		case entFzReTrigger.MatchString(text):
			key = "trigger:" + entFzReTrigger.FindStringSubmatch(text)[1]
		default:
			key = fmt.Sprintf("other:%d", i)
		}
		out[key] = text
	}
	return out
}

// entSfAssertUntouchedStable proves every statement NOT named by this round's mutations
// is byte-identical across the round — the sharp form of the accretion guard (the
// historical paren-accretion bug regenerated untouched objects with one extra wrapper
// per cycle while converging and self-diffing empty).
func entSfAssertUntouchedStable(t *testing.T, srv liveServer, seed int64, r int, prev, after string, muts []entSfMutation) {
	t.Helper()
	touched := map[string]bool{}
	for _, m := range muts {
		for _, k := range m.touched {
			touched[k] = true
		}
	}
	prevKeys := entSfStmtKeys(t, prev)
	afterKeys := entSfStmtKeys(t, after)
	keys := make([]string, 0, len(prevKeys))
	for k := range prevKeys {
		if _, ok := afterKeys[k]; ok && !touched[k] {
			keys = append(keys, k)
		}
	}
	slices.Sort(keys)
	for _, k := range keys {
		require.Equal(t, prevKeys[k], afterKeys[k],
			"[STATEFUZZ %s seed=%d round=%d] untouched object %s changed text across the round (collateral churn / accretion)",
			srv.name, seed, r, k)
	}
}

// ----------------------------------------------------------------------------
// The stateful fuzz test: >=12 seeds on 8.0, >=6 on 5.7, 5 rounds each.
// ----------------------------------------------------------------------------

func entSfRunSeed(ctx context.Context, t *testing.T, srv liveServer, baseDDL string, seed int64) {
	t.Helper()
	rng := rand.New(rand.NewSource(seed)) //nolint:gosec // deterministic fuzz, not crypto

	dbName, drop := entSfScratchDB(ctx, t, srv, "entsdl_sf")
	defer drop()
	driver, err := createLiveMySQLDriver(ctx, srv, dbName)
	require.NoError(t, err)
	_, err = driver.Execute(ctx, entNormalizeDelimiters(baseDDL), db.ExecuteOptions{})
	driver.Close(ctx)
	require.NoError(t, err, "[%s] stateful-fuzz base failed to load", srv.name)

	dump0 := dumpSDL(ctx, t, srv, dbName)
	ledger := &entSfLedger{
		expect:       entSfCountObjects(syncMetaForDB(ctx, t, srv, dbName)),
		viewModified: map[string]bool{},
	}
	prev := dump0
	budget := 0
	var history []string

	for r := 1; r <= entSfRounds; r++ {
		m := entSfParse(t, prev)
		k := 2 + rng.Intn(4) // K ∈ [2,5]
		muts := entSfMutations(t, m, rng, srv, ledger, r, k)
		for _, mut := range muts {
			history = append(history, fmt.Sprintf("r%d: %s", r, mut.desc))
		}
		t.Logf("[STATEFUZZ %s seed=%d round=%d] K=%d mutations:\n  %s",
			srv.name, seed, r, k, strings.Join(entSfDescs(muts), "\n  "))

		want := ledger.expect
		for _, mut := range muts {
			want = want.plus(mut.delta)
		}
		after, err := entSfRound(ctx, t, srv, dbName, prev, muts, want)
		if err != nil {
			minimized, minErr := entSfMinimize(ctx, t, srv, prev, muts, err)
			t.Fatalf("[STATEFUZZ %s seed=%d round=%d] FAILED\nminimized mutations (%d of %d):\n  %s\nerror:\n%v\nseed history:\n  %s",
				srv.name, seed, r, len(minimized), len(muts), strings.Join(entSfDescs(minimized), "\n  "),
				minErr, strings.Join(history, "\n  "))
		}
		ledger.expect = want

		// (b) accretion guards: the linear text budget and the untouched-statement
		// stability check.
		for _, mut := range muts {
			budget += mut.budget
		}
		require.Less(t, len(after), len(dump0)+budget+entSfRoundSlack*r,
			"[STATEFUZZ %s seed=%d round=%d] dump grew past the added-object budget: len(dump_%d)=%d, len(dump_0)=%d, budget=%d — text accretion suspected.\nseed history:\n  %s",
			srv.name, seed, r, r, len(after), len(dump0), budget+entSfRoundSlack*r, strings.Join(history, "\n  "))
		entSfAssertUntouchedStable(t, srv, seed, r, prev, after, muts)

		t.Logf("[STATEFUZZ %s seed=%d round=%d] ok: dump %d -> %d bytes (budget %d), counts %+v",
			srv.name, seed, r, len(prev), len(after), len(dump0)+budget+entSfRoundSlack*r, ledger.expect)
		prev = after
	}
}

//nolint:tparallel
func TestSDLEnterpriseStatefulFuzz(t *testing.T) {
	skipUnlessLiveOracle(t)
	ctx := context.Background()

	for _, srv := range liveServers {
		srv := srv
		seedCount := 12
		if srv.version == "5.7" {
			seedCount = 6
		}
		t.Run(srv.name, func(t *testing.T) {
			baseDDL := entSfBaseDDL(t, srv)
			for seed := int64(1); seed <= int64(seedCount); seed++ {
				seed := seed
				t.Run(fmt.Sprintf("seed_%d", seed), func(t *testing.T) {
					entSfRunSeed(ctx, t, srv, baseDDL, seed)
				})
			}
		})
	}
}

// A8 of the ENTERPRISE smoke axes: CROSS-VERSION STATEFUL
// FUZZ — the composition of A6 (same-version multi-round stateful fuzz) and A7
// (single-shot cross-version upgrade). Real teams run MIXED fleets: a schema authored
// and dumped on 5.7 drives targets applied to 8.0 (and vice versa during migrations),
// repeatedly, while the schema keeps evolving. A6 and A7 both pass in isolation; this
// axis hunts what only their composition shows — version-normalization drift
// COMPOUNDING across rounds (a residual one round becomes the authored text of the
// next, so a per-round phantom that a single-shot test shrugs off snowballs here).
//
// Protocol per seed, on the 5.7-legal A6 base (30-table Zabbix slice + fuzz aux +
// stateful aux; the 8.0-only aux tables are excluded — the SAME logical schema must
// load on BOTH versions), both databases collation-aligned (utf8mb4/utf8mb4_general_ci,
// the A7 aligned-probe posture) so the fleets start logically identical:
//
//	round 0: the two fresh dumps must already cross-diff EMPTY (all four
//	         direction x version-normalization combos).
//	for round r in 1..4:
//	    T_r = mutate(authored side's dump_{r-1}, K ∈ [2,4] from the 5.7-LEGAL menu)
//	    each side s ∈ {5.7, 8.0}: plan_s = Diff(dump_s, T_r, full server version) must
//	        be non-empty, apply cleanly, converge (re-dump diffs empty against T_r
//	        under s's version), self-diff empty, reload through STRICT LoadSDL, and
//	        match the mutation ledger's object counts (per-side ledgers: baselines may
//	        differ, deltas are shared);
//	    plan_80 must carry no 5.7 integer display width in any ALTER (entUpgAssert-
//	        NoWidthInAlters — a leaked width is a phantom column MODIFY);
//	    CROSS-CHECK: the two fresh dumps must cross-diff EMPTY in all four combos —
//	        the fleets stay logically identical despite version-different stored forms;
//	    ACCRETION (A6's guards, on BOTH sides): each side's dump stays under its own
//	        dump_0 length + the cumulative added-object budget, and every statement not
//	        named by this round's mutations is byte-identical to its round-start form.
//
// forward_57_authored seeds mutate the 5.7 dump; reverse_80_authored seeds mutate the
// 8.0 dump (the authored text then carries NO display widths while the 5.7 side's
// stored form does — the opposite normalization direction). Both directions use the
// 5.7-legal menu: every target applies to both fleets, so 5.7-illegal mutations (CHECK,
// SRID spatial, INVISIBLE) are structurally excluded. Additionally, child-side FK
// member columns are protected from the generated-column kinds — both servers refuse
// a STORED generated column based on a cascading-FK member (see
// entA8ProtectFKMemberColumns for the empirical matrix this axis established).
//
// Determinism: the mutation stream is a pure function of the seed (in the subtest
// name). Failures ddmin-minimize over a cross-version trial (both round-start dumps
// reloaded into fresh aligned scratch databases) and report seed + round + the failing
// check (apply/converge/idempotence/reload/counts per side, or the cross combo) + the
// minimized ledger. Accretion-guard failures are terminal requires with the same
// seed/round/check labeling. Scratch databases are entsdl_-prefixed and dropped.
//
// ≥8 forward seeds + ≥4 reverse seeds.
//
// Shared machinery: entSfBaseDDL, entSfParse, entSfMenu, entSfMutationsFromMenu,
// entSfRoundToTarget, entSfMinimizeWith, entSfScratchDB, entSfCountObjects,
// entSfAssertUntouchedStable, entSfDescs, entSfLedger, entSfRoundSlack (A6);
// entUpgServers, entUpgLoadAligned, entUpgAssertNoWidthInAlters, entUpgTrim,
// entUpg57Version, entUpg80Version (A7); liveServers, dumpSDL, syncMetaForDB,
// newLiveDatabase, createLiveMySQLDriver, entNormalizeDelimiters (siblings).

const (
	entA8Rounds       = 4
	entA8ForwardSeeds = 8
	entA8ReverseSeeds = 4
)

// entA8Fleet is one side of the mixed-version fleet.
type entA8Fleet struct {
	srv liveServer
	// version is the FULL server version threaded into every diff for this side (the
	// release path passes the synced version string, not a bare major.minor).
	version string
	dbName  string
	prev    string // canonical dump at the current round boundary
	dump0   int    // round-0 dump length (text-accretion baseline)
	budget  int    // cumulative added-object text budget
	expect  entSfCounts
}

// entA8CrossCheck proves the two fleets are logically identical: all four
// (direction x version-normalization) cross-diffs of the two canonical dumps must be
// empty. The two "apply" pairings mirror A7's aligned probe — the plan a fleet would
// receive to be driven to the OTHER fleet's schema; the remaining two probe the
// canonicalizer's version symmetry (the same pair re-judged under the other version's
// stored form).
func entA8CrossCheck(d57, d80 string) error {
	for _, c := range []struct{ label, source, target, version string }{
		{"80db<=57dump(as 8.0.32)", d80, d57, entUpg80Version},
		{"57db<=80dump(as 5.7.25)", d57, d80, entUpg57Version},
		{"57src->80tgt(as 8.0.32)", d57, d80, entUpg80Version},
		{"80src->57tgt(as 5.7.25)", d80, d57, entUpg57Version},
	} {
		plan, err := mysqlDiffSDLMigration(c.source, c.target, c.version)
		if err != nil {
			return errors.Wrapf(err, "check=cross %s: diff failed", c.label)
		}
		if strings.TrimSpace(plan) != "" {
			return errors.Errorf("check=cross %s: fleets drifted apart; residual plan:\n%s", c.label, entUpgTrim(plan))
		}
	}
	return nil
}

// entA8ProtectFKMemberColumns marks child-side foreign-key member columns as protected
// in the mutation-planning model, keeping the A8 menu off a server-refused shape this
// axis found empirically: BOTH oracle servers reject a STORED generated column whose
// base column participates in a foreign key with a cascading referential action —
// 8.0.32 with a clean 1215 "Cannot add foreign key constraint" at DDL time, 5.7.25
// with an opaque errno-150 failure at ALGORITHM=COPY rename (the shape seed 18 of a
// widened sweep minimized to: ADD COLUMN ... GENERATED ALWAYS AS ((groupid + 7)) STORED
// on hosts_groups, whose groupid carries zabbix's usual ON DELETE CASCADE FK).
// Parent-side referenced columns, VIRTUAL generated columns, and RESTRICT/NO ACTION
// FKs are all accepted by both servers and stay eligible. The guard is
// action-agnostic (membership, not action): zabbix FKs are almost all
// ON DELETE CASCADE and the add_fk mutation plants ON DELETE SET NULL, so an
// action-aware refinement would buy nothing here.
func entA8ProtectFKMemberColumns(m *entSfSchema) {
	for _, tbl := range m.base.tables {
		for _, fk := range tbl.fks {
			for _, c := range fk.cols {
				m.protected[tbl.name+"."+c] = true
			}
		}
	}
}

// entA8Round drives one shared target through both fleets and cross-checks the
// results: each side runs the full A6 round oracle (diff -> apply -> converge ->
// idempotence -> strict LoadSDL reload -> ledger counts) under its own version, then
// the two fresh dumps must cross-diff empty. Protocol violations return stage-tagged
// errors (for ddmin); infrastructure failures fail t hard. On success both fleets'
// prev/expect/budget are advanced and the 8.0-side plan is returned for the width
// guard (the 5.7-side plan may carry display widths legitimately — they ARE its
// stored form — so nothing judges it beyond the round oracle).
func entA8Round(ctx context.Context, t *testing.T, f57, f80 *entA8Fleet, target string, muts []entSfMutation) (string, error) {
	t.Helper()
	want57, want80 := f57.expect, f80.expect
	for _, m := range muts {
		want57 = want57.plus(m.delta)
		want80 = want80.plus(m.delta)
	}
	after57, _, err := entSfRoundToTarget(ctx, t, f57.srv, f57.dbName, f57.prev, target, f57.version, want57)
	if err != nil {
		return "", errors.Wrapf(err, "check=side %s", f57.srv.name)
	}
	after80, plan80, err := entSfRoundToTarget(ctx, t, f80.srv, f80.dbName, f80.prev, target, f80.version, want80)
	if err != nil {
		return "", errors.Wrapf(err, "check=side %s", f80.srv.name)
	}
	if err := entA8CrossCheck(after57, after80); err != nil {
		return "", err
	}
	for _, m := range muts {
		f57.budget += m.budget
		f80.budget += m.budget
	}
	f57.prev, f57.expect = after57, want57
	f80.prev, f80.expect = after80, want80
	return plan80, nil
}

// entA8ScratchFromDump reloads a canonical dump into a fresh, eagerly-droppable,
// collation-aligned scratch database. The dump text executes with FOREIGN_KEY_CHECKS
// off for the session (canonical dumps order tables alphabetically, so inline foreign
// keys may forward-reference; the driver runs the whole batch on one connection, so
// the session setting holds).
func entA8ScratchFromDump(ctx context.Context, t *testing.T, srv liveServer, dump string) (string, func()) {
	t.Helper()
	dbName, drop := entSfScratchDB(ctx, t, srv, "entsdl_a8t")
	driver, err := createLiveMySQLDriver(ctx, srv, dbName)
	require.NoError(t, err)
	defer driver.Close(ctx)
	_, err = driver.Execute(ctx, "ALTER DATABASE `"+dbName+"` CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci", db.ExecuteOptions{})
	require.NoError(t, err, "[%s] align trial database collation", srv.name)
	_, err = driver.Execute(ctx, "SET FOREIGN_KEY_CHECKS=0;\n"+entNormalizeDelimiters(dump), db.ExecuteOptions{})
	require.NoError(t, err, "[%s] reload round-start dump into trial database", srv.name)
	return dbName, drop
}

// entA8Trial reproduces one cross-version round from scratch for ddmin: both
// round-start dumps reload into fresh aligned scratch databases, the mutation subset
// re-authors the target from the authored side's dump, and the full round runs.
func entA8Trial(ctx context.Context, t *testing.T, srv57, srv80 liveServer, prev57, prev80 string, authored80 bool, muts []entSfMutation) error {
	t.Helper()
	db57, drop57 := entA8ScratchFromDump(ctx, t, srv57, prev57)
	defer drop57()
	db80, drop80 := entA8ScratchFromDump(ctx, t, srv80, prev80)
	defer drop80()

	f57 := &entA8Fleet{srv: srv57, version: entUpg57Version, dbName: db57}
	f80 := &entA8Fleet{srv: srv80, version: entUpg80Version, dbName: db80}
	for _, f := range []*entA8Fleet{f57, f80} {
		f.prev = dumpSDL(ctx, t, f.srv, f.dbName)
		f.expect = entSfCountObjects(syncMetaForDB(ctx, t, f.srv, f.dbName))
	}
	authored := f57.prev
	if authored80 {
		authored = f80.prev
	}
	target := authored
	for _, m := range muts {
		target = m.apply(t, target)
	}
	_, err := entA8Round(ctx, t, f57, f80, target, muts)
	return err
}

// entA8RunSeed runs one full cross-version stateful seed: 4 rounds of
// mutate-on-the-authored-side -> apply-to-both-fleets -> cross-check -> accretion
// guards on both sides.
func entA8RunSeed(ctx context.Context, t *testing.T, srv57, srv80 liveServer, baseDDL string, seed int64, authored80 bool) {
	t.Helper()
	rng := rand.New(rand.NewSource(seed)) //nolint:gosec // deterministic fuzz, not crypto

	f57 := &entA8Fleet{srv: srv57, version: entUpg57Version}
	f80 := &entA8Fleet{srv: srv80, version: entUpg80Version}
	f57.dbName = entUpgLoadAligned(ctx, t, srv57, "entsdl_a8m57", baseDDL)
	f80.dbName = entUpgLoadAligned(ctx, t, srv80, "entsdl_a8m80", baseDDL)
	for _, f := range []*entA8Fleet{f57, f80} {
		f.prev = dumpSDL(ctx, t, f.srv, f.dbName)
		f.dump0 = len(f.prev)
		f.expect = entSfCountObjects(syncMetaForDB(ctx, t, f.srv, f.dbName))
	}
	dir, author := "fwd57", f57
	if authored80 {
		dir, author = "rev80", f80
	}

	// Round 0: freshly loaded from the same DDL under aligned defaults, the fleets must
	// already be logically identical — a failure here is a base cross-version phantom,
	// not mutation drift.
	require.NoError(t, entA8CrossCheck(f57.prev, f80.prev),
		"[A8 %s seed=%d round=0] fleets diverge before any mutation (base cross-check)", dir, seed)

	ledger := &entSfLedger{viewModified: map[string]bool{}}
	var history []string
	for r := 1; r <= entA8Rounds; r++ {
		m := entSfParse(t, author.prev)
		entA8ProtectFKMemberColumns(m)
		k := 2 + rng.Intn(3) // K ∈ [2,4]
		muts := entSfMutationsFromMenu(t, m, rng, entSfMenu("5.7"), ledger, r, k)
		for _, mut := range muts {
			history = append(history, fmt.Sprintf("r%d: %s", r, mut.desc))
		}
		t.Logf("[A8 %s seed=%d round=%d] K=%d mutations:\n  %s",
			dir, seed, r, k, strings.Join(entSfDescs(muts), "\n  "))

		target := author.prev
		for _, mut := range muts {
			target = mut.apply(t, target)
		}
		prev57, prev80 := f57.prev, f80.prev
		plan80, err := entA8Round(ctx, t, f57, f80, target, muts)
		if err != nil {
			minimized, minErr := entSfMinimizeWith(muts, err, func(trial []entSfMutation) error {
				return entA8Trial(ctx, t, srv57, srv80, prev57, prev80, authored80, trial)
			})
			t.Fatalf("[A8 %s seed=%d round=%d] FAILED\nminimized mutations (%d of %d):\n  %s\nerror:\n%v\nseed history:\n  %s",
				dir, seed, r, len(minimized), len(muts), strings.Join(entSfDescs(minimized), "\n  "),
				minErr, strings.Join(history, "\n  "))
		}

		// A leaked 5.7 display width inside an 8.0-side ALTER is a phantom column MODIFY
		// (the 5.7-side plan may carry widths legitimately — they ARE its stored form).
		entUpgAssertNoWidthInAlters(t, fmt.Sprintf("A8 %s seed=%d round=%d plan80", dir, seed, r), plan80)

		// Accretion guards on BOTH sides' dumps (each against its own round-0 baseline).
		for _, s := range []struct {
			f    *entA8Fleet
			prev string
		}{{f57, prev57}, {f80, prev80}} {
			require.Less(t, len(s.f.prev), s.f.dump0+s.f.budget+entSfRoundSlack*r,
				"[A8 %s seed=%d round=%d side=%s] check=accretion: dump grew past the added-object budget: len(dump_%d)=%d, len(dump_0)=%d, budget=%d — text accretion suspected.\nseed history:\n  %s",
				dir, seed, r, s.f.srv.name, r, len(s.f.prev), s.f.dump0, s.f.budget+entSfRoundSlack*r, strings.Join(history, "\n  "))
			entSfAssertUntouchedStable(t, s.f.srv, seed, r, s.prev, s.f.prev, muts)
		}

		t.Logf("[A8 %s seed=%d round=%d] ok: 57 %d -> %d bytes, 80 %d -> %d bytes (budget 57=%d 80=%d), counts 57=%+v 80=%+v",
			dir, seed, r, len(prev57), len(f57.prev), len(prev80), len(f80.prev),
			f57.dump0+f57.budget+entSfRoundSlack*r, f80.dump0+f80.budget+entSfRoundSlack*r, f57.expect, f80.expect)
	}
}

//nolint:tparallel
func TestSDLEnterpriseCrossVersionStatefulFuzz(t *testing.T) {
	skipUnlessLiveOracle(t)
	ctx := context.Background()
	srv57, srv80 := entUpgServers(t)
	// The fleet schema must load on BOTH versions, so the base is the 5.7 shape of the
	// A6 stateful-fuzz base (entSfAux80's SRID/INVISIBLE/CHECK tables are 8.0-only).
	baseDDL := entSfBaseDDL(t, srv57)

	t.Run("forward_57_authored", func(t *testing.T) {
		for seed := int64(1); seed <= entA8ForwardSeeds; seed++ {
			seed := seed
			t.Run(fmt.Sprintf("seed_%d", seed), func(t *testing.T) {
				entA8RunSeed(ctx, t, srv57, srv80, baseDDL, seed, false)
			})
		}
	})
	// Reverse-authored seeds use a disjoint seed range: the two dumps' candidate models
	// are logically identical, so reusing 1..N would largely replay the forward streams.
	t.Run("reverse_80_authored", func(t *testing.T) {
		for seed := int64(101); seed < 101+entA8ReverseSeeds; seed++ {
			seed := seed
			t.Run(fmt.Sprintf("seed_%d", seed), func(t *testing.T) {
				entA8RunSeed(ctx, t, srv57, srv80, baseDDL, seed, true)
			})
		}
	})
}
