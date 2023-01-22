package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/lib/pq"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	api "github.com/bytebase/bytebase/backend/legacyapi"
)

// FindDatabase finds a list of Database instances.
func (s *Store) FindDatabase(ctx context.Context, find *api.DatabaseFind) ([]*api.Database, error) {
	// We don't have caller for searching IncludeAllDatabase.
	v2Find := &FindDatabaseMessage{}
	if find.InstanceID != nil {
		instance, err := s.GetInstanceV2(ctx, &FindInstanceMessage{UID: find.InstanceID})
		if err != nil {
			return nil, err
		}
		v2Find.InstanceID = &instance.ResourceID
	}
	if find.ProjectID != nil {
		project, err := s.GetProjectV2(ctx, &FindProjectMessage{UID: find.ProjectID})
		if err != nil {
			return nil, err
		}
		v2Find.ProjectID = &project.ResourceID
	}
	if find.Name != nil {
		v2Find.DatabaseName = find.Name
	}

	databases, err := s.ListDatabases(ctx, v2Find)
	if err != nil {
		return nil, err
	}
	var databaseList []*api.Database
	for _, database := range databases {
		composedDatabase, err := s.composeDatabase(ctx, database)
		if err != nil {
			return nil, err
		}
		if find.SyncStatus != nil && composedDatabase.SyncStatus != *find.SyncStatus {
			continue
		}
		databaseList = append(databaseList, composedDatabase)
	}
	return databaseList, nil
}

// GetDatabase gets an instance of Database.
func (s *Store) GetDatabase(ctx context.Context, find *api.DatabaseFind) (*api.Database, error) {
	v2Find := &FindDatabaseMessage{
		ShowDeleted: true,
	}
	if find.InstanceID != nil {
		instance, err := s.GetInstanceV2(ctx, &FindInstanceMessage{UID: find.InstanceID})
		if err != nil {
			return nil, err
		}
		v2Find.InstanceID = &instance.ResourceID
	}
	if find.ProjectID != nil {
		project, err := s.GetProjectV2(ctx, &FindProjectMessage{UID: find.ProjectID})
		if err != nil {
			return nil, err
		}
		v2Find.ProjectID = &project.ResourceID
	}
	if find.Name != nil {
		v2Find.DatabaseName = find.Name
	}
	if find.ID != nil {
		v2Find.UID = find.ID
	}
	v2Find.IncludeAllDatabase = find.IncludeAllDatabase

	databases, err := s.ListDatabases(ctx, v2Find)
	if err != nil {
		return nil, err
	}
	if len(databases) == 0 {
		return nil, nil
	} else if len(databases) > 1 {
		return nil, &common.Error{Code: common.Conflict, Err: errors.Errorf("found %d databases with filter %+v, expect 1. ", len(databases), v2Find)}
	}
	database := databases[0]
	composedDatabase, err := s.composeDatabase(ctx, database)
	if err != nil {
		return nil, err
	}
	return composedDatabase, nil
}

// private functions.
func (s *Store) composeDatabase(ctx context.Context, database *DatabaseMessage) (*api.Database, error) {
	instance, err := s.GetInstanceV2(ctx, &FindInstanceMessage{ResourceID: &database.InstanceID})
	if err != nil {
		return nil, err
	}
	project, err := s.GetProjectV2(ctx, &FindProjectMessage{ResourceID: &database.ProjectID})
	if err != nil {
		return nil, err
	}
	composedDatabase := &api.Database{
		ID:                   database.UID,
		ProjectID:            project.UID,
		InstanceID:           instance.UID,
		Name:                 database.DatabaseName,
		SchemaVersion:        database.SchemaVersion,
		SyncStatus:           database.SyncState,
		LastSuccessfulSyncTs: database.SuccessfulSyncTimeTs,
	}
	composedProject, err := s.GetProjectByID(ctx, project.UID)
	if err != nil {
		return nil, err
	}
	composedDatabase.Project = composedProject
	composedInstance, err := s.GetInstanceByID(ctx, instance.UID)
	if err != nil {
		return nil, err
	}
	composedDatabase.Instance = composedInstance

	// For now, only wildcard(*) database has data sources and we disallow it to be returned to the client.
	// So we set this value to an empty array until we need to develop a data source for a non-wildcard database.
	composedDatabase.DataSourceList = nil

	// Compose labels.
	var labelList []*api.DatabaseLabel
	for key, value := range database.Labels {
		labelList = append(labelList, &api.DatabaseLabel{
			Key:   key,
			Value: value,
		})
	}
	labels, err := json.Marshal(labelList)
	if err != nil {
		return nil, err
	}
	composedDatabase.Labels = string(labels)

	return composedDatabase, nil
}

