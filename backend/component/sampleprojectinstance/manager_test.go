package sampleprojectinstance

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/testcontainer"
	"github.com/bytebase/bytebase/backend/component/dbfactory"
	"github.com/bytebase/bytebase/backend/enterprise"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/migrator"
	_ "github.com/bytebase/bytebase/backend/plugin/db/pg"
	"github.com/bytebase/bytebase/backend/runner/schemasync"
	"github.com/bytebase/bytebase/backend/store"
)

func TestFailureKindOf(t *testing.T) {
	require.Equal(t, FailureUnknown, FailureKindOf(errors.New("unexpected")))
	require.Equal(t, FailureFailedPrecondition, FailureKindOf(newFailure(FailureFailedPrecondition, errors.New("invalid target"))))
	require.Equal(t, FailureUnavailable, FailureKindOf(errors.Join(errors.New("wrapped"), newFailure(FailureUnavailable, errors.New("offline")))))
	require.Equal(t, FailureDeadlineExceeded, FailureKindOf(newFailure(FailureDeadlineExceeded, context.DeadlineExceeded)))
}

func TestMapTargetErrorUsesManagerFailureVocabulary(t *testing.T) {
	require.Equal(t, FailureFailedPrecondition, FailureKindOf(mapTargetError(staticTargetError("invalid target"))))
	require.Equal(t, FailureUnavailable, FailureKindOf(mapTargetError(errors.New("offline"))))
	require.Equal(t, FailureDeadlineExceeded, FailureKindOf(mapTargetError(context.DeadlineExceeded)))
}

func TestManagerValidateTargetRejectsInvalidConfiguration(t *testing.T) {
	manager, err := NewManagerFromURL(nil, "postgresql://control:secret@127.0.0.1:5432/postgres?sslmode=prefer", nil, ManagerOptions{})
	require.Nil(t, manager)
	require.ErrorContains(t, err, "unsupported sslmode")
	require.NotContains(t, err.Error(), "secret")
}

func TestManagerAvailabilityRecoversAfterCachedValidationFailure(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping PostgreSQL testcontainer test in short mode")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	t.Cleanup(cancel)
	container := testcontainer.GetTestPgContainer(ctx, t)
	t.Cleanup(func() { container.Close(context.Background()) })

	now := time.Now()
	target := newLocalTarget(t, container)
	manager := newManager(nil, target, nil, ManagerOptions{Clock: func() time.Time { return now }})
	require.True(t, manager.Available(ctx))

	port := target.config.Port
	target.config.Port = 1
	require.True(t, manager.Available(ctx))
	now = now.Add(targetAvailabilityCacheTTL)
	require.False(t, manager.Available(ctx))

	target.config.Port = port
	require.False(t, manager.Available(ctx))
	now = now.Add(targetAvailabilityCacheTTL)
	require.True(t, manager.Available(ctx))
}

func TestPreparationLifecycleContextRenewsDeadline(t *testing.T) {
	oldLifecycle, oldCancel := context.WithDeadline(context.Background(), time.Now().Add(time.Second))
	defer oldCancel()

	freshLifecycle, freshCancel := preparationLifecycleContext(oldLifecycle)
	defer freshCancel()
	freshDeadline, ok := freshLifecycle.Deadline()
	require.True(t, ok)
	require.Greater(t, time.Until(freshDeadline), prepareDeadline-time.Second)
}

func TestManagerCompensatesConcreteMetadataFailure(t *testing.T) {
	ctx, db, s, target, manager := newConcreteManager(t)
	_, err := db.ExecContext(ctx, `
		CREATE FUNCTION fail_sample_instance_insert() RETURNS trigger
		LANGUAGE plpgsql AS $$
		BEGIN
			RAISE EXCEPTION 'injected instance persistence failure';
		END;
		$$;
		CREATE TRIGGER fail_sample_instance_insert
		BEFORE INSERT ON instance
		FOR EACH ROW
		WHEN (NEW.resource_id LIKE 'sample-%')
		EXECUTE FUNCTION fail_sample_instance_insert();
	`)
	require.NoError(t, err)

	_, err = manager.Prepare(ctx, PrepareRequest{WorkspaceID: "workspace-a", ProjectID: "project-a"})
	require.Error(t, err)
	require.Nil(t, mustGetSampleProjectInstance(ctx, t, s))
	assertNoSampleAllocations(ctx, t, target)

	_, err = db.ExecContext(ctx, `
		DROP TRIGGER fail_sample_instance_insert ON instance;
		DROP FUNCTION fail_sample_instance_insert();
	`)
	require.NoError(t, err)
	_, err = manager.Prepare(ctx, PrepareRequest{WorkspaceID: "workspace-a", ProjectID: "project-a"})
	require.NoError(t, err)
}

