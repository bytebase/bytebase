// Package sampleprojectinstance manages the lifecycle of Cloud sample project
// instances.
package sampleprojectinstance

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"time"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/store"
)

const (
	sampleProjectInstanceTitle = "Sample Project Instance"
	testEnvironmentID          = "test"
	sampleLifetime             = 7 * 24 * time.Hour
	prepareDeadline            = 3 * time.Minute
	provisionDeadline          = 2 * time.Minute
	compensationDeadline       = time.Minute
	cleanupAttemptDeadline     = time.Minute
	staleReservationAge        = time.Hour
)

// ErrorKind classifies lifecycle failures for the API layer.
type ErrorKind string

const (
	// ErrorKindFailedPrecondition indicates an unavailable or already consumed
	// sample-project-instance entitlement.
	ErrorKindFailedPrecondition ErrorKind = "failed_precondition"
	// ErrorKindUnavailable indicates a retryable target or compensation failure.
	ErrorKindUnavailable ErrorKind = "unavailable"
	// ErrorKindInternal indicates an invariant or unexpected internal failure.
	ErrorKindInternal ErrorKind = "internal"
	// ErrorKindDeadlineExceeded indicates lifecycle work exceeded its deadline.
	ErrorKindDeadlineExceeded ErrorKind = "deadline_exceeded"
)

// Error is a typed lifecycle error suitable for Connect status translation.
type Error struct {
	Kind ErrorKind
	Err  error
}

func (e *Error) Error() string {
	if e.Err == nil {
		return string(e.Kind)
	}
	return e.Err.Error()
}

func (e *Error) Unwrap() error {
	return e.Err
}

func (e *Error) Is(target error) bool {
	other, ok := target.(*Error)
	return ok && e.Kind == other.Kind
}

var (
	// ErrFailedPrecondition identifies a consumed or invalid entitlement.
	ErrFailedPrecondition = &Error{Kind: ErrorKindFailedPrecondition}
	// ErrUnavailable identifies retryable target lifecycle failures.
	ErrUnavailable = &Error{Kind: ErrorKindUnavailable}
	// ErrInternal identifies lifecycle invariants and unexpected failures.
	ErrInternal = &Error{Kind: ErrorKindInternal}
	// ErrDeadlineExceeded identifies lifecycle deadline failures.
	ErrDeadlineExceeded = &Error{Kind: ErrorKindDeadlineExceeded}
)

// ErrorKindOf returns the typed manager classification, if available.
func ErrorKindOf(err error) ErrorKind {
	var lifecycleErr *Error
	if errors.As(err, &lifecycleErr) {
		return lifecycleErr.Kind
	}
	return ""
}

func lifecycleError(kind ErrorKind, err error) error {
	return &Error{Kind: kind, Err: err}
}

// committedError tells Prepare to commit the locked control-plane mutation
// before returning the underlying lifecycle failure to its caller.
type committedError struct {
	err error
}

func (e *committedError) Error() string {
	return e.err.Error()
}

func (e *committedError) Unwrap() error {
	return e.err
}

// TargetErrorKind distinguishes static target configuration errors from
// transient target connectivity failures.
type TargetErrorKind string

const (
	// TargetErrorStatic indicates malformed configuration or an insufficient
	// target privilege/isolation baseline.
	TargetErrorStatic TargetErrorKind = "static"
	// TargetErrorUnavailable indicates a reachable target could not complete
	// an operation.
	TargetErrorUnavailable TargetErrorKind = "unavailable"
	// TargetErrorInvariant indicates a deterministic allocation collision or
	// another target state that violates the lifecycle contract.
	TargetErrorInvariant TargetErrorKind = "invariant"
)

// TargetError is returned by Target implementations without exposing
// credentials in the error text.
type TargetError struct {
	Kind TargetErrorKind
	Err  error
}

func (e *TargetError) Error() string {
	if e.Err == nil {
		return string(e.Kind)
	}
	return e.Err.Error()
}

func (e *TargetError) Unwrap() error {
	return e.Err
}

// NewTargetError constructs a typed target error.
func NewTargetError(kind TargetErrorKind, err error) error {
	return &TargetError{Kind: kind, Err: err}
}

// AllocationNames are the deterministic names allocated to one workspace.
type AllocationNames struct {
	InstanceID string
	Database   string
	Role       string
}

