package sampleprojectinstance

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/common/testcontainer"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/migrator"
	"github.com/bytebase/bytebase/backend/store"
)

func TestManagerPrepareCreatesActivatedSampleProjectInstance(t *testing.T) {
	ctx, _, s := newManagerStore(t)
	target := &fakeTarget{}
	metadata := &fakeMetadata{}
	names := sampleNames("workspace-a")
	syncer := &fakeSyncer{databaseName: names.Database}
	now := time.Date(2026, time.August, 17, 9, 0, 0, 0, time.UTC)
	manager := NewManager(s, target, metadata, syncer, ManagerOptions{
		Clock:  func() time.Time { return now },
		Random: bytes.NewReader(bytes.Repeat([]byte{0}, 32)),
	})

	instance, err := manager.Prepare(ctx, PrepareRequest{
		WorkspaceID: "workspace-a",
		ProjectID:   "project-a",
	})
	require.NoError(t, err)
	require.Equal(t, names.InstanceID, instance.ResourceID)
	require.Equal(t, 1, target.validateCalls)
	require.Len(t, target.provisions, 1)
	require.Equal(t, names.Database, target.provisions[0].Database)
	require.Equal(t, names.Role, target.provisions[0].Role)
	require.Equal(t, "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA", target.provisions[0].Password)
	require.Equal(t, "Sample Project Instance", metadata.registration.Title)
	require.Equal(t, "test", metadata.registration.EnvironmentID)
	require.Equal(t, storepb.Engine_POSTGRES, metadata.registration.Engine)
	require.Equal(t, []string{target.provisions[0].Database}, metadata.registration.SyncDatabaseNames)
	require.Equal(t, 1, syncer.asyncCalls)

	var expiresAt time.Time
	require.NoError(t, s.WithLockedSampleProjectInstance(ctx, "workspace-a", func(_ context.Context, _ *store.SampleProjectInstanceTx, message *store.SampleProjectInstanceMessage) error {
		expiresAt = *message.ExpiresAt
		return nil
	}))
	require.Equal(t, now.Add(7*24*time.Hour), expiresAt)
}

func TestManagerPrepareReturnsSameProjectAllocationBeforeTargetValidation(t *testing.T) {
	ctx, _, s := newManagerStore(t)
	now := time.Date(2026, time.August, 17, 9, 0, 0, 0, time.UTC)
	seedReservation(ctx, t, s, "workspace-a", "project-a", now)
	target := &fakeTarget{validateErr: NewTargetError(TargetErrorUnavailable, errors.New("offline"))}
	metadata := &fakeMetadata{
		state: MetadataState{
			ProjectActive:   true,
			InstanceMatches: true,
			Instance:        &store.InstanceMessage{ResourceID: sampleNames("workspace-a").InstanceID},
			Database: &store.DatabaseMessage{
				InstanceID:   sampleNames("workspace-a").InstanceID,
				DatabaseName: sampleNames("workspace-a").Database,
			},
		},
	}
	canCreateCalls := 0
	manager := NewManager(s, target, metadata, &fakeSyncer{}, ManagerOptions{
		Clock: func() time.Time { return now },
	})

	instance, err := manager.Prepare(ctx, PrepareRequest{
		WorkspaceID: "workspace-a",
		ProjectID:   "project-a",
		CanCreate:   func(context.Context) error { canCreateCalls++; return nil },
	})
	require.NoError(t, err)
	require.Equal(t, metadata.state.Instance, instance)
	require.Zero(t, target.validateCalls)
	require.Zero(t, canCreateCalls)
}

func TestManagerPrepareRejectsConsumedOrCrossProjectAllocation(t *testing.T) {
	ctx, _, s := newManagerStore(t)
	now := time.Date(2026, time.August, 17, 9, 0, 0, 0, time.UTC)
	seedReservation(ctx, t, s, "workspace-a", "project-a", now)
	manager := NewManager(s, &fakeTarget{}, &fakeMetadata{}, &fakeSyncer{}, ManagerOptions{Clock: func() time.Time { return now }})

	_, err := manager.Prepare(ctx, PrepareRequest{WorkspaceID: "workspace-a", ProjectID: "project-b"})
	require.ErrorIs(t, err, ErrFailedPrecondition)
	require.Equal(t, ErrorKindFailedPrecondition, ErrorKindOf(err))

	require.NoError(t, s.WithLockedSampleProjectInstance(ctx, "workspace-a", func(ctx context.Context, tx *store.SampleProjectInstanceTx, _ *store.SampleProjectInstanceMessage) error {
		return tx.MarkDeleted(ctx, now)
	}))
	_, err = manager.Prepare(ctx, PrepareRequest{WorkspaceID: "workspace-a", ProjectID: "project-a"})
	require.ErrorIs(t, err, ErrFailedPrecondition)
}

