package snowflake

import (
	"context"
	"errors"
	"strings"
	"testing"

	metadatapb "github.com/bytebase/omni/metadata"
	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/store/model"
)

func completionMetadata() map[string]*model.DatabaseMetadata {
	db := model.NewDatabaseMetadata(&metadatapb.DatabaseSchemaMetadata{
		Name: "DB",
		Schemas: []*metadatapb.SchemaMetadata{
			{
				Name: "PUBLIC",
				Tables: []*metadatapb.TableMetadata{
					{Name: "ORDERS", Columns: []*metadatapb.ColumnMetadata{{Name: "ID", Type: "NUMBER"}, {Name: "SRC", Type: "VARIANT"}}},
					{Name: "Order Items", Columns: []*metadatapb.ColumnMetadata{{Name: "Item ID", Type: "NUMBER"}}},
				},
				Views:             []*metadatapb.ViewMetadata{{Name: "ACTIVE_ORDERS", Columns: []*metadatapb.ColumnMetadata{{Name: "ID", Type: "NUMBER"}}}},
				MaterializedViews: []*metadatapb.MaterializedViewMetadata{{Name: "ORDERS_SUMMARY"}},
			},
			{
				Name:   "SALES",
				Tables: []*metadatapb.TableMetadata{{Name: "DEALS", Columns: []*metadatapb.ColumnMetadata{{Name: "AMOUNT", Type: "NUMBER"}}}},
			},
		},
	}, nil, nil, storepb.Engine_SNOWFLAKE, true)
	other := model.NewDatabaseMetadata(&metadatapb.DatabaseSchemaMetadata{
		Name: "OTHERDB",
		Schemas: []*metadatapb.SchemaMetadata{
			{Name: "PUBLIC", Tables: []*metadatapb.TableMetadata{{Name: "OTHER_T", Columns: []*metadatapb.ColumnMetadata{{Name: "OTHER_COL", Type: "NUMBER"}}}}},
		},
	}, nil, nil, storepb.Engine_SNOWFLAKE, true)
	return map[string]*model.DatabaseMetadata{"DB": db, "OTHERDB": other}
}

func completionContext() base.CompletionContext {
	dbs := completionMetadata()
	return base.CompletionContext{
		Scene:           base.SceneTypeAll,
		InstanceID:      "instance",
		DefaultDatabase: "DB",
		DefaultSchema:   "PUBLIC",
		Metadata: func(_ context.Context, _, name string) (string, *model.DatabaseMetadata, error) {
			if db, ok := dbs[name]; ok {
				return name, db, nil
			}
			return "", nil, errors.New("database not found")
		},
	}
}

// complete completes text at its first |.
func complete(t *testing.T, cCtx base.CompletionContext, text string) []base.Candidate {
	t.Helper()
	caret := strings.Index(text, "|")
	require.GreaterOrEqual(t, caret, 0)
	got, err := base.Completion(context.Background(), storepb.Engine_SNOWFLAKE, cCtx, text[:caret]+text[caret+1:], 1, caret)
	require.NoError(t, err)
	return got
}

func hasCandidate(got []base.Candidate, typ base.CandidateType, text string) bool {
	for _, c := range got {
		if c.Type == typ && c.Text == text {
			return true
		}
	}
	return false
}

