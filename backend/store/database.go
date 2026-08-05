package store

import (
	"context"
	"database/sql"
	"fmt"
	"slices"
	"strings"

	"github.com/google/cel-go/cel"
	celast "github.com/google/cel-go/common/ast"
	celoperators "github.com/google/cel-go/common/operators"
	celoverloads "github.com/google/cel-go/common/overloads"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/qb"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

// DatabaseMessage is the message for database.
type DatabaseMessage struct {
	ProjectID         string
	InstanceProjectID *string
	InstanceID        string
	DatabaseName      string

	EnvironmentID          *string
	EffectiveEnvironmentID *string

	Deleted  bool
	Metadata *storepb.DatabaseMetadata
	Engine   storepb.Engine
}

func (d *DatabaseMessage) String() string {
	return common.FormatDatabase(d.InstanceID, d.DatabaseName)
}

// ResourceName returns the canonical database resource name for the owning instance scope.
func (d *DatabaseMessage) ResourceName() string {
	if d.InstanceProjectID != nil {
		return common.FormatProjectDatabase(*d.InstanceProjectID, d.InstanceID, d.DatabaseName)
	}
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
	// Workspace scopes the update to databases whose instance belongs to this workspace.
	// Empty string skips filtering (cross-workspace).
	Workspace           string
	ProjectID           *string
	FindByEnvironmentID *string
	// Empty string will unset the environment.
	EnvironmentID *string
}

// FindDatabaseMessage is the message for finding databases.
type FindDatabaseMessage struct {
	// Workspace filters databases by the parent instance's workspace.
	// Empty string skips filtering (for cross-workspace queries like runners).
	Workspace              string
	ProjectID              *string
	EffectiveEnvironmentID *string
	InstanceID             *string
	DatabaseName           *string
	DatabaseNames          []string
	Engine                 *storepb.Engine
	// When this is used, we will return databases from archived instances or environments.
	// This is used for existing tasks with archived databases.
	ShowDeleted bool

	FilterQ     *qb.Query
	Limit       *int
	Offset      *int
	OrderByKeys []*OrderByKey
}

// removeDatabaseCache invalidates both workspace-scoped and unscoped cache entries for a database.
func (s *Store) removeDatabaseCache(ctx context.Context, instanceID, databaseName string) {
	// Remove unscoped (runner) cache entry.
	s.databaseCache.Remove(getDatabaseCacheKey("", instanceID, databaseName))
	// Remove workspace-scoped (API) cache entry.
	// Query workspace directly from instance table to avoid workspace-filtering issues with GetInstance.
	var workspace string
	if err := s.GetDB().QueryRowContext(ctx,
		"SELECT workspace FROM instance WHERE resource_id = $1", instanceID,
	).Scan(&workspace); err != nil {
		return
	}
	s.databaseCache.Remove(getDatabaseCacheKey(workspace, instanceID, databaseName))
}

// GetDatabase gets a database.
func (s *Store) GetDatabase(ctx context.Context, find *FindDatabaseMessage) (*DatabaseMessage, error) {
	if find.InstanceID != nil && find.DatabaseName != nil {
		if v, ok := s.databaseCache.Get(getDatabaseCacheKey(find.Workspace, *find.InstanceID, *find.DatabaseName)); ok && s.enableCache {
			return v, nil
		}
	}

	databases, err := s.ListDatabases(ctx, find)
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

	s.databaseCache.Add(getDatabaseCacheKey(find.Workspace, database.InstanceID, database.DatabaseName), database)
	return database, nil
}

// ListDatabases lists all databases.
func (s *Store) ListDatabases(ctx context.Context, find *FindDatabaseMessage) ([]*DatabaseMessage, error) {
	from := qb.Q().Space("db")
	where := qb.Q().Space("TRUE")

	if filterQ := find.FilterQ; filterQ != nil {
		where.And("?", filterQ)
		// Check if the filter requires the db_schema table for table filtering
		sql, _, err := filterQ.ToSQL()
		if err == nil && strings.Contains(sql, "ds.metadata") {
			from.Space("INNER JOIN db_schema ds ON db.instance = ds.instance AND db.name = ds.db_name")
		}
	}

	from.Space("LEFT JOIN instance ON db.instance = instance.resource_id")

	if find.Workspace != "" {
		where.And("instance.workspace = ?", find.Workspace)
	}
	if v := find.ProjectID; v != nil {
		where.And("db.project = ?", *v)
	}
	if v := find.EffectiveEnvironmentID; v != nil {
		where.And(`COALESCE(
			db.environment,
			instance.environment
		) = ?`, *v)
	}
	if v := find.InstanceID; v != nil {
		where.And("db.instance = ?", *v)
	}
	if v := find.DatabaseName; v != nil {
		where.And("db.name = ?", *v)
	}
	if len(find.DatabaseNames) > 0 {
		where.And("db.name = ANY(?)", find.DatabaseNames)
	}
	if v := find.Engine; v != nil {
		where.And("instance.metadata->>'engine' = ?", v.String())
	}
	if !find.ShowDeleted {
		where.And("instance.deleted = ?", false)
		where.And("db.deleted = ?", false)
	}

	q := qb.Q().Space(`
		SELECT
			db.project,
			instance.project,
			COALESCE(
				db.environment,
				instance.environment
			),
			db.environment,
			db.instance,
			db.name,
			db.deleted,
			db.metadata,
			instance.metadata->>'engine'
		FROM ?
		WHERE ?
	`, from, where)

	if len(find.OrderByKeys) > 0 {
		orderBy := []string{}
		for _, v := range find.OrderByKeys {
			orderBy = append(orderBy, fmt.Sprintf("%s %s", v.Key, v.SortOrder.String()))
		}
		q.Space(fmt.Sprintf("ORDER BY %s", strings.Join(orderBy, ", ")))
	} else {
		q.Space("ORDER BY db.project, db.instance, db.name")
	}

	if v := find.Limit; v != nil {
		q.Space("LIMIT ?", *v)
	}
	if v := find.Offset; v != nil {
		q.Space("OFFSET ?", *v)
	}

	query, args, err := q.ToSQL()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to build sql %+v", q)
	}

	var databases []*DatabaseMessage
	rows, err := s.GetDB().QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		databaseMessage := &DatabaseMessage{}
		var metadataString string
		var instanceProject, effectiveEnvironment, environment, engine sql.NullString
		if err := rows.Scan(
			&databaseMessage.ProjectID,
			&instanceProject,
			&effectiveEnvironment,
			&environment,
			&databaseMessage.InstanceID,
			&databaseMessage.DatabaseName,
			&databaseMessage.Deleted,
			&metadataString,
			&engine,
		); err != nil {
			return nil, err
		}
		if instanceProject.Valid {
			databaseMessage.InstanceProjectID = &instanceProject.String
		}
		if effectiveEnvironment.Valid {
			databaseMessage.EffectiveEnvironmentID = &effectiveEnvironment.String
		}
		if environment.Valid {
			databaseMessage.EnvironmentID = &environment.String
		}
		if engine.Valid {
			if v, ok := storepb.Engine_value[engine.String]; ok {
				databaseMessage.Engine = storepb.Engine(v)
			}
		}

		var metadata storepb.DatabaseMetadata
		if err := common.ProtojsonUnmarshaler.Unmarshal([]byte(metadataString), &metadata); err != nil {
			return nil, err
		}
		databaseMessage.Metadata = &metadata

		databases = append(databases, databaseMessage)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	for _, database := range databases {
		s.databaseCache.Add(getDatabaseCacheKey(find.Workspace, database.InstanceID, database.DatabaseName), database)
	}
	return databases, nil
}