func TestManagerPrepareCompensatesAndDeletesFailedReservation(t *testing.T) {
	ctx, _, s := newManagerStore(t)
	target := &fakeTarget{}
	metadata := &fakeMetadata{createErr: errors.New("metadata write failed")}
	manager := NewManager(s, target, metadata, &fakeSyncer{}, ManagerOptions{
		Clock:  time.Now,
		Random: bytes.NewReader(bytes.Repeat([]byte{1}, 32)),
	})

	_, err := manager.Prepare(ctx, PrepareRequest{WorkspaceID: "workspace-a", ProjectID: "project-a"})
	require.Error(t, err)
	require.Len(t, target.removes, 1)
	require.Equal(t, 1, metadata.removeCalls)
	require.Error(t, s.WithLockedSampleProjectInstance(ctx, "workspace-a", func(context.Context, *store.SampleProjectInstanceTx, *store.SampleProjectInstanceMessage) error {
		return nil
	}))
}

func TestManagerPrepareSurvivesClientCancellationAfterReservation(t *testing.T) {
	fixtureCtx, _, s := newManagerStore(t)
	ctx, cancel := context.WithCancel(fixtureCtx)
	t.Cleanup(cancel)
	names := sampleNames("workspace-a")
	target := &fakeTarget{
		validate: func(lifecycleCtx context.Context) error {
			cancel()
			select {
			case <-lifecycleCtx.Done():
				return errors.New("lifecycle context was canceled by client")
			default:
				return nil
			}
		},
	}
	manager := NewManager(s, target, &fakeMetadata{}, &fakeSyncer{databaseName: names.Database}, ManagerOptions{
		Random: bytes.NewReader(bytes.Repeat([]byte{3}, 32)),
	})

	_, err := manager.Prepare(ctx, PrepareRequest{WorkspaceID: "workspace-a", ProjectID: "project-a"})
	require.NoError(t, err)
	require.Error(t, ctx.Err())
	require.NoError(t, s.WithLockedSampleProjectInstance(fixtureCtx, "workspace-a", func(_ context.Context, _ *store.SampleProjectInstanceTx, reservation *store.SampleProjectInstanceMessage) error {
		require.NotNil(t, reservation.ExpiresAt)
		return nil
	}))
}

func TestManagerPrepareDiscardsReservationForCollisionValidationAndLimitFailures(t *testing.T) {
	for _, test := range []struct {
		name      string
		target    *fakeTarget
		metadata  *fakeMetadata
		canCreate func(context.Context) error
		kind      ErrorKind
	}{
		{
			name:   "deterministic metadata collision",
			target: &fakeTarget{},
			metadata: &fakeMetadata{state: MetadataState{
				Instance: &store.InstanceMessage{ResourceID: sampleNames("workspace-a").InstanceID},
			}},
			kind: ErrorKindInternal,
		},
		{
			name:     "target validation",
			target:   &fakeTarget{validateErr: NewTargetError(TargetErrorStatic, errors.New("invalid target"))},
			metadata: &fakeMetadata{},
			kind:     ErrorKindFailedPrecondition,
		},
		{
			name:      "instance limit",
			target:    &fakeTarget{},
			metadata:  &fakeMetadata{},
			canCreate: func(context.Context) error { return errors.New("limit reached") },
			kind:      "",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			ctx, _, s := newManagerStore(t)
			manager := NewManager(s, test.target, test.metadata, &fakeSyncer{}, ManagerOptions{})

			_, err := manager.Prepare(ctx, PrepareRequest{WorkspaceID: "workspace-a", ProjectID: "project-a", CanCreate: test.canCreate})
			require.Error(t, err)
			if test.kind != "" {
				require.Equal(t, test.kind, ErrorKindOf(err))
			}
			require.Zero(t, test.metadata.removeCalls)
			require.Empty(t, test.target.provisions)
			require.Error(t, s.WithLockedSampleProjectInstance(ctx, "workspace-a", func(context.Context, *store.SampleProjectInstanceTx, *store.SampleProjectInstanceMessage) error {
				return nil
			}))
		})
	}
}

