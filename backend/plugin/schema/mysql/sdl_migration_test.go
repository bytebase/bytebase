package mysql

import (
	"strings"
	"testing"

	metadatapb "github.com/bytebase/omni/metadata"
	"github.com/bytebase/omni/mysql/catalog"
	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	"github.com/bytebase/bytebase/backend/plugin/schema"
	"github.com/bytebase/bytebase/backend/store/model"
)

// metadataFromSDL builds a model.DatabaseMetadata from SDL text using the registered MySQL
// GetDatabaseMetadata, so drop-advice tests can run fully offline.
func metadataFromSDL(t *testing.T, sdl string) *model.DatabaseMetadata {
	t.Helper()
	proto, err := schema.GetDatabaseMetadata(storepb.Engine_MYSQL, sdl)
	require.NoError(t, err)
	return model.NewDatabaseMetadata(proto, nil, nil, storepb.Engine_MYSQL, true)
}

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
			require.NotContains(t, a.Content, sdlDatabaseContext)
		}

		joined := ""
		for _, a := range advices {
			joined += a.Content + "\n"
		}
		require.Contains(t, joined, "book", "expected dropped table named:\n%s", joined)
		require.Contains(t, joined, "bio", "expected dropped column named:\n%s", joined)
	})
}

func TestSDLLoadErrorReportsCallerLineNumbers(t *testing.T) {
	// Invalid token on line 3 of the caller's SDL; omni pre-creates the default
	// database without a synthetic SQL prologue, so line numbers match caller text.
	target := "CREATE TABLE `t` (\n" +
		"  `id` int NOT NULL,\n" +
		"  BOGUS TOKEN HERE,\n" +
		"  PRIMARY KEY (`id`)\n" +
		");\n"
	_, err := mysqlDiffSDLMigration("", target, "8.0.32")
	require.Error(t, err)
	require.Contains(t, err.Error(), "(line 3, column")
	require.NotContains(t, err.Error(), "(line 5, column")
}

// TestSDLRegistrationsAreMySQLOnly pins that OceanBase is not registered for
// the MySQL SDL entry points pending OceanBase oracle validation.
func TestSDLRegistrationsAreMySQLOnly(t *testing.T) {
	_, err := schema.DiffSDLMigration(storepb.Engine_OCEANBASE, "", "", "")
	require.ErrorContains(t, err, "not supported")

	_, err = schema.SDLDropAdvices(storepb.Engine_OCEANBASE, "", nil, "")
	require.ErrorContains(t, err, "not supported")

	_, err = schema.GetMultiFileDatabaseDefinition(storepb.Engine_OCEANBASE, schema.GetDefinitionContext{}, &metadatapb.DatabaseSchemaMetadata{})
	require.ErrorContains(t, err, "not supported")

	_, err = schema.DiffSDLMigration(storepb.Engine_MYSQL, "", "", "")
	require.NoError(t, err)
	_, err = schema.SDLDropAdvices(storepb.Engine_MYSQL, "", nil, "")
	require.NoError(t, err)
	_, err = schema.GetMultiFileDatabaseDefinition(storepb.Engine_MYSQL, schema.GetDefinitionContext{}, &metadatapb.DatabaseSchemaMetadata{})
	require.NoError(t, err)
}

// metadataFromProto wraps a raw proto in the model type the diff entry points take.
func metadataFromProto(proto *metadatapb.DatabaseSchemaMetadata) *model.DatabaseMetadata {
	return model.NewDatabaseMetadata(proto, nil, nil, storepb.Engine_MYSQL, true)
}

// TestDiffMigrationLoadsMetadataNotSDL proves MySQL's metadata diff loads the
// snapshots into the catalog rather than round-tripping them through SDL text: the
// view body does not parse, so the SDL path fails on it and the metadata path does
// not.
func TestDiffMigrationLoadsMetadataNotSDL(t *testing.T) {
	brokenView := &metadatapb.ViewMetadata{Name: "v_broken", Definition: "select ((broken from"}
	id := &metadatapb.ColumnMetadata{Name: "id", Type: "int"}
	extra := &metadatapb.ColumnMetadata{Name: "extra", Type: "varchar(10)", Nullable: true, Default: "NULL"}
	oldProto := oneTableSnapshot("d", []*metadatapb.ColumnMetadata{id}, brokenView)
	newProto := oneTableSnapshot("d", []*metadatapb.ColumnMetadata{id, extra}, brokenView)

	sql, err := schema.DiffMigration(storepb.Engine_MYSQL, metadataFromProto(oldProto), metadataFromProto(newProto))
	require.NoError(t, err)
	require.Equal(t, "ALTER TABLE `t` ADD COLUMN `extra` varchar(10) DEFAULT NULL", sql)

	sdl, err := schema.MetadataToSDL(storepb.Engine_MYSQL, metadataFromProto(oldProto))
	require.NoError(t, err)
	_, err = schema.DiffSDLMigration(storepb.Engine_MYSQL, sdl, sdl, "")
	require.Error(t, err, "the SDL path must reject the view body")
}

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
			name: "table_actually_named_bbcatalog",
			in:   "ALTER TABLE `bbcatalog`.`bbcatalog` ADD COLUMN `c` int",
			want: "ALTER TABLE `bbcatalog` ADD COLUMN `c` int",
		},
		{
			name: "alias_named_bbcatalog_preserved",
			in:   "CREATE OR REPLACE VIEW `bbcatalog`.`v` AS SELECT `bbcatalog`.`id` FROM `bbcatalog`.`t1` AS `bbcatalog` JOIN `bbcatalog`.`t2` ON `bbcatalog`.`id` = `t2`.`id`",
			want: "CREATE OR REPLACE VIEW `v` AS SELECT `bbcatalog`.`id` FROM `t1` AS `bbcatalog` JOIN `t2` ON `bbcatalog`.`id` = `t2`.`id`",
		},
		{
			name: "three_part_reference",
			in:   "select `bbcatalog`.`t`.`c` from `bbcatalog`.`t`",
			want: "select `t`.`c` from `t`",
		},
		{
			name: "no_qualifier_untouched",
			in:   "ALTER TABLE `t` ADD COLUMN `c` int DEFAULT 'bbcatalog'",
			want: "ALTER TABLE `t` ADD COLUMN `c` int DEFAULT 'bbcatalog'",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, catalog.StripDatabaseContext(tc.in, sdlDatabaseContext))
		})
	}
}

