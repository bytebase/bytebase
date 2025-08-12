package store

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/pkg/errors"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/bytebase/bytebase/backend/common"
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
	where, args := []string{"TRUE"}, []any{}

	if v := find.ProjectID; v != nil {
		where, args = append(where, fmt.Sprintf("changelist.project = $%d", len(args)+1)), append(args, *v)
	}
	if v := find.ResourceID; v != nil {
		where, args = append(where, fmt.Sprintf("changelist.name = $%d", len(args)+1)), append(args, *v)
	}

	tx, err := s.GetDB().BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	rows, err := tx.QueryContext(ctx, fmt.Sprintf(`
		SELECT
			creator_id,
			updated_at,
			project,
			name,
			payload
		FROM changelist
		WHERE %s`, strings.Join(where, " AND ")),
		args...,
	)
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

	query := `
		INSERT INTO changelist (
			creator_id,
			project,
			name,
			payload
		)
		VALUES ($1, $2, $3, $4)
		RETURNING updated_at;
	`

	tx, err := s.GetDB().BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	if err := tx.QueryRowContext(ctx, query,
		create.CreatorID,
		create.ProjectID,
		create.ResourceID,
		payload,
	).Scan(
		&create.UpdatedAt,
	); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return create, nil
}

// UpdateChangelist updates a changelist.
func (s *Store) UpdateChangelist(ctx context.Context, update *UpdateChangelistMessage) error {
	tx, err := s.GetDB().BeginTx(ctx, nil)
	if err != nil {
		return errors.Wrapf(err, "failed to begin transaction")
	}

	set, args := []string{"updated_at = $1"}, []any{time.Now()}
	if v := update.Payload; v != nil {
		payload, err := protojson.Marshal(update.Payload)
		if err != nil {
			return err
		}
		set, args = append(set, fmt.Sprintf("payload = $%d", len(args)+1)), append(args, payload)
	}
	args = append(args, update.ProjectID, update.ResourceID)

	if _, err := tx.ExecContext(ctx, fmt.Sprintf(`
		UPDATE changelist
		SET `+strings.Join(set, ", ")+`
		WHERE project = $%d AND name = $%d`, len(set)+1, len(set)+2), args...); err != nil {
		return err
	}

	return tx.Commit()
}

// DeleteChangelist deletes a changelist.
func (s *Store) DeleteChangelist(ctx context.Context, projectID, resourceID string) error {
	tx, err := s.GetDB().BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, `
		DELETE FROM changelist
		WHERE changelist.project = $1 AND changelist.name = $2;`,
		projectID, resourceID); err != nil {
		return err
	}

	return tx.Commit()
}
