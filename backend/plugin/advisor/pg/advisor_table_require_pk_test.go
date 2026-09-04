package pg

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	"github.com/bytebase/bytebase/backend/component/sheet"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	"github.com/bytebase/bytebase/backend/store/model"
)

// TestTableRequirePKSearchPath proves unqualified names resolve through the
// database search_path, matching the walkthrough that builds FinalMetadata.
// Both tables live only in app_schema; public is empty.
func TestTableRequirePKSearchPath(t *testing.T) {
	dbSchema := &storepb.DatabaseSchemaMetadata{
		Name:       "test",
		SearchPath: "app_schema, public",
		Schemas: []*storepb.SchemaMetadata{
			{Name: "public"},
			{
				Name: "app_schema",
				Tables: []*storepb.TableMetadata{
					{
						Name: "with_pk",
						Columns: []*storepb.ColumnMetadata{
							{Name: "id", Type: "integer"},
							{Name: "name", Type: "text"},
						},
						Indexes: []*storepb.IndexMetadata{
							{Name: "with_pk_pkey", Expressions: []string{"id"}, Unique: true, Primary: true},
						},
					},
					{
						Name: "without_pk",
						Columns: []*storepb.ColumnMetadata{
							{Name: "id", Type: "integer"},
						},
					},
				},
			},
		},
	}

	tests := []struct {
		name        string
		stmt        string
		wantContent string // empty means no advice
	}{
		{
			name:        "drop PK on unqualified table in non-public schema",
			stmt:        "ALTER TABLE with_pk DROP CONSTRAINT with_pk_pkey;",
			wantContent: `Table "app_schema"."with_pk" requires PRIMARY KEY`,
		},
		{
			name:        "drop PK column on unqualified table in non-public schema",
			stmt:        "ALTER TABLE with_pk DROP COLUMN id;",
			wantContent: `Table "app_schema"."with_pk" requires PRIMARY KEY`,
		},
		{
			name: "unrelated ALTER on unqualified table with PK",
			stmt: "ALTER TABLE with_pk ADD COLUMN note TEXT;",
		},
		{
			name: "unrelated ALTER on unqualified table without PK",
			stmt: "ALTER TABLE without_pk ADD COLUMN note TEXT;",
		},
		{
			name:        "create unqualified table lands in first search_path schema",
			stmt:        "CREATE TABLE fresh(id INT);",
			wantContent: `Table "app_schema"."fresh" requires PRIMARY KEY`,
		},
		{
			name: "create then drop unqualified table in same batch",
			stmt: "CREATE TABLE fresh(id INT); DROP TABLE fresh;",
		},
	}

	rule := &storepb.SQLReviewRule{
		Type:   storepb.SQLReviewRule_TABLE_REQUIRE_PK,
		Level:  storepb.SQLReviewRule_WARNING,
		Engine: storepb.Engine_POSTGRES,
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			original, ok := proto.Clone(dbSchema).(*storepb.DatabaseSchemaMetadata)
			require.True(t, ok)
			final, ok := proto.Clone(dbSchema).(*storepb.DatabaseSchemaMetadata)
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
