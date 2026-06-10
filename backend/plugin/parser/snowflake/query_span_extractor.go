package snowflake

import (
	"context"

	"github.com/bytebase/omni/snowflake/ast"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

// querySpanExtractor walks an omni Snowflake AST and produces a base.QuerySpan
// describing the column-level lineage of a single SELECT/set-operation query.
//
// It is the omni-AST port of the legacy ANTLR listener-based extractor: the
// bytebase-side machinery (base.PseudoTable / base.PhysicalTable / getField /
// findTableSchema, which resolve a column reference back to its physical source
// column via the database metadata getter) is preserved unchanged; only the
// tree-walking was rewritten from ANTLR parse-tree contexts onto omni's
// hand-written ast.Node type-switches.
type querySpanExtractor struct {
	ctx context.Context

	defaultDatbase string
	defaultSchema  string
	// https://docs.snowflake.com/en/sql-reference/identifiers-syntax
	ignoreCaseSensitive bool

	gCtx base.GetQuerySpanContext
	// Private fields.
	// ctes is used to record the common table expressions (CTEs) in the query.
	// It should be shrunk to 0 after each query span extraction.
	ctes []*base.PseudoTable

	// tableSourcesFrom is used to record the table sources from the query.
	tableSourcesFrom []base.TableSource
}

func newQuerySpanExtractor(defaultDatabase, defaultSchema string, gCtx base.GetQuerySpanContext, ignoreCaseSensitive bool) *querySpanExtractor {
	if defaultSchema == "" {
		// Fall back to the default schema `PUBLIC`.
		// Reference: https://docs.snowflake.com/en/sql-reference/name-resolution#name-resolution-in-queries
		defaultSchema = "PUBLIC"
	}
	return &querySpanExtractor{
		defaultDatbase:      defaultDatabase,
		defaultSchema:       defaultSchema,
		ignoreCaseSensitive: ignoreCaseSensitive,
		gCtx:                gCtx,
	}
}

func (q *querySpanExtractor) getQuerySpan(ctx context.Context, statement string) (*base.QuerySpan, error) {
	q.ctx = ctx

	file, err := parseSnowflakeAST(statement)
	if err != nil {
		return nil, err
	}
	if file == nil || len(file.Stmts) == 0 {
		return nil, nil
	}
	if len(file.Stmts) != 1 {
		return nil, errors.Errorf("expected exactly 1 statement, got %d", len(file.Stmts))
	}

	node := file.Stmts[0]

	accessTables := getAccessTables(q.defaultDatbase, q.defaultSchema, node)
	// We do not support simultaneous access to the system table and the user table
	// because we do not synchronize the schema of the system table.
	// This causes an error (NOT_FOUND) when using querySpanExtractor.findTableSchema.
	// As a result, we exclude getting query span results for accessing only the system table.
	allSystems, mixed := isMixedQuery(accessTables, q.ignoreCaseSensitive)
	if mixed {
		return nil, base.MixUserSystemTablesError
	}

	queryType := getQueryType(node)
	// A SELECT/set-op that touches only system tables is reclassified as
	// SelectInfoSchema, matching the legacy queryTypeListener.allSystems branch.
	if queryType == base.Select && allSystems {
		queryType = base.SelectInfoSchema
	}

	if queryType != base.Select {
		return &base.QuerySpan{
			Type:          queryType,
			SourceColumns: base.SourceColumnSet{},
			Results:       []base.QuerySpanResult{},
		}, nil
	}

	result, err := q.extractPseudoTableFromQueryNode(node)
	if err != nil {
		var resourceNotFound *base.ResourceNotFoundError
		if errors.As(err, &resourceNotFound) {
			return &base.QuerySpan{
				SourceColumns: accessTables,
				Results:       []base.QuerySpanResult{},
				NotFoundError: resourceNotFound,
			}, nil
		}
		return nil, err
	}

	return &base.QuerySpan{
		Type:          queryType,
		SourceColumns: accessTables,
		Results:       result.GetQuerySpanResult(),
	}, nil
}

// extractPseudoTableFromQueryNode extracts the result columns of a query
// expression node: a *ast.SelectStmt (optionally carrying a WITH clause) or a
// *ast.SetOperationStmt.
func (q *querySpanExtractor) extractPseudoTableFromQueryNode(node ast.Node) (*base.PseudoTable, error) {
	switch n := node.(type) {
	case *ast.SelectStmt:
		// A WITH clause on this SELECT introduces CTEs visible to the SELECT body
		// and to later CTEs; push them, then pop on the way out so sibling scopes
		// don't see them. The legacy extractor handled WITH at the
		// query_statement level; omni hangs WITH off the SelectStmt itself.
		if len(n.With) > 0 {
			originalCTECount := len(q.ctes)
			if err := q.extractCTEFromWith(n.With, isRecursiveWith(n.With)); err != nil {
				return nil, err
			}
			defer func() {
				q.ctes = q.ctes[:originalCTECount]
			}()
		}
		return q.extractPseudoTableFromSelectStmt(n)
	case *ast.SetOperationStmt:
		return q.extractPseudoTableFromSetOperation(n)
	default:
		return nil, errors.Errorf("unsupported query node type %T", node)
	}
}

// isRecursiveWith reports whether any CTE in the WITH list carries the RECURSIVE
// flag. Snowflake applies RECURSIVE to the whole WITH list, so a single
// recursive CTE makes the list recursive (mirroring the legacy
// withExpression.RECURSIVE() check).
func isRecursiveWith(ctes []*ast.CTE) bool {
	for _, cte := range ctes {
		if cte.Recursive {
			return true
		}
	}
	return false
}

func (q *querySpanExtractor) extractCTEFromWith(ctes []*ast.CTE, recursiveWith bool) error {
	for _, cte := range ctes {
		normalizedCTEName := normalizeSnowflakeIdentifier(cte.Name)
		isRecursiveCTE := recursiveWith || cte.Recursive || queryNodeHasSetOperation(cte.Query)
		if isRecursiveCTE {
			if err := q.extractRecursiveCTE(normalizedCTEName, cte); err != nil {
				return err
			}
			continue
		}

		pseudoTable, err := q.extractPseudoTableFromQueryNode(cte.Query)
		if err != nil {
			return errors.Wrapf(err, "failed to extract sensitive fields of the CTE %q", normalizedCTEName)
		}
		pseudoTable, err = applyCTEColumnList(normalizedCTEName, pseudoTable, cte.Columns)
		if err != nil {
			return errors.Wrapf(err, "failed to extract sensitive fields of the CTE %q", normalizedCTEName)
		}
		q.ctes = append(q.ctes, pseudoTable)
	}
	return nil
}

// queryNodeHasSetOperation reports whether a CTE body is (or wraps) a
// set-operation, which Snowflake treats as recursive-capable. omni nests
// set-ops as *ast.SetOperationStmt; a leading WITH-bearing SELECT whose body is
// a set-op is not representable here (omni hangs WITH on the SelectStmt), so a
// plain check of the top node suffices.
func queryNodeHasSetOperation(node ast.Node) bool {
	_, ok := node.(*ast.SetOperationStmt)
	return ok
}

func (q *querySpanExtractor) extractRecursiveCTE(cteName string, cte *ast.CTE) error {
	setOp, ok := cte.Query.(*ast.SetOperationStmt)
	if !ok {
		// A recursive CTE whose body is not a set-operation has no recursive term
		// to iterate; treat it as a plain CTE (extract its single SELECT body).
		pseudoTable, err := q.extractPseudoTableFromQueryNode(cte.Query)
		if err != nil {
			return errors.Wrapf(err, "failed to extract sensitive fields of recursive CTE %q", cteName)
		}
		pseudoTable, err = applyCTEColumnList(cteName, pseudoTable, cte.Columns)
		if err != nil {
			return err
		}
		q.ctes = append(q.ctes, pseudoTable)
		return nil
	}

	// Anchor term: the left branch of the set-operation. It must be resolvable
	// without the CTE itself being visible yet.
	anchorTableSource, err := q.extractPseudoTableFromQueryNode(setOp.Left)
	if err != nil {
		return errors.Wrapf(err, "failed to extract sensitive fields of the anchor clause of recursive CTE %q", cteName)
	}
	anchorTableSource, err = applyCTEColumnList(cteName, anchorTableSource, cte.Columns)
	if err != nil {
		return errors.Wrapf(err, "failed to extract sensitive fields of the anchor clause of recursive CTE %q", cteName)
	}

	q.ctes = append(q.ctes, anchorTableSource)
	// Iterate the recursive term (the right branch) until the lineage of the CTE
	// columns reaches a fixed point, mirroring the legacy loop.
	for {
		recursivePartTableSource, err := q.extractPseudoTableFromQueryNode(setOp.Right)
		if err != nil {
			return errors.Wrapf(err, "failed to extract sensitive fields of the recursive clause of recursive CTE %q", cteName)
		}
		recursivePartTableSource, err = applyCTEColumnList(cteName, recursivePartTableSource, cte.Columns)
		if err != nil {
			return errors.Wrapf(err, "failed to extract sensitive fields of the recursive clause of recursive CTE %q", cteName)
		}

		currentCTE := q.ctes[len(q.ctes)-1]
		mergedColumns, changed, err := mergeQuerySpanResults(currentCTE.GetQuerySpanResult(), recursivePartTableSource.GetQuerySpanResult(), cteName)
		if err != nil {
			return err
		}
		q.ctes[len(q.ctes)-1] = &base.PseudoTable{
			Name:    cteName,
			Columns: mergedColumns,
		}
		if !changed {
			break
		}
	}
	return nil
}

func applyCTEColumnList(cteName string, pseudoTable *base.PseudoTable, columns []ast.Ident) (*base.PseudoTable, error) {
	if pseudoTable == nil {
		pseudoTable = &base.PseudoTable{
			Name:    cteName,
			Columns: []base.QuerySpanResult{},
		}
	}

	result := &base.PseudoTable{
		Name:    cteName,
		Columns: make([]base.QuerySpanResult, len(pseudoTable.GetQuerySpanResult())),
	}
	copy(result.Columns, pseudoTable.GetQuerySpanResult())

	if len(columns) == 0 {
		return result, nil
	}
	if len(columns) != len(result.GetQuerySpanResult()) {
		return nil, errors.Errorf("the number of columns in the CTE %q returns %d fields, but the column list returns %d fields", cteName, len(result.GetQuerySpanResult()), len(columns))
	}
	for i, columnName := range columns {
		result.Columns[i].Name = normalizeSnowflakeIdentifier(columnName)
	}
	return result, nil
}

func mergeQuerySpanResults(currentColumns, newColumns []base.QuerySpanResult, cteName string) ([]base.QuerySpanResult, bool, error) {
	if len(currentColumns) != len(newColumns) {
		return nil, false, errors.Errorf("recursive clause returns %d fields, but anchor clause returns %d fields in recursive CTE %q", len(newColumns), len(currentColumns), cteName)
	}

	mergedColumns := make([]base.QuerySpanResult, len(currentColumns))
	copy(mergedColumns, currentColumns)
	changed := false
	for i := range mergedColumns {
		var hasChange bool
		mergedColumns[i].SourceColumns, hasChange = base.MergeSourceColumnSet(mergedColumns[i].SourceColumns, newColumns[i].SourceColumns)
		changed = changed || hasChange
	}
	return mergedColumns, changed, nil
}

// extractPseudoTableFromSetOperation extracts result columns from a
// set-operation query (UNION/INTERSECT/EXCEPT). omni nests chained set-ops
// left-associatively, so the left branch may itself be a *ast.SetOperationStmt.
// The left branch's column names are kept; each right branch is merged
// positionally into the left's columns (the executor requires matching column
// counts), mirroring the legacy mergeSetOperatorColumns logic.
func (q *querySpanExtractor) extractPseudoTableFromSetOperation(setOp *ast.SetOperationStmt) (*base.PseudoTable, error) {
	left, err := q.extractPseudoTableFromQueryNode(setOp.Left)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to extract the left part of the set operation")
	}
	right, err := q.extractPseudoTableFromQueryNode(setOp.Right)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to extract the right part of the set operation")
	}
	if left == nil {
		return right, nil
	}
	if right == nil {
		return left, nil
	}
	if err := mergeSetOperatorColumns(left, right); err != nil {
		return nil, err
	}
	return left, nil
}

