// Package util provides the library for database drivers.
package util

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/pkg/errors"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/durationpb"

	mssqldb "github.com/microsoft/go-mssqldb"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/component/masker"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	mysqlparser "github.com/bytebase/bytebase/backend/plugin/parser/mysql"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
)

var noneMasker = masker.NewNoneMasker()

// FormatErrorWithQuery will format the error with failed query.
func FormatErrorWithQuery(err error, query string) error {
	return errors.Wrapf(err, "failed to execute query %q", query)
}

// Query will execute a readonly / SELECT query.
func Query(ctx context.Context, dbType storepb.Engine, conn *sql.Conn, statement string, queryContext *db.QueryContext) (*v1pb.QueryResult, error) {
	// TODO(d): use a Redshift extraction for shared database.
	if dbType == storepb.Engine_REDSHIFT && queryContext != nil && queryContext.ShareDB {
		statement = strings.ReplaceAll(statement, fmt.Sprintf("%s.", queryContext.CurrentDatabase), "")
	}

	startTime := time.Now()
	readOnly := false
	if queryContext != nil {
		readOnly = queryContext.ReadOnly
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

	result, err := rowsToQueryResult(dbType, rows)
	if err != nil {
		// nolint
		return &v1pb.QueryResult{
			Error: err.Error(),
		}, nil
	}
	result.Latency = durationpb.New(time.Since(startTime))
	result.Statement = strings.TrimLeft(strings.TrimRight(statement, " \n\t;"), " \n\t")
	return result, nil
}

// RunStatement runs a SQL statement in a given connection.
func RunStatement(ctx context.Context, engineType storepb.Engine, conn *sql.Conn, statement string) ([]*v1pb.QueryResult, error) {
	singleSQLs, err := base.SplitMultiSQL(engineType, statement)
	if err != nil {
		return nil, err
	}
	singleSQLs = base.FilterEmptySQL(singleSQLs)
	if len(singleSQLs) == 0 {
		return nil, nil
	}

	var results []*v1pb.QueryResult
	for _, singleSQL := range singleSQLs {
		startTime := time.Now()
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
		results = append(results, adminQuery(ctx, engineType, conn, singleSQL.Text))
	}

	return results, nil
}

func adminQuery(ctx context.Context, dbType storepb.Engine, conn *sql.Conn, statement string) *v1pb.QueryResult {
	startTime := time.Now()
	rows, err := conn.QueryContext(ctx, statement)
	if err != nil {
		return &v1pb.QueryResult{
			Error: err.Error(),
		}
	}
	defer rows.Close()

	result, err := rowsToQueryResult(dbType, rows)
	if err != nil {
		return &v1pb.QueryResult{
			Error: err.Error(),
		}
	}
	result.Latency = durationpb.New(time.Since(startTime))
	result.Statement = strings.TrimLeft(strings.TrimRight(statement, " \n\t;"), " \n\t")
	return result
}

func rowsToQueryResult(dbType storepb.Engine, rows *sql.Rows) (*v1pb.QueryResult, error) {
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

	result := &v1pb.QueryResult{
		ColumnNames:     columnNames,
		ColumnTypeNames: columnTypeNames,
	}
	if err := readRows(result, dbType, rows, columnTypeNames); err != nil {
		return nil, err
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return result, nil
}

func readRows(result *v1pb.QueryResult, dbType storepb.Engine, rows *sql.Rows, columnTypeNames []string) error {
	if len(columnTypeNames) == 0 {
		// No rows.
		// The oracle driver will panic if there is no rows such as EXPLAIN PLAN FOR statement.
		return nil
	}
	for rows.Next() {
		// wantBytesValue want to convert StringValue to BytesValue when columnTypeName is BIT or VARBIT
		wantBytesValue := make([]bool, len(columnTypeNames))
		scanArgs := make([]any, len(columnTypeNames))
		for i, v := range columnTypeNames {
			// TODO(steven need help): Consult a common list of data types from database driver documentation. e.g. MySQL,PostgreSQL.
			if dbType == storepb.Engine_MSSQL {
				switch v {
				case "UNIQUEIDENTIFIER":
					scanArgs[i] = new(mssqldb.UniqueIdentifier)
					continue
				case "NULLUNIQUEIDENTIFIER":
					scanArgs[i] = new(mssqldb.NullUniqueIdentifier)
					continue
				case "GEOMETRY":
					scanArgs[i] = new(sql.NullString)
					wantBytesValue[i] = true
					continue
				}
			}
			switch v {
			case "VARCHAR", "TEXT", "UUID", "TIMESTAMP":
				scanArgs[i] = new(sql.NullString)
			case "BOOL":
				scanArgs[i] = new(sql.NullBool)
			case "INT", "INTEGER":
				scanArgs[i] = new(sql.NullInt64)
			case "FLOAT", "DOUBLE":
				scanArgs[i] = new(sql.NullFloat64)
			case "BIT", "VARBIT":
				wantBytesValue[i] = true
				scanArgs[i] = new(sql.NullString)
			default:
				scanArgs[i] = new(sql.NullString)
			}
		}

		if err := rows.Scan(scanArgs...); err != nil {
			return err
		}

		var rowData v1pb.QueryRow
		for i := range columnTypeNames {
			rowData.Values = append(rowData.Values, noneMasker.Mask(&masker.MaskData{
				Data:      scanArgs[i],
				WantBytes: wantBytesValue[i],
			}))
		}

		result.Rows = append(result.Rows, &rowData)
		n := len(result.Rows)
		if (n&(n-1) == 0) && proto.Size(result) > common.MaximumSQLResultSize {
			result.Error = common.MaximumSQLResultSizeExceeded
			return nil
		}
	}

	return nil
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
	// ClickHouse uses 0 and 1.
	switch s {
	case "YES", "Y", "1":
		return true, nil
	case "NO", "N", "0":
		return false, nil
	default:
		return false, errors.Errorf("unrecognized isNullable type %q", s)
	}
}
