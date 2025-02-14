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

// DatabaseMessage is the message for database.
type DatabaseMessage struct {
	UID                    int
	ProjectID              string
	InstanceID             string
	EnvironmentID          string
	EffectiveEnvironmentID string

	DatabaseName string
	SyncState    api.SyncStatus
	SyncAt       time.Time
	Secrets      *storepb.Secrets
	DataShare    bool
	// ServiceName is the Oracle specific field.
	ServiceName string
	Metadata    *storepb.DatabaseMetadata
	// Output only
	SchemaVersion string
}

// UpdateDatabaseMessage is the mssage for updating a database.
type UpdateDatabaseMessage struct {
	InstanceID   string
	DatabaseName string

	ProjectID           *string
	SyncState           *api.SyncStatus
	SyncAt              *time.Time
	SourceBackupID      *int
	Secrets             *storepb.Secrets
	DataShare           *bool
	ServiceName         *string
	UpdateEnvironmentID bool
	EnvironmentID       string

	// MetadataUpsert upserts the top-level messages.
	MetadataUpsert *storepb.DatabaseMetadata
}

// FindDatabaseMessage is the message for finding databases.
type FindDatabaseMessage struct {
	ProjectID              *string
	EffectiveEnvironmentID *string
	InstanceID             *string
	DatabaseName           *string
	UID                    *int
	Engine                 *storepb.Engine
	// When this is used, we will return databases from archived instances or environments.
	// This is used for existing tasks with archived databases.
	ShowDeleted bool

	// IgnoreCaseSensitive is used to ignore case sensitive when finding database.
	IgnoreCaseSensitive bool

	Limit  *int
	Offset *int
}

// GetDatabaseV2 gets a database.
func (s *Store) GetDatabaseV2(ctx context.Context, find *FindDatabaseMessage) (*DatabaseMessage, error) {
	if find.InstanceID != nil && find.DatabaseName != nil {
		if v, ok := s.databaseCache.Get(getDatabaseCacheKey(*find.InstanceID, *find.DatabaseName)); ok {
			return v, nil
		}
	}
	if find.UID != nil {
		if v, ok := s.databaseIDCache.Get(*find.UID); ok {
			return v, nil
		}
	}

	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
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
	s.databaseIDCache.Add(database.UID, database)
	return database, nil
}

// ListDatabases lists all databases.
func (s *Store) ListDatabases(ctx context.Context, find *FindDatabaseMessage) ([]*DatabaseMessage, error) {
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
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
		s.databaseIDCache.Add(database.UID, database)
	}
	return databases, nil
}

// CreateDatabaseDefault creates a new database in the default project.
func (s *Store) CreateDatabaseDefault(ctx context.Context, create *DatabaseMessage) (*DatabaseMessage, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	databaseUID, err := s.createDatabaseDefaultImpl(ctx, tx, create.ProjectID, create.InstanceID, create)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	// Invalidate an update the cache.
	s.databaseCache.Remove(getDatabaseCacheKey(create.InstanceID, create.DatabaseName))
	s.databaseIDCache.Remove(databaseUID)
	return s.GetDatabaseV2(ctx, &FindDatabaseMessage{UID: &databaseUID})
}

// createDatabaseDefault only creates a default database with charset, collation only in the default project.
func (*Store) createDatabaseDefaultImpl(ctx context.Context, tx *Tx, projectID, instanceID string, create *DatabaseMessage) (int, error) {
	secretsString, err := protojson.Marshal(&storepb.Secrets{})
	if err != nil {
		return 0, err
	}

	query := `
		INSERT INTO db (
			instance,
			project,
			name,
			sync_status,
			schema_version,
			secrets,
			datashare,
			service_name
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (instance, name) DO UPDATE SET
			project = EXCLUDED.project,
			sync_status = EXCLUDED.sync_status,
			datashare = EXCLUDED.datashare,
			service_name = EXCLUDED.service_name
		RETURNING id`
	var databaseUID int
	if err := tx.QueryRowContext(ctx, query,
		instanceID,
		projectID,
		create.DatabaseName,
		api.OK,
		"",            /* schema_version */
		secretsString, /* secrets */
		create.DataShare,
		create.ServiceName,
	).Scan(
		&databaseUID,
	); err != nil {
		return 0, err
	}
	return databaseUID, nil
}

