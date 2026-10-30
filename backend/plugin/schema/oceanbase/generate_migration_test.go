package oceanbase

import (
	"testing"

	metadatapb "github.com/bytebase/omni/metadata"
	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/schema"
	"github.com/bytebase/bytebase/backend/store/model"
)

func TestDiffMigrationEventWithoutDefinition(t *testing.T) {
	databaseOf := func(events ...*metadatapb.EventMetadata) *model.DatabaseMetadata {
		return model.NewDatabaseMetadata(&metadatapb.DatabaseSchemaMetadata{
			Name:    "db",
			Schemas: []*metadatapb.SchemaMetadata{{Name: "", Events: events}},
		}, nil, &storepb.DatabaseConfig{}, storepb.Engine_OCEANBASE, true)
	}
	withEvent := databaseOf(&metadatapb.EventMetadata{Name: "e"})

	migration, err := schema.DiffMigration(storepb.Engine_OCEANBASE, databaseOf(), withEvent)
	require.NoError(t, err)
	require.Empty(t, migration)

	migration, err = schema.DiffMigration(storepb.Engine_OCEANBASE, withEvent, databaseOf())
	require.NoError(t, err)
	require.Equal(t, "DROP EVENT IF EXISTS `e`;\n\n", migration)
}
