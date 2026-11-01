package mysql

import (
	"strings"

	"github.com/bytebase/omni/mysql/ast"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

// clause is where a name sits in a SELECT, which decides what it can resolve
// to. query_span_predicate_test.go checks each rule against MySQL and MariaDB.
type clause int

const (
	// clauseSelectList, clauseFrom, and clauseWhere see the FROM clause only.
	clauseSelectList clause = iota
	clauseFrom
	clauseWhere
	// clauseGroupBy sees the FROM clause, then the select list.
	clauseGroupBy
	// clauseHaving sees a GROUP BY column, then the select list, then the FROM
	// clause; inside an aggregate it resolves like clauseGroupBy.
	clauseHaving
	// clauseOrderBy resolves no names itself, since the ORDER BY keys are not
	// predicates; queries nested in it see the select list.
	clauseOrderBy
)

// outerScope is what an enclosing SELECT shows a query nested in it.
type outerScope struct {
	tables []base.TableSource
	// items is the select list the nested query can name, set only where
	// MySQL lets it: the select list (items before it), GROUP BY, HAVING, and
	// ORDER BY.
	items []base.QuerySpanResult
}

// selectPosition is the per-SELECT state a nested SELECT saves and restores.
type selectPosition struct {
	clause      clause
	selectItems []base.QuerySpanResult
	groupBy     []ast.ExprNode
	inAggregate bool
	inCondition bool
}

func (q *querySpanExtractor) position() selectPosition {
	return selectPosition{
		clause:      q.clause,
		selectItems: q.selectItems,
		groupBy:     q.groupBy,
		inAggregate: q.inAggregate,
		inCondition: q.inCondition,
	}
}

func (q *querySpanExtractor) setPosition(p selectPosition) {
	q.clause = p.clause
	q.selectItems = p.selectItems
	q.groupBy = p.groupBy
	q.inAggregate = p.inAggregate
	q.inCondition = p.inCondition
}

func (q *querySpanExtractor) selectItemsVisibleToSubquery() []base.QuerySpanResult {
	switch q.clause {
	case clauseSelectList, clauseGroupBy, clauseHaving, clauseOrderBy:
		return q.selectItems
	default:
		return nil
	}
}

// resolveInCurrentSelect resolves a name in the SELECT being extracted.
func (q *querySpanExtractor) resolveInCurrentSelect(databaseName, tableName, fieldName string) (base.SourceColumnSet, bool) {
	fromClause := func(databaseName, tableName string) (base.SourceColumnSet, bool) {
		if columns, ok := q.findColumnInTables(q.tableSourceFrom, databaseName, tableName, fieldName); ok {
			return columns, true
		}
		return q.findColumnInTables(q.priorTableInFrom, databaseName, tableName, fieldName)
	}
	if databaseName != "" || tableName != "" {
		return fromClause(databaseName, tableName)
	}
	switch {
	case q.clause == clauseHaving && !q.inAggregate:
		// MySQL warns when a GROUP BY column and a select alias share the name
		// and takes the column; MariaDB refuses the query.
		if group := q.groupByColumnNamed(fieldName); group != nil {
			if columns, ok := fromClause(group.Schema, group.Table); ok {
				return columns, true
			}
		}
		if columns, ok := selectItemsNamed(q.selectItems, fieldName); ok {
			return columns, true
		}
		return fromClause("", "")
	case q.clause == clauseHaving, q.clause == clauseGroupBy:
		if columns, ok := fromClause("", ""); ok {
			return columns, true
		}
		return selectItemsNamed(q.selectItems, fieldName)
	default:
		return fromClause("", "")
	}
}

func (q *querySpanExtractor) groupByColumnNamed(name string) *ast.ColumnRef {
	for _, expr := range q.groupBy {
		if ref, ok := expr.(*ast.ColumnRef); ok && !ref.Star && strings.EqualFold(ref.Column, name) {
			return ref
		}
	}
	return nil
}

// selectItemsNamed returns the sources of every select item called name. Two
// different items with one name make MySQL refuse the query.
func selectItemsNamed(items []base.QuerySpanResult, name string) (base.SourceColumnSet, bool) {
	columns := make(base.SourceColumnSet)
	found := false
	for _, item := range items {
		if strings.EqualFold(item.Name, name) {
			mergeSourceColumnSetInto(columns, item.SourceColumns)
			found = true
		}
	}
	return columns, found
}

// collectConditionColumns records the columns expr reads, where the database
// evaluates it on unmasked values to decide which rows come back.
func (q *omniQuerySpanExtractor) collectConditionColumns(expr ast.ExprNode) {
	if expr == nil {
		return
	}
	inCondition := q.inCondition
	q.inCondition = true
	field, err := q.extractOmniExpr(expr)
	q.inCondition = inCondition
	if err != nil {
		q.recordColumnsNamedIn(expr)
		return
	}
	mergeSourceColumnSetInto(q.predicateColumns, field.SourceColumns)
}

// collectNestedQueryColumns records the predicate columns of the queries
// nested in node, whose own value only groups or orders the rows.
func (q *omniQuerySpanExtractor) collectNestedQueryColumns(node ast.Node) {
	if node == nil {
		return
	}
	ast.Inspect(node, func(n ast.Node) bool {
		sel, ok := n.(*ast.SelectStmt)
		if !ok {
			return true
		}
		if _, _, err := q.extractOmniSelectSourceColumns(sel); err != nil {
			q.recordColumnsNamedIn(sel)
		}
		return false
	})
}

// collectTrailingClauses records what ORDER BY and LIMIT decide: the queries
// nested in ORDER BY, and the row count. The ORDER BY keys only order rows.
func (q *omniQuerySpanExtractor) collectTrailingClauses(orderBy []*ast.OrderByItem, limit *ast.Limit) {
	q.clause = clauseOrderBy
	for _, item := range orderBy {
		if item != nil {
			q.collectNestedQueryColumns(item.Expr)
		}
	}
	if limit != nil {
		q.collectConditionColumns(limit.Count)
		q.collectConditionColumns(limit.Offset)
	}
}

// collectExplainedQueryColumns records the predicate columns of the query
// EXPLAIN ANALYZE runs, whose plan reports the rows each condition kept. The
// plan is not a result set, so the query's lineage is dropped.
func (q *omniQuerySpanExtractor) collectExplainedQueryColumns(stmt ast.StmtNode) {
	if _, err := q.extractOmniSelectRoot(stmt); err != nil {
		q.recordColumnsNamedIn(stmt)
	}
}

func (q *omniQuerySpanExtractor) recordJoinedColumn(table base.TableSource, name string) {
	if field, ok := querySpanResultNamed(table.GetQuerySpanResult(), name); ok {
		mergeSourceColumnSetInto(q.predicateColumns, field.SourceColumns)
		return
	}
	q.recordColumnsNamed("", "", name)
}

// recordColumnsNamedIn over-approximates the columns node reads by name, for
// a condition the extractor could not resolve.
func (q *omniQuerySpanExtractor) recordColumnsNamedIn(node ast.Node) {
	ast.Inspect(node, func(n ast.Node) bool {
		if ref, ok := n.(*ast.ColumnRef); ok && !ref.Star {
			q.recordColumnsNamed(ref.Schema, ref.Table, ref.Column)
		}
		return true
	})
}

// recordColumnsNamed records every column called fieldName that the reference
// could mean: in any visible table or select list, and in any table the
// statement reads. A qualifier that names none of them is ignored.
func (q *omniQuerySpanExtractor) recordColumnsNamed(databaseName, tableName, fieldName string) {
	qualifierMatched := false
	recordTables := func(tables []base.TableSource) {
		for _, table := range tables {
			if !q.tableSourceMatches(table, databaseName, tableName) {
				continue
			}
			qualifierMatched = true
			for _, column := range table.GetQuerySpanResult() {
				if strings.EqualFold(column.Name, fieldName) {
					mergeSourceColumnSetInto(q.predicateColumns, column.SourceColumns)
				}
			}
		}
	}
	for _, tables := range q.tableScopes() {
		recordTables(tables)
	}
	if databaseName == "" && tableName == "" {
		columns, _ := selectItemsNamed(q.selectItems, fieldName)
		mergeSourceColumnSetInto(q.predicateColumns, columns)
		for _, scope := range q.outerScopes {
			columns, _ := selectItemsNamed(scope.items, fieldName)
			mergeSourceColumnSetInto(q.predicateColumns, columns)
		}
	}

	for access := range q.accessTables {
		if access.Table == "" {
			continue
		}
		if tableName != "" && qualifierMatched && !q.tableSourceMatches(&base.PhysicalTable{Database: access.Database, Name: access.Table}, databaseName, tableName) {
			continue
		}
		for _, column := range q.metadataColumnsNamed(access.Database, access.Table, fieldName) {
			q.predicateColumns[base.ColumnResource{Database: access.Database, Table: access.Table, Column: column}] = true
		}
	}
}

func (q *omniQuerySpanExtractor) metadataColumnsNamed(databaseName, tableName, fieldName string) []string {
	if q.gCtx.GetDatabaseMetadataFunc == nil {
		return nil
	}
	_, metadata, err := q.gCtx.GetDatabaseMetadataFunc(q.ctx, q.gCtx.InstanceID, databaseName)
	if err != nil || metadata == nil {
		return nil
	}
	schema := metadata.GetSchemaMetadata("")
	if schema == nil {
		return nil
	}
	var names []string
	for _, name := range schema.ListTableNames() {
		if name != tableName && (!q.ignoreCaseSensitive || !strings.EqualFold(name, tableName)) {
			continue
		}
		for _, column := range schema.GetTable(name).GetProto().GetColumns() {
			if strings.EqualFold(column.Name, fieldName) {
				names = append(names, column.Name)
			}
		}
	}
	return names
}

// isAggregateFunction reports whether MySQL resolves a HAVING name inside the
// function as a GROUP BY name. ANY_VALUE is not one of them.
func isAggregateFunction(name string) bool {
	switch strings.ToLower(name) {
	case "avg", "bit_and", "bit_or", "bit_xor", "count", "group_concat",
		"json_arrayagg", "json_objectagg", "max", "min", "std", "stddev",
		"stddev_pop", "stddev_samp", "sum", "var_pop", "var_samp", "variance":
		return true
	default:
		return false
	}
}

func mergeSourceColumnSetInto(dst, src base.SourceColumnSet) {
	for column := range src {
		dst[column] = true
	}
}
