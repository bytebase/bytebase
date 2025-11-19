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

	// Verify index is on materialized view (using ON ONLY for SDL)
	require.Contains(t, sdl, "ON ONLY \"public\".\"product_summary_mv\"")

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
	require.Contains(t, mvFile.Content, "ON ONLY \"public\".\"user_stats_mv\"")
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

	t.Run("ADD Index to existing MV via SDL chunks", func(t *testing.T) {
		// This tests the exact scenario from the user's screenshot:
		// Adding an index to an existing materialized view should generate
		// CREATE INDEX statement without trying to recreate the MV

		// Previous SDL (MV without index)
		previousSDL := `CREATE MATERIALIZED VIEW "public"."customer_segmentation_mv" AS
		SELECT customer_id, SUM(amount) as total_spending
		FROM orders
		GROUP BY customer_id;`

		// Current SDL (MV with new index)
		currentSDL := `CREATE MATERIALIZED VIEW "public"."customer_segmentation_mv" AS
		SELECT customer_id, SUM(amount) as total_spending
		FROM orders
		GROUP BY customer_id;

		CREATE INDEX "idx_customer_seg_spending" ON "public"."customer_segmentation_mv" (total_spending);`

		// Get SDL diff (simulates bb rollout behavior)
		diff, err := GetSDLDiff(currentSDL, previousSDL, nil, nil)
		require.NoError(t, err)
		require.NotNil(t, diff)

		// Debug: log all diffs
		t.Logf("MaterializedViewChanges: %d", len(diff.MaterializedViewChanges))
		for i, mvDiff := range diff.MaterializedViewChanges {
			t.Logf("  MV[%d]: Action=%v, Schema=%s, Name=%s, IndexChanges=%d, HasNewMV=%v, HasNewAST=%v",
				i, mvDiff.Action, mvDiff.SchemaName, mvDiff.MaterializedViewName,
				len(mvDiff.IndexChanges), mvDiff.NewMaterializedView != nil, mvDiff.NewASTNode != nil)
		}

		// Verify we have a MaterializedViewDiff with IndexChanges
		require.Equal(t, 1, len(diff.MaterializedViewChanges), "Should have 1 MV change")
		mvDiff := diff.MaterializedViewChanges[0]
		require.Equal(t, schema.MetadataDiffActionAlter, mvDiff.Action)
		require.Equal(t, "public", mvDiff.SchemaName)
		require.Equal(t, "customer_segmentation_mv", mvDiff.MaterializedViewName)
		require.Equal(t, 1, len(mvDiff.IndexChanges), "Should have 1 index change")
		require.Equal(t, schema.MetadataDiffActionCreate, mvDiff.IndexChanges[0].Action)

		// Generate migration
		migration, err := schema.GenerateMigration(storepb.Engine_POSTGRES, diff)
		require.NoError(t, err)
		t.Logf("Generated migration:\n%s", migration)

		// Verify migration only creates the index, doesn't recreate the MV
		require.Contains(t, migration, "CREATE INDEX")
		require.Contains(t, migration, "idx_customer_seg_spending")
		require.NotContains(t, migration, "DROP MATERIALIZED VIEW", "Should not drop MV for index-only changes")
		require.NotContains(t, migration, "CREATE MATERIALIZED VIEW", "Should not recreate MV for index-only changes")
	})
}
