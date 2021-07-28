package store

import (
	"context"
	"fmt"
	"strings"

	"github.com/bytebase/bytebase"
	"github.com/bytebase/bytebase/api"
	"go.uber.org/zap"
)

var (
	_ api.BackupService = (*BackupService)(nil)
)

// BackupService represents a service for managing backup.
type BackupService struct {
	l  *zap.Logger
	db *DB
}

// NewBackupService returns a new instance of BackupService.
func NewBackupService(logger *zap.Logger, db *DB) *BackupService {
	return &BackupService{l: logger, db: db}
}

// CreateBackup creates a new backup.
func (s *BackupService) CreateBackup(ctx context.Context, create *api.BackupCreate) (*api.Backup, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.Rollback()

	backup, err := s.createBackup(ctx, tx, create)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, FormatError(err)
	}

	return backup, nil
}

// FindBackupList retrieves a list of backups based on find.
func (s *BackupService) FindBackupList(ctx context.Context, find *api.BackupFind) ([]*api.Backup, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.Rollback()

	list, err := s.findBackupList(ctx, tx, find)
	if err != nil {
		return []*api.Backup{}, err
	}

	return list, nil
}

// PatchBackup updates an existing backup by ID.
// Returns ENOTFOUND if backup does not exist.
func (s *BackupService) PatchBackup(ctx context.Context, patch *api.BackupPatch) (*api.Backup, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.Rollback()

	backup, err := s.patchBackup(ctx, tx, patch)
	if err != nil {
		return nil, FormatError(err)
	}

	if err := tx.Commit(); err != nil {
		return nil, FormatError(err)
	}

	return backup, nil
}

// createBackup creates a new backup.
func (s *BackupService) createBackup(ctx context.Context, tx *Tx, create *api.BackupCreate) (*api.Backup, error) {
	// Insert row into backup.
	row, err := tx.QueryContext(ctx, `
		INSERT INTO backup (
			creator_id,
			updater_id,
			database_id,
			name,
			`+"`status`,"+`
			`+"`type`,"+`
			storage_backend,
			path,
			comment
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		RETURNING id, creator_id, created_ts, updater_id, updated_ts, database_id, name, `+"`status`,"+` `+"`type`, storage_backend, path, comment"+`
	`,
		create.CreatorId,
		create.CreatorId,
		create.DatabaseId,
		create.Name,
		create.Status,
		create.Type,
		create.StorageBackend,
		create.Path,
		create.Comment,
	)

	if err != nil {
		return nil, FormatError(err)
	}
	defer row.Close()

	row.Next()
	var backup api.Backup
	if err := row.Scan(
		&backup.ID,
		&backup.CreatorId,
		&backup.CreatedTs,
		&backup.UpdaterId,
		&backup.UpdatedTs,
		&backup.DatabaseId,
		&backup.Name,
		&backup.Status,
		&backup.Type,
		&backup.StorageBackend,
		&backup.Path,
		&backup.Comment,
	); err != nil {
		return nil, FormatError(err)
	}

	return &backup, nil
}

func (s *BackupService) findBackupList(ctx context.Context, tx *Tx, find *api.BackupFind) (_ []*api.Backup, err error) {
	// Build WHERE clause.
	where, args := []string{"1 = 1"}, []interface{}{}
	if v := find.ID; v != nil {
		where, args = append(where, "id = ?"), append(args, *v)
	}
	if v := find.DatabaseId; v != nil {
		where, args = append(where, "database_id = ?"), append(args, *v)
	}
	if v := find.Name; v != nil {
		where, args = append(where, "name = ?"), append(args, *v)
	}

	rows, err := tx.QueryContext(ctx, `
		SELECT
		    id,
		    creator_id,
		    created_ts,
		    updater_id,
		    updated_ts,
			database_id,
		    name,
			`+"`status`,"+`
			`+"`type`,"+`
			storage_backend,
			path,
			comment
		FROM backup
		WHERE `+strings.Join(where, " AND "),
		args...,
	)
	if err != nil {
		return nil, FormatError(err)
	}
	defer rows.Close()

	// Iterate over result set and deserialize rows into list.
	list := make([]*api.Backup, 0)
	for rows.Next() {
		var backup api.Backup
		if err := rows.Scan(
			&backup.ID,
			&backup.CreatorId,
			&backup.CreatedTs,
			&backup.UpdaterId,
			&backup.UpdatedTs,
			&backup.DatabaseId,
			&backup.Name,
			&backup.Status,
			&backup.Type,
			&backup.StorageBackend,
			&backup.Path,
			&backup.Comment,
		); err != nil {
			return nil, FormatError(err)
		}

		list = append(list, &backup)
	}
	if err := rows.Err(); err != nil {
		return nil, FormatError(err)
	}

	return list, nil
}

// patchBackup updates a backup by ID. Returns the new state of the backup after update.
func (s *BackupService) patchBackup(ctx context.Context, tx *Tx, patch *api.BackupPatch) (*api.Backup, error) {
	// Build UPDATE clause.
	set, args := []string{"updater_id = ?"}, []interface{}{patch.UpdaterId}
	set, args = append(set, "status = ?"), append(args, patch.Status)

	args = append(args, patch.ID)

	// Execute update query with RETURNING.
	row, err := tx.QueryContext(ctx, `
		UPDATE backup
		SET `+strings.Join(set, ", ")+`
		WHERE id = ?
		RETURNING id, creator_id, created_ts, updater_id, updated_ts, database_id, name, `+"`status`,"+` `+"`type`, storage_backend, path, comment"+`
	`,
		args...,
	)
	if err != nil {
		return nil, FormatError(err)
	}
	defer row.Close()

	if row.Next() {
		var backup api.Backup
		if err := row.Scan(
			&backup.ID,
			&backup.CreatorId,
			&backup.CreatedTs,
			&backup.UpdaterId,
			&backup.UpdatedTs,
			&backup.DatabaseId,
			&backup.Name,
			&backup.Status,
			&backup.Type,
			&backup.StorageBackend,
			&backup.Path,
			&backup.Comment,
		); err != nil {
			return nil, FormatError(err)
		}
		return &backup, nil
	}

	return nil, &bytebase.Error{Code: bytebase.ENOTFOUND, Message: fmt.Sprintf("backup ID not found: %d", patch.ID)}
}
