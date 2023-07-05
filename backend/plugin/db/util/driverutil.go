// Package util provides the library for database drivers.
package util

import (
	"bufio"
	"context"
	"database/sql"
	"fmt"
	"io"
	"math/big"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/blang/semver/v4"
	"github.com/google/uuid"
	"github.com/paulmach/orb"
	"github.com/paulmach/orb/encoding/wkt"
	"github.com/pkg/errors"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/plugin/db"
	parser "github.com/bytebase/bytebase/backend/plugin/parser/sql"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
)

// FormatErrorWithQuery will format the error with failed query.
func FormatErrorWithQuery(err error, query string) error {
	if regexp.MustCompile("does not exist").MatchString(err.Error()) {
		return common.Wrapf(err, common.NotFound, "failed to execute query %q", query)
	}
	return common.Wrapf(err, common.DbExecutionError, "failed to execute query %q", query)
}

// ApplyMultiStatements will apply the split statements from scanner.
// This function only used for SQLite, snowflake and clickhouse.
// For MySQL and PostgreSQL, use parser.SplitMultiSQLStream instead.
func ApplyMultiStatements(sc io.Reader, f func(string) error) error {
	// TODO(rebelice): use parser/tokenizer to split SQL statements.
	reader := bufio.NewReader(sc)
	var sb strings.Builder
	delimiter := false
	comment := false
	done := false
	for !done {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				done = true
			} else {
				return err
			}
		}
		line = strings.TrimRight(line, "\r\n")

		execute := false
		switch {
		case strings.HasPrefix(line, "/*"):
			if strings.Contains(line, "*/") {
				if !strings.HasSuffix(line, "*/") {
					return errors.Errorf("`*/` must be the end of the line; new statement should start as a new line")
				}
			} else {
				comment = true
			}
			continue
		case comment && !strings.Contains(line, "*/"):
			// Skip the line when in comment mode.
			continue
		case comment && strings.Contains(line, "*/"):
			if !strings.HasSuffix(line, "*/") {
				return errors.Errorf("`*/` must be the end of the line; new statement should start as a new line")
			}
			comment = false
			continue
		case sb.Len() == 0 && line == "":
			continue
		case strings.HasPrefix(line, "--"):
			continue
		case line == "DELIMITER ;;":
			delimiter = true
			continue
		case line == "DELIMITER ;" && delimiter:
			delimiter = false
			execute = true
		case strings.HasSuffix(line, ";"):
			_, _ = sb.WriteString(line)
			_, _ = sb.WriteString("\n")
			if !delimiter {
				execute = true
			}
		default:
			_, _ = sb.WriteString(line)
			_, _ = sb.WriteString("\n")
			continue
		}
		if execute {
			s := sb.String()
			s = strings.Trim(s, "\n\t ")
			if s != "" {
				if err := f(s); err != nil {
					return errors.Wrapf(err, "execute query %q failed", s)
				}
			}
			sb.Reset()
		}
	}
	// Apply the remaining content.
	s := sb.String()
	s = strings.Trim(s, "\n\t ")
	if s != "" {
		if err := f(s); err != nil {
			return errors.Wrapf(err, "execute query %q failed", s)
		}
	}

	return nil
}

