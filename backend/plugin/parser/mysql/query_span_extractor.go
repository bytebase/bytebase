package mysql

import (
	"context"
	"fmt"
	"strings"

	"github.com/antlr4-go/antlr/v4"
	mysql "github.com/bytebase/mysql-parser"
	parser "github.com/bytebase/mysql-parser"
	"github.com/pkg/errors"

	parsererror "github.com/bytebase/bytebase/backend/plugin/parser/errors"
	"github.com/bytebase/bytebase/backend/store/model"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

// querySpanExtractor is the extractor to extract the query span from a single statement.
type querySpanExtractor struct {
	ctx         context.Context
	connectedDB string

	listDBFunc func(ctx context.Context) ([]string, error)
	f          base.GetDatabaseMetadataFunc

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
}

// newQuerySpanExtractor creates a new query span extractor, the databaseMetadata and the ast are in the read guard.
func newQuerySpanExtractor(connectedDB string, getDatabaseMetadata base.GetDatabaseMetadataFunc, listAllDatabaseNames func(ctx context.Context) ([]string, error), ignoreCaseSensitive bool) *querySpanExtractor {
	return &querySpanExtractor{
		connectedDB:         connectedDB,
		f:                   getDatabaseMetadata,
		listDBFunc:          listAllDatabaseNames,
		ignoreCaseSensitive: ignoreCaseSensitive,
	}
}

func (q *querySpanExtractor) getQuerySpan(ctx context.Context, stmt string) (*base.QuerySpan, error) {
	q.ctx = ctx

	accessTables, err := getAccessTables(q.connectedDB, stmt)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get access tables from statement: %s", stmt)
	}
	// We do not support simultaneous access to the system table and the user table
	// because we do not synchronize the schema of the system table.
	// This causes an error (NOT_FOUND) when using querySpanExtractor.findTableSchema.
	// As a result, we exclude getting query span results for accessing only the system table.
	allSystems, mixed := isMixedQuery(accessTables, q.ignoreCaseSensitive)
	if mixed != nil {
		return nil, mixed
	}
	if allSystems {
		return &base.QuerySpan{
			Results:       []base.QuerySpanResult{},
			SourceColumns: base.SourceColumnSet{},
		}, nil
	}

	list, err := ParseMySQL(stmt)
	if err != nil {
		return nil, err
	}
	if len(list) == 0 {
		return &base.QuerySpan{
			Results:       []base.QuerySpanResult{},
			SourceColumns: make(base.SourceColumnSet),
		}, nil
	}
	if len(list) != 1 {
		return nil, errors.Errorf("expecting only one statement to get query span, but got %d", len(list))
	}

	// We assumes the caller had handled the statement type case,
	// so we only need to handle the determined statement type here.
	// In order to decrease the maintenance cost, we use listener to handle
	// the select statement precisely instead of using type switch.
	listener := newSelectOnlyListener(q)
	antlr.ParseTreeWalkerDefault.Walk(listener, list[0].Tree)

	if listener.err != nil {
		return nil, listener.err
	}
	return &base.QuerySpan{
		Results:       listener.querySpan.Results,
		SourceColumns: accessTables,
	}, nil
}

func (q *querySpanExtractor) extractContext(ctx antlr.ParserRuleContext) (*base.PseudoTable, error) {
	if ctx == nil {
		return nil, nil
	}

	switch ctx := ctx.(type) {
	case mysql.ISelectStatementContext:
		return q.extractSelectStatement(ctx)
	default:
		return nil, nil
	}
}

// extractSelectStatement extracts the table source from the select statement.
// It regards the result of the select statement as the pseudo table.
func (q *querySpanExtractor) extractSelectStatement(ctx mysql.ISelectStatementContext) (*base.PseudoTable, error) {
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
	}

	panic("unreachable")
}

func (q *querySpanExtractor) extractQueryExpression(ctx mysql.IQueryExpressionContext) (*base.PseudoTable, error) {
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
	}

	panic("unreachable")
}

func (q *querySpanExtractor) extractQueryExpressionParens(ctx mysql.IQueryExpressionParensContext) (*base.PseudoTable, error) {
	if ctx == nil {
		return nil, nil
	}

	switch {
	case ctx.QueryExpression() != nil:
		return q.extractQueryExpression(ctx.QueryExpression())
	case ctx.QueryExpressionParens() != nil:
		return q.extractQueryExpressionParens(ctx.QueryExpressionParens())
	}

	return nil, nil
}

