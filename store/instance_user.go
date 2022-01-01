package store

import (
	"context"
	"strings"

	"github.com/bytebase/bytebase/api"
	"go.uber.org/zap"
)

var (
	_ api.InstanceUserService = (*InstanceUserService)(nil)
)

// InstanceUserService represents a service for managing instanceUser.
type InstanceUserService struct {
	l  *zap.Logger
	db *DB
}

// NewInstanceUserService returns a new instance of InstanceUserService.
func NewInstanceUserService(logger *zap.Logger, db *DB) *InstanceUserService {
	return &InstanceUserService{l: logger, db: db}
}

// UpsertInstanceUser would update the existing user if name matches.
func (s *InstanceUserService) UpsertInstanceUser(ctx context.Context, upsert *api.InstanceUserUpsert) (*api.InstanceUser, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.Rollback()

	instanceUser, err := upsertInstanceUser(ctx, tx, upsert)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, FormatError(err)
	}

	return instanceUser, nil
}

// FindInstanceUserList retrieves a list of instanceUsers based on find.
func (s *InstanceUserService) FindInstanceUserList(ctx context.Context, find *api.InstanceUserFind) ([]*api.InstanceUser, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.Rollback()

	list, err := findInstanceUserList(ctx, tx, find)
	if err != nil {
		return []*api.InstanceUser{}, err
	}

	return list, nil
}

// DeleteInstanceUser deletes an existing instance user by ID.
func (s *InstanceUserService) DeleteInstanceUser(ctx context.Context, delete *api.InstanceUserDelete) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return FormatError(err)
	}
	defer tx.Rollback()

	err = deleteInstanceUser(ctx, tx, delete)
	if err != nil {
		return FormatError(err)
	}

	if err := tx.Commit(); err != nil {
		return FormatError(err)
	}

	return nil
}

// upsertInstanceUser upserts a new instanceUser.
func upsertInstanceUser(ctx context.Context, tx *Tx, upsert *api.InstanceUserUpsert) (*api.InstanceUser, error) {
	// Upsert row into database.
	row, err := tx.QueryContext(ctx, `
		INSERT INTO instance_user (
			creator_id,
			updater_id,
			instance_id,
			name,
			grant
		)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT (instance_id, name) DO UPDATE SET
			updater_id = excluded.updater_id,
			grant = excluded.grant
		RETURNING id, instance_id, name, grant
	`,
		upsert.CreatorID,
		upsert.CreatorID,
		upsert.InstanceID,
		upsert.Name,
		upsert.Grant,
	)

	if err != nil {
		return nil, FormatError(err)
	}
	defer row.Close()

	row.Next()
	var instanceUser api.InstanceUser
	if err := row.Scan(
		&instanceUser.ID,
		&instanceUser.InstanceID,
		&instanceUser.Name,
		&instanceUser.Grant,
	); err != nil {
		return nil, FormatError(err)
	}

	return nil, err
}

func findInstanceUserList(ctx context.Context, tx *Tx, find *api.InstanceUserFind) (_ []*api.InstanceUser, err error) {
	// Build WHERE clause.
	where, args := []string{"1 = 1"}, []interface{}{}
	where, args = append(where, "instance_id = ?"), append(args, find.InstanceID)

	rows, err := tx.QueryContext(ctx, `
		SELECT
			id,
			instance_id,
			name,
			grant
		FROM instance_user
		WHERE `+strings.Join(where, " AND ")+`
		ORDER BY name ASC
		`,
		args...,
	)
	if err != nil {
		return nil, FormatError(err)
	}
	defer rows.Close()

	// Iterate over result set and deserialize rows into list.
	list := make([]*api.InstanceUser, 0)
	for rows.Next() {
		var instanceUser api.InstanceUser
		if err := rows.Scan(
			&instanceUser.ID,
			&instanceUser.InstanceID,
			&instanceUser.Name,
			&instanceUser.Grant,
		); err != nil {
			return nil, FormatError(err)
		}

		list = append(list, &instanceUser)
	}
	if err := rows.Err(); err != nil {
		return nil, FormatError(err)
	}

	return list, nil
}

// deleteInstanceUser permanently deletes a instance user by ID.
func deleteInstanceUser(ctx context.Context, tx *Tx, delete *api.InstanceUserDelete) error {
	// Remove row from database.
	if _, err := tx.ExecContext(ctx, `DELETE FROM instance_user WHERE id = ?`, delete.ID); err != nil {
		return FormatError(err)
	}
	return nil
}
