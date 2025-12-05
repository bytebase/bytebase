package bigquery

import (
	"context"
	"slices"
	"strings"

	"github.com/antlr4-go/antlr/v4"
	parser "github.com/bytebase/parser/googlesql"

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
	if len(parseResults) == 0 {
		return &base.QuerySpan{
			SourceColumns: base.SourceColumnSet{},
			Results:       []base.QuerySpanResult{},
		}, nil
	}
	if len(parseResults) != 1 {
		return nil, errors.Errorf("expecting only one statement to get query span, but got %d", len(parseResults))
	}
	tree := parseResults[0].Tree
	q.ctx = ctx
	accessTables, err := getAccessTables(q.defaultDatabase, tree)
	if err != nil {
		return nil, err
	}
	allSystems, mixed := isMixedQuery(accessTables)
	if mixed {
		return nil, base.MixUserSystemTablesError
	}

	queryTypeListener := &queryTypeListener{
		allSystems: allSystems,
		result:     base.QueryTypeUnknown,
	}
	antlr.ParseTreeWalkerDefault.Walk(queryTypeListener, tree)
	if queryTypeListener.err != nil {
		return nil, queryTypeListener.err
	}
	if queryTypeListener.result != base.Select {
		return &base.QuerySpan{
			Type:          queryTypeListener.result,
			SourceColumns: base.SourceColumnSet{},
			Results:       []base.QuerySpanResult{},
		}, nil
	}

	// TODO(zp): statement type check.
	querySpanResult, err := q.getQuerySpanResult(tree.(*parser.RootContext).Stmts().AllUnterminated_sql_statement()[0].Sql_statement_body())
	if err != nil {
		return nil, err
	}

	return &base.QuerySpan{
		Type:          base.Select,
		Results:       querySpanResult,
		SourceColumns: accessTables,
	}, nil
}

func (q *querySpanExtractor) getQuerySpanResult(tree parser.ISql_statement_bodyContext) ([]base.QuerySpanResult, error) {
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
	originalCTELength := len(q.ctes)
	defer func() {
		q.ctes = q.ctes[:originalCTELength]
	}()
	if queryWithoutPipe.With_clause() != nil {
		if err := q.recordCTE(queryWithoutPipe.With_clause()); err != nil {
			return nil, err
		}
	}
	return q.extractTableSourceFromQueryPrimaryOrSetOperation(queryWithoutPipe.Query_primary_or_set_operation())
}

func (q *querySpanExtractor) recordRecursiveCTE(aliasedQuery parser.IAliased_queryContext) error {
	cteName := unquoteIdentifierByRule(aliasedQuery.Identifier())
	query := aliasedQuery.Parenthesized_query().Query().Query_without_pipe_operators()
	if query.With_clause() != nil {
		return errors.Errorf("WITH is not allowed inside WITH RECURSIVE")
	}
	if query.Query_primary_or_set_operation().Query_set_operation() == nil {
		return q.recordNonRecursiveCTE(aliasedQuery)
	}
	prefix := query.Query_primary_or_set_operation().Query_set_operation().Query_set_operation_prefix()
	anchor, err := q.extractTableSourceFromQueryPrimary(prefix.Query_primary())
	if err != nil {
		return err
	}
	// XXX(zp): How about two union?
	recursiveItem := prefix.AllQuery_set_operation_item()[0]
	tempCte := &base.PseudoTable{
		Name:    cteName,
		Columns: anchor.GetQuerySpanResult(),
	}
	q.ctes = append(q.ctes, tempCte)
	originalSize := len(q.ctes)
	for {
		originalSize := len(q.ctes)
		recursivePartTableSource, err := q.extractTableSourceFromQueryPrimary(recursiveItem.Query_primary())
		if err != nil {
			return err
		}
		anchorQuerySpanResults := q.ctes[originalSize-1].GetQuerySpanResult()
		recursivePartQuerySpanResults := recursivePartTableSource.GetQuerySpanResult()
		if len(anchorQuerySpanResults) != len(recursivePartQuerySpanResults) {
			return errors.Errorf("recursive cte %s clause returns %d fields, but anchor clause returns %d fields", cteName, len(anchorQuerySpanResults), len(recursivePartQuerySpanResults))
		}
		changed := false
		for i := range anchorQuerySpanResults {
			var hasChange bool
			anchorQuerySpanResults[i].SourceColumns, hasChange = base.MergeSourceColumnSet(anchorQuerySpanResults[i].SourceColumns, recursivePartQuerySpanResults[i].SourceColumns)
			changed = changed || hasChange
		}
		tempCte := &base.PseudoTable{
			Name:    cteName,
			Columns: anchorQuerySpanResults,
		}
		q.ctes = q.ctes[:originalSize-1]
		if !changed {
			break
		}
		q.ctes = append(q.ctes, tempCte)
	}
	q.ctes = q.ctes[:originalSize-1]
	q.ctes = append(q.ctes, tempCte)
	return nil
}

