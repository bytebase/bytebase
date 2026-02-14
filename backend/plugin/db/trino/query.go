package trino

import (
	"database/sql"
	"strings"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
)

func makeValueByTypeName(typeName string, _ *sql.ColumnType) any {
	switch typeName {
	case "VARCHAR", "TEXT", "CHAR", "TIMESTAMP", "TIMESTAMP WITH TIME ZONE", "TIMESTAMP WITHOUT TIME ZONE":
		return new(sql.NullString)
	case "BOOLEAN":
		return new(sql.NullBool)
	case "TINYINT", "SMALLINT", "INTEGER", "BIGINT":
		return new(sql.NullInt64)
	case "DECIMAL", "REAL", "DOUBLE":
		return new(sql.NullFloat64)
	case "VARBINARY", "BINARY":
		return new([]byte)
	default:
		return new(sql.NullString)
	}
}

func convertValue(typeName string, _ *sql.ColumnType, value any) *v1pb.RowValue {
	switch raw := value.(type) {
	case *sql.NullString:
		if raw.Valid {
			if strings.Contains(typeName, "TIMESTAMP") {
				t, err := time.Parse(time.RFC3339, raw.String)
				if err == nil {
					return &v1pb.RowValue{
						Kind: &v1pb.RowValue_TimestampValue{
							TimestampValue: &v1pb.RowValue_Timestamp{
								GoogleTimestamp: timestamppb.New(t),
								Accuracy:        6,
							},
						},
					}
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
	default:
	}
	return util.NullRowValue
}
