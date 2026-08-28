package store

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/testcontainer"
)

func newLifecycleGateStore(t *testing.T) (context.Context, *Store) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel)
	container := testcontainer.GetTestPgContainer(ctx, t)
	t.Cleanup(func() { container.Close(context.Background()) })
	db := container.GetDB()
	_, err := db.ExecContext(ctx, `
		CREATE TABLE project (
			resource_id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			deleted BOOLEAN NOT NULL DEFAULT FALSE
		);
		CREATE TABLE instance (
			resource_id TEXT PRIMARY KEY,
			deleted BOOLEAN NOT NULL DEFAULT FALSE
		);
		INSERT INTO project (resource_id, name, deleted) VALUES
			('project-a', 'Project A', FALSE),
			('project-b', 'Project B', FALSE),
			('project-archived', 'Archived', TRUE);
		INSERT INTO instance (resource_id, deleted) VALUES
			('instance-a', FALSE),
			('instance-archived', TRUE);
	`)
	require.NoError(t, err)
	return ctx, &Store{dbConnManager: &DBConnectionManager{db: db}}
}

func projectScope(projectID string, requirement lifecycleRequirement) lifecycleScope {
	scope := lifecycleScope{}
	scope.addProject(projectID, requirement)
	return scope
}

func instanceScope(instanceID string, requirement lifecycleRequirement) lifecycleScope {
	scope := lifecycleScope{}
	scope.addInstance(instanceID, requirement)
	return scope
}

func requireBusy(t *testing.T, err error) {
	t.Helper()
	require.Equal(t, common.Conflict, common.ErrorCode(err))
	require.EqualError(t, err, "resource is busy; retry")
	require.ErrorIs(t, err, ErrLifecycleBusy)
}

func TestLifecycleGateSharedWritesCoexist(t *testing.T) {
	ctx, store := newLifecycleGateStore(t)
	started := make(chan struct{})
	release := make(chan struct{})
	first := make(chan error, 1)
	go func() {
		first <- store.runLifecycleWrite(ctx, projectScope("project-a", lifecycleActive), func(*sql.Tx) error {
			close(started)
			<-release
			return nil
		})
	}()
	<-started

	require.NoError(t, store.runLifecycleWrite(ctx, projectScope("project-a", lifecycleActive), func(*sql.Tx) error {
		return nil
	}))
	close(release)
	require.NoError(t, <-first)
}

func TestLifecycleGateExclusiveConflictsWithSharedInBothOrders(t *testing.T) {
	t.Run("shared first", func(t *testing.T) {
		ctx, store := newLifecycleGateStore(t)
		started := make(chan struct{})
		release := make(chan struct{})
		first := make(chan error, 1)
		go func() {
			first <- store.runLifecycleWrite(ctx, projectScope("project-a", lifecycleExisting), func(*sql.Tx) error {
				close(started)
				<-release
				return nil
			})
		}()
		<-started
		called := false
		requireBusy(t, store.runLifecycleTransition(ctx, lifecycleScope{}, projectScope("project-a", lifecycleExisting), func(*sql.Tx) error {
			called = true
			return nil
		}))
		require.False(t, called)
		close(release)
		require.NoError(t, <-first)
		require.NoError(t, store.runLifecycleTransition(ctx, lifecycleScope{}, projectScope("project-a", lifecycleExisting), func(*sql.Tx) error { return nil }))
	})

	t.Run("exclusive first", func(t *testing.T) {
		ctx, store := newLifecycleGateStore(t)
		started := make(chan struct{})
		release := make(chan struct{})
		first := make(chan error, 1)
		go func() {
			first <- store.runLifecycleTransition(ctx, lifecycleScope{}, projectScope("project-a", lifecycleExisting), func(*sql.Tx) error {
				close(started)
				<-release
				return nil
			})
		}()
		<-started
		called := false
		requireBusy(t, store.runLifecycleWrite(ctx, projectScope("project-a", lifecycleExisting), func(*sql.Tx) error {
			called = true
			return nil
		}))
		require.False(t, called)
		close(release)
		require.NoError(t, <-first)
		require.NoError(t, store.runLifecycleWrite(ctx, projectScope("project-a", lifecycleExisting), func(*sql.Tx) error { return nil }))
	})
}