func (q *querySpanExtractor) recordNonRecursiveCTE(aliasedQuery parser.IAliased_queryContext) error {
	cteName := unquoteIdentifierByRule(aliasedQuery.Identifier())
	query := aliasedQuery.Parenthesized_query().Query()
	tableSource, err := q.extractNormalCTE(query)
	if err != nil {
		return err
	}
	q.ctes = append(q.ctes, &base.PseudoTable{
		Name:    cteName,
		Columns: tableSource.GetQuerySpanResult(),
	})
	return nil
}

func (q *querySpanExtractor) recordCTE(withClause parser.IWith_clauseContext) error {
	allAliasedQuery := withClause.AllAliased_query()
	recursive := withClause.RECURSIVE_SYMBOL() != nil
	for _, aliasedQuery := range allAliasedQuery {
		// TODO(zp): Actually, BigQuery do not rely on the RECURSIVE keyword, instead, it detects the recursive CTE
		// by the reference of the CTE itself in the CTE body. Also, check other engines.
		if recursive {
			if err := q.recordRecursiveCTE(aliasedQuery); err != nil {
				return err
			}
		} else {
			if err := q.recordNonRecursiveCTE(aliasedQuery); err != nil {
				return err
			}
		}
	}
	return nil
}

func (q *querySpanExtractor) extractNormalCTE(query parser.IQueryContext) (base.TableSource, error) {
	querySpanExtractor := &querySpanExtractor{
		ctx:             q.ctx,
		defaultDatabase: q.defaultDatabase,
		gCtx:            q.gCtx,
		ctes:            q.ctes,
	}
	tableSource, err := querySpanExtractor.extractTableSourceFromQuery(query)
	if err != nil {
		return nil, err
	}
	return tableSource, nil
}

func (q *querySpanExtractor) extractTableSourceFromQueryPrimaryOrSetOperation(queryPrimaryOrSetOperation parser.IQuery_primary_or_set_operationContext) (base.TableSource, error) {
	if queryPrimaryOrSetOperation.Query_primary() != nil {
		return q.extractTableSourceFromQueryPrimary(queryPrimaryOrSetOperation.Query_primary())
	}
	if queryPrimaryOrSetOperation.Query_set_operation() != nil {
		return q.extractTableSourceFromQuerySetOperation(queryPrimaryOrSetOperation.Query_set_operation())
	}
	return nil, nil
}

