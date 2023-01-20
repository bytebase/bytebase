package store

import (
	"context"
	"fmt"
	"strings"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/api"
	"github.com/bytebase/bytebase/backend/common"
)

// InstanceUserMessage is the mssage for instance user.
type InstanceUserMessage struct {
	Name  string
	Grant string
}

// FindInstanceUserMessage is the message for finding instance users.
type FindInstanceUserMessage struct {
	InstanceUID int
	Name        *string
}

// GetInstanceUser gets an instance users.
func (s *Store) GetInstanceUser(ctx context.Context, find *FindInstanceUserMessage) (*InstanceUserMessage, error) {
	instanceUsers, err := s.ListInstanceUsers(ctx, find)
	if err != nil {
		return nil, err
	}
	if len(instanceUsers) == 0 {
		return nil, nil
	} else if len(instanceUsers) > 1 {
		return nil, &common.Error{Code: common.Conflict, Err: errors.Errorf("found %d instance users with filter %+v, expect 1", len(instanceUsers), find)}
	}
	return instanceUsers[0], nil
}

// ListInstanceUsers lists all the instance users.
func (s *Store) ListInstanceUsers(ctx context.Context, find *FindInstanceUserMessage) ([]*InstanceUserMessage, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.Rollback()

	instanceUsers, err := listInstanceUsersImpl(ctx, tx, find)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, FormatError(err)
	}

	return instanceUsers, nil
}

// UpsertInstanceUsers will reconcile instance users for the instance.
func (s *Store) UpsertInstanceUsers(ctx context.Context, instanceUID int, instanceUsers []*InstanceUserMessage) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	oldInstanceUsers, err := listInstanceUsersImpl(ctx, tx, &FindInstanceUserMessage{InstanceUID: instanceUID})
	if err != nil {
		return err
	}
	deletes, upserts := getInstanceUsersDiff(oldInstanceUsers, instanceUsers)
	if len(deletes) == 0 && len(upserts) == 0 {
		return nil
	}

	// Delete instance users that no longer exist.
	if len(deletes) > 0 {
		deleteArgs := []interface{}{instanceUID}
		var deletePlaceholders []string
		for i, d := range deletes {
			deleteArgs = append(deleteArgs, d)
			deletePlaceholders = append(deletePlaceholders, fmt.Sprintf("$%d", i+2))
		}
		deleteQuery := fmt.Sprintf(`
			DELETE FROM instance_user WHERE instance_id = $1 AND name IN (%s)
		`, strings.Join(deletePlaceholders, ", "))
		if _, err := tx.ExecContext(ctx, deleteQuery, deleteArgs...); err != nil {
			return err
		}
	}
	// Upsert instance users.
	if len(upserts) > 0 {
		args := []interface{}{}
		var placeholders []string
		for i, instanceUser := range upserts {
			args = append(args, api.SystemBotID, api.SystemBotID, instanceUID, instanceUser.Name, instanceUser.Grant)
			placeholders = append(placeholders, fmt.Sprintf("($%d, $%d, $%d, $%d, $%d)", 5*i+1, 5*i+2, 5*i+3, 5*i+4, 5*i+5))
		}
		query := fmt.Sprintf(`
			INSERT INTO instance_user (
				creator_id,
				updater_id,
				instance_id,
				name,
				"grant"
			)
			VALUES %s
			ON CONFLICT (instance_id, name) DO UPDATE SET
				updater_id = excluded.updater_id,
				"grant" = excluded.grant;
		`, strings.Join(placeholders, ", "))
		if _, err := tx.ExecContext(ctx, query, args...); err != nil {
			return err
		}
	}

	if err := tx.Commit(); err != nil {
		return FormatError(err)
	}

	return nil
}

func getInstanceUsersDiff(oldInstanceUsers, instanceUsers []*InstanceUserMessage) ([]string, []*InstanceUserMessage) {
	instanceUserNamesMap := make(map[string]bool)
	oldInstanceUserMap := make(map[string]*InstanceUserMessage)
	var deletes []string
	var upserts []*InstanceUserMessage
	for _, instanceUser := range instanceUsers {
		instanceUserNamesMap[instanceUser.Name] = true
	}
	for _, oldInstanceUser := range oldInstanceUsers {
		oldInstanceUserMap[oldInstanceUser.Name] = oldInstanceUser
	}

	for _, oldInstanceUser := range oldInstanceUsers {
		if _, ok := instanceUserNamesMap[oldInstanceUser.Name]; !ok {
			deletes = append(deletes, oldInstanceUser.Name)
		}
	}
	for _, instanceUser := range instanceUsers {
		if old, ok := oldInstanceUserMap[instanceUser.Name]; !ok || old.Grant != instanceUser.Grant {
			upserts = append(upserts, instanceUser)
		}
	}
	return deletes, upserts
}

func listInstanceUsersImpl(ctx context.Context, tx *Tx, find *FindInstanceUserMessage) ([]*InstanceUserMessage, error) {
	where, args := []string{"TRUE"}, []interface{}{}

	where, args = append(where, fmt.Sprintf("instance_id = $%d", len(args)+1)), append(args, find.InstanceUID)
	if v := find.Name; v != nil {
		where, args = append(where, fmt.Sprintf("name = $%d", len(args)+1)), append(args, *v)
	}

	var instanceUsers []*InstanceUserMessage
	rows, err := tx.QueryContext(ctx, `
		SELECT
			name, "grant"
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
	for rows.Next() {
		var instanceUser InstanceUserMessage
		if err := rows.Scan(
			&instanceUser.Name,
			&instanceUser.Grant,
		); err != nil {
			return nil, FormatError(err)
		}
		instanceUsers = append(instanceUsers, &instanceUser)
	}
	if err := rows.Err(); err != nil {
		return nil, FormatError(err)
	}
	return instanceUsers, nil
}
