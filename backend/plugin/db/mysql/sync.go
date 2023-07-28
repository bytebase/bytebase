package mysql

import (
	"context"
	"database/sql"
	"fmt"
	"math/rand"
	"sort"
	"strconv"
	"strings"
	"time"

	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"
	"google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
	parser "github.com/bytebase/bytebase/backend/plugin/parser/sql"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

var (
	systemDatabases = map[string]bool{
		"information_schema": true,
		"mysql":              true,
		"performance_schema": true,
		"sys":                true,
		// TiDB only
		"metrics_schema": true,
		// OceanBase only
		"oceanbase":  true,
		"SYS":        true,
		"LBACSYS":    true,
		"ORAAUDITOR": true,
		"__public":   true,
	}
)

// SyncInstance syncs the instance.
func (driver *Driver) SyncInstance(ctx context.Context) (*db.InstanceMetadata, error) {
	version, err := driver.getVersion(ctx)
	if err != nil {
		return nil, err
	}

	lowerCaseTableNamesText, err := driver.getServerVariable(ctx, "lower_case_table_names")
	if err != nil {
		return nil, err
	}
	lowerCaseTableNames, err := strconv.Atoi(lowerCaseTableNamesText)
	if err != nil {
		return nil, errors.Wrapf(err, "invalid lower_case_table_names value: %s", lowerCaseTableNamesText)
	}

	users, err := driver.getInstanceRoles(ctx)
	if err != nil {
		return nil, err
	}

	excludedDatabases := []string{
		// Skip our internal "bytebase" database
		"'bytebase'",
	}
	// Skip all system databases
	for k := range systemDatabases {
		excludedDatabases = append(excludedDatabases, fmt.Sprintf("'%s'", k))
	}

	// Query db info
	where := fmt.Sprintf("LOWER(SCHEMA_NAME) NOT IN (%s)", strings.Join(excludedDatabases, ", "))
	query := `
		SELECT
			SCHEMA_NAME,
			DEFAULT_CHARACTER_SET_NAME,
			DEFAULT_COLLATION_NAME
		FROM information_schema.SCHEMATA
		WHERE ` + where
	rows, err := driver.db.QueryContext(ctx, query)
	if err != nil {
		return nil, util.FormatErrorWithQuery(err, query)
	}
	defer rows.Close()

	var databases []*storepb.DatabaseMetadata
	for rows.Next() {
		database := &storepb.DatabaseMetadata{}
		if err := rows.Scan(
			&database.Name,
			&database.CharacterSet,
			&database.Collation,
		); err != nil {
			return nil, err
		}
		databases = append(databases, database)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return &db.InstanceMetadata{
		Version:       version,
		InstanceRoles: users,
		Databases:     databases,
		Metadata: &storepb.InstanceMetadata{
			MysqlLowerCaseTableNames: int32(lowerCaseTableNames),
		},
	}, nil
}

// SyncDBSchema syncs a single database schema.
func (driver *Driver) SyncDBSchema(ctx context.Context) (*storepb.DatabaseMetadata, error) {
	schemaMetadata := &storepb.SchemaMetadata{
		Name: "",
	}

	// Query MySQL version
	version, err := driver.getVersion(ctx)
	if err != nil {
		return nil, err
	}
	isMySQL8 := strings.HasPrefix(version, "8.0")

	// Query index info.
	indexMap := make(map[db.TableKey]map[string]*storepb.IndexMetadata)
	indexQuery := `
		SELECT
			TABLE_NAME,
			INDEX_NAME,
			COLUMN_NAME,
			'',
			SEQ_IN_INDEX,
			INDEX_TYPE,
			CASE NON_UNIQUE WHEN 0 THEN 1 ELSE 0 END AS IS_UNIQUE,
			1,
			INDEX_COMMENT
		FROM information_schema.STATISTICS
		WHERE TABLE_SCHEMA = ?
		ORDER BY TABLE_NAME, INDEX_NAME, SEQ_IN_INDEX`
	if isMySQL8 {
		indexQuery = `
			SELECT
				TABLE_NAME,
				INDEX_NAME,
				COLUMN_NAME,
				EXPRESSION,
				SEQ_IN_INDEX,
				INDEX_TYPE,
				CASE NON_UNIQUE WHEN 0 THEN 1 ELSE 0 END AS IS_UNIQUE,
				CASE IS_VISIBLE WHEN 'YES' THEN 1 ELSE 0 END,
				INDEX_COMMENT
			FROM information_schema.STATISTICS
			WHERE TABLE_SCHEMA = ?
			ORDER BY TABLE_NAME, INDEX_NAME, SEQ_IN_INDEX`
	}
	indexRows, err := driver.db.QueryContext(ctx, indexQuery, driver.databaseName)
	if err != nil {
		return nil, util.FormatErrorWithQuery(err, indexQuery)
	}
	defer indexRows.Close()
	for indexRows.Next() {
		var tableName, indexName, indexType, comment, expression string
		var columnName sql.NullString
		var expressionName sql.NullString
		var position int
		var unique, visible bool
		if err := indexRows.Scan(
			&tableName,
			&indexName,
			&columnName,
			&expressionName,
			&position,
			&indexType,
			&unique,
			&visible,
			&comment,
		); err != nil {
			return nil, err
		}
		if columnName.Valid {
			expression = columnName.String
		} else if expressionName.Valid {
			expression = expressionName.String
		}

		key := db.TableKey{Schema: "", Table: tableName}
		if _, ok := indexMap[key]; !ok {
			indexMap[key] = make(map[string]*storepb.IndexMetadata)
		}
		if _, ok := indexMap[key][indexName]; !ok {
			indexMap[key][indexName] = &storepb.IndexMetadata{
				Name:    indexName,
				Type:    indexType,
				Unique:  unique,
				Primary: indexName == "PRIMARY",
				Visible: visible,
				Comment: comment,
			}
		}
		indexMap[key][indexName].Expressions = append(indexMap[key][indexName].Expressions, expression)
	}
	if err := indexRows.Err(); err != nil {
		return nil, util.FormatErrorWithQuery(err, indexQuery)
	}

	// Query column info.
	columnMap := make(map[db.TableKey][]*storepb.ColumnMetadata)
	columnQuery := `
		SELECT
			TABLE_NAME,
			IFNULL(COLUMN_NAME, ''),
			ORDINAL_POSITION,
			COLUMN_DEFAULT,
			IS_NULLABLE,
			COLUMN_TYPE,
			IFNULL(CHARACTER_SET_NAME, ''),
			IFNULL(COLLATION_NAME, ''),
			COLUMN_COMMENT
		FROM information_schema.COLUMNS
		WHERE TABLE_SCHEMA = ?
		ORDER BY TABLE_NAME, ORDINAL_POSITION`
	columnRows, err := driver.db.QueryContext(ctx, columnQuery, driver.databaseName)
	if err != nil {
		return nil, util.FormatErrorWithQuery(err, columnQuery)
	}
	defer columnRows.Close()
	for columnRows.Next() {
		column := &storepb.ColumnMetadata{}
		var tableName, nullable string
		var defaultStr sql.NullString
		if err := columnRows.Scan(
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
			return nil, err
		}
		if defaultStr.Valid {
			column.Default = &wrapperspb.StringValue{Value: defaultStr.String}
		}
		isNullBool, err := util.ConvertYesNo(nullable)
		if err != nil {
			return nil, err
		}
		column.Nullable = isNullBool

		key := db.TableKey{Schema: "", Table: tableName}
		columnMap[key] = append(columnMap[key], column)
	}
	if err := columnRows.Err(); err != nil {
		return nil, util.FormatErrorWithQuery(err, columnQuery)
	}

	// Query view info.
	viewMap := make(map[db.TableKey]*storepb.ViewMetadata)
	viewQuery := `
		SELECT
			TABLE_NAME,
			VIEW_DEFINITION
		FROM information_schema.VIEWS
		WHERE TABLE_SCHEMA = ?`
	viewRows, err := driver.db.QueryContext(ctx, viewQuery, driver.databaseName)
	if err != nil {
		return nil, util.FormatErrorWithQuery(err, viewQuery)
	}
	defer viewRows.Close()
	for viewRows.Next() {
		view := &storepb.ViewMetadata{}
		if err := viewRows.Scan(
			&view.Name,
			&view.Definition,
		); err != nil {
			return nil, err
		}
		key := db.TableKey{Schema: "", Table: view.Name}
		viewMap[key] = view
	}
	if err := viewRows.Err(); err != nil {
		return nil, util.FormatErrorWithQuery(err, viewQuery)
	}

	// Query foreign key info.
	foreignKeysMap, err := driver.getForeignKeyList(ctx, driver.databaseName)
	if err != nil {
		return nil, err
	}

	// Query table info.
	tableQuery := `
		SELECT
			TABLE_NAME,
			TABLE_TYPE,
			IFNULL(ENGINE, ''),
			IFNULL(TABLE_COLLATION, ''),
			IFNULL(TABLE_ROWS, 0),
			IFNULL(DATA_LENGTH, 0),
			IFNULL(INDEX_LENGTH, 0),
			IFNULL(DATA_FREE, 0),
			IFNULL(CREATE_OPTIONS, ''),
			IFNULL(TABLE_COMMENT, '')
		FROM information_schema.TABLES
		WHERE TABLE_SCHEMA = ?
		ORDER BY TABLE_NAME`
	tableRows, err := driver.db.QueryContext(ctx, tableQuery, driver.databaseName)
	if err != nil {
		return nil, util.FormatErrorWithQuery(err, tableQuery)
	}
	defer tableRows.Close()
	for tableRows.Next() {
		var tableName, tableType, engine, collation, createOptions, comment string
		var rowCount, dataSize, indexSize, dataFree int64
		// Workaround TiDB bug https://github.com/pingcap/tidb/issues/27970
		var tableCollation sql.NullString
		if err := tableRows.Scan(
			&tableName,
			&tableType,
			&engine,
			&collation,
			&rowCount,
			&dataSize,
			&indexSize,
			&dataFree,
			&createOptions,
			&comment,
		); err != nil {
			return nil, err
		}

		key := db.TableKey{Schema: "", Table: tableName}
		switch tableType {
		case baseTableType:
			tableMetadata := &storepb.TableMetadata{
				Name:          tableName,
				Columns:       columnMap[key],
				ForeignKeys:   foreignKeysMap[key],
				Engine:        engine,
				Collation:     collation,
				RowCount:      rowCount,
				DataSize:      dataSize,
				IndexSize:     indexSize,
				DataFree:      dataFree,
				CreateOptions: createOptions,
				Comment:       comment,
			}
			if tableCollation.Valid {
				tableMetadata.Collation = tableCollation.String
			}
			var indexNames []string
			if indexes, ok := indexMap[key]; ok {
				for indexName := range indexes {
					indexNames = append(indexNames, indexName)
				}
				sort.Strings(indexNames)
				for _, indexName := range indexNames {
					tableMetadata.Indexes = append(tableMetadata.Indexes, indexes[indexName])
				}
			}

			schemaMetadata.Tables = append(schemaMetadata.Tables, tableMetadata)
		case viewTableType:
			if view, ok := viewMap[key]; ok {
				view.Comment = comment
				schemaMetadata.Views = append(schemaMetadata.Views, view)
			}
		}
	}
	if err := tableRows.Err(); err != nil {
		return nil, util.FormatErrorWithQuery(err, tableQuery)
	}

	databaseMetadata := &storepb.DatabaseMetadata{
		Name:    driver.databaseName,
		Schemas: []*storepb.SchemaMetadata{schemaMetadata},
	}
	// Query db info.
	databaseQuery := `
		SELECT
			DEFAULT_CHARACTER_SET_NAME,
			DEFAULT_COLLATION_NAME
		FROM information_schema.SCHEMATA
		WHERE SCHEMA_NAME = ?`
	if err := driver.db.QueryRowContext(ctx, databaseQuery, driver.databaseName).Scan(
		&databaseMetadata.CharacterSet,
		&databaseMetadata.Collation,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, common.Errorf(common.NotFound, "database %q not found", driver.databaseName)
		}
		return nil, err
	}

	return databaseMetadata, err
}

func (driver *Driver) getForeignKeyList(ctx context.Context, databaseName string) (map[db.TableKey][]*storepb.ForeignKeyMetadata, error) {
	fkQuery := `
		SELECT
			fks.TABLE_NAME,
			fks.CONSTRAINT_NAME,
			kcu.COLUMN_NAME,
			'',
			fks.REFERENCED_TABLE_NAME,
			kcu.REFERENCED_COLUMN_NAME,
			fks.DELETE_RULE,
			fks.UPDATE_RULE,
			fks.MATCH_OPTION
		FROM INFORMATION_SCHEMA.REFERENTIAL_CONSTRAINTS fks
			JOIN INFORMATION_SCHEMA.KEY_COLUMN_USAGE kcu
			ON fks.CONSTRAINT_SCHEMA = kcu.TABLE_SCHEMA
				AND fks.TABLE_NAME = kcu.TABLE_NAME
				AND fks.CONSTRAINT_NAME = kcu.CONSTRAINT_NAME
		WHERE LOWER(fks.CONSTRAINT_SCHEMA) = ?
		ORDER BY fks.TABLE_NAME, fks.CONSTRAINT_NAME, kcu.ORDINAL_POSITION;
	`

	fkRows, err := driver.db.QueryContext(ctx, fkQuery, databaseName)
	if err != nil {
		return nil, util.FormatErrorWithQuery(err, fkQuery)
	}
	defer fkRows.Close()
	foreignKeysMap := make(map[db.TableKey][]*storepb.ForeignKeyMetadata)
	var buildingFk *storepb.ForeignKeyMetadata
	var buildingTable string
	for fkRows.Next() {
		var tableName string
		var fk storepb.ForeignKeyMetadata
		var column, referencedColumn string
		if err := fkRows.Scan(
			&tableName,
			&fk.Name,
			&column,
			&fk.ReferencedSchema,
			&fk.ReferencedTable,
			&referencedColumn,
			&fk.OnDelete,
			&fk.OnUpdate,
			&fk.MatchType,
		); err != nil {
			return nil, err
		}

		fk.Columns = append(fk.Columns, column)
		fk.ReferencedColumns = append(fk.ReferencedColumns, referencedColumn)
		if buildingFk == nil {
			buildingTable = tableName
			buildingFk = &fk
		} else {
			if tableName == buildingTable && buildingFk.Name == fk.Name {
				buildingFk.Columns = append(buildingFk.Columns, fk.Columns[0])
				buildingFk.ReferencedColumns = append(buildingFk.ReferencedColumns, fk.ReferencedColumns[0])
			} else {
				key := db.TableKey{Schema: "", Table: buildingTable}
				foreignKeysMap[key] = append(foreignKeysMap[key], buildingFk)
				buildingTable = tableName
				buildingFk = &fk
			}
		}
	}
	if err := fkRows.Err(); err != nil {
		return nil, util.FormatErrorWithQuery(err, fkQuery)
	}

	if buildingFk != nil {
		key := db.TableKey{Schema: "", Table: buildingTable}
		foreignKeysMap[key] = append(foreignKeysMap[key], buildingFk)
	}

	return foreignKeysMap, nil
}

type slowLog struct {
	database string
	details  *storepb.SlowQueryDetails
}

// SyncSlowQuery syncs slow query from mysql.slow_log.
func (driver *Driver) SyncSlowQuery(ctx context.Context, logDateTs time.Time) (map[string]*storepb.SlowQueryStatistics, error) {
	var timeZone string
	// The MySQL function convert_tz requires loading the time zone table into MySQL.
	// So we convert time zone in backend instead of MySQL server
	// https://stackoverflow.com/questions/14454304/convert-tz-returns-null
	timeZoneQuery := `SELECT @@log_timestamps, @@system_time_zone;`
	timeZoneRows, err := driver.db.QueryContext(ctx, timeZoneQuery)
	if err != nil {
		return nil, util.FormatErrorWithQuery(err, timeZoneQuery)
	}
	defer timeZoneRows.Close()
	for timeZoneRows.Next() {
		var logTimeZone, systemTimeZone string
		if err := timeZoneRows.Scan(&logTimeZone, &systemTimeZone); err != nil {
			return nil, err
		}

		timeZone = logTimeZone
		if strings.ToLower(logTimeZone) == "system" {
			timeZone = systemTimeZone
		}
	}
	if err := timeZoneRows.Err(); err != nil {
		return nil, util.FormatErrorWithQuery(err, timeZoneQuery)
	}

	logs := make([]*slowLog, 0, db.SlowQueryMaxSamplePerDay)
	query := `
		SELECT
			start_time,
			query_time,
			lock_time,
			rows_sent,
			rows_examined,
			db,
			CONVERT(sql_text USING utf8) AS sql_text
		FROM
			mysql.slow_log
		WHERE
			start_time >= ?
			AND start_time < ?
	`

	slowLogRows, err := driver.db.QueryContext(ctx, query, logDateTs.Format("2006-01-02"), logDateTs.AddDate(0, 0, 1).Format("2006-01-02"))
	if err != nil {
		return nil, util.FormatErrorWithQuery(err, query)
	}
	defer slowLogRows.Close()
	for slowLogRows.Next() {
		log := slowLog{
			details: &storepb.SlowQueryDetails{},
		}
		var startTime, queryTime, lockTime string
		if err := slowLogRows.Scan(
			&startTime,
			&queryTime,
			&lockTime,
			&log.details.RowsSent,
			&log.details.RowsExamined,
			&log.database,
			&log.details.SqlText,
		); err != nil {
			return nil, err
		}

		location, err := time.LoadLocation(timeZone)
		if err != nil {
			return nil, err
		}
		startTimeTs, err := time.ParseInLocation("2006-01-02 15:04:05.999999", startTime, location)
		if err != nil {
			return nil, err
		}
		log.details.StartTime = timestamppb.New(startTimeTs)

		queryTimeDuration, err := parseDuration(queryTime)
		if err != nil {
			return nil, err
		}
		log.details.QueryTime = durationpb.New(queryTimeDuration)

		lockTimeDuration, err := parseDuration(lockTime)
		if err != nil {
			return nil, err
		}
		log.details.LockTime = durationpb.New(lockTimeDuration)

		// Use Reservoir Sampling to sample slow logs.
		// See https://en.wikipedia.org/wiki/Reservoir_sampling
		if len(logs) < db.SlowQueryMaxSamplePerDay {
			logs = append(logs, &log)
		} else {
			pos := rand.Intn(len(logs))
			logs[pos] = &log
		}
	}

	if err := slowLogRows.Err(); err != nil {
		return nil, util.FormatErrorWithQuery(err, query)
	}

	return analyzeSlowLog(logs)
}

func parseDuration(s string) (time.Duration, error) {
	if s == "" {
		return 0, nil
	}
	list := strings.Split(s, ":")
	if len(list) != 3 {
		return 0, errors.Errorf("invalid duration: %s", s)
	}
	duration := fmt.Sprintf("%sh%sm%ss", list[0], list[1], list[2])
	return time.ParseDuration(duration)
}

func analyzeSlowLog(logs []*slowLog) (map[string]*storepb.SlowQueryStatistics, error) {
	logMap := make(map[string]map[string]*storepb.SlowQueryStatisticsItem)

	for _, log := range logs {
		databaseList := extractDatabase(log.database, log.details.SqlText)
		fingerprint, err := parser.GetSQLFingerprint(parser.MySQL, log.details.SqlText)
		if err != nil {
			return nil, errors.Wrapf(err, "get sql fingerprint failed, sql: %s", log.details.SqlText)
		}
		if len(fingerprint) > db.SlowQueryMaxLen {
			fingerprint = fingerprint[:db.SlowQueryMaxLen]
		}
		if len(log.details.SqlText) > db.SlowQueryMaxLen {
			log.details.SqlText = log.details.SqlText[:db.SlowQueryMaxLen]
		}

		for _, db := range databaseList {
			var dbLog map[string]*storepb.SlowQueryStatisticsItem
			var exists bool
			if dbLog, exists = logMap[db]; !exists {
				dbLog = make(map[string]*storepb.SlowQueryStatisticsItem)
				logMap[db] = dbLog
			}
			dbLog[fingerprint] = mergeSlowLog(fingerprint, dbLog[fingerprint], log.details)
		}
	}

	var result = make(map[string]*storepb.SlowQueryStatistics)

	for db, dblog := range logMap {
		var statisticsList storepb.SlowQueryStatistics
		for _, statistics := range dblog {
			statisticsList.Items = append(statisticsList.Items, statistics)
		}
		result[db] = &statisticsList
	}

	return result, nil
}

func mergeSlowLog(fingerprint string, statistics *storepb.SlowQueryStatisticsItem, details *storepb.SlowQueryDetails) *storepb.SlowQueryStatisticsItem {
	if statistics == nil {
		return &storepb.SlowQueryStatisticsItem{
			SqlFingerprint:      fingerprint,
			Count:               1,
			LatestLogTime:       details.StartTime,
			TotalQueryTime:      details.QueryTime,
			MaximumQueryTime:    details.QueryTime,
			TotalRowsSent:       details.RowsSent,
			MaximumRowsSent:     details.RowsSent,
			TotalRowsExamined:   details.RowsExamined,
			MaximumRowsExamined: details.RowsExamined,
			Samples:             []*storepb.SlowQueryDetails{details},
		}
	}
	statistics.Count++
	if statistics.LatestLogTime.AsTime().Before(details.StartTime.AsTime()) {
		statistics.LatestLogTime = details.StartTime
	}
	statistics.TotalQueryTime = durationpb.New(statistics.TotalQueryTime.AsDuration() + details.QueryTime.AsDuration())
	if statistics.MaximumQueryTime.AsDuration() < details.QueryTime.AsDuration() {
		statistics.MaximumQueryTime = details.QueryTime
	}
	statistics.TotalRowsSent += details.RowsSent
	if statistics.MaximumRowsSent < details.RowsSent {
		statistics.MaximumRowsSent = details.RowsSent
	}
	statistics.TotalRowsExamined += details.RowsExamined
	if statistics.MaximumRowsExamined < details.RowsExamined {
		statistics.MaximumRowsExamined = details.RowsExamined
	}
	if len(statistics.Samples) < db.SlowQueryMaxSamplePerFingerprint {
		statistics.Samples = append(statistics.Samples, details)
	} else {
		// Use Reservoir Sampling to sample slow logs.
		pos := rand.Intn(len(statistics.Samples))
		statistics.Samples[pos] = details
	}
	return statistics
}

func extractDatabase(defaultDB string, sql string) []string {
	list, err := parser.ExtractDatabaseList(parser.MySQL, sql, "")
	if err != nil {
		// If we can't extract the database, we just use the default database.
		log.Debug("extract database failed", zap.Error(err), zap.String("sql", sql))
		return []string{defaultDB}
	}

	var result []string
	for _, db := range list {
		if db == "" {
			result = append(result, defaultDB)
		} else {
			result = append(result, db)
		}
	}
	if len(result) == 0 {
		result = append(result, defaultDB)
	}
	return result
}

// CheckSlowQueryLogEnabled checks whether the slow query log is enabled.
func (driver *Driver) CheckSlowQueryLogEnabled(ctx context.Context) error {
	showSlowQueryLog := "SHOW GLOBAL VARIABLES LIKE 'slow_query_log'"

	slowQueryLogRows, err := driver.db.QueryContext(ctx, showSlowQueryLog)
	if err != nil {
		return util.FormatErrorWithQuery(err, showSlowQueryLog)
	}
	defer slowQueryLogRows.Close()
	for slowQueryLogRows.Next() {
		var name, value string
		if err := slowQueryLogRows.Scan(&name, &value); err != nil {
			return err
		}
		if value != "ON" {
			return errors.New("slow query log is not enabled: slow_query_log = " + value)
		}
	}
	if err := slowQueryLogRows.Err(); err != nil {
		return util.FormatErrorWithQuery(err, showSlowQueryLog)
	}

	showLogOutput := "SHOW GLOBAL VARIABLES LIKE 'log_output'"
	logOutputRows, err := driver.db.QueryContext(ctx, showLogOutput)
	if err != nil {
		return util.FormatErrorWithQuery(err, showLogOutput)
	}
	defer logOutputRows.Close()
	for logOutputRows.Next() {
		var name, value string
		if err := logOutputRows.Scan(&name, &value); err != nil {
			return err
		}
		if !strings.Contains(value, "TABLE") {
			return errors.New("slow query log is not contained in TABLE: log_output = " + value)
		}
	}
	if err := logOutputRows.Err(); err != nil {
		return util.FormatErrorWithQuery(err, showLogOutput)
	}

	return nil
}
