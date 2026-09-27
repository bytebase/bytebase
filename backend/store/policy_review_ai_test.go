package store_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/testcontainer"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/store"

	_ "github.com/bytebase/bytebase/backend/plugin/db/pg"
)

// TestGetEffectiveReviewAIPolicy pins that both levels apply: the workspace
// and the project policy come back side by side, a level without a policy
// or with one that is not enforced is empty, and another workspace's rows
// under the same resource names never leak in.
func TestGetEffectiveReviewAIPolicy(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	db, stores, _ := testcontainer.NewMetadataDB(t)
	_, err := db.ExecContext(ctx, `
		INSERT INTO workspace (resource_id) VALUES ('ws'), ('other');
		INSERT INTO project (resource_id, workspace, name) VALUES ('p', 'ws', 'P');
	`)
	require.NoError(t, err)

	upsert := func(workspace string, resourceType storepb.Policy_Resource, resource string, enforce bool, content string) {
		payload, err := protojson.Marshal(&storepb.ReviewAIPolicy{Content: content})
		require.NoError(t, err)
		_, err = stores.CreatePolicy(ctx, &store.PolicyMessage{
			Workspace:    workspace,
			ResourceType: resourceType,
			Resource:     resource,
			Type:         storepb.Policy_REVIEW_AI,
			Payload:      string(payload),
			Enforce:      enforce,
		})
		require.NoError(t, err)
	}
	effective := func(workspace string) *store.EffectiveReviewAIPolicy {
		p, err := stores.GetEffectiveReviewAIPolicy(ctx, workspace, "p")
		require.NoError(t, err)
		return p
	}

	// Neither level has a policy.
	require.Equal(t, &store.EffectiveReviewAIPolicy{}, effective("ws"))

	// Workspace only.
	upsert("ws", storepb.Policy_WORKSPACE, common.FormatWorkspace("ws"), true, "W1")
	require.Equal(t, &store.EffectiveReviewAIPolicy{Workspace: "W1"}, effective("ws"))

	// Both levels come back side by side; the reviewer, not the store, merges them.
	upsert("ws", storepb.Policy_PROJECT, common.FormatProject("p"), true, "P1")
	require.Equal(t, &store.EffectiveReviewAIPolicy{Workspace: "W1", Project: "P1"}, effective("ws"))

	// A project policy that is not enforced counts as absent.
	upsert("ws", storepb.Policy_PROJECT, common.FormatProject("p"), false, "P1")
	require.Equal(t, &store.EffectiveReviewAIPolicy{Workspace: "W1"}, effective("ws"))

	// Same for the workspace.
	upsert("ws", storepb.Policy_PROJECT, common.FormatProject("p"), true, "P2")
	upsert("ws", storepb.Policy_WORKSPACE, common.FormatWorkspace("ws"), false, "W1")
	require.Equal(t, &store.EffectiveReviewAIPolicy{Project: "P2"}, effective("ws"))

	// The policy key is (workspace, resource_type, resource, type): rows of
	// another workspace under the same resource names are a different key and
	// never leak in, in either direction.
	upsert("other", storepb.Policy_WORKSPACE, common.FormatWorkspace("other"), true, "OW")
	upsert("other", storepb.Policy_PROJECT, common.FormatProject("p"), true, "OP")
	require.Equal(t, &store.EffectiveReviewAIPolicy{Project: "P2"}, effective("ws"))
	require.Equal(t, &store.EffectiveReviewAIPolicy{Workspace: "OW", Project: "OP"}, effective("other"))
}
