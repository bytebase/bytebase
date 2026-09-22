package store_test

import (
	"context"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/store"
)

const ownerAndDeveloperPolicy = `{"bindings":[` +
	`{"role":"roles/projectOwner","members":["users/owner@example.com"]},` +
	`{"role":"roles/projectDeveloper","members":["users/dev@example.com"]}]}`

func iamPolicy(t *testing.T, roleMembers map[string][]string) *storepb.IamPolicy {
	t.Helper()
	policy := &storepb.IamPolicy{}
	for role, members := range roleMembers {
		policy.Bindings = append(policy.Bindings, &storepb.Binding{Role: role, Members: members})
	}
	return policy
}

// projectPolicyMembers reads the members bound to one role of project-a's
// stored policy, in the workspace named.
func projectPolicyMembers(ctx context.Context, t *testing.T, s *store.Store, workspace, role string) []string {
	t.Helper()
	policy, err := s.GetProjectIamPolicy(ctx, workspace, "project-a")
	require.NoError(t, err)
	for _, binding := range policy.Policy.Bindings {
		if binding.Role == role {
			return binding.Members
		}
	}
	return nil
}

// Two admins editing permissions at once used to lose one edit: the etag was
// compared against a read taken before validation, outside the write
// transaction, so both writers passed the compare and the later one clobbered
// the other. If the lost edit was a revoke, access stayed granted.
func TestSetIamPolicyCompareAndSwap(t *testing.T) {
	t.Parallel()
	const seedSQL = `
		INSERT INTO policy (workspace, resource_type, resource, type, payload, inherit_from_parent)
			VALUES ('default', 'PROJECT', 'projects/project-a', 'IAM', '` + ownerAndDeveloperPolicy + `', FALSE);
	`
	fixture := newStorePostgresFixture(t, seedSQL)
	ctx, s := fixture.ctx, fixture.store
	resource := common.FormatProject("project-a")

	current, err := s.GetProjectIamPolicy(ctx, "default", "project-a")
	require.NoError(t, err)
	require.NotEmpty(t, current.Etag)

	// One admin revokes the developer, presenting the etag they read.
	revoked := iamPolicy(t, map[string][]string{"roles/projectOwner": {"users/owner@example.com"}})
	replaced, updated, err := s.SetIamPolicy(ctx, &store.SetIamPolicyMessage{
		Workspace:    "default",
		ResourceType: storepb.Policy_PROJECT,
		Resource:     resource,
		Policy:       revoked,
		ExpectedEtag: current.Etag,
	})
	require.NoError(t, err)
	require.Len(t, replaced.Policy.Bindings, 2, "the replaced policy is what the write actually overwrote")
	require.NotEqual(t, current.Etag, updated.Etag, "the etag must move with the write")

	// The second admin still holds the etag from before the revoke. Their write
	// would put the developer back, so it must be rejected, not applied.
	regranted := iamPolicy(t, map[string][]string{
		"roles/projectOwner":     {"users/owner@example.com"},
		"roles/projectDeveloper": {"users/dev@example.com", "users/dev2@example.com"},
	})
	_, _, err = s.SetIamPolicy(ctx, &store.SetIamPolicyMessage{
		Workspace:    "default",
		ResourceType: storepb.Policy_PROJECT,
		Resource:     resource,
		Policy:       regranted,
		ExpectedEtag: current.Etag,
	})
	require.ErrorIs(t, err, store.ErrIamPolicyEtagMismatch)
	require.Nil(t, projectPolicyMembers(ctx, t, s, "default", "roles/projectDeveloper"),
		"the revoke must survive the rejected write")

	// Refetching and reapplying is what the caller does next, and it lands.
	current, err = s.GetProjectIamPolicy(ctx, "default", "project-a")
	require.NoError(t, err)
	_, _, err = s.SetIamPolicy(ctx, &store.SetIamPolicyMessage{
		Workspace:    "default",
		ResourceType: storepb.Policy_PROJECT,
		Resource:     resource,
		Policy:       regranted,
		ExpectedEtag: current.Etag,
	})
	require.NoError(t, err)
	require.ElementsMatch(t, []string{"users/dev@example.com", "users/dev2@example.com"},
		projectPolicyMembers(ctx, t, s, "default", "roles/projectDeveloper"))

	// An empty etag is a caller that never read the policy, and asks for no
	// check -- the behavior every etag-less client has always had.
	_, _, err = s.SetIamPolicy(ctx, &store.SetIamPolicyMessage{
		Workspace:    "default",
		ResourceType: storepb.Policy_PROJECT,
		Resource:     resource,
		Policy:       revoked,
		ExpectedEtag: "",
	})
	require.NoError(t, err)
	require.Nil(t, projectPolicyMembers(ctx, t, s, "default", "roles/projectDeveloper"))
}

