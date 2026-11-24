package redshift

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
	content, err := os.ReadFile("redshift.go")
	require.NoError(t, err)
	source := string(content)
	require.NotContains(t, source, "github.com/antlr4-go/antlr/v4")
	require.NotContains(t, source, "github.com/bytebase/parser/redshift")
	require.NotContains(t, source, "CodeCompletionCore")
}

func TestCompletion_Seam(t *testing.T) {
	metadata := model.NewDatabaseMetadata(&metadatapb.DatabaseSchemaMetadata{
		Name: "db",
		Schemas: []*metadatapb.SchemaMetadata{
			{
				Name: "public",
				Tables: []*metadatapb.TableMetadata{
					{Name: "orders", Columns: []*metadatapb.ColumnMetadata{{Name: "id", Type: "integer"}}},
					{Name: "Order Items", Columns: []*metadatapb.ColumnMetadata{{Name: "Item ID", Type: "bigint"}}},
				},
				ExternalTables: []*metadatapb.ExternalTableMetadata{
					{Name: "spectrum_orders", Columns: []*metadatapb.ColumnMetadata{{Name: "external_id", Type: "varchar"}}},
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
	}, nil, nil, storepb.Engine_REDSHIFT, false)

	cCtx := base.CompletionContext{
		Scene:           base.SceneTypeAll,
		InstanceID:      "instance",
		DefaultDatabase: "db",
		DefaultSchema:   "public",
		Metadata: func(context.Context, string, string) (string, *model.DatabaseMetadata, error) {
			return "db", metadata, nil
		},
	}

	// Registry dispatch, quoting, and candidate type mapping.
	gotFrom, err := base.Completion(context.Background(), storepb.Engine_REDSHIFT, cCtx, "SELECT 1 FROM ", 1, len("SELECT 1 FROM "))
	require.NoError(t, err)
	require.True(t, hasCandidate(gotFrom, base.CandidateTypeSchema, "analytics"), "got %v", gotFrom)
	require.True(t, hasCandidate(gotFrom, base.CandidateTypeTable, "orders"), "got %v", gotFrom)
	require.True(t, hasCandidate(gotFrom, base.CandidateTypeTable, "spectrum_orders"), "got %v", gotFrom)
	require.True(t, hasCandidate(gotFrom, base.CandidateTypeTable, `"Order Items"`), "got %v", gotFrom)
	require.True(t, hasCandidate(gotFrom, base.CandidateTypeView, "active_orders"), "got %v", gotFrom)
	require.True(t, hasCandidate(gotFrom, base.CandidateTypeMaterializedView, "orders_summary"), "got %v", gotFrom)
	require.True(t, hasCandidate(gotFrom, base.CandidateTypeSequence, "order_seq"), "got %v", gotFrom)

	// Redshift-specific slot proves Engine_REDSHIFT dispatches to omni Redshift.
	copySQL := "COPY orders FROM 's3://bucket/file' "
	gotCopy, err := base.Completion(context.Background(), storepb.Engine_REDSHIFT, cCtx, copySQL, 1, len(copySQL))
	require.NoError(t, err)
	require.True(t, hasCandidate(gotCopy, base.CandidateTypeKeyword, "IAM_ROLE"), "got %v", gotCopy)

	// DefaultSchema wiring resolves non-public schema unqualified.
	analyticsCtx := cCtx
	analyticsCtx.DefaultSchema = "analytics"
	gotCols, err := base.Completion(context.Background(), storepb.Engine_REDSHIFT, analyticsCtx, "SELECT  FROM events", 1, len("SELECT "))
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
		got, err := base.Completion(context.Background(), storepb.Engine_REDSHIFT, badCtx, "SELECT 1 FROM ", 1, len("SELECT 1 FROM "))
		require.NoError(t, err)
		for _, c := range got {
			require.NotContains(t, []base.CandidateType{
				base.CandidateTypeSchema, base.CandidateTypeTable, base.CandidateTypeView,
				base.CandidateTypeMaterializedView, base.CandidateTypeColumn, base.CandidateTypeSequence,
			}, c.Type)
		}
	}
}
