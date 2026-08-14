package store_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/store"
)

// The bindings column holds a protojson array at the jsonb root, not a wrapper
// object. That shape is load-bearing: the access queries probe it with `@>`
// against the GIN index, and a wrapper would need an expression index instead.
// Nothing else asserts the shape directly, so a change to it would surface as
// sharing silently matching no rows.
func TestSavedQueryBindingsStoredShape(t *testing.T) {
	const seedSQL = `
		INSERT INTO saved_query (resource_id, creator, project, name, statement)
			VALUES ('saved-query-a', 'owner@example.com', 'project-a', 'Saved Query A', 'SELECT 1;');
	`
	fixture := newProjectDeletionLockOrderFixture(t, seedSQL)

	bindings := []*storepb.SavedQueryBinding{{
		Level:   storepb.SavedQueryBinding_EDITOR,
		Members: []string{"group:eng@example.com", "user:grantee@example.com"},
	}}
	etag, err := store.SavedQueryPolicyEtag(nil)
	require.NoError(t, err)

	applied, err := fixture.store.SetSavedQueryBindings(fixture.ctx, "project-a", "saved-query-a", bindings, etag)
	require.NoError(t, err)
	require.True(t, applied)

	// A jsonb array at the root, so `jsonb_typeof` sees an array.
	var typ string
	require.NoError(t, fixture.db.QueryRowContext(fixture.ctx,
		"SELECT jsonb_typeof(bindings) FROM saved_query WHERE resource_id = 'saved-query-a'").Scan(&typ))
	require.Equal(t, "array", typ)

	// The containment probe the access clause builds must match a member listed
	// alongside others, for both principal types.
	for _, probe := range []string{
		`[{"members":["user:grantee@example.com"]}]`,
		`[{"members":["group:eng@example.com"]}]`,
	} {
		var matched bool
		require.NoError(t, fixture.db.QueryRowContext(fixture.ctx,
			"SELECT bindings @> $1::jsonb FROM saved_query WHERE resource_id = 'saved-query-a'", probe).Scan(&matched))
		require.True(t, matched, "probe %s must match the stored bindings", probe)
	}

	// A principal named by no binding must not match.
	var strangerMatched bool
	require.NoError(t, fixture.db.QueryRowContext(fixture.ctx,
		"SELECT bindings @> $1::jsonb FROM saved_query WHERE resource_id = 'saved-query-a'",
		`[{"members":["user:stranger@example.com"]}]`).Scan(&strangerMatched))
	require.False(t, strangerMatched)

	// Compare-and-swap: the etag moved with the write, so replaying the old one
	// is rejected rather than clobbering the grants just stored.
	_, err = fixture.store.SetSavedQueryBindings(fixture.ctx, "project-a", "saved-query-a", nil, etag)
	require.ErrorIs(t, err, store.ErrSavedQueryEtagMismatch)

	newEtag, err := store.SavedQueryPolicyEtag(bindings)
	require.NoError(t, err)
	applied, err = fixture.store.SetSavedQueryBindings(fixture.ctx, "project-a", "saved-query-a", nil, newEtag)
	require.NoError(t, err)
	require.True(t, applied)

	// Revoking everything leaves an empty array, which reads back as no grants
	// and as the same etag an untouched saved query has.
	require.NoError(t, fixture.db.QueryRowContext(fixture.ctx,
		"SELECT jsonb_typeof(bindings) FROM saved_query WHERE resource_id = 'saved-query-a'").Scan(&typ))
	require.Equal(t, "array", typ)

	// A saved query that no longer exists reports "not applied" rather than an
	// error, so the caller decides what a vanished row means.
	applied, err = fixture.store.SetSavedQueryBindings(fixture.ctx, "project-a", "missing", nil, etag)
	require.NoError(t, err)
	require.False(t, applied)

	// The write is scoped to the project the caller named. A purge re-parents
	// its members' saved queries to the default project, so a request resolved
	// against the old project must land on nothing rather than on a row that
	// moved out from under it.
	_, err = fixture.db.ExecContext(fixture.ctx,
		"UPDATE saved_query SET project = 'default' WHERE resource_id = 'saved-query-a'")
	require.NoError(t, err)
	applied, err = fixture.store.SetSavedQueryBindings(fixture.ctx, "project-a", "saved-query-a", bindings, newEtag)
	require.NoError(t, err)
	require.False(t, applied, "a re-parented saved query is gone from the project the caller named")
}

// A group is named in a binding by its email, so renaming one has to carry its
// grants along or every saved query shared with it becomes unreachable to its
// members.
func TestSavedQueryBindingsFollowGroupRename(t *testing.T) {
	const seedSQL = `
		INSERT INTO user_group (id, email, workspace, name, description, payload)
			VALUES ('group-1', 'eng@example.com', 'default', 'Eng', '', '{}');
		INSERT INTO saved_query (resource_id, creator, project, name, statement)
			VALUES ('saved-query-a', 'owner@example.com', 'project-a', 'Saved Query A', 'SELECT 1;');
	`
	fixture := newProjectDeletionLockOrderFixture(t, seedSQL)

	emptyEtag, err := store.SavedQueryPolicyEtag(nil)
	require.NoError(t, err)
	applied, err := fixture.store.SetSavedQueryBindings(fixture.ctx, "project-a", "saved-query-a",
		[]*storepb.SavedQueryBinding{{
			Level:   storepb.SavedQueryBinding_EDITOR,
			Members: []string{"group:eng@example.com", "user:someone@example.com"},
		}}, emptyEtag)
	require.NoError(t, err)
	require.True(t, applied)

	newEmail := "engineering@example.com"
	_, err = fixture.store.UpdateGroup(fixture.ctx, &store.UpdateGroupMessage{
		ID:        "group-1",
		Workspace: "default",
		Email:     &newEmail,
	})
	require.NoError(t, err)

	// The renamed group's grant follows it; the untouched user member does not move.
	var matchedNew, matchedOld, matchedUser bool
	require.NoError(t, fixture.db.QueryRowContext(fixture.ctx, `
		SELECT bindings @> '[{"members":["group:engineering@example.com"]}]'::jsonb,
		       bindings @> '[{"members":["group:eng@example.com"]}]'::jsonb,
		       bindings @> '[{"members":["user:someone@example.com"]}]'::jsonb
		FROM saved_query WHERE resource_id = 'saved-query-a'
	`).Scan(&matchedNew, &matchedOld, &matchedUser))
	require.True(t, matchedNew, "the grant must follow the rename")
	require.False(t, matchedOld, "the stale token must be gone")
	require.True(t, matchedUser, "unrelated members must be untouched")
}
