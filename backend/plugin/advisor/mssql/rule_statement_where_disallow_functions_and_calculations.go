package mssql

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/bytebase/omni/mssql/ast"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	"github.com/bytebase/bytebase/backend/store/model"
)

func init() {
	advisor.Register(storepb.Engine_MSSQL, storepb.SQLReviewRule_STATEMENT_WHERE_DISALLOW_FUNCTIONS_AND_CALCULATIONS, &DisallowFuncAndCalculationsAdvisor{})
}

type DisallowFuncAndCalculationsAdvisor struct{}

var _ advisor.Advisor = (*DisallowFuncAndCalculationsAdvisor)(nil)

func (*DisallowFuncAndCalculationsAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	rule := &disallowFuncAndCalcOmniRule{
		OmniBaseRule: OmniBaseRule{Level: level, Title: checkCtx.Rule.Type.String()},
	}
	if checkCtx.DBSchema != nil {
		rule.dbMetadata = model.NewDatabaseMetadata(checkCtx.DBSchema, nil, nil, storepb.Engine_MSSQL, checkCtx.IsObjectCaseSensitive)
	}

	return RunOmniRules(checkCtx.ParsedStatements, []OmniRule{rule}), nil
}

type disallowFuncAndCalcOmniRule struct {
	OmniBaseRule
	dbMetadata *model.DatabaseMetadata
}

func (*disallowFuncAndCalcOmniRule) Name() string {
	return "DisallowFuncAndCalcOmniRule"
}

func (r *disallowFuncAndCalcOmniRule) OnStatement(node ast.Node) {
	switch n := node.(type) {
	case *ast.SelectStmt:
		r.checkSelect(n)
	case *ast.InsertStmt:
		r.onInsert(n)
	case *ast.CreateTableAsSelectStmt:
		r.checkSelectNode(n.Query)
	case *ast.CreateViewStmt:
		r.checkSelectNode(n.Query)
	case *ast.CreateMaterializedViewStmt:
		r.checkSelectNode(n.Query)
	case *ast.CreateExternalTableAsSelectStmt:
		r.checkSelectNode(n.Query)
	case *ast.CreateRemoteTableAsSelectStmt:
		r.checkSelectNode(n.Query)
	case *ast.UpdateStmt:
		r.onUpdate(n)
	case *ast.DeleteStmt:
		r.onDelete(n)
	case *ast.MergeStmt:
		r.checkCTEs(n.WithClause)
		r.checkMerge(n)
	default:
	}
}

// checkSelectNode runs checkSelect if node is a *ast.SelectStmt; no-ops otherwise.
func (r *disallowFuncAndCalcOmniRule) checkSelectNode(node ast.Node) {
	if sel, ok := node.(*ast.SelectStmt); ok {
		r.checkSelect(sel)
	}
}

// onInsert handles T-SQL INSERT. Source can be a SELECT or a VALUES list; both
// may embed scalar subqueries with their own WHERE clauses.
func (r *disallowFuncAndCalcOmniRule) onInsert(n *ast.InsertStmt) {
	r.checkCTEs(n.WithClause)
	r.recurseIntoNestedSelectsInNode(n.Source, nil)
}

// onUpdate handles T-SQL UPDATE. It covers the WHERE, derived tables in
// Relation/FromClause (each with its own WHERE), and scalar subqueries hidden
// in SET assignments.
func (r *disallowFuncAndCalcOmniRule) onUpdate(n *ast.UpdateStmt) {
	r.checkCTEs(n.WithClause)
	tables := r.collectUpdateDeleteTables(n.Relation, n.FromClause)
	r.checkWhere(n.WhereClause, tables)
	r.recurseDerivedTables(n.Relation, n.FromClause, tables)
	r.recurseSetClauseValues(n.SetClause, tables)
}