func TestManagerCompensatesConcreteDiscoveryFailure(t *testing.T) {
	ctx, db, s, target, manager := newConcreteManager(t)
	const advisoryLockID = 4242
	lockConn, err := db.Conn(ctx)
	require.NoError(t, err)
	defer lockConn.Close()
	_, err = lockConn.ExecContext(ctx, "SELECT pg_advisory_lock($1)", advisoryLockID)
	require.NoError(t, err)
	_, err = db.ExecContext(ctx, fmt.Sprintf(`
		CREATE FUNCTION block_sample_instance_insert() RETURNS trigger
		LANGUAGE plpgsql AS $$
		BEGIN
			PERFORM pg_advisory_xact_lock(%d);
			RETURN NEW;
		END;
		$$;
		CREATE TRIGGER block_sample_instance_insert
		BEFORE INSERT ON instance
		FOR EACH ROW
		WHEN (NEW.resource_id LIKE 'sample-%%')
		EXECUTE FUNCTION block_sample_instance_insert();
	`, advisoryLockID))
	require.NoError(t, err)

	prepareErr := make(chan error, 1)
	go func() {
		_, err := manager.Prepare(ctx, PrepareRequest{WorkspaceID: "workspace-a", ProjectID: "project-a"})
		prepareErr <- err
	}()
	require.Eventually(t, func() bool {
		var waiting bool
		err := db.QueryRowContext(ctx, `
			SELECT EXISTS (
				SELECT 1
				FROM pg_locks
				WHERE locktype = 'advisory' AND objid = $1 AND NOT granted
			)
		`, advisoryLockID).Scan(&waiting)
		return err == nil && waiting
	}, 30*time.Second, 50*time.Millisecond)

	reservation := mustGetSampleProjectInstance(ctx, t, s)
	require.NotNil(t, reservation)
	names := AllocationNames{Database: reservation.DBName, Role: reservation.RoleName}
	admin, err := target.connect(ctx, "", "", "")
	require.NoError(t, err)
	_, err = admin.Exec(ctx, fmt.Sprintf("ALTER ROLE %s NOLOGIN", quoteIdentifier(names.Role)))
	require.NoError(t, err)
	require.NoError(t, admin.Close(ctx))
	_, err = lockConn.ExecContext(ctx, "SELECT pg_advisory_unlock($1)", advisoryLockID)
	require.NoError(t, err)
	require.Error(t, <-prepareErr)
	require.Nil(t, mustGetSampleProjectInstance(ctx, t, s))
	assertAllocationAbsent(ctx, t, target, names)

	_, err = db.ExecContext(ctx, `
		DROP TRIGGER block_sample_instance_insert ON instance;
		DROP FUNCTION block_sample_instance_insert();
	`)
	require.NoError(t, err)
	_, err = manager.Prepare(ctx, PrepareRequest{WorkspaceID: "workspace-a", ProjectID: "project-a"})
	require.NoError(t, err)
}