// DatabaseMessage is the message for database.
type DatabaseMessage struct {
	UID           int
	ProjectID     string
	EnvironmentID string
	InstanceID    string

	DatabaseName         string
	SyncState            api.SyncStatus
	SuccessfulSyncTimeTs int64
	SchemaVersion        string
	Labels               map[string]string
}

// UpdateDatabaseMessage is the mssage for updating a database.
type UpdateDatabaseMessage struct {
	EnvironmentID string
	InstanceID    string
	DatabaseName  string

	ProjectID            *string
	SyncState            *api.SyncStatus
	SuccessfulSyncTimeTs *int64
	SchemaVersion        *string
	Labels               *map[string]string
	SourceBackupID       *int
}

// FindDatabaseMessage is the message for finding databases.
type FindDatabaseMessage struct {
	ProjectID     *string
	EnvironmentID *string
	InstanceID    *string
	DatabaseName  *string
	UID           *int
	// When this is used, we will return databases from archived instances or environments.
	// This is used for existing tasks with archived databases.
	ShowDeleted bool

	// TODO(d): deprecate this field when we migrate all datasource to v1 store.
	IncludeAllDatabase bool
}

// GetDatabaseV2 gets a database.
func (s *Store) GetDatabaseV2(ctx context.Context, find *FindDatabaseMessage) (*DatabaseMessage, error) {
	if find.EnvironmentID != nil && find.InstanceID != nil && find.DatabaseName != nil {
		if database, ok := s.databaseCache.Load(getDatabaseCacheKey(*find.EnvironmentID, *find.InstanceID, *find.DatabaseName)); ok {
			return database.(*DatabaseMessage), nil
		}
	}
	if find.UID != nil {
		if database, ok := s.databaseIDCache.Load(*find.UID); ok {
			return database.(*DatabaseMessage), nil
		}
	}

	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, FormatError(err)
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
		return nil, FormatError(err)
	}

	s.databaseCache.Store(getDatabaseCacheKey(database.EnvironmentID, database.InstanceID, database.DatabaseName), database)
	s.databaseIDCache.Store(database.UID, database)
	return database, nil
}

// ListDatabases lists all databases.
func (s *Store) ListDatabases(ctx context.Context, find *FindDatabaseMessage) ([]*DatabaseMessage, error) {
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.Rollback()

	databases, err := s.listDatabaseImplV2(ctx, tx, find)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, FormatError(err)
	}

	for _, database := range databases {
		s.databaseCache.Store(getDatabaseCacheKey(database.EnvironmentID, database.InstanceID, database.DatabaseName), database)
		s.databaseIDCache.Store(database.UID, database)
	}
	return databases, nil
}

// CreateDatabaseDefault creates a new database in the default project.
func (s *Store) CreateDatabaseDefault(ctx context.Context, create *DatabaseMessage) error {
	instance, err := s.GetInstanceV2(ctx, &FindInstanceMessage{EnvironmentID: &create.EnvironmentID, ResourceID: &create.InstanceID})
	if err != nil {
		return err
	}
	if instance == nil {
		return errors.Errorf("instance %q not found", create.InstanceID)
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return FormatError(err)
	}
	defer tx.Rollback()

	databaseUID, err := s.createDatabaseDefaultImpl(ctx, tx, instance.UID, create)
	if err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return FormatError(err)
	}

	// Invalidate an update the cache.
	s.databaseCache.Delete(getDatabaseCacheKey(instance.EnvironmentID, instance.ResourceID, create.DatabaseName))
	s.databaseIDCache.Delete(databaseUID)
	if _, err = s.GetDatabaseV2(ctx, &FindDatabaseMessage{UID: &databaseUID}); err != nil {
		return err
	}
	return nil
}

