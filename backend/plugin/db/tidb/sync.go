package tidb

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/log"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/db/mysql"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
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
		// TiDB only
		"metrics_schema": true,
	}
	systemDatabaseClause = func() string {
		var l []string
		for k := range systemDatabases {
			l = append(l, fmt.Sprintf("'%s'", k))
		}
		return strings.Join(l, ", ")
	}()

	pkAutoRandomBitsRegex = regexp.MustCompile(`PK_AUTO_RANDOM_BITS=(\d+)`)
	rangeBitsRegex        = regexp.MustCompile(`RANGE BITS=(\d+)`)
)

// SyncInstance syncs the instance.
func (d *Driver) SyncInstance(ctx context.Context) (*db.InstanceMetadata, error) {
	version, err := d.getVersion(ctx)
	if err != nil {
		return nil, err
	}

	lowerCaseTableNames := 0
	lowerCaseTableNamesText, err := d.getServerVariable(ctx, "lower_case_table_names")
	if err != nil {
		slog.Debug("failed to get lower_case_table_names variable", log.BBError(err))
	} else {
		lowerCaseTableNames, err = strconv.Atoi(lowerCaseTableNamesText)
		if err != nil {
			slog.Debug("failed to parse lower_case_table_names variable", log.BBError(err))
		}
	}

	instanceRoles, err := d.getInstanceRoles(ctx)
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
	rows, err := d.db.QueryContext(ctx, query)
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
		Metadata: &storepb.Instance{
			MysqlLowerCaseTableNames: int32(lowerCaseTableNames),
			Roles:                    instanceRoles,
		},
	}, nil
}

