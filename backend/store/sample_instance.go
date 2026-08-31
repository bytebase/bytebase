package store

import (
	"context"
	"database/sql"
	"slices"
	"time"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
)

// SampleInstanceSetupMessage is the persisted lifecycle envelope for one
// implementation-owned sample setup.
type SampleInstanceSetupMessage struct {
	WorkspaceID string
	ReplicaID   string
	Payload     []byte
	CreatedAt   time.Time
	UpdatedAt   time.Time
	ActivatedAt *time.Time
	ExpiresAt   *time.Time
	DeletedAt   *time.Time
}

// SampleInstanceSetupTx owns a locked setup transaction during cleanup.
type SampleInstanceSetupTx struct {
	tx        *sql.Tx
	workspace string
	replica   string
}

// SampleInstanceCleanupResult describes one locked cleanup attempt.
type SampleInstanceCleanupResult struct {
	WorkspaceID string
	Found       bool
	CallbackErr error
}

// GetSampleInstanceSetup returns the setup for a workspace.
func (s *Store) GetSampleInstanceSetup(ctx context.Context, workspaceID string) (*SampleInstanceSetupMessage, error) {
	return getSampleInstanceSetup(ctx, s.GetDB(), workspaceID)
}

// WithLockedSampleInstanceSetup locks one undeleted setup for a lifecycle
// operation and invokes callback in the same transaction.
func (s *Store) WithLockedSampleInstanceSetup(ctx context.Context, workspaceID string, callback func(context.Context, *SampleInstanceSetupTx, *SampleInstanceSetupMessage) error) error {
	tx, err := s.GetDB().BeginTx(ctx, nil)
	if err != nil {
		return errors.Wrap(err, "failed to begin sample instance setup lifecycle")
	}
	defer tx.Rollback()

	setup, err := scanSampleInstanceSetup(tx.QueryRowContext(ctx, `
		SELECT workspace, replica_id, payload, created_at, updated_at,
			activated_at, expires_at, deleted_at
		FROM sample_instance_setup
		WHERE workspace = $1 AND deleted_at IS NULL
		FOR UPDATE
	`, workspaceID))
	if errors.Is(err, sql.ErrNoRows) {
		return errors.Wrap(tx.Commit(), "failed to commit empty sample instance setup lifecycle")
	}
	if err != nil {
		return errors.Wrap(err, "failed to lock sample instance setup lifecycle")
	}
	if err := callback(ctx, &SampleInstanceSetupTx{tx: tx, workspace: workspaceID, replica: setup.ReplicaID}, setup); err != nil {
		return err
	}
	return errors.Wrap(tx.Commit(), "failed to commit sample instance setup lifecycle")
}