// createDatabaseDefault only creates a default database with charset, collation only in the default project.
func (*Store) createDatabaseDefaultImpl(ctx context.Context, tx *Tx, instanceUID int, create *DatabaseMessage) (int, error) {
	// We will do on conflict update the column updater_id for returning the ID because on conflict do nothing will not return anything.
	query := `
		INSERT INTO db (
			creator_id,
			updater_id,
			instance_id,
			project_id,
			name,
			sync_status,
			last_successful_sync_ts,
			schema_version
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		ON CONFLICT (instance_id, name) DO UPDATE SET
			updater_id = EXCLUDED.updater_id
		RETURNING id`
	var databaseUID int
	if err := tx.QueryRowContext(ctx, query,
		api.SystemBotID,
		api.SystemBotID,
		instanceUID,
		api.DefaultProjectUID,
		create.DatabaseName,
		api.OK,
		0,  /* last_successful_sync_ts */
		"", /* schema_version */
	).Scan(
		&databaseUID,
	); err != nil {
		return 0, FormatError(err)
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
	instance, err := s.GetInstanceV2(ctx, &FindInstanceMessage{EnvironmentID: &create.EnvironmentID, ResourceID: &create.InstanceID})
	if err != nil {
		return nil, err
	}
	if instance == nil {
		return nil, errors.Errorf("instance %q not found", create.InstanceID)
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
			name,
			sync_status,
			last_successful_sync_ts,
			schema_version
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		ON CONFLICT (instance_id, name) DO UPDATE SET
			project_id = EXCLUDED.project_id,
			name = EXCLUDED.name,
			schema_version = EXCLUDED.schema_version
		RETURNING id`
	var databaseUID int
	if err := tx.QueryRowContext(ctx, query,
		api.SystemBotID,
		api.SystemBotID,
		instance.UID,
		project.UID,
		create.DatabaseName,
		api.OK,
		create.SuccessfulSyncTimeTs,
		create.SchemaVersion,
	).Scan(
		&databaseUID,
	); err != nil {
		return nil, FormatError(err)
	}
	if err := s.setDatabaseLabels(ctx, tx, databaseUID, create.Labels, api.SystemBotID); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}

	// Invalidate and update the cache.
	s.databaseCache.Delete(getDatabaseCacheKey(instance.EnvironmentID, instance.ResourceID, create.DatabaseName))
	s.databaseIDCache.Delete(databaseUID)
	return s.GetDatabaseV2(ctx, &FindDatabaseMessage{UID: &databaseUID})
}

// UpdateDatabase updates a database.
func (s *Store) UpdateDatabase(ctx context.Context, patch *UpdateDatabaseMessage, updaterID int) (*DatabaseMessage, error) {
	instance, err := s.GetInstanceV2(ctx, &FindInstanceMessage{EnvironmentID: &patch.EnvironmentID, ResourceID: &patch.InstanceID})
	if err != nil {
		return nil, err
	}

	set, args := []string{"updater_id = $1"}, []interface{}{fmt.Sprintf("%d", updaterID)}
	if v := patch.ProjectID; v != nil {
		project, err := s.GetProjectV2(ctx, &FindProjectMessage{ResourceID: patch.ProjectID})
		if err != nil {
			return nil, err
		}
		set, args = append(set, fmt.Sprintf("project_id = $%d", len(args)+1)), append(args, project.UID)
	}
	if v := patch.SyncState; v != nil {
		set, args = append(set, fmt.Sprintf("sync_status = $%d", len(args)+1)), append(args, *v)
	}
	if v := patch.SuccessfulSyncTimeTs; v != nil {
		set, args = append(set, fmt.Sprintf("last_successful_sync_ts = $%d", len(args)+1)), append(args, *v)
	}
	if v := patch.SchemaVersion; v != nil {
		set, args = append(set, fmt.Sprintf("schema_version = $%d", len(args)+1)), append(args, *v)
	}
	args = append(args, instance.UID, patch.DatabaseName)

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.Rollback()
	var databaseUID int
	if err := tx.QueryRowContext(ctx, fmt.Sprintf(`
		UPDATE db
		SET `+strings.Join(set, ", ")+`
		WHERE instance_id = $%d AND name = $%d
		RETURNING id
	`, len(set)+1, len(set)+2),
		args...,
	).Scan(
		&databaseUID,
	); err != nil {
		return nil, FormatError(err)
	}
	if patch.Labels != nil {
		if err := s.setDatabaseLabels(ctx, tx, databaseUID, *(patch.Labels), updaterID); err != nil {
			return nil, err
		}
	}
	if err := tx.Commit(); err != nil {
		return nil, FormatError(err)
	}

	// Invalidate and update the cache.
	s.databaseCache.Delete(getDatabaseCacheKey(patch.EnvironmentID, patch.InstanceID, patch.DatabaseName))
	s.databaseIDCache.Delete(databaseUID)
	return s.GetDatabaseV2(ctx, &FindDatabaseMessage{UID: &databaseUID})
}

// BatchUpdateDatabaseProject updates the project for databases in batch.
func (s *Store) BatchUpdateDatabaseProject(ctx context.Context, databases []*DatabaseMessage, projectID string, updaterID int) ([]*DatabaseMessage, error) {
	project, err := s.GetProjectV2(ctx, &FindProjectMessage{ResourceID: &projectID})
	if err != nil {
		return nil, err
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.Rollback()

	var wheres []string
	args := []interface{}{project.UID, updaterID}
	for i, database := range databases {
		wheres = append(wheres, fmt.Sprintf("(environment.resource_id = $%d AND instance.resource_id = $%d AND db.name = $%d)", 3*i+3, 3*i+4, 3*i+5))
		args = append(args, database.EnvironmentID, database.InstanceID, database.DatabaseName)
	}
	if _, err := tx.ExecContext(ctx, fmt.Sprintf(`
			UPDATE db
			SET project_id = $1, updater_id = $2
			FROM instance JOIN environment ON instance.environment_id = environment.id
			WHERE db.instance_id = instance.id AND (%s);`, strings.Join(wheres, " OR ")),
		args...,
	); err != nil {
		return nil, FormatError(err)
	}

	if err := tx.Commit(); err != nil {
		return nil, FormatError(err)
	}

	var updatedDatabases []*DatabaseMessage
	for _, database := range databases {
		updatedDatabase := *database
		updatedDatabase.ProjectID = project.ResourceID
		s.databaseCache.Store(getDatabaseCacheKey(database.EnvironmentID, database.InstanceID, database.DatabaseName), &updatedDatabase)
		s.databaseIDCache.Store(database.UID, &updatedDatabase)
		updatedDatabases = append(updatedDatabases, &updatedDatabase)
	}
	return updatedDatabases, nil
}

func (s *Store) getDatabaseImplV2(ctx context.Context, tx *Tx, find *FindDatabaseMessage) (*DatabaseMessage, error) {
	databaseList, err := s.listDatabaseImplV2(ctx, tx, find)
	if err != nil {
		return nil, err
	}
	if len(databaseList) == 0 {
		return nil, &common.Error{Code: common.NotFound, Err: errors.Errorf("database not found with %v", find)}
	}
	if len(databaseList) > 1 {
		return nil, &common.Error{Code: common.NotFound, Err: errors.Errorf("found %d data source databases with %v, but expect 1", len(databaseList), find)}
	}

	return databaseList[0], nil
}

func (*Store) listDatabaseImplV2(ctx context.Context, tx *Tx, find *FindDatabaseMessage) ([]*DatabaseMessage, error) {
	where, args := []string{"TRUE"}, []interface{}{}
	if !find.IncludeAllDatabase {
		where, args = append(where, fmt.Sprintf("db.name != $%d", len(args)+1)), append(args, api.AllDatabaseName)
	}
	if v := find.ProjectID; v != nil {
		where, args = append(where, fmt.Sprintf("project.resource_id = $%d", len(args)+1)), append(args, *v)
	}
	if v := find.EnvironmentID; v != nil {
		where, args = append(where, fmt.Sprintf("environment.resource_id = $%d", len(args)+1)), append(args, *v)
	}
	if v := find.InstanceID; v != nil {
		where, args = append(where, fmt.Sprintf("instance.resource_id = $%d", len(args)+1)), append(args, *v)
	}
	if v := find.DatabaseName; v != nil {
		where, args = append(where, fmt.Sprintf("db.name = $%d", len(args)+1)), append(args, *v)
	}
	if v := find.UID; v != nil {
		where, args = append(where, fmt.Sprintf("db.id = $%d", len(args)+1)), append(args, *v)
	}
	if !find.ShowDeleted {
		where, args = append(where, fmt.Sprintf("environment.row_status = $%d", len(args)+1)), append(args, api.Normal)
		where, args = append(where, fmt.Sprintf("instance.row_status = $%d", len(args)+1)), append(args, api.Normal)
	}

	var databaseMessages []*DatabaseMessage
	rows, err := tx.QueryContext(ctx, fmt.Sprintf(`
		SELECT
			db.id,
			project.resource_id AS project_id,
			environment.resource_id AS environment_id,
			instance.resource_id AS instance_id,
			db.name,
			db.sync_status,
			db.last_successful_sync_ts,
			db.schema_version,
			ARRAY_AGG (
				db_label.key
			) keys,
			ARRAY_AGG (
				db_label.value
			) label_values
		FROM db
		LEFT JOIN project ON db.project_id = project.id
		LEFT JOIN instance ON db.instance_id = instance.id
		LEFT JOIN environment ON instance.environment_id = environment.id
		LEFT JOIN db_label ON db.id = db_label.database_id
		WHERE %s
		GROUP BY db.id, project.resource_id, environment.resource_id, instance.resource_id`, strings.Join(where, " AND ")),
		args...,
	)
	if err != nil {
		return nil, FormatError(err)
	}
	defer rows.Close()
	for rows.Next() {
		databaseMessage := DatabaseMessage{
			Labels: make(map[string]string),
		}
		var keys, values []sql.NullString
		if err := rows.Scan(
			&databaseMessage.UID,
			&databaseMessage.ProjectID,
			&databaseMessage.EnvironmentID,
			&databaseMessage.InstanceID,
			&databaseMessage.DatabaseName,
			&databaseMessage.SyncState,
			&databaseMessage.SuccessfulSyncTimeTs,
			&databaseMessage.SchemaVersion,
			pq.Array(&keys),
			pq.Array(&values),
		); err != nil {
			return nil, FormatError(err)
		}
		if len(keys) != len(values) {
			return nil, errors.Errorf("invalid length of database label keys and values")
		}
		for i := 0; i < len(keys); i++ {
			if !keys[i].Valid || !values[i].Valid {
				continue
			}
			databaseMessage.Labels[keys[i].String] = values[i].String
		}
		// System default environment label.
		// The value of bb.environment is resource ID of the environment.
		databaseMessage.Labels[api.EnvironmentLabelKey] = databaseMessage.EnvironmentID

		databaseMessages = append(databaseMessages, &databaseMessage)
	}

	return databaseMessages, nil
}

func (s *Store) setDatabaseLabels(ctx context.Context, tx *Tx, databaseUID int, labels map[string]string, updaterID int) error {
	oldLabels, err := s.getDatabaseLabels(ctx, tx, databaseUID)
	if err != nil {
		return err
	}
	upserts := make(map[string]string)
	var deleteKeys []string
	for key, value := range labels {
		// We will skip writing the system label, environment.
		if key == api.EnvironmentLabelKey {
			continue
		}
		if oldValue, ok := oldLabels[key]; !ok || oldValue != value {
			upserts[key] = value
		}
	}
	for key := range oldLabels {
		if _, ok := labels[key]; !ok {
			deleteKeys = append(deleteKeys, key)
		}
	}
	if err := s.upsertLabels(ctx, tx, databaseUID, upserts, updaterID); err != nil {
		return err
	}
	return s.deleteLabels(ctx, tx, databaseUID, deleteKeys)
}

func (*Store) upsertLabels(ctx context.Context, tx *Tx, databaseUID int, labels map[string]string, updaterID int) error {
	query := `
		INSERT INTO db_label (
			row_status,
			creator_id,
			updater_id,
			database_id,
			key,
			value
		)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT(database_id, key) DO UPDATE SET
			row_status = excluded.row_status,
			updater_id = excluded.updater_id,
			value = excluded.value
	`
	for key, value := range labels {
		if _, err := tx.ExecContext(ctx, query,
			api.Normal,
			updaterID,
			updaterID,
			databaseUID,
			key,
			value,
		); err != nil {
			return err
		}
	}
	return nil
}

func (*Store) deleteLabels(ctx context.Context, tx *Tx, databaseUID int, keys []string) error {
	query := `
		DELETE FROM db_label
		WHERE database_id = $1 AND key = $2
	`
	for _, key := range keys {
		if _, err := tx.ExecContext(ctx, query,
			databaseUID,
			key,
		); err != nil {
			return err
		}
	}
	return nil
}

func (*Store) getDatabaseLabels(ctx context.Context, tx *Tx, databaseUID int) (map[string]string, error) {
	labels := make(map[string]string)
	rows, err := tx.QueryContext(ctx, `
		SELECT
			key,
			value
		FROM db_label
		WHERE database_id = $1`,
		databaseUID,
	)
	if err != nil {
		return nil, FormatError(err)
	}
	defer rows.Close()
	for rows.Next() {
		var key, value string
		if err := rows.Scan(
			&key,
			&value,
		); err != nil {
			return nil, FormatError(err)
		}
		labels[key] = value
	}
	if err := rows.Err(); err != nil {
		return nil, FormatError(err)
	}
	return labels, nil
}
