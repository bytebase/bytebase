package mysql

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"math/rand"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/transform"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"
	"google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/blang/semver/v4"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	mysqlparser "github.com/bytebase/bytebase/backend/plugin/parser/mysql"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

const (
	autoIncrementSymbol    = "AUTO_INCREMENT"
	autoRandSymbol         = "AUTO_RANDOM"
	pkAutoRandomBitsSymbol = "PK_AUTO_RANDOM_BITS"
	virtualGenerated       = "VIRTUAL GENERATED"
	storedGenerated        = "STORED GENERATED"
)

var (
	systemDatabases = map[string]bool{
		"information_schema": true,
		"mysql":              true,
		"performance_schema": true,
		"sys":                true,
		// OceanBase only
		"oceanbase":  true,
		"SYS":        true,
		"LBACSYS":    true,
		"ORAAUDITOR": true,
		"__public":   true,
	}
	systemDatabaseClause = func() string {
		var l []string
		for k := range systemDatabases {
			l = append(l, fmt.Sprintf("'%s'", k))
		}
		return strings.Join(l, ", ")
	}()

	viewDefMatcher = regexp.MustCompile("CREATE ALGORITHM=(UNDEFINED|MERGE|TEMPTABLE) DEFINER=`([^`]+)`@`([^`]+)` SQL SECURITY (DEFINER|INVOKER) VIEW `([^`]+)`( \\((`([^`]+)`)+\\))? AS (?P<def>.+)")
)

// SyncInstance syncs the instance.
func (driver *Driver) SyncInstance(ctx context.Context) (*db.InstanceMetadata, error) {
	version, _, err := driver.getVersion(ctx)
	if err != nil {
		return nil, err
	}

	lowerCaseTableNames := 0
	lowerCaseTableNamesText, err := driver.getServerVariable(ctx, "lower_case_table_names")
	if err != nil {
		slog.Debug("failed to get lower_case_table_names variable", log.BBError(err))
	} else {
		lowerCaseTableNames, err = strconv.Atoi(lowerCaseTableNamesText)
		if err != nil {
			slog.Debug("failed to parse lower_case_table_names variable", log.BBError(err))
		}
	}

	instanceRoles, err := driver.getInstanceRoles(ctx)
	if err != nil {
		return nil, err
	}

	// Query db info
	where := fmt.Sprintf("LOWER(SCHEMA_NAME) NOT IN (%s)", systemDatabaseClause)
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

	var databases []*storepb.DatabaseSchemaMetadata
	for rows.Next() {
		database := &storepb.DatabaseSchemaMetadata{}
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
		Version:   version,
		Databases: databases,
		Metadata: &storepb.InstanceMetadata{
			MysqlLowerCaseTableNames: int32(lowerCaseTableNames),
			Roles:                    instanceRoles,
		},
	}, nil
}

func (driver *Driver) getServerVariable(ctx context.Context, varName string) (string, error) {
	db := driver.GetDB()
	query := fmt.Sprintf("SHOW VARIABLES LIKE '%s'", varName)
	var varNameFound, value string
	if err := db.QueryRowContext(ctx, query).Scan(&varNameFound, &value); err != nil {
		if err == sql.ErrNoRows {
			return "", common.FormatDBErrorEmptyRowWithQuery(query)
		}
		return "", util.FormatErrorWithQuery(err, query)
	}
	if varName != varNameFound {
		return "", errors.Errorf("expecting variable %s, but got %s", varName, varNameFound)
	}
	return value, nil
}

func containsInvisibleChars(data []byte) bool {
	// Iterate over the byte slice as runes
	for len(data) > 0 {
		r, size := utf8.DecodeRune(data)
		if r == utf8.RuneError && size == 1 {
			// If the byte slice contains invalid UTF-8 characters, treat it as invisible
			return true
		}
		// Check if the rune is not printable
		if !unicode.IsPrint(r) {
			return true
		}
		// Move to the next rune
		data = data[size:]
	}
	return false
}

func utf8ToISO88591(utf8Str string) (string, error) {
	// Create a transformer that encodes UTF-8 to ISO-8859-1
	encoder := charmap.ISO8859_1.NewEncoder()

	// Transform the input UTF-8 string into ISO-8859-1
	isoBytes, _, err := transform.String(encoder, utf8Str)
	if err != nil {
		return "", err
	}

	return isoBytes, nil
}