// onDelete handles T-SQL DELETE, including any JOINed sources in the FROM clause
// that carry their own WHERE clauses inside derived tables.
func (r *disallowFuncAndCalcOmniRule) onDelete(n *ast.DeleteStmt) {
	r.checkCTEs(n.WithClause)
	tables := r.collectUpdateDeleteTables(n.Relation, n.FromClause)
	r.checkWhere(n.WhereClause, tables)
	r.recurseDerivedTables(n.Relation, n.FromClause, tables)
}

// collectUpdateDeleteTables merges tables referenced by the target Relation
// with any JOINed tables in the extra FROM clause (T-SQL UPDATE/DELETE shape).
func (r *disallowFuncAndCalcOmniRule) collectUpdateDeleteTables(relation ast.TableExpr, from *ast.List) tablesByAlias {
	tables := r.collectTablesFromRelation(relation)
	for alias, ref := range r.collectTableNamesFromList(from) {
		tables[alias] = ref
	}
	return tables
}

// recurseDerivedTables walks the relation and every FROM item for nested SELECTs
// whose own WHERE clause must be checked in its own scope.
func (r *disallowFuncAndCalcOmniRule) recurseDerivedTables(relation ast.TableExpr, from *ast.List, outer tablesByAlias) {
	r.recurseIntoNestedSelectsInNode(relation, outer)
	if from == nil {
		return
	}
	for _, item := range from.Items {
		r.recurseIntoNestedSelectsInNode(item, outer)
	}
}

// recurseSetClauseValues walks each SET assignment's value for scalar subqueries.
func (r *disallowFuncAndCalcOmniRule) recurseSetClauseValues(setClause *ast.List, outer tablesByAlias) {
	if setClause == nil {
		return
	}
	for _, item := range setClause.Items {
		if se, ok := item.(*ast.SetExpr); ok && se.Value != nil {
			r.recurseIntoNestedSelectsInNode(se.Value, outer)
		}
	}
}

// checkMerge handles T-SQL MERGE statements. The ON condition and each WHEN-clause
// condition are treated like a WHERE for index-aware checking, scoped to the target
// plus source tables. Nested SELECTs inside the source or conditions are recursed.
func (r *disallowFuncAndCalcOmniRule) checkMerge(n *ast.MergeStmt) {
	tables := make(tablesByAlias)
	if ref, ok := n.Target.(*ast.TableRef); ok {
		addTableRef(r, tables, ref)
	}
	// Extract source tables (can be TableRef, SubqueryExpr, JoinClause, etc.).
	if n.Source != nil {
		if ref, ok := n.Source.(*ast.TableRef); ok {
			// Bare table source: honor SourceAlias if set, otherwise use the ref's own alias/object.
			if n.SourceAlias != "" {
				tables[r.normalizeIdent(n.SourceAlias)] = tableRef{database: ref.Database, schema: ref.Schema, object: ref.Object}
			} else {
				addTableRef(r, tables, ref)
			}
		} else {
			// Non-TableRef source (subquery, join, etc.). Recurse into the inner SELECT
			// in its own scope. When SourceAlias is set, register it as an opaque entry
			// so qualified refs like `src.col` resolve to no indexed columns (safely
			// returning false) rather than falling through to cross-table matches.
			if n.SourceAlias != "" {
				tables[r.normalizeIdent(n.SourceAlias)] = tableRef{}
			}
			ast.Inspect(n.Source, func(m ast.Node) bool {
				if sub, ok := m.(*ast.SelectStmt); ok {
					r.checkSelect(sub)
					return false
				}
				if _, ok := m.(*ast.SubqueryExpr); ok {
					return true
				}
				// When there's a SourceAlias, don't leak inner TableRefs into outer scope.
				if _, ok := m.(*ast.TableRef); ok {
					if n.SourceAlias == "" {
						addTableRef(r, tables, m.(*ast.TableRef))
					}
					return false
				}
				return true
			})
		}
	}
	r.checkWhere(n.OnCondition, tables)
	if n.WhenClauses != nil {
		for _, item := range n.WhenClauses.Items {
			when, ok := item.(*ast.MergeWhenClause)
			if !ok {
				continue
			}
			r.checkWhere(when.Condition, tables)
			// MERGE actions can embed scalar subqueries with their own WHERE clauses,
			// e.g. WHEN MATCHED THEN UPDATE SET x = (SELECT ... WHERE ABS(col) > 0).
			switch a := when.Action.(type) {
			case *ast.MergeUpdateAction:
				if a != nil {
					r.recurseSetClauseValues(a.SetClause, tables)
				}
			case *ast.MergeInsertAction:
				if a != nil && a.Values != nil {
					for _, v := range a.Values.Items {
						r.recurseIntoNestedSelectsInNode(v, tables)
					}
				}
			default:
			}
		}
	}
}

