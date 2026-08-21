// Package sampleprojectinstance manages the lifecycle of Cloud sample project
// instances.
package sampleprojectinstance

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
	"time"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/runner/schemasync"
	"github.com/bytebase/bytebase/backend/store"
)

const (
	sampleProjectInstanceTitle = "Sample Project Instance"
	testEnvironmentID          = "test"
	sampleLifetime             = 7 * 24 * time.Hour
	prepareDeadline            = 3 * time.Minute
	provisionDeadline          = 2 * time.Minute
	compensationDeadline       = time.Minute
	cleanupValidationDeadline  = 10 * time.Second
	cleanupAttemptDeadline     = time.Minute
	staleReservationAge        = time.Hour
	replicaHeartbeatStaleness  = time.Minute
	pollInitialDelay           = 10 * time.Millisecond
	pollMaxDelay               = time.Second
)

// FailureKind classifies failures at the Manager seam.
type FailureKind string

const (
	// FailureUnknown is the default for unexpected failures.
	FailureUnknown FailureKind = "unknown"
	// FailureFailedPrecondition indicates an unavailable or already consumed
	// Sample Project Instance entitlement.
	FailureFailedPrecondition FailureKind = "failed_precondition"
	// FailureUnavailable indicates a retryable target or compensation failure.
	FailureUnavailable FailureKind = "unavailable"
	// FailureDeadlineExceeded indicates lifecycle work exceeded its deadline.
	FailureDeadlineExceeded FailureKind = "deadline_exceeded"
)

type failure struct {
	kind FailureKind
	err  error
}

func (e *failure) Error() string {
	if e.err == nil {
		return string(e.kind)
	}
	return e.err.Error()
}

func (e *failure) Unwrap() error {
	return e.err
}

// FailureKindOf returns the Manager failure classification.
func FailureKindOf(err error) FailureKind {
	var classified *failure
	if errors.As(err, &classified) {
		return classified.kind
	}
	return FailureUnknown
}

func newFailure(kind FailureKind, err error) error {
	if kind == FailureUnknown {
		return err
	}
	return &failure{kind: kind, err: err}
}

// AllocationNames are the deterministic names allocated to one persisted reservation.
type AllocationNames struct {
	Database string
	Role     string
}

func sampleNames(instanceID string) AllocationNames {
	sum := sha256.Sum256([]byte(instanceID))
	token := fmt.Sprintf("%x", sum[:16])
	return AllocationNames{
		Database: "bb_sample_" + token,
		Role:     "bb_sample_role_" + token,
	}
}

// PrepareRequest identifies the Project that will own the sample instance.
type PrepareRequest struct {
	WorkspaceID       string
	ProjectID         string
	CheckCreatePolicy func(context.Context) (CreatePolicyResult, error)
}

// CreatePolicyResult is the transport-neutral capacity decision supplied by
// the API adapter. A nil DeniedReason allows creation.
type CreatePolicyResult struct {
	DeniedReason error
}

// PrepareResult is the result of preparation. PolicyDenied is set only when
// creation was rejected by the supplied capacity policy.
type PrepareResult struct {
	Instance     *store.InstanceMessage
	PolicyDenied error
}

// registration is the normal Instance registration requested by Manager.
// createMetadata persists the datasource with Store.CreateInstance so the
// password is encrypted at rest.
type registration struct {
	WorkspaceID       string
	ProjectID         string
	EnvironmentID     string
	InstanceID        string
	Title             string
	Engine            storepb.Engine
	AdminDataSource   *storepb.DataSource
	SyncDatabaseNames []string
}

// metadataState is the exact resource state needed to decide idempotency.
type metadataState struct {
	ProjectActive   bool
	InstanceMatches bool
	Instance        *store.InstanceMessage
	Database        *store.DatabaseMessage
}

func (s metadataState) active() bool {
	return s.ProjectActive && s.InstanceMatches && s.Instance != nil && !s.Instance.Deleted && s.Database != nil && !s.Database.Deleted
}

