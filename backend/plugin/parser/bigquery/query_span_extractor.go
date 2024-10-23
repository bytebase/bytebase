package bigquery

import (
	"context"
	"strings"

	"github.com/antlr4-go/antlr/v4"
	parser "github.com/bytebase/google-sql-parser"

	parsererror "github.com/bytebase/bytebase/backend/plugin/parser/errors"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

type querySpanExtractor struct {
	ctx             context.Context
	defaultDatabase string

	gCtx base.GetQuerySpanContext

	ctes []*base.PseudoTable

	// outerTableSources is the table sources from the outer query span.
	// it's used to resolve the column name in the correlated sub-query.
	outerTableSources []base.TableSource

	// tableSourceFrom is the table sources from the from clause.
	tableSourceFrom []base.TableSource
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
		if v := item.Select_column_expr(); v != nil {
			var expression parser.IExpressionContext
			if v.Expression() != nil {
				expression = v.Expression()
			} else if v.Select_column_expr_with_as_alias() != nil {
				expression = v.Select_column_expr_with_as_alias().Expression()
			}

			var alias parser.IIdentifierContext
			if v.Select_column_expr_with_as_alias() != nil {
				alias = v.Select_column_expr_with_as_alias().Identifier()
			} else if v.Identifier() != nil {
				alias = v.Identifier()
			}

			name, sourceColumnSet, err := q.extractSourceColumnSetFromExpr(expression)
			if err != nil {
				return nil, err
			}
			aliasName := strings.ToUpper(name)
			if alias != nil {
				aliasName = strings.ToUpper(unquoteIdentifierByRule(alias))
			}
			resultFields = append(resultFields, base.QuerySpanResult{
				Name:          aliasName,
				SourceColumns: sourceColumnSet,
			})
		}
	}
	return &base.PseudoTable{
		Name:    "",
		Columns: resultFields,
	}, nil
}

func (q *querySpanExtractor) extractTableSourceFromParenthesizedQuery(parenthesizedQuery parser.IParenthesized_queryContext) (base.TableSource, error) {
	return q.extractTableSourceFromQuery(parenthesizedQuery.Query())
}