func TestManagerFromURLDefersEmptyConfigurationFailure(t *testing.T) {
	ctx, _, s := newManagerStore(t)
	now := time.Date(2026, time.August, 17, 9, 0, 0, 0, time.UTC)
	seedReservation(ctx, t, s, "workspace-a", "project-a", now)
	metadata := &fakeMetadata{
		state: MetadataState{
			ProjectActive:   true,
			InstanceMatches: true,
			Instance:        &store.InstanceMessage{ResourceID: sampleNames("workspace-a").InstanceID},
			Database: &store.DatabaseMessage{
				InstanceID:   sampleNames("workspace-a").InstanceID,
				DatabaseName: sampleNames("workspace-a").Database,
			},
		},
	}
	manager := NewManagerFromURL(s, "", metadata, &fakeSyncer{}, ManagerOptions{})

	instance, err := manager.Prepare(ctx, PrepareRequest{WorkspaceID: "workspace-a", ProjectID: "project-a"})
	require.NoError(t, err)
	require.Equal(t, sampleNames("workspace-a").InstanceID, instance.ResourceID)
	metadata.state = MetadataState{}
	_, err = manager.Prepare(ctx, PrepareRequest{WorkspaceID: "workspace-b", ProjectID: "project-b"})
	require.ErrorIs(t, err, ErrFailedPrecondition)
	require.Error(t, s.WithLockedSampleProjectInstance(ctx, "workspace-b", func(context.Context, *store.SampleProjectInstanceTx, *store.SampleProjectInstanceMessage) error {
		return nil
	}))
}

func TestManagerFromURLReturnsStaticErrorForExistingReservation(t *testing.T) {
	ctx, _, s := newManagerStore(t)
	names := sampleNames("workspace-a")
	_, _, err := s.ReserveSampleProjectInstance(ctx, &store.SampleProjectInstanceMessage{
		WorkspaceID: "workspace-a",
		ProjectID:   "project-a",
		InstanceID:  names.InstanceID,
		DBName:      names.Database,
		RoleName:    names.Role,
	})
	require.NoError(t, err)
	manager := NewManagerFromURL(s, "", &fakeMetadata{}, &fakeSyncer{}, ManagerOptions{})

	_, err = manager.Prepare(ctx, PrepareRequest{WorkspaceID: "workspace-a", ProjectID: "project-a"})
	require.ErrorIs(t, err, ErrFailedPrecondition)
	require.NoError(t, s.WithLockedSampleProjectInstance(ctx, "workspace-a", func(context.Context, *store.SampleProjectInstanceTx, *store.SampleProjectInstanceMessage) error {
		return nil
	}))
}

func TestManagerPrepareDoesNotRemoveResourcesAfterDeterministicTargetCollision(t *testing.T) {
	ctx, _, s := newManagerStore(t)
	target := &fakeTarget{
		provisionErr: NewTargetError(TargetErrorInvariant, errors.New("deterministic target collision")),
	}
	manager := NewManager(s, target, &fakeMetadata{}, &fakeSyncer{}, ManagerOptions{})

	_, err := manager.Prepare(ctx, PrepareRequest{WorkspaceID: "workspace-a", ProjectID: "project-a"})
	require.ErrorIs(t, err, ErrInternal)
	require.Empty(t, target.removes)
	require.Error(t, s.WithLockedSampleProjectInstance(ctx, "workspace-a", func(context.Context, *store.SampleProjectInstanceTx, *store.SampleProjectInstanceMessage) error {
		return nil
	}))
}

func TestManagerPrepareReservesOneMinuteForCompensation(t *testing.T) {
	ctx, _, s := newManagerStore(t)
	target := &fakeTarget{}
	metadata := &fakeMetadata{}
	syncer := &fakeSyncer{syncErr: errors.New("discovery failed")}
	startedAt := time.Now()
	manager := NewManager(s, target, metadata, syncer, ManagerOptions{})

	_, err := manager.Prepare(ctx, PrepareRequest{WorkspaceID: "workspace-a", ProjectID: "project-a"})
	require.Error(t, err)
	require.False(t, metadata.lookupDeadline.IsZero())
	require.False(t, syncer.deadline.IsZero())
	require.False(t, target.removeDeadline.IsZero())
	require.WithinDuration(t, startedAt.Add(3*time.Minute), metadata.lookupDeadline, 5*time.Second)
	require.WithinDuration(t, startedAt.Add(2*time.Minute), syncer.deadline, 5*time.Second)
	require.WithinDuration(t, startedAt.Add(time.Minute), target.removeDeadline, 5*time.Second)
}

