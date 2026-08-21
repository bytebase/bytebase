package store

import (
	"context"
	"database/sql"
	"time"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
)

// SampleProjectInstanceMessage is the control-plane record for a sample
// Project Instance. It deliberately does not reference Bytebase metadata.
type SampleProjectInstanceMessage struct {
	WorkspaceID string
	ProjectID   string
	InstanceID  string
	DBName      string
	RoleName    string
	ReplicaID   string
	CreatedAt   time.Time
	ExpiresAt   *time.Time
	DeletedAt   *time.Time
}

// SampleProjectInstanceTx owns a cleanup-record transaction while physical
// cleanup runs. Preparation uses short Store operations instead.
type SampleProjectInstanceTx struct {
	tx        *sql.Tx
	workspace string
	project   string
	instance  string
	replica   string
}

// GetSampleProjectInstance returns the independent lifecycle record for a
// workspace without locking it.
func (s *Store) GetSampleProjectInstance(ctx context.Context, workspaceID string) (*SampleProjectInstanceMessage, error) {
	message, err := scanSampleProjectInstance(s.GetDB().QueryRowContext(ctx, `
		SELECT workspace, project, instance, db_name, role_name, replica_id,
			created_at, expires_at, deleted_at
		FROM sample_project_instance
		WHERE workspace = $1
	`, workspaceID))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, errors.Wrap(err, "failed to get sample Project Instance")
	}
	return message, nil
}

// ReserveSampleProjectInstance consumes a workspace's lifetime entitlement.
// It commits before a subsequent lifecycle operation locks the reservation.
// It returns whether this call created the reservation.
func (s *Store) ReserveSampleProjectInstance(ctx context.Context, create *SampleProjectInstanceMessage) (*SampleProjectInstanceMessage, bool, error) {
	tx, err := s.GetDB().BeginTx(ctx, nil)
	if err != nil {
		return nil, false, errors.Wrap(err, "failed to begin sample Project Instance reservation transaction")
	}
	defer tx.Rollback()

	var deleted bool
	if err := tx.QueryRowContext(ctx, `
		SELECT deleted
		FROM workspace
		WHERE resource_id = $1
		FOR UPDATE
	`, create.WorkspaceID).Scan(&deleted); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, false, common.Errorf(common.NotFound, "workspace %s not found", create.WorkspaceID)
		}
		return nil, false, errors.Wrapf(err, "failed to lock workspace %s", create.WorkspaceID)
	}
	if deleted {
		return nil, false, common.Errorf(common.NotFound, "workspace %s is deleted", create.WorkspaceID)
	}

	existing, err := scanSampleProjectInstance(tx.QueryRowContext(ctx, `
		SELECT workspace, project, instance, db_name, role_name, replica_id,
			created_at, expires_at, deleted_at
		FROM sample_project_instance
		WHERE workspace = $1
	`, create.WorkspaceID))
	if err == nil {
		if err := tx.Commit(); err != nil {
			return nil, false, errors.Wrap(err, "failed to commit sample Project Instance reservation lookup")
		}
		return existing, false, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return nil, false, errors.Wrap(err, "failed to inspect sample Project Instance reservation")
	}

	reserved, created, err := insertSampleProjectInstance(ctx, tx, create)
	if err != nil {
		return nil, false, err
	}
	if err := tx.Commit(); err != nil {
		return nil, false, errors.Wrap(err, "failed to commit sample Project Instance reservation")
	}
	if !created {
		reserved, err = s.GetSampleProjectInstance(ctx, create.WorkspaceID)
		if err != nil {
			return nil, false, err
		}
		if reserved == nil {
			return nil, false, errors.Errorf("sample Project Instance reservation for workspace %s disappeared", create.WorkspaceID)
		}
	}
	return reserved, created, nil
}

