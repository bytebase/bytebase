package store

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/google/cel-go/cel"
	celast "github.com/google/cel-go/common/ast"
	celoperators "github.com/google/cel-go/common/operators"
	celoverloads "github.com/google/cel-go/common/overloads"
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
	Engine   storepb.Engine
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

	FilterQ     *qb.Query
	Limit       *int
	Offset      *int
	OrderByKeys []*OrderByKey
}

// GetDatabase gets a database.
func (s *Store) GetDatabase(ctx context.Context, find *FindDatabaseMessage) (*DatabaseMessage, error) {
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
	rows, err := tx.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		databaseMessage := &DatabaseMessage{}
		var metadataString string
		var effectiveEnvironment, environment, engine sql.NullString
		if err := rows.Scan(
			&databaseMessage.ProjectID,
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
	return s.GetDatabase(ctx, &FindDatabaseMessage{InstanceID: &create.InstanceID, DatabaseName: &create.DatabaseName, ShowDeleted: true})
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
	return s.GetDatabase(ctx, &FindDatabaseMessage{InstanceID: &create.InstanceID, DatabaseName: &create.DatabaseName, ShowDeleted: true})
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
		database, err := s.GetDatabase(ctx, &FindDatabaseMessage{
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

	q := qb.Q().Space("UPDATE db SET ? WHERE instance = ? AND name = ? RETURNING id", set, patch.InstanceID, patch.DatabaseName)

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
	return s.GetDatabase(ctx, &FindDatabaseMessage{InstanceID: &patch.InstanceID, DatabaseName: &patch.DatabaseName, ShowDeleted: true})
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

	if len(databases) > 0 {
		var dbInstances, dbNames []string
		for _, database := range databases {
			dbInstances = append(dbInstances, database.InstanceID)
			dbNames = append(dbNames, database.DatabaseName)
		}
		where.Or(`(db.instance, db.name) IN (SELECT * FROM unnest(?::TEXT[], ?::TEXT[]))`, dbInstances, dbNames)
	}

	if where.Len() == 0 {
		return nil, errors.Errorf("empty where")
	}

	q := qb.Q().Space("UPDATE db SET ? WHERE ?", set, where)

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
				instance, err := s.GetInstance(ctx, &FindInstanceMessage{ResourceID: &database.InstanceID})
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

func GetListDatabaseFilter(filter string) (*qb.Query, error) {
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
		case "drifted":
			drifted, ok := value.(bool)
			if !ok {
				return nil, errors.Errorf("invalid drifted filter %q", value)
			}
			condition := "IS"
			if !drifted {
				condition = "IS NOT"
			}
			return qb.Q().Space(fmt.Sprintf("(db.metadata->>'drifted')::boolean %s TRUE", condition)), nil
		case "exclude_unassigned":
			if excludeUnassigned, ok := value.(bool); excludeUnassigned && ok {
				return qb.Q().Space("db.project != ?", common.DefaultProjectID), nil
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
			case celoverloads.Matches:
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
						WHERE t->>'name' LIKE ?`, "%"+strValue+"%"), nil
				default:
					return nil, errors.Errorf(`only "name" or "table" support %q operator, but found %q`, celoverloads.Matches, variable)
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
