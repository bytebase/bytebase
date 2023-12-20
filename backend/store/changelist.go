package store

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/pkg/errors"
	"google.golang.org/protobuf/encoding/protojson"

	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

// ChangelistMessage is the message for a changelist.
type ChangelistMessage struct {
	ProjectID  string
	ResourceID string

	Payload *storepb.Changelist

	// Output only fields
	UID         int
	CreatorID   int
	UpdaterID   int
	CreatedTime time.Time
	UpdatedTime time.Time
}

// FindChangelistMessage is the API message for finding changelists.
type FindChangelistMessage struct {
	ProjectID  *string
	ProjectIDs *[]string
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
		where, args = append(where, fmt.Sprintf("project.resource_id = $%d", len(args)+1)), append(args, *v)
	}
	if v := find.ProjectIDs; v != nil {
		where, args = append(where, fmt.Sprintf("project.resource_id = ANY($%d)", len(args)+1)), append(args, *v)
	}
	if v := find.ResourceID; v != nil {
		where, args = append(where, fmt.Sprintf("changelist.name = $%d", len(args)+1)), append(args, *v)
	}

	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	rows, err := tx.QueryContext(ctx, fmt.Sprintf(`
		SELECT
			changelist.id,
			changelist.creator_id,
			changelist.created_ts,
			changelist.updater_id,
			changelist.updated_ts,
			project.resource_id AS project_id,
			changelist.name,
			changelist.payload
		FROM changelist
		LEFT JOIN project ON changelist.project_id = project.id
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
		var createdTs, updatedTs int64
		var payload []byte
		if err := rows.Scan(
			&changelist.UID,
			&changelist.CreatorID,
			&createdTs,
			&changelist.UpdaterID,
			&updatedTs,
			&changelist.ProjectID,
			&changelist.ResourceID,
			&payload,
		); err != nil {
			return nil, err
		}
		changelistPayload := &storepb.Changelist{}
		if err := protojsonUnmarshaler.Unmarshal(payload, changelistPayload); err != nil {
			return nil, err
		}
		changelist.Payload = changelistPayload
		changelist.CreatedTime = time.Unix(createdTs, 0)
		changelist.UpdatedTime = time.Unix(updatedTs, 0)

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
	project, err := s.GetProjectV2(ctx, &FindProjectMessage{ResourceID: &create.ProjectID})
	if err != nil {
		return nil, err
	}
	if create.Payload == nil {
		create.Payload = &storepb.Changelist{}
	}
	create.UpdaterID = create.CreatorID
	payload, err := protojson.Marshal(create.Payload)
	if err != nil {
		return nil, err
	}

	query := `
		INSERT INTO changelist (
			creator_id,
			updater_id,
			project_id,
			name,
			payload
		)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_ts, updated_ts;
	`

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	var createdTs, updatedTs int64
	if err := tx.QueryRowContext(ctx, query,
		create.CreatorID,
		create.CreatorID,
		project.UID,
		create.ResourceID,
		payload,
	).Scan(
		&create.UID,
		&createdTs,
		&updatedTs,
	); err != nil {
		return nil, err
	}
	create.CreatedTime = time.Unix(createdTs, 0)
	create.UpdatedTime = time.Unix(updatedTs, 0)
	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return create, nil
}

// UpdateChangelist updates a changelist.
func (s *Store) UpdateChangelist(ctx context.Context, update *UpdateChangelistMessage) error {
	project, err := s.GetProjectV2(ctx, &FindProjectMessage{ResourceID: &update.ProjectID})
	if err != nil {
		return err
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return errors.Wrapf(err, "failed to begin transaction")
	}

	set, args := []string{"updater_id = $1"}, []any{update.UpdaterID}
	if v := update.Payload; v != nil {
		payload, err := protojson.Marshal(update.Payload)
		if err != nil {
			return err
		}
		set, args = append(set, fmt.Sprintf("payload = $%d", len(args)+1)), append(args, payload)
	}
	args = append(args, project.UID, update.ResourceID)

	if _, err := tx.ExecContext(ctx, fmt.Sprintf(`
		UPDATE changelist
		SET `+strings.Join(set, ", ")+`
		WHERE changelist.project_id = $%d AND changelist.name = $%d`, len(set)+1, len(set)+2), args...); err != nil {
		return err
	}

	return tx.Commit()
}

// DeleteChangelist deletes a changelist.
func (s *Store) DeleteChangelist(ctx context.Context, projectID, resourceID string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, `
		DELETE FROM changelist
		USING project
		WHERE changelist.project_id = project.id AND project.resource_id = $1 AND changelist.name = $2;`,
		projectID, resourceID); err != nil {
		return err
	}

	return tx.Commit()
}