// Query will execute a readonly / SELECT query.
// The result is then JSON marshaled and returned to the frontend.
func Query(ctx context.Context, dbType db.Type, conn *sql.Conn, statement string, queryContext *db.QueryContext) ([]any, error) {
	readOnly := queryContext.ReadOnly
	limit := queryContext.Limit
	if !readOnly {
		return queryAdmin(ctx, dbType, conn, statement, limit)
	}

	statement = strings.TrimRight(statement, " \n\t;")
	// Limit SQL query result size.
	if !strings.HasPrefix(statement, "EXPLAIN") && limit > 0 {
		if dbType == db.MySQL || dbType == db.MariaDB {
			// MySQL 5.7 doesn't support WITH clause.
			statement = getMySQLStatementWithResultLimit(statement, limit)
		} else if dbType == db.Oracle {
			statement = getOracleStatementWithResultLimit(statement, limit)
		} else if dbType == db.MSSQL {
			statement = getMSSQLStatementWithResultLimit(statement, limit)
		} else {
			statement = getStatementWithResultLimit(statement, limit)
		}
	}

	// TiDB doesn't support READ ONLY transactions. We have to skip the flag for it.
	// https://github.com/pingcap/tidb/issues/34626
	// Clickhouse doesn't support READ ONLY transactions (Error: sql: driver does not support read-only transactions).
	// Snowflake doesn't support READ ONLY transactions.
	// https://github.com/snowflakedb/gosnowflake/blob/0450f0b16a4679b216baecd3fd6cdce739dbb683/connection.go#L166
	if dbType == db.TiDB || dbType == db.ClickHouse || dbType == db.Snowflake || dbType == db.Spanner || dbType == db.Redis || dbType == db.Oracle || dbType == db.MSSQL {
		readOnly = false
	}
	tx, err := conn.BeginTx(ctx, &sql.TxOptions{ReadOnly: readOnly})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	rows, err := tx.QueryContext(ctx, statement)
	if err != nil {
		return nil, FormatErrorWithQuery(err, statement)
	}
	defer rows.Close()

	columnNames, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	fieldList, err := extractSensitiveField(dbType, statement, queryContext.CurrentDatabase, queryContext.SensitiveSchemaInfo)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to extract sensitive fields: %q", statement)
	}

	if len(fieldList) != 0 && len(fieldList) != len(columnNames) {
		return nil, errors.Errorf("failed to extract sensitive fields: %q", statement)
	}

	var fieldMaskInfo []bool
	var fieldSensitiveInfo []bool

	for i := range columnNames {
		sensitive := len(fieldList) > 0 && fieldList[i].Sensitive
		fieldSensitiveInfo = append(fieldSensitiveInfo, sensitive)
		fieldMaskInfo = append(fieldMaskInfo, sensitive && queryContext.EnableSensitive)
	}

	columnTypes, err := rows.ColumnTypes()
	if err != nil {
		return nil, err
	}

	var columnTypeNames []string
	for _, v := range columnTypes {
		// DatabaseTypeName returns the database system name of the column type.
		// refer: https://pkg.go.dev/database/sql#ColumnType.DatabaseTypeName
		columnTypeNames = append(columnTypeNames, strings.ToUpper(v.DatabaseTypeName()))
	}

	data, err := readRows(rows, dbType, columnTypes, columnTypeNames, fieldMaskInfo)
	if err != nil {
		return nil, err
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return []any{columnNames, columnTypeNames, data, fieldMaskInfo, fieldSensitiveInfo}, nil
}

// Query2 will execute a readonly / SELECT query.
// TODO(rebelice): remove Query function and rename Query2 to Query after frontend is ready to use the new API.
func Query2(ctx context.Context, dbType db.Type, conn *sql.Conn, statement string, queryContext *db.QueryContext) (*v1pb.QueryResult, error) {
	tx, err := conn.BeginTx(ctx, &sql.TxOptions{ReadOnly: queryContext.ReadOnly})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	rows, err := tx.QueryContext(ctx, statement)
	if err != nil {
		return nil, FormatErrorWithQuery(err, statement)
	}
	defer rows.Close()

	columnNames, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	// TODO(d): use a Redshift extraction for shared database.
	if dbType == db.Redshift && queryContext.ShareDB {
		statement = strings.ReplaceAll(statement, fmt.Sprintf("%s.", queryContext.CurrentDatabase), "")
	}
	fieldList, err := extractSensitiveField(dbType, statement, queryContext.CurrentDatabase, queryContext.SensitiveSchemaInfo)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to extract sensitive fields: %q", statement)
	}
	if len(fieldList) != 0 && len(fieldList) != len(columnNames) {
		return nil, errors.Errorf("failed to extract sensitive fields: %q", statement)
	}

	var fieldMaskInfo []bool
	var fieldSensitiveInfo []bool
	for i := range columnNames {
		sensitive := len(fieldList) > 0 && fieldList[i].Sensitive
		fieldSensitiveInfo = append(fieldSensitiveInfo, sensitive)
		fieldMaskInfo = append(fieldMaskInfo, sensitive && queryContext.EnableSensitive)
	}

	columnTypes, err := rows.ColumnTypes()
	if err != nil {
		return nil, err
	}

	var columnTypeNames []string
	for _, v := range columnTypes {
		// DatabaseTypeName returns the database system name of the column type.
		// refer: https://pkg.go.dev/database/sql#ColumnType.DatabaseTypeName
		columnTypeNames = append(columnTypeNames, strings.ToUpper(v.DatabaseTypeName()))
	}

	data, err := readRows2(rows, columnTypes, columnTypeNames, fieldMaskInfo)
	if err != nil {
		return nil, err
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return &v1pb.QueryResult{
		ColumnNames:     columnNames,
		ColumnTypeNames: columnTypeNames,
		Rows:            data,
		Masked:          fieldMaskInfo,
		Sensitive:       fieldSensitiveInfo,
	}, nil
}

