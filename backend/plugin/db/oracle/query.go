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
	"github.com/bytebase/bytebase/backend/plugin/db/util"
	plsqlparser "github.com/bytebase/bytebase/backend/plugin/parser/plsql"
	"github.com/bytebase/bytebase/backend/utils"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
)

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

func convertValue(typeName string, value any) *v1pb.RowValue {
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
			if typeName == "DATE" || typeName == "TIMESTAMPDTY" {
				timeStripped := time.Date(raw.Time.Year(), raw.Time.Month(), raw.Time.Day(), raw.Time.Hour(), raw.Time.Minute(), raw.Time.Second(), raw.Time.Nanosecond(), time.UTC)
				return &v1pb.RowValue{
					Kind: &v1pb.RowValue_TimestampValue{
						TimestampValue: timestamppb.New(timeStripped),
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
						TimestampValue: timestamppb.New(t),
					},
				}
			}
			zone, offset := raw.Time.Zone()
			return &v1pb.RowValue{
				Kind: &v1pb.RowValue_TimestampTzValue{
					TimestampTzValue: &v1pb.RowValue_TimestampTZ{
						Timestamp: timestamppb.New(raw.Time),
						Zone:      zone,
						Offset:    int32(offset),
					},
				},
			}
		}
	}
	return util.NullRowValue
}

// singleStatement must be a selectStatement for oracle.
func getStatementWithResultLimitFor11g(statement string, limitCount int) string {
	return fmt.Sprintf("SELECT * FROM (%s) WHERE ROWNUM <= %d", util.TrimStatement(statement), limitCount)
}

func getStatementWithResultLimit(statement string, limit int) string {
	trimmedStatement := strings.ToLower(strings.TrimLeftFunc(statement, unicode.IsSpace))
	// Add limit for select statement only
	if !strings.HasPrefix(trimmedStatement, "select") && !strings.HasPrefix(trimmedStatement, "with") {
		return statement
	}
	stmt, err := getStatementWithResultLimitInline(statement, limit)
	if err != nil {
		slog.Error("fail to add limit clause", slog.String("statement", statement), log.BBError(err))
		return getStatementWithResultLimitFor11g(statement, limit)
	}
	return stmt
}

// singleStatement must be a selectStatement for oracle.
func getStatementWithResultLimitInline(statement string, limitCount int) (string, error) {
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
