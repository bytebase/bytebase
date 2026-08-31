package store_test

import (
	"context"
	"database/sql"
	"testing"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/store"
)

func holdExclusiveLifecycleGate(t *testing.T, ctx context.Context, db *sql.DB, namespace store.AdvisoryLockKey, resourceID string) func() {
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

	release := holdExclusiveLifecycleGate(t, fixture.ctx, fixture.db, store.AdvisoryLockKeyProjectLifecycle, "project-a")
	_, err = fixture.store.BatchUpdateIssueStatuses(fixture.ctx, "project-a", []int64{issue.UID}, storepb.Issue_DONE)
	require.ErrorIs(t, err, store.ErrLifecycleBusy)
	release()

	current, err := fixture.store.GetIssue(fixture.ctx, &store.FindIssueMessage{ProjectIDs: []string{"project-a"}, UID: &issue.UID})
	require.NoError(t, err)
	require.Equal(t, storepb.Issue_OPEN, current.Status)
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
			release := holdExclusiveLifecycleGate(t, ctx, db, gate.namespace, gate.resourceID)
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
	release := holdExclusiveLifecycleGate(t, ctx, db, store.AdvisoryLockKeyInstanceLifecycle, "archived-instance")
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