// RunStatement runs a SQL statement in a given connection.
func RunStatement(ctx context.Context, engineType parser.EngineType, conn *sql.Conn, statement string) ([]*v1pb.QueryResult, error) {
	singleSQLs, err := parser.SplitMultiSQL(engineType, statement)
	if err != nil {
		return nil, err
	}
	if len(singleSQLs) == 0 {
		return nil, nil
	}

	var results []*v1pb.QueryResult
	for _, singleSQL := range singleSQLs {
		startTime := time.Now()
		if singleSQL.Empty {
			continue
		}
		if parser.IsMySQLAffectedRowsStatement(singleSQL.Text) {
			sqlResult, err := conn.ExecContext(ctx, singleSQL.Text)
			if err != nil {
				return nil, err
			}
			affectedRows, err := sqlResult.RowsAffected()
			if err != nil {
				log.Info("rowsAffected returns error", zap.Error(err))
			}

			field := []string{"Affected Rows"}
			types := []string{"INT"}
			rows := []*v1pb.QueryRow{
				{
					Values: []*v1pb.RowValue{
						{
							Kind: &v1pb.RowValue_Int64Value{
								Int64Value: affectedRows,
							},
						},
					},
				},
			}
			results = append(results, &v1pb.QueryResult{
				ColumnNames:     field,
				ColumnTypeNames: types,
				Rows:            rows,
				Latency:         durationpb.New(time.Since(startTime)),
				Statement:       strings.TrimLeft(strings.TrimRight(singleSQL.Text, " \n\t;"), " \n\t"),
			})
			continue
		}
		results = append(results, adminQuery(ctx, conn, singleSQL.Text))
	}

	return results, nil
}

func adminQuery(ctx context.Context, conn *sql.Conn, statement string) *v1pb.QueryResult {
	startTime := time.Now()
	rows, err := conn.QueryContext(ctx, statement)
	if err != nil {
		return &v1pb.QueryResult{
			Error: err.Error(),
		}
	}
	defer rows.Close()

	result, err := rowsToQueryResult(rows)
	if err != nil {
		return &v1pb.QueryResult{
			Error: err.Error(),
		}
	}
	result.Latency = durationpb.New(time.Since(startTime))
	result.Statement = strings.TrimLeft(strings.TrimRight(statement, " \n\t;"), " \n\t")
	return result
}

