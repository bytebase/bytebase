package store

import (
	"context"
	"strings"
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
	ResourceID string
	CreatedAt  time.Time

	Schema    string
	PlanTitle string
}

type FindChangelogMessage struct {
	ResourceID   *string
	InstanceID   *string
	DatabaseName *string

	Status          *ChangelogStatus
	CreatedAtBefore *time.Time
	CreatedAtAfter  *time.Time

	Limit  *int
	Offset *int

	// If false, Schema is omitted (empty string).
	ShowFull       bool
	HasSyncHistory bool
}

type UpdateChangelogMessage struct {
	ResourceID string

	SyncHistoryUID *int64
	Status         *ChangelogStatus
	DumpVersion    *int32
}

func (s *Store) CreateChangelog(ctx context.Context, create *ChangelogMessage) (string, error) {
	p, err := protojson.Marshal(create.Payload)
	if err != nil {
		return "", errors.Wrapf(err, "failed to marshal")
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
		RETURNING resource_id
	`, create.InstanceID, create.DatabaseName, create.Status, create.SyncHistoryUID, p)

	query, args, err := q.ToSQL()
	if err != nil {
		return "", errors.Wrapf(err, "failed to build sql")
	}

	var resourceID string
	if err := s.GetDB().QueryRowContext(ctx, query, args...).Scan(&resourceID); err != nil {
		return "", errors.Wrapf(err, "failed to insert")
	}

	return resourceID, nil
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

	query, args, err := qb.Q().Space("UPDATE changelog SET ? WHERE resource_id = ?", set, update.ResourceID).ToSQL()
	if err != nil {
		return errors.Wrapf(err, "failed to build sql")
	}

	if _, err := s.GetDB().ExecContext(ctx, query, args...); err != nil {
		return errors.Wrapf(err, "failed to update")
	}

	return nil
}

func (s *Store) ListChangelogs(ctx context.Context, find *FindChangelogMessage) ([]*ChangelogMessage, error) {
	// Avoid SQL-level string functions (e.g. LEFT()) on raw_dump — the column may
	// contain invalid UTF-8 from TiDB/OceanBase schema syncs (SQLSTATE 22021).
	// BASIC view skips the column entirely; FULL view fetches it and sanitizes in Go.
	schemaExpr := "''"
	if find.ShowFull {
		schemaExpr = "COALESCE(sh_cur.raw_dump, '')"
	}

	q := qb.Q().Space(`
		SELECT
			changelog.resource_id,
			changelog.created_at,
			changelog.instance,
			changelog.db_name,
			changelog.status,
			changelog.sync_history_id,
			` + schemaExpr + `,
			changelog.payload,
			COALESCE(plan.name, '')
		FROM changelog
		LEFT JOIN sync_history sh_cur ON sh_cur.id = changelog.sync_history_id
		LEFT JOIN LATERAL (
			SELECT task.plan_id
			FROM task
			WHERE task.resource_id = (regexp_match(changelog.payload->>'taskRun', 'tasks/([^/]+)/'))[1]
		) task_info ON TRUE
		LEFT JOIN plan ON plan.resource_id = task_info.plan_id
		WHERE TRUE
	`)

	if v := find.ResourceID; v != nil {
		q.And("changelog.resource_id = ?", *v)
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
	if v := find.CreatedAtBefore; v != nil {
		q.And("changelog.created_at <= ?", *v)
	}
	if v := find.CreatedAtAfter; v != nil {
		q.And("changelog.created_at >= ?", *v)
	}

	q.Space("ORDER BY changelog.created_at DESC")
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
			&c.ResourceID,
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

		// Sanitize invalid UTF-8 from external database schema dumps (e.g. TiDB/OceanBase with non-UTF-8 Chinese text).
		c.Schema = strings.ToValidUTF8(c.Schema, "")

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