// The compare has to read the row it is about to write, not a cached copy.
// Nothing in a server deployment invalidates another process's cache, and the
// recovery CLI opens its own store against the same database, so a cache can
// disagree with the row -- and comparing against it would accept exactly the
// stale write the etag exists to reject.
func TestSetIamPolicyComparesAgainstTheRowNotTheCache(t *testing.T) {
	t.Parallel()
	const seedSQL = `
		INSERT INTO policy (workspace, resource_type, resource, type, payload, inherit_from_parent)
			VALUES ('default', 'PROJECT', 'projects/project-a', 'IAM', '` + ownerAndDeveloperPolicy + `', FALSE);
	`
	fixture := newStorePostgresFixture(t, seedSQL)
	ctx := fixture.ctx

	cached, err := store.New(ctx, fixture.pgURL, true)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, cached.Close()) })

	// Warm the cache, then write through a store that does not share it.
	stale, err := cached.GetProjectIamPolicy(ctx, "default", "project-a")
	require.NoError(t, err)
	require.NotEmpty(t, stale.Etag)

	_, _, err = fixture.store.SetIamPolicy(ctx, &store.SetIamPolicyMessage{
		Workspace:    "default",
		ResourceType: storepb.Policy_PROJECT,
		Resource:     common.FormatProject("project-a"),
		Policy:       iamPolicy(t, map[string][]string{"roles/projectOwner": {"users/owner@example.com"}}),
		ExpectedEtag: stale.Etag,
	})
	require.NoError(t, err)

	// The cached read still reports the pre-write etag, which is the whole
	// hazard; the write must reject it anyway.
	cachedAfter, err := cached.GetProjectIamPolicy(ctx, "default", "project-a")
	require.NoError(t, err)
	require.Equal(t, stale.Etag, cachedAfter.Etag, "the cache is expected to be stale here")

	_, _, err = cached.SetIamPolicy(ctx, &store.SetIamPolicyMessage{
		Workspace:    "default",
		ResourceType: storepb.Policy_PROJECT,
		Resource:     common.FormatProject("project-a"),
		Policy:       iamPolicy(t, map[string][]string{"roles/projectDeveloper": {"users/dev@example.com"}}),
		ExpectedEtag: stale.Etag,
	})
	require.ErrorIs(t, err, store.ErrIamPolicyEtagMismatch)

	// Rejecting is only half of it. The caller refetches to pick up the etag it
	// lost to, and being served the same stale one from here would conflict
	// again forever -- so the entry the write proved stale is gone.
	recovered, err := cached.GetProjectIamPolicy(ctx, "default", "project-a")
	require.NoError(t, err)
	require.NotEqual(t, stale.Etag, recovered.Etag, "the stale cache entry must not survive the conflict")
	require.Nil(t, projectPolicyMembers(ctx, t, cached, "default", "roles/projectDeveloper"))

	// And the retry that etag carries now lands.
	_, _, err = cached.SetIamPolicy(ctx, &store.SetIamPolicyMessage{
		Workspace:    "default",
		ResourceType: storepb.Policy_PROJECT,
		Resource:     common.FormatProject("project-a"),
		Policy:       iamPolicy(t, map[string][]string{"roles/projectDeveloper": {"users/dev@example.com"}}),
		ExpectedEtag: recovered.Etag,
	})
	require.NoError(t, err)
}