// CreateDatabaseDefault creates a database discovered by schema sync.
func (s *Store) CreateDatabaseDefault(ctx context.Context, create *DatabaseMessage) (*DatabaseMessage, error) {
	err := s.withDatabasePurgeFence(ctx, create.InstanceID, create.DatabaseName, create.ProjectID, nil, func(tx *sql.Tx, ownership *databaseOwnership) error {
		projectID, err := ownership.projectForDefaultCreate(create.ProjectID)
		if err != nil {
			return err
		}
		query, args, err := qb.Q().Space(`INSERT INTO db (instance, project, name, deleted)
			VALUES (?, ?, ?, ?)
			ON CONFLICT (instance, name) DO UPDATE SET deleted = EXCLUDED.deleted`,
			create.InstanceID, projectID, create.DatabaseName, false).ToSQL()
		if err != nil {
			return errors.Wrap(err, "failed to build sql")
		}
		_, err = tx.ExecContext(ctx, query, args...)
		return err
	})
	if err != nil {
		return nil, err
	}
	s.removeDatabaseCache(ctx, create.InstanceID, create.DatabaseName)
	return s.GetDatabase(ctx, &FindDatabaseMessage{InstanceID: &create.InstanceID, DatabaseName: &create.DatabaseName, ShowDeleted: true})
}

// UpsertDatabase upserts a database.
func (s *Store) UpsertDatabase(ctx context.Context, create *DatabaseMessage) (*DatabaseMessage, error) {
	metadata, err := protojson.Marshal(create.Metadata)
	if err != nil {
		return nil, err
	}
	var environment *string
	if create.EnvironmentID != nil && *create.EnvironmentID != "" {
		environment = create.EnvironmentID
	}
	err = s.withDatabasePurgeFence(ctx, create.InstanceID, create.DatabaseName, create.ProjectID, nil, func(tx *sql.Tx, ownership *databaseOwnership) error {
		projectID, err := ownership.projectForUpsert(create.ProjectID)
		if err != nil {
			return err
		}
		query, args, err := qb.Q().Space(`INSERT INTO db (instance, project, environment, name, deleted, metadata)
			VALUES (?, ?, ?, ?, ?, ?)
			ON CONFLICT (instance, name) DO UPDATE SET
				project = EXCLUDED.project, environment = EXCLUDED.environment,
				name = EXCLUDED.name, metadata = EXCLUDED.metadata`,
			create.InstanceID, projectID, environment, create.DatabaseName, create.Deleted, metadata).ToSQL()
		if err != nil {
			return errors.Wrap(err, "failed to build sql")
		}
		_, err = tx.ExecContext(ctx, query, args...)
		return err
	})
	if err != nil {
		return nil, err
	}
	s.removeDatabaseCache(ctx, create.InstanceID, create.DatabaseName)
	return s.GetDatabase(ctx, &FindDatabaseMessage{InstanceID: &create.InstanceID, DatabaseName: &create.DatabaseName, ShowDeleted: true})
}

