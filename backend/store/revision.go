package store

import (
	"context"
	"database/sql"
	"time"

	"github.com/pkg/errors"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/store/qb"
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

// PendingReleaseVersion is a release version a database has not applied: it
// has no live VERSIONED revision with that version.
type PendingReleaseVersion struct {
	DatabaseName string
	Version      string
	// The newest deleted VERSIONED revision of this version, when one exists.
	DeletedAt *time.Time
	Deleter   string
}

// ListPendingReleaseVersions returns, for the given databases of one instance,
// each of versions that the database has not applied, by the executor's rule
// (runVersionedRelease): only a live VERSIONED revision counts. It returns
// only the pending pairs, never one per applied revision.
func (s *Store) ListPendingReleaseVersions(ctx context.Context, instanceID string, databaseNames, versions []string) ([]*PendingReleaseVersion, error) {
	if len(databaseNames) == 0 || len(versions) == 0 {
		return nil, nil
	}
	versioned := storepb.SchemaChangeType_VERSIONED.String()
	q := qb.Q().Space(`
		SELECT target.name, release_version.version, deleted.deleted_at, COALESCE(deleted.deleter, '')
		FROM unnest(?::TEXT[]) AS target(name)
		CROSS JOIN unnest(?::TEXT[]) AS release_version(version)
		LEFT JOIN LATERAL (
			SELECT revision.deleted_at, revision.deleter
			FROM revision
			WHERE revision.instance = ?
				AND revision.db_name = target.name
				AND revision.payload->>'type' = ?
				AND revision.version = release_version.version
				AND revision.deleted_at IS NOT NULL
			ORDER BY revision.deleted_at DESC, revision.resource_id DESC
			LIMIT 1
		) AS deleted ON TRUE
		WHERE NOT EXISTS (
			SELECT 1
			FROM revision
			WHERE revision.instance = ?
				AND revision.db_name = target.name
				AND revision.payload->>'type' = ?
				AND revision.version = release_version.version
				AND revision.deleted_at IS NULL
		)`, databaseNames, versions, instanceID, versioned, instanceID, versioned)
	query, args, err := q.ToSQL()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to build sql")
	}
	rows, err := s.GetDB().QueryContext(ctx, query, args...)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to query pending release versions")
	}
	defer rows.Close()

	var pending []*PendingReleaseVersion
	for rows.Next() {
		p := &PendingReleaseVersion{}
		if err := rows.Scan(&p.DatabaseName, &p.Version, &p.DeletedAt, &p.Deleter); err != nil {
			return nil, errors.Wrapf(err, "failed to scan pending release version")
		}
		pending = append(pending, p)
	}
	if err := rows.Err(); err != nil {
		return nil, errors.Wrapf(err, "failed to read pending release versions")
	}
	return pending, nil
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
	// version is unique only per (instance, db_name, type) and only while
	// deleted_at IS NULL, so it does not identify a row on its own — least of
	// all under ShowDeleted. resource_id is the primary key.
	q.Space("ORDER BY revision.version DESC, revision.resource_id DESC")
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
		return nil, common.Errorf(common.NotFound, "revision not found: %s", resourceID)
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

	err = s.withDatabaseWrite(ctx, revision.InstanceID, revision.DatabaseName, nil, func(tx *sql.Tx, _ *databaseOwnership) error {
		return tx.QueryRowContext(ctx, query,
			revision.InstanceID,
			revision.DatabaseName,
			revision.Version,
			p,
		).Scan(&revision.ResourceID, &revision.CreatedAt)
	})
	if err != nil {
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
