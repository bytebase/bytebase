package tidb

import (
	"context"
	"slices"
	"strings"

	"github.com/bytebase/omni/tidb/ast"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/store/model"
)

// rowIDColumnName is TiDB's hidden row handle: readable on a table without a
// clustered index, listed in no catalog, so never in synced metadata.
// https://docs.pingcap.com/tidb/stable/tidb-rowid/
const rowIDColumnName = "_tidb_rowid"

type querySpanExtractor struct {
	ctx               context.Context
	defaultDatabase   string
	statement         string
	metaCache         map[string]*model.DatabaseMetadata
	gCtx              base.GetQuerySpanContext
	lowerTableViewMap map[string]map[string]bool

	ctes              []*base.PseudoTable
	outerTableSources []base.TableSource
	tableSourcesFrom  []base.TableSource

	// baseTableSources marks the aliases and joins built only from base
	// tables; base.PseudoTable drops that identity.
	baseTableSources map[base.TableSource]bool

	// viewResolutionStack tracks the views currently being expanded so a
	// cyclic view reference (a view that transitively references itself) is
	// reported as an error instead of recursing until the stack overflows.
	viewResolutionStack map[string]bool
}

func newQuerySpanExtractor(defaultDatabase string, gCtx base.GetQuerySpanContext) *querySpanExtractor {
	return &querySpanExtractor{
		defaultDatabase:   defaultDatabase,
		metaCache:         make(map[string]*model.DatabaseMetadata),
		lowerTableViewMap: make(map[string]map[string]bool),
		gCtx:              gCtx,
		baseTableSources:  make(map[base.TableSource]bool),
	}
}

func (q *querySpanExtractor) getQuerySpan(ctx context.Context, statement string) (*base.QuerySpan, error) {
	q.ctx = ctx
	q.statement = statement

	list, err := ParseTiDBOmni(statement)
	if err != nil {
		return nil, err
	}
	if list == nil || len(list.Items) != 1 {
		got := 0
		if list != nil {
			got = len(list.Items)
		}
		return nil, errors.Errorf("expecting 1 statement, but got %d", got)
	}

	node := list.Items[0]

	accessNode := node
	includeUncached := false
	if explain, ok := node.(*ast.ExplainStmt); ok {
		if explain.Stmt == nil {
			return nil, errors.New("EXPLAIN by digest or connection is not supported: referenced tables cannot be determined")
		}
		// A plan carries the tables it reads. System tables are left out: they are
		// not tracked databases, and a plain EXPLAIN may mix them with user tables.
		if !explain.Analyze {
			sourceColumns := q.getAccessTables(explain.Stmt, true)
			for resource := range sourceColumns {
				if isSystemResource(resource) {
					delete(sourceColumns, resource)
				}
			}
			return &base.QuerySpan{Type: base.Explain, Results: []base.QuerySpanResult{}, SourceColumns: sourceColumns}, nil
		}
		accessNode = explain.Stmt
		includeUncached = true
	}
	accessTables := q.getAccessTables(accessNode, includeUncached)
	allSystems, mixed := isMixedQuery(accessTables)
	if mixed {
		return nil, base.MixUserSystemTablesError
	}

	queryType, isExplainAnalyze := getQueryType(node, allSystems)
	if skipQuerySpan(node, queryType) {
		return &base.QuerySpan{
			Type:           queryType,
			ExplainAnalyze: isExplainAnalyze,
			Results:        []base.QuerySpanResult{},
			SourceColumns:  accessTables,
		}, nil
	}

	// For EXPLAIN ANALYZE SELECT, we determine the query type and access tables from the inner SELECT statement.
	if isExplainAnalyze {
		return &base.QuerySpan{
			Type:           queryType,
			ExplainAnalyze: true,
			Results:        []base.QuerySpanResult{},
			SourceColumns:  accessTables,
		}, nil
	}

	tableSource, err := q.extractTableSourceFromNode(node)
	if err != nil {
		// Surface ResourceNotFoundError on the span so the SQL service can resync stale metadata and retry.
		var resourceNotFound *base.ResourceNotFoundError
		if errors.As(err, &resourceNotFound) {
			sourceColumns := make(base.SourceColumnSet, len(accessTables)+1)
			for k := range accessTables {
				sourceColumns[k] = true
			}
			// Add the not-found target only when it carries a distinct ACL/resync anchor.
			// For unqualified column-not-found, target.Table is empty and the FROM tables
			// in accessTables already provide the right table-level ACL keys, so skip.
			target := notFoundSyncTarget(resourceNotFound, q.defaultDatabase)
			if len(sourceColumns) == 0 || target.Table != "" {
				sourceColumns[target] = true
			}
			return &base.QuerySpan{
				Type:          queryType,
				Results:       []base.QuerySpanResult{},
				SourceColumns: sourceColumns,
				NotFoundError: resourceNotFound,
			}, nil
		}
		return nil, err
	}
	if tableSource == nil {
		return &base.QuerySpan{
			Type:          queryType,
			Results:       []base.QuerySpanResult{},
			SourceColumns: accessTables,
		}, nil
	}

	return &base.QuerySpan{
		Type:          queryType,
		Results:       tableSource.GetQuerySpanResult(),
		SourceColumns: accessTables,
	}, nil
}

// notFoundSyncTarget derives a sync target for the resync loop, falling back to defaultDatabase when the error lacks one.
func notFoundSyncTarget(e *base.ResourceNotFoundError, defaultDatabase string) base.ColumnResource {
	r := base.ColumnResource{Database: defaultDatabase}
	if e.Database != nil && *e.Database != "" {
		r.Database = *e.Database
	}
	if e.Schema != nil {
		r.Schema = *e.Schema
	}
	if e.Table != nil {
		r.Table = *e.Table
	}
	if e.Column != nil {
		r.Column = *e.Column
	}
	return r
}

