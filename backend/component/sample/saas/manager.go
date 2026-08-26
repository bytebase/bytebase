// Package saas manages project-level sample instances backed by an external
// PostgreSQL target.
package saas

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"sync"
	"time"

	"google.golang.org/protobuf/encoding/protojson"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/component/sample"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/runner/schemasync"
	"github.com/bytebase/bytebase/backend/store"
)

const (
	sampleTitle                = "Sample Project Instance"
	sampleLifetime             = 7 * 24 * time.Hour
	prepareDeadline            = 3 * time.Minute
	provisionDeadline          = 2 * time.Minute
	compensationDeadline       = time.Minute
	cleanupValidationDeadline  = 10 * time.Second
	targetAvailabilityDeadline = 3 * time.Second
	targetAvailabilityCacheTTL = time.Minute
	cleanupAttemptDeadline     = time.Minute
	staleReservationAge        = time.Hour
	replicaHeartbeatStaleness  = time.Minute
	pollInitialDelay           = 10 * time.Millisecond
	pollMaxDelay               = time.Second
)

// Manager is the SaaS sample implementation.
type Manager struct {
	store     *store.Store
	target    *postgresTarget
	syncer    *schemasync.Syncer
	clock     func() time.Time
	random    io.Reader
	logger    *slog.Logger
	replicaID string

	availabilityMu        sync.Mutex
	availabilityCheckedAt time.Time
	available             bool
}

// NewManager creates a SaaS sample manager from a PostgreSQL target URL.
func NewManager(stores *store.Store, targetURL string, syncer *schemasync.Syncer, options sample.ManagerOptions) (*Manager, error) {
	target, err := newPostgresTarget(targetURL)
	if err != nil {
		return nil, err
	}
	if options.Clock == nil {
		options.Clock = time.Now
	}
	if options.Random == nil {
		options.Random = rand.Reader
	}
	if options.Logger == nil {
		options.Logger = slog.Default()
	}
	return &Manager{
		store:     stores,
		target:    target,
		syncer:    syncer,
		clock:     options.Clock,
		random:    options.Random,
		logger:    options.Logger,
		replicaID: options.ReplicaID,
	}, nil
}

func sampleNames(instanceID string) (string, string) {
	sum := sha256.Sum256([]byte(instanceID))
	token := fmt.Sprintf("%x", sum[:8])
	return "bb_sample_" + token, "bb_sample_role_" + token
}

// CheckAvailable checks cached target readiness, refreshing stale state with a
// short probe.
func (m *Manager) CheckAvailable(ctx context.Context) error {
	if m == nil || m.target == nil {
		return sample.NewFailure(sample.FailureUnavailable, errors.New("SaaS sample target is not configured"))
	}
	m.availabilityMu.Lock()
	defer m.availabilityMu.Unlock()
	now := m.clock()
	if !m.availabilityCheckedAt.IsZero() && now.Before(m.availabilityCheckedAt.Add(targetAvailabilityCacheTTL)) {
		if m.available {
			return nil
		}
		return sample.NewFailure(sample.FailureUnavailable, errors.New("SaaS sample target is unavailable"))
	}
	validationCtx, cancel := context.WithTimeout(ctx, targetAvailabilityDeadline)
	defer cancel()
	err := m.target.validate(validationCtx)
	m.available = err == nil
	m.availabilityCheckedAt = m.clock()
	if err != nil {
		return sample.NewFailure(sample.FailureUnavailable, err)
	}
	return nil
}

func (m *Manager) recordAvailability(available bool) {
	m.availabilityMu.Lock()
	defer m.availabilityMu.Unlock()
	m.available = available
	m.availabilityCheckedAt = m.clock()
}

