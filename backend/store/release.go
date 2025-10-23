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

type ReleaseMessage struct {
	ProjectID string
	Digest    string
	Payload   *storepb.ReleasePayload

	// output only
	UID        int64
	Deleted    bool
	CreatorUID int
	At         time.Time
}

type FindReleaseMessage struct {
	ProjectID   *string
	UID         *int64
	Digest      *string
	Limit       *int
	Offset      *int
	ShowDeleted bool
}

type UpdateReleaseMessage struct {
	UID int64

	Deleted *bool
	Payload *storepb.ReleasePayload
}

func (s *Store) CreateRelease(ctx context.Context, release *ReleaseMessage, creatorUID int) (*ReleaseMessage, error) {
	p, err := protojson.Marshal(release.Payload)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to marshal release payload")
	}

	q := qb.Q().Space(`
		INSERT INTO release (
			creator_id,
			project,
			digest,
			payload
		) VALUES (
			?,
			?,
			?,
			?
		) RETURNING id, created_at
	`, creatorUID, release.ProjectID, release.Digest, p)

	query, args, err := q.ToSQL()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to build sql")
	}

	tx, err := s.GetDB().BeginTx(ctx, nil)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to begin tx")
	}
	defer tx.Rollback()

	var id int64
	var createdTime time.Time
	if err := tx.QueryRowContext(ctx, query, args...).Scan(&id, &createdTime); err != nil {
		return nil, errors.Wrapf(err, "failed to insert release")
	}

	if err := tx.Commit(); err != nil {
		return nil, errors.Wrapf(err, "failed to commit tx")
	}

	release.UID = id
	release.CreatorUID = creatorUID
	release.At = createdTime

	return release, nil
}

func (s *Store) GetRelease(ctx context.Context, uid int64) (*ReleaseMessage, error) {
	releases, err := s.ListReleases(ctx, &FindReleaseMessage{UID: &uid, ShowDeleted: true})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to list releases")
	}
	if len(releases) == 0 {
		return nil, nil
	}
	if len(releases) > 1 {
		return nil, errors.Errorf("found %d releases with uid=%v, expect 1", len(releases), uid)
	}
	return releases[0], nil
}

func (s *Store) ListReleases(ctx context.Context, find *FindReleaseMessage) ([]*ReleaseMessage, error) {
	q := qb.Q().Space(`
		SELECT
			id,
			deleted,
			project,
			digest,
			creator_id,
			created_at,
			payload
		FROM release
		WHERE TRUE
	`)

	if v := find.ProjectID; v != nil {
		q.And("project = ?", *v)
	}
	if v := find.UID; v != nil {
		q.And("id = ?", *v)
	}
	if v := find.Digest; v != nil {
		q.And("digest = ?", *v)
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

	tx, err := s.GetDB().BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to begin tx")
	}
	defer tx.Rollback()

	rows, err := tx.QueryContext(ctx, query, args...)
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

		if err := rows.Scan(
			&r.UID,
			&r.Deleted,
			&r.ProjectID,
			&r.Digest,
			&r.CreatorUID,
			&r.At,
			&payload,
		); err != nil {
			return nil, errors.Wrapf(err, "failed to scan rows")
		}

		if err := common.ProtojsonUnmarshaler.Unmarshal(payload, r.Payload); err != nil {
			return nil, errors.Wrapf(err, "failed to unmarshal payload")
		}

		releases = append(releases, &r)
	}

	if err := rows.Err(); err != nil {
		return nil, errors.Wrapf(err, "rows err")
	}

	if err := tx.Commit(); err != nil {
		return nil, errors.Wrapf(err, "failed to commit tx")
	}

	return releases, nil
}

func (s *Store) UpdateRelease(ctx context.Context, update *UpdateReleaseMessage) (*ReleaseMessage, error) {
	q := qb.Q().Space("UPDATE release SET")
	hasUpdate := false

	if v := update.Deleted; v != nil {
		if hasUpdate {
			q.Space(",")
		}
		q.Space("deleted = ?", *v)
		hasUpdate = true
	}
	if v := update.Payload; v != nil {
		payload, err := protojson.Marshal(update.Payload)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to marshal payload")
		}
		if hasUpdate {
			q.Space(",")
		}
		q.Space("payload = ?", payload)
		hasUpdate = true
	}

	if !hasUpdate {
		return nil, errors.New("no update field provided")
	}

	q.Space("WHERE id = ?", update.UID)

	query, args, err := q.ToSQL()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to build sql")
	}

	tx, err := s.GetDB().BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, query, args...); err != nil {
		return nil, errors.Wrapf(err, "failed to query row")
	}

	if err := tx.Commit(); err != nil {
		return nil, errors.Wrapf(err, "failed to commit tx")
	}

	return s.GetRelease(ctx, update.UID)
}