// ClaimSampleProjectInstance transfers a pending reservation to replicaID when
// the caller observed its exact current attempt. A replica may retry its own
// expired attempt; another replica must also observe that the owner heartbeat
// is stale or absent. The returned reservation has the new attempt timestamp.
func (s *Store) ClaimSampleProjectInstance(
	ctx context.Context,
	workspaceID, instanceID string,
	observedCreatedAt time.Time,
	replicaID string,
	attemptAge, heartbeatStaleness time.Duration,
) (*SampleProjectInstanceMessage, bool, error) {
	message, err := scanSampleProjectInstance(s.GetDB().QueryRowContext(ctx, `
		UPDATE sample_project_instance
		SET replica_id = $1,
			created_at = now()
		WHERE workspace = $2
			AND instance = $3
			AND created_at = $4
			AND expires_at IS NULL
			AND deleted_at IS NULL
			AND created_at <= now() - $5::INTERVAL
			AND (
				replica_id = $1
				OR NOT EXISTS (
					SELECT 1
					FROM replica_heartbeat
					WHERE replica_id = sample_project_instance.replica_id
						AND last_heartbeat >= now() - $6::INTERVAL
				)
			)
		RETURNING workspace, project, instance, db_name, role_name, replica_id,
			created_at, expires_at, deleted_at
	`, replicaID, workspaceID, instanceID, observedCreatedAt, attemptAge.String(), heartbeatStaleness.String()))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, errors.Wrap(err, "failed to claim sample Project Instance reservation")
	}
	return message, true, nil
}

// ActivateSampleProjectInstance sets a pending reservation's immutable
// expiration while it is still owned by replicaID. It locks the lifecycle
// record before its Project and workspace, preserving child-to-parent order.
// A false result means another worker changed ownership or lifecycle state.
func (s *Store) ActivateSampleProjectInstance(
	ctx context.Context,
	workspaceID, instanceID, replicaID string,
	expiresAt time.Time,
) (bool, error) {
	tx, err := s.GetDB().BeginTx(ctx, nil)
	if err != nil {
		return false, errors.Wrap(err, "failed to begin sample Project Instance activation transaction")
	}
	defer tx.Rollback()

	message, err := scanSampleProjectInstance(tx.QueryRowContext(ctx, `
		SELECT workspace, project, instance, db_name, role_name, replica_id,
			created_at, expires_at, deleted_at
		FROM sample_project_instance
		WHERE workspace = $1
			AND instance = $2
			AND replica_id = $3
			AND expires_at IS NULL
			AND deleted_at IS NULL
		FOR UPDATE
	`, workspaceID, instanceID, replicaID))
	if errors.Is(err, sql.ErrNoRows) {
		if err := tx.Commit(); err != nil {
			return false, errors.Wrap(err, "failed to commit unchanged sample Project Instance activation")
		}
		return false, nil
	}
	if err != nil {
		return false, errors.Wrap(err, "failed to lock sample Project Instance activation")
	}

	var projectDeleted bool
	if err := tx.QueryRowContext(ctx, `
		SELECT deleted
		FROM project
		WHERE resource_id = $1 AND workspace = $2
		FOR UPDATE
	`, message.ProjectID, message.WorkspaceID).Scan(&projectDeleted); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, common.Errorf(common.NotFound, "project %s not found", message.ProjectID)
		}
		return false, errors.Wrapf(err, "failed to lock project %s for sample Project Instance activation", message.ProjectID)
	}
	if projectDeleted {
		return false, common.Errorf(common.NotFound, "project %s is deleted", message.ProjectID)
	}

	var workspaceDeleted bool
	if err := tx.QueryRowContext(ctx, `
		SELECT deleted
		FROM workspace
		WHERE resource_id = $1
		FOR UPDATE
	`, message.WorkspaceID).Scan(&workspaceDeleted); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, common.Errorf(common.NotFound, "workspace %s not found", message.WorkspaceID)
		}
		return false, errors.Wrapf(err, "failed to lock workspace %s for sample Project Instance activation", message.WorkspaceID)
	}
	if workspaceDeleted {
		return false, common.Errorf(common.NotFound, "workspace %s is deleted", message.WorkspaceID)
	}

	result, err := tx.ExecContext(ctx, `
		UPDATE sample_project_instance
		SET expires_at = $1
		WHERE workspace = $2
			AND instance = $3
			AND replica_id = $4
			AND expires_at IS NULL
			AND deleted_at IS NULL
	`, expiresAt, workspaceID, instanceID, replicaID)
	if err != nil {
		return false, errors.Wrap(err, "failed to activate sample Project Instance")
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return false, errors.Wrap(err, "failed to inspect sample Project Instance activation")
	}
	if err := tx.Commit(); err != nil {
		return false, errors.Wrap(err, "failed to commit sample Project Instance activation")
	}
	return rows == 1, nil
}