func (q *querySpanExtractor) extractQueryExpressionBody(ctx mysql.IQueryExpressionBodyContext) (*base.PseudoTable, error) {
	var result base.TableSource
	unionNum := 0
	for _, child := range ctx.GetChildren() {
		switch child := child.(type) {
		case *mysql.QueryPrimaryContext:
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
		case *mysql.QueryExpressionParensContext:
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

func (q *querySpanExtractor) extractQueryPrimary(ctx mysql.IQueryPrimaryContext) (base.TableSource, error) {
	switch {
	case ctx.QuerySpecification() != nil:
		return q.extractQuerySpecification(ctx.QuerySpecification())
	case ctx.TableValueConstructor() != nil:
		return q.extractTableValueConstructor(ctx.TableValueConstructor())
	case ctx.ExplicitTable() != nil:
		return q.extractExplicitTable(ctx.ExplicitTable())
	}

	panic("unreachable")
}

func (q *querySpanExtractor) extractExplicitTable(ctx mysql.IExplicitTableContext) (base.TableSource, error) {
	if ctx == nil {
		return nil, nil
	}

	databaseName, tableName := NormalizeMySQLTableRef(ctx.TableRef())
	databaseName, tableSource, err := q.findTableSchema(databaseName, tableName)
	if err != nil {
		return nil, err
	}

	return tableSource, nil
}

func (q *querySpanExtractor) extractTableValueConstructor(ctx mysql.ITableValueConstructorContext) (*base.PseudoTable, error) {
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
		case *mysql.ExprContext:
			_, sourceColumns, err := q.extractSourceColumnSetFromExpr(child)
			if err != nil {
				return nil, err
			}
			columns = append(columns, base.QuerySpanResult{
				Name:          child.GetParser().GetTokenStream().GetTextFromRuleContext(child),
				SourceColumns: sourceColumns,
			})
		case antlr.TerminalNode:
			if child.GetSymbol().GetTokenType() == mysql.MySQLParserDEFAULT_SYMBOL {
				columns = append(columns, base.QuerySpanResult{
					Name:          "DEFAULT",
					SourceColumns: make(base.SourceColumnSet),
				})
			}
		}
	}

	return &base.PseudoTable{
		Name:    "",
		Columns: columns,
	}, nil
}

func (q *querySpanExtractor) extractQuerySpecification(ctx mysql.IQuerySpecificationContext) (*base.PseudoTable, error) {
	var fromSources []base.TableSource
	var err error
	if ctx.FromClause() != nil {
		originalLength := len(q.tableSourceFrom)
		defer func() {
			q.tableSourceFrom = q.tableSourceFrom[:originalLength]
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

func (q *querySpanExtractor) extractSelectItemList(ctx mysql.ISelectItemListContext, fromSpanResult []base.TableSource) ([]base.QuerySpanResult, error) {
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

func (q *querySpanExtractor) extractSelectItem(ctx mysql.ISelectItemContext) ([]base.QuerySpanResult, error) {
	if ctx == nil {
		return nil, nil
	}

	switch {
	case ctx.TableWild() != nil:
		return q.extractTableWild(ctx.TableWild())
	case ctx.Expr() != nil:
		fieldName, sourceColumns, err := q.extractSourceColumnSetFromExpr(ctx.Expr())
		if err != nil {
			return nil, err
		}
		if ctx.SelectAlias() != nil {
			fieldName = NormalizeMySQLIdentifier(ctx.SelectAlias().Identifier())
		} else if fieldName == "" {
			fieldName = ctx.GetParser().GetTokenStream().GetTextFromRuleContext(ctx)
		}

		return []base.QuerySpanResult{
			{
				Name:          fieldName,
				SourceColumns: sourceColumns,
			},
		}, nil
	}

	panic("unreachable")
}

func (q *querySpanExtractor) extractSourceColumnSetFromExpr(ctx antlr.ParserRuleContext) (string, base.SourceColumnSet, error) {
	if ctx == nil {
		return "", make(base.SourceColumnSet), nil
	}

	// The closure of expr rules.
	switch ctx := ctx.(type) {
	case mysql.ISubqueryContext:
		baseSet := make(base.SourceColumnSet)
		// Subquery in SELECT fields is special.
		// It can be the non-associated or associated subquery.
		// For associated subquery, we should set the fromFieldList as the outerSchemaInfo.
		// So that the subquery can access the outer schema.
		// The reason for new extractor is that we still need the current fromFieldList, overriding it is not expected.
		subqueryExtractor := &querySpanExtractor{
			ctx: q.ctx,
			// The connectedDB is the same as the outer query.
			connectedDB:       q.connectedDB,
			f:                 q.f,
			listDBFunc:        q.listDBFunc,
			ctes:              q.ctes,
			outerTableSources: append(q.outerTableSources, q.tableSourceFrom...),
			tableSourceFrom:   []base.TableSource{},
		}
		tableSource, err := subqueryExtractor.extractSubquery(ctx)
		if err != nil {
			return "", nil, err
		}
		spanResult := tableSource.GetQuerySpanResult()
		for _, field := range spanResult {
			baseSet, _ = base.MergeSourceColumnSet(field.SourceColumns, field.SourceColumns)
		}
		return "", baseSet, nil
	case mysql.IColumnRefContext:
		databaseName, tableName, fieldName := NormalizeMySQLFieldIdentifier(ctx.FieldIdentifier())
		sourceColumnSet, err := q.getFieldColumnSource(databaseName, tableName, fieldName)
		if err != nil {
			return "", nil, err
		}
		return fieldName, sourceColumnSet, nil
	}

	var list []antlr.ParserRuleContext
	for _, child := range ctx.GetChildren() {
		if child, ok := child.(antlr.ParserRuleContext); ok {
			list = append(list, child)
		}
	}

	fieldName, sourceColumnSet, err := q.extractSourceColumnSetFromExprList(list)
	if err != nil {
		return "", nil, err
	}
	if len(ctx.GetChildren()) > 1 {
		fieldName = ""
	}
	return fieldName, sourceColumnSet, nil
}

func (q *querySpanExtractor) extractSourceColumnSetFromExprList(ctxs []antlr.ParserRuleContext) (string, base.SourceColumnSet, error) {
	baseSet := make(base.SourceColumnSet)
	var fieldName string
	var set base.SourceColumnSet
	var err error
	for _, ctx := range ctxs {
		fieldName, set, err = q.extractSourceColumnSetFromExpr(ctx)
		if err != nil {
			return "", nil, err
		}
		baseSet, _ = base.MergeSourceColumnSet(baseSet, set)
	}

	return fieldName, baseSet, nil
}

func (q *querySpanExtractor) extractTableWild(ctx mysql.ITableWildContext) ([]base.QuerySpanResult, error) {
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
		return nil, &parsererror.ResourceNotFoundError{
			Err:      errors.Errorf("failed to find table to calculate asterisk"),
			Database: &databaseName,
			Table:    &tableName,
		}
	}
	return querySpanResults, nil
}

// extractTableSourcesFromFromClause extracts the table sources from the from clause.
// The result can be empty while the from clause is dual.
func (q *querySpanExtractor) extractTableSourcesFromFromClause(ctx mysql.IFromClauseContext) ([]base.TableSource, error) {
	// DUAL is purely for the convenience of people who require that all SELECT statements should have FROM and possibly other clauses.
	// MySQL may ignore the clauses. MySQL does not require FROM DUAL if no tables are referenced.
	if ctx.DUAL_SYMBOL() != nil {
		return []base.TableSource{}, nil
	}

	return q.extractTableReferenceList(ctx.TableReferenceList())
}

func (q *querySpanExtractor) extractTableReferenceList(ctx mysql.ITableReferenceListContext) ([]base.TableSource, error) {
	var result []base.TableSource
	for _, tableReference := range ctx.AllTableReference() {
		tableResource, err := q.extractTableReference(tableReference)
		if err != nil {
			return nil, err
		}
		result = append(result, tableResource)
	}

	return result, nil
}

func (q *querySpanExtractor) extractTableReference(ctx mysql.ITableReferenceContext) (base.TableSource, error) {
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
func (q *querySpanExtractor) extractJoinedTable(l base.TableSource, r mysql.IJoinedTableContext) (base.TableSource, error) {
	rightTableSource, err := q.extractTableReference(r.TableReference())
	if err != nil {
		return nil, err
	}
	q.tableSourceFrom = append(q.tableSourceFrom, rightTableSource)

	tp := JOIN

	if v := r.InnerJoinType(); v != nil {
		if v.INNER_SYMBOL() != nil {
			tp = INNER_JOIN
		} else if v.CROSS_SYMBOL() != nil {
			tp = CROSS_JOIN
		} else if v.STRAIGHT_JOIN_SYMBOL() != nil {
			tp = STRAIGHT_JOIN
		}
	} else if v := r.OuterJoinType(); v != nil {
		if v.LEFT_SYMBOL() != nil {
			tp = LEFT_OUTER_JOIN
		} else if v.RIGHT_SYMBOL() != nil {
			tp = RIGHT_OUTER_JOIN
		}
	} else if v := r.NaturalJoinType(); v != nil {
		tp = NATRUAL_INNER_JOIN
		if v.LEFT_SYMBOL() != nil {
			tp = NATRUAL_LEFT_OUTER_JOIN
		} else if v.RIGHT_SYMBOL() != nil {
			tp = NATRUAL_RIGHT_OUTER_JOIN
		}
	}

	var usingIdentifiers []string
	if r.IdentifierListWithParentheses() != nil && r.IdentifierListWithParentheses().IdentifierList() != nil {
		usingIdentifiers = NormalizeMySQLIdentifierList(r.IdentifierListWithParentheses().IdentifierList())
	}

	joinedTableSource, err := q.joinTableSources(l, rightTableSource, tp, usingIdentifiers)
	if err != nil {
		return nil, err
	}

	return joinedTableSource, nil
}

type joinType int

const (
	JOIN joinType = iota
	INNER_JOIN
	CROSS_JOIN
	STRAIGHT_JOIN
	LEFT_OUTER_JOIN
	RIGHT_OUTER_JOIN
	NATRUAL_INNER_JOIN
	NATRUAL_LEFT_OUTER_JOIN
	NATRUAL_RIGHT_OUTER_JOIN
)

// joinTableSources joins the left and right table sources with the join type.
func (q *querySpanExtractor) joinTableSources(l, r base.TableSource, tp joinType, using []string) (base.TableSource, error) {
	switch tp {
	case JOIN, INNER_JOIN, CROSS_JOIN, STRAIGHT_JOIN, LEFT_OUTER_JOIN, RIGHT_OUTER_JOIN:
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
		}, nil
	case NATRUAL_INNER_JOIN, NATRUAL_LEFT_OUTER_JOIN, NATRUAL_RIGHT_OUTER_JOIN:
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
		}, nil
	}

	return nil, nil
}

func (q *querySpanExtractor) extractTableFactor(ctx mysql.ITableFactorContext) (base.TableSource, error) {
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
			result, err = q.joinTableSources(result, tableSource, CROSS_JOIN, nil)
			if err != nil {
				return nil, err
			}
		}
		return result, nil
	case ctx.TableFunction() != nil:
		return q.extractTableFunction(ctx.TableFunction())
	}

	panic("unreachable")
}

func (q *querySpanExtractor) extractTableFunction(ctx mysql.ITableFunctionContext) (base.TableSource, error) {
	if ctx == nil {
		return nil, nil
	}

	name, sourceColumnSet, err := q.extractSourceColumnSetFromExpr(ctx.Expr())
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
		})
	}

	return &base.PseudoTable{
		Name:    name,
		Columns: result,
	}, nil
}