// UpdateDatabase updates a database.
func (s *Store) UpdateDatabase(ctx context.Context, patch *UpdateDatabaseMessage) (*DatabaseMessage, error) {
	requestedProjectID := ""
	if patch.ProjectID != nil {
		requestedProjectID = *patch.ProjectID
	}
	err := s.withDatabasePurgeFence(ctx, patch.InstanceID, patch.DatabaseName, requestedProjectID, nil, func(tx *sql.Tx, ownership *databaseOwnership) error {
		if !ownership.exists {
			return common.Errorf(common.NotFound, "database %s not found", common.FormatDatabase(patch.InstanceID, patch.DatabaseName))
		}
		projectID, err := ownership.projectForUpdate(patch.ProjectID)
		if err != nil {
			return err
		}
		set := qb.Q()
		if patch.ProjectID != nil {
			set.Comma("project = ?", projectID)
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
		if len(patch.MetadataUpdates) > 0 {
			metadata := &storepb.DatabaseMetadata{}
			if err := common.ProtojsonUnmarshaler.Unmarshal(ownership.metadata, metadata); err != nil {
				return errors.Wrapf(err, "failed to unmarshal metadata for database %q", common.FormatDatabase(patch.InstanceID, patch.DatabaseName))
			}
			for _, update := range patch.MetadataUpdates {
				update(metadata)
			}
			metadataBytes, err := protojson.Marshal(metadata)
			if err != nil {
				return err
			}
			set.Comma("metadata = ?", metadataBytes)
		}
		if set.Len() == 0 {
			return errors.New("no fields to update")
		}
		query, args, err := qb.Q().Space("UPDATE db SET ? WHERE instance = ? AND name = ?", set, patch.InstanceID, patch.DatabaseName).ToSQL()
		if err != nil {
			return errors.Wrap(err, "failed to build sql")
		}
		_, err = tx.ExecContext(ctx, query, args...)
		return err
	})
	if err != nil {
		return nil, err
	}
	s.removeDatabaseCache(ctx, patch.InstanceID, patch.DatabaseName)
	return s.GetDatabase(ctx, &FindDatabaseMessage{InstanceID: &patch.InstanceID, DatabaseName: &patch.DatabaseName, ShowDeleted: true})
}

// BatchUpdateDatabases updates databases in batch.
//
// The batch runs in one transaction after acquiring every affected source and
// destination project purge fence and instance purge fence in deterministic
// sorted order. Without the fences, a concurrent project purge could sweep the
// destination project's rows and make the terminal project deletion fail on
// the db.project foreign key.
func (s *Store) BatchUpdateDatabases(ctx context.Context, databases []*DatabaseMessage, update *BatchUpdateDatabases) error {
	set := qb.Q()
	if update.ProjectID != nil {
		if *update.ProjectID == "" {
			return common.Errorf(common.Invalid, "database project cannot be empty")
		}
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
		return errors.New("no update field specified")
	}
	where := qb.Q()
	if v := update.FindByEnvironmentID; v != nil {
		where.Or("db.environment = ?", *v)
	}
	if len(databases) > 0 {
		instances, names := make([]string, 0, len(databases)), make([]string, 0, len(databases))
		for _, database := range databases {
			instances, names = append(instances, database.InstanceID), append(names, database.DatabaseName)
		}
		where.Or(`(db.instance, db.name) IN (SELECT * FROM unnest(?::TEXT[], ?::TEXT[]))`, instances, names)
	}
	if where.Len() == 0 {
		return errors.New("empty where")
	}
	if update.Workspace != "" {
		where.And("db.instance IN (SELECT resource_id FROM instance WHERE workspace = ?)", update.Workspace)
	}

	// Discover every affected database before acquiring purge fences so the
	// source/destination project and instance fence sets are known up front.
	// The transaction below re-reads these targets under the fences.
	targets, err := s.listDatabaseBatchTargets(ctx, where)
	if err != nil {
		return err
	}
	instances := make([]string, 0, len(targets))
	projects := make([]string, 0, len(targets))
	for _, target := range targets {
		instances = append(instances, target.instanceID)
		projects = append(projects, target.projectID)
		if target.instanceProject != "" {
			projects = append(projects, target.instanceProject)
		}
	}
	if update.ProjectID != nil {
		projects = append(projects, *update.ProjectID)
	}
	slices.Sort(instances)
	instances = slices.Compact(instances)
	slices.Sort(projects)
	projects = slices.Compact(projects)

	var updated []databaseBatchTarget
	err = s.withDatabaseBatchPurgeFence(ctx, instances, projects, func(tx *sql.Tx) error {
		locked, err := lockDatabaseBatchTargets(ctx, tx, where, targets)
		if err != nil {
			return err
		}
		instances, err := lockDatabaseBatchInstances(ctx, tx, locked)
		if err != nil {
			return err
		}
		for instanceID, instance := range instances {
			if instance.deleted {
				return common.Errorf(common.Conflict, "instance %s is archived", instanceID)
			}
		}
		// Revalidate lifecycle under the transaction: every target must still
		// be covered by the pre-read purge fences. A purge that committed
		// before the fences were acquired shows up here as a retry instead of
		// an FK failure.
		for _, target := range locked {
			if !slices.Contains(projects, target.projectID) {
				return errors.Errorf("database ownership changed to project %s for %s; retry", target.projectID, common.FormatDatabase(target.instanceID, target.databaseName))
			}
			if instanceProject := instances[target.instanceID].projectID; instanceProject != "" && !slices.Contains(projects, instanceProject) {
				return errors.Errorf("database ownership changed to project instance %s for %s; retry", instanceProject, common.FormatDatabase(target.instanceID, target.databaseName))
			}
		}
		// Lock every affected project, including the destination, after the
		// database rows and instances. A purged destination fails cleanly with
		// NotFound instead of an FK violation.
		projectIDs := make([]string, 0, len(locked)+1)
		for _, target := range locked {
			projectIDs = append(projectIDs, target.projectID)
			if instanceProject := instances[target.instanceID].projectID; instanceProject != "" {
				projectIDs = append(projectIDs, instanceProject)
			}
		}
		if update.ProjectID != nil {
			projectIDs = append(projectIDs, *update.ProjectID)
		}
		slices.Sort(projectIDs)
		projectIDs = slices.Compact(projectIDs)
		for _, projectID := range projectIDs {
			var foundProjectID string
			if err := tx.QueryRowContext(ctx, "SELECT resource_id FROM project WHERE resource_id = $1 FOR UPDATE", projectID).Scan(&foundProjectID); err != nil {
				if errors.Is(err, sql.ErrNoRows) {
					return common.Errorf(common.NotFound, "project %s not found", projectID)
				}
				return errors.Wrapf(err, "failed to lock project %s", projectID)
			}
		}
		// Atomic validation: any invalid target rejects the whole batch before
		// any row is updated.
		if update.ProjectID != nil {
			for _, target := range locked {
				if instanceProject := instances[target.instanceID].projectID; instanceProject != "" {
					if target.projectID != instanceProject || instanceProject != *update.ProjectID {
						return common.Errorf(common.Invalid, "cannot move a project instance database to another project")
					}
				}
			}
		}
		if len(locked) == 0 {
			return nil
		}
		updateInstances, updateNames := make([]string, 0, len(locked)), make([]string, 0, len(locked))
		for _, target := range locked {
			updateInstances = append(updateInstances, target.instanceID)
			updateNames = append(updateNames, target.databaseName)
		}
		q := qb.Q().Space(`
			UPDATE db SET ?
			FROM unnest(?::TEXT[], ?::TEXT[]) AS u(instance, name)
			WHERE db.instance = u.instance AND db.name = u.name
		`, set, updateInstances, updateNames)
		query, args, err := q.ToSQL()
		if err != nil {
			return errors.Wrap(err, "failed to build batch database update sql")
		}
		if _, err := tx.ExecContext(ctx, query, args...); err != nil {
			return errors.Wrap(err, "failed to update databases")
		}
		updated = locked
		return nil
	})
	if err != nil {
		return err
	}
	for _, target := range updated {
		s.removeDatabaseCache(ctx, target.instanceID, target.databaseName)
	}
	return nil
}

type databaseBatchTarget struct {
	instanceID      string
	databaseName    string
	projectID       string
	instanceProject string
}

// listDatabaseBatchTargets finds the databases matched by the batch update
// where clause. It runs outside the purge fences; the result only determines
// which fences to acquire and which rows to lock inside the transaction.
func (s *Store) listDatabaseBatchTargets(ctx context.Context, where *qb.Query) ([]databaseBatchTarget, error) {
	q := qb.Q().Space(`
		SELECT db.instance, db.name, db.project, instance.project
		FROM db
		JOIN instance ON instance.resource_id = db.instance
		WHERE ?
		ORDER BY db.instance, db.name
	`, where)
	query, args, err := q.ToSQL()
	if err != nil {
		return nil, errors.Wrap(err, "failed to build batch database target query")
	}
	rows, err := s.GetDB().QueryContext(ctx, query, args...)
	if err != nil {
		return nil, errors.Wrap(err, "failed to find batch database targets")
	}
	defer rows.Close()
	var targets []databaseBatchTarget
	for rows.Next() {
		var target databaseBatchTarget
		var instanceProject sql.NullString
		if err := rows.Scan(&target.instanceID, &target.databaseName, &target.projectID, &instanceProject); err != nil {
			return nil, errors.Wrap(err, "failed to scan batch database target")
		}
		if instanceProject.Valid {
			target.instanceProject = instanceProject.String
		}
		targets = append(targets, target)
	}
	if err := rows.Err(); err != nil {
		return nil, errors.Wrap(err, "failed to read batch database targets")
	}
	return targets, nil
}

// withDatabaseBatchPurgeFence serializes a batch database write with direct
// project and instance purge. Advisory fences are acquired in deterministic
// sorted order (all project fences, then all instance fences) before any row
// lock.
func (s *Store) withDatabaseBatchPurgeFence(
	ctx context.Context,
	instances []string,
	projects []string,
	write func(*sql.Tx) error,
) error {
	tx, err := s.GetDB().BeginTx(ctx, nil)
	if err != nil {
		return errors.Wrap(err, "failed to begin batch database write transaction")
	}
	defer tx.Rollback()
	for _, projectID := range projects {
		if err := acquireProjectPurgeLock(ctx, tx, projectID); err != nil {
			return errors.Wrapf(err, "failed to lock project purge fence for %s", projectID)
		}
	}
	for _, instanceID := range instances {
		if err := acquireInstancePurgeLock(ctx, tx, instanceID); err != nil {
			return errors.Wrapf(err, "failed to lock instance purge fence for %s", instanceID)
		}
	}
	if err := write(tx); err != nil {
		return err
	}
	return errors.Wrap(tx.Commit(), "failed to commit batch database write transaction")
}

// lockDatabaseBatchTargets locks the pre-read batch targets in full
// primary-key order and re-applies the original where clause so rows that no
// longer match are skipped. Only db rows are locked.
func lockDatabaseBatchTargets(ctx context.Context, tx *sql.Tx, where *qb.Query, targets []databaseBatchTarget) ([]databaseBatchTarget, error) {
	if len(targets) == 0 {
		return nil, nil
	}
	instances, names := make([]string, 0, len(targets)), make([]string, 0, len(targets))
	for _, target := range targets {
		instances, names = append(instances, target.instanceID), append(names, target.databaseName)
	}
	q := qb.Q().Space(`
		SELECT db.instance, db.name, db.project, instance.project
		FROM db
		JOIN instance ON instance.resource_id = db.instance
		WHERE (db.instance, db.name) IN (SELECT * FROM unnest(?::TEXT[], ?::TEXT[]))
		  AND ?
		ORDER BY db.instance, db.name
		FOR UPDATE OF db
	`, instances, names, where)
	query, args, err := q.ToSQL()
	if err != nil {
		return nil, errors.Wrap(err, "failed to build batch database lock query")
	}
	rows, err := tx.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, errors.Wrap(err, "failed to lock batch database targets")
	}
	defer rows.Close()
	var locked []databaseBatchTarget
	for rows.Next() {
		var target databaseBatchTarget
		var instanceProject sql.NullString
		if err := rows.Scan(&target.instanceID, &target.databaseName, &target.projectID, &instanceProject); err != nil {
			return nil, errors.Wrap(err, "failed to scan locked batch database target")
		}
		if instanceProject.Valid {
			target.instanceProject = instanceProject.String
		}
		locked = append(locked, target)
	}
	if err := rows.Err(); err != nil {
		return nil, errors.Wrap(err, "failed to read locked batch database targets")
	}
	return locked, nil
}

