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
	require.NotNil(t, final.GetSchema("public").GetTable("users"))

	// Mutate final
	_, err = final.GetSchema("public").CreateTable("products")
	require.Nil(t, err)

	// Verify origin is unchanged
	require.Nil(t, origin.GetSchema("public").GetTable("products"))

	// Verify final has the new table
	require.NotNil(t, final.GetSchema("public").GetTable("products"))
}
