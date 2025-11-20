package pg

import (
	"testing"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/schema"
)

// TestCheckConstraintOrderStability tests that CHECK constraints maintain consistent order
// when generating SDL multiple times from the same metadata
func TestCheckConstraintOrderStability(t *testing.T) {
	t.Run("single_file", func(t *testing.T) {
		testCheckConstraintOrder(t, false)
	})
	t.Run("multi_file", func(t *testing.T) {
		testCheckConstraintOrder(t, true)
	})
}

func testCheckConstraintOrder(t *testing.T, multiFile bool) {
	// Simulate database metadata with multiple CHECK constraints
	// (similar to the packages_debian_group_distributions table shown in the screenshot)
	dbMetadata := &storepb.DatabaseSchemaMetadata{
		Name: "testdb",
		Schemas: []*storepb.SchemaMetadata{
			{
				Name: "public",
				Tables: []*storepb.TableMetadata{
					{
						Name: "packages_debian_group_distributions",
						Columns: []*storepb.ColumnMetadata{
							{Name: "id", Type: "bigint", Nullable: false},
							{Name: "suite", Type: "character varying(255)", Nullable: false},
							{Name: "codename", Type: "character varying(255)", Nullable: false},
							{Name: "signed_file", Type: "text", Nullable: true},
							{Name: "description", Type: "character varying(255)", Nullable: true},
							{Name: "origin", Type: "character varying(255)", Nullable: true},
							{Name: "file", Type: "character varying(255)", Nullable: true},
							{Name: "label", Type: "character varying(255)", Nullable: true},
							{Name: "version", Type: "character varying(255)", Nullable: true},
							{Name: "file_signature", Type: "character varying(4096)", Nullable: true},
						},
						CheckConstraints: []*storepb.CheckConstraintMetadata{
							{Name: "check_e7c928a24b", Expression: "(char_length((suite)::text) <= 255)"},
							{Name: "check_590e18405a", Expression: "(char_length((codename)::text) <= 255)"},
							{Name: "check_0007e0bf61", Expression: "(char_length(signed_file) <= 255)"},
							{Name: "check_310ac457b8", Expression: "(char_length((description)::text) <= 255)"},
							{Name: "check_3d6f87fc31", Expression: "(char_length((file_signature)::text) <= 4096)"},
							{Name: "check_3fdadf4a0c", Expression: "(char_length((version)::text) <= 255)"},
							{Name: "check_b057cd840a", Expression: "(char_length((origin)::text) <= 255)"},
							{Name: "check_be5ed8d307", Expression: "(char_length((file)::text) <= 255)"},
							{Name: "check_d3244bfc0b", Expression: "(char_length((label)::text) <= 255)"},
						},
					},
				},
			},
		},
	}

	ctx := schema.GetDefinitionContext{
		SkipBackupSchema: false,
		PrintHeader:      false,
		SDLFormat:        true,
		MultiFileFormat:  multiFile,
	}

	// Generate SDL first time
	sdl1, err := GetDatabaseDefinition(ctx, dbMetadata)
	require.NoError(t, err)
	require.NotEmpty(t, sdl1)

	t.Logf("First SDL generation:\n%s", sdl1)

	// Generate SDL second time from the same metadata
	sdl2, err := GetDatabaseDefinition(ctx, dbMetadata)
	require.NoError(t, err)
	require.NotEmpty(t, sdl2)

	t.Logf("Second SDL generation:\n%s", sdl2)

	// The two SDL outputs should be identical
	require.Equal(t, sdl1, sdl2, "SDL generation should be deterministic - CHECK constraint order should be stable")
}