// A check that compares the new policy against the old one -- the workspace
// seat guard -- has to see the policy the write replaces. Reading it in the
// caller leaves a gap another request can land in, so two writes both pass a
// guard neither would have passed against what the other stored.
func TestSetIamPolicyValidatesAgainstTheReplacedPolicy(t *testing.T) {
	t.Parallel()
	const seedSQL = `
		INSERT INTO policy (workspace, resource_type, resource, type, payload, inherit_from_parent)
			VALUES ('default', 'PROJECT', 'projects/project-a', 'IAM', '` + ownerAndDeveloperPolicy + `', FALSE);
	`
	fixture := newStorePostgresFixture(t, seedSQL)
	ctx, s := fixture.ctx, fixture.store

	var seen []string
	rejected := errors.New("too many members")
	_, _, err := s.SetIamPolicy(ctx, &store.SetIamPolicyMessage{
		Workspace:    "default",
		ResourceType: storepb.Policy_PROJECT,
		Resource:     common.FormatProject("project-a"),
		Policy:       iamPolicy(t, map[string][]string{"roles/projectOwner": {"users/owner@example.com"}}),
		ValidateReplaced: func(previous *storepb.IamPolicy) error {
			for _, binding := range previous.Bindings {
				seen = append(seen, binding.Role)
			}
			return rejected
		},
	})
	require.ErrorIs(t, err, rejected)
	require.ElementsMatch(t, []string{"roles/projectOwner", "roles/projectDeveloper"}, seen,
		"the check must see the policy stored, not the one being written")
	require.Equal(t, []string{"users/dev@example.com"},
		projectPolicyMembers(ctx, t, s, "default", "roles/projectDeveloper"),
		"a rejected check must leave the policy untouched")
}

// policy is keyed by (workspace, resource_type, resource, type), and two
// workspaces name their projects independently -- so the same resource string
// exists in both. A write must touch only the workspace it named.
func TestSetIamPolicyIsScopedToItsWorkspace(t *testing.T) {
	t.Parallel()
	const seedSQL = `
		INSERT INTO workspace (resource_id) VALUES ('other');
		INSERT INTO policy (workspace, resource_type, resource, type, payload, inherit_from_parent)
			VALUES ('default', 'PROJECT', 'projects/project-a', 'IAM', '` + ownerAndDeveloperPolicy + `', FALSE),
			       ('other', 'PROJECT', 'projects/project-a', 'IAM', '` + ownerAndDeveloperPolicy + `', FALSE);
	`
	fixture := newStorePostgresFixture(t, seedSQL)
	ctx, s := fixture.ctx, fixture.store

	_, _, err := s.SetIamPolicy(ctx, &store.SetIamPolicyMessage{
		Workspace:    "default",
		ResourceType: storepb.Policy_PROJECT,
		Resource:     common.FormatProject("project-a"),
		Policy:       iamPolicy(t, map[string][]string{"roles/projectOwner": {"users/owner@example.com"}}),
	})
	require.NoError(t, err)

	require.Nil(t, projectPolicyMembers(ctx, t, s, "default", "roles/projectDeveloper"))
	require.Equal(t, []string{"users/dev@example.com"},
		projectPolicyMembers(ctx, t, s, "other", "roles/projectDeveloper"),
		"the other workspace's identically named project must be untouched")
}

