package mariadb

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

func hasCandidate(cands []base.Candidate, typ base.CandidateType, text string) bool {
	for _, c := range cands {
		if c.Type == typ && c.Text == text {
			return true
		}
	}
	return false
}

func hasKeyword(cands []base.Candidate, text string) bool {
	for _, c := range cands {
		if c.Type == base.CandidateTypeKeyword && strings.EqualFold(c.Text, text) {
			return true
		}
	}
	return false
}

func TestCompletion_Seam(t *testing.T) {
	meta := &metadatapb.DatabaseSchemaMetadata{
		Name: "appdb",
		Schemas: []*metadatapb.SchemaMetadata{{
			Tables: []*metadatapb.TableMetadata{
				{Name: "users", Columns: []*metadatapb.ColumnMetadata{{Name: "id", Type: "int"}}},
				{Name: "order", Columns: []*metadatapb.ColumnMetadata{{Name: "select", Type: "varchar"}}},
			},
			Views: []*metadatapb.ViewMetadata{
				{Name: "active_users", Definition: "SELECT id FROM users"},
			},
		}},
	}
	cCtx := base.CompletionContext{
		Scene:           base.SceneTypeAll,
		DefaultDatabase: "appdb",
		Metadata: func(_ context.Context, _, db string) (string, *model.DatabaseMetadata, error) {
			if db != "appdb" {
				return "", nil, nil
			}
			return db, model.NewDatabaseMetadata(meta, nil, nil, storepb.Engine_MARIADB, true), nil
		},
		ListDatabaseNames: func(context.Context, string) ([]string, error) {
			return []string{"appdb", "otherdb"}, nil
		},
	}

	// Registry dispatch, metadata callbacks, quoting, and candidate type mapping.
	stmt := "SELECT * FROM "
	got, err := base.Completion(context.Background(), storepb.Engine_MARIADB, cCtx, stmt, 1, len(stmt))
	require.NoError(t, err)
	require.True(t, hasCandidate(got, base.CandidateTypeTable, "users"), "got %v", got)
	require.True(t, hasCandidate(got, base.CandidateTypeTable, "`order`"), "got %v", got)
	require.True(t, hasCandidate(got, base.CandidateTypeView, "active_users"), "got %v", got)
	require.True(t, hasCandidate(got, base.CandidateTypeDatabase, "otherdb"), "got %v", got)

	colStmt := "SELECT  FROM users"
	gotCols, err := base.Completion(context.Background(), storepb.Engine_MARIADB, cCtx, colStmt, 1, len("SELECT "))
	require.NoError(t, err)
	require.True(t, hasCandidate(gotCols, base.CandidateTypeColumn, "id"), "got %v", gotCols)
	require.True(t, hasCandidate(gotCols, base.CandidateTypeFunction, "COUNT()"), "got %v", gotCols)

	// Scene wiring: SceneTypeQuery maps to omni ReadOnly mode.
	queryCtx := cCtx
	queryCtx.Scene = base.SceneTypeQuery
	gotAll, err := base.Completion(context.Background(), storepb.Engine_MARIADB, cCtx, "SELECT 1; ", 1, len("SELECT 1; "))
	require.NoError(t, err)
	gotQuery, err := base.Completion(context.Background(), storepb.Engine_MARIADB, queryCtx, "SELECT 1; ", 1, len("SELECT 1; "))
	require.NoError(t, err)
	require.True(t, hasKeyword(gotAll, "INSERT"))
	require.False(t, hasKeyword(gotQuery, "INSERT"))
	require.True(t, hasKeyword(gotQuery, "SELECT"))

	// Nil or failing metadata callback degrades cleanly.
	for _, badCtx := range []base.CompletionContext{
		{Scene: base.SceneTypeAll, DefaultDatabase: "appdb"},
		{
			Scene:           base.SceneTypeAll,
			DefaultDatabase: "appdb",
			Metadata: func(context.Context, string, string) (string, *model.DatabaseMetadata, error) {
				return "", nil, errors.New("metadata unavailable")
			},
		},
	} {
		got, err := base.Completion(context.Background(), storepb.Engine_MARIADB, badCtx, "SELECT * FROM ", 1, len("SELECT * FROM "))
		require.NoError(t, err)
		require.False(t, hasCandidate(got, base.CandidateTypeTable, "users"))
	}
}
