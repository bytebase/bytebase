package store

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/common"
	"github.com/bytebase/bytebase/plugin/db"
)

// InstanceCreate is the API message to create the instance.
// TODO(ed): This is an temporary struct to compatible with OpenAPI and JSONAPI. Find way to move it into the API package.
type InstanceCreate struct {
	// Standard fields
	// Value is assigned from the jwt subject field passed by the client.
	CreatorID int

	// Related fields
	EnvironmentID  int
	DataSourceList []*api.DataSourceCreate

	// Domain specific fields
	Name         string
	Engine       db.Type
	ExternalLink string
	Host         string
	Port         string
	Database     string
}

// InstancePatch is the API message for patching an instance.
// TODO(ed): This is an temporary struct to compatible with OpenAPI and JSONAPI. Find way to move it into the API package.
type InstancePatch struct {
	ID int

	// Standard fields
	RowStatus *string
	// Value is assigned from the jwt subject field passed by the client.
	UpdaterID int

	// Related fields
	DataSourceList []*api.DataSourceCreate

	// Domain specific fields
	Name          *string
	EngineVersion *string
	ExternalLink  *string
	Host          *string
	Port          *string
	Database      *string
}

// instanceRaw is the store model for an Instance.
// Fields have exactly the same meanings as Instance.
type instanceRaw struct {
	ID         int
	ResourceID string

	// Standard fields
	RowStatus api.RowStatus
	CreatorID int
	CreatedTs int64
	UpdaterID int
	UpdatedTs int64

	// Related fields
	EnvironmentID int

	// Domain specific fields
	Name          string
	Engine        db.Type
	EngineVersion string
	ExternalLink  string
	Host          string
	Port          string
	Database      string
}

// toInstance creates an instance of Instance based on the instanceRaw.
// This is intended to be called when we need to compose an Instance relationship.
func (raw *instanceRaw) toInstance() *api.Instance {
	return &api.Instance{
		ID:         raw.ID,
		ResourceID: raw.ResourceID,
		RowStatus:  raw.RowStatus,

		// Related fields
		EnvironmentID: raw.EnvironmentID,

		// Domain specific fields
		Name:          raw.Name,
		Engine:        raw.Engine,
		EngineVersion: raw.EngineVersion,
		ExternalLink:  raw.ExternalLink,
		Host:          raw.Host,
		Port:          raw.Port,
		Database:      raw.Database,
	}
}

// CreateInstance creates an instance of Instance.
func (s *Store) CreateInstance(ctx context.Context, create *InstanceCreate) (*api.Instance, error) {
	instanceRaw, err := s.createInstanceRaw(ctx, create)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to create Instance with InstanceCreate[%+v]", create)
	}
	instance, err := s.composeInstance(ctx, instanceRaw)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to compose Instance with instanceRaw[%+v]", instanceRaw)
	}
	return instance, nil
}

// GetInstanceByID gets an instance of Instance.
func (s *Store) GetInstanceByID(ctx context.Context, id int) (*api.Instance, error) {
	find := &api.InstanceFind{ID: &id}
	instanceRaw, err := s.getInstanceRaw(ctx, find)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get Instance with ID %d", id)
	}
	if instanceRaw == nil {
		return nil, nil
	}
	instance, err := s.composeInstance(ctx, instanceRaw)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to compose Instance with instanceRaw[%+v]", instanceRaw)
	}
	return instance, nil
}

// FindInstance finds a list of Instance instances.
func (s *Store) FindInstance(ctx context.Context, find *api.InstanceFind) ([]*api.Instance, error) {
	instanceRawList, err := s.findInstanceRaw(ctx, find)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to find Instance list with InstanceFind[%+v]", find)
	}
	var instanceList []*api.Instance
	for _, raw := range instanceRawList {
		instance, err := s.composeInstance(ctx, raw)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to compose Instance with instanceRaw[%+v]", raw)
		}
		instanceList = append(instanceList, instance)
	}
	return instanceList, nil
}