// DeletePendingSampleProjectInstance removes a pending reservation only if it
// is still owned by replicaID. A false result means it was superseded or has
// already transitioned to another lifecycle state.
func (s *Store) DeletePendingSampleProjectInstance(ctx context.Context, workspaceID, instanceID, replicaID string) (bool, error) {
	result, err := s.GetDB().ExecContext(ctx, `
		DELETE FROM sample_project_instance
		WHERE workspace = $1
			AND instance = $2
			AND replica_id = $3
			AND expires_at IS NULL
			AND deleted_at IS NULL
	`, workspaceID, instanceID, replicaID)
	if err != nil {
		return false, errors.Wrap(err, "failed to delete pending sample Project Instance reservation")
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return false, errors.Wrap(err, "failed to inspect pending sample Project Instance reservation deletion")
	}
	return rows == 1, nil
}

// MarkDeleted marks an activated reservation as physically deleted.
func (tx *SampleProjectInstanceTx) MarkDeleted(ctx context.Context, deletedAt time.Time) error {
	result, err := tx.tx.ExecContext(ctx, `
		UPDATE sample_project_instance
		SET deleted_at = $1
		WHERE workspace = $2
			AND instance = $3
			AND replica_id = $4
			AND expires_at IS NOT NULL
			AND deleted_at IS NULL
	`, deletedAt, tx.workspace, tx.instance, tx.replica)
	if err != nil {
		return errors.Wrap(err, "failed to mark sample Project Instance deleted")
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return errors.Wrap(err, "failed to inspect sample Project Instance deletion update")
	}
	if rows != 1 {
		return errors.Errorf("sample Project Instance %s cannot be marked deleted", tx.workspace)
	}
	return nil
}

// DeleteReservation removes an unactivated reservation after failed
// provisioning.
func (tx *SampleProjectInstanceTx) DeleteReservation(ctx context.Context) error {
	result, err := tx.tx.ExecContext(ctx, `
		DELETE FROM sample_project_instance
		WHERE workspace = $1
			AND instance = $2
			AND replica_id = $3
			AND expires_at IS NULL
			AND deleted_at IS NULL
	`, tx.workspace, tx.instance, tx.replica)
	if err != nil {
		return errors.Wrap(err, "failed to delete sample Project Instance reservation")
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return errors.Wrap(err, "failed to inspect sample Project Instance reservation deletion")
	}
	if rows != 1 {
		return errors.Errorf("sample Project Instance %s cannot be deleted", tx.workspace)
	}
	return nil
}

// CountSampleProjectInstancesForCleanup reports eligible expired or stale
// reservations without claiming them. It is used only for a validation-failure
// cleanup log.
func (s *Store) CountSampleProjectInstancesForCleanup(ctx context.Context, now, staleBefore time.Time) (int, error) {
	var count int
	if err := s.GetDB().QueryRowContext(ctx, `
		SELECT COUNT(*)
		FROM sample_project_instance
		WHERE deleted_at IS NULL
			AND (
				(expires_at IS NOT NULL AND expires_at <= $1)
				OR (expires_at IS NULL AND created_at <= $2)
			)
	`, now, staleBefore).Scan(&count); err != nil {
		return 0, errors.Wrap(err, "failed to count sample Project Instance cleanup records")
	}
	return count, nil
}