func (r *disallowFuncAndCalcOmniRule) checkCTEs(wc *ast.WithClause) {
	if wc == nil || wc.CTEs == nil {
		return
	}
	for _, item := range wc.CTEs.Items {
		if cte, ok := item.(*ast.CommonTableExpr); ok {
			if sel, ok := cte.Query.(*ast.SelectStmt); ok {
				r.checkSelect(sel)
			}
		}
	}
}

func (r *disallowFuncAndCalcOmniRule) checkSelect(sel *ast.SelectStmt) {
	r.checkSelectInScope(sel, nil)
}

// checkSelectInScope is like checkSelect but also threads an outer table scope down
// so that correlated subqueries can resolve references to outer aliases. Local
// (inner) tables take precedence for unqualified columns per SQL name resolution;
// outer tables are consulted only for qualified references.
func (r *disallowFuncAndCalcOmniRule) checkSelectInScope(sel *ast.SelectStmt, outer tablesByAlias) {
	if sel == nil {
		return
	}
	// CTEs run before the referencing SELECT and aren't correlated — check without outer scope.
	r.checkCTEs(sel.WithClause)
	if sel.Op != ast.SetOpNone {
		r.checkSelectInScope(sel.Larg, outer)
		r.checkSelectInScope(sel.Rarg, outer)
		return
	}
	local := r.collectTableNamesFromList(sel.FromClause)
	combined := make(tablesByAlias, len(local)+len(outer))
	for alias, ref := range outer {
		combined[alias] = ref
	}
	for alias, ref := range local {
		combined[alias] = ref
	}
	r.checkWhereWithScopes(sel.WhereClause, local, outer, r.localHasUnresolvedSources(sel.FromClause, local))
	r.recurseIntoNestedSelects(sel, combined)
}

// localHasUnresolvedSources reports whether the given FROM list and local alias
// map contain sources whose columns we cannot enumerate — CTE references,
// derived tables (subqueries in FROM), or plain tables absent from metadata.
// When true, isIndexed must not fall back to outer scope on unqualified refs.
func (r *disallowFuncAndCalcOmniRule) localHasUnresolvedSources(from *ast.List, local tablesByAlias) bool {
	if from != nil {
		for _, item := range from.Items {
			derivedFound := false
			ast.Inspect(item, func(n ast.Node) bool {
				switch n.(type) {
				case *ast.SubqueryExpr, *ast.SelectStmt:
					derivedFound = true
					return false
				}
				return !derivedFound
			})
			if derivedFound {
				return true
			}
		}
	}
	for _, ref := range local {
		if ref == (tableRef{}) {
			continue
		}
		if r.tableMetadata(ref) == nil {
			return true
		}
	}
	return false
}

// recurseIntoNestedSelectsInNode walks the given AST node and recurses into any
// nested *ast.SelectStmt, threading the given outer scope.
func (r *disallowFuncAndCalcOmniRule) recurseIntoNestedSelectsInNode(n ast.Node, outer tablesByAlias) {
	if n == nil {
		return
	}
	ast.Inspect(n, func(m ast.Node) bool {
		if sub, ok := m.(*ast.SelectStmt); ok {
			r.checkSelectInScope(sub, outer)
			return false
		}
		return true
	})
}

