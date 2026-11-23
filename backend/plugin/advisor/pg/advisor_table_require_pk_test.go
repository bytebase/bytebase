package pg

import (
	"context"
	"testing"

	metadatapb "github.com/bytebase/omni/metadata"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	"github.com/bytebase/bytebase/backend/component/sheet"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	"github.com/bytebase/bytebase/backend/store/model"
)

// TestTableRequirePKSearchPath proves Bytebase's walkthrough flows search_path
// resolution into TABLE_REQUIRE_PK across configured search_path, in-batch SET
// search_path, and in-batch drops. Pure TableRequirePK multi-schema decision
// logic lives in omni/pg/v1review.
func TestTableRequirePKSearchPath(t *testing.T) {
	dbSchema := &metadatapb.DatabaseSchemaMetadata{
		Name:       "test",
		SearchPath: "app_schema, public",
		Schemas: []*metadatapb.SchemaMetadata{
			{
				Name: "public",
				Tables: []*metadatapb.TableMetadata{
					{
						Name:    "shadow",
						Columns: []*metadatapb.ColumnMetadata{{Name: "id", Type: "integer"}},
						Indexes: []*metadatapb.IndexMetadata{
							{Name: "shadow_pkey", Expressions: []string{"id"}, Unique: true, Primary: true},
						},
					},
				},
			},
			{
				Name: "app_schema",
				Tables: []*metadatapb.TableMetadata{
					{
						Name:    "shadow",
						Columns: []*metadatapb.ColumnMetadata{{Name: "id", Type: "integer"}},
					},
					{
						Name:    "with_pk",
						Columns: []*metadatapb.ColumnMetadata{{Name: "id", Type: "integer"}},
						Indexes: []*metadatapb.IndexMetadata{
							{Name: "with_pk_pkey", Expressions: []string{"id"}, Unique: true, Primary: true},
						},
					},
				},
			},
		},
	}

	tests := []struct {
		name        string
		searchPath  string
		stmt        string
		wantContent string
	}{
		{
			name:        "configured search_path resolves unqualified drop PK",
			stmt:        "ALTER TABLE with_pk DROP CONSTRAINT with_pk_pkey;",
			wantContent: `Table "app_schema"."with_pk" requires PRIMARY KEY`,
		},
		{
			name:        "in-batch SET search_path redirects unqualified name",
			searchPath:  "public",
			stmt:        "SET search_path TO app_schema, public; ALTER TABLE with_pk DROP CONSTRAINT with_pk_pkey;",
			wantContent: `Table "app_schema"."with_pk" requires PRIMARY KEY`,
		},
		{
			name:        "in-batch drop redirects unqualified ALTER to remaining schema",
			stmt:        "DROP TABLE app_schema.shadow; ALTER TABLE shadow DROP CONSTRAINT shadow_pkey;",
			wantContent: `Table "public"."shadow" requires PRIMARY KEY`,
		},
	}

	rule := &storepb.SQLReviewRule{
		Type:   storepb.SQLReviewRule_TABLE_REQUIRE_PK,
		Level:  storepb.SQLReviewRule_WARNING,
		Engine: storepb.Engine_POSTGRES,
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			original, ok := proto.Clone(dbSchema).(*metadatapb.DatabaseSchemaMetadata)
			require.True(t, ok)
			if tc.searchPath != "" {
				original.SearchPath = tc.searchPath
			}
			final, ok := proto.Clone(original).(*metadatapb.DatabaseSchemaMetadata)
			require.True(t, ok)
			checkCtx := advisor.Context{
				DBType:           storepb.Engine_POSTGRES,
				OriginalMetadata: model.NewDatabaseMetadata(original, nil, nil, storepb.Engine_POSTGRES, true),
				FinalMetadata:    model.NewDatabaseMetadata(final, nil, nil, storepb.Engine_POSTGRES, true),
				CurrentDatabase:  "test",
				DBSchema:         dbSchema,
				NoAppendBuiltin:  true,
			}

			advice, err := advisor.SQLReviewCheck(context.Background(), sheet.NewManager(), tc.stmt, []*storepb.SQLReviewRule{rule}, checkCtx)
			require.NoError(t, err)

			var got []*storepb.Advice
			for _, a := range advice {
				if a.Code == code.TableNoPK.Int32() {
					got = append(got, a)
				}
			}
			if tc.wantContent == "" {
				require.Empty(t, got)
				return
			}
			require.Len(t, got, 1)
			require.Contains(t, got[0].Content, tc.wantContent)
		})
	}
}
