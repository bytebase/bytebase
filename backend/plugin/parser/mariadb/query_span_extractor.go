package mariadb

import (
	"context"
	"slices"
	"strconv"
	"strings"

	"github.com/bytebase/omni/mariadb/ast"
	metadatapb "github.com/bytebase/omni/metadata"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/store/model"
)

type querySpanExtractor struct {
	ctx             context.Context
	defaultDatabase string

	gCtx base.GetQuerySpanContext

	ignoreCaseSensitive bool

	// ctes records common table expressions visible to the current select scope.
	ctes []*base.PseudoTable

	// outerScopes resolves correlated subquery column references, innermost last.
	outerScopes []outerScope

	// tableSourceFrom is the set of table sources visible from the current FROM clause.
	tableSourceFrom []base.TableSource

	// priorTableInFrom resolves JSON_TABLE document expressions against preceding FROM items.
	priorTableInFrom []base.TableSource

	// clause, selectItems, groupBy, and inAggregate are the position in the
	// current SELECT that decides how an unqualified name resolves.
	clause      clause
	selectItems []base.QuerySpanResult
	groupBy     []ast.ExprNode
	inAggregate bool
	// inCondition and outputInCondition hide an EXISTS subquery's output,
	// which decides nothing: inCondition within a condition of this SELECT,
	// outputInCondition when this query's own output feeds a condition.
	inCondition       bool
	outputInCondition bool

	// predicateColumns is shared with the extractors of nested queries.
	predicateColumns base.SourceColumnSet
	// ctePredicateColumns holds each CTE's predicate columns until a query
	// reads the CTE; MySQL never evaluates an unreferenced CTE.
	ctePredicateColumns map[*base.PseudoTable]base.SourceColumnSet
	// accessTables are the tables the statement reads, for the name fallback.
	accessTables base.SourceColumnSet

	viewResolutionStack map[string]bool
	// source is the original SQL text, used to slice expression names via Loc.
	source string
}