func skipQuerySpan(node ast.Node, queryType base.QueryType) bool {
	if queryType == base.Select {
		return false
	}

	// This is for internal view in a SELECT statement.
	if _, ok := node.(*ast.CreateViewStmt); ok {
		return false
	}

	return true
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
	database := resource.Database
	database = strings.ToLower(database)
	return systemDatabases[database]
}

var systemDatabases = map[string]bool{
	"information_schema": true,
	"mysql":              true,
	"performance_schema": true,
	"sys":                true,
	// TiDB only
	"metrics_schema": true,
}

func (q *querySpanExtractor) getAccessTables(node ast.Node, includeUncached bool) base.SourceColumnSet {
	accessesMap := make(base.SourceColumnSet)
	// ANALYZE executes before metadata can be refreshed. Keep every physical
	// target for authorization, excluding only references to in-scope CTEs.
	c := &accessTableCollector{resolveCTEs: includeUncached}
	ast.Walk(c, node)
	for _, table := range c.tables {
		databaseName := table.Schema
		if databaseName == "" {
			databaseName = q.defaultDatabase
		}

		// This is a false-positive behavior, the table we found may not be the table the query actually accesses.
		// For example, the query is `WITH t1 AS (SELECT 1) SELECT * FROM t1` and we have a physical table `t1` in the database exactly,
		// what we found is the physical table `t1`, but the query actually accesses the CTE `t1`.
		if !includeUncached && !q.existsTableMetadata(databaseName, table.Name) {
			continue
		}

		resource := base.ColumnResource{
			Database: databaseName,
			Table:    table.Name,
		}
		accessesMap[resource] = true
	}
	return accessesMap
}

// accessTableCollector walks an omni AST collecting the base TableRef nodes a
// statement reads, skipping references to in-scope CTEs (when resolveCTEs) and
// the alias-only tables named by a locking clause.
type accessTableCollector struct {
	tables        []*ast.TableRef
	resolveCTEs   bool
	cteScopes     []map[string]bool
	lockingTables map[*ast.TableRef]bool
}

func (c *accessTableCollector) Visit(node ast.Node) ast.Visitor {
	if node == nil {
		return nil
	}
	return c.enter(node)
}

func (c *accessTableCollector) enter(node ast.Node) ast.Visitor {
	// Locking clauses reference FROM aliases, whose physical sources are visited separately.
	if sel, ok := node.(*ast.SelectStmt); ok && sel.ForUpdate != nil {
		if c.lockingTables == nil {
			c.lockingTables = make(map[*ast.TableRef]bool)
		}
		for _, table := range sel.ForUpdate.Tables {
			c.lockingTables[table] = true
		}
	}
	// A recursive CTE's name is visible inside its own definition.
	if cte, ok := node.(*ast.CommonTableExpr); ok && c.resolveCTEs && cte.Recursive && len(c.cteScopes) > 0 {
		c.cteScopes[len(c.cteScopes)-1][strings.ToLower(cte.Name)] = true
	}
	if table, ok := node.(*ast.TableRef); ok {
		if c.lockingTables[table] {
			return nil
		}
		if c.resolveCTEs && table.Schema == "" {
			for _, scope := range c.cteScopes {
				if scope[strings.ToLower(table.Name)] {
					return nil
				}
			}
		}
		c.tables = append(c.tables, table)
		return nil
	}
	if c.resolveCTEs {
		switch node.(type) {
		case *ast.SelectStmt, *ast.UpdateStmt, *ast.DeleteStmt:
			c.cteScopes = append(c.cteScopes, make(map[string]bool))
			return &accessTableScopeGuard{c: c}
		}
		if cte, ok := node.(*ast.CommonTableExpr); ok {
			// A non-recursive CTE becomes visible only after its definition.
			return &accessTableCTEGuard{c: c, name: strings.ToLower(cte.Name)}
		}
	}
	return c
}

// accessTableScopeGuard pops the CTE scope its enclosing query pushed.
type accessTableScopeGuard struct {
	c *accessTableCollector
}

func (g *accessTableScopeGuard) Visit(node ast.Node) ast.Visitor {
	if node == nil {
		g.c.cteScopes = g.c.cteScopes[:len(g.c.cteScopes)-1]
		return nil
	}
	return g.c.enter(node)
}

// accessTableCTEGuard registers the CTE name in the enclosing scope on leave.
type accessTableCTEGuard struct {
	c    *accessTableCollector
	name string
}

func (g *accessTableCTEGuard) Visit(node ast.Node) ast.Visitor {
	if node == nil {
		if len(g.c.cteScopes) > 0 {
			g.c.cteScopes[len(g.c.cteScopes)-1][g.name] = true
		}
		return nil
	}
	return g.c.enter(node)
}

func (q *querySpanExtractor) existsTableMetadata(databaseName string, tableName string) bool {
	if databaseName == "" {
		databaseName = q.defaultDatabase
	}

	if q.lowerTableViewMap[databaseName] != nil {
		return q.lowerTableViewMap[databaseName][strings.ToLower(tableName)]
	}

	databaseMetadata, err := q.getDatabaseMetadata(databaseName)
	if err != nil {
		return false
	}

	if databaseMetadata == nil {
		return false
	}
	schemaMetadata := databaseMetadata.GetSchemaMetadata("")
	if schemaMetadata == nil {
		return false
	}

	tableViewMap := make(map[string]bool)
	for _, table := range schemaMetadata.ListTableNames() {
		tableViewMap[strings.ToLower(table)] = true
	}
	for _, view := range schemaMetadata.ListViewNames() {
		tableViewMap[strings.ToLower(view)] = true
	}
	q.lowerTableViewMap[databaseName] = tableViewMap

	return q.lowerTableViewMap[databaseName][strings.ToLower(tableName)]
}

