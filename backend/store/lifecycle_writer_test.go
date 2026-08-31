package store_test

import (
	"context"
	"database/sql"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/store"
)

func holdExclusiveLifecycleGate(ctx context.Context, t *testing.T, db *sql.DB, namespace store.AdvisoryLockKey, resourceID string) func() {
	t.Helper()
	conn, err := db.Conn(ctx)
	require.NoError(t, err)
	_, err = conn.ExecContext(ctx, "SELECT pg_advisory_lock($1, hashtext($2))", int64(namespace), resourceID)
	require.NoError(t, err)

	released := false
	release := func() {
		if released {
			return
		}
		released = true
		_, err := conn.ExecContext(context.Background(), "SELECT pg_advisory_unlock($1, hashtext($2))", int64(namespace), resourceID)
		require.NoError(t, err)
		require.NoError(t, conn.Close())
	}
	t.Cleanup(release)
	return release
}

func TestBatchUpdateIssueStatusesFailsWhenProjectLifecycleGateHeld(t *testing.T) {
	fixture := newProjectDeletionLockOrderFixture(t, "UPDATE project SET deleted = FALSE WHERE resource_id = 'project-a';")
	issue, err := fixture.store.CreateIssue(fixture.ctx, &store.IssueMessage{
		ProjectID:    "project-a",
		CreatorEmail: "creator@example.com",
		Title:        "issue",
		Type:         storepb.Issue_DATABASE_CHANGE,
		Payload:      &storepb.Issue{},
	})
	require.NoError(t, err)

	release := holdExclusiveLifecycleGate(fixture.ctx, t, fixture.db, store.AdvisoryLockKeyProjectLifecycle, "project-a")
	_, err = fixture.store.BatchUpdateIssueStatuses(fixture.ctx, "project-a", []int64{issue.UID}, storepb.Issue_DONE)
	require.ErrorIs(t, err, store.ErrLifecycleBusy)
	release()

	current, err := fixture.store.GetIssue(fixture.ctx, &store.FindIssueMessage{ProjectIDs: []string{"project-a"}, UID: &issue.UID})
	require.NoError(t, err)
	require.Equal(t, storepb.Issue_OPEN, current.Status)
}

func TestUpdateProjectsRequiresActiveProject(t *testing.T) {
	fixture := newProjectDeletionLockOrderFixture(t, "")
	title := "updated while archived"
	err := fixture.store.UpdateProjects(fixture.ctx, &store.UpdateProjectMessage{
		ResourceID: "project-a",
		Workspace:  "default",
		Title:      &title,
	})
	require.Equal(t, common.NotFound, common.ErrorCode(err))

	var storedTitle string
	require.NoError(t, fixture.db.QueryRowContext(fixture.ctx, "SELECT name FROM project WHERE resource_id = 'project-a'").Scan(&storedTitle))
	require.Equal(t, "Project A", storedTitle)
}

func TestDeleteRevisionGatesProjectOwnedDeleter(t *testing.T) {
	fixture := newProjectDeletionLockOrderFixture(t, `
		UPDATE project SET deleted = FALSE WHERE resource_id = 'project-a';
		INSERT INTO project (resource_id, workspace, name) VALUES ('project-b', 'default', 'Project B');
		INSERT INTO instance (resource_id, workspace) VALUES ('instance-b', 'default');
		INSERT INTO db (instance, name, project) VALUES ('instance-b', 'database-b', 'project-b');
		INSERT INTO revision (resource_id, instance, db_name, version) VALUES
			('revision-service-account', 'instance-b', 'database-b', 'v1'),
			('revision-workload-identity', 'instance-b', 'database-b', 'v2');
		INSERT INTO service_account (email, name, workspace, service_key_hash, project)
		VALUES ('deleter@project-a.service.bytebase.com', 'Deleter', 'default', 'unused', 'project-a');
		INSERT INTO workload_identity (email, name, workspace, project)
		VALUES ('deleter@project-a.workload.bytebase.com', 'Deleter', 'default', 'project-a');
	`)

	for _, test := range []struct {
		name       string
		revisionID string
		deleter    string
	}{
		{name: "service account", revisionID: "revision-service-account", deleter: "deleter@project-a.service.bytebase.com"},
		{name: "workload identity", revisionID: "revision-workload-identity", deleter: "deleter@project-a.workload.bytebase.com"},
	} {
		t.Run(test.name, func(t *testing.T) {
			release := holdExclusiveLifecycleGate(fixture.ctx, t, fixture.db, store.AdvisoryLockKeyProjectLifecycle, "project-a")
			err := fixture.store.DeleteRevision(fixture.ctx, test.revisionID, "instance-b", "database-b", test.deleter)
			require.ErrorIs(t, err, store.ErrLifecycleBusy)
			release()

			var deleter, deletedAt sql.NullString
			require.NoError(t, fixture.db.QueryRowContext(fixture.ctx, `
				SELECT deleter, deleted_at::text FROM revision WHERE resource_id = $1
			`, test.revisionID).Scan(&deleter, &deletedAt))
			require.False(t, deleter.Valid)
			require.False(t, deletedAt.Valid)
		})
	}
}