func rowsToQueryResult(rows *sql.Rows) (*v1pb.QueryResult, error) {
	columnNames, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	columnTypes, err := rows.ColumnTypes()
	if err != nil {
		return nil, err
	}

	var columnTypeNames []string
	for _, v := range columnTypes {
		// DatabaseTypeName returns the database system name of the column type.
		// refer: https://pkg.go.dev/database/sql#ColumnType.DatabaseTypeName
		columnTypeNames = append(columnTypeNames, strings.ToUpper(v.DatabaseTypeName()))
	}

	data, err := readRows2(rows, columnTypes, columnTypeNames, nil)
	if err != nil {
		return nil, err
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return &v1pb.QueryResult{
		ColumnNames:     columnNames,
		ColumnTypeNames: columnTypeNames,
		Rows:            data,
	}, nil
}

// query will execute a query.
func queryAdmin(ctx context.Context, dbType db.Type, conn *sql.Conn, statement string, _ int) ([]any, error) {
	rows, err := conn.QueryContext(ctx, statement)
	if err != nil {
		// TODO(d): ClickHouse will return "driver: bad connection" if we use non-SELECT statement for Query(). We need to ignore the error.
		if dbType == db.ClickHouse {
			return nil, nil
		}
		return nil, FormatErrorWithQuery(err, statement)
	}
	defer rows.Close()

	columnNames, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	columnTypes, err := rows.ColumnTypes()
	if err != nil {
		return nil, err
	}

	var columnTypeNames []string
	for _, v := range columnTypes {
		// DatabaseTypeName returns the database system name of the column type.
		// refer: https://pkg.go.dev/database/sql#ColumnType.DatabaseTypeName
		columnTypeNames = append(columnTypeNames, strings.ToUpper(v.DatabaseTypeName()))
	}

	data, err := readRows(rows, dbType, columnTypes, columnTypeNames, nil)
	if err != nil {
		return nil, err
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	// queryAdmin doesn't mask the sensitive fields.
	// Return the all false boolean slice here as the placeholder.
	sensitiveInfo := make([]bool, len(columnNames))
	return []any{columnNames, columnTypeNames, data, sensitiveInfo}, nil
}

// TODO(rebelice): remove the readRows and rename readRows2 to readRows if legacy API is deprecated.
func readRows2(rows *sql.Rows, columnTypes []*sql.ColumnType, columnTypeNames []string, fieldMaskInfo []bool) ([]*v1pb.QueryRow, error) {
	var data []*v1pb.QueryRow
	if len(columnTypes) == 0 {
		// No rows.
		// The oracle driver will panic if there is no rows such as EXPLAIN PLAN FOR statement.
		return data, nil
	}
	for rows.Next() {
		scanArgs := make([]any, len(columnTypes))
		for i, v := range columnTypeNames {
			// TODO(steven need help): Consult a common list of data types from database driver documentation. e.g. MySQL,PostgreSQL.
			switch v {
			case "VARCHAR", "TEXT", "UUID", "TIMESTAMP":
				scanArgs[i] = new(sql.NullString)
			case "BOOL":
				scanArgs[i] = new(sql.NullBool)
			case "INT", "INTEGER":
				scanArgs[i] = new(sql.NullInt64)
			case "FLOAT":
				scanArgs[i] = new(sql.NullFloat64)
			default:
				scanArgs[i] = new(sql.NullString)
			}
		}

		if err := rows.Scan(scanArgs...); err != nil {
			return nil, err
		}

		var rowData v1pb.QueryRow
		for i := range columnTypes {
			if len(fieldMaskInfo) > 0 && fieldMaskInfo[i] {
				rowData.Values = append(rowData.Values, &v1pb.RowValue{Kind: &v1pb.RowValue_StringValue{StringValue: "******"}})
				continue
			}
			if v, ok := (scanArgs[i]).(*sql.NullBool); ok && v.Valid {
				rowData.Values = append(rowData.Values, &v1pb.RowValue{Kind: &v1pb.RowValue_BoolValue{BoolValue: v.Bool}})
				continue
			}
			if v, ok := (scanArgs[i]).(*sql.NullString); ok && v.Valid {
				rowData.Values = append(rowData.Values, &v1pb.RowValue{Kind: &v1pb.RowValue_StringValue{StringValue: v.String}})
				continue
			}
			if v, ok := (scanArgs[i]).(*sql.NullInt64); ok && v.Valid {
				rowData.Values = append(rowData.Values, &v1pb.RowValue{Kind: &v1pb.RowValue_Int64Value{Int64Value: v.Int64}})
				continue
			}
			if v, ok := (scanArgs[i]).(*sql.NullInt32); ok && v.Valid {
				rowData.Values = append(rowData.Values, &v1pb.RowValue{Kind: &v1pb.RowValue_Int32Value{Int32Value: v.Int32}})
				continue
			}
			if v, ok := (scanArgs[i]).(*sql.NullFloat64); ok && v.Valid {
				rowData.Values = append(rowData.Values, &v1pb.RowValue{Kind: &v1pb.RowValue_DoubleValue{DoubleValue: v.Float64}})
				continue
			}
			// If none of them match, set nil to its value.
			rowData.Values = append(rowData.Values, &v1pb.RowValue{Kind: &v1pb.RowValue_NullValue{NullValue: structpb.NullValue_NULL_VALUE}})
		}

		data = append(data, &rowData)
	}

	return data, nil
}

func readRows(rows *sql.Rows, dbType db.Type, columnTypes []*sql.ColumnType, columnTypeNames []string, fieldMaskInfo []bool) ([]any, error) {
	if dbType == db.ClickHouse {
		return readRowsForClickhouse(rows, columnTypes, columnTypeNames, fieldMaskInfo)
	}
	data := []any{}
	for rows.Next() {
		scanArgs := make([]any, len(columnTypes))
		for i, v := range columnTypeNames {
			// TODO(steven need help): Consult a common list of data types from database driver documentation. e.g. MySQL,PostgreSQL.
			switch v {
			case "VARCHAR", "TEXT", "UUID", "TIMESTAMP":
				scanArgs[i] = new(sql.NullString)
			case "BOOL":
				scanArgs[i] = new(sql.NullBool)
			case "INT", "INTEGER":
				scanArgs[i] = new(sql.NullInt64)
			case "FLOAT":
				scanArgs[i] = new(sql.NullFloat64)
			default:
				scanArgs[i] = new(sql.NullString)
			}
		}

		if err := rows.Scan(scanArgs...); err != nil {
			return nil, err
		}

		rowData := []any{}
		for i := range columnTypes {
			if len(fieldMaskInfo) > 0 && fieldMaskInfo[i] {
				rowData = append(rowData, "******")
				continue
			}
			if v, ok := (scanArgs[i]).(*sql.NullBool); ok && v.Valid {
				rowData = append(rowData, v.Bool)
				continue
			}
			if v, ok := (scanArgs[i]).(*sql.NullString); ok && v.Valid {
				rowData = append(rowData, v.String)
				continue
			}
			if v, ok := (scanArgs[i]).(*sql.NullInt64); ok && v.Valid {
				rowData = append(rowData, v.Int64)
				continue
			}
			if v, ok := (scanArgs[i]).(*sql.NullInt32); ok && v.Valid {
				rowData = append(rowData, v.Int32)
				continue
			}
			if v, ok := (scanArgs[i]).(*sql.NullFloat64); ok && v.Valid {
				rowData = append(rowData, v.Float64)
				continue
			}
			// If none of them match, set nil to its value.
			rowData = append(rowData, nil)
		}

		data = append(data, rowData)
	}

	return data, nil
}

func getStatementWithResultLimit(stmt string, limit int) string {
	return fmt.Sprintf("WITH result AS (%s) SELECT * FROM result LIMIT %d;", stmt, limit)
}

func getMySQLStatementWithResultLimit(stmt string, limit int) string {
	return fmt.Sprintf("SELECT * FROM (%s) result LIMIT %d;", stmt, limit)
}

func getOracleStatementWithResultLimit(stmt string, limit int) string {
	return fmt.Sprintf("SELECT * FROM (%s) WHERE ROWNUM <= %d", stmt, limit)
}

func getMSSQLStatementWithResultLimit(stmt string, limit int) string {
	// TODO(d): support SELECT 1 (mssql: No column name was specified for column 1 of 'result').
	return fmt.Sprintf("WITH result AS (%s) SELECT TOP %d * FROM result;", stmt, limit)
}

// FindMigrationHistoryList will find the list of migration history.
func FindMigrationHistoryList(ctx context.Context, findMigrationHistoryListQuery string, queryParams []any, sqldb *sql.DB) ([]*db.MigrationHistory, error) {
	// To support `pg` option, the util layer will not know which database where `migration_history` table is,
	// so we need to connect to the database provided by params.
	tx, err := sqldb.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	rows, err := tx.QueryContext(ctx, findMigrationHistoryListQuery, queryParams...)
	if err != nil {
		return nil, FormatErrorWithQuery(err, findMigrationHistoryListQuery)
	}
	defer rows.Close()

	// Iterate over result set and deserialize rows into migrationHistoryList.
	var migrationHistoryList []*db.MigrationHistory
	for rows.Next() {
		var history db.MigrationHistory
		var storedVersion string
		if err := rows.Scan(
			&history.ID,
			&history.Creator,
			&history.CreatedTs,
			&history.Updater,
			&history.UpdatedTs,
			&history.ReleaseVersion,
			&history.Namespace,
			&history.Sequence,
			&history.Source,
			&history.Type,
			&history.Status,
			&storedVersion,
			&history.Description,
			&history.Statement,
			&history.Schema,
			&history.SchemaPrev,
			&history.ExecutionDurationNs,
			&history.IssueID,
			&history.Payload,
		); err != nil {
			return nil, err
		}

		useSemanticVersion, version, semanticVersionSuffix, err := FromStoredVersion(storedVersion)
		if err != nil {
			return nil, err
		}
		history.UseSemanticVersion, history.Version, history.SemanticVersionSuffix = useSemanticVersion, version, semanticVersionSuffix
		migrationHistoryList = append(migrationHistoryList, &history)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return migrationHistoryList, nil
}

// NonSemanticPrefix is the prefix for non-semantic version.
const NonSemanticPrefix = "0000.0000.0000-"

// ToStoredVersion converts semantic or non-semantic version to stored version format.
// Non-semantic version will have additional "0000.0000.0000-" prefix.
// Semantic version will add zero padding to MAJOR, MINOR, PATCH version with a timestamp suffix.
func ToStoredVersion(useSemanticVersion bool, version, semanticVersionSuffix string) (string, error) {
	if !useSemanticVersion {
		return fmt.Sprintf("%s%s", NonSemanticPrefix, version), nil
	}
	v, err := semver.Make(version)
	if err != nil {
		return "", err
	}
	major, minor, patch := fmt.Sprintf("%d", v.Major), fmt.Sprintf("%d", v.Minor), fmt.Sprintf("%d", v.Patch)
	if len(major) > 4 || len(minor) > 4 || len(patch) > 4 {
		return "", errors.Errorf("invalid version %q, major, minor, patch version should be < 10000", version)
	}
	return fmt.Sprintf("%04s.%04s.%04s-%s", major, minor, patch, semanticVersionSuffix), nil
}

// FromStoredVersion converts stored version to semantic or non-semantic version.
func FromStoredVersion(storedVersion string) (bool, string, string, error) {
	if strings.HasPrefix(storedVersion, NonSemanticPrefix) {
		return false, strings.TrimPrefix(storedVersion, NonSemanticPrefix), "", nil
	}
	idx := strings.Index(storedVersion, "-")
	if idx < 0 {
		return false, "", "", errors.Errorf("invalid stored version %q, version should contain '-'", storedVersion)
	}
	prefix, suffix := storedVersion[:idx], storedVersion[idx+1:]
	parts := strings.Split(prefix, ".")
	if len(parts) != 3 {
		return false, "", "", errors.Errorf("invalid stored version %q, version prefix %q should be in semantic version format", storedVersion, prefix)
	}
	major, err := strconv.Atoi(parts[0])
	if err != nil {
		return false, "", "", errors.Errorf("invalid stored version %q, version prefix %q should be in semantic version format", storedVersion, prefix)
	}
	minor, err := strconv.Atoi(parts[1])
	if err != nil {
		return false, "", "", errors.Errorf("invalid stored version %q, version prefix %q should be in semantic version format", storedVersion, prefix)
	}
	patch, err := strconv.Atoi(parts[2])
	if err != nil {
		return false, "", "", errors.Errorf("invalid stored version %q, version prefix %q should be in semantic version format", storedVersion, prefix)
	}
	if major >= 10000 || minor >= 10000 || patch >= 10000 {
		return false, "", "", errors.Errorf("invalid stored version %q, major, minor, patch version of %q should be < 10000", storedVersion, prefix)
	}
	return true, fmt.Sprintf("%d.%d.%d", major, minor, patch), suffix, nil
}

// IsAffectedRowsStatement returns true if the statement will return the number of affected rows.
func IsAffectedRowsStatement(stmt string) bool {
	affectedRowsStatementPrefix := []string{"INSERT ", "UPDATE ", "DELETE "}
	upperStatement := strings.TrimLeft(strings.ToUpper(stmt), " \t\r\n")
	for _, prefix := range affectedRowsStatementPrefix {
		if strings.HasPrefix(upperStatement, prefix) {
			return true
		}
	}
	return false
}

// ConvertYesNo converts YES/NO to bool.
func ConvertYesNo(s string) (bool, error) {
	switch s {
	case "YES", "Y":
		return true, nil
	case "NO", "N":
		return false, nil
	default:
		return false, errors.Errorf("unrecognized isNullable type %q", s)
	}
}

func readRowsForClickhouse(rows *sql.Rows, columnTypes []*sql.ColumnType, columnTypeNames []string, fieldMaskInfo []bool) ([]any, error) {
	data := []any{}

	for rows.Next() {
		cols := make([]any, len(columnTypes))
		for i, name := range columnTypeNames {
			// The ClickHouse driver uses *Type rather than sql.NullType to scan nullable fields
			// as described in https://github.com/ClickHouse/clickhouse-go/issues/754
			// TODO: remove this workaround once fixed.
			if strings.HasPrefix(name, "TUPLE") || strings.HasPrefix(name, "ARRAY") || strings.HasPrefix(name, "MAP") {
				// For TUPLE, ARRAY, MAP type in ClickHouse, we pass any and the driver will do the rest.
				var it any
				cols[i] = &it
			} else {
				// We use ScanType to get the correct *Type and then do type assertions
				// following https://github.com/ClickHouse/clickhouse-go/blob/main/TYPES.md
				cols[i] = reflect.New(columnTypes[i].ScanType()).Interface()
			}
		}

		if err := rows.Scan(cols...); err != nil {
			return nil, err
		}

		rowData := []any{}
		for i := range cols {
			if len(fieldMaskInfo) > 0 && fieldMaskInfo[i] {
				rowData = append(rowData, "******")
				continue
			}

			// handle TUPLE ARRAY MAP
			if v, ok := cols[i].(*any); ok && v != nil {
				rowData = append(rowData, *v)
				continue
			}

			// not nullable
			if v, ok := cols[i].(*int); ok && v != nil {
				rowData = append(rowData, *v)
				continue
			}
			if v, ok := cols[i].(*int8); ok && v != nil {
				rowData = append(rowData, *v)
				continue
			}
			if v, ok := cols[i].(*int16); ok && v != nil {
				rowData = append(rowData, *v)
				continue
			}
			if v, ok := cols[i].(*int32); ok && v != nil {
				rowData = append(rowData, *v)
				continue
			}
			if v, ok := cols[i].(*int64); ok && v != nil {
				rowData = append(rowData, *v)
				continue
			}
			if v, ok := cols[i].(*uint); ok && v != nil {
				rowData = append(rowData, *v)
				continue
			}
			if v, ok := cols[i].(*uint8); ok && v != nil {
				rowData = append(rowData, *v)
				continue
			}
			if v, ok := cols[i].(*uint16); ok && v != nil {
				rowData = append(rowData, *v)
				continue
			}
			if v, ok := cols[i].(*uint32); ok && v != nil {
				rowData = append(rowData, *v)
				continue
			}
			if v, ok := cols[i].(*uint64); ok && v != nil {
				rowData = append(rowData, *v)
				continue
			}
			if v, ok := cols[i].(*float32); ok && v != nil {
				rowData = append(rowData, *v)
				continue
			}
			if v, ok := cols[i].(*float64); ok && v != nil {
				rowData = append(rowData, *v)
				continue
			}
			if v, ok := cols[i].(*string); ok && v != nil {
				rowData = append(rowData, *v)
				continue
			}
			if v, ok := cols[i].(*bool); ok && v != nil {
				rowData = append(rowData, *v)
				continue
			}
			if v, ok := cols[i].(*time.Time); ok && v != nil {
				rowData = append(rowData, *v)
				continue
			}
			if v, ok := cols[i].(*big.Int); ok && v != nil {
				rowData = append(rowData, v.String())
				continue
			}
			if v, ok := cols[i].(*decimal.Decimal); ok && v != nil {
				rowData = append(rowData, v.String())
				continue
			}
			if v, ok := cols[i].(*uuid.UUID); ok && v != nil {
				rowData = append(rowData, v.String())
				continue
			}
			if v, ok := cols[i].(*orb.Point); ok && v != nil {
				rowData = append(rowData, wkt.MarshalString(*v))
				continue
			}
			if v, ok := cols[i].(*orb.Polygon); ok && v != nil {
				rowData = append(rowData, wkt.MarshalString(*v))
				continue
			}
			if v, ok := cols[i].(*orb.Ring); ok && v != nil {
				rowData = append(rowData, wkt.MarshalString(*v))
				continue
			}
			if v, ok := cols[i].(*orb.MultiPolygon); ok && v != nil {
				rowData = append(rowData, wkt.MarshalString(*v))
				continue
			}

			// nullable
			if v, ok := cols[i].(**int); ok && *v != nil {
				rowData = append(rowData, **v)
				continue
			}
			if v, ok := cols[i].(**int8); ok && *v != nil {
				rowData = append(rowData, **v)
				continue
			}
			if v, ok := cols[i].(**int16); ok && *v != nil {
				rowData = append(rowData, **v)
				continue
			}
			if v, ok := cols[i].(**int32); ok && *v != nil {
				rowData = append(rowData, **v)
				continue
			}
			if v, ok := cols[i].(**int64); ok && *v != nil {
				rowData = append(rowData, **v)
				continue
			}
			if v, ok := cols[i].(**uint); ok && *v != nil {
				rowData = append(rowData, **v)
				continue
			}
			if v, ok := cols[i].(**uint8); ok && *v != nil {
				rowData = append(rowData, **v)
				continue
			}
			if v, ok := cols[i].(**uint16); ok && *v != nil {
				rowData = append(rowData, **v)
				continue
			}
			if v, ok := cols[i].(**uint32); ok && *v != nil {
				rowData = append(rowData, **v)
				continue
			}
			if v, ok := cols[i].(**uint64); ok && *v != nil {
				rowData = append(rowData, **v)
				continue
			}
			if v, ok := cols[i].(**float32); ok && *v != nil {
				rowData = append(rowData, **v)
				continue
			}
			if v, ok := cols[i].(**float64); ok && *v != nil {
				rowData = append(rowData, **v)
				continue
			}
			if v, ok := cols[i].(**string); ok && *v != nil {
				rowData = append(rowData, **v)
				continue
			}
			if v, ok := cols[i].(**bool); ok && *v != nil {
				rowData = append(rowData, **v)
				continue
			}
			if v, ok := cols[i].(**time.Time); ok && *v != nil {
				rowData = append(rowData, **v)
				continue
			}
			if v, ok := cols[i].(**big.Int); ok && *v != nil {
				rowData = append(rowData, (*v).String())
				continue
			}
			if v, ok := cols[i].(**decimal.Decimal); ok && *v != nil {
				rowData = append(rowData, (*v).String())
				continue
			}
			if v, ok := cols[i].(**uuid.UUID); ok && *v != nil {
				rowData = append(rowData, (*v).String())
				continue
			}
			rowData = append(rowData, nil)
		}

		data = append(data, rowData)
	}

	return data, nil
}