func newQuerySpanExtractor(defaultDatabase string, gCtx base.GetQuerySpanContext, ignoreCaseSensitive bool) *querySpanExtractor {
	return &querySpanExtractor{
		defaultDatabase:     defaultDatabase,
		gCtx:                gCtx,
		ignoreCaseSensitive: ignoreCaseSensitive,
		predicateColumns:    make(base.SourceColumnSet),
		ctePredicateColumns: make(map[*base.PseudoTable]base.SourceColumnSet),
	}
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

// joinTableSources returns the columns a join exposes. A column merged by
// USING or NATURAL is COALESCE(left, right), and in matched rows both sides
// hold its value, so it carries both sides' sources.
func joinTableSources(l, r base.TableSource, tp joinType, using []string) base.TableSource {
	switch tp {
	case Join, InnerJoin, CrossJoin, StraightJoin, LeftOuterJoin, RightOuterJoin,
		NaturalInnerJoin, NaturalLeftOuterJoin, NaturalRightOuterJoin:
	default:
		return nil
	}
	joined := make(map[string]bool)
	for _, name := range joinedColumnNames(l, r, tp, using) {
		joined[strings.ToLower(name)] = true
	}
	rightColumns := r.GetQuerySpanResult()
	var columns []base.QuerySpanResult
	for _, field := range l.GetQuerySpanResult() {
		if joined[strings.ToLower(field.Name)] {
			if rightField, ok := querySpanResultNamed(rightColumns, field.Name); ok {
				field.SourceColumns, _ = base.MergeSourceColumnSet(field.SourceColumns, rightField.SourceColumns)
			}
		}
		columns = append(columns, field)
	}
	for _, field := range rightColumns {
		if !joined[strings.ToLower(field.Name)] {
			columns = append(columns, field)
		}
	}
	return &base.PseudoTable{
		Columns: columns,
	}
}

// joinedColumnNames returns the columns a join merges: the USING list, or for
// a NATURAL join every column name both sides share.
func joinedColumnNames(l, r base.TableSource, tp joinType, using []string) []string {
	switch tp {
	case NaturalInnerJoin, NaturalLeftOuterJoin, NaturalRightOuterJoin:
		var names []string
		rightColumns := r.GetQuerySpanResult()
		for _, field := range l.GetQuerySpanResult() {
			if _, ok := querySpanResultNamed(rightColumns, field.Name); ok {
				names = append(names, field.Name)
			}
		}
		return names
	default:
		return using
	}
}

func querySpanResultNamed(results []base.QuerySpanResult, name string) (base.QuerySpanResult, bool) {
	for _, result := range results {
		if strings.EqualFold(result.Name, name) {
			return result, true
		}
	}
	return base.QuerySpanResult{}, false
}

func (q *querySpanExtractor) getAllTableColumnSources(databaseName, tableName string) ([]base.QuerySpanResult, bool) {
	for _, tables := range q.tableScopes() {
		for i := len(tables) - 1; i >= 0; i-- {
			if q.tableSourceMatches(tables[i], databaseName, tableName) {
				return tables[i].GetQuerySpanResult(), true
			}
		}
	}
	return nil, false
}

// getFieldColumnSource resolves a column reference as MySQL does: the current
// SELECT first, by the rules of the clause the reference is in, then each
// enclosing SELECT outward, its tables before the select list it exposes.
func (q *querySpanExtractor) getFieldColumnSource(databaseName, tableName, fieldName string) (base.SourceColumnSet, error) {
	if sourceColumnSet, ok := q.resolveInCurrentSelect(databaseName, tableName, fieldName); ok {
		return sourceColumnSet, nil
	}
	for i := len(q.outerScopes) - 1; i >= 0; i-- {
		scope := q.outerScopes[i]
		if sourceColumnSet, ok := q.findColumnInTables(scope.tables, databaseName, tableName, fieldName); ok {
			return sourceColumnSet, nil
		}
		if databaseName == "" && tableName == "" {
			if sourceColumnSet, ok := selectItemsNamed(scope.items, fieldName); ok {
				return sourceColumnSet, nil
			}
		}
	}

	return nil, &base.ResourceNotFoundError{
		Database: &databaseName,
		Table:    &tableName,
		Column:   &fieldName,
	}
}

// tableScopes returns the table sources a name can resolve to, nearest first.
func (q *querySpanExtractor) tableScopes() [][]base.TableSource {
	scopes := [][]base.TableSource{q.tableSourceFrom, q.priorTableInFrom}
	for i := len(q.outerScopes) - 1; i >= 0; i-- {
		scopes = append(scopes, q.outerScopes[i].tables)
	}
	return scopes
}

func (q *querySpanExtractor) tableSourceMatches(tableSource base.TableSource, databaseName, tableName string) bool {
	if q.ignoreCaseSensitive {
		if databaseName != "" && !strings.EqualFold(databaseName, tableSource.GetDatabaseName()) {
			return false
		}
		if tableName != "" && !strings.EqualFold(tableName, tableSource.GetTableName()) {
			return false
		}
		return true
	}
	if databaseName != "" && databaseName != tableSource.GetDatabaseName() {
		return false
	}
	if tableName != "" && tableName != tableSource.GetTableName() {
		return false
	}
	return true
}

// findColumnInTables searches tables last to first, so a join's merged
// columns, appended after its operands, win for an unqualified name.
func (q *querySpanExtractor) findColumnInTables(tables []base.TableSource, databaseName, tableName, fieldName string) (base.SourceColumnSet, bool) {
	for i := len(tables) - 1; i >= 0; i-- {
		if !q.tableSourceMatches(tables[i], databaseName, tableName) {
			continue
		}
		for _, column := range tables[i].GetQuerySpanResult() {
			if strings.EqualFold(column.Name, fieldName) {
				return column.SourceColumns, true
			}
		}
	}
	return nil, false
}

func (q *querySpanExtractor) findTableSchema(databaseName, tableName string) (base.TableSource, error) {
	if databaseName == "" {
		for i := len(q.ctes) - 1; i >= 0; i-- {
			table := q.ctes[i]
			if table.Name == tableName {
				mergeSourceColumnSetInto(q.predicateColumns, q.ctePredicateColumns[table])
				return table, nil
			}
		}
	}

	if databaseName == "" {
		databaseName = q.defaultDatabase
	}

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

	var viewSchema *metadatapb.ViewMetadata
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
		columns, err := q.getColumnsForView(dbMetadata.GetProto().GetName(), viewSchema.Name, viewSchema.Definition)
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

func (q *querySpanExtractor) getColumnsForView(databaseName, viewName, definition string) ([]base.QuerySpanResult, error) {
	key := mysqlViewResolutionKey(databaseName, viewName)
	if q.viewResolutionStack[key] {
		return nil, errors.Errorf("cyclic view reference detected while resolving %q", viewName)
	}

	newQ := newQuerySpanExtractor(q.defaultDatabase, q.gCtx, q.ignoreCaseSensitive)
	newQ.viewResolutionStack = cloneViewResolutionStack(q.viewResolutionStack)
	newQ.viewResolutionStack[key] = true
	span, err := newQ.getQuerySpan(q.ctx, definition)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get query span for view")
	}
	if span.NotFoundError != nil {
		return nil, span.NotFoundError
	}
	return span.Results, nil
}

func mysqlViewResolutionKey(databaseName, viewName string) string {
	return databaseName + "\x00" + viewName
}

func cloneViewResolutionStack(in map[string]bool) map[string]bool {
	out := make(map[string]bool, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

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

var reservedSystemDatabases = map[string]bool{
	"information_schema": true,
	"performance_schema": true,
}

var onDiskSystemDatabases = map[string]bool{
	"mysql": true,
}

func isSystemResource(resource base.ColumnResource, ignoreCaseSensitive bool) bool {
	if reservedSystemDatabases[strings.ToLower(resource.Database)] {
		return true
	}
	database := resource.Database
	if ignoreCaseSensitive {
		database = strings.ToLower(database)
	}
	return onDiskSystemDatabases[database]
}

// getQuerySpan is the package-internal entry point behind GetQuerySpan, and
// what the golden harness drives directly.
func (q *querySpanExtractor) getQuerySpan(ctx context.Context, statement string) (*base.QuerySpan, error) {
	q.ctx = ctx
	q.source = statement

	stmts, err := ParseMariaDB(statement)
	if err != nil {
		return nil, err
	}
	if stmts == nil || len(stmts.Items) == 0 {
		return &base.QuerySpan{
			Type:             base.Select,
			SourceColumns:    base.SourceColumnSet{},
			Results:          []base.QuerySpanResult{},
			PredicateColumns: base.SourceColumnSet{},
		}, nil
	}
	if len(stmts.Items) != 1 {
		return nil, errors.Errorf("expected exactly 1 statement, got %d", len(stmts.Items))
	}

	root := stmts.Items[0]
	accessTables := collectAccessTables(root, q.defaultDatabase)
	allSystems, mixed := isMixedQuery(accessTables, q.ignoreCaseSensitive)
	if mixed {
		return nil, base.MixUserSystemTablesError
	}

	queryType := classifyQueryType(root, allSystems, q.source)
	if allSystems {
		accessTables = base.SourceColumnSet{}
	}
	explain, isExplain := root.(*ast.ExplainStmt)
	explainAnalyze := isExplain && explain.Analyze

	var results []base.QuerySpanResult
	predicateColumns := base.SourceColumnSet{}
	if queryType == base.Select {
		q.accessTables = accessTables
		tableSource, err := q.extractSelectRoot(root)
		if err != nil {
			var resourceNotFound *base.ResourceNotFoundError
			if errors.As(err, &resourceNotFound) {
				return &base.QuerySpan{
					Type:             base.Select,
					SourceColumns:    accessTables,
					Results:          []base.QuerySpanResult{},
					PredicateColumns: q.predicateColumns,
					NotFoundError:    resourceNotFound,
				}, nil
			}
			return nil, err
		}
		results = tableSource.GetQuerySpanResult()
		predicateColumns = q.predicateColumns
	}
	return &base.QuerySpan{
		Type:             queryType,
		ExplainAnalyze:   explainAnalyze,
		SourceColumns:    accessTables,
		Results:          results,
		PredicateColumns: predicateColumns,
	}, nil
}

func (q *querySpanExtractor) extractSelectRoot(root ast.Node) (*base.PseudoTable, error) {
	switch n := root.(type) {
	case *ast.SelectStmt:
		return q.extractSelectStmt(n)
	case *ast.ExplainStmt:
		if n.Analyze {
			q.collectExplainedQueryColumns(n.Stmt)
		}
		return &base.PseudoTable{}, nil
	case *ast.TableStmt:
		return q.extractTableStmt(n)
	case *ast.ValuesStmt:
		return q.extractValuesStmt(n)
	default:
		return &base.PseudoTable{}, nil
	}
}

func (q *querySpanExtractor) extractSelectStmt(stmt *ast.SelectStmt) (*base.PseudoTable, error) {
	if stmt == nil {
		return &base.PseudoTable{}, nil
	}
	if stmt.Into != nil {
		return nil, errors.New("meet unsupported select statement with into")
	}

	originalTableSourceFrom := q.tableSourceFrom
	originalPriorLength := len(q.priorTableInFrom)
	originalCTELength := len(q.ctes)
	originalPosition := q.position()
	defer func() {
		q.tableSourceFrom = originalTableSourceFrom
		q.priorTableInFrom = q.priorTableInFrom[:originalPriorLength]
		q.ctes = q.ctes[:originalCTELength]
		q.setPosition(originalPosition)
	}()
	q.setPosition(selectPosition{})

	if err := q.processCTEs(stmt.CTEs); err != nil {
		return nil, err
	}
	var table *base.PseudoTable
	var err error
	switch {
	case stmt.ParenSource != nil:
		table, err = q.extractSelectStmt(stmt.ParenSource)
	case stmt.TableSource != nil:
		table, err = q.extractTableStmt(stmt.TableSource)
	case stmt.ValuesSource != nil:
		table, err = q.extractValuesStmt(stmt.ValuesSource)
	case stmt.SetOp != ast.SetOpNone:
		table, err = q.extractSetOp(stmt)
	default:
		return q.extractQuerySpecification(stmt)
	}
	if err != nil {
		return nil, err
	}
	// The ORDER BY and LIMIT written after a parenthesized query, a TABLE or
	// VALUES primary, or a set operation belong to the wrapper.
	q.selectItems = table.GetQuerySpanResult()
	q.collectTrailingClauses(stmt.OrderBy, stmt.Limit)
	return table, nil
}

// extractQuerySpecification extracts a SELECT ... FROM ... block,
// recording the columns its conditions read as it resolves each clause.
func (q *querySpanExtractor) extractQuerySpecification(stmt *ast.SelectStmt) (*base.PseudoTable, error) {
	q.clause = clauseFrom
	fromSources, err := q.extractTableSources(stmt.From)
	if err != nil {
		return nil, err
	}
	q.tableSourceFrom = append(q.tableSourceFrom, fromSources...)

	q.clause = clauseWhere
	q.collectConditionColumns(stmt.Where)

	q.clause = clauseSelectList
	results, err := q.extractTargetList(stmt.TargetList, fromSources)
	if err != nil {
		return nil, err
	}
	q.selectItems = results

	// MariaDB GROUP BY items carry an optional ASC/DESC now; span extraction
	// only cares about the expression each one groups on.
	q.groupBy = make([]ast.ExprNode, 0, len(stmt.GroupBy))
	for _, item := range stmt.GroupBy {
		if item != nil {
			q.groupBy = append(q.groupBy, item.Expr)
		}
	}
	q.clause = clauseGroupBy
	for _, item := range stmt.GroupBy {
		if item != nil {
			q.collectNestedQueryColumns(item.Expr)
		}
	}
	q.clause = clauseHaving
	q.collectConditionColumns(stmt.Having)
	q.clause = clauseOrderBy
	for _, window := range stmt.WindowClause {
		q.collectNestedQueryColumns(window)
	}
	q.collectTrailingClauses(stmt.OrderBy, stmt.Limit)
	return &base.PseudoTable{
		Columns: results,
	}, nil
}

func (q *querySpanExtractor) extractTableStmt(stmt *ast.TableStmt) (*base.PseudoTable, error) {
	if stmt == nil || stmt.Table == nil {
		return &base.PseudoTable{}, nil
	}
	if stmt.Into != nil {
		return nil, errors.New("meet unsupported table statement with into")
	}
	tableSource, err := q.findTableSchema(stmt.Table.Schema, stmt.Table.Name)
	if err != nil {
		return nil, err
	}
	q.selectItems = tableSource.GetQuerySpanResult()
	q.collectTrailingClauses(stmt.OrderBy, stmt.Limit)
	return &base.PseudoTable{Columns: tableSource.GetQuerySpanResult()}, nil
}

// extractValuesStmt merges every row into its column, since each row is
// returned.
func (q *querySpanExtractor) extractValuesStmt(stmt *ast.ValuesStmt) (*base.PseudoTable, error) {
	if stmt == nil || len(stmt.Rows) == 0 {
		return &base.PseudoTable{}, nil
	}
	var columns []base.QuerySpanResult
	for i, row := range stmt.Rows {
		for j, expr := range row {
			field, err := q.extractExpr(expr)
			if err != nil {
				return nil, err
			}
			if i == 0 {
				columns = append(columns, field)
				continue
			}
			if j < len(columns) {
				columns[j].SourceColumns, _ = base.MergeSourceColumnSet(columns[j].SourceColumns, field.SourceColumns)
				columns[j].IsPlainField = columns[j].IsPlainField && field.IsPlainField
			}
		}
	}
	q.selectItems = columns
	q.collectTrailingClauses(stmt.OrderBy, stmt.Limit)
	return &base.PseudoTable{Columns: columns}, nil
}

func (q *querySpanExtractor) extractSetOp(stmt *ast.SelectStmt) (*base.PseudoTable, error) {
	if stmt == nil || stmt.SetOp == ast.SetOpNone {
		return q.extractSelectStmt(stmt)
	}
	left, err := q.extractSelectStmt(stmt.Left)
	if err != nil {
		return nil, err
	}
	right, err := q.extractSelectStmt(stmt.Right)
	if err != nil {
		return nil, err
	}
	leftResults := left.GetQuerySpanResult()
	rightResults := right.GetQuerySpanResult()
	if len(leftResults) != len(rightResults) {
		return nil, errors.Errorf("MySQL UNION operator left has %d fields, right has %d fields", len(leftResults), len(rightResults))
	}
	// INTERSECT and EXCEPT keep a row by whether the other branch has it, so
	// both branches' output decides which rows come back.
	filters := stmt.SetOp == ast.SetOpIntersect || stmt.SetOp == ast.SetOpExcept
	results := make([]base.QuerySpanResult, 0, len(leftResults))
	for i := range leftResults {
		sourceColumns, _ := base.MergeSourceColumnSet(leftResults[i].SourceColumns, rightResults[i].SourceColumns)
		if filters {
			mergeSourceColumnSetInto(q.predicateColumns, sourceColumns)
		}
		results = append(results, base.QuerySpanResult{
			Name:          leftResults[i].Name,
			SourceColumns: sourceColumns,
			IsPlainField:  false,
		})
	}
	return &base.PseudoTable{Columns: results}, nil
}

// processCTEs extracts each CTE into its own predicate set, which
// findTableSchema merges into the reading query's.
func (q *querySpanExtractor) processCTEs(ctes []*ast.CommonTableExpr) error {
	for _, cte := range ctes {
		readerPredicateColumns := q.predicateColumns
		q.predicateColumns = make(base.SourceColumnSet)
		tableSource, err := q.extractCTE(cte)
		ctePredicateColumns := q.predicateColumns
		q.predicateColumns = readerPredicateColumns
		if err != nil {
			return err
		}
		if tableSource != nil {
			q.ctePredicateColumns[tableSource] = ctePredicateColumns
			q.ctes = append(q.ctes, tableSource)
		}
	}
	return nil
}

func (q *querySpanExtractor) extractCTE(cte *ast.CommonTableExpr) (*base.PseudoTable, error) {
	if cte == nil {
		return nil, nil
	}
	if cte.Recursive {
		return q.extractRecursiveCTE(cte)
	}
	return q.extractNonRecursiveCTE(cte)
}

func (q *querySpanExtractor) extractNonRecursiveCTE(cte *ast.CommonTableExpr) (*base.PseudoTable, error) {
	tableSource, err := q.extractSelectStmt(cte.Select)
	if err != nil {
		return nil, err
	}
	results := cloneQuerySpanResults(tableSource.GetQuerySpanResult())
	if len(cte.Columns) > 0 {
		if len(cte.Columns) != len(results) {
			return nil, errors.Errorf("MySQL CTE column list should have the same length, but got %d and %d", len(cte.Columns), len(results))
		}
		for i := range cte.Columns {
			results[i].Name = cte.Columns[i]
		}
	}
	return &base.PseudoTable{Name: cte.Name, Columns: results}, nil
}

func (q *querySpanExtractor) extractRecursiveCTE(cte *ast.CommonTableExpr) (*base.PseudoTable, error) {
	selectStmt := ast.UnwrapParenSource(cte.Select)
	if selectStmt == nil || selectStmt.SetOp == ast.SetOpNone {
		return q.extractNonRecursiveCTE(cte)
	}
	initialTable, err := q.extractRecursiveCTEAnchor(selectStmt)
	if err != nil {
		return nil, err
	}
	results := cloneQuerySpanResults(initialTable.GetQuerySpanResult())
	if len(cte.Columns) > 0 {
		if len(cte.Columns) != len(results) {
			return nil, errors.Errorf("The common table expression and column names list have different column counts")
		}
		for i := range cte.Columns {
			results[i].Name = cte.Columns[i]
		}
	}
	cteInfo := &base.PseudoTable{Name: cte.Name, Columns: results}
	q.ctes = append(q.ctes, cteInfo)
	defer func() {
		q.ctes = q.ctes[:len(q.ctes)-1]
	}()

	for {
		recursiveTable, err := q.extractSelectStmt(cte.Select)
		if err != nil {
			return nil, err
		}
		recursiveResults := recursiveTable.GetQuerySpanResult()
		if len(recursiveResults) != len(cteInfo.Columns) {
			return nil, errors.Errorf("The common table expression and column names list have different column counts")
		}
		changed := false
		for i := range recursiveResults {
			var ok bool
			cteInfo.Columns[i].SourceColumns, ok = base.MergeSourceColumnSet(cteInfo.Columns[i].SourceColumns, recursiveResults[i].SourceColumns)
			cteInfo.Columns[i].IsPlainField = false
			changed = changed || ok
		}
		if !changed {
			break
		}
	}
	return cteInfo, nil
}

func (q *querySpanExtractor) extractRecursiveCTEAnchor(stmt *ast.SelectStmt) (*base.PseudoTable, error) {
	stmt = ast.UnwrapParenSource(stmt)
	if stmt == nil || stmt.SetOp == ast.SetOpNone {
		return q.extractSelectStmt(stmt)
	}
	if stmt.Left != nil {
		return q.extractRecursiveCTEAnchor(stmt.Left)
	}
	return q.extractRecursiveCTEAnchor(stmt.Right)
}

func cloneQuerySpanResults(results []base.QuerySpanResult) []base.QuerySpanResult {
	clone := make([]base.QuerySpanResult, len(results))
	copy(clone, results)
	return clone
}

func (q *querySpanExtractor) extractTableSources(tableExprs []ast.TableExpr) ([]base.TableSource, error) {
	var result []base.TableSource
	for _, tableExpr := range tableExprs {
		tableSource, err := q.extractTableSource(tableExpr)
		if err != nil {
			return nil, err
		}
		if tableSource == nil {
			continue
		}
		q.priorTableInFrom = append(q.priorTableInFrom, tableSource)
		result = append(result, tableSource)
	}
	return result, nil
}

func (q *querySpanExtractor) extractTableSource(tableExpr ast.TableExpr) (base.TableSource, error) {
	switch t := tableExpr.(type) {
	case nil:
		return nil, nil
	case *ast.TableRef:
		if isDualTable(t) {
			return &base.PseudoTable{Name: t.Name}, nil
		}
		tableSource, err := q.findTableSchema(t.Schema, t.Name)
		if err != nil {
			return nil, err
		}
		if t.Alias == "" {
			return tableSource, nil
		}
		return &base.PseudoTable{
			Name:    t.Alias,
			Columns: tableSource.GetQuerySpanResult(),
		}, nil
	case *ast.SubqueryExpr:
		return q.extractSubquery(t)
	case *ast.JoinClause:
		return q.extractJoin(t)
	case *ast.JsonTableExpr:
		return q.extractJSONTable(t)
	default:
		return nil, errors.Errorf("unsupported omni MySQL table source %T", tableExpr)
	}
}

func (q *querySpanExtractor) extractJSONTable(expr *ast.JsonTableExpr) (base.TableSource, error) {
	if expr == nil {
		return nil, nil
	}
	name := expr.Alias
	if name == "" {
		name = q.exprName(expr.Expr)
	}
	field, err := q.extractExpr(expr.Expr)
	if err != nil {
		return nil, err
	}
	// JSON_TABLE returns a row per match in the document.
	mergeSourceColumnSetInto(q.predicateColumns, field.SourceColumns)
	var columns []base.QuerySpanResult
	for _, column := range flattenJSONTableColumns(expr.Columns) {
		columns = append(columns, base.QuerySpanResult{
			Name:          column.Name,
			SourceColumns: field.SourceColumns,
			IsPlainField:  field.IsPlainField,
		})
	}
	return &base.PseudoTable{Name: name, Columns: columns}, nil
}

func flattenJSONTableColumns(columns []*ast.JsonTableColumn) []*ast.JsonTableColumn {
	var result []*ast.JsonTableColumn
	for _, column := range columns {
		if column == nil {
			continue
		}
		if column.Nested {
			result = append(result, flattenJSONTableColumns(column.NestedCols)...)
			continue
		}
		result = append(result, column)
	}
	return result
}

func (q *querySpanExtractor) extractJoin(join *ast.JoinClause) (base.TableSource, error) {
	if join == nil {
		return nil, nil
	}
	left, err := q.extractTableSource(join.Left)
	if err != nil {
		return nil, err
	}
	if left != nil {
		q.tableSourceFrom = append(q.tableSourceFrom, left)
	}
	right, err := q.extractTableSource(join.Right)
	if err != nil {
		return nil, err
	}
	if right != nil {
		q.tableSourceFrom = append(q.tableSourceFrom, right)
	}

	var using []string
	switch condition := join.Condition.(type) {
	case *ast.OnCondition:
		if condition != nil {
			q.collectConditionColumns(condition.Expr)
		}
	case *ast.UsingCondition:
		if condition != nil {
			using = condition.Columns
		}
	default:
	}
	tp := toJoinType(join.Type)
	if left != nil && right != nil {
		// USING and NATURAL compare the joined columns of both sides.
		for _, name := range joinedColumnNames(left, right, tp, using) {
			q.recordJoinedColumn(left, name)
			q.recordJoinedColumn(right, name)
		}
	}
	return joinTableSources(left, right, tp, using), nil
}

func toJoinType(joinType ast.JoinType) joinType {
	switch joinType {
	case ast.JoinInner:
		return InnerJoin
	case ast.JoinLeft:
		return LeftOuterJoin
	case ast.JoinRight:
		return RightOuterJoin
	case ast.JoinCross:
		return CrossJoin
	case ast.JoinNatural:
		return NaturalInnerJoin
	case ast.JoinStraight:
		return StraightJoin
	case ast.JoinNaturalLeft:
		return NaturalLeftOuterJoin
	case ast.JoinNaturalRight:
		return NaturalRightOuterJoin
	default:
		return Join
	}
}

func (q *querySpanExtractor) extractSubquery(expr *ast.SubqueryExpr) (*base.PseudoTable, error) {
	if expr == nil {
		return &base.PseudoTable{}, nil
	}
	// A derived table cannot see the FROM clause it sits in unless it is
	// LATERAL, and then only the items before it: earlier FROM items, and the
	// operands extractJoin has already added for the join it is in.
	scope := outerScope{}
	if expr.Lateral {
		scope.tables = slices.Concat(q.tableSourceFrom, q.priorTableInFrom)
	}
	subqueryExtractor := q.cloneWithScope(scope)
	tableSource, err := subqueryExtractor.extractSelectStmt(expr.Select)
	if err != nil {
		return nil, err
	}
	results := cloneQuerySpanResults(tableSource.GetQuerySpanResult())
	if len(expr.Columns) > 0 {
		if len(expr.Columns) != len(results) {
			return nil, errors.Errorf("derived table column list length %d doesn't match result column length %d", len(expr.Columns), len(results))
		}
		for i := range expr.Columns {
			results[i].Name = expr.Columns[i]
		}
	}
	if expr.Alias == "" {
		return &base.PseudoTable{Name: tableSource.Name, Columns: results}, nil
	}
	return &base.PseudoTable{
		Name:    expr.Alias,
		Columns: results,
	}, nil
}

// cloneForSubquery returns the extractor for a query nested in an
// expression of the current SELECT, which sees its tables and, where MySQL
// allows, its select list.
func (q *querySpanExtractor) cloneForSubquery() *querySpanExtractor {
	return q.cloneWithScope(outerScope{
		tables: slices.Concat(q.tableSourceFrom, q.priorTableInFrom),
		items:  q.selectItemsVisibleToSubquery(),
	})
}

// cloneWithScope returns an extractor for a nested query that sees scope
// as its nearest enclosing SELECT. The nested query records its predicate
// columns into the same set.
func (q *querySpanExtractor) cloneWithScope(scope outerScope) *querySpanExtractor {
	outerScopes := slices.Clone(q.outerScopes)
	if len(scope.tables) > 0 || len(scope.items) > 0 {
		outerScopes = append(outerScopes, scope)
	}
	return &querySpanExtractor{
		ctx:                 q.ctx,
		defaultDatabase:     q.defaultDatabase,
		gCtx:                q.gCtx,
		ctes:                q.ctes,
		outerScopes:         outerScopes,
		tableSourceFrom:     []base.TableSource{},
		ignoreCaseSensitive: q.ignoreCaseSensitive,
		outputInCondition:   q.inCondition || q.outputInCondition,
		predicateColumns:    q.predicateColumns,
		ctePredicateColumns: q.ctePredicateColumns,
		accessTables:        q.accessTables,
		viewResolutionStack: cloneViewResolutionStack(q.viewResolutionStack),
		source:              q.source,
	}
}

func (q *querySpanExtractor) extractTargetList(targets []ast.ExprNode, fromSources []base.TableSource) ([]base.QuerySpanResult, error) {
	var result []base.QuerySpanResult
	for _, target := range targets {
		switch t := target.(type) {
		case *ast.ResTarget:
			if _, ok := t.Val.(*ast.StarExpr); ok {
				fields, err := q.extractTableWild("", "", fromSources)
				if err != nil {
					return nil, err
				}
				result = append(result, fields...)
				continue
			}
			if columnRef, ok := t.Val.(*ast.ColumnRef); ok && columnRef.Star {
				fields, err := q.extractColumnWild(columnRef, fromSources)
				if err != nil {
					return nil, err
				}
				result = append(result, fields...)
				continue
			}
		case *ast.StarExpr:
			fields, err := q.extractTableWild("", "", fromSources)
			if err != nil {
				return nil, err
			}
			result = append(result, fields...)
			continue
		case *ast.ColumnRef:
			if t.Star {
				fields, err := q.extractColumnWild(t, fromSources)
				if err != nil {
					return nil, err
				}
				result = append(result, fields...)
				continue
			}
		default:
		}

		// A subquery in this item can name the items before it.
		q.selectItems = result
		field, err := q.extractTarget(target)
		if err != nil {
			return nil, err
		}
		result = append(result, field)
	}
	return result, nil
}

func (q *querySpanExtractor) extractTarget(target ast.ExprNode) (base.QuerySpanResult, error) {
	switch t := target.(type) {
	case *ast.ResTarget:
		field, err := q.extractExpr(t.Val)
		if err != nil {
			return base.QuerySpanResult{}, err
		}
		if t.Name != "" {
			field.Name = t.Name
		}
		return field, nil
	default:
		return q.extractExpr(target)
	}
}

func (q *querySpanExtractor) extractExpr(expr ast.ExprNode) (base.QuerySpanResult, error) {
	switch e := expr.(type) {
	case nil:
		return base.QuerySpanResult{
			SourceColumns: base.SourceColumnSet{},
			IsPlainField:  true,
		}, nil
	case *ast.ColumnRef:
		if e.Star {
			return base.QuerySpanResult{}, errors.Errorf("asterisk is not a scalar expression")
		}
		sourceColumns, err := q.getFieldColumnSource(e.Schema, e.Table, e.Column)
		if err != nil {
			return base.QuerySpanResult{}, err
		}
		return base.QuerySpanResult{
			Name:          e.Column,
			SourceColumns: sourceColumns,
			IsPlainField:  true,
		}, nil
	case *ast.StarExpr:
		return base.QuerySpanResult{}, errors.Errorf("asterisk is not a scalar expression")
	case *ast.IntLit, *ast.FloatLit, *ast.StringLit, *ast.BoolLit, *ast.NullLit, *ast.HexLit, *ast.BitLit, *ast.TemporalLit:
		return base.QuerySpanResult{
			Name:          q.exprName(e),
			SourceColumns: base.SourceColumnSet{},
			IsPlainField:  true,
		}, nil
	case *ast.VariableRef:
		return base.QuerySpanResult{
			Name:          q.exprName(e),
			SourceColumns: base.SourceColumnSet{},
			IsPlainField:  true,
		}, nil
	case *ast.ParenExpr:
		field, err := q.extractExpr(e.Expr)
		if err != nil {
			return base.QuerySpanResult{}, err
		}
		field.Name = q.exprName(e)
		field.IsPlainField = false
		return field, nil
	case *ast.BinaryExpr:
		sourceColumns, err := q.mergeExprSources(e.Left, e.Right)
		if err != nil {
			return base.QuerySpanResult{}, err
		}
		return base.QuerySpanResult{Name: q.exprName(e), SourceColumns: sourceColumns, IsPlainField: false}, nil
	case *ast.UnaryExpr:
		sourceColumns, err := q.mergeExprSources(e.Operand)
		if err != nil {
			return base.QuerySpanResult{}, err
		}
		return base.QuerySpanResult{Name: q.exprName(e), SourceColumns: sourceColumns, IsPlainField: false}, nil
	case *ast.FuncCallExpr:
		exprs := append([]ast.ExprNode{}, e.Args...)
		exprs = append(exprs, e.Separator)
		for _, item := range e.OrderBy {
			if item != nil {
				exprs = append(exprs, item.Expr)
			}
		}
		if e.Over != nil {
			exprs = append(exprs, e.Over.PartitionBy...)
			for _, item := range e.Over.OrderBy {
				if item != nil {
					exprs = append(exprs, item.Expr)
				}
			}
		}
		// HAVING resolves the arguments of an aggregate like a GROUP BY name.
		inAggregate := q.inAggregate
		if q.clause == clauseHaving && e.Over == nil && isAggregateFunction(e.Name) {
			q.inAggregate = true
		}
		sourceColumns, err := q.mergeExprSources(exprs...)
		q.inAggregate = inAggregate
		if err != nil {
			return base.QuerySpanResult{}, err
		}
		return base.QuerySpanResult{Name: q.exprName(e), SourceColumns: sourceColumns, IsPlainField: false}, nil
	case *ast.SubqueryExpr:
		sourceColumns, isPlain, err := q.extractSelectSourceColumns(e.Select)
		if err != nil {
			return base.QuerySpanResult{}, err
		}
		return base.QuerySpanResult{Name: q.exprName(e), SourceColumns: sourceColumns, IsPlainField: isPlain}, nil
	case *ast.ExistsExpr:
		sourceColumns, _, err := q.extractSelectSourceColumns(e.Select)
		if err != nil {
			return base.QuerySpanResult{}, err
		}
		if q.inCondition || q.outputInCondition {
			// EXISTS reports whether a row comes back, not what the row holds.
			sourceColumns = base.SourceColumnSet{}
		}
		return base.QuerySpanResult{Name: q.exprName(e), SourceColumns: sourceColumns, IsPlainField: false}, nil
	case *ast.CaseExpr:
		var exprs []ast.ExprNode
		exprs = append(exprs, e.Operand)
		for _, when := range e.Whens {
			if when == nil {
				continue
			}
			exprs = append(exprs, when.Cond, when.Result)
		}
		exprs = append(exprs, e.Default)
		sourceColumns, err := q.mergeExprSources(exprs...)
		if err != nil {
			return base.QuerySpanResult{}, err
		}
		return base.QuerySpanResult{Name: q.exprName(e), SourceColumns: sourceColumns, IsPlainField: false}, nil
	case *ast.BetweenExpr:
		sourceColumns, err := q.mergeExprSources(e.Expr, e.Low, e.High)
		if err != nil {
			return base.QuerySpanResult{}, err
		}
		return base.QuerySpanResult{Name: q.exprName(e), SourceColumns: sourceColumns, IsPlainField: false}, nil
	case *ast.InExpr:
		exprs := append([]ast.ExprNode{e.Expr}, e.List...)
		sourceColumns, err := q.mergeExprSources(exprs...)
		if err != nil {
			return base.QuerySpanResult{}, err
		}
		if e.Select != nil {
			selectSourceColumns, _, err := q.extractSelectSourceColumns(e.Select)
			if err != nil {
				return base.QuerySpanResult{}, err
			}
			sourceColumns, _ = base.MergeSourceColumnSet(sourceColumns, selectSourceColumns)
		}
		return base.QuerySpanResult{Name: q.exprName(e), SourceColumns: sourceColumns, IsPlainField: false}, nil
	case *ast.LikeExpr:
		sourceColumns, err := q.mergeExprSources(e.Expr, e.Pattern, e.Escape)
		if err != nil {
			return base.QuerySpanResult{}, err
		}
		return base.QuerySpanResult{Name: q.exprName(e), SourceColumns: sourceColumns, IsPlainField: false}, nil
	case *ast.IsExpr:
		sourceColumns, err := q.mergeExprSources(e.Expr)
		if err != nil {
			return base.QuerySpanResult{}, err
		}
		return base.QuerySpanResult{Name: q.exprName(e), SourceColumns: sourceColumns, IsPlainField: false}, nil
	case *ast.CastExpr:
		sourceColumns, err := q.mergeExprSources(e.Expr)
		if err != nil {
			return base.QuerySpanResult{}, err
		}
		return base.QuerySpanResult{Name: q.exprName(e), SourceColumns: sourceColumns, IsPlainField: false}, nil
	case *ast.ExtractExpr:
		sourceColumns, err := q.mergeExprSources(e.Expr)
		if err != nil {
			return base.QuerySpanResult{}, err
		}
		return base.QuerySpanResult{Name: q.exprName(e), SourceColumns: sourceColumns, IsPlainField: false}, nil
	case *ast.IntervalExpr:
		sourceColumns, err := q.mergeExprSources(e.Value)
		if err != nil {
			return base.QuerySpanResult{}, err
		}
		return base.QuerySpanResult{Name: q.exprName(e), SourceColumns: sourceColumns, IsPlainField: false}, nil
	case *ast.CollateExpr:
		sourceColumns, err := q.mergeExprSources(e.Expr)
		if err != nil {
			return base.QuerySpanResult{}, err
		}
		return base.QuerySpanResult{Name: q.exprName(e), SourceColumns: sourceColumns, IsPlainField: false}, nil
	case *ast.MatchExpr:
		exprs := make([]ast.ExprNode, 0, len(e.Columns)+1)
		for _, column := range e.Columns {
			exprs = append(exprs, column)
		}
		exprs = append(exprs, e.Against)
		sourceColumns, err := q.mergeExprSources(exprs...)
		if err != nil {
			return base.QuerySpanResult{}, err
		}
		return base.QuerySpanResult{Name: q.exprName(e), SourceColumns: sourceColumns, IsPlainField: false}, nil
	case *ast.ConvertExpr:
		sourceColumns, err := q.mergeExprSources(e.Expr)
		if err != nil {
			return base.QuerySpanResult{}, err
		}
		return base.QuerySpanResult{Name: q.exprName(e), SourceColumns: sourceColumns, IsPlainField: false}, nil
	case *ast.DefaultExpr:
		return base.QuerySpanResult{Name: q.exprName(e), SourceColumns: base.SourceColumnSet{}, IsPlainField: true}, nil
	case *ast.RowExpr:
		sourceColumns, err := q.mergeExprSources(e.Items...)
		if err != nil {
			return base.QuerySpanResult{}, err
		}
		return base.QuerySpanResult{Name: q.exprName(e), SourceColumns: sourceColumns, IsPlainField: false}, nil
	case *ast.KeywordArg:
		// Keyword arguments (TIMESTAMPDIFF(SECOND, ...), GET_FORMAT(DATE, ...))
		// are unit selectors, not column references — no lineage.
		return base.QuerySpanResult{Name: e.Keyword, SourceColumns: base.SourceColumnSet{}, IsPlainField: false}, nil
	default:
		return base.QuerySpanResult{}, errors.Errorf("unsupported omni MariaDB expression %T", expr)
	}
}

func (q *querySpanExtractor) extractSelectSourceColumns(selectStmt *ast.SelectStmt) (base.SourceColumnSet, bool, error) {
	subqueryExtractor := q.cloneForSubquery()
	tableSource, err := subqueryExtractor.extractSelectStmt(selectStmt)
	if err != nil {
		return nil, false, err
	}
	spanResults := tableSource.GetQuerySpanResult()
	sourceColumns := make(base.SourceColumnSet)
	for _, field := range spanResults {
		sourceColumns, _ = base.MergeSourceColumnSet(sourceColumns, field.SourceColumns)
	}
	isPlain := false
	if len(spanResults) == 1 {
		isPlain = spanResults[0].IsPlainField
	}
	return sourceColumns, isPlain, nil
}

func (q *querySpanExtractor) mergeExprSources(exprs ...ast.ExprNode) (base.SourceColumnSet, error) {
	result := make(base.SourceColumnSet)
	for _, expr := range exprs {
		if expr == nil {
			continue
		}
		field, err := q.extractExpr(expr)
		if err != nil {
			return nil, err
		}
		result, _ = base.MergeSourceColumnSet(result, field.SourceColumns)
	}
	return result, nil
}

func (q *querySpanExtractor) extractColumnWild(columnRef *ast.ColumnRef, fromSources []base.TableSource) ([]base.QuerySpanResult, error) {
	if columnRef == nil {
		return nil, nil
	}
	return q.extractTableWild(columnRef.Schema, columnRef.Table, fromSources)
}

func (q *querySpanExtractor) extractTableWild(databaseName, tableName string, fromSources []base.TableSource) ([]base.QuerySpanResult, error) {
	if databaseName == "" && tableName == "" {
		var result []base.QuerySpanResult
		for _, fromSource := range fromSources {
			result = append(result, fromSource.GetQuerySpanResult()...)
		}
		return result, nil
	}

	result, ok := q.getAllTableColumnSources(databaseName, tableName)
	if !ok {
		return nil, &base.ResourceNotFoundError{
			Database: &databaseName,
			Table:    &tableName,
		}
	}
	return result, nil
}

func (q *querySpanExtractor) exprName(expr ast.ExprNode) string {
	switch e := expr.(type) {
	case *ast.ColumnRef:
		var parts []string
		if e.Schema != "" {
			parts = append(parts, e.Schema)
		}
		if e.Table != "" {
			parts = append(parts, e.Table)
		}
		if e.Star {
			parts = append(parts, "*")
		} else {
			parts = append(parts, e.Column)
		}
		return strings.Join(parts, ".")
	case *ast.BinaryExpr:
		return q.exprName(e.Left) + binaryOpName(e) + q.exprName(e.Right)
	case *ast.InExpr:
		var list []string
		for _, expr := range e.List {
			list = append(list, q.exprName(expr))
		}
		operator := " in "
		if e.Not {
			operator = " not in "
		}
		return q.exprName(e.Expr) + operator + "(" + strings.Join(list, ", ") + ")"
	case *ast.IntLit:
		if name := q.slice(e.Loc); name != "" {
			return name
		}
		return strconv.FormatInt(e.Value, 10)
	case *ast.FloatLit:
		if name := q.slice(e.Loc); name != "" {
			return name
		}
		return e.Value
	case *ast.StringLit:
		if name := q.slice(e.Loc); name != "" {
			return name
		}
		return e.Value
	case *ast.BoolLit:
		if name := q.slice(e.Loc); name != "" {
			return name
		}
		return strconv.FormatBool(e.Value)
	case *ast.NullLit:
		if name := q.slice(e.Loc); name != "" {
			return name
		}
		return "NULL"
	case *ast.HexLit:
		if name := q.slice(e.Loc); name != "" {
			return name
		}
		return e.Value
	case *ast.BitLit:
		if name := q.slice(e.Loc); name != "" {
			return name
		}
		return e.Value
	case *ast.TemporalLit:
		if name := q.slice(e.Loc); name != "" {
			return name
		}
		return e.Value
	default:
		if loc, ok := nodeLoc(e); ok {
			return q.slice(loc)
		}
		return ""
	}
}

func binaryOpName(expr *ast.BinaryExpr) string {
	if expr == nil {
		return ""
	}
	if expr.OriginalOp != "" {
		return expr.OriginalOp
	}
	return binaryOpNames[expr.Op]
}

var binaryOpNames = map[ast.BinaryOp]string{
	ast.BinOpAdd:         "+",
	ast.BinOpSub:         "-",
	ast.BinOpMul:         "*",
	ast.BinOpDiv:         "/",
	ast.BinOpMod:         "%",
	ast.BinOpEq:          "=",
	ast.BinOpNe:          "!=",
	ast.BinOpLt:          "<",
	ast.BinOpGt:          ">",
	ast.BinOpLe:          "<=",
	ast.BinOpGe:          ">=",
	ast.BinOpAnd:         " and ",
	ast.BinOpOr:          " or ",
	ast.BinOpXor:         " xor ",
	ast.BinOpBitAnd:      "&",
	ast.BinOpBitOr:       "|",
	ast.BinOpBitXor:      "^",
	ast.BinOpShiftLeft:   "<<",
	ast.BinOpShiftRight:  ">>",
	ast.BinOpDivInt:      " div ",
	ast.BinOpRegexp:      " regexp ",
	ast.BinOpLikeEscape:  " like ",
	ast.BinOpNullSafeEq:  "<=>",
	ast.BinOpAssign:      ":=",
	ast.BinOpJsonExtract: "->",
	ast.BinOpJsonUnquote: "->>",
	ast.BinOpSoundsLike:  " sounds like ",
}

func (q *querySpanExtractor) slice(loc ast.Loc) string {
	if loc.Start < 0 || loc.End <= loc.Start || loc.End > len(q.source) {
		return ""
	}
	return strings.TrimSpace(q.source[loc.Start:loc.End])
}

func nodeLoc(node ast.Node) (ast.Loc, bool) {
	if loc, ok := exprNodeLoc(node); ok {
		return loc, true
	}
	return literalNodeLoc(node)
}

func exprNodeLoc(node ast.Node) (ast.Loc, bool) {
	switch n := node.(type) {
	case *ast.ResTarget:
		return n.Loc, true
	case *ast.ColumnRef:
		return n.Loc, true
	case *ast.BinaryExpr:
		return n.Loc, true
	case *ast.UnaryExpr:
		return n.Loc, true
	case *ast.FuncCallExpr:
		return n.Loc, true
	case *ast.SubqueryExpr:
		return n.Loc, true
	case *ast.CaseExpr:
		return n.Loc, true
	case *ast.BetweenExpr:
		return n.Loc, true
	case *ast.InExpr:
		return n.Loc, true
	case *ast.LikeExpr:
		return n.Loc, true
	case *ast.IsExpr:
		return n.Loc, true
	case *ast.ExistsExpr:
		return n.Loc, true
	case *ast.CastExpr:
		return n.Loc, true
	case *ast.ExtractExpr:
		return n.Loc, true
	case *ast.IntervalExpr:
		return n.Loc, true
	case *ast.CollateExpr:
		return n.Loc, true
	case *ast.MatchExpr:
		return n.Loc, true
	case *ast.ConvertExpr:
		return n.Loc, true
	case *ast.DefaultExpr:
		return n.Loc, true
	case *ast.RowExpr:
		return n.Loc, true
	case *ast.ParenExpr:
		return n.Loc, true
	case *ast.StarExpr:
		return n.Loc, true
	default:
		return ast.Loc{}, false
	}
}

func literalNodeLoc(node ast.Node) (ast.Loc, bool) {
	switch n := node.(type) {
	case *ast.IntLit:
		return n.Loc, true
	case *ast.FloatLit:
		return n.Loc, true
	case *ast.StringLit:
		return n.Loc, true
	case *ast.BoolLit:
		return n.Loc, true
	case *ast.NullLit:
		return n.Loc, true
	case *ast.HexLit:
		return n.Loc, true
	case *ast.BitLit:
		return n.Loc, true
	case *ast.TemporalLit:
		return n.Loc, true
	case *ast.VariableRef:
		return n.Loc, true
	default:
		return ast.Loc{}, false
	}
}

func classifyQueryType(node ast.Node, allSystems bool, source string) base.QueryType {
	switch n := node.(type) {
	case *ast.SelectStmt, *ast.TableStmt, *ast.ValuesStmt:
		if allSystems {
			return base.SelectInfoSchema
		}
		return base.Select
	case *ast.ExplainStmt:
		if isDescribeStatement(source, n.Loc) {
			return base.SelectInfoSchema
		}
		if n.Analyze {
			switch n.Stmt.(type) {
			case *ast.SelectStmt, *ast.TableStmt, *ast.ValuesStmt:
				return base.Select
			default:
				return base.DML
			}
		}
		return base.Explain
	case *ast.ShowStmt:
		return base.SelectInfoSchema
	case *ast.SetStmt, *ast.SetDefaultRoleStmt, *ast.SetResourceGroupStmt, *ast.SetRoleStmt, *ast.SetTransactionStmt:
		return base.Select
	case *ast.SetPasswordStmt, *ast.HelpStmt:
		return base.QueryTypeUnknown
	case *ast.CreateDatabaseStmt, *ast.CreateTableStmt, *ast.CreateIndexStmt, *ast.CreateViewStmt,
		*ast.CreateEventStmt, *ast.CreateTriggerStmt, *ast.CreateFunctionStmt,
		*ast.CreateUserStmt, *ast.CreateRoleStmt, *ast.CreateTablespaceStmt, *ast.CreateServerStmt,
		*ast.CreateLogfileGroupStmt, *ast.CreateSpatialRefSysStmt, *ast.CreateResourceGroupStmt,
		*ast.AlterDatabaseStmt, *ast.AlterTableStmt, *ast.AlterViewStmt, *ast.AlterEventStmt,
		*ast.AlterUserStmt, *ast.AlterRoutineStmt, *ast.AlterTablespaceStmt, *ast.AlterServerStmt,
		*ast.AlterLogfileGroupStmt, *ast.AlterResourceGroupStmt, *ast.AlterInstanceStmt,
		*ast.DropDatabaseStmt, *ast.DropTableStmt, *ast.DropIndexStmt, *ast.DropViewStmt,
		*ast.DropEventStmt, *ast.DropTriggerStmt, *ast.DropRoutineStmt,
		*ast.DropUserStmt, *ast.DropRoleStmt, *ast.DropTablespaceStmt, *ast.DropServerStmt,
		*ast.DropLogfileGroupStmt, *ast.DropSpatialRefSysStmt, *ast.DropResourceGroupStmt,
		*ast.RenameTableStmt, *ast.RenameUserStmt, *ast.TruncateStmt, *ast.ImportTableStmt,
		*ast.GrantStmt, *ast.RevokeStmt, *ast.GrantRoleStmt, *ast.RevokeRoleStmt,
		*ast.InstallPluginStmt, *ast.UninstallPluginStmt, *ast.InstallComponentStmt,
		*ast.UninstallComponentStmt, *ast.CloneStmt:
		return base.DDL
	case *ast.InsertStmt, *ast.UpdateStmt, *ast.DeleteStmt, *ast.BeginStmt, *ast.CommitStmt,
		*ast.RollbackStmt, *ast.SavepointStmt, *ast.LockTablesStmt, *ast.UnlockTablesStmt,
		*ast.LoadDataStmt, *ast.PrepareStmt, *ast.ExecuteStmt, *ast.DeallocateStmt,
		*ast.CallStmt, *ast.DoStmt, *ast.HandlerOpenStmt, *ast.HandlerReadStmt, *ast.HandlerCloseStmt,
		*ast.AnalyzeTableStmt, *ast.OptimizeTableStmt, *ast.CheckTableStmt, *ast.RepairTableStmt,
		*ast.FlushStmt, *ast.KillStmt, *ast.ShutdownStmt, *ast.RestartStmt, *ast.XAStmt,
		*ast.SignalStmt, *ast.ResignalStmt, *ast.GetDiagnosticsStmt, *ast.DeclareVarStmt,
		*ast.DeclareConditionStmt, *ast.DeclareHandlerStmt, *ast.DeclareCursorStmt,
		*ast.IfStmt, *ast.WhileStmt, *ast.RepeatStmt, *ast.LoopStmt, *ast.LeaveStmt,
		*ast.IterateStmt, *ast.ReturnStmt, *ast.OpenCursorStmt, *ast.FetchCursorStmt,
		*ast.CloseCursorStmt, *ast.LockInstanceStmt, *ast.UnlockInstanceStmt, *ast.BinlogStmt,
		*ast.CacheIndexStmt, *ast.LoadIndexIntoCacheStmt, *ast.ResetPersistStmt,
		*ast.ChangeReplicationSourceStmt, *ast.ChangeReplicationFilterStmt,
		*ast.StartReplicaStmt, *ast.StopReplicaStmt, *ast.ResetReplicaStmt,
		*ast.PurgeBinaryLogsStmt, *ast.ResetMasterStmt,
		*ast.StartGroupReplicationStmt, *ast.StopGroupReplicationStmt:
		return base.DML
	default:
		return base.QueryTypeUnknown
	}
}

func isDescribeStatement(source string, loc ast.Loc) bool {
	statement := strings.TrimSpace(source)
	if loc.Start >= 0 && loc.End > loc.Start && loc.End <= len(source) {
		statement = strings.TrimSpace(source[loc.Start:loc.End])
	}
	upper := strings.ToUpper(statement)
	fields := strings.Fields(upper)
	return len(fields) > 0 && (fields[0] == "DESCRIBE" || fields[0] == "DESC")
}

func collectAccessTables(root ast.Node, defaultDatabase string) base.SourceColumnSet {
	result := make(base.SourceColumnSet)
	collectAccessTablesFromNode(result, root, defaultDatabase)
	return result
}

func collectAccessTablesFromNode(result base.SourceColumnSet, node ast.Node, defaultDatabase string) {
	switch n := node.(type) {
	case nil:
	case *ast.SelectStmt:
		if n == nil {
			return
		}
		for _, cte := range n.CTEs {
			if cte != nil {
				collectAccessTablesFromNode(result, cte.Select, defaultDatabase)
			}
		}
		if n.Left != nil {
			collectAccessTablesFromNode(result, n.Left, defaultDatabase)
		}
		if n.Right != nil {
			collectAccessTablesFromNode(result, n.Right, defaultDatabase)
		}
		if inner := n.ParenSource; inner != nil {
			collectAccessTablesFromNode(result, inner, defaultDatabase)
		}
		if tableStmt := n.TableSource; tableStmt != nil {
			collectAccessTablesFromNode(result, tableStmt, defaultDatabase)
		}
		if valuesStmt := n.ValuesSource; valuesStmt != nil {
			collectAccessTablesFromNode(result, valuesStmt, defaultDatabase)
		}
		for _, target := range n.TargetList {
			collectAccessTablesFromExpr(result, target, defaultDatabase)
		}
		for _, tableExpr := range n.From {
			collectAccessTablesFromTableExpr(result, tableExpr, defaultDatabase)
		}
		collectAccessTablesFromExpr(result, n.Where, defaultDatabase)
		for _, groupBy := range n.GroupBy {
			if groupBy != nil {
				collectAccessTablesFromExpr(result, groupBy.Expr, defaultDatabase)
			}
		}
		collectAccessTablesFromExpr(result, n.Having, defaultDatabase)
		for _, orderBy := range n.OrderBy {
			if orderBy != nil {
				collectAccessTablesFromExpr(result, orderBy.Expr, defaultDatabase)
			}
		}
	case *ast.ExplainStmt:
		collectAccessTablesFromNode(result, n.Stmt, defaultDatabase)
	case *ast.InsertStmt:
		collectAccessTablesFromTableRef(result, n.Table, defaultDatabase)
		for _, row := range n.Values {
			for _, expr := range row {
				collectAccessTablesFromExpr(result, expr, defaultDatabase)
			}
		}
		collectAccessTablesFromNode(result, n.Select, defaultDatabase)
		collectAccessTablesFromNode(result, n.TableSource, defaultDatabase)
		for _, assignment := range n.SetList {
			if assignment != nil {
				collectAccessTablesFromExpr(result, assignment.Value, defaultDatabase)
			}
		}
		for _, assignment := range n.OnDuplicateKey {
			if assignment != nil {
				collectAccessTablesFromExpr(result, assignment.Value, defaultDatabase)
			}
		}
	case *ast.UpdateStmt:
		for _, tableExpr := range n.Tables {
			collectAccessTablesFromTableExpr(result, tableExpr, defaultDatabase)
		}
		for _, assignment := range n.SetList {
			if assignment != nil {
				collectAccessTablesFromExpr(result, assignment.Value, defaultDatabase)
			}
		}
		collectAccessTablesFromExpr(result, n.Where, defaultDatabase)
		for _, orderBy := range n.OrderBy {
			if orderBy != nil {
				collectAccessTablesFromExpr(result, orderBy.Expr, defaultDatabase)
			}
		}
	case *ast.DeleteStmt:
		for _, tableExpr := range n.Tables {
			collectAccessTablesFromTableExpr(result, tableExpr, defaultDatabase)
		}
		for _, tableExpr := range n.Using {
			collectAccessTablesFromTableExpr(result, tableExpr, defaultDatabase)
		}
		collectAccessTablesFromExpr(result, n.Where, defaultDatabase)
		for _, orderBy := range n.OrderBy {
			if orderBy != nil {
				collectAccessTablesFromExpr(result, orderBy.Expr, defaultDatabase)
			}
		}
	case *ast.CreateTableStmt:
		collectAccessTablesFromTableRef(result, n.Table, defaultDatabase)
		collectAccessTablesFromTableRef(result, n.Like, defaultDatabase)
		collectAccessTablesFromNode(result, n.Select, defaultDatabase)
	case *ast.CreateIndexStmt:
		collectAccessTablesFromTableRef(result, n.Table, defaultDatabase)
		for _, column := range n.Columns {
			if column != nil {
				collectAccessTablesFromExpr(result, column.Expr, defaultDatabase)
			}
		}
	case *ast.CreateViewStmt:
		collectAccessTablesFromTableRef(result, n.Name, defaultDatabase)
		collectAccessTablesFromNode(result, n.Select, defaultDatabase)
	case *ast.AlterTableStmt:
		collectAccessTablesFromTableRef(result, n.Table, defaultDatabase)
		for _, command := range n.Commands {
			if command == nil {
				continue
			}
			collectAccessTablesFromExpr(result, command.DefaultExpr, defaultDatabase)
			collectAccessTablesFromTableRef(result, command.ExchangeTable, defaultDatabase)
			collectAccessTablesFromOrderBy(result, command.OrderByItems, defaultDatabase)
		}
	case *ast.DropTableStmt:
		collectAccessTablesFromTableRefs(result, n.Tables, defaultDatabase)
	case *ast.DropIndexStmt:
		collectAccessTablesFromTableRef(result, n.Table, defaultDatabase)
	case *ast.DropViewStmt:
		collectAccessTablesFromTableRefs(result, n.Views, defaultDatabase)
	case *ast.RenameTableStmt:
		for _, pair := range n.Pairs {
			if pair == nil {
				continue
			}
			collectAccessTablesFromTableRef(result, pair.Old, defaultDatabase)
			collectAccessTablesFromTableRef(result, pair.New, defaultDatabase)
		}
	case *ast.TruncateStmt:
		collectAccessTablesFromTableRefs(result, n.Tables, defaultDatabase)
	case *ast.AnalyzeTableStmt:
		collectAccessTablesFromTableRefs(result, n.Tables, defaultDatabase)
	case *ast.OptimizeTableStmt:
		collectAccessTablesFromTableRefs(result, n.Tables, defaultDatabase)
	case *ast.CheckTableStmt:
		collectAccessTablesFromTableRefs(result, n.Tables, defaultDatabase)
	case *ast.RepairTableStmt:
		collectAccessTablesFromTableRefs(result, n.Tables, defaultDatabase)
	case *ast.TableStmt:
		if n == nil {
			return
		}
		collectAccessTablesFromTableRef(result, n.Table, defaultDatabase)
		collectAccessTablesFromOrderBy(result, n.OrderBy, defaultDatabase)
		collectAccessTablesFromLimit(result, n.Limit, defaultDatabase)
	case *ast.ValuesStmt:
		for _, row := range n.Rows {
			for _, expr := range row {
				collectAccessTablesFromExpr(result, expr, defaultDatabase)
			}
		}
		collectAccessTablesFromOrderBy(result, n.OrderBy, defaultDatabase)
		collectAccessTablesFromLimit(result, n.Limit, defaultDatabase)
	case *ast.CallStmt:
		for _, arg := range n.Args {
			collectAccessTablesFromExpr(result, arg, defaultDatabase)
		}
	case *ast.SetStmt:
		for _, assignment := range n.Assignments {
			if assignment != nil {
				collectAccessTablesFromExpr(result, assignment.Value, defaultDatabase)
			}
		}
	case *ast.DoStmt:
		for _, expr := range n.Exprs {
			collectAccessTablesFromExpr(result, expr, defaultDatabase)
		}
	case *ast.HandlerOpenStmt:
		collectAccessTablesFromTableRef(result, n.Table, defaultDatabase)
	case *ast.HandlerReadStmt:
		collectAccessTablesFromTableRef(result, n.Table, defaultDatabase)
		collectAccessTablesFromExpr(result, n.Where, defaultDatabase)
	case *ast.HandlerCloseStmt:
		collectAccessTablesFromTableRef(result, n.Table, defaultDatabase)
	case *ast.LockTablesStmt:
		for _, table := range n.Tables {
			if table != nil {
				collectAccessTablesFromTableRef(result, table.Table, defaultDatabase)
			}
		}
	default:
	}
}

func collectAccessTablesFromOrderBy(result base.SourceColumnSet, orderByItems []*ast.OrderByItem, defaultDatabase string) {
	for _, orderBy := range orderByItems {
		if orderBy != nil {
			collectAccessTablesFromExpr(result, orderBy.Expr, defaultDatabase)
		}
	}
}

func collectAccessTablesFromLimit(result base.SourceColumnSet, limit *ast.Limit, defaultDatabase string) {
	if limit == nil {
		return
	}
	collectAccessTablesFromExpr(result, limit.Count, defaultDatabase)
	collectAccessTablesFromExpr(result, limit.Offset, defaultDatabase)
}

func collectAccessTablesFromTableRefs(result base.SourceColumnSet, tableRefs []*ast.TableRef, defaultDatabase string) {
	for _, tableRef := range tableRefs {
		collectAccessTablesFromTableRef(result, tableRef, defaultDatabase)
	}
}

func collectAccessTablesFromTableRef(result base.SourceColumnSet, tableRef *ast.TableRef, defaultDatabase string) {
	if tableRef == nil {
		return
	}
	if isDualTable(tableRef) {
		return
	}
	database := tableRef.Schema
	if database == "" {
		database = defaultDatabase
	}
	result[base.ColumnResource{
		Database: database,
		Table:    tableRef.Name,
	}] = true
}

func isDualTable(tableRef *ast.TableRef) bool {
	return tableRef != nil && tableRef.Schema == "" && strings.EqualFold(tableRef.Name, "dual")
}

func collectAccessTablesFromTableExpr(result base.SourceColumnSet, tableExpr ast.TableExpr, defaultDatabase string) {
	switch n := tableExpr.(type) {
	case *ast.TableRef:
		collectAccessTablesFromTableRef(result, n, defaultDatabase)
	case *ast.JoinClause:
		collectAccessTablesFromTableExpr(result, n.Left, defaultDatabase)
		collectAccessTablesFromTableExpr(result, n.Right, defaultDatabase)
		switch condition := n.Condition.(type) {
		case *ast.OnCondition:
			collectAccessTablesFromExpr(result, condition.Expr, defaultDatabase)
		default:
		}
	case *ast.SubqueryExpr:
		collectAccessTablesFromNode(result, n.Select, defaultDatabase)
	case *ast.JsonTableExpr:
		collectAccessTablesFromExpr(result, n.Expr, defaultDatabase)
		collectAccessTablesFromExpr(result, n.Path, defaultDatabase)
	default:
	}
}

func collectAccessTablesFromExpr(result base.SourceColumnSet, expr ast.ExprNode, defaultDatabase string) {
	switch e := expr.(type) {
	case nil:
	case *ast.ResTarget:
		collectAccessTablesFromExpr(result, e.Val, defaultDatabase)
	case *ast.BinaryExpr:
		collectAccessTablesFromExpr(result, e.Left, defaultDatabase)
		collectAccessTablesFromExpr(result, e.Right, defaultDatabase)
	case *ast.UnaryExpr:
		collectAccessTablesFromExpr(result, e.Operand, defaultDatabase)
	case *ast.FuncCallExpr:
		for _, arg := range e.Args {
			collectAccessTablesFromExpr(result, arg, defaultDatabase)
		}
		collectAccessTablesFromExpr(result, e.Separator, defaultDatabase)
		for _, orderBy := range e.OrderBy {
			if orderBy != nil {
				collectAccessTablesFromExpr(result, orderBy.Expr, defaultDatabase)
			}
		}
		if e.Over != nil {
			for _, partitionBy := range e.Over.PartitionBy {
				collectAccessTablesFromExpr(result, partitionBy, defaultDatabase)
			}
			for _, orderBy := range e.Over.OrderBy {
				if orderBy != nil {
					collectAccessTablesFromExpr(result, orderBy.Expr, defaultDatabase)
				}
			}
		}
	case *ast.SubqueryExpr:
		collectAccessTablesFromNode(result, e.Select, defaultDatabase)
	case *ast.CaseExpr:
		collectAccessTablesFromExpr(result, e.Operand, defaultDatabase)
		for _, when := range e.Whens {
			if when == nil {
				continue
			}
			collectAccessTablesFromExpr(result, when.Cond, defaultDatabase)
			collectAccessTablesFromExpr(result, when.Result, defaultDatabase)
		}
		collectAccessTablesFromExpr(result, e.Default, defaultDatabase)
	case *ast.BetweenExpr:
		collectAccessTablesFromExpr(result, e.Expr, defaultDatabase)
		collectAccessTablesFromExpr(result, e.Low, defaultDatabase)
		collectAccessTablesFromExpr(result, e.High, defaultDatabase)
	case *ast.InExpr:
		collectAccessTablesFromExpr(result, e.Expr, defaultDatabase)
		for _, item := range e.List {
			collectAccessTablesFromExpr(result, item, defaultDatabase)
		}
		collectAccessTablesFromNode(result, e.Select, defaultDatabase)
	case *ast.LikeExpr:
		collectAccessTablesFromExpr(result, e.Expr, defaultDatabase)
		collectAccessTablesFromExpr(result, e.Pattern, defaultDatabase)
		collectAccessTablesFromExpr(result, e.Escape, defaultDatabase)
	case *ast.IsExpr:
		collectAccessTablesFromExpr(result, e.Expr, defaultDatabase)
	case *ast.ExistsExpr:
		collectAccessTablesFromNode(result, e.Select, defaultDatabase)
	case *ast.CastExpr:
		collectAccessTablesFromExpr(result, e.Expr, defaultDatabase)
	case *ast.ExtractExpr:
		collectAccessTablesFromExpr(result, e.Expr, defaultDatabase)
	case *ast.IntervalExpr:
		collectAccessTablesFromExpr(result, e.Value, defaultDatabase)
	case *ast.CollateExpr:
		collectAccessTablesFromExpr(result, e.Expr, defaultDatabase)
	case *ast.MatchExpr:
		for _, column := range e.Columns {
			collectAccessTablesFromExpr(result, column, defaultDatabase)
		}
		collectAccessTablesFromExpr(result, e.Against, defaultDatabase)
	case *ast.ConvertExpr:
		collectAccessTablesFromExpr(result, e.Expr, defaultDatabase)
	case *ast.RowExpr:
		for _, item := range e.Items {
			collectAccessTablesFromExpr(result, item, defaultDatabase)
		}
	case *ast.ParenExpr:
		collectAccessTablesFromExpr(result, e.Expr, defaultDatabase)
	default:
	}
}
