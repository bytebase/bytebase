package oracle

import (
	"database/sql"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
)

// ========== Type Conversion Functions ==========

// makeValueByTypeName creates appropriate Go types for Oracle column types.
// DATE: date.
// TIMESTAMPDTY: timestamp.
// TIMESTAMPTZ_DTY: timestamp with time zone.
// TIMESTAMPLTZ_DTY: timezone with local time zone.
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
	case "BIT", "VARBIT":
		return new([]byte)
	case "DATE", "TIMESTAMPDTY", "TIMESTAMPLTZ_DTY", "TIMESTAMPTZ_DTY":
		return new(sql.NullTime)
	default:
		return new(sql.NullString)
	}
}

func convertValue(typeName string, columnType *sql.ColumnType, value any) *v1pb.RowValue {
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
	case *sql.NullTime:
		if raw.Valid {
			return convertTimestamp(typeName, columnType, raw.Time)
		}
	default:
	}
	return util.NullRowValue
}

// convertTimestamp handles Oracle timestamp type conversions.
// The go-ora driver retrieves the database timezone from the wire protocol and appends it to the timestamp.
// To ensure consistency with Oracle Date expectations, we handle different timestamp types appropriately.
// https://github.com/sijms/go-ora/blob/2962e725e7a756a667a546fb360ef09afd4c8bd0/v2/parameter.go#L616
func convertTimestamp(typeName string, columnType *sql.ColumnType, t time.Time) *v1pb.RowValue {
	_, scale, _ := columnType.DecimalSize()

	switch typeName {
	case "DATE", "TIMESTAMPDTY":
		// Strip timezone information for DATE and TIMESTAMP types
		timeStripped := time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), time.UTC)
		return &v1pb.RowValue{
			Kind: &v1pb.RowValue_TimestampValue{
				TimestampValue: &v1pb.RowValue_Timestamp{
					GoogleTimestamp: timestamppb.New(timeStripped),
					Accuracy:        int32(scale),
				},
			},
		}

	case "TIMESTAMPLTZ_DTY":
		// Handle local timezone timestamp
		// This timestamp is not consistent with sqlplus likely due to db and session timezone.
		// TODO(d): fix the go-ora library.
		s := t.Format("2006-01-02 15:04:05.000000000")
		parsedTime, err := time.Parse(time.DateTime, s)
		if err != nil {
			return util.NullRowValue
		}
		return &v1pb.RowValue{
			Kind: &v1pb.RowValue_TimestampValue{
				TimestampValue: &v1pb.RowValue_Timestamp{
					GoogleTimestamp: timestamppb.New(parsedTime),
					Accuracy:        int32(scale),
				},
			},
		}

	default:
		// Handle timestamp with timezone
		zone, offset := t.Zone()
		return &v1pb.RowValue{
			Kind: &v1pb.RowValue_TimestampTzValue{
				TimestampTzValue: &v1pb.RowValue_TimestampTZ{
					GoogleTimestamp: timestamppb.New(t),
					Zone:            zone,
					Offset:          int32(offset),
					Accuracy:        int32(scale),
				},
			},
		}
	}
}