func sampleNames(workspaceID string) AllocationNames {
	sum := sha256.Sum256([]byte(workspaceID))
	token := fmt.Sprintf("%x", sum[:16])
	return AllocationNames{
		InstanceID: "sample-" + token,
		Database:   "bb_sample_" + token,
		Role:       "bb_sample_role_" + token,
	}
}

// PrepareRequest identifies the Project that will own the sample instance.
type PrepareRequest struct {
	WorkspaceID string
	ProjectID   string
	CanCreate   func(context.Context) error
}

// Registration is the normal Instance registration requested by Manager.
// Implementations must persist the datasource with Store.CreateInstance so the
// password is encrypted at rest.
type Registration struct {
	WorkspaceID       string
	ProjectID         string
	EnvironmentID     string
	InstanceID        string
	Title             string
	Engine            storepb.Engine
	Allocation        Allocation
	AdminDataSource   *storepb.DataSource
	SyncDatabaseNames []string
}

// MetadataState is the exact resource state needed to decide idempotency.
type MetadataState struct {
	ProjectActive   bool
	InstanceMatches bool
	Instance        *store.InstanceMessage
	Database        *store.DatabaseMessage
}

func (s MetadataState) active() bool {
	return s.ProjectActive && s.InstanceMatches && s.Instance != nil && !s.Instance.Deleted && s.Database != nil && !s.Database.Deleted
}

func (s MetadataState) matches(reservation *store.SampleProjectInstanceMessage) bool {
	return s.active() &&
		s.Instance.ResourceID == reservation.InstanceID &&
		s.Database.InstanceID == reservation.InstanceID &&
		s.Database.DatabaseName == reservation.DBName
}

// MetadataStore owns normal Bytebase Instance and Database metadata. The
// concrete adapter belongs to API/server wiring, not this lifecycle package.
type MetadataStore interface {
	Lookup(ctx context.Context, allocation Allocation, instanceID, workspaceID, projectID string) (MetadataState, error)
	Create(ctx context.Context, registration Registration) (*store.InstanceMessage, error)
	Remove(ctx context.Context, allocation Allocation, instanceID, workspaceID, projectID string) error
}

// SchemaSync performs synchronous database discovery then schedules the
// ordinary asynchronous schema sync.
type SchemaSync interface {
	SyncInstance(ctx context.Context, instance *store.InstanceMessage) (*store.InstanceMessage, []*store.DatabaseMessage, error)
	SyncDatabasesAsync(databases []*store.DatabaseMessage)
}

// TargetService is the direct PostgreSQL target lifecycle boundary.
type TargetService interface {
	Validate(ctx context.Context) error
	Provision(ctx context.Context, allocation Allocation) error
	Remove(ctx context.Context, allocation Allocation) error
	InstanceConfig(allocation Allocation) (*InstanceConfig, error)
}

// ManagerOptions control non-production seams used by lifecycle tests.
type ManagerOptions struct {
	Clock  func() time.Time
	Random io.Reader
	Logger *slog.Logger
}

// Manager orchestrates reservation, physical provisioning, metadata
// registration, discovery, expiration, and cleanup.
type Manager struct {
	store     *store.Store
	target    TargetService
	targetErr error
	metadata  MetadataStore
	schema    SchemaSync
	clock     func() time.Time
	random    io.Reader
	logger    *slog.Logger
}

// NewManagerFromURL creates a startup-safe manager from raw target
// configuration. Invalid static configuration is retained for lifecycle calls
// instead of failing process startup.
func NewManagerFromURL(
	s *store.Store,
	targetURL string,
	metadata MetadataStore,
	schema SchemaSync,
	options ManagerOptions,
) *Manager {
	target, err := NewTarget(targetURL)
	manager := NewManager(s, target, metadata, schema, options)
	manager.targetErr = err
	return manager
}

// NewManager creates a Sample Project Instance lifecycle manager.
func NewManager(
	s *store.Store,
	target TargetService,
	metadata MetadataStore,
	schema SchemaSync,
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
		store:    s,
		target:   target,
		metadata: metadata,
		schema:   schema,
		clock:    options.Clock,
		random:   options.Random,
		logger:   options.Logger,
	}
}

