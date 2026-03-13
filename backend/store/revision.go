package store

import (
	"context"
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
	ResourceID string
	CreatedAt  time.Time
	Deleter    *string
	DeletedAt  *time.Time
}

type FindRevisionMessage struct {
	InstanceID   string
	ResourceID   *string
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
			resource_id,
			instance,
			db_name,
			created_at,
			deleter,
			deleted_at,
			version,
			payload
		FROM revision
		WHERE instance = ?
	`, find.InstanceID)

	if v := find.ResourceID; v != nil {
		q.And("resource_id = ?", *v)
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

	rows, err := s.GetDB().QueryContext(ctx, query, args...)
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
			&r.ResourceID,
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

	return revisions, nil
}

func (s *Store) GetRevision(ctx context.Context, resourceID, instanceID, databaseName string) (*RevisionMessage, error) {
	revisions, err := s.ListRevisions(ctx, &FindRevisionMessage{
		InstanceID:   instanceID,
		ResourceID:   &resourceID,
		DatabaseName: &databaseName,
		ShowDeleted:  true,
	})
	if err != nil {
		return nil, err
	}
	if len(revisions) == 0 {
		return nil, errors.Errorf("revision not found: %s", resourceID)
	}
	if len(revisions) > 1 {
		return nil, errors.Errorf("found multiple revisions for resource_id: %s", resourceID)
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
		RETURNING resource_id, created_at
	`

	p, err := protojson.Marshal(revision.Payload)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to marshal revision payload")
	}

	if err := s.GetDB().QueryRowContext(ctx, query,
		revision.InstanceID,
		revision.DatabaseName,
		revision.Version,
		p,
	).Scan(&revision.ResourceID, &revision.CreatedAt); err != nil {
		return nil, errors.Wrapf(err, "failed to query and scan")
	}

	return revision, nil
}

func (s *Store) DeleteRevision(ctx context.Context, resourceID, instanceID, databaseName string, deleter string) error {
	query :=
		`UPDATE revision
		SET deleter = $1, deleted_at = now()
		WHERE resource_id = $2 AND instance = $3 AND db_name = $4`

	if _, err := s.GetDB().ExecContext(ctx, query, deleter, resourceID, instanceID, databaseName); err != nil {
		return errors.Wrapf(err, "failed to exec")
	}

	return nil
}
