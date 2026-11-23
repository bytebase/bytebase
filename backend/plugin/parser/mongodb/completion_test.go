package mongodb

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
	getMeta := func(_ context.Context, _, databaseName string) (string, *model.DatabaseMetadata, error) {
		if databaseName != "test" {
			return "", nil, nil
		}
		return "test", model.NewDatabaseMetadata(&metadatapb.DatabaseSchemaMetadata{
			Name: databaseName,
			Schemas: []*metadatapb.SchemaMetadata{
				{
					Tables: []*metadatapb.TableMetadata{
						{Name: "users"},
						{Name: "my-collection"},
					},
				},
			},
		}, nil, nil, storepb.Engine_MONGODB, true), nil
	}

	// Registry dispatch, Metadata callback, OmniCandidateType (Collection->Table, DbMethod->Function),
	// and special collection name formatting via omni CompleteSession.
	dotCands, err := base.Completion(context.Background(), storepb.Engine_MONGODB, base.CompletionContext{
		Scene:           base.SceneTypeAll,
		DefaultDatabase: "test",
		Metadata:        getMeta,
	}, "db.", 1, 3)
	require.NoError(t, err)
	require.Contains(t, dotCands, base.Candidate{Type: base.CandidateTypeTable, Text: "users"})
	require.Contains(t, dotCands, base.Candidate{Type: base.CandidateTypeTable, Text: `getCollection("my-collection")`})
	require.Contains(t, dotCands, base.Candidate{Type: base.CandidateTypeFunction, Text: "runCommand"})

	// Bracket access + keyword/show target type mapping.
	bracketCands, err := base.Completion(context.Background(), storepb.Engine_MONGODB, base.CompletionContext{
		Scene:           base.SceneTypeAll,
		DefaultDatabase: "test",
		Metadata:        getMeta,
	}, "db[", 1, 3)
	require.NoError(t, err)
	require.Contains(t, bracketCands, base.Candidate{Type: base.CandidateTypeTable, Text: `"my-collection"]`})

	topCands, err := base.Completion(context.Background(), storepb.Engine_MONGODB, base.CompletionContext{
		Scene: base.SceneTypeAll,
	}, "", 1, 0)
	require.NoError(t, err)
	require.Contains(t, topCands, base.Candidate{Type: base.CandidateTypeKeyword, Text: "show"})
}