// UpsertDatabase upserts a database.
func (s *Store) UpsertDatabase(ctx context.Context, create *DatabaseMessage) (*DatabaseMessage, error) {
	secretsString, err := protojson.Marshal(create.Secrets)
	if err != nil {
		return nil, err
	}
	metadataString, err := protojson.Marshal(create.Metadata)
	if err != nil {
		return nil, err
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	var environment *string
	if create.EnvironmentID != "" {
		environment = &create.EnvironmentID
	}
	query := `
		INSERT INTO db (
			instance,
			project,
			environment,
			name,
			sync_status,
			sync_at,
			schema_version,
			secrets,
			datashare,
			service_name,
			metadata
		)
		VALUES ($1, $2, $3, $4, $5, $6, '', $7, $8, $9, $10)
		ON CONFLICT (instance, name) DO UPDATE SET
			project = EXCLUDED.project,
			environment = EXCLUDED.environment,
			name = EXCLUDED.name,
			schema_version = EXCLUDED.schema_version,
			metadata = EXCLUDED.metadata
		RETURNING id`
	var databaseUID int
	if err := tx.QueryRowContext(ctx, query,
		create.InstanceID,
		create.ProjectID,
		environment,
		create.DatabaseName,
		create.SyncState,
		create.SyncAt,
		secretsString,
		create.DataShare,
		create.ServiceName,
		metadataString,
	).Scan(
		&databaseUID,
	); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}

	// Invalidate and update the cache.
	s.databaseCache.Remove(getDatabaseCacheKey(create.InstanceID, create.DatabaseName))
	s.databaseIDCache.Remove(databaseUID)
	return s.GetDatabaseV2(ctx, &FindDatabaseMessage{UID: &databaseUID, ShowDeleted: true})
}

// UpdateDatabase updates a database.
func (s *Store) UpdateDatabase(ctx context.Context, patch *UpdateDatabaseMessage) (*DatabaseMessage, error) {
	set, args := []string{}, []any{}
	if v := patch.ProjectID; v != nil {
		set, args = append(set, fmt.Sprintf("project = $%d", len(args)+1)), append(args, *v)
	}
	if patch.UpdateEnvironmentID {
		var environment *string
		if patch.EnvironmentID != "" {
			environment = &patch.EnvironmentID
		}
		set, args = append(set, fmt.Sprintf("environment = $%d", len(args)+1)), append(args, environment)
	}
	if v := patch.SyncState; v != nil {
		set, args = append(set, fmt.Sprintf("sync_status = $%d", len(args)+1)), append(args, *v)
	}
	if v := patch.SyncAt; v != nil {
		set, args = append(set, fmt.Sprintf("sync_at = $%d", len(args)+1)), append(args, *v)
	}
	if v := patch.Secrets; v != nil {
		secretsString, err := protojson.Marshal(v)
		if err != nil {
			return nil, err
		}
		set, args = append(set, fmt.Sprintf("secrets = $%d", len(args)+1)), append(args, secretsString)
	}
	if v := patch.DataShare; v != nil {
		set, args = append(set, fmt.Sprintf("datashare = $%d", len(args)+1)), append(args, *v)
	}
	if v := patch.ServiceName; v != nil {
		set, args = append(set, fmt.Sprintf("service_name = $%d", len(args)+1)), append(args, *v)
	}
	if v := patch.MetadataUpsert; v != nil {
		metadataBytes, err := protojson.Marshal(v)
		if err != nil {
			return nil, err
		}
		if v.Labels != nil && len(v.Labels) == 0 {
			set, args = append(set, fmt.Sprintf("metadata = metadata || $%d || $%d", len(args)+1, len(args)+2)), append(args, metadataBytes, `{"labels": {}}`)
		} else {
			set, args = append(set, fmt.Sprintf("metadata = metadata || $%d", len(args)+1)), append(args, metadataBytes)
		}
	}
	args = append(args, patch.InstanceID, patch.DatabaseName)

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	var databaseUID int
	if err := tx.QueryRowContext(ctx, fmt.Sprintf(`
		UPDATE db
		SET `+strings.Join(set, ", ")+`
		WHERE instance = $%d AND name = $%d
		RETURNING id
	`, len(args)-1, len(args)),
		args...,
	).Scan(
		&databaseUID,
	); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}

	// Invalidate and update database cache.
	s.databaseCache.Remove(getDatabaseCacheKey(patch.InstanceID, patch.DatabaseName))
	s.databaseIDCache.Remove(databaseUID)
	return s.GetDatabaseV2(ctx, &FindDatabaseMessage{UID: &databaseUID, ShowDeleted: true})
}

