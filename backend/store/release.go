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
	query := `
		INSERT INTO release (
			creator_id,
			project,
			digest,
			payload
		) VALUES (
			$1,
			$2,
			$3,
			$4
		) RETURNING id, created_at
	`

	p, err := protojson.Marshal(release.Payload)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to marshal release payload")
	}

	tx, err := s.GetDB().BeginTx(ctx, nil)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to begin tx")
	}
	defer tx.Rollback()

	var id int64
	var createdTime time.Time
	if err := tx.QueryRowContext(ctx, query,
		creatorUID,
		release.ProjectID,
		release.Digest,
		p,
	).Scan(&id, &createdTime); err != nil {
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
	where, args := []string{"TRUE"}, []any{}
	if v := find.ProjectID; v != nil {
		where = append(where, fmt.Sprintf("project = $%d", len(args)+1))
		args = append(args, *v)
	}
	if v := find.UID; v != nil {
		where = append(where, fmt.Sprintf("id= $%d", len(args)+1))
		args = append(args, *v)
	}
	if v := find.Digest; v != nil {
		where = append(where, fmt.Sprintf("digest = $%d", len(args)+1))
		args = append(args, *v)
	}
	if !find.ShowDeleted {
		where, args = append(where, fmt.Sprintf("deleted = $%d", len(args)+1)), append(args, false)
	}

	query := fmt.Sprintf(`
		SELECT
			id,
			deleted,
			project,
			digest,
			creator_id,
			created_at,
			payload
		FROM release
		WHERE %s
		ORDER BY release.id DESC
	`, strings.Join(where, " AND "))

	if v := find.Limit; v != nil {
		query += fmt.Sprintf(" LIMIT %d", *v)
	}
	if v := find.Offset; v != nil {
		query += fmt.Sprintf(" OFFSET %d", *v)
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
	set, args := []string{}, []any{}

	if v := update.Deleted; v != nil {
		set, args = append(set, fmt.Sprintf(`deleted = $%d`, len(args)+1)), append(args, *v)
	}
	if v := update.Payload; v != nil {
		payload, err := protojson.Marshal(update.Payload)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to marshal payload")
		}
		set, args = append(set, fmt.Sprintf("payload = $%d", len(args)+1)), append(args, payload)
	}
	if len(set) == 0 {
		return nil, errors.New("no update field provided")
	}

	args = append(args, update.UID)

	tx, err := s.GetDB().BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, fmt.Sprintf(`UPDATE release SET %s WHERE id = $%d`, strings.Join(set, ", "), len(args)), args...); err != nil {
		return nil, errors.Wrapf(err, "failed to query row")
	}

	if err := tx.Commit(); err != nil {
		return nil, errors.Wrapf(err, "failed to commit tx")
	}

	return s.GetRelease(ctx, update.UID)
}
