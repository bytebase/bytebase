// Package util provides the library for database drivers.
package util

import (
	"bufio"
	"context"
	"database/sql"
	"fmt"
	"io"
	"log/slog"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	mysqlparser "github.com/bytebase/bytebase/backend/plugin/parser/mysql"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
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
func Query(ctx context.Context, dbType storepb.Engine, conn *sql.Conn, statement string, queryContext *db.QueryContext) (*v1pb.QueryResult, error) {
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
	if dbType == storepb.Engine_REDSHIFT && queryContext.ShareDB {
		statement = strings.ReplaceAll(statement, fmt.Sprintf("%s.", queryContext.CurrentDatabase), "")
	}
	fieldList, err := base.ExtractSensitiveField(dbType, statement, queryContext.CurrentDatabase, queryContext.SensitiveSchemaInfo)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to extract sensitive fields: %q", statement)
	}
	if len(fieldList) != 0 && len(fieldList) != len(columnNames) {
		return nil, errors.Errorf("failed to extract sensitive fields: %q", statement)
	}

	var fieldMaskingLevels []storepb.MaskingLevel
	var fieldMaskInfo []bool
	var fieldSensitiveInfo []bool
	for i := range columnNames {
		maskingLevel := storepb.MaskingLevel_NONE
		if len(fieldList) > i && queryContext.EnableSensitive {
			maskingLevel = fieldList[i].MaskingAttributes.MaskingLevel
		}
		fieldMaskingLevels = append(fieldMaskingLevels, maskingLevel)
		sensitive := len(fieldList) > i && (maskingLevel == storepb.MaskingLevel_FULL || maskingLevel == storepb.MaskingLevel_PARTIAL)
		fieldMaskInfo = append(fieldMaskInfo, sensitive && queryContext.EnableSensitive)
		fieldSensitiveInfo = append(fieldSensitiveInfo, sensitive)
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

	data, err := readRows(rows, columnTypeNames, fieldMaskingLevels)
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
func RunStatement(ctx context.Context, engineType storepb.Engine, conn *sql.Conn, statement string) ([]*v1pb.QueryResult, error) {
	singleSQLs, err := base.SplitMultiSQL(engineType, statement)
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
		if mysqlparser.IsMySQLAffectedRowsStatement(singleSQL.Text) {
			sqlResult, err := conn.ExecContext(ctx, singleSQL.Text)
			if err != nil {
				return nil, err
			}
			affectedRows, err := sqlResult.RowsAffected()
			if err != nil {
				slog.Info("rowsAffected returns error", log.BBError(err))
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

	data, err := readRows(rows, columnTypeNames, nil)
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

func readRows(rows *sql.Rows, columnTypeNames []string, fieldMaskingLevels []storepb.MaskingLevel) ([]*v1pb.QueryRow, error) {
	var data []*v1pb.QueryRow
	if len(columnTypeNames) == 0 {
		// No rows.
		// The oracle driver will panic if there is no rows such as EXPLAIN PLAN FOR statement.
		return data, nil
	}
	for rows.Next() {
		// wantBytesValue want to convert StringValue to BytesValue when columnTypeName is BIT or VARBIT
		wantBytesValue := make([]bool, len(columnTypeNames))
		scanArgs := make([]any, len(columnTypeNames))
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
			case "BIT", "VARBIT":
				wantBytesValue[i] = true
				scanArgs[i] = new(sql.NullString)
			default:
				scanArgs[i] = new(sql.NullString)
			}
		}

		if err := rows.Scan(scanArgs...); err != nil {
			return nil, err
		}

		var rowData v1pb.QueryRow
		for i := range columnTypeNames {
			if len(fieldMaskingLevels) > i && fieldMaskingLevels[i] == storepb.MaskingLevel_FULL {
				rowData.Values = append(rowData.Values, &v1pb.RowValue{Kind: &v1pb.RowValue_StringValue{StringValue: "******"}})
				continue
			}
			if v, ok := (scanArgs[i]).(*sql.NullBool); ok && v.Valid {
				if len(fieldMaskingLevels) > i && fieldMaskingLevels[i] == storepb.MaskingLevel_PARTIAL {
					s := fmt.Sprintf("%t", v.Bool)
					result := getMiddlePartOfString(s)
					rowData.Values = append(rowData.Values, &v1pb.RowValue{Kind: &v1pb.RowValue_StringValue{StringValue: paddingAsterisk(result)}})
				} else {
					rowData.Values = append(rowData.Values, &v1pb.RowValue{Kind: &v1pb.RowValue_BoolValue{BoolValue: v.Bool}})
				}
				continue
			}
			if v, ok := (scanArgs[i]).(*sql.NullString); ok && v.Valid {
				if wantBytesValue[i] {
					if len(fieldMaskingLevels) > i && fieldMaskingLevels[i] == storepb.MaskingLevel_PARTIAL {
						result := getMiddlePartOfString(v.String)
						rowData.Values = append(rowData.Values, &v1pb.RowValue{Kind: &v1pb.RowValue_StringValue{StringValue: paddingAsterisk(result)}})
					} else {
						rowData.Values = append(rowData.Values, &v1pb.RowValue{Kind: &v1pb.RowValue_BytesValue{BytesValue: []byte(v.String)}})
					}
				} else {
					if len(fieldMaskingLevels) > i && fieldMaskingLevels[i] == storepb.MaskingLevel_PARTIAL {
						result := getMiddlePartOfString(v.String)
						rowData.Values = append(rowData.Values, &v1pb.RowValue{Kind: &v1pb.RowValue_StringValue{StringValue: paddingAsterisk(result)}})
					} else {
						rowData.Values = append(rowData.Values, &v1pb.RowValue{Kind: &v1pb.RowValue_StringValue{StringValue: v.String}})
					}
				}
				continue
			}
			if v, ok := (scanArgs[i]).(*sql.NullInt64); ok && v.Valid {
				if len(fieldMaskingLevels) > i && fieldMaskingLevels[i] == storepb.MaskingLevel_PARTIAL {
					s := strconv.FormatInt(v.Int64, 10)
					result := getMiddlePartOfString(s)
					rowData.Values = append(rowData.Values, &v1pb.RowValue{Kind: &v1pb.RowValue_StringValue{StringValue: paddingAsterisk(result)}})
				} else {
					rowData.Values = append(rowData.Values, &v1pb.RowValue{Kind: &v1pb.RowValue_Int64Value{Int64Value: v.Int64}})
				}
				continue
			}
			if v, ok := (scanArgs[i]).(*sql.NullInt32); ok && v.Valid {
				if len(fieldMaskingLevels) > i && fieldMaskingLevels[i] == storepb.MaskingLevel_PARTIAL {
					s := strconv.FormatInt(int64(v.Int32), 10)
					result := getMiddlePartOfString(s)
					rowData.Values = append(rowData.Values, &v1pb.RowValue{Kind: &v1pb.RowValue_StringValue{StringValue: paddingAsterisk(result)}})
				} else {
					rowData.Values = append(rowData.Values, &v1pb.RowValue{Kind: &v1pb.RowValue_Int32Value{Int32Value: v.Int32}})
				}
				continue
			}
			if v, ok := (scanArgs[i]).(*sql.NullFloat64); ok && v.Valid {
				if len(fieldMaskingLevels) > i && fieldMaskingLevels[i] == storepb.MaskingLevel_PARTIAL {
					s := strconv.FormatFloat(v.Float64, 'f', -1, 64)
					result := getMiddlePartOfString(s)
					rowData.Values = append(rowData.Values, &v1pb.RowValue{Kind: &v1pb.RowValue_StringValue{StringValue: paddingAsterisk(result)}})
				} else {
					rowData.Values = append(rowData.Values, &v1pb.RowValue{Kind: &v1pb.RowValue_DoubleValue{DoubleValue: v.Float64}})
				}
				continue
			}
			// If none of them match, set nil to its value.
			if len(fieldMaskingLevels) > i && fieldMaskingLevels[i] == storepb.MaskingLevel_PARTIAL {
				rowData.Values = append(rowData.Values, &v1pb.RowValue{Kind: &v1pb.RowValue_StringValue{StringValue: "**UL**"}})
			} else {
				rowData.Values = append(rowData.Values, &v1pb.RowValue{Kind: &v1pb.RowValue_NullValue{NullValue: structpb.NullValue_NULL_VALUE}})
			}
		}

		data = append(data, &rowData)
	}

	return data, nil
}

func paddingAsterisk(s string) string {
	return fmt.Sprintf("**%s**", s)
}

// getMiddlePartOfString will get the middle part of the string.
func getMiddlePartOfString(stmt string) string {
	if len(stmt) == 0 || len(stmt) == 1 {
		return ""
	}
	if len(stmt) == 2 || len(stmt) == 3 {
		return string(stmt[1])
	}

	s := []rune(stmt)
	if len(s)%4 != 0 {
		s = s[:len(s)/4*4]
	}

	var ret []rune
	ret = append(ret, s[len(s)/4:len(s)/2]...)
	ret = append(ret, s[len(s)/2:len(s)/4*3]...)
	return string(ret)
}

func getStatementWithResultLimit(stmt string, limit int) string {
	return fmt.Sprintf("WITH result AS (%s) SELECT * FROM result LIMIT %d;", stmt, limit)
}

func getMySQLStatementWithResultLimit(stmt string, limit int) string {
	return fmt.Sprintf("SELECT * FROM (%s) result LIMIT %d;", stmt, limit)
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
