package pg

import (
	"testing"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/schema"
	"github.com/bytebase/bytebase/backend/store/model"
)

// TestMaterializedViewIndexInSDLOutput tests that materialized view indexes are included in SDL output
func TestMaterializedViewIndexInSDLOutput(t *testing.T) {
	metadata := &storepb.DatabaseSchemaMetadata{
		Schemas: []*storepb.SchemaMetadata{
			{
				Name: "public",
				MaterializedViews: []*storepb.MaterializedViewMetadata{
					{
						Name:       "product_summary_mv",
						Definition: "SELECT id, name FROM products",
						Comment:    "Product summary view",
						Indexes: []*storepb.IndexMetadata{
							{
								Name:         "idx_product_name",
								Expressions:  []string{"name"},
								Type:         "btree",
								Unique:       false,
								Primary:      false,
								IsConstraint: false,
								Comment:      "Index on product name",
							},
							{
								Name:         "idx_product_id_unique",
								Expressions:  []string{"id"},
								Type:         "btree",
								Unique:       true,
								Primary:      false,
								IsConstraint: false,
							},
						},
					},
				},
			},
		},
	}

	// Test single-file SDL format
	sdl, err := getSDLFormat(metadata)
	require.NoError(t, err)

	// Verify materialized view is present
	require.Contains(t, sdl, "CREATE MATERIALIZED VIEW")
	require.Contains(t, sdl, "product_summary_mv")

	// Verify indexes are present
	require.Contains(t, sdl, "CREATE INDEX \"idx_product_name\"")
	require.Contains(t, sdl, "CREATE UNIQUE INDEX \"idx_product_id_unique\"")

	// Verify index is on materialized view (no ON ONLY for non-partitioned)
	require.Contains(t, sdl, "ON \"public\".\"product_summary_mv\"")
	require.NotContains(t, sdl, "ON ONLY")

	// Verify comments
	require.Contains(t, sdl, "COMMENT ON MATERIALIZED VIEW")
	require.Contains(t, sdl, "Product summary view")
	require.Contains(t, sdl, "COMMENT ON INDEX")
	require.Contains(t, sdl, "Index on product name")
}

// TestMaterializedViewIndexInMultiFileSDL tests that materialized view indexes are included in multi-file SDL
func TestMaterializedViewIndexInMultiFileSDL(t *testing.T) {
	metadata := &storepb.DatabaseSchemaMetadata{
		Schemas: []*storepb.SchemaMetadata{
			{
				Name: "public",
				MaterializedViews: []*storepb.MaterializedViewMetadata{
					{
						Name:       "user_stats_mv",
						Definition: "SELECT user_id, COUNT(*) as count FROM orders GROUP BY user_id",
						Indexes: []*storepb.IndexMetadata{
							{
								Name:         "idx_user_stats_user_id",
								Expressions:  []string{"user_id"},
								Type:         "btree",
								Unique:       false,
								Primary:      false,
								IsConstraint: false,
							},
						},
					},
				},
			},
		},
	}

	// Test multi-file SDL format
	result, err := GetMultiFileDatabaseDefinition(schema.GetDefinitionContext{}, metadata)
	require.NoError(t, err)
	require.NotNil(t, result)

	// Find the materialized view file
	var mvFile *schema.File
	for i := range result.Files {
		if result.Files[i].Name == "schemas/public/materialized_views/user_stats_mv.sql" {
			mvFile = &result.Files[i]
			break
		}
	}
	require.NotNil(t, mvFile, "Materialized view file not found")

	// Verify content includes both MV and index
	require.Contains(t, mvFile.Content, "CREATE MATERIALIZED VIEW")
	require.Contains(t, mvFile.Content, "user_stats_mv")
	require.Contains(t, mvFile.Content, "CREATE INDEX \"idx_user_stats_user_id\"")
	require.Contains(t, mvFile.Content, "ON \"public\".\"user_stats_mv\"")
	require.NotContains(t, mvFile.Content, "ON ONLY")
}