func TestManagerCleanupReturnsDeferredStaticConfigurationFailureWithoutClaiming(t *testing.T) {
	ctx, db, s := newManagerStore(t)
	now := time.Date(2026, time.August, 17, 9, 0, 0, 0, time.UTC)
	_, err := db.ExecContext(ctx, `
		INSERT INTO sample_project_instance (workspace, project, instance, db_name, role_name, created_at)
		VALUES ('workspace-a', 'project-a', 'sample-stale', 'bb_sample_stale', 'bb_sample_role_stale', $1)
	`, now.Add(-time.Hour))
	require.NoError(t, err)
	manager := NewManagerFromURL(s, "not-a-postgres-url", &fakeMetadata{}, &fakeSyncer{}, ManagerOptions{})

	err = manager.Cleanup(ctx, now)
	require.ErrorIs(t, err, ErrFailedPrecondition)
	var count int
	require.NoError(t, db.QueryRowContext(ctx, `SELECT COUNT(*) FROM sample_project_instance WHERE workspace = 'workspace-a'`).Scan(&count))
	require.Equal(t, 1, count)
}

func TestManagerPrepareReconcilesNullReservationBeforeRetrying(t *testing.T) {
	ctx, _, s := newManagerStore(t)
	now := time.Date(2026, time.August, 17, 9, 0, 0, 0, time.UTC)
	names := sampleNames("workspace-a")
	_, _, err := s.ReserveSampleProjectInstance(ctx, &store.SampleProjectInstanceMessage{
		WorkspaceID: "workspace-a",
		ProjectID:   "project-a",
		InstanceID:  names.InstanceID,
		DBName:      names.Database,
		RoleName:    names.Role,
	})
	require.NoError(t, err)
	target := &fakeTarget{}
	metadata := &fakeMetadata{lookupErr: errors.New("metadata unavailable")}
	manager := NewManager(s, target, metadata, &fakeSyncer{databaseName: names.Database}, ManagerOptions{
		Clock:  func() time.Time { return now },
		Random: bytes.NewReader(bytes.Repeat([]byte{2}, 32)),
	})

	_, err = manager.Prepare(ctx, PrepareRequest{WorkspaceID: "workspace-a", ProjectID: "project-a"})
	require.Equal(t, ErrorKindInternal, ErrorKindOf(err))
	require.NoError(t, s.WithLockedSampleProjectInstance(ctx, "workspace-a", func(context.Context, *store.SampleProjectInstanceTx, *store.SampleProjectInstanceMessage) error {
		return nil
	}))
	metadata.lookupErr = nil
	_, err = manager.Prepare(ctx, PrepareRequest{WorkspaceID: "workspace-a", ProjectID: "project-a"})
	require.NoError(t, err)
	require.Equal(t, 1, metadata.removeCalls)
	require.Len(t, target.removes, 1)
	require.Len(t, target.provisions, 1)
	require.NoError(t, s.WithLockedSampleProjectInstance(ctx, "workspace-a", func(_ context.Context, _ *store.SampleProjectInstanceTx, reservation *store.SampleProjectInstanceMessage) error {
		require.Equal(t, now, reservation.CreatedAt)
		require.NotNil(t, reservation.ExpiresAt)
		return nil
	}))
}

func TestManagerCleanupExpiresAndReconcilesStaleReservations(t *testing.T) {
	ctx, db, s := newManagerStore(t)
	now := time.Date(2026, time.August, 17, 9, 0, 0, 0, time.UTC)
	names := sampleNames("workspace-a")
	_, _, err := s.ReserveSampleProjectInstance(ctx, &store.SampleProjectInstanceMessage{
		WorkspaceID: "workspace-a",
		ProjectID:   "project-a",
		InstanceID:  names.InstanceID,
		DBName:      names.Database,
		RoleName:    names.Role,
	})
	require.NoError(t, err)
	require.NoError(t, s.WithLockedSampleProjectInstance(ctx, "workspace-a", func(ctx context.Context, tx *store.SampleProjectInstanceTx, _ *store.SampleProjectInstanceMessage) error {
		return tx.SetExpiration(ctx, now.Add(-time.Second))
	}))
	_, err = db.ExecContext(ctx, `
		INSERT INTO sample_project_instance (workspace, project, instance, db_name, role_name, created_at)
		VALUES ('workspace-b', 'project-b', 'sample-stale', 'bb_sample_stale', 'bb_sample_role_stale', $1)
	`, now.Add(-time.Hour))
	require.NoError(t, err)
	target := &fakeTarget{}
	metadata := &fakeMetadata{}
	manager := NewManager(s, target, metadata, &fakeSyncer{}, ManagerOptions{Clock: func() time.Time { return now }})

	require.NoError(t, manager.Cleanup(ctx, now))
	require.Len(t, target.removes, 2)
	require.Equal(t, 1, metadata.removeCalls)
	require.NoError(t, s.WithLockedSampleProjectInstance(ctx, "workspace-a", func(_ context.Context, _ *store.SampleProjectInstanceTx, message *store.SampleProjectInstanceMessage) error {
		require.Equal(t, &now, message.DeletedAt)
		return nil
	}))
	require.Error(t, s.WithLockedSampleProjectInstance(ctx, "workspace-b", func(context.Context, *store.SampleProjectInstanceTx, *store.SampleProjectInstanceMessage) error {
		return nil
	}))
}

