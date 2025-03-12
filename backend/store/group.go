package store

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"google.golang.org/protobuf/encoding/protojson"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

// FindGroupMessage is the message for finding groups.
type FindGroupMessage struct {
	Email *string
}

// UpdateGroupMessage is the message to update a group.
type UpdateGroupMessage struct {
	Title       *string
	Description *string
	Payload     *storepb.GroupPayload
}

// GroupMessage is the message for a group.
type GroupMessage struct {
	Email       string
	Title       string
	Description string
	Payload     *storepb.GroupPayload
}

// GetGroup gets a group.
func (s *Store) GetGroup(ctx context.Context, email string) (*GroupMessage, error) {
	if v, ok := s.groupCache.Get(email); ok {
		return v, nil
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	groups, err := s.listGroupImpl(ctx, tx, &FindGroupMessage{
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

// ListGroups list all groups.
func (s *Store) ListGroups(ctx context.Context, find *FindGroupMessage) ([]*GroupMessage, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	groups, err := s.listGroupImpl(ctx, tx, find)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}

	for _, group := range groups {
		s.groupCache.Add(group.Email, group)
	}
	return groups, nil
}

func (*Store) listGroupImpl(ctx context.Context, txn *sql.Tx, find *FindGroupMessage) ([]*GroupMessage, error) {
	where, args := []string{"TRUE"}, []any{}
	if v := find.Email; v != nil {
		where, args = append(where, fmt.Sprintf("email = $%d", len(args)+1)), append(args, *v)
	}

	var groups []*GroupMessage
	rows, err := txn.QueryContext(ctx, fmt.Sprintf(`
	SELECT
		email,
		name,
		description,
		payload
	FROM user_group
	WHERE %s
	ORDER BY email
	`, strings.Join(where, " AND ")), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var group GroupMessage
		var payload []byte
		if err := rows.Scan(
			&group.Email,
			&group.Title,
			&group.Description,
			&payload,
		); err != nil {
			return nil, err
		}
		groupPayload := storepb.GroupPayload{}
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

// CreateGroup creates a group.
func (s *Store) CreateGroup(ctx context.Context, create *GroupMessage) (*GroupMessage, error) {
	if create.Payload == nil {
		create.Payload = &storepb.GroupPayload{}
	}

	query := `
		INSERT INTO user_group (
			email,
			name,
			description,
			payload
		) VALUES ($1, $2, $3, $4)
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

	if _, err := tx.ExecContext(
		ctx,
		query,
		create.Email,
		create.Title,
		create.Description,
		payloadBytes,
	); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, errors.Wrap(err, "failed to commit")
	}

	s.groupCache.Add(create.Email, create)
	return create, nil
}

// UpdateGroup updates a group.
func (s *Store) UpdateGroup(ctx context.Context, email string, patch *UpdateGroupMessage) (*GroupMessage, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to begin transaction")
	}

	set, args := []string{}, []any{}

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

	var group GroupMessage
	var payload []byte

	if err := tx.QueryRowContext(ctx, fmt.Sprintf(`
		UPDATE user_group
		SET %s
		WHERE email = $%d
		RETURNING
			email,
			name,
			description,
			payload
		`, strings.Join(set, ", "), len(set)+1), args...).Scan(
		&group.Email,
		&group.Title,
		&group.Description,
		&payload,
	); err != nil {
		return nil, err
	}

	groupPayload := storepb.GroupPayload{}
	if err := common.ProtojsonUnmarshaler.Unmarshal(payload, &groupPayload); err != nil {
		return nil, err
	}
	group.Payload = &groupPayload

	if err := tx.Commit(); err != nil {
		return nil, errors.Wrap(err, "failed to commit transaction")
	}

	s.groupCache.Add(group.Email, &group)
	return &group, nil
}

// DeleteGroup deletes a group.
func (s *Store) DeleteGroup(ctx context.Context, email string) error {
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

	s.groupCache.Remove(email)
	return nil
}