// PrepareSampleProjectInstance provisions and registers one seven-day sample instance.
func (m *Manager) PrepareSampleProjectInstance(ctx context.Context, request sample.PrepareRequest) (*store.InstanceMessage, error) {
	if m == nil || m.store == nil || m.target == nil || m.syncer == nil || m.replicaID == "" {
		return nil, errors.New("SaaS sample manager is not configured")
	}
	if request.WorkspaceID == "" || request.ProjectID == "" {
		return nil, sample.NewFailure(sample.FailureFailedPrecondition, errors.New("SaaS sample requires workspace and project"))
	}
	lifecycleCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), prepareDeadline)
	defer cancel()
	setup, created, err := m.reserve(lifecycleCtx, request)
	if err != nil {
		return nil, err
	}
	delay := pollInitialDelay
	for {
		payload, err := decodePayload(setup)
		if err != nil {
			return nil, err
		}
		if payload.ProjectId != request.ProjectID || setup.DeletedAt != nil {
			return nil, sample.NewFailure(sample.FailureFailedPrecondition, errors.New("sample setup entitlement is already consumed"))
		}
		if created {
			return m.prepareOwned(lifecycleCtx, setup, payload, false)
		}
		if setup.ActivatedAt != nil {
			return m.activeSetup(lifecycleCtx, setup, payload)
		}
		claimed, ok, err := m.store.ClaimSampleInstanceSetup(
			lifecycleCtx,
			setup.WorkspaceID,
			setup.UpdatedAt,
			m.replicaID,
			prepareDeadline,
			replicaHeartbeatStaleness,
		)
		if err != nil {
			return nil, errors.Join(errors.New("failed to claim abandoned SaaS sample setup"), err)
		}
		if ok {
			claimedPayload, err := decodePayload(claimed)
			if err != nil {
				return nil, err
			}
			return m.prepareOwned(lifecycleCtx, claimed, claimedPayload, true)
		}
		if err := sample.SleepContext(lifecycleCtx, delay); err != nil {
			return nil, sample.NewFailure(sample.FailureDeadlineExceeded, err)
		}
		if delay < pollMaxDelay {
			delay *= 2
			if delay > pollMaxDelay {
				delay = pollMaxDelay
			}
		}
		setup, err = m.store.GetSampleInstanceSetup(lifecycleCtx, request.WorkspaceID)
		if err != nil {
			return nil, errors.Join(errors.New("failed to poll SaaS sample setup"), err)
		}
		created = false
		if setup == nil {
			setup, created, err = m.reserve(lifecycleCtx, request)
			if err != nil {
				return nil, err
			}
		}
	}
}

func (m *Manager) reserve(ctx context.Context, request sample.PrepareRequest) (*store.SampleInstanceSetupMessage, bool, error) {
	instanceID, err := randomInstanceID(m.random)
	if err != nil {
		return nil, false, errors.Join(errors.New("failed to generate SaaS sample instance ID"), err)
	}
	database, role := sampleNames(instanceID)
	payload := &storepb.SaaSSampleInstanceSetupPayload{
		ProjectId:    request.ProjectID,
		InstanceId:   instanceID,
		Title:        sampleTitle,
		DatabaseName: database,
		RoleName:     role,
	}
	environments, err := m.store.GetEnvironment(ctx, request.WorkspaceID)
	if err != nil {
		return nil, false, errors.Join(errors.New("failed to list SaaS sample environments"), err)
	}
	if values := environments.GetEnvironments(); len(values) > 0 {
		payload.EnvironmentId = &values[0].Id
	}
	encoded, err := protojson.Marshal(payload)
	if err != nil {
		return nil, false, errors.Join(errors.New("failed to encode SaaS sample setup"), err)
	}
	return m.store.ReserveSampleInstanceSetup(ctx, &store.SampleInstanceSetupMessage{
		WorkspaceID: request.WorkspaceID,
		ReplicaID:   m.replicaID,
		Payload:     encoded,
	})
}

func decodePayload(setup *store.SampleInstanceSetupMessage) (*storepb.SaaSSampleInstanceSetupPayload, error) {
	if setup == nil {
		return nil, sample.NewFailure(sample.FailureFailedPrecondition, errors.New("sample setup is missing"))
	}
	payload := &storepb.SaaSSampleInstanceSetupPayload{}
	if err := common.ProtojsonUnmarshaler.Unmarshal(setup.Payload, payload); err != nil {
		return nil, sample.NewFailure(sample.FailureFailedPrecondition, errors.New("invalid SaaS sample setup payload"))
	}
	if payload.ProjectId == "" || payload.InstanceId == "" || payload.DatabaseName == "" || payload.RoleName == "" {
		return nil, sample.NewFailure(sample.FailureFailedPrecondition, errors.New("incomplete SaaS sample setup payload"))
	}
	return payload, nil
}