// extractTableSourceFromNode extracts the output columns of a query node (a
// SELECT, a set operation, or a CREATE VIEW body) as a table source.
func (q *querySpanExtractor) extractTableSourceFromNode(node ast.Node) (base.TableSource, error) {
	if node == nil {
		return nil, nil
	}
	switch node := node.(type) {
	case *ast.SelectStmt:
		return q.extractSelect(node)
	case *ast.CreateViewStmt:
		tableSource, err := q.extractTableSourceFromNode(node.Select)
		if err != nil {
			return nil, err
		}
		querySpanResult := tableSource.GetQuerySpanResult()

		viewName := ""
		if node.Name != nil {
			viewName = node.Name.Name
		}
		if len(node.Columns) == 0 {
			return base.NewPseudoTable(viewName, querySpanResult), nil
		}

		if len(node.Columns) != len(querySpanResult) {
			return nil, errors.Errorf("The used SELECT statements have a different number of columns for view %s", viewName)
		}

		var columns []base.QuerySpanResult
		for i, item := range querySpanResult {
			columns = append(columns, base.QuerySpanResult{
				Name:          strings.ToLower(node.Columns[i]),
				SourceColumns: item.SourceColumns,
			})
		}
		return base.NewPseudoTable(viewName, columns), nil
	}
	return nil, nil
}

func (q *querySpanExtractor) extractNonRecursiveCTE(node *ast.CommonTableExpr) (*base.PseudoTable, error) {
	tableSource, err := q.extractTableSourceFromNode(node.Select)
	if err != nil {
		return nil, errors.Wrap(err, "failed to extract table source from CTE query")
	}
	querySpanResults := tableSource.GetQuerySpanResult()
	if len(node.Columns) > 0 {
		if len(node.Columns) != len(querySpanResults) {
			return nil, errors.Errorf("cte table expr has %d columns, but alias has %d columns", len(querySpanResults), len(node.Columns))
		}
		for i, name := range node.Columns {
			// The column name for MySQL is case insensitive.
			querySpanResults[i].Name = strings.ToLower(name)
		}
	}

	return &base.PseudoTable{
		Name:    node.Name,
		Columns: querySpanResults,
	}, nil
}