func (s metadataState) matches(reservation *store.SampleProjectInstanceMessage) bool {
	return s.active() &&
		s.Instance.ResourceID == reservation.InstanceID &&
		s.Database.InstanceID == reservation.InstanceID &&
		s.Database.DatabaseName == reservation.DBName
}

// ManagerOptions control manager identity and lifecycle test seams.
type ManagerOptions struct {
	Clock     func() time.Time
	Random    io.Reader
	Logger    *slog.Logger
	ReplicaID string
}

// Manager orchestrates reservation, physical provisioning, metadata
// registration, discovery, expiration, and cleanup.
type Manager struct {
	store     *store.Store
	target    *Target
	targetErr error
	syncer    *schemasync.Syncer
	clock     func() time.Time
	random    io.Reader
	logger    *slog.Logger
	replicaID string
}

// NewManagerFromURL creates a startup-safe manager from raw target
// configuration. Invalid static configuration is retained for lifecycle calls
// instead of failing process startup.
func NewManagerFromURL(
	s *store.Store,
	targetURL string,
	syncer *schemasync.Syncer,
	options ManagerOptions,
) *Manager {
	target, err := NewTarget(targetURL)
	manager := NewManager(s, target, syncer, options)
	manager.targetErr = err
	return manager
}

// NewManager creates a Sample Project Instance lifecycle manager.
func NewManager(
	s *store.Store,
	target *Target,
	syncer *schemasync.Syncer,
	options ManagerOptions,
) *Manager {
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
		store:     s,
		target:    target,
		syncer:    syncer,
		clock:     options.Clock,
		random:    options.Random,
		logger:    options.Logger,
		replicaID: options.ReplicaID,
	}
}

// Prepare provisions and registers a seven-day sample instance for a project.
func (m *Manager) Prepare(ctx context.Context, request PrepareRequest) (*PrepareResult, error) {
	if m.store == nil || (m.target == nil && m.targetErr == nil) || m.syncer == nil || m.replicaID == "" {
		return nil, errors.New("sample project instance manager is not configured")
	}
	if request.WorkspaceID == "" || request.ProjectID == "" {
		return nil, newFailure(FailureFailedPrecondition, errors.New("sample project instance requires workspace and project"))
	}
	lifecycleCtx, lifecycleCancel := context.WithTimeout(context.WithoutCancel(ctx), prepareDeadline)
	defer lifecycleCancel()
	reservation, created, err := m.reserve(lifecycleCtx, request)
	if err != nil {
		return nil, err
	}
	delay := pollInitialDelay
	for {
		if created {
			return m.prepareOwned(lifecycleCtx, reservation, request, false)
		}
		if reservation.ProjectID != request.ProjectID || reservation.DeletedAt != nil {
			return nil, newFailure(FailureFailedPrecondition, errors.New("sample project instance entitlement is already consumed"))
		}
		if reservation.ExpiresAt != nil {
			return m.activeReservation(lifecycleCtx, reservation, request)
		}

		// ClaimSampleProjectInstance atomically validates that the observed
		// reservation has not changed and that the current attempt has exceeded
		// prepareDeadline. A different owner must also have a stale heartbeat.
		claimed, claimedOK, err := m.store.ClaimSampleProjectInstance(lifecycleCtx, reservation.WorkspaceID, reservation.InstanceID, reservation.CreatedAt, m.replicaID, prepareDeadline, replicaHeartbeatStaleness)
		if err != nil {
			return nil, errors.Join(errors.New("failed to claim abandoned sample project instance reservation"), err)
		}
		if claimedOK {
			return m.prepareOwned(lifecycleCtx, claimed, request, true)
		}
		if err := sleepContext(lifecycleCtx, delay); err != nil {
			return nil, newFailure(FailureDeadlineExceeded, err)
		}
		if delay < pollMaxDelay {
			delay *= 2
			if delay > pollMaxDelay {
				delay = pollMaxDelay
			}
		}
		reservation, err = m.store.GetSampleProjectInstance(lifecycleCtx, request.WorkspaceID)
		if err != nil {
			return nil, errors.Join(errors.New("failed to poll sample project instance reservation"), err)
		}
		created = false
		if reservation == nil {
			reservation, created, err = m.reserve(lifecycleCtx, request)
			if err != nil {
				return nil, err
			}
		}
	}
}