// recurseIntoNestedSelects finds nested SelectStmts in the non-WHERE parts of sel
// (FROM derived tables, SELECT list, HAVING, GROUP BY, ORDER BY) and invokes
// checkSelectInScope on them with the given outer scope so correlated subqueries resolve.
func (r *disallowFuncAndCalcOmniRule) recurseIntoNestedSelects(sel *ast.SelectStmt, outer tablesByAlias) {
	visitor := func(m ast.Node) bool {
		if sub, ok := m.(*ast.SelectStmt); ok {
			r.checkSelectInScope(sub, outer)
			return false
		}
		return true
	}
	visitList := func(l *ast.List) {
		if l == nil {
			return
		}
		for _, item := range l.Items {
			if item != nil {
				ast.Inspect(item, visitor)
			}
		}
	}
	// WhereClause is handled by checkWhere; nested selects there are handled by
	// checkWhere's own ast.Inspect. No need to revisit it here.
	visitList(sel.FromClause)
	visitList(sel.TargetList)
	if sel.HavingClause != nil {
		ast.Inspect(sel.HavingClause, visitor)
	}
	visitList(sel.GroupByClause)
	visitList(sel.OrderByClause)
}

// tableRef captures the real database, schema, and object name for an aliased table reference.
type tableRef struct {
	database string // empty means current database
	schema   string // empty means default (dbo)
	object   string
}

// tablesByAlias maps alias (or object name if no alias) to the underlying tableRef.
type tablesByAlias = map[string]tableRef

// collectTableNamesFromList extracts table refs from a FROM clause List, preserving schema.
// Descent is stopped at subquery boundaries so that derived tables don't leak their
// inner TableRefs into the outer scope, but a derived table's own alias (captured
// via AliasedTableRef wrapping a SubqueryExpr) is recorded as an opaque entry so
// qualifier-based resolution can tell it apart from an outer alias of the same name.
func (r *disallowFuncAndCalcOmniRule) collectTableNamesFromList(from *ast.List) tablesByAlias {
	tables := make(tablesByAlias)
	if from == nil {
		return tables
	}
	for _, item := range from.Items {
		ast.Inspect(item, func(n ast.Node) bool {
			// AliasedTableRef wraps a real TableRef or a SubqueryExpr with an alias.
			// For subquery sources, record the alias before we stop descent so that
			// inner qualified refs resolve locally rather than to an outer table.
			if atr, ok := n.(*ast.AliasedTableRef); ok {
				if _, isSub := atr.Table.(*ast.SubqueryExpr); isSub && atr.Alias != "" {
					key := r.normalizeIdent(atr.Alias)
					if _, exists := tables[key]; !exists {
						tables[key] = tableRef{}
					}
					return false
				}
				return true
			}
			if _, ok := n.(*ast.SubqueryExpr); ok {
				return false
			}
			if _, ok := n.(*ast.SelectStmt); ok {
				return false
			}
			t, ok := n.(*ast.TableRef)
			if !ok {
				return true
			}
			addTableRef(r, tables, t)
			return false
		})
	}
	return tables
}

// collectTablesFromRelation builds a table map from a single table reference (UPDATE/DELETE).
func (r *disallowFuncAndCalcOmniRule) collectTablesFromRelation(te ast.TableExpr) tablesByAlias {
	tables := make(tablesByAlias)
	if te == nil {
		return tables
	}
	if ref, ok := te.(*ast.TableRef); ok {
		addTableRef(r, tables, ref)
	}
	return tables
}

// normalizeIdent lowercases identifiers when the DB is configured case-insensitive
// (respecting IsObjectCaseSensitive). MSSQL default collation is CI, but a CS DB
// must be honored too.
func (r *disallowFuncAndCalcOmniRule) normalizeIdent(name string) string {
	if r.dbMetadata != nil && r.dbMetadata.GetIsObjectCaseSensitive() {
		return name
	}
	return strings.ToLower(name)
}