func (q *querySpanExtractor) extractTableSourceFromQuerySetOperation(querySetOperation parser.IQuery_set_operationContext) (base.TableSource, error) {
	tableSource, err := q.extractTableSourceFromQueryPrimary(querySetOperation.Query_set_operation_prefix().Query_primary())
	if err != nil {
		return nil, err
	}
	leftSpanResults := tableSource
	for _, item := range querySetOperation.Query_set_operation_prefix().AllQuery_set_operation_item() {
		newQ := &querySpanExtractor{
			ctx:             q.ctx,
			defaultDatabase: q.defaultDatabase,
			gCtx:            q.gCtx,
			ctes:            q.ctes,
		}
		rightSpanResults, err := newQ.extractTableSourceFromQueryPrimary(item.Query_primary())
		if err != nil {
			return nil, err
		}
		leftQuerySpanResult, rightQuerySpanResult := leftSpanResults.GetQuerySpanResult(), rightSpanResults.GetQuerySpanResult()
		if len(leftQuerySpanResult) != len(rightQuerySpanResult) {
			return nil, errors.Errorf("left select has %d columns, but right select has %d columns", len(leftQuerySpanResult), len(rightQuerySpanResult))
		}
		var result []base.QuerySpanResult
		for i, leftSpanResult := range leftQuerySpanResult {
			rightSpanResult := rightQuerySpanResult[i]
			newResourceColumns, _ := base.MergeSourceColumnSet(leftSpanResult.SourceColumns, rightSpanResult.SourceColumns)
			result = append(result, base.QuerySpanResult{
				Name:          leftSpanResult.Name,
				SourceColumns: newResourceColumns,
			})
		}
		leftSpanResults = &base.PseudoTable{
			Name:    "",
			Columns: result,
		}
		// FIXME(zp): Consider UNION alias.
	}
	return leftSpanResults, nil
}

