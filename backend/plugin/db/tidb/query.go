package tidb

import (
	"database/sql"

	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
)

func makeValueByTypeName(typeName string, _ *sql.ColumnType) any {
	switch typeName {
	case "VARCHAR", "TEXT", "UUID", "DATETIME", "TIMESTAMP":
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

func convertValue(typeName string, columnType *sql.ColumnType, value any) *v1pb.RowValue {
	switch raw := value.(type) {
	case *sql.NullString:
		if raw.Valid {
			_, scale, _ := columnType.DecimalSize()
			if typeName == "TIMESTAMP" || typeName == "DATETIME" {
				return util.BuildTimestampOrStringRowValue(raw.String, scale)
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
	default:
	}
	return util.NullRowValue
}
