package store_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// A batch purge is one transaction. The loop it replaced purged project by
// project, so a project that failed its archived-state guard left every project
// before it irreversibly gone. These lock down that the batch is all-or-nothing
// and that it still reaches every named project's rows.

// seedPurgeableProjects adds two more archived projects (project-b, project-c)
// beside the fixture's project-a, plus one webhook, one database group, one
// query history row and one project policy for each of the three and for the
// surviving default project.
const seedPurgeableProjects = `
	INSERT INTO project (resource_id, workspace, name, deleted) VALUES
		('project-b', 'default', 'Project B', TRUE),
		('project-c', 'default', 'Project C', TRUE);
	INSERT INTO project_webhook (resource_id, project) VALUES
		('hook-a', 'project-a'), ('hook-b', 'project-b'),
		('hook-c', 'project-c'), ('hook-default', 'default');
	INSERT INTO db_group (project, resource_id) VALUES
		('project-a', 'group'), ('project-b', 'group'),
		('project-c', 'group'), ('default', 'group');
	INSERT INTO query_history (resource_id, creator, project, database, statement, type) VALUES
		('qh-a', 'user@example.com', 'project-a', 'instances/i/databases/d', 'SELECT 1', 'QUERY'),
		('qh-b', 'user@example.com', 'project-b', 'instances/i/databases/d', 'SELECT 1', 'QUERY'),
		('qh-c', 'user@example.com', 'project-c', 'instances/i/databases/d', 'SELECT 1', 'QUERY'),
		('qh-default', 'user@example.com', 'default', 'instances/i/databases/d', 'SELECT 1', 'QUERY');
	INSERT INTO policy (workspace, resource_type, resource, type) VALUES
		('default', 'PROJECT', 'projects/project-a', 'IAM'),
		('default', 'PROJECT', 'projects/project-b', 'IAM'),
		('default', 'PROJECT', 'projects/project-c', 'IAM'),
		('default', 'PROJECT', 'projects/default', 'IAM');
`

// projectRowCounts reports how many rows each purged table still holds for a
// project, so a test can assert the whole set survived or the whole set went.
func projectRowCounts(t *testing.T, fixture *storePostgresFixture, projectID string) map[string]int {
	t.Helper()
	counts := map[string]int{}
	for table, query := range map[string]string{
		"project":         "SELECT count(*) FROM project WHERE resource_id = $1",
		"project_webhook": "SELECT count(*) FROM project_webhook WHERE project = $1",
		"db_group":        "SELECT count(*) FROM db_group WHERE project = $1",
		"query_history":   "SELECT count(*) FROM query_history WHERE project = $1",
		"policy":          "SELECT count(*) FROM policy WHERE resource = 'projects/' || $1",
	} {
		var count int
		require.NoError(t, fixture.db.QueryRowContext(fixture.ctx, query, projectID).Scan(&count))
		counts[table] = count
	}
	return counts
}

func TestDeleteProjectsPurgesEveryNamedProject(t *testing.T) {
	fixture := newStorePostgresFixture(t, seedPurgeableProjects)

	require.NoError(t, fixture.store.DeleteProjects(fixture.ctx, "default", "project-a", "project-c"))

	for _, projectID := range []string{"project-a", "project-c"} {
		for table, count := range projectRowCounts(t, fixture, projectID) {
			require.Zero(t, count, "%s rows for purged project %s", table, projectID)
		}
	}
	// The projects that were not named keep every row, including the policy
	// rows the purge selects by rendered resource name.
	for _, projectID := range []string{"project-b", "default"} {
		for table, count := range projectRowCounts(t, fixture, projectID) {
			require.Equal(t, 1, count, "%s rows for surviving project %s", table, projectID)
		}
	}
}