func TestCompletion(t *testing.T) {
	type candidate struct {
		typ  base.CandidateType
		text string
	}
	tests := []struct {
		name   string
		text   string
		want   []candidate
		absent []candidate
	}{
		{
			name: "relations",
			text: "SELECT * FROM |",
			want: []candidate{
				{base.CandidateTypeTable, "ORDERS"},
				{base.CandidateTypeTable, `"Order Items"`},
				{base.CandidateTypeView, "ACTIVE_ORDERS"},
				{base.CandidateTypeMaterializedView, "ORDERS_SUMMARY"},
				{base.CandidateTypeSchema, "SALES"},
				{base.CandidateTypeKeyword, "LATERAL"},
			},
			absent: []candidate{{base.CandidateTypeTable, "DEALS"}},
		},
		{
			name: "Snowflake's clauses",
			text: "SELECT * FROM orders |",
			want: []candidate{
				{base.CandidateTypeKeyword, "QUALIFY"},
				{base.CandidateTypeKeyword, "MATCH_RECOGNIZE"},
				{base.CandidateTypeKeyword, "SAMPLE"},
			},
		},
		{
			name: "columns and functions",
			text: "SELECT | FROM orders",
			want: []candidate{
				{base.CandidateTypeColumn, "ID"},
				{base.CandidateTypeColumn, "SRC"},
				{base.CandidateTypeFunction, "PARSE_JSON()"},
				{base.CandidateTypeKeyword, "DISTINCT"},
			},
		},
		{
			name: "quoted columns",
			text: `SELECT | FROM "Order Items"`,
			want: []candidate{{base.CandidateTypeColumn, `"Item ID"`}},
		},
		{
			name:   "schema qualified",
			text:   "SELECT * FROM sales.|",
			want:   []candidate{{base.CandidateTypeTable, "DEALS"}},
			absent: []candidate{{base.CandidateTypeTable, "ORDERS"}},
		},
		{
			name: "another database",
			text: "SELECT * FROM otherdb.public.|",
			want: []candidate{{base.CandidateTypeTable, "OTHER_T"}},
		},
		{
			name: "FLATTEN columns",
			text: "SELECT f.| FROM orders o, LATERAL FLATTEN(input => o.src) f",
			want: []candidate{{base.CandidateTypeColumn, "VALUE"}, {base.CandidateTypeColumn, "KEY"}},
		},
		{
			name:   "semi-structured path",
			text:   "SELECT src:| FROM orders",
			absent: []candidate{{base.CandidateTypeColumn, "ID"}},
		},
	}
	cCtx := completionContext()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := complete(t, cCtx, tt.text)
			for _, w := range tt.want {
				require.True(t, hasCandidate(got, w.typ, w.text), "missing %v %q in %v", w.typ, w.text, got)
			}
			for _, a := range tt.absent {
				require.False(t, hasCandidate(got, a.typ, a.text), "unexpected %v %q", a.typ, a.text)
			}
		})
	}
}

func TestCompletionDefaultSchema(t *testing.T) {
	cCtx := completionContext()
	cCtx.DefaultSchema = "SALES"
	got := complete(t, cCtx, "SELECT * FROM |")
	// A query resolves unqualified names in the current schema, then PUBLIC.
	require.True(t, hasCandidate(got, base.CandidateTypeTable, "DEALS"), "got %v", got)
	require.True(t, hasCandidate(got, base.CandidateTypeTable, "ORDERS"), "got %v", got)

	got = complete(t, cCtx, "INSERT INTO |")
	require.True(t, hasCandidate(got, base.CandidateTypeTable, "DEALS"), "got %v", got)
	require.False(t, hasCandidate(got, base.CandidateTypeTable, "ORDERS"), "got %v", got)
}

func TestCompletionWithoutMetadata(t *testing.T) {
	for _, cCtx := range []base.CompletionContext{
		{Scene: base.SceneTypeAll, DefaultDatabase: "DB", DefaultSchema: "PUBLIC"},
		{
			Scene:           base.SceneTypeAll,
			DefaultDatabase: "DB",
			DefaultSchema:   "PUBLIC",
			Metadata: func(context.Context, string, string) (string, *model.DatabaseMetadata, error) {
				return "", nil, errors.New("metadata unavailable")
			},
		},
	} {
		got := complete(t, cCtx, "SELECT * FROM |")
		require.True(t, hasCandidate(got, base.CandidateTypeKeyword, "LATERAL"), "got %v", got)
		for _, c := range got {
			require.NotContains(t, []base.CandidateType{
				base.CandidateTypeSchema, base.CandidateTypeTable, base.CandidateTypeView,
				base.CandidateTypeMaterializedView, base.CandidateTypeColumn,
			}, c.Type)
		}
	}
}