func (q *querySpanExtractor) extractRecursiveCTE(node *ast.CommonTableExpr) (*base.PseudoTable, error) {
	// omni folds UNION/INTERSECT/EXCEPT into SelectStmt; a recursive CTE whose
	// body is not a set operation has no recursive arm.
	selectStmt := ast.UnwrapParenSource(node.Select)
	if selectStmt == nil || selectStmt.SetOp == ast.SetOpNone {
		return q.extractNonRecursiveCTE(node)
	}

	// The CTE may hide an outer CTE of the same name with its own inner WITH; in
	// that case the recursion does not reference itself and it is non-recursive.
	for _, cte := range selectStmt.CTEs {
		if strings.EqualFold(cte.Name, node.Name) {
			return q.extractNonRecursiveCTE(node)
		}
	}

	// The body's own WITH clause must be visible to the anchor, e.g. the `seed`
	// in `WITH RECURSIVE r AS (WITH seed AS (...) SELECT ... FROM seed UNION ALL
	// SELECT ... FROM r) ...`. The fixpoint loop below re-registers these via
	// extractSelect, so this scope is scoped to the initial anchor extraction.
	previousInnerCteLength := len(q.ctes)
	for _, cte := range selectStmt.CTEs {
		cteTableSource, cteErr := q.extractCTE(cte)
		if cteErr != nil {
			q.ctes = q.ctes[:previousInnerCteLength]
			return nil, cteErr
		}
		q.ctes = append(q.ctes, cteTableSource)
	}
	initialTableSource, err := q.extractRecursiveCTEAnchor(selectStmt)
	q.ctes = q.ctes[:previousInnerCteLength]
	if err != nil {
		return nil, errors.Wrap(err, "failed to extract table source from initial part of recursive CTE")
	}
	initialQuerySpanResult := cloneQuerySpanResults(initialTableSource.GetQuerySpanResult())
	if len(node.Columns) > 0 {
		if len(node.Columns) != len(initialQuerySpanResult) {
			return nil, errors.Errorf("cte table expr has %d columns, but alias has %d columns", len(initialQuerySpanResult), len(node.Columns))
		}
		for i, name := range node.Columns {
			// The column name for MySQL is case insensitive.
			initialQuerySpanResult[i].Name = strings.ToLower(name)
		}
	}

	cteTableResource := &base.PseudoTable{
		Name:    node.Name,
		Columns: initialQuerySpanResult,
	}

	// Simulate the recursive process: iterate until the source column sets of
	// the CTE columns stop growing. The number of iterations never exceeds the
	// number of columns.
	q.ctes = append(q.ctes, cteTableResource)
	defer func() {
		q.ctes = q.ctes[:len(q.ctes)-1]
	}()
	for {
		recursiveTableSource, err := q.extractSelect(node.Select)
		if err != nil {
			return nil, errors.Wrap(err, "failed to extract table source from recursive part of recursive CTE")
		}
		recursiveQuerySpanResult := recursiveTableSource.GetQuerySpanResult()
		if len(recursiveQuerySpanResult) != len(initialQuerySpanResult) {
			return nil, errors.Errorf("cte table expr has %d columns, but recursive part has %d columns", len(initialQuerySpanResult), len(recursiveQuerySpanResult))
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
		cteTableResource.Columns = cloneQuerySpanResults(initialQuerySpanResult)
	}
	cteTableResource.Columns = initialQuerySpanResult
	return cteTableResource, nil
}

// extractRecursiveCTEAnchor extracts the non-recursive anchor of a recursive
// CTE: the leftmost SELECT of its set-operation tree.
func (q *querySpanExtractor) extractRecursiveCTEAnchor(stmt *ast.SelectStmt) (base.TableSource, error) {
	stmt = ast.UnwrapParenSource(stmt)
	if stmt == nil || stmt.SetOp == ast.SetOpNone {
		return q.extractSelect(stmt)
	}
	if stmt.Left != nil {
		return q.extractRecursiveCTEAnchor(stmt.Left)
	}
	return q.extractRecursiveCTEAnchor(stmt.Right)
}

func (q *querySpanExtractor) extractCTE(node *ast.CommonTableExpr) (*base.PseudoTable, error) {
	if node.Recursive {
		return q.extractRecursiveCTE(node)
	}
	return q.extractNonRecursiveCTE(node)
}

func (q *querySpanExtractor) extractSelect(node *ast.SelectStmt) (base.TableSource, error) {
	if node == nil {
		return new(base.PseudoTable), nil
	}
	if len(node.CTEs) > 0 {
		previousCteOuterLength := len(q.ctes)
		defer func() {
			q.ctes = q.ctes[:previousCteOuterLength]
		}()
		for _, cte := range node.CTEs {
			cteTableSource, err := q.extractCTE(cte)
			if err != nil {
				return nil, err
			}
			q.ctes = append(q.ctes, cteTableSource)
		}
	}

	switch {
	case node.ParenSource != nil:
		return q.extractSelect(node.ParenSource)
	case node.SetOp != ast.SetOpNone:
		return q.extractSetOp(node)
	case node.TableSource != nil:
		if node.TableSource.Table == nil {
			return new(base.PseudoTable), nil
		}
		return q.findTableSchema(node.TableSource.Table.Schema, node.TableSource.Table.Name)
	case node.ValuesSource != nil:
		return q.extractValues(node.ValuesSource)
	}

	var fromFieldList []base.TableSource
	if len(node.From) > 0 {
		previousTableSourcesFromLength := len(q.tableSourcesFrom)
		defer func() {
			q.tableSourcesFrom = q.tableSourcesFrom[:previousTableSourcesFromLength]
		}()
		for _, tableExpr := range node.From {
			tableSource, err := q.extractTableSource(tableExpr)
			if err != nil {
				return nil, errors.Wrap(err, "failed to extract table source from FROM clause")
			}
			if tableSource == nil {
				continue
			}
			q.tableSourcesFrom = append(q.tableSourcesFrom, tableSource)
			fromFieldList = append(fromFieldList, tableSource)
		}
	}

	result := new(base.PseudoTable)
	for _, field := range node.TargetList {
		columns, err := q.extractTargetColumns(field, fromFieldList)
		if err != nil {
			return nil, err
		}
		result.Columns = append(result.Columns, columns...)
	}

	return result, nil
}

// extractTargetColumns resolves one select-list item to its output columns,
// expanding `*` and `table.*` against the FROM sources.
func (q *querySpanExtractor) extractTargetColumns(field ast.ExprNode, fromFieldList []base.TableSource) ([]base.QuerySpanResult, error) {
	expr := field
	name := ""
	if resTarget, ok := field.(*ast.ResTarget); ok {
		expr = resTarget.Val
		name = resTarget.Name
	}

	// Bare `*`.
	if _, ok := expr.(*ast.StarExpr); ok {
		var columns []base.QuerySpanResult
		for _, tableSource := range fromFieldList {
			columns = append(columns, tableSource.GetQuerySpanResult()...)
		}
		return columns, nil
	}

	// `table.*` / `db.table.*`.
	if columnRef, ok := expr.(*ast.ColumnRef); ok && columnRef.Star {
		return q.extractTableWildcard(columnRef.Schema, columnRef.Table, fromFieldList)
	}

	sourceColumnSet, err := q.extractSourceColumnSetFromExpression(expr)
	if err != nil {
		return nil, err
	}
	if name == "" {
		name = q.deriveFieldName(expr)
	}
	return []base.QuerySpanResult{{
		Name:          name,
		SourceColumns: sourceColumnSet,
	}}, nil
}

func (q *querySpanExtractor) extractTableWildcard(databaseName, tableName string, fromFieldList []base.TableSource) ([]base.QuerySpanResult, error) {
	for _, tableSource := range fromFieldList {
		sameDatabase := strings.EqualFold(databaseName, tableSource.GetDatabaseName()) || (databaseName == "" && strings.EqualFold(tableSource.GetDatabaseName(), q.defaultDatabase))
		sameTable := strings.EqualFold(tableName, tableSource.GetTableName())
		if sameDatabase && sameTable {
			return tableSource.GetQuerySpanResult(), nil
		}
	}
	if sources, ok := q.getAllTableColumnSources(databaseName, tableName); ok {
		return sources, nil
	}
	return nil, &base.ResourceNotFoundError{
		Err:      errors.New("failed to find table to calculate asterisk"),
		Database: &databaseName,
		Table:    &tableName,
	}
}

// deriveFieldName reproduces the auto-derived name of an unaliased select item:
// a bare column's name, else the item's original source text.
func (q *querySpanExtractor) deriveFieldName(expr ast.ExprNode) string {
	if columnRef, ok := expr.(*ast.ColumnRef); ok && !columnRef.Star {
		return columnRef.Column
	}
	if loc, ok := omniExprLoc(expr); ok && loc.Start >= 0 && loc.End > loc.Start && loc.End <= len(q.statement) {
		return strings.TrimSpace(q.statement[loc.Start:loc.End])
	}
	return ""
}

func (q *querySpanExtractor) extractValues(node *ast.ValuesStmt) (base.TableSource, error) {
	result := new(base.PseudoTable)
	if node == nil {
		return result, nil
	}
	for i, row := range node.Rows {
		for j, expr := range row {
			sourceColumnSet, err := q.extractSourceColumnSetFromExpression(expr)
			if err != nil {
				return nil, err
			}
			if i == 0 {
				result.Columns = append(result.Columns, base.QuerySpanResult{
					Name:          q.deriveFieldName(expr),
					SourceColumns: sourceColumnSet,
				})
				continue
			}
			if j < len(result.Columns) {
				result.Columns[j].SourceColumns, _ = base.MergeSourceColumnSet(result.Columns[j].SourceColumns, sourceColumnSet)
			}
		}
	}
	return result, nil
}

func (q *querySpanExtractor) extractSourceColumnSetFromExpressionList(list []ast.ExprNode) (base.SourceColumnSet, error) {
	result := make(base.SourceColumnSet)
	for _, node := range list {
		sourceColumnSet, err := q.extractSourceColumnSetFromExpression(node)
		if err != nil {
			return nil, err
		}
		result, _ = base.MergeSourceColumnSet(result, sourceColumnSet)
	}
	return result, nil
}

func (q *querySpanExtractor) extractSourceColumnSetFromExpression(in ast.ExprNode) (base.SourceColumnSet, error) {
	if in == nil {
		return base.SourceColumnSet{}, nil
	}

	switch node := in.(type) {
	case *ast.ColumnRef:
		if node.Star {
			return base.SourceColumnSet{}, nil
		}
		database, table, column := node.Schema, node.Table, node.Column
		sources, ok := q.getFieldColumnSource(database, table, column)
		if ok {
			return sources, nil
		}
		// The row handle carries no column data, hence no lineage.
		if q.isRowIDReference(database, table, column) {
			return base.SourceColumnSet{}, nil
		}
		return base.SourceColumnSet{}, &base.ResourceNotFoundError{
			Err:      errors.New("cannot find the column ref"),
			Database: &database,
			Table:    &table,
			Column:   &column,
		}
	case *ast.BinaryExpr:
		return q.extractSourceColumnSetFromExpressionList([]ast.ExprNode{node.Left, node.Right})
	case *ast.UnaryExpr:
		return q.extractSourceColumnSetFromExpression(node.Operand)
	case *ast.FuncCallExpr:
		var exprs []ast.ExprNode
		exprs = append(exprs, node.Args...)
		for _, item := range node.OrderBy {
			if item != nil {
				exprs = append(exprs, item.Expr)
			}
		}
		if node.Over != nil {
			exprs = append(exprs, node.Over.PartitionBy...)
			for _, item := range node.Over.OrderBy {
				if item != nil {
					exprs = append(exprs, item.Expr)
				}
			}
		}
		return q.extractSourceColumnSetFromExpressionList(exprs)
	case *ast.CastExpr:
		return q.extractSourceColumnSetFromExpression(node.Expr)
	case *ast.ConvertExpr:
		return q.extractSourceColumnSetFromExpression(node.Expr)
	case *ast.InExpr:
		nodeList := []ast.ExprNode{node.Expr}
		nodeList = append(nodeList, node.List...)
		sourceColumnSet, err := q.extractSourceColumnSetFromExpressionList(nodeList)
		if err != nil {
			return nil, err
		}
		if node.Select != nil {
			selectColumns, err := q.extractSubquerySourceColumnSet(node.Select)
			if err != nil {
				return nil, err
			}
			sourceColumnSet, _ = base.MergeSourceColumnSet(sourceColumnSet, selectColumns)
		}
		return sourceColumnSet, nil
	case *ast.LikeExpr:
		return q.extractSourceColumnSetFromExpressionList([]ast.ExprNode{node.Expr, node.Pattern, node.Escape})
	case *ast.SubqueryExpr:
		return q.extractSubquerySourceColumnSet(node.Select)
	case *ast.ExistsExpr:
		return q.extractSubquerySourceColumnSet(node.Select)
	case *ast.IsExpr:
		return q.extractSourceColumnSetFromExpression(node.Expr)
	case *ast.BetweenExpr:
		return q.extractSourceColumnSetFromExpressionList([]ast.ExprNode{node.Expr, node.Low, node.High})
	case *ast.CaseExpr:
		var nodeList []ast.ExprNode
		nodeList = append(nodeList, node.Operand)
		for _, whenClause := range node.Whens {
			if whenClause == nil {
				continue
			}
			nodeList = append(nodeList, whenClause.Cond, whenClause.Result)
		}
		nodeList = append(nodeList, node.Default)
		return q.extractSourceColumnSetFromExpressionList(nodeList)
	case *ast.ParenExpr:
		return q.extractSourceColumnSetFromExpression(node.Expr)
	case *ast.RowExpr:
		return q.extractSourceColumnSetFromExpressionList(node.Items)
	case *ast.MatchExpr:
		var nodeList []ast.ExprNode
		for _, column := range node.Columns {
			nodeList = append(nodeList, column)
		}
		nodeList = append(nodeList, node.Against)
		return q.extractSourceColumnSetFromExpressionList(nodeList)
	case *ast.IntervalExpr:
		return q.extractSourceColumnSetFromExpression(node.Value)
	case *ast.CollateExpr:
		return q.extractSourceColumnSetFromExpression(node.Expr)
	case *ast.ExtractExpr:
		return q.extractSourceColumnSetFromExpression(node.Expr)
	case *ast.MemberOfExpr:
		return q.extractSourceColumnSetFromExpressionList([]ast.ExprNode{node.Value, node.Array})
	case *ast.PositionExpr:
		return q.extractSourceColumnSetFromExpressionList([]ast.ExprNode{node.Substr, node.Str})
	default:
		// Literals, variable references, default values, keyword arguments and
		// the like carry no column lineage.
	}
	return base.SourceColumnSet{}, nil
}

// extractSubquerySourceColumnSet extracts the merged source columns of a
// subquery, giving it the current FROM as its outer scope so a correlated
// reference resolves against the enclosing query.
func (q *querySpanExtractor) extractSubquerySourceColumnSet(selectStmt *ast.SelectStmt) (base.SourceColumnSet, error) {
	if selectStmt == nil {
		return base.SourceColumnSet{}, nil
	}
	subqueryExtractor := &querySpanExtractor{
		ctx:                 q.ctx,
		defaultDatabase:     q.defaultDatabase,
		statement:           q.statement,
		metaCache:           q.metaCache,
		gCtx:                q.gCtx,
		lowerTableViewMap:   q.lowerTableViewMap,
		ctes:                q.ctes,
		outerTableSources:   slices.Concat(q.outerTableSources, q.tableSourcesFrom),
		tableSourcesFrom:    []base.TableSource{},
		baseTableSources:    q.baseTableSources,
		viewResolutionStack: cloneViewResolutionStack(q.viewResolutionStack),
	}
	tableSource, err := subqueryExtractor.extractTableSourceFromNode(selectStmt)
	if err != nil {
		return base.SourceColumnSet{}, errors.Wrap(err, "failed to extract table source from subquery")
	}
	sourceColumnSet := base.SourceColumnSet{}
	for _, field := range tableSource.GetQuerySpanResult() {
		sourceColumnSet, _ = base.MergeSourceColumnSet(sourceColumnSet, field.SourceColumns)
	}
	return sourceColumnSet, nil
}

func (q *querySpanExtractor) getAllTableColumnSources(databaseName, tableName string) ([]base.QuerySpanResult, bool) {
	findInTableSource := func(tableSource base.TableSource) ([]base.QuerySpanResult, bool) {
		if databaseName != "" && !strings.EqualFold(databaseName, tableSource.GetDatabaseName()) {
			return nil, false
		}
		if databaseName == "" && tableSource.GetDatabaseName() != "" && !strings.EqualFold(tableSource.GetDatabaseName(), q.defaultDatabase) {
			return nil, false
		}
		if tableName != "" && !strings.EqualFold(tableName, tableSource.GetTableName()) {
			return nil, false
		}
		return tableSource.GetQuerySpanResult(), true
	}

	// The statement's own FROM is searched first, as in candidateTableSources.
	for _, tableSource := range q.tableSourcesFrom {
		if sourceColumnSet, ok := findInTableSource(tableSource); ok {
			return sourceColumnSet, true
		}
	}

	for i := len(q.outerTableSources) - 1; i >= 0; i-- {
		tableSource := q.outerTableSources[i]
		if sourceColumnSet, ok := findInTableSource(tableSource); ok {
			return sourceColumnSet, true
		}
	}

	return nil, false
}

func (q *querySpanExtractor) getFieldColumnSource(databaseName, tableName, fieldName string) (base.SourceColumnSet, bool) {
	for _, tableSource := range q.candidateTableSources(databaseName, tableName) {
		for _, field := range tableSource.GetQuerySpanResult() {
			if strings.EqualFold(field.Name, fieldName) {
				return field.SourceColumns, true
			}
		}
	}
	return base.SourceColumnSet{}, false
}

// isRowIDReference reports whether an unresolved column reference is the row
// handle. Every table source it can bind to must be built from base tables,
// where TiDB rejects this name for a real column (ERROR 1166). A view or
// subquery can own one that cached metadata lacks, and only the not-found
// error triggers the resync that finds it.
func (q *querySpanExtractor) isRowIDReference(databaseName, tableName, fieldName string) bool {
	if !strings.EqualFold(fieldName, rowIDColumnName) {
		return false
	}
	candidates := q.candidateTableSources(databaseName, tableName)
	for _, tableSource := range candidates {
		if !q.isBaseTableSource(tableSource) {
			return false
		}
	}
	return len(candidates) > 0
}

func (q *querySpanExtractor) isBaseTableSource(tableSource base.TableSource) bool {
	if _, ok := tableSource.(*base.PhysicalTable); ok {
		return true
	}
	return q.baseTableSources[tableSource]
}

// candidateTableSources returns the table sources a column reference with the
// given qualifiers can bind to, in resolution order. The statement's own FROM
// is closer than any enclosing query, so it is searched first.
func (q *querySpanExtractor) candidateTableSources(databaseName, tableName string) []base.TableSource {
	var candidates []base.TableSource
	appendIfMatch := func(tableSource base.TableSource) {
		if databaseName != "" && !strings.EqualFold(databaseName, tableSource.GetDatabaseName()) {
			return
		}
		if tableName != "" && !strings.EqualFold(tableName, tableSource.GetTableName()) {
			return
		}
		candidates = append(candidates, tableSource)
	}

	for _, tableSource := range q.tableSourcesFrom {
		appendIfMatch(tableSource)
	}
	for i := len(q.outerTableSources) - 1; i >= 0; i-- {
		appendIfMatch(q.outerTableSources[i])
	}
	return candidates
}

func (q *querySpanExtractor) extractTableSource(tableExpr ast.TableExpr) (base.TableSource, error) {
	switch t := tableExpr.(type) {
	case nil:
		return nil, nil
	case *ast.TableRef:
		tableSource, err := q.findTableSchema(t.Schema, t.Name)
		if err != nil {
			return nil, err
		}
		if t.Alias == "" {
			return tableSource, nil
		}
		aliased := base.NewPseudoTable(t.Alias, tableSource.GetQuerySpanResult())
		if q.isBaseTableSource(tableSource) {
			q.baseTableSources[aliased] = true
		}
		return aliased, nil
	case *ast.SubqueryExpr:
		return q.extractDerivedTable(t)
	case *ast.JoinClause:
		return q.extractJoin(t)
	default:
		return nil, errors.Errorf("unsupported table source %T", tableExpr)
	}
}

func (q *querySpanExtractor) extractDerivedTable(node *ast.SubqueryExpr) (base.TableSource, error) {
	tableSource, err := q.extractTableSourceFromNode(node.Select)
	if err != nil {
		return nil, err
	}
	results := cloneQuerySpanResults(tableSource.GetQuerySpanResult())
	if len(node.Columns) > 0 {
		if len(node.Columns) != len(results) {
			return nil, errors.Errorf("derived table column list length %d doesn't match result column length %d", len(node.Columns), len(results))
		}
		for i := range node.Columns {
			results[i].Name = strings.ToLower(node.Columns[i])
		}
	}
	name := node.Alias
	if name == "" {
		name = tableSource.GetTableName()
	}
	return base.NewPseudoTable(name, results), nil
}

func (q *querySpanExtractor) extractJoin(node *ast.JoinClause) (base.TableSource, error) {
	leftTableSource, err := q.extractTableSource(node.Left)
	if err != nil {
		return nil, errors.Wrap(err, "failed to extract table source from left table")
	}
	if leftTableSource != nil {
		q.tableSourcesFrom = append(q.tableSourcesFrom, leftTableSource)
	}
	rightTableSource, err := q.extractTableSource(node.Right)
	if err != nil {
		return nil, errors.Wrap(err, "failed to extract table source from right table")
	}
	if rightTableSource == nil {
		return leftTableSource, nil
	}
	q.tableSourcesFrom = append(q.tableSourcesFrom, rightTableSource)
	joined := q.mergeJoinTableSource(node, leftTableSource, rightTableSource)
	if leftTableSource != nil && q.isBaseTableSource(leftTableSource) && q.isBaseTableSource(rightTableSource) {
		q.baseTableSources[joined] = true
	}
	return joined, nil
}

func (*querySpanExtractor) mergeJoinTableSource(node *ast.JoinClause, leftTableSource, rightTableSource base.TableSource) *base.PseudoTable {
	leftSpanResult, rightSpanResult := leftTableSource.GetQuerySpanResult(), rightTableSource.GetQuerySpanResult()

	result := new(base.PseudoTable)

	leftSpanResultIdx, rightSpanResultIdx := make(map[string]int), make(map[string]int)
	for i, spanResult := range leftSpanResult {
		// Column name in TiDB is case insensitive.
		leftSpanResultIdx[strings.ToLower(spanResult.Name)] = i
	}
	for i, spanResult := range rightSpanResult {
		rightSpanResultIdx[strings.ToLower(spanResult.Name)] = i
	}

	natural := node.Type == ast.JoinNatural || node.Type == ast.JoinNaturalLeft || node.Type == ast.JoinNaturalRight
	var using []string
	if usingCondition, ok := node.Condition.(*ast.UsingCondition); ok && usingCondition != nil {
		using = usingCondition.Columns
	}

	if natural {
		// Natural Join will merge the same column name field.
		for _, spanResult := range leftSpanResult {
			if rightIdx, exists := rightSpanResultIdx[strings.ToLower(spanResult.Name)]; exists {
				spanResult.SourceColumns, _ = base.MergeSourceColumnSet(spanResult.SourceColumns, rightSpanResult[rightIdx].SourceColumns)
			}
			result.Columns = append(result.Columns, spanResult)
		}
		for _, spanResult := range rightSpanResult {
			if _, exists := leftSpanResultIdx[strings.ToLower(spanResult.Name)]; !exists {
				result.Columns = append(result.Columns, spanResult)
			}
		}
	} else if len(using) != 0 {
		// ... JOIN ... USING (...) will merge the column in USING.
		usingMap := make(map[string]bool)
		for _, column := range using {
			usingMap[strings.ToLower(column)] = true
		}

		for _, spanResult := range leftSpanResult {
			_, existsInUsingMap := usingMap[strings.ToLower(spanResult.Name)]
			rightIdx, existsInRightField := rightSpanResultIdx[strings.ToLower(spanResult.Name)]
			if existsInUsingMap && existsInRightField {
				rightSpanResult[rightIdx].SourceColumns, _ = base.MergeSourceColumnSet(spanResult.SourceColumns, rightSpanResult[rightIdx].SourceColumns)
			}
			result.Columns = append(result.Columns, spanResult)
		}

		for _, spanResult := range rightSpanResult {
			_, existsInUsing := usingMap[strings.ToLower(spanResult.Name)]
			_, existsInLeftField := leftSpanResultIdx[strings.ToLower(spanResult.Name)]
			if existsInUsing && existsInLeftField {
				continue
			}
			result.Columns = append(result.Columns, spanResult)
		}
	} else {
		result.Columns = append(result.Columns, leftSpanResult...)
		result.Columns = append(result.Columns, rightSpanResult...)
	}

	return result
}

func (q *querySpanExtractor) findTableSchema(databaseName string, tableName string) (base.TableSource, error) {
	// Each CTE name in one WITH clause must be unique, but we can use the same
	// name in a different-level CTE, so we loop in reverse to use the closest.
	if databaseName == "" {
		for i := len(q.ctes) - 1; i >= 0; i-- {
			cte := q.ctes[i]
			// TiDB fixes lower_case_table_names at 2, so `FROM c` reads CTE `C`
			// even when a base table `c` exists.
			if strings.EqualFold(cte.Name, tableName) {
				return cte, nil
			}
		}
	}

	if databaseName == "" {
		databaseName = q.defaultDatabase
	}

	dbMetadata, err := q.getDatabaseMetadata(databaseName)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get database metadata for %q", databaseName)
	}
	if dbMetadata == nil {
		return nil, &base.ResourceNotFoundError{
			Database: &databaseName,
		}
	}
	emptySchema := ""
	schema := dbMetadata.GetSchemaMetadata("")
	if schema == nil {
		return nil, &base.ResourceNotFoundError{
			Database: &databaseName,
			Schema:   &emptySchema,
		}
	}
	lowerTableName := strings.ToLower(tableName)
	for _, table := range schema.ListTableNames() {
		if lowerTableName == strings.ToLower(table) {
			var columns []string
			tableMeta := schema.GetTable(table)
			if tableMeta != nil {
				for _, column := range tableMeta.GetProto().GetColumns() {
					columns = append(columns, column.Name)
				}
				return &base.PhysicalTable{
					Database: databaseName,
					Name:     tableMeta.GetProto().Name,
					Columns:  columns,
				}, nil
			}
		}
	}

	for _, view := range schema.ListViewNames() {
		if lowerTableName == strings.ToLower(view) {
			viewMeta := schema.GetView(view)
			if viewMeta != nil {
				columns, err := q.getColumnsForView(databaseName, viewMeta.Name, viewMeta.Definition)
				if err != nil {
					return nil, err
				}
				return &base.PhysicalView{
					Database: databaseName,
					Name:     viewMeta.Name,
					Columns:  columns,
				}, nil
			}
		}
	}

	return nil, &base.ResourceNotFoundError{
		Database: &databaseName,
		Schema:   &emptySchema,
		Table:    &tableName,
	}
}