func TestDeleteProjectsPurgeRollsBackWhenOneProjectIsNotArchived(t *testing.T) {
	fixture := newStorePostgresFixture(t, seedPurgeableProjects+`
		UPDATE project SET deleted = FALSE WHERE resource_id = 'project-c';
	`)

	err := fixture.store.DeleteProjects(fixture.ctx, "default", "project-a", "project-b", "project-c")
	require.ErrorContains(t, err, "project project-c not found or not marked as deleted")

	// project-a and project-b are archived and sort before project-c, so the
	// loop this replaced would have purged both before reaching the failure.
	for _, projectID := range []string{"project-a", "project-b", "project-c", "default"} {
		for table, count := range projectRowCounts(t, fixture, projectID) {
			require.Equal(t, 1, count, "%s rows for project %s after the rolled-back purge", table, projectID)
		}
	}
}

func TestDeleteProjectsPurgeRollsBackWhenOneProjectIsMissing(t *testing.T) {
	fixture := newStorePostgresFixture(t, seedPurgeableProjects)

	err := fixture.store.DeleteProjects(fixture.ctx, "default", "project-a", "project-missing")
	require.ErrorContains(t, err, "project project-missing not found or not marked as deleted")

	for table, count := range projectRowCounts(t, fixture, "project-a") {
		require.Equal(t, 1, count, "%s rows for project-a after the rolled-back purge", table)
	}
}

// A name repeated in one batch must purge once rather than fail the
// purged-row count against the number of names given.
func TestDeleteProjectsPurgeDedupesRepeatedNames(t *testing.T) {
	fixture := newStorePostgresFixture(t, seedPurgeableProjects)

	require.NoError(t, fixture.store.DeleteProjects(fixture.ctx, "default", "project-a", "project-a"))

	for table, count := range projectRowCounts(t, fixture, "project-a") {
		require.Zero(t, count, "%s rows for purged project-a", table)
	}
}

// TestDeleteProjectsConcurrentBatchesLockOrder pins the terminal outcomes when
// two batch purges name the same projects in opposite order. Both sort their
// names, so they take the project row locks in the same order and neither
// deadlocks: the first commits and the second finds nothing left to purge.
func TestDeleteProjectsConcurrentBatchesLockOrder(t *testing.T) {
	fixture := newStorePostgresFixture(t, seedPurgeableProjects)
	const barrierID = 9971
	barrier := newMaintenanceLockBarrier(fixture.ctx, t, fixture.db, barrierID)
	// The purge locks the project rows just before this statement, so the first
	// batch parks here holding them.
	installMaintenanceLockBarrier(t, fixture.db, barrierID,
		"BEFORE DELETE ON project FOR EACH STATEMENT")

	firstResult := make(chan error, 1)
	go func() {
		firstResult <- fixture.store.DeleteProjects(fixture.ctx, "default", "project-a", "project-b")
	}()
	waitForMaintenanceBarrier(fixture.ctx, t, fixture.db, barrierID)
	firstPID := maintenanceBarrierWaitingPID(fixture.ctx, t, fixture.db, barrierID)

	secondResult := make(chan error, 1)
	go func() {
		secondResult <- fixture.store.DeleteProjects(fixture.ctx, "default", "project-b", "project-a")
	}()
	waitForBackendBlockedByPID(fixture.ctx, t, fixture.db, firstPID)
	barrier.release(t)

	requireMaintenanceResult(t, firstResult)
	select {
	case err := <-secondResult:
		require.ErrorContains(t, err, "not found or not marked as deleted")
	case <-time.After(maintenanceLockWait):
		t.Fatal("timed out waiting for the second batch purge")
	}

	for _, projectID := range []string{"project-a", "project-b"} {
		for table, count := range projectRowCounts(t, fixture, projectID) {
			require.Zero(t, count, "%s rows for purged project %s", table, projectID)
		}
	}
	for table, count := range projectRowCounts(t, fixture, "project-c") {
		require.Equal(t, 1, count, "%s rows for untouched project-c", table)
	}
}
