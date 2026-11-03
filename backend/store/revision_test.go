package store_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/common/testcontainer"
	"github.com/bytebase/bytebase/backend/store"
)

// Two instances each hold a database named app with the same version, and
// instance-a also has a sibling, so a filter that ignored the instance or the
// listed names would return the other scope's rows.
const revisionNamesFixtureSQL = `
	INSERT INTO workspace (resource_id) VALUES ('default');
	INSERT INTO project (resource_id, workspace, name) VALUES ('project-a', 'default', 'Project A');
	INSERT INTO instance (resource_id, workspace) VALUES ('instance-a', 'default'), ('instance-b', 'default');
	INSERT INTO db (instance, name, project) VALUES
		('instance-a', 'app', 'project-a'),
		('instance-a', 'sibling', 'project-a'),
		('instance-a', 'other', 'project-a'),
		('instance-b', 'app', 'project-a');
	INSERT INTO revision (instance, db_name, version, payload) VALUES
		('instance-a', 'app', '001', '{"type":"VERSIONED"}'),
		('instance-a', 'sibling', '001', '{"type":"VERSIONED"}'),
		('instance-a', 'other', '001', '{"type":"VERSIONED"}'),
		('instance-b', 'app', '001', '{"type":"VERSIONED"}');
`

func TestListRevisionsByDatabaseNamesStaysInItsInstance(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	db, s, _ := testcontainer.NewMetadataDB(t)
	_, err := db.ExecContext(ctx, revisionNamesFixtureSQL)
	require.NoError(t, err)

	scopes := func(instanceID string, names []string) []string {
		revisions, err := s.ListRevisions(ctx, &store.FindRevisionMessage{
			InstanceID:    instanceID,
			DatabaseNames: &names,
		})
		require.NoError(t, err)
		var got []string
		for _, r := range revisions {
			got = append(got, r.InstanceID+"/"+r.DatabaseName)
		}
		return got
	}

	// Both scopes hold app; each list returns its own.
	require.ElementsMatch(t, []string{"instance-a/app", "instance-a/sibling"}, scopes("instance-a", []string{"app", "sibling"}))
	require.Equal(t, []string{"instance-b/app"}, scopes("instance-b", []string{"app"}))
	// A name the list leaves out is not returned.
	require.Equal(t, []string{"instance-a/app"}, scopes("instance-a", []string{"app"}))
}
