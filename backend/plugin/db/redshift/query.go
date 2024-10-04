package redshift

import (
	"database/sql"
	"strings"

	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/bytebase/bytebase/backend/plugin/db/util"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
)

func makeValueByTypeName(typeName string, _ *sql.ColumnType) any {
	switch typeName {
	case "VARCHAR", "TEXT", "UUID", "TIMESTAMP":
		return new(sql.NullString)
	case "BOOL":
		return new(sql.NullBool)
	case "INT", "INTEGER", "TINYINT", "SMALLINT", "MEDIUMINT", "BIGINT", "INT2", "INT4", "INT8":
		return new(sql.NullInt64)
	case "FLOAT", "DOUBLE", "FLOAT4", "FLOAT8":
		return new(sql.NullFloat64)
	case "TIMESTAMPTZ":
		return new(sql.NullTime)
	case "BIT", "VARBIT":
		return new([]byte)
	default:
		return new(sql.NullString)
	}
}

func convertValue(typeName string, value any) *v1pb.RowValue {
	switch raw := value.(type) {
	case *sql.NullString:
		if raw.Valid {
			if typeName == "TIMESTAMP" {
				// Convert "2024-09-24T03:23:28.921281Z" to "2024-09-24 03:23:28.921281"
				timestampWithoutTimezone := strings.ReplaceAll(strings.ReplaceAll(raw.String, "T", " "), "Z", "")
				return &v1pb.RowValue{
					Kind: &v1pb.RowValue_StringValue{
						StringValue: timestampWithoutTimezone,
					},
				}
			}
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
	case *sql.NullTime:
		if raw.Valid {
			return &v1pb.RowValue{
				Kind: &v1pb.RowValue_TimestampTzValue{
					TimestampTzValue: &v1pb.RowValue_TimestampTZ{
						Timestamp: timestamppb.New(raw.Time),
					},
				},
			}
		}
	}
	return util.NullRowValue
}