type databaseBatchInstance struct {
	projectID string
	deleted   bool
}

// lockDatabaseBatchInstances locks the instances of the locked targets in
// sorted order and returns their current lifecycle and owning project.
func lockDatabaseBatchInstances(ctx context.Context, tx *sql.Tx, targets []databaseBatchTarget) (map[string]databaseBatchInstance, error) {
	instanceIDs := make([]string, 0, len(targets))
	for _, target := range targets {
		instanceIDs = append(instanceIDs, target.instanceID)
	}
	slices.Sort(instanceIDs)
	instanceIDs = slices.Compact(instanceIDs)
	instances := make(map[string]databaseBatchInstance, len(instanceIDs))
	if len(instanceIDs) == 0 {
		return instances, nil
	}
	q := qb.Q().Space(`
		SELECT resource_id, project, deleted
		FROM instance
		WHERE resource_id IN (SELECT * FROM unnest(?::TEXT[]))
		ORDER BY resource_id
		FOR NO KEY UPDATE
	`, instanceIDs)
	query, args, err := q.ToSQL()
	if err != nil {
		return nil, errors.Wrap(err, "failed to build batch database instance lock query")
	}
	rows, err := tx.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, errors.Wrap(err, "failed to lock batch database instances")
	}
	defer rows.Close()
	for rows.Next() {
		var instanceID string
		var project sql.NullString
		var deleted bool
		if err := rows.Scan(&instanceID, &project, &deleted); err != nil {
			return nil, errors.Wrap(err, "failed to scan locked batch database instance")
		}
		instance := databaseBatchInstance{deleted: deleted}
		if project.Valid {
			instance.projectID = project.String
		}
		instances[instanceID] = instance
	}
	if err := rows.Err(); err != nil {
		return nil, errors.Wrap(err, "failed to read locked batch database instances")
	}
	for _, instanceID := range instanceIDs {
		if _, ok := instances[instanceID]; !ok {
			return nil, common.Errorf(common.NotFound, "instance %s not found", instanceID)
		}
	}
	return instances, nil
}

