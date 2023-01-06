package store

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/common"
)

// UpsertInstanceUser would update the existing user if name matches.
func (s *Store) UpsertInstanceUser(ctx context.Context, upsert *api.InstanceUserUpsert) (*api.InstanceUser, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.Rollback()

	instanceUser, err := upsertInstanceUserImpl(ctx, tx, upsert)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, FormatError(err)
	}

	return instanceUser, nil
}

// GetInstanceUser gets an instance of IntanceUser.
func (s *Store) GetInstanceUser(ctx context.Context, find *api.InstanceUserFind) (*api.InstanceUser, error) {
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.Rollback()

	list, err := findInstanceUserImpl(ctx, tx, find)
	if err != nil {
		return nil, err
	}

	if len(list) == 0 {
		return nil, nil
	} else if len(list) > 1 {
		return nil, &common.Error{Code: common.Conflict, Err: errors.Errorf("found %d instance users with filter %+v, expect 1", len(list), find)}
	}
	return list[0], nil
}

// FindInstanceUserByInstanceID retrieves a list of instanceUsers based on find.
func (s *Store) FindInstanceUserByInstanceID(ctx context.Context, id int) ([]*api.InstanceUser, error) {
	find := &api.InstanceUserFind{
		InstanceID: &id,
	}

	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.Rollback()

	list, err := findInstanceUserImpl(ctx, tx, find)
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
	defer tx.Rollback()

	if err := deleteInstanceUser(ctx, tx, delete); err != nil {
		return FormatError(err)
	}

	if err := tx.Commit(); err != nil {
		return FormatError(err)
	}

	return nil
}

// upsertInstanceUserImpl upserts a new instanceUser.
func upsertInstanceUserImpl(ctx context.Context, tx *Tx, upsert *api.InstanceUserUpsert) (*api.InstanceUser, error) {
	// Upsert row into database.
	query := `
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
	`
	var instanceUser api.InstanceUser
	if err := tx.QueryRowContext(ctx, query,
		upsert.CreatorID,
		upsert.CreatorID,
		upsert.InstanceID,
		upsert.Name,
		upsert.Grant,
	).Scan(
		&instanceUser.ID,
		&instanceUser.InstanceID,
		&instanceUser.Name,
		&instanceUser.Grant,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, common.FormatDBErrorEmptyRowWithQuery(query)
		}
		return nil, FormatError(err)
	}
	return &instanceUser, nil
}

func findInstanceUserImpl(ctx context.Context, tx *Tx, find *api.InstanceUserFind) ([]*api.InstanceUser, error) {
	// Build WHERE clause.
	where, args := []string{"1 = 1"}, []interface{}{}

	if v := find.ID; v != nil {
		where, args = append(where, fmt.Sprintf("id = $%d", len(args)+1)), append(args, *v)
	}
	if v := find.InstanceID; v != nil {
		where, args = append(where, fmt.Sprintf("instance_id = $%d", len(args)+1)), append(args, *v)
	}

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
func deleteInstanceUser(ctx context.Context, tx *Tx, delete *api.InstanceUserDelete) error {
	// Remove row from database.
	if _, err := tx.ExecContext(ctx, `DELETE FROM instance_user WHERE id = $1`, delete.ID); err != nil {
		return FormatError(err)
	}
	return nil
}