func TestManagerCompensatesWhenWorkspaceDeletedDuringProvisioning(t *testing.T) {
	ctx, db, s, target, manager := newConcreteManager(t)
	const advisoryLockID = 4243
	lockConn, err := db.Conn(ctx)
	require.NoError(t, err)
	defer lockConn.Close()
	_, err = lockConn.ExecContext(ctx, "SELECT pg_advisory_lock($1)", advisoryLockID)
	require.NoError(t, err)
	lockReleased := false
	defer func() {
		if !lockReleased {
			_, _ = lockConn.ExecContext(context.Background(), "SELECT pg_advisory_unlock($1)", advisoryLockID)
		}
	}()
	_, err = db.ExecContext(ctx, fmt.Sprintf(`
		CREATE FUNCTION block_sample_instance_activation() RETURNS trigger
		LANGUAGE plpgsql AS $$
		BEGIN
			PERFORM pg_advisory_xact_lock(%d);
			RETURN NEW;
		END;
		$$;
		CREATE TRIGGER block_sample_instance_activation
		BEFORE INSERT ON instance
		FOR EACH ROW
		WHEN (NEW.resource_id LIKE 'sample-%%')
		EXECUTE FUNCTION block_sample_instance_activation();
	`, advisoryLockID))
	require.NoError(t, err)

	prepareErr := make(chan error, 1)
	go func() {
		_, err := manager.Prepare(ctx, PrepareRequest{WorkspaceID: "workspace-a", ProjectID: "project-a"})
		prepareErr <- err
	}()
	require.Eventually(t, func() bool {
		var waiting bool
		err := db.QueryRowContext(ctx, `
			SELECT EXISTS (
				SELECT 1
				FROM pg_locks
				WHERE locktype = 'advisory' AND objid = $1 AND NOT granted
			)
		`, advisoryLockID).Scan(&waiting)
		return err == nil && waiting
	}, 30*time.Second, 50*time.Millisecond)
	require.NoError(t, s.DeleteWorkspace(ctx, "workspace-a"))
	_, err = lockConn.ExecContext(ctx, "SELECT pg_advisory_unlock($1)", advisoryLockID)
	require.NoError(t, err)
	lockReleased = true

	err = <-prepareErr
	require.Equal(t, FailureFailedPrecondition, FailureKindOf(err))
	require.Nil(t, mustGetSampleProjectInstance(ctx, t, s))
	assertNoSampleAllocations(ctx, t, target)
	var activeInstances int
	require.NoError(t, db.QueryRowContext(ctx, `
		SELECT COUNT(*)
		FROM instance
		WHERE workspace = 'workspace-a'
			AND resource_id LIKE 'sample-%'
			AND deleted = FALSE
	`).Scan(&activeInstances))
	require.Zero(t, activeInstances)
}

func TestManagerRetriesWhenConcurrentFailureDeletesReservation(t *testing.T) {
	ctx, _, _, _, manager := newConcreteManager(t)
	policyDenied := errors.New("injected capacity denial")
	firstPolicyEntered := make(chan struct{})
	releaseFirstPolicy := make(chan struct{})
	type prepareResult struct {
		result *PrepareResult
		err    error
	}
	prepare := func(checkPolicy func(context.Context) (CreatePolicyResult, error)) <-chan prepareResult {
		result := make(chan prepareResult, 1)
		go func() {
			prepared, err := manager.Prepare(ctx, PrepareRequest{
				WorkspaceID:       "workspace-a",
				ProjectID:         "project-a",
				CheckCreatePolicy: checkPolicy,
			})
			result <- prepareResult{result: prepared, err: err}
		}()
		return result
	}

	first := prepare(func(policyCtx context.Context) (CreatePolicyResult, error) {
		close(firstPolicyEntered)
		select {
		case <-releaseFirstPolicy:
			return CreatePolicyResult{DeniedReason: policyDenied}, nil
		case <-policyCtx.Done():
			return CreatePolicyResult{}, policyCtx.Err()
		}
	})
	<-firstPolicyEntered

	secondPolicyCalled := make(chan struct{}, 1)
	second := prepare(func(context.Context) (CreatePolicyResult, error) {
		secondPolicyCalled <- struct{}{}
		return CreatePolicyResult{DeniedReason: policyDenied}, nil
	})
	close(releaseFirstPolicy)

	for _, result := range []prepareResult{<-first, <-second} {
		require.NoError(t, result.err)
		require.NotNil(t, result.result)
		require.ErrorIs(t, result.result.PolicyDenied, policyDenied)
	}
	require.Len(t, secondPolicyCalled, 1)
}

func TestManagerDoesNotHoldMetadataConnectionWhileProvisioning(t *testing.T) {
	ctx, _, s, target, manager := newConcreteManager(t)
	maxOpenConnections := s.GetDB().Stats().MaxOpenConnections
	s.GetDB().SetMaxOpenConns(1)
	t.Cleanup(func() { s.GetDB().SetMaxOpenConns(maxOpenConnections) })
	entered := make(chan struct{})
	release := make(chan struct{})
	released := false
	defer func() {
		if !released {
			close(release)
		}
	}()
	target.provisionHook = func(stage provisionStage) error {
		if stage == provisionStageRoleCreated {
			close(entered)
			<-release
		}
		return nil
	}
	prepared := make(chan error, 1)
	go func() {
		_, err := manager.Prepare(ctx, PrepareRequest{WorkspaceID: "workspace-a", ProjectID: "project-a"})
		prepared <- err
	}()
	<-entered

	_, err := s.GetEnvironment(ctx, "workspace-a")
	require.NoError(t, err)

	close(release)
	released = true
	require.NoError(t, <-prepared)
}