// Prepare provisions and registers a seven-day sample instance for a project.
func (m *Manager) Prepare(ctx context.Context, request PrepareRequest) (*store.InstanceMessage, error) {
	if m.store == nil || (m.target == nil && m.targetErr == nil) || m.metadata == nil || m.schema == nil {
		return nil, lifecycleError(ErrorKindInternal, errors.New("sample project instance manager is not configured"))
	}
	if request.WorkspaceID == "" || request.ProjectID == "" {
		return nil, lifecycleError(ErrorKindFailedPrecondition, errors.New("sample project instance requires workspace and project"))
	}
	names := sampleNames(request.WorkspaceID)
	reservation, created, err := m.store.ReserveSampleProjectInstance(ctx, &store.SampleProjectInstanceMessage{
		WorkspaceID: request.WorkspaceID,
		ProjectID:   request.ProjectID,
		InstanceID:  names.InstanceID,
		DBName:      names.Database,
		RoleName:    names.Role,
	})
	if err != nil {
		return nil, err
	}

	lifecycleCtx, lifecycleCancel := context.WithTimeout(context.WithoutCancel(ctx), prepareDeadline)
	defer lifecycleCancel()
	var instance *store.InstanceMessage
	var discoveredDatabases []*store.DatabaseMessage
	var finalErr error
	err = m.store.WithLockedSampleProjectInstance(lifecycleCtx, reservation.WorkspaceID, func(lockCtx context.Context, tx *store.SampleProjectInstanceTx, locked *store.SampleProjectInstanceMessage) error {
		var prepareErr error
		instance, prepareErr = m.prepareLocked(lockCtx, tx, locked, request, created, &discoveredDatabases)
		var committed *committedError
		if errors.As(prepareErr, &committed) {
			finalErr = committed.err
			return nil
		}
		return prepareErr
	})
	if err != nil {
		return nil, err
	}
	if finalErr != nil {
		return nil, finalErr
	}
	if len(discoveredDatabases) > 0 {
		m.schema.SyncDatabasesAsync(discoveredDatabases)
	}
	return instance, nil
}