func TestUpdateInstanceFailsWhenLifecycleGateHeld(t *testing.T) {
	ctx, db, stores := newInstanceProjectFixture(t)
	projectID := "project-a"
	instance, err := stores.CreateInstance(ctx, &store.InstanceMessage{
		ResourceID: "project-instance",
		Workspace:  "default",
		ProjectID:  &projectID,
		Metadata:   testInstanceMetadata(),
	})
	require.NoError(t, err)

	for _, gate := range []struct {
		name       string
		namespace  store.AdvisoryLockKey
		resourceID string
	}{
		{name: "project", namespace: store.AdvisoryLockKeyProjectLifecycle, resourceID: projectID},
		{name: "instance", namespace: store.AdvisoryLockKeyInstanceLifecycle, resourceID: instance.ResourceID},
	} {
		t.Run(gate.name, func(t *testing.T) {
			release := holdExclusiveLifecycleGate(ctx, t, db, gate.namespace, gate.resourceID)
			metadata := testInstanceMetadata()
			metadata.Title = "changed"
			_, err := stores.UpdateInstance(ctx, &store.UpdateInstanceMessage{
				ResourceID: &instance.ResourceID,
				Workspace:  instance.Workspace,
				Metadata:   metadata,
			})
			require.ErrorIs(t, err, store.ErrLifecycleBusy)
			release()

			current, err := stores.GetInstanceByResourceID(ctx, instance.ResourceID)
			require.NoError(t, err)
			require.Empty(t, current.Metadata.Title)
		})
	}
}

func TestBatchUpdateInstancesByEnvironmentUsesExistingLifecycleGate(t *testing.T) {
	ctx, db, stores := newInstanceProjectFixture(t)
	_, err := db.ExecContext(ctx, `
		INSERT INTO instance (resource_id, workspace, environment, deleted)
		VALUES ('archived-instance', 'default', 'environment-a', TRUE)`)
	require.NoError(t, err)

	environmentID, unset := "environment-a", ""
	release := holdExclusiveLifecycleGate(ctx, t, db, store.AdvisoryLockKeyInstanceLifecycle, "archived-instance")
	_, err = stores.UpdateInstance(ctx, &store.UpdateInstanceMessage{
		Workspace:           "default",
		FindByEnvironmentID: &environmentID,
		EnvironmentID:       &unset,
	})
	require.ErrorIs(t, err, store.ErrLifecycleBusy)
	release()

	_, err = stores.UpdateInstance(ctx, &store.UpdateInstanceMessage{
		Workspace:           "default",
		FindByEnvironmentID: &environmentID,
		EnvironmentID:       &unset,
	})
	require.NoError(t, err)
	var environment sql.NullString
	require.NoError(t, db.QueryRowContext(ctx, "SELECT environment FROM instance WHERE resource_id = 'archived-instance'").Scan(&environment))
	require.False(t, environment.Valid)
}
