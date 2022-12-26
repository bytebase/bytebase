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
	"github.com/bytebase/bytebase/metric"
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
	ID int

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
		ID: raw.ID,

		// Standard fields
		RowStatus: raw.RowStatus,
		CreatorID: raw.CreatorID,
		CreatedTs: raw.CreatedTs,
		UpdaterID: raw.UpdaterID,
		UpdatedTs: raw.UpdatedTs,

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
	return instance, nil
}

// CountInstance counts the number of instances.
func (s *Store) CountInstance(ctx context.Context, find *api.InstanceFind) (int, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, FormatError(err)
	}
	defer tx.Rollback()

	where, args := findInstanceQuery(find)

	query := `SELECT COUNT(*) FROM instance WHERE ` + where
	var count int
	if err := tx.QueryRowContext(ctx, query, args...).Scan(&count); err != nil {
		if err == sql.ErrNoRows {
			return 0, common.FormatDBErrorEmptyRowWithQuery(query)
		}
		return 0, FormatError(err)
	}
	return count, nil
}

// CountInstanceGroupByEngineAndEnvironmentID counts the number of instances and group by engine and environment_id.
// Used by the metric collector.
func (s *Store) CountInstanceGroupByEngineAndEnvironmentID(ctx context.Context) ([]*metric.InstanceCountMetric, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.Rollback()

	rows, err := tx.QueryContext(ctx, `
		SELECT engine, environment_id, row_status, COUNT(*)
		FROM instance
		WHERE (id <= 101 AND updater_id != 1) OR id > 101
		GROUP BY engine, environment_id, row_status`,
	)
	if err != nil {
		return nil, FormatError(err)
	}
	defer rows.Close()

	var res []*metric.InstanceCountMetric

	for rows.Next() {
		var metric metric.InstanceCountMetric
		if err := rows.Scan(&metric.Engine, &metric.EnvironmentID, &metric.RowStatus, &metric.Count); err != nil {
			return nil, FormatError(err)
		}
		res = append(res, &metric)
	}
	if err := rows.Err(); err != nil {
		return nil, FormatError(err)
	}

	return res, nil
}