// PatchInstance patches an instance of Instance.
func (s *Store) PatchInstance(ctx context.Context, patch *InstancePatch) (*api.Instance, error) {
	instanceRaw, err := s.patchInstanceRaw(ctx, patch)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to patch Instance with InstancePatch[%+v]", patch)
	}
	instance, err := s.composeInstance(ctx, instanceRaw)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to compose Instance with instanceRaw[%+v]", instanceRaw)
	}
	s.instanceCache.Delete(getInstanceCacheKey(instance.Environment.ResourceID, instanceRaw.ResourceID))
	s.instanceIDCache.Delete(instance.ID)
	return instance, nil
}

// private function.
func (s *Store) composeInstance(ctx context.Context, raw *instanceRaw) (*api.Instance, error) {
	instance := raw.toInstance()

	env, err := s.GetEnvironmentByID(ctx, instance.EnvironmentID)
	if err != nil {
		return nil, err
	}
	instance.Environment = env

	dataSourceList, err := s.findDataSource(ctx, &api.DataSourceFind{
		InstanceID: &instance.ID,
	})
	if err != nil {
		return nil, err
	}
	instance.DataSourceList = dataSourceList
	return instance, nil
}

// createInstanceRaw creates a new instance.
func (s *Store) createInstanceRaw(ctx context.Context, create *InstanceCreate) (*instanceRaw, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.Rollback()

	instance, err := createInstanceImpl(ctx, tx, create)
	if err != nil {
		return nil, err
	}

	// Create * database
	allDatabaseUID, err := s.createDatabaseDefaultImpl(ctx, tx, instance.ID, &DatabaseMessage{DatabaseName: api.AllDatabaseName})
	if err != nil {
		return nil, err
	}

	for _, dataSource := range create.DataSourceList {
		dataSourceCreate := &api.DataSourceCreate{
			CreatorID:  create.CreatorID,
			InstanceID: instance.ID,
			DatabaseID: allDatabaseUID,
			Name:       dataSource.Name,
			Type:       dataSource.Type,
			Username:   dataSource.Username,
			Password:   dataSource.Password,
			SslKey:     dataSource.SslKey,
			SslCert:    dataSource.SslCert,
			SslCa:      dataSource.SslCa,
			Host:       dataSource.Host,
			Port:       dataSource.Port,
			Options:    dataSource.Options,
			Database:   dataSource.Database,
		}
		if err := s.createDataSourceRawTx(ctx, tx, dataSourceCreate); err != nil {
			return nil, err
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, FormatError(err)
	}

	if err := s.cache.UpsertCache(instanceCacheNamespace, instance.ID, instance); err != nil {
		return nil, err
	}

	return instance, nil
}

// findInstanceRaw retrieves a list of instances based on find.
func (s *Store) findInstanceRaw(ctx context.Context, find *api.InstanceFind) ([]*instanceRaw, error) {
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.Rollback()

	list, err := findInstanceImpl(ctx, tx, find)
	if err != nil {
		return nil, err
	}

	return list, nil
}

// getInstanceRaw retrieves a single instance based on find.
// Returns ECONFLICT if finding more than 1 matching records.
func (s *Store) getInstanceRaw(ctx context.Context, find *api.InstanceFind) (*instanceRaw, error) {
	if find.ID != nil {
		instanceRaw := &instanceRaw{}
		has, err := s.cache.FindCache(instanceCacheNamespace, *find.ID, instanceRaw)
		if err != nil {
			return nil, err
		}
		if has {
			return instanceRaw, nil
		}
	}

	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.Rollback()

	list, err := findInstanceImpl(ctx, tx, find)
	if err != nil {
		return nil, err
	}

	if len(list) == 0 {
		return nil, nil
	} else if len(list) > 1 {
		return nil, &common.Error{Code: common.Conflict, Err: errors.Errorf("found %d instances with filter %+v, expect 1", len(list), find)}
	}

	instance := list[0]
	if err := s.cache.UpsertCache(instanceCacheNamespace, instance.ID, instance); err != nil {
		return nil, err
	}
	return instance, nil
}

// patchInstanceRaw updates an existing instance by ID.
// Returns ENOTFOUND if instance does not exist.
func (s *Store) patchInstanceRaw(ctx context.Context, patch *InstancePatch) (*instanceRaw, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.Rollback()

	instance, err := patchInstanceImpl(ctx, tx, patch)
	if err != nil {
		return nil, FormatError(err)
	}

	if patch.DataSourceList != nil {
		allDatabaseName := api.AllDatabaseName
		databaseList, err := s.listDatabaseImplV2(ctx, tx, &FindDatabaseMessage{
			InstanceID:         &instance.ResourceID,
			DatabaseName:       &allDatabaseName,
			IncludeAllDatabase: true,
		})
		if err != nil {
			return nil, err
		}
		if len(databaseList) == 0 {
			return nil, &common.Error{Code: common.NotFound, Err: errors.Errorf("data source database not found for instance %q", instance.ResourceID)}
		}
		if len(databaseList) > 1 {
			return nil, &common.Error{Code: common.NotFound, Err: errors.Errorf("found %d data source databases for instance %q", len(databaseList), instance.ResourceID)}
		}
		allDatabase := databaseList[0]

		if err := s.clearDataSourceImpl(ctx, tx, patch.ID, allDatabase.UID); err != nil {
			return nil, err
		}
		s.cache.DeleteCache(dataSourceCacheNamespace, patch.ID)

		for _, dataSource := range patch.DataSourceList {
			dataSourceCreate := &api.DataSourceCreate{
				CreatorID:  patch.UpdaterID,
				InstanceID: instance.ID,
				DatabaseID: allDatabase.UID,
				Name:       dataSource.Name,
				Type:       dataSource.Type,
				Username:   dataSource.Username,
				Password:   dataSource.Password,
				SslKey:     dataSource.SslKey,
				SslCert:    dataSource.SslCert,
				SslCa:      dataSource.SslCa,
				Host:       dataSource.Host,
				Port:       dataSource.Port,
				Options:    dataSource.Options,
				Database:   dataSource.Database,
			}
			if err := s.createDataSourceRawTx(ctx, tx, dataSourceCreate); err != nil {
				return nil, err
			}
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, FormatError(err)
	}

	if err := s.cache.UpsertCache(instanceCacheNamespace, instance.ID, instance); err != nil {
		return nil, err
	}

	return instance, nil
}

// createInstanceImpl creates a new instance.
func createInstanceImpl(ctx context.Context, tx *Tx, create *InstanceCreate) (*instanceRaw, error) {
	// TODO(d): allow users to set resource_id.
	resourceID := fmt.Sprintf("instance-%s", uuid.New().String()[:8])
	// Insert row into database.
	query := `
		INSERT INTO instance (
			creator_id,
			updater_id,
			environment_id,
			name,
			engine,
			external_link,
			host,
			port,
			database,
			resource_id
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id, resource_id, row_status, creator_id, created_ts, updater_id, updated_ts, environment_id, name, engine, engine_version, external_link, host, port, database
	`
	var instanceRaw instanceRaw
	if err := tx.QueryRowContext(ctx, query,
		create.CreatorID,
		create.CreatorID,
		create.EnvironmentID,
		create.Name,
		create.Engine,
		create.ExternalLink,
		create.Host,
		create.Port,
		create.Database,
		resourceID,
	).Scan(
		&instanceRaw.ID,
		&instanceRaw.ResourceID,
		&instanceRaw.RowStatus,
		&instanceRaw.CreatorID,
		&instanceRaw.CreatedTs,
		&instanceRaw.UpdaterID,
		&instanceRaw.UpdatedTs,
		&instanceRaw.EnvironmentID,
		&instanceRaw.Name,
		&instanceRaw.Engine,
		&instanceRaw.EngineVersion,
		&instanceRaw.ExternalLink,
		&instanceRaw.Host,
		&instanceRaw.Port,
		&instanceRaw.Database,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, common.FormatDBErrorEmptyRowWithQuery(query)
		}
		return nil, FormatError(err)
	}
	return &instanceRaw, nil
}

func findInstanceImpl(ctx context.Context, tx *Tx, find *api.InstanceFind) ([]*instanceRaw, error) {
	where, args := findInstanceQuery(find)

	rows, err := tx.QueryContext(ctx, `
		SELECT
			id,
			resource_id,
			row_status,
			creator_id,
			created_ts,
			updater_id,
			updated_ts,
			environment_id,
			name,
			engine,
			engine_version,
			external_link,
			host,
			port,
			database
		FROM instance
		WHERE `+where,
		args...,
	)
	if err != nil {
		return nil, FormatError(err)
	}
	defer rows.Close()

	// Iterate over result set and deserialize rows into instanceRawList.
	var instanceRawList []*instanceRaw
	for rows.Next() {
		var instanceRaw instanceRaw
		if err := rows.Scan(
			&instanceRaw.ID,
			&instanceRaw.ResourceID,
			&instanceRaw.RowStatus,
			&instanceRaw.CreatorID,
			&instanceRaw.CreatedTs,
			&instanceRaw.UpdaterID,
			&instanceRaw.UpdatedTs,
			&instanceRaw.EnvironmentID,
			&instanceRaw.Name,
			&instanceRaw.Engine,
			&instanceRaw.EngineVersion,
			&instanceRaw.ExternalLink,
			&instanceRaw.Host,
			&instanceRaw.Port,
			&instanceRaw.Database,
		); err != nil {
			return nil, FormatError(err)
		}
		instanceRawList = append(instanceRawList, &instanceRaw)
	}
	if err := rows.Err(); err != nil {
		return nil, FormatError(err)
	}

	return instanceRawList, nil
}

// patchInstanceImpl updates an instance by ID. Returns the new state of the instance after update.
func patchInstanceImpl(ctx context.Context, tx *Tx, patch *InstancePatch) (*instanceRaw, error) {
	// Build UPDATE clause.
	set, args := []string{"updater_id = $1"}, []interface{}{patch.UpdaterID}
	if v := patch.RowStatus; v != nil {
		set, args = append(set, fmt.Sprintf("row_status = $%d", len(args)+1)), append(args, api.RowStatus(*v))
	}
	if v := patch.Name; v != nil {
		set, args = append(set, fmt.Sprintf("name = $%d", len(args)+1)), append(args, *v)
	}
	if v := patch.EngineVersion; v != nil {
		set, args = append(set, fmt.Sprintf("engine_version = $%d", len(args)+1)), append(args, *v)
	}
	if v := patch.ExternalLink; v != nil {
		set, args = append(set, fmt.Sprintf("external_link = $%d", len(args)+1)), append(args, *v)
	}
	if v := patch.Host; v != nil {
		set, args = append(set, fmt.Sprintf("host = $%d", len(args)+1)), append(args, *v)
	}
	if v := patch.Port; v != nil {
		set, args = append(set, fmt.Sprintf("port = $%d", len(args)+1)), append(args, *v)
	}
	if v := patch.Database; v != nil {
		set, args = append(set, fmt.Sprintf("database = $%d", len(args)+1)), append(args, *v)
	}

	args = append(args, patch.ID)

	var instanceRaw instanceRaw
	// Execute update query with RETURNING.
	if err := tx.QueryRowContext(ctx, fmt.Sprintf(`
		UPDATE instance
		SET `+strings.Join(set, ", ")+`
		WHERE id = $%d
		RETURNING id, resource_id, row_status, creator_id, created_ts, updater_id, updated_ts, environment_id, name, engine, engine_version, external_link, host, port, database
	`, len(args)),
		args...,
	).Scan(
		&instanceRaw.ID,
		&instanceRaw.ResourceID,
		&instanceRaw.RowStatus,
		&instanceRaw.CreatorID,
		&instanceRaw.CreatedTs,
		&instanceRaw.UpdaterID,
		&instanceRaw.UpdatedTs,
		&instanceRaw.EnvironmentID,
		&instanceRaw.Name,
		&instanceRaw.Engine,
		&instanceRaw.EngineVersion,
		&instanceRaw.ExternalLink,
		&instanceRaw.Host,
		&instanceRaw.Port,
		&instanceRaw.Database,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, &common.Error{Code: common.NotFound, Err: errors.Errorf("instance ID not found: %d", patch.ID)}
		}
		return nil, FormatError(err)
	}
	return &instanceRaw, nil
}

