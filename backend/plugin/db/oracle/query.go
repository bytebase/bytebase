package oracle

import (
	"database/sql"
	"fmt"
	"log/slog"
	"strconv"
	"strings"

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
	case "VARCHAR", "TEXT", "UUID", "DATE", "TIMESTAMPDTY", "TIMESTAMPLTZ_DTY":
		return new(sql.NullString)
	case "BOOL":
		return new(sql.NullBool)
	case "INT", "INTEGER", "TINYINT", "SMALLINT", "MEDIUMINT", "BIGINT", "INT2", "INT4", "INT8":
		return new(sql.NullInt64)
	case "FLOAT", "DOUBLE", "FLOAT4", "FLOAT8":
		return new(sql.NullFloat64)
	case "BIT", "VARBIT":
		return new([]byte)
	case "TIMESTAMPTZ_DTY":
		return new(sql.NullTime)
	default:
		return new(sql.NullString)
	}
}

func convertValue(_ string, value any) *v1pb.RowValue {
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
			return &v1pb.RowValue{
				Kind: &v1pb.RowValue_TimestampTzValue{
					TimestampTzValue: &v1pb.RowValue_TimestampTZ{
						Timestamp: timestamppb.New(raw.Time),
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