func (q *querySpanExtractor) extractTableSourceFromQueryPrimary(queryPrimary parser.IQuery_primaryContext) (base.TableSource, error) {
	if queryPrimary.Select_() != nil {
		return q.extractTableSourceFromSelect(queryPrimary.Select_())
	}
	if parenthesizedQuery := queryPrimary.Parenthesized_query(); parenthesizedQuery != nil {
		// Table subquery shares the ctes of outer query. On the contrary,
		// the subquery should not effect the outer query.
		// https://cloud.google.com/bigquery/docs/reference/standard-sql/subqueries#correlated_subquery_concepts
		originalCtesLength := len(q.ctes)
		originalTableSourceFrom := len(q.tableSourceFrom)
		defer func() {
			q.ctes = q.ctes[:originalCtesLength]
			q.tableSourceFrom = q.tableSourceFrom[:originalTableSourceFrom]
		}()
		subqueryExtractor := &querySpanExtractor{
			ctx:               q.ctx,
			gCtx:              q.gCtx,
			defaultDatabase:   q.defaultDatabase,
			ctes:              q.ctes,
			outerTableSources: q.outerTableSources,
			tableSourceFrom:   q.tableSourceFrom,
		}
		tableSource, err := subqueryExtractor.extractTableSourceFromParenthesizedQuery(parenthesizedQuery)
		if err != nil {
			return nil, err
		}

		var alias string
		if v := queryPrimary.Opt_as_alias_with_required_as(); v != nil {
			if v.Identifier() != nil {
				alias = unquoteIdentifierByRule(v.Identifier())
			}
		}
		if alias != "" {
			return &base.PseudoTable{
				Name:    alias,
				Columns: tableSource.GetQuerySpanResult(),
			}, nil
		}

		return tableSource, nil
	}
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
		switch {
		case item.Select_column_star() != nil:
			fields := append([]base.QuerySpanResult{}, fromFields...)
			fields, err := q.starModify(fields, item.Select_column_star().Star_modifiers())
			if err != nil {
				return nil, err
			}
			resultFields = append(resultFields, fields...)
		case item.Select_column_dot_star() != nil:
			v := item.Select_column_dot_star()
			wildFields, err := q.extractWildFromExpr(v.Expression_higher_prec_than_and())
			if err != nil {
				return nil, err
			}
			fields, err := q.starModify(wildFields, item.Select_column_dot_star().Star_modifiers())
			if err != nil {
				return nil, err
			}
			resultFields = append(resultFields, fields...)
		case item.Select_column_expr() != nil:
			v := item.Select_column_expr()
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
		default:
			// Skip unrecognized select list item types
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

func (q *querySpanExtractor) starModify(fields []base.QuerySpanResult, starModifier parser.IStar_modifiersContext) ([]base.QuerySpanResult, error) {
	if starModifier == nil {
		return fields, nil
	}

	type fieldItem struct {
		id    int
		field base.QuerySpanResult
	}
	var fieldItems []fieldItem
	for i, field := range fields {
		fieldItems = append(fieldItems, fieldItem{
			id:    i,
			field: field,
		})
	}
	fieldItemMap := make(map[string]fieldItem)
	for _, fieldItem := range fieldItems {
		fieldItemMap[fieldItem.field.Name] = fieldItem
	}

	if except := starModifier.Star_except_list(); except != nil {
		allIdentifiers := except.AllIdentifier()
		for _, identifier := range allIdentifiers {
			identifierNormalized := unquoteIdentifierByRule(identifier)
			if _, ok := fieldItemMap[identifierNormalized]; !ok {
				return nil, errors.Errorf("field %s does not exist in the select clause", identifierNormalized)
			}
			delete(fieldItemMap, identifierNormalized)
		}
	}

	if replace := starModifier.Star_replace_list(); replace != nil {
		allReplaceItems := replace.AllStar_replace_item()
		for _, replaceItem := range allReplaceItems {
			_, set, err := q.extractSourceColumnSetFromExpr(replaceItem.Expression())
			if err != nil {
				return nil, err
			}
			asIdentifier := unquoteIdentifierByRule(replaceItem.Identifier())
			querySpanResult := base.QuerySpanResult{
				Name:          asIdentifier,
				SourceColumns: set,
			}
			if _, ok := fieldItemMap[asIdentifier]; !ok {
				return nil, errors.Errorf("field %s does not exist in the select clause", asIdentifier)
			}
			fieldItemMap[asIdentifier] = fieldItem{
				id:    fieldItemMap[asIdentifier].id,
				field: querySpanResult,
			}
		}
	}

	fieldItems = nil
	for _, fieldItem := range fieldItemMap {
		fieldItems = append(fieldItems, fieldItem)
	}
	slices.SortFunc(fieldItems, func(i, j fieldItem) int {
		if i.id < j.id {
			return -1
		}
		if i.id > j.id {
			return 1
		}
		return 0
	})

	var result []base.QuerySpanResult
	for _, fieldItem := range fieldItems {
		result = append(result, fieldItem.field)
	}
	return result, nil
}

func (q *querySpanExtractor) extractWildFromExpr(ctx antlr.ParserRuleContext) ([]base.QuerySpanResult, error) {
	if ctx == nil {
		return []base.QuerySpanResult{}, nil
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
	switch ctx := ctx.(type) {
	case *parser.Parenthesized_queryContext:
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
			return nil, err
		}
		spanResult := tableSource.GetQuerySpanResult()
		return spanResult, nil
	case *parser.Expression_higher_prec_than_andContext:
		possibleColumnResources := getPossibleColumnResources(ctx)
		for _, columnResources := range possibleColumnResources {
			l := len(columnResources)
			if l == 1 {
				// a.*, a can be the table name or the field name.
				results, ok := q.getAllTableColumnSources("", columnResources[0])
				if !ok {
					results, err := q.getFieldColumnSource("", "", columnResources[0])
					if err != nil {
						return nil, err
					}
					return []base.QuerySpanResult{
						{
							Name:          columnResources[0],
							SourceColumns: results,
						},
					}, nil
				}
				return results, nil
			}
			if l >= 2 {
				// a.b.*, can be resolved as
				// 1. a is the table name, b is the field name.
				// 2. a is the field name, b is the field name.
				results, err := q.getFieldColumnSource("", columnResources[0], columnResources[1])
				if err != nil {
					results, err := q.getFieldColumnSource("", "", columnResources[0])
					if err != nil {
						return nil, err
					}
					return []base.QuerySpanResult{
						{
							Name:          columnResources[0],
							SourceColumns: results,
						},
					}, nil
				}
				return []base.QuerySpanResult{
					{
						Name:          columnResources[1],
						SourceColumns: results,
					},
				}, nil
			}
		}
		return nil, nil
	default:
		return nil, errors.Errorf("unsupported type in wild expr: %T", ctx)
	}
}

func (q *querySpanExtractor) getAllTableColumnSources(datasetName, tableName string) ([]base.QuerySpanResult, bool) {
	findInTableSource := func(tableSource base.TableSource) ([]base.QuerySpanResult, bool) {
		if datasetName != "" && !strings.EqualFold(datasetName, tableSource.GetDatabaseName()) {
			return nil, false
		}
		if tableName != "" && !strings.EqualFold(tableName, tableSource.GetTableName()) {
			return nil, false
		}

		// If the table name is empty, we should check if there are ambiguous fields,
		// but we delegate this responsibility to the db-server, we do the fail-open strategy here.

		return tableSource.GetQuerySpanResult(), true
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
		if querySpanResult, ok := findInTableSource(q.outerTableSources[i]); ok {
			return querySpanResult, true
		}
	}

	for i := len(q.tableSourceFrom) - 1; i >= 0; i-- {
		if querySpanResult, ok := findInTableSource(q.tableSourceFrom[i]); ok {
			return querySpanResult, true
		}
	}

	return nil, false
}

func (q *querySpanExtractor) getFieldColumnSource(_, tableName, fieldName string) (base.SourceColumnSet, error) {
	// Bigquery column name is case-insensitive.
	findInTableSource := func(tableSource base.TableSource) (base.SourceColumnSet, bool) {
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

	return nil, &base.ResourceNotFoundError{
		Database: nil,
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

type joinType int

const (
	crossJoin = iota
	innerJoin
	fullOuterJoin
	leftOuterJoin
	rightOuterJoin
)

func (q *querySpanExtractor) extractTableSourceFromFromClause(fromClause parser.IFrom_clauseContext) (base.TableSource, error) {
	contents := fromClause.From_clause_contents()
	tableSource, err := q.extractTableSourceFromTablePrimary(contents.Table_primary())
	q.tableSourceFrom = append(q.tableSourceFrom, tableSource)
	if err != nil {
		return nil, err
	}
	var anchor base.TableSource
	anchor = &base.PseudoTable{
		Name:    "",
		Columns: tableSource.GetQuerySpanResult(),
	}
	allSuffixes := contents.AllFrom_clause_contents_suffix()
	for _, suffix := range allSuffixes {
		joinType := getJoinTypeFromFromClauseContentsSuffix(suffix)
		tableSource, err = q.extractTableSourceFromTablePrimary(suffix.Table_primary())
		if err != nil {
			return nil, err
		}
		q.tableSourceFrom = append(q.tableSourceFrom, tableSource)
		var usingColumns []string
		if suffix.On_or_using_clause_list() != nil {
			allJoinOnOrUsingClause := suffix.On_or_using_clause_list().AllOn_or_using_clause()
			for _, joinOnOrUsingClause := range allJoinOnOrUsingClause {
				if usingClause := joinOnOrUsingClause.Using_clause(); usingClause != nil {
					identifiers := usingClause.AllIdentifier()
					for _, identifier := range identifiers {
						usingColumns = append(usingColumns, unquoteIdentifierByRule(identifier))
					}
				}
			}
		}
		anchor = joinTable(anchor, joinType, usingColumns, tableSource)
	}
	q.tableSourceFrom = append(q.tableSourceFrom, anchor)
	return anchor, nil
}

func joinTable(anchor base.TableSource, tp joinType, usingColumns []string, tableSource base.TableSource) base.TableSource {
	var resultField []base.QuerySpanResult
	switch tp {
	case crossJoin, innerJoin, fullOuterJoin, leftOuterJoin, rightOuterJoin:
		using := make(map[string]bool)
		for _, usingColumn := range usingColumns {
			using[usingColumn] = true
		}
		var lFields []base.QuerySpanResult
		var rFields []base.QuerySpanResult
		usingMerge := make(map[string][]base.QuerySpanResult)
		for _, rField := range tableSource.GetQuerySpanResult() {
			if _, ok := using[strings.ToUpper(rField.Name)]; ok {
				usingMerge[strings.ToUpper(rField.Name)] = append(usingMerge[strings.ToUpper(rField.Name)], rField)
			} else {
				rFields = append(rFields, rField)
			}
		}
		lFields = append(lFields, anchor.GetQuerySpanResult()...)

		for _, lField := range lFields {
			columnSet := lField.SourceColumns
			if _, ok := using[strings.ToUpper(lField.Name)]; ok {
				mergeItems := usingMerge[strings.ToUpper(lField.Name)]
				for _, mergeItem := range mergeItems {
					columnSet, _ = base.MergeSourceColumnSet(columnSet, mergeItem.SourceColumns)
				}
			}
			resultField = append(resultField, base.QuerySpanResult{
				Name:          lField.Name,
				SourceColumns: columnSet,
			})
		}

		resultField = append(resultField, rFields...)
	default:
		// For unhandled join types, combine all fields from both sources
		resultField = append(resultField, anchor.GetQuerySpanResult()...)
		resultField = append(resultField, tableSource.GetQuerySpanResult()...)
	}
	return &base.PseudoTable{
		Name:    "",
		Columns: resultField,
	}
}

func getJoinTypeFromJoinType(joinType parser.IJoin_typeContext) joinType {
	if joinType == nil {
		return innerJoin
	}
	switch {
	case joinType.CROSS_SYMBOL() != nil:
		return crossJoin
	case joinType.INNER_SYMBOL() != nil:
		return innerJoin
	case joinType.LEFT_SYMBOL() != nil:
		return leftOuterJoin
	case joinType.RIGHT_SYMBOL() != nil:
		return rightOuterJoin
	default:
		return crossJoin
	}
}

func getJoinTypeFromJoinItem(joinItem parser.IJoin_itemContext) joinType {
	return getJoinTypeFromJoinType(joinItem.Join_type())
}

func getJoinTypeFromFromClauseContentsSuffix(fromClauseContentsSuffix parser.IFrom_clause_contents_suffixContext) joinType {
	if fromClauseContentsSuffix.COMMA_SYMBOL() != nil {
		return crossJoin
	}
	if fromClauseContentsSuffix.JOIN_SYMBOL() != nil {
		return getJoinTypeFromJoinType(fromClauseContentsSuffix.Join_type())
	}
	return crossJoin
}

func (q *querySpanExtractor) extractTableSourceFromTablePrimary(tablePrimary parser.ITable_primaryContext) (base.TableSource, error) {
	if tablePrimary.Tvf_with_suffixes() != nil {
		// We do not support table value function because we do not have the returnning columns information.
		return nil, errors.Errorf("unsupported table value function: %s", tablePrimary.GetText())
	}
	if tablePrimary.Table_path_expression() != nil {
		return q.extractTableSourceFromTablePathExpression(tablePrimary.Table_path_expression())
	}
	if subquery := tablePrimary.Table_subquery(); subquery != nil {
		return q.extractTableSourceFromTableSubquery(subquery)
	}
	if join := tablePrimary.Join(); join != nil {
		anchor, err := q.extractTableSourceFromTablePrimary(join.Table_primary())
		if err != nil {
			return nil, err
		}
		q.tableSourceFrom = append(q.tableSourceFrom, anchor)
		for _, item := range join.AllJoin_item() {
			joinType := getJoinTypeFromJoinItem(item)
			tableSource, err := q.extractTableSourceFromTablePrimary(item.Table_primary())
			if err != nil {
				return nil, err
			}
			q.tableSourceFrom = append(q.tableSourceFrom, tableSource)
			var usingColumns []string
			if item.On_or_using_clause_list() != nil {
				allJoinOnOrUsingClause := item.On_or_using_clause_list().AllOn_or_using_clause()
				for _, joinOnOrUsingClause := range allJoinOnOrUsingClause {
					if usingClause := joinOnOrUsingClause.Using_clause(); usingClause != nil {
						identifiers := usingClause.AllIdentifier()
						for _, identifier := range identifiers {
							usingColumns = append(usingColumns, unquoteIdentifierByRule(identifier))
						}
					}
				}
			}
			anchor = joinTable(anchor, joinType, usingColumns, tableSource)
		}
		return anchor, nil
	}
	if tablePrimary.Table_primary() != nil {
		return q.extractTableSourceFromTablePrimary(tablePrimary.Table_primary())
	}

	return nil, nil
}

func (q *querySpanExtractor) extractTableSourceFromTableSubquery(subquery parser.ITable_subqueryContext) (base.TableSource, error) {
	parenthesizedQuery := subquery.Parenthesized_query()

	// Table subquery shares the ctes of outer query. On the contrary,
	// the subquery should not effect the outer query.
	// https://cloud.google.com/bigquery/docs/reference/standard-sql/subqueries#correlated_subquery_concepts
	originalCtesLength := len(q.ctes)
	originalTableSourceFrom := len(q.tableSourceFrom)
	defer func() {
		q.ctes = q.ctes[:originalCtesLength]
		q.tableSourceFrom = q.tableSourceFrom[:originalTableSourceFrom]
	}()
	subqueryExtractor := &querySpanExtractor{
		ctx:               q.ctx,
		defaultDatabase:   q.defaultDatabase,
		gCtx:              q.gCtx,
		ctes:              q.ctes,
		outerTableSources: q.outerTableSources,
		tableSourceFrom:   q.tableSourceFrom,
	}
	tableSource, err := subqueryExtractor.extractTableSourceFromParenthesizedQuery(parenthesizedQuery)
	if err != nil {
		return nil, err
	}

	var alias string
	if v := subquery.Opt_pivot_or_unpivot_clause_and_alias(); v != nil {
		if v.Identifier() != nil {
			alias = unquoteIdentifierByRule(v.Identifier())
		}
	}
	if alias != "" {
		return &base.PseudoTable{
			Name:    alias,
			Columns: tableSource.GetQuerySpanResult(),
		}, nil
	}

	return tableSource, nil
}

func (q *querySpanExtractor) extractTableSourceFromTablePathExpression(tablePathExpression parser.ITable_path_expressionContext) (base.TableSource, error) {
	tablePathExprBase := tablePathExpression.Table_path_expression_base()
	if tablePathExprBase.Unnest_expression() != nil {
		// We do not support unnest expression because we do not have the returnning columns information.
		return nil, errors.Errorf("unsupported unnest expression: %s", tablePathExprBase.GetText())
	}
	var tableName string
	var datasetName string

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
	return tableSource, nil
}

func (q *querySpanExtractor) findTableSchema(datasetName string, tableName string) (base.TableSource, error) {
	// https://cloud.google.com/bigquery/docs/reference/standard-sql/lexical#case_sensitivity
	// Dataset and table names are case-sensitive unless the is_case_insensitive option is set to TRUE.
	if datasetName == "" {
		for i := len(q.ctes) - 1; i >= 0; i-- {
			table := q.ctes[i]
			if strings.EqualFold(table.Name, tableName) {
				return table, nil
			}
		}
	}

	if datasetName == "" {
		datasetName = q.defaultDatabase
	}

	_, databaseMetadata, err := q.gCtx.GetDatabaseMetadataFunc(q.ctx, q.gCtx.InstanceID, datasetName)
	if err != nil {
		return nil, err
	}
	if databaseMetadata == nil {
		return nil, errors.Errorf("dataset %q not found", datasetName)
	}

	schema := databaseMetadata.GetSchemaMetadata("")
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
	for _, column := range table.GetProto().GetColumns() {
		columns = append(columns, column.Name)
	}
	return &base.PhysicalTable{
		Server:   "",
		Database: datasetName,
		Schema:   "",
		Name:     table.GetProto().Name,
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