func findInstanceQuery(find *api.InstanceFind) (string, []interface{}) {
	// Build WHERE clause.
	where, args := []string{"1 = 1"}, []interface{}{}
	if v := find.ID; v != nil {
		where, args = append(where, fmt.Sprintf("id = $%d", len(args)+1)), append(args, *v)
	}
	if v := find.RowStatus; v != nil {
		where, args = append(where, fmt.Sprintf("row_status = $%d", len(args)+1)), append(args, *v)
	}
	if v := find.EnvironmentID; v != nil {
		where, args = append(where, fmt.Sprintf("environment_id = $%d", len(args)+1)), append(args, *v)
	}
	if v := find.Host; v != nil {
		where, args = append(where, fmt.Sprintf("host = $%d", len(args)+1)), append(args, *v)
	}
	if v := find.Port; v != nil {
		where, args = append(where, fmt.Sprintf("port = $%d", len(args)+1)), append(args, *v)
	}
	if v := find.Name; v != nil {
		where, args = append(where, fmt.Sprintf("name = $%d", len(args)+1)), append(args, *v)
	}

	return strings.Join(where, " AND "), args
}

// InstanceMessage is the mssage for instance.
type InstanceMessage struct {
	EnvironmentID string
	UID           int
	ResourceID    string
	Title         string
	Engine        db.Type
	EngineVersion string
	ExternalLink  string
	Deleted       bool
	DataSources   []*DataSourceMessage
}

