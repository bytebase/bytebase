package tidb

import (
	"database/sql"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/pkg/errors"
	"google.golang.org/protobuf/types/known/timestamppb"

	tidbast "github.com/pingcap/tidb/pkg/parser/ast"
	"github.com/pingcap/tidb/pkg/parser/format"
	tidbdriver "github.com/pingcap/tidb/pkg/types/parser_driver"

	"github.com/bytebase/bytebase/backend/common/log"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
	tidbparser "github.com/bytebase/bytebase/backend/plugin/parser/tidb"
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
				t, err := time.Parse(time.DateTime, raw.String)
				if err != nil {
					slog.Error("failed to parse time value", log.BBError(err))
					return util.NullRowValue
				}
				return &v1pb.RowValue{
					Kind: &v1pb.RowValue_TimestampValue{
						TimestampValue: &v1pb.RowValue_Timestamp{
							GoogleTimestamp: timestamppb.New(t),
							Accuracy:        int32(scale),
						},
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
	default:
	}
	return util.NullRowValue
}

func getStatementWithResultLimit(statement string, limit int) string {
	stmt, err := getStatementWithResultLimitInline(statement, limit)
	if err != nil {
		slog.Error("fail to add limit clause", slog.String("statement", statement), log.BBError(err))
		return fmt.Sprintf("WITH result AS (%s) SELECT * FROM result LIMIT %d;", util.TrimStatement(statement), limit)
	}
	return stmt
}

func getStatementWithResultLimitInline(statement string, limit int) (string, error) {
	stmtList, err := tidbparser.ParseTiDB(statement, "", "")
	if err != nil {
		return "", errors.Wrapf(err, "failed to parse tidb statement: %s", statement)
	}
	if len(stmtList) != 1 {
		return "", errors.Errorf("expect one single statement in the query, %s", statement)
	}
	restoreFlags := format.DefaultRestoreFlags | format.RestoreStringWithoutDefaultCharset
	stmt := stmtList[0]
	switch stmt := stmt.(type) {
	case *tidbast.SelectStmt:
		if stmt.Limit != nil && stmt.Limit.Count != nil {
			if v, ok := stmt.Limit.Count.(*tidbdriver.ValueExpr); ok {
				userLimit := int(v.GetInt64())
				if limit < userLimit {
					userLimit = limit
				}
				stmt.Limit.Count = tidbast.NewValueExpr(int64(userLimit), "", "")
			}
		} else {
			stmt.Limit = &tidbast.Limit{
				Count: tidbast.NewValueExpr(int64(limit), "", ""),
			}
		}
		var buffer strings.Builder
		ctx := format.NewRestoreCtx(restoreFlags, &buffer)
		if err := stmt.Restore(ctx); err != nil {
			return "", err
		}
		return buffer.String(), nil
	case *tidbast.SetOprStmt:
		if stmt.Limit != nil && stmt.Limit.Count != nil {
			if v, ok := stmt.Limit.Count.(*tidbdriver.ValueExpr); ok {
				userLimit := int(v.GetInt64())
				if limit < userLimit {
					userLimit = limit
				}
				stmt.Limit.Count = tidbast.NewValueExpr(int64(userLimit), "", "")
			}
		} else {
			stmt.Limit = &tidbast.Limit{
				Count: tidbast.NewValueExpr(int64(limit), "", ""),
			}
		}
		var buffer strings.Builder
		ctx := format.NewRestoreCtx(restoreFlags, &buffer)
		if err := stmt.Restore(ctx); err != nil {
			return "", err
		}
		return buffer.String(), nil
	default:
	}
	return statement, nil
}
