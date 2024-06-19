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
	api "github.com/bytebase/bytebase/backend/legacyapi"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

// ReviewConfigMessage is the API message for sql review.
type ReviewConfigMessage struct {
	ID         string
	CreatorUID int
	UpdaterUID int
	Enforce    bool
	Name       string
	Payload    *storepb.ReviewConfigPayload

	// Output only fields
	CreatedTime time.Time
	UpdatedTime time.Time
}

// FindReviewConfigMessage is the API message for finding sql review.
type FindReviewConfigMessage struct {
	ID *string
}

// PatchReviewConfigMessage is the message to patch a sql review.
type PatchReviewConfigMessage struct {
	ID        string
	UpdaterID int
	Name      *string
	Enforce   *bool
	Payload   *storepb.ReviewConfigPayload
}

// GetReviewConfig gets sql review by id.
func (s *Store) GetReviewConfig(ctx context.Context, id string) (*ReviewConfigMessage, error) {
	list, err := s.ListReviewConfigs(ctx, &FindReviewConfigMessage{ID: &id})
	if err != nil {
		return nil, err
	}
	if len(list) == 0 {
		return nil, nil
	}
	if len(list) > 1 {
		return nil, &common.Error{Code: common.Conflict, Err: errors.Errorf("found %d sql review", len(list))}
	}

	return list[0], nil
}

// ListReviewConfigs lists all sql review.
func (s *Store) ListReviewConfigs(ctx context.Context, find *FindReviewConfigMessage) ([]*ReviewConfigMessage, error) {
	where, args := []string{"TRUE"}, []any{}

	if v := find.ID; v != nil {
		where, args = append(where, fmt.Sprintf("id = $%d", len(args)+1)), append(args, *v)
	}

	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	rows, err := tx.QueryContext(ctx, fmt.Sprintf(`
		SELECT
			id,
			row_status,
			creator_id,
			created_ts,
			updater_id,
			updated_ts,
			name,
			payload
		FROM review_config
		WHERE %s`, strings.Join(where, " AND ")),
		args...,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sqlReviewList []*ReviewConfigMessage
	for rows.Next() {
		var sqlReview ReviewConfigMessage
		var createdTs int64
		var updatedTs int64
		var payload []byte
		var rowStatus string

		if err := rows.Scan(
			&sqlReview.ID,
			&rowStatus,
			&sqlReview.CreatorUID,
			&createdTs,
			&sqlReview.UpdaterUID,
			&updatedTs,
			&sqlReview.Name,
			&payload,
		); err != nil {
			return nil, err
		}

		reviewConfigPyload := &storepb.ReviewConfigPayload{}
		if err := common.ProtojsonUnmarshaler.Unmarshal(payload, reviewConfigPyload); err != nil {
			return nil, err
		}

		sqlReview.Enforce = rowStatus == string(api.Normal)
		sqlReview.Payload = reviewConfigPyload
		sqlReview.CreatedTime = time.Unix(createdTs, 0)
		sqlReview.UpdatedTime = time.Unix(updatedTs, 0)
		sqlReviewList = append(sqlReviewList, &sqlReview)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return sqlReviewList, nil
}

// CreateReviewConfig creates the sql review config.
func (s *Store) CreateReviewConfig(ctx context.Context, create *ReviewConfigMessage) (*ReviewConfigMessage, error) {
	if create.ID == "" {
		return nil, errors.Errorf("empty config id")
	}
	payload, err := protojson.Marshal(create.Payload)
	if err != nil {
		return nil, err
	}

	query := `
		INSERT INTO review_config (
			id,
			creator_id,
			updater_id,
			name,
			payload
		)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING created_ts, updated_ts
	`

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	create.UpdaterUID = create.CreatorUID

	var createdTs int64
	var updatedTs int64
	if err := tx.QueryRowContext(ctx, query,
		create.ID,
		create.CreatorUID,
		create.CreatorUID,
		create.Name,
		payload,
	).Scan(
		&createdTs,
		&updatedTs,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, common.FormatDBErrorEmptyRowWithQuery(query)
		}
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, errors.Wrapf(err, "failed to commit transaction")
	}

	create.Enforce = true
	create.CreatedTime = time.Unix(createdTs, 0)
	create.UpdatedTime = time.Unix(updatedTs, 0)

	return create, nil
}

// DeleteReviewConfig deletes sql review by ID.
func (s *Store) DeleteReviewConfig(ctx context.Context, id string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, `DELETE FROM review_config WHERE id = $1`, id); err != nil {
		return err
	}

	return tx.Commit()
}

// UpdateReviewConfig updates the sql review.
func (s *Store) UpdateReviewConfig(ctx context.Context, patch *PatchReviewConfigMessage) (*ReviewConfigMessage, error) {
	set, args := []string{"updater_id = $1", "updated_ts = $2"}, []any{patch.UpdaterID, time.Now().Unix()}
	if v := patch.Enforce; v != nil {
		rowStatus := api.Normal
		if !*patch.Enforce {
			rowStatus = api.Archived
		}
		set, args = append(set, fmt.Sprintf(`"row_status" = $%d`, len(args)+1)), append(args, rowStatus)
	}
	if v := patch.Name; v != nil {
		set, args = append(set, fmt.Sprintf("name = $%d", len(args)+1)), append(args, *v)
	}
	if v := patch.Payload; v != nil {
		payload, err := protojson.Marshal(v)
		if err != nil {
			return nil, err
		}
		set, args = append(set, fmt.Sprintf("payload = $%d", len(args)+1)), append(args, payload)
	}
	args = append(args, patch.ID)

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	query := fmt.Sprintf(`
		UPDATE review_config
		SET `+strings.Join(set, ", ")+`
		WHERE id = $%d
		RETURNING
			id,
			row_status,
			creator_id,
			created_ts,
			updater_id,
			updated_ts,
			name,
			payload
		`, len(args))

	var sqlReview ReviewConfigMessage
	var createdTs int64
	var updatedTs int64
	var payload []byte
	var rowStatus string

	if err := tx.QueryRowContext(ctx, query, args...).Scan(
		&sqlReview.ID,
		&rowStatus,
		&sqlReview.CreatorUID,
		&createdTs,
		&sqlReview.UpdaterUID,
		&updatedTs,
		&sqlReview.Name,
		&payload,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	reviewConfigPyload := &storepb.ReviewConfigPayload{}
	if err := common.ProtojsonUnmarshaler.Unmarshal(payload, reviewConfigPyload); err != nil {
		return nil, err
	}

	sqlReview.Enforce = rowStatus == string(api.Normal)
	sqlReview.Payload = reviewConfigPyload
	sqlReview.CreatedTime = time.Unix(createdTs, 0)
	sqlReview.UpdatedTime = time.Unix(updatedTs, 0)

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return &sqlReview, nil
}
