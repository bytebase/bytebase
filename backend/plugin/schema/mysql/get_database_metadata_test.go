package mysql

import (
	"testing"

	metadatapb "github.com/bytebase/omni/metadata"
	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/schema"
)

func TestGetDatabaseMetadataRegistry(t *testing.T) {
	empty, err := schema.GetDatabaseMetadata(storepb.Engine_MYSQL, "   \n")
	require.NoError(t, err)
	require.Empty(t, empty.Schemas)

	meta, err := schema.GetDatabaseMetadata(storepb.Engine_MYSQL, "CREATE TABLE `t` (`id` int NOT NULL, PRIMARY KEY (`id`));")
	require.NoError(t, err)
	require.Len(t, meta.Schemas, 1)
	require.Len(t, meta.Schemas[0].Tables, 1)
	require.Equal(t, "t", meta.Schemas[0].Tables[0].Name)

	_, err = schema.GetDatabaseMetadata(storepb.Engine_MYSQL, "CREATE TABLE (")
	require.Error(t, err)
}

// TestGetDatabaseMetadataParsesPrettyView proves GetDatabaseMetadata accepts the
// multi-line CREATE OR REPLACE VIEW the SDL dump pretty-prints.
func TestGetDatabaseMetadataParsesPrettyView(t *testing.T) {
	sdl, err := GetDatabaseDefinition(schema.GetDefinitionContext{SDLFormat: true}, &metadatapb.DatabaseSchemaMetadata{
		Name: "testdb",
		Schemas: []*metadatapb.SchemaMetadata{{
			Tables: []*metadatapb.TableMetadata{{
				Name:    "t1",
				Engine:  "InnoDB",
				Columns: []*metadatapb.ColumnMetadata{{Name: "a", Type: "int"}, {Name: "b", Type: "varchar(10)", Nullable: true, Default: "NULL"}},
				Indexes: []*metadatapb.IndexMetadata{{Name: "PRIMARY", Primary: true, Unique: true, Expressions: []string{"a"}, Visible: true}},
			}},
			Views: []*metadatapb.ViewMetadata{{
				Name:       "v_pretty",
				Definition: "select `testdb`.`t1`.`a` AS `a`,`testdb`.`t1`.`b` AS `b` from `testdb`.`t1` where (`testdb`.`t1`.`a` > 0)",
			}},
		}},
	})
	require.NoError(t, err)
	require.Contains(t, sdl, "AS\nselect\n  `t1`.`a` AS `a`,", "fixture must actually be pretty-printed")

	meta, err := GetDatabaseMetadata(sdl)
	require.NoError(t, err)
	require.Len(t, meta.Schemas, 1)
	require.Len(t, meta.Schemas[0].Views, 1)
	view := meta.Schemas[0].Views[0]
	require.Equal(t, "v_pretty", view.Name)
	require.NotContains(t, view.Definition, "\n")
}
