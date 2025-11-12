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

	"github.com/bytebase/parser/plsql"

	"github.com/bytebase/bytebase/backend/common/log"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
	plsqlparser "github.com/bytebase/bytebase/backend/plugin/parser/plsql"
	"github.com/bytebase/bytebase/backend/utils"
)

const dbVersion12 = 12

// ========== Type Conversion Functions ==========

// makeValueByTypeName creates appropriate Go types for Oracle column types.
// DATE: date.
// TIMESTAMPDTY: timestamp.
// TIMESTAMPTZ_DTY: timestamp with time zone.
// TIMESTAMPLTZ_DTY: timezone with local time zone.
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
			return convertTimestamp(typeName, columnType, raw.Time)
		}
	}
	return util.NullRowValue
}

// convertTimestamp handles Oracle timestamp type conversions.
// The go-ora driver retrieves the database timezone from the wire protocol and appends it to the timestamp.
// To ensure consistency with Oracle Date expectations, we handle different timestamp types appropriately.
// https://github.com/sijms/go-ora/blob/2962e725e7a756a667a546fb360ef09afd4c8bd0/v2/parameter.go#L616
func convertTimestamp(typeName string, columnType *sql.ColumnType, t time.Time) *v1pb.RowValue {
	_, scale, _ := columnType.DecimalSize()

	switch typeName {
	case "DATE", "TIMESTAMPDTY":
		// Strip timezone information for DATE and TIMESTAMP types
		timeStripped := time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), time.UTC)
		return &v1pb.RowValue{
			Kind: &v1pb.RowValue_TimestampValue{
				TimestampValue: &v1pb.RowValue_Timestamp{
					GoogleTimestamp: timestamppb.New(timeStripped),
					Accuracy:        int32(scale),
				},
			},
		}

	case "TIMESTAMPLTZ_DTY":
		// Handle local timezone timestamp
		// This timestamp is not consistent with sqlplus likely due to db and session timezone.
		// TODO(d): fix the go-ora library.
		s := t.Format("2006-01-02 15:04:05.000000000")
		parsedTime, err := time.Parse(time.DateTime, s)
		if err != nil {
			return util.NullRowValue
		}
		return &v1pb.RowValue{
			Kind: &v1pb.RowValue_TimestampValue{
				TimestampValue: &v1pb.RowValue_Timestamp{
					GoogleTimestamp: timestamppb.New(parsedTime),
					Accuracy:        int32(scale),
				},
			},
		}

	default:
		// Handle timestamp with timezone
		zone, offset := t.Zone()
		return &v1pb.RowValue{
			Kind: &v1pb.RowValue_TimestampTzValue{
				TimestampTzValue: &v1pb.RowValue_TimestampTZ{
					GoogleTimestamp: timestamppb.New(t),
					Zone:            zone,
					Offset:          int32(offset),
					Accuracy:        int32(scale),
				},
			},
		}
	}
}

// ========== Limit Handling Functions ==========

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