func mergeSetOperatorColumns(left, right *base.PseudoTable) error {
	if left == nil || right == nil {
		return nil
	}
	leftColumns := left.GetQuerySpanResult()
	rightColumns := right.GetQuerySpanResult()
	if len(leftColumns) != len(rightColumns) {
		return errors.Errorf("the number of columns in the left part of the set operation returns %d fields, but the right part returns %d fields", len(leftColumns), len(rightColumns))
	}
	for i := range rightColumns {
		left.Columns[i].SourceColumns, _ = base.MergeSourceColumnSet(leftColumns[i].SourceColumns, rightColumns[i].SourceColumns)
	}
	return nil
}

func (q *querySpanExtractor) extractPseudoTableFromSelectStmt(ctx *ast.SelectStmt) (*base.PseudoTable, error) {
	if ctx == nil {
		return nil, nil
	}

	if len(ctx.From) > 0 {
		tableSourceFrom, err := q.extractTableSourceFromFrom(ctx.From)
		if err != nil {
			return nil, err
		}
		if tableSourceFrom != nil {
			originalFromFieldsLength := len(q.tableSourcesFrom)
			q.tableSourcesFrom = append(q.tableSourcesFrom, tableSourceFrom)
			defer func() {
				q.tableSourcesFrom = q.tableSourcesFrom[:originalFromFieldsLength]
			}()
		}
	}

	result := &base.PseudoTable{
		Name:    "",
		Columns: make([]base.QuerySpanResult, 0),
	}

	for _, target := range ctx.Targets {
		if target == nil {
			continue
		}
		if target.Star {
			// SELECT * or SELECT t.* [EXCLUDE (cols)].
			var normalizedDatabaseName, normalizedSchemaName, normalizedTableName string
			if star, ok := target.Expr.(*ast.StarExpr); ok && star != nil && star.Qualifier != nil {
				normalizedDatabaseName, normalizedSchemaName, normalizedTableName = normalizeSnowflakeObjectName(star.Qualifier, "", "")
			}
			left, err := q.getAllFieldsOfTableInFromOrOuterCTE(normalizedDatabaseName, normalizedSchemaName, normalizedTableName)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to extract sensitive fields of the query statement")
			}
			left = filterExcludedColumns(left, target.Exclude)
			result.Columns = append(result.Columns, left...)
			continue
		}

		// Non-star target: a column reference resolves to a single source column;
		// any other expression collects every column it references.
		columnName, querySpanResult, err := q.extractQuerySpanResultFromTargetExpr(target.Expr)
		if err != nil {
			return nil, err
		}
		if !target.Alias.IsEmpty() {
			columnName = normalizeSnowflakeIdentifier(target.Alias)
		}
		result.Columns = append(result.Columns, base.QuerySpanResult{
			Name:          columnName,
			SourceColumns: querySpanResult.SourceColumns,
			IsPlainField:  querySpanResult.IsPlainField,
		})
	}

	return result, nil
}

