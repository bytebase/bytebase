package trino

import (
	"database/sql"
	"strings"
	"time"

	"github.com/trinodb/trino-go-client/trino"
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
	// The driver hands a VARIANT column a trino.Variant, which the default
	// *sql.NullString below cannot hold, failing the whole query.
	case "VARIANT":
		return new(trino.Variant)
	// The driver decodes these to time.Time. Left to the default below, Go
	// renders them through *sql.NullString as RFC3339Nano, so a DATE came back
	// as 2026-01-02T00:00:00Z and a TIME as 0000-01-01T12:34:56Z.
	case "DATE", "TIME", "TIME WITH TIME ZONE":
		return new(sql.NullTime)
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
	case *sql.NullTime:
		if raw.Valid {
			return &v1pb.RowValue{
				Kind: &v1pb.RowValue_StringValue{
					StringValue: formatDateOrTime(typeName, raw.Time),
				},
			}
		}
	case *trino.Variant:
		// String is the JSON text the server returns for CAST(v AS JSON). A
		// VARIANT null is valid and renders as "null", unlike a SQL NULL.
		if raw.Valid {
			return &v1pb.RowValue{
				Kind: &v1pb.RowValue_StringValue{
					StringValue: raw.String(),
				},
			}
		}
	default:
	}
	return util.NullRowValue
}

// formatDateOrTime renders the time.Time the driver decoded a DATE or a TIME
// into. The fraction follows util.FormatTimeOnly, so a TIME(3) of 12:34:56.700
// reads 12:34:56.700000 as it does on PostgreSQL and SQL Server.
func formatDateOrTime(typeName string, t time.Time) string {
	switch typeName {
	case "DATE":
		return t.Format(time.DateOnly)
	case "TIME WITH TIME ZONE":
		return util.FormatTimeOnlyTZ(t)
	default:
		return util.FormatTimeOnly(t)
	}
}
