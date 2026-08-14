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

	applied, err := fixture.store.SetSavedQueryBindings(fixture.ctx, "saved-query-a", bindings, etag)
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
	_, err = fixture.store.SetSavedQueryBindings(fixture.ctx, "saved-query-a", nil, etag)
	require.ErrorIs(t, err, store.ErrSavedQueryEtagMismatch)

	newEtag, err := store.SavedQueryPolicyEtag(bindings)
	require.NoError(t, err)
	applied, err = fixture.store.SetSavedQueryBindings(fixture.ctx, "saved-query-a", nil, newEtag)
	require.NoError(t, err)
	require.True(t, applied)

	// Revoking everything leaves an empty array, which reads back as no grants
	// and as the same etag an untouched saved query has.
	require.NoError(t, fixture.db.QueryRowContext(fixture.ctx,
		"SELECT jsonb_typeof(bindings) FROM saved_query WHERE resource_id = 'saved-query-a'").Scan(&typ))
	require.Equal(t, "array", typ)

	// A saved query that no longer exists reports "not applied" rather than an
	// error, so the caller decides what a vanished row means.
	applied, err = fixture.store.SetSavedQueryBindings(fixture.ctx, "missing", nil, etag)
	require.NoError(t, err)
	require.False(t, applied)
}