func (m *Manager) prepareOwned(ctx context.Context, setup *store.SampleInstanceSetupMessage, payload *storepb.SaaSSampleInstanceSetupPayload, takeover bool) (*store.InstanceMessage, error) {
	workCtx, cancel := preparationWorkContext(ctx)
	defer cancel()
	if takeover {
		if err := m.reconcile(workCtx, setup, payload); err != nil {
			return nil, sample.NewFailure(sample.FailureUnavailable, err)
		}
	}
	validationCtx, validationCancel := context.WithTimeout(workCtx, cleanupValidationDeadline)
	err := m.target.validate(validationCtx)
	validationCancel()
	m.recordAvailability(err == nil)
	if err != nil {
		return nil, m.discardReservation(ctx, setup, mapTargetError(err))
	}

	password, err := randomPassword(m.random)
	if err != nil {
		return nil, m.discardReservation(ctx, setup, err)
	}
	physical := allocation{database: payload.DatabaseName, role: payload.RoleName, password: password}
	config, err := m.target.instanceConfig(physical)
	if err != nil {
		return nil, m.discardReservation(ctx, setup, mapTargetError(err))
	}
	provisionCtx, provisionCancel := context.WithTimeout(workCtx, provisionDeadline)
	err = m.target.provision(provisionCtx, physical)
	timedOut := errors.Is(provisionCtx.Err(), context.DeadlineExceeded)
	provisionCancel()
	if err != nil {
		if timedOut {
			err = sample.NewFailure(sample.FailureDeadlineExceeded, context.DeadlineExceeded)
		} else {
			err = mapTargetError(err)
		}
		return m.compensate(ctx, setup, payload, err)
	}
	projectID := payload.ProjectId
	registered, err := sample.CreateMetadata(workCtx, m.store, sample.Registration{
		WorkspaceID:       setup.WorkspaceID,
		InstanceProjectID: &projectID,
		EnvironmentID:     payload.EnvironmentId,
		InstanceID:        payload.InstanceId,
		Title:             payload.Title,
		AdminDataSource:   config.adminDataSource,
		SyncDatabaseNames: config.syncDatabaseNames,
	})
	if err != nil {
		return m.compensate(ctx, setup, payload, errors.Join(errors.New("failed to create SaaS sample metadata"), err))
	}
	synced, _, databases, err := m.syncer.SyncInstance(workCtx, registered)
	if err != nil {
		return m.compensate(ctx, setup, payload, errors.Join(errors.New("failed to discover SaaS sample database"), err))
	}
	if len(databases) != 1 || databases[0].DatabaseName != payload.DatabaseName || databases[0].Deleted {
		return m.compensate(ctx, setup, payload, errors.New("SaaS sample database discovery invariant failed"))
	}
	if err := sample.TransferDatabase(workCtx, m.store, payload.ProjectId, payload.InstanceId, payload.DatabaseName); err != nil {
		return m.compensate(ctx, setup, payload, err)
	}
	activatedAt := m.clock()
	expiresAt := activatedAt.Add(sampleLifetime)
	activated, err := m.store.ActivateSampleInstanceSetup(
		workCtx,
		setup.WorkspaceID,
		m.replicaID,
		[]string{payload.ProjectId},
		activatedAt,
		&expiresAt,
	)
	if err != nil || !activated {
		return m.handleActivationFailure(ctx, setup, payload, errors.Join(errors.New("failed to activate SaaS sample setup"), err))
	}
	m.syncer.SyncDatabasesAsync(databases)
	return synced, nil
}

func (m *Manager) activeSetup(ctx context.Context, setup *store.SampleInstanceSetupMessage, payload *storepb.SaaSSampleInstanceSetupPayload) (*store.InstanceMessage, error) {
	projectID := payload.ProjectId
	state, err := sample.LookupMetadata(ctx, m.store, sample.MetadataLookup{
		WorkspaceID:       setup.WorkspaceID,
		InstanceProjectID: &projectID,
		DatabaseProjectID: payload.ProjectId,
		InstanceID:        payload.InstanceId,
		DatabaseName:      payload.DatabaseName,
	})
	if err != nil {
		return nil, err
	}
	if !state.Active() {
		return nil, sample.NewFailure(sample.FailureFailedPrecondition, errors.New("SaaS sample metadata is not active"))
	}
	return state.Instance, nil
}

