package store

import (
	"context"
	"database/sql"
	"strings"

	"github.com/pkg/errors"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/qb"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

// DatabaseMessage is the message for database.
type DatabaseMessage struct {
	ProjectID    string
	InstanceID   string
	DatabaseName string

	EnvironmentID          *string
	EffectiveEnvironmentID *string

	Deleted  bool
	Metadata *storepb.DatabaseMetadata
}

func (d *DatabaseMessage) String() string {
	return common.FormatDatabase(d.InstanceID, d.DatabaseName)
}

// UpdateDatabaseMessage is the mssage for updating a database.
type UpdateDatabaseMessage struct {
	InstanceID   string
	DatabaseName string

	ProjectID *string
	Deleted   *bool
	// Empty string will unset the environment.
	EnvironmentID   *string
	MetadataUpdates []func(*storepb.DatabaseMetadata)
}

// BatchUpdateDatabases is the message for batch updating databases.
type BatchUpdateDatabases struct {
	ProjectID           *string
	FindByEnvironmentID *string
	// Empty string will unset the environment.
	EnvironmentID *string
}

// FindDatabaseMessage is the message for finding databases.
type FindDatabaseMessage struct {
	ProjectID              *string
	EffectiveEnvironmentID *string
	InstanceID             *string
	DatabaseName           *string
	Engine                 *storepb.Engine
	// When this is used, we will return databases from archived instances or environments.
	// This is used for existing tasks with archived databases.
	ShowDeleted bool

	// IsCaseSensitive is used to ignore case sensitive when finding database.
	IsCaseSensitive bool

	Filter *ListResourceFilter
	Limit  *int
	Offset *int
}

// GetDatabaseV2 gets a database.
func (s *Store) GetDatabaseV2(ctx context.Context, find *FindDatabaseMessage) (*DatabaseMessage, error) {
	if find.InstanceID != nil && find.DatabaseName != nil {
		if v, ok := s.databaseCache.Get(getDatabaseCacheKey(*find.InstanceID, *find.DatabaseName)); ok && s.enableCache {
			return v, nil
		}
	}

	tx, err := s.GetDB().BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	databases, err := s.listDatabaseImplV2(ctx, tx, find)
	if err != nil {
		return nil, err
	}
	if len(databases) == 0 {
		return nil, nil
	}
	if len(databases) > 1 {
		return nil, &common.Error{Code: common.Conflict, Err: errors.Errorf("found %d database with filter %+v, expect 1", len(databases), find)}
	}
	database := databases[0]

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	s.databaseCache.Add(getDatabaseCacheKey(database.InstanceID, database.DatabaseName), database)
	return database, nil
}

// ListDatabases lists all databases.
func (s *Store) ListDatabases(ctx context.Context, find *FindDatabaseMessage) ([]*DatabaseMessage, error) {
	tx, err := s.GetDB().BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	databases, err := s.listDatabaseImplV2(ctx, tx, find)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	for _, database := range databases {
		s.databaseCache.Add(getDatabaseCacheKey(database.InstanceID, database.DatabaseName), database)
	}
	return databases, nil
}

// CreateDatabaseDefault creates a new database in the default project.
func (s *Store) CreateDatabaseDefault(ctx context.Context, create *DatabaseMessage) (*DatabaseMessage, error) {
	tx, err := s.GetDB().BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	if _, err := s.createDatabaseDefaultImpl(ctx, tx, create.ProjectID, create.InstanceID, create); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	// Invalidate an update the cache.
	s.databaseCache.Remove(getDatabaseCacheKey(create.InstanceID, create.DatabaseName))
	return s.GetDatabaseV2(ctx, &FindDatabaseMessage{InstanceID: &create.InstanceID, DatabaseName: &create.DatabaseName, ShowDeleted: true})
}

// createDatabaseDefault only creates a default database with charset, collation only in the default project.
func (*Store) createDatabaseDefaultImpl(ctx context.Context, txn *sql.Tx, projectID, instanceID string, create *DatabaseMessage) (int, error) {
	q := qb.Q().Space(`
		INSERT INTO db (
			instance,
			project,
			name,
			deleted
		)
		VALUES (?, ?, ?, ?)
		ON CONFLICT (instance, name) DO UPDATE SET
			deleted = EXCLUDED.deleted
		RETURNING id`,
		instanceID,
		projectID,
		create.DatabaseName,
		false,
	)

	query, args, err := q.ToSQL()
	if err != nil {
		return 0, errors.Wrapf(err, "failed to build sql")
	}

	var databaseUID int
	if err := txn.QueryRowContext(ctx, query, args...).Scan(&databaseUID); err != nil {
		return 0, err
	}
	return databaseUID, nil
}

// UpsertDatabase upserts a database.
func (s *Store) UpsertDatabase(ctx context.Context, create *DatabaseMessage) (*DatabaseMessage, error) {
	metadataString, err := protojson.Marshal(create.Metadata)
	if err != nil {
		return nil, err
	}

	tx, err := s.GetDB().BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	var environment *string
	if create.EnvironmentID != nil && *create.EnvironmentID != "" {
		environment = create.EnvironmentID
	}

	q := qb.Q().Space(`
		INSERT INTO db (
			instance,
			project,
			environment,
			name,
			deleted,
			metadata
		)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT (instance, name) DO UPDATE SET
			project = EXCLUDED.project,
			environment = EXCLUDED.environment,
			name = EXCLUDED.name,
			metadata = EXCLUDED.metadata
		RETURNING id`,
		create.InstanceID,
		create.ProjectID,
		environment,
		create.DatabaseName,
		create.Deleted,
		metadataString,
	)

	query, args, err := q.ToSQL()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to build sql")
	}

	var databaseUID int
	if err := tx.QueryRowContext(ctx, query, args...).Scan(&databaseUID); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}

	// Invalidate and update the cache.
	s.databaseCache.Remove(getDatabaseCacheKey(create.InstanceID, create.DatabaseName))
	return s.GetDatabaseV2(ctx, &FindDatabaseMessage{InstanceID: &create.InstanceID, DatabaseName: &create.DatabaseName, ShowDeleted: true})
}