func (q *querySpanExtractor) extractTableReferenceListParens(ctx mysql.ITableReferenceListParensContext) ([]base.TableSource, error) {
	switch {
	case ctx.TableReferenceList() != nil:
		return q.extractTableReferenceList(ctx.TableReferenceList())
	case ctx.TableReferenceListParens() != nil:
		return q.extractTableReferenceListParens(ctx.TableReferenceListParens())
	}

	panic("unreachable")
}

func (q *querySpanExtractor) extractSubquery(ctx mysql.ISubqueryContext) (*base.PseudoTable, error) {
	return q.extractQueryExpressionParens(ctx.QueryExpressionParens())
}

func (q *querySpanExtractor) extractDerivedTable(ctx mysql.IDerivedTableContext) (base.TableSource, error) {
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

func extractColumnInternalRefList(ctx mysql.IColumnInternalRefListContext) []string {
	if ctx == nil {
		return nil
	}

	var result []string
	for _, columnInternalRef := range ctx.AllColumnInternalRef() {
		result = append(result, NormalizeMySQLIdentifier(columnInternalRef.Identifier()))
	}
	return result
}

func (q *querySpanExtractor) extractSingleTable(ctx mysql.ISingleTableContext) (base.TableSource, error) {
	databaseName, tableName := NormalizeMySQLTableRef(ctx.TableRef())
	databaseName, tableSource, err := q.findTableSchema(databaseName, tableName)
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

func (q *querySpanExtractor) extractSingleTableParens(ctx mysql.ISingleTableParensContext) (base.TableSource, error) {
	switch {
	case ctx.SingleTable() != nil:
		return q.extractSingleTable(ctx.SingleTable())
	case ctx.SingleTableParens() != nil:
		return q.extractSingleTableParens(ctx.SingleTableParens())
	}

	panic("unreachable")
}

// extractCommonTableExpression extracts the pseudo table from the common table expression.
func (q *querySpanExtractor) extractCommonTableExpression(ctx mysql.ICommonTableExpressionContext, recursive bool) (*base.PseudoTable, error) {
	if recursive {
		return q.extractRecursiveCTE(ctx)
	}
	return q.extractNonRecursiveCTE(ctx)
}

// extractRecursiveCTE extracts the pseudo table from the recursive common table expression.
func (q *querySpanExtractor) extractRecursiveCTE(ctx mysql.ICommonTableExpressionContext) (*base.PseudoTable, error) {
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
func (q *querySpanExtractor) extractNonRecursiveCTE(ctx mysql.ICommonTableExpressionContext) (*base.PseudoTable, error) {
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
	*mysql.BaseMySQLParserListener

	extractor                     *querySpanExtractor
	cteInfo                       *base.PseudoTable
	selfName                      string
	outerCTEs                     []mysql.IWithClauseContext
	foundFirstQueryExpressionBody bool
	inCTE                         bool
	err                           error
}

// EnterQueryExpression is called when production queryExpression is entered.
func (l *recursiveCTEExtractListener) EnterQueryExpression(ctx *mysql.QueryExpressionContext) {
	if l.foundFirstQueryExpressionBody || l.inCTE || l.err != nil {
		return
	}
	if ctx.WithClause() != nil {
		l.outerCTEs = append(l.outerCTEs, ctx.WithClause())
	}
}

// EnterCommonTableExpression is called when production commonTableExpression is entered.
func (l *recursiveCTEExtractListener) EnterWithClause(_ *mysql.WithClauseContext) {
	l.inCTE = true
}

// ExitCommonTableExpression is called when production commonTableExpression is exited.
func (l *recursiveCTEExtractListener) ExitWithClause(_ *mysql.WithClauseContext) {
	l.inCTE = false
}

// EnterQueryExpressionBody is called when production queryExpressionBody is entered.
func (l *recursiveCTEExtractListener) EnterQueryExpressionBody(ctx *mysql.QueryExpressionBodyContext) {
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
		case *mysql.QueryPrimaryContext:
			if !findRecursivePart {
				resource, err := ExtractResourceList("", "", child.GetParser().GetTokenStream().GetTextFromRuleContext(child))
				if err != nil {
					l.err = err
					return
				}

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
		case *mysql.QueryExpressionParensContext:
			queryExpression := extractQueryExpression(child)
			if queryExpression == nil {
				// Never happen.
				l.err = errors.Errorf("MySQL query expression parens should have query expression, but got nil")
				return
			}

			if !findRecursivePart {
				resource, err := ExtractResourceList("", "", queryExpression.GetParser().GetTokenStream().GetTextFromRuleContext(queryExpression))
				if err != nil {
					l.err = err
					return
				}

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
			case *mysql.QueryPrimaryContext:
				var err error
				tableSource, err := l.extractor.extractQueryPrimary(item)
				if err != nil {
					l.err = err
					return
				}
				itemFields = tableSource.GetQuerySpanResult()
			case *mysql.QueryExpressionContext:
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
			if databaseName != "" && tableName != tableSource.GetTableName() {
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

	return nil, &parsererror.ResourceNotFoundError{
		Database: &databaseName,
		Table:    &tableName,
		Column:   &fieldName,
	}
}

func (q *querySpanExtractor) findTableSchema(databaseName, tableName string) (string, base.TableSource, error) {
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
				return "", table, nil
			}
		}
	}

	if databaseName == "" {
		databaseName = q.connectedDB
	}

	var dbSchema *model.DatabaseMetadata
	allDatabaseNames, err := q.listDBFunc(q.ctx)
	if err != nil {
		return "", nil, errors.Wrap(err, "failed to list databases")
	}
	if q.ignoreCaseSensitive {
		for _, db := range allDatabaseNames {
			if strings.EqualFold(db, databaseName) {
				_, dbSchema, err = q.f(q.ctx, db)
				if err != nil {
					return "", nil, errors.Wrapf(err, "failed to get database metadata for database %q", db)
				}
				break
			}
		}
	} else {
		for _, db := range allDatabaseNames {
			if db == databaseName {
				_, dbSchema, err = q.f(q.ctx, db)
				if err != nil {
					return "", nil, errors.Wrapf(err, "failed to get database metadata for database %q", db)
				}
				break
			}
		}
	}
	if dbSchema == nil {
		return "", nil, &parsererror.ResourceNotFoundError{
			Database: &databaseName,
		}
	}

	emptySchema := ""
	schema := dbSchema.GetSchema(emptySchema)
	if schema == nil {
		return "", nil, &parsererror.ResourceNotFoundError{
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
		columnNames := make([]string, 0, len(tableSchema.GetColumns()))
		for _, column := range tableSchema.GetColumns() {
			columnNames = append(columnNames, column.Name)
		}
		return databaseName, &base.PhysicalTable{
			Name:     tableName,
			Schema:   emptySchema,
			Database: databaseName,
			Server:   "",
			Columns:  columnNames,
		}, nil
	}

	var viewSchema *model.ViewMetadata
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
			return "", nil, errors.Wrapf(err, "failed to get columns for view %q", tableName)
		}
		return databaseName, &base.PhysicalView{
			Name:     tableName,
			Schema:   emptySchema,
			Database: databaseName,
			Server:   "",
			Columns:  columns,
		}, nil
	}

	return "", nil, &parsererror.ResourceNotFoundError{
		Database: &databaseName,
		Schema:   &emptySchema,
		Table:    &tableName,
	}
}

func (q *querySpanExtractor) getColumnsForView(definition string) ([]base.QuerySpanResult, error) {
	newQ := newQuerySpanExtractor(q.connectedDB, q.f, q.listDBFunc, q.ignoreCaseSensitive)
	span, err := newQ.getQuerySpan(q.ctx, definition)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get query span for view")
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

	if _, ok := parent.(*mysql.SimpleStatementContext); !ok {
		return
	}

	fields, err := s.extractor.extractContext(ctx)
	if err != nil {
		s.err = err
		return
	}

	s.querySpan.Results = append(s.querySpan.Results, fields.Columns...)
	return
}

func getAccessTables(currentDatabase string, statement string) (base.SourceColumnSet, error) {
	treeList, err := ParseMySQL(statement)
	if err != nil {
		return nil, err
	}

	l := &accessTableListener{
		currentDatabase: currentDatabase,
		sourceColumnSet: make(base.SourceColumnSet),
	}

	result := make(base.SourceColumnSet)
	for _, tree := range treeList {
		if tree == nil {
			continue
		}
		antlr.ParseTreeWalkerDefault.Walk(l, tree.Tree)
		result, _ = base.MergeSourceColumnSet(result, l.sourceColumnSet)
	}

	return result, nil
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
func isMixedQuery(m base.SourceColumnSet, ignoreCaseSensitive bool) (allSystems bool, mixed error) {
	userMsg, systemMsg := "", ""
	for table := range m {
		if msg := isSystemResource(table, ignoreCaseSensitive); msg != "" {
			systemMsg = msg
			continue
		}
		userMsg = fmt.Sprintf("user table %q.%q", table.Schema, table.Table)
		if systemMsg != "" {
			return false, errors.Errorf("cannot access %s and %s at the same time", userMsg, systemMsg)
		}
	}

	if userMsg != "" && systemMsg != "" {
		return false, errors.Errorf("cannot access %s and %s at the same time", userMsg, systemMsg)
	}

	return userMsg == "" && systemMsg != "", nil
}

func isSystemResource(resource base.ColumnResource, ignoreCaseSensitive bool) string {
	if ignoreCaseSensitive {
		if strings.EqualFold(resource.Database, "information_schema") {
			return fmt.Sprintf("system schema %q", resource.Table)
		}
		if strings.EqualFold(resource.Database, "performance_schema") {
			return fmt.Sprintf("system schema %q", resource.Table)
		}
		if strings.EqualFold(resource.Database, "mysql") {
			return fmt.Sprintf("system schema %q", resource.Table)
		}
	} else {
		if resource.Database == "information_schema" {
			return fmt.Sprintf("system schema %q", resource.Table)
		}
		if resource.Database == "performance_schema" {
			return fmt.Sprintf("system schema %q", resource.Table)
		}
		if resource.Database == "mysql" {
			return fmt.Sprintf("system schema %q", resource.Table)
		}
	}
	return ""
}