func newManagerStore(t *testing.T) (context.Context, *sql.DB, *store.Store) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
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

func seedReservation(ctx context.Context, t *testing.T, s *store.Store, workspaceID, projectID string, now time.Time) {
	t.Helper()
	names := sampleNames(workspaceID)
	_, _, err := s.ReserveSampleProjectInstance(ctx, &store.SampleProjectInstanceMessage{
		WorkspaceID: workspaceID,
		ProjectID:   projectID,
		InstanceID:  names.InstanceID,
		DBName:      names.Database,
		RoleName:    names.Role,
	})
	require.NoError(t, err)
	require.NoError(t, s.WithLockedSampleProjectInstance(ctx, workspaceID, func(ctx context.Context, tx *store.SampleProjectInstanceTx, _ *store.SampleProjectInstanceMessage) error {
		return tx.SetExpiration(ctx, now.Add(time.Hour))
	}))
}

type fakeTarget struct {
	validateCalls  int
	validateErr    error
	validate       func(context.Context) error
	provisionErr   error
	removeErr      error
	provisions     []Allocation
	removes        []Allocation
	removeDeadline time.Time
}

func (t *fakeTarget) Validate(ctx context.Context) error {
	t.validateCalls++
	if t.validate != nil {
		return t.validate(ctx)
	}
	return t.validateErr
}

func (t *fakeTarget) Provision(_ context.Context, allocation Allocation) error {
	t.provisions = append(t.provisions, allocation)
	return t.provisionErr
}

func (t *fakeTarget) Remove(ctx context.Context, allocation Allocation) error {
	t.removeDeadline, _ = ctx.Deadline()
	t.removes = append(t.removes, allocation)
	return t.removeErr
}

func (*fakeTarget) InstanceConfig(allocation Allocation) (*InstanceConfig, error) {
	return &InstanceConfig{
		AdminDataSource:   &storepb.DataSource{Id: "admin", Username: allocation.Role, Password: allocation.Password},
		SyncDatabaseNames: []string{allocation.Database},
	}, nil
}

type fakeMetadata struct {
	state          MetadataState
	lookupErr      error
	createErr      error
	removeErr      error
	registration   Registration
	removeCalls    int
	lookupDeadline time.Time
}

func (m *fakeMetadata) Lookup(ctx context.Context, _ Allocation, _, _, _ string) (MetadataState, error) {
	m.lookupDeadline, _ = ctx.Deadline()
	return m.state, m.lookupErr
}

func (m *fakeMetadata) Create(_ context.Context, registration Registration) (*store.InstanceMessage, error) {
	m.registration = registration
	if m.createErr != nil {
		return nil, m.createErr
	}
	instance := &store.InstanceMessage{ResourceID: registration.InstanceID}
	m.state = MetadataState{
		ProjectActive:   true,
		InstanceMatches: true,
		Instance:        instance,
		Database:        &store.DatabaseMessage{DatabaseName: registration.Allocation.Database},
	}
	return instance, nil
}

func (m *fakeMetadata) Remove(context.Context, Allocation, string, string, string) error {
	m.removeCalls++
	return m.removeErr
}

type fakeSyncer struct {
	syncErr      error
	asyncCalls   int
	databaseName string
	deadline     time.Time
}

func (s *fakeSyncer) SyncInstance(ctx context.Context, instance *store.InstanceMessage) (*store.InstanceMessage, []*store.DatabaseMessage, error) {
	s.deadline, _ = ctx.Deadline()
	if s.syncErr != nil {
		return nil, nil, s.syncErr
	}
	return instance, []*store.DatabaseMessage{{DatabaseName: s.databaseName}}, nil
}

func (s *fakeSyncer) SyncDatabasesAsync([]*store.DatabaseMessage) {
	s.asyncCalls++
}