// SyncDBSchema syncs a single database schema.
func (driver *Driver) SyncDBSchema(ctx context.Context) (*storepb.DatabaseSchemaMetadata, error) {
	schemaMetadata := &storepb.SchemaMetadata{
		Name: "",
	}

	// Query MySQL version
	version, rest, err := driver.getVersion(ctx)
	if err != nil {
		return nil, err
	}
	semVersion, err := semver.Make(version)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to parse MySQL version %s to semantic version", version)
	}
	atLeast8_0_13 := semVersion.GE(semver.MustParse("8.0.13"))
	atLeast8_0_16 := semVersion.GE(semver.MustParse("8.0.16"))
	atLeast5_7_0 := semVersion.GE(semver.MustParse("5.7.0"))

	// Query index info.
	indexMap := make(map[db.TableKey]map[string]*storepb.IndexMetadata)
	indexQuery := `
		SELECT
			TABLE_NAME,
			INDEX_NAME,
			COLUMN_NAME,
			COLLATION,
			IFNULL(SUB_PART, -1),
			'',
			SEQ_IN_INDEX,
			INDEX_TYPE,
			CASE NON_UNIQUE WHEN 0 THEN 1 ELSE 0 END AS IS_UNIQUE,
			1,
			INDEX_COMMENT
		FROM information_schema.STATISTICS
		WHERE TABLE_SCHEMA = ?
		ORDER BY TABLE_NAME, INDEX_NAME, SEQ_IN_INDEX`
	// MySQL 8.0.13 introduced the EXPRESSION column in the INFORMATION_SCHEMA.STATISTICS table.
	// https://dev.mysql.com/doc/refman/8.0/en/information-schema-statistics-table.html
	// MariaDB doesn't have the EXPRESSION column.
	// https://mariadb.com/docs/server/ref/mdb/information-schema/STATISTICS
	if atLeast8_0_13 && !strings.Contains(rest, "MariaDB") {
		indexQuery = `
			SELECT
				TABLE_NAME,
				INDEX_NAME,
				COLUMN_NAME,
				COLLATION,
				IFNULL(SUB_PART, -1),
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
		var collation sql.NullString
		var position int
		var subPart int64
		var unique, visible bool
		if err := indexRows.Scan(
			&tableName,
			&indexName,
			&columnName,
			&collation,
			&subPart,
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
			// It's a bit late or not necessary to differentiate the column name or expression.
			// We add parentheses around expression.
			expression = fmt.Sprintf("(%s)", expressionName.String)
		}

		desc := false
		if collation.Valid && collation.String == "D" {
			desc = true
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
		indexMap[key][indexName].KeyLength = append(indexMap[key][indexName].KeyLength, subPart)
		indexMap[key][indexName].Descending = append(indexMap[key][indexName].Descending, desc)
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
			CASE WHEN COLUMN_DEFAULT is NULL THEN NULL ELSE QUOTE(COLUMN_DEFAULT) END,
			IS_NULLABLE,
			COLUMN_TYPE,
			IFNULL(CHARACTER_SET_NAME, ''),
			IFNULL(COLLATION_NAME, ''),
			QUOTE(COLUMN_COMMENT),
			convert(GENERATION_EXPRESSION using BINARY),
			EXTRA
		FROM information_schema.COLUMNS
		WHERE TABLE_SCHEMA = ?
		ORDER BY TABLE_NAME, ORDINAL_POSITION`
	if !atLeast5_7_0 {
		// GENERATION_EXPRESSION does not exist in MySQL 5.6.
		columnQuery = `
		SELECT
			TABLE_NAME,
			IFNULL(COLUMN_NAME, ''),
			ORDINAL_POSITION,
			CASE WHEN COLUMN_DEFAULT is NULL THEN NULL ELSE QUOTE(COLUMN_DEFAULT) END,
			IS_NULLABLE,
			COLUMN_TYPE,
			IFNULL(CHARACTER_SET_NAME, ''),
			IFNULL(COLLATION_NAME, ''),
			QUOTE(COLUMN_COMMENT),
			NULL,
			EXTRA
		FROM information_schema.COLUMNS
		WHERE TABLE_SCHEMA = ?
		ORDER BY TABLE_NAME, ORDINAL_POSITION`
	}
	columnRows, err := driver.db.QueryContext(ctx, columnQuery, driver.databaseName)
	if err != nil {
		return nil, util.FormatErrorWithQuery(err, columnQuery)
	}
	defer columnRows.Close()
	for columnRows.Next() {
		column := &storepb.ColumnMetadata{}
		var tableName, nullable, extra, tp string
		var defaultStr sql.NullString
		var generationExpr []byte
		if err := columnRows.Scan(
			&tableName,
			&column.Name,
			&column.Position,
			&defaultStr,
			&nullable,
			&tp,
			&column.CharacterSet,
			&column.Collation,
			&column.Comment,
			&generationExpr,
			&extra,
		); err != nil {
			return nil, err
		}
		// Quoted string has a single quote around it.
		column.Comment = stripSingleQuote(column.Comment)
		if defaultStr.Valid {
			defaultStr.String = stripSingleQuote(defaultStr.String)
		}

		nullableBool, err := util.ConvertYesNo(nullable)
		if err != nil {
			return nil, err
		}
		column.Type = GetColumnTypeCanonicalSynonym(tp)
		column.Nullable = nullableBool
		setColumnMetadataDefault(column, defaultStr, nullableBool, extra)
		key := db.TableKey{Schema: "", Table: tableName}
		columnMap[key] = append(columnMap[key], column)
		invisible := containsInvisibleChars(generationExpr)
		iso88591Text, convertedErr := utf8ToISO88591(string(generationExpr))
		text := string(generationExpr)
		if invisible && convertedErr == nil {
			text = iso88591Text
		}
		if extra != "" && strings.Contains(strings.ToUpper(extra), virtualGenerated) && len(generationExpr) != 0 {
			column.Generation = &storepb.GenerationMetadata{
				Type:       storepb.GenerationMetadata_TYPE_VIRTUAL,
				Expression: text,
			}
		} else if extra != "" && strings.Contains(strings.ToUpper(extra), storedGenerated) && len(generationExpr) != 0 {
			column.Generation = &storepb.GenerationMetadata{
				Type:       storepb.GenerationMetadata_TYPE_STORED,
				Expression: text,
			}
		}
	}
	if err := columnRows.Err(); err != nil {
		return nil, util.FormatErrorWithQuery(err, columnQuery)
	}

	// Check constraints info.
	checkMap := make(map[db.TableKey][]*storepb.CheckConstraintMetadata)
	if atLeast8_0_16 {
		checkQuery := `
		SELECT
			tc.TABLE_NAME,
			cc.CONSTRAINT_NAME,
			cc.CHECK_CLAUSE
		FROM information_schema.CHECK_CONSTRAINTS cc
			JOIN information_schema.TABLE_CONSTRAINTS tc ON cc.CONSTRAINT_NAME = tc.CONSTRAINT_NAME
		WHERE tc.CONSTRAINT_TYPE = 'CHECK' AND tc.TABLE_SCHEMA = ?
	`
		checkRows, err := driver.db.QueryContext(ctx, checkQuery, driver.databaseName)
		if err != nil {
			return nil, util.FormatErrorWithQuery(err, checkQuery)
		}
		defer checkRows.Close()
		for checkRows.Next() {
			check := &storepb.CheckConstraintMetadata{}
			var tableName string
			if err := checkRows.Scan(
				&tableName,
				&check.Name,
				&check.Expression,
			); err != nil {
				return nil, err
			}
			key := db.TableKey{Schema: "", Table: tableName}
			checkMap[key] = append(checkMap[key], check)
		}
		if err := checkRows.Err(); err != nil {
			return nil, util.FormatErrorWithQuery(err, checkQuery)
		}
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
		view.Columns = columnMap[key]
		viewMap[key] = view
	}
	if err := viewRows.Err(); err != nil {
		return nil, util.FormatErrorWithQuery(err, viewQuery)
	}
	for key := range viewMap {
		def, err := driver.reconcileViewDefinition(ctx, driver.databaseName, key.Table)
		if err != nil {
			return nil, err
		}
		if def != "" {
			viewMap[key].Definition = def
		}
	}

	// Query triggers.
	triggerList, err := driver.getTriggerList(ctx, driver.databaseName)
	if err != nil {
		return nil, err
	}
	schemaMetadata.Triggers = triggerList

	// Query events.
	eventList, err := driver.getEventList(ctx, driver.databaseName)
	if err != nil {
		return nil, err
	}
	schemaMetadata.Events = eventList

	// Query foreign key info.
	foreignKeysMap, err := driver.getForeignKeyList(ctx, driver.databaseName)
	if err != nil {
		return nil, err
	}

	partitionTables := make(map[db.TableKey][]*storepb.TablePartitionMetadata)
	// Query partition info.
	if driver.dbType == storepb.Engine_MYSQL {
		partitionTables, err = driver.listPartitionTables(ctx, driver.databaseName)
		if err != nil {
			return nil, err
		}
	}

	functions, procedures, err := driver.syncRoutines(ctx, driver.databaseName)
	if err != nil {
		return nil, err
	}
	schemaMetadata.Functions = functions
	schemaMetadata.Procedures = procedures

	// Query table info.
	tableQuery := `
		SELECT
			TABLES.TABLE_NAME,
			TABLES.TABLE_TYPE,
			IFNULL(TABLES.ENGINE, ''),
			IFNULL(TABLES.TABLE_COLLATION, ''),
			IFNULL(TABLES.TABLE_ROWS, 0),
			IFNULL(TABLES.DATA_LENGTH, 0),
			IFNULL(TABLES.INDEX_LENGTH, 0),
			IFNULL(TABLES.DATA_FREE, 0),
			IFNULL(TABLES.CREATE_OPTIONS, ''),
			QUOTE(IFNULL(TABLES.TABLE_COMMENT, '')),
			IFNULL(CCSA.CHARACTER_SET_NAME, '')
		FROM information_schema.TABLES TABLES
		LEFT JOIN information_schema.COLLATION_CHARACTER_SET_APPLICABILITY CCSA
		ON TABLES.TABLE_COLLATION = CCSA.COLLATION_NAME
		WHERE TABLE_SCHEMA = ?
		ORDER BY TABLE_NAME`
	tableRows, err := driver.db.QueryContext(ctx, tableQuery, driver.databaseName)
	if err != nil {
		return nil, util.FormatErrorWithQuery(err, tableQuery)
	}
	defer tableRows.Close()
	for tableRows.Next() {
		var tableName, tableType, engine, collation, createOptions, comment, charset string
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
			&charset,
		); err != nil {
			return nil, err
		}
		// Quoted string has a single quote around it.
		comment = stripSingleQuote(comment)

		key := db.TableKey{Schema: "", Table: tableName}
		switch tableType {
		case baseTableType:
			columns := columnMap[key]
			tableMetadata := &storepb.TableMetadata{
				Name:             tableName,
				Columns:          columns,
				ForeignKeys:      foreignKeysMap[key],
				Engine:           engine,
				Collation:        collation,
				RowCount:         rowCount,
				DataSize:         dataSize,
				IndexSize:        indexSize,
				DataFree:         dataFree,
				CreateOptions:    createOptions,
				Comment:          comment,
				Partitions:       partitionTables[key],
				CheckConstraints: checkMap[key],
				Charset:          charset,
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
				schemaMetadata.Views = append(schemaMetadata.Views, view)
			}
		}
	}
	if err := tableRows.Err(); err != nil {
		return nil, util.FormatErrorWithQuery(err, tableQuery)
	}

	databaseMetadata := &storepb.DatabaseSchemaMetadata{
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

func (driver *Driver) getEventList(ctx context.Context, databaseName string) ([]*storepb.EventMetadata, error) {
	listEventsQuery := `
	SELECT
		EVENT_NAME,
		TIME_ZONE,
		SQL_MODE,
		CHARACTER_SET_CLIENT,
		COLLATION_CONNECTION
	FROM INFORMATION_SCHEMA.EVENTS
	WHERE EVENT_SCHEMA = ?
	ORDER BY EVENT_NAME ASC;
	`
	eventRows, err := driver.db.QueryContext(ctx, listEventsQuery, databaseName)
	if err != nil {
		return nil, util.FormatErrorWithQuery(err, listEventsQuery)
	}
	defer eventRows.Close()
	var events []*storepb.EventMetadata
	for eventRows.Next() {
		var name, timeZone, sqlMode, charsetClient, collationConnection string
		if err := eventRows.Scan(
			&name,
			&timeZone,
			&sqlMode,
			&charsetClient,
			&collationConnection,
		); err != nil {
			return nil, err
		}
		eventDef, err := driver.getCreateEventStmt(ctx, databaseName, name)
		if err != nil {
			return nil, err
		}
		event := &storepb.EventMetadata{
			Name:                name,
			TimeZone:            timeZone,
			Definition:          eventDef,
			SqlMode:             sqlMode,
			CharacterSetClient:  charsetClient,
			CollationConnection: collationConnection,
		}
		events = append(events, event)
	}
	if err := eventRows.Err(); err != nil {
		return nil, util.FormatErrorWithQuery(err, listEventsQuery)
	}
	return events, nil
}

func (driver *Driver) getCreateEventStmt(ctx context.Context, databaseName string, name string) (string, error) {
	query := fmt.Sprintf("SHOW CREATE EVENT `%s`.`%s`", databaseName, name)
	rows, err := driver.db.QueryContext(ctx, query)
	if err != nil {
		return "", util.FormatErrorWithQuery(err, query)
	}
	defer rows.Close()

	var createEvent sql.NullString

	columns, err := rows.Columns()
	if err != nil {
		return "", err
	}
	defIdx := -1
	for i, column := range columns {
		if strings.EqualFold(column, "Create Event") {
			defIdx = i
			break
		}
	}

	if defIdx == -1 {
		return "", errors.Errorf("failed to find column Create Event")
	}

	for rows.Next() {
		dests := make([]any, len(columns))
		for i := 0; i < len(columns); i++ {
			if i == defIdx {
				dests[i] = &createEvent
				continue
			}
			dests[i] = new(string)
		}

		if err := rows.Scan(dests...); err != nil {
			return "", err
		}
	}

	if err := rows.Err(); err != nil {
		return "", err
	}

	if createEvent.Valid {
		return createEvent.String, nil
	}
	return "", nil
}

func (driver *Driver) getTriggerList(ctx context.Context, databaseName string) ([]*storepb.TriggerMetadata, error) {
	triggersQuery := `
	SELECT 
		TRIGGER_NAME,
		EVENT_OBJECT_TABLE,
		EVENT_MANIPULATION,
		ACTION_TIMING,
		ACTION_STATEMENT,
		SQL_MODE,
		CHARACTER_SET_CLIENT,
		COLLATION_CONNECTION
	FROM INFORMATION_SCHEMA.TRIGGERS
	WHERE TRIGGER_SCHEMA = ?
	ORDER BY EVENT_OBJECT_TABLE ASC, EVENT_MANIPULATION ASC, ACTION_TIMING ASC, ACTION_ORDER ASC;
	`
	triggerRows, err := driver.db.QueryContext(ctx, triggersQuery, databaseName)
	if err != nil {
		return nil, util.FormatErrorWithQuery(err, triggersQuery)
	}
	defer triggerRows.Close()
	var triggers []*storepb.TriggerMetadata
	for triggerRows.Next() {
		var name, table, event, timing, statement, sqlMode, charsetClient, collationConnection string
		if err := triggerRows.Scan(
			&name,
			&table,
			&event,
			&timing,
			&statement,
			&sqlMode,
			&charsetClient,
			&collationConnection,
		); err != nil {
			return nil, err
		}
		trigger := &storepb.TriggerMetadata{
			Name:                name,
			TableName:           table,
			Event:               event,
			Timing:              timing,
			Body:                statement,
			SqlMode:             sqlMode,
			CharacterSetClient:  charsetClient,
			CollationConnection: collationConnection,
		}
		triggers = append(triggers, trigger)
	}
	if err := triggerRows.Err(); err != nil {
		return nil, util.FormatErrorWithQuery(err, triggersQuery)
	}
	return triggers, nil
}

func (driver *Driver) syncRoutines(ctx context.Context, databaseName string) ([]*storepb.FunctionMetadata, []*storepb.ProcedureMetadata, error) {
	// Query functions and procedure info.
	routinesQuery := `
		SELECT
			ROUTINE_NAME,
			ROUTINE_TYPE,
			SQL_MODE,
			CHARACTER_SET_CLIENT,
			COLLATION_CONNECTION,
			DATABASE_COLLATION
		FROM
			INFORMATION_SCHEMA.ROUTINES
		WHERE ROUTINE_SCHEMA = ? AND ROUTINE_TYPE IN ('FUNCTION', 'PROCEDURE')
		ORDER BY ROUTINE_TYPE, ROUTINE_NAME;
	`
	routineRows, err := driver.db.QueryContext(ctx, routinesQuery, driver.databaseName)
	if err != nil {
		return nil, nil, util.FormatErrorWithQuery(err, routinesQuery)
	}
	defer routineRows.Close()
	var functions []*storepb.FunctionMetadata
	var procedures []*storepb.ProcedureMetadata
	for routineRows.Next() {
		var name, routineType string
		var sqlMode, charsetClient, collationConnection, databaseCollation sql.NullString
		if err := routineRows.Scan(
			&name,
			&routineType,
			&sqlMode,
			&charsetClient,
			&collationConnection,
			&databaseCollation,
		); err != nil {
			return nil, nil, err
		}
		if strings.EqualFold(routineType, "PROCEDURE") {
			procedureDef, err := driver.getCreateProcedureStmt(ctx, databaseName, name)
			if err != nil {
				return nil, nil, err
			}
			procedures = append(procedures, &storepb.ProcedureMetadata{
				Name:                name,
				Definition:          procedureDef,
				SqlMode:             sqlMode.String,
				CharacterSetClient:  charsetClient.String,
				CollationConnection: collationConnection.String,
				DatabaseCollation:   databaseCollation.String,
			})
		} else {
			functionDef, err := driver.getCreateFunctionStmt(ctx, databaseName, name)
			if err != nil {
				return nil, nil, err
			}
			functions = append(functions, &storepb.FunctionMetadata{
				Name:                name,
				Definition:          functionDef,
				SqlMode:             sqlMode.String,
				CharacterSetClient:  charsetClient.String,
				CollationConnection: collationConnection.String,
				DatabaseCollation:   databaseCollation.String,
			})
		}
	}
	if err := routineRows.Err(); err != nil {
		return nil, nil, util.FormatErrorWithQuery(err, routinesQuery)
	}

	return functions, procedures, nil
}

func (driver *Driver) getCreateFunctionStmt(ctx context.Context, databaseName, functionName string) (string, error) {
	query := fmt.Sprintf("SHOW CREATE FUNCTION `%s`.`%s`", databaseName, functionName)
	rows, err := driver.db.QueryContext(ctx, query)
	if err != nil {
		return "", util.FormatErrorWithQuery(err, query)
	}
	defer rows.Close()

	var createFunction sql.NullString

	columns, err := rows.Columns()
	if err != nil {
		return "", err
	}
	defIdx := -1
	for i, column := range columns {
		if strings.EqualFold(column, "Create Function") {
			defIdx = i
			break
		}
	}

	if defIdx == -1 {
		return "", errors.Errorf("failed to find column Create Function")
	}

	for rows.Next() {
		dests := make([]any, len(columns))
		for i := 0; i < len(columns); i++ {
			if i == defIdx {
				dests[i] = &createFunction
				continue
			}
			dests[i] = new(string)
		}

		if err := rows.Scan(dests...); err != nil {
			return "", err
		}
	}

	if err := rows.Err(); err != nil {
		return "", err
	}

	if createFunction.Valid {
		f := createFunction.String

		functionSymbolIdx := strings.Index(f, " FUNCTION ")
		if functionSymbolIdx >= 0 {
			f = fmt.Sprintf("CREATE%s", f[functionSymbolIdx:])
		}

		if charsetIdx := strings.Index(f, " CHARSET "); charsetIdx != -1 {
			if newLineIdx := strings.Index(f, "\n"); newLineIdx != -1 {
				f = f[:charsetIdx] + f[newLineIdx:]
			}
		}

		return f, nil
	}
	return "", nil
}

func (driver *Driver) getCreateProcedureStmt(ctx context.Context, databaseName, functionName string) (string, error) {
	query := fmt.Sprintf("SHOW CREATE PROCEDURE `%s`.`%s`", databaseName, functionName)
	rows, err := driver.db.QueryContext(ctx, query)
	if err != nil {
		return "", util.FormatErrorWithQuery(err, query)
	}
	defer rows.Close()

	var createProcedure sql.NullString

	columns, err := rows.Columns()
	if err != nil {
		return "", err
	}
	defIdx := -1
	for i, column := range columns {
		if strings.EqualFold(column, "Create Procedure") {
			defIdx = i
			break
		}
	}

	if defIdx == -1 {
		return "", errors.Errorf("failed to find column Create Procedure")
	}

	for rows.Next() {
		dests := make([]any, len(columns))
		for i := 0; i < len(columns); i++ {
			if i == defIdx {
				dests[i] = &createProcedure
				continue
			}
			dests[i] = new(string)
		}

		if err := rows.Scan(dests...); err != nil {
			return "", err
		}
	}

	if err := rows.Err(); err != nil {
		return "", err
	}

	if createProcedure.Valid {
		p := createProcedure.String

		procedureSymbolIdx := strings.Index(p, " PROCEDURE ")
		if procedureSymbolIdx >= 0 {
			p = fmt.Sprintf("CREATE%s", p[procedureSymbolIdx:])
		}

		return p, nil
	}
	return "", nil
}

func setColumnMetadataDefault(column *storepb.ColumnMetadata, defaultStr sql.NullString, nullableBool bool, extra string) {
	if defaultStr.Valid {
		// In MySQL 5.7, the extra value is empty for a column with CURRENT_TIMESTAMP default.
		switch {
		case isCurrentTimestampLike(defaultStr.String):
			column.DefaultValue = &storepb.ColumnMetadata_DefaultExpression{DefaultExpression: defaultStr.String}
		case strings.Contains(extra, "DEFAULT_GENERATED"):
			column.DefaultValue = &storepb.ColumnMetadata_DefaultExpression{DefaultExpression: fmt.Sprintf("(%s)", defaultStr.String)}
		default:
			// For non-generated and non CURRENT_XXX default value, use string.
			column.DefaultValue = &storepb.ColumnMetadata_Default{Default: &wrapperspb.StringValue{Value: defaultStr.String}}
		}
	} else if strings.Contains(strings.ToUpper(extra), autoIncrementSymbol) {
		// TODO(zp): refactor column default value.
		// Use the upper case to consistent with MySQL Dump.
		column.DefaultValue = &storepb.ColumnMetadata_DefaultExpression{DefaultExpression: autoIncrementSymbol}
	} else if nullableBool {
		// This is NULL if the column has an explicit default of NULL,
		// or if the column definition includes no DEFAULT clause.
		// https://dev.mysql.com/doc/refman/8.0/en/information-schema-columns-table.html
		column.DefaultValue = &storepb.ColumnMetadata_DefaultNull{
			DefaultNull: true,
		}
	}

	if strings.Contains(extra, "on update CURRENT_TIMESTAMP") {
		re := regexp.MustCompile(`CURRENT_TIMESTAMP\((\d+)\)`)
		match := re.FindStringSubmatch(extra)
		if len(match) > 0 {
			digits := match[1]
			column.OnUpdate = fmt.Sprintf("CURRENT_TIMESTAMP(%s)", digits)
		} else {
			column.OnUpdate = "CURRENT_TIMESTAMP"
		}
	}
}

func isCurrentTimestampLike(s string) bool {
	upper := strings.ToUpper(s)
	if strings.HasPrefix(upper, "CURRENT_TIMESTAMP") {
		return true
	}
	if strings.HasPrefix(upper, "CURRENT_DATE") {
		return true
	}
	return false
}

func (driver *Driver) reconcileViewDefinition(ctx context.Context, databaseName, viewName string) (string, error) {
	query := fmt.Sprintf("SHOW CREATE VIEW `%s`.`%s`", databaseName, viewName)
	var createStmt, unused string
	if err := driver.db.QueryRowContext(ctx, query).Scan(&unused, &createStmt, &unused, &unused); err != nil {
		if noRows := errors.Is(err, sql.ErrNoRows); noRows {
			slog.Warn("no rows return for query show create view", slog.String("viewName", viewName), slog.String("databaseName", databaseName))
			return "", nil
		}
		return "", errors.Wrapf(err, "failed to scan row for query: %s", query)
	}

	def, err := getViewDefFromCreateView(createStmt)
	if err != nil {
		slog.Warn("failed to get view definition", slog.String("viewName", viewName), slog.String("databaseName", databaseName), log.BBError(err))
		return "", nil
	}

	return def, nil
}

func getViewDefFromCreateView(createView string) (string, error) {
	viewDefMatching := viewDefMatcher.FindStringSubmatch(createView)
	if len(viewDefMatching) == 0 {
		return "", errors.Errorf("failed to match view definition, %s", createView)
	}
	for i, name := range viewDefMatcher.SubexpNames() {
		if name == "def" && i < len(viewDefMatching) {
			return viewDefMatching[i], nil
		}
	}
	return "", errors.Errorf("failed to match view definition, %s", createView)
}

func (driver *Driver) listPartitionTables(ctx context.Context, databaseName string) (map[db.TableKey][]*storepb.TablePartitionMetadata, error) {
	const query string = `
		SELECT
			TABLE_NAME,
			PARTITION_NAME,
			SUBPARTITION_NAME,
			PARTITION_METHOD,
			SUBPARTITION_METHOD,
			PARTITION_EXPRESSION,
			SUBPARTITION_EXPRESSION,
			PARTITION_DESCRIPTION
		FROM INFORMATION_SCHEMA.PARTITIONS
		WHERE TABLE_SCHEMA = ? AND PARTITION_NAME IS NOT NULL
		ORDER BY TABLE_NAME ASC, PARTITION_NAME ASC, SUBPARTITION_NAME ASC, PARTITION_ORDINAL_POSITION ASC, SUBPARTITION_ORDINAL_POSITION ASC;
	`
	// Prepare the query statement.
	stmt, err := driver.db.PrepareContext(ctx, query)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to prepare query: %s", query)
	}
	defer stmt.Close()
	rows, err := stmt.QueryContext(ctx, databaseName)
	if err != nil {
		return nil, util.FormatErrorWithQuery(err, query)
	}
	defer rows.Close()

	type partitionKey struct {
		tableName     string
		partitionName string
	}

	partitionMap := make(map[partitionKey]int)
	result := make(map[db.TableKey][]*storepb.TablePartitionMetadata)

	for rows.Next() {
		var tableName, partitionName, partitionMethod string
		var subpartitionName, subpartitionMethod, subpartitionExpression, partitionExpression, partitionDescription sql.NullString
		if err := rows.Scan(
			&tableName,
			&partitionName,
			&subpartitionName,
			&partitionMethod,
			&subpartitionMethod,
			&partitionExpression,
			&subpartitionExpression,
			&partitionDescription,
		); err != nil {
			return nil, errors.Wrapf(err, "failed to scan row")
		}
		partitionKey := partitionKey{tableName: tableName, partitionName: partitionName}
		tableKey := db.TableKey{Schema: "", Table: tableName}

		if _, ok := partitionMap[partitionKey]; !ok {
			// Partition
			tp := convertToStorepbTablePartitionType(partitionMethod)
			if tp == storepb.TablePartitionMetadata_TYPE_UNSPECIFIED {
				slog.Warn("unknown partition type", slog.String("partitionMethod", partitionMethod))
				continue
			}
			// For the key partition, it can take zero or more columns, the partition expression is null if taken zero columns.
			expression := ""
			if partitionExpression.Valid {
				expression = partitionExpression.String
			}

			value := ""
			if partitionDescription.Valid {
				value = partitionDescription.String
			}

			partition := &storepb.TablePartitionMetadata{
				Name:          partitionName,
				Type:          tp,
				Expression:    expression,
				Value:         value,
				Subpartitions: []*storepb.TablePartitionMetadata{},
			}
			partitionMap[partitionKey] = len(result[tableKey])
			result[tableKey] = append(result[tableKey], partition)
		}

		if subpartitionName.Valid {
			tp := convertToStorepbTablePartitionType(subpartitionMethod.String)
			if tp == storepb.TablePartitionMetadata_TYPE_UNSPECIFIED {
				slog.Warn("unknown subpartition type", slog.String("subpartitionMethod", subpartitionMethod.String))
				continue
			}
			// For the key partition, it can take zero or more columns, the partition expression is null if taken zero columns.
			expression := ""
			if subpartitionExpression.Valid {
				expression = subpartitionExpression.String
			}

			subPartition := &storepb.TablePartitionMetadata{
				Name:          subpartitionName.String,
				Type:          tp,
				Expression:    expression,
				Value:         "",
				Subpartitions: []*storepb.TablePartitionMetadata{},
			}

			if idx, ok := partitionMap[partitionKey]; !ok {
				slog.Warn("subpartition without partition", slog.String("tableName", tableName), slog.String("partitionName", partitionName), slog.String("subpartitionName", subpartitionName.String))
			} else {
				result[tableKey][idx].Subpartitions = append(result[tableKey][idx].Subpartitions, subPartition)
			}
		}
	}

	if err := rows.Err(); err != nil {
		return nil, errors.Wrapf(err, "failed to scan row")
	}

	// We cannot get use whether the table is partitioned by server default from metadata, so we need to
	// use regexp for dump table string.
	for tableKey, partitions := range result {
		if len(partitions) == 0 {
			continue
		}
		showQuery := fmt.Sprintf("SHOW CREATE TABLE `%s`.`%s`", databaseName, tableKey.Table)
		showRows, err := driver.db.QueryContext(ctx, showQuery)
		if err != nil {
			slog.Warn("failed to execute query", slog.String("query", showQuery), log.BBError(err))
		}
		for showRows.Next() {
			var tableName, createTable string
			if err := showRows.Scan(&tableName, &createTable); err != nil {
				slog.Warn("failed to scan row", slog.String("query", showQuery), log.BBError(err))
			}
			partitionRegexp := regexp.MustCompile(`[^B]PARTITIONS (?P<partitionNum>\d+)`)
			subPartitionRegexp := regexp.MustCompile(`SUBPARTITIONS (?P<subPartitionNum>\d+)`)
			partitionNum := 0
			subPartitionNum := 0
			if partitionRegexp.MatchString(createTable) {
				partitionNum, err = strconv.Atoi(partitionRegexp.FindStringSubmatch(createTable)[1])
				if err != nil {
					slog.Warn("failed to parse partition number", slog.String("query", showQuery), log.BBError(err))
				}
			}
			if subPartitionRegexp.MatchString(createTable) {
				subPartitionNum, err = strconv.Atoi(subPartitionRegexp.FindStringSubmatch(createTable)[1])
				if err != nil {
					slog.Warn("failed to parse subpartition number", slog.String("query", showQuery), log.BBError(err))
				}
			}
			for _, partition := range partitions {
				if partitionNum != 0 {
					partition.UseDefault = strconv.Itoa(partitionNum)
				}
				for _, subPartition := range partition.Subpartitions {
					if subPartitionNum != 0 {
						subPartition.UseDefault = strconv.Itoa(subPartitionNum)
					}
				}
			}
		}
		if err := showRows.Err(); err != nil {
			slog.Warn("rows err", slog.String("query", showQuery), log.BBError(err))
		}
		// nolint
		if err := showRows.Close(); err != nil {
			slog.Warn("failed to close row", slog.String("query", showQuery), log.BBError(err))
		}
	}

	return result, nil
}