func TestManagerHealthyOwnerProvisionsOnce(t *testing.T) {
	ctx, _, s, target, first := newConcreteManager(t)
	licenseService, err := enterprise.NewLicenseService(common.ReleaseModeDev, s, false, "")
	require.NoError(t, err)
	second := newManager(s, target, schemasync.NewSyncer(s, dbfactory.New(s, licenseService), licenseService, nil), ManagerOptions{ReplicaID: "replica-b"})
	require.NoError(t, s.UpsertReplicaHeartbeat(ctx, "replica-a"))

	entered := make(chan struct{})
	release := make(chan struct{})
	var provisions atomic.Int32
	var enteredOnce sync.Once
	target.provisionHook = func(stage provisionStage) error {
		if stage == provisionStageRoleCreated {
			provisions.Add(1)
			enteredOnce.Do(func() { close(entered) })
			<-release
		}
		return nil
	}
	type prepareResult struct {
		result *PrepareResult
		err    error
	}
	prepare := func(manager *Manager) <-chan prepareResult {
		result := make(chan prepareResult, 1)
		go func() {
			prepared, err := manager.Prepare(ctx, PrepareRequest{WorkspaceID: "workspace-a", ProjectID: "project-a"})
			result <- prepareResult{result: prepared, err: err}
		}()
		return result
	}
	firstResult := prepare(first)
	<-entered
	secondResult := prepare(second)
	close(release)

	firstPrepared := <-firstResult
	secondPrepared := <-secondResult
	require.NoError(t, firstPrepared.err)
	require.NoError(t, secondPrepared.err)
	require.Equal(t, firstPrepared.result.Instance.ResourceID, secondPrepared.result.Instance.ResourceID)
	require.EqualValues(t, 1, provisions.Load())
}

func TestManagerTakesOverStaleReservationAfterReconciling(t *testing.T) {
	ctx, db, s, target, first := newConcreteManager(t)
	reservation, created, err := first.reserve(ctx, PrepareRequest{WorkspaceID: "workspace-a", ProjectID: "project-a"})
	require.NoError(t, err)
	require.True(t, created)
	allocation := Allocation{Database: reservation.DBName, Role: reservation.RoleName, Password: "sample-password"}
	require.NoError(t, target.Provision(ctx, allocation))
	_, err = db.ExecContext(ctx, `
		UPDATE sample_project_instance
		SET created_at = now() - INTERVAL '3 minutes 1 second',
			replica_id = 'stopped-replica'
		WHERE workspace = 'workspace-a'
	`)
	require.NoError(t, err)

	licenseService, err := enterprise.NewLicenseService(common.ReleaseModeDev, s, false, "")
	require.NoError(t, err)
	second := newManager(s, target, schemasync.NewSyncer(s, dbfactory.New(s, licenseService), licenseService, nil), ManagerOptions{ReplicaID: "replica-b"})
	prepared, err := second.Prepare(ctx, PrepareRequest{WorkspaceID: "workspace-a", ProjectID: "project-a"})
	require.NoError(t, err)
	require.NotNil(t, prepared.Instance)

	record := mustGetSampleProjectInstance(ctx, t, s)
	require.Equal(t, "replica-b", record.ReplicaID)
	require.NotNil(t, record.ExpiresAt)
	admin, err := target.connect(ctx, "", "", "")
	require.NoError(t, err)
	defer admin.Close(ctx)
	var exists bool
	require.NoError(t, admin.QueryRow(ctx, "SELECT EXISTS (SELECT 1 FROM pg_database WHERE datname = $1)", record.DBName).Scan(&exists))
	require.True(t, exists)
}

func TestManagerUsesFirstEnvironment(t *testing.T) {
	ctx, _, s, _, manager := newConcreteManager(t)
	_, err := s.UpsertSetting(ctx, &store.SettingMessage{
		Name:      storepb.SettingName_ENVIRONMENT,
		Workspace: "workspace-a",
		Value: &storepb.EnvironmentSetting{Environments: []*storepb.EnvironmentSetting_Environment{
			{Title: "Production", Id: "prod"},
			{Title: "Test", Id: "test"},
		}},
	})
	require.NoError(t, err)

	result, err := manager.Prepare(ctx, PrepareRequest{WorkspaceID: "workspace-a", ProjectID: "project-a"})
	require.NoError(t, err)
	require.Equal(t, "prod", *result.Instance.EnvironmentID)
	reservation := mustGetSampleProjectInstance(ctx, t, s)
	require.NotNil(t, reservation)
	names := sampleNames(reservation.InstanceID)
	require.Equal(t, names.Database, reservation.DBName)
	require.Equal(t, names.Role, reservation.RoleName)
}

