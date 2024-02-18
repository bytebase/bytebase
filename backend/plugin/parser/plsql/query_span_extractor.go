package plsql

import (
	"context"

	"github.com/antlr4-go/antlr/v4"
	"github.com/pkg/errors"

	plsql "github.com/bytebase/plsql-parser"

	parsererror "github.com/bytebase/bytebase/backend/plugin/parser/errors"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/store/model"
)

type querySpanExtractor struct {
	ctx               context.Context
	connectedDatabase string
	defaultSchema     string
	metaCache         map[string]*model.DatabaseMetadata
	f                 base.GetDatabaseMetadataFunc

	ctes []*base.PseudoTable

	outerTableSources []base.TableSource
	tableSourcesFrom  []base.TableSource
}

func newQuerySpanExtractor(connectionDatabase string, defaultSchema string, f base.GetDatabaseMetadataFunc) *querySpanExtractor {
	return &querySpanExtractor{
		connectedDatabase: connectionDatabase,
		defaultSchema:     defaultSchema,
		metaCache:         make(map[string]*model.DatabaseMetadata),
		f:                 f,
	}
}

func (q *querySpanExtractor) getDatabaseMetadata(schema string) (string, *model.DatabaseMetadata, error) {
	// There are two models for the database metadata, one is schema based, the other is database based.
	// We deal with two models in f, so we use schema name here.
	// The f will return the real database name and the metadata.
	// We just return them to the caller.
	if meta, ok := q.metaCache[schema]; ok {
		return schema, meta, nil
	}
	databaseName, meta, err := q.f(q.ctx, schema)
	if err != nil {
		return "", nil, errors.Wrapf(err, "failed to get database metadata for schema: %s", schema)
	}
	q.metaCache[databaseName] = meta
	return databaseName, meta, nil
}

func (q *querySpanExtractor) getQuerySpan(ctx context.Context, statement string) (*base.QuerySpan, error) {
	q.ctx = ctx

	// TODO: Implement the logic to extract access tables.

	// TODO: Implement the logic to extract system tables.

	tree, _, err := ParsePLSQL(statement)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to parse statement: %s", statement)
	}
	if tree == nil {
		return nil, nil
	}

	listener := &selectListener{
		q: q,
	}
	antlr.ParseTreeWalkerDefault.Walk(listener, tree)
	if listener.err != nil {
		return nil, errors.Wrapf(listener.err, "failed to extract query span from statement: %s", statement)
	}
	resources, err := ExtractResourceList(q.connectedDatabase, q.defaultSchema, statement)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to extract resource list from statement: %s", statement)
	}
	columnSet := make(base.SourceColumnSet)
	for _, resource := range resources {
		if !q.existsTableMetadata(resource) {
			continue
		}
		columnSet[base.ColumnResource{
			Server:   resource.LinkedServer,
			Database: resource.Database,
			Schema:   resource.Schema,
			Table:    resource.Table,
		}] = true
	}
	result := base.QuerySpan{
		SourceColumns: columnSet,
	}
	if listener.result != nil {
		result.Results = listener.result.Results
	}
	return &result, nil
}

func (q *querySpanExtractor) existsTableMetadata(resource base.SchemaResource) bool {
	if resource.Table == "DUAL" {
		return false
	}
	if resource.Schema == "" {
		resource.Schema = q.defaultSchema
	}
	_, meta, err := q.getDatabaseMetadata(resource.Schema)
	if err != nil {
		return false
	}
	if meta == nil {
		return false
	}
	schema := meta.GetSchema(resource.Schema)
	if schema == nil {
		return false
	}

	return schema.GetTable(resource.Table) != nil ||
		schema.GetView(resource.Table) != nil ||
		schema.GetMaterializedView(resource.Table) != nil ||
		schema.GetExternalTable(resource.Table) != nil
}

type selectListener struct {
	*plsql.BasePlSqlParserListener

	q      *querySpanExtractor
	result *base.QuerySpan
	err    error
}

func (l *selectListener) EnterSelect_statement(ctx *plsql.Select_statementContext) {
	if l.err != nil {
		return // Skip if there is already an error.
	}
	parent := ctx.GetParent()
	if parent == nil {
		return
	}

	if _, ok := parent.(*plsql.Data_manipulation_language_statementsContext); ok {
		if _, ok := parent.GetParent().(*plsql.Unit_statementContext); ok {
			if l.result != nil {
				l.err = errors.New("multiple select statements")
				return
			}

			tableSource, err := l.q.plsqlExtractContext(ctx)
			if err != nil {
				l.err = err
				return
			}

			l.result = &base.QuerySpan{
				Results: tableSource.GetQuerySpanResult(),
			}
			return
		}
	}
}

func (q *querySpanExtractor) plsqlExtractContext(ctx antlr.ParserRuleContext) (base.TableSource, error) {
	if ctx == nil {
		return nil, nil
	}

	switch ctx := ctx.(type) {
	case *plsql.Select_statementContext:
		return q.plsqlExtractSelect(ctx)
	default:
		return nil, errors.Errorf("unsupported context type: %T", ctx)
	}
}

func (q *querySpanExtractor) plsqlExtractSelect(ctx plsql.ISelect_statementContext) (base.TableSource, error) {
	selectOnlyStatement := ctx.Select_only_statement()
	if selectOnlyStatement == nil {
		return nil, nil
	}

	return q.plsqlExtractSelectOnlyStatement(selectOnlyStatement)
}

func (q *querySpanExtractor) plsqlExtractSelectOnlyStatement(ctx plsql.ISelect_only_statementContext) (base.TableSource, error) {
	if ctx == nil {
		return nil, nil
	}

	subquery := ctx.Subquery()
	if subquery == nil {
		return nil, nil
	}

	return q.plsqlExtractSubquery(subquery)
}