// BatchUpdateDatabaseProject updates the project for databases in batch.
func (s *Store) BatchUpdateDatabaseProject(ctx context.Context, databases []*DatabaseMessage, projectID string) ([]*DatabaseMessage, error) {
	if len(databases) == 0 {
		return nil, errors.Errorf("there is no database in the project")
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	var wheres []string
	args := []any{projectID}
	for i, database := range databases {
		wheres = append(wheres, fmt.Sprintf("(db.instance = $%d AND db.name = $%d)", 2*i+2, 2*i+3))
		args = append(args, database.InstanceID, database.DatabaseName)
	}
	if _, err := tx.ExecContext(ctx, fmt.Sprintf(`
			UPDATE db
			SET project = $1
			WHERE %s;`, strings.Join(wheres, " OR ")),
		args...,
	); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	var updatedDatabases []*DatabaseMessage
	for _, database := range databases {
		updatedDatabase := *database
		updatedDatabase.ProjectID = projectID
		s.databaseCache.Add(getDatabaseCacheKey(database.InstanceID, database.DatabaseName), &updatedDatabase)
		s.databaseIDCache.Add(database.UID, &updatedDatabase)
		updatedDatabases = append(updatedDatabases, &updatedDatabase)
	}
	return updatedDatabases, nil
}

func (*Store) listDatabaseImplV2(ctx context.Context, tx *Tx, find *FindDatabaseMessage) ([]*DatabaseMessage, error) {
	where, args := []string{"TRUE"}, []any{}
	if v := find.ProjectID; v != nil {
		where, args = append(where, fmt.Sprintf("db.project = $%d", len(args)+1)), append(args, *v)
	}
	if v := find.EffectiveEnvironmentID; v != nil {
		where, args = append(where, fmt.Sprintf(`
		COALESCE(
			(SELECT environment.resource_id FROM environment where environment.resource_id = db.environment),
			(SELECT environment.resource_id FROM environment JOIN instance ON environment.resource_id = instance.environment WHERE instance.resource_id = db.instance)
		) = $%d`, len(args)+1)), append(args, *v)
	}
	if v := find.InstanceID; v != nil {
		where, args = append(where, fmt.Sprintf("db.instance = $%d", len(args)+1)), append(args, *v)
	}
	if v := find.DatabaseName; v != nil {
		if find.IgnoreCaseSensitive {
			where, args = append(where, fmt.Sprintf("LOWER(db.name) = LOWER($%d)", len(args)+1)), append(args, *v)
		} else {
			where, args = append(where, fmt.Sprintf("db.name = $%d", len(args)+1)), append(args, *v)
		}
	}
	if v := find.UID; v != nil {
		where, args = append(where, fmt.Sprintf("db.id = $%d", len(args)+1)), append(args, *v)
	}
	if v := find.Engine; v != nil {
		where, args = append(where, fmt.Sprintf("instance.engine = $%d", len(args)+1)), append(args, *v)
	}
	if !find.ShowDeleted {
		where, args = append(where, fmt.Sprintf(`
			COALESCE(
				(SELECT environment.deleted AS instance_environment_status FROM environment JOIN instance ON environment.resource_id = instance.environment WHERE instance.resource_id = db.instance),
				$%d
			) = $%d`, len(args)+1, len(args)+2)), append(args, false, false)
		where, args = append(where, fmt.Sprintf(`
			COALESCE(
				(SELECT environment.deleted AS db_environment_status FROM environment WHERE environment.resource_id = db.environment),
				$%d
			) = $%d`, len(args)+1, len(args)+2)), append(args, false, false)

		where, args = append(where, fmt.Sprintf("instance.deleted = $%d", len(args)+1)), append(args, false)
		// We don't show databases that are deleted by users already.
		where, args = append(where, fmt.Sprintf("db.sync_status = $%d", len(args)+1)), append(args, api.OK)
	}

	query := fmt.Sprintf(`
		SELECT
			db.id,
			db.project,
			COALESCE(
				(SELECT environment.resource_id FROM environment WHERE environment.resource_id = db.environment),
				(SELECT environment.resource_id FROM environment JOIN instance ON environment.resource_id = instance.environment WHERE instance.resource_id = db.instance)
			),
			(SELECT environment.resource_id FROM environment WHERE environment.resource_id = db.environment),
			db.instance,
			db.name,
			db.sync_status,
			db.sync_at,
			COALESCE(
				(
					SELECT revision.version
					FROM revision
					WHERE revision.instance = db.instance AND revision.db_name = db.name AND deleted_at IS NOT NULL
					ORDER BY revision.version DESC
					LIMIT 1
				),
				''
			),
			db.secrets,
			db.datashare,
			db.service_name,
			db.metadata
		FROM db
		LEFT JOIN instance ON db.instance = instance.resource_id
		WHERE %s
		ORDER BY db.project, db.instance, db.name`, strings.Join(where, " AND "))
	if v := find.Limit; v != nil {
		query += fmt.Sprintf(" LIMIT %d", *v)
	}
	if v := find.Offset; v != nil {
		query += fmt.Sprintf(" OFFSET %d", *v)
	}

	var databaseMessages []*DatabaseMessage
	rows, err := tx.QueryContext(ctx, query,
		args...,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		databaseMessage := &DatabaseMessage{}
		var secretsString, metadataString string
		var effectiveEnvironment, environment sql.NullString
		if err := rows.Scan(
			&databaseMessage.UID,
			&databaseMessage.ProjectID,
			&effectiveEnvironment,
			&environment,
			&databaseMessage.InstanceID,
			&databaseMessage.DatabaseName,
			&databaseMessage.SyncState,
			&databaseMessage.SyncAt,
			&databaseMessage.SchemaVersion,
			&secretsString,
			&databaseMessage.DataShare,
			&databaseMessage.ServiceName,
			&metadataString,
		); err != nil {
			return nil, err
		}
		if effectiveEnvironment.Valid {
			databaseMessage.EffectiveEnvironmentID = effectiveEnvironment.String
		}
		if environment.Valid {
			databaseMessage.EnvironmentID = environment.String
		}

		var secret storepb.Secrets
		if err := common.ProtojsonUnmarshaler.Unmarshal([]byte(secretsString), &secret); err != nil {
			return nil, err
		}
		databaseMessage.Secrets = &secret
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
