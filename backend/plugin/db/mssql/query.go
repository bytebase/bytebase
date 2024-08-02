package mssql

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/antlr4-go/antlr/v4"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/proto"

	tsql "github.com/bytebase/tsql-parser"
	mssqldb "github.com/microsoft/go-mssqldb"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/component/masker"
	tsqlparser "github.com/bytebase/bytebase/backend/plugin/parser/tsql"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
)

var noneMasker = masker.NewNoneMasker()

func rowsToQueryResult(rows *sql.Rows) (*v1pb.QueryResult, error) {
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

	if len(columnTypeNames) > 0 {
		for rows.Next() {
			values := make([]any, len(columnTypeNames))
			// isByteValues want to convert StringValue to BytesValue when columnTypeName is BIT or VARBIT
			isByteValues := make([]bool, len(columnTypeNames))
			for i, v := range columnTypeNames {
				values[i], isByteValues[i] = makeValueByTypeName(v)
			}

			if err := rows.Scan(values...); err != nil {
				return nil, err
			}

			var rowData v1pb.QueryRow
			for i := range columnTypeNames {
				rowData.Values = append(rowData.Values, noneMasker.Mask(&masker.MaskData{
					Data:      values[i],
					WantBytes: isByteValues[i],
				}))
			}

			result.Rows = append(result.Rows, &rowData)
			n := len(result.Rows)
			if (n&(n-1) == 0) && proto.Size(result) > common.MaximumSQLResultSize {
				result.Error = common.MaximumSQLResultSizeExceeded
				break
			}
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return result, nil
}

// makeValueByTypeName makes a value and wantByte bool by the type name.
func makeValueByTypeName(typeName string) (any, bool) {
	switch typeName {
	case "TINYINT":
		return new(sql.NullInt64), false
	case "SMALLINT":
		return new(sql.NullInt64), false
	case "INT":
		return new(sql.NullInt64), false
	case "BIGINT":
		return new(sql.NullInt64), false
	case "REAL":
		return new(sql.NullFloat64), false
	case "FLOAT":
		return new(sql.NullFloat64), false
	case "VARBINARY":
		// TODO(zp): Null bytes?
		return new(sql.NullString), true
	case "VARCHAR":
		return new(sql.NullString), false
	case "NVARCHAR":
		return new(sql.NullString), false
	case "BIT":
		return new(sql.NullBool), false
	case "DECIMAL":
		return new(sql.NullString), false
	case "SMALLMONEY":
		return new(sql.NullString), false
	case "MONEY":
		return new(sql.NullString), false

	// TODO(zp): Scan to string now, switch to use time.Time while masking support it.
	// // Source values of type [time.Time] may be scanned into values of type
	// *time.Time, *interface{}, *string, or *[]byte. When converting to
	// the latter two, [time.RFC3339Nano] is used.
	case "SMALLDATETIME":
		return new(sql.NullString), false
	case "DATETIME":
		return new(sql.NullString), false
	case "DATETIME2":
		return new(sql.NullString), false
	case "DATE":
		return new(sql.NullString), false
	case "TIME":
		return new(sql.NullString), false
	case "DATETIMEOFFSET":
		return new(sql.NullString), false

	case "CHAR":
		return new(sql.NullString), false
	case "NCHAR":
		return new(sql.NullString), false
	case "UNIQUEIDENTIFIER":
		return new(mssqldb.NullUniqueIdentifier), false
	case "XML":
		return new(sql.NullString), false
	case "TEXT":
		return new(sql.NullString), false
	case "NTEXT":
		return new(sql.NullString), false
	case "IMAGE":
		return new(sql.NullString), true
	case "BINARY":
		return new(sql.NullString), true
	case "SQL_VARIANT":
		return new(sql.NullString), true
	}
	return new(sql.NullString), true
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