func (q *querySpanExtractor) plsqlExtractSubquery(ctx plsql.ISubqueryContext) (base.TableSource, error) {
	if ctx == nil {
		return nil, nil
	}

	subqueryBasicElements := ctx.Subquery_basic_elements()
	if subqueryBasicElements == nil {
		return nil, nil
	}

	leftTableSource, err := q.plsqlExtractSubqueryBasicElements(subqueryBasicElements)
	if err != nil {
		return nil, err
	}

	if len(ctx.AllSubquery_operation_part()) == 0 {
		return leftTableSource, nil
	}

	leftQuerySpanResult := leftTableSource.GetQuerySpanResult()

	for _, subqueryOperationPart := range ctx.AllSubquery_operation_part() {
		rightTableSource, err := q.plsqlExtractSubqueryOperationPart(subqueryOperationPart)
		if err != nil {
			return nil, err
		}
		rightQuerySpanResult := rightTableSource.GetQuerySpanResult()
		if len(leftQuerySpanResult) != len(rightQuerySpanResult) {
			return nil, errors.Errorf("subquery operation part has different column count: %d vs %d", len(leftQuerySpanResult), len(rightQuerySpanResult))
		}
		var result []base.QuerySpanResult
		for i, leftSpanResult := range leftQuerySpanResult {
			rightSpanREsult := rightQuerySpanResult[i]
			newSourceColumns, _ := base.MergeSourceColumnSet(leftSpanResult.SourceColumns, rightSpanREsult.SourceColumns)
			result = append(result, base.QuerySpanResult{
				Name:          leftSpanResult.Name,
				SourceColumns: newSourceColumns,
			})
		}
		leftQuerySpanResult = result
	}
	return &base.PseudoTable{
		Name:    "",
		Columns: leftQuerySpanResult,
	}, nil
}

func (q *querySpanExtractor) plsqlExtractSubqueryBasicElements(ctx plsql.ISubquery_basic_elementsContext) (base.TableSource, error) {
	if ctx == nil {
		return nil, nil
	}

	if ctx.Query_block() != nil {
		return q.plsqlExtractQueryBlock(ctx.Query_block())
	}

	if ctx.Subquery() != nil {
		return q.plsqlExtractSubquery(ctx.Subquery())
	}

	return nil, nil
}

func (q *querySpanExtractor) plsqlExtractQueryBlock(ctx plsql.IQuery_blockContext) (base.TableSource, error) {
	if ctx == nil {
		return nil, nil
	}

	withClause := ctx.Subquery_factoring_clause()
	if withClause != nil {
		previousCteOuterLength := len(q.ctes)
		defer func() {
			q.ctes = q.ctes[:previousCteOuterLength]
		}()

		for _, cte := range withClause.AllFactoring_element() {
			cteTable, err := q.plsqlExtractFactoringElement(cte)
			if err != nil {
				return nil, err
			}
			q.ctes = append(q.ctes, cteTable)
		}
	}

	fromClause := ctx.From_clause()
	var fromTableSource []base.TableSource
	if fromClause != nil {
		tableSources, err := q.plsqlExtractFromClause(fromClause)
		if err != nil {
			return nil, err
		}
		q.tableSourcesFrom = append(q.tableSourcesFrom, tableSources...)
		fromTableSource = tableSources
	}
	defer func() {
		q.tableSourcesFrom = nil
	}()

	result := new(base.PseudoTable)

	// Extract select list.
	selectedList := ctx.Selected_list()
	if selectedList != nil {
		if selectedList.ASTERISK() != nil {
			var columns []base.QuerySpanResult
			for _, tableSource := range fromTableSource {
				columns = append(columns, tableSource.GetQuerySpanResult()...)
			}
			result.Columns = append(result.Columns, columns...)
			return result, nil
		}

		selectListElements := selectedList.AllSelect_list_elements()
		for _, element := range selectListElements {
			if element.ASTERISK() != nil {
				schemaName, tableName := normalizeTableViewName(q.defaultSchema, element.Tableview_name())
				find := false
				for _, tableSource := range fromTableSource {
					if (schemaName == "" || schemaName == tableSource.GetSchemaName()) &&
						tableName == tableSource.GetTableName() {
						find = true
						result.Columns = append(result.Columns, tableSource.GetQuerySpanResult()...)
						break
					}
				}
				if !find {
					sources, ok := q.getAllTableColumnSources(schemaName, tableName)
					if ok {
						result.Columns = append(result.Columns, sources...)
						find = true
					}
				}
				if !find {
					return nil, &parsererror.ResourceNotFoundError{
						Err:      errors.Errorf("failed to find table to calculate asterisk"),
						Database: &q.defaultSchema,
						Schema:   &schemaName,
						Table:    &tableName,
					}
				}
			} else {
				fieldName, sourceColumnSet, err := q.plsqlExtractSourceColumnSetFromExpression(element.Expression())
				if err != nil {
					return nil, err
				}
				if element.Column_alias() != nil {
					fieldName = normalizeColumnAlias(element.Column_alias())
				} else if fieldName == "" {
					fieldName = element.Expression().GetText()
				}
				result.Columns = append(result.Columns, base.QuerySpanResult{
					Name:          fieldName,
					SourceColumns: sourceColumnSet,
				})
			}
		}
	}

	return result, nil
}

