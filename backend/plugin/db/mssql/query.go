package mssql

import (
	"database/sql"
	"fmt"
	"log/slog"
	"strconv"
	"strings"

	"github.com/antlr4-go/antlr/v4"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/types/known/timestamppb"

	tsql "github.com/bytebase/tsql-parser"
	mssqldb "github.com/microsoft/go-mssqldb"

	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
	tsqlparser "github.com/bytebase/bytebase/backend/plugin/parser/tsql"
	"github.com/bytebase/bytebase/backend/utils"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
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
	case "SMALLDATETIME", "DATETIME", "DATETIME2", "DATE", "TIME":
		return new(sql.NullString)
	case "DATETIMEOFFSET":
		return new(sql.NullTime)
	case "IMAGE":
		return new([]byte)
	case "BINARY":
		return new([]byte)
	case "SQL_VARIANT":
		return new([]byte)
	}
	return new(sql.NullString)
}

func convertValue(typeName string, value any) *v1pb.RowValue {
	switch raw := value.(type) {
	case *sql.NullString:
		if raw.Valid {
			if typeName == "DATETIME" || typeName == "DATETIME2" || typeName == "SMALLDATETIME" {
				// Convert "2024-09-24T03:23:28.921281Z" to "2024-09-24 03:23:28.921281"
				timestampWithoutTimezone := strings.ReplaceAll(strings.ReplaceAll(raw.String, "T", " "), "Z", "")
				return &v1pb.RowValue{
					Kind: &v1pb.RowValue_StringValue{
						StringValue: timestampWithoutTimezone,
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
			return &v1pb.RowValue{
				Kind: &v1pb.RowValue_TimestampValue{TimestampValue: timestamppb.New(raw.Time)},
			}
		}
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
	result, err := tsqlparser.ParseTSQL(singleStatement)
	if err != nil {
		return "", err
	}

	listener := &tsqlRewriter{
		limitCount: limitCount,
		hasFetch:   false,
		hasTop:     false,
	}

	listener.rewriter = *antlr.NewTokenStreamRewriter(result.Tokens)
	antlr.ParseTreeWalkerDefault.Walk(listener, result.Tree)
	if listener.err != nil {
		return "", errors.Wrapf(listener.err, "statement: %s", singleStatement)
	}

	res := listener.rewriter.GetTextDefault()
	res = strings.TrimRightFunc(res, utils.IsSpaceOrSemicolon) + ";"

	return res, nil
}

type tsqlRewriter struct {
	*tsql.BaseTSqlParserListener

	rewriter   antlr.TokenStreamRewriter
	err        error
	limitCount int
	hasFetch   bool
	hasTop     bool
}

// EnterSelect_statement_standalone is called when production select_statement_standalone is entered.
func (r *tsqlRewriter) EnterSelect_statement_standalone(ctx *tsql.Select_statement_standaloneContext) {
	r.handleSelectStatementDryRun(ctx.Select_statement())
	r.handleSelectStatement(ctx.Select_statement())
}

func (r *tsqlRewriter) handleSelectStatementDryRun(ctx tsql.ISelect_statementContext) {
	if ctx.Select_order_by_clause() != nil {
		r.handleSelectOrderByDryRun(ctx.Select_order_by_clause())
	}

	if ctx.Query_expression() != nil {
		if ctx.Query_expression().AllSql_union() != nil && len(ctx.Query_expression().AllSql_union()) > 0 {
			r.handleSqlunionDryRun(ctx.Query_expression())
		}
		if ctx.Query_expression().Select_order_by_clause() != nil {
			r.handleSelectOrderByDryRun(ctx.Query_expression().Select_order_by_clause())
		}
		r.handleQuerySpecificationDryRun(ctx.Query_expression().Query_specification())
	}
}

func (r *tsqlRewriter) handleSqlunionDryRun(ctx tsql.IQuery_expressionContext) {
	if ctx.AllSql_union() == nil || len(ctx.AllSql_union()) == 0 {
		// non-union
		return
	}
	// handle union left side
	r.handleQuerySpecificationDryRun(ctx.Query_specification())
	// handle union right side
	r.handleQuerySpecificationDryRun(ctx.Get_sql_union().Query_specification())
}

func (r *tsqlRewriter) handleSelectOrderByDryRun(ctx tsql.ISelect_order_by_clauseContext) {
	if ctx.GetFetch_rows() != nil {
		r.hasFetch = true

		r.overrideFetchRows(ctx)
	}
}

func (r *tsqlRewriter) handleQuerySpecificationDryRun(ctx tsql.IQuery_specificationContext) {
	if ctx.Top_clause() != nil {
		r.hasTop = true

		r.overrideTopClause(ctx.Top_clause())
	}
}

func (r *tsqlRewriter) handleSelectStatement(ctx tsql.ISelect_statementContext) {
	// check outermost order by clause.
	if ctx.Select_order_by_clause() != nil {
		r.handleSelectOrderBy(ctx.Select_order_by_clause())
	}
	if r.hasFetch {
		return
	}

	if ctx.Query_expression() != nil {
		// check union, except, intersect.
		if ctx.Query_expression().AllSql_union() != nil && len(ctx.Query_expression().AllSql_union()) > 0 {
			r.handleSqlunion(ctx.Query_expression())
			return
		}

		// process orderby in outermost query_expression.
		if ctx.Query_expression().Select_order_by_clause() != nil {
			r.handleSelectOrderBy(ctx.Query_expression().Select_order_by_clause())
		}
		if r.hasFetch {
			return
		}
		r.handleQuerySpecification(ctx.Query_expression().Query_specification())

		if r.hasTop {
			return
		}
	}
}

func (r *tsqlRewriter) handleSqlunion(ctx tsql.IQuery_expressionContext) {
	if r.hasTop {
		return
	}
	if ctx.AllSql_union() == nil || len(ctx.AllSql_union()) == 0 {
		// non-union
		return
	}
	// handle union left side
	querySpecification := ctx.Query_specification()
	if querySpecification.GetAllOrDistinct() != nil {
		r.hasTop = true
		r.rewriter.InsertAfterDefault(querySpecification.GetAllOrDistinct().GetStop(), fmt.Sprintf(" TOP %d", r.limitCount))
		return
	}
	r.rewriter.InsertAfterDefault(querySpecification.SELECT().GetSourceInterval().Stop, fmt.Sprintf(" TOP %d", r.limitCount))

	// handle union right side
	querySpecification = ctx.Get_sql_union().Query_specification()
	if querySpecification.GetAllOrDistinct() != nil {
		r.hasTop = true
		r.rewriter.InsertAfterDefault(querySpecification.GetAllOrDistinct().GetStop(), fmt.Sprintf(" TOP %d", r.limitCount))
		return
	}
	r.rewriter.InsertAfterDefault(querySpecification.SELECT().GetSourceInterval().Stop, fmt.Sprintf(" TOP %d", r.limitCount))
}

func (r *tsqlRewriter) handleQuerySpecification(ctx tsql.IQuery_specificationContext) {
	if r.hasFetch {
		return
	}
	if ctx.Top_clause() != nil {
		r.hasTop = true

		r.overrideTopClause(ctx.Top_clause())
		return
	}

	// append after select_optional_clauses
	if ctx.GetAllOrDistinct() != nil {
		r.hasTop = true
		r.rewriter.InsertAfterDefault(ctx.GetAllOrDistinct().GetStop(), fmt.Sprintf(" TOP %d", r.limitCount))
		return
	}
	// append after select keyword.
	r.rewriter.InsertAfterDefault(ctx.SELECT().GetSourceInterval().Stop, fmt.Sprintf(" TOP %d", r.limitCount))
	r.hasTop = true
}

func (r *tsqlRewriter) handleSelectOrderBy(ctx tsql.ISelect_order_by_clauseContext) {
	if r.hasTop {
		return
	}

	r.hasFetch = true
	if ctx.GetFetch_rows() != nil {
		r.overrideFetchRows(ctx)
		return
	}

	// OFFSET is required.
	if ctx.GetOffset_rows() == nil {
		r.rewriter.InsertAfterDefault(ctx.Order_by_clause().GetStop().GetTokenIndex(), fmt.Sprintf(" OFFSET 0 ROWS FETCH NEXT %d ROWS ONLY", r.limitCount))
	} else {
		r.rewriter.InsertAfterDefault(ctx.GetOffset_rows().GetTokenIndex(), fmt.Sprintf(" FETCH NEXT %d ROWS ONLY", r.limitCount))
	}
}

func (r *tsqlRewriter) overrideTopClause(topClause tsql.ITop_clauseContext) {
	var limit int
	topCount := topClause.Top_count()
	if topCount != nil {
		userLimitText := topCount.GetText()
		limit, _ = strconv.Atoi(userLimitText)
	}
	if limit == 0 || r.limitCount < limit {
		limit = r.limitCount
	}

	r.rewriter.ReplaceDefault(topClause.GetStart().GetTokenIndex(), topClause.GetStop().GetTokenIndex(), fmt.Sprintf("TOP %d", limit))
}

func (r *tsqlRewriter) overrideFetchRows(ctx tsql.ISelect_order_by_clauseContext) {
	// Offset must exists.
	if len(ctx.AllExpression()) > 1 {
		expression := ctx.Expression(1)
		if expression != nil {
			userLimitText := expression.GetText()
			limit, _ := strconv.Atoi(userLimitText)
			if limit == 0 || r.limitCount < limit {
				limit = r.limitCount
			}
			r.rewriter.ReplaceDefault(expression.GetStart().GetTokenIndex(), expression.GetStop().GetTokenIndex(), fmt.Sprintf("%d", limit))
		}
	}
}