// shouldSkipLimit checks if the statement needs a limit clause
func shouldSkipLimit(stmt string) bool {
	ok, err := skipAddLimit(stmt)
	return err == nil && ok
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
	results, err := plsqlparser.ParsePLSQL(statement)
	if err != nil {
		return "", err
	}
	if len(results) == 0 {
		return "", errors.New("no parse results")
	}
	if len(results) > 1 {
		return "", errors.Errorf("expected single statement, got %d statements", len(results))
	}
	tree := results[0].Tree
	stream := results[0].Tokens

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

// ========== PLSQL AST Walker and Rewriter ==========

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

// ========== Skip Limit Logic for DUAL Queries ==========

// skipAddLimit checks if the statement needs a limit clause.
// For Oracle, we think the statement like "SELECT xxx FROM DUAL" does not need a limit clause.
// More details, xxx can not be a subquery.
func skipAddLimit(stmt string) (bool, error) {
	results, err := plsqlparser.ParsePLSQL(stmt)
	if err != nil {
		return false, err
	}
	if len(results) == 0 {
		return false, nil
	}
	// Multiple statements should not skip limit
	if len(results) > 1 {
		return false, nil
	}
	tree := results[0].Tree

	selectStatement := extractSimpleSelectStatement(tree)
	if selectStatement == nil {
		return false, nil
	}

	// Check if select has additional clauses that prevent skipping limit
	if hasAdditionalClauses(selectStatement) {
		return false, nil
	}

	queryBlock := extractQueryBlock(selectStatement)
	if queryBlock == nil {
		return false, nil
	}

	// Check if query block has complex features
	if hasComplexQueryFeatures(queryBlock) {
		return false, nil
	}

	// Must be a simple SELECT FROM DUAL
	if !isFromDual(queryBlock) {
		return false, nil
	}

	// Check selected elements don't contain subqueries
	return !hasSubqueriesInSelection(queryBlock), nil
}

// extractSimpleSelectStatement extracts a simple SELECT statement from the parse tree.
// Returns nil if the tree doesn't represent a simple SELECT.
func extractSimpleSelectStatement(tree antlr.Tree) *plsql.Select_statementContext {
	sqlScript, ok := tree.(*plsql.Sql_scriptContext)
	if !ok {
		return nil
	}

	if len(sqlScript.AllSql_plus_command()) > 0 || len(sqlScript.AllUnit_statement()) != 1 {
		return nil
	}

	unitStatement := sqlScript.Unit_statement(0)
	if unitStatement == nil {
		return nil
	}

	dml := unitStatement.Data_manipulation_language_statements()
	if dml == nil {
		return nil
	}

	if selectStmt := dml.Select_statement(); selectStmt != nil {
		if stmt, ok := selectStmt.(*plsql.Select_statementContext); ok {
			return stmt
		}
	}
	return nil
}

// hasAdditionalClauses checks if a SELECT statement has clauses that prevent limit skipping.
func hasAdditionalClauses(selectStatement *plsql.Select_statementContext) bool {
	return len(selectStatement.AllFor_update_clause()) != 0 ||
		len(selectStatement.AllOrder_by_clause()) != 0 ||
		len(selectStatement.AllOffset_clause()) != 0 ||
		len(selectStatement.AllFetch_clause()) != 0
}

// extractQueryBlock extracts the query block from a SELECT statement.
func extractQueryBlock(selectStatement *plsql.Select_statementContext) *plsql.Query_blockContext {
	selectOnly := selectStatement.Select_only_statement()
	if selectOnly == nil {
		return nil
	}

	subquery := selectOnly.Subquery()
	if subquery == nil || len(subquery.AllSubquery_operation_part()) != 0 {
		return nil
	}

	subqueryBasicElements := subquery.Subquery_basic_elements()
	if subqueryBasicElements == nil || subqueryBasicElements.Subquery() != nil {
		return nil
	}

	if queryBlock := subqueryBasicElements.Query_block(); queryBlock != nil {
		if block, ok := queryBlock.(*plsql.Query_blockContext); ok {
			return block
		}
	}
	return nil
}

// hasComplexQueryFeatures checks if a query block has complex features.
func hasComplexQueryFeatures(queryBlock *plsql.Query_blockContext) bool {
	return queryBlock.Subquery_factoring_clause() != nil ||
		queryBlock.DISTINCT() != nil ||
		queryBlock.ALL() != nil ||
		queryBlock.UNIQUE() != nil ||
		queryBlock.Into_clause() != nil ||
		queryBlock.Where_clause() != nil ||
		queryBlock.Hierarchical_query_clause() != nil ||
		queryBlock.Group_by_clause() != nil ||
		queryBlock.Model_clause() != nil ||
		queryBlock.Order_by_clause() != nil ||
		queryBlock.Fetch_clause() != nil
}

// isFromDual checks if the query is selecting from DUAL table.
func isFromDual(queryBlock *plsql.Query_blockContext) bool {
	from := queryBlock.From_clause()
	return from != nil && strings.EqualFold(from.GetText(), "FROMDUAL")
}

// hasSubqueriesInSelection checks if the selected elements contain subqueries.
func hasSubqueriesInSelection(queryBlock *plsql.Query_blockContext) bool {
	selectedList := queryBlock.Selected_list()
	if selectedList == nil || selectedList.ASTERISK() != nil {
		return true
	}

	for _, selectedElement := range selectedList.AllSelect_list_elements() {
		if selectedElement.Table_wild() != nil {
			return true
		}

		l := subqueryListener{}
		antlr.ParseTreeWalkerDefault.Walk(&l, selectedElement)
		if l.hasSubquery {
			return true
		}
	}

	return false
}

type subqueryListener struct {
	*plsql.BasePlSqlParserListener
	hasSubquery bool
}

func (l *subqueryListener) EnterSubquery(*plsql.SubqueryContext) {
	l.hasSubquery = true
}
