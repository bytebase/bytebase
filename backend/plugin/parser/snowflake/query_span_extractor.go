package snowflake

import (
	"context"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"

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

	// statementText is the single statement being extracted; expression result
	// names slice it by node Loc (the legacy extractor used ctx.GetText()).
	statementText string

	// joinMemberSources records the individual relations inside JOIN trees of
	// the current FROM scope. tableSourcesFrom holds one (unnamed, merged)
	// pseudo-table per FROM item, so qualified references like T2.B or T2.*
	// inside a join could not resolve by table name; lookups fall back to this
	// list. Scoped/truncated together with tableSourcesFrom.
	joinMemberSources []base.TableSource
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
	q.statementText = statement

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

	// USE and SET are session commands that getQueryType classifies as
	// base.Select (legacy queryTypeListener parity), but they are not query
	// expressions and produce no result columns. The legacy selectOnlyListener
	// simply never fired for them, so the legacy extractor returned a span with
	// empty results and no error; mirror that here instead of falling into
	// extractPseudoTableFromQueryNode's "unsupported query node type" error.
	switch node.(type) {
	case *ast.UseStmt, *ast.SetStmt:
		return &base.QuerySpan{
			Type:          queryType,
			SourceColumns: accessTables,
			Results:       []base.QuerySpanResult{},
		}, nil
	default:
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
		// A WITH clause preceding a set operation is attached by omni to the
		// LEFTMOST SelectStmt, but its CTEs are visible to EVERY operand
		// (legacy handled WITH at the query_statement level). Hoist them over
		// the whole set-op extraction; the leftmost SelectStmt pushes them
		// again during its own extraction, which is harmless shadowing.
		if leftmost := leftmostSelect(n); leftmost != nil && len(leftmost.With) > 0 {
			originalCTECount := len(q.ctes)
			if err := q.extractCTEFromWith(leftmost.With, isRecursiveWith(leftmost.With)); err != nil {
				return nil, err
			}
			defer func() {
				q.ctes = q.ctes[:originalCTECount]
			}()
		}
		return q.extractPseudoTableFromSetOperation(n)
	case *ast.ResultScanStmt:
		// FAIL CLOSED: a result-pipe (->>) query reads the previous statement's
		// result set, which has no resolvable schema here; resolving the trailing
		// SELECT against $1 would produce wrong lineage. Same posture as PIVOT.
		return nil, errors.New("result-pipe (->>) statements are not supported for query span extraction yet")
	case *ast.ShowStmt:
		// Only reachable for SHOW ... ->> <query> (a plain SHOW classifies
		// SelectInfoSchema and returns before extraction). Same fail-closed
		// posture as ResultScanStmt: $1's schema is not resolvable.
		return nil, errors.New("result-pipe (->>) statements are not supported for query span extraction yet")
	default:
		return nil, errors.Errorf("unsupported query node type %T", node)
	}
}

