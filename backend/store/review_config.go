package store

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/pkg/errors"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/bytebase/bytebase/backend/common"
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
	where, args := []string{"TRUE"}, []any{}

	if v := find.ID; v != nil {
		where, args = append(where, fmt.Sprintf("id = $%d", len(args)+1)), append(args, *v)
	}

	tx, err := s.GetDB().BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	rows, err := tx.QueryContext(ctx, fmt.Sprintf(`
		SELECT
			id,
			enabled,
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
			name,
			payload
		)
		VALUES ($1, $2, $3)
	`

	tx, err := s.GetDB().BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, query,
		create.ID,
		create.Name,
		payload,
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

	return create, nil
}

// DeleteReviewConfig deletes sql review by ID.
func (s *Store) DeleteReviewConfig(ctx context.Context, id string) error {
	tx, err := s.GetDB().BeginTx(ctx, nil)
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
	set, args := []string{}, []any{}
	if v := patch.Enforce; v != nil {
		set, args = append(set, fmt.Sprintf("enabled = $%d", len(args)+1)), append(args, *v)
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

	tx, err := s.GetDB().BeginTx(ctx, nil)
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
			enabled,
			name,
			payload
		`, len(args))

	var sqlReview ReviewConfigMessage
	var payload []byte

	if err := tx.QueryRowContext(ctx, query, args...).Scan(
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

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return &sqlReview, nil
}
