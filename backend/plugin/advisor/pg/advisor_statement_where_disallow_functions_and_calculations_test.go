// Package pg is the advisor for postgres database.
package pg

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/component/sheet"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/store/model"
)

// TestSearchPathResolution proves the rule honors PG's search_path GUC for
// unqualified table refs. Codex P2 (#3104277513): hardcoding `public` makes
// deployments using a different search_path silently miss violations.
func TestSearchPathResolution(t *testing.T) {
	// Custom mock: indexed `tech_book(name)` lives ONLY in `app_schema`.
	// public is empty. SearchPath puts app_schema first.
	dbSchema := &storepb.DatabaseSchemaMetadata{
		Name:       "test",
		SearchPath: "app_schema, public",
		Schemas: []*storepb.SchemaMetadata{
			{Name: "public"},
			{
				Name: "app_schema",
				Tables: []*storepb.TableMetadata{
					{
						Name: "tech_book",
						Columns: []*storepb.ColumnMetadata{
							{Name: "id", Type: "integer"},
							{Name: "name", Type: "text"},
						},
						Indexes: []*storepb.IndexMetadata{
							{Name: "tech_book_name_idx", Expressions: []string{"name"}},
						},
					},
				},
			},
		},
	}

	stmt := "SELECT * FROM tech_book WHERE UPPER(name) = 'X';"
	rule := &storepb.SQLReviewRule{
		Type:  storepb.SQLReviewRule_STATEMENT_WHERE_DISALLOW_FUNCTIONS_AND_CALCULATIONS,
		Level: storepb.SQLReviewRule_WARNING,
	}
	originalMetadata := model.NewDatabaseMetadata(dbSchema, nil, nil, storepb.Engine_POSTGRES, true /* isCaseSensitive */)
	checkCtx := advisor.Context{
		DBType:                storepb.Engine_POSTGRES,
		OriginalMetadata:      originalMetadata,
		FinalMetadata:         originalMetadata,
		CurrentDatabase:       "test",
		DBSchema:              dbSchema,
		IsObjectCaseSensitive: true,
		NoAppendBuiltin:       true,
	}

	sm := sheet.NewManager()
	advice, err := advisor.SQLReviewCheck(context.Background(), sm, stmt, []*storepb.SQLReviewRule{rule}, checkCtx)
	require.NoError(t, err)
	require.Len(t, advice, 1, "expected exactly one advice for UPPER(name) on indexed col in app_schema (search_path)")
	require.Contains(t, advice[0].Content, `Function "UPPER" is applied to indexed column "name"`)
}

// TestSearchPathSessionUserResolution proves the rule resolves the `$user`
// search_path entry against checkCtx.SessionUser. Codex P2 (#3104477258):
// using parameterless GetSearchPath drops `$user`, so deployments with
// `search_path = "$user", public` miss tables in the session-user schema.
func TestSearchPathSessionUserResolution(t *testing.T) {
	dbSchema := &storepb.DatabaseSchemaMetadata{
		Name:       "test",
		SearchPath: `"$user", public`,
		Schemas: []*storepb.SchemaMetadata{
			{Name: "public"},
			{
				Name: "alice",
				Tables: []*storepb.TableMetadata{
					{
						Name: "tech_book",
						Columns: []*storepb.ColumnMetadata{
							{Name: "id", Type: "integer"},
							{Name: "name", Type: "text"},
						},
						Indexes: []*storepb.IndexMetadata{
							{Name: "tech_book_name_idx", Expressions: []string{"name"}},
						},
					},
				},
			},
		},
	}

	stmt := "SELECT * FROM tech_book WHERE UPPER(name) = 'X';"
	rule := &storepb.SQLReviewRule{
		Type:  storepb.SQLReviewRule_STATEMENT_WHERE_DISALLOW_FUNCTIONS_AND_CALCULATIONS,
		Level: storepb.SQLReviewRule_WARNING,
	}
	originalMetadata := model.NewDatabaseMetadata(dbSchema, nil, nil, storepb.Engine_POSTGRES, true)
	checkCtx := advisor.Context{
		DBType:                storepb.Engine_POSTGRES,
		OriginalMetadata:      originalMetadata,
		FinalMetadata:         originalMetadata,
		CurrentDatabase:       "test",
		DBSchema:              dbSchema,
		IsObjectCaseSensitive: true,
		SessionUser:           "alice",
		NoAppendBuiltin:       true,
	}

	sm := sheet.NewManager()
	advice, err := advisor.SQLReviewCheck(context.Background(), sm, stmt, []*storepb.SQLReviewRule{rule}, checkCtx)
	require.NoError(t, err)
	require.Len(t, advice, 1, "expected exactly one advice for UPPER(name) in $user-resolved schema (alice)")
	require.Contains(t, advice[0].Content, `Function "UPPER" is applied to indexed column "name"`)
}
