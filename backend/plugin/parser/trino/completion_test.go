package trino

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/store/model"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

func TestComplete(t *testing.T) {
	testCases := []struct {
		name             string
		statement        string
		caretLine        int
		caretOffset      int
		expectedTypes    []base.CandidateType
		expectedTexts    []string
		notExpectedTexts []string
	}{
		{
			name:          "Empty statement",
			statement:     "",
			caretLine:     0,
			caretOffset:   0,
			expectedTypes: []base.CandidateType{base.CandidateTypeKeyword},
			expectedTexts: []string{
				"SELECT", "INSERT", "CREATE", "ALTER", "DROP",
			},
		},
		{
			name:          "After SELECT",
			statement:     "SELECT ",
			caretLine:     0,
			caretOffset:   7,
			expectedTypes: []base.CandidateType{base.CandidateTypeColumn},
		},
		{
			name:          "After FROM",
			statement:     "SELECT id FROM ",
			caretLine:     0,
			caretOffset:   15,
			expectedTypes: []base.CandidateType{base.CandidateTypeTable, base.CandidateTypeView},
		},
		{
			name:          "Starting with SEL",
			statement:     "SEL",
			caretLine:     0,
			caretOffset:   3,
			expectedTypes: []base.CandidateType{base.CandidateTypeKeyword},
			expectedTexts: []string{
				"SELECT",
			},
			notExpectedTexts: []string{
				"INSERT", "DELETE", "UPDATE",
			},
		},
		{
			name:          "After JOIN",
			statement:     "SELECT * FROM users JOIN ",
			caretLine:     0,
			caretOffset:   25,
			expectedTypes: []base.CandidateType{base.CandidateTypeTable, base.CandidateTypeView},
		},
		{
			name:          "After function name",
			statement:     "SELECT COUNT(",
			caretLine:     0,
			caretOffset:   13,
			expectedTypes: []base.CandidateType{base.CandidateTypeColumn, base.CandidateTypeFunction},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create mock metadata for the database schema
			schemaMetadata := &storepb.SchemaMetadata{
				Name: "public",
				Tables: []*storepb.TableMetadata{
					{
						Name: "users",
						Columns: []*storepb.ColumnMetadata{
							{Name: "id", Type: "INT"},
							{Name: "name", Type: "VARCHAR"},
							{Name: "email", Type: "VARCHAR"},
						},
					},
					{
						Name: "orders",
						Columns: []*storepb.ColumnMetadata{
							{Name: "id", Type: "INT"},
							{Name: "user_id", Type: "INT"},
							{Name: "total", Type: "DECIMAL"},
						},
					},
				},
				Views: []*storepb.ViewMetadata{
					{Name: "active_users"},
				},
			}

			// Create mock database schema metadata
			dbSchemaMetadata := &storepb.DatabaseSchemaMetadata{
				Name:    "test_db",
				Schemas: []*storepb.SchemaMetadata{schemaMetadata},
			}

			// Create the DatabaseSchema object
			dbSchema := model.NewDatabaseSchema(dbSchemaMetadata, nil, nil, storepb.Engine_TRINO, false)

			// Set up mock functions for completion context
			mockMetadataFunc := func(_ context.Context, _ string, _ string) (string, *model.DatabaseMetadata, error) {
				return "test_db", dbSchema.GetDatabaseMetadata(), nil
			}

			mockListDBNamesFunc := func(_ context.Context, _ string) ([]string, error) {
				return []string{"public", "analytics"}, nil
			}

			ctx := context.Background()
			cCtx := base.CompletionContext{
				InstanceID:        "test-instance",
				DefaultDatabase:   "test_db",
				DefaultSchema:     "public",
				Metadata:          mockMetadataFunc,
				ListDatabaseNames: mockListDBNamesFunc,
			}

			// Call the completion function
			candidates, err := Complete(ctx, cCtx, tc.statement, tc.caretLine, tc.caretOffset)
			require.NoError(t, err, "Complete should not error")

			// Verify results based on expected candidate types
			if len(tc.expectedTypes) > 0 {
				var foundTypes []base.CandidateType
				for _, candidate := range candidates {
					foundTypes = append(foundTypes, candidate.Type)
				}

				// Check that at least one candidate of each expected type exists
				for _, expectedType := range tc.expectedTypes {
					found := false
					for _, typ := range foundTypes {
						if typ == expectedType {
							found = true
							break
						}
					}
					assert.True(t, found, "Expected to find candidate of type %v", expectedType)
				}
			}

			// Check for expected texts
			for _, expectedText := range tc.expectedTexts {
				found := false
				for _, candidate := range candidates {
					if candidate.Text == expectedText {
						found = true
						break
					}
				}
				assert.True(t, found, "Expected to find candidate with text '%s'", expectedText)
			}

			// Check for texts that should not be present
			for _, notExpectedText := range tc.notExpectedTexts {
				found := false
				for _, candidate := range candidates {
					if candidate.Text == notExpectedText {
						found = true
						break
					}
				}
				assert.False(t, found, "Did not expect to find candidate with text '%s'", notExpectedText)
			}
		})
	}
}