// ReserveSampleInstanceSetup atomically consumes a workspace's setup
// entitlement and persists the complete implementation payload.
func (s *Store) ReserveSampleInstanceSetup(ctx context.Context, create *SampleInstanceSetupMessage) (*SampleInstanceSetupMessage, bool, error) {
	tx, err := s.GetDB().BeginTx(ctx, nil)
	if err != nil {
		return nil, false, errors.Wrap(err, "failed to begin sample instance setup reservation")
	}
	defer tx.Rollback()

	var deleted bool
	if err := tx.QueryRowContext(ctx, `
		SELECT deleted FROM workspace WHERE resource_id = $1 FOR UPDATE
	`, create.WorkspaceID).Scan(&deleted); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, false, common.Errorf(common.NotFound, "workspace %s not found", create.WorkspaceID)
		}
		return nil, false, errors.Wrapf(err, "failed to lock workspace %s", create.WorkspaceID)
	}
	if deleted {
		return nil, false, common.Errorf(common.NotFound, "workspace %s is deleted", create.WorkspaceID)
	}

	existing, err := getSampleInstanceSetup(ctx, tx, create.WorkspaceID)
	if err != nil {
		return nil, false, err
	}
	if existing != nil {
		if err := tx.Commit(); err != nil {
			return nil, false, errors.Wrap(err, "failed to commit sample instance setup lookup")
		}
		return existing, false, nil
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO sample_instance_setup (workspace, replica_id, payload)
		VALUES ($1, $2, $3::jsonb)
	`, create.WorkspaceID, create.ReplicaID, create.Payload); err != nil {
		return nil, false, errors.Wrap(err, "failed to reserve sample instance setup")
	}
	reserved, err := getSampleInstanceSetup(ctx, tx, create.WorkspaceID)
	if err != nil {
		return nil, false, err
	}
	if err := tx.Commit(); err != nil {
		return nil, false, errors.Wrap(err, "failed to commit sample instance setup reservation")
	}
	return reserved, true, nil
}

// ClaimSampleInstanceSetup transfers an abandoned pending setup to replicaID.
func (s *Store) ClaimSampleInstanceSetup(ctx context.Context, workspaceID string, observedUpdatedAt time.Time, replicaID string, attemptAge, heartbeatStaleness time.Duration) (*SampleInstanceSetupMessage, bool, error) {
	var workspace string
	err := s.GetDB().QueryRowContext(ctx, `
		UPDATE sample_instance_setup
		SET replica_id = $1, updated_at = now()
		WHERE workspace = $2
			AND updated_at = $3
			AND activated_at IS NULL
			AND deleted_at IS NULL
			AND updated_at <= now() - $4::INTERVAL
			AND (
				replica_id = $1
				OR NOT EXISTS (
					SELECT 1 FROM replica_heartbeat
					WHERE replica_id = sample_instance_setup.replica_id
						AND last_heartbeat >= now() - $5::INTERVAL
				)
			)
		RETURNING workspace
	`, replicaID, workspaceID, observedUpdatedAt, attemptAge.String(), heartbeatStaleness.String()).Scan(&workspace)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, errors.Wrap(err, "failed to claim sample instance setup")
	}
	setup, err := s.GetSampleInstanceSetup(ctx, workspace)
	return setup, err == nil, err
}

// ActivateSampleInstanceSetup marks a setup active after validating its owning
// workspace and projects are still active.
func (s *Store) ActivateSampleInstanceSetup(ctx context.Context, workspaceID, replicaID string, projectIDs []string, activatedAt time.Time, expiresAt *time.Time) (bool, error) {
	tx, err := s.GetDB().BeginTx(ctx, nil)
	if err != nil {
		return false, errors.Wrap(err, "failed to begin sample instance setup activation")
	}
	defer tx.Rollback()

	setup, err := scanSampleInstanceSetup(tx.QueryRowContext(ctx, `
		SELECT workspace, replica_id, payload, created_at, updated_at,
			activated_at, expires_at, deleted_at
		FROM sample_instance_setup
		WHERE workspace = $1 AND replica_id = $2
			AND activated_at IS NULL AND deleted_at IS NULL
		FOR UPDATE
	`, workspaceID, replicaID))
	if errors.Is(err, sql.ErrNoRows) {
		if err := tx.Commit(); err != nil {
			return false, errors.Wrap(err, "failed to commit unchanged sample instance setup activation")
		}
		return false, nil
	}
	if err != nil {
		return false, errors.Wrap(err, "failed to lock sample instance setup activation")
	}

	projects := slices.Clone(projectIDs)
	slices.Sort(projects)
	projects = slices.Compact(projects)
	for _, projectID := range projects {
		var deleted bool
		if err := tx.QueryRowContext(ctx, `
			SELECT deleted FROM project
			WHERE workspace = $1 AND resource_id = $2
			FOR UPDATE
		`, workspaceID, projectID).Scan(&deleted); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return false, common.Errorf(common.NotFound, "project %s not found", projectID)
			}
			return false, errors.Wrapf(err, "failed to lock project %s for sample instance setup activation", projectID)
		}
		if deleted {
			return false, common.Errorf(common.NotFound, "project %s is deleted", projectID)
		}
	}

	var workspaceDeleted bool
	if err := tx.QueryRowContext(ctx, `
		SELECT deleted FROM workspace WHERE resource_id = $1 FOR UPDATE
	`, workspaceID).Scan(&workspaceDeleted); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, common.Errorf(common.NotFound, "workspace %s not found", workspaceID)
		}
		return false, errors.Wrapf(err, "failed to lock workspace %s", workspaceID)
	}
	if workspaceDeleted {
		return false, common.Errorf(common.NotFound, "workspace %s is deleted", workspaceID)
	}

	result, err := tx.ExecContext(ctx, `
		UPDATE sample_instance_setup
		SET activated_at = $1, expires_at = $2, updated_at = $1
		WHERE workspace = $3 AND replica_id = $4
			AND activated_at IS NULL AND deleted_at IS NULL
	`, activatedAt, expiresAt, setup.WorkspaceID, setup.ReplicaID)
	if err != nil {
		return false, errors.Wrap(err, "failed to activate sample instance setup")
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return false, errors.Wrap(err, "failed to inspect sample instance setup activation")
	}
	if err := tx.Commit(); err != nil {
		return false, errors.Wrap(err, "failed to commit sample instance setup activation")
	}
	return rows == 1, nil
}

// DeletePendingSampleInstanceSetup removes an unactivated setup still owned by
// replicaID.
func (s *Store) DeletePendingSampleInstanceSetup(ctx context.Context, workspaceID, replicaID string) (bool, error) {
	result, err := s.GetDB().ExecContext(ctx, `
		DELETE FROM sample_instance_setup
		WHERE workspace = $1 AND replica_id = $2
			AND activated_at IS NULL AND deleted_at IS NULL
	`, workspaceID, replicaID)
	if err != nil {
		return false, errors.Wrap(err, "failed to delete pending sample instance setup")
	}
	rows, err := result.RowsAffected()
	return rows == 1, errors.Wrap(err, "failed to inspect pending sample instance setup deletion")
}

// CountSampleInstanceSetupsForCleanup counts stale pending or expired active
// setups.
func (s *Store) CountSampleInstanceSetupsForCleanup(ctx context.Context, now, staleBefore time.Time) (int, error) {
	var count int
	err := s.GetDB().QueryRowContext(ctx, `
		SELECT COUNT(*) FROM sample_instance_setup
		WHERE deleted_at IS NULL AND (
			(activated_at IS NULL AND updated_at <= $2)
			OR (activated_at IS NOT NULL AND expires_at IS NOT NULL AND expires_at <= $1)
		)
	`, now, staleBefore).Scan(&count)
	return count, errors.Wrap(err, "failed to count sample instance setups for cleanup")
}

// WithLockedSampleInstanceSetupForCleanup locks one eligible setup of the
// requested lifecycle state and invokes callback in the same transaction.
func (s *Store) WithLockedSampleInstanceSetupForCleanup(ctx context.Context, now, staleBefore time.Time, afterWorkspace string, callback func(context.Context, *SampleInstanceSetupTx, *SampleInstanceSetupMessage) error) (*SampleInstanceCleanupResult, error) {
	tx, err := s.GetDB().BeginTx(ctx, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to begin sample instance cleanup")
	}
	defer tx.Rollback()

	var workspace string
	err = tx.QueryRowContext(ctx, `
		SELECT workspace FROM sample_instance_setup
		WHERE deleted_at IS NULL AND (
			(activated_at IS NULL AND updated_at <= $2)
			OR (activated_at IS NOT NULL AND expires_at IS NOT NULL AND expires_at <= $1)
		)
			AND workspace > $3
		ORDER BY workspace
		FOR UPDATE SKIP LOCKED
		LIMIT 1
	`, now, staleBefore, afterWorkspace).Scan(&workspace)
	if errors.Is(err, sql.ErrNoRows) {
		if err := tx.Commit(); err != nil {
			return nil, errors.Wrap(err, "failed to commit empty sample instance cleanup")
		}
		return &SampleInstanceCleanupResult{}, nil
	}
	if err != nil {
		return nil, errors.Wrap(err, "failed to lock sample instance setup for cleanup")
	}
	setup, err := getSampleInstanceSetup(ctx, tx, workspace)
	if err != nil {
		return nil, err
	}
	result := &SampleInstanceCleanupResult{WorkspaceID: workspace, Found: true}
	if err := callback(ctx, &SampleInstanceSetupTx{tx: tx, workspace: workspace, replica: setup.ReplicaID}, setup); err != nil {
		result.CallbackErr = err
	}
	if err := tx.Commit(); err != nil {
		return nil, errors.Wrap(err, "failed to commit sample instance cleanup")
	}
	return result, nil
}

// MarkDeleted marks an activated setup physically removed.
func (tx *SampleInstanceSetupTx) MarkDeleted(ctx context.Context, deletedAt time.Time) error {
	result, err := tx.tx.ExecContext(ctx, `
		UPDATE sample_instance_setup SET deleted_at = $1, updated_at = $1
		WHERE workspace = $2 AND replica_id = $3
			AND activated_at IS NOT NULL AND deleted_at IS NULL
	`, deletedAt, tx.workspace, tx.replica)
	if err != nil {
		return errors.Wrap(err, "failed to mark sample instance setup deleted")
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return errors.Wrap(err, "failed to inspect sample instance setup deletion")
	}
	if rows != 1 {
		return errors.Errorf("sample instance setup %q cannot be marked deleted", tx.workspace)
	}
	return nil
}

// DeleteReservation removes the locked pending setup.
func (tx *SampleInstanceSetupTx) DeleteReservation(ctx context.Context) error {
	result, err := tx.tx.ExecContext(ctx, `
		DELETE FROM sample_instance_setup
		WHERE workspace = $1 AND replica_id = $2
			AND activated_at IS NULL AND deleted_at IS NULL
	`, tx.workspace, tx.replica)
	if err != nil {
		return errors.Wrap(err, "failed to delete sample instance setup reservation")
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return errors.Wrap(err, "failed to inspect sample instance setup reservation deletion")
	}
	if rows != 1 {
		return errors.Errorf("sample instance setup %q cannot be deleted", tx.workspace)
	}
	return nil
}

type sampleInstanceDBTX interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

func getSampleInstanceSetup(ctx context.Context, db sampleInstanceDBTX, workspaceID string) (*SampleInstanceSetupMessage, error) {
	setup, err := scanSampleInstanceSetup(db.QueryRowContext(ctx, `
		SELECT workspace, replica_id, payload, created_at, updated_at,
			activated_at, expires_at, deleted_at
		FROM sample_instance_setup WHERE workspace = $1
	`, workspaceID))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, errors.Wrap(err, "failed to get sample instance setup")
	}
	return setup, nil
}

type sampleInstanceScanner interface {
	Scan(...any) error
}

func scanSampleInstanceSetup(scanner sampleInstanceScanner) (*SampleInstanceSetupMessage, error) {
	setup := &SampleInstanceSetupMessage{}
	err := scanner.Scan(
		&setup.WorkspaceID,
		&setup.ReplicaID,
		&setup.Payload,
		&setup.CreatedAt,
		&setup.UpdatedAt,
		&setup.ActivatedAt,
		&setup.ExpiresAt,
		&setup.DeletedAt,
	)
	return setup, err
}