func (q *querySpanExtractor) plsqlExtractSourceColumnSetFromExpression(ctx antlr.ParserRuleContext) (string, base.SourceColumnSet, error) {
	if ctx == nil {
		return "", nil, nil
	}

	switch rule := ctx.(type) {
	case plsql.IColumn_nameContext:
		schemaName, tableName, columnName, err := plsqlNormalizeColumnName("", rule)
		if err != nil {
			return "", nil, err
		}
		return columnName, q.getFieldColumnSource(schemaName, tableName, columnName), nil
	case plsql.IIdentifierContext:
		id := NormalizeIdentifierContext(rule)
		return id, q.getFieldColumnSource("", "", id), nil
	case plsql.IConstantContext:
		list := rule.AllQuoted_string()
		if len(list) == 1 && rule.DATE() == nil && rule.TIMESTAMP() == nil && rule.INTERVAL() == nil {
			// This case may be a column name...
			return q.plsqlExtractSourceColumnSetFromExpression(list[0])
		}
	case plsql.IQuoted_stringContext:
		if rule.Variable_name() != nil {
			return q.plsqlExtractSourceColumnSetFromExpression(rule.Variable_name())
		}
		return "", nil, nil
	case plsql.IVariable_nameContext:
		if rule.Bind_variable() != nil {
			// TODO: handle bind variable
			return "", nil, nil
		}
		var list []string
		for _, item := range rule.AllId_expression() {
			list = append(list, NormalizeIDExpression(item))
		}
		switch len(list) {
		case 1:
			return list[0], q.getFieldColumnSource("", "", list[0]), nil
		case 2:
			return list[1], q.getFieldColumnSource("", list[0], list[1]), nil
		case 3:
			return list[2], q.getFieldColumnSource(list[0], list[1], list[2]), nil
		default:
			return "", nil, nil
		}
	case plsql.IGeneral_elementContext:
		var list []antlr.ParserRuleContext
		for _, item := range rule.AllGeneral_element_part() {
			list = append(list, item)
		}
		return q.plsqlExtractSourceColumnSetFromExpressionList(list)
	case plsql.IGeneral_element_partContext:
		// This case is for functions, such as CONCAT(a, b)
		if rule.Function_argument() != nil {
			_, maskingLevel, err := q.plsqlExtractSourceColumnSetFromExpression(rule.Function_argument())
			return "", maskingLevel, err
		}

		// This case is for column names, such as root.a.b
		var list []string
		for _, item := range rule.AllId_expression() {
			list = append(list, NormalizeIDExpression(item))
		}
		switch len(list) {
		case 1:
			return list[0], q.getFieldColumnSource("", "", list[0]), nil
		case 2:
			return list[1], q.getFieldColumnSource("", list[0], list[1]), nil
		case 3:
			return list[2], q.getFieldColumnSource(list[0], list[1], list[2]), nil
		default:
			return "", nil, nil
		}
	case plsql.IExpressionContext:
		if rule.Logical_expression() != nil {
			return q.plsqlExtractSourceColumnSetFromExpression(rule.Logical_expression())
		}

		return q.plsqlExtractSourceColumnSetFromExpression(rule.Cursor_expression())
	case plsql.ICursor_expressionContext:
		return q.plsqlExtractSourceColumnSetFromExpression(rule.Subquery())
	case plsql.IQuery_blockContext:
		// For associated subquery, we should set the fromFieldList as the outerSchemaInfo.
		// So that the subquery can access the outer schema.
		// The reason for new q is that we still need the current fromFieldList, overriding it is not expected.
		subqueryExtractor := &querySpanExtractor{
			ctx:               q.ctx,
			defaultSchema:     q.defaultSchema,
			metaCache:         q.metaCache,
			f:                 q.f,
			outerTableSources: append(q.outerTableSources, q.tableSourcesFrom...),
			tableSourcesFrom:  []base.TableSource{},
		}
		tableSource, err := subqueryExtractor.plsqlExtractQueryBlock(rule)
		if err != nil {
			return "", nil, errors.Wrap(err, "failed to extract select only statement")
		}
		spanResult := tableSource.GetQuerySpanResult()

		sourceColumnSet := make(base.SourceColumnSet)

		for _, field := range spanResult {
			sourceColumnSet, _ = base.MergeSourceColumnSet(sourceColumnSet, field.SourceColumns)
		}
		return "", sourceColumnSet, nil
	case plsql.ISubqueryContext:
		// For associated subquery, we should set the fromFieldList as the outerSchemaInfo.
		// So that the subquery can access the outer schema.
		// The reason for new q is that we still need the current fromFieldList, overriding it is not expected.
		subqueryExtractor := &querySpanExtractor{
			ctx:               q.ctx,
			defaultSchema:     q.defaultSchema,
			metaCache:         q.metaCache,
			f:                 q.f,
			outerTableSources: append(q.outerTableSources, q.tableSourcesFrom...),
			tableSourcesFrom:  []base.TableSource{},
		}
		tableSource, err := subqueryExtractor.plsqlExtractSubquery(rule)
		if err != nil {
			return "", nil, errors.Wrap(err, "failed to extract select only statement")
		}
		spanResult := tableSource.GetQuerySpanResult()

		sourceColumnSet := make(base.SourceColumnSet)

		for _, field := range spanResult {
			sourceColumnSet, _ = base.MergeSourceColumnSet(sourceColumnSet, field.SourceColumns)
		}
		return "", sourceColumnSet, nil
	case plsql.ILogical_expressionContext:
		if rule.Unary_logical_expression() != nil {
			return q.plsqlExtractSourceColumnSetFromExpression(rule.Unary_logical_expression())
		}
		var list []antlr.ParserRuleContext
		for _, item := range rule.AllLogical_expression() {
			list = append(list, item)
		}
		return q.plsqlExtractSourceColumnSetFromExpressionList(list)
	case plsql.IUnary_logical_expressionContext:
		return q.plsqlExtractSourceColumnSetFromExpression(rule.Multiset_expression())
	case plsql.IMultiset_expressionContext:
		var list []antlr.ParserRuleContext
		if rule.Relational_expression() != nil {
			list = append(list, rule.Relational_expression())
		}
		if rule.Concatenation() != nil {
			list = append(list, rule.Concatenation())
		}
		return q.plsqlExtractSourceColumnSetFromExpressionList(list)
	case plsql.IRelational_expressionContext:
		var list []antlr.ParserRuleContext
		for _, item := range rule.AllRelational_expression() {
			list = append(list, item)
		}
		if rule.Compound_expression() != nil {
			list = append(list, rule.Compound_expression())
		}
		return q.plsqlExtractSourceColumnSetFromExpressionList(list)
	case plsql.ICompound_expressionContext:
		var list []antlr.ParserRuleContext
		for _, item := range rule.AllConcatenation() {
			list = append(list, item)
		}
		if rule.In_elements() != nil {
			list = append(list, rule.In_elements())
		}
		if rule.Between_elements() != nil {
			list = append(list, rule.Between_elements())
		}
		return q.plsqlExtractSourceColumnSetFromExpressionList(list)
	case plsql.IIn_elementsContext:
		var list []antlr.ParserRuleContext
		if rule.Subquery() != nil {
			list = append(list, rule.Subquery())
		}
		for _, item := range rule.AllConcatenation() {
			list = append(list, item)
		}
		return q.plsqlExtractSourceColumnSetFromExpressionList(list)
	case plsql.IBetween_elementsContext:
		var list []antlr.ParserRuleContext
		for _, item := range rule.AllConcatenation() {
			list = append(list, item)
		}
		return q.plsqlExtractSourceColumnSetFromExpressionList(list)
	case plsql.IConcatenationContext:
		var list []antlr.ParserRuleContext
		if rule.Model_expression() != nil {
			list = append(list, rule.Model_expression())
		}
		if rule.Interval_expression() != nil {
			list = append(list, rule.Interval_expression())
		}
		for _, item := range rule.AllConcatenation() {
			list = append(list, item)
		}
		return q.plsqlExtractSourceColumnSetFromExpressionList(list)
	case plsql.IModel_expressionContext:
		var list []antlr.ParserRuleContext
		if rule.Unary_expression() != nil {
			list = append(list, rule.Unary_expression())
		}
		if rule.Model_expression_element() != nil {
			list = append(list, rule.Model_expression_element())
		}
		return q.plsqlExtractSourceColumnSetFromExpressionList(list)
	case plsql.IInterval_expressionContext:
		var list []antlr.ParserRuleContext
		for _, item := range rule.AllConcatenation() {
			list = append(list, item)
		}
		return q.plsqlExtractSourceColumnSetFromExpressionList(list)
	case plsql.IUnary_expressionContext:
		var list []antlr.ParserRuleContext
		if rule.Unary_expression() != nil {
			list = append(list, rule.Unary_expression())
		}
		if rule.Case_statement() != nil {
			list = append(list, rule.Case_statement())
		}
		if rule.Quantified_expression() != nil {
			list = append(list, rule.Quantified_expression())
		}
		if rule.Standard_function() != nil {
			list = append(list, rule.Standard_function())
		}
		if rule.Atom() != nil {
			list = append(list, rule.Atom())
		}
		return q.plsqlExtractSourceColumnSetFromExpressionList(list)
	case plsql.ICase_statementContext:
		var list []antlr.ParserRuleContext
		if rule.Simple_case_statement() != nil {
			list = append(list, rule.Simple_case_statement())
		}
		if rule.Searched_case_statement() != nil {
			list = append(list, rule.Searched_case_statement())
		}
		return q.plsqlExtractSourceColumnSetFromExpressionList(list)
	case plsql.ISimple_case_statementContext:
		var list []antlr.ParserRuleContext
		if rule.Expression() != nil {
			list = append(list, rule.Expression())
		}
		for _, item := range rule.AllSimple_case_when_part() {
			list = append(list, item)
		}
		if rule.Case_else_part() != nil {
			list = append(list, rule.Case_else_part())
		}
		return q.plsqlExtractSourceColumnSetFromExpressionList(list)
	case plsql.ISimple_case_when_partContext:
		// not handle seq_of_statements
		var list []antlr.ParserRuleContext
		for _, item := range rule.AllExpression() {
			list = append(list, item)
		}
		return q.plsqlExtractSourceColumnSetFromExpressionList(list)
	case plsql.ICase_else_partContext:
		// not handle seq_of_statements
		return q.plsqlExtractSourceColumnSetFromExpressionList([]antlr.ParserRuleContext{rule.Expression()})
	case plsql.ISearched_case_statementContext:
		var list []antlr.ParserRuleContext
		for _, item := range rule.AllSearched_case_when_part() {
			list = append(list, item)
		}
		if rule.Case_else_part() != nil {
			list = append(list, rule.Case_else_part())
		}
		return q.plsqlExtractSourceColumnSetFromExpressionList(list)
	case plsql.ISearched_case_when_partContext:
		// not handle seq_of_statements
		var list []antlr.ParserRuleContext
		for _, item := range rule.AllExpression() {
			list = append(list, item)
		}
		return q.plsqlExtractSourceColumnSetFromExpressionList(list)
	case plsql.IQuantified_expressionContext:
		var list []antlr.ParserRuleContext
		if rule.Expression() != nil {
			list = append(list, rule.Expression())
		}
		if rule.Select_only_statement() != nil {
			list = append(list, rule.Select_only_statement())
		}
		return q.plsqlExtractSourceColumnSetFromExpressionList(list)
	case plsql.ISelect_only_statementContext:
		// For associated subquery, we should set the fromFieldList as the outerSchemaInfo.
		// So that the subquery can access the outer schema.
		// The reason for new q is that we still need the current fromFieldList, overriding it is not expected.
		subqueryExtractor := &querySpanExtractor{
			ctx:               q.ctx,
			defaultSchema:     q.defaultSchema,
			metaCache:         q.metaCache,
			f:                 q.f,
			outerTableSources: append(q.outerTableSources, q.tableSourcesFrom...),
			tableSourcesFrom:  []base.TableSource{},
		}
		tableSource, err := subqueryExtractor.plsqlExtractSelectOnlyStatement(rule)
		if err != nil {
			return "", nil, errors.Wrap(err, "failed to extract select only statement")
		}
		spanResult := tableSource.GetQuerySpanResult()

		sourceColumnSet := make(base.SourceColumnSet)

		for _, field := range spanResult {
			sourceColumnSet, _ = base.MergeSourceColumnSet(sourceColumnSet, field.SourceColumns)
		}
		return "", sourceColumnSet, nil
	case plsql.IStandard_functionContext:
		var list []antlr.ParserRuleContext
		if rule.String_function() != nil {
			list = append(list, rule.String_function())
		}
		if rule.Numeric_function_wrapper() != nil {
			list = append(list, rule.Numeric_function_wrapper())
		}
		if rule.Json_function() != nil {
			list = append(list, rule.Json_function())
		}
		if rule.Other_function() != nil {
			list = append(list, rule.Other_function())
		}
		return q.plsqlExtractSourceColumnSetFromExpressionList(list)
	case plsql.IString_functionContext:
		var list []antlr.ParserRuleContext
		for _, item := range rule.AllExpression() {
			list = append(list, item)
		}
		if rule.Table_element() != nil {
			list = append(list, rule.Table_element())
		}
		if rule.Standard_function() != nil {
			list = append(list, rule.Standard_function())
		}
		if rule.Concatenation() != nil {
			list = append(list, rule.Concatenation())
		}
		return q.plsqlExtractSourceColumnSetFromExpressionList(list)
	case plsql.INumeric_function_wrapperContext:
		var list []antlr.ParserRuleContext
		if rule.Numeric_function() != nil {
			list = append(list, rule.Numeric_function())
		}
		if rule.Single_column_for_loop() != nil {
			list = append(list, rule.Single_column_for_loop())
		}
		if rule.Multi_column_for_loop() != nil {
			list = append(list, rule.Multi_column_for_loop())
		}
		return q.plsqlExtractSourceColumnSetFromExpressionList(list)
	case plsql.INumeric_functionContext:
		var list []antlr.ParserRuleContext
		if rule.Expression() != nil {
			list = append(list, rule.Expression())
		}
		if rule.Expressions() != nil {
			list = append(list, rule.Expressions())
		}
		if rule.Concatenation() != nil {
			list = append(list, rule.Concatenation())
		}
		// TODO(rebelice): handle over_clause
		_, sensitive, err := q.plsqlExtractSourceColumnSetFromExpressionList(list)
		return "", sensitive, err
	case plsql.IExpressionsContext:
		var list []antlr.ParserRuleContext
		for _, item := range rule.AllExpression() {
			list = append(list, item)
		}
		return q.plsqlExtractSourceColumnSetFromExpressionList(list)
	case plsql.ISingle_column_for_loopContext:
		var list []antlr.ParserRuleContext
		if rule.Column_name() != nil {
			list = append(list, rule.Column_name())
		}
		for _, item := range rule.AllExpression() {
			list = append(list, item)
		}
		return q.plsqlExtractSourceColumnSetFromExpressionList(list)
	case plsql.IMulti_column_for_loopContext:
		var list []antlr.ParserRuleContext
		if rule.Paren_column_list() != nil {
			list = append(list, rule.Paren_column_list())
		}
		if rule.Subquery() != nil {
			list = append(list, rule.Subquery())
		}
		if rule.Expressions() != nil {
			list = append(list, rule.Expressions())
		}
		return q.plsqlExtractSourceColumnSetFromExpressionList(list)
	case plsql.IJson_functionContext:
		var list []antlr.ParserRuleContext
		for _, item := range rule.AllJson_array_element() {
			list = append(list, item)
		}
		for _, item := range rule.AllExpression() {
			list = append(list, item)
		}
		if rule.Json_object_content() != nil {
			list = append(list, rule.Json_object_content())
		}
		return q.plsqlExtractSourceColumnSetFromExpressionList(list)
	case plsql.IJson_array_elementContext:
		var list []antlr.ParserRuleContext
		if rule.Expression() != nil {
			list = append(list, rule.Expression())
		}
		if rule.Json_function() != nil {
			list = append(list, rule.Json_function())
		}
		return q.plsqlExtractSourceColumnSetFromExpressionList(list)
	case plsql.IJson_object_contentContext:
		var list []antlr.ParserRuleContext
		for _, item := range rule.AllJson_object_entry() {
			list = append(list, item)
		}
		return q.plsqlExtractSourceColumnSetFromExpressionList(list)
	case plsql.IJson_object_entryContext:
		var list []antlr.ParserRuleContext
		for _, item := range rule.AllExpression() {
			list = append(list, item)
		}
		if rule.Identifier() != nil {
			list = append(list, rule.Identifier())
		}
		return q.plsqlExtractSourceColumnSetFromExpressionList(list)
	case plsql.IOther_functionContext:
		var list []antlr.ParserRuleContext
		if rule.Function_argument_analytic() != nil {
			list = append(list, rule.Function_argument_analytic())
		}
		if rule.Function_argument_modeling() != nil {
			list = append(list, rule.Function_argument_modeling())
		}
		for _, item := range rule.AllConcatenation() {
			list = append(list, item)
		}
		if rule.Subquery() != nil {
			list = append(list, rule.Subquery())
		}
		if rule.Table_element() != nil {
			list = append(list, rule.Table_element())
		}
		if rule.Function_argument() != nil {
			list = append(list, rule.Function_argument())
		}
		if rule.Argument() != nil {
			list = append(list, rule.Argument())
		}
		if rule.Expressions() != nil {
			list = append(list, rule.Expressions())
		}
		for _, item := range rule.AllExpression() {
			list = append(list, item)
		}
		// TODO: handle xmltable
		return q.plsqlExtractSourceColumnSetFromExpressionList(list)
	case plsql.IFunction_argument_analyticContext:
		var list []antlr.ParserRuleContext
		for _, item := range rule.AllArgument() {
			list = append(list, item)
		}
		return q.plsqlExtractSourceColumnSetFromExpressionList(list)
	case plsql.IArgumentContext:
		return q.plsqlExtractSourceColumnSetFromExpression(rule.Expression())
	case plsql.IFunction_argument_modelingContext:
		// TODO(rebelice): implement standard function with USING
		return "", nil, nil
	case plsql.ITable_elementContext:
		// handled as column name
		var str []string
		for _, item := range rule.AllId_expression() {
			str = append(str, NormalizeIDExpression(item))
		}
		switch len(str) {
		case 1:
			return str[0], q.getFieldColumnSource("", "", str[0]), nil
		case 2:
			return str[1], q.getFieldColumnSource("", str[0], str[1]), nil
		case 3:
			return str[2], q.getFieldColumnSource(str[0], str[1], str[2]), nil
		default:
			return "", nil, nil
		}
	case plsql.IFunction_argumentContext:
		var list []antlr.ParserRuleContext
		for _, item := range rule.AllArgument() {
			list = append(list, item)
		}
		return q.plsqlExtractSourceColumnSetFromExpressionList(list)
	case plsql.IAtomContext:
		var list []antlr.ParserRuleContext
		if rule.Table_element() != nil {
			list = append(list, rule.Table_element())
		}
		if rule.Subquery() != nil {
			list = append(list, rule.Subquery())
		}
		for _, item := range rule.AllSubquery_operation_part() {
			list = append(list, item)
		}
		if rule.Expressions() != nil {
			list = append(list, rule.Expressions())
		}
		if rule.Constant() != nil {
			list = append(list, rule.Constant())
		}
		if rule.General_element() != nil {
			list = append(list, rule.General_element())
		}
		return q.plsqlExtractSourceColumnSetFromExpressionList(list)
	case plsql.ISubquery_operation_partContext:
		return q.plsqlExtractSourceColumnSetFromExpression(rule.Subquery_basic_elements())
	case plsql.ISubquery_basic_elementsContext:
		var list []antlr.ParserRuleContext
		if rule.Query_block() != nil {
			list = append(list, rule.Query_block())
		}
		if rule.Subquery() != nil {
			list = append(list, rule.Subquery())
		}
		return q.plsqlExtractSourceColumnSetFromExpressionList(list)
	case plsql.IModel_expression_elementContext:
		var list []antlr.ParserRuleContext
		for _, item := range rule.AllExpression() {
			list = append(list, item)
		}
		for _, item := range rule.AllSingle_column_for_loop() {
			list = append(list, item)
		}
		if rule.Multi_column_for_loop() != nil {
			list = append(list, rule.Multi_column_for_loop())
		}
		return q.plsqlExtractSourceColumnSetFromExpressionList(list)
	}

	return "", nil, nil
}