// extractQuerySpanResultFromTargetExpr resolves one non-star SELECT-list
// expression into its output name and source columns. A bare column reference
// (the IsPlainField case) is resolved against the FROM scope back to its
// physical source column; any compound expression instead unions the source
// columns of every column reference inside it (IsPlainField=false).
func (q *querySpanExtractor) extractQuerySpanResultFromTargetExpr(expr ast.Node) (string, base.QuerySpanResult, error) {
	if colRef, ok := expr.(*ast.ColumnRef); ok {
		normalizedDatabaseName, normalizedSchemaName, normalizedTableName, normalizedColumnName := normalizeColumnRef(colRef)
		querySpanResult, err := q.getField(normalizedDatabaseName, normalizedSchemaName, normalizedTableName, normalizedColumnName)
		if err != nil {
			return "", base.QuerySpanResult{}, errors.Wrapf(err, "failed to check whether the column %q is sensitive", normalizedColumnName)
		}
		return querySpanResult.Name, querySpanResult, nil
	}

	name := exprDisplayName(expr)
	sourceColumns, err := q.collectSourceColumnsFromExpr(expr)
	if err != nil {
		return "", base.QuerySpanResult{}, err
	}
	return name, base.QuerySpanResult{
		Name:          name,
		SourceColumns: sourceColumns,
		IsPlainField:  false,
	}, nil
}

