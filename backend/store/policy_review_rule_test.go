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

// TestGetEffectiveReviewRulePolicy pins the nearest-wins resolution: the
// project's own policy applies as is, a project without one uses the
// workspace's, and with neither every rule is on. A policy that is not
// enforced counts as absent, and one without SYNTAX leaves no rule on.
func TestGetEffectiveReviewRulePolicy(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	db, stores, _ := testcontainer.NewMetadataDB(t)
	_, err := db.ExecContext(ctx, `
		INSERT INTO workspace (resource_id) VALUES ('ws');
		INSERT INTO project (resource_id, workspace, name) VALUES ('p', 'ws', 'P');
	`)
	require.NoError(t, err)

	payload := func(rules ...storepb.ReviewRuleType) string {
		b, err := protojson.Marshal(&storepb.ReviewRulePolicy{Rules: rules})
		require.NoError(t, err)
		return string(b)
	}
	upsert := func(resourceType storepb.Policy_Resource, resource string, enforce bool, rules ...storepb.ReviewRuleType) {
		_, err := stores.CreatePolicy(ctx, &store.PolicyMessage{
			Workspace:    "ws",
			ResourceType: resourceType,
			Resource:     resource,
			Type:         storepb.Policy_REVIEW_RULE,
			Payload:      payload(rules...),
			Enforce:      enforce,
		})
		require.NoError(t, err)
	}
	effective := func() []storepb.ReviewRuleType {
		p, err := stores.GetEffectiveReviewRulePolicy(ctx, "ws", "p")
		require.NoError(t, err)
		return p.Rules
	}
	every := store.GetDefaultReviewRulePolicy().Rules
	require.Len(t, every, len(storepb.ReviewRuleType_name)-1)

	// Neither level has a policy: every rule.
	require.Equal(t, every, effective())

	// Workspace only.
	upsert(storepb.Policy_WORKSPACE, common.FormatWorkspace("ws"), true, storepb.ReviewRuleType_SYNTAX, storepb.ReviewRuleType_REQUIRE_WHERE)
	require.Equal(t, []storepb.ReviewRuleType{storepb.ReviewRuleType_SYNTAX, storepb.ReviewRuleType_REQUIRE_WHERE}, effective())

	// The project's own policy replaces the workspace's, no union.
	upsert(storepb.Policy_PROJECT, common.FormatProject("p"), true, storepb.ReviewRuleType_SYNTAX, storepb.ReviewRuleType_DISALLOW_TRUNCATE)
	require.Equal(t, []storepb.ReviewRuleType{storepb.ReviewRuleType_SYNTAX, storepb.ReviewRuleType_DISALLOW_TRUNCATE}, effective())

	// Without SYNTAX no rule is on, whatever else the list names.
	upsert(storepb.Policy_PROJECT, common.FormatProject("p"), true, storepb.ReviewRuleType_DISALLOW_TRUNCATE)
	require.Empty(t, effective())

	// An empty project list is a policy too: everything off.
	upsert(storepb.Policy_PROJECT, common.FormatProject("p"), true)
	require.Empty(t, effective())

	// A project policy that is not enforced counts as absent.
	upsert(storepb.Policy_PROJECT, common.FormatProject("p"), false, storepb.ReviewRuleType_DISALLOW_TRUNCATE)
	require.Equal(t, []storepb.ReviewRuleType{storepb.ReviewRuleType_SYNTAX, storepb.ReviewRuleType_REQUIRE_WHERE}, effective())

	// A workspace policy without SYNTAX leaves no rule on for the projects
	// that follow it.
	upsert(storepb.Policy_WORKSPACE, common.FormatWorkspace("ws"), true, storepb.ReviewRuleType_REQUIRE_WHERE)
	require.Empty(t, effective())

	// Same for the workspace: back to every rule.
	upsert(storepb.Policy_WORKSPACE, common.FormatWorkspace("ws"), false, storepb.ReviewRuleType_SYNTAX)
	require.Equal(t, every, effective())

	// The same resource under another workspace is a colliding local key: the
	// policy primary key is (workspace, resource_type, resource, type). Each
	// workspace resolves to its own row and neither sees the other's.
	_, err = db.ExecContext(ctx, `INSERT INTO workspace (resource_id) VALUES ('other');`)
	require.NoError(t, err)
	_, err = stores.CreatePolicy(ctx, &store.PolicyMessage{
		Workspace:    "other",
		ResourceType: storepb.Policy_PROJECT,
		Resource:     common.FormatProject("p"),
		Type:         storepb.Policy_REVIEW_RULE,
		Payload:      payload(storepb.ReviewRuleType_SYNTAX, storepb.ReviewRuleType_REQUIRE_IS_NULL),
		Enforce:      true,
	})
	require.NoError(t, err)
	otherEffective, err := stores.GetEffectiveReviewRulePolicy(ctx, "other", "p")
	require.NoError(t, err)
	require.Equal(t, []storepb.ReviewRuleType{storepb.ReviewRuleType_SYNTAX, storepb.ReviewRuleType_REQUIRE_IS_NULL}, otherEffective.Rules)
	require.Equal(t, every, effective())

	upsert(storepb.Policy_PROJECT, common.FormatProject("p"), true, storepb.ReviewRuleType_SYNTAX, storepb.ReviewRuleType_DISALLOW_RENAME)
	require.Equal(t, []storepb.ReviewRuleType{storepb.ReviewRuleType_SYNTAX, storepb.ReviewRuleType_DISALLOW_RENAME}, effective())
	otherEffective, err = stores.GetEffectiveReviewRulePolicy(ctx, "other", "p")
	require.NoError(t, err)
	require.Equal(t, []storepb.ReviewRuleType{storepb.ReviewRuleType_SYNTAX, storepb.ReviewRuleType_REQUIRE_IS_NULL}, otherEffective.Rules)
}
