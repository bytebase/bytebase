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
	CreatedAt   time.Time
	ExpiresAt   *time.Time
	DeletedAt   *time.Time
}

// SampleProjectInstanceTx owns a locked sample Project Instance record for the
// duration of a lifecycle operation.
type SampleProjectInstanceTx struct {
	tx        *sql.Tx
	workspace string
}

// GetSampleProjectInstance returns the independent lifecycle record for a
// workspace without locking it.
func (s *Store) GetSampleProjectInstance(ctx context.Context, workspaceID string) (*SampleProjectInstanceMessage, error) {
	message, err := scanSampleProjectInstance(s.GetDB().QueryRowContext(ctx, `
		SELECT workspace, project, instance, db_name, role_name, created_at, expires_at, deleted_at
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
		SELECT workspace, project, instance, db_name, role_name, created_at, expires_at, deleted_at
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

// WithLockedSampleProjectInstance locks one reservation through the callback.
// The callback may set its expiration exactly once, mark an activated
// reservation deleted, or remove an unactivated reservation.
func (s *Store) WithLockedSampleProjectInstance(
	ctx context.Context,
	workspaceID string,
	callback func(context.Context, *SampleProjectInstanceTx, *SampleProjectInstanceMessage) error,
) error {
	tx, err := s.GetDB().BeginTx(ctx, nil)
	if err != nil {
		return errors.Wrap(err, "failed to begin sample Project Instance lifecycle transaction")
	}
	defer tx.Rollback()

	message, err := getLockedSampleProjectInstance(ctx, tx, workspaceID)
	if err != nil {
		return err
	}
	if err := callback(ctx, &SampleProjectInstanceTx{tx: tx, workspace: workspaceID}, message); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return errors.Wrap(err, "failed to commit sample Project Instance lifecycle transaction")
	}
	return nil
}

// SetExpiration sets an immutable expiration on the locked reservation.
func (tx *SampleProjectInstanceTx) SetExpiration(ctx context.Context, expiresAt time.Time) error {
	result, err := tx.tx.ExecContext(ctx, `
		UPDATE sample_project_instance
		SET expires_at = $1
		WHERE workspace = $2
			AND expires_at IS NULL
	`, expiresAt, tx.workspace)
	if err != nil {
		return errors.Wrap(err, "failed to set sample Project Instance expiration")
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return errors.Wrap(err, "failed to inspect sample Project Instance expiration update")
	}
	if rows != 1 {
		return errors.Errorf("sample Project Instance %s expiration is already set", tx.workspace)
	}
	return nil
}

// MarkDeleted marks an activated reservation as physically deleted.
func (tx *SampleProjectInstanceTx) MarkDeleted(ctx context.Context, deletedAt time.Time) error {
	result, err := tx.tx.ExecContext(ctx, `
		UPDATE sample_project_instance
		SET deleted_at = $1
		WHERE workspace = $2
			AND expires_at IS NOT NULL
			AND deleted_at IS NULL
	`, deletedAt, tx.workspace)
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
			AND expires_at IS NULL
	`, tx.workspace)
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

// ResetCreatedAt restarts an unactivated reservation after deterministic
// reconciliation has removed all partially created resources.
func (tx *SampleProjectInstanceTx) ResetCreatedAt(ctx context.Context, createdAt time.Time) error {
	result, err := tx.tx.ExecContext(ctx, `
		UPDATE sample_project_instance
		SET created_at = $1
		WHERE workspace = $2
			AND expires_at IS NULL
			AND deleted_at IS NULL
	`, createdAt, tx.workspace)
	if err != nil {
		return errors.Wrap(err, "failed to reset sample Project Instance reservation creation time")
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return errors.Wrap(err, "failed to inspect sample Project Instance reservation reset")
	}
	if rows != 1 {
		return errors.Errorf("sample Project Instance %s cannot be reset", tx.workspace)
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
		SELECT workspace, project, instance, db_name, role_name, created_at, expires_at, deleted_at
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
	recordTx := &SampleProjectInstanceTx{tx: tx, workspace: message.WorkspaceID}
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
		INSERT INTO sample_project_instance (workspace, project, instance, db_name, role_name)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (workspace) DO NOTHING
		RETURNING workspace, project, instance, db_name, role_name, created_at, expires_at, deleted_at
	`, create.WorkspaceID, create.ProjectID, create.InstanceID, create.DBName, create.RoleName)
	message, err := scanSampleProjectInstance(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, false, nil
	}
	return message, err == nil, err
}

func getLockedSampleProjectInstance(ctx context.Context, tx *sql.Tx, workspaceID string) (*SampleProjectInstanceMessage, error) {
	row := tx.QueryRowContext(ctx, `
		SELECT workspace, project, instance, db_name, role_name, created_at, expires_at, deleted_at
		FROM sample_project_instance
		WHERE workspace = $1
		FOR UPDATE
	`, workspaceID)
	message, err := scanSampleProjectInstance(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, common.Errorf(common.NotFound, "sample Project Instance for workspace %s not found", workspaceID)
	}
	return message, err
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
		&message.CreatedAt,
		&message.ExpiresAt,
		&message.DeletedAt,
	); err != nil {
		return nil, err
	}
	return message, nil
}