// collectSourceColumnsFromExpr walks an arbitrary SELECT-list / predicate
// expression and unions the source columns of every column reference it
// contains. Column references are resolved against the current FROM scope;
// subqueries nested in the expression contribute their own source columns. This
// is the omni-AST replacement for the legacy extractQuerySpanResultResultFromExpr
// type-switch over the dozens of ANTLR expression contexts.
func (q *querySpanExtractor) collectSourceColumnsFromExpr(expr ast.Node) (base.SourceColumnSet, error) {
	result := make(base.SourceColumnSet)
	if expr == nil {
		return result, nil
	}

	var walkErr error
	ast.Inspect(expr, func(node ast.Node) bool {
		if node == nil || walkErr != nil {
			return false
		}
		switch n := node.(type) {
		case *ast.ColumnRef:
			normalizedDatabaseName, normalizedSchemaName, normalizedTableName, normalizedColumnName := normalizeColumnRef(n)
			querySpanResult, err := q.getField(normalizedDatabaseName, normalizedSchemaName, normalizedTableName, normalizedColumnName)
			if err != nil {
				walkErr = errors.Wrapf(err, "failed to check whether the column %q is sensitive", normalizedColumnName)
				return false
			}
			result, _ = base.MergeSourceColumnSet(result, querySpanResult.SourceColumns)
			return false
		case *ast.SubqueryExpr:
			sub, err := q.extractPseudoTableFromQueryNode(n.Query)
			if err != nil {
				walkErr = err
				return false
			}
			for _, field := range sub.GetQuerySpanResult() {
				result, _ = base.MergeSourceColumnSet(result, field.SourceColumns)
			}
			return false
		case *ast.ExistsExpr:
			sub, err := q.extractPseudoTableFromQueryNode(n.Query)
			if err != nil {
				walkErr = err
				return false
			}
			for _, field := range sub.GetQuerySpanResult() {
				result, _ = base.MergeSourceColumnSet(result, field.SourceColumns)
			}
			return false
		case *ast.StarExpr:
			// COUNT(*) and similar: a star with no resolvable column contributes no
			// specific source column (matching the legacy aggregate STAR branch).
			return false
		}
		return true
	})
	if walkErr != nil {
		return nil, walkErr
	}
	return result, nil
}

// exprDisplayName picks the output column name for an unaliased non-star target,
// matching the legacy behavior where the result name fell back to the
// expression's source text. For a bare column reference the trailing part is
// used; for anything else omni's source range is unavailable here, so the name
// is left empty (callers that have an AS alias override it anyway).
func exprDisplayName(expr ast.Node) string {
	switch e := expr.(type) {
	case *ast.ColumnRef:
		if len(e.Parts) > 0 {
			return normalizeSnowflakeIdentifier(e.Parts[len(e.Parts)-1])
		}
	case *ast.ParenExpr:
		return exprDisplayName(e.Expr)
	case *ast.CastExpr:
		return exprDisplayName(e.Expr)
	case *ast.CollateExpr:
		return exprDisplayName(e.Expr)
	}
	return ""
}

