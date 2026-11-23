package mysql

import (
	"strings"
	"testing"

	metadatapb "github.com/bytebase/omni/metadata"
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

	t.Run("replace_vs_drop_view_and_function", func(t *testing.T) {
		currentObj := `
CREATE TABLE author (id INT PRIMARY KEY) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
CREATE VIEW v_keep AS SELECT id FROM author;
CREATE VIEW v_drop AS SELECT id FROM author;
CREATE FUNCTION f_rep(a INT) RETURNS INT DETERMINISTIC RETURN a + 1;
CREATE FUNCTION f_drop(a INT) RETURNS INT DETERMINISTIC RETURN a;
`
		targetObj := `
CREATE TABLE author (id INT PRIMARY KEY) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
CREATE VIEW v_keep AS SELECT id + 1 AS id FROM author;
CREATE VIEW v_new AS SELECT id FROM author;
CREATE FUNCTION f_rep(a INT) RETURNS INT DETERMINISTIC RETURN a + 2;
CREATE FUNCTION f_new(a INT) RETURNS INT DETERMINISTIC RETURN a;
`
		advices, err := mysqlSDLDropAdvices(strings.TrimSpace(targetObj), metadataFromSDL(t, currentObj), "8.0")
		require.NoError(t, err)

		var drops, replaces []string
		for _, a := range advices {
			require.Equal(t, storepb.Advice_WARNING, a.Status)
			switch a.Code {
			case code.SDLDropOperation.Int32():
				drops = append(drops, a.Content)
			case code.SDLReplaceOperation.Int32():
				replaces = append(replaces, a.Content)
			default:
				t.Fatalf("unexpected advice code %d: %+v", a.Code, a)
			}
		}
		require.ElementsMatch(t, []string{
			"Dropping view 'v_drop' will affect dependent objects.",
			"Dropping function 'f_drop' will affect dependent objects.",
		}, drops)
		require.ElementsMatch(t, []string{
			"View 'v_keep' definition will be replaced.",
			"Function 'f_rep' definition will be replaced.",
		}, replaces)
	})
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

// TestSDLMigrationForwardsSessionContextAndStripsDatabaseContext verifies that
// schema.SDLMigration forwards per-object session context from metadata into
// omni and strips the synthetic catalog database qualifier from migration SQL.
func TestSDLMigrationForwardsSessionContextAndStripsDatabaseContext(t *testing.T) {
	current := metadataFromProto(&metadatapb.DatabaseSchemaMetadata{
		Name: "app",
		Schemas: []*metadatapb.SchemaMetadata{{
			Functions: []*metadatapb.FunctionMetadata{{
				Name:                "f",
				Definition:          "CREATE FUNCTION `f`(a int) RETURNS int DETERMINISTIC RETURN a + 1",
				SqlMode:             "PIPES_AS_CONCAT",
				CharacterSetClient:  "utf8mb4",
				CollationConnection: "utf8mb4_0900_ai_ci",
			}},
		}},
	})
	target := "CREATE FUNCTION f(a INT) RETURNS INT DETERMINISTIC RETURN a + 2;\n"

	sql, err := schema.SDLMigration(storepb.Engine_MYSQL, target, current, "8.0.32")
	require.NoError(t, err)
	require.Contains(t, sql, "SET sql_mode = 'PIPES_AS_CONCAT'")
	require.NotContains(t, sql, sdlDatabaseContext)

	sourceSDL, err := schema.MetadataToSDL(storepb.Engine_MYSQL, current)
	require.NoError(t, err)
	bareSQL, err := schema.DiffSDLMigration(storepb.Engine_MYSQL, sourceSDL, target, "8.0.32")
	require.NoError(t, err)
	require.NotContains(t, bareSQL, "SET sql_mode")
	require.NotContains(t, bareSQL, sdlDatabaseContext)
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