func (q *querySpanExtractor) getColumnsForView(databaseName, viewName, definition string) ([]base.QuerySpanResult, error) {
	key := viewResolutionKey(databaseName, viewName)
	if q.viewResolutionStack[key] {
		return nil, errors.Errorf("cyclic view reference detected while resolving %q", viewName)
	}
	newQ := newQuerySpanExtractor(q.defaultDatabase, q.gCtx)
	newQ.viewResolutionStack = cloneViewResolutionStack(q.viewResolutionStack)
	newQ.viewResolutionStack[key] = true
	span, err := newQ.getQuerySpan(q.ctx, definition)
	if err != nil {
		return nil, err
	}
	if span.NotFoundError != nil {
		return nil, span.NotFoundError
	}
	return span.Results, nil
}

func viewResolutionKey(databaseName, viewName string) string {
	return databaseName + "\x00" + viewName
}

func cloneViewResolutionStack(in map[string]bool) map[string]bool {
	out := make(map[string]bool, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func cloneQuerySpanResults(results []base.QuerySpanResult) []base.QuerySpanResult {
	clone := make([]base.QuerySpanResult, len(results))
	copy(clone, results)
	return clone
}

func (q *querySpanExtractor) getDatabaseMetadata(databaseName string) (*model.DatabaseMetadata, error) {
	if meta, ok := q.metaCache[databaseName]; ok {
		return meta, nil
	}
	_, meta, err := q.gCtx.GetDatabaseMetadataFunc(q.ctx, q.gCtx.InstanceID, databaseName)
	if err != nil {
		return nil, err
	}
	q.metaCache[databaseName] = meta
	return meta, nil
}

func (q *querySpanExtractor) extractSetOp(node *ast.SelectStmt) (base.TableSource, error) {
	if node == nil || node.SetOp == ast.SetOpNone {
		return q.extractSelect(node)
	}
	left, err := q.extractSelect(node.Left)
	if err != nil {
		return nil, err
	}
	right, err := q.extractSelect(node.Right)
	if err != nil {
		return nil, err
	}
	return mergeTableSource(left, right)
}

func mergeTableSource(left, right base.TableSource) (base.TableSource, error) {
	leftSpanResult, rightSpanResult := left.GetQuerySpanResult(), right.GetQuerySpanResult()

	if len(leftSpanResult) != len(rightSpanResult) {
		return nil, errors.Errorf("left table source has %d columns, but right table source has %d columns", len(leftSpanResult), len(rightSpanResult))
	}

	var result []base.QuerySpanResult
	for i, leftResult := range leftSpanResult {
		rightResult := rightSpanResult[i]
		newResourceColumns, _ := base.MergeSourceColumnSet(leftResult.SourceColumns, rightResult.SourceColumns)
		result = append(result, base.QuerySpanResult{
			Name:          leftResult.Name,
			SourceColumns: newResourceColumns,
		})
	}
	return &base.PseudoTable{
		Name:    "",
		Columns: result,
	}, nil
}
