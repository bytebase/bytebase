package store

import (
	"context"
	"database/sql"
	"strings"

	"github.com/bytebase/bytebase/api"
)

// UpsertInstanceUser would update the existing user if name matches.
func (s *Store) UpsertInstanceUser(ctx context.Context, upsert *api.InstanceUserUpsert) (*api.InstanceUser, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.PTx.Rollback()

	instanceUser, err := upsertInstanceUserImpl(ctx, tx.PTx, upsert)
	if err != nil {
		return nil, err
	}

	if err := tx.PTx.Commit(); err != nil {
		return nil, FormatError(err)
	}

	return instanceUser, nil
}

// FindInstanceUserByInstanceID retrieves a list of instanceUsers based on find.
func (s *Store) FindInstanceUserByInstanceID(ctx context.Context, id int) ([]*api.InstanceUser, error) {
	find := &api.InstanceUserFind{
		InstanceID: id,
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.PTx.Rollback()

	list, err := findInstanceUserImpl(ctx, tx.PTx, find)
	if err != nil {
		return nil, err
	}

	return list, nil
}

// DeleteInstanceUser deletes an existing instance user by ID.
func (s *Store) DeleteInstanceUser(ctx context.Context, delete *api.InstanceUserDelete) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return FormatError(err)
	}
	defer tx.PTx.Rollback()

	if err := deleteInstanceUser(ctx, tx.PTx, delete); err != nil {
		return FormatError(err)
	}

	if err := tx.PTx.Commit(); err != nil {
		return FormatError(err)
	}

	return nil
}

// upsertInstanceUserImpl upserts a new instanceUser.
func upsertInstanceUserImpl(ctx context.Context, tx *sql.Tx, upsert *api.InstanceUserUpsert) (*api.InstanceUser, error) {
	// Upsert row into database.
	row, err := tx.QueryContext(ctx, `
		INSERT INTO instance_user (
			creator_id,
			updater_id,
			instance_id,
			name,
			"grant"
		)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (instance_id, name) DO UPDATE SET
			updater_id = excluded.updater_id,
			"grant" = excluded.grant
		RETURNING id, instance_id, name, "grant"
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

func findInstanceUserImpl(ctx context.Context, tx *sql.Tx, find *api.InstanceUserFind) ([]*api.InstanceUser, error) {
	// Build WHERE clause.
	where, args := []string{"1 = 1"}, []interface{}{}
	where, args = append(where, "instance_id = $1"), append(args, find.InstanceID)

	rows, err := tx.QueryContext(ctx, `
		SELECT
			id,
			instance_id,
			name,
			"grant"
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

	// Iterate over result set and deserialize rows into instanceUserList.
	var instanceUserList []*api.InstanceUser
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

		instanceUserList = append(instanceUserList, &instanceUser)
	}
	if err := rows.Err(); err != nil {
		return nil, FormatError(err)
	}

	return instanceUserList, nil
}

// deleteInstanceUser permanently deletes a instance user by ID.
func deleteInstanceUser(ctx context.Context, tx *sql.Tx, delete *api.InstanceUserDelete) error {
	// Remove row from database.
	if _, err := tx.ExecContext(ctx, `DELETE FROM instance_user WHERE id = $1`, delete.ID); err != nil {
		return FormatError(err)
	}
	return nil
}
