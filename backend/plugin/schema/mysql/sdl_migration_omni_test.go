package mysql

import (
	"strings"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"

	"github.com/bytebase/omni/mysql/catalog"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
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
