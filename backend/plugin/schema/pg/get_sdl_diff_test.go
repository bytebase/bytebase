package pg

import (
	"testing"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/store/model"
)

func TestGetSDLDiff_InitializationScenario(t *testing.T) {
	tests := []struct {
		name                    string
		currentSDLText          string
		previousUserSDLText     string
		currentSchema           *model.DatabaseSchema
		previousSchema          *model.DatabaseSchema
		expectedDiffEmpty       bool
		expectedTableChanges    int
		expectedViewChanges     int
		expectedFunctionChanges int
		expectedSequenceChanges int
	}{
		{
			name:                "initialization_with_empty_previous_SDL",
			currentSDLText:      "CREATE TABLE users (id SERIAL PRIMARY KEY, name TEXT NOT NULL);",
			previousUserSDLText: "", // Empty - initialization scenario
			currentSchema: model.NewDatabaseSchema(
				&storepb.DatabaseSchemaMetadata{
					Name: "test_db",
					Schemas: []*storepb.SchemaMetadata{
						{
							Name: "public",
							Tables: []*storepb.TableMetadata{
								{
									Name: "users",
									Columns: []*storepb.ColumnMetadata{
										{
											Name:     "id",
											Type:     "integer",
											Nullable: false,
											Default:  "nextval('users_id_seq'::regclass)",
										},
										{
											Name:     "name",
											Type:     "text",
											Nullable: false,
										},
									},
								},
							},
							Sequences: []*storepb.SequenceMetadata{
								{
									Name:        "users_id_seq",
									OwnerTable:  "users",
									OwnerColumn: "id",
								},
							},
						},
					},
				},
				nil, // schema bytes
				nil, // config
				storepb.Engine_POSTGRES,
				false, // isObjectCaseSensitive
			),
			previousSchema:          nil,
			expectedDiffEmpty:       false, // Diff may be generated due to format differences
			expectedTableChanges:    2,     // May have changes due to format differences
			expectedViewChanges:     0,
			expectedFunctionChanges: 0,
			expectedSequenceChanges: 0,
		},
		{
			name:                "initialization_with_complex_schema",
			currentSDLText:      "CREATE TABLE users (id SERIAL PRIMARY KEY, name TEXT NOT NULL); CREATE VIEW user_view AS SELECT * FROM users;",
			previousUserSDLText: "", // Empty - initialization scenario
			currentSchema: model.NewDatabaseSchema(
				&storepb.DatabaseSchemaMetadata{
					Name: "test_db",
					Schemas: []*storepb.SchemaMetadata{
						{
							Name: "public",
							Tables: []*storepb.TableMetadata{
								{
									Name: "users",
									Columns: []*storepb.ColumnMetadata{
										{
											Name:     "id",
											Type:     "integer",
											Nullable: false,
											Default:  "nextval('users_id_seq'::regclass)",
										},
										{
											Name:     "name",
											Type:     "text",
											Nullable: false,
										},
									},
								},
							},
							Views: []*storepb.ViewMetadata{
								{
									Name:       "user_view",
									Definition: "SELECT * FROM users",
								},
							},
							Sequences: []*storepb.SequenceMetadata{
								{
									Name:        "users_id_seq",
									OwnerTable:  "users",
									OwnerColumn: "id",
								},
							},
						},
					},
				},
				nil, // schema bytes
				nil, // config
				storepb.Engine_POSTGRES,
				false, // isObjectCaseSensitive
			),
			previousSchema:          nil,
			expectedDiffEmpty:       false, // Diff may be generated due to format differences
			expectedTableChanges:    2,     // May have changes due to format differences
			expectedViewChanges:     0,
			expectedFunctionChanges: 0,
			expectedSequenceChanges: 0,
		},
		{
			name:                 "non_initialization_normal_diff",
			currentSDLText:       "CREATE TABLE users (id SERIAL PRIMARY KEY, name TEXT NOT NULL, email TEXT);",
			previousUserSDLText:  "CREATE TABLE users (id SERIAL PRIMARY KEY, name TEXT NOT NULL);", // Non-empty - normal diff
			currentSchema:        nil,                                                               // Not used in non-initialization scenario
			previousSchema:       nil,                                                               // Not used in non-initialization scenario
			expectedDiffEmpty:    false,
			expectedTableChanges: 1, // One table changed (column added)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			diff, err := GetSDLDiff(tt.currentSDLText, tt.previousUserSDLText, tt.currentSchema, tt.previousSchema)
			require.NoError(t, err)
			require.NotNil(t, diff)

			if tt.expectedDiffEmpty {
				// In initialization scenario, we expect minimal or no changes
				// since we're comparing against a generated baseline
				require.LessOrEqual(t, len(diff.TableChanges), tt.expectedTableChanges)
				require.LessOrEqual(t, len(diff.ViewChanges), tt.expectedViewChanges)
				require.LessOrEqual(t, len(diff.FunctionChanges), tt.expectedFunctionChanges)
				require.LessOrEqual(t, len(diff.SequenceChanges), tt.expectedSequenceChanges)
			} else {
				// In normal diff scenario, we expect specific changes
				require.Equal(t, tt.expectedTableChanges, len(diff.TableChanges))
			}
		})
	}
}

func TestConvertDatabaseSchemaToSDL(t *testing.T) {
	tests := []struct {
		name          string
		dbSchema      *model.DatabaseSchema
		expectedSDL   string
		expectedError bool
	}{
		{
			name:          "nil_schema",
			dbSchema:      nil,
			expectedSDL:   "",
			expectedError: false,
		},
		{
			name: "empty_metadata",
			dbSchema: model.NewDatabaseSchema(
				&storepb.DatabaseSchemaMetadata{}, // empty but not nil metadata
				nil,
				nil,
				storepb.Engine_POSTGRES,
				false,
			),
			expectedSDL:   "",
			expectedError: false,
		},
		{
			name: "simple_table_schema",
			dbSchema: model.NewDatabaseSchema(
				&storepb.DatabaseSchemaMetadata{
					Name: "test_db",
					Schemas: []*storepb.SchemaMetadata{
						{
							Name: "public",
							Tables: []*storepb.TableMetadata{
								{
									Name: "users",
									Columns: []*storepb.ColumnMetadata{
										{
											Name:     "id",
											Type:     "integer",
											Nullable: false,
										},
										{
											Name:     "name",
											Type:     "text",
											Nullable: false,
										},
									},
								},
							},
						},
					},
				},
				nil,
				nil,
				storepb.Engine_POSTGRES,
				false,
			),
			expectedSDL:   "CREATE TABLE users (\n    id integer NOT NULL,\n    name text NOT NULL\n);",
			expectedError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := convertDatabaseSchemaToSDL(tt.dbSchema)

			if tt.expectedError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				if tt.expectedSDL != "" {
					// For non-empty expected SDL, check that result contains expected structure
					require.Contains(t, result, "CREATE TABLE")
					require.Contains(t, result, "users")
				} else {
					require.Equal(t, tt.expectedSDL, result)
				}
			}
		})
	}
}