func (d *Driver) getServerVariable(ctx context.Context, varName string) (string, error) {
	db := d.GetDB()
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

// SyncDBSchema syncs a single database schema.
func (d *Driver) SyncDBSchema(ctx context.Context) (*storepb.DatabaseSchemaMetadata, error) {
	schemaMetadata := &storepb.SchemaMetadata{
		Name: "",
	}

	// Query index info.
	indexMap := make(map[db.TableKey]map[string]*storepb.IndexMetadata)

	indexQuery := `
		SELECT
			TABLE_NAME,
			KEY_NAME,
			COLUMN_NAME,
			SEQ_IN_INDEX,
			NON_UNIQUE,
			INDEX_COMMENT,
			SUB_PART,
			EXPRESSION
		FROM information_schema.TIDB_INDEXES
		WHERE TABLE_SCHEMA = ?`
	indexRows, err := d.db.QueryContext(ctx, indexQuery, d.databaseName)
	if err != nil {
		return nil, util.FormatErrorWithQuery(err, indexQuery)
	}
	defer indexRows.Close()

	for indexRows.Next() {
		var tableName, keyName, comment string
		var columnName, expressionName sql.NullString
		var position int
		var nonUnique bool
		var subPart sql.NullInt64

		if err := indexRows.Scan(
			&tableName,
			&keyName,
			&columnName,
			&position,
			&nonUnique,
			&comment,
			&subPart,
			&expressionName,
		); err != nil {
			return nil, err
		}

		// Determine expression from column name or expression field
		var expression string
		if expressionName.Valid {
			expression = fmt.Sprintf("(%s)", expressionName.String)
		} else if columnName.Valid {
			expression = columnName.String
		}

		key := db.TableKey{Schema: "", Table: tableName}
		if _, ok := indexMap[key]; !ok {
			indexMap[key] = make(map[string]*storepb.IndexMetadata)
		}
		if _, ok := indexMap[key][keyName]; !ok {
			indexMap[key][keyName] = &storepb.IndexMetadata{
				Name:    keyName,
				Type:    "BTREE", // Default to BTREE as TiDB generally uses BTREE indexes
				Unique:  !nonUnique,
				Primary: keyName == "PRIMARY",
				Visible: true, // Visible is always true for TiDB indexes
				Comment: comment,
			}
		}

		// Add expression to index metadata
		indexMap[key][keyName].Expressions = append(indexMap[key][keyName].Expressions, expression)

		// Add key length to index metadata
		if subPart.Valid {
			indexMap[key][keyName].KeyLength = append(indexMap[key][keyName].KeyLength, subPart.Int64)
		} else {
			// -1 means no key length limit.
			indexMap[key][keyName].KeyLength = append(indexMap[key][keyName].KeyLength, -1)
		}
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
			GENERATION_EXPRESSION,
			EXTRA
		FROM information_schema.COLUMNS
		WHERE TABLE_SCHEMA = ?
		ORDER BY TABLE_NAME, ORDINAL_POSITION`
	columnRows, err := d.db.QueryContext(ctx, columnQuery, d.databaseName)
	if err != nil {
		return nil, util.FormatErrorWithQuery(err, columnQuery)
	}
	defer columnRows.Close()
	for columnRows.Next() {
		column := &storepb.ColumnMetadata{}
		var tableName, nullable, extra string
		var defaultStr, generationExpr sql.NullString
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
			&generationExpr,
			&extra,
		); err != nil {
			return nil, err
		}
		// Quoted string has a single quote around it and is escaped by QUOTE().
		column.Comment = stripSingleQuote(column.Comment)

		nullableBool, err := util.ConvertYesNo(nullable)
		if err != nil {
			return nil, err
		}
		column.Nullable = nullableBool
		setColumnMetadataDefault(column, defaultStr, nullableBool, extra)

		// Handle generated columns
		if generationExpr.Valid && generationExpr.String != "" {
			if strings.Contains(strings.ToUpper(extra), virtualGenerated) {
				column.Generation = &storepb.GenerationMetadata{
					Type:       storepb.GenerationMetadata_TYPE_VIRTUAL,
					Expression: generationExpr.String,
				}
			} else if strings.Contains(strings.ToUpper(extra), storedGenerated) {
				column.Generation = &storepb.GenerationMetadata{
					Type:       storepb.GenerationMetadata_TYPE_STORED,
					Expression: generationExpr.String,
				}
			}
		}

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
	viewRows, err := d.db.QueryContext(ctx, viewQuery, d.databaseName)
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
		// Note: TiDB/MySQL does not support view comments, so view.Comment remains empty
		viewMap[key] = view
	}
	if err := viewRows.Err(); err != nil {
		return nil, util.FormatErrorWithQuery(err, viewQuery)
	}

	// Query foreign key info.
	foreignKeysMap, err := d.getForeignKeyList(ctx, d.databaseName)
	if err != nil {
		return nil, err
	}

	// Query sequence info.
	sequences, err := d.getSequenceList(ctx, d.databaseName)
	if err != nil {
		return nil, err
	}

	// Query partition info.
	partitionTables, err := d.listPartitionTables(ctx, d.databaseName)
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
			QUOTE(IFNULL(TABLE_COMMENT, '')),
			IFNULL(TIDB_ROW_ID_SHARDING_INFO, ''),
			IFNULL(TIDB_PK_TYPE, '')
		FROM information_schema.TABLES
		WHERE TABLE_SCHEMA = ?
		ORDER BY TABLE_NAME`

	tableRows, err := d.db.QueryContext(ctx, tableQuery, d.databaseName)
	if err != nil {
		return nil, util.FormatErrorWithQuery(err, tableQuery)
	}
	defer tableRows.Close()
	for tableRows.Next() {
		var tableName, tableType, engine, collation, createOptions, comment, shardingInfo, pkType string
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
			&shardingInfo,
			&pkType,
		); err != nil {
			return nil, err
		}
		// Quoted string has a single quote around it and is escaped by QUOTE().
		comment = stripSingleQuote(comment)

		key := db.TableKey{Schema: "", Table: tableName}
		switch tableType {
		case baseTableType:
			columns := columnMap[key]
			// Set auto random default value for TiDB.
			if strings.Contains(shardingInfo, pkAutoRandomBitsSymbol) {
				autoRandText := autoRandSymbol
				if randomBitsMatch := pkAutoRandomBitsRegex.FindStringSubmatch(shardingInfo); len(randomBitsMatch) > 1 {
					if rangeBitsMatch := rangeBitsRegex.FindStringSubmatch(shardingInfo); len(rangeBitsMatch) > 1 {
						autoRandText += fmt.Sprintf("(%s, %s)", randomBitsMatch[1], rangeBitsMatch[1])
					} else {
						autoRandText += fmt.Sprintf("(%s)", randomBitsMatch[1])
					}
				}
				if indexes, ok := indexMap[key]; ok {
					for _, index := range indexes {
						if index.Primary {
							if len(index.Expressions) > 0 {
								columnName := index.Expressions[0]
								for i, column := range columns {
									if column.Name == columnName {
										newColumn := columns[i]
										// Store AUTO_RANDOM in Default field (migration from DefaultExpression to Default)
										newColumn.Default = autoRandText
										break
									}
								}
							}
						}
					}
				}
			}

			tableMetadata := &storepb.TableMetadata{
				Name:           tableName,
				Columns:        columns,
				ForeignKeys:    foreignKeysMap[key],
				Engine:         engine,
				Collation:      collation,
				RowCount:       rowCount,
				DataSize:       dataSize,
				IndexSize:      indexSize,
				DataFree:       dataFree,
				CreateOptions:  createOptions,
				Comment:        comment,
				Charset:        convertCollationToCharset(collation),
				Partitions:     partitionTables[key],
				ShardingInfo:   shardingInfo,
				PrimaryKeyType: pkType,
			}
			if tableCollation.Valid {
				tableMetadata.Collation = tableCollation.String
			}
			var indexNames []string
			if indexes, ok := indexMap[key]; ok {
				for indexName := range indexes {
					indexNames = append(indexNames, indexName)
				}
				slices.Sort(indexNames)
				for _, indexName := range indexNames {
					tableMetadata.Indexes = append(tableMetadata.Indexes, indexes[indexName])
				}
			}

			schemaMetadata.Tables = append(schemaMetadata.Tables, tableMetadata)
		case viewTableType:
			if view, ok := viewMap[key]; ok {
				schemaMetadata.Views = append(schemaMetadata.Views, view)
			}
		default:
			// Skip unknown table types (e.g., SEQUENCE, SYSTEM VERSIONED)
			slog.Debug("skipping unknown table type", slog.String("tableType", tableType), slog.String("tableName", tableName))
		}
	}
	if err := tableRows.Err(); err != nil {
		return nil, util.FormatErrorWithQuery(err, tableQuery)
	}

	// Add sequences to schema metadata
	schemaMetadata.Sequences = sequences

	databaseMetadata := &storepb.DatabaseSchemaMetadata{
		Name:    d.databaseName,
		Schemas: []*storepb.SchemaMetadata{schemaMetadata},
	}
	// Query db info.
	databaseQuery := `
		SELECT
			DEFAULT_CHARACTER_SET_NAME,
			DEFAULT_COLLATION_NAME
		FROM information_schema.SCHEMATA
		WHERE SCHEMA_NAME = ?`
	if err := d.db.QueryRowContext(ctx, databaseQuery, d.databaseName).Scan(
		&databaseMetadata.CharacterSet,
		&databaseMetadata.Collation,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, common.Errorf(common.NotFound, "database %q not found", d.databaseName)
		}
		return nil, err
	}

	return databaseMetadata, err
}