func (m *Manager) reserve(ctx context.Context, request PrepareRequest) (*store.SampleProjectInstanceMessage, bool, error) {
	instanceID, err := randomInstanceID(m.random)
	if err != nil {
		return nil, false, errors.Join(errors.New("failed to generate sample project instance ID"), err)
	}
	names := sampleNames(instanceID)
	return m.store.ReserveSampleProjectInstance(ctx, &store.SampleProjectInstanceMessage{
		WorkspaceID: request.WorkspaceID,
		ProjectID:   request.ProjectID,
		InstanceID:  instanceID,
		DBName:      names.Database,
		RoleName:    names.Role,
		ReplicaID:   m.replicaID,
	})
}

func (m *Manager) activeReservation(
	ctx context.Context,
	reservation *store.SampleProjectInstanceMessage,
	request PrepareRequest,
) (*PrepareResult, error) {
	state, err := m.lookupMetadata(ctx, Allocation{Database: reservation.DBName, Role: reservation.RoleName}, reservation.InstanceID, request.WorkspaceID, request.ProjectID)
	if err != nil {
		return nil, errors.Join(errors.New("failed to inspect sample project instance metadata"), err)
	}
	if !state.matches(reservation) {
		return nil, newFailure(FailureFailedPrecondition, errors.New("sample project instance entitlement is already consumed"))
	}
	return &PrepareResult{Instance: state.Instance}, nil
}

