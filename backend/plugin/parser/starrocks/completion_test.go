package starrocks

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

func completionContext() base.CompletionContext {
	db := model.NewDatabaseMetadata(&metadatapb.DatabaseSchemaMetadata{
		Name: "appdb",
		Schemas: []*metadatapb.SchemaMetadata{{
			Tables: []*metadatapb.TableMetadata{
				{Name: "users", Columns: []*metadatapb.ColumnMetadata{{Name: "id", Type: "BIGINT"}, {Name: "name", Type: "VARCHAR(64)"}}},
				{Name: "order", Columns: []*metadatapb.ColumnMetadata{{Name: "select", Type: "INT"}}},
			},
			Views: []*metadatapb.ViewMetadata{{Name: "active_users", Columns: []*metadatapb.ColumnMetadata{{Name: "id", Type: "BIGINT"}}}},
		}},
	}, nil, nil, storepb.Engine_STARROCKS, true)
	return base.CompletionContext{
		Scene:           base.SceneTypeAll,
		InstanceID:      "instance",
		DefaultDatabase: "appdb",
		Metadata: func(_ context.Context, _, name string) (string, *model.DatabaseMetadata, error) {
			if name == "appdb" {
				return name, db, nil
			}
			return "", nil, errors.New("database not found")
		},
		ListDatabaseNames: func(context.Context, string) ([]string, error) {
			return []string{"appdb", "otherdb"}, nil
		},
	}
}

// complete completes text at its first |.
func complete(t *testing.T, cCtx base.CompletionContext, text string) []base.Candidate {
	t.Helper()
	caret := strings.Index(text, "|")
	require.GreaterOrEqual(t, caret, 0)
	got, err := base.Completion(context.Background(), storepb.Engine_STARROCKS, cCtx, text[:caret]+text[caret+1:], 1, caret)
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
	cCtx := completionContext()

	got := complete(t, cCtx, "SELECT * FROM |")
	require.True(t, hasCandidate(got, base.CandidateTypeTable, "users"), "got %v", got)
	require.True(t, hasCandidate(got, base.CandidateTypeTable, "`order`"), "got %v", got)
	require.True(t, hasCandidate(got, base.CandidateTypeView, "active_users"), "got %v", got)
	require.True(t, hasCandidate(got, base.CandidateTypeDatabase, "otherdb"), "got %v", got)

	got = complete(t, cCtx, "SELECT u.| FROM users u")
	require.True(t, hasCandidate(got, base.CandidateTypeColumn, "id"), "got %v", got)
	require.True(t, hasCandidate(got, base.CandidateTypeColumn, "name"), "got %v", got)

	got = complete(t, cCtx, "SELECT * FROM users |")
	require.True(t, hasCandidate(got, base.CandidateTypeKeyword, "TABLET"), "got %v", got)
	require.True(t, hasCandidate(got, base.CandidateTypeKeyword, "WHERE"), "got %v", got)

	// Without metadata only keywords come back.
	got = complete(t, base.CompletionContext{Scene: base.SceneTypeAll, DefaultDatabase: "appdb"}, "SELECT * FROM |")
	require.False(t, hasCandidate(got, base.CandidateTypeTable, "users"))
}
