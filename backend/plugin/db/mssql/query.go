package mssql

import (
	"database/sql"
	"fmt"
	"log/slog"
	"time"

	omnimssql "github.com/bytebase/omni/mssql"
	"google.golang.org/protobuf/types/known/timestamppb"

	mssqldb "github.com/microsoft/go-mssqldb"

	"github.com/bytebase/bytebase/backend/common/log"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
)

func makeValueByTypeName(typeName string, _ *sql.ColumnType) any {
	switch typeName {
	case "UNIQUEIDENTIFIER":
		return new(mssqldb.NullUniqueIdentifier)
	case "TINYINT", "SMALLINT", "INT", "BIGINT":
		return new(sql.NullInt64)
	case "VARCHAR", "NVARCHAR", "CHAR", "NCHAR", "TEXT", "NTEXT", "XML":
		return new(sql.NullString)
	case "REAL", "FLOAT":
		return new(sql.NullFloat64)
	case "VARBINARY":
		return new([]byte)
	// BIT type must use sql.NullBool. All SQL Editors show 0/1 for BIT type instead of true/false.
	// So we have to do extra conversion from bool to int.
	case "BIT":
		return new(sql.NullBool)
	case "DECIMAL":
		return new(sql.NullString)
	case "SMALLMONEY":
		return new(sql.NullString)
	case "MONEY":
		return new(sql.NullString)
	// TODO(zp): Scan to string now, switch to use time.Time while masking support it.
	// // Source values of type [time.Time] may be scanned into values of type
	// *time.Time, *interface{}, *string, or *[]byte. When converting to
	// the latter two, [time.RFC3339Nano] is used.
	case "TIME", "DATE", "SMALLDATETIME", "DATETIME", "DATETIME2":
		return new(sql.NullTime)
	case "DATETIMEOFFSET":
		return new(sql.NullTime)
	case "IMAGE":
		return new([]byte)
	case "BINARY":
		return new([]byte)
	case "SQL_VARIANT":
		return new([]byte)
	case "GEOMETRY", "GEOGRAPHY":
		return new([]byte)
	default:
		// For unknown types, default to sql.NullString which can handle most values
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
			var v int64
			if raw.Bool {
				v = 1
			}
			return &v1pb.RowValue{
				Kind: &v1pb.RowValue_Int64Value{
					Int64Value: v,
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
	case *mssqldb.NullUniqueIdentifier:
		if raw.Valid {
			return &v1pb.RowValue{
				Kind: &v1pb.RowValue_StringValue{
					StringValue: raw.UUID.String(),
				},
			}
		}
	case *sql.NullTime:
		if raw.Valid {
			if columnType.DatabaseTypeName() == "TIME" {
				return &v1pb.RowValue{
					Kind: &v1pb.RowValue_StringValue{
						StringValue: raw.Time.Format(time.TimeOnly),
					},
				}
			}
			if columnType.DatabaseTypeName() == "DATE" {
				return &v1pb.RowValue{
					Kind: &v1pb.RowValue_StringValue{
						StringValue: raw.Time.Format(time.DateOnly),
					},
				}
			}
			_, scale, _ := columnType.DecimalSize()
			if typeName == "DATETIME" || typeName == "DATETIME2" || typeName == "SMALLDATETIME" {
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

func getStatementWithResultLimit(statement string, limit int) string {
	stmt, err := getStatementWithResultLimitInline(statement, limit)
	if err != nil {
		slog.Error("fail to add limit clause", slog.String("statement", statement), log.BBError(err))
		return fmt.Sprintf("WITH result AS (%s) SELECT TOP %d * FROM result;", util.TrimStatement(statement), limit)
	}
	return stmt
}

func getStatementWithResultLimitInline(singleStatement string, limitCount int) (string, error) {
	return omnimssql.StatementWithResultLimit(singleStatement, limitCount)
}