func (m *Manager) prepareLocked(
	lifecycleCtx context.Context,
	tx *store.SampleProjectInstanceTx,
	reservation *store.SampleProjectInstanceMessage,
	request PrepareRequest,
	created bool,
	discoveredDatabases *[]*store.DatabaseMessage,
) (*store.InstanceMessage, error) {
	if reservation.ProjectID != request.ProjectID || reservation.DeletedAt != nil {
		return nil, lifecycleError(ErrorKindFailedPrecondition, errors.New("sample project instance entitlement is already consumed"))
	}
	allocation := Allocation{Database: reservation.DBName, Role: reservation.RoleName}
	if reservation.InstanceID != sampleNames(request.WorkspaceID).InstanceID {
		return nil, lifecycleError(ErrorKindInternal, errors.New("sample project instance allocation invariant failed"))
	}

	state, err := m.metadata.Lookup(lifecycleCtx, allocation, reservation.InstanceID, request.WorkspaceID, request.ProjectID)
	if err != nil {
		if created {
			return nil, m.discardReservation(lifecycleCtx, tx, lifecycleError(ErrorKindInternal, errors.Join(errors.New("failed to inspect sample project instance metadata"), err)))
		}
		return nil, lifecycleError(ErrorKindInternal, errors.Join(errors.New("failed to inspect sample project instance metadata"), err))
	}
	if reservation.ExpiresAt != nil {
		if !state.matches(reservation) {
			return nil, lifecycleError(ErrorKindFailedPrecondition, errors.New("sample project instance entitlement is already consumed"))
		}
		return state.Instance, nil
	}
	if created && (state.Instance != nil || state.Database != nil) {
		return nil, m.discardReservation(lifecycleCtx, tx, lifecycleError(ErrorKindInternal, errors.New("sample project instance deterministic metadata collision")))
	}

	workCtx, workCancel := preparationWorkContext(lifecycleCtx)
	defer workCancel()
	if !created {
		if m.targetErr != nil {
			return nil, mapTargetError(m.targetErr)
		}
		m.logger.InfoContext(workCtx, "reconciling stale sample project instance reservation", "workspace", request.WorkspaceID)
		if err := m.reconcile(workCtx, allocation, request); err != nil {
			m.logger.ErrorContext(workCtx, "failed to reconcile stale sample project instance reservation", "workspace", request.WorkspaceID, "error", err)
			return nil, lifecycleError(ErrorKindUnavailable, err)
		}
		if err := tx.ResetCreatedAt(workCtx, m.clock()); err != nil {
			return nil, lifecycleError(ErrorKindInternal, errors.Join(errors.New("failed to reset sample project instance reservation"), err))
		}
	}

	if m.targetErr != nil {
		return nil, m.discardReservation(lifecycleCtx, tx, mapTargetError(m.targetErr))
	}
	if err := m.target.Validate(workCtx); err != nil {
		return nil, m.discardReservation(lifecycleCtx, tx, mapTargetError(err))
	}
	if request.CanCreate != nil {
		if err := request.CanCreate(workCtx); err != nil {
			return nil, m.discardReservation(lifecycleCtx, tx, err)
		}
	}

	password, err := randomPassword(m.random)
	if err != nil {
		return nil, m.discardReservation(lifecycleCtx, tx, lifecycleError(ErrorKindInternal, errors.Join(errors.New("failed to generate sample project instance password"), err)))
	}
	allocation.Password = password
	config, err := m.target.InstanceConfig(allocation)
	if err != nil {
		return nil, m.discardReservation(lifecycleCtx, tx, mapTargetError(err))
	}
	if len(config.SyncDatabaseNames) != 1 || config.SyncDatabaseNames[0] != allocation.Database {
		return nil, m.discardReservation(lifecycleCtx, tx, lifecycleError(ErrorKindInternal, errors.New("sample project instance sync filter invariant failed")))
	}

	m.logger.InfoContext(workCtx, "preparing sample project instance", "workspace", request.WorkspaceID, "project", request.ProjectID)
	provisionCtx, provisionCancel := context.WithTimeout(workCtx, provisionDeadline)
	err = m.target.Provision(provisionCtx, allocation)
	timedOut := errors.Is(provisionCtx.Err(), context.DeadlineExceeded)
	provisionCancel()
	if err != nil {
		if timedOut {
			err = lifecycleError(ErrorKindDeadlineExceeded, context.DeadlineExceeded)
		} else if isTargetInvariant(err) {
			return nil, m.discardReservation(lifecycleCtx, tx, mapTargetError(err))
		} else {
			err = mapTargetError(err)
		}
		return nil, m.compensate(lifecycleCtx, tx, allocation, request, err)
	}
	registered, err := m.metadata.Create(workCtx, Registration{
		WorkspaceID:       request.WorkspaceID,
		ProjectID:         request.ProjectID,
		EnvironmentID:     testEnvironmentID,
		InstanceID:        reservation.InstanceID,
		Title:             sampleProjectInstanceTitle,
		Engine:            storepb.Engine_POSTGRES,
		Allocation:        allocation,
		AdminDataSource:   config.AdminDataSource,
		SyncDatabaseNames: config.SyncDatabaseNames,
	})
	if err != nil {
		return nil, m.compensate(lifecycleCtx, tx, allocation, request, lifecycleError(ErrorKindInternal, errors.Join(errors.New("failed to create sample project instance metadata"), err)))
	}
	synced, databases, err := m.schema.SyncInstance(workCtx, registered)
	if err != nil {
		return nil, m.compensate(lifecycleCtx, tx, allocation, request, mapDiscoveryError(workCtx, err))
	}
	if len(databases) != 1 || databases[0].DatabaseName != allocation.Database || databases[0].Deleted {
		return nil, m.compensate(lifecycleCtx, tx, allocation, request, lifecycleError(ErrorKindInternal, errors.New("sample project instance discovery invariant failed")))
	}
	if err := tx.SetExpiration(workCtx, m.clock().Add(sampleLifetime)); err != nil {
		return nil, m.compensate(lifecycleCtx, tx, allocation, request, lifecycleError(ErrorKindInternal, errors.Join(errors.New("failed to activate sample project instance"), err)))
	}
	*discoveredDatabases = databases
	return synced, nil
}

func (m *Manager) reconcile(ctx context.Context, allocation Allocation, request PrepareRequest) error {
	if err := m.metadata.Remove(ctx, allocation, sampleNames(request.WorkspaceID).InstanceID, request.WorkspaceID, request.ProjectID); err != nil {
		return errors.Join(errors.New("failed to remove partial sample project instance metadata"), err)
	}
	if err := m.target.Remove(ctx, allocation); err != nil {
		return errors.Join(errors.New("failed to remove partial sample project instance target resources"), err)
	}
	return nil
}