// normalizeColumnRef splits an omni ColumnRef (1-4 parts) into its normalized
// (database, schema, table, column) components. A 1-part ref is a bare column;
// 2-part is table.column; 3-part is schema.table.column; 4-part is
// database.schema.table.column.
func normalizeColumnRef(ref *ast.ColumnRef) (database, schema, table, column string) {
	if ref == nil {
		return "", "", "", ""
	}
	parts := ref.Parts
	switch len(parts) {
	case 1:
		return "", "", "", normalizeSnowflakeIdentifier(parts[0])
	case 2:
		return "", "", normalizeSnowflakeIdentifier(parts[0]), normalizeSnowflakeIdentifier(parts[1])
	case 3:
		return "", normalizeSnowflakeIdentifier(parts[0]), normalizeSnowflakeIdentifier(parts[1]), normalizeSnowflakeIdentifier(parts[2])
	case 4:
		return normalizeSnowflakeIdentifier(parts[0]), normalizeSnowflakeIdentifier(parts[1]), normalizeSnowflakeIdentifier(parts[2]), normalizeSnowflakeIdentifier(parts[3])
	default:
		if len(parts) == 0 {
			return "", "", "", ""
		}
		return "", "", "", normalizeSnowflakeIdentifier(parts[len(parts)-1])
	}
}

func filterExcludedColumns(columns []base.QuerySpanResult, exclude []ast.Ident) []base.QuerySpanResult {
	if len(exclude) == 0 {
		return columns
	}

	excludedColumns := make(map[string]bool, len(exclude))
	for _, columnName := range exclude {
		excludedColumns[normalizeSnowflakeIdentifier(columnName)] = true
	}

	var result []base.QuerySpanResult
	for _, column := range columns {
		if !excludedColumns[column.Name] {
			result = append(result, column)
		}
	}
	return result
}

// extractTableSourceFromFrom resolves a FROM clause (a list of *ast.TableRef and
// *ast.JoinExpr items) into a single combined table source exposing every column
// it provides. Multiple top-level FROM items are an implicit CROSS JOIN, so
// their columns are concatenated into one pseudo-table (matching the legacy
// extractTableSourceFromTableSources). A single FROM item is returned as-is so
// its alias/name is preserved for qualified column resolution; two or more items
// collapse into an unnamed pseudo-table.
func (q *querySpanExtractor) extractTableSourceFromFrom(froms []ast.Node) (base.TableSource, error) {
	var result base.TableSource
	for _, item := range froms {
		tableSource, err := q.extractTableSourceFromItem(item)
		if err != nil {
			return nil, err
		}
		if tableSource == nil {
			continue
		}
		if result == nil {
			result = tableSource
			continue
		}
		result = &base.PseudoTable{
			Name:    "",
			Columns: append(result.GetQuerySpanResult(), tableSource.GetQuerySpanResult()...),
		}
	}
	return result, nil
}

func (q *querySpanExtractor) extractTableSourceFromItem(item ast.Node) (base.TableSource, error) {
	switch n := item.(type) {
	case *ast.TableRef:
		return q.extractTableSourceFromTableRef(n)
	case *ast.JoinExpr:
		return q.extractTableSourceFromJoin(n)
	default:
		return nil, nil
	}
}

func (q *querySpanExtractor) extractTableSourceFromJoin(join *ast.JoinExpr) (base.TableSource, error) {
	if join == nil {
		return nil, nil
	}
	left, err := q.extractTableSourceFromItem(join.Left)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to extract sensitive fields of the left part of the join")
	}
	right, err := q.extractTableSourceFromItem(join.Right)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to extract sensitive fields of the right part of the join")
	}

	var leftColumns, rightColumns []base.QuerySpanResult
	if left != nil {
		leftColumns = left.GetQuerySpanResult()
	}
	if right != nil {
		rightColumns = right.GetQuerySpanResult()
	}

	// Snowflake has 6 join types: INNER, LEFT OUTER, RIGHT OUTER, FULL OUTER,
	// CROSS, and NATURAL. Only NATURAL JOIN collapses the duplicated join keys.
	if join.Natural {
		rightMap := make(map[string]bool)
		for _, rightColumn := range rightColumns {
			rightMap[rightColumn.Name] = true
		}
		var result []base.QuerySpanResult
		for _, leftColumn := range leftColumns {
			delete(rightMap, leftColumn.Name)
			result = append(result, leftColumn)
		}
		for _, rightColumn := range rightColumns {
			if rightMap[rightColumn.Name] {
				result = append(result, rightColumn)
			}
		}
		return &base.PseudoTable{
			Name:    "",
			Columns: result,
		}, nil
	}

	var result []base.QuerySpanResult
	result = append(result, leftColumns...)
	result = append(result, rightColumns...)
	return &base.PseudoTable{
		Name:    "",
		Columns: result,
	}, nil
}

