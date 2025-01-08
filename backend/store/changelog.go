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

type ChangelogStatus string

const (
	ChangelogStatusPending ChangelogStatus = "PENDING"
	ChangelogStatusDone    ChangelogStatus = "DONE"
	ChangelogStatusFailed  ChangelogStatus = "FAILED"
)

type ChangelogMessage struct {
	DatabaseUID int
	Payload     *storepb.ChangelogPayload

	PrevSyncHistoryUID *int64
	SyncHistoryUID     *int64
	Status             ChangelogStatus

	// output only
	UID         int64
	CreatorUID  int
	CreatedTime time.Time

	PrevSchema    string
	Schema        string
	Statement     string
	StatementSize int64
}

type FindChangelogMessage struct {
	UID         *int64
	DatabaseUID *int

	TypeList        []string
	Status          *ChangelogStatus
	ResourcesFilter *string

	Limit  *int
	Offset *int

	// If false, PrevSchema, Schema are truncated
	ShowFull       bool
	HasSyncHistory bool
}

type UpdateChangelogMessage struct {
	UID int64

	SyncHistoryUID *int64
	RevisionUID    *int64
	Status         *ChangelogStatus
}

func (s *Store) CreateChangelog(ctx context.Context, create *ChangelogMessage, creatorUID int) (int64, error) {
	query := `
		INSERT INTO changelog (
			creator_id,
			database_id,
			status,
			prev_sync_history_id,
			sync_history_id,
			payload
		) VALUES (
		 	$1,
			$2,
			$3,
			$4,
			$5,
			$6
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
	if err := tx.QueryRowContext(ctx, query, creatorUID, create.DatabaseUID, create.Status, create.PrevSyncHistoryUID, create.SyncHistoryUID, p).Scan(&id); err != nil {
		return 0, errors.Wrapf(err, "failed to insert")
	}

	if err := tx.Commit(); err != nil {
		return 0, errors.Wrapf(err, "failed to commit tx")
	}

	return id, nil
}

func (s *Store) UpdateChangelog(ctx context.Context, update *UpdateChangelogMessage) error {
	args := []any{update.UID}
	var set []string

	if v := update.SyncHistoryUID; v != nil {
		set = append(set, fmt.Sprintf("sync_history_id = $%d", len(args)+1))
		args = append(args, *v)
	}
	if v := update.RevisionUID; v != nil {
		set = append(set, fmt.Sprintf(`payload = payload || '{"revision": "%d"}'`, *v))
	}
	if v := update.Status; v != nil {
		set = append(set, fmt.Sprintf("status = $%d", len(args)+1))
		args = append(args, *v)
	}

	if len(set) == 0 {
		return errors.Errorf("update nothing")
	}

	query := fmt.Sprintf(`
		UPDATE changelog
		SET %s
		WHERE id = $1
	`, strings.Join(set, " , "))

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
	if v := find.ResourcesFilter; v != nil {
		text, err := generateResourceFilter(*v, "changelog.payload")
		if err != nil {
			return nil, errors.Wrapf(err, "failed to generate resource filter from %q", *v)
		}
		if text != "" {
			where = append(where, text)
		}
	}
	if v := find.Status; v != nil {
		where, args = append(where, fmt.Sprintf("changelog.status = $%d", len(args)+1)), append(args, string(*v))
	}
	if find.HasSyncHistory {
		where = append(where, "changelog.sync_history_id IS NOT NULL")
	}
	if len(find.TypeList) > 0 {
		where = append(where, fmt.Sprintf("changelog.payload->>'type' = ANY($%d)", len(args)+1))
		args = append(args, find.TypeList)
	}

	truncateSize := 512
	if common.IsDev() {
		truncateSize = 4
	}
	shPreField := fmt.Sprintf("LEFT(sh_pre.raw_dump, %d)", truncateSize)
	if find.ShowFull {
		shPreField = "sh_pre.raw_dump"
	}
	shCurField := fmt.Sprintf("LEFT(sh_cur.raw_dump, %d)", truncateSize)
	if find.ShowFull {
		shCurField = "sh_cur.raw_dump"
	}
	sheetField := fmt.Sprintf("LEFT(sheet_blob.content, %d)", truncateSize)
	if find.ShowFull {
		sheetField = "sheet_blob.content"
	}

	query := fmt.Sprintf(`
		SELECT
			changelog.id,
			changelog.creator_id,
			changelog.created_ts,
			changelog.database_id,
			changelog.status,
			changelog.prev_sync_history_id,
			changelog.sync_history_id,
			COALESCE(%s, ''),
			COALESCE(%s, ''),
			COALESCE(%s, ''),
			COALESCE(OCTET_LENGTH(sheet_blob.content), 0),
			changelog.payload
		FROM changelog
		LEFT JOIN sync_history sh_pre ON sh_pre.id = changelog.prev_sync_history_id
		LEFT JOIN sync_history sh_cur ON sh_cur.id = changelog.sync_history_id
		LEFT JOIN sheet ON sheet.id::text = split_part(changelog.payload->>'sheet', '/', 4)
		LEFT JOIN sheet_blob ON sheet.sha256 = sheet_blob.sha256
		WHERE %s
		ORDER BY changelog.id DESC
	`,
		shPreField,
		shCurField,
		sheetField,
		strings.Join(where, " AND "))
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
			&c.Status,
			&c.PrevSyncHistoryUID,
			&c.SyncHistoryUID,
			&c.PrevSchema,
			&c.Schema,
			&c.Statement,
			&c.StatementSize,
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
