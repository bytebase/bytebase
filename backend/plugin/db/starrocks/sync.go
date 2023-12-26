package starrocks

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

const (
	autoIncrementSymbol = "AUTO_INCREMENT"
)

var (
	systemDatabases = map[string]bool{
		"information_schema": true,
		"_statistics_":       true,
	}
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
		Version:       version,
		InstanceRoles: users,
		Databases:     databases,
		Metadata: &storepb.InstanceMetadata{
			MysqlLowerCaseTableNames: int32(lowerCaseTableNames),
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

// SyncDBSchema syncs a single database schema.
func (driver *Driver) SyncDBSchema(ctx context.Context) (*storepb.DatabaseSchemaMetadata, error) {
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
		ORDER BY TABLE_NAME, ORDINAL_POSITION`, driver.databaseName)
	columnRows, err := driver.db.QueryContext(ctx, columnQuery)
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
		if defaultStr.Valid {
			if strings.Contains(extra, "DEFAULT_GENERATED") {
				column.DefaultValue = &storepb.ColumnMetadata_DefaultExpression{DefaultExpression: fmt.Sprintf("(%s)", defaultStr.String)}
			} else {
				column.DefaultValue = &storepb.ColumnMetadata_Default{Default: &wrapperspb.StringValue{Value: defaultStr.String}}
			}
		} else {
			// TODO(zp): refactor column default value.
			if strings.Contains(strings.ToUpper(extra), autoIncrementSymbol) {
				// Use the upper case to consistent with MySQL Dump.
				column.DefaultValue = &storepb.ColumnMetadata_DefaultExpression{DefaultExpression: autoIncrementSymbol}
			}
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
	viewQuery := fmt.Sprintf(`
		SELECT
			TABLE_NAME,
			VIEW_DEFINITION
		FROM information_schema.VIEWS
		WHERE TABLE_SCHEMA = '%s'`, driver.databaseName)
	viewRows, err := driver.db.QueryContext(ctx, viewQuery)
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
		ORDER BY TABLE_NAME`, driver.databaseName)
	tableRows, err := driver.db.QueryContext(ctx, tableQuery)
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
	databaseQuery := fmt.Sprintf(`
		SELECT
			DEFAULT_CHARACTER_SET_NAME,
			DEFAULT_COLLATION_NAME
		FROM information_schema.SCHEMATA
		WHERE SCHEMA_NAME = '%s'`, driver.databaseName)
	if err := driver.db.QueryRowContext(ctx, databaseQuery).Scan(
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

// SyncSlowQuery syncs slow query.
func (*Driver) SyncSlowQuery(_ context.Context, _ time.Time) (map[string]*storepb.SlowQueryStatistics, error) {
	return nil, errors.Errorf("not implemented")
}

// CheckSlowQueryLogEnabled checks whether the slow query log is enabled.
func (*Driver) CheckSlowQueryLogEnabled(_ context.Context) error {
	return errors.Errorf("not implemented")
}