func (m *Manager) prepareOwned(
	lifecycleCtx context.Context,
	reservation *store.SampleProjectInstanceMessage,
	request PrepareRequest,
	takeover bool,
) (*PrepareResult, error) {
	allocation := Allocation{Database: reservation.DBName, Role: reservation.RoleName}

	state, err := m.lookupMetadata(lifecycleCtx, allocation, reservation.InstanceID, request.WorkspaceID, request.ProjectID)
	if err != nil {
		err = errors.Join(errors.New("failed to inspect sample project instance metadata"), err)
		if takeover {
			return nil, newFailure(FailureUnavailable, err)
		}
		return nil, m.discardReservation(lifecycleCtx, reservation, err)
	}
	if reservation.ExpiresAt != nil {
		if !state.matches(reservation) {
			return nil, newFailure(FailureFailedPrecondition, errors.New("sample project instance entitlement is already consumed"))
		}
		return &PrepareResult{Instance: state.Instance}, nil
	}
	if !takeover && (state.Instance != nil || state.Database != nil) {
		return nil, m.discardReservation(lifecycleCtx, reservation, errors.New("sample project instance metadata collision"))
	}

	workCtx, workCancel := preparationWorkContext(lifecycleCtx)
	defer workCancel()
	if takeover {
		if m.targetErr != nil {
			return nil, mapTargetError(m.targetErr)
		}
		m.logger.InfoContext(workCtx, "reconciling stale sample project instance reservation", "workspace", request.WorkspaceID)
		if err := m.reconcile(workCtx, allocation, reservation.InstanceID, request); err != nil {
			m.logger.ErrorContext(workCtx, "failed to reconcile stale sample project instance reservation", "workspace", request.WorkspaceID, "error", err)
			return nil, newFailure(FailureUnavailable, err)
		}
	}

	if m.targetErr != nil {
		return nil, m.discardReservation(lifecycleCtx, reservation, mapTargetError(m.targetErr))
	}
	if err := m.target.Validate(workCtx); err != nil {
		return nil, m.discardReservation(lifecycleCtx, reservation, mapTargetError(err))
	}
	if request.CheckCreatePolicy != nil {
		policy, err := request.CheckCreatePolicy(workCtx)
		if err != nil {
			return nil, m.discardReservation(lifecycleCtx, reservation, err)
		}
		if policy.DeniedReason != nil {
			return m.denyByPolicy(lifecycleCtx, reservation, policy.DeniedReason)
		}
	}
	environments, err := m.store.GetEnvironment(workCtx, request.WorkspaceID)
	if err != nil {
		return nil, m.discardReservation(lifecycleCtx, reservation, errors.Join(errors.New("failed to inspect sample project instance environments"), err))
	}
	environmentID := ""
	for _, environment := range environments.GetEnvironments() {
		if environmentID == "" || environment.Id == testEnvironmentID {
			environmentID = environment.Id
		}
		if environment.Id == testEnvironmentID {
			break
		}
	}
	if environmentID == "" {
		return nil, m.discardReservation(lifecycleCtx, reservation, newFailure(FailureFailedPrecondition, errors.New("sample project instance requires an environment")))
	}

	password, err := randomPassword(m.random)
	if err != nil {
		return nil, m.discardReservation(lifecycleCtx, reservation, errors.Join(errors.New("failed to generate sample project instance password"), err))
	}
	allocation.Password = password
	config, err := m.target.InstanceConfig(allocation)
	if err != nil {
		return nil, m.discardReservation(lifecycleCtx, reservation, mapTargetError(err))
	}
	if len(config.SyncDatabaseNames) != 1 || config.SyncDatabaseNames[0] != allocation.Database {
		return nil, m.discardReservation(lifecycleCtx, reservation, errors.New("sample project instance sync filter invariant failed"))
	}

	m.logger.InfoContext(workCtx, "preparing sample project instance", "workspace", request.WorkspaceID, "project", request.ProjectID)
	provisionCtx, provisionCancel := context.WithTimeout(workCtx, provisionDeadline)
	err = m.target.Provision(provisionCtx, allocation)
	timedOut := errors.Is(provisionCtx.Err(), context.DeadlineExceeded)
	provisionCancel()
	if err != nil {
		return m.compensate(lifecycleCtx, reservation, allocation, request, mapProvisionError(err, timedOut))
	}
	registered, err := m.createMetadata(workCtx, registration{
		WorkspaceID:       request.WorkspaceID,
		ProjectID:         request.ProjectID,
		EnvironmentID:     environmentID,
		InstanceID:        reservation.InstanceID,
		Title:             sampleProjectInstanceTitle,
		Engine:            storepb.Engine_POSTGRES,
		AdminDataSource:   config.AdminDataSource,
		SyncDatabaseNames: config.SyncDatabaseNames,
	})
	if err != nil {
		return m.compensate(lifecycleCtx, reservation, allocation, request, errors.Join(errors.New("failed to create sample project instance metadata"), err))
	}
	synced, _, databases, err := m.syncer.SyncInstance(workCtx, registered)
	if err != nil {
		return m.compensate(lifecycleCtx, reservation, allocation, request, mapDiscoveryError(workCtx, err))
	}
	if len(databases) != 1 || databases[0].DatabaseName != allocation.Database || databases[0].Deleted {
		return m.compensate(lifecycleCtx, reservation, allocation, request, errors.New("sample project instance discovery invariant failed"))
	}
	activated, activationErr := m.store.ActivateSampleProjectInstance(workCtx, reservation.WorkspaceID, reservation.InstanceID, m.replicaID, m.clock().Add(sampleLifetime))
	if !activated {
		activationErr = errors.Join(errors.New("failed to activate sample project instance"), activationErr)
		if common.ErrorCode(activationErr) == common.NotFound {
			activationErr = newFailure(FailureFailedPrecondition, activationErr)
		}
		return m.handleActivationFailure(lifecycleCtx, reservation, allocation, request, activationErr)
	}
	m.syncer.SyncDatabasesAsync(databases)
	return &PrepareResult{Instance: synced}, nil
}

func mapProvisionError(err error, timedOut bool) error {
	if timedOut {
		return newFailure(FailureDeadlineExceeded, context.DeadlineExceeded)
	}
	return mapTargetError(err)
}

func (m *Manager) reconcile(ctx context.Context, allocation Allocation, instanceID string, request PrepareRequest) error {
	metadataErr := m.removeMetadata(ctx, instanceID, request.WorkspaceID, request.ProjectID)
	if metadataErr != nil {
		metadataErr = errors.Join(errors.New("failed to remove partial sample project instance metadata"), metadataErr)
	}
	targetErr := m.target.Remove(ctx, allocation)
	if targetErr != nil {
		targetErr = errors.Join(errors.New("failed to remove partial sample project instance target resources"), targetErr)
	}
	return errors.Join(metadataErr, targetErr)
}

