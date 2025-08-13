package oracle

import (
	"database/sql"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/antlr4-go/antlr/v4"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/types/known/timestamppb"

	plsql "github.com/bytebase/plsql-parser"

	"github.com/bytebase/bytebase/backend/common/log"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
	plsqlparser "github.com/bytebase/bytebase/backend/plugin/parser/plsql"
	"github.com/bytebase/bytebase/backend/utils"
)

const dbVersion12 = 12

func makeValueByTypeName(typeName string, _ *sql.ColumnType) any {
	// DATE: date.
	// TIMESTAMPDTY: timestamp.
	// TIMESTAMPTZ_DTY: timestamp with time zone.
	// TIMESTAMPLTZ_DTY: timezone with local time zone.

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
			// The go-ora driver retrieves the database timezone from the wire protocol and appends it to the timestamp.
			// To ensure consistency with Oracle Date expectations, we should remove the timezone information.
			// https://github.com/sijms/go-ora/blob/2962e725e7a756a667a546fb360ef09afd4c8bd0/v2/parameter.go#L616
			_, scale, _ := columnType.DecimalSize()
			if typeName == "DATE" || typeName == "TIMESTAMPDTY" {
				timeStripped := time.Date(raw.Time.Year(), raw.Time.Month(), raw.Time.Day(), raw.Time.Hour(), raw.Time.Minute(), raw.Time.Second(), raw.Time.Nanosecond(), time.UTC)
				return &v1pb.RowValue{
					Kind: &v1pb.RowValue_TimestampValue{
						TimestampValue: &v1pb.RowValue_Timestamp{
							GoogleTimestamp: timestamppb.New(timeStripped),
							Accuracy:        int32(scale),
						},
					},
				}
			}
			if typeName == "TIMESTAMPLTZ_DTY" {
				s := raw.Time.Format("2006-01-02 15:04:05.000000000")
				t, err := time.Parse(time.DateTime, s)
				if err != nil {
					return util.NullRowValue
				}
				// This timestamp is not consistent with sqlplus likely due to db and session timezone.
				// TODO(d): fix the go-ora library.
				return &v1pb.RowValue{
					Kind: &v1pb.RowValue_TimestampValue{
						TimestampValue: &v1pb.RowValue_Timestamp{
							GoogleTimestamp: timestamppb.New(t),
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

// addLimitFor11g adds a ROWNUM-based limit for Oracle 11g and earlier versions.
// Uses the legacy approach with subquery and ROWNUM.
func addLimitFor11g(statement string, limitCount int) string {
	if !isSelectOrWithStatement(statement) {
		return statement
	}
	return fmt.Sprintf("SELECT * FROM (%s) WHERE ROWNUM <= %d", util.TrimStatement(statement), limitCount)
}

// isSelectOrWithStatement checks if the statement is a SELECT or WITH statement
func isSelectOrWithStatement(statement string) bool {
	trimmedStatement := strings.ToLower(strings.TrimLeftFunc(statement, unicode.IsSpace))
	return strings.HasPrefix(trimmedStatement, "select") || strings.HasPrefix(trimmedStatement, "with")
}

// addLimitFor12cAndLater adds a FETCH NEXT clause for Oracle 12c and later versions.
// Uses the modern SQL standard approach, falling back to 11g approach on error.
func addLimitFor12cAndLater(statement string, limit int) string {
	if !isSelectOrWithStatement(statement) {
		return statement
	}

	stmt, err := addFetchNextClause(statement, limit)
	if err != nil {
		slog.Error("failed to add FETCH NEXT clause, falling back to ROWNUM",
			slog.String("statement", statement), log.BBError(err))
		return addLimitFor11g(statement, limit)
	}
	return stmt
}

// addFetchNextClause adds a FETCH NEXT clause to a SELECT statement using AST parsing.
// This provides more precise placement of the limit clause compared to simple string wrapping.
func addFetchNextClause(statement string, limitCount int) (string, error) {
	tree, stream, err := plsqlparser.ParsePLSQL(statement)
	if err != nil {
		return "", err
	}

	listener := &plsqlRewriter{
		limitCount:        limitCount,
		selectFetch:       false,
		outerMostSubQuery: true,
	}

	listener.rewriter = *antlr.NewTokenStreamRewriter(stream)
	antlr.ParseTreeWalkerDefault.Walk(listener, tree)
	if listener.err != nil {
		return "", errors.Wrapf(listener.err, "statement: %s", statement)
	}

	res := listener.rewriter.GetTextDefault()
	// https://stackoverflow.com/questions/27987882/how-can-i-solve-ora-00911-invalid-character-error
	res = strings.TrimRightFunc(res, utils.IsSpaceOrSemicolon)

	return res, nil
}

// addResultLimit adds a limit clause to the statement based on Oracle version
func addResultLimit(stmt string, limit int, engineVersion string) string {
	// Check if we should skip adding limit (e.g., for simple DUAL queries)
	if shouldSkipLimit(stmt) {
		return stmt
	}

	// Determine Oracle version
	if isOracle11gOrEarlier(engineVersion) {
		return addLimitFor11g(stmt, limit)
	}
	return addLimitFor12cAndLater(stmt, limit)
}

// isOracle11gOrEarlier checks if the Oracle version is 11g or earlier
func isOracle11gOrEarlier(engineVersion string) bool {
	versionIdx := strings.Index(engineVersion, ".")
	if versionIdx < 0 {
		return true // Default to 11g behavior for invalid version
	}
	versionNumber, err := strconv.Atoi(engineVersion[:versionIdx])
	if err != nil {
		return true // Default to 11g behavior for parsing errors
	}
	return versionNumber < dbVersion12
}

// shouldSkipLimit checks if the statement needs a limit clause
func shouldSkipLimit(stmt string) bool {
	ok, err := skipAddLimit(stmt)
	return err == nil && ok
}

type plsqlRewriter struct {
	*plsql.BasePlSqlParserListener

	rewriter antlr.TokenStreamRewriter
	err      error
	// fetch in select_statement
	selectFetch bool
	// fetch in subquery
	outerMostSubQuery bool
	limitCount        int
}

func (r *plsqlRewriter) EnterSelect_statement(ctx *plsql.Select_statementContext) {
	if ctx.AllFetch_clause() != nil && len(ctx.AllFetch_clause()) > 0 {
		r.selectFetch = true
		return
	}
}

func (r *plsqlRewriter) EnterSubquery(ctx *plsql.SubqueryContext) {
	if !r.outerMostSubQuery || r.selectFetch {
		return
	}
	r.outerMostSubQuery = false
	// union | intersect | minus
	if ctx.AllSubquery_operation_part() != nil && len(ctx.AllSubquery_operation_part()) > 0 {
		lastPart := ctx.Subquery_operation_part(len(ctx.AllSubquery_operation_part()) - 1)
		if lastPart.Subquery_basic_elements().Query_block().Fetch_clause() != nil {
			r.overrideFetchClause(lastPart.Subquery_basic_elements().Query_block().Fetch_clause())
			return
		}
		if subqueryOp, ok := lastPart.(*plsql.Subquery_operation_partContext); ok {
			r.rewriter.InsertAfterDefault(subqueryOp.GetStop().GetTokenIndex(), fmt.Sprintf(" FETCH NEXT %d ROWS ONLY", r.limitCount))
			return
		}
	}

	// otherwise (subquery and normally)
	basicElements := ctx.Subquery_basic_elements()
	if basicElements.Query_block().Fetch_clause() != nil {
		r.overrideFetchClause(basicElements.Query_block().Fetch_clause())
		return
	}
	r.rewriter.InsertAfterDefault(basicElements.GetStop().GetTokenIndex(), fmt.Sprintf(" FETCH NEXT %d ROWS ONLY", r.limitCount))
}

func (r *plsqlRewriter) overrideFetchClause(fetchClause plsql.IFetch_clauseContext) {
	expression := fetchClause.Expression()
	if expression != nil {
		userLimitText := expression.GetText()
		limit, _ := strconv.Atoi(userLimitText)
		if limit == 0 || r.limitCount < limit {
			limit = r.limitCount
		}
		r.rewriter.ReplaceDefault(expression.GetStart().GetTokenIndex(), expression.GetStop().GetTokenIndex(), fmt.Sprintf("%d", limit))
	}
}
