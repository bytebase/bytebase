// Package pg is the advisor for postgres database.
package pg

import (
	"context"
	"testing"

	metadatapb "github.com/bytebase/omni/metadata"
	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/component/sheet"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/store/model"
)

// TestWhereDisallowFunctionsSearchPathSeam proves the PG adapter passes the
// resolved search_path (including $user expansion via SessionUser) to omni's
// StatementWhereDisallowFunctionsAndCalculations.
func TestWhereDisallowFunctionsSearchPathSeam(t *testing.T) {
	rule := &storepb.SQLReviewRule{
		Type:   storepb.SQLReviewRule_STATEMENT_WHERE_DISALLOW_FUNCTIONS_AND_CALCULATIONS,
		Level:  storepb.SQLReviewRule_WARNING,
		Engine: storepb.Engine_POSTGRES,
	}
	for _, tc := range []struct {
		name        string
		searchPath  string
		schemaName  string
		sessionUser string
	}{
		{
			name:       "explicit search_path",
			searchPath: "app_schema, public",
			schemaName: "app_schema",
		},
		{
			name:        "session user expands $user entry",
			searchPath:  `"$user", public`,
			schemaName:  "alice",
			sessionUser: "alice",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dbSchema := &metadatapb.DatabaseSchemaMetadata{
				Name:       "test",
				SearchPath: tc.searchPath,
				Schemas: []*metadatapb.SchemaMetadata{
					{Name: "public"},
					{
						Name: tc.schemaName,
						Tables: []*metadatapb.TableMetadata{{
							Name:    "tech_book",
							Columns: []*metadatapb.ColumnMetadata{{Name: "id", Type: "integer"}, {Name: "name", Type: "text"}},
							Indexes: []*metadatapb.IndexMetadata{{Name: "tech_book_name_idx", Expressions: []string{"name"}}},
						}},
					},
				},
			}
			md := model.NewDatabaseMetadata(dbSchema, nil, nil, storepb.Engine_POSTGRES, true)
			advice, err := advisor.SQLReviewCheck(context.Background(), sheet.NewManager(),
				"SELECT * FROM tech_book WHERE UPPER(name) = 'X';",
				[]*storepb.SQLReviewRule{rule},
				advisor.Context{
					DBType:                storepb.Engine_POSTGRES,
					OriginalMetadata:      md,
					FinalMetadata:         md,
					CurrentDatabase:       "test",
					DBSchema:              dbSchema,
					IsObjectCaseSensitive: true,
					SessionUser:           tc.sessionUser,
					NoAppendBuiltin:       true,
				},
			)
			require.NoError(t, err)
			require.Len(t, advice, 1)
			require.Contains(t, advice[0].Content, `Function "UPPER" is applied to indexed column "name"`)
		})
	}
}
