package store

import (
	"context"
	"fmt"
	"strings"
	"time"

	"google.golang.org/protobuf/encoding/protojson"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

// FindUserGroupMessage is the message for finding groups.
type FindUserGroupMessage struct {
	Email *string
}

// UpdateUserGroupMessage is the message to update a group.
type UpdateUserGroupMessage struct {
	Title       *string
	Description *string
	Payload     *storepb.UserGroupPayload
}

// UserGroupMessage is the message for a group.
type UserGroupMessage struct {
	Email       string
	Title       string
	Description string
	CreatorUID  int
	Payload     *storepb.UserGroupPayload
	CreatedTime time.Time
}

// GetUserGroup gets a group.
func (s *Store) GetUserGroup(ctx context.Context, email string) (*UserGroupMessage, error) {
	if v, ok := s.userGroupCache.Get(email); ok {
		return v, nil
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	groups, err := s.listUserGroupImpl(ctx, tx, &FindUserGroupMessage{
		Email: &email,
	})
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}

	if len(groups) == 0 {
		return nil, nil
	} else if len(groups) > 1 {
		return nil, &common.Error{Code: common.Conflict, Err: errors.Errorf("found %d groups with email %+v, expect 1", len(groups), email)}
	}
	return groups[0], nil
}

// ListUserGroups list all groups.
func (s *Store) ListUserGroups(ctx context.Context, find *FindUserGroupMessage) ([]*UserGroupMessage, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	groups, err := s.listUserGroupImpl(ctx, tx, find)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}

	for _, group := range groups {
		s.userGroupCache.Add(group.Email, group)
	}
	return groups, nil
}

func (*Store) listUserGroupImpl(ctx context.Context, tx *Tx, find *FindUserGroupMessage) ([]*UserGroupMessage, error) {
	where, args := []string{"TRUE"}, []any{}
	if v := find.Email; v != nil {
		where, args = append(where, fmt.Sprintf("email = $%d", len(args)+1)), append(args, *v)
	}

	var groups []*UserGroupMessage
	rows, err := tx.QueryContext(ctx, fmt.Sprintf(`
	SELECT
		email,
		created_ts,
		creator_id,
		name,
		description,
		payload
	FROM user_group
	WHERE %s
	ORDER BY created_ts DESC
	`, strings.Join(where, " AND ")), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var group UserGroupMessage
		var payload []byte
		var createdTs int64
		if err := rows.Scan(
			&group.Email,
			&createdTs,
			&group.CreatorUID,
			&group.Title,
			&group.Description,
			&payload,
		); err != nil {
			return nil, err
		}
		group.CreatedTime = time.Unix(createdTs, 0)
		groupPayload := storepb.UserGroupPayload{}
		if err := common.ProtojsonUnmarshaler.Unmarshal(payload, &groupPayload); err != nil {
			return nil, err
		}
		group.Payload = &groupPayload
		groups = append(groups, &group)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return groups, nil
}

// CreateUserGroup creates a group.
func (s *Store) CreateUserGroup(ctx context.Context, create *UserGroupMessage, creatorUID int) (*UserGroupMessage, error) {
	if create.Payload == nil {
		create.Payload = &storepb.UserGroupPayload{}
	}
	create.CreatorUID = creatorUID

	query := `
		INSERT INTO user_group (
			email,
			creator_id,
			updater_id,
			name,
			description,
			payload
		) VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING created_ts
	`
	payloadBytes, err := protojson.Marshal(create.Payload)
	if err != nil {
		return nil, err
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to begin tx")
	}
	defer tx.Rollback()

	var createdTs int64
	if err := tx.QueryRowContext(
		ctx,
		query,
		create.Email,
		create.CreatorUID,
		create.CreatorUID,
		create.Title,
		create.Description,
		payloadBytes,
	).Scan(&createdTs); err != nil {
		return nil, err
	}
	create.CreatedTime = time.Unix(createdTs, 0)

	if err := tx.Commit(); err != nil {
		return nil, errors.Wrap(err, "failed to commit")
	}

	s.userGroupCache.Add(create.Email, create)
	return create, nil
}

// UpdateUserGroup updates a group.
func (s *Store) UpdateUserGroup(ctx context.Context, email string, patch *UpdateUserGroupMessage, updaterID int) (*UserGroupMessage, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to begin transaction")
	}

	set, args := []string{"updater_id = $1", "updated_ts = $2"}, []any{updaterID, time.Now().Unix()}

	if v := patch.Title; v != nil {
		set, args = append(set, fmt.Sprintf("name = $%d", len(args)+1)), append(args, *v)
	}
	if v := patch.Description; v != nil {
		set, args = append(set, fmt.Sprintf("description = $%d", len(args)+1)), append(args, *v)
	}
	if v := patch.Payload; v != nil {
		payload, err := protojson.Marshal(v)
		if err != nil {
			return nil, err
		}
		set, args = append(set, fmt.Sprintf("payload = $%d", len(args)+1)), append(args, payload)
	}
	args = append(args, email)

	var group UserGroupMessage
	var payload []byte
	var createdTs int64

	if err := tx.QueryRowContext(ctx, fmt.Sprintf(`
		UPDATE user_group
		SET %s
		WHERE email = $%d
		RETURNING
			email,
			created_ts,
			creator_id,
			name,
			description,
			payload
		`, strings.Join(set, ", "), len(set)+1), args...).Scan(
		&group.Email,
		&createdTs,
		&group.CreatorUID,
		&group.Title,
		&group.Description,
		&payload,
	); err != nil {
		return nil, err
	}

	group.CreatedTime = time.Unix(createdTs, 0)
	groupPayload := storepb.UserGroupPayload{}
	if err := common.ProtojsonUnmarshaler.Unmarshal(payload, &groupPayload); err != nil {
		return nil, err
	}
	group.Payload = &groupPayload

	if err := tx.Commit(); err != nil {
		return nil, errors.Wrap(err, "failed to commit transaction")
	}

	s.userGroupCache.Add(group.Email, &group)
	return &group, nil
}

// DeleteUserGroup deletes a group.
func (s *Store) DeleteUserGroup(ctx context.Context, email string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return errors.Wrap(err, "failed to begin transaction")
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, `DELETE FROM user_group WHERE email = $1`, email); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return errors.Wrap(err, "failed to commit transaction")
	}

	s.userGroupCache.Remove(email)
	return nil
}