func (q *querySpanExtractor) getFieldColumnSource(schemaName, tableName, columnName string) base.SourceColumnSet {
	findInTableSource := func(tableSource base.TableSource) (base.SourceColumnSet, bool) {
		if schemaName != "" && schemaName != tableSource.GetSchemaName() {
			return nil, false
		}
		if tableName != "" && tableName != tableSource.GetTableName() {
			return nil, false
		}

		querySpanResult := tableSource.GetQuerySpanResult()
		for _, field := range querySpanResult {
			if field.Name == columnName {
				return field.SourceColumns, true
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
			return sourceColumnSet
		}
	}

	for _, tableSource := range q.tableSourcesFrom {
		if sourceColumnSet, ok := findInTableSource(tableSource); ok {
			return sourceColumnSet
		}
	}

	return base.SourceColumnSet{}
}

func (q *querySpanExtractor) plsqlExtractSourceColumnSetFromExpressionList(list []antlr.ParserRuleContext) (string, base.SourceColumnSet, error) {
	var fieldName string
	result := make(base.SourceColumnSet)
	for _, item := range list {
		name, sourceColumnSet, err := q.plsqlExtractSourceColumnSetFromExpression(item)
		if err != nil {
			return "", nil, err
		}
		result, _ = base.MergeSourceColumnSet(result, sourceColumnSet)
		if len(list) == 1 {
			fieldName = name
		}
	}
	return fieldName, result, nil
}

func (q *querySpanExtractor) getAllTableColumnSources(schemaName string, tableName string) ([]base.QuerySpanResult, bool) {
	findInTableSource := func(tableSource base.TableSource) ([]base.QuerySpanResult, bool) {
		if schemaName != "" && schemaName != tableSource.GetSchemaName() {
			return nil, false
		}
		if tableName != "" && tableName != tableSource.GetTableName() {
			return nil, false
		}

		// If the table name is empty, we should check if there are ambiguous fields,
		// but we delegate this responsibility to the db-server, we do the fail-open strategy here.
		return tableSource.GetQuerySpanResult(), true
	}

	for i := len(q.outerTableSources) - 1; i >= 0; i-- {
		if result, ok := findInTableSource(q.outerTableSources[i]); ok {
			return result, true
		}
	}

	for _, tableSource := range q.tableSourcesFrom {
		if result, ok := findInTableSource(tableSource); ok {
			return result, true
		}
	}

	return []base.QuerySpanResult{}, false
}

func (q *querySpanExtractor) plsqlExtractFromClause(ctx plsql.IFrom_clauseContext) ([]base.TableSource, error) {
	if ctx == nil {
		return nil, nil
	}

	tableRefList := ctx.Table_ref_list()
	if tableRefList == nil {
		return nil, nil
	}

	var result []base.TableSource
	for _, tableRef := range tableRefList.AllTable_ref() {
		tableSource, err := q.plsqlExtractTableRef(tableRef)
		if err != nil {
			return nil, errors.Wrap(err, "failed to extract table ref")
		}
		result = append(result, tableSource)
	}
	return result, nil
}

func (q *querySpanExtractor) plsqlExtractTableRef(ctx plsql.ITable_refContext) (base.TableSource, error) {
	if ctx == nil {
		return nil, nil
	}

	tableRefAux := ctx.Table_ref_aux()
	if tableRefAux == nil {
		return nil, nil
	}

	leftTableSource, err := q.plsqlExtractTableRefAux(tableRefAux)
	if err != nil {
		return nil, err
	}

	joins := ctx.AllJoin_clause()
	if len(joins) == 0 {
		return leftTableSource, nil
	}

	q.tableSourcesFrom = append(q.tableSourcesFrom, leftTableSource)
	for _, join := range joins {
		rightTableSource, err := q.plsqlExtractJoin(join)
		if err != nil {
			return nil, err
		}
		q.tableSourcesFrom = append(q.tableSourcesFrom, rightTableSource)
		leftTableSource, err = q.mergeJoinTableSource(join, leftTableSource, rightTableSource)
		if err != nil {
			return nil, err
		}
	}

	return leftTableSource, nil
}

func (q *querySpanExtractor) mergeJoinTableSource(ctx plsql.IJoin_clauseContext, leftTableSource, rightTableSource base.TableSource) (base.TableSource, error) {
	if ctx == nil {
		return nil, nil
	}

	leftSpanResult, rightSpanResult := leftTableSource.GetQuerySpanResult(), rightTableSource.GetQuerySpanResult()
	result := new(base.PseudoTable)

	leftSpanResultIdx, rightSpanResultIdx := make(map[string]int), make(map[string]int)
	for i, spanResult := range leftSpanResult {
		leftSpanResultIdx[spanResult.Name] = i
	}
	for i, spanResult := range rightSpanResult {
		rightSpanResultIdx[spanResult.Name] = i
	}

	if ctx.NATURAL() != nil {
		for idx, spanResult := range leftSpanResult {
			if _, ok := rightSpanResultIdx[spanResult.Name]; ok {
				spanResult.SourceColumns, _ = base.MergeSourceColumnSet(spanResult.SourceColumns, rightSpanResult[idx].SourceColumns)
			}
			result.Columns = append(result.Columns, spanResult)
		}
		for _, spanResult := range rightSpanResult {
			if _, ok := leftSpanResultIdx[spanResult.Name]; !ok {
				result.Columns = append(result.Columns, spanResult)
			}
		}
		return result, nil
	}

	if len(ctx.AllJoin_using_part()) != 0 {
		usingMap := make(map[string]bool)
		for _, part := range ctx.AllJoin_using_part() {
			for _, column := range part.Paren_column_list().Column_list().AllColumn_name() {
				_, _, name, err := plsqlNormalizeColumnName(q.defaultSchema, column)
				if err != nil {
					return nil, err
				}
				usingMap[name] = true
			}
		}

		for _, field := range leftSpanResult {
			_, existsInUsingMap := usingMap[field.Name]
			rightIdx, existsInRight := rightSpanResultIdx[field.Name]
			if existsInUsingMap && existsInRight {
				field.SourceColumns, _ = base.MergeSourceColumnSet(field.SourceColumns, rightSpanResult[rightIdx].SourceColumns)
			}
			result.Columns = append(result.Columns, field)
		}
		for _, field := range rightSpanResult {
			if _, existsInUsingMap := usingMap[field.Name]; !existsInUsingMap {
				result.Columns = append(result.Columns, field)
			}
		}
		return result, nil
	}

	result.Columns = append(result.Columns, leftSpanResult...)
	result.Columns = append(result.Columns, rightSpanResult...)
	return result, nil
}

func (q *querySpanExtractor) plsqlExtractJoin(ctx plsql.IJoin_clauseContext) (base.TableSource, error) {
	if ctx == nil {
		return nil, nil
	}

	tableRefAux := ctx.Table_ref_aux()
	if tableRefAux == nil {
		return nil, nil
	}

	return q.plsqlExtractTableRefAux(tableRefAux)
}

func (q *querySpanExtractor) plsqlExtractTableRefAux(ctx plsql.ITable_ref_auxContext) (base.TableSource, error) {
	if ctx == nil {
		return nil, nil
	}

	tableRefAuxInternal := ctx.Table_ref_aux_internal()
	tableSource, err := q.plsqlExtractTableRefAuxInternal(tableRefAuxInternal)
	if err != nil {
		return nil, errors.Wrap(err, "failed to extract table ref aux internal")
	}

	tableAlias := ctx.Table_alias()
	if tableAlias == nil {
		return tableSource, nil
	}

	alias := normalizeTableAlias(tableAlias)
	return &base.PseudoTable{
		Name:    alias,
		Columns: tableSource.GetQuerySpanResult(),
	}, nil
}

func (q *querySpanExtractor) plsqlExtractTableRefAuxInternal(ctx plsql.ITable_ref_aux_internalContext) (base.TableSource, error) {
	if ctx == nil {
		return nil, nil
	}

	switch rule := ctx.(type) {
	case *plsql.Table_ref_aux_internal_oneContext:
		return q.plsqlExtractDmlTableExpressionClause(rule.Dml_table_expression_clause())
	case *plsql.Table_ref_aux_internal_twoContext:
		if len(rule.AllSubquery_operation_part()) == 0 {
			return q.plsqlExtractTableRef(rule.Table_ref())
		}

		leftSpanResults, err := q.plsqlExtractTableRef(rule.Table_ref())
		if err != nil {
			return nil, errors.Wrap(err, "failed to extract left table ref")
		}
		leftQuerySpanResult := leftSpanResults.GetQuerySpanResult()
		for _, part := range rule.AllSubquery_operation_part() {
			rightSpanResults, err := q.plsqlExtractSubqueryOperationPart(part)
			if err != nil {
				return nil, errors.Wrap(err, "failed to extract right subquery operation part")
			}
			rightQuerySpanResult := rightSpanResults.GetQuerySpanResult()
			if len(leftQuerySpanResult) != len(rightQuerySpanResult) {
				return nil, errors.Errorf("left and right query span result length mismatch: %d != %d", len(leftQuerySpanResult), len(rightQuerySpanResult))
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
			leftQuerySpanResult = result
		}
		return &base.PseudoTable{
			Name:    "",
			Columns: leftQuerySpanResult,
		}, nil
	case *plsql.Table_ref_aux_internal_threeContext:
		return q.plsqlExtractDmlTableExpressionClause(rule.Dml_table_expression_clause())
	default:
		return nil, errors.Errorf("unknown table_ref_aux_internal rule: %T", rule)
	}
}

func (q *querySpanExtractor) plsqlExtractDmlTableExpressionClause(ctx plsql.IDml_table_expression_clauseContext) (base.TableSource, error) {
	if ctx == nil {
		return nil, nil
	}

	tableViewName := ctx.Tableview_name()
	if tableViewName != nil {
		schema, table := normalizeTableViewName(q.defaultSchema, tableViewName)
		return q.plsqlFindTableSchema(schema, table)
	}

	if ctx.Select_statement() != nil {
		return q.plsqlExtractSelect(ctx.Select_statement())
	}

	// TODO(rebelice): handle other cases for DML_TABLE_EXPRESSION_CLAUSE
	return nil, errors.Errorf("unsupported dml_table_expression_clause: %T", ctx)
}

func (q *querySpanExtractor) plsqlFindTableSchema(schemaName, tableName string) (base.TableSource, error) {
	if tableName == "DUAL" {
		return &base.PseudoTable{
			Name:    "DUAL",
			Columns: []base.QuerySpanResult{},
		}, nil
	}

	// Each CTE name in one WITH clause must be unique, but we can use the same name in the different level CTE, such as:
	//
	//  with tt2 as (
	//    with tt2 as (select * from t)
	//    select max(a) from tt2)
	//  select * from tt2
	//
	// This query has two CTE can be called `tt2`, and the FROM clause 'from tt2' uses the closer tt2 CTE.
	// This is the reason we loop the slice in reversed order.
	if schemaName == q.defaultSchema {
		for i := len(q.ctes) - 1; i >= 0; i-- {
			table := q.ctes[i]
			if table.Name == tableName {
				return table, nil
			}
		}
	}

	databaseName, dbSchema, err := q.getDatabaseMetadata(schemaName)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get database metadata for: %s", schemaName)
	}
	if dbSchema == nil {
		return nil, &parsererror.ResourceNotFoundError{
			Database: &schemaName,
		}
	}
	schema := dbSchema.GetSchema(schemaName)
	if schema == nil {
		return nil, &parsererror.ResourceNotFoundError{
			Database: &databaseName,
			Schema:   &schemaName,
		}
	}
	table := schema.GetTable(tableName)
	view := schema.GetView(tableName)
	materializedView := schema.GetMaterializedView(tableName)
	foreignTable := schema.GetExternalTable(tableName)
	if table == nil && view == nil && materializedView == nil && foreignTable == nil {
		return nil, &parsererror.ResourceNotFoundError{
			Database: &databaseName,
			Schema:   &schemaName,
			Table:    &tableName,
		}
	}

	if table != nil {
		var columns []string
		for _, column := range table.GetColumns() {
			columns = append(columns, column.Name)
		}
		return &base.PhysicalTable{
			Server:   "",
			Database: databaseName,
			Schema:   schemaName,
			Name:     tableName,
			Columns:  columns,
		}, nil
	}

	if foreignTable != nil {
		var columns []string
		for _, column := range foreignTable.GetColumns() {
			columns = append(columns, column.Name)
		}
		return &base.PhysicalTable{
			Server:   "",
			Database: databaseName,
			Schema:   schemaName,
			Name:     tableName,
			Columns:  columns,
		}, nil
	}

	if view != nil && view.Definition != "" {
		columns, err := q.getColumnsForView(view.Definition)
		if err != nil {
			return nil, err
		}
		return &base.PseudoTable{
			Name:    tableName,
			Columns: columns,
		}, nil
	}

	if materializedView != nil && materializedView.Definition != "" {
		columns, err := q.getColumnsForMaterializedView(materializedView.Definition)
		if err != nil {
			return nil, err
		}
		return &base.PseudoTable{
			Name:    tableName,
			Columns: columns,
		}, nil
	}
	return nil, nil
}

func (q *querySpanExtractor) getColumnsForView(definition string) ([]base.QuerySpanResult, error) {
	newQ := newQuerySpanExtractor(q.connectedDatabase, q.defaultSchema, q.f)
	span, err := newQ.getQuerySpan(q.ctx, definition)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get query span for view definition: %s", definition)
	}
	return span.Results, nil
}