func TestManagerAllowsNoEnvironment(t *testing.T) {
	ctx, _, s, _, manager := newConcreteManager(t)
	_, err := s.UpsertSetting(ctx, &store.SettingMessage{
		Name:      storepb.SettingName_ENVIRONMENT,
		Workspace: "workspace-a",
		Value:     &storepb.EnvironmentSetting{},
	})
	require.NoError(t, err)

	result, err := manager.Prepare(ctx, PrepareRequest{WorkspaceID: "workspace-a", ProjectID: "project-a"})
	require.NoError(t, err)
	require.Nil(t, result.Instance.EnvironmentID)
}

func TestManagerCompensatesProvisionFailures(t *testing.T) {
	stages := []provisionStage{
		provisionStageRoleCreated,
		provisionStageDatabaseCreated,
	}
	for _, stage := range stages {
		t.Run(string(stage), func(t *testing.T) {
			ctx, _, s, target, manager := newConcreteManager(t)
			target.provisionHook = func(current provisionStage) error {
				if current == stage {
					return errors.New("injected provisioning failure")
				}
				return nil
			}

			_, err := manager.Prepare(ctx, PrepareRequest{WorkspaceID: "workspace-a", ProjectID: "project-a"})
			require.Equal(t, FailureUnavailable, FailureKindOf(err))
			require.Nil(t, mustGetSampleProjectInstance(ctx, t, s))
			assertNoSampleAllocations(ctx, t, target)
		})
	}
}

func TestManagerPreservesAndRecoversReservationWhenCompensationFails(t *testing.T) {
	ctx, db, s, target, manager := newConcreteManager(t)
	provisionFailed := false
	target.provisionHook = func(stage provisionStage) error {
		if stage == provisionStageDatabaseCreated && !provisionFailed {
			provisionFailed = true
			return errors.New("injected provisioning failure")
		}
		if stage == provisionStageCleanupRole {
			return errors.New("injected cleanup failure")
		}
		return nil
	}

	_, err := manager.Prepare(ctx, PrepareRequest{WorkspaceID: "workspace-a", ProjectID: "project-a"})
	require.Equal(t, FailureUnavailable, FailureKindOf(err))
	reservation := mustGetSampleProjectInstance(ctx, t, s)
	require.NotNil(t, reservation)
	require.Nil(t, reservation.ExpiresAt)
	_, err = db.ExecContext(ctx, `
		UPDATE sample_project_instance
		SET created_at = now() - INTERVAL '3 minutes 1 second'
		WHERE workspace = 'workspace-a'
	`)
	require.NoError(t, err)

	target.provisionHook = nil
	result, err := manager.Prepare(ctx, PrepareRequest{WorkspaceID: "workspace-a", ProjectID: "project-a"})
	require.NoError(t, err)
	require.NotNil(t, result.Instance)
	reservation = mustGetSampleProjectInstance(ctx, t, s)
	require.NotNil(t, reservation.ExpiresAt)
}

func TestManagerReconcileAttemptsTargetCleanupWhenMetadataRemovalFails(t *testing.T) {
	ctx, db, _, target, manager := newConcreteManager(t)
	_, err := db.ExecContext(ctx, `
		INSERT INTO project (resource_id, workspace, name)
		VALUES ('project-b', 'workspace-a', 'Project B')
	`)
	require.NoError(t, err)
	reservation, created, err := manager.reserve(ctx, PrepareRequest{WorkspaceID: "workspace-a", ProjectID: "project-a"})
	require.NoError(t, err)
	require.True(t, created)
	allocation := Allocation{Database: reservation.DBName, Role: reservation.RoleName, Password: "sample-password"}
	require.NoError(t, target.Provision(ctx, allocation))
	_, err = manager.createMetadata(ctx, registration{
		WorkspaceID:       "workspace-a",
		ProjectID:         "project-b",
		EnvironmentID:     "test",
		InstanceID:        reservation.InstanceID,
		Title:             sampleProjectInstanceTitle,
		Engine:            storepb.Engine_POSTGRES,
		AdminDataSource:   &storepb.DataSource{Id: "admin", Type: storepb.DataSourceType_ADMIN},
		SyncDatabaseNames: []string{reservation.DBName},
	})
	require.NoError(t, err)

	err = manager.reconcile(ctx, allocation, reservation.InstanceID, PrepareRequest{WorkspaceID: "workspace-a", ProjectID: "project-a"})
	require.Error(t, err)
	assertAllocationAbsent(ctx, t, target, AllocationNames{Database: reservation.DBName, Role: reservation.RoleName})
}