type databaseOwnership struct {
	exists          bool
	projectID       string
	metadata        []byte
	instanceProject *string
}

// withDatabasePurgeFence serializes a database write with direct project and
// instance purge. Database descendants may be absent, so row locks alone cannot
// prevent a purge from passing their table before the writer inserts one.
//
// Database sync intentionally supports an archived (but still existing) project
// owner. It does not support a purged or soft-deleted instance: that state is a
// direct-purge boundary and no new database data may be written through it.
func (s *Store) withDatabasePurgeFence(
	ctx context.Context,
	instanceID, databaseName, requestedProjectID string,
	lockChild func(*sql.Tx) error,
	write func(*sql.Tx, *databaseOwnership) error,
) error {
	projectID, err := s.databasePurgeProject(ctx, instanceID, databaseName, requestedProjectID)
	if err != nil {
		return err
	}

	tx, err := s.GetDB().BeginTx(ctx, nil)
	if err != nil {
		return errors.Wrap(err, "failed to begin database write transaction")
	}
	defer tx.Rollback()
	projectFences := []string{projectID}
	if requestedProjectID != "" {
		projectFences = append(projectFences, requestedProjectID)
	}
	slices.Sort(projectFences)
	projectFences = slices.Compact(projectFences)
	for _, fence := range projectFences {
		if err := acquireProjectPurgeLock(ctx, tx, fence); err != nil {
			return errors.Wrapf(err, "failed to lock project purge fence for %s", fence)
		}
	}
	if err := acquireInstancePurgeLock(ctx, tx, instanceID); err != nil {
		return errors.Wrapf(err, "failed to lock instance purge fence for %s", instanceID)
	}
	if lockChild != nil {
		if err := lockChild(tx); err != nil {
			return err
		}
	}

	ownership, err := lockDatabaseOwnership(ctx, tx, instanceID, databaseName)
	if err != nil {
		return err
	}
	if ownership.instanceProject != nil {
		projectID = *ownership.instanceProject
	} else if requestedProjectID == "" && ownership.exists {
		projectID = ownership.projectID
	}
	if !slices.Contains(projectFences, projectID) {
		return errors.Errorf("database ownership changed to project %s; retry", projectID)
	}
	for _, projectFence := range projectFences {
		var foundProjectID string
		if err := tx.QueryRowContext(ctx, "SELECT resource_id FROM project WHERE resource_id = $1 FOR UPDATE", projectFence).Scan(&foundProjectID); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return common.Errorf(common.NotFound, "project %s not found", projectFence)
			}
			return errors.Wrapf(err, "failed to lock project %s", projectFence)
		}
	}
	if err := write(tx, ownership); err != nil {
		return err
	}
	return errors.Wrap(tx.Commit(), "failed to commit database write transaction")
}