// FindInstanceWithDatabaseBackupEnabled finds instances with at least one database who enables backup policy.
func (s *Store) FindInstanceWithDatabaseBackupEnabled(ctx context.Context, engineType db.Type) ([]*api.Instance, error) {
	rows, err := s.db.db.QueryContext(ctx, `
		SELECT DISTINCT
			instance.id,
			instance.row_status,
			instance.creator_id,
			instance.created_ts,
			instance.updater_id,
			instance.updated_ts,
			instance.environment_id,
			instance.name,
			instance.engine,
			instance.engine_version,
			instance.external_link,
			instance.host,
			instance.port,
			instance.database
		FROM instance
		JOIN db ON db.instance_id = instance.id
		JOIN backup_setting AS bs ON db.id = bs.database_id
		WHERE bs.enabled = true AND instance.row_status = $1 AND instance.engine = $2
	`, api.Normal, engineType)

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

// GetInstanceAdminPasswordByID gets admin password of instance.
func (s *Store) GetInstanceAdminPasswordByID(ctx context.Context, instanceID int) (string, error) {
	dataSourceFind := &api.DataSourceFind{
		InstanceID: &instanceID,
	}
	dataSourceRawList, err := s.findDataSource(ctx, dataSourceFind)
	if err != nil {
		return "", err
	}
	for _, dataSourceRaw := range dataSourceRawList {
		if dataSourceRaw.Type == api.Admin {
			return dataSourceRaw.Password, nil
		}
	}
	return "", &common.Error{Code: common.NotFound, Err: errors.Errorf("missing admin password for instance with ID %d", instanceID)}
}

// GetInstanceSslSuiteByID gets ssl suite of instance.
func (s *Store) GetInstanceSslSuiteByID(ctx context.Context, instanceID int) (db.TLSConfig, error) {
	dataSourceFind := &api.DataSourceFind{
		InstanceID: &instanceID,
	}
	dataSourceRawList, err := s.findDataSource(ctx, dataSourceFind)
	if err != nil {
		return db.TLSConfig{}, err
	}
	for _, dataSourceRaw := range dataSourceRawList {
		return db.TLSConfig{
			SslCA:   dataSourceRaw.SslCa,
			SslKey:  dataSourceRaw.SslKey,
			SslCert: dataSourceRaw.SslCert,
		}, nil
	}
	return db.TLSConfig{}, &common.Error{Code: common.NotFound, Err: errors.Errorf("missing ssl suite for instance with ID %d", instanceID)}
}

//
// private function
//

func (s *Store) composeInstance(ctx context.Context, raw *instanceRaw) (*api.Instance, error) {
	instance := raw.toInstance()

	creator, err := s.GetPrincipalByID(ctx, instance.CreatorID)
	if err != nil {
		return nil, err
	}
	instance.Creator = creator

	updater, err := s.GetPrincipalByID(ctx, instance.UpdaterID)
	if err != nil {
		return nil, err
	}
	instance.Updater = updater

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
	for _, dataSource := range instance.DataSourceList {
		if dataSource.Creator, err = s.GetPrincipalByID(ctx, dataSource.CreatorID); err != nil {
			return nil, err
		}
		if dataSource.Updater, err = s.GetPrincipalByID(ctx, dataSource.UpdaterID); err != nil {
			return nil, err
		}
	}

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
	databaseCreate := &api.DatabaseCreate{
		CreatorID:     create.CreatorID,
		ProjectID:     api.DefaultProjectID,
		InstanceID:    instance.ID,
		EnvironmentID: instance.EnvironmentID,
		Name:          api.AllDatabaseName,
		CharacterSet:  api.DefaultCharacterSetName,
		Collation:     api.DefaultCollationName,
	}
	allDatabase, err := s.createDatabaseRawTx(ctx, tx, databaseCreate)
	if err != nil {
		return nil, err
	}

	for _, dataSource := range create.DataSourceList {
		dataSourceCreate := &api.DataSourceCreate{
			CreatorID:    create.CreatorID,
			InstanceID:   instance.ID,
			DatabaseID:   allDatabase.ID,
			Name:         dataSource.Name,
			Type:         dataSource.Type,
			Username:     dataSource.Username,
			Password:     dataSource.Password,
			SslKey:       dataSource.SslKey,
			SslCert:      dataSource.SslCert,
			SslCa:        dataSource.SslCa,
			HostOverride: dataSource.HostOverride,
			PortOverride: dataSource.PortOverride,
			Options:      dataSource.Options,
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
	tx, err := s.db.BeginTx(ctx, nil)
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

	tx, err := s.db.BeginTx(ctx, nil)
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
		dbName := api.AllDatabaseName
		dbFind := &api.DatabaseFind{
			InstanceID:         &patch.ID,
			Name:               &dbName,
			IncludeAllDatabase: true,
		}
		databaseList, err := s.findDatabaseImpl(ctx, tx, dbFind)
		if err != nil {
			return nil, err
		}
		if len(databaseList) == 0 {
			return nil, &common.Error{Code: common.NotFound, Err: errors.Errorf("cannot find database with filter %+v. ", dbFind)}
		}
		if len(databaseList) > 1 {
			return nil, &common.Error{Code: common.Conflict, Err: errors.Errorf("found %d databases with filter %+v, expect 1. ", len(databaseList), dbFind)}
		}
		database := databaseList[0]

		if err := s.clearDataSourceImpl(ctx, tx, patch.ID, database.ID); err != nil {
			return nil, err
		}
		s.cache.DeleteCache(dataSourceCacheNamespace, patch.ID)

		for _, dataSource := range patch.DataSourceList {
			dataSourceCreate := &api.DataSourceCreate{
				CreatorID:    patch.UpdaterID,
				InstanceID:   instance.ID,
				DatabaseID:   database.ID,
				Name:         dataSource.Name,
				Type:         dataSource.Type,
				Username:     dataSource.Username,
				Password:     dataSource.Password,
				SslKey:       dataSource.SslKey,
				SslCert:      dataSource.SslCert,
				SslCa:        dataSource.SslCa,
				HostOverride: dataSource.HostOverride,
				PortOverride: dataSource.PortOverride,
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
		RETURNING id, row_status, creator_id, created_ts, updater_id, updated_ts, environment_id, name, engine, engine_version, external_link, host, port, database
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
		RETURNING id, row_status, creator_id, created_ts, updater_id, updated_ts, environment_id, name, engine, engine_version, external_link, host, port, database
	`, len(args)),
		args...,
	).Scan(
		&instanceRaw.ID,
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