// TestMaterializedViewIndexDependencyOrdering tests that materialized view indexes have correct dependency ordering in migrations
func TestMaterializedViewIndexDependencyOrdering(t *testing.T) {
	t.Run("CREATE: MV before Index", func(t *testing.T) {
		// Test that when creating a materialized view with indexes,
		// CREATE MATERIALIZED VIEW comes before CREATE INDEX

		oldMetadata := &storepb.DatabaseSchemaMetadata{
			Schemas: []*storepb.SchemaMetadata{
				{
					Name: "public",
				},
			},
		}

		newMetadata := &storepb.DatabaseSchemaMetadata{
			Schemas: []*storepb.SchemaMetadata{
				{
					Name: "public",
					MaterializedViews: []*storepb.MaterializedViewMetadata{
						{
							Name:       "customer_segmentation_mv",
							Definition: "SELECT customer_id, SUM(amount) as total_spending FROM orders GROUP BY customer_id",
							Indexes: []*storepb.IndexMetadata{
								{
									Name:        "idx_customer_seg_spending",
									Expressions: []string{"total_spending"},
									Type:        "btree",
									Unique:      false,
									Primary:     false,
								},
							},
						},
					},
				},
			},
		}

		// Convert to model.DatabaseSchema
		oldSchema := model.NewDatabaseMetadata(oldMetadata, nil, nil, storepb.Engine_POSTGRES, false)
		newSchema := model.NewDatabaseMetadata(newMetadata, nil, nil, storepb.Engine_POSTGRES, false)

		// Get diff
		diff, err := schema.GetDatabaseSchemaDiff(storepb.Engine_POSTGRES, oldSchema, newSchema)
		require.NoError(t, err)

		// Generate migration
		migration, err := schema.GenerateMigration(storepb.Engine_POSTGRES, diff)
		require.NoError(t, err)
		t.Logf("Generated migration:\n%s", migration)
		require.NotEmpty(t, migration)

		// Find positions of CREATE MATERIALIZED VIEW and CREATE INDEX
		mvPos := -1
		idxPos := -1
		for i := 0; i < len(migration); i++ {
			if mvPos == -1 && i+len("CREATE MATERIALIZED VIEW") <= len(migration) {
				if migration[i:i+len("CREATE MATERIALIZED VIEW")] == "CREATE MATERIALIZED VIEW" {
					mvPos = i
				}
			}
			if idxPos == -1 && i+len("CREATE INDEX") <= len(migration) {
				if migration[i:i+len("CREATE INDEX")] == "CREATE INDEX" {
					idxPos = i
				}
			}
		}

		require.NotEqual(t, -1, mvPos, "CREATE MATERIALIZED VIEW not found in migration")
		require.NotEqual(t, -1, idxPos, "CREATE INDEX not found in migration")
		require.Less(t, mvPos, idxPos, "CREATE MATERIALIZED VIEW should come before CREATE INDEX")
	})

	t.Run("DROP: Whole MV with indexes", func(t *testing.T) {
		// Test that when dropping an entire materialized view with indexes,
		// PostgreSQL automatically drops indexes so we don't need separate DROP INDEX statements

		oldMetadata := &storepb.DatabaseSchemaMetadata{
			Schemas: []*storepb.SchemaMetadata{
				{
					Name: "public",
					MaterializedViews: []*storepb.MaterializedViewMetadata{
						{
							Name:       "customer_segmentation_mv",
							Definition: "SELECT customer_id, SUM(amount) as total_spending FROM orders GROUP BY customer_id",
							Indexes: []*storepb.IndexMetadata{
								{
									Name:        "idx_customer_seg_spending",
									Expressions: []string{"total_spending"},
									Type:        "btree",
									Unique:      false,
									Primary:     false,
								},
							},
						},
					},
				},
			},
		}

		newMetadata := &storepb.DatabaseSchemaMetadata{
			Schemas: []*storepb.SchemaMetadata{
				{
					Name: "public",
				},
			},
		}

		// Convert to model.DatabaseSchema
		oldSchema := model.NewDatabaseMetadata(oldMetadata, nil, nil, storepb.Engine_POSTGRES, false)
		newSchema := model.NewDatabaseMetadata(newMetadata, nil, nil, storepb.Engine_POSTGRES, false)

		// Get diff
		diff, err := schema.GetDatabaseSchemaDiff(storepb.Engine_POSTGRES, oldSchema, newSchema)
		require.NoError(t, err)

		// Generate migration
		migration, err := schema.GenerateMigration(storepb.Engine_POSTGRES, diff)
		require.NoError(t, err)
		t.Logf("Generated migration:\n%s", migration)
		require.NotEmpty(t, migration)

		// Verify DROP MATERIALIZED VIEW is present
		require.Contains(t, migration, "DROP MATERIALIZED VIEW")
		// Note: PostgreSQL automatically drops indexes when dropping a materialized view,
		// so we don't need (and shouldn't have) separate DROP INDEX statements
		require.NotContains(t, migration, "DROP INDEX")
	})
}
