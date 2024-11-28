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
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

type RevisionMessage struct {
	DatabaseUID int
	Version     string
	Payload     *storepb.RevisionPayload

	// output only
	UID         int64
	CreatorUID  int
	CreatedTime time.Time
	DeleterUID  *int
	DeletedTime *time.Time
}

type FindRevisionMessage struct {
	UID         *int64
	DatabaseUID *int

	Version *string

	Limit  *int
	Offset *int

	ShowDeleted bool
}

func (s *Store) ListRevisions(ctx context.Context, find *FindRevisionMessage) ([]*RevisionMessage, error) {
	where, args := []string{"TRUE"}, []any{}

	if v := find.UID; v != nil {
		where = append(where, fmt.Sprintf("id = $%d", len(args)+1))
		args = append(args, *v)
	}
	if v := find.DatabaseUID; v != nil {
		where = append(where, fmt.Sprintf("database_id = $%d", len(args)+1))
		args = append(args, *v)
	}
	if v := find.Version; v != nil {
		where = append(where, fmt.Sprintf("version = $%d", len(args)+1))
		args = append(args, *v)
	}
	if !find.ShowDeleted {
		where = append(where, "deleted_ts IS NULL")
	}

	limitOffsetClause := ""
	if v := find.Limit; v != nil {
		limitOffsetClause += fmt.Sprintf(" LIMIT %d", *v)
	}
	if v := find.Offset; v != nil {
		limitOffsetClause += fmt.Sprintf(" OFFSET %d", *v)
	}

	query := fmt.Sprintf(`
		SELECT
			id,
			database_id,
			creator_id,
			created_ts,
			deleter_id,
			deleted_ts,
			payload
		FROM revision
		WHERE %s
		ORDER BY version DESC
		%s
	`, strings.Join(where, " AND "), limitOffsetClause)

	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to begin tx")
	}
	defer tx.Rollback()

	rows, err := tx.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to query context")
	}
	defer rows.Close()

	var revisions []*RevisionMessage
	for rows.Next() {
		r := RevisionMessage{
			Payload: &storepb.RevisionPayload{},
		}
		var p []byte
		if err := rows.Scan(
			&r.UID,
			&r.DatabaseUID,
			&r.CreatorUID,
			&r.CreatedTime,
			&r.DeleterUID,
			&r.DeletedTime,
			&p,
		); err != nil {
			return nil, errors.Wrapf(err, "failed to scan")
		}

		if err := common.ProtojsonUnmarshaler.Unmarshal(p, r.Payload); err != nil {
			return nil, errors.Wrapf(err, "failed to unmarshal")
		}

		revisions = append(revisions, &r)
	}

	if err := rows.Err(); err != nil {
		return nil, errors.Wrapf(err, "rows err")
	}

	if err := tx.Commit(); err != nil {
		return nil, errors.Wrapf(err, "failed to commit tx")
	}

	return revisions, nil
}

func (s *Store) GetRevision(ctx context.Context, uid int64) (*RevisionMessage, error) {
	revisions, err := s.ListRevisions(ctx, &FindRevisionMessage{UID: &uid, ShowDeleted: true})
	if err != nil {
		return nil, err
	}
	if len(revisions) == 0 {
		return nil, errors.Errorf("revision not found: %d", uid)
	}
	if len(revisions) > 1 {
		return nil, errors.Errorf("found multiple revisions for uid: %d", uid)
	}
	return revisions[0], nil
}

func (s *Store) CreateRevision(ctx context.Context, revision *RevisionMessage, creatorUID int) (*RevisionMessage, error) {
	query := `
		INSERT INTO revision (
			database_id,
			creator_id,
			version,
			payload
		) VALUES (
		 	$1,
			$2,
			$3,
			$4
		)
		RETURNING id, created_ts
	`

	p, err := protojson.Marshal(revision.Payload)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to marshal revision payload")
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to begin tx")
	}
	defer tx.Rollback()

	var id int64
	var createdTime time.Time
	if err := tx.QueryRowContext(ctx, query,
		revision.DatabaseUID,
		creatorUID,
		revision.Version,
		p,
	).Scan(&id, &createdTime); err != nil {
		return nil, errors.Wrapf(err, "failed to query and scan")
	}

	if err := tx.Commit(); err != nil {
		return nil, errors.Wrapf(err, "failed to commit tx")
	}

	revision.UID = id
	revision.CreatedTime = createdTime

	return revision, nil
}

func (s *Store) DeleteRevision(ctx context.Context, uid int64, deleterUID int) error {
	query :=
		`UPDATE revision
		SET deleter_id = $1, deleted_ts = now()
		WHERE id = $2`

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return errors.Wrapf(err, "failed to begin tx")
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, query, deleterUID, uid); err != nil {
		return errors.Wrapf(err, "failed to exec")
	}

	if err := tx.Commit(); err != nil {
		return errors.Wrapf(err, "failed to commit tx")
	}

	return nil
}