func (m *Manager) compensate(
	lifecycleCtx context.Context,
	reservation *store.SampleProjectInstanceMessage,
	allocation Allocation,
	request PrepareRequest,
	original error,
) (*PrepareResult, error) {
	m.logger.WarnContext(lifecycleCtx, "compensating failed sample project instance preparation", "workspace", request.WorkspaceID, "error", original)
	compensationCtx, cancel := context.WithTimeout(lifecycleCtx, compensationDeadline)
	defer cancel()
	current, err := m.store.GetSampleProjectInstance(compensationCtx, reservation.WorkspaceID)
	if err != nil {
		return nil, newFailure(FailureUnavailable, errors.Join(original, errors.New("failed to verify sample project instance compensation ownership"), err))
	}
	if current == nil || current.InstanceID != reservation.InstanceID || current.ReplicaID != m.replicaID || current.ExpiresAt != nil || current.DeletedAt != nil {
		return nil, newFailure(FailureUnavailable, errors.Join(original, errors.New("sample project instance reservation ownership changed before compensation")))
	}
	if err := m.reconcile(compensationCtx, allocation, reservation.InstanceID, request); err != nil {
		m.logger.ErrorContext(lifecycleCtx, "sample project instance compensation failed", "workspace", request.WorkspaceID, "error", err)
		return nil, newFailure(FailureUnavailable, err)
	}
	deleted, err := m.store.DeletePendingSampleProjectInstance(compensationCtx, reservation.WorkspaceID, reservation.InstanceID, m.replicaID)
	if err != nil || !deleted {
		if err == nil {
			err = errors.New("sample project instance reservation ownership changed during compensation")
		}
		m.logger.ErrorContext(lifecycleCtx, "failed to remove compensated sample project instance reservation", "workspace", request.WorkspaceID, "error", err)
		return nil, newFailure(FailureUnavailable, err)
	}
	return nil, original
}

func (m *Manager) discardReservation(ctx context.Context, reservation *store.SampleProjectInstanceMessage, original error) error {
	deleted, err := m.store.DeletePendingSampleProjectInstance(ctx, reservation.WorkspaceID, reservation.InstanceID, m.replicaID)
	if err != nil || !deleted {
		if err == nil {
			err = errors.New("sample project instance reservation ownership changed before deletion")
		}
		m.logger.ErrorContext(ctx, "failed to discard sample project instance reservation", "error", err)
		return newFailure(FailureUnavailable, err)
	}
	return original
}

func (m *Manager) denyByPolicy(ctx context.Context, reservation *store.SampleProjectInstanceMessage, reason error) (*PrepareResult, error) {
	deleted, err := m.store.DeletePendingSampleProjectInstance(ctx, reservation.WorkspaceID, reservation.InstanceID, m.replicaID)
	if err != nil || !deleted {
		if err == nil {
			err = errors.New("sample project instance reservation ownership changed before policy denial")
		}
		m.logger.ErrorContext(ctx, "failed to discard denied sample project instance reservation", "error", err)
		return nil, newFailure(FailureUnavailable, err)
	}
	return &PrepareResult{PolicyDenied: reason}, nil
}

// handleActivationFailure avoids compensating resources after ownership might
// have changed. Only an exact, still-pending reservation owned by this replica
// proves that this attempt may safely compensate.
func (m *Manager) handleActivationFailure(
	ctx context.Context,
	reservation *store.SampleProjectInstanceMessage,
	allocation Allocation,
	request PrepareRequest,
	original error,
) (*PrepareResult, error) {
	current, err := m.store.GetSampleProjectInstance(ctx, reservation.WorkspaceID)
	if err != nil {
		return nil, newFailure(FailureUnavailable, errors.Join(original, err))
	}
	if current != nil && current.InstanceID == reservation.InstanceID && current.ExpiresAt != nil {
		return m.activeReservation(ctx, current, request)
	}
	if current == nil || current.InstanceID != reservation.InstanceID || current.ReplicaID != m.replicaID || current.ExpiresAt != nil || current.DeletedAt != nil {
		return nil, newFailure(FailureUnavailable, original)
	}
	return m.compensate(ctx, reservation, allocation, request, original)
}

