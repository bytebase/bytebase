package store_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/bytebase/bytebase/backend/common/testpg"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/store"

	_ "github.com/bytebase/bytebase/backend/plugin/db/pg"
)

// TestListProjectIamPoliciesIsScoped pins the predicate of the batch read that
// replaced a GetProjectIamPolicy loop. The policy primary key is
// (workspace, resource_type, resource, type) and the query names all four; this
// asserts each one actually filters, so a dropped clause cannot silently widen
// the read into another workspace, another resource type, or — worst — another
// policy type, whose payload would then be unmarshalled as an IamPolicy.
func TestListProjectIamPoliciesIsScoped(t *testing.T) {
	ctx := context.Background()
	db, stores, _ := testpg.New(t)
	_, err := db.ExecContext(ctx, `
		INSERT INTO workspace (resource_id) VALUES ('ws-a'), ('ws-b');
		INSERT INTO project (resource_id, workspace, name) VALUES
			('mine', 'ws-a', 'Mine'),
			('rollout-only', 'ws-a', 'Rollout only'),
			('no-policy', 'ws-a', 'No policy'),
			('other-workspace', 'ws-b', 'Other workspace');
	`)
	require.NoError(t, err)

	iamPayload := func(role string) string {
		return fmt.Sprintf(`{"bindings":[{"role":%q,"members":["users/c@example.com"]}]}`, role)
	}
	create := func(workspace, projectID string, policyType storepb.Policy_Type, payload string) {
		_, err := stores.CreatePolicy(ctx, &store.PolicyMessage{
			Workspace:    workspace,
			ResourceType: storepb.Policy_PROJECT,
			Resource:     common.FormatProject(projectID),
			Type:         policyType,
			Payload:      payload,
			Enforce:      true,
		})
		require.NoError(t, err)
	}
	create("ws-a", "mine", storepb.Policy_IAM, iamPayload("roles/projectOwner"))
	// Same project, a different policy type: must not be read as an IAM policy.
	create("ws-a", "mine", storepb.Policy_ROLLOUT, `{"automatic":true}`)
	create("ws-a", "rollout-only", storepb.Policy_ROLLOUT, `{"automatic":true}`)
	create("ws-b", "other-workspace", storepb.Policy_IAM, iamPayload("roles/projectOwner"))

	got, err := stores.ListProjectIamPolicies(ctx, "ws-a",
		[]string{"mine", "rollout-only", "no-policy", "other-workspace"})
	require.NoError(t, err)

	// Only the one project in this workspace with an IAM policy.
	require.Len(t, got, 1)
	require.Contains(t, got, "mine")
	require.Len(t, got["mine"].GetBindings(), 1)
	require.Equal(t, "roles/projectOwner", got["mine"].GetBindings()[0].GetRole())

	// A project whose only policy is another type is absent, not misparsed.
	require.NotContains(t, got, "rollout-only")
	// A project with no policy at all is absent; callers treat that as empty.
	require.NotContains(t, got, "no-policy")
	// Another workspace's project is never reachable, even when named.
	require.NotContains(t, got, "other-workspace")

	// Naming only the foreign project returns nothing rather than its policy.
	foreign, err := stores.ListProjectIamPolicies(ctx, "ws-a", []string{"other-workspace"})
	require.NoError(t, err)
	require.Empty(t, foreign)

	// Reading it from its own workspace does work, so the empty result above is
	// the workspace filter and not a missing row.
	own, err := stores.ListProjectIamPolicies(ctx, "ws-b", []string{"other-workspace"})
	require.NoError(t, err)
	require.Len(t, own, 1)

	// No projects named is not "all projects".
	none, err := stores.ListProjectIamPolicies(ctx, "ws-a", nil)
	require.NoError(t, err)
	require.Empty(t, none)
}
