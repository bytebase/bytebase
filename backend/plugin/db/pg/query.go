package pg

import (
	"database/sql"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/pkg/errors"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/bytebase/omni/pg/ast"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/bytebase/bytebase/backend/common/log"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
	pgparser "github.com/bytebase/bytebase/backend/plugin/parser/pg"
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
	case "BIT", "VARBIT", "BYTEA":
		return new([]byte)
	case "GEOMETRY", "GEOGRAPHY":
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
		// For negative intervals like "-00:04:37.530865", ignore the leading minus
		// Only consider minus signs that appear after the decimal point (timezone indicators)
		if minusIndex > dotIndex {
			endIndex = minusIndex
		}
	}

	// Validate slice bounds to prevent runtime panic
	if endIndex <= dotIndex {
		return rawStr
	}

	decimalPart := rawStr[dotIndex+1 : endIndex]
	if len(decimalPart) < acc {
		rawStr = rawStr[:endIndex] + strings.Repeat("0", acc-len(decimalPart)) + rawStr[endIndex:]
	}
	return rawStr
}

// getStatementWithResultLimit returns the statement with LIMIT clause if not exists.
func getStatementWithResultLimit(statement string, limit int) string {
	stmt, err := getStatementWithResultLimitInline(statement, limit)
	if err != nil {
		slog.Error("fail to add limit clause", slog.String("statement", statement), log.BBError(err))
		// Fallback to CTE approach for problematic queries
		return fmt.Sprintf("WITH result AS (\n%s\n) SELECT * FROM result LIMIT %d;", util.TrimStatement(statement), limit)
	}
	return stmt
}

func getStatementWithResultLimitInline(statement string, limitCount int) (string, error) {
	if strings.TrimSpace(statement) == "" {
		return "", errors.New("empty statement")
	}

	stmts, err := pgparser.ParsePg(statement)
	if err != nil {
		return "", errors.Wrap(err, "failed to parse statement")
	}

	if len(stmts) != 1 {
		return "", errors.Errorf("expected exactly one statement, got %d", len(stmts))
	}

	sel, ok := stmts[0].AST.(*ast.SelectStmt)
	if !ok {
		// Non-SELECT statement, return as-is.
		return statement, nil
	}

	return rewriteSelectLimit(statement, sel, limitCount)
}

// rewriteSelectLimit adds or adjusts the LIMIT clause of a SELECT statement
// using byte-offset positions from the omni AST to surgically edit the original SQL text.
func rewriteSelectLimit(sql string, sel *ast.SelectStmt, limitCount int) (string, error) {
	if sel.LimitCount != nil {
		// Already has LIMIT — replace the value if ours is lower.
		existingLimit := extractIntFromNode(sel.LimitCount)
		if existingLimit > 0 && existingLimit <= limitCount {
			return sql, nil // existing limit is already lower or equal, keep it
		}
		loc := nodeLocOf(sel.LimitCount)
		if loc.Start >= 0 && loc.End > loc.Start && loc.End <= len(sql) {
			return sql[:loc.Start] + fmt.Sprintf("%d", limitCount) + sql[loc.End:], nil
		}
		// LimitCount is a non-constant expression (e.g. LIMIT $1, LIMIT (1+2)).
		// Cannot safely rewrite in-place; let the caller fall back to CTE wrapper.
		return "", errors.Errorf("cannot rewrite non-constant LIMIT expression")
	}

	// No LIMIT clause — find the right insertion point.
	// PostgreSQL grammar order: ... ORDER BY ... LIMIT ... FOR UPDATE ...
	// LIMIT goes BEFORE FOR UPDATE but AFTER everything else.
	insertPos, beforeLocking := findLimitInsertPosition(sel)
	if beforeLocking {
		// Inserting at the start of FOR UPDATE/SHARE. The original whitespace
		// before FOR becomes the separator before LIMIT; we add a trailing
		// space to separate the limit value from FOR.
		return sql[:insertPos] + fmt.Sprintf("LIMIT %d ", limitCount) + sql[insertPos:], nil
	}
	return sql[:insertPos] + fmt.Sprintf(" LIMIT %d", limitCount) + sql[insertPos:], nil
}

// findLimitInsertPosition returns the byte offset where " LIMIT N" should be inserted,
// and whether the insertion is before a locking clause (FOR UPDATE/SHARE).
func findLimitInsertPosition(sel *ast.SelectStmt) (int, bool) {
	// LIMIT must appear before FOR UPDATE/SHARE.
	if sel.LockingClause != nil {
		span := ast.ListSpan(sel.LockingClause)
		if span.Start > 0 {
			return span.Start, true
		}
	}

	// Otherwise insert at SelectStmt.Loc.End (after everything, including outer parens).
	end := sel.Loc.End
	if end <= 0 {
		return 0, false
	}
	return end, false
}

// extractIntFromNode extracts an integer value from a LIMIT/OFFSET node.
func extractIntFromNode(node ast.Node) int {
	switch n := node.(type) {
	case *ast.Integer:
		return int(n.Ival)
	case *ast.A_Const:
		if iv, ok := n.Val.(*ast.Integer); ok {
			return int(iv.Ival)
		}
		return 0
	default:
		return 0
	}
}

// nodeLocOf returns the Loc of a node, handling common wrapper types.
func nodeLocOf(node ast.Node) ast.Loc {
	switch n := node.(type) {
	case *ast.A_Const:
		return n.Loc
	default:
		return ast.Loc{Start: -1, End: -1}
	}
}