// A resource whose IAM policy was never written has no row to lock, so the
// first write creates one rather than failing the compare.
func TestSetIamPolicyCreatesTheFirstPolicy(t *testing.T) {
	t.Parallel()
	fixture := newStorePostgresFixture(t, "")
	ctx, s := fixture.ctx, fixture.store

	absent, err := s.GetProjectIamPolicy(ctx, "default", "project-a")
	require.NoError(t, err)
	require.Empty(t, absent.Policy.Bindings)

	replaced, updated, err := s.SetIamPolicy(ctx, &store.SetIamPolicyMessage{
		Workspace:    "default",
		ResourceType: storepb.Policy_PROJECT,
		Resource:     common.FormatProject("project-a"),
		Policy:       iamPolicy(t, map[string][]string{"roles/projectOwner": {"users/owner@example.com"}}),
		ExpectedEtag: absent.Etag,
	})
	require.NoError(t, err)
	require.Empty(t, replaced.Policy.Bindings)
	require.NotEmpty(t, updated.Etag)
	require.Equal(t, []string{"users/owner@example.com"},
		projectPolicyMembers(ctx, t, s, "default", "roles/projectOwner"))
}

// PatchWorkspaceIamPolicy merges one member's roles rather than replacing the
// policy, and now does it inside the locked transaction. Its merge semantics
// must be unchanged by that move.
func TestPatchWorkspaceIamPolicyMerges(t *testing.T) {
	t.Parallel()
	fixture := newStorePostgresFixture(t, "")
	ctx, s := fixture.ctx, fixture.store

	_, err := s.PatchWorkspaceIamPolicy(ctx, &store.PatchIamPolicyMessage{
		Workspace: "default",
		Member:    common.FormatUserEmail("first@example.com"),
		Roles:     []string{"roles/workspaceAdmin"},
	})
	require.NoError(t, err)

	// A second member joining must not displace the first.
	policy, err := s.PatchWorkspaceIamPolicy(ctx, &store.PatchIamPolicyMessage{
		Workspace: "default",
		Member:    common.FormatUserEmail("second@example.com"),
		Roles:     []string{"roles/workspaceMember"},
	})
	require.NoError(t, err)

	members := map[string][]string{}
	for _, binding := range policy.Policy.Bindings {
		members[binding.Role] = binding.Members
	}
	require.Equal(t, []string{"users/first@example.com"}, members["roles/workspaceAdmin"])
	require.Equal(t, []string{"users/second@example.com"}, members["roles/workspaceMember"])

	// Setting a member's roles to a different set removes the ones dropped.
	_, err = s.PatchWorkspaceIamPolicy(ctx, &store.PatchIamPolicyMessage{
		Workspace: "default",
		Member:    common.FormatUserEmail("first@example.com"),
		Roles:     []string{"roles/workspaceMember"},
	})
	require.NoError(t, err)

	stored, err := s.GetWorkspaceIamPolicy(ctx, "default")
	require.NoError(t, err)
	members = map[string][]string{}
	for _, binding := range stored.Policy.Bindings {
		members[binding.Role] = binding.Members
	}
	require.Empty(t, members["roles/workspaceAdmin"])
	require.ElementsMatch(t, []string{"users/second@example.com", "users/first@example.com"},
		members["roles/workspaceMember"])
}

// A merge that reads the policy before the write -- through a cache another
// node never invalidates -- writes back a policy missing whatever that node
// changed. Approving a role grant while an admin revokes someone in the same
// project put the revoked member back.
func TestPatchIamPolicyMergesIntoTheStoredPolicy(t *testing.T) {
	t.Parallel()
	const seedSQL = `
		INSERT INTO policy (workspace, resource_type, resource, type, payload, inherit_from_parent)
			VALUES ('default', 'PROJECT', 'projects/project-a', 'IAM', '` + ownerAndDeveloperPolicy + `', FALSE);
	`
	fixture := newStorePostgresFixture(t, seedSQL)
	ctx := fixture.ctx
	resource := common.FormatProject("project-a")

	merging, err := store.New(ctx, fixture.pgURL, true)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, merging.Close()) })

	// This node has read the policy and cached it. Another node then revokes the
	// developer.
	_, err = merging.GetProjectIamPolicy(ctx, "default", "project-a")
	require.NoError(t, err)

	_, _, err = fixture.store.SetIamPolicy(ctx, &store.SetIamPolicyMessage{
		Workspace:    "default",
		ResourceType: storepb.Policy_PROJECT,
		Resource:     resource,
		Policy:       iamPolicy(t, map[string][]string{"roles/projectOwner": {"users/owner@example.com"}}),
	})
	require.NoError(t, err)

	// The merge must land on the revoked policy, not on the cached one.
	_, err = merging.PatchIamPolicy(ctx, "default", storepb.Policy_PROJECT, resource,
		func(policy *storepb.IamPolicy) error {
			policy.Bindings = append(policy.Bindings, &storepb.Binding{
				Role:    "roles/projectReleaser",
				Members: []string{"users/releaser@example.com"},
			})
			return nil
		})
	require.NoError(t, err)

	require.Equal(t, []string{"users/releaser@example.com"},
		projectPolicyMembers(ctx, t, fixture.store, "default", "roles/projectReleaser"))
	require.Nil(t, projectPolicyMembers(ctx, t, fixture.store, "default", "roles/projectDeveloper"),
		"the merge must not resurrect the revoked member")
}

