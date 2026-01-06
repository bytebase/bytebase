package store

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/protobuf/encoding/protojson"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/qb"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

type ChangelogStatus string

const (
	ChangelogStatusPending ChangelogStatus = "PENDING"
	ChangelogStatusDone    ChangelogStatus = "DONE"
	ChangelogStatusFailed  ChangelogStatus = "FAILED"
)

type ChangelogMessage struct {
	InstanceID   string
	DatabaseName string
	Payload      *storepb.ChangelogPayload

	SyncHistoryUID *int64
	Status         ChangelogStatus

	// output only
	UID       int64
	CreatedAt time.Time

	Schema    string
	PlanTitle string
}

type FindChangelogMessage struct {
	UID          *int64
	InstanceID   *string
	DatabaseName *string

	TypeList []string
	Status   *ChangelogStatus

	Limit  *int
	Offset *int

	// If false, PrevSchema, Schema are truncated
	ShowFull       bool
	HasSyncHistory bool
}

type UpdateChangelogMessage struct {
	UID int64

	SyncHistoryUID *int64
	Status         *ChangelogStatus
	DumpVersion    *int32
}

func (s *Store) CreateChangelog(ctx context.Context, create *ChangelogMessage) (int64, error) {
	tx, err := s.GetDB().BeginTx(ctx, nil)
	if err != nil {
		return 0, errors.Wrapf(err, "failed to begin tx")
	}
	defer tx.Rollback()

	p, err := protojson.Marshal(create.Payload)
	if err != nil {
		return 0, errors.Wrapf(err, "failed to marshal")
	}

	q := qb.Q().Space(`
		INSERT INTO changelog (
			instance,
			db_name,
			status,
			sync_history_id,
			payload
		) VALUES (
		 	?,
			?,
			?,
			?,
			?
		)
		RETURNING id
	`, create.InstanceID, create.DatabaseName, create.Status, create.SyncHistoryUID, p)

	query, args, err := q.ToSQL()
	if err != nil {
		return 0, errors.Wrapf(err, "failed to build sql")
	}

	var id int64
	if err := tx.QueryRowContext(ctx, query, args...).Scan(&id); err != nil {
		return 0, errors.Wrapf(err, "failed to insert")
	}

	if err := tx.Commit(); err != nil {
		return 0, errors.Wrapf(err, "failed to commit tx")
	}

	return id, nil
}

func (s *Store) UpdateChangelog(ctx context.Context, update *UpdateChangelogMessage) error {
	set := qb.Q()
	if v := update.SyncHistoryUID; v != nil {
		set.Comma("sync_history_id = ?", *v)
	}
	if v := update.Status; v != nil {
		set.Comma("status = ?", *v)
	}
	if v := update.DumpVersion; v != nil {
		set.Comma("payload = payload || jsonb_build_object('dumpVersion', ?::INT)", *v)
	}

	if set.Len() == 0 {
		return errors.Errorf("update nothing")
	}

	query, args, err := qb.Q().Space("UPDATE changelog SET ? WHERE id = ?", set, update.UID).ToSQL()
	if err != nil {
		return errors.Wrapf(err, "failed to build sql")
	}

	tx, err := s.GetDB().BeginTx(ctx, nil)
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
	truncateSize := 512
	if common.IsDev() {
		truncateSize = 4
	}
	shCurField := fmt.Sprintf("LEFT(sh_cur.raw_dump, %d)", truncateSize)
	if find.ShowFull {
		shCurField = "sh_cur.raw_dump"
	}

	q := qb.Q().Space(fmt.Sprintf(`
		SELECT
			changelog.id,
			changelog.created_at,
			changelog.instance,
			changelog.db_name,
			changelog.status,
			changelog.sync_history_id,
			COALESCE(%s, ''),
			changelog.payload,
			COALESCE(plan.name, '')
		FROM changelog
		LEFT JOIN sync_history sh_cur ON sh_cur.id = changelog.sync_history_id
		LEFT JOIN LATERAL (
			SELECT task.plan_id
			FROM task
			WHERE task.id = (regexp_match(changelog.payload->>'taskRun', 'tasks/(\d+)/'))[1]::int
		) task_info ON TRUE
		LEFT JOIN plan ON plan.id = task_info.plan_id
		WHERE TRUE
	`,
		shCurField,
	))

	if v := find.UID; v != nil {
		q.And("changelog.id = ?", *v)
	}
	if v := find.InstanceID; v != nil {
		q.And("changelog.instance = ?", *v)
	}
	if v := find.DatabaseName; v != nil {
		q.And("changelog.db_name = ?", *v)
	}
	if v := find.Status; v != nil {
		q.And("changelog.status = ?", string(*v))
	}
	if find.HasSyncHistory {
		q.And("changelog.sync_history_id IS NOT NULL")
	}
	if len(find.TypeList) > 0 {
		q.And("changelog.payload->>'type' = ANY(?)", find.TypeList)
	}

	q.Space("ORDER BY changelog.id DESC")
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
			&c.CreatedAt,
			&c.InstanceID,
			&c.DatabaseName,
			&c.Status,
			&c.SyncHistoryUID,
			&c.Schema,
			&payload,
			&c.PlanTitle,
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

func (s *Store) GetChangelog(ctx context.Context, find *FindChangelogMessage) (*ChangelogMessage, error) {
	changelogs, err := s.ListChangelogs(ctx, find)
	if err != nil {
		return nil, err
	}
	if len(changelogs) == 0 {
		return nil, nil
	}
	if len(changelogs) > 1 {
		return nil, errors.Errorf("found %d changelogs with find %v, expect 1", len(changelogs), *find)
	}
	return changelogs[0], nil
}
