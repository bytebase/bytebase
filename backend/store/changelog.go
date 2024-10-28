package store

import (
	"context"
	"fmt"
	"strings"
	"time"

	"google.golang.org/protobuf/encoding/protojson"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

type ChangelogMessage struct {
	DatabaseUID int
	Payload     *storepb.ChangelogPayload

	// output only
	UID         int64
	CreatorUID  int
	CreatedTime time.Time
}

type FindChangelogMessage struct {
	UID         *int64
	DatabaseUID *int

	Limit  *int
	Offset *int
}

type UpdateChangelogMessage struct {
	UID int64

	SyncHistoryUID *int64
	RevisionUID    *int64
	Status         *storepb.ChangelogTask_Status
}

func (s *Store) CreateChangelog(ctx context.Context, create *ChangelogMessage, creatorUID int) (int64, error) {
	query := `
		INSERT INTO changelog (
			creator_id,
			database_id,
			payload
		) VALUES (
		 	$1,
			$2,
			$3
		)
		RETURNING id
	`

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, errors.Wrapf(err, "failed to begin tx")
	}
	defer tx.Rollback()

	p, err := protojson.Marshal(create.Payload)
	if err != nil {
		return 0, errors.Wrapf(err, "failed to marshal")
	}

	var id int64
	if err := tx.QueryRowContext(ctx, query, creatorUID, create.DatabaseUID, p).Scan(&id); err != nil {
		return 0, errors.Wrapf(err, "failed to insert")
	}

	if err := tx.Commit(); err != nil {
		return 0, errors.Wrapf(err, "failed to commit tx")
	}

	return id, nil
}

func (s *Store) UpdateChangelog(ctx context.Context, update *UpdateChangelogMessage) error {
	args := []any{update.UID}
	var payloadSet []string

	if v := update.SyncHistoryUID; v != nil {
		payloadSet = append(payloadSet, fmt.Sprintf("jsonb_build_object('syncHistoryId', $%d::TEXT)", len(args)+1))
		args = append(args, fmt.Sprintf("%d", *v))
	}
	if v := update.RevisionUID; v != nil {
		payloadSet = append(payloadSet, fmt.Sprintf("jsonb_build_object('revision', $%d::TEXT)", len(args)+1))
		args = append(args, fmt.Sprintf("%d", *v))
	}
	if v := update.Status; v != nil {
		payloadSet = append(payloadSet, fmt.Sprintf("jsonb_build_object('status', $%d::TEXT)", len(args)+1))
		args = append(args, v.String())
	}

	if len(payloadSet) == 0 {
		return errors.Errorf("update nothing")
	}

	query := fmt.Sprintf(`
		UPDATE changelog
		SET payload = jsonb_set(payload, '{task}', payload->'task' || %s)
		WHERE id = $1
	`, strings.Join(payloadSet, " || "))

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return errors.Wrapf(err, "failed to begin tx")
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, query, args...); err != nil {
		return errors.Wrapf(err, "failed to update")
	}

	if err := tx.Commit(); err != nil {
		return errors.Wrapf(err, "failed to commit")
	}

	return nil
}

func (s *Store) ListChangelogs(ctx context.Context, find *FindChangelogMessage) ([]*ChangelogMessage, error) {
	where, args := []string{"TRUE"}, []any{}
	if v := find.UID; v != nil {
		where, args = append(where, fmt.Sprintf("changelog.id = $%d", len(args)+1)), append(args, *v)
	}
	if v := find.DatabaseUID; v != nil {
		where, args = append(where, fmt.Sprintf("changelog.database_id = $%d", len(args)+1)), append(args, *v)
	}

	query := fmt.Sprintf(`
		SELECT
			changelog.id,
			changelog.creator_id,
			changelog.created_ts,
			changelog.database_id,
			payload
		FROM changelog
		WHERE %s
		ORDER BY changelog.id DESC
	`, strings.Join(where, " AND "))
	if v := find.Limit; v != nil {
		query += fmt.Sprintf(" LIMIT %d", *v)
	}
	if v := find.Offset; v != nil {
		query += fmt.Sprintf(" OFFSET %d", *v)
	}

	rows, err := s.db.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to query")
	}
	defer rows.Close()

	var changelogs []*ChangelogMessage
	for rows.Next() {
		c := ChangelogMessage{
			Payload: &storepb.ChangelogPayload{},
		}
		var payload []byte

		if err := rows.Scan(
			&c.UID,
			&c.CreatorUID,
			&c.CreatedTime,
			&c.DatabaseUID,
			&payload,
		); err != nil {
			return nil, errors.Wrapf(err, "failed to scan")
		}

		if err := common.ProtojsonUnmarshaler.Unmarshal(payload, c.Payload); err != nil {
			return nil, errors.Wrapf(err, "failed to unmarshal")
		}

		changelogs = append(changelogs, &c)
	}

	if err := rows.Err(); err != nil {
		return nil, errors.Wrapf(err, "rows err")
	}

	return changelogs, nil
}

func (s *Store) GetChangelog(ctx context.Context, uid int64) (*ChangelogMessage, error) {
	changelogs, err := s.ListChangelogs(ctx, &FindChangelogMessage{
		UID: &uid,
	})
	if err != nil {
		return nil, err
	}
	if len(changelogs) == 0 {
		return nil, nil
	}
	if len(changelogs) > 1 {
		return nil, errors.Errorf("found %d changelogs with UID %d, expect 1", len(changelogs), uid)
	}
	return changelogs[0], nil
}