func (s *Store) databasePurgeProject(ctx context.Context, instanceID, databaseName, requestedProjectID string) (string, error) {
	var instanceProject, databaseProject sql.NullString
	if err := s.GetDB().QueryRowContext(ctx, `
		SELECT instance.project, db.project
		FROM instance
		LEFT JOIN db ON db.instance = instance.resource_id AND db.name = $2
		WHERE instance.resource_id = $1
	`, instanceID, databaseName).Scan(&instanceProject, &databaseProject); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", common.Errorf(common.NotFound, "instance %s not found", instanceID)
		}
		return "", errors.Wrapf(err, "failed to find database purge project for instance %s", instanceID)
	}
	if instanceProject.Valid {
		return instanceProject.String, nil
	}
	if databaseProject.Valid {
		return databaseProject.String, nil
	}
	if requestedProjectID != "" {
		return requestedProjectID, nil
	}
	return "", common.Errorf(common.NotFound, "database %s not found", common.FormatDatabase(instanceID, databaseName))
}

func lockDatabaseOwnership(ctx context.Context, tx *sql.Tx, instanceID, databaseName string) (*databaseOwnership, error) {
	ownership := &databaseOwnership{}
	var databaseProject sql.NullString
	if err := tx.QueryRowContext(ctx, `
		SELECT project, metadata FROM db
		WHERE instance = $1 AND name = $2
		FOR UPDATE
	`, instanceID, databaseName).Scan(&databaseProject, &ownership.metadata); err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, errors.Wrapf(err, "failed to lock database %s", common.FormatDatabase(instanceID, databaseName))
	}
	if databaseProject.Valid {
		ownership.exists = true
		ownership.projectID = databaseProject.String
	}

	var instanceProject sql.NullString
	var deleted bool
	if err := tx.QueryRowContext(ctx, `
		SELECT project, deleted FROM instance WHERE resource_id = $1 FOR NO KEY UPDATE
	`, instanceID).Scan(&instanceProject, &deleted); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, common.Errorf(common.NotFound, "instance %s not found", instanceID)
		}
		return nil, errors.Wrapf(err, "failed to lock instance %s", instanceID)
	}
	if deleted {
		return nil, common.Errorf(common.NotFound, "instance %s is deleted", instanceID)
	}
	if instanceProject.Valid {
		projectID := instanceProject.String
		ownership.instanceProject = &projectID
	}
	return ownership, nil
}