func newConcreteManager(t *testing.T) (context.Context, *sql.DB, *store.Store, *Target, *Manager) {
	t.Helper()
	ctx, db, s := newManagerStore(t)
	_, err := db.ExecContext(ctx, `
		INSERT INTO project (resource_id, workspace, name)
		VALUES ('project-a', 'workspace-a', 'Project A')
	`)
	require.NoError(t, err)
	_, err = s.UpsertSetting(ctx, &store.SettingMessage{
		Name:      storepb.SettingName_ENVIRONMENT,
		Workspace: "workspace-a",
		Value: &storepb.EnvironmentSetting{Environments: []*storepb.EnvironmentSetting_Environment{
			{Title: "Test", Id: "test"},
		}},
	})
	require.NoError(t, err)

	container := testcontainer.GetTestTLSPgContainer(ctx, t)
	t.Cleanup(func() { container.Close(context.Background()) })
	targetURL := fmt.Sprintf(
		"postgres://postgres:root-password@localhost:%s/postgres?sslmode=verify-full&sslrootcert=%s",
		container.GetPort(),
		url.QueryEscape(container.GetTLSCAPath()),
	)
	target, err := NewTarget(targetURL)
	require.NoError(t, err)
	admin, err := target.connect(ctx, "", "", "")
	require.NoError(t, err)
	require.NoError(t, prepareBaseline(ctx, admin))
	require.NoError(t, admin.Close(ctx))
	licenseService, err := enterprise.NewLicenseService(common.ReleaseModeDev, s, false, "")
	require.NoError(t, err)
	syncer := schemasync.NewSyncer(s, dbfactory.New(s, licenseService), licenseService, nil)
	return ctx, db, s, target, newManager(s, target, syncer, ManagerOptions{ReplicaID: "replica-a"})
}

func assertAllocationAbsent(ctx context.Context, t *testing.T, target *Target, names AllocationNames) {
	t.Helper()
	conn, err := target.connect(ctx, "", "", "")
	require.NoError(t, err)
	defer conn.Close(ctx)
	var databaseExists, roleExists bool
	require.NoError(t, conn.QueryRow(ctx, "SELECT EXISTS (SELECT 1 FROM pg_database WHERE datname = $1)", names.Database).Scan(&databaseExists))
	require.NoError(t, conn.QueryRow(ctx, "SELECT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = $1)", names.Role).Scan(&roleExists))
	require.False(t, databaseExists)
	require.False(t, roleExists)
}

func assertNoSampleAllocations(ctx context.Context, t *testing.T, target *Target) {
	t.Helper()
	conn, err := target.connect(ctx, "", "", "")
	require.NoError(t, err)
	defer conn.Close(ctx)
	var databaseCount, roleCount int
	require.NoError(t, conn.QueryRow(ctx, "SELECT COUNT(*) FROM pg_database WHERE datname LIKE 'bb\\_sample\\_%' ESCAPE '\\'").Scan(&databaseCount))
	require.NoError(t, conn.QueryRow(ctx, "SELECT COUNT(*) FROM pg_roles WHERE rolname LIKE 'bb\\_sample\\_role\\_%' ESCAPE '\\'").Scan(&roleCount))
	require.Zero(t, databaseCount)
	require.Zero(t, roleCount)
}

func mustGetSampleProjectInstance(ctx context.Context, t *testing.T, s *store.Store) *store.SampleProjectInstanceMessage {
	t.Helper()
	reservation, err := s.GetSampleProjectInstance(ctx, "workspace-a")
	require.NoError(t, err)
	return reservation
}

func newManagerStore(t *testing.T) (context.Context, *sql.DB, *store.Store) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	t.Cleanup(cancel)
	container := testcontainer.GetTestPgContainer(ctx, t)
	t.Cleanup(func() { container.Close(context.Background()) })
	db := container.GetDB()
	require.NoError(t, migrator.MigrateSchema(ctx, db))
	_, err := db.ExecContext(ctx, `INSERT INTO workspace (resource_id) VALUES ('workspace-a'), ('workspace-b')`)
	require.NoError(t, err)
	pgURL := fmt.Sprintf("host=%s port=%s user=postgres password=root-password database=postgres", container.GetHost(), container.GetPort())
	s, err := store.New(ctx, pgURL, false)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, s.Close()) })
	return ctx, db, s
}
