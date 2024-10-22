package bigquery

import (
	"context"
	"strings"

	"github.com/antlr4-go/antlr/v4"
	parser "github.com/bytebase/google-sql-parser"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

type querySpanExtractor struct {
	ctx             context.Context
	defaultDatabase string

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

	// TODO(zp): statement type check.
	querySpanResult, err := q.getQuerySpanResult(tree.(*parser.RootContext).Stmts().AllStmt()[0])
	if err != nil {
		return nil, err
	}

	return &base.QuerySpan{
		Results:       querySpanResult,
		SourceColumns: accessTables,
	}, nil
}

func (q *querySpanExtractor) getQuerySpanResult(tree parser.IStmtContext) ([]base.QuerySpanResult, error) {
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

func (q *querySpanExtractor) extractTableSourceFromSelect(selectCtx parser.ISelectContext) (base.TableSource, error) {
	var fromFields []base.QuerySpanResult
	var resultFields []base.QuerySpanResult
	if selectCtx.From_clause() != nil {
		tableSource, err := q.extractTableSourceFromFromClause(selectCtx.From_clause())
		if err != nil {
			return nil, err
		}
		fromFields = tableSource.GetQuerySpanResult()
	}

	itemList := selectCtx.Select_clause().Select_list().AllSelect_list_item()
	for _, item := range itemList {
		// TODO(zp): handle other select item.
		if item.Select_column_star() != nil {
			resultFields = append(resultFields, fromFields...)
		}

	}
	return &base.PseudoTable{
		Name:    "",
		Columns: resultFields,
	}, nil
}

func (q *querySpanExtractor) extractSourceColumnSetFromExpr(ctx antlr.ParserRuleContext) (base.SourceColumnSet, error) {
	if ctx == nil {
		return make(base.SourceColumnSet), nil
	}

	// BigQuery support project the field from json, for example, SELECT a.b.c FROM ..., the AST looks like:
	// /
	// └── expression
	//     └── expression_higher_prec_than_and
	//         ├── expression_higher_prec_than_and
	//         │   ├── expression_higher_prec_than_and
	//         │   │   ├── expression_higher_prec_than_and
	//         │   │   ├── DOT_SYMBOL
	//         │   │   └── identifier(a)
	//         │   ├── DOT_SYMBOL
	//         │   └── identifier(b)
	//         ├── DOT_SYMBOL
	//         └── identifier(c)
	// We use DFS algorithm here to find the tallest [expression_higher_prec_than_and DOT_SYMBOL identifier] subtree and
	// treat it as identifier.
	return nil, nil
}

func getPossibleColumnResources(ctx antlr.ParserRuleContext) [][]string {
	// Traverse the tree to find the tallest subtree which matches the pattern:
	// [expression_higher_prec_than_and DOT_SYMBOL identifier]
	// The result is the possible column resources.
	var path []antlr.ParserRuleContext
	path = append(path, ctx)

	var result [][]string
	for len(path) > 0 {
		element := path[len(path)-1]
		path = path[:len(path)-1]
		appendChild := true
		if element, ok := element.(*parser.Expression_higher_prec_than_andContext); ok {
			// If all the terminal node in the subtree is IDENTIFIER, DOT_SYMBOL, we treat is as a possible column resource.
			terminalNodes := getAllChildTerminalNode(element)
			valid := true
			if len(terminalNodes) == 0 {
				valid = false
			}
			firstNode := terminalNodes[0]
			lastNode := terminalNodes[len(terminalNodes)-1]
			if firstNode.GetSymbol().GetTokenType() != parser.GoogleSQLParserIDENTIFIER {
				valid = false
			}
			if lastNode.GetSymbol().GetTokenType() != parser.GoogleSQLParserIDENTIFIER {
				valid = false
			}
			previousType := parser.GoogleSQLParserDOT_SYMBOL
			for _, terminalNode := range terminalNodes {
				switch previousType {
				case parser.GoogleSQLParserIDENTIFIER:
					if terminalNode.GetSymbol().GetTokenType() != parser.GoogleSQLParserDOT_SYMBOL {
						valid = false
						break
					}
					previousType = parser.GoogleSQLParserIDENTIFIER
					if terminalNode.GetSymbol().GetTokenType() != parser.GoogleSQLParserIDENTIFIER {
						valid = false
						break
					}
					previousType = parser.GoogleSQLParserDOT_SYMBOL
				}
			}
			if valid {
				appendChild = false
				var identifierPath []string
				for _, terminalNode := range terminalNodes {
					if terminalNode.GetSymbol().GetTokenType() == parser.GoogleSQLParserIDENTIFIER {
						identifierPath = append(identifierPath, unquoteIdentifierByText(terminalNode.GetText()))
					}
				}
				result = append(result, identifierPath)
			}
		}
		if appendChild {
			for _, child := range element.GetChildren() {
				if child, ok := child.(antlr.ParserRuleContext); ok {
					path = append(path, child)
				}
			}
		}
	}
	return result
}

func getAllChildTerminalNode(ctx antlr.ParserRuleContext) []antlr.TerminalNode {
	allChilds := ctx.GetChildren()
	var result []antlr.TerminalNode
	for _, child := range allChilds {
		if childTerminal, ok := child.(antlr.TerminalNode); ok {
			result = append(result, childTerminal)
			continue
		}
		if childRuleCtx, ok := child.(antlr.ParserRuleContext); ok {
			subResult := getAllChildTerminalNode(childRuleCtx)
			result = append(result, subResult...)
		}
	}
	return result
}

func (q *querySpanExtractor) extractTableSourceFromFromClause(fromClause parser.IFrom_clauseContext) (base.TableSource, error) {
	contents := fromClause.From_clause_contents()
	tableSource, err := q.extractTableSourceFromTablePrimary(contents.Table_primary())
	if err != nil {
		return nil, err
	}
	return tableSource, nil
	// TODO(zp): handle suffix, alias.
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
					tableName = unquoteIdentifierByRule(allIdentifiers[len(allIdentifiers)-1])
					if len(allIdentifiers) > 1 {
						datasetName = unquoteIdentifierByRule(allIdentifiers[len(allIdentifiers)-2])
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
	tableName := unquoteIdentifierByRule(lastIdentifier)
	columnSource.Table = tableName

	if len(allIdentifiers) >= 2 {
		identifier := unquoteIdentifierByRule(allIdentifiers[len(allIdentifiers)-2])
		if strings.EqualFold(identifier, "INFORMATION_SCHEMA") {
			columnSource.Schema = identifier
		} else {
			columnSource.Database = identifier
		}
	}

	l.sourceColumnSet[columnSource] = true
}

func unquoteIdentifierByRule(identifier parser.IIdentifierContext) string {
	if len(identifier.GetText()) >= 3 && strings.HasPrefix(identifier.GetText(), "`") && strings.HasSuffix(identifier.GetText(), "`") {
		return identifier.GetText()[1 : len(identifier.GetText())-1]
	}
	return identifier.GetText()
}

func unquoteIdentifierByText(identifier string) string {
	if len(identifier) >= 3 && strings.HasPrefix(identifier, "`") && strings.HasSuffix(identifier, "`") {
		return identifier[1 : len(identifier)-1]
	}
	return identifier
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
	return strings.EqualFold(resource.Schema, "INFORMATION_SCHEMA")
}
