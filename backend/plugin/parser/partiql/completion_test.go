package partiql

import (
	"context"
	"testing"

	metadatapb "github.com/bytebase/omni/metadata"
	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/store/model"
)

func TestCompletionSeam(t *testing.T) {
	meta := model.NewDatabaseMetadata(&metadatapb.DatabaseSchemaMetadata{
		Name: "xx-ap-east-1",
		Schemas: []*metadatapb.SchemaMetadata{
			{
				Tables: []*metadatapb.TableMetadata{
					{
						Name: "Music",
						Columns: []*metadatapb.ColumnMetadata{
							{Name: "Artist"},
							{Name: "SongTitle"},
						},
					},
				},
			},
		},
	}, nil, nil, storepb.Engine_DYNAMODB, true)

	getter := func(_ context.Context, _, db string) (string, *model.DatabaseMetadata, error) {
		if db == "xx-ap-east-1" {
			return "", meta, nil
		}
		return "", nil, nil
	}

	// Table completion through base.Completion registry + Metadata callback.
	fromCands, err := base.Completion(context.Background(), storepb.Engine_DYNAMODB, base.CompletionContext{
		Scene:           base.SceneTypeAll,
		DefaultDatabase: "xx-ap-east-1",
		Metadata:        getter,
	}, "SELECT * FROM ", 1, len("SELECT * FROM "))
	require.NoError(t, err)
	require.Contains(t, fromCands, base.Candidate{Type: base.CandidateTypeTable, Text: "Music"})

	// Column completion with multi-line + multi-byte UTF-8 lineOffsetToBytePos.
	colStmt := "-- 音乐\nSELECT  FROM Music"
	colCands, err := base.Completion(context.Background(), storepb.Engine_DYNAMODB, base.CompletionContext{
		Scene:           base.SceneTypeAll,
		DefaultDatabase: "xx-ap-east-1",
		Metadata:        getter,
	}, colStmt, 2, len("SELECT "))
	require.NoError(t, err)
	require.Contains(t, colCands, base.Candidate{Type: base.CandidateTypeColumn, Text: "Artist"})
	require.Contains(t, colCands, base.Candidate{Type: base.CandidateTypeColumn, Text: "SongTitle"})

	// SceneTypeQuery filters out DML/DDL keywords while SceneTypeAll retains them.
	allCands, err := base.Completion(context.Background(), storepb.Engine_DYNAMODB, base.CompletionContext{
		Scene: base.SceneTypeAll,
	}, "INS", 1, 3)
	require.NoError(t, err)
	require.Contains(t, allCands, base.Candidate{Type: base.CandidateTypeKeyword, Text: "INSERT"})

	queryCands, err := base.Completion(context.Background(), storepb.Engine_DYNAMODB, base.CompletionContext{
		Scene: base.SceneTypeQuery,
	}, "INS", 1, 3)
	require.NoError(t, err)
	require.NotContains(t, queryCands, base.Candidate{Type: base.CandidateTypeKeyword, Text: "INSERT"})
}