func (m *Manager) reconcile(ctx context.Context, setup *store.SampleInstanceSetupMessage, payload *storepb.SaaSSampleInstanceSetupPayload) error {
	projectID := payload.ProjectId
	metadataErr := sample.PurgePartialMetadata(ctx, m.store, setup.WorkspaceID, &projectID, payload.InstanceId)
	targetErr := m.target.remove(ctx, allocation{database: payload.DatabaseName, role: payload.RoleName})
	return errors.Join(metadataErr, targetErr)
}

func (m *Manager) compensate(ctx context.Context, setup *store.SampleInstanceSetupMessage, payload *storepb.SaaSSampleInstanceSetupPayload, original error) (*store.InstanceMessage, error) {
	compensationCtx, cancel := context.WithTimeout(ctx, compensationDeadline)
	defer cancel()
	current, err := m.store.GetSampleInstanceSetup(compensationCtx, setup.WorkspaceID)
	if err != nil {
		return nil, sample.NewFailure(sample.FailureUnavailable, errors.Join(original, err))
	}
	if current != nil && current.ActivatedAt != nil && current.DeletedAt == nil {
		currentPayload, err := decodePayload(current)
		if err != nil {
			return nil, err
		}
		return m.activeSetup(compensationCtx, current, currentPayload)
	}
	if current == nil || current.ReplicaID != m.replicaID || current.DeletedAt != nil {
		return nil, sample.NewFailure(sample.FailureUnavailable, errors.Join(original, errors.New("SaaS sample setup ownership changed before compensation")))
	}
	if err := m.reconcile(compensationCtx, current, payload); err != nil {
		return nil, sample.NewFailure(sample.FailureUnavailable, errors.Join(original, err))
	}
	deleted, err := m.store.DeletePendingSampleInstanceSetup(compensationCtx, setup.WorkspaceID, m.replicaID)
	if err != nil || !deleted {
		return nil, sample.NewFailure(sample.FailureUnavailable, errors.Join(original, err))
	}
	return nil, original
}

func (m *Manager) discardReservation(ctx context.Context, setup *store.SampleInstanceSetupMessage, original error) error {
	deleted, err := m.store.DeletePendingSampleInstanceSetup(ctx, setup.WorkspaceID, m.replicaID)
	if err != nil || !deleted {
		return sample.NewFailure(sample.FailureUnavailable, errors.Join(original, err))
	}
	return original
}

func (m *Manager) handleActivationFailure(ctx context.Context, setup *store.SampleInstanceSetupMessage, payload *storepb.SaaSSampleInstanceSetupPayload, original error) (*store.InstanceMessage, error) {
	current, err := m.store.GetSampleInstanceSetup(ctx, setup.WorkspaceID)
	if err != nil {
		return nil, sample.NewFailure(sample.FailureUnavailable, errors.Join(original, err))
	}
	if current != nil && current.ActivatedAt != nil && current.DeletedAt == nil {
		currentPayload, err := decodePayload(current)
		if err != nil {
			return nil, err
		}
		return m.activeSetup(ctx, current, currentPayload)
	}
	if current == nil || current.ReplicaID != m.replicaID || current.DeletedAt != nil {
		return nil, sample.NewFailure(sample.FailureUnavailable, original)
	}
	return m.compensate(ctx, setup, payload, original)
}

// ListInstances returns the provisioned SaaS instances.
func (m *Manager) ListInstances(ctx context.Context, workspaceID string) ([]*sample.Instance, error) {
	if m == nil {
		return nil, nil
	}
	setup, err := m.store.GetSampleInstanceSetup(ctx, workspaceID)
	if err != nil || setup == nil || setup.ActivatedAt == nil {
		return nil, err
	}
	payload, err := decodePayload(setup)
	if err != nil {
		return nil, err
	}
	instance := &sample.Instance{Name: common.FormatInstance(payload.InstanceId)}
	if setup.DeletedAt == nil {
		instance.ExpireTime = setup.ExpiresAt
	}
	return []*sample.Instance{instance}, nil
}

// Start is a no-op because the external PostgreSQL server is already running.
func (*Manager) Start(context.Context, string) error { return nil }

