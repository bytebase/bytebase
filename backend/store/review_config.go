package store

import (
	"context"
	"database/sql"

	"github.com/pkg/errors"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/qb"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

// ReviewConfigMessage is the API message for sql review.
type ReviewConfigMessage struct {
	ID      string
	Enforce bool
	Name    string
	Payload *storepb.ReviewConfigPayload
}

// FindReviewConfigMessage is the API message for finding sql review.
type FindReviewConfigMessage struct {
	ID *string
}

// PatchReviewConfigMessage is the message to patch a sql review.
type PatchReviewConfigMessage struct {
	ID      string
	Name    *string
	Enforce *bool
	Payload *storepb.ReviewConfigPayload
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
	q := qb.Q().Space(`
		SELECT
			id,
			enabled,
			name,
			payload
		FROM review_config
		WHERE TRUE
	`)

	if v := find.ID; v != nil {
		q.And("id = ?", *v)
	}

	query, args, err := q.ToSQL()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to build sql")
	}

	rows, err := s.GetDB().QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sqlReviewList []*ReviewConfigMessage
	for rows.Next() {
		var sqlReview ReviewConfigMessage
		var payload []byte

		if err := rows.Scan(
			&sqlReview.ID,
			&sqlReview.Enforce,
			&sqlReview.Name,
			&payload,
		); err != nil {
			return nil, err
		}

		reviewConfigPyload := &storepb.ReviewConfigPayload{}
		if err := common.ProtojsonUnmarshaler.Unmarshal(payload, reviewConfigPyload); err != nil {
			return nil, err
		}

		sqlReview.Payload = reviewConfigPyload
		sqlReviewList = append(sqlReviewList, &sqlReview)
	}
	if err := rows.Err(); err != nil {
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

	q := qb.Q().Space(`
		INSERT INTO review_config (
			id,
			name,
			payload
		)
		VALUES (?, ?, ?)
	`, create.ID, create.Name, payload)

	query, args, err := q.ToSQL()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to build sql")
	}

	if _, err := s.GetDB().ExecContext(ctx, query, args...); err != nil {
		if err == sql.ErrNoRows {
			return nil, common.FormatDBErrorEmptyRowWithQuery(query)
		}
		return nil, err
	}

	create.Enforce = true

	return create, nil
}

// DeleteReviewConfig deletes sql review by ID.
func (s *Store) DeleteReviewConfig(ctx context.Context, id string) error {
	q := qb.Q().Space("DELETE FROM review_config WHERE id = ?", id)

	query, args, err := q.ToSQL()
	if err != nil {
		return errors.Wrapf(err, "failed to build sql")
	}

	if _, err := s.GetDB().ExecContext(ctx, query, args...); err != nil {
		return err
	}

	return nil
}

// UpdateReviewConfig updates the sql review.
func (s *Store) UpdateReviewConfig(ctx context.Context, patch *PatchReviewConfigMessage) (*ReviewConfigMessage, error) {
	set := qb.Q()
	if v := patch.Enforce; v != nil {
		set.Comma("enabled = ?", *v)
	}
	if v := patch.Name; v != nil {
		set.Comma("name = ?", *v)
	}
	if v := patch.Payload; v != nil {
		payload, err := protojson.Marshal(v)
		if err != nil {
			return nil, err
		}
		set.Comma("payload = ?", payload)
	}
	if set.Len() == 0 {
		return nil, errors.New("no fields to update")
	}

	q := qb.Q().Space(`
		UPDATE review_config
		SET ?
		WHERE id = ?
		RETURNING
			id,
			enabled,
			name,
			payload
	`, set, patch.ID)

	query, args, err := q.ToSQL()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to build sql")
	}

	var sqlReview ReviewConfigMessage
	var payload []byte

	if err := s.GetDB().QueryRowContext(ctx, query, args...).Scan(
		&sqlReview.ID,
		&sqlReview.Enforce,
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

	sqlReview.Payload = reviewConfigPyload

	return &sqlReview, nil
}
