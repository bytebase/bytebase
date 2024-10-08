package store

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/pkg/errors"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

type ReleaseMessage struct {
	ProjectUID int
	Payload    *storepb.ReleasePayload

	// output only
	UID         int64
	CreatorUID  int
	CreatedTime time.Time
}

type FindReleaseMessage struct {
	ProjectUID int
	Limit      *int
	Offset     *int
}

func (s *Store) CreateRelease(ctx context.Context, release *ReleaseMessage, creatorUID int) (*ReleaseMessage, error) {
	query := `
		INSERT INTO release (
			creator_id,
			project_id,
			payload
		) VALUES (
			$1,
			$2,
			$3
		) RETURNING id, created_ts
	`

	p, err := protojson.Marshal(release.Payload)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to marshal release payload")
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to begin tx")
	}
	defer tx.Rollback()

	var id int64
	var createdTime time.Time
	if err := tx.QueryRowContext(ctx, query,
		creatorUID,
		release.ProjectUID,
		p,
	).Scan(&id, &createdTime); err != nil {
		return nil, errors.Wrapf(err, "failed to insert release")
	}

	if err := tx.Commit(); err != nil {
		return nil, errors.Wrapf(err, "failed to commit tx")
	}

	release.UID = id
	release.CreatorUID = creatorUID
	release.CreatedTime = createdTime

	return release, nil
}

func (s *Store) ListReleases(ctx context.Context, find *FindReleaseMessage) ([]*ReleaseMessage, error) {
	query := `
		SELECT
			id,
			project_id,
			creator_id,
			created_ts,
			payload
		FROM release
		WHERE project_id = $1
	`
	if v := find.Limit; v != nil {
		query += fmt.Sprintf(" LIMIT %d", *v)
	}
	if v := find.Offset; v != nil {
		query += fmt.Sprintf(" OFFSET %d", *v)
	}

	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to begin tx")
	}
	defer tx.Rollback()

	rows, err := tx.QueryContext(ctx, query, find.ProjectUID)
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
			&r.ProjectUID,
			&r.CreatorUID,
			&r.CreatedTime,
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
