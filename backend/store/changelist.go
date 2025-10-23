package store

import (
	"context"
	"database/sql"
	"time"

	"github.com/pkg/errors"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/qb"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

// ChangelistMessage is the message for a changelist.
type ChangelistMessage struct {
	ProjectID  string
	ResourceID string

	Payload *storepb.Changelist

	CreatorID int
	UpdatedAt time.Time
}

// FindChangelistMessage is the API message for finding changelists.
type FindChangelistMessage struct {
	ProjectID  *string
	ResourceID *string
}

// UpdateChangelistMessage is the message to update a changelist.
type UpdateChangelistMessage struct {
	ProjectID  string
	ResourceID string
	UpdaterID  int
	Payload    *storepb.Changelist
}

// GetChangelist gets a changelist.
func (s *Store) GetChangelist(ctx context.Context, find *FindChangelistMessage) (*ChangelistMessage, error) {
	changelists, err := s.ListChangelists(ctx, find)
	if err != nil {
		return nil, err
	}
	if len(changelists) == 0 {
		return nil, nil
	}
	if len(changelists) > 1 {
		return nil, errors.Errorf("expected 1 changelist, got %d", len(changelists))
	}
	return changelists[0], nil
}

// ListChangelists returns a list of changelists.
func (s *Store) ListChangelists(ctx context.Context, find *FindChangelistMessage) ([]*ChangelistMessage, error) {
	q := qb.Q().Space(`
		SELECT
			creator_id,
			updated_at,
			project,
			name,
			payload
		FROM changelist
		WHERE TRUE
	`)

	if v := find.ProjectID; v != nil {
		q.And("changelist.project = ?", *v)
	}
	if v := find.ResourceID; v != nil {
		q.And("changelist.name = ?", *v)
	}

	query, args, err := q.ToSQL()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to build sql")
	}

	tx, err := s.GetDB().BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	rows, err := tx.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var changelists []*ChangelistMessage
	for rows.Next() {
		var changelist ChangelistMessage
		var payload []byte
		if err := rows.Scan(
			&changelist.CreatorID,
			&changelist.UpdatedAt,
			&changelist.ProjectID,
			&changelist.ResourceID,
			&payload,
		); err != nil {
			return nil, err
		}
		changelistPayload := &storepb.Changelist{}
		if err := common.ProtojsonUnmarshaler.Unmarshal(payload, changelistPayload); err != nil {
			return nil, err
		}
		changelist.Payload = changelistPayload

		changelists = append(changelists, &changelist)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return changelists, nil
}

// CreateChangelist creates a changelist.
func (s *Store) CreateChangelist(ctx context.Context, create *ChangelistMessage) (*ChangelistMessage, error) {
	if create.Payload == nil {
		create.Payload = &storepb.Changelist{}
	}
	payload, err := protojson.Marshal(create.Payload)
	if err != nil {
		return nil, err
	}

	q := qb.Q().Space(`
		INSERT INTO changelist (
			creator_id,
			project,
			name,
			payload
		)
		VALUES (?, ?, ?, ?)
		RETURNING updated_at
	`, create.CreatorID, create.ProjectID, create.ResourceID, payload)

	query, args, err := q.ToSQL()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to build sql")
	}

	tx, err := s.GetDB().BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	if err := tx.QueryRowContext(ctx, query, args...).Scan(&create.UpdatedAt); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return create, nil
}

// UpdateChangelist updates a changelist.
func (s *Store) UpdateChangelist(ctx context.Context, update *UpdateChangelistMessage) error {
	q := qb.Q().Space("UPDATE changelist SET updated_at = ?", time.Now())
	if v := update.Payload; v != nil {
		payload, err := protojson.Marshal(update.Payload)
		if err != nil {
			return err
		}
		q.Comma("payload = ?", payload)
	}
	q.Space("WHERE project = ? AND name = ?", update.ProjectID, update.ResourceID)

	query, args, err := q.ToSQL()
	if err != nil {
		return errors.Wrapf(err, "failed to build sql")
	}

	tx, err := s.GetDB().BeginTx(ctx, nil)
	if err != nil {
		return errors.Wrapf(err, "failed to begin transaction")
	}

	if _, err := tx.ExecContext(ctx, query, args...); err != nil {
		return err
	}

	return tx.Commit()
}

// DeleteChangelist deletes a changelist.
func (s *Store) DeleteChangelist(ctx context.Context, projectID, resourceID string) error {
	q := qb.Q().Space(`
		DELETE FROM changelist
		WHERE changelist.project = ? AND changelist.name = ?
	`, projectID, resourceID)

	query, args, err := q.ToSQL()
	if err != nil {
		return errors.Wrapf(err, "failed to build sql")
	}

	tx, err := s.GetDB().BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, query, args...); err != nil {
		return err
	}

	return tx.Commit()
}
