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

type RevisionMessage struct {
	InstanceID   string
	DatabaseName string
	Version      string
	Payload      *storepb.RevisionPayload

	// output only
	UID       int64
	CreatedAt time.Time
	Deleter   *string
	DeletedAt *time.Time
}

type FindRevisionMessage struct {
	UID          *int64
	InstanceID   *string
	DatabaseName *string
	Type         *storepb.SchemaChangeType

	Version  *string
	Versions *[]string

	Limit  *int
	Offset *int

	ShowDeleted bool
}

// ListRevisions lists revisions.
// The results are ordered by version desc.
func (s *Store) ListRevisions(ctx context.Context, find *FindRevisionMessage) ([]*RevisionMessage, error) {
	q := qb.Q().Space(`
		SELECT
			id,
			instance,
			db_name,
			created_at,
			deleter,
			deleted_at,
			version,
			payload
		FROM revision
		WHERE TRUE
	`)

	if v := find.UID; v != nil {
		q.And("id = ?", *v)
	}
	if v := find.InstanceID; v != nil {
		q.And("instance = ?", *v)
	}
	if v := find.DatabaseName; v != nil {
		q.And("db_name = ?", *v)
	}
	if v := find.Type; v != nil {
		q.And("payload->>'type' = ?", v.String())
	}
	if v := find.Version; v != nil {
		q.And("version = ?", *v)
	}
	if v := find.Versions; v != nil {
		q.And("version = ANY(?)", *v)
	}
	if !find.ShowDeleted {
		q.And("deleted_at IS NULL")
	}
	q.Space("ORDER BY version DESC")
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
			&r.InstanceID,
			&r.DatabaseName,
			&r.CreatedAt,
			&r.Deleter,
			&r.DeletedAt,
			&r.Version,
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

func (s *Store) GetRevision(ctx context.Context, uid int64, instanceID, databaseName string) (*RevisionMessage, error) {
	revisions, err := s.ListRevisions(ctx, &FindRevisionMessage{
		UID:          &uid,
		InstanceID:   &instanceID,
		DatabaseName: &databaseName,
		ShowDeleted:  true,
	})
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

func (s *Store) CreateRevision(ctx context.Context, revision *RevisionMessage) (*RevisionMessage, error) {
	query := `
		INSERT INTO revision (
			instance,
			db_name,
			version,
			payload
		) VALUES (
		 	$1,
			$2,
			$3,
			$4
		)
		RETURNING id, created_at
	`

	p, err := protojson.Marshal(revision.Payload)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to marshal revision payload")
	}

	tx, err := s.GetDB().BeginTx(ctx, nil)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to begin tx")
	}
	defer tx.Rollback()

	var id int64
	if err := tx.QueryRowContext(ctx, query,
		revision.InstanceID,
		revision.DatabaseName,
		revision.Version,
		p,
	).Scan(&id, &revision.CreatedAt); err != nil {
		return nil, errors.Wrapf(err, "failed to query and scan")
	}

	if err := tx.Commit(); err != nil {
		return nil, errors.Wrapf(err, "failed to commit tx")
	}

	revision.UID = id

	return revision, nil
}

func (s *Store) DeleteRevision(ctx context.Context, uid int64, instanceID, databaseName string, deleter string) error {
	query :=
		`UPDATE revision
		SET deleter = $1, deleted_at = now()
		WHERE id = $2 AND instance = $3 AND db_name = $4`

	tx, err := s.GetDB().BeginTx(ctx, nil)
	if err != nil {
		return errors.Wrapf(err, "failed to begin tx")
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, query, deleter, uid, instanceID, databaseName); err != nil {
		return errors.Wrapf(err, "failed to exec")
	}

	if err := tx.Commit(); err != nil {
		return errors.Wrapf(err, "failed to commit tx")
	}

	return nil
}
