package redshift

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/jackc/pgx/v5/pgtype"

	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
)

func makeValueByTypeName(typeName string, _ *sql.ColumnType) any {
	switch typeName {
	case "VARCHAR", "TEXT", "UUID":
		return new(sql.NullString)
	case "BOOL":
		return new(sql.NullBool)
	case "INT", "INTEGER", "TINYINT", "SMALLINT", "MEDIUMINT", "BIGINT", "INT2", "INT4", "INT8":
		return new(sql.NullInt64)
	case "FLOAT", "DOUBLE", "FLOAT4", "FLOAT8":
		return new(sql.NullFloat64)
	case "DATE":
		return new(pgtype.Date)
	case "TIMESTAMP", "TIMESTAMPTZ":
		return new(pgtype.Timestamptz)
	case "BIT", "VARBIT":
		return new([]byte)
	default:
		return new(sql.NullString)
	}
}

var timeTzOID = fmt.Sprintf("%d", pgtype.TimetzOID)

func convertValue(typeName string, columnType *sql.ColumnType, value any) *v1pb.RowValue {
	switch raw := value.(type) {
	case *sql.NullString:
		if raw.Valid {
			// TODO: Fix DatabaseTypeName for 1266, Object ID for TIME WITHOUT TIME ZONE
			if columnType.DatabaseTypeName() == "TIME" || columnType.DatabaseTypeName() == timeTzOID || columnType.DatabaseTypeName() == "INTERVAL" {
				return &v1pb.RowValue{
					Kind: &v1pb.RowValue_StringValue{
						StringValue: padZeroes(raw.String, 6),
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
	case *pgtype.Date:
		if raw.Valid {
			if raw.InfinityModifier != pgtype.Finite {
				return &v1pb.RowValue{
					Kind: &v1pb.RowValue_StringValue{
						StringValue: raw.InfinityModifier.String(),
					},
				}
			}
			if typeName == "DATE" {
				return &v1pb.RowValue{
					Kind: &v1pb.RowValue_StringValue{
						StringValue: raw.Time.Format(time.DateOnly),
					},
				}
			}
		}
	case *pgtype.Timestamptz:
		if raw.Valid {
			if raw.InfinityModifier != pgtype.Finite {
				return &v1pb.RowValue{
					Kind: &v1pb.RowValue_StringValue{
						StringValue: raw.InfinityModifier.String(),
					},
				}
			}
			_, scale, _ := columnType.DecimalSize()
			if scale == -1 {
				scale = 6
			}
			if typeName == "TIMESTAMP" {
				return &v1pb.RowValue{
					Kind: &v1pb.RowValue_TimestampValue{
						TimestampValue: &v1pb.RowValue_Timestamp{
							GoogleTimestamp: timestamppb.New(raw.Time),
							Accuracy:        int32(scale),
						},
					},
				}
			}
			zone, offset := raw.Time.Zone()
			return &v1pb.RowValue{
				Kind: &v1pb.RowValue_TimestampTzValue{
					TimestampTzValue: &v1pb.RowValue_TimestampTZ{
						GoogleTimestamp: timestamppb.New(raw.Time),
						Zone:            zone,
						Offset:          int32(offset),
						Accuracy:        int32(scale),
					},
				},
			}
		}
	default:
	}
	return util.NullRowValue
}

// Padding 0's to nanosecond precision to make sure it's always 6 digits.
// Since the data cannot be formatted into a time.Time, we need to pad it here.
// Accuracy is passed as argument since we cannot determine the precision of the data type using DecimalSize().
func padZeroes(rawStr string, acc int) string {
	dotIndex := strings.Index(rawStr, ".")
	if dotIndex < 0 {
		return rawStr
	}
	// End index is used to cut off the time zone information.
	endIndex := len(rawStr)
	if plusIndex := strings.Index(rawStr, "+"); plusIndex >= 0 {
		endIndex = plusIndex
	} else if minusIndex := strings.Index(rawStr, "-"); minusIndex >= 0 {
		endIndex = minusIndex
	}
	decimalPart := rawStr[dotIndex+1 : endIndex]
	if len(decimalPart) < acc {
		rawStr = rawStr[:endIndex] + strings.Repeat("0", acc-len(decimalPart)) + rawStr[endIndex:]
	}
	return rawStr
}