// UpdateInstanceMessage is the mssage for updating an instance.
type UpdateInstanceMessage struct {
	UpdaterID     int
	EnvironmentID string
	ResourceID    string

	Title        *string
	ExternalLink *string
	RowStatus    *api.RowStatus
	DataSources  []*DataSourceMessage
}

// FindInstanceMessage is the message for finding instances.
type FindInstanceMessage struct {
	UID           *int
	EnvironmentID *string
	ResourceID    *string
	ShowDeleted   bool
}

// GetInstanceV2 gets an instance by the resource_id.
func (s *Store) GetInstanceV2(ctx context.Context, find *FindInstanceMessage) (*InstanceMessage, error) {
	if find.EnvironmentID != nil && find.ResourceID != nil {
		if instance, ok := s.instanceCache.Load(getInstanceCacheKey(*find.EnvironmentID, *find.ResourceID)); ok {
			return instance.(*InstanceMessage), nil
		}
	}
	if find.UID != nil {
		if instance, ok := s.instanceIDCache.Load(*find.UID); ok {
			return instance.(*InstanceMessage), nil
		}
	}

	// We will always return the resource regardless of its deleted state.
	find.ShowDeleted = true
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.Rollback()

	instances, err := s.listInstanceImplV2(ctx, tx, find)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to list instances with find instance message %+v", find)
	}
	if len(instances) == 0 {
		return nil, nil
	}
	if len(instances) > 1 {
		return nil, errors.Errorf("find %d instances with find instance message %+v, expected 1", len(instances), find)
	}
	if err := tx.Commit(); err != nil {
		return nil, FormatError(err)
	}

	instance := instances[0]
	s.instanceCache.Store(getInstanceCacheKey(instance.EnvironmentID, instance.ResourceID), instance)
	s.instanceIDCache.Store(instance.UID, instance)
	return instance, nil
}

