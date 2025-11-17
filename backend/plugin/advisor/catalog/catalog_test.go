package catalog

import (
	"testing"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

func TestNewCatalogWithMetadata(t *testing.T) {
	metadata := &storepb.DatabaseSchemaMetadata{
		Name: "testdb",
		Schemas: []*storepb.SchemaMetadata{
			{
				Name: "public",
				Tables: []*storepb.TableMetadata{
					{Name: "users"},
				},
			},
		},
	}

	origin, final, err := NewCatalogWithMetadata(metadata, storepb.Engine_POSTGRES, true)

	require.Nil(t, err)
	require.NotNil(t, origin)
	require.NotNil(t, final)

	// Verify both have the table initially
	require.NotNil(t, origin.GetSchema("public").GetTable("users"))
	finalSchema, err := final.GetSchema("public")
	require.Nil(t, err)
	finalTable, err := finalSchema.GetTable("users")
	require.Nil(t, err)
	require.NotNil(t, finalTable)

	// Mutate final - DatabaseState CreateTable returns (*TableState, *WalkThroughError)
	_, walkErr := finalSchema.CreateTable("products")
	require.Nil(t, walkErr)

	// Verify origin is unchanged (DatabaseMetadata)
	require.Nil(t, origin.GetSchema("public").GetTable("products"))

	// Verify final has the new table (DatabaseState)
	newTable, walkErr := finalSchema.GetTable("products")
	require.Nil(t, walkErr)
	require.NotNil(t, newTable)
}