func addTableRef(r *disallowFuncAndCalcOmniRule, tables tablesByAlias, t *ast.TableRef) {
	key := t.Alias
	if key == "" {
		key = t.Object
	}
	normKey := r.normalizeIdent(key)
	newRef := tableRef{database: t.Database, schema: t.Schema, object: t.Object}
	// If an unaliased same-name from a different schema/database already occupies this
	// key (rare: `FROM dbo.orders, sales.orders` without aliases), mark the entry as
	// ambiguous so isIndexed returns false rather than silently resolving to the wrong
	// table. Users should alias these tables to disambiguate.
	if existing, ok := tables[normKey]; ok && (existing.database != newRef.database || existing.schema != newRef.schema || existing.object != newRef.object) {
		tables[normKey] = tableRef{}
		return
	}
	tables[normKey] = newRef
}

// tableMetadata fetches the store-model table for the given ref, applying the
// cross-DB guard and MSSQL's dbo-first-then-sorted-others schema resolution.
// Returns nil when not resolvable.
func (r *disallowFuncAndCalcOmniRule) tableMetadata(ref tableRef) *model.TableMetadata {
	if r.dbMetadata == nil {
		return nil
	}
	if ref.database != "" && !strings.EqualFold(ref.database, r.dbMetadata.DatabaseName()) {
		return nil
	}
	if ref.schema != "" {
		if s := r.dbMetadata.GetSchemaMetadata(ref.schema); s != nil {
			return s.GetTable(ref.object)
		}
		return nil
	}
	if s := r.dbMetadata.GetSchemaMetadata("dbo"); s != nil {
		if t := s.GetTable(ref.object); t != nil {
			return t
		}
	}
	// Sort schemas for deterministic fallback order (ListSchemaNames is map-backed).
	names := r.dbMetadata.ListSchemaNames()
	slices.Sort(names)
	for _, name := range names {
		if strings.EqualFold(name, "dbo") {
			continue
		}
		if s := r.dbMetadata.GetSchemaMetadata(name); s != nil {
			if t := s.GetTable(ref.object); t != nil {
				return t
			}
		}
	}
	return nil
}

// indexedColumns returns indexed column names for the table at the given database+schema+object.
func (r *disallowFuncAndCalcOmniRule) indexedColumns(ref tableRef) map[string]bool {
	table := r.tableMetadata(ref)
	if table == nil {
		return nil
	}
	cols := make(map[string]bool)
	for _, idx := range table.GetProto().GetIndexes() {
		for _, expr := range idx.GetExpressions() {
			cols[strings.ToLower(expr)] = true
		}
	}
	if len(cols) == 0 {
		return nil
	}
	return cols
}

// allColumns returns every column name defined for the given table reference.
func (r *disallowFuncAndCalcOmniRule) allColumns(ref tableRef) map[string]bool {
	table := r.tableMetadata(ref)
	if table == nil {
		return nil
	}
	cols := make(map[string]bool)
	for _, c := range table.GetProto().GetColumns() {
		cols[strings.ToLower(c.GetName())] = true
	}
	if len(cols) == 0 {
		return nil
	}
	return cols
}

// indexedByAlias maps alias (or object if no alias) to the set of indexed columns on that table.
type indexedByAlias = map[string]map[string]bool

// scopeIndexed carries per-alias column metadata for the local (inner) and outer scopes.
// Unqualified refs resolve against local first (any table that declares the column);
// only when no local table declares the column — and the local scope is fully
// resolvable — do we fall back to outer. If any local source has unknown columns
// (CTEs, derived tables, tables without metadata), we stay silent to avoid
// misattributing an inner column to an outer indexed column of the same name.
type scopeIndexed struct {
	local           indexedByAlias
	localAll        indexedByAlias
	localAliases    map[string]bool
	outer           indexedByAlias
	localUnresolved bool
}

func (s scopeIndexed) empty() bool {
	return len(s.local) == 0 && len(s.outer) == 0
}