func (q *querySpanExtractor) getColumnsForMaterializedView(definition string) ([]base.QuerySpanResult, error) {
	newQ := newQuerySpanExtractor(q.connectedDatabase, q.defaultSchema, q.f)
	span, err := newQ.getQuerySpan(q.ctx, definition)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get query span for materialized view definition: %s", definition)
	}
	return span.Results, nil
}

func (q *querySpanExtractor) plsqlExtractFactoringElement(ctx plsql.IFactoring_elementContext) (*base.PseudoTable, error) {
	if ctx == nil {
		return nil, nil
	}

	// Deal with recursive CTE first.
	tableName := NormalizeIdentifierContext(ctx.Query_name().Identifier())

	if yes, lastPart := q.plsqlIsRecursiveCTE(ctx); yes {
		subquery := ctx.Subquery()
		initialTableSource, err := q.plsqlExtractSubqueryExceptLastPart(subquery)
		if err != nil {
			return nil, err
		}

		initialQuerySpanResult := initialTableSource.GetQuerySpanResult()

		if ctx.Paren_column_list() != nil {
			var columnNames []string
			for _, column := range ctx.Paren_column_list().Column_list().AllColumn_name() {
				_, _, columnName, err := plsqlNormalizeColumnName("", column)
				if err != nil {
					return nil, err
				}
				columnNames = append(columnNames, columnName)
			}
			if len(columnNames) != len(initialQuerySpanResult) {
				return nil, errors.Errorf("column name count mismatch: %d != %d", len(columnNames), len(initialQuerySpanResult))
			}
			for i, columnName := range columnNames {
				initialQuerySpanResult[i].Name = columnName
			}
		}

		cteTableResource := &base.PseudoTable{
			Name:    tableName,
			Columns: initialQuerySpanResult,
		}

		// Compute dependent closures.
		// There are two ways to compute dependent closures:
		//   1. find the all dependent edges, then use graph theory traversal to find the closure.
		//   2. Iterate to simulate the CTE recursive process, each turn check whether the columns have changed, and stop if not change.
		//
		// Consider the option 2 can easy to implementation, because the simulate process has been written.
		// On the other hand, the number of iterations of the entire algorithm will not exceed the length of fields.
		// In actual use, the length of fields will not be more than 20 generally.
		// So I think it's OK for now.
		// If any performance issues in use, optimize here.
		q.ctes = append(q.ctes, cteTableResource)
		defer func() {
			q.ctes = q.ctes[:len(q.ctes)-1]
		}()

		for {
			recursiveTableSource, err := q.plsqlExtractSubqueryBasicElements(lastPart.Subquery_basic_elements())
			if err != nil {
				return nil, err
			}
			recursiveQuerySpanResult := recursiveTableSource.GetQuerySpanResult()
			if len(initialQuerySpanResult) != len(recursiveQuerySpanResult) {
				return nil, errors.Errorf("initial and recursive query span result length mismatch: %d != %d", len(initialQuerySpanResult), len(recursiveQuerySpanResult))
			}

			changed := false
			for i, spanQueryResult := range recursiveQuerySpanResult {
				newResourceColumns, hasDiff := base.MergeSourceColumnSet(initialQuerySpanResult[i].SourceColumns, spanQueryResult.SourceColumns)
				if hasDiff {
					changed = true
					initialQuerySpanResult[i].SourceColumns = newResourceColumns
				}
			}

			if !changed {
				break
			}
			q.ctes[len(q.ctes)-1].Columns = initialQuerySpanResult
		}
		return cteTableResource, nil
	}

	return q.plsqlExtractNonRecursiveCTE(ctx)
}