func (ownership *databaseOwnership) projectForDefaultCreate(requestedProjectID string) (string, error) {
	if ownership.instanceProject != nil {
		if ownership.exists && ownership.projectID != *ownership.instanceProject {
			return "", common.Errorf(common.Invalid, "database on project instance must belong to project %s", *ownership.instanceProject)
		}
		return *ownership.instanceProject, nil
	}
	if requestedProjectID == "" {
		return "", common.Errorf(common.Invalid, "database project cannot be empty")
	}
	return requestedProjectID, nil
}

func (ownership *databaseOwnership) projectForUpsert(requestedProjectID string) (string, error) {
	if ownership.instanceProject != nil {
		if ownership.exists && (requestedProjectID != *ownership.instanceProject || ownership.projectID != *ownership.instanceProject) {
			return "", common.Errorf(common.Invalid, "database on project instance must belong to project %s", *ownership.instanceProject)
		}
		return *ownership.instanceProject, nil
	}
	if requestedProjectID == "" {
		return "", common.Errorf(common.Invalid, "database project cannot be empty")
	}
	return requestedProjectID, nil
}

func (ownership *databaseOwnership) projectForUpdate(requestedProjectID *string) (string, error) {
	projectID := ownership.projectID
	if requestedProjectID != nil {
		projectID = *requestedProjectID
	}
	if projectID == "" {
		return "", common.Errorf(common.Invalid, "database project cannot be empty")
	}
	if ownership.instanceProject != nil && (ownership.projectID != *ownership.instanceProject || projectID != *ownership.instanceProject) {
		return "", common.Errorf(common.Invalid, "database on project instance must belong to project %s", *ownership.instanceProject)
	}
	return projectID, nil
}