func convertCollationToCharset(collation string) string {
	// See mappings from "SHOW CHARACTER SET;".
	tokens := strings.Split(collation, "_")
	return tokens[0]
}

func isTimeConstant(s string) bool {
	// 0000-00-00 00:00:00 is a special case in TiDB.
	if s == "0000-00-00 00:00:00" {
		return true
	}
	_, err := time.Parse(time.DateTime, s)
	return err == nil
}

func setColumnMetadataDefault(column *storepb.ColumnMetadata, defaultStr sql.NullString, nullableBool bool, extra string) {
	if defaultStr.Valid {
		// TiDB is MySQL-compatible, so use MySQL's default handling approach
		unquotedDefault := mysql.UnquoteMySQLString(defaultStr.String)
		switch {
		case mysql.IsCurrentTimestampLike(unquotedDefault):
			column.Default = unquotedDefault
		case strings.Contains(extra, "DEFAULT_GENERATED"):
			// for case:
			//  CREATE TABLE t1(
			//    update_time TIMESTAMP DEFAULT '0000-00-00 00:00:00' ON UPDATE CURRENT_TIMESTAMP,
			//  );
			// In this case, the extra value is "DEFAULT_GENERATED on update CURRENT_TIMESTAMP".
			// But the default value is a constant.
			if isTimeConstant(defaultStr.String) {
				column.Default = defaultStr.String
			} else {
				unescapedDefault := mysql.UnescapeExpressionDefault(unquotedDefault)
				column.Default = fmt.Sprintf("(%s)", unescapedDefault)
			}
		default:
			// For non-generated and non CURRENT_XXX default value, preserve quotes for mysqldump compatibility
			column.Default = defaultStr.String
		}
	} else if strings.Contains(strings.ToUpper(extra), autoIncrementSymbol) {
		// TODO(zp): refactor column default value.
		// Use the upper case to consistent with MySQL Dump.
		column.Default = autoIncrementSymbol
	} else if nullableBool {
		// This is NULL if the column has an explicit default of NULL,
		// or if the column definition includes no DEFAULT clause.
		// https://dev.mysql.com/doc/refman/8.0/en/information-schema-columns-table.html
		column.Default = "NULL"
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

func (d *Driver) listPartitionTables(ctx context.Context, databaseName string) (map[db.TableKey][]*storepb.TablePartitionMetadata, error) {
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
	stmt, err := d.db.PrepareContext(ctx, query)
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
			if partitionExpression.Valid {
				expression = subpartitionExpression.String
			}

			value := ""
			if partitionDescription.Valid {
				value = partitionDescription.String
			}

			subPartition := &storepb.TablePartitionMetadata{
				Name:          subpartitionName.String,
				Type:          tp,
				Expression:    expression,
				Value:         value,
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

func (d *Driver) getForeignKeyList(ctx context.Context, databaseName string) (map[db.TableKey][]*storepb.ForeignKeyMetadata, error) {
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
		WHERE kcu.POSITION_IN_UNIQUE_CONSTRAINT IS NOT NULL AND LOWER(fks.CONSTRAINT_SCHEMA) = ?
		ORDER BY fks.TABLE_NAME, fks.CONSTRAINT_NAME, kcu.ORDINAL_POSITION;
	`

	fkRows, err := d.db.QueryContext(ctx, fkQuery, databaseName)
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

func stripSingleQuote(s string) string {
	if len(s) >= 2 && s[0] == '\'' && s[len(s)-1] == '\'' {
		return s[1 : len(s)-1]
	}
	return s
}

func (d *Driver) getSequenceList(ctx context.Context, databaseName string) ([]*storepb.SequenceMetadata, error) {
	query := `
		SELECT
			SEQUENCE_NAME,
			START,
			MIN_VALUE,
			MAX_VALUE,
			INCREMENT,
			CYCLE,
			CACHE_VALUE,
			IFNULL(COMMENT, '')
		FROM information_schema.SEQUENCES
		WHERE SEQUENCE_SCHEMA = ?
		ORDER BY SEQUENCE_NAME`

	rows, err := d.db.QueryContext(ctx, query, databaseName)
	if err != nil {
		return nil, util.FormatErrorWithQuery(err, query)
	}
	defer rows.Close()

	var sequences []*storepb.SequenceMetadata
	for rows.Next() {
		sequence := &storepb.SequenceMetadata{}
		var cycleOption int64

		if err := rows.Scan(
			&sequence.Name,
			&sequence.Start,
			&sequence.MinValue,
			&sequence.MaxValue,
			&sequence.Increment,
			&cycleOption,
			&sequence.CacheSize,
			&sequence.Comment,
		); err != nil {
			return nil, err
		}

		// TiDB sequences are always numeric, set default data type
		sequence.DataType = "BIGINT"

		// Convert cycle option to boolean (TiDB uses 0/1 instead of YES/NO)
		sequence.Cycle = cycleOption != 0

		sequences = append(sequences, sequence)
	}

	if err := rows.Err(); err != nil {
		return nil, util.FormatErrorWithQuery(err, query)
	}

	return sequences, nil
}