func (q *querySpanExtractor) extractTableSourceFromTableRef(ref *ast.TableRef) (base.TableSource, error) {
	if ref == nil {
		return nil, nil
	}

	var result []base.QuerySpanResult

	switch {
	case ref.Name != nil:
		_, tableSource, err := q.findTableSchema(ref.Name, q.defaultDatbase, q.defaultSchema)
		if err != nil {
			return nil, err
		}
		result = append(result, tableSource.GetQuerySpanResult()...)
	case ref.Subquery != nil:
		tableSource, err := q.extractPseudoTableFromQueryNode(ref.Subquery)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to extract sensitive fields of subquery in FROM")
		}
		result = append(result, tableSource.GetQuerySpanResult()...)
	case ref.FuncCall != nil:
		// TODO(zp): In data-warehouse, defining a table function that returns
		// multiple rows is widespread; we should parse the function definition to
		// extract the sensitive fields. For now, a table function exposes no known
		// columns (mirroring the legacy TABLE()/flatten handling).
		result = nil
	default:
		result = nil
	}

	// If the table reference carries an alias, expose its columns under the alias
	// name so qualified column references (alias.column) resolve to it.
	if !ref.Alias.IsEmpty() {
		aliasName := normalizeSnowflakeIdentifier(ref.Alias)
		return &base.PseudoTable{
			Name:    aliasName,
			Columns: result,
		}, nil
	}

	return &base.PseudoTable{
		Name:    "",
		Columns: result,
	}, nil
}

func (q *querySpanExtractor) findTableSchema(objectName *ast.ObjectName, normalizedFallbackDatabaseName, normalizedFallbackSchemaName string) (string, base.TableSource, error) {
	normalizedDatabaseName, normalizedSchemaName, normalizedTableName := normalizeSnowflakeObjectName(objectName, "", "")
	// For snowflake, we should find the table schema in ctes by ascending order.
	if normalizedDatabaseName == "" && normalizedSchemaName == "" {
		for _, cte := range q.ctes {
			if normalizedTableName == cte.Name {
				return normalizedDatabaseName, cte, nil
			}
		}
	}
	normalizedDatabaseName, normalizedSchemaName, normalizedTableName = normalizeSnowflakeObjectName(objectName, normalizedFallbackDatabaseName, normalizedFallbackSchemaName)
	allDatabases, err := q.gCtx.ListDatabaseNamesFunc(q.ctx, q.gCtx.InstanceID)
	if err != nil {
		return "", nil, errors.Wrapf(err, "failed to get all databases")
	}

	for _, databaseName := range allDatabases {
		if normalizedDatabaseName != "" && normalizedDatabaseName != databaseName {
			continue
		}
		_, database, err := q.gCtx.GetDatabaseMetadataFunc(q.ctx, q.gCtx.InstanceID, normalizedDatabaseName)
		if err != nil {
			return "", nil, errors.Wrapf(err, "failed to get database %s", normalizedDatabaseName)
		}
		allSchemaNames := database.ListSchemaNames()
		for _, schemaSchema := range allSchemaNames {
			if normalizedSchemaName != "" && normalizedSchemaName != schemaSchema {
				continue
			}
			schema := database.GetSchemaMetadata(normalizedSchemaName)
			if schema == nil {
				return "", nil, errors.Errorf(`schema %s.%s is not found`, normalizedDatabaseName, normalizedSchemaName)
			}
			allTableNames := schema.ListTableNames()
			for _, tableName := range allTableNames {
				if normalizedTableName != tableName {
					continue
				}
				tableSchema := schema.GetTable(normalizedTableName)
				if tableSchema == nil {
					return "", nil, errors.Errorf(`table %s.%s.%s is not found`, normalizedDatabaseName, normalizedSchemaName, normalizedTableName)
				}
				columns := tableSchema.GetProto().GetColumns()
				return normalizedDatabaseName, &base.PhysicalTable{
					Name:     tableSchema.GetProto().Name,
					Database: normalizedDatabaseName,
					Schema:   normalizedSchemaName,
					Columns: func() []string {
						var result []string
						for _, column := range columns {
							result = append(result, column.Name)
						}
						return result
					}(),
				}, nil
			}
		}
	}
	return "", nil, errors.Errorf(`table %s.%s.%s is not found`, normalizedDatabaseName, normalizedSchemaName, normalizedTableName)
}

