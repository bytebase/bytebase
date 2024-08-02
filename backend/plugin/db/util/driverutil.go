// Package util provides the library for database drivers.
package util

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/pkg/errors"
	"google.golang.org/protobuf/proto"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/component/masker"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
)

var noneMasker = masker.NewNoneMasker()

// FormatErrorWithQuery will format the error with failed query.
func FormatErrorWithQuery(err error, query string) error {
	return errors.Wrapf(err, "failed to execute query %q", query)
}

func BuildAffectedRowsResult(affectedRows int64) *v1pb.QueryResult {
	return &v1pb.QueryResult{
		ColumnNames:     []string{"Affected Rows"},
		ColumnTypeNames: []string{"INT"},
		Rows: []*v1pb.QueryRow{
			{
				Values: []*v1pb.RowValue{
					{
						Kind: &v1pb.RowValue_Int64Value{
							Int64Value: affectedRows,
						},
					},
				},
			},
		},
	}
}

func RowsToQueryResult(rows *sql.Rows) (*v1pb.QueryResult, error) {
	columnNames, err := rows.Columns()
	if err != nil {
		return nil, err
	}
	columnTypes, err := rows.ColumnTypes()
	if err != nil {
		return nil, err
	}
	// DatabaseTypeName returns the database system name of the column type.
	// refer: https://pkg.go.dev/database/sql#ColumnType.DatabaseTypeName
	var columnTypeNames []string
	for _, v := range columnTypes {
		columnTypeNames = append(columnTypeNames, strings.ToUpper(v.DatabaseTypeName()))
	}
	result := &v1pb.QueryResult{
		ColumnNames:     columnNames,
		ColumnTypeNames: columnTypeNames,
	}

	if len(columnTypeNames) > 0 {
		for rows.Next() {
			// isByteValues want to convert StringValue to BytesValue when columnTypeName is BIT or VARBIT
			isByteValues := make([]bool, len(columnTypeNames))
			values := make([]any, len(columnTypeNames))
			for i, v := range columnTypeNames {
				// TODO(steven need help): Consult a common list of data types from database driver documentation. e.g. MySQL,PostgreSQL.
				switch v {
				case "VARCHAR", "TEXT", "UUID", "TIMESTAMP":
					values[i] = new(sql.NullString)
				case "BOOL":
					values[i] = new(sql.NullBool)
				case "INT", "INTEGER":
					values[i] = new(sql.NullInt64)
				case "FLOAT", "DOUBLE":
					values[i] = new(sql.NullFloat64)
				case "BIT", "VARBIT":
					isByteValues[i] = true
					values[i] = new(sql.NullString)
				default:
					values[i] = new(sql.NullString)
				}
			}

			if err := rows.Scan(values...); err != nil {
				return nil, err
			}

			var rowData v1pb.QueryRow
			for i := range columnTypeNames {
				rowData.Values = append(rowData.Values, noneMasker.Mask(&masker.MaskData{
					Data:      values[i],
					WantBytes: isByteValues[i],
				}))
			}

			result.Rows = append(result.Rows, &rowData)
			n := len(result.Rows)
			if (n&(n-1) == 0) && proto.Size(result) > common.MaximumSQLResultSize {
				result.Error = common.MaximumSQLResultSizeExceeded
				break
			}
		}
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return result, nil
}

// TrimStatement trims the unused characters from the statement for making getStatementWithResultLimit() happy.
func TrimStatement(statement string) string {
	return strings.TrimLeft(strings.TrimRight(statement, " \n\t;"), " \n\t")
}

func MySQLPrependBytebaseAppComment(statement string) string {
	return fmt.Sprintf("/*app=bytebase*/ %s", statement)
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
