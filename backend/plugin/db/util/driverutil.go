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
		runningStatement := singleSQL.Text
		if engineType == storepb.Engine_MYSQL {
			runningStatement = MySQLPrependBytebaseAppComment(singleSQL.Text)
		}
		if mysqlparser.IsMySQLAffectedRowsStatement(singleSQL.Text) {
			sqlResult, err := conn.ExecContext(ctx, runningStatement)
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
				Statement:       singleSQL.Text,
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

	result, err := RowsToQueryResult(dbType, rows)
	if err != nil {
		return &v1pb.QueryResult{
			Error: err.Error(),
		}
	}
	result.Latency = durationpb.New(time.Since(startTime))
	result.Statement = statement
	return result
}

func RowsToQueryResult(dbType storepb.Engine, rows *sql.Rows) (*v1pb.QueryResult, error) {
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
				scanArgs[i], wantBytesValue[i] = mssqlMakeScanDestByTypeName(v)
				continue
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

// TODO(zp): This function should be moved back to the driver-specific package, put it here
// now to avoid cyclic dependency problem.
func mssqlMakeScanDestByTypeName(tn string) (dest any, wantByte bool) {
	switch strings.ToUpper(tn) {
	case "TINYINT":
		return new(sql.NullInt64), false
	case "SMALLINT":
		return new(sql.NullInt64), false
	case "INT":
		return new(sql.NullInt64), false
	case "BIGINT":
		return new(sql.NullInt64), false
	case "REAL":
		return new(sql.NullFloat64), false
	case "FLOAT":
		return new(sql.NullFloat64), false
	case "VARBINARY":
		// TODO(zp): Null bytes?
		return new(sql.NullString), true
	case "VARCHAR":
		return new(sql.NullString), false
	case "NVARCHAR":
		return new(sql.NullString), false
	case "BIT":
		return new(sql.NullBool), false
	case "DECIMAL":
		return new(sql.NullString), false
	case "SMALLMONEY":
		return new(sql.NullString), false
	case "MONEY":
		return new(sql.NullString), false

	// TODO(zp): Scan to string now, switch to use time.Time while masking support it.
	// // Source values of type [time.Time] may be scanned into values of type
	// *time.Time, *interface{}, *string, or *[]byte. When converting to
	// the latter two, [time.RFC3339Nano] is used.
	case "SMALLDATETIME":
		return new(sql.NullString), false
	case "DATETIME":
		return new(sql.NullString), false
	case "DATETIME2":
		return new(sql.NullString), false
	case "DATE":
		return new(sql.NullString), false
	case "TIME":
		return new(sql.NullString), false
	case "DATETIMEOFFSET":
		return new(sql.NullString), false

	case "CHAR":
		return new(sql.NullString), false
	case "NCHAR":
		return new(sql.NullString), false
	case "UNIQUEIDENTIFIER":
		return new(mssqldb.NullUniqueIdentifier), false
	case "XML":
		return new(sql.NullString), false
	case "TEXT":
		return new(sql.NullString), false
	case "NTEXT":
		return new(sql.NullString), false
	case "IMAGE":
		return new(sql.NullString), true
	case "BINARY":
		return new(sql.NullString), true
	case "SQL_VARIANT":
		return new(sql.NullString), true
	}
	return new(sql.NullString), true
}

// TrimStatement trims the unused characters from the statement for making getStatementWithResultLimit() happy.
func TrimStatement(statement string) string {
	return strings.TrimLeft(strings.TrimRight(statement, " \n\t;"), " \n\t")
}

func MySQLPrependBytebaseAppComment(statement string) string {
	return fmt.Sprintf("/*app=bytebase*/ %s", statement)
}
