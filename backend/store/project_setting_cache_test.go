package store_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/store"
)

// A project setting decides authorization — AllowLastPlanEditorApproval gates
// who may approve an issue — and GetProject serves it from cache on the
// single-process deployments that enable caching. Invalidating the cache
// before the write leaves a window in which a concurrent reader loads the
// pre-update row and caches it, so the stale setting outlives the write and
// every later authorization check runs on it.
func TestUpdateProjectsDoesNotLeaveAStaleSettingCached(t *testing.T) {
	fixture := newStorePostgresFixture(t, `
		UPDATE project SET setting = '{"allowLastPlanEditorApproval": true}' WHERE resource_id = 'project-a';
	`)
	cached, err := store.New(fixture.ctx, fixture.pgURL, true)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, cached.Close()) })

	projectID := "project-a"
	readSetting := func() bool {
		project, err := cached.GetProject(fixture.ctx, &store.FindProjectMessage{Workspace: "default", ResourceID: &projectID, ShowDeleted: true})
		require.NoError(t, err)
		require.NotNil(t, project)
		return project.Setting.GetAllowLastPlanEditorApproval()
	}
	require.True(t, readSetting(), "seeded setting")

	const barrierID = 9981
	barrier := newMaintenanceLockBarrier(fixture.ctx, t, fixture.db, barrierID)
	installMaintenanceLockBarrier(t, fixture.db, barrierID, "BEFORE UPDATE ON project FOR EACH STATEMENT")

	updateDone := make(chan error, 1)
	go func() {
		updateDone <- cached.UpdateProjects(fixture.ctx, &store.UpdateProjectMessage{
			Workspace:  "default",
			ResourceID: projectID,
			Setting:    &storepb.Project{AllowLastPlanEditorApproval: false},
		})
	}()
	waitForMaintenanceBarrier(fixture.ctx, t, fixture.db, barrierID)

	// A reader lands while the write is in flight and caches the pre-update row.
	require.True(t, readSetting(), "the in-flight write is not visible yet")

	barrier.release(t)
	requireMaintenanceResult(t, updateDone)

	require.False(t, readSetting(),
		"the committed setting must be readable; a cache entry from the write window must not outlive it")
}
