package plsql

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
	content, err := os.ReadFile("plsql.go")
	require.NoError(t, err)
	source := string(content)
	require.NotContains(t, source, "github.com/antlr4-go/antlr/v4")
	require.NotContains(t, source, "github.com/bytebase/parser/plsql")
	require.NotContains(t, source, "CodeCompletionCore")
}

func TestCompletion_Seam(t *testing.T) {
	schemas := map[string]*metadatapb.DatabaseSchemaMetadata{
		"SCHEMA1": {
			Name: "SCHEMA1",
			Schemas: []*metadatapb.SchemaMetadata{{
				Tables: []*metadatapb.TableMetadata{
					{
						Name:     "T1",
						Columns:  []*metadatapb.ColumnMetadata{{Name: "C1", Type: "NUMBER"}},
						Indexes:  []*metadatapb.IndexMetadata{{Name: "IDX_T1"}},
						Triggers: []*metadatapb.TriggerMetadata{{Name: "TRG_T1"}},
					},
					{Name: "lower_tbl", Columns: []*metadatapb.ColumnMetadata{{Name: "MixedCol", Type: "NUMBER"}}},
				},
				Views:             []*metadatapb.ViewMetadata{{Name: "V1", Definition: "CREATE VIEW V1 AS SELECT C1 FROM T1"}},
				MaterializedViews: []*metadatapb.MaterializedViewMetadata{{Name: "MV1", Definition: "SELECT C1 FROM T1"}},
				Sequences:         []*metadatapb.SequenceMetadata{{Name: "SEQ1"}},
				Synonyms:          []*metadatapb.SynonymMetadata{{Name: "T1_SYN", TargetName: "T1"}},
				Functions:         []*metadatapb.FunctionMetadata{{Name: "GET_TOTAL"}},
				Procedures:        []*metadatapb.ProcedureMetadata{{Name: "SYNC_DATA"}},
			}},
		},
		"SCHEMA2": {
			Name: "SCHEMA2",
			Schemas: []*metadatapb.SchemaMetadata{{
				Tables: []*metadatapb.TableMetadata{{Name: "REMOTE_T", Columns: []*metadatapb.ColumnMetadata{{Name: "R1", Type: "NUMBER"}}}},
			}},
		},
	}

	var gotInstanceIDs []string
	cCtx := base.CompletionContext{
		Scene:           base.SceneTypeAll,
		InstanceID:      "inst-1",
		DefaultDatabase: "SCHEMA1",
		Metadata: func(_ context.Context, instanceID, schema string) (string, *model.DatabaseMetadata, error) {
			gotInstanceIDs = append(gotInstanceIDs, instanceID)
			if m, ok := schemas[schema]; ok {
				return schema, model.NewDatabaseMetadata(m, nil, nil, storepb.Engine_ORACLE, true), nil
			}
			return "", nil, nil
		},
		ListDatabaseNames: func(_ context.Context, instanceID string) ([]string, error) {
			gotInstanceIDs = append(gotInstanceIDs, instanceID)
			return []string{"SCHEMA1", "SCHEMA2"}, nil
		},
	}

	// Registry dispatch, DefaultDatabase->DefaultSchema wiring, quoting, and relation candidate mapping.
	gotFrom, err := base.Completion(context.Background(), storepb.Engine_ORACLE, cCtx, "SELECT * FROM ", 1, len("SELECT * FROM "))
	require.NoError(t, err)
	for _, want := range []struct {
		typ  base.CandidateType
		text string
	}{
		{base.CandidateTypeSchema, "SCHEMA1"},
		{base.CandidateTypeSchema, "SCHEMA2"},
		{base.CandidateTypeTable, "T1"},
		{base.CandidateTypeTable, `"lower_tbl"`},
		{base.CandidateTypeTable, "T1_SYN"},
		{base.CandidateTypeView, "V1"},
		{base.CandidateTypeMaterializedView, "MV1"},
		{base.CandidateTypeSequence, "SEQ1"},
	} {
		require.True(t, hasCandidate(gotFrom, want.typ, want.text), "missing %s %q in %v", want.typ, want.text, gotFrom)
	}
	require.NotEmpty(t, gotInstanceIDs)
	for _, id := range gotInstanceIDs {
		require.Equal(t, "inst-1", id)
	}

	// Open double-quote seam omits outer quotes; cross-schema metadata loads lazily.
	gotQuoted, err := base.Completion(context.Background(), storepb.Engine_ORACLE, cCtx, `SELECT * FROM "low`, 1, len(`SELECT * FROM "low`))
	require.NoError(t, err)
	require.True(t, hasCandidate(gotQuoted, base.CandidateTypeTable, "lower_tbl"), "got %v", gotQuoted)

	gotCross, err := base.Completion(context.Background(), storepb.Engine_ORACLE, cCtx, "SELECT * FROM SCHEMA2.", 1, len("SELECT * FROM SCHEMA2."))
	require.NoError(t, err)
	require.True(t, hasCandidate(gotCross, base.CandidateTypeTable, "REMOTE_T"), "got %v", gotCross)

	// Column, function, routine, index, trigger, and UTF-16 caret offset seam (U+1F600 = 2 UTF-16 code units).
	emojiSQL := "SELECT '😀',  FROM T1"
	gotCols, err := base.Completion(context.Background(), storepb.Engine_ORACLE, cCtx, emojiSQL, 1, len("SELECT '")+2+len("', "))
	require.NoError(t, err)
	require.True(t, hasCandidate(gotCols, base.CandidateTypeColumn, "C1"), "got %v", gotCols)
	require.True(t, hasCandidate(gotCols, base.CandidateTypeFunction, "GET_TOTAL()"), "got %v", gotCols)

	for _, tc := range []struct {
		sql  string
		typ  base.CandidateType
		text string
	}{
		{"DROP PROCEDURE ", base.CandidateTypeRoutine, "SYNC_DATA"},
		{"DROP INDEX ", base.CandidateTypeIndex, "IDX_T1"},
		{"DROP TRIGGER ", base.CandidateTypeTrigger, "TRG_T1"},
	} {
		got, err := base.Completion(context.Background(), storepb.Engine_ORACLE, cCtx, tc.sql, 1, len(tc.sql))
		require.NoError(t, err)
		require.True(t, hasCandidate(got, tc.typ, tc.text), "sql=%q got %v", tc.sql, got)
	}

	// SceneTypeQuery wires to omni ReadOnly mode, filtering write keywords and write statements.
	queryCtx := cCtx
	queryCtx.Scene = base.SceneTypeQuery
	gotAllStart, err := base.Completion(context.Background(), storepb.Engine_ORACLE, cCtx, "", 1, 0)
	require.NoError(t, err)
	gotQueryStart, err := base.Completion(context.Background(), storepb.Engine_ORACLE, queryCtx, "", 1, 0)
	require.NoError(t, err)
	require.True(t, hasCandidate(gotAllStart, base.CandidateTypeKeyword, "INSERT"), "got %v", gotAllStart)
	require.False(t, hasCandidate(gotQueryStart, base.CandidateTypeKeyword, "INSERT"), "got %v", gotQueryStart)
	require.True(t, hasCandidate(gotQueryStart, base.CandidateTypeKeyword, "SELECT"), "got %v", gotQueryStart)
	gotDrop, err := base.Completion(context.Background(), storepb.Engine_ORACLE, queryCtx, "DROP TABLE ", 1, len("DROP TABLE "))
	require.NoError(t, err)
	require.Nil(t, gotDrop)

	// Nil or failing metadata callbacks degrade cleanly without object candidates.
	for _, badCtx := range []base.CompletionContext{
		{Scene: base.SceneTypeAll, DefaultDatabase: "SCHEMA1"},
		{
			Scene:           base.SceneTypeAll,
			DefaultDatabase: "SCHEMA1",
			Metadata: func(context.Context, string, string) (string, *model.DatabaseMetadata, error) {
				return "", nil, errors.New("metadata unavailable")
			},
			ListDatabaseNames: func(context.Context, string) ([]string, error) {
				return nil, errors.New("list unavailable")
			},
		},
	} {
		got, err := base.Completion(context.Background(), storepb.Engine_ORACLE, badCtx, "SELECT * FROM ", 1, len("SELECT * FROM "))
		require.NoError(t, err)
		require.False(t, hasCandidate(got, base.CandidateTypeSchema, "SCHEMA2"))
		for _, c := range got {
			require.NotContains(t, []base.CandidateType{
				base.CandidateTypeTable, base.CandidateTypeView,
				base.CandidateTypeMaterializedView, base.CandidateTypeColumn, base.CandidateTypeSequence,
			}, c.Type)
		}
	}
}