// Cleanup removes expired target resources and reconciles stale reservations.
func (m *Manager) Cleanup(ctx context.Context, now time.Time) error {
	if m.store == nil || (m.target == nil && m.targetErr == nil) {
		return errors.New("sample project instance manager is not configured")
	}
	targetErr := m.targetErr
	if targetErr == nil {
		validationCtx, validationCancel := context.WithTimeout(ctx, cleanupValidationDeadline)
		targetErr = m.target.ValidateForCleanup(validationCtx)
		validationCancel()
	}
	if targetErr != nil {
		count, countErr := m.store.CountSampleProjectInstancesForCleanup(ctx, now, now.Add(-staleReservationAge))
		if countErr != nil {
			m.logger.ErrorContext(ctx, "failed to count outstanding sample project instance cleanup records", "error", countErr)
		}
		m.logger.ErrorContext(ctx, "sample project instance cleanup target validation failed", "outstanding", count, "error", targetErr)
		return mapTargetError(targetErr)
	}
	var cleanupErr error
	for afterWorkspace := ""; ; {
		result, err := m.store.WithLockedSampleProjectInstanceCleanupRecord(ctx, now, now.Add(-staleReservationAge), afterWorkspace, func(callbackCtx context.Context, _ *store.SampleProjectInstanceTx, reservation *store.SampleProjectInstanceMessage) error {
			attemptCtx, cancel := context.WithTimeout(callbackCtx, cleanupAttemptDeadline)
			defer cancel()
			allocation := Allocation{Database: reservation.DBName, Role: reservation.RoleName}
			if reservation.ExpiresAt == nil {
				m.logger.InfoContext(callbackCtx, "reconciling stale sample project instance reservation", "workspace", reservation.WorkspaceID)
				err := m.reconcile(attemptCtx, allocation, reservation.InstanceID, PrepareRequest{WorkspaceID: reservation.WorkspaceID, ProjectID: reservation.ProjectID})
				if err != nil {
					m.logger.ErrorContext(callbackCtx, "failed to reconcile stale sample project instance reservation", "workspace", reservation.WorkspaceID, "error", err)
				}
				return err
			}
			m.logger.InfoContext(callbackCtx, "expiring sample project instance", "workspace", reservation.WorkspaceID)
			if err := m.target.Remove(attemptCtx, allocation); err != nil {
				m.logger.ErrorContext(callbackCtx, "failed to expire sample project instance", "workspace", reservation.WorkspaceID, "error", err)
				return err
			}
			return nil
		})
		if err != nil {
			return errors.Join(cleanupErr, err)
		}
		if !result.Found {
			return cleanupErr
		}
		afterWorkspace = result.WorkspaceID
		if result.CallbackErr != nil {
			cleanupErr = errors.Join(cleanupErr, result.CallbackErr)
		}
	}
}

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

func sleepContext(ctx context.Context, duration time.Duration) error {
	timer := time.NewTimer(duration)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func randomPassword(reader io.Reader) (string, error) {
	bytes := make([]byte, 32)
	if _, err := io.ReadFull(reader, bytes); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(bytes), nil
}

func randomInstanceID(reader io.Reader) (string, error) {
	bytes := make([]byte, 16)
	if _, err := io.ReadFull(reader, bytes); err != nil {
		return "", err
	}
	return "sample-" + hex.EncodeToString(bytes), nil
}

func mapTargetError(err error) error {
	if errors.Is(err, context.DeadlineExceeded) {
		return newFailure(FailureDeadlineExceeded, err)
	}
	if isStaticTargetError(err) {
		return newFailure(FailureFailedPrecondition, err)
	}
	return newFailure(FailureUnavailable, err)
}

func mapDiscoveryError(ctx context.Context, err error) error {
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return newFailure(FailureDeadlineExceeded, context.DeadlineExceeded)
	}
	return mapTargetError(err)
}