func (q *querySpanExtractor) getAllFieldsOfTableInFromOrOuterCTE(normalizedDatabaseName, normalizedSchemaName, normalizedTableName string) ([]base.QuerySpanResult, error) {
	type maskType = uint8
	const (
		maskNone         maskType = 0
		maskDatabaseName maskType = 1 << iota
		maskSchemaName
		maskTableName
	)
	mask := maskNone
	if normalizedTableName != "" {
		mask |= maskTableName
	}
	if normalizedSchemaName != "" {
		if mask&maskTableName == 0 {
			return nil, errors.Errorf(`table name %s is specified without column name`, normalizedTableName)
		}
		mask |= maskSchemaName
	}
	if normalizedDatabaseName != "" {
		if mask&maskSchemaName == 0 {
			return nil, errors.Errorf(`database name %s is specified without schema name`, normalizedDatabaseName)
		}
		mask |= maskDatabaseName
	}

	var result []base.QuerySpanResult

	if mask&maskDatabaseName == 0 && mask&maskSchemaName == 0 && mask&maskTableName != 0 {
		for _, cte := range q.ctes {
			if normalizedTableName == cte.Name {
				result = append(result, cte.GetQuerySpanResult()...)
				return result, nil
			}
		}
	}

	for _, tableSource := range q.tableSourcesFrom {
		if mask&maskDatabaseName != 0 && normalizedDatabaseName != tableSource.GetDatabaseName() {
			continue
		}
		if mask&maskSchemaName != 0 && normalizedSchemaName != tableSource.GetSchemaName() {
			continue
		}
		if mask&maskTableName != 0 && normalizedTableName != tableSource.GetTableName() {
			continue
		}
		result = append(result, tableSource.GetQuerySpanResult()...)
		return result, nil
	}

	return nil, errors.Errorf(`no matching table %q.%q.%q`, normalizedDatabaseName, normalizedSchemaName, normalizedTableName)
}

// getField iterates through the tableSourcesFrom sequentially until we find the first matching object and return the column name, and returns the fieldInfo.
func (q *querySpanExtractor) getField(normalizedDatabaseName, normalizedSchemaName, normalizedTableName, normalizedColumnName string) (base.QuerySpanResult, error) {
	type maskType = uint8
	const (
		maskNone         maskType = 0
		maskDatabaseName maskType = 1 << iota
		maskSchemaName
		maskTableName
		maskColumnName
	)
	mask := maskNone
	if normalizedColumnName != "" {
		mask |= maskColumnName
	}
	if normalizedTableName != "" {
		if mask&maskColumnName == 0 {
			return base.QuerySpanResult{}, errors.Errorf(`table name %s is specified without column name`, normalizedTableName)
		}
		mask |= maskTableName
	}
	if normalizedSchemaName != "" {
		if mask&maskTableName == 0 {
			return base.QuerySpanResult{}, errors.Errorf(`schema name %s is specified without table name`, normalizedSchemaName)
		}
		mask |= maskSchemaName
	}
	if normalizedDatabaseName != "" {
		if mask&maskSchemaName == 0 {
			return base.QuerySpanResult{}, errors.Errorf(`database name %s is specified without schema name`, normalizedDatabaseName)
		}
		mask |= maskDatabaseName
	}

	if mask == maskNone {
		return base.QuerySpanResult{}, errors.Errorf(`no object name is specified`)
	}

	// We just need to iterate through the tableSourcesFrom sequentially until we find the first matching object.

	// It is safe if there are two or more objects in the tableSourcesFrom have the same column name, because the executor
	// will throw a compilation error if the column name is ambiguous.
	// For example, there are two tables T1 and T2, and both of them have a column named "C1". The following query will throw
	// a compilation error:
	// SELECT C1 FROM T1, T2;
	//
	// But users can specify the table name to avoid the compilation error:
	// SELECT T1.C1 FROM T1, T2;
	//
	// Further more, users can not use the original table name if they specify the alias name:
	// SELECT T1.C1 FROM T1 AS T3, T2; -- invalid identifier 'ADDRESS.ID'

	for _, tableSource := range q.tableSourcesFrom {
		if mask&maskDatabaseName != 0 && normalizedDatabaseName != tableSource.GetDatabaseName() {
			continue
		}
		if mask&maskSchemaName != 0 && normalizedSchemaName != tableSource.GetSchemaName() {
			continue
		}
		if mask&maskTableName != 0 && normalizedTableName != tableSource.GetTableName() {
			continue
		}
		for _, field := range tableSource.GetQuerySpanResult() {
			if mask&maskColumnName != 0 && normalizedColumnName != field.Name {
				continue
			}
			return field, nil
		}
	}
	return base.QuerySpanResult{}, errors.Errorf(`no matching column %q.%q.%q.%q`, normalizedDatabaseName, normalizedSchemaName, normalizedTableName, normalizedColumnName)
}