func GetListDatabaseFilter(workspace, filter string) (*qb.Query, error) {
	if filter == "" {
		return nil, nil
	}

	e, err := cel.NewEnv()
	if err != nil {
		return nil, errors.Errorf("failed to create cel env")
	}
	ast, iss := e.Parse(filter)
	if iss != nil {
		return nil, errors.Errorf("failed to parse filter %v, error: %v", filter, iss.String())
	}

	var getFilter func(expr celast.Expr) (*qb.Query, error)

	parseToLabelFilterSQL := func(resource, key string, value any) (*qb.Query, error) {
		switch v := value.(type) {
		case string:
			return qb.Q().Space(fmt.Sprintf("%s->'labels'->>'%s' = ?", resource, key), v), nil
		case []any:
			if len(v) == 0 {
				return nil, errors.Errorf("empty label filter")
			}
			labelValueList := []any{}
			for _, raw := range v {
				labelValueList = append(labelValueList, raw.(string))
			}
			return qb.Q().Space(fmt.Sprintf("%s->'labels'->>'%s' = ANY(?)", resource, key), labelValueList), nil
		default:
			return nil, errors.Errorf("empty value %v for label filter", value)
		}
	}

	parseToEngineSQL := func(expr celast.Expr) (*qb.Query, error) {
		variable, value := getVariableAndValueFromExpr(expr)
		if variable != "engine" {
			return nil, errors.Errorf(`only "engine" support "engine in [xx]"/"!(engine in [xx])" operator`)
		}

		rawEngineList, ok := value.([]any)
		if !ok {
			return nil, errors.Errorf("invalid engine value %q", value)
		}
		if len(rawEngineList) == 0 {
			return nil, errors.Errorf("empty engine filter")
		}
		engineList := []any{}
		for _, rawEngine := range rawEngineList {
			engineValue, ok := storepb.Engine_value[rawEngine.(string)]
			if !ok {
				return nil, errors.Errorf("invalid engine filter %q", rawEngine)
			}
			engine := storepb.Engine(engineValue)
			engineList = append(engineList, engine.String())
		}

		return qb.Q().Space("instance.metadata->>'engine' = ANY(?)", engineList), nil
	}

	parseToSQL := func(variable, value any) (*qb.Query, error) {
		switch variable {
		case "project":
			projectID, err := common.GetProjectID(value.(string))
			if err != nil {
				return nil, errors.Errorf("invalid project filter %q", value)
			}
			return qb.Q().Space("db.project = ?", projectID), nil
		case "instance":
			instanceID, err := common.GetInstanceID(value.(string))
			if err != nil {
				return nil, errors.Errorf("invalid instance filter %q", value)
			}
			return qb.Q().Space("db.instance = ?", instanceID), nil
		case "environment":
			environment, ok := value.(string)
			if !ok {
				return nil, errors.Errorf("failed to parse value %v to string", value)
			}
			if environment != "" {
				environmentID, err := common.GetEnvironmentID(environment)
				if err != nil {
					return nil, errors.Errorf("invalid environment filter %q", value)
				}
				return qb.Q().Space("COALESCE(db.environment, instance.environment) = ?", environmentID), nil
			}
			return qb.Q().Space("db.environment IS NULL AND instance.environment IS NULL"), nil
		case "engine":
			engineValue, ok := storepb.Engine_value[value.(string)]
			if !ok {
				return nil, errors.Errorf("invalid engine filter %q", value)
			}
			engine := storepb.Engine(engineValue)
			return qb.Q().Space("instance.metadata->>'engine' = ?", engine.String()), nil
		case "name":
			return qb.Q().Space("db.name = ?", value), nil
		case "exclude_unassigned":
			if excludeUnassigned, ok := value.(bool); excludeUnassigned && ok {
				return qb.Q().Space("db.project != ? AND db.project != 'default'", common.DefaultProjectID(workspace)), nil
			}
			return qb.Q().Space("TRUE"), nil
		case "table":
			return qb.Q().Space(`
				EXISTS (
					SELECT 1
					FROM json_array_elements(ds.metadata->'schemas') AS s,
						 json_array_elements(s->'tables') AS t
					WHERE t->>'name' = ?
				)
			`, value.(string)), nil
		default:
			varStr, ok := variable.(string)
			if !ok {
				return nil, errors.Errorf("unsupport variable %q", variable)
			}
			if labelKey, ok := strings.CutPrefix(varStr, "labels."); ok {
				return parseToLabelFilterSQL("db.metadata", labelKey, value)
			}
			return nil, errors.Errorf("unsupport variable %q", variable)
		}
	}

	getFilter = func(expr celast.Expr) (*qb.Query, error) {
		q := qb.Q()
		switch expr.Kind() {
		case celast.CallKind:
			functionName := expr.AsCall().FunctionName()
			switch functionName {
			case celoperators.LogicalOr:
				for _, arg := range expr.AsCall().Args() {
					qq, err := getFilter(arg)
					if err != nil {
						return nil, err
					}
					q.Or("?", qq)
				}
				return qb.Q().Space("(?)", q), nil
			case celoperators.LogicalAnd:
				for _, arg := range expr.AsCall().Args() {
					qq, err := getFilter(arg)
					if err != nil {
						return nil, err
					}
					q.And("?", qq)
				}
				return qb.Q().Space("(?)", q), nil
			case celoperators.Equals:
				variable, value := getVariableAndValueFromExpr(expr)
				return parseToSQL(variable, value)
			case celoverloads.Contains:
				variable := expr.AsCall().Target().AsIdent()
				args := expr.AsCall().Args()
				if len(args) != 1 {
					return nil, errors.Errorf(`invalid args for %q`, variable)
				}
				value := args[0].AsLiteral().Value()
				strValue, ok := value.(string)
				if !ok {
					return nil, errors.Errorf("expect string, got %T, hint: filter literals should be string", value)
				}
				strValue = strings.ToLower(strValue)

				switch variable {
				case "name":
					return qb.Q().Space("LOWER(db.name) LIKE ?", "%"+strValue+"%"), nil
				case "table":
					return qb.Q().Space(`EXISTS (
						SELECT 1
						FROM json_array_elements(ds.metadata->'schemas') AS s,
						 	 json_array_elements(s->'tables') AS t
						WHERE t->>'name' LIKE ?)`, "%"+strValue+"%"), nil
				default:
					return nil, errors.Errorf(`only "name" or "table" support %q operator, but found %q`, celoverloads.Contains, variable)
				}
			case celoperators.In:
				variable, value := getVariableAndValueFromExpr(expr)
				if variable == "engine" {
					return parseToEngineSQL(expr)
				} else if labelKey, ok := strings.CutPrefix(variable, "labels."); ok {
					return parseToLabelFilterSQL("db.metadata", labelKey, value)
				}
				return nil, errors.Errorf("unsupport variable %q", variable)
			case celoperators.LogicalNot:
				args := expr.AsCall().Args()
				if len(args) != 1 {
					return nil, errors.Errorf(`only support !(engine in ["{engine1}", "{engine2}"]) format`)
				}
				qq, err := getFilter(args[0])
				if err != nil {
					return nil, err
				}
				return qb.Q().Space("(NOT (?))", qq), nil
			default:
				return nil, errors.Errorf("unexpected function %v", functionName)
			}
		default:
			return nil, errors.Errorf("unexpected expr kind %v", expr.Kind())
		}
	}

	filterQ, err := getFilter(ast.NativeRep().Expr())
	if err != nil {
		return nil, err
	}
	return qb.Q().Space("(?)", filterQ), nil
}

func GetDatabaseOrders(orderBy string) ([]*OrderByKey, error) {
	keys, err := parseOrderBy(orderBy)
	if err != nil {
		return nil, err
	}

	orderByKeys := []*OrderByKey{}
	for _, orderByKey := range keys {
		switch orderByKey.Key {
		case "name":
			orderByKeys = append(orderByKeys, &OrderByKey{
				Key:       "db.name",
				SortOrder: orderByKey.SortOrder,
			})
		case "instance":
			orderByKeys = append(orderByKeys, &OrderByKey{
				Key:       "db.instance",
				SortOrder: orderByKey.SortOrder,
			})
		case "project":
			orderByKeys = append(orderByKeys, &OrderByKey{
				Key:       "db.project",
				SortOrder: orderByKey.SortOrder,
			})
		default:
			return nil, errors.Errorf(`invalid order key "%v"`, orderByKey.Key)
		}
	}
	return orderByKeys, nil
}