// A mutation that fails leaves the policy alone rather than committing a
// half-applied merge.
func TestPatchIamPolicyRollsBackAFailedMutation(t *testing.T) {
	t.Parallel()
	const seedSQL = `
		INSERT INTO policy (workspace, resource_type, resource, type, payload, inherit_from_parent)
			VALUES ('default', 'PROJECT', 'projects/project-a', 'IAM', '` + ownerAndDeveloperPolicy + `', FALSE);
	`
	fixture := newStorePostgresFixture(t, seedSQL)
	ctx, s := fixture.ctx, fixture.store

	boom := errors.New("boom")
	_, err := s.PatchIamPolicy(ctx, "default", storepb.Policy_PROJECT, common.FormatProject("project-a"),
		func(policy *storepb.IamPolicy) error {
			policy.Bindings = nil
			return boom
		})
	require.ErrorIs(t, err, boom)

	require.Equal(t, []string{"users/dev@example.com"},
		projectPolicyMembers(ctx, t, s, "default", "roles/projectDeveloper"))
}

// A write must not publish what it just wrote into the cache. The advisory lock
// ends with the commit, so two writers can commit in one order and reach the
// cache in the other, and the loser's snapshot would then sit in a cache with no
// expiry -- which is what permission checks read. Observed here without racing
// goroutines: after a write, the row is changed out of band, and a read that
// still saw the written value would prove the write had published it.
func TestSetIamPolicyDoesNotPublishItsOwnWrite(t *testing.T) {
	t.Parallel()
	const seedSQL = `
		INSERT INTO policy (workspace, resource_type, resource, type, payload, inherit_from_parent)
			VALUES ('default', 'PROJECT', 'projects/project-a', 'IAM', '` + ownerAndDeveloperPolicy + `', FALSE);
	`
	fixture := newStorePostgresFixture(t, seedSQL)
	ctx := fixture.ctx

	cached, err := store.New(ctx, fixture.pgURL, true)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, cached.Close()) })

	_, _, err = cached.SetIamPolicy(ctx, &store.SetIamPolicyMessage{
		Workspace:    "default",
		ResourceType: storepb.Policy_PROJECT,
		Resource:     common.FormatProject("project-a"),
		Policy:       iamPolicy(t, map[string][]string{"roles/projectOwner": {"users/owner@example.com"}}),
	})
	require.NoError(t, err)

	// Stand in for the writer that committed second: the row moves on without
	// this process's cache being told.
	_, err = fixture.db.ExecContext(ctx, `
		UPDATE policy SET payload = $1, updated_at = now()
		WHERE workspace = 'default' AND resource_type = 'PROJECT'
		  AND resource = 'projects/project-a' AND type = 'IAM'`,
		`{"bindings":[{"role":"roles/projectReleaser","members":["users/releaser@example.com"]}]}`)
	require.NoError(t, err)

	require.Equal(t, []string{"users/releaser@example.com"},
		projectPolicyMembers(ctx, t, cached, "default", "roles/projectReleaser"),
		"the write must leave nothing cached, so this read reaches the row")
	require.Nil(t, projectPolicyMembers(ctx, t, cached, "default", "roles/projectOwner"))
}