// resolveIndexedColumns builds a per-alias map of indexed columns.
func (r *disallowFuncAndCalcOmniRule) resolveIndexedColumns(tables tablesByAlias) indexedByAlias {
	all := make(indexedByAlias)
	for alias, ref := range tables {
		if cols := r.indexedColumns(ref); cols != nil {
			all[alias] = cols
		}
	}
	if len(all) == 0 {
		return nil
	}
	return all
}

// resolveAllColumns builds a per-alias map of every known column for each table.
func (r *disallowFuncAndCalcOmniRule) resolveAllColumns(tables tablesByAlias) indexedByAlias {
	all := make(indexedByAlias)
	for alias, ref := range tables {
		if cols := r.allColumns(ref); cols != nil {
			all[alias] = cols
		}
	}
	if len(all) == 0 {
		return nil
	}
	return all
}

// isIndexed reports whether a column reference is indexed, respecting its table qualifier
// and SQL name resolution for correlated subqueries.
//   - Qualified (ref.Table set): check local first, then outer.
//   - Unqualified: if any local table declares the column, local wins. Only if no
//     local table declares the column do we fall back to outer indexed columns.
func (r *disallowFuncAndCalcOmniRule) isIndexed(ref *ast.ColumnRef, s scopeIndexed) bool {
	if ref == nil || s.empty() {
		return false
	}
	col := strings.ToLower(ref.Column)
	if ref.Table != "" {
		alias := r.normalizeIdent(ref.Table)
		// Local scope takes precedence: if the alias exists locally (even as a
		// derived table or CTE without index metadata), the ref resolves there.
		// Otherwise the qualified alias is definitively outer, even when other
		// local sources have unresolved metadata — collectTableNamesFromList
		// records every local alias (including derived tables) so this check
		// is sound.
		if s.localAliases[alias] {
			return s.local[alias][col]
		}
		return s.outer[alias][col]
	}
	localClaims := false
	for _, cols := range s.localAll {
		if cols[col] {
			localClaims = true
			break
		}
	}
	if localClaims {
		for _, cols := range s.local {
			if cols[col] {
				return true
			}
		}
		return false
	}
	// No local alias declared the column. Only fall back to outer when the local
	// scope is fully resolvable; otherwise a CTE/derived-table column could be
	// misattributed to an outer indexed column of the same name.
	if s.localUnresolved {
		return false
	}
	for _, cols := range s.outer {
		if cols[col] {
			return true
		}
	}
	return false
}

func (r *disallowFuncAndCalcOmniRule) checkWhere(where ast.ExprNode, tables tablesByAlias) {
	r.checkWhereWithScopes(where, tables, nil, false)
}

func (r *disallowFuncAndCalcOmniRule) checkWhereWithScopes(where ast.ExprNode, local, outer tablesByAlias, localUnresolved bool) {
	if where == nil {
		return
	}
	localAliases := make(map[string]bool, len(local))
	for alias := range local {
		localAliases[alias] = true
	}
	s := scopeIndexed{
		local:           r.resolveIndexedColumns(local),
		localAll:        r.resolveAllColumns(local),
		localAliases:    localAliases,
		outer:           r.resolveIndexedColumns(outer),
		localUnresolved: localUnresolved,
	}
	combined := make(tablesByAlias, len(local)+len(outer))
	for alias, ref := range outer {
		combined[alias] = ref
	}
	for alias, ref := range local {
		combined[alias] = ref
	}
	ast.Inspect(where, func(n ast.Node) bool {
		if sub, ok := n.(*ast.SelectStmt); ok {
			r.checkSelectInScope(sub, combined)
			return false
		}
		if !s.empty() {
			r.checkWhereNode(n, s)
		}
		return true
	})
}

