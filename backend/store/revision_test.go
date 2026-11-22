package store_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/common/testcontainer"
	"github.com/bytebase/bytebase/backend/store"
)

// instance-a/app has 001 live, 002 deleted twice, 003 never applied and 004
// only as a declarative revision. Its sibling has 003 live, and instance-b
// holds another app with 003 live, so a query that ignored the database or
// the instance would count those.
const pendingReleaseVersionsFixtureSQL = `
	INSERT INTO workspace (resource_id) VALUES ('default');
	INSERT INTO project (resource_id, workspace, name) VALUES ('project-a', 'default', 'Project A');
	INSERT INTO instance (resource_id, workspace) VALUES ('instance-a', 'default'), ('instance-b', 'default');
	INSERT INTO db (instance, name, project) VALUES
		('instance-a', 'app', 'project-a'),
		('instance-a', 'sibling', 'project-a'),
		('instance-b', 'app', 'project-a');
	INSERT INTO revision (instance, db_name, version, payload, deleter, deleted_at) VALUES
		('instance-a', 'app', '001', '{"type":"VERSIONED"}', NULL, NULL),
		('instance-a', 'app', '002', '{"type":"VERSIONED"}', 'old@example.com', '2026-10-01T08:00:00Z'),
		('instance-a', 'app', '002', '{"type":"VERSIONED"}', 'alex@example.com', '2026-10-05T09:30:00Z'),
		('instance-a', 'app', '004', '{"type":"DECLARATIVE"}', NULL, NULL),
		('instance-a', 'sibling', '003', '{"type":"VERSIONED"}', NULL, NULL),
		('instance-b', 'app', '003', '{"type":"VERSIONED"}', NULL, NULL);
`

func TestListPendingReleaseVersionsReturnsOnlyWhatEachDatabaseLacks(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	db, s, _ := testcontainer.NewMetadataDB(t)
	_, err := db.ExecContext(ctx, pendingReleaseVersionsFixtureSQL)
	require.NoError(t, err)

	versions := []string{"001", "002", "003", "004"}
	pendingOf := func(instanceID string, names []string) map[string]*store.PendingReleaseVersion {
		t.Helper()
		pending, err := s.ListPendingReleaseVersions(ctx, instanceID, names, versions)
		require.NoError(t, err)
		got := map[string]*store.PendingReleaseVersion{}
		for _, p := range pending {
			got[p.DatabaseName+"/"+p.Version] = p
		}
		return got
	}

	// A live VERSIONED revision is applied; a deleted or declarative one is
	// not, as in the executor. The sibling's live 003 is its own.
	a := pendingOf("instance-a", []string{"app", "sibling"})
	require.ElementsMatch(t,
		[]string{"app/002", "app/003", "app/004", "sibling/001", "sibling/002", "sibling/004"},
		keysOf(a))
	require.NotNil(t, a["app/002"].DeletedAt)
	require.Equal(t, time.Date(2026, 10, 5, 9, 30, 0, 0, time.UTC), a["app/002"].DeletedAt.UTC())
	require.Equal(t, "alex@example.com", a["app/002"].Deleter)
	require.Nil(t, a["app/003"].DeletedAt)
	require.Empty(t, a["app/003"].Deleter)

	// The other instance's app has its own revisions, and a database the
	// list leaves out is not returned.
	require.ElementsMatch(t, []string{"app/001", "app/002", "app/004"}, keysOf(pendingOf("instance-b", []string{"app"})))
	require.ElementsMatch(t, []string{"sibling/001", "sibling/002", "sibling/004"}, keysOf(pendingOf("instance-a", []string{"sibling"})))
}

func keysOf(m map[string]*store.PendingReleaseVersion) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}
