package bigquery

import (
	"context"
	"fmt"
	"strings"

	"github.com/antlr4-go/antlr/v4"
	parser "github.com/bytebase/google-sql-parser"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

type querySpanExtractor struct {
	ctx             context.Context
	defaultDatabase string

	// ctes is the list of common table expressions.
	ctes []*base.PseudoTable

	gCtx base.GetQuerySpanContext
}

func newQuerySpanExtractor(defaultDatabase string, gCtx base.GetQuerySpanContext, _ bool) *querySpanExtractor {
	return &querySpanExtractor{
		defaultDatabase: defaultDatabase,
		gCtx:            gCtx,
	}
}

func (q *querySpanExtractor) getQuerySpan(ctx context.Context, stmt string) (*base.QuerySpan, error) {
	parseResults, err := ParseBigQuerySQL(stmt)
	if err != nil {
		return nil, err
	}
	tree := parseResults.Tree
	q.ctx = ctx
	accessTables, err := getAccessTables(q.defaultDatabase, tree)
	if err != nil {
		return nil, err
	}
	allSystems, mixed := isMixedQuery(accessTables)
	if mixed {
		return nil, base.MixUserSystemTablesError
	}
	if allSystems {
		return &base.QuerySpan{
			SourceColumns: base.SourceColumnSet{},
			Results:       []base.QuerySpanResult{},
		}, nil
	}

	return &base.QuerySpan{
		SourceColumns: accessTables,
	}, nil
}

func (q *querySpanExtractor) getQuerySpanResult(tree *parser.StmtContext) ([]base.QuerySpanResult, error) {
	if tree.Query_statement() == nil {
		return nil, errors.Errorf("unsupported non-query statement")
	}

	tableSource, err := q.extractTableSourceFromQuery(tree.Query_statement().Query())
	if err != nil {
		return nil, err
	}

	return tableSource.GetQuerySpanResult(), nil
}

func (q *querySpanExtractor) extractTableSourceFromQuery(query parser.IQueryContext) (base.TableSource, error) {
	queryWithoutPipe := query.Query_without_pipe_operators()
	if queryWithoutPipe == nil {
		return nil, errors.Errorf("unsupported query with pipe operators")
	}
	return q.extractTableSourceFromQueryWithoutPipe(queryWithoutPipe)
}

func (q *querySpanExtractor) extractTableSourceFromQueryWithoutPipe(queryWithoutPipe parser.IQuery_without_pipe_operatorsContext) (base.TableSource, error) {
	// TODO(zp): handle CTE.
	var withClause parser.IWith_clauseContext
	if queryWithoutPipe.With_clause() != nil {
		withClause = queryWithoutPipe.With_clause()
	} else if queryWithoutPipe.With_clause_with_trailing_comma() != nil {
		withClause = queryWithoutPipe.With_clause_with_trailing_comma().With_clause()
	}
	fmt.Println(withClause)

	return q.extractTableSourceFromQueryPrimaryOrSetOperation(queryWithoutPipe.Query_primary_or_set_operation())
}

func (q *querySpanExtractor) extractTableSourceFromQueryPrimaryOrSetOperation(queryPrimaryOrSetOperation parser.IQuery_primary_or_set_operationContext) (base.TableSource, error) {
	if queryPrimaryOrSetOperation.Query_primary() != nil {
		return q.extractTableSourceFromQueryPrimary(queryPrimaryOrSetOperation.Query_primary())
	}
	// TODO(zp): handle set operation.
	return nil, nil
}

func (q *querySpanExtractor) extractTableSourceFromQueryPrimary(queryPrimary parser.IQuery_primaryContext) (base.TableSource, error) {
	if queryPrimary.Select_() != nil {
		return q.extractTableSourceFromSelect(queryPrimary.Select_())
	}
	// TODO(zp): handle parenthesized query.
	return nil, nil
}

func (q *querySpanExtractor) extractTableSourceFromSelect(select_ parser.ISelectContext) (base.TableSource, error) {
	var fromFields []base.QuerySpanResult
	if select_.From_clause() != nil {

	}
}

func (q *querySpanExtractor) extractTableSourceFromFromClause(fromClause parser.IFrom_clauseContext) (*base.PseudoTable, error) {
	contents := fromClause.From_clause_contents()
	// TODO(zp): handle suffix.

}

func (q *querySpanExtractor) extractTableSourceFromTablePrimary(tablePrimary parser.ITable_primaryContext) (base.TableSource, error) {
	if tablePrimary.Tvf_with_suffixes() != nil {
		// We do not support table value function because we do not have the returnning columns information.
		return nil, errors.Errorf("unsupported table value function: %s", tablePrimary.GetText())
	}
	if tablePrimary.Table_path_expression() != nil {
		return q.extractTableSourceFromTablePathExpression(tablePrimary.Table_path_expression())
	}

	// TODO(zp): handle other case
	return nil, nil
}

func (q *querySpanExtractor) extractTableSourceFromTablePathExpression(tablePathExpression parser.ITable_path_expressionContext) (base.TableSource, error) {
	base := tablePathExpression.Table_path_expression_base()
	if base.Unnest_expression() != nil {
		// We do not support unnest expression because we do not have the returnning columns information.
		return nil, errors.Errorf("unsupported unnest expression: %s", base.GetText())
	}
	var tableName string
	datasetName := q.defaultDatabase

	if slashedOrDashedPathExpression := base.Maybe_slashed_or_dashed_path_expression(); slashedOrDashedPathExpression != nil {
		if slashedOrDashedPathExpression.Maybe_dashed_path_expression() != nil {
			if maybeDashedPathExpr := slashedOrDashedPathExpression.Maybe_dashed_path_expression(); maybeDashedPathExpr != nil {
				// TODO(zp): support dashed path expression, for example, REGION-us
				if maybeDashedPathExpr.Dashed_path_expression() != nil {
					return nil, errors.Errorf("unsupported dashed path expression: %s", base.GetText())
				}
				// REFACTOR(zp): refactor the code to extract table name and dataset name.
				allIdentifiers := maybeDashedPathExpr.Path_expression().AllIdentifier()
				if len(allIdentifiers) > 0 {
					tableName = unquoteIdentifier(allIdentifiers[len(allIdentifiers)-1])
					if len(allIdentifiers) > 1 {
						datasetName = unquoteIdentifier(allIdentifiers[len(allIdentifiers)-2])
					}
				}
			}
			if slashedOrDashedPathExpression.Slashed_path_expression() != nil {
				return nil, errors.Errorf("unsupported slashed path expression: %s", base.GetText())
			}
		}
	}

	tabelSource, err := q.findTableSchema(datasetName, tableName)
	if err != nil {
		return nil, err
	}
	// TODO(zp): add in q.from
	return tabelSource, nil
}

func (q *querySpanExtractor) findTableSchema(datasetName string, tableName string) (base.TableSource, error) {
	// https://cloud.google.com/bigquery/docs/reference/standard-sql/lexical#case_sensitivity
	// Dataset and table names are case-sensitive unless the is_case_insensitive option is set to TRUE.
	_, databaseMetadata, err := q.gCtx.GetDatabaseMetadataFunc(q.ctx, q.gCtx.InstanceID, datasetName)
	if err != nil {
		return nil, err
	}
	if databaseMetadata == nil {
		return nil, errors.Errorf("dataset %q not found", datasetName)
	}

	schema := databaseMetadata.GetSchema("")
	if schema == nil {
		return nil, errors.Errorf("table %q not found", tableName)
	}

	table := schema.GetTable(tableName)
	if table == nil {
		return nil, errors.Errorf("table %q not found", tableName)
	}

	var columns []string
	for _, column := range table.GetColumns() {
		columns = append(columns, column.Name)
	}
	return &base.PhysicalTable{
		Server:   "",
		Database: datasetName,
		Schema:   "",
		Name:     tableName,
		Columns:  columns,
	}, nil
}

func getAccessTables(defaultDatabase string, tree antlr.Tree) (base.SourceColumnSet, error) {
	l := newAccessTableListener(defaultDatabase)
	result := make(base.SourceColumnSet)
	antlr.ParseTreeWalkerDefault.Walk(l, tree)
	if l.err != nil {
		return nil, l.err
	}
	result, _ = base.MergeSourceColumnSet(result, l.sourceColumnSet)
	return result, nil
}

type accessTableListener struct {
	*parser.BaseGoogleSQLParserListener

	currentDatabase string
	sourceColumnSet base.SourceColumnSet
	err             error
}

func newAccessTableListener(currentDatabase string) *accessTableListener {
	return &accessTableListener{
		currentDatabase: currentDatabase,
		sourceColumnSet: make(base.SourceColumnSet),
	}
}

func (l *accessTableListener) EnterTable_path_expression(ctx *parser.Table_path_expressionContext) {
	if l.err != nil {
		return
	}
	// TODO(zp): Handle other unusual table path expression.
	exprBase := ctx.Table_path_expression_base()
	slashedOrDashedPathExpr := exprBase.Maybe_slashed_or_dashed_path_expression()
	if slashedOrDashedPathExpr == nil {
		l.err = errors.Errorf("unsupported table path expression: %s", ctx.GetText())
		return
	}

	dashedPathExpr := slashedOrDashedPathExpr.Maybe_dashed_path_expression()
	if dashedPathExpr == nil {
		l.err = errors.Errorf("unsupported slashed table path expression: %s", ctx.GetText())
		return
	}

	pathExpr := dashedPathExpr.Path_expression()
	if pathExpr == nil {
		l.err = errors.Errorf("unsupported dashed table path expression: %s", ctx.GetText())
	}

	// Table name syntax: https://cloud.google.com/bigquery/docs/reference/standard-sql/lexical
	// Most of the time, the syntax can be [project_id.][dataset_id.]table_id.
	// One difference is that the user access INFORMATION_SCHEMA in dataset,
	// the syntax would be [project_id.]([region_id.]|[dataset_id.])INFORMATION_SCHEMA.VIEW_NAME.
	// In this case, we treat the INFORMATION_SCHEMA as schema name.
	allIdentifiers := pathExpr.AllIdentifier()
	if len(allIdentifiers) == 0 {
		return
	}

	columnSource := base.ColumnResource{
		Database: l.currentDatabase,
	}
	lastIdentifier := allIdentifiers[len(allIdentifiers)-1]
	tableName := unquoteIdentifier(lastIdentifier)
	columnSource.Table = tableName

	if len(allIdentifiers) >= 2 {
		identifier := unquoteIdentifier(allIdentifiers[len(allIdentifiers)-2])
		if strings.EqualFold(identifier, "INFORMATION_SCHEMA") {
			columnSource.Schema = identifier
		} else {
			columnSource.Database = identifier
		}
	}

	l.sourceColumnSet[columnSource] = true
}

func unquoteIdentifier(identifier parser.IIdentifierContext) string {
	if len(identifier.GetText()) >= 3 && strings.HasPrefix(identifier.GetText(), "`") && strings.HasSuffix(identifier.GetText(), "`") {
		return identifier.GetText()[1 : len(identifier.GetText())-1]
	}
	return identifier.GetText()
}

func isMixedQuery(m base.SourceColumnSet) (bool, bool) {
	hasSystem, hasUser := false, false
	for table := range m {
		if isSystemResource(table) {
			hasSystem = true
		} else {
			hasUser = true
		}
	}

	if hasSystem && hasUser {
		return false, true
	}

	return !hasUser && hasSystem, false
}

func isSystemResource(resource base.ColumnResource) bool {
	if strings.EqualFold(resource.Schema, "INFORMATION_SCHEMA") {
		return true
	}
	return false
}
