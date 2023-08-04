package snowflake

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/pkg/errors"
	"github.com/snowflakedb/gosnowflake"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

var (
	systemSchemas = map[string]bool{
		"information_schema": true,
	}
)

// SyncInstance syncs the instance.
func (driver *Driver) SyncInstance(ctx context.Context) (*db.InstanceMetadata, error) {
	version, err := driver.getVersion(ctx)
	if err != nil {
		return nil, err
	}

	instanceRoles, err := driver.getInstanceRoles(ctx)
	if err != nil {
		return nil, err
	}

	// Query db info
	databases, err := driver.getDatabases(ctx)
	if err != nil {
		return nil, err
	}

	var filteredDatabases []*storepb.DatabaseSchemaMetadata
	for _, database := range databases {
		if database == bytebaseDatabase {
			continue
		}
		filteredDatabases = append(filteredDatabases, &storepb.DatabaseSchemaMetadata{Name: database})
	}

	return &db.InstanceMetadata{
		Version:       version,
		InstanceRoles: instanceRoles,
		Databases:     filteredDatabases,
	}, nil
}

// SyncDBSchema syncs a single database schema.
func (driver *Driver) SyncDBSchema(ctx context.Context) (*storepb.DatabaseSchemaMetadata, error) {
	// Query db info
	databases, err := driver.getDatabases(ctx)
	if err != nil {
		return nil, err
	}

	databaseMetadata := &storepb.DatabaseSchemaMetadata{
		Name: driver.databaseName,
	}
	found := false
	for _, database := range databases {
		if database == driver.databaseName {
			found = true
			break
		}
	}
	if !found {
		return nil, common.Errorf(common.NotFound, "database %q not found", driver.databaseName)
	}

	schemaList, err := driver.getSchemaList(ctx, driver.databaseName)
	if err != nil {
		return nil, err
	}
	tableMap, viewMap, err := driver.getTableSchema(ctx, driver.databaseName)
	if err != nil {
		return nil, err
	}
	streamMap, err := driver.getStreamSchema(ctx, driver.databaseName)
	if err != nil {
		return nil, err
	}
	taskMap, err := driver.getTaskSchema(ctx, driver.databaseName)
	if err != nil {
		return nil, err
	}

	schemaNameMap := make(map[string]bool)
	for _, schemaName := range schemaList {
		schemaNameMap[schemaName] = true
	}
	for schemaName := range tableMap {
		schemaNameMap[schemaName] = true
	}
	for schemaName := range viewMap {
		schemaNameMap[schemaName] = true
	}
	for schemaName := range streamMap {
		schemaNameMap[schemaName] = true
	}
	var schemaNames []string
	for schemaName := range schemaNameMap {
		schemaNames = append(schemaNames, schemaName)
	}
	sort.Strings(schemaNames)
	for _, schemaName := range schemaNames {
		var tables []*storepb.TableMetadata
		var views []*storepb.ViewMetadata
		var streams []*storepb.StreamMetadata
		var tasks []*storepb.TaskMetadata
		var exists bool
		if tables, exists = tableMap[schemaName]; !exists {
			tables = []*storepb.TableMetadata{}
		}
		if views, exists = viewMap[schemaName]; !exists {
			views = []*storepb.ViewMetadata{}
		}
		if streams, exists = streamMap[schemaName]; !exists {
			streams = []*storepb.StreamMetadata{}
		}
		if tasks, exists = taskMap[schemaName]; !exists {
			tasks = []*storepb.TaskMetadata{}
		}
		databaseMetadata.Schemas = append(databaseMetadata.Schemas, &storepb.SchemaMetadata{
			Name:    schemaName,
			Tables:  tables,
			Views:   views,
			Streams: streams,
			Tasks:   tasks,
		})
	}
	return databaseMetadata, nil
}