func (m *Manager) compensate(
	lifecycleCtx context.Context,
	tx *store.SampleProjectInstanceTx,
	allocation Allocation,
	request PrepareRequest,
	original error,
) error {
	m.logger.WarnContext(lifecycleCtx, "compensating failed sample project instance preparation", "workspace", request.WorkspaceID, "error", original)
	compensationCtx, cancel := context.WithTimeout(lifecycleCtx, compensationDeadline)
	defer cancel()
	metadataErr := m.metadata.Remove(compensationCtx, allocation, sampleNames(request.WorkspaceID).InstanceID, request.WorkspaceID, request.ProjectID)
	targetErr := m.target.Remove(compensationCtx, allocation)
	if metadataErr != nil || targetErr != nil {
		err := errors.Join(metadataErr, targetErr)
		m.logger.ErrorContext(lifecycleCtx, "sample project instance compensation failed", "workspace", request.WorkspaceID, "error", err)
		return lifecycleError(ErrorKindUnavailable, err)
	}
	if err := tx.DeleteReservation(compensationCtx); err != nil {
		m.logger.ErrorContext(lifecycleCtx, "failed to remove compensated sample project instance reservation", "workspace", request.WorkspaceID, "error", err)
		return lifecycleError(ErrorKindUnavailable, err)
	}
	return &committedError{err: original}
}

func (m *Manager) discardReservation(ctx context.Context, tx *store.SampleProjectInstanceTx, original error) error {
	if err := tx.DeleteReservation(ctx); err != nil {
		m.logger.ErrorContext(ctx, "failed to discard sample project instance reservation", "error", err)
		return lifecycleError(ErrorKindUnavailable, err)
	}
	return &committedError{err: original}
}

// Cleanup removes expired target resources and reconciles stale reservations.
func (m *Manager) Cleanup(ctx context.Context, now time.Time) error {
	if m.store == nil || (m.target == nil && m.targetErr == nil) || m.metadata == nil {
		return lifecycleError(ErrorKindInternal, errors.New("sample project instance manager is not configured"))
	}
	targetErr := m.targetErr
	if targetErr == nil {
		targetErr = m.target.Validate(ctx)
	}
	if targetErr != nil {
		count, countErr := m.store.CountSampleProjectInstancesForCleanup(ctx, now, now.Add(-staleReservationAge))
		if countErr != nil {
			m.logger.ErrorContext(ctx, "failed to count outstanding sample project instance cleanup records", "error", countErr)
		}
		m.logger.ErrorContext(ctx, "sample project instance cleanup target validation failed", "outstanding", count, "error", targetErr)
		return mapTargetError(targetErr)
	}
	err := m.store.WithLockedSampleProjectInstanceCleanupBatch(ctx, now, now.Add(-staleReservationAge), func(callbackCtx context.Context, _ *store.SampleProjectInstanceTx, reservation *store.SampleProjectInstanceMessage) error {
		attemptCtx, cancel := context.WithTimeout(callbackCtx, cleanupAttemptDeadline)
		defer cancel()
		allocation := Allocation{Database: reservation.DBName, Role: reservation.RoleName}
		if reservation.ExpiresAt == nil {
			m.logger.InfoContext(callbackCtx, "reconciling stale sample project instance reservation", "workspace", reservation.WorkspaceID)
			err := m.reconcile(attemptCtx, allocation, PrepareRequest{WorkspaceID: reservation.WorkspaceID, ProjectID: reservation.ProjectID})
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
	return err
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

func randomPassword(reader io.Reader) (string, error) {
	bytes := make([]byte, 32)
	if _, err := io.ReadFull(reader, bytes); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(bytes), nil
}

func mapTargetError(err error) error {
	if errors.Is(err, context.DeadlineExceeded) {
		return lifecycleError(ErrorKindDeadlineExceeded, err)
	}
	var targetErr *TargetError
	if errors.As(err, &targetErr) {
		if targetErr.Kind == TargetErrorStatic {
			return lifecycleError(ErrorKindFailedPrecondition, err)
		}
		if targetErr.Kind == TargetErrorInvariant {
			return lifecycleError(ErrorKindInternal, err)
		}
		return lifecycleError(ErrorKindUnavailable, err)
	}
	return lifecycleError(ErrorKindUnavailable, err)
}

func isTargetInvariant(err error) bool {
	var targetErr *TargetError
	return errors.As(err, &targetErr) && targetErr.Kind == TargetErrorInvariant
}

func mapDiscoveryError(ctx context.Context, err error) error {
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return lifecycleError(ErrorKindDeadlineExceeded, context.DeadlineExceeded)
	}
	return mapTargetError(err)
}