// UpdateDatabase updates a database.
func (s *Store) UpdateDatabase(ctx context.Context, patch *UpdateDatabaseMessage) (*DatabaseMessage, error) {
	set := qb.Q()

	if v := patch.ProjectID; v != nil {
		set.Comma("project = ?", *v)
	}
	if v := patch.EnvironmentID; v != nil {
		if *v == "" {
			set.Comma("environment = NULL")
		} else {
			set.Comma("environment = ?", *v)
		}
	}
	if v := patch.Deleted; v != nil {
		set.Comma("deleted = ?", *v)
	}
	if fs := patch.MetadataUpdates; len(fs) > 0 {
		database, err := s.GetDatabaseV2(ctx, &FindDatabaseMessage{
			InstanceID:   &patch.InstanceID,
			DatabaseName: &patch.DatabaseName,
		})
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get database %q", common.FormatDatabase(patch.InstanceID, patch.DatabaseName))
		}
		md := proto.CloneOf(database.Metadata)
		for _, f := range fs {
			f(md)
		}
		metadataBytes, err := protojson.Marshal(md)
		if err != nil {
			return nil, err
		}
		set.Comma("metadata = ?", metadataBytes)
	}

	if set.Len() == 0 {
		return nil, errors.New("no fields to update")
	}

	q := qb.Q().Space("UPDATE db SET ?", set).
		Space("WHERE instance = ? AND name = ? RETURNING id", patch.InstanceID, patch.DatabaseName)

	query, args, err := q.ToSQL()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to build sql")
	}

	tx, err := s.GetDB().BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	var databaseUID int
	if err := tx.QueryRowContext(ctx, query, args...).Scan(&databaseUID); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}

	// Invalidate and update database cache.
	s.databaseCache.Remove(getDatabaseCacheKey(patch.InstanceID, patch.DatabaseName))
	return s.GetDatabaseV2(ctx, &FindDatabaseMessage{InstanceID: &patch.InstanceID, DatabaseName: &patch.DatabaseName, ShowDeleted: true})
}