func (r *disallowFuncAndCalcOmniRule) checkWhereNode(n ast.Node, indexed scopeIndexed) {
	switch expr := n.(type) {
	case *ast.FuncCallExpr:
		if col := r.findIndexedColumnInFuncArgs(expr.Args, indexed); col != "" {
			funcName := ""
			if expr.Name != nil {
				funcName = strings.ToUpper(tableRefText(expr.Name))
			}
			r.addFunctionAdvice(funcName, col, expr.Loc)
		}
	case *ast.BinaryExpr:
		if isArithmeticOp(expr.Op) {
			if col := r.unwrapAndFindIndexedColumn(expr.Left, indexed); col != "" {
				r.addCalculationAdvice(col, expr.Loc)
			} else if col := r.unwrapAndFindIndexedColumn(expr.Right, indexed); col != "" {
				r.addCalculationAdvice(col, expr.Loc)
			}
		}
	case *ast.UnaryExpr:
		// UnaryPlus is a no-op and does not prevent index usage; skip it.
		if expr.Op == ast.UnaryMinus || expr.Op == ast.UnaryBitNot {
			if col := r.unwrapAndFindIndexedColumn(expr.Operand, indexed); col != "" {
				r.addCalculationAdvice(col, expr.Loc)
			}
		}
	default:
	}
}

func (r *disallowFuncAndCalcOmniRule) addFunctionAdvice(funcName, col string, loc ast.Loc) {
	r.AddAdvice(&storepb.Advice{
		Status:        r.Level,
		Code:          code.StatementDisallowFunctionsAndCalculations.Int32(),
		Title:         r.Title,
		Content:       fmt.Sprintf("Function %q is applied to indexed column %q in the WHERE clause, which prevents index usage", funcName, col),
		StartPosition: &storepb.Position{Line: r.LocToLine(loc)},
	})
}

func (r *disallowFuncAndCalcOmniRule) addCalculationAdvice(col string, loc ast.Loc) {
	r.AddAdvice(&storepb.Advice{
		Status:        r.Level,
		Code:          code.StatementDisallowFunctionsAndCalculations.Int32(),
		Title:         r.Title,
		Content:       fmt.Sprintf("Calculation is applied to indexed column %q in the WHERE clause, which prevents index usage", col),
		StartPosition: &storepb.Position{Line: r.LocToLine(loc)},
	})
}

// findIndexedColumnInFuncArgs searches function arguments for an indexed ColumnRef,
// descending through passthrough wrappers (ParenExpr, CastExpr, ConvertExpr) so that
// `ABS((c1))` and `ABS(CAST(c1 AS int))` are still flagged.
// BinaryExpr/UnaryExpr (arithmetic) are NOT crossed — those are handled by the
// calculation advice path, so `ABS(c1 + 1)` is reported once as a calculation.
func (r *disallowFuncAndCalcOmniRule) findIndexedColumnInFuncArgs(args *ast.List, indexed scopeIndexed) string {
	if args == nil {
		return ""
	}
	for _, item := range args.Items {
		arg, ok := item.(ast.ExprNode)
		if !ok {
			continue
		}
		if col := r.unwrapAndFindIndexedColumn(arg, indexed); col != "" {
			return col
		}
	}
	return ""
}

// unwrapAndFindIndexedColumn peels ParenExpr/CastExpr/ConvertExpr wrappers off expr
// and returns the column name if the inner expression is an indexed ColumnRef.
func (r *disallowFuncAndCalcOmniRule) unwrapAndFindIndexedColumn(expr ast.ExprNode, indexed scopeIndexed) string {
	for expr != nil {
		switch e := expr.(type) {
		case *ast.ColumnRef:
			if r.isIndexed(e, indexed) {
				return e.Column
			}
			return ""
		case *ast.ParenExpr:
			expr = e.Expr
		case *ast.CastExpr:
			expr = e.Expr
		case *ast.ConvertExpr:
			expr = e.Expr
		default:
			return ""
		}
	}
	return ""
}

func isArithmeticOp(op ast.BinaryOp) bool {
	switch op {
	case ast.BinOpAdd, ast.BinOpSub, ast.BinOpMul, ast.BinOpDiv, ast.BinOpMod,
		ast.BinOpBitAnd, ast.BinOpBitOr, ast.BinOpBitXor:
		return true
	default:
		return false
	}
}
