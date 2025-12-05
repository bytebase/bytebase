package mysql

import (
	"context"
	"slices"
	"strings"

	"github.com/antlr4-go/antlr/v4"
	parser "github.com/bytebase/parser/mysql"
	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/store/model"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

// querySpanExtractor is the extractor to extract the query span from a single statement.
type querySpanExtractor struct {
	ctx             context.Context
	defaultDatabase string

	gCtx base.GetQuerySpanContext

	ignoreCaseSensitive bool

	// ctes is the common table expressions, which is used to record the cte schema.
	// It should be reset to original state while quit the nested cte. For example:
	// WITH cte_1 AS (WITH cte_2 AS (SELECT * FROM t) SELECT * FROM cte_2) SELECT * FROM cte_2;
	// MySQL will throw a runtime error: (1146, "Table 'junk.cte_2' doesn't exist"), the `junk` is the database name.
	ctes []*base.PseudoTable

	// outerTableSources is the table sources from the outer query span.
	// it's used to resolve the column name in the correlated sub-query.
	outerTableSources []base.TableSource

	// tableSourceFrom is the table sources from the from clause.
	tableSourceFrom []base.TableSource

	// priorTableInFrom is the table sources from the from clause before the current table source.
	// It's used to resolve the column name in JSON_TABLE functions.
	priorTableInFrom []base.TableSource
}

// newQuerySpanExtractor creates a new query span extractor, the databaseMetadata and the ast are in the read guard.
func newQuerySpanExtractor(defaultDatabase string, gCtx base.GetQuerySpanContext, ignoreCaseSensitive bool) *querySpanExtractor {
	return &querySpanExtractor{
		defaultDatabase:     defaultDatabase,
		gCtx:                gCtx,
		ignoreCaseSensitive: ignoreCaseSensitive,
	}
}

func (q *querySpanExtractor) getQuerySpan(ctx context.Context, stmt string) (*base.QuerySpan, error) {
	parseResults, err := ParseMySQL(stmt)
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

	accessTables := getAccessTables(q.defaultDatabase, tree)
	// We do not support simultaneous access to the system table and the user table
	// because we do not synchronize the schema of the system table.
	// This causes an error (NOT_FOUND) when using querySpanExtractor.findTableSchema.
	// As a result, we exclude getting query span results for accessing only the system table.
	allSystems, mixed := isMixedQuery(accessTables, q.ignoreCaseSensitive)
	if mixed {
		return nil, base.MixUserSystemTablesError
	}

	queryTypeListener := &queryTypeListener{
		allSystems: allSystems,
		result:     base.QueryTypeUnknown,
	}
	antlr.ParseTreeWalkerDefault.Walk(queryTypeListener, tree)

	if queryTypeListener.result != base.Select {
		return &base.QuerySpan{
			Type:          queryTypeListener.result,
			SourceColumns: base.SourceColumnSet{},
			Results:       []base.QuerySpanResult{},
		}, nil
	}

	// For Explain Analyze SELECT, we determine the query type and return the access tables.
	if queryTypeListener.isExplainAnalyze {
		return &base.QuerySpan{
			Type:          queryTypeListener.result,
			SourceColumns: accessTables,
			Results:       []base.QuerySpanResult{},
		}, nil
	}

	q.ctx = ctx
	// We assume the caller had handled the statement type case,
	// so we only need to handle the determined statement type here.
	// In order to decrease the maintenance cost, we use listener to handle
	// the select statement precisely instead of using type switch.
	listener := newSelectOnlyListener(q)
	antlr.ParseTreeWalkerDefault.Walk(listener, tree)
	err = listener.err
	if err != nil {
		var resourceNotFound *base.ResourceNotFoundError
		if errors.As(err, &resourceNotFound) {
			return &base.QuerySpan{
				Type:          base.Select,
				SourceColumns: accessTables,
				Results:       []base.QuerySpanResult{},
				NotFoundError: resourceNotFound,
			}, nil
		}

		return nil, err
	}
	return &base.QuerySpan{
		Type:          base.Select,
		SourceColumns: accessTables,
		Results:       listener.querySpan.Results,
	}, nil
}

func (q *querySpanExtractor) extractContext(ctx antlr.ParserRuleContext) (*base.PseudoTable, error) {
	if ctx == nil {
		return nil, nil
	}

	switch ctx := ctx.(type) {
	case parser.ISelectStatementContext:
		return q.extractSelectStatement(ctx)
	default:
		return nil, nil
	}
}

// extractSelectStatement extracts the table source from the select statement.
// It regards the result of the select statement as the pseudo table.
func (q *querySpanExtractor) extractSelectStatement(ctx parser.ISelectStatementContext) (*base.PseudoTable, error) {
	if ctx == nil {
		return base.NewPseudoTable("", []base.QuerySpanResult{}), nil
	}

	switch {
	case ctx.QueryExpression() != nil:
		return q.extractQueryExpression(ctx.QueryExpression())
	case ctx.QueryExpressionParens() != nil:
		return q.extractQueryExpressionParens(ctx.QueryExpressionParens())
	case ctx.SelectStatementWithInto() != nil:
		return nil, errors.New("meet unsupported select statement with into")
	default:
		return nil, errors.New("unexpected select statement")
	}
}

func (q *querySpanExtractor) extractQueryExpression(ctx parser.IQueryExpressionContext) (*base.PseudoTable, error) {
	if ctx == nil {
		return base.NewPseudoTable("", []base.QuerySpanResult{}), nil
	}

	if ctx.WithClause() != nil {
		originalCtesLength := len(q.ctes)
		defer func() {
			q.ctes = q.ctes[:originalCtesLength]
		}()
		recursive := ctx.WithClause().RECURSIVE_SYMBOL() != nil
		for _, cte := range ctx.WithClause().AllCommonTableExpression() {
			cteTable, err := q.extractCommonTableExpression(cte, recursive)
			if err != nil {
				return nil, err
			}
			q.ctes = append(q.ctes, cteTable)
		}
	}

	switch {
	case ctx.QueryExpressionParens() != nil:
		return q.extractQueryExpressionParens(ctx.QueryExpressionParens())
	case ctx.QueryExpressionBody() != nil:
		return q.extractQueryExpressionBody(ctx.QueryExpressionBody())
	default:
		panic("unreachable")
	}
}

func (q *querySpanExtractor) extractQueryExpressionParens(ctx parser.IQueryExpressionParensContext) (*base.PseudoTable, error) {
	if ctx == nil {
		return nil, nil
	}

	switch {
	case ctx.QueryExpression() != nil:
		return q.extractQueryExpression(ctx.QueryExpression())
	case ctx.QueryExpressionParens() != nil:
		return q.extractQueryExpressionParens(ctx.QueryExpressionParens())
	default:
		return nil, nil
	}
}

func (q *querySpanExtractor) extractQueryExpressionBody(ctx parser.IQueryExpressionBodyContext) (*base.PseudoTable, error) {
	var result base.TableSource
	unionNum := 0
	for _, child := range ctx.GetChildren() {
		switch child := child.(type) {
		case *parser.QueryPrimaryContext:
			unionNum++
			ts, err := q.extractQueryPrimary(child)
			if err != nil {
				return nil, err
			}
			if result == nil {
				result = ts
			} else {
				if len(result.GetQuerySpanResult()) != len(ts.GetQuerySpanResult()) {
					return nil, errors.Errorf("MySQL %d UNION operator left has %d fields, right has %d fields", unionNum, len(result.GetQuerySpanResult()), len(ts.GetQuerySpanResult()))
				}
				// The UNION operator will merge the result columns of the left and right table sources.
				newQuerySpanResults := make([]base.QuerySpanResult, 0, len(result.GetQuerySpanResult()))
				resultQuerySpanResult := result.GetQuerySpanResult()
				for i := range ts.GetQuerySpanResult() {
					newSourceColumnSet, _ := base.MergeSourceColumnSet(resultQuerySpanResult[i].SourceColumns, ts.GetQuerySpanResult()[i].SourceColumns)
					newQuerySpanResults = append(newQuerySpanResults, base.QuerySpanResult{
						Name:          resultQuerySpanResult[i].Name,
						SourceColumns: newSourceColumnSet,
					})
				}
				result = &base.PseudoTable{
					Name:    "",
					Columns: newQuerySpanResults,
				}
			}
		case *parser.QueryExpressionParensContext:
			ts, err := q.extractQueryExpressionParens(child)
			if err != nil {
				return nil, err
			}
			if result == nil {
				result = ts
			} else {
				if len(result.GetQuerySpanResult()) != len(ts.GetQuerySpanResult()) {
					return nil, errors.Errorf("MySQL %d UNION operator left has %d fields, right has %d fields", unionNum, len(result.GetQuerySpanResult()), len(ts.GetQuerySpanResult()))
				}
				// The UNION operator will merge the result columns of the left and right table sources.
				newQuerySpanResults := make([]base.QuerySpanResult, 0, len(result.GetQuerySpanResult()))
				resultQuerySpanResult := result.GetQuerySpanResult()
				for i := range ts.GetQuerySpanResult() {
					newSourceColumnSet, _ := base.MergeSourceColumnSet(resultQuerySpanResult[i].SourceColumns, ts.GetQuerySpanResult()[i].SourceColumns)
					newQuerySpanResults = append(newQuerySpanResults, base.QuerySpanResult{
						Name:          resultQuerySpanResult[i].Name,
						SourceColumns: newSourceColumnSet,
					})
				}
				result = &base.PseudoTable{
					Name:    "",
					Columns: newQuerySpanResults,
				}
			}
		}
	}

	return &base.PseudoTable{
		Name:    "",
		Columns: result.GetQuerySpanResult(),
	}, nil
}

func (q *querySpanExtractor) extractQueryPrimary(ctx parser.IQueryPrimaryContext) (base.TableSource, error) {
	switch {
	case ctx.QuerySpecification() != nil:
		return q.extractQuerySpecification(ctx.QuerySpecification())
	case ctx.TableValueConstructor() != nil:
		return q.extractTableValueConstructor(ctx.TableValueConstructor())
	case ctx.ExplicitTable() != nil:
		return q.extractExplicitTable(ctx.ExplicitTable())
	default:
		panic("unreachable")
	}
}

func (q *querySpanExtractor) extractExplicitTable(ctx parser.IExplicitTableContext) (base.TableSource, error) {
	if ctx == nil {
		return nil, nil
	}

	databaseName, tableName := NormalizeMySQLTableRef(ctx.TableRef())
	tableSource, err := q.findTableSchema(databaseName, tableName)
	if err != nil {
		return nil, err
	}

	return tableSource, nil
}

func (q *querySpanExtractor) extractTableValueConstructor(ctx parser.ITableValueConstructorContext) (*base.PseudoTable, error) {
	if ctx == nil {
		return nil, nil
	}

	firstRow := ctx.RowValueExplicit(0)
	if firstRow == nil {
		panic("unreachable zero row in table value constructor")
	}

	values := firstRow.Values()
	if values == nil {
		return &base.PseudoTable{
			Name:    "",
			Columns: []base.QuerySpanResult{},
		}, nil
	}

	var columns []base.QuerySpanResult

	for _, child := range values.GetChildren() {
		switch child := child.(type) {
		case *parser.ExprContext:
			_, sourceColumns, isPlain, err := q.extractSourceColumnSetFromExpr(child)
			if err != nil {
				return nil, err
			}
			columns = append(columns, base.QuerySpanResult{
				Name:          child.GetParser().GetTokenStream().GetTextFromRuleContext(child),
				SourceColumns: sourceColumns,
				IsPlainField:  isPlain,
			})
		case antlr.TerminalNode:
			if child.GetSymbol().GetTokenType() == parser.MySQLParserDEFAULT_SYMBOL {
				columns = append(columns, base.QuerySpanResult{
					Name:          "DEFAULT",
					SourceColumns: make(base.SourceColumnSet),
					IsPlainField:  true,
				})
			}
		}
	}

	return &base.PseudoTable{
		Name:    "",
		Columns: columns,
	}, nil
}

func (q *querySpanExtractor) extractQuerySpecification(ctx parser.IQuerySpecificationContext) (*base.PseudoTable, error) {
	var fromSources []base.TableSource
	var err error
	if ctx.FromClause() != nil {
		originalLength := len(q.tableSourceFrom)
		defer func() {
			q.tableSourceFrom = q.tableSourceFrom[:originalLength]
		}()
		originalPriorLength := len(q.priorTableInFrom)
		defer func() {
			q.priorTableInFrom = q.priorTableInFrom[:originalPriorLength]
		}()
		fromSources, err = q.extractTableSourcesFromFromClause(ctx.FromClause())
		if err != nil {
			return nil, err
		}
		q.tableSourceFrom = append(q.tableSourceFrom, fromSources...)
	}

	querySpanResult, err := q.extractSelectItemList(ctx.SelectItemList(), fromSources)
	if err != nil {
		return nil, err
	}
	return &base.PseudoTable{
		Name:    "",
		Columns: querySpanResult,
	}, nil
}

func (q *querySpanExtractor) extractSelectItemList(ctx parser.ISelectItemListContext, fromSpanResult []base.TableSource) ([]base.QuerySpanResult, error) {
	if ctx == nil {
		return nil, nil
	}

	var result []base.QuerySpanResult

	if ctx.MULT_OPERATOR() != nil {
		for _, fromSpan := range fromSpanResult {
			result = append(result, fromSpan.GetQuerySpanResult()...)
		}
	}

	for _, selectItem := range ctx.AllSelectItem() {
		spanResult, err := q.extractSelectItem(selectItem)
		if err != nil {
			return nil, err
		}
		result = append(result, spanResult...)
	}

	return result, nil
}

func (q *querySpanExtractor) extractSelectItem(ctx parser.ISelectItemContext) ([]base.QuerySpanResult, error) {
	if ctx == nil {
		return nil, nil
	}

	switch {
	case ctx.TableWild() != nil:
		return q.extractTableWild(ctx.TableWild())
	case ctx.Expr() != nil:
		fieldName, sourceColumns, isPlain, err := q.extractSourceColumnSetFromExpr(ctx.Expr())
		if err != nil {
			return nil, err
		}
		if ctx.SelectAlias() != nil {
			fieldName = NormalizeMySQLSelectAlias(ctx.SelectAlias())
		} else if fieldName == "" {
			fieldName = ctx.GetParser().GetTokenStream().GetTextFromRuleContext(ctx)
		}

		return []base.QuerySpanResult{
			{
				Name:          fieldName,
				SourceColumns: sourceColumns,
				IsPlainField:  isPlain,
			},
		}, nil
	default:
		panic("unreachable")
	}
}

func (q *querySpanExtractor) extractSourceColumnSetFromExpr(ctx antlr.ParserRuleContext) (string, base.SourceColumnSet, bool, error) {
	if ctx == nil {
		return "", make(base.SourceColumnSet), true, nil
	}

	// The closure of expr rules.
	switch ctx := ctx.(type) {
	case parser.ISubqueryContext:
		baseSet := make(base.SourceColumnSet)
		// Subquery in SELECT fields is special.
		// It can be the non-associated or associated subquery.
		// For associated subquery, we should set the fromFieldList as the outerSchemaInfo.
		// So that the subquery can access the outer schema.
		// The reason for new extractor is that we still need the current fromFieldList, overriding it is not expected.
		subqueryExtractor := &querySpanExtractor{
			ctx: q.ctx,
			// The defaultDatabase is the same as the outer query.
			defaultDatabase:   q.defaultDatabase,
			gCtx:              q.gCtx,
			ctes:              q.ctes,
			outerTableSources: append(q.outerTableSources, q.tableSourceFrom...),
			tableSourceFrom:   []base.TableSource{},
		}
		tableSource, err := subqueryExtractor.extractSubquery(ctx)
		if err != nil {
			return "", nil, false, err
		}
		spanResult := tableSource.GetQuerySpanResult()
		isPlain := false
		if len(spanResult) == 1 {
			isPlain = spanResult[0].IsPlainField
		}
		for _, field := range spanResult {
			baseSet, _ = base.MergeSourceColumnSet(field.SourceColumns, field.SourceColumns)
		}
		return "", baseSet, isPlain, nil
	case parser.IColumnRefContext:
		databaseName, tableName, fieldName := NormalizeMySQLFieldIdentifier(ctx.FieldIdentifier())
		sourceColumnSet, err := q.getFieldColumnSource(databaseName, tableName, fieldName)
		if err != nil {
			return "", nil, false, err
		}
		return fieldName, sourceColumnSet, true, nil
	}

	var list []antlr.ParserRuleContext
	for _, child := range ctx.GetChildren() {
		if child, ok := child.(antlr.ParserRuleContext); ok {
			list = append(list, child)
		}
	}

	fieldName, sourceColumnSet, plain, err := q.extractSourceColumnSetFromExprList(list)
	if err != nil {
		return "", nil, false, err
	}
	if len(ctx.GetChildren()) > 1 {
		fieldName = ""
	}
	return fieldName, sourceColumnSet, plain && len(ctx.GetChildren()) < 2, nil
}

func (q *querySpanExtractor) extractSourceColumnSetFromExprList(ctxs []antlr.ParserRuleContext) (string, base.SourceColumnSet, bool, error) {
	baseSet := make(base.SourceColumnSet)
	var fieldName string
	var set base.SourceColumnSet
	var err error
	plain := true
	for _, ctx := range ctxs {
		var isPlain bool
		fieldName, set, isPlain, err = q.extractSourceColumnSetFromExpr(ctx)
		if err != nil {
			return "", nil, false, err
		}
		plain = plain && isPlain
		baseSet, _ = base.MergeSourceColumnSet(baseSet, set)
	}

	return fieldName, baseSet, plain, nil
}

func (q *querySpanExtractor) extractTableWild(ctx parser.ITableWildContext) ([]base.QuerySpanResult, error) {
	if ctx == nil {
		return nil, nil
	}

	var databaseName, tableName string
	if len(ctx.AllIdentifier()) == 2 {
		databaseName = NormalizeMySQLIdentifier(ctx.Identifier(0))
		tableName = NormalizeMySQLIdentifier(ctx.Identifier(1))
	} else {
		tableName = NormalizeMySQLIdentifier(ctx.Identifier(0))
	}
	querySpanResults, ok := q.getAllTableColumnSources(databaseName, tableName)
	if !ok {
		return nil, &base.ResourceNotFoundError{
			Err:      errors.Errorf("failed to find table to calculate asterisk"),
			Database: &databaseName,
			Table:    &tableName,
		}
	}
	return querySpanResults, nil
}

// extractTableSourcesFromFromClause extracts the table sources from the from clause.
// The result can be empty while the from clause is dual.
func (q *querySpanExtractor) extractTableSourcesFromFromClause(ctx parser.IFromClauseContext) ([]base.TableSource, error) {
	// DUAL is purely for the convenience of people who require that all SELECT statements should have FROM and possibly other clauses.
	// MySQL may ignore the clauses. MySQL does not require FROM DUAL if no tables are referenced.
	if ctx.DUAL_SYMBOL() != nil {
		return []base.TableSource{}, nil
	}

	return q.extractTableReferenceList(ctx.TableReferenceList())
}

func (q *querySpanExtractor) extractTableReferenceList(ctx parser.ITableReferenceListContext) ([]base.TableSource, error) {
	var result []base.TableSource
	for _, tableReference := range ctx.AllTableReference() {
		tableResource, err := q.extractTableReference(tableReference)
		if err != nil {
			return nil, err
		}
		q.priorTableInFrom = append(q.priorTableInFrom, tableResource)
		result = append(result, tableResource)
	}

	return result, nil
}

func (q *querySpanExtractor) extractTableReference(ctx parser.ITableReferenceContext) (base.TableSource, error) {
	if ctx.TableFactor() == nil {
		return nil, errors.Errorf("MySQL table reference should have table factor")
	}

	tableSource, err := q.extractTableFactor(ctx.TableFactor())
	if err != nil {
		return nil, err
	}

	if len(ctx.AllJoinedTable()) == 0 {
		return tableSource, nil
	}

	q.tableSourceFrom = append(q.tableSourceFrom, tableSource)

	for _, joinedTable := range ctx.AllJoinedTable() {
		tableSource, err = q.extractJoinedTable(tableSource, joinedTable)
		if err != nil {
			return nil, err
		}
	}

	return tableSource, nil
}

// extractJoinedTable extracts the joined table from the left and right table sources.
func (q *querySpanExtractor) extractJoinedTable(l base.TableSource, r parser.IJoinedTableContext) (base.TableSource, error) {
	rightTableSource, err := q.extractTableReference(r.TableReference())
	if err != nil {
		return nil, err
	}
	q.tableSourceFrom = append(q.tableSourceFrom, rightTableSource)

	tp := Join

	if v := r.InnerJoinType(); v != nil {
		if v.INNER_SYMBOL() != nil {
			tp = InnerJoin
		} else if v.CROSS_SYMBOL() != nil {
			tp = CrossJoin
		} else if v.STRAIGHT_JOIN_SYMBOL() != nil {
			tp = StraightJoin
		}
	} else if v := r.OuterJoinType(); v != nil {
		if v.LEFT_SYMBOL() != nil {
			tp = LeftOuterJoin
		} else if v.RIGHT_SYMBOL() != nil {
			tp = RightOuterJoin
		}
	} else if v := r.NaturalJoinType(); v != nil {
		tp = NaturalInnerJoin
		if v.LEFT_SYMBOL() != nil {
			tp = NaturalLeftOuterJoin
		} else if v.RIGHT_SYMBOL() != nil {
			tp = NaturalRightOuterJoin
		}
	}

	var usingIdentifiers []string
	if r.IdentifierListWithParentheses() != nil && r.IdentifierListWithParentheses().IdentifierList() != nil {
		usingIdentifiers = NormalizeMySQLIdentifierList(r.IdentifierListWithParentheses().IdentifierList())
	}

	joinedTableSource := joinTableSources(l, rightTableSource, tp, usingIdentifiers)

	return joinedTableSource, nil
}

type joinType int

const (
	Join joinType = iota
	InnerJoin
	CrossJoin
	StraightJoin
	LeftOuterJoin
	RightOuterJoin
	NaturalInnerJoin
	NaturalLeftOuterJoin
	NaturalRightOuterJoin
)

// joinTableSources joins the left and right table sources with the join type.
func joinTableSources(l, r base.TableSource, tp joinType, using []string) base.TableSource {
	switch tp {
	case Join, InnerJoin, CrossJoin, StraightJoin, LeftOuterJoin, RightOuterJoin:
		var columns []base.QuerySpanResult
		// In MySQL, JOIN, CROSS JOIN, and INNER JOIN are syntactic equivalents (they can replace each other).
		leftFieldsMap := make(map[string]bool)
		for _, field := range l.GetQuerySpanResult() {
			leftFieldsMap[strings.ToLower(field.Name)] = true
		}
		rightFieldsMap := make(map[string]bool)
		for _, field := range r.GetQuerySpanResult() {
			rightFieldsMap[strings.ToLower(field.Name)] = true
		}
		// Using will merge the result columns of the left and right table sources.
		lowercaseUsingFields := make(map[string]bool)
		for _, field := range using {
			lowercaseUsingFields[strings.ToLower(field)] = true
		}
		for _, field := range l.GetQuerySpanResult() {
			columns = append(columns, field)
			if _, ok := lowercaseUsingFields[strings.ToLower(field.Name)]; ok {
				delete(rightFieldsMap, field.Name)
			}
		}
		for _, field := range r.GetQuerySpanResult() {
			if _, ok := rightFieldsMap[strings.ToLower(field.Name)]; ok {
				columns = append(columns, field)
			}
		}
		return &base.PseudoTable{
			Name:    "",
			Columns: columns,
		}
	case NaturalInnerJoin, NaturalLeftOuterJoin, NaturalRightOuterJoin:
		// Natural join will merge all the columns with the same name.
		leftFieldsMap := make(map[string]bool)
		for _, field := range l.GetQuerySpanResult() {
			leftFieldsMap[strings.ToLower(field.Name)] = true
		}
		rightFieldsMap := make(map[string]bool)
		for _, field := range r.GetQuerySpanResult() {
			rightFieldsMap[strings.ToLower(field.Name)] = true
		}

		var columns []base.QuerySpanResult
		for _, field := range l.GetQuerySpanResult() {
			columns = append(columns, field)
			delete(rightFieldsMap, strings.ToLower(field.Name))
		}

		for _, field := range r.GetQuerySpanResult() {
			if _, ok := rightFieldsMap[strings.ToLower(field.Name)]; ok {
				columns = append(columns, field)
			}
		}
		return &base.PseudoTable{
			Name:    "",
			Columns: columns,
		}
	default:
		return nil
	}
}

func (q *querySpanExtractor) extractTableFactor(ctx parser.ITableFactorContext) (base.TableSource, error) {
	switch {
	case ctx.SingleTable() != nil:
		return q.extractSingleTable(ctx.SingleTable())
	case ctx.SingleTableParens() != nil:
		return q.extractSingleTableParens(ctx.SingleTableParens())
	case ctx.DerivedTable() != nil:
		return q.extractDerivedTable(ctx.DerivedTable())
	case ctx.TableReferenceListParens() != nil:
		// TableReferenceListParens is tableFactor rules it a MySQL syntax extension,
		// for instance:
		// SELECT * FORM (t1, t2) JOIN t3 ON 1;
		// This syntax is quivalent to
		// SELECT * FROM (t1 CROSS JOIN t2) JOIN t3 ON 1;
		tableSources, err := q.extractTableReferenceListParens(ctx.TableReferenceListParens())
		if err != nil {
			return nil, err
		}
		var result base.TableSource
		for i, tableSource := range tableSources {
			q.tableSourceFrom = append(q.tableSourceFrom, tableSource)
			if i == 0 {
				result = tableSource
				continue
			}
			result = joinTableSources(result, tableSource, CrossJoin, nil)
		}
		return result, nil
	case ctx.TableFunction() != nil:
		return q.extractTableFunction(ctx.TableFunction())
	default:
		panic("unreachable")
	}
}

func (q *querySpanExtractor) extractTableFunction(ctx parser.ITableFunctionContext) (base.TableSource, error) {
	if ctx == nil {
		return nil, nil
	}

	name, sourceColumnSet, isPlain, err := q.extractSourceColumnSetFromExpr(ctx.Expr())
	if err != nil {
		return nil, err
	}

	columnList := mysqlExtractColumnsClause(ctx.ColumnsClause())

	if ctx.TableAlias() != nil {
		name = NormalizeMySQLIdentifier(ctx.TableAlias().Identifier())
	} else if name == "" {
		name = ctx.GetParser().GetTokenStream().GetTextFromRuleContext(ctx.Expr())
	}

	var result []base.QuerySpanResult
	for _, column := range columnList {
		result = append(result, base.QuerySpanResult{
			Name:          column,
			SourceColumns: sourceColumnSet,
			IsPlainField:  isPlain,
		})
	}

	return &base.PseudoTable{
		Name:    name,
		Columns: result,
	}, nil
}

func (q *querySpanExtractor) extractTableReferenceListParens(ctx parser.ITableReferenceListParensContext) ([]base.TableSource, error) {
	switch {
	case ctx.TableReferenceList() != nil:
		return q.extractTableReferenceList(ctx.TableReferenceList())
	case ctx.TableReferenceListParens() != nil:
		return q.extractTableReferenceListParens(ctx.TableReferenceListParens())
	default:
		panic("unreachable")
	}
}

func (q *querySpanExtractor) extractSubquery(ctx parser.ISubqueryContext) (*base.PseudoTable, error) {
	return q.extractQueryExpressionParens(ctx.QueryExpressionParens())
}

func (q *querySpanExtractor) extractDerivedTable(ctx parser.IDerivedTableContext) (base.TableSource, error) {
	tableSource, err := q.extractSubquery(ctx.Subquery())
	if err != nil {
		return nil, err
	}

	var aliasName string
	if ctx.TableAlias() != nil {
		aliasName = NormalizeMySQLIdentifier(ctx.TableAlias().Identifier())
	}
	if aliasName != "" {
		tableSource = &base.PseudoTable{
			Name:    aliasName,
			Columns: tableSource.GetQuerySpanResult(),
		}
	}

	if ctx.ColumnInternalRefList() != nil {
		columnList := extractColumnInternalRefList(ctx.ColumnInternalRefList())
		if len(columnList) != len(tableSource.GetQuerySpanResult()) {
			return nil, errors.Errorf("column list length %d doesn't match the derived table column length %d", len(columnList), len(tableSource.GetQuerySpanResult()))
		}
		for i := range columnList {
			tableSource.GetQuerySpanResult()[i].Name = columnList[i]
		}
	}

	return tableSource, nil
}

func extractColumnInternalRefList(ctx parser.IColumnInternalRefListContext) []string {
	if ctx == nil {
		return nil
	}

	var result []string
	for _, columnInternalRef := range ctx.AllColumnInternalRef() {
		result = append(result, NormalizeMySQLIdentifier(columnInternalRef.Identifier()))
	}
	return result
}

func (q *querySpanExtractor) extractSingleTable(ctx parser.ISingleTableContext) (base.TableSource, error) {
	databaseName, tableName := NormalizeMySQLTableRef(ctx.TableRef())
	tableSource, err := q.findTableSchema(databaseName, tableName)
	if err != nil {
		return nil, err
	}

	var aliasName string
	if ctx.TableAlias() != nil {
		aliasName = NormalizeMySQLIdentifier(ctx.TableAlias().Identifier())
	}

	if aliasName != "" {
		return &base.PseudoTable{
			Name:    aliasName,
			Columns: tableSource.GetQuerySpanResult(),
		}, nil
	}
	return tableSource, nil
}

func (q *querySpanExtractor) extractSingleTableParens(ctx parser.ISingleTableParensContext) (base.TableSource, error) {
	switch {
	case ctx.SingleTable() != nil:
		return q.extractSingleTable(ctx.SingleTable())
	case ctx.SingleTableParens() != nil:
		return q.extractSingleTableParens(ctx.SingleTableParens())
	default:
		panic("unreachable")
	}
}

// extractCommonTableExpression extracts the pseudo table from the common table expression.
func (q *querySpanExtractor) extractCommonTableExpression(ctx parser.ICommonTableExpressionContext, recursive bool) (*base.PseudoTable, error) {
	if recursive {
		return q.extractRecursiveCTE(ctx)
	}
	return q.extractNonRecursiveCTE(ctx)
}

// extractRecursiveCTE extracts the pseudo table from the recursive common table expression.
func (q *querySpanExtractor) extractRecursiveCTE(ctx parser.ICommonTableExpressionContext) (*base.PseudoTable, error) {
	cteName := NormalizeMySQLIdentifier(ctx.Identifier())
	l := &recursiveCTEExtractListener{
		extractor: q,
		cteInfo: &base.PseudoTable{
			Name:    cteName,
			Columns: []base.QuerySpanResult{},
		},
		selfName:                      cteName,
		foundFirstQueryExpressionBody: false,
		inCTE:                         false,
	}
	if ctx.ColumnInternalRefList() != nil {
		columnList := mysqlExtractColumnInternalRefList(ctx.ColumnInternalRefList())
		for i := range columnList {
			l.cteInfo.Columns = append(l.cteInfo.Columns, base.QuerySpanResult{
				Name:          columnList[i],
				SourceColumns: make(base.SourceColumnSet),
			})
		}
	}
	antlr.ParseTreeWalkerDefault.Walk(l, ctx.Subquery())
	if l.err != nil {
		return nil, l.err
	}

	return l.cteInfo, nil
}

// extractNonRecursiveCTE extracts the pseudo table from the non-recursive common table expression.
func (q *querySpanExtractor) extractNonRecursiveCTE(ctx parser.ICommonTableExpressionContext) (*base.PseudoTable, error) {
	tableSource, err := q.extractSubquery(ctx.Subquery())
	if err != nil {
		return nil, err
	}
	spanResults := tableSource.GetQuerySpanResult()
	if ctx.ColumnInternalRefList() != nil {
		columnList := mysqlExtractColumnInternalRefList(ctx.ColumnInternalRefList())
		if len(columnList) != len(tableSource.GetQuerySpanResult()) {
			return nil, errors.Errorf("MySQL CTE column list should have the same length, but got %d and %d", len(columnList), len(tableSource.GetQuerySpanResult()))
		}
		for i := range columnList {
			spanResults[i].Name = columnList[i]
		}
	}
	cteName := NormalizeMySQLIdentifier(ctx.Identifier())
	result := &base.PseudoTable{
		Name:    cteName,
		Columns: spanResults,
	}
	return result, nil
}

type recursiveCTEExtractListener struct {
	*parser.BaseMySQLParserListener

	extractor                     *querySpanExtractor
	cteInfo                       *base.PseudoTable
	selfName                      string
	outerCTEs                     []parser.IWithClauseContext
	foundFirstQueryExpressionBody bool
	inCTE                         bool
	err                           error
}

// EnterQueryExpression is called when production queryExpression is entered.
func (l *recursiveCTEExtractListener) EnterQueryExpression(ctx *parser.QueryExpressionContext) {
	if l.foundFirstQueryExpressionBody || l.inCTE || l.err != nil {
		return
	}
	if ctx.WithClause() != nil {
		l.outerCTEs = append(l.outerCTEs, ctx.WithClause())
	}
}

// EnterWithClause is called when production commonTableExpression is entered.
func (l *recursiveCTEExtractListener) EnterWithClause(_ *parser.WithClauseContext) {
	l.inCTE = true
}

// ExitWithClause is called when production commonTableExpression is exited.
func (l *recursiveCTEExtractListener) ExitWithClause(_ *parser.WithClauseContext) {
	l.inCTE = false
}

// EnterQueryExpressionBody is called when production queryExpressionBody is entered.
func (l *recursiveCTEExtractListener) EnterQueryExpressionBody(ctx *parser.QueryExpressionBodyContext) {
	if l.err != nil {
		return
	}
	if l.inCTE {
		return
	}
	if l.foundFirstQueryExpressionBody {
		return
	}

	l.foundFirstQueryExpressionBody = true

	// Deal with outer CTEs.
	cetOuterLength := len(l.extractor.ctes)
	defer func() {
		l.extractor.ctes = l.extractor.ctes[:cetOuterLength]
	}()
	for _, outerCTE := range l.outerCTEs {
		recursive := outerCTE.RECURSIVE_SYMBOL() != nil
		for _, cte := range outerCTE.AllCommonTableExpression() {
			cteTable, err := l.extractor.extractCommonTableExpression(cte, recursive)
			if err != nil {
				l.err = err
				return
			}
			l.extractor.ctes = append(l.extractor.ctes, cteTable)
		}
	}

	var initialPart []base.QuerySpanResult
	var recursivePart []antlr.ParserRuleContext

	findRecursivePart := false
	for _, child := range ctx.GetChildren() {
		switch child := child.(type) {
		case *parser.QueryPrimaryContext:
			if !findRecursivePart {
				resource := extractTableRefs("", child)

				for _, item := range resource {
					if item.Database == "" && item.Table == l.selfName {
						findRecursivePart = true
						break
					}
				}
			}

			if findRecursivePart {
				recursivePart = append(recursivePart, child)
			} else {
				tableSource, err := l.extractor.extractQueryPrimary(child)
				if err != nil {
					l.err = err
					return
				}
				if len(initialPart) == 0 {
					initialPart = tableSource.GetQuerySpanResult()
				} else {
					if len(initialPart) != len(tableSource.GetQuerySpanResult()) {
						l.err = errors.Errorf("MySQL UNION field list should have the same length, but got %d and %d", len(initialPart), len(tableSource.GetQuerySpanResult()))
						return
					}
					for i := range initialPart {
						initialPart[i].SourceColumns, _ = base.MergeSourceColumnSet(initialPart[i].SourceColumns, tableSource.GetQuerySpanResult()[i].SourceColumns)
					}
				}
			}
		case *parser.QueryExpressionParensContext:
			queryExpression := extractQueryExpression(child)
			if queryExpression == nil {
				// Never happen.
				l.err = errors.Errorf("MySQL query expression parens should have query expression, but got nil")
				return
			}

			if !findRecursivePart {
				resource := extractTableRefs("", queryExpression)

				for _, item := range resource {
					if item.Database == "" && item.Table == l.selfName {
						findRecursivePart = true
						break
					}
				}
			}

			if findRecursivePart {
				recursivePart = append(recursivePart, child)
			} else {
				tableSource, err := l.extractor.extractQueryExpression(queryExpression)
				if err != nil {
					l.err = err
					return
				}
				if len(initialPart) == 0 {
					initialPart = tableSource.GetQuerySpanResult()
				} else {
					if len(initialPart) != len(tableSource.GetQuerySpanResult()) {
						l.err = errors.Errorf("MySQL UNION field list should have the same length, but got %d and %d", len(initialPart), len(tableSource.GetQuerySpanResult()))
						return
					}
					for i := range initialPart {
						initialPart[i].SourceColumns, _ = base.MergeSourceColumnSet(initialPart[i].SourceColumns, tableSource.GetQuerySpanResult()[i].SourceColumns)
					}
				}
			}
		}
	}

	// Compute dependent closures.
	// There are two ways to compute dependent closures:
	//   1. find the all dependent edges, then use graph theory traversal to find the closure.
	//   2. Iterate to simulate the CTE recursive process, each turn check whether the Sensitive state has changed, and stop if no change.
	//
	// Consider the option 2 can easy to implementation, because the simulate process has been written.
	// On the other hand, the number of iterations of the entire algorithm will not exceed the length of fields.
	// In actual use, the length of fields will not be more than 20 generally.
	// So I think it's OK for now.
	// If any performance issues in use, optimize here.
	if len(l.cteInfo.GetQuerySpanResult()) == 0 {
		for _, item := range initialPart {
			l.cteInfo.Columns = append(l.cteInfo.Columns, base.QuerySpanResult{
				Name:          item.Name,
				SourceColumns: item.SourceColumns,
			})
		}
	} else {
		if len(initialPart) != len(l.cteInfo.GetQuerySpanResult()) {
			l.err = errors.Errorf("The common table expression and column names list have different column counts")
			return
		}
		for i := range initialPart {
			l.cteInfo.Columns[i].SourceColumns, _ = base.MergeSourceColumnSet(initialPart[i].SourceColumns, l.cteInfo.GetQuerySpanResult()[i].SourceColumns)
		}
	}

	if len(recursivePart) == 0 {
		return
	}

	l.extractor.ctes = append(l.extractor.ctes, l.cteInfo)
	defer func() {
		l.extractor.ctes = l.extractor.ctes[:len(l.extractor.ctes)-1]
	}()
	for {
		var fieldList []base.QuerySpanResult
		for _, item := range recursivePart {
			var itemFields []base.QuerySpanResult
			switch item := item.(type) {
			case *parser.QueryPrimaryContext:
				var err error
				tableSource, err := l.extractor.extractQueryPrimary(item)
				if err != nil {
					l.err = err
					return
				}
				itemFields = tableSource.GetQuerySpanResult()
			case *parser.QueryExpressionContext:
				var err error
				tableSource, err := l.extractor.extractQueryExpression(item)
				if err != nil {
					l.err = err
					return
				}
				itemFields = tableSource.GetQuerySpanResult()
			}
			if len(fieldList) == 0 {
				fieldList = itemFields
			} else {
				if len(fieldList) != len(itemFields) {
					l.err = errors.Errorf("MySQL UNION field list should have the same length, but got %d and %d", len(fieldList), len(itemFields))
					return
				}
				for i := range fieldList {
					fieldList[i].SourceColumns, _ = base.MergeSourceColumnSet(fieldList[i].SourceColumns, itemFields[i].SourceColumns)
				}
			}
		}

		if len(fieldList) != len(l.cteInfo.GetQuerySpanResult()) {
			// The error content comes from MySQL.
			l.err = errors.Errorf("The common table expression and column names list have different column counts")
			return
		}

		changed := false
		for i, field := range fieldList {
			var ok bool
			l.cteInfo.Columns[i].SourceColumns, ok = base.MergeSourceColumnSet(l.cteInfo.GetQuerySpanResult()[i].SourceColumns, field.SourceColumns)
			changed = changed || ok
		}

		if !changed {
			break
		}
		l.extractor.ctes[len(l.extractor.ctes)-1] = l.cteInfo
	}
}

func (q *querySpanExtractor) getAllTableColumnSources(databaseName, tableName string) ([]base.QuerySpanResult, bool) {
	findInTableSource := func(tableSource base.TableSource) ([]base.QuerySpanResult, bool) {
		if q.ignoreCaseSensitive {
			if databaseName != "" && !strings.EqualFold(databaseName, tableSource.GetDatabaseName()) {
				return nil, false
			}
			if tableName != "" && !strings.EqualFold(tableName, tableSource.GetTableName()) {
				return nil, false
			}
		} else {
			if databaseName != "" && databaseName != tableSource.GetDatabaseName() {
				return nil, false
			}
			if tableName != "" && tableName != tableSource.GetTableName() {
				return nil, false
			}
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

func (q *querySpanExtractor) getFieldColumnSource(databaseName, tableName, fieldName string) (base.SourceColumnSet, error) {
	databaseName = q.filterClusterName(databaseName)
	findInTableSource := func(tableSource base.TableSource) (base.SourceColumnSet, bool) {
		if q.ignoreCaseSensitive {
			if databaseName != "" && !strings.EqualFold(databaseName, tableSource.GetDatabaseName()) {
				return nil, false
			}
			if tableName != "" && !strings.EqualFold(tableName, tableSource.GetTableName()) {
				return nil, false
			}
		} else {
			if databaseName != "" && databaseName != tableSource.GetDatabaseName() {
				return nil, false
			}
			if tableName != "" && tableName != tableSource.GetTableName() {
				return nil, false
			}
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

	for i := len(q.priorTableInFrom) - 1; i >= 0; i-- {
		if sourceColumnSet, ok := findInTableSource(q.priorTableInFrom[i]); ok {
			return sourceColumnSet, nil
		}
	}

	return nil, &base.ResourceNotFoundError{
		Database: &databaseName,
		Table:    &tableName,
		Column:   &fieldName,
	}
}

func (q *querySpanExtractor) filterClusterName(databaseName string) string {
	if q.gCtx.Engine == storepb.Engine_STARROCKS {
		// For StarRocks, user can use `cluster_name:database_name` as the database name.
		// But for the query span in Bytebase, we only care about the database name.
		list := strings.Split(databaseName, ":")
		if len(list) > 1 {
			databaseName = list[len(list)-1]
		}
	}
	return databaseName
}

func (q *querySpanExtractor) findTableSchema(databaseName, tableName string) (base.TableSource, error) {
	// Each CTE name in one WITH clause must be unique, but we can use the same name in the different level CTE, such as:
	//
	//  with tt2 as (
	//    with tt2 as (select * from t)
	//    select max(a) from tt2)
	//  select * from tt2
	//
	// This query has two CTE can be called `tt2`, and the FROM clause 'from tt2' uses the closer tt2 CTE.
	// This is the reason we loop the slice in reversed order.
	if databaseName == "" {
		for i := len(q.ctes) - 1; i >= 0; i-- {
			table := q.ctes[i]
			if table.Name == tableName {
				return table, nil
			}
		}
	}

	if databaseName == "" {
		databaseName = q.defaultDatabase
	}

	databaseName = q.filterClusterName(databaseName)

	var dbMetadata *model.DatabaseMetadata
	allDatabaseNames, err := q.gCtx.ListDatabaseNamesFunc(q.ctx, q.gCtx.InstanceID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to list databases")
	}
	if q.ignoreCaseSensitive {
		for _, db := range allDatabaseNames {
			if strings.EqualFold(db, databaseName) {
				_, dbMetadata, err = q.gCtx.GetDatabaseMetadataFunc(q.ctx, q.gCtx.InstanceID, db)
				if err != nil {
					return nil, errors.Wrapf(err, "failed to get database metadata for database %q", db)
				}
				break
			}
		}
	} else {
		for _, db := range allDatabaseNames {
			if db == databaseName {
				_, dbMetadata, err = q.gCtx.GetDatabaseMetadataFunc(q.ctx, q.gCtx.InstanceID, db)
				if err != nil {
					return nil, errors.Wrapf(err, "failed to get database metadata for database %q", db)
				}
				break
			}
		}
	}
	if dbMetadata == nil {
		return nil, &base.ResourceNotFoundError{
			Database: &databaseName,
		}
	}

	emptySchema := ""
	schema := dbMetadata.GetSchemaMetadata(emptySchema)
	if schema == nil {
		return nil, &base.ResourceNotFoundError{
			Database: &databaseName,
			Schema:   &emptySchema,
		}
	}

	var tableSchema *model.TableMetadata
	if q.ignoreCaseSensitive {
		for _, table := range schema.ListTableNames() {
			if strings.EqualFold(table, tableName) {
				tableSchema = schema.GetTable(table)
				break
			}
		}
	} else {
		tableSchema = schema.GetTable(tableName)
	}
	if tableSchema != nil {
		columnNames := make([]string, 0, len(tableSchema.GetProto().GetColumns()))
		for _, column := range tableSchema.GetProto().GetColumns() {
			columnNames = append(columnNames, column.Name)
		}
		return &base.PhysicalTable{
			Name:     tableSchema.GetProto().Name,
			Schema:   emptySchema,
			Database: dbMetadata.GetProto().GetName(),
			Server:   "",
			Columns:  columnNames,
		}, nil
	}

	var viewSchema *storepb.ViewMetadata
	if q.ignoreCaseSensitive {
		for _, view := range schema.ListViewNames() {
			if strings.EqualFold(view, tableName) {
				viewSchema = schema.GetView(view)
				break
			}
		}
	} else {
		viewSchema = schema.GetView(tableName)
	}
	if viewSchema != nil {
		columns, err := q.getColumnsForView(viewSchema.Definition)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get columns for view %q", tableName)
		}
		return &base.PhysicalView{
			Name:     viewSchema.Name,
			Schema:   emptySchema,
			Database: dbMetadata.GetProto().GetName(),
			Server:   "",
			Columns:  columns,
		}, nil
	}

	return nil, &base.ResourceNotFoundError{
		Database: &databaseName,
		Schema:   &emptySchema,
		Table:    &tableName,
	}
}

func (q *querySpanExtractor) getColumnsForView(definition string) ([]base.QuerySpanResult, error) {
	newQ := newQuerySpanExtractor(q.defaultDatabase, q.gCtx, q.ignoreCaseSensitive)
	span, err := newQ.getQuerySpan(q.ctx, definition)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get query span for view")
	}
	if span.NotFoundError != nil {
		return nil, span.NotFoundError
	}
	return span.Results, nil
}

// selectOnlyListener is the listener to listen the top level select statement only.
type selectOnlyListener struct {
	*parser.BaseMySQLParserListener

	// The only misson of the listener is the find the precise select statement.
	// All the eval work will be handled by the querySpanExtractor.
	extractor *querySpanExtractor
	querySpan *base.QuerySpan
	err       error
}

func newSelectOnlyListener(extractor *querySpanExtractor) *selectOnlyListener {
	return &selectOnlyListener{
		extractor: extractor,
		querySpan: &base.QuerySpan{
			Results:       []base.QuerySpanResult{},
			SourceColumns: make(base.SourceColumnSet),
		},
	}
}

func (s *selectOnlyListener) EnterSelectStatement(ctx *parser.SelectStatementContext) {
	parent := ctx.GetParent()
	if parent == nil {
		return
	}

	if _, ok := parent.(*parser.SimpleStatementContext); !ok {
		return
	}

	fields, err := s.extractor.extractContext(ctx)
	if err != nil {
		s.err = err
		return
	}

	s.querySpan.Results = append(s.querySpan.Results, fields.Columns...)
}

func getAccessTables(currentDatabase string, tree antlr.Tree) base.SourceColumnSet {
	l := newAccessTableListener(currentDatabase)

	result := make(base.SourceColumnSet)
	antlr.ParseTreeWalkerDefault.Walk(l, tree)
	result, _ = base.MergeSourceColumnSet(result, l.sourceColumnSet)

	return result
}

type accessTableListener struct {
	*parser.BaseMySQLParserListener

	currentDatabase string
	sourceColumnSet base.SourceColumnSet
}

func newAccessTableListener(currentDatabase string) *accessTableListener {
	return &accessTableListener{
		currentDatabase: currentDatabase,
		sourceColumnSet: make(base.SourceColumnSet),
	}
}

// EnterTableRef is called when production tableRef is entered.
func (l *accessTableListener) EnterTableRef(ctx *parser.TableRefContext) {
	sourceColumn := base.ColumnResource{
		Database: l.currentDatabase,
	}
	if ctx.DotIdentifier() != nil {
		sourceColumn.Table = NormalizeMySQLIdentifier(ctx.DotIdentifier().Identifier())
	}
	db, table := normalizeMySQLQualifiedIdentifier(ctx.QualifiedIdentifier())
	if db != "" {
		sourceColumn.Database = db
	}
	sourceColumn.Table = table
	l.sourceColumnSet[sourceColumn] = true
}

// isMixedQuery checks whether the query accesses the user table and system table at the same time.
// It returns whether all tables are system tables and whether there is a mixture.
func isMixedQuery(m base.SourceColumnSet, ignoreCaseSensitive bool) (bool, bool) {
	hasSystem, hasUser := false, false
	for table := range m {
		if isSystemResource(table, ignoreCaseSensitive) {
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

var systemDatabases = map[string]bool{
	"information_schema": true,
	"performance_schema": true,
	"mysql":              true,
}

func isSystemResource(resource base.ColumnResource, ignoreCaseSensitive bool) bool {
	database := resource.Database
	if ignoreCaseSensitive {
		database = strings.ToLower(database)
	}
	return systemDatabases[database]
}

func mysqlExtractColumnsClause(ctx parser.IColumnsClauseContext) []string {
	if ctx == nil {
		return nil
	}

	var result []string
	for _, column := range ctx.AllJtColumn() {
		result = append(result, mysqlExtractJtColumn(column)...)
	}

	return result
}

func mysqlExtractColumnInternalRefList(ctx parser.IColumnInternalRefListContext) []string {
	if ctx == nil {
		return nil
	}

	var result []string
	for _, columnInternalRef := range ctx.AllColumnInternalRef() {
		result = append(result, NormalizeMySQLIdentifier(columnInternalRef.Identifier()))
	}
	return result
}

func extractQueryExpression(ctx parser.IQueryExpressionParensContext) parser.IQueryExpressionContext {
	if ctx == nil {
		return nil
	}

	switch {
	case ctx.QueryExpression() != nil:
		return ctx.QueryExpression()
	case ctx.QueryExpressionParens() != nil:
		return extractQueryExpression(ctx.QueryExpressionParens())
	default:
		return nil
	}
}

func mysqlExtractJtColumn(ctx parser.IJtColumnContext) []string {
	if ctx == nil {
		return []string{}
	}

	switch {
	case ctx.Identifier() != nil:
		return []string{NormalizeMySQLIdentifier(ctx.Identifier())}
	case ctx.ColumnsClause() != nil:
		return mysqlExtractColumnsClause(ctx.ColumnsClause())
	default:
		return []string{}
	}
}

func extractTableRefs(database string, ctx antlr.ParserRuleContext) []base.SchemaResource {
	l := &resourceExtractListener{
		currentDatabase: database,
		resourceMap:     make(map[string]base.SchemaResource),
	}

	var result []base.SchemaResource
	antlr.ParseTreeWalkerDefault.Walk(l, ctx)
	for _, resource := range l.resourceMap {
		result = append(result, resource)
	}

	slices.SortFunc(result, func(i, j base.SchemaResource) int {
		if i.String() < j.String() {
			return -1
		}
		if i.String() > j.String() {
			return 1
		}
		return 0
	})

	return result
}

type resourceExtractListener struct {
	*parser.BaseMySQLParserListener

	currentDatabase string
	resourceMap     map[string]base.SchemaResource
}

// EnterTableRef is called when production tableRef is entered.
func (l *resourceExtractListener) EnterTableRef(ctx *parser.TableRefContext) {
	resource := base.SchemaResource{Database: l.currentDatabase}
	if ctx.DotIdentifier() != nil {
		resource.Table = NormalizeMySQLIdentifier(ctx.DotIdentifier().Identifier())
	}
	db, table := normalizeMySQLQualifiedIdentifier(ctx.QualifiedIdentifier())
	if db != "" {
		resource.Database = db
	}
	resource.Table = table
	l.resourceMap[resource.String()] = resource
}
