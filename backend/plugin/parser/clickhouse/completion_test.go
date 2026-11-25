package clickhouse

import (
	"context"
	"testing"

	metadatapb "github.com/bytebase/omni/metadata"
	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/store/model"
)

func TestClickHouseDiagnose(t *testing.T) {
	// ClickHouse-only syntax MySQL's grammar rejects.
	diags, err := base.Diagnose(context.Background(), base.DiagnoseContext{}, storepb.Engine_CLICKHOUSE,
		"SELECT * FROM t FINAL SAMPLE 0.1 SETTINGS max_threads = 1;\nSELECT arrayJoin([1, 2]) FORMAT JSONEachRow;")
	require.NoError(t, err)
	require.Empty(t, diags)

	diags, err = base.Diagnose(context.Background(), base.DiagnoseContext{}, storepb.Engine_CLICKHOUSE, "SELECT 1;\nSELECT * FROM t WHERE )")
	require.NoError(t, err)
	require.Len(t, diags, 1)
	require.Equal(t, uint32(1), diags[0].Range.Start.Line)
}

func TestClickHouseCompletion(t *testing.T) {
	meta := &metadatapb.DatabaseSchemaMetadata{
		Name: "db",
		Schemas: []*metadatapb.SchemaMetadata{{
			Tables: []*metadatapb.TableMetadata{
				{Name: "events", Columns: []*metadatapb.ColumnMetadata{{Name: "ts", Type: "DateTime"}}},
			},
		}},
	}
	cCtx := base.CompletionContext{
		Scene:           base.SceneTypeAll,
		DefaultDatabase: "db",
		Metadata: func(_ context.Context, _, name string) (string, *model.DatabaseMetadata, error) {
			if name != "db" {
				return "", nil, nil
			}
			return name, model.NewDatabaseMetadata(meta, nil, nil, storepb.Engine_CLICKHOUSE, true), nil
		},
		ListDatabaseNames: func(context.Context, string) ([]string, error) {
			return []string{"db", "other"}, nil
		},
	}
	for _, tc := range []struct {
		statement string
		caret     int
		want      base.Candidate
	}{
		{statement: "SELECT * FROM ", want: base.Candidate{Type: base.CandidateTypeTable, Text: "events"}},
		{statement: "SELECT * FROM ", want: base.Candidate{Type: base.CandidateTypeDatabase, Text: "other"}},
		{statement: "SELECT  FROM events", caret: len("SELECT "), want: base.Candidate{Type: base.CandidateTypeColumn, Text: "ts"}},
		// ClickHouse's own keywords, which MySQL's grammar does not offer.
		{statement: "SELECT * FROM events ", want: base.Candidate{Type: base.CandidateTypeKeyword, Text: "PREWHERE"}},
		{statement: "SELECT * FROM events ", want: base.Candidate{Type: base.CandidateTypeKeyword, Text: "FINAL"}},
	} {
		caret := tc.caret
		if caret == 0 {
			caret = len(tc.statement)
		}
		got, err := base.Completion(context.Background(), storepb.Engine_CLICKHOUSE, cCtx, tc.statement, 1, caret)
		require.NoError(t, err)
		found := false
		for _, c := range got {
			if c.Type == tc.want.Type && c.Text == tc.want.Text {
				found = true
			}
		}
		require.True(t, found, "%q: want %v in %v", tc.statement, tc.want, got)
	}
}
