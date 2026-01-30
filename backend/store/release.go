package store

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/pkg/errors"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/qb"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

type ReleaseMessage struct {
	ProjectID string
	ReleaseID string
	Payload   *storepb.ReleasePayload

	Deleted   bool
	Creator   string
	At        time.Time
	Train     string
	Iteration int32
	Category  string
}

type FindReleaseMessage struct {
	ProjectID   *string
	ReleaseID   *string
	Category    *string
	Limit       *int
	Offset      *int
	ShowDeleted bool
}

type UpdateReleaseMessage struct {
	ProjectID string
	ReleaseID string

	Deleted  *bool
	Payload  *storepb.ReleasePayload
	Category *string
}

func (s *Store) CreateRelease(ctx context.Context, release *ReleaseMessage, creator string) (*ReleaseMessage, error) {
	p, err := protojson.Marshal(release.Payload)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to marshal release payload")
	}

	tx, err := s.GetDB().BeginTx(ctx, nil)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to begin tx")
	}
	defer tx.Rollback()

	// Atomically get next iteration for (project, train)
	// Lock all rows for this (project, train) to prevent concurrent inserts with same iteration
	var maxIteration sql.NullInt64
	err = tx.QueryRowContext(ctx,
		`SELECT COALESCE(MAX(iteration), -1) FROM (
			SELECT iteration FROM release WHERE project = $1 AND train = $2 FOR UPDATE
		) sub`,
		release.ProjectID, release.Train,
	).Scan(&maxIteration)
	if err != nil && err != sql.ErrNoRows {
		return nil, errors.Wrapf(err, "failed to get max iteration")
	}

	nextIteration := int32(0)
	if maxIteration.Valid {
		nextIteration = int32(maxIteration.Int64) + 1
	}

	// Compute release_id = train + formatted iteration
	releaseID := fmt.Sprintf("%s%02d", release.Train, nextIteration)

	// Convert empty creator to NULL for system-generated releases
	var creatorPtr any
	if creator == "" {
		creatorPtr = nil
	} else {
		creatorPtr = creator
	}

	q := qb.Q().Space(`
		INSERT INTO release (
			creator,
			project,
			payload,
			release_id,
			train,
			iteration,
			category
		) VALUES (
			?,
			?,
			?,
			?,
			?,
			?,
			?
		) RETURNING id, created_at
	`, creatorPtr, release.ProjectID, p, releaseID, release.Train, nextIteration, release.Category)

	query, args, err := q.ToSQL()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to build sql")
	}

	var createdTime time.Time
	var id int64 // Still needed for RETURNING clause
	if err := tx.QueryRowContext(ctx, query, args...).Scan(&id, &createdTime); err != nil {
		return nil, errors.Wrapf(err, "failed to insert release")
	}

	if err := tx.Commit(); err != nil {
		return nil, errors.Wrapf(err, "failed to commit tx")
	}

	release.Creator = creator
	release.At = createdTime
	release.ReleaseID = releaseID
	release.Iteration = nextIteration

	return release, nil
}

func (s *Store) GetRelease(ctx context.Context, find *FindReleaseMessage) (*ReleaseMessage, error) {
	releases, err := s.ListReleases(ctx, find)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to list releases")
	}
	if len(releases) == 0 {
		return nil, nil
	}
	if len(releases) > 1 {
		return nil, errors.Errorf("found %d releases, expect 1", len(releases))
	}
	return releases[0], nil
}

func (s *Store) ListReleases(ctx context.Context, find *FindReleaseMessage) ([]*ReleaseMessage, error) {
	q := qb.Q().Space(`
		SELECT
			deleted,
			project,
			creator,
			created_at,
			payload,
			release_id,
			train,
			iteration,
			category
		FROM release
		WHERE TRUE
	`)

	if v := find.ProjectID; v != nil {
		q.And("project = ?", *v)
	}
	if v := find.ReleaseID; v != nil {
		q.And("release_id = ?", *v)
	}
	if v := find.Category; v != nil {
		q.And("category = ?", *v)
	}
	if !find.ShowDeleted {
		q.And("deleted = ?", false)
	}

	q.Space("ORDER BY release.id DESC")
	if v := find.Limit; v != nil {
		q.Space("LIMIT ?", *v)
	}
	if v := find.Offset; v != nil {
		q.Space("OFFSET ?", *v)
	}

	query, args, err := q.ToSQL()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to build sql")
	}

	rows, err := s.GetDB().QueryContext(ctx, query, args...)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to query rows")
	}
	defer rows.Close()

	var releases []*ReleaseMessage
	for rows.Next() {
		r := ReleaseMessage{
			Payload: &storepb.ReleasePayload{},
		}
		var payload []byte
		var creator sql.NullString

		if err := rows.Scan(
			&r.Deleted,
			&r.ProjectID,
			&creator,
			&r.At,
			&payload,
			&r.ReleaseID,
			&r.Train,
			&r.Iteration,
			&r.Category,
		); err != nil {
			return nil, errors.Wrapf(err, "failed to scan rows")
		}
		if creator.Valid {
			r.Creator = creator.String
		}

		if err := common.ProtojsonUnmarshaler.Unmarshal(payload, r.Payload); err != nil {
			return nil, errors.Wrapf(err, "failed to unmarshal payload")
		}

		releases = append(releases, &r)
	}

	if err := rows.Err(); err != nil {
		return nil, errors.Wrapf(err, "rows err")
	}

	return releases, nil
}

func (s *Store) UpdateRelease(ctx context.Context, update *UpdateReleaseMessage) (*ReleaseMessage, error) {
	set := qb.Q()
	if v := update.Deleted; v != nil {
		set.Comma("deleted = ?", *v)
	}
	if v := update.Payload; v != nil {
		payload, err := protojson.Marshal(update.Payload)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to marshal payload")
		}
		set.Comma("payload = ?", payload)
	}
	if v := update.Category; v != nil {
		set.Comma("category = ?", *v)
	}

	if set.Len() == 0 {
		return nil, errors.New("no update field provided")
	}

	query, args, err := qb.Q().Space("UPDATE release SET ? WHERE project = ? AND release_id = ?", set, update.ProjectID, update.ReleaseID).ToSQL()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to build sql")
	}

	if _, err := s.GetDB().ExecContext(ctx, query, args...); err != nil {
		return nil, errors.Wrapf(err, "failed to query row")
	}

	return s.GetRelease(ctx, &FindReleaseMessage{
		ProjectID: &update.ProjectID,
		ReleaseID: &update.ReleaseID,
	})
}

func (s *Store) ListReleaseCategories(ctx context.Context, projectID string) ([]string, error) {
	query := `
		SELECT DISTINCT category
		FROM release
		WHERE project = $1
		  AND category != ''
		  AND deleted = FALSE
		ORDER BY category ASC
	`

	rows, err := s.GetDB().QueryContext(ctx, query, projectID)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to query rows")
	}
	defer rows.Close()

	var categories []string
	for rows.Next() {
		var category string
		if err := rows.Scan(&category); err != nil {
			return nil, errors.Wrapf(err, "failed to scan row")
		}
		categories = append(categories, category)
	}

	if err := rows.Err(); err != nil {
		return nil, errors.Wrapf(err, "rows err")
	}

	return categories, nil
}
