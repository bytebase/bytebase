package starrocks

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"strconv"
	"strings"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/log"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/db/mysql"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
)

const (
	autoIncrementSymbol = "AUTO_INCREMENT"
)

// systemDatabaseExclusion returns the quoted, comma-joined system databases to
// exclude from instance discovery, per the driver's engine. StarRocks adds its
// read-only `sys` metadatabase (Doris has no sys), matching the parser-side
// system-database classification.
func (d *Driver) systemDatabaseExclusion() string {
	dbs := []string{"'information_schema'", "'_statistics_'"}
	if d.dbType == storepb.Engine_STARROCKS {
		dbs = append(dbs, "'sys'")
	}
	return strings.Join(dbs, ", ")
}

// SyncInstance syncs the instance.
func (d *Driver) SyncInstance(ctx context.Context) (*db.InstanceMetadata, error) {
	version, _, err := d.getVersion(ctx)
	if err != nil {
		return nil, err
	}

	var lowerCaseTableNames int32
	lowerCaseTableNamesText, err := d.getServerVariable(ctx, "lower_case_table_names")
	if err != nil {
		slog.Debug("failed to get lower_case_table_names variable", log.BBError(err))
	} else {
		v, err := strconv.ParseInt(lowerCaseTableNamesText, 10, 32)
		if err != nil {
			slog.Debug("failed to parse lower_case_table_names variable", log.BBError(err))
		} else {
			lowerCaseTableNames = int32(v)
		}
	}

	// Query db info
	where := fmt.Sprintf("LOWER(SCHEMA_NAME) NOT IN (%s)", d.systemDatabaseExclusion())
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
		if d.dbType == storepb.Engine_DORIS || d.dbType == storepb.Engine_STARROCKS {
			database.CharacterSet = ""
			database.Collation = ""
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
			MysqlLowerCaseTableNames: lowerCaseTableNames,
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

// isSyncRollup reports whether a StarRocks information_schema.materialized_views row
// describes a synchronous rollup rather than a standalone (async) materialized view.
// Sync rollups are rollup indexes on the base table: they appear in that catalog with
// REFRESH_TYPE='ROLLUP' but have no information_schema.tables row, so they are excluded
// from the materialized-view set. Only the confirmed 'ROLLUP' value is excluded — async
// refresh types and any unknown/empty value are kept.
func isSyncRollup(refreshType string) bool {
	return refreshType == "ROLLUP"
}

// isMaterializedView reports whether a table row is a materialized view, based on its
// table type and membership in the materialized-view set. StarRocks reports
// materialized views with TABLE_TYPE='VIEW', Doris with 'BASE TABLE', and some setups
// with 'MATERIALIZED VIEW'; in every case, presence in materializedViewMap is the
// authoritative signal.
func isMaterializedView(tableType string, key db.TableKey, materializedViewMap map[db.TableKey]*storepb.MaterializedViewMetadata) bool {
	if _, ok := materializedViewMap[key]; !ok {
		return false
	}
	switch tableType {
	case baseTableType, viewTableType, materializedViewType:
		return true
	default:
		return false
	}
}

// getMaterializedViews returns the materialized views of the current database keyed by
// table name. Doris and StarRocks expose them through different catalogs, so the query
// and projection are engine-specific. The query is best-effort: an engine/version that
// does not support it is logged and yields an empty set rather than failing the sync.
func (d *Driver) getMaterializedViews(ctx context.Context) (map[db.TableKey]*storepb.MaterializedViewMetadata, error) {
	materializedViewMap := make(map[db.TableKey]*storepb.MaterializedViewMetadata)

	var query string
	switch d.dbType {
	case storepb.Engine_DORIS:
		// Doris exposes materialized views via the mv_infos() table-valued function.
		// https://doris.apache.org/docs/sql-manual/sql-functions/table-valued-functions/mv_infos
		query = fmt.Sprintf(`SELECT Name, QuerySql FROM mv_infos("database"="%s")`, d.databaseName)
	case storepb.Engine_STARROCKS:
		// StarRocks exposes materialized views via information_schema.materialized_views.
		// REFRESH_TYPE separates async MVs from synchronous rollups, which are excluded.
		// IFNULL guards the bare-string scans below: database/sql errors on a NULL->string scan.
		query = fmt.Sprintf(`SELECT TABLE_NAME, IFNULL(REFRESH_TYPE, ''), IFNULL(MATERIALIZED_VIEW_DEFINITION, '') FROM information_schema.materialized_views WHERE TABLE_SCHEMA = '%s'`, d.databaseName)
	default:
		return materializedViewMap, nil
	}

	rows, err := d.db.QueryContext(ctx, query)
	if err != nil {
		// The catalog may be unavailable on older engine versions; log and continue.
		slog.Debug("failed to query materialized views, might not be supported in this version", log.BBError(err))
		return materializedViewMap, nil
	}
	defer rows.Close()

	for rows.Next() {
		materializedView := &storepb.MaterializedViewMetadata{}
		if d.dbType == storepb.Engine_STARROCKS {
			var refreshType string
			if err := rows.Scan(&materializedView.Name, &refreshType, &materializedView.Definition); err != nil {
				return nil, err
			}
			if isSyncRollup(refreshType) {
				continue
			}
		} else if err := rows.Scan(&materializedView.Name, &materializedView.Definition); err != nil {
			return nil, err
		}
		key := db.TableKey{Schema: "", Table: materializedView.Name}
		materializedViewMap[key] = materializedView
	}
	if err := rows.Err(); err != nil {
		return nil, util.FormatErrorWithQuery(err, query)
	}
	return materializedViewMap, nil
}

// SyncDBSchema syncs a single database schema.
func (d *Driver) SyncDBSchema(ctx context.Context) (*storepb.DatabaseSchemaMetadata, error) {
	schemaMetadata := &storepb.SchemaMetadata{
		Name: "",
	}

	// There is not yet a way to list indexes from information_schema.
	// Query column info.
	columnMap := make(map[db.TableKey][]*storepb.ColumnMetadata)
	columnQuery := fmt.Sprintf(`
		SELECT
			TABLE_NAME,
			IFNULL(COLUMN_NAME, ''),
			ORDINAL_POSITION,
			COLUMN_DEFAULT,
			IS_NULLABLE,
			COLUMN_TYPE,
			IFNULL(CHARACTER_SET_NAME, ''),
			IFNULL(COLLATION_NAME, ''),
			COLUMN_COMMENT,
			EXTRA
		FROM information_schema.columns
		WHERE TABLE_SCHEMA = '%s'
		ORDER BY TABLE_NAME, ORDINAL_POSITION`, d.databaseName)
	columnRows, err := d.db.QueryContext(ctx, columnQuery)
	if err != nil {
		return nil, util.FormatErrorWithQuery(err, columnQuery)
	}
	defer columnRows.Close()
	for columnRows.Next() {
		column := &storepb.ColumnMetadata{}
		var tableName, nullable, extra string
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
			&extra,
		); err != nil {
			return nil, err
		}
		isNullBool, err := util.ConvertYesNo(nullable)
		if err != nil {
			return nil, err
		}
		if defaultStr.Valid {
			// StarRocks is MySQL-compatible, so use MySQL's default handling approach
			unquotedDefault := mysql.UnquoteMySQLString(defaultStr.String)
			switch {
			case mysql.IsCurrentTimestampLike(unquotedDefault):
				column.Default = unquotedDefault
			case strings.Contains(extra, "DEFAULT_GENERATED"):
				unescapedDefault := mysql.UnescapeExpressionDefault(unquotedDefault)
				column.Default = fmt.Sprintf("(%s)", unescapedDefault)
			default:
				// For non-generated and non CURRENT_XXX default value, preserve quotes for mysqldump compatibility
				column.Default = defaultStr.String
			}
		} else if strings.Contains(strings.ToUpper(extra), autoIncrementSymbol) {
			// TODO(zp): refactor column default value.
			// Use the upper case to consistent with MySQL Dump.
			column.Default = autoIncrementSymbol
		} else if isNullBool {
			// This is NULL if the column has an explicit default of NULL,
			// or if the column definition includes no DEFAULT clause.
			// https://dev.mysql.com/doc/refman/8.0/en/information-schema-columns-table.html
			column.Default = "NULL"
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
	viewQuery := fmt.Sprintf(`
		SELECT
			TABLE_NAME,
			VIEW_DEFINITION
		FROM information_schema.VIEWS
		WHERE TABLE_SCHEMA = '%s'`, d.databaseName)
	viewRows, err := d.db.QueryContext(ctx, viewQuery)
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

	// Query materialized view info (engine-specific catalog; best-effort).
	materializedViewMap, err := d.getMaterializedViews(ctx)
	if err != nil {
		return nil, err
	}

	// Query table info.
	tableQuery := fmt.Sprintf(`
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
		WHERE TABLE_SCHEMA = '%s'
		ORDER BY TABLE_NAME`, d.databaseName)
	tableRows, err := d.db.QueryContext(ctx, tableQuery)
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
		// StarRocks reports materialized views as TABLE_TYPE='VIEW', Doris as 'BASE TABLE';
		// route any row whose name is in the materialized-view set to MaterializedViews.
		if isMaterializedView(tableType, key, materializedViewMap) {
			materializedView := materializedViewMap[key]
			materializedView.Comment = comment
			schemaMetadata.MaterializedViews = append(schemaMetadata.MaterializedViews, materializedView)
			continue
		}
		switch tableType {
		case baseTableType:
			tableMetadata := &storepb.TableMetadata{
				Name:          tableName,
				Columns:       columnMap[key],
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
			// TODO(d): add index information whenever it is available.
			schemaMetadata.Tables = append(schemaMetadata.Tables, tableMetadata)
		case viewTableType:
			if view, ok := viewMap[key]; ok {
				view.Comment = comment
				schemaMetadata.Views = append(schemaMetadata.Views, view)
			}
		default:
			// Includes 'MATERIALIZED VIEW' rows absent from the materialized-view set
			// and any other unrecognized type.
			slog.Debug("skipping unhandled table type", slog.String("tableName", tableName), slog.String("tableType", tableType))
		}
	}
	if err := tableRows.Err(); err != nil {
		return nil, util.FormatErrorWithQuery(err, tableQuery)
	}

	databaseMetadata := &storepb.DatabaseSchemaMetadata{
		Name:    d.databaseName,
		Schemas: []*storepb.SchemaMetadata{schemaMetadata},
	}
	// Query db info.
	databaseQuery := fmt.Sprintf(`
		SELECT
			DEFAULT_CHARACTER_SET_NAME,
			DEFAULT_COLLATION_NAME
		FROM information_schema.SCHEMATA
		WHERE SCHEMA_NAME = '%s'`, d.databaseName)
	if err := d.db.QueryRowContext(ctx, databaseQuery).Scan(
		&databaseMetadata.CharacterSet,
		&databaseMetadata.Collation,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, common.Errorf(common.NotFound, "database %q not found", d.databaseName)
		}
		return nil, err
	}
	// "characterSet":"utf8\u0000", "collation":"utf8_general_ci\u0000".
	// ERROR: unsupported Unicode escape sequence (SQLSTATE 22P05).
	if d.dbType == storepb.Engine_DORIS || d.dbType == storepb.Engine_STARROCKS {
		databaseMetadata.CharacterSet = ""
		databaseMetadata.Collation = ""
	}

	return databaseMetadata, err
}
