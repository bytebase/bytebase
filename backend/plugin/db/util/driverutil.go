// Package util provides the library for database drivers.
//
//nolint:revive
package util

import (
	"database/sql"
	"fmt"
	"strings"
	"unicode"

	"github.com/pkg/errors"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/bytebase/bytebase/backend/common"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/utils"
)

// FormatErrorWithQuery will format the error with failed query.
func FormatErrorWithQuery(err error, query string) error {
	return errors.Wrapf(err, "failed to execute query %q", query)
}

func BuildAffectedRowsResult(affectedRows int64, messages []*v1pb.QueryResult_Message) *v1pb.QueryResult {
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
		Messages: messages,
	}
}

var NullRowValue = &v1pb.RowValue{
	Kind: &v1pb.RowValue_NullValue{
		NullValue: structpb.NullValue_NULL_VALUE,
	},
}

func RowsToQueryResult(rows *sql.Rows, valueMaker func(string, *sql.ColumnType) any, rowValueConverter func(string, *sql.ColumnType, any) *v1pb.RowValue, limit int64) (*v1pb.QueryResult, error) {
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
				values[i] = valueMaker(v, columnTypes[i])
			}

			if err := rows.Scan(values...); err != nil {
				return nil, err
			}

			row := &v1pb.QueryRow{}
			for i := range columnLength {
				row.Values = append(row.Values, rowValueConverter(columnTypeNames[i], columnTypes[i], values[i]))
			}

			result.Rows = append(result.Rows, row)
			n := len(result.Rows)
			if (n&(n-1) == 0) && int64(proto.Size(result)) > limit {
				result.Error = common.FormatMaximumSQLResultSizeMessage(limit)
				break
			}
		}
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}
	result.RowsCount = int64(len(result.Rows))
	return result, nil
}

func MakeCommonValueByTypeName(typeName string, _ *sql.ColumnType) any {
	switch typeName {
	case "VARCHAR", "TEXT", "UUID", "TIMESTAMP":
		return new(sql.NullString)
	case "BOOL":
		return new(sql.NullBool)
	case "INT", "INTEGER", "TINYINT", "SMALLINT", "MEDIUMINT", "BIGINT", "INT2", "INT4", "INT8":
		return new(sql.NullInt64)
	case "FLOAT", "DOUBLE", "FLOAT4", "FLOAT8":
		return new(sql.NullFloat64)
	case "BIT", "VARBIT":
		return new([]byte)
	default:
		return new(sql.NullString)
	}
}

func ConvertCommonValue(_ string, _ *sql.ColumnType, value any) *v1pb.RowValue {
	switch raw := value.(type) {
	case *sql.NullString:
		if raw.Valid {
			return &v1pb.RowValue{
				Kind: &v1pb.RowValue_StringValue{
					StringValue: raw.String,
				},
			}
		}
	case *sql.NullInt64:
		if raw.Valid {
			return &v1pb.RowValue{
				Kind: &v1pb.RowValue_Int64Value{
					Int64Value: raw.Int64,
				},
			}
		}
	case *[]byte:
		if len(*raw) > 0 {
			return &v1pb.RowValue{
				Kind: &v1pb.RowValue_BytesValue{
					BytesValue: *raw,
				},
			}
		}
	case *sql.NullBool:
		if raw.Valid {
			return &v1pb.RowValue{
				Kind: &v1pb.RowValue_BoolValue{
					BoolValue: raw.Bool,
				},
			}
		}
	case *sql.NullFloat64:
		if raw.Valid {
			return &v1pb.RowValue{
				Kind: &v1pb.RowValue_DoubleValue{
					DoubleValue: raw.Float64,
				},
			}
		}
	}
	return NullRowValue
}

// TrimStatement trims the unused characters from the statement for making getStatementWithResultLimit() happy.
func TrimStatement(statement string) string {
	return strings.TrimLeftFunc(strings.TrimRightFunc(statement, utils.IsSpaceOrSemicolon), unicode.IsSpace)
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

func GetColumnIndex(columns []string, name string) (int, bool) {
	for i, c := range columns {
		if c == name {
			return i, true
		}
	}
	return 0, false
}