func (q *querySpanExtractor) extractSourceColumnSetFromExpr(ctx antlr.ParserRuleContext) (string, base.SourceColumnSet, error) {
	if ctx == nil {
		return "", make(base.SourceColumnSet), nil
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
	var name string
	switch ctx := ctx.(type) {
	// TODO(zp): handle subquery
	case *parser.Parenthesized_queryContext:
		baseSet := make(base.SourceColumnSet)
		subqueryExtractor := &querySpanExtractor{
			ctx:               q.ctx,
			defaultDatabase:   q.defaultDatabase,
			gCtx:              q.gCtx,
			ctes:              q.ctes,
			outerTableSources: append(q.outerTableSources, q.tableSourceFrom...),
			tableSourceFrom:   []base.TableSource{},
		}
		tableSource, err := subqueryExtractor.extractTableSourceFromParenthesizedQuery(ctx)
		if err != nil {
			return "", nil, err
		}
		spanResult := tableSource.GetQuerySpanResult()
		for _, field := range spanResult {
			baseSet, _ = base.MergeSourceColumnSet(field.SourceColumns, field.SourceColumns)
		}
		return "", baseSet, nil
	case *parser.Expression_higher_prec_than_andContext:
		baseSet := make(base.SourceColumnSet)
		possibleColumnResources := getPossibleColumnResources(ctx)
		for _, columnResources := range possibleColumnResources {
			l := len(columnResources)
			if l == 1 {
				sourceColumnSet, err := q.getFieldColumnSource("", "", columnResources[0])
				if err != nil {
					return "", nil, err
				}
				baseSet, _ = base.MergeSourceColumnSet(baseSet, sourceColumnSet)
				name = columnResources[0]
			}
			if l >= 2 {
				// a.b, a can be the table name or the field name.
				sourceColumnSet, err := q.getFieldColumnSource("", "", columnResources[0])
				if err != nil {
					sourceColumnSet, err = q.getFieldColumnSource("", columnResources[0], columnResources[1])
					if err != nil {
						return "", nil, err
					}
					baseSet, _ = base.MergeSourceColumnSet(baseSet, sourceColumnSet)
					name = columnResources[1]
				} else {
					baseSet, _ = base.MergeSourceColumnSet(baseSet, sourceColumnSet)
					name = columnResources[0]
				}
			}
		}
		return name, baseSet, nil
	default:
	}

	baseSet := make(base.SourceColumnSet)
	children := ctx.GetChildren()
	for _, child := range children {
		child, ok := child.(antlr.ParserRuleContext)
		if !ok {
			continue
		}
		fieldName, sourceColumnSet, err := q.extractSourceColumnSetFromExpr(child)
		if err != nil {
			return "", nil, err
		}
		name = fieldName
		baseSet, _ = base.MergeSourceColumnSet(baseSet, sourceColumnSet)
	}
	if len(children) > 1 {
		name = ""
	}

	return name, baseSet, nil
}

func (q *querySpanExtractor) getFieldColumnSource(databaseName, tableName, fieldName string) (base.SourceColumnSet, error) {
	// Bigquery column name is case-insensitive.
	findInTableSource := func(tableSource base.TableSource) (base.SourceColumnSet, bool) {
		if databaseName != "" && !strings.EqualFold(databaseName, tableSource.GetDatabaseName()) {
			return nil, false
		}
		if tableName != "" && !strings.EqualFold(tableName, tableSource.GetTableName()) {
			return nil, false
		}
		// If the table name is empty, we should check if there are ambiguous fields,
		// but we delegate this responsibility to the db-server, we do the fail-open strategy here.

		querySpanResult := tableSource.GetQuerySpanResult()
		for _, column := range querySpanResult {
			if strings.EqualFold(column.Name, fieldName) {
				return column.SourceColumns, true
			}
		}
		return nil, false
	}

	// One sub-query may have multi-outer schemas and the multi-outer schemas can use the same name, such as:
	//
	//  select (
	//    select (
	//      select max(a) > x1.a from t
	//    )
	//    from t1 as x1
	//    limit 1
	//  )
	//  from t as x1;
	//
	// This query has two tables can be called `x1`, and the expression x1.a uses the closer x1 table.
	// This is the reason we loop the slice in reversed order.
	for i := len(q.outerTableSources) - 1; i >= 0; i-- {
		if sourceColumnSet, ok := findInTableSource(q.outerTableSources[i]); ok {
			return sourceColumnSet, nil
		}
	}

	for i := len(q.tableSourceFrom) - 1; i >= 0; i-- {
		if sourceColumnSet, ok := findInTableSource(q.tableSourceFrom[i]); ok {
			return sourceColumnSet, nil
		}
	}

	return nil, &parsererror.ResourceNotFoundError{
		Database: &databaseName,
		Table:    &tableName,
		Column:   &fieldName,
	}
}

func isValidExpressionHigherPrecThanAnd(ctx antlr.ParserRuleContext) bool {
	if _, ok := ctx.(*parser.Expression_higher_prec_than_andContext); !ok {
		return false
	}

	terminalNodes := getAllChildTerminalNode(ctx)
	if len(terminalNodes) == 1 && terminalNodes[0].GetSymbol().GetTokenType() == parser.GoogleSQLParserIDENTIFIER {
		return true
	}
	child := ctx.GetChildren()
	if len(child) == 3 {
		first := child[0]
		_, ok := first.(*parser.Expression_higher_prec_than_andContext)
		if !ok {
			return false
		}
		if !isValidExpressionHigherPrecThanAnd(first.(*parser.Expression_higher_prec_than_andContext)) {
			return false
		}
		second := child[1]
		_, ok = second.(*antlr.TerminalNodeImpl)
		if !ok {
			return false
		}
		if second.(*antlr.TerminalNodeImpl).GetSymbol().GetTokenType() != parser.GoogleSQLParserDOT_SYMBOL {
			return false
		}
		third := child[2]
		_, ok = third.(*parser.IdentifierContext)
		return ok
	}
	return false
}

func getPossibleColumnResources(ctx antlr.ParserRuleContext) [][]string {
	// Traverse the tree to find the tallest subtree which matches the pattern:
	// [expression_higher_prec_than_and DOT_SYMBOL identifier]
	// The result is the possible column resources.
	var path []antlr.Tree
	path = append(path, ctx)

	var result [][]string
	for len(path) > 0 {
		element := path[len(path)-1]
		path = path[:len(path)-1]
		if _, ok := element.(*parser.Parenthesized_queryContext); ok {
			// NOTE: while adding the case in extractSourceColumnSetFromExpr, we should skip the case here.
			continue
		}
		appendChild := true
		if element, ok := element.(antlr.ParserRuleContext); ok {
			valid := isValidExpressionHigherPrecThanAnd(element)
			if valid {
				appendChild = false
				allTerminalNodes := getAllChildTerminalNode(element)
				var columnResources []string
				for _, terminalNode := range allTerminalNodes {
					if terminalNode.GetSymbol().GetTokenType() == parser.GoogleSQLParserIDENTIFIER {
						columnResources = append(columnResources, unquoteIdentifierByText(terminalNode.GetText()))
					}
				}
				result = append(result, columnResources)
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
	q.tableSourceFrom = append(q.tableSourceFrom, tableSource)
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
	tablePathExprBase := tablePathExpression.Table_path_expression_base()
	if tablePathExprBase.Unnest_expression() != nil {
		// We do not support unnest expression because we do not have the returnning columns information.
		return nil, errors.Errorf("unsupported unnest expression: %s", tablePathExprBase.GetText())
	}
	var tableName string
	datasetName := q.defaultDatabase

	if slashedOrDashedPathExpression := tablePathExprBase.Maybe_slashed_or_dashed_path_expression(); slashedOrDashedPathExpression != nil {
		if slashedOrDashedPathExpression.Maybe_dashed_path_expression() != nil {
			if maybeDashedPathExpr := slashedOrDashedPathExpression.Maybe_dashed_path_expression(); maybeDashedPathExpr != nil {
				// TODO(zp): support dashed path expression, for example, REGION-us
				if maybeDashedPathExpr.Dashed_path_expression() != nil {
					return nil, errors.Errorf("unsupported dashed path expression: %s", tablePathExprBase.GetText())
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
				return nil, errors.Errorf("unsupported slashed path expression: %s", tablePathExprBase.GetText())
			}
		}
	}

	tableSource, err := q.findTableSchema(datasetName, tableName)
	if err != nil {
		return nil, err
	}

	if o := tablePathExpression.Opt_pivot_or_unpivot_clause_and_alias(); o != nil {
		if o.Identifier() != nil {
			tableSource = &base.PseudoTable{
				Name:    unquoteIdentifierByRule(o.Identifier()),
				Columns: tableSource.GetQuerySpanResult(),
			}
		}
	}
	// TODO(zp): add in q.from
	return tableSource, nil
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

	tables := schema.ListTableNames()
	var originalTableName string
	for _, t := range tables {
		if strings.EqualFold(t, tableName) {
			originalTableName = t
			break
		}
	}
	if originalTableName == "" {
		return nil, errors.Errorf("table %q not found", tableName)
	}
	table := schema.GetTable(originalTableName)
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
