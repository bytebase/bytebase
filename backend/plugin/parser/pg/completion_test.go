package pg

import (
	"context"
	"errors"
	"os"
	"testing"

	metadatapb "github.com/bytebase/omni/metadata"
	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/store/model"
)

func hasCandidate(cands []base.Candidate, typ base.CandidateType, text string) bool {
	for _, c := range cands {
		if c.Type == typ && c.Text == text {
			return true
		}
	}
	return false
}

func TestCompletionDoesNotDependOnANTLR(t *testing.T) {
	content, err := os.ReadFile("pg.go")
	require.NoError(t, err)
	source := string(content)
	require.NotContains(t, source, "github.com/antlr4-go/antlr/v4")
	require.NotContains(t, source, "github.com/bytebase/parser/postgresql")
	require.NotContains(t, source, "CodeCompletionCore")
}

func TestCompletion_Seam(t *testing.T) {
	metadata := model.NewDatabaseMetadata(&metadatapb.DatabaseSchemaMetadata{
		Name: "db",
		Schemas: []*metadatapb.SchemaMetadata{
			{
				Name: "public",
				Tables: []*metadatapb.TableMetadata{
					{
						Name: "orders",
						Columns: []*metadatapb.ColumnMetadata{
							{Name: "id", Type: "integer"},
							{Name: "amount", Type: "numeric"},
						},
					},
					{
						Name:    "Order Items",
						Columns: []*metadatapb.ColumnMetadata{{Name: "Item ID", Type: "bigint"}},
					},
				},
				Views: []*metadatapb.ViewMetadata{
					{Name: "active_orders", Columns: []*metadatapb.ColumnMetadata{{Name: "id", Type: "integer"}}},
				},
				MaterializedViews: []*metadatapb.MaterializedViewMetadata{
					{Name: "orders_summary", Definition: "SELECT id FROM orders"},
				},
				Sequences: []*metadatapb.SequenceMetadata{{Name: "order_seq"}},
			},
			{
				Name: "analytics",
				Tables: []*metadatapb.TableMetadata{
					{Name: "events", Columns: []*metadatapb.ColumnMetadata{{Name: "event_id", Type: "bigint"}}},
				},
			},
		},
	}, nil, nil, storepb.Engine_POSTGRES, true)

	cCtx := base.CompletionContext{
		Scene:           base.SceneTypeAll,
		InstanceID:      "instance",
		DefaultDatabase: "db",
		DefaultSchema:   "public",
		Metadata: func(context.Context, string, string) (string, *model.DatabaseMetadata, error) {
			return "db", metadata, nil
		},
	}

	// Registry dispatch covers POSTGRES, SNOWFLAKE, and COCKROACHDB.
	for _, engine := range []storepb.Engine{
		storepb.Engine_POSTGRES,
		storepb.Engine_SNOWFLAKE,
		storepb.Engine_COCKROACHDB,
	} {
		got, err := base.Completion(context.Background(), engine, cCtx, "SELECT * FROM ", 1, len("SELECT * FROM "))
		require.NoError(t, err, "engine %v", engine)
		require.True(t, hasCandidate(got, base.CandidateTypeTable, "orders"), "engine %v: got %v", engine, got)
		require.True(t, hasCandidate(got, base.CandidateTypeTable, `"Order Items"`), "engine %v: got %v", engine, got)
		require.True(t, hasCandidate(got, base.CandidateTypeSchema, "analytics"), "engine %v: got %v", engine, got)
		require.True(t, hasCandidate(got, base.CandidateTypeView, "active_orders"), "engine %v: got %v", engine, got)
		require.True(t, hasCandidate(got, base.CandidateTypeMaterializedView, "orders_summary"), "engine %v: got %v", engine, got)
		require.True(t, hasCandidate(got, base.CandidateTypeSequence, "order_seq"), "engine %v: got %v", engine, got)
	}

	// DefaultSchema wiring prepends non-public schema to search_path.
	analyticsCtx := cCtx
	analyticsCtx.DefaultSchema = "analytics"
	gotCols, err := base.Completion(context.Background(), storepb.Engine_POSTGRES, analyticsCtx, "SELECT  FROM events", 1, len("SELECT "))
	require.NoError(t, err)
	require.True(t, hasCandidate(gotCols, base.CandidateTypeColumn, "event_id"), "got %v", gotCols)

	// Nil or failing metadata callback degrades cleanly without object candidates.
	for _, badCtx := range []base.CompletionContext{
		{Scene: base.SceneTypeAll, DefaultDatabase: "db", DefaultSchema: "public"},
		{
			Scene:           base.SceneTypeAll,
			DefaultDatabase: "db",
			DefaultSchema:   "public",
			Metadata: func(context.Context, string, string) (string, *model.DatabaseMetadata, error) {
				return "", nil, errors.New("metadata unavailable")
			},
		},
	} {
		got, err := base.Completion(context.Background(), storepb.Engine_POSTGRES, badCtx, "SELECT 1 FROM ", 1, len("SELECT 1 FROM "))
		require.NoError(t, err)
		for _, c := range got {
			require.NotContains(t, []base.CandidateType{
				base.CandidateTypeSchema, base.CandidateTypeTable, base.CandidateTypeView,
				base.CandidateTypeMaterializedView, base.CandidateTypeColumn, base.CandidateTypeSequence,
			}, c.Type)
		}
	}
}