// ListInstancesV2 lists all instance.
func (s *Store) ListInstancesV2(ctx context.Context, find *FindInstanceMessage) ([]*InstanceMessage, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.Rollback()

	instances, err := s.listInstanceImplV2(ctx, tx, find)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, FormatError(err)
	}

	for _, instance := range instances {
		s.instanceCache.Store(getInstanceCacheKey(instance.EnvironmentID, instance.ResourceID), instance)
		s.instanceIDCache.Store(instance.UID, instance)
	}
	return instances, nil
}

// CreateInstanceV2 creates the instance.
func (s *Store) CreateInstanceV2(ctx context.Context, environmentID string, instanceCreate *InstanceMessage, creatorID int) (*InstanceMessage, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.Rollback()

	// TODO(d): use the same query for environment.
	environment, err := s.getEnvironmentImplV2(ctx, tx, &FindEnvironmentMessage{
		ResourceID: &environmentID,
	})
	if err != nil {
		return nil, err
	}
	if environment == nil {
		return nil, common.Errorf(common.NotFound, "environment %s not found", environmentID)
	}

	var instanceID int
	if err := tx.QueryRowContext(ctx, `
			INSERT INTO instance (
				resource_id,
				creator_id,
				updater_id,
				environment_id,
				name,
				engine,
				external_link,
				host,
				port
			)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
			RETURNING id
		`,
		instanceCreate.ResourceID,
		creatorID,
		creatorID,
		environment.UID,
		instanceCreate.Title,
		instanceCreate.Engine,
		instanceCreate.ExternalLink,
		"",
		"",
	).Scan(&instanceID); err != nil {
		return nil, FormatError(err)
	}

	for _, ds := range instanceCreate.DataSources {
		if err := s.addDataSourceToInstanceImplV2(ctx, tx, instanceID, creatorID, ds); err != nil {
			return nil, err
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, FormatError(err)
	}

	instance := &InstanceMessage{
		EnvironmentID: environmentID,
		ResourceID:    instanceCreate.ResourceID,
		UID:           instanceID,
		Title:         instanceCreate.Title,
		Engine:        instanceCreate.Engine,
		ExternalLink:  instanceCreate.ExternalLink,
		DataSources:   instanceCreate.DataSources,
	}
	s.instanceCache.Store(getInstanceCacheKey(instance.EnvironmentID, instance.ResourceID), instance)
	s.instanceIDCache.Store(instance.UID, instance)
	return instance, nil
}

// UpdateInstanceV2 updates an instance.
func (s *Store) UpdateInstanceV2(ctx context.Context, patch *UpdateInstanceMessage) (*InstanceMessage, error) {
	set, args := []string{"updater_id = $1"}, []interface{}{fmt.Sprintf("%d", patch.UpdaterID)}
	if v := patch.Title; v != nil {
		set, args = append(set, fmt.Sprintf("name = $%d", len(args)+1)), append(args, *v)
	}
	if v := patch.ExternalLink; v != nil {
		set, args = append(set, fmt.Sprintf("external_link = $%d", len(args)+1)), append(args, *v)
	}
	if v := patch.RowStatus; v != nil {
		set, args = append(set, fmt.Sprintf("row_status = $%d", len(args)+1)), append(args, *v)
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.Rollback()

	// TODO(d): use the same query for environment.
	environment, err := s.getEnvironmentImplV2(ctx, tx, &FindEnvironmentMessage{
		ResourceID: &patch.EnvironmentID,
	})
	if err != nil {
		return nil, err
	}
	if environment == nil {
		return nil, common.Errorf(common.NotFound, "environment %s not found", patch.EnvironmentID)
	}

	args = append(args, patch.ResourceID, environment.UID)

	instance := &InstanceMessage{
		EnvironmentID: patch.EnvironmentID,
	}
	var rowStatus string
	if err := tx.QueryRowContext(ctx, fmt.Sprintf(`
			UPDATE instance
			SET `+strings.Join(set, ", ")+`
			WHERE resource_id = $%d AND environment_id = $%d
			RETURNING
				id,
				resource_id,
				name,
				engine,
				external_link,
				row_status
		`, len(args)-1, len(args)),
		args...,
	).Scan(
		&instance.UID,
		&instance.ResourceID,
		&instance.Title,
		&instance.Engine,
		&instance.ExternalLink,
		&rowStatus,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, FormatError(err)
	}

	if patch.DataSources != nil {
		allDatabaseName := api.AllDatabaseName
		databaseList, err := s.listDatabaseImplV2(ctx, tx, &FindDatabaseMessage{
			InstanceID:         &instance.ResourceID,
			DatabaseName:       &allDatabaseName,
			IncludeAllDatabase: true,
		})
		if err != nil {
			return nil, err
		}
		if len(databaseList) == 0 {
			return nil, &common.Error{Code: common.NotFound, Err: errors.Errorf("data source database not found for instance %q", instance.ResourceID)}
		}
		if len(databaseList) > 1 {
			return nil, &common.Error{Code: common.NotFound, Err: errors.Errorf("found %d data source databases for instance %q", len(databaseList), instance.ResourceID)}
		}
		allDatabase := databaseList[0]

		if err := s.clearDataSourceImpl(ctx, tx, instance.UID, allDatabase.UID); err != nil {
			return nil, err
		}

		for _, ds := range patch.DataSources {
			if err := s.addDataSourceToInstanceImplV2(ctx, tx, instance.UID, patch.UpdaterID, ds); err != nil {
				return nil, err
			}
		}
	}
	instance.Deleted = convertRowStatusToDeleted(rowStatus)
	dataSourceList, err := s.listDataSourceV2(ctx, tx, patch.ResourceID)
	if err != nil {
		return nil, FormatError(err)
	}
	instance.DataSources = dataSourceList

	if err := tx.Commit(); err != nil {
		return nil, FormatError(err)
	}

	s.instanceCache.Store(getInstanceCacheKey(instance.EnvironmentID, instance.ResourceID), instance)
	s.instanceIDCache.Store(instance.UID, instance)
	return instance, nil
}

func (s *Store) listInstanceImplV2(ctx context.Context, tx *Tx, find *FindInstanceMessage) ([]*InstanceMessage, error) {
	where, args := []string{"1 = 1"}, []interface{}{}
	if v := find.EnvironmentID; v != nil {
		where, args = append(where, fmt.Sprintf("environment.resource_id = $%d", len(args)+1)), append(args, *v)
	}
	if v := find.ResourceID; v != nil {
		where, args = append(where, fmt.Sprintf("instance.resource_id = $%d", len(args)+1)), append(args, *v)
	}
	if v := find.UID; v != nil {
		where, args = append(where, fmt.Sprintf("instance.id = $%d", len(args)+1)), append(args, *v)
	}
	if !find.ShowDeleted {
		where, args = append(where, fmt.Sprintf("instance.row_status = $%d", len(args)+1)), append(args, api.Normal)
	}

	var instanceMessages []*InstanceMessage
	rows, err := tx.QueryContext(ctx, `
		SELECT
			environment.resource_id as environment_id,
			instance.id AS instance_uid,
			instance.resource_id AS resource_id,
			instance.name AS name,
			engine,
			engine_version,
			external_link,
			instance.row_status AS row_status
		FROM instance
		LEFT JOIN environment ON environment.id = instance.environment_id
		WHERE `+strings.Join(where, " AND "),
		args...,
	)
	if err != nil {
		return nil, FormatError(err)
	}
	defer rows.Close()
	for rows.Next() {
		var instanceMessage InstanceMessage
		var rowStatus string
		if err := rows.Scan(
			&instanceMessage.EnvironmentID,
			&instanceMessage.UID,
			&instanceMessage.ResourceID,
			&instanceMessage.Title,
			&instanceMessage.Engine,
			&instanceMessage.EngineVersion,
			&instanceMessage.ExternalLink,
			&rowStatus,
		); err != nil {
			return nil, FormatError(err)
		}
		instanceMessage.Deleted = convertRowStatusToDeleted(rowStatus)
		instanceMessages = append(instanceMessages, &instanceMessage)
	}

	for _, instanceMessage := range instanceMessages {
		dataSourceList, err := s.listDataSourceV2(ctx, tx, instanceMessage.ResourceID)
		if err != nil {
			return nil, FormatError(err)
		}
		instanceMessage.DataSources = dataSourceList
	}

	return instanceMessages, nil
}

// FindInstanceWithDatabaseBackupEnabled finds instances with at least one database who enables backup policy.
func (s *Store) FindInstanceWithDatabaseBackupEnabled(ctx context.Context) ([]*InstanceMessage, error) {
	rows, err := s.db.db.QueryContext(ctx, `
		SELECT DISTINCT
			instance.id
		FROM instance
		JOIN db ON db.instance_id = instance.id
		JOIN backup_setting AS bs ON db.id = bs.database_id
		WHERE bs.enabled = true AND instance.row_status = $1
	`, api.Normal)
	if err != nil {
		return nil, FormatError(err)
	}
	defer rows.Close()
	var instanceUIDs []int
	for rows.Next() {
		var instanceUID int
		if err := rows.Scan(
			&instanceUID,
		); err != nil {
			return nil, FormatError(err)
		}
		instanceUIDs = append(instanceUIDs, instanceUID)
	}
	if err := rows.Err(); err != nil {
		return nil, FormatError(err)
	}

	var instances []*InstanceMessage
	for _, instanceUID := range instanceUIDs {
		instance, err := s.GetInstanceV2(ctx, &FindInstanceMessage{UID: &instanceUID})
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get instance %v", instanceUID)
		}
		if instance == nil {
			continue
		}
		instances = append(instances, instance)
	}
	return instances, nil
}