func convertToStorepbTablePartitionType(tp string) storepb.TablePartitionMetadata_Type {
	switch strings.ToUpper(tp) {
	case "RANGE":
		return storepb.TablePartitionMetadata_RANGE
	case "RANGE COLUMNS":
		return storepb.TablePartitionMetadata_RANGE_COLUMNS
	case "LIST":
		return storepb.TablePartitionMetadata_LIST
	case "LIST COLUMNS":
		return storepb.TablePartitionMetadata_LIST_COLUMNS
	case "HASH":
		return storepb.TablePartitionMetadata_HASH
	case "KEY":
		return storepb.TablePartitionMetadata_KEY
	case "LINEAR HASH":
		return storepb.TablePartitionMetadata_LINEAR_HASH
	case "LINEAR KEY":
		return storepb.TablePartitionMetadata_LINEAR_KEY
	default:
		return storepb.TablePartitionMetadata_TYPE_UNSPECIFIED
	}
}

func (driver *Driver) getForeignKeyList(ctx context.Context, databaseName string) (map[db.TableKey][]*storepb.ForeignKeyMetadata, error) {
	fkQuery := `
		SELECT
			TABLE_NAME,
			CONSTRAINT_NAME,
			REFERENCED_TABLE_NAME,
			DELETE_RULE,
			UPDATE_RULE,
			MATCH_OPTION
		FROM INFORMATION_SCHEMA.REFERENTIAL_CONSTRAINTS
		WHERE LOWER(CONSTRAINT_SCHEMA) = ?;
	`

	kcuQuery := `
		SELECT
			TABLE_NAME,
			CONSTRAINT_NAME,
			COLUMN_NAME,
			REFERENCED_COLUMN_NAME
		FROM INFORMATION_SCHEMA.KEY_COLUMN_USAGE
		WHERE POSITION_IN_UNIQUE_CONSTRAINT IS NOT NULL AND LOWER(CONSTRAINT_SCHEMA) = ?
		ORDER BY TABLE_NAME, CONSTRAINT_NAME, ORDINAL_POSITION;
	`

	fkRows, err := driver.db.QueryContext(ctx, fkQuery, databaseName)
	if err != nil {
		return nil, util.FormatErrorWithQuery(err, fkQuery)
	}
	defer fkRows.Close()
	fkMap := make(map[db.IndexKey]*storepb.ForeignKeyMetadata)
	for fkRows.Next() {
		var tableName string
		var fk storepb.ForeignKeyMetadata
		if err := fkRows.Scan(
			&tableName,
			&fk.Name,
			&fk.ReferencedTable,
			&fk.OnDelete,
			&fk.OnUpdate,
			&fk.MatchType,
		); err != nil {
			return nil, err
		}
		key := db.IndexKey{Schema: "", Table: tableName, Index: fk.Name}
		fkMap[key] = &fk
	}
	if err := fkRows.Err(); err != nil {
		return nil, util.FormatErrorWithQuery(err, fkQuery)
	}

	kcuQueryRows, err := driver.db.QueryContext(ctx, kcuQuery, databaseName)
	if err != nil {
		return nil, util.FormatErrorWithQuery(err, kcuQuery)
	}
	defer kcuQueryRows.Close()
	for kcuQueryRows.Next() {
		var tableName, fkName, column, referencedColumn string
		if err := kcuQueryRows.Scan(
			&tableName,
			&fkName,
			&column,
			&referencedColumn,
		); err != nil {
			return nil, err
		}
		key := db.IndexKey{Schema: "", Table: tableName, Index: fkName}
		if fk, ok := fkMap[key]; ok {
			fk.Columns = append(fk.Columns, column)
			fk.ReferencedColumns = append(fk.ReferencedColumns, referencedColumn)
		}
	}
	if err := kcuQueryRows.Err(); err != nil {
		return nil, util.FormatErrorWithQuery(err, kcuQuery)
	}
	unordered := make(map[db.TableKey][]*storepb.ForeignKeyMetadata)
	for key, fk := range fkMap {
		tableKey := db.TableKey{Schema: "", Table: key.Table}
		unordered[tableKey] = append(unordered[tableKey], fk)
	}

	orderedResult := make(map[db.TableKey][]*storepb.ForeignKeyMetadata)
	for key, fks := range unordered {
		sort.Slice(fks, func(i, j int) bool {
			return fks[i].Name < fks[j].Name
		})
		orderedResult[key] = fks
	}
	return orderedResult, nil
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

	location, err := time.LoadLocation(timeZone)
	if err != nil {
		slog.Debug("failed to load time zone", slog.String("timeZone", timeZone), log.BBError(err))
		location, err = time.LoadLocation("Local")
		if err != nil {
			// This should never happen
			slog.Debug("failed to load time zone", slog.String("timeZone", "Local"), log.BBError(err))
		}
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

	return analyzeSlowLog(driver.dbType, logs)
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

func analyzeSlowLog(engine storepb.Engine, logs []*slowLog) (map[string]*storepb.SlowQueryStatistics, error) {
	logMap := make(map[string]map[string]*storepb.SlowQueryStatisticsItem)

	for _, log := range logs {
		databaseList := extractDatabase(engine, log.database, log.details.SqlText)
		fingerprint, err := mysqlparser.GetFingerprint(log.details.SqlText)
		if err != nil {
			return nil, errors.Wrapf(err, "get sql fingerprint failed, sql: %s", log.details.SqlText)
		}
		if len(fingerprint) > db.SlowQueryMaxLen {
			fingerprint, _ = common.TruncateString(fingerprint, db.SlowQueryMaxLen)
		}
		if len(log.details.SqlText) > db.SlowQueryMaxLen {
			log.details.SqlText, _ = common.TruncateString(log.details.SqlText, db.SlowQueryMaxLen)
		}

		for _, db := range databaseList {
			if _, ok := logMap[db]; !ok {
				logMap[db] = make(map[string]*storepb.SlowQueryStatisticsItem)
			}
			logMap[db][fingerprint] = mergeSlowLog(fingerprint, logMap[db][fingerprint], log.details)
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

func extractDatabase(engne storepb.Engine, defaultDB string, sql string) []string {
	resources, err := base.ExtractResourceList(engne, defaultDB /* currentDatabase */, "" /* currentSchema */, sql)
	if err != nil {
		// If we can't extract the database, we just use the default database.
		slog.Debug("extract database failed", log.BBError(err), slog.String("sql", sql))
		return []string{defaultDB}
	}
	databaseMap := make(map[string]bool)
	for _, resource := range resources {
		databaseMap[resource.Database] = true
	}
	var databases []string
	for database := range databaseMap {
		databases = append(databases, database)
	}
	if len(databases) == 0 {
		databases = append(databases, defaultDB)
	}
	return databases
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

func stripSingleQuote(s string) string {
	if len(s) >= 2 && s[0] == '\'' && s[len(s)-1] == '\'' {
		return s[1 : len(s)-1]
	}
	return s
}
