package pg

import (
	"testing"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/schema"
	"github.com/bytebase/bytebase/backend/store/model"
)

func TestQuoteIndexExpression(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"userId", `"userId"`},
		{"accountName", `"accountName"`},
		{"user_id", "user_id"},
		{"id", "id"},
		{`"userId"`, `"userId"`},
		{"lower(name)", "lower(name)"},
		{"(a + b)", "(a + b)"},
		{"col::text", "col::text"},
		{"", ""},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			require.Equal(t, tt.expected, quoteIndexExpression(tt.input))
		})
	}
}

// TestCamelCaseIndexColumnQuoting_SDL reproduces https://github.com/bytebase/bytebase/issues/19348
// via the SDL diff path (AST-based).
func TestCamelCaseIndexColumnQuoting_SDL(t *testing.T) {
	previousSDL := ""

	currentSDL := `
CREATE TABLE "public"."ba_account" (
    "userId" text NOT NULL,
    "accountName" text
);

CREATE INDEX ba_account_userId_idx ON "public"."ba_account" ("userId");
`

	currentSchema := model.NewDatabaseMetadata(
		&storepb.DatabaseSchemaMetadata{
			Name:    "testdb",
			Schemas: []*storepb.SchemaMetadata{},
		}, nil, nil, storepb.Engine_POSTGRES, false)

	diff, err := GetSDLDiff(currentSDL, previousSDL, currentSchema)
	require.NoError(t, err)
	require.NotNil(t, diff)

	ddl, err := generateMigration(diff)
	require.NoError(t, err)

	t.Logf("Generated DDL:\n%s", ddl)

	require.Contains(t, ddl, `("userId")`, "CamelCase column in CREATE INDEX must be quoted")
}

// TestCamelCaseIndexColumnQuoting_SyncSchema reproduces https://github.com/bytebase/bytebase/issues/19348
// via the Sync Schema path: the source schema is exported (with proper quoting), then
// parsed into metadata by GetDatabaseMetadata (which calls generateIndexDefinition).
// The parsed metadata's Definition must preserve quoting for CamelCase columns.
func TestCamelCaseIndexColumnQuoting_SyncSchema(t *testing.T) {
	// This is the exported schema from the source database (pg_get_indexdef quotes correctly).
	sourceSchemaSQL := `
CREATE TABLE "public"."ba_account" (
    "userId" text NOT NULL,
    "accountName" text
);

CREATE INDEX "ba_account_userid_idx" ON "public"."ba_account" ("userId");
`

	// Parse the source schema into metadata (this is what getTargetDBMetadata does).
	parsedMetadata, err := GetDatabaseMetadata(sourceSchemaSQL)
	require.NoError(t, err)

	// Verify the parsed index Definition has quoted "userId".
	var indexDef string
	for _, s := range parsedMetadata.Schemas {
		for _, tbl := range s.Tables {
			for _, idx := range tbl.Indexes {
				if idx.Name == "ba_account_userid_idx" {
					indexDef = idx.Definition
				}
			}
		}
	}
	require.NotEmpty(t, indexDef, "index definition should be set")
	t.Logf("Parsed index Definition: %s", indexDef)
	require.Contains(t, indexDef, `"userId"`, "CamelCase column must be quoted in parsed Definition")

	// Now simulate the full Sync Schema diff: target has no index, source has it.
	oldMetadata := &storepb.DatabaseSchemaMetadata{
		Name: "testdb",
		Schemas: []*storepb.SchemaMetadata{
			{
				Name: "public",
				Tables: []*storepb.TableMetadata{
					{
						Name: "ba_account",
						Columns: []*storepb.ColumnMetadata{
							{Name: "userId", Type: "text"},
							{Name: "accountName", Type: "text"},
						},
					},
				},
			},
		},
	}

	oldSchema := model.NewDatabaseMetadata(oldMetadata, nil, nil, storepb.Engine_POSTGRES, false)
	newSchema := model.NewDatabaseMetadata(parsedMetadata, nil, nil, storepb.Engine_POSTGRES, false)

	diff, err := schema.GetDatabaseSchemaDiff(storepb.Engine_POSTGRES, oldSchema, newSchema)
	require.NoError(t, err)
	require.NotNil(t, diff)

	ddl, err := generateMigration(diff)
	require.NoError(t, err)

	t.Logf("Generated DDL:\n%s", ddl)

	require.Contains(t, ddl, `"userId"`, "CamelCase column in CREATE INDEX must be quoted")
}