func TestDiffSDLMigrationEmptyTargetDropsEverything(t *testing.T) {
	source := "CREATE TABLE t (id INT PRIMARY KEY);\n"
	sql, err := schema.DiffSDLMigration(storepb.Engine_MYSQL, source, "", "8.0.32")
	require.NoError(t, err)
	require.Contains(t, sql, "DROP TABLE")
	require.NotContains(t, sql, sdlDatabaseContext)
}

func TestLoadCatalogFallbackBareTimestamp(t *testing.T) {
	const table = "CREATE TABLE `t` (`ts` timestamp) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;\n"

	// LOCK TABLES is not accepted by LoadSDL, forcing the LoadSQL fallback.
	target := "LOCK TABLES `t` WRITE;\n" + table + "UNLOCK TABLES;\n"

	sql80, err := schema.DiffSDLMigration(storepb.Engine_MYSQL, table, target, "8.0.32")
	require.NoError(t, err)
	require.Empty(t, sql80, "8.0 source/target with identical bare TIMESTAMP must not phantom-diff; got %q", sql80)
}

func TestSDLRejectsCTAS(t *testing.T) {
	const source = "CREATE TABLE `t` (`id` int NOT NULL, PRIMARY KEY (`id`)) ENGINE=InnoDB;\n"

	for _, target := range []string{
		"CREATE TABLE `t` AS SELECT 1 AS `id`;\n",
		"LOCK TABLES `t` WRITE;\nCREATE TABLE `t` AS SELECT 1 AS `id`;\nUNLOCK TABLES;\n",
	} {
		_, err := schema.DiffSDLMigration(storepb.Engine_MYSQL, source, target, "8.0.32")
		require.ErrorContains(t, err, "CREATE TABLE ... AS SELECT")
	}
}

func TestSDLRejectsServersBelow80(t *testing.T) {
	const table = "CREATE TABLE `t` (`id` int NOT NULL, PRIMARY KEY (`id`)) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;\n"
	const changed = "CREATE TABLE `t` (`id` int NOT NULL, `c` int, PRIMARY KEY (`id`)) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;\n"
	meta := metadataFromSDL(t, table)

	for _, version := range []string{"5.7.44", "5.7", "5.6.51", "5.7.44-log", " 5.7.25 ", "5.7.mysql_aurora.2.11.2"} {
		_, err := schema.DiffSDLMigration(storepb.Engine_MYSQL, table, changed, version)
		require.ErrorContains(t, err, "requires MySQL 8.0 or later", "diff, version %q", version)

		_, err = schema.SDLMigration(storepb.Engine_MYSQL, changed, meta, version)
		require.ErrorContains(t, err, "requires MySQL 8.0 or later", "rollout, version %q", version)

		advices, err := schema.SDLDropAdvices(storepb.Engine_MYSQL, changed, meta, version)
		require.NoError(t, err, "drop advices, version %q", version)
		require.Len(t, advices, 1, "drop advices, version %q", version)
		require.Equal(t, storepb.Advice_ERROR, advices[0].Status)
		require.Equal(t, code.Unsupported.Int32(), advices[0].Code)
		require.Contains(t, advices[0].Content, "requires MySQL 8.0 or later")
	}

	for _, version := range []string{"8.0.32", "8.0", "8.4.0", "9.0.1", "8.0.mysql_aurora.3.05.2", "", "not-a-version"} {
		migration, err := schema.DiffSDLMigration(storepb.Engine_MYSQL, table, changed, version)
		require.NoError(t, err, "diff, version %q", version)
		require.Contains(t, migration, "ADD COLUMN", "diff, version %q", version)

		advices, err := schema.SDLDropAdvices(storepb.Engine_MYSQL, changed, meta, version)
		require.NoError(t, err, "drop advices, version %q", version)
		require.Empty(t, advices, "drop advices, version %q", version)
	}
}
