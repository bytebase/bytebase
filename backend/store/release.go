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

type ReleaseMessage struct {
	ProjectUID int
	Payload    *storepb.ReleasePayload

	// output only
	UID         int64
	Deleted     bool
	CreatorUID  int
	CreatedTime time.Time
}

type FindReleaseMessage struct {
	ProjectUID  *int
	UID         *int64
	Limit       *int
	Offset      *int
	ShowDeleted bool
}

type UpdateReleaseMessage struct {
	UID int64

	Deleted *bool
	Payload *storepb.ReleasePayload
}

// TODO(p0ny): enforce file order by version.
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

func (s *Store) GetRelease(ctx context.Context, uid int64) (*ReleaseMessage, error) {
	releases, err := s.ListReleases(ctx, &FindReleaseMessage{UID: &uid})
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
	if v := find.ProjectUID; v != nil {
		where = append(where, fmt.Sprintf("release.project_id = $%d", len(args)+1))
		args = append(args, *v)
	}
	if v := find.UID; v != nil {
		where = append(where, fmt.Sprintf("release.id= $%d", len(args)+1))
		args = append(args, *v)
	}
	if !find.ShowDeleted {
		where, args = append(where, fmt.Sprintf("release.row_status = $%d", len(args)+1)), append(args, api.Normal)
	}

	query := fmt.Sprintf(`
		SELECT
			id,
			row_status,
			project_id,
			creator_id,
			created_ts,
			payload
		FROM release
		WHERE %s
	`, strings.Join(where, " AND "))

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
		var rowStatus string
		var payload []byte

		if err := rows.Scan(
			&r.UID,
			&rowStatus,
			&r.ProjectUID,
			&r.CreatorUID,
			&r.CreatedTime,
			&payload,
		); err != nil {
			return nil, errors.Wrapf(err, "failed to scan rows")
		}

		r.Deleted = convertRowStatusToDeleted(rowStatus)
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
		rowStatus := api.Normal
		if *v {
			rowStatus = api.Archived
		}
		set, args = append(set, fmt.Sprintf(`row_status = $%d`, len(args)+1)), append(args, rowStatus)
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

	tx, err := s.db.BeginTx(ctx, nil)
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
