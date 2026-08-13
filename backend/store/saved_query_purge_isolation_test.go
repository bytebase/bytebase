package store_test

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// saved_query_star is keyed (saved_query, principal) and its parent key is
// globally unique, so it carries no project-scoped id-reuse risk. What it does
// carry is the purge: DeleteProject is the one writer whose star predicate
// reaches outside the project being purged, because a project principal may
// have starred another project's saved query. This locks down that reach —
// rows belonging to a surviving project must not be collateral.

func TestDeleteProjectPurgeKeepsOtherProjectSavedQueryStars(t *testing.T) {
	const seedSQL = `
		INSERT INTO project (resource_id, workspace, name) VALUES ('project-b', 'default', 'Project B');
		INSERT INTO service_account (name, email, workspace, service_key_hash, project)
			VALUES ('SA A', 'sa-a@example.com', 'default', '', 'project-a'),
			       ('SA B', 'sa-b@example.com', 'default', '', 'project-b');
		INSERT INTO saved_query (resource_id, creator, project, name, statement) VALUES
			('sq-a-sa',   'sa-a@example.com',   'project-a', 'A by SA A',   ''),
			('sq-a-user', 'user@example.com',   'project-a', 'A by user',   ''),
			('sq-b-sa',   'sa-b@example.com',   'project-b', 'B by SA B',   ''),
			('sq-b-user', 'user@example.com',   'project-b', 'B by user',   '');
		INSERT INTO saved_query_star (saved_query, principal) VALUES
			('sq-a-sa',   'sa-a@example.com'),
			('sq-a-user', 'user@example.com'),
			('sq-b-sa',   'sa-a@example.com'),
			('sq-b-sa',   'sa-b@example.com'),
			('sq-b-user', 'user@example.com');
	`

	fixture := newProjectDeletionLockOrderFixture(t, seedSQL)
	require.NoError(t, fixture.store.DeleteProject(fixture.ctx, "default", "project-a"))

	savedQueryExists := func(resourceID string) bool {
		var exists bool
		require.NoError(t, fixture.db.QueryRowContext(fixture.ctx,
			"SELECT EXISTS (SELECT 1 FROM saved_query WHERE resource_id = $1)", resourceID).Scan(&exists))
		return exists
	}
	starExists := func(savedQuery, principal string) bool {
		var exists bool
		require.NoError(t, fixture.db.QueryRowContext(fixture.ctx,
			"SELECT EXISTS (SELECT 1 FROM saved_query_star WHERE saved_query = $1 AND principal = $2)",
			savedQuery, principal).Scan(&exists))
		return exists
	}

	// Project B is untouched: its saved queries, and the stars its own
	// principals hold on them, all survive.
	require.True(t, savedQueryExists("sq-b-sa"), "project B saved query was purged with project A")
	require.True(t, savedQueryExists("sq-b-user"))
	require.True(t, starExists("sq-b-sa", "sa-b@example.com"), "project B principal's star was purged with project A")
	require.True(t, starExists("sq-b-user", "user@example.com"), "a user's star on project B was purged with project A")

	// Project A's own rows go: the SA-created query with it, and the
	// user-created one is re-parented to the default project rather than
	// deleted, so its star has to survive with it.
	require.False(t, savedQueryExists("sq-a-sa"))
	require.False(t, starExists("sq-a-sa", "sa-a@example.com"))
	require.True(t, savedQueryExists("sq-a-user"))
	require.True(t, starExists("sq-a-user", "user@example.com"))

	var reparented string
	require.NoError(t, fixture.db.QueryRowContext(fixture.ctx,
		"SELECT project FROM saved_query WHERE resource_id = 'sq-a-user'").Scan(&reparented))
	require.Equal(t, "default", reparented)

	// The purged project's service account is gone, so the star it held on
	// project B's saved query goes with it — that is the one cross-project
	// deletion the purge is allowed to make.
	require.False(t, starExists("sq-b-sa", "sa-a@example.com"))
}
