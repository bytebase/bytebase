package mssql

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/antlr4-go/antlr/v4"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"

	tsql "github.com/bytebase/tsql-parser"
	mssqldb "github.com/microsoft/go-mssqldb"

	"github.com/bytebase/bytebase/backend/common"
	tsqlparser "github.com/bytebase/bytebase/backend/plugin/parser/tsql"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
)

var nullRowValue = &v1pb.RowValue{
	Kind: &v1pb.RowValue_NullValue{
		NullValue: structpb.NullValue_NULL_VALUE,
	},
}

func rowsToQueryResult(rows *sql.Rows, limit int64) (*v1pb.QueryResult, error) {
	columnNames, err := rows.Columns()
	if err != nil {
		return nil, err
	}
	columnTypes, err := rows.ColumnTypes()
	if err != nil {
		return nil, err
	}
	// DatabaseTypeName returns the database system name of the column type.
	// refer: https://pkg.go.dev/database/sql#ColumnType.DatabaseTypeName
	var columnTypeNames []string
	for _, v := range columnTypes {
		columnTypeNames = append(columnTypeNames, strings.ToUpper(v.DatabaseTypeName()))
	}
	result := &v1pb.QueryResult{
		ColumnNames:     columnNames,
		ColumnTypeNames: columnTypeNames,
	}
	columnLength := len(columnNames)

	if columnLength > 0 {
		for rows.Next() {
			values := make([]any, columnLength)
			for i, v := range columnTypeNames {
				values[i] = makeValueByTypeName(v)
			}

			if err := rows.Scan(values...); err != nil {
				return nil, err
			}

			row := &v1pb.QueryRow{}
			for i := 0; i < columnLength; i++ {
				rowValue := nullRowValue
				switch raw := values[i].(type) {
				case *sql.NullString:
					if raw.Valid {
						rowValue = &v1pb.RowValue{
							Kind: &v1pb.RowValue_StringValue{
								StringValue: raw.String,
							},
						}
					}
				case *sql.NullInt64:
					if raw.Valid {
						rowValue = &v1pb.RowValue{
							Kind: &v1pb.RowValue_Int64Value{
								Int64Value: raw.Int64,
							},
						}
					}
				case *[]byte:
					if len(*raw) > 0 {
						rowValue = &v1pb.RowValue{
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
						rowValue = &v1pb.RowValue{
							Kind: &v1pb.RowValue_Int64Value{
								Int64Value: v,
							},
						}
					}
				case *sql.NullFloat64:
					if raw.Valid {
						rowValue = &v1pb.RowValue{
							Kind: &v1pb.RowValue_DoubleValue{
								DoubleValue: raw.Float64,
							},
						}
					}
				case *mssqldb.NullUniqueIdentifier:
					if raw.Valid {
						rowValue = &v1pb.RowValue{
							Kind: &v1pb.RowValue_StringValue{
								StringValue: raw.UUID.String(),
							},
						}
					}
				}

				row.Values = append(row.Values, rowValue)
			}

			result.Rows = append(result.Rows, row)
			n := len(result.Rows)
			if (n&(n-1) == 0) && int64(proto.Size(result)) > limit {
				result.Error = common.FormatMaximumSQLResultSizeMessage(limit)
				break
			}
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return result, nil
}

func makeValueByTypeName(typeName string) any {
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
	case "SMALLDATETIME", "DATETIME", "DATETIME2", "DATE", "TIME", "DATETIMEOFFSET":
		return new(sql.NullString)
	case "IMAGE":
		return new([]byte)
	case "BINARY":
		return new([]byte)
	case "SQL_VARIANT":
		return new([]byte)
	}
	return new(sql.NullString)
}

// singleStatement must be a selectStatement for mssql.
func getMSSQLStatementWithResultLimit(singleStatement string, limitCount int) (string, error) {
	result, err := tsqlparser.ParseTSQL(singleStatement)
	if err != nil {
		return "", err
	}

	listener := &tsqlRewriter{
		limitCount: limitCount,
		hasOffset:  false,
		hasFetch:   false,
		hasTop:     false,
	}

	listener.rewriter = *antlr.NewTokenStreamRewriter(result.Tokens)
	antlr.ParseTreeWalkerDefault.Walk(listener, result.Tree)
	if listener.err != nil {
		return "", errors.Wrapf(listener.err, "statement: %s", singleStatement)
	}

	res := listener.rewriter.GetTextDefault()
	res = strings.TrimRight(res, " \t\n\r\f;") + ";"

	return res, nil
}

type tsqlRewriter struct {
	*tsql.BaseTSqlParserListener

	rewriter   antlr.TokenStreamRewriter
	err        error
	limitCount int
	hasOffset  bool
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
	r.hasOffset = r.hasOffset || ctx.GetOffset_rows() != nil
	r.hasFetch = r.hasFetch || ctx.GetFetch_rows() != nil
}

func (r *tsqlRewriter) handleQuerySpecificationDryRun(ctx tsql.IQuery_specificationContext) {
	r.hasTop = r.hasTop || ctx.Top_clause() != nil
}

func (r *tsqlRewriter) handleSelectStatement(ctx tsql.ISelect_statementContext) {
	// check outermost order by clause.
	if ctx.Select_order_by_clause() != nil {
		r.handleSelectOrderBy(ctx.Select_order_by_clause())
	}
	if r.hasOffset && r.hasFetch {
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
		if r.hasOffset && r.hasFetch {
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
	idx := querySpecification.SELECT().GetSourceInterval().Stop
	r.rewriter.InsertAfterDefault(idx, fmt.Sprintf(" TOP %d", r.limitCount))

	// handle union right side
	querySpecification = ctx.Get_sql_union().Query_specification()
	if querySpecification.GetAllOrDistinct() != nil {
		r.hasTop = true
		r.rewriter.InsertAfterDefault(querySpecification.GetAllOrDistinct().GetStop(), fmt.Sprintf(" TOP %d", r.limitCount))
		return
	}
	idx = querySpecification.SELECT().GetSourceInterval().Stop
	r.rewriter.InsertAfterDefault(idx, fmt.Sprintf(" TOP %d", r.limitCount))
}

func (r *tsqlRewriter) handleQuerySpecification(ctx tsql.IQuery_specificationContext) {
	if r.hasOffset {
		return
	}
	if ctx.Top_clause() != nil {
		r.hasTop = true
		return
	}

	// append after select_optional_clauses
	if ctx.GetAllOrDistinct() != nil {
		r.hasTop = true
		r.rewriter.InsertAfterDefault(ctx.GetAllOrDistinct().GetStop(), fmt.Sprintf(" TOP %d", r.limitCount))
		return
	}
	// append after select keyword.
	idx := ctx.SELECT().GetSourceInterval().Stop
	r.rewriter.InsertAfterDefault(idx, fmt.Sprintf(" TOP %d", r.limitCount))
	r.hasTop = true
}

func (r *tsqlRewriter) handleSelectOrderBy(ctx tsql.ISelect_order_by_clauseContext) {
	if r.hasTop {
		return
	}
	if ctx.GetOffset_rows() != nil {
		r.hasOffset = true
	}
	if ctx.GetFetch_rows() != nil {
		r.hasFetch = true
	}
	// respect original value
	if r.hasFetch && r.hasOffset {
		return
	}

	// no offset, add offet and fetch
	if ctx.GetOffset_rows() == nil {
		r.hasOffset = true
		r.hasFetch = true
		r.rewriter.InsertAfterDefault(ctx.Order_by_clause().GetStop().GetTokenIndex(), fmt.Sprintf(" OFFSET 0 ROWS FETCH NEXT %d ROWS ONLY", r.limitCount))
		return
	}
	r.hasOffset = true
	// has offset, but no fetch, add fetch
	if ctx.GetFetch_rows() == nil {
		r.hasFetch = true
		idx := ctx.GetOffset_rows().GetTokenIndex()
		r.rewriter.InsertAfterDefault(idx, fmt.Sprintf(" FETCH NEXT %d ROWS ONLY", r.limitCount))
		return
	}
}