// BatchUpdateDatabases update databases in batch.
func (s *Store) BatchUpdateDatabases(ctx context.Context, databases []*DatabaseMessage, update *BatchUpdateDatabases) ([]*DatabaseMessage, error) {
	set := qb.Q()

	if update.ProjectID != nil {
		set.Comma("project = ?", *update.ProjectID)
	}
	if v := update.EnvironmentID; v != nil {
		if *v == "" {
			set.Comma("environment = NULL")
		} else {
			set.Comma("environment = ?", *v)
		}
	}
	if set.Len() == 0 {
		return nil, errors.New("no update field specified")
	}

	where := qb.Q()

	if v := update.FindByEnvironmentID; v != nil {
		where.Or("environment = ?", *v)
	}

	for _, database := range databases {
		where.Or("(db.instance = ? AND db.name = ?)", database.InstanceID, database.DatabaseName)
	}
	if where.Len() == 0 {
		return nil, errors.Errorf("empty where")
	}

	q := qb.Q().Space("UPDATE db SET ?", set).
		Space("WHERE ?", where)

	query, args, err := q.ToSQL()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to build sql")
	}

	tx, err := s.GetDB().BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, query, args...); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	var updatedDatabases []*DatabaseMessage
	for _, database := range databases {
		updatedDatabase := *database
		// Update cache for project field.
		if update.ProjectID != nil {
			updatedDatabase.ProjectID = *update.ProjectID
		}
		// Update cache for environment field and effective environment field.
		if update.EnvironmentID != nil {
			updatedDatabase.EnvironmentID = update.EnvironmentID
			if *update.EnvironmentID == "" {
				instance, err := s.GetInstanceV2(ctx, &FindInstanceMessage{ResourceID: &database.InstanceID})
				if err != nil {
					// Should not reach here.
					return nil, err
				}
				updatedDatabase.EffectiveEnvironmentID = instance.EnvironmentID
			} else {
				updatedDatabase.EffectiveEnvironmentID = update.EnvironmentID
			}
		}
		s.databaseCache.Add(getDatabaseCacheKey(database.InstanceID, database.DatabaseName), &updatedDatabase)
		updatedDatabases = append(updatedDatabases, &updatedDatabase)
	}
	return updatedDatabases, nil
}

func (*Store) listDatabaseImplV2(ctx context.Context, txn *sql.Tx, find *FindDatabaseMessage) ([]*DatabaseMessage, error) {
	q := qb.Q().Space(`
		SELECT
			db.project,
			COALESCE(
				db.environment,
				instance.environment
			),
			db.environment,
			db.instance,
			db.name,
			db.deleted,
			db.metadata
		FROM db`)

	joinQuery := ""
	if filter := find.Filter; filter != nil {
		if strings.Contains(filter.Where, "ds.metadata->'schemas'") {
			joinQuery = "INNER JOIN db_schema ds ON db.instance = ds.instance AND db.name = ds.db_name"
		}
	}
	if joinQuery != "" {
		q.Space(joinQuery)
	}

	q.Space("LEFT JOIN instance ON db.instance = instance.resource_id WHERE TRUE")

	if filter := find.Filter; filter != nil {
		q.And(filter.Where, filter.Args...)
	}
	if v := find.ProjectID; v != nil {
		q.And("db.project = ?", *v)
	}
	if v := find.EffectiveEnvironmentID; v != nil {
		q.And(`COALESCE(
			db.environment,
			instance.environment
		) = ?`, *v)
	}
	if v := find.InstanceID; v != nil {
		q.And("db.instance = ?", *v)
	}
	if v := find.DatabaseName; v != nil {
		if find.IsCaseSensitive {
			q.And("db.name = ?", *v)
		} else {
			q.And("LOWER(db.name) = LOWER(?)", *v)
		}
	}
	if v := find.Engine; v != nil {
		q.And("instance.metadata->>'engine' = ?", *v)
	}
	if !find.ShowDeleted {
		q.And("instance.deleted = ?", false)
		q.And("db.deleted = ?", false)
	}

	q.Space("ORDER BY db.project, db.instance, db.name")

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

	var databaseMessages []*DatabaseMessage
	rows, err := txn.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		databaseMessage := &DatabaseMessage{}
		var metadataString string
		var effectiveEnvironment, environment sql.NullString
		if err := rows.Scan(
			&databaseMessage.ProjectID,
			&effectiveEnvironment,
			&environment,
			&databaseMessage.InstanceID,
			&databaseMessage.DatabaseName,
			&databaseMessage.Deleted,
			&metadataString,
		); err != nil {
			return nil, err
		}
		if effectiveEnvironment.Valid {
			databaseMessage.EffectiveEnvironmentID = &effectiveEnvironment.String
		}
		if environment.Valid {
			databaseMessage.EnvironmentID = &environment.String
		}

		var metadata storepb.DatabaseMetadata
		if err := common.ProtojsonUnmarshaler.Unmarshal([]byte(metadataString), &metadata); err != nil {
			return nil, err
		}
		databaseMessage.Metadata = &metadata

		databaseMessages = append(databaseMessages, databaseMessage)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return databaseMessages, nil
}
