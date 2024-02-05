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

func newQuerySpanExtractor(defaultSchema string, f base.GetDatabaseMetadataFunc) *querySpanExtractor {
	return &querySpanExtractor{
		defaultSchema: defaultSchema,
		metaCache:     make(map[string]*model.DatabaseMetadata),
		f:             f,
	}
}

func (q *querySpanExtractor) getDatabaseMetadata(database string) (*model.DatabaseMetadata, error) {
	if meta, ok := q.metaCache[database]; ok {
		return meta, nil
	}
	meta, err := q.f(q.ctx, database)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get database metadata for: %s", q.defaultSchema)
	}
	q.metaCache[database] = meta
	return meta, nil
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
		extractor: q,
	}
	antlr.ParseTreeWalkerDefault.Walk(listener, tree)
	return listener.result, listener.err
}

type selectListener struct {
	*plsql.BasePlSqlParserListener

	extractor *querySpanExtractor
	result    *base.QuerySpan
	err       error
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

			tableSource, err := l.extractor.plsqlExtractContext(ctx)
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

	// TODO: Implement the logic to extract join.
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
	if fromClause != nil {
		tableSources, err := q.plsqlExtractFromClause(fromClause)
		if err != nil {
			return nil, err
		}
		q.outerTableSources = append(q.outerTableSources, tableSources...)
	}
	defer func() {
		q.outerTableSources = nil
	}()
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
			return nil, err
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
}

func (q *querySpanExtractor) plsqlExtractTableRefAux(ctx plsql.ITable_ref_auxContext) (base.TableSource, error) {
	if ctx == nil {
		return nil, nil
	}

	tableRefAuxInternal := ctx.Table_ref_aux_internal()
	tableSource, err := q.plsqlExtractTableRefAuxInternal(tableRefAuxInternal)
}

func (q *querySpanExtractor) plsqlExtractTableRefAuxInternal(ctx plsql.ITable_ref_aux_internalContext) (base.TableSource, error) {
	if ctx == nil {
		return nil, nil
	}

	switch rule := ctx.(type) {
	case *plsql.Table_ref_aux_internal_oneContext:
		return q.plsqlExtractDmlTableExpressionClause(rule.Dml_table_expression_clause())
	case *plsql.Table_ref_aux_internal_twoContext:
		// TODO(rebelice): handle subquery_operation_part
		return q.plsqlExtractTableRef(rule.Table_ref())
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

	}
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
	if schemaName == "" {
		for i := len(q.ctes) - 1; i >= 0; i-- {
			table := q.ctes[i]
			if table.Name == tableName {
				return table, nil
			}
		}
	}

	if schemaName == "" {
		schemaName = q.defaultSchema
	}
	dbSchema, err := q.getDatabaseMetadata(schemaName)
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
			Database: &schemaName,
			Schema:   &schemaName,
		}
	}
	table := schema.GetTable(tableName)
	view := schema.GetView(tableName)
	materializedView := schema.GetMaterializedView(tableName)
	foreignTable := schema.GetExternalTable(tableName)
	if table == nil && view == nil && materializedView == nil && foreignTable == nil {
		return nil, &parsererror.ResourceNotFoundError{
			Database: &schemaName,
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
			Database: schemaName,
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
			Database: schemaName,
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
