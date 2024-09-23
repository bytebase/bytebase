package pg

import (
	"database/sql"
	"strings"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/bytebase/bytebase/backend/common"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
)

var nullRowValue = &v1pb.RowValue{
	Kind: &v1pb.RowValue_NullValue{
		NullValue: structpb.NullValue_NULL_VALUE,
	},
}

func rowsToQueryResult(rows *sql.Rows, limit int64) (*v1pb.QueryResult, error) {
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
				case *[]byte:
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
				case *sql.NullTime:
					if raw.Valid {
						rowValue = &v1pb.RowValue{
							Kind: &v1pb.RowValue_TimestampValue{
								TimestampValue: timestamppb.New(raw.Time),
							},
						}
					}
				}

				row.Values = append(row.Values, rowValue)
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

	return result, nil
}

func makeValueByTypeName(typeName string) any {
	switch typeName {
	case "VARCHAR", "TEXT", "UUID", "DATE":
		return new(sql.NullString)
	case "TIMESTAMP", "TIME":
		return new(sql.NullTime)
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
