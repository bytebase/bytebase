package pg

import (
	"database/sql"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/antlr4-go/antlr/v4"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/types/known/timestamppb"

	parser "github.com/bytebase/parser/postgresql"
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

	parseResults, err := pgparser.ParsePostgreSQL(statement)
	if err != nil {
		return "", errors.Wrap(err, "failed to parse statement")
	}

	if len(parseResults) != 1 {
		return "", errors.Errorf("expected exactly one statement, got %d", len(parseResults))
	}

	parseResult := parseResults[0]

	listener := &postgresqlRewriter{
		limitCount:     limitCount,
		outerMostQuery: true,
		rewriter:       antlr.NewTokenStreamRewriter(parseResult.Tokens),
	}

	antlr.ParseTreeWalkerDefault.Walk(listener, parseResult.Tree)
	if listener.err != nil {
		return "", errors.Wrapf(listener.err, "statement: %s", statement)
	}

	return listener.rewriter.GetTextDefault(), nil
}

type postgresqlRewriter struct {
	*parser.BasePostgreSQLParserListener

	rewriter       *antlr.TokenStreamRewriter
	err            error
	outerMostQuery bool
	limitCount     int
}

// EnterSelectstmt is called when entering a select statement
func (r *postgresqlRewriter) EnterSelectstmt(ctx *parser.SelectstmtContext) {
	if !r.outerMostQuery {
		return
	}
	r.outerMostQuery = false

	// Recursively find the select_no_parens and handle it
	if selectNoParens := r.findSelectNoParens(ctx); selectNoParens != nil {
		r.handleSelectNoParens(selectNoParens)
	}
}

// findSelectNoParens recursively finds the select_no_parens from a selectstmt
func (r *postgresqlRewriter) findSelectNoParens(ctx *parser.SelectstmtContext) parser.ISelect_no_parensContext {
	// Direct select_no_parens
	if selectNoParens := ctx.Select_no_parens(); selectNoParens != nil {
		return selectNoParens
	}

	// Through select_with_parens
	if selectWithParens := ctx.Select_with_parens(); selectWithParens != nil {
		return r.findSelectNoParensFromWithParens(selectWithParens)
	}

	return nil
}

// findSelectNoParensFromWithParens recursively finds select_no_parens from select_with_parens
func (r *postgresqlRewriter) findSelectNoParensFromWithParens(ctx parser.ISelect_with_parensContext) parser.ISelect_no_parensContext {
	// Check for direct select_no_parens
	if selectNoParens := ctx.Select_no_parens(); selectNoParens != nil {
		return selectNoParens
	}

	// Check for nested select_with_parens
	if innerSelectWithParens := ctx.Select_with_parens(); innerSelectWithParens != nil {
		return r.findSelectNoParensFromWithParens(innerSelectWithParens)
	}

	return nil
}

// EnterStmt is called when entering any statement
func (r *postgresqlRewriter) EnterStmt(ctx *parser.StmtContext) {
	// Only process SELECT statements
	if ctx.Selectstmt() == nil {
		// Not a SELECT statement, skip processing
		r.outerMostQuery = false
	}
}

// handleSelectNoParens processes select statements without parentheses
func (r *postgresqlRewriter) handleSelectNoParens(ctx parser.ISelect_no_parensContext) {
	// Check if there's already a limit clause
	var hasLimit bool
	var limitClause parser.ILimit_clauseContext

	f := ctx.GetText()
	slog.Debug("Processing select_no_parens", slog.String("text", f))

	// Check the for_locking_clause with opt_select_limit branch
	if ctx.For_locking_clause() != nil && ctx.Opt_select_limit() != nil {
		if ctx.Opt_select_limit().Select_limit() != nil {
			if ctx.Opt_select_limit().Select_limit().Limit_clause() != nil {
				hasLimit = true
				limitClause = ctx.Opt_select_limit().Select_limit().Limit_clause()
			}
		}
	}
	// Check the select_limit with opt_for_locking_clause branch
	if ctx.Select_limit() != nil {
		if ctx.Select_limit().Limit_clause() != nil {
			hasLimit = true
			limitClause = ctx.Select_limit().Limit_clause()
		}
	}

	if hasLimit && limitClause != nil {
		// Extract and compare the existing limit value
		if limitClause.Select_limit_value() != nil {
			limitValueText := limitClause.Select_limit_value().GetText()
			if limitValueText != "ALL" {
				existingLimit, err := strconv.Atoi(limitValueText)
				if err == nil {
					if existingLimit == 0 || r.limitCount < existingLimit {
						// Replace the existing limit value
						limitValueCtx := limitClause.Select_limit_value()
						if limitValueCtx.A_expr() != nil {
							r.rewriter.ReplaceTokenDefault(
								limitValueCtx.GetStart(),
								limitValueCtx.GetStop(),
								fmt.Sprintf("%d", r.limitCount),
							)
						}
					}
					// else: existing limit is already lower, keep it
				}
			}
		}
		return
	}

	// No limit clause exists, add one
	// Find the appropriate position to insert LIMIT
	var insertPosition antlr.Token

	// Insert after FOR UPDATE/SHARE clause if present
	if ctx.For_locking_clause() != nil {
		insertPosition = ctx.For_locking_clause().GetStop()
	}
	// Insert after ORDER BY clause if present
	if ctx.Opt_sort_clause() != nil && ctx.Opt_sort_clause().Sort_clause() != nil {
		insertPosition = ctx.Opt_sort_clause().Sort_clause().GetStop()
	}
	// Otherwise insert after the select_clause
	if insertPosition == nil && ctx.Select_clause() != nil {
		insertPosition = ctx.Select_clause().GetStop()
	}

	if insertPosition != nil {
		r.rewriter.InsertAfterToken("default", insertPosition, fmt.Sprintf(" LIMIT %d", r.limitCount))
	}
}