func (q *querySpanExtractor) plsqlExtractNonRecursiveCTE(ctx plsql.IFactoring_elementContext) (*base.PseudoTable, error) {
	if ctx == nil {
		return nil, nil
	}

	tableSource, err := q.plsqlExtractSubquery(ctx.Subquery())
	if err != nil {
		return nil, err
	}

	querySpanResult := tableSource.GetQuerySpanResult()

	if ctx.Paren_column_list() != nil {
		var columnNames []string
		for _, column := range ctx.Paren_column_list().Column_list().AllColumn_name() {
			_, _, columnName, err := plsqlNormalizeColumnName("", column)
			if err != nil {
				return nil, err
			}
			columnNames = append(columnNames, columnName)
		}
		if len(columnNames) != len(querySpanResult) {
			return nil, errors.Errorf("column name count mismatch: %d != %d", len(columnNames), len(querySpanResult))
		}
		for i, columnName := range columnNames {
			querySpanResult[i].Name = columnName
		}
	}

	tableName := NormalizeIdentifierContext(ctx.Query_name().Identifier())
	return &base.PseudoTable{
		Name:    tableName,
		Columns: querySpanResult,
	}, nil
}

func (q *querySpanExtractor) plsqlExtractSubqueryExceptLastPart(ctx plsql.ISubqueryContext) (base.TableSource, error) {
	if ctx == nil {
		return nil, nil
	}

	subqueryBasicElements := ctx.Subquery_basic_elements()
	if subqueryBasicElements == nil {
		return nil, nil
	}

	leftTableSource, err := q.plsqlExtractSubqueryBasicElements(subqueryBasicElements)
	if err != nil {
		return nil, err
	}

	leftQuerySpanResult := leftTableSource.GetQuerySpanResult()

	allParts := ctx.AllSubquery_operation_part()
	for _, part := range allParts[:len(allParts)-1] {
		rightTableSource, err := q.plsqlExtractSubqueryOperationPart(part)
		if err != nil {
			return nil, err
		}

		rightQueryStanResult := rightTableSource.GetQuerySpanResult()
		if len(leftQuerySpanResult) != len(rightQueryStanResult) {
			return nil, errors.Errorf("left and right query span result length mismatch: %d != %d", len(leftQuerySpanResult), len(rightQueryStanResult))
		}
		var result []base.QuerySpanResult
		for i, leftSpanResult := range leftQuerySpanResult {
			rightSpanResult := rightQueryStanResult[i]
			newResourceColumns, _ := base.MergeSourceColumnSet(leftSpanResult.SourceColumns, rightSpanResult.SourceColumns)
			result = append(result, base.QuerySpanResult{
				Name:          leftSpanResult.Name,
				SourceColumns: newResourceColumns,
			})
		}
		leftQuerySpanResult = result
	}

	return &base.PseudoTable{
		Name:    "",
		Columns: leftQuerySpanResult,
	}, nil
}

func (q *querySpanExtractor) plsqlExtractSubqueryOperationPart(ctx plsql.ISubquery_operation_partContext) (base.TableSource, error) {
	if ctx == nil {
		return nil, nil
	}

	return q.plsqlExtractSubqueryBasicElements(ctx.Subquery_basic_elements())
}

func (*querySpanExtractor) plsqlIsRecursiveCTE(ctx plsql.IFactoring_elementContext) (bool, plsql.ISubquery_operation_partContext) {
	subquery := ctx.Subquery()
	allParts := subquery.AllSubquery_operation_part()
	if len(allParts) == 0 {
		return false, nil
	}
	lastPart := allParts[len(allParts)-1]
	return lastPart.ALL() != nil, lastPart
}