// getAccessTables extracts the list of resources from the SELECT statement, and
// normalizes the object names with the NON-EMPTY currentNormalizedDatabase and
// currentNormalizedSchema. It collects every *ast.TableRef with a table name —
// including CTE references and subquery tables — mirroring the legacy
// accessTablesListener, which added every object_ref it visited.
//
// It uses a hand-written recursion rather than ast.Inspect because omni's
// generated AST walker does NOT descend into a SelectStmt's WITH clause, SELECT
// targets, GROUP BY, ORDER BY, or FETCH; those sub-trees can carry table
// references (CTE bodies, scalar subqueries) that the legacy listener — which
// walked the full ANTLR parse tree — would have recorded.
func getAccessTables(currentNormalizedDatabase, currentNormalizedSchema string, node ast.Node) base.SourceColumnSet {
	resourceMap := make(base.SourceColumnSet)
	collectAccessTables(node, currentNormalizedDatabase, currentNormalizedSchema, resourceMap)
	return resourceMap
}

// collectAccessTables recursively records every table reference reachable from
// node into resourceMap. It explicitly descends into the SelectStmt sub-parts
// that omni's generated walker skips; for everything else it falls back to
// ast.Inspect, which reaches the remaining FROM/WHERE/JOIN/subquery table refs.
func collectAccessTables(node ast.Node, defaultDatabase, defaultSchema string, resourceMap base.SourceColumnSet) {
	if node == nil {
		return
	}
	switch n := node.(type) {
	case *ast.File:
		for _, stmt := range n.Stmts {
			collectAccessTables(stmt, defaultDatabase, defaultSchema, resourceMap)
		}
		return
	case *ast.SetOperationStmt:
		collectAccessTables(n.Left, defaultDatabase, defaultSchema, resourceMap)
		collectAccessTables(n.Right, defaultDatabase, defaultSchema, resourceMap)
		return
	case *ast.SelectStmt:
		for _, cte := range n.With {
			collectAccessTables(cte.Query, defaultDatabase, defaultSchema, resourceMap)
		}
		for _, item := range n.From {
			collectAccessTables(item, defaultDatabase, defaultSchema, resourceMap)
		}
		for _, target := range n.Targets {
			if target != nil {
				collectAccessTables(target.Expr, defaultDatabase, defaultSchema, resourceMap)
			}
		}
		collectAccessTables(n.Where, defaultDatabase, defaultSchema, resourceMap)
		if n.GroupBy != nil {
			for _, item := range n.GroupBy.Items {
				collectAccessTables(item, defaultDatabase, defaultSchema, resourceMap)
			}
		}
		collectAccessTables(n.Having, defaultDatabase, defaultSchema, resourceMap)
		collectAccessTables(n.Qualify, defaultDatabase, defaultSchema, resourceMap)
		for _, item := range n.OrderBy {
			if item != nil {
				collectAccessTables(item.Expr, defaultDatabase, defaultSchema, resourceMap)
			}
		}
		return
	case *ast.JoinExpr:
		collectAccessTables(n.Left, defaultDatabase, defaultSchema, resourceMap)
		collectAccessTables(n.Right, defaultDatabase, defaultSchema, resourceMap)
		collectAccessTables(n.On, defaultDatabase, defaultSchema, resourceMap)
		return
	case *ast.TableRef:
		if n.Name != nil {
			database, schema, table := normalizeSnowflakeObjectName(n.Name, defaultDatabase, defaultSchema)
			resourceMap[base.ColumnResource{
				Database: database,
				Schema:   schema,
				Table:    table,
			}] = true
		}
		if n.Subquery != nil {
			collectAccessTables(n.Subquery, defaultDatabase, defaultSchema, resourceMap)
		}
		if n.FuncCall != nil {
			collectAccessTables(n.FuncCall, defaultDatabase, defaultSchema, resourceMap)
		}
		return
	case *ast.SubqueryExpr:
		collectAccessTables(n.Query, defaultDatabase, defaultSchema, resourceMap)
		return
	case *ast.ExistsExpr:
		collectAccessTables(n.Query, defaultDatabase, defaultSchema, resourceMap)
		return
	}

	// For any other expression node, walk its children with ast.Inspect (which
	// fully traverses non-SelectStmt sub-trees) and recurse into the
	// query-bearing nodes it surfaces. This catches table refs nested inside
	// arbitrary expressions (e.g. a scalar subquery inside a CASE in WHERE).
	ast.Inspect(node, func(child ast.Node) bool {
		if child == nil || child == node {
			return true
		}
		switch child.(type) {
		case *ast.SelectStmt, *ast.SetOperationStmt, *ast.TableRef, *ast.JoinExpr, *ast.SubqueryExpr, *ast.ExistsExpr:
			collectAccessTables(child, defaultDatabase, defaultSchema, resourceMap)
			return false
		}
		return true
	})
}

// isMixedQuery checks whether the query accesses the user table and system table at the same time.
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

func isSystemResource(base.ColumnResource, bool) bool {
	// TODO(zp): fix me.
	return false
}