// Cleanup removes stale pending and expired SaaS sample setups.
func (m *Manager) Cleanup(ctx context.Context) error {
	if m == nil || m.store == nil || m.target == nil {
		return errors.New("SaaS sample manager is not configured")
	}
	now := m.clock()
	staleBefore := now.Add(-staleReservationAge)
	count, err := m.store.CountSampleInstanceSetupsForCleanup(ctx, now, staleBefore)
	if err != nil || count == 0 {
		return err
	}
	validationCtx, cancel := context.WithTimeout(ctx, cleanupValidationDeadline)
	err = m.target.validateForCleanup(validationCtx)
	cancel()
	if err != nil {
		return mapTargetError(err)
	}
	var cleanupErr error
	for afterWorkspace := ""; ; {
		result, err := m.store.WithLockedSampleInstanceSetupForCleanup(
			ctx,
			now,
			staleBefore,
			afterWorkspace,
			func(callbackCtx context.Context, tx *store.SampleInstanceSetupTx, setup *store.SampleInstanceSetupMessage) error {
				attemptCtx, attemptCancel := context.WithTimeout(callbackCtx, cleanupAttemptDeadline)
				defer attemptCancel()
				payload, err := decodePayload(setup)
				if err != nil {
					return err
				}
				if setup.ActivatedAt == nil {
					if err := m.reconcile(attemptCtx, setup, payload); err != nil {
						return err
					}
					return tx.DeleteReservation(attemptCtx)
				}
				if err := m.target.remove(attemptCtx, allocation{database: payload.DatabaseName, role: payload.RoleName}); err != nil {
					return err
				}
				projectID := payload.ProjectId
				if _, err := sample.ArchiveMetadata(attemptCtx, m.store, setup.WorkspaceID, &projectID, payload.InstanceId); err != nil {
					return err
				}
				return tx.MarkDeleted(attemptCtx, now)
			},
		)
		if err != nil {
			return errors.Join(cleanupErr, err)
		}
		if !result.Found {
			return cleanupErr
		}
		afterWorkspace = result.WorkspaceID
		cleanupErr = errors.Join(cleanupErr, result.CallbackErr)
	}
}

// ValidateInstanceRestore rejects restoring a physically deleted sample.
func (m *Manager) ValidateInstanceRestore(ctx context.Context, workspaceID, instanceID string) error {
	setup, err := m.store.GetSampleInstanceSetup(ctx, workspaceID)
	if err != nil || setup == nil {
		return err
	}
	payload, err := decodePayload(setup)
	if err != nil {
		return err
	}
	if payload.InstanceId == instanceID && setup.DeletedAt != nil {
		return sample.NewFailure(sample.FailureFailedPrecondition, errors.New("expired sample instance cannot be restored"))
	}
	return nil
}

// HandleInstanceLifecycle is a no-op because user archive/restore does not
// change external PostgreSQL resources before expiration.
func (*Manager) HandleInstanceLifecycle(context.Context, string, string, bool) error { return nil }

// Stop is a no-op for the external target.
func (*Manager) Stop() {}

func preparationWorkContext(lifecycleCtx context.Context) (context.Context, context.CancelFunc) {
	workDeadline := time.Now().Add(provisionDeadline)
	if lifecycleDeadline, ok := lifecycleCtx.Deadline(); ok {
		reservedDeadline := lifecycleDeadline.Add(-compensationDeadline)
		if reservedDeadline.Before(workDeadline) {
			workDeadline = reservedDeadline
		}
	}
	return context.WithDeadline(lifecycleCtx, workDeadline)
}

func randomPassword(reader io.Reader) (string, error) {
	value := make([]byte, 32)
	if _, err := io.ReadFull(reader, value); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(value), nil
}

func randomInstanceID(reader io.Reader) (string, error) {
	value := make([]byte, 8)
	if _, err := io.ReadFull(reader, value); err != nil {
		return "", err
	}
	return "sample-" + hex.EncodeToString(value), nil
}

func mapTargetError(err error) error {
	if errors.Is(err, context.DeadlineExceeded) {
		return sample.NewFailure(sample.FailureDeadlineExceeded, err)
	}
	if isStaticTargetError(err) {
		return sample.NewFailure(sample.FailureFailedPrecondition, err)
	}
	return sample.NewFailure(sample.FailureUnavailable, err)
}