// leftmostSelect descends a set-operation chain's left spine to the SelectStmt
// that textually leads the statement (where omni attaches a leading WITH).
func leftmostSelect(node ast.Node) *ast.SelectStmt {
	switch n := node.(type) {
	case *ast.SelectStmt:
		return n
	case *ast.SetOperationStmt:
		return leftmostSelect(n.Left)
	default:
		return nil
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
		// UNION [ALL] BY NAME merges columns by name, which the recursive
		// fixed-point merge below does not model (it merges positionally and
		// would attribute A's lineage to B for swapped columns). Route BY NAME
		// set-op bodies through the plain extraction path, which handles BY
		// NAME correctly; a genuinely self-referencing BY NAME CTE then fails
		// resolution explicitly instead of producing wrong lineage. (Legacy
		// ANTLR could not parse BY NAME at all, so there is no legacy behavior
		// to preserve here.)
		if isRecursiveCTE && setOpChainHasByName(cte.Query) {
			isRecursiveCTE = false
		}
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

// setOpChainHasByName reports whether any set operation in a (possibly nested)
// set-operation chain uses the BY NAME column-matching mode.
func setOpChainHasByName(node ast.Node) bool {
	setOp, ok := node.(*ast.SetOperationStmt)
	if !ok {
		return false
	}
	if setOp.ByName {
		return true
	}
	return setOpChainHasByName(setOp.Left) || setOpChainHasByName(setOp.Right)
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
// The left branch's column names are kept; each right branch is merged into the
// left's columns — positionally for plain set operators (the executor requires
// matching column counts), or by column name for UNION [ALL] BY NAME.
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
	if err := mergeSetOperatorColumns(left, right, setOp.ByName); err != nil {
		return nil, err
	}
	return left, nil
}

func mergeSetOperatorColumns(left, right *base.PseudoTable, byName bool) error {
	if left == nil || right == nil {
		return nil
	}
	leftColumns := left.GetQuerySpanResult()
	rightColumns := right.GetQuerySpanResult()

	if byName {
		// UNION [ALL] BY NAME (Snowflake-specific) aligns the branches by
		// case-insensitive column name instead of by position, and the column
		// counts may differ (columns missing on one side are NULL-filled by the
		// engine). Mirror omni's own snowflake/analysis by-name merge: keep the
		// left columns in order, merging in the lineage of the same-named right
		// column, then append the right-only columns.
		rightByName := make(map[string]base.QuerySpanResult, len(rightColumns))
		for _, rightColumn := range rightColumns {
			rightByName[strings.ToUpper(rightColumn.Name)] = rightColumn
		}
		seen := make(map[string]bool, len(leftColumns))
		merged := make([]base.QuerySpanResult, 0, len(leftColumns))
		for _, leftColumn := range leftColumns {
			key := strings.ToUpper(leftColumn.Name)
			seen[key] = true
			if rightColumn, ok := rightByName[key]; ok {
				leftColumn.SourceColumns, _ = base.MergeSourceColumnSet(leftColumn.SourceColumns, rightColumn.SourceColumns)
			}
			merged = append(merged, leftColumn)
		}
		for _, rightColumn := range rightColumns {
			if !seen[strings.ToUpper(rightColumn.Name)] {
				merged = append(merged, rightColumn)
			}
		}
		left.Columns = merged
		return nil
	}

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
		// Snapshot BEFORE extracting the FROM scope: extractTableSourceFromFrom
		// appends JOIN member relations to q.joinMemberSources as it walks join
		// trees, so a later snapshot would bake those into the restore point and
		// leak them into sibling/outer scopes.
		originalFromFieldsLength := len(q.tableSourcesFrom)
		originalJoinMemberLength := len(q.joinMemberSources)
		defer func() {
			q.tableSourcesFrom = q.tableSourcesFrom[:originalFromFieldsLength]
			q.joinMemberSources = q.joinMemberSources[:originalJoinMemberLength]
		}()
		tableSourceFrom, err := q.extractTableSourceFromFrom(ctx.From)
		if err != nil {
			return nil, err
		}
		if tableSourceFrom != nil {
			q.tableSourcesFrom = append(q.tableSourcesFrom, tableSourceFrom)
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
			left = filterIlikeColumns(left, target.Ilike)
			left = filterExcludedColumns(left, target.Exclude)
			left, err = q.applyStarReplaces(left, target.Replace)
			if err != nil {
				return nil, err
			}
			left = applyStarRenames(left, target.Rename)
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

	// Positional column reference $N (optionally qualified, d.$1): the legacy
	// extractor resolved it to the N-th column of the FROM scope (or of the
	// qualified relation), erroring on out-of-range positions. Session
	// variables ($var, Positional=false) fall through to the generic path like
	// any other opaque expression.
	if dollar, ok := expr.(*ast.DollarRef); ok && dollar.Positional {
		querySpanResult, err := q.resolvePositionalRef(dollar)
		if err != nil {
			return "", base.QuerySpanResult{}, err
		}
		return querySpanResult.Name, querySpanResult, nil
	}

	name := q.exprDisplayName(expr)
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
	// walk is the ast.Inspect callback. It is a named closure so the manual
	// recursions below — for sub-trees that omni's GENERATED walker does not
	// descend into because they are non-Node embedded structs — can re-enter
	// the same traversal. Collecting into a set keeps the manual recursion
	// idempotent: it stays correct (merely redundant) if the upstream walker
	// later learns to visit these children itself.
	var walk func(node ast.Node) bool
	walk = func(node ast.Node) bool {
		if node == nil || walkErr != nil {
			return false
		}
		switch n := node.(type) {
		case *ast.DollarRef:
			// Positional $N inside a compound expression (e.g. TO_DATE($1),
			// $1 + 1) contributes the N-th FROM-scope column's lineage. Session
			// variables ($var) carry no column lineage.
			if !n.Positional {
				return true
			}
			querySpanResult, err := q.resolvePositionalRef(n)
			if err != nil {
				walkErr = err
				return false
			}
			result, _ = base.MergeSourceColumnSet(result, querySpanResult.SourceColumns)
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
		case *ast.CaseExpr:
			// omni's generated walker only visits CaseExpr.Operand and
			// CaseExpr.Else; the WHEN branches are []*WhenClause — a non-Node
			// embedded struct — so without this manual recursion
			// `CASE WHEN A > 0 THEN B END` would contribute no lineage at all.
			for _, when := range n.Whens {
				if when == nil {
					continue
				}
				ast.Inspect(when.Cond, walk)
				ast.Inspect(when.Result, walk)
			}
			// Continue the normal traversal for Operand and Else.
			return true
		case *ast.FuncCallExpr:
			// omni's generated walker only visits FuncCallExpr.Args; the
			// WITHIN GROUP (ORDER BY ...) items and the OVER (...) window
			// specification are non-Node embedded structs, so
			// `ROW_NUMBER() OVER (ORDER BY A)` would otherwise contribute no
			// lineage at all.
			for _, item := range n.OrderBy {
				if item != nil {
					ast.Inspect(item.Expr, walk)
				}
			}
			if n.Over != nil {
				for _, partition := range n.Over.PartitionBy {
					ast.Inspect(partition, walk)
				}
				for _, item := range n.Over.OrderBy {
					if item != nil {
						ast.Inspect(item.Expr, walk)
					}
				}
				if n.Over.Frame != nil {
					ast.Inspect(n.Over.Frame.Start.Offset, walk)
					ast.Inspect(n.Over.Frame.End.Offset, walk)
				}
			}
			// Continue the normal traversal for Args.
			return true
		default:
		}
		return true
	}
	ast.Inspect(expr, walk)
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
func (q *querySpanExtractor) exprDisplayName(expr ast.Node) string {
	switch e := expr.(type) {
	case *ast.ColumnRef:
		if len(e.Parts) > 0 {
			return normalizeSnowflakeIdentifier(e.Parts[len(e.Parts)-1])
		}
	case *ast.ParenExpr:
		return q.exprDisplayName(e.Expr)
	case *ast.CastExpr:
		return q.exprDisplayName(e.Expr)
	case *ast.CollateExpr:
		return q.exprDisplayName(e.Expr)
	default:
	}
	// Fall back to the expression's verbatim source text (the legacy extractor
	// used ctx.GetText()): omni nodes carry byte Locs into the statement.
	if loc := ast.NodeLoc(expr); loc.End > loc.Start && loc.Start >= 0 && loc.End <= len(q.statementText) {
		if name := strings.TrimSpace(q.statementText[loc.Start:loc.End]); utf8.ValidString(name) {
			return name
		}
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
	merged := false
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
		// Comma-list FROM items merge into one unnamed scope for positional/star
		// purposes; record each named item in joinMemberSources so qualified
		// references (alias.col, table.col) keep resolving — the same fallback
		// JOIN trees use.
		if !merged {
			q.joinMemberSources = append(q.joinMemberSources, result)
			merged = true
		}
		q.joinMemberSources = append(q.joinMemberSources, tableSource)
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
	if left != nil {
		q.joinMemberSources = append(q.joinMemberSources, left)
	}
	right, err := q.extractTableSourceFromItem(join.Right)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to extract sensitive fields of the right part of the join")
	}
	if right != nil {
		q.joinMemberSources = append(q.joinMemberSources, right)
	}

	var leftColumns, rightColumns []base.QuerySpanResult
	if left != nil {
		leftColumns = left.GetQuerySpanResult()
	}
	if right != nil {
		rightColumns = right.GetQuerySpanResult()
	}

	// Snowflake has 6 join types: INNER, LEFT OUTER, RIGHT OUTER, FULL OUTER,
	// CROSS, and NATURAL. NATURAL JOIN and JOIN ... USING collapse the shared
	// join keys to a single output column; getting the collapse wrong is worse
	// than wrong lineage, because a duplicated shared column shifts every later
	// positional mask by one.

	if join.Natural {
		// NATURAL JOIN: each shared column appears once, carrying the lineage of
		// BOTH sides (the legacy extractor collapsed but kept only the left
		// side's lineage). Left columns keep their order, then the right-only
		// columns follow.
		rightIndexByName := make(map[string]int, len(rightColumns))
		for i, rightColumn := range rightColumns {
			if _, ok := rightIndexByName[rightColumn.Name]; !ok {
				rightIndexByName[rightColumn.Name] = i
			}
		}
		sharedNames := make(map[string]bool)
		var result []base.QuerySpanResult
		for _, leftColumn := range leftColumns {
			if i, ok := rightIndexByName[leftColumn.Name]; ok {
				sharedNames[leftColumn.Name] = true
				leftColumn.SourceColumns, _ = base.MergeSourceColumnSet(leftColumn.SourceColumns, rightColumns[i].SourceColumns)
			}
			result = append(result, leftColumn)
		}
		for _, rightColumn := range rightColumns {
			if !sharedNames[rightColumn.Name] {
				result = append(result, rightColumn)
			}
		}
		return &base.PseudoTable{
			Name:    "",
			Columns: result,
		}, nil
	}

	if len(join.Using) > 0 {
		// JOIN ... USING (cols): per
		// https://docs.snowflake.com/en/sql-reference/constructs/join the output
		// contains the USING columns first (in the order specified, each ONCE,
		// carrying the merged lineage of both sides), then the left table
		// columns not in USING, then the right table columns not in USING.
		// The legacy extractor had no USING handling at all and duplicated the
		// shared columns.
		usingNames := make([]string, 0, len(join.Using))
		usingSet := make(map[string]bool, len(join.Using))
		for _, col := range join.Using {
			name := normalizeSnowflakeIdentifier(col)
			if usingSet[name] {
				// Defensive: a duplicated USING column is invalid SQL; keep one.
				continue
			}
			usingSet[name] = true
			usingNames = append(usingNames, name)
		}

		var result []base.QuerySpanResult
		for _, name := range usingNames {
			var merged *base.QuerySpanResult
			for _, leftColumn := range leftColumns {
				if leftColumn.Name == name {
					columnCopy := leftColumn
					merged = &columnCopy
					break
				}
			}
			for _, rightColumn := range rightColumns {
				if rightColumn.Name == name {
					if merged == nil {
						columnCopy := rightColumn
						merged = &columnCopy
					} else {
						merged.SourceColumns, _ = base.MergeSourceColumnSet(merged.SourceColumns, rightColumn.SourceColumns)
					}
					break
				}
			}
			if merged == nil {
				// The USING column resolves on neither side (invalid SQL or an
				// unresolvable source); emit an empty-lineage column to keep the
				// positional column count stable.
				merged = &base.QuerySpanResult{
					Name:          name,
					SourceColumns: make(base.SourceColumnSet),
					IsPlainField:  false,
				}
			}
			result = append(result, *merged)
		}
		for _, leftColumn := range leftColumns {
			if !usingSet[leftColumn.Name] {
				result = append(result, leftColumn)
			}
		}
		for _, rightColumn := range rightColumns {
			if !usingSet[rightColumn.Name] {
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

	// FAIL CLOSED on PIVOT/UNPIVOT: the pivoted relation's projection (dynamic
	// pivot columns) is not modeled here yet, so resolving the bare base table
	// would silently produce wrong lineage/positions for masking. Returning an
	// error makes GetQuerySpan degrade explicitly instead. (omni models the
	// clause on TableRef since #300; lineage support is a follow-up.)
	if ref.Pivot != nil || ref.Unpivot != nil || ref.Nested != nil {
		return nil, errors.New("PIVOT/UNPIVOT table sources are not supported for query span extraction yet")
	}

	var result []base.QuerySpanResult
	// underlying keeps the resolved (named) table source so an unaliased
	// reference retains its identity — qualified references (T2.B, T2.*) must
	// resolve by the original table name.
	var underlying base.TableSource

	switch {
	case ref.Name != nil:
		_, tableSource, err := q.findTableSchema(ref.Name, q.defaultDatbase, q.defaultSchema)
		if err != nil {
			return nil, err
		}
		underlying = tableSource
		result = append(result, tableSource.GetQuerySpanResult()...)
	case ref.Subquery != nil:
		tableSource, err := q.extractPseudoTableFromQueryNode(ref.Subquery)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to extract sensitive fields of subquery in FROM")
		}
		result = append(result, tableSource.GetQuerySpanResult()...)
	case ref.FuncCall != nil:
		// NOTE(zp): In data-warehouse, defining a table function that returns
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

	// No alias: keep the resolved table source (with its real name) so
	// qualified lookups by the original table name still match.
	if underlying != nil {
		return underlying, nil
	}

	return &base.PseudoTable{
		Name:    "",
		Columns: result,
	}, nil
}

func (q *querySpanExtractor) findTableSchema(objectName *ast.ObjectName, normalizedFallbackDatabaseName, normalizedFallbackSchemaName string) (string, base.TableSource, error) {
	normalizedDatabaseName, normalizedSchemaName, normalizedTableName := normalizeSnowflakeObjectName(objectName, "", "")
	// Scan CTEs newest-first so an inner scope's CTE shadows an outer one with
	// the same name (q.ctes is a stack of scopes; duplicate names within one
	// WITH list are a Snowflake error, so this only affects cross-scope shadowing).
	if normalizedDatabaseName == "" && normalizedSchemaName == "" {
		for i := len(q.ctes) - 1; i >= 0; i-- {
			cte := q.ctes[i]
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
		// Newest-first: inner-scope CTEs shadow outer ones (see findTableSchema).
		for i := len(q.ctes) - 1; i >= 0; i-- {
			cte := q.ctes[i]
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

	// Qualified reference that did not match a FROM item: it may name a
	// relation inside a JOIN tree (tableSourcesFrom holds only the merged,
	// unnamed join pseudo-table).
	if mask&maskTableName != 0 {
		for _, tableSource := range q.joinMemberSources {
			if mask&maskDatabaseName != 0 && normalizedDatabaseName != tableSource.GetDatabaseName() {
				continue
			}
			if mask&maskSchemaName != 0 && normalizedSchemaName != tableSource.GetSchemaName() {
				continue
			}
			if normalizedTableName != tableSource.GetTableName() {
				continue
			}
			result = append(result, tableSource.GetQuerySpanResult()...)
			return result, nil
		}
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
	// Qualified reference into a JOIN tree: tableSourcesFrom holds only the
	// merged, unnamed join pseudo-table, so match the member relations.
	if mask&maskTableName != 0 {
		for _, tableSource := range q.joinMemberSources {
			if mask&maskDatabaseName != 0 && normalizedDatabaseName != tableSource.GetDatabaseName() {
				continue
			}
			if mask&maskSchemaName != 0 && normalizedSchemaName != tableSource.GetSchemaName() {
				continue
			}
			if normalizedTableName != tableSource.GetTableName() {
				continue
			}
			for _, field := range tableSource.GetQuerySpanResult() {
				if mask&maskColumnName != 0 && normalizedColumnName != field.Name {
					continue
				}
				return field, nil
			}
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
// It uses a hand-written recursion as belt-and-suspenders: it predates the
// omni walker-coverage fix, when the generated walker skipped a SelectStmt's
// WITH clause, SELECT targets, GROUP BY, ORDER BY, and FETCH. The walker now
// descends into all of those (the SQL-review advisors rely on plain
// ast.Inspect reaching CTE bodies and SELECT-list subqueries, locked by yaml
// cases), so this recursion is no longer load-bearing for coverage; it is kept
// because it is correct, tested, and independent of walker regressions.
func getAccessTables(currentNormalizedDatabase, currentNormalizedSchema string, node ast.Node) base.SourceColumnSet {
	resourceMap := make(base.SourceColumnSet)
	collectAccessTables(node, currentNormalizedDatabase, currentNormalizedSchema, resourceMap)
	return resourceMap
}

// collectAccessTables recursively records every table reference reachable from
// node into resourceMap. It explicitly descends into the SelectStmt sub-parts
// (WITH/targets/GROUP BY/ORDER BY/FETCH; redundant with the fixed omni walker,
// see getAccessTables); for everything else it falls back to ast.Inspect.
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

// resolvePositionalRef resolves a positional column reference $N (optionally
// qualified, d.$1) to the N-th column of the FROM scope (or of the qualified
// relation), mirroring the legacy DOLLAR/Column_position branch: errors on a
// non-numeric, < 1, or out-of-range position.
func (q *querySpanExtractor) resolvePositionalRef(dollar *ast.DollarRef) (base.QuerySpanResult, error) {
	position, err := strconv.Atoi(dollar.Name)
	if err != nil {
		return base.QuerySpanResult{}, errors.Wrapf(err, "failed to parse column position %q to integer", dollar.Name)
	}
	if position < 1 {
		return base.QuerySpanResult{}, errors.Errorf("column position %d is invalid because it is less than 1", position)
	}
	var normalizedDatabaseName, normalizedSchemaName, normalizedTableName string
	if dollar.Qualifier != nil {
		normalizedDatabaseName, normalizedSchemaName, normalizedTableName = normalizeSnowflakeObjectName(dollar.Qualifier, "", "")
	}
	left, err := q.getAllFieldsOfTableInFromOrOuterCTE(normalizedDatabaseName, normalizedSchemaName, normalizedTableName)
	if err != nil {
		return base.QuerySpanResult{}, errors.Wrapf(err, "failed to resolve positional column reference $%s", dollar.Name)
	}
	if position > len(left) {
		return base.QuerySpanResult{}, errors.Errorf("column position $%d is invalid: the FROM clause only returns %d columns", position, len(left))
	}
	return left[position-1], nil
}

// applyStarRenames applies the `SELECT * RENAME (col AS alias, ...)` transform
// to a star expansion: the matching result columns keep their lineage but take
// the alias as their output name (Snowflake returns the renamed column).
// Matching uses normalized identifiers, like the rest of the resolver.
func applyStarRenames(columns []base.QuerySpanResult, renames []ast.StarRename) []base.QuerySpanResult {
	if len(renames) == 0 {
		return columns
	}
	aliasByColumn := make(map[string]string, len(renames))
	for _, r := range renames {
		aliasByColumn[normalizeSnowflakeIdentifier(r.Col)] = normalizeSnowflakeIdentifier(r.Alias)
	}
	// Copy before renaming: columns may be the FROM-scope table source's
	// internal slice (getAllFieldsOfTableInFromOrOuterCTE returns it directly,
	// and base.PseudoTable documents that callers must copy before modifying);
	// mutating it in place would corrupt the scope for later references.
	renamed := make([]base.QuerySpanResult, len(columns))
	copy(renamed, columns)
	for i := range renamed {
		if alias, ok := aliasByColumn[renamed[i].Name]; ok {
			renamed[i].Name = alias
		}
	}
	return renamed
}

// filterIlikeColumns applies the `SELECT * ILIKE '<pattern>'` transform: only
// columns whose name matches the pattern (case-insensitive; % = any run,
// _ = any single character) survive the star expansion. Applied FIRST, before
// EXCLUDE/REPLACE/RENAME, per the documented transform order. The positional
// masker depends on the span matching the actual result-set shape, so this
// must genuinely filter (not over-attribute).
func filterIlikeColumns(columns []base.QuerySpanResult, pattern *ast.Literal) []base.QuerySpanResult {
	if pattern == nil {
		return columns
	}
	matcher := ilikeMatcher(pattern.Value)
	filtered := make([]base.QuerySpanResult, 0, len(columns))
	for _, column := range columns {
		if matcher(column.Name) {
			filtered = append(filtered, column)
		}
	}
	return filtered
}

// ilikeMatcher compiles a Snowflake ILIKE pattern into a case-insensitive
// matcher over the whole name (% -> any run, _ -> any single rune). Go regexp
// (?i) covers ASCII case-insensitivity; it does not replicate Snowflake SQL
// collation for non-ASCII identifiers (low risk: column names are near-always
// ASCII).
func ilikeMatcher(pattern string) func(string) bool {
	var sb strings.Builder
	sb.WriteString(`(?is)\A`)
	for _, r := range pattern {
		switch r {
		case '%':
			sb.WriteString(`.*`)
		case '_':
			sb.WriteString(`.`)
		default:
			sb.WriteString(regexp.QuoteMeta(string(r)))
		}
	}
	sb.WriteString(`\z`)
	re, err := regexp.Compile(sb.String())
	if err != nil {
		// A pattern that fails to compile matches nothing (fail closed).
		return func(string) bool { return false }
	}
	return re.MatchString
}

// applyStarReplaces applies the `SELECT * REPLACE (expr AS col, ...)` transform
// to a star expansion: the matching result column keeps its position and name
// but its lineage becomes the replacement EXPRESSION's source columns
// (Snowflake returns expr in place of the original column value). Copy-on-write
// like applyStarRenames — the input may be a table source's internal slice.
func (q *querySpanExtractor) applyStarReplaces(columns []base.QuerySpanResult, replaces []ast.StarReplace) ([]base.QuerySpanResult, error) {
	if len(replaces) == 0 {
		return columns, nil
	}
	exprByColumn := make(map[string]ast.Node, len(replaces))
	for _, r := range replaces {
		exprByColumn[normalizeSnowflakeIdentifier(r.Col)] = r.Expr
	}
	replaced := make([]base.QuerySpanResult, len(columns))
	copy(replaced, columns)
	for i := range replaced {
		expr, ok := exprByColumn[replaced[i].Name]
		if !ok {
			continue
		}
		sourceColumns, err := q.collectSourceColumnsFromExpr(expr)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to resolve star REPLACE expression for column %q", replaced[i].Name)
		}
		replaced[i].SourceColumns = sourceColumns
		replaced[i].IsPlainField = false
	}
	return replaced, nil
}
