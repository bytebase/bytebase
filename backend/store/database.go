package store

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/pkg/errors"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/bytebase/bytebase/backend/common"
	api "github.com/bytebase/bytebase/backend/legacyapi"
	"github.com/bytebase/bytebase/backend/store/model"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

// DatabaseMessage is the message for database.
type DatabaseMessage struct {
	UID                    int
	ProjectID              string
	InstanceID             string
	EnvironmentID          string
	EffectiveEnvironmentID string

	DatabaseName         string
	SyncState            api.SyncStatus
	SuccessfulSyncTimeTs int64
	SchemaVersion        model.Version
	Secrets              *storepb.Secrets
	DataShare            bool
	// ServiceName is the Oracle specific field.
	ServiceName string
	Metadata    *storepb.DatabaseMetadata
}

// UpdateDatabaseMessage is the mssage for updating a database.
type UpdateDatabaseMessage struct {
	InstanceID   string
	DatabaseName string

	ProjectID            *string
	SyncState            *api.SyncStatus
	SuccessfulSyncTimeTs *int64
	SchemaVersion        *model.Version
	SourceBackupID       *int
	Secrets              *storepb.Secrets
	DataShare            *bool
	ServiceName          *string
	EnvironmentID        *string

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
func (s *Store) CreateDatabaseDefault(ctx context.Context, create *DatabaseMessage) error {
	project, err := s.GetProjectV2(ctx, &FindProjectMessage{ResourceID: &create.ProjectID})
	if err != nil {
		return err
	}
	if project == nil {
		return errors.Errorf("project %q not found", create.ProjectID)
	}
	instance, err := s.GetInstanceV2(ctx, &FindInstanceMessage{ResourceID: &create.InstanceID})
	if err != nil {
		return err
	}
	if instance == nil {
		return errors.Errorf("instance %q not found", create.InstanceID)
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	databaseUID, err := s.createDatabaseDefaultImpl(ctx, tx, project.UID, instance.UID, create)
	if err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	// Invalidate an update the cache.
	s.databaseCache.Remove(getDatabaseCacheKey(instance.ResourceID, create.DatabaseName))
	s.databaseIDCache.Remove(databaseUID)
	if _, err = s.GetDatabaseV2(ctx, &FindDatabaseMessage{UID: &databaseUID}); err != nil {
		return err
	}
	return nil
}

// createDatabaseDefault only creates a default database with charset, collation only in the default project.
func (*Store) createDatabaseDefaultImpl(ctx context.Context, tx *Tx, projectUID, instanceUID int, create *DatabaseMessage) (int, error) {
	secretsString, err := protojson.Marshal(&storepb.Secrets{})
	if err != nil {
		return 0, err
	}

	// We will do on conflict update the column updater_id for returning the ID because on conflict do nothing will not return anything.
	// We will also move the deleted database into default project.
	query := `
		INSERT INTO db (
			creator_id,
			updater_id,
			instance_id,
			project_id,
			name,
			sync_status,
			last_successful_sync_ts,
			schema_version,
			secrets,
			datashare,
			service_name
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		ON CONFLICT (instance_id, name) DO UPDATE SET
			updater_id = EXCLUDED.updater_id,
			project_id = EXCLUDED.project_id,
			sync_status = EXCLUDED.sync_status,
			last_successful_sync_ts = EXCLUDED.last_successful_sync_ts,
			datashare = EXCLUDED.datashare,
			service_name = EXCLUDED.service_name
		RETURNING id`
	var databaseUID int
	if err := tx.QueryRowContext(ctx, query,
		api.SystemBotID,
		api.SystemBotID,
		instanceUID,
		projectUID,
		create.DatabaseName,
		api.OK,
		0,             /* last_successful_sync_ts */
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
	project, err := s.GetProjectV2(ctx, &FindProjectMessage{ResourceID: &create.ProjectID})
	if err != nil {
		return nil, err
	}
	if project == nil {
		return nil, errors.Errorf("project %q not found", create.ProjectID)
	}
	instance, err := s.GetInstanceV2(ctx, &FindInstanceMessage{ResourceID: &create.InstanceID})
	if err != nil {
		return nil, err
	}
	if instance == nil {
		return nil, errors.Errorf("instance %q not found", create.InstanceID)
	}
	var environmentUID *int
	if create.EnvironmentID != "" {
		environment, err := s.GetEnvironmentV2(ctx, &FindEnvironmentMessage{ResourceID: &create.EnvironmentID})
		if err != nil {
			return nil, err
		}
		if environment == nil {
			return nil, errors.Errorf("environment %q not found", create.EnvironmentID)
		}
		environmentUID = &environment.UID
	}
	storedVersion, err := create.SchemaVersion.Marshal()
	if err != nil {
		return nil, err
	}

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

	// We will do on conflict update the column updater_id for returning the ID because on conflict do nothing will not return anything.
	query := `
		INSERT INTO db (
			creator_id,
			updater_id,
			instance_id,
			project_id,
			environment_id,
			name,
			sync_status,
			last_successful_sync_ts,
			schema_version,
			secrets,
			datashare,
			service_name,
			metadata
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		ON CONFLICT (instance_id, name) DO UPDATE SET
			project_id = EXCLUDED.project_id,
			environment_id = EXCLUDED.environment_id,
			name = EXCLUDED.name,
			schema_version = EXCLUDED.schema_version,
			metadata = EXCLUDED.metadata
		RETURNING id`
	var databaseUID int
	if err := tx.QueryRowContext(ctx, query,
		api.SystemBotID,
		api.SystemBotID,
		instance.UID,
		project.UID,
		environmentUID,
		create.DatabaseName,
		create.SyncState,
		create.SuccessfulSyncTimeTs,
		storedVersion,
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
	s.databaseCache.Remove(getDatabaseCacheKey(instance.ResourceID, create.DatabaseName))
	s.databaseIDCache.Remove(databaseUID)
	return s.GetDatabaseV2(ctx, &FindDatabaseMessage{UID: &databaseUID, ShowDeleted: true})
}

// UpdateDatabase updates a database.
func (s *Store) UpdateDatabase(ctx context.Context, patch *UpdateDatabaseMessage, updaterID int) (*DatabaseMessage, error) {
	instance, err := s.GetInstanceV2(ctx, &FindInstanceMessage{ResourceID: &patch.InstanceID})
	if err != nil {
		return nil, err
	}

	set, args := []string{"updater_id = $1"}, []any{fmt.Sprintf("%d", updaterID)}
	if v := patch.ProjectID; v != nil {
		project, err := s.GetProjectV2(ctx, &FindProjectMessage{ResourceID: patch.ProjectID})
		if err != nil {
			return nil, err
		}
		set, args = append(set, fmt.Sprintf("project_id = $%d", len(args)+1)), append(args, project.UID)
	}
	if v := patch.EnvironmentID; v != nil {
		if *v == "" {
			set = append(set, "environment_id = NULL")
		} else {
			environment, err := s.GetEnvironmentV2(ctx, &FindEnvironmentMessage{ResourceID: patch.EnvironmentID})
			if err != nil {
				return nil, err
			}
			if environment == nil {
				return nil, errors.Errorf("environment %v not found", *v)
			}
			set, args = append(set, fmt.Sprintf("environment_id = $%d", len(args)+1)), append(args, environment.UID)
		}
	}
	if v := patch.SyncState; v != nil {
		set, args = append(set, fmt.Sprintf("sync_status = $%d", len(args)+1)), append(args, *v)
	}
	if v := patch.SuccessfulSyncTimeTs; v != nil {
		set, args = append(set, fmt.Sprintf("last_successful_sync_ts = $%d", len(args)+1)), append(args, *v)
	}
	if v := patch.SchemaVersion; v != nil {
		storedVersion, err := patch.SchemaVersion.Marshal()
		if err != nil {
			return nil, err
		}
		set, args = append(set, fmt.Sprintf("schema_version = $%d", len(args)+1)), append(args, storedVersion)
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
	args = append(args, instance.UID, patch.DatabaseName)

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	var databaseUID int
	if err := tx.QueryRowContext(ctx, fmt.Sprintf(`
		UPDATE db
		SET `+strings.Join(set, ", ")+`
		WHERE instance_id = $%d AND name = $%d
		RETURNING id
	`, len(args)-1, len(args)),
		args...,
	).Scan(
		&databaseUID,
	); err != nil {
		return nil, err
	}
	// When we update the project ID of the database, we should update the project ID of the related sheets in the same transaction.
	if patch.ProjectID != nil {
		sheetList, err := s.ListSheets(ctx, &FindSheetMessage{DatabaseUID: &databaseUID}, updaterID)
		if err != nil {
			return nil, err
		}
		if len(sheetList) > 0 {
			project, err := s.GetProjectV2(ctx, &FindProjectMessage{ResourceID: patch.ProjectID})
			if err != nil {
				return nil, err
			}
			for _, sheet := range sheetList {
				if _, err := patchSheetImpl(ctx, tx, &PatchSheetMessage{UID: sheet.UID, ProjectUID: &project.UID, UpdaterID: updaterID}); err != nil {
					return nil, err
				}
			}
		}
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}

	// Invalidate and update the cache.
	s.databaseCache.Remove(getDatabaseCacheKey(patch.InstanceID, patch.DatabaseName))
	s.databaseIDCache.Remove(databaseUID)
	return s.GetDatabaseV2(ctx, &FindDatabaseMessage{UID: &databaseUID})
}

// BatchUpdateDatabaseProject updates the project for databases in batch.
func (s *Store) BatchUpdateDatabaseProject(ctx context.Context, databases []*DatabaseMessage, projectID string, updaterID int) ([]*DatabaseMessage, error) {
	if len(databases) == 0 {
		return nil, errors.Errorf("there is no database in the project")
	}
	project, err := s.GetProjectV2(ctx, &FindProjectMessage{ResourceID: &projectID})
	if err != nil {
		return nil, err
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	var wheres []string
	args := []any{project.UID, updaterID}
	for i, database := range databases {
		wheres = append(wheres, fmt.Sprintf("(instance.resource_id = $%d AND db.name = $%d)", 2*i+3, 2*i+4))
		args = append(args, database.InstanceID, database.DatabaseName)
	}
	databaseClause := ""
	if len(wheres) > 0 {
		databaseClause = fmt.Sprintf(" AND (%s)", strings.Join(wheres, " OR "))
	}
	if _, err := tx.ExecContext(ctx, fmt.Sprintf(`
			UPDATE db
			SET project_id = $1, updater_id = $2
			FROM instance
			WHERE db.instance_id = instance.id %s;`, databaseClause),
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
		updatedDatabase.ProjectID = project.ResourceID
		s.databaseCache.Add(getDatabaseCacheKey(database.InstanceID, database.DatabaseName), &updatedDatabase)
		s.databaseIDCache.Add(database.UID, &updatedDatabase)
		updatedDatabases = append(updatedDatabases, &updatedDatabase)
	}
	return updatedDatabases, nil
}

func (*Store) listDatabaseImplV2(ctx context.Context, tx *Tx, find *FindDatabaseMessage) ([]*DatabaseMessage, error) {
	where, args := []string{"TRUE"}, []any{}
	if v := find.ProjectID; v != nil {
		where, args = append(where, fmt.Sprintf("project.resource_id = $%d", len(args)+1)), append(args, *v)
	}
	if v := find.EffectiveEnvironmentID; v != nil {
		where, args = append(where, fmt.Sprintf("COALESCE(COALESCE((SELECT environment.resource_id FROM environment where environment.id = db.environment_id), (SELECT environment.resource_id FROM environment JOIN instance ON environment.id = instance.environment_id WHERE instance.id = db.instance_id)), '') = $%d", len(args)+1)), append(args, *v)
	}
	if v := find.InstanceID; v != nil {
		where, args = append(where, fmt.Sprintf("instance.resource_id = $%d", len(args)+1)), append(args, *v)
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
		where, args = append(where, fmt.Sprintf("COALESCE((SELECT environment.row_status AS instance_environment_status FROM environment JOIN instance ON environment.id = instance.environment_id WHERE instance.id = db.instance_id), $%d) = $%d", len(args)+1, len(args)+2)), append(args, api.Normal, api.Normal)
		where, args = append(where, fmt.Sprintf("COALESCE((SELECT environment.row_status AS db_environment_status FROM environment WHERE environment.id = db.environment_id), $%d) = $%d", len(args)+1, len(args)+2)), append(args, api.Normal, api.Normal)

		where, args = append(where, fmt.Sprintf("instance.row_status = $%d", len(args)+1)), append(args, api.Normal)
		// We don't show databases that are deleted by users already.
		where, args = append(where, fmt.Sprintf("db.sync_status = $%d", len(args)+1)), append(args, api.OK)
	}

	var databaseMessages []*DatabaseMessage
	rows, err := tx.QueryContext(ctx, fmt.Sprintf(`
		SELECT
			db.id,
			project.resource_id AS project_id,
			COALESCE(COALESCE((SELECT environment.resource_id FROM environment WHERE environment.id = db.environment_id), (SELECT environment.resource_id FROM environment JOIN instance ON environment.id = instance.environment_id WHERE instance.id = db.instance_id)), ''),
			COALESCE((SELECT environment.resource_id FROM environment WHERE environment.id = db.environment_id), ''),
			instance.resource_id AS instance_id,
			db.name,
			db.sync_status,
			db.last_successful_sync_ts,
			db.schema_version,
			db.secrets,
			db.datashare,
			db.service_name,
			db.metadata
		FROM db
		LEFT JOIN project ON db.project_id = project.id
		LEFT JOIN instance ON db.instance_id = instance.id
		WHERE %s
		GROUP BY db.id, project.resource_id, instance.resource_id`, strings.Join(where, " AND ")),
		args...,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		databaseMessage := &DatabaseMessage{}
		var storedVersion, secretsString, metadataString string
		if err := rows.Scan(
			&databaseMessage.UID,
			&databaseMessage.ProjectID,
			&databaseMessage.EffectiveEnvironmentID,
			&databaseMessage.EnvironmentID,
			&databaseMessage.InstanceID,
			&databaseMessage.DatabaseName,
			&databaseMessage.SyncState,
			&databaseMessage.SuccessfulSyncTimeTs,
			&storedVersion,
			&secretsString,
			&databaseMessage.DataShare,
			&databaseMessage.ServiceName,
			&metadataString,
		); err != nil {
			return nil, err
		}
		version, err := model.NewVersion(storedVersion)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to parse schema version %q", storedVersion)
		}
		databaseMessage.SchemaVersion = version

		var secret storepb.Secrets
		if err := protojson.Unmarshal([]byte(secretsString), &secret); err != nil {
			return nil, err
		}
		databaseMessage.Secrets = &secret
		var metadata storepb.DatabaseMetadata
		if err := protojson.Unmarshal([]byte(metadataString), &metadata); err != nil {
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