func TestLifecycleGateOverlapUsesExclusiveGate(t *testing.T) {
	ctx, store := newLifecycleGateStore(t)
	started := make(chan struct{})
	release := make(chan struct{})
	first := make(chan error, 1)
	go func() {
		first <- store.runLifecycleWrite(ctx, projectScope("project-a", lifecycleExisting), func(*sql.Tx) error {
			close(started)
			<-release
			return nil
		})
	}()
	<-started
	called := false
	err := store.runLifecycleTransition(ctx, projectScope("project-a", lifecycleExisting), projectScope("project-a", lifecycleExisting), func(*sql.Tx) error {
		called = true
		return nil
	})
	requireBusy(t, err)
	require.False(t, called)
	close(release)
	require.NoError(t, <-first)
}

func TestLifecycleGateInstanceExclusiveConflictsWithShared(t *testing.T) {
	ctx, store := newLifecycleGateStore(t)
	started := make(chan struct{})
	release := make(chan struct{})
	first := make(chan error, 1)
	go func() {
		first <- store.runLifecycleWrite(ctx, instanceScope("instance-a", lifecycleExisting), func(*sql.Tx) error {
			close(started)
			<-release
			return nil
		})
	}()
	<-started
	requireBusy(t, store.runLifecycleTransition(ctx, lifecycleScope{}, instanceScope("instance-a", lifecycleExisting), func(*sql.Tx) error { return nil }))
	close(release)
	require.NoError(t, <-first)
}

func TestLifecycleGateUnrelatedRootsDoNotConflict(t *testing.T) {
	ctx, store := newLifecycleGateStore(t)
	started := make(chan struct{})
	release := make(chan struct{})
	first := make(chan error, 1)
	go func() {
		first <- store.runLifecycleTransition(ctx, lifecycleScope{}, projectScope("project-a", lifecycleExisting), func(*sql.Tx) error {
			close(started)
			<-release
			return nil
		})
	}()
	<-started

	require.NoError(t, store.runLifecycleWrite(ctx, projectScope("project-b", lifecycleExisting), func(*sql.Tx) error { return nil }))
	close(release)
	require.NoError(t, <-first)
}

func TestLifecycleGateMergesStateRequirements(t *testing.T) {
	ctx, store := newLifecycleGateStore(t)
	called := false
	err := store.runLifecycleTransition(
		ctx,
		projectScope("project-archived", lifecycleExisting),
		projectScope("project-archived", lifecycleActive),
		func(*sql.Tx) error {
			called = true
			return nil
		},
	)
	require.Equal(t, common.NotFound, common.ErrorCode(err))
	require.False(t, called)
}

func TestLifecycleGateOrdersProjectsBeforeInstances(t *testing.T) {
	scope := instanceScope("instance-b", lifecycleExisting)
	scope.addProject("project-b", lifecycleExisting)
	scope.addProject("project-a", lifecycleActive)
	scope.addInstance("instance-a", lifecycleActive)

	require.Equal(t, []lifecycleGate{
		{kind: lifecycleProject, resourceID: "project-a", requirement: lifecycleActive},
		{kind: lifecycleProject, resourceID: "project-b", requirement: lifecycleExisting},
		{kind: lifecycleInstance, resourceID: "instance-a", requirement: lifecycleActive},
		{kind: lifecycleInstance, resourceID: "instance-b", requirement: lifecycleExisting},
	}, scope.orderedGates())
}

func TestLifecycleGateCommitsAndRollsBackCallback(t *testing.T) {
	ctx, store := newLifecycleGateStore(t)
	require.NoError(t, store.runLifecycleWrite(ctx, projectScope("project-a", lifecycleActive), func(tx *sql.Tx) error {
		_, err := tx.ExecContext(ctx, "UPDATE project SET name = 'Committed' WHERE resource_id = 'project-a'")
		return err
	}))
	var name string
	require.NoError(t, store.GetDB().QueryRowContext(ctx, "SELECT name FROM project WHERE resource_id = 'project-a'").Scan(&name))
	require.Equal(t, "Committed", name)

	rollback := errors.New("rollback")
	err := store.runLifecycleWrite(ctx, projectScope("project-a", lifecycleActive), func(tx *sql.Tx) error {
		if _, err := tx.ExecContext(ctx, "UPDATE project SET name = 'Rolled back' WHERE resource_id = 'project-a'"); err != nil {
			return err
		}
		return rollback
	})
	require.ErrorIs(t, err, rollback)
	require.NoError(t, store.GetDB().QueryRowContext(ctx, "SELECT name FROM project WHERE resource_id = 'project-a'").Scan(&name))
	require.Equal(t, "Committed", name)
}