// SampleProjectInstanceCleanupResult reports the outcome of one cleanup
// attempt. CallbackErr does not roll back the transaction: the caller advances
// AfterWorkspace and may report the callback error after attempting later
// records.
type SampleProjectInstanceCleanupResult struct {
	WorkspaceID string
	Found       bool
	CallbackErr error
}

// WithLockedSampleProjectInstanceCleanupRecord locks one expired or stale
// reservation after afterWorkspace in workspace order. The row lock is held
// only while callback runs. A successful callback updates the lifecycle record;
// a failed callback leaves it unchanged after committing and returns its error
// in the result so callers can continue from the returned workspace.
func (s *Store) WithLockedSampleProjectInstanceCleanupRecord(
	ctx context.Context,
	now time.Time,
	staleBefore time.Time,
	afterWorkspace string,
	callback func(context.Context, *SampleProjectInstanceTx, *SampleProjectInstanceMessage) error,
) (*SampleProjectInstanceCleanupResult, error) {
	tx, err := s.GetDB().BeginTx(ctx, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to begin sample Project Instance cleanup transaction")
	}
	defer tx.Rollback()

	message, err := scanSampleProjectInstance(tx.QueryRowContext(ctx, `
		SELECT workspace, project, instance, db_name, role_name, replica_id,
			created_at, expires_at, deleted_at
		FROM sample_project_instance
		WHERE deleted_at IS NULL
			AND (
				(expires_at IS NOT NULL AND expires_at <= $1)
				OR (expires_at IS NULL AND created_at <= $2)
			)
			AND workspace > $3
		ORDER BY workspace
		FOR UPDATE SKIP LOCKED
		LIMIT 1
	`, now, staleBefore, afterWorkspace))
	if errors.Is(err, sql.ErrNoRows) {
		if err := tx.Commit(); err != nil {
			return nil, errors.Wrap(err, "failed to commit empty sample Project Instance cleanup transaction")
		}
		return &SampleProjectInstanceCleanupResult{}, nil
	}
	if err != nil {
		return nil, errors.Wrap(err, "failed to lock sample Project Instance cleanup record")
	}

	result := &SampleProjectInstanceCleanupResult{
		WorkspaceID: message.WorkspaceID,
		Found:       true,
	}
	recordTx := &SampleProjectInstanceTx{tx: tx, workspace: message.WorkspaceID, project: message.ProjectID, instance: message.InstanceID, replica: message.ReplicaID}
	if err := callback(ctx, recordTx, message); err != nil {
		result.CallbackErr = err
	} else if message.ExpiresAt == nil {
		if err := recordTx.DeleteReservation(ctx); err != nil {
			return nil, err
		}
	} else if err := recordTx.MarkDeleted(ctx, now); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, errors.Wrap(err, "failed to commit sample Project Instance cleanup transaction")
	}
	return result, nil
}

func insertSampleProjectInstance(ctx context.Context, tx *sql.Tx, create *SampleProjectInstanceMessage) (*SampleProjectInstanceMessage, bool, error) {
	row := tx.QueryRowContext(ctx, `
		INSERT INTO sample_project_instance (workspace, project, instance, db_name, role_name, replica_id)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (workspace) DO NOTHING
		RETURNING workspace, project, instance, db_name, role_name, replica_id,
			created_at, expires_at, deleted_at
	`, create.WorkspaceID, create.ProjectID, create.InstanceID, create.DBName, create.RoleName, create.ReplicaID)
	message, err := scanSampleProjectInstance(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, false, nil
	}
	return message, err == nil, err
}

type sampleProjectInstanceScanner interface {
	Scan(...any) error
}

func scanSampleProjectInstance(scanner sampleProjectInstanceScanner) (*SampleProjectInstanceMessage, error) {
	message := &SampleProjectInstanceMessage{}
	if err := scanner.Scan(
		&message.WorkspaceID,
		&message.ProjectID,
		&message.InstanceID,
		&message.DBName,
		&message.RoleName,
		&message.ReplicaID,
		&message.CreatedAt,
		&message.ExpiresAt,
		&message.DeletedAt,
	); err != nil {
		return nil, err
	}
	return message, nil
}
