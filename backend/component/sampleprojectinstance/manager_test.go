package sampleprojectinstance

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/testcontainer"
	"github.com/bytebase/bytebase/backend/component/dbfactory"
	"github.com/bytebase/bytebase/backend/enterprise"
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
	require.Equal(t, FailureFailedPrecondition, FailureKindOf(mapTargetError(newTargetFailure(targetFailureStatic, errors.New("invalid target")))))
	require.Equal(t, FailureUnavailable, FailureKindOf(mapTargetError(newTargetFailure(targetFailureUnavailable, errors.New("offline")))))
	require.Equal(t, FailureDeadlineExceeded, FailureKindOf(mapTargetError(context.DeadlineExceeded)))
	require.Equal(t, FailureUnknown, FailureKindOf(mapTargetError(newTargetFailure(targetFailureInvariant, errors.New("collision")))))
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
	require.Nil(t, mustGetSampleProjectInstance(ctx, t, s, "workspace-a"))
	assertAllocationAbsent(ctx, t, target, sampleNames("workspace-a"))

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

	names := sampleNames("workspace-a")
	admin, err := target.connect(ctx, "", "", "")
	require.NoError(t, err)
	_, err = admin.Exec(ctx, fmt.Sprintf("ALTER ROLE %s NOLOGIN", quoteIdentifier(names.Role)))
	require.NoError(t, err)
	require.NoError(t, admin.Close(ctx))
	_, err = lockConn.ExecContext(ctx, "SELECT pg_advisory_unlock($1)", advisoryLockID)
	require.NoError(t, err)
	require.Error(t, <-prepareErr)
	require.Nil(t, mustGetSampleProjectInstance(ctx, t, s, "workspace-a"))
	assertAllocationAbsent(ctx, t, target, names)

	_, err = db.ExecContext(ctx, `
		DROP TRIGGER block_sample_instance_insert ON instance;
		DROP FUNCTION block_sample_instance_insert();
	`)
	require.NoError(t, err)
	_, err = manager.Prepare(ctx, PrepareRequest{WorkspaceID: "workspace-a", ProjectID: "project-a"})
	require.NoError(t, err)
}

func TestManagerPersistsAndRecoversConcreteProvisionOwnership(t *testing.T) {
	ctx, _, s, target, manager := newConcreteManager(t)
	names := sampleNames("workspace-a")
	admin, err := target.connect(ctx, "", "", "")
	require.NoError(t, err)
	defer admin.Close(ctx)
	_, err = admin.Exec(ctx, fmt.Sprintf("CREATE DATABASE %s", quoteIdentifier(names.Database)))
	require.NoError(t, err)
	_, err = admin.Exec(ctx, fmt.Sprintf("REVOKE ALL PRIVILEGES ON DATABASE %s FROM PUBLIC", quoteIdentifier(names.Database)))
	require.NoError(t, err)
	target.provisionHook = func(stage provisionStage) error {
		if stage == provisionStageCleanupRole {
			return errors.New("injected role cleanup failure")
		}
		return nil
	}

	_, err = manager.Prepare(ctx, PrepareRequest{WorkspaceID: "workspace-a", ProjectID: "project-a"})
	require.Equal(t, FailureUnavailable, FailureKindOf(err))
	reservation := mustGetSampleProjectInstance(ctx, t, s, "workspace-a")
	require.True(t, reservation.OwnershipKnown)
	require.False(t, reservation.DatabaseCreated)
	require.True(t, reservation.RoleCreated)

	target.provisionHook = nil
	_, err = admin.Exec(ctx, fmt.Sprintf("DROP DATABASE %s", quoteIdentifier(names.Database)))
	require.NoError(t, err)
	_, err = manager.Prepare(ctx, PrepareRequest{WorkspaceID: "workspace-a", ProjectID: "project-a"})
	require.NoError(t, err)
}

func newConcreteManager(t *testing.T) (context.Context, *sql.DB, *store.Store, *Target, *Manager) {
	t.Helper()
	ctx, db, s := newManagerStore(t)
	_, err := db.ExecContext(ctx, `
		INSERT INTO project (resource_id, workspace, name)
		VALUES ('project-a', 'workspace-a', 'Project A')
	`)
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
	t.Cleanup(func() {
		names := sampleNames("workspace-a")
		_ = target.Remove(context.Background(), Allocation{Database: names.Database, Role: names.Role})
	})

	licenseService, err := enterprise.NewLicenseService(common.ReleaseModeDev, s, false, "")
	require.NoError(t, err)
	syncer := schemasync.NewSyncer(s, dbfactory.New(s, licenseService), licenseService)
	return ctx, db, s, target, NewManager(s, target, syncer, ManagerOptions{})
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

func mustGetSampleProjectInstance(ctx context.Context, t *testing.T, s *store.Store, workspaceID string) *store.SampleProjectInstanceMessage {
	t.Helper()
	reservation, err := s.GetSampleProjectInstance(ctx, workspaceID)
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
