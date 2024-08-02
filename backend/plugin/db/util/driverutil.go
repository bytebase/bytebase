// Package util provides the library for database drivers.
package util

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/pkg/errors"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/bytebase/bytebase/backend/common"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
)

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

var nullRowValue = &v1pb.RowValue{
	Kind: &v1pb.RowValue_NullValue{
		NullValue: structpb.NullValue_NULL_VALUE,
	},
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
	columnLength := len(columnNames)

	if columnLength > 0 {
		for rows.Next() {
			values := make([]any, columnLength)
			for i, v := range columnTypeNames {
				values[i] = makeValueByTypeName(v)
			}

			if err := rows.Scan(values...); err != nil {
				return nil, err
			}

			row := &v1pb.QueryRow{}
			for i := 0; i < columnLength; i++ {
				rowValue := nullRowValue
				switch raw := values[i].(type) {
				case *sql.NullString:
					if raw.Valid {
						rowValue = &v1pb.RowValue{
							Kind: &v1pb.RowValue_StringValue{
								StringValue: raw.String,
							},
						}
					}
				case *sql.NullInt64:
					if raw.Valid {
						rowValue = &v1pb.RowValue{
							Kind: &v1pb.RowValue_Int64Value{
								Int64Value: raw.Int64,
							},
						}
					}
				case *sql.RawBytes:
					if len(*raw) > 0 {
						rowValue = &v1pb.RowValue{
							Kind: &v1pb.RowValue_BytesValue{
								BytesValue: *raw,
							},
						}
					}
				case *sql.NullBool:
					if raw.Valid {
						rowValue = &v1pb.RowValue{
							Kind: &v1pb.RowValue_BoolValue{
								BoolValue: raw.Bool,
							},
						}
					}
				case *sql.NullFloat64:
					if raw.Valid {
						rowValue = &v1pb.RowValue{
							Kind: &v1pb.RowValue_DoubleValue{
								DoubleValue: raw.Float64,
							},
						}
					}
				}

				row.Values = append(row.Values, rowValue)
			}

			result.Rows = append(result.Rows, row)
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

func makeValueByTypeName(typeName string) any {
	switch typeName {
	case "VARCHAR", "TEXT", "UUID", "TIMESTAMP":
		return new(sql.NullString)
	case "BOOL":
		return new(sql.NullBool)
	case "INT", "INTEGER":
		return new(sql.NullInt64)
	case "FLOAT", "DOUBLE":
		return new(sql.NullFloat64)
	case "BIT", "VARBIT":
		return new(sql.RawBytes)
	default:
		return new(sql.NullString)
	}
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