func (driver *Driver) getSchemaList(ctx context.Context, database string) ([]string, error) {
	// Query table info
	var excludedSchemaList []string
	// Skip all system schemas.
	for k := range systemSchemas {
		excludedSchemaList = append(excludedSchemaList, fmt.Sprintf("'%s'", k))
	}
	excludeWhere := fmt.Sprintf("LOWER(SCHEMA_NAME) NOT IN (%s)", strings.Join(excludedSchemaList, ", "))

	query := fmt.Sprintf(`
		SELECT
			SCHEMA_NAME
		FROM "%s".INFORMATION_SCHEMA.SCHEMATA
		WHERE %s`, database, excludeWhere)

	rows, err := driver.db.QueryContext(ctx, query)
	if err != nil {
		return nil, util.FormatErrorWithQuery(err, query)
	}
	defer rows.Close()

	var result []string
	for rows.Next() {
		var schemaName string
		if err := rows.Scan(&schemaName); err != nil {
			return nil, err
		}
		result = append(result, schemaName)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return result, nil
}

// getStreamSchema returns the stream map of the given database.
//
// Key: normalized schema name
//
// Value: stream list in the schema.
func (driver *Driver) getStreamSchema(ctx context.Context, database string) (map[string][]*storepb.StreamMetadata, error) {
	streamMap := make(map[string][]*storepb.StreamMetadata)

	streamQuery := fmt.Sprintf(`SHOW STREAMS IN DATABASE "%s";`, database)
	streamMetaRows, err := driver.db.QueryContext(ctx, streamQuery)
	if err != nil {
		return nil, util.FormatErrorWithQuery(err, streamQuery)
	}
	defer streamMetaRows.Close()
	for streamMetaRows.Next() {
		var schemaName, streamName, tableName, owner, comment, tp, mode string
		var stale bool
		var unused any
		if err := streamMetaRows.Scan(&unused, &streamName, &unused, &schemaName, &owner, &comment, &tableName, &unused, &unused, &tp, &stale, &mode, &unused, &unused); err != nil {
			return nil, err
		}
		storePbStreamType := storepb.StreamMetadata_TYPE_UNSPECIFIED
		if tp == "DELTA" {
			storePbStreamType = storepb.StreamMetadata_TYPE_DELTA
		}
		storePbMode := storepb.StreamMetadata_MODE_UNSPECIFIED
		switch mode {
		case "DEFAULT":
			storePbMode = storepb.StreamMetadata_MODE_DEFAULT
		case "APPEND_ONLY":
			storePbMode = storepb.StreamMetadata_MODE_APPEND_ONLY
		case "INSERT_ONLY":
			storePbMode = storepb.StreamMetadata_MODE_INSERT_ONLY
		}
		streamMetadata := &storepb.StreamMetadata{
			Name:      streamName,
			TableName: tableName,
			Owner:     owner,
			Comment:   comment,
			Type:      storePbStreamType,
			Stale:     stale,
			Mode:      storePbMode,
		}
		streamMap[schemaName] = append(streamMap[schemaName], streamMetadata)
	}
	if err := streamMetaRows.Err(); err != nil {
		return nil, err
	}

	for schemaName, streamList := range streamMap {
		for _, stream := range streamList {
			definitionQuery := fmt.Sprintf("SELECT GET_DDL('STREAM', '%s', TRUE);", fmt.Sprintf(`"%s"."%s"."%s"`, database, schemaName, stream.Name))
			var definition string
			if err := driver.db.QueryRow(definitionQuery).Scan(&definition); err != nil {
				return nil, err
			}
			stream.Definition = definition
		}
	}

	for _, streamList := range streamMap {
		sort.Slice(streamList, func(i, j int) bool {
			return streamList[i].Name < streamList[j].Name
		})
	}
	return streamMap, nil
}

// ArrayString is a custom type for scanning array of string.
type ArrayString []string

// Scan implements the sql.Scanner interface.
func (a *ArrayString) Scan(src any) error {
	switch v := src.(type) {
	case string:
		return json.Unmarshal([]byte(v), a)
	case []byte:
		return json.Unmarshal(v, a)
	default:
		return errors.New("invalid type")
	}
}

// getTaskSchema returns the task map of the given database.
//
// Key: normalized schema name
//
// Value: stream list in the schema.
func (driver *Driver) getTaskSchema(ctx context.Context, database string) (map[string][]*storepb.TaskMetadata, error) {
	taskMap := make(map[string][]*storepb.TaskMetadata)

	taskQuery := fmt.Sprintf(`SHOW TASKS IN DATABASE "%s";`, database)
	streamMetaRows, err := driver.db.QueryContext(ctx, taskQuery)
	if err != nil {
		return nil, util.FormatErrorWithQuery(err, taskQuery)
	}
	defer streamMetaRows.Close()
	for streamMetaRows.Next() {
		var schemaName, taskName, id, owner, comment, warehouse, state string
		var nullSchedule, nullCondition sql.NullString
		var predecessors ArrayString
		var unused any
		if err := streamMetaRows.Scan(&unused, &taskName, &id, &unused, &schemaName, &owner, &comment, &warehouse, &nullSchedule, &gosnowflake.DataTypeArray, &state, &unused, &nullCondition, &unused, &unused, &unused, &unused); err != nil {
			return nil, err
		}
		storePbState := storepb.TaskMetadata_STATE_UNSPECIFIED
		switch state {
		case "started":
			storePbState = storepb.TaskMetadata_STATE_STARTED
		case "suspended":
			storePbState = storepb.TaskMetadata_STATE_SUSPENDED
		}
		var schedule, condition string
		if nullSchedule.Valid {
			schedule = nullSchedule.String
		}
		if nullCondition.Valid {
			condition = nullCondition.String
		}
		taskMetadata := &storepb.TaskMetadata{
			Name:         taskName,
			Id:           id,
			Owner:        owner,
			Comment:      comment,
			Warehouse:    warehouse,
			Schedule:     schedule,
			Predecessors: predecessors,
			State:        storePbState,
			Condition:    condition,
		}
		taskMap[schemaName] = append(taskMap[schemaName], taskMetadata)
	}
	if err := streamMetaRows.Err(); err != nil {
		return nil, err
	}

	for schemaName, taskList := range taskMap {
		for _, task := range taskList {
			definitionQuery := fmt.Sprintf("SELECT GET_DDL('TASK', '%s', TRUE);", fmt.Sprintf(`"%s"."%s"."%s"`, database, schemaName, task.Name))
			var definition string
			if err := driver.db.QueryRow(definitionQuery).Scan(&definition); err != nil {
				return nil, err
			}
			task.Definition = definition
		}
	}

	for _, taskList := range taskMap {
		sort.Slice(taskList, func(i, j int) bool {
			return taskList[i].Name < taskList[j].Name
		})
	}
	return taskMap, nil
}

func (driver *Driver) getTableSchema(ctx context.Context, database string) (map[string][]*storepb.TableMetadata, map[string][]*storepb.ViewMetadata, error) {
	tableMap, viewMap := make(map[string][]*storepb.TableMetadata), make(map[string][]*storepb.ViewMetadata)

	// Query table info
	var excludedSchemaList []string
	// Skip all system schemas.
	for k := range systemSchemas {
		excludedSchemaList = append(excludedSchemaList, fmt.Sprintf("'%s'", k))
	}
	excludeWhere := fmt.Sprintf("LOWER(TABLE_SCHEMA) NOT IN (%s)", strings.Join(excludedSchemaList, ", "))

	// Query column info.
	columnMap := make(map[db.TableKey][]*storepb.ColumnMetadata)
	columnQuery := fmt.Sprintf(`
		SELECT
			TABLE_SCHEMA,
			TABLE_NAME,
			IFNULL(COLUMN_NAME, ''),
			ORDINAL_POSITION,
			COLUMN_DEFAULT,
			IS_NULLABLE,
			DATA_TYPE,
			IFNULL(CHARACTER_SET_NAME, ''),
			IFNULL(COLLATION_NAME, ''),
			IFNULL(COMMENT, '')
		FROM "%s".INFORMATION_SCHEMA.COLUMNS
		WHERE %s
		ORDER BY TABLE_SCHEMA, TABLE_NAME, ORDINAL_POSITION`, database, excludeWhere)
	columnRows, err := driver.db.QueryContext(ctx, columnQuery)
	if err != nil {
		return nil, nil, util.FormatErrorWithQuery(err, columnQuery)
	}
	defer columnRows.Close()
	for columnRows.Next() {
		var schemaName, tableName, nullable string
		var defaultStr sql.NullString
		column := &storepb.ColumnMetadata{}
		if err := columnRows.Scan(
			&schemaName,
			&tableName,
			&column.Name,
			&column.Position,
			&defaultStr,
			&nullable,
			&column.Type,
			&column.CharacterSet,
			&column.Collation,
			&column.Comment,
		); err != nil {
			return nil, nil, err
		}
		if defaultStr.Valid {
			column.Default = &wrapperspb.StringValue{Value: defaultStr.String}
		}
		isNullBool, err := util.ConvertYesNo(nullable)
		if err != nil {
			return nil, nil, err
		}
		column.Nullable = isNullBool

		key := db.TableKey{Schema: schemaName, Table: tableName}
		columnMap[key] = append(columnMap[key], column)
	}
	if err := columnRows.Err(); err != nil {
		return nil, nil, util.FormatErrorWithQuery(err, columnQuery)
	}

	tableQuery := fmt.Sprintf(`
		SELECT
			TABLE_SCHEMA,
			TABLE_NAME,
			ROW_COUNT,
			BYTES,
			IFNULL(COMMENT, '')
		FROM "%s".INFORMATION_SCHEMA.TABLES
		WHERE TABLE_TYPE = 'BASE TABLE' AND %s
		ORDER BY TABLE_SCHEMA, TABLE_NAME`, database, excludeWhere)
	tableRows, err := driver.db.QueryContext(ctx, tableQuery)
	if err != nil {
		return nil, nil, util.FormatErrorWithQuery(err, tableQuery)
	}
	defer tableRows.Close()
	for tableRows.Next() {
		var schemaName string
		table := &storepb.TableMetadata{}
		if err := tableRows.Scan(
			&schemaName,
			&table.Name,
			&table.RowCount,
			&table.DataSize,
			&table.Comment,
		); err != nil {
			return nil, nil, err
		}
		if columns, ok := columnMap[db.TableKey{Schema: schemaName, Table: table.Name}]; ok {
			table.Columns = columns
		}

		tableMap[schemaName] = append(tableMap[schemaName], table)
	}
	if err := tableRows.Err(); err != nil {
		return nil, nil, util.FormatErrorWithQuery(err, tableQuery)
	}

	viewQuery := fmt.Sprintf(`
		SELECT
			TABLE_SCHEMA,
			TABLE_NAME,
			IFNULL(VIEW_DEFINITION, ''),
			IFNULL(COMMENT, '')
		FROM "%s".INFORMATION_SCHEMA.VIEWS
		WHERE %s
		ORDER BY TABLE_SCHEMA, TABLE_NAME`, database, excludeWhere)
	viewRows, err := driver.db.QueryContext(ctx, viewQuery)
	if err != nil {
		return nil, nil, util.FormatErrorWithQuery(err, viewQuery)
	}
	defer viewRows.Close()
	for viewRows.Next() {
		view := &storepb.ViewMetadata{}
		var schemaName string
		if err := viewRows.Scan(
			&schemaName,
			&view.Name,
			&view.Definition,
			&view.Comment,
		); err != nil {
			return nil, nil, err
		}
		if columns, ok := columnMap[db.TableKey{Schema: schemaName, Table: view.Name}]; ok {
			for _, column := range columns {
				// TODO(zp): We get column by query the INFORMATION_SCHEMA.COLUMNS, which does not contains the view column belongs to which database.
				// So in the Snowflake, one view column may belongs to different databases, it may cause some confusing behavior in the Data Masking.
				view.DependentColumns = append(view.DependentColumns, &storepb.DependentColumn{
					Schema: schemaName,
					Table:  view.Name,
					Column: column.Name,
				})
			}
		}

		viewMap[schemaName] = append(viewMap[schemaName], view)
	}
	if err := viewRows.Err(); err != nil {
		return nil, nil, util.FormatErrorWithQuery(err, viewQuery)
	}

	return tableMap, viewMap, nil
}

// SyncSlowQuery syncs the slow query.
func (*Driver) SyncSlowQuery(_ context.Context, _ time.Time) (map[string]*storepb.SlowQueryStatistics, error) {
	return nil, errors.Errorf("not implemented")
}

// CheckSlowQueryLogEnabled checks if slow query log is enabled.
func (*Driver) CheckSlowQueryLogEnabled(_ context.Context) error {
	return errors.Errorf("not implemented")
}