// TestCamelCaseConstraintQuoting_PrimaryKey verifies that PRIMARY KEY constraints
// on CamelCase columns are properly quoted in generated ALTER TABLE statements.
func TestCamelCaseConstraintQuoting_PrimaryKey(t *testing.T) {
	// Old state: table without primary key
	oldMetadata := &storepb.DatabaseSchemaMetadata{
		Name: "testdb",
		Schemas: []*storepb.SchemaMetadata{
			{
				Name: "public",
				Tables: []*storepb.TableMetadata{
					{
						Name: "ba_account",
						Columns: []*storepb.ColumnMetadata{
							{Name: "userId", Type: "text"},
						},
					},
				},
			},
		},
	}

	// New state: table with primary key on CamelCase column
	newMetadata := &storepb.DatabaseSchemaMetadata{
		Name: "testdb",
		Schemas: []*storepb.SchemaMetadata{
			{
				Name: "public",
				Tables: []*storepb.TableMetadata{
					{
						Name: "ba_account",
						Columns: []*storepb.ColumnMetadata{
							{Name: "userId", Type: "text"},
						},
						Indexes: []*storepb.IndexMetadata{
							{
								Name:         "ba_account_pkey",
								Expressions:  []string{"userId"},
								Type:         "btree",
								Primary:      true,
								Unique:       true,
								IsConstraint: true,
							},
						},
					},
				},
			},
		},
	}

	oldSchema := model.NewDatabaseMetadata(oldMetadata, nil, nil, storepb.Engine_POSTGRES, false)
	newSchema := model.NewDatabaseMetadata(newMetadata, nil, nil, storepb.Engine_POSTGRES, false)

	diff, err := schema.GetDatabaseSchemaDiff(storepb.Engine_POSTGRES, oldSchema, newSchema)
	require.NoError(t, err)
	require.NotNil(t, diff)

	ddl, err := generateMigration(diff)
	require.NoError(t, err)

	t.Logf("Generated DDL:\n%s", ddl)

	require.Contains(t, ddl, `PRIMARY KEY ("userId")`, "CamelCase column in PRIMARY KEY must be quoted")
}

// TestCamelCaseConstraintQuoting_Unique verifies that UNIQUE constraints
// on CamelCase columns are properly quoted in generated ALTER TABLE statements.
func TestCamelCaseConstraintQuoting_Unique(t *testing.T) {
	// Old state: table without unique constraint
	oldMetadata := &storepb.DatabaseSchemaMetadata{
		Name: "testdb",
		Schemas: []*storepb.SchemaMetadata{
			{
				Name: "public",
				Tables: []*storepb.TableMetadata{
					{
						Name: "ba_account",
						Columns: []*storepb.ColumnMetadata{
							{Name: "userId", Type: "text"},
						},
					},
				},
			},
		},
	}

	// New state: table with unique constraint on CamelCase column
	newMetadata := &storepb.DatabaseSchemaMetadata{
		Name: "testdb",
		Schemas: []*storepb.SchemaMetadata{
			{
				Name: "public",
				Tables: []*storepb.TableMetadata{
					{
						Name: "ba_account",
						Columns: []*storepb.ColumnMetadata{
							{Name: "userId", Type: "text"},
						},
						Indexes: []*storepb.IndexMetadata{
							{
								Name:         "ba_account_userId_key",
								Expressions:  []string{"userId"},
								Type:         "btree",
								Unique:       true,
								IsConstraint: true,
							},
						},
					},
				},
			},
		},
	}

	oldSchema := model.NewDatabaseMetadata(oldMetadata, nil, nil, storepb.Engine_POSTGRES, false)
	newSchema := model.NewDatabaseMetadata(newMetadata, nil, nil, storepb.Engine_POSTGRES, false)

	diff, err := schema.GetDatabaseSchemaDiff(storepb.Engine_POSTGRES, oldSchema, newSchema)
	require.NoError(t, err)
	require.NotNil(t, diff)

	ddl, err := generateMigration(diff)
	require.NoError(t, err)

	t.Logf("Generated DDL:\n%s", ddl)

	require.Contains(t, ddl, `UNIQUE ("userId")`, "CamelCase column in UNIQUE constraint must be quoted")
}
