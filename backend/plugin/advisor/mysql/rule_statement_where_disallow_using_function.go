package mysql

import (
	"context"
	"fmt"
	"strings"

	"github.com/bytebase/omni/mysql/ast"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	"github.com/bytebase/bytebase/backend/store/model"
)

var (
	_ advisor.Advisor = (*StatementWhereDisallowUsingFunctionAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MYSQL, storepb.SQLReviewRule_STATEMENT_WHERE_DISALLOW_FUNCTIONS_AND_CALCULATIONS, &StatementWhereDisallowUsingFunctionAdvisor{})
}

type StatementWhereDisallowUsingFunctionAdvisor struct {
}

func (*StatementWhereDisallowUsingFunctionAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	rule := &whereDisallowFuncOmniRule{
		OmniBaseRule: OmniBaseRule{
			Level: level,
			Title: checkCtx.Rule.Type.String(),
		},
	}
	if checkCtx.DBSchema != nil {
		rule.dbMetadata = model.NewDatabaseMetadata(checkCtx.DBSchema, nil, nil, storepb.Engine_MYSQL, checkCtx.IsObjectCaseSensitive)
	}

	return RunOmniRules(checkCtx.ParsedStatements, []OmniRule{rule}), nil
}

type whereDisallowFuncOmniRule struct {
	OmniBaseRule
	dbMetadata *model.DatabaseMetadata
}

func (*whereDisallowFuncOmniRule) Name() string {
	return "StatementWhereDisallowUsingFunctionRule"
}

func (r *whereDisallowFuncOmniRule) OnStatement(node ast.Node) {
	switch n := node.(type) {
	case *ast.SelectStmt:
		r.checkSelect(n)
	case *ast.InsertStmt:
		r.onInsert(n)
	case *ast.CreateTableStmt:
		r.onCreateTable(n)
	case *ast.CreateViewStmt:
		r.onCreateView(n)
	case *ast.AlterViewStmt:
		r.onAlterView(n)
	case *ast.UpdateStmt:
		r.onUpdate(n)
	case *ast.DeleteStmt:
		r.onDelete(n)
	default:
	}
}

// onInsert handles INSERT [... SELECT | ... VALUES (...) | ... SET ... | ... ON DUPLICATE KEY UPDATE].
// Each of SELECT/VALUES/SET/ON DUPLICATE KEY can embed scalar subqueries with their own WHERE.
func (r *whereDisallowFuncOmniRule) onInsert(n *ast.InsertStmt) {
	if n.Select != nil {
		r.checkSelect(n.Select)
	}
	for _, row := range n.Values {
		for _, v := range row {
			r.recurseIntoNestedSelectsInNode(v, nil)
		}
	}
	r.recurseAssignmentValues(n.SetList, nil)
	r.recurseAssignmentValues(n.OnDuplicateKey, nil)
}

func (r *whereDisallowFuncOmniRule) onCreateTable(n *ast.CreateTableStmt) {
	if n.Select != nil {
		r.checkSelect(n.Select)
	}
}

func (r *whereDisallowFuncOmniRule) onCreateView(n *ast.CreateViewStmt) {
	if n.Select != nil {
		r.checkSelect(n.Select)
	}
}

func (r *whereDisallowFuncOmniRule) onAlterView(n *ast.AlterViewStmt) {
	if n.Select != nil {
		r.checkSelect(n.Select)
	}
}

// onUpdate handles UPDATE WHERE and any scalar subqueries hidden in SET expressions
// or derived tables in the table list (e.g. JOIN (SELECT ... WHERE ...) s).
func (r *whereDisallowFuncOmniRule) onUpdate(n *ast.UpdateStmt) {
	tables := r.collectTableNames(n.Tables)
	r.checkWhere(n.Where, tables)
	for _, te := range n.Tables {
		r.recurseIntoNestedSelectsInNode(te, tables)
	}
	r.recurseAssignmentValues(n.SetList, tables)
}

// onDelete handles DELETE WHERE, including multi-table DELETE ... USING and derived
// tables that carry their own WHERE clauses.
func (r *whereDisallowFuncOmniRule) onDelete(n *ast.DeleteStmt) {
	tables := r.collectTableNames(n.Tables)
	for alias, ref := range r.collectTableNames(n.Using) {
		tables[alias] = ref
	}
	r.checkWhere(n.Where, tables)
	for _, te := range n.Tables {
		r.recurseIntoNestedSelectsInNode(te, tables)
	}
	for _, te := range n.Using {
		r.recurseIntoNestedSelectsInNode(te, tables)
	}
}

// recurseAssignmentValues walks the RHS of each assignment (UPDATE SET, INSERT SET,
// INSERT ... ON DUPLICATE KEY UPDATE) for scalar subqueries.
func (r *whereDisallowFuncOmniRule) recurseAssignmentValues(assignments []*ast.Assignment, outer tablesByAlias) {
	for _, a := range assignments {
		if a != nil && a.Value != nil {
			r.recurseIntoNestedSelectsInNode(a.Value, outer)
		}
	}
}

func (r *whereDisallowFuncOmniRule) checkSelect(sel *ast.SelectStmt) {
	r.checkSelectInScope(sel, nil)
}

// checkSelectInScope is like checkSelect but also threads an outer table scope down
// so that correlated subqueries can resolve references to outer aliases. Local
// (inner) tables take precedence for unqualified columns per SQL name resolution;
// outer tables are consulted only for qualified references (e.g. outer_alias.col).
func (r *whereDisallowFuncOmniRule) checkSelectInScope(sel *ast.SelectStmt, outer tablesByAlias) {
	if sel == nil {
		return
	}
	// CTEs are not correlated with their referencing SELECT, so check them without outer scope.
	for _, cte := range sel.CTEs {
		if cte.Select != nil {
			r.checkSelect(cte.Select)
		}
	}
	if sel.SetOp != ast.SetOpNone {
		r.checkSelectInScope(sel.Left, outer)
		r.checkSelectInScope(sel.Right, outer)
		return
	}
	local := r.collectTableNames(sel.From)
	// Build a combined map for downstream recursion into nested subqueries, where
	// local aliases are "outer" relative to that deeper scope.
	combined := make(tablesByAlias, len(local)+len(outer))
	for alias, ref := range outer {
		combined[alias] = ref
	}
	for alias, ref := range local {
		combined[alias] = ref
	}
	r.checkWhereWithScopes(sel.Where, local, outer, r.localHasUnresolvedSources(sel.From, local))
	// Recurse into nested SelectStmts elsewhere in this SELECT, passing current combined
	// scope as outer so correlated references in scalar/derived subqueries resolve.
	r.recurseIntoNestedSelects(sel, combined)
}

// localHasUnresolvedSources reports whether the given FROM and local alias map
// contain sources whose columns we cannot enumerate from metadata — CTE refs,
// derived tables (subqueries in FROM), and plain tables absent from metadata.
// When true, we must not fall back to outer scope for unqualified columns,
// because the inner source likely owns the name.
func (r *whereDisallowFuncOmniRule) localHasUnresolvedSources(from []ast.TableExpr, local tablesByAlias) bool {
	for _, te := range from {
		derivedFound := false
		ast.Inspect(te, func(n ast.Node) bool {
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
func (r *whereDisallowFuncOmniRule) recurseIntoNestedSelectsInNode(n ast.Node, outer tablesByAlias) {
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

// recurseIntoNestedSelects finds nested *ast.SelectStmt nodes in the non-WHERE parts of
// the given SELECT and recursively invokes checkSelectInScope on them with outer scope.
func (r *whereDisallowFuncOmniRule) recurseIntoNestedSelects(sel *ast.SelectStmt, outer tablesByAlias) {
	visit := func(n ast.Node) {
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
	for _, from := range sel.From {
		visit(from)
	}
	for _, t := range sel.TargetList {
		visit(t)
	}
	visit(sel.Having)
	for _, g := range sel.GroupBy {
		visit(g)
	}
	for _, o := range sel.OrderBy {
		if o != nil {
			visit(o.Expr)
		}
	}
}

// tableRef captures the real database and table name for an aliased table reference.
// In MySQL terminology, TableRef.Schema is the database name.
type tableRef struct {
	database string // empty means current database
	name     string
}

// tablesByAlias maps alias (or name if no alias) to the underlying tableRef.
type tablesByAlias = map[string]tableRef

// collectTableNames extracts table refs (with database qualifier) from FROM/table expressions.
// Descent is stopped at subquery boundaries so that derived tables don't leak their
// inner TableRefs into the outer scope, but the derived table's own alias is captured
// as an opaque tableRef{} entry so qualifier-based resolution can still tell it apart
// from an outer alias of the same name.
// Alias keys are normalized per IsObjectCaseSensitive.
func (r *whereDisallowFuncOmniRule) collectTableNames(tableExprs []ast.TableExpr) tablesByAlias {
	tables := make(tablesByAlias)
	for _, from := range tableExprs {
		ast.Inspect(from, func(n ast.Node) bool {
			// Stop at subquery boundaries — inner SELECTs are a separate scope — but
			// record the derived table's alias so qualifier lookups recognize it as local.
			if sq, ok := n.(*ast.SubqueryExpr); ok {
				if sq.Alias != "" {
					if _, exists := tables[r.normalizeIdent(sq.Alias)]; !exists {
						tables[r.normalizeIdent(sq.Alias)] = tableRef{}
					}
				}
				return false
			}
			if _, ok := n.(*ast.SelectStmt); ok {
				return false
			}
			t, ok := n.(*ast.TableRef)
			if !ok {
				return true
			}
			key := t.Alias
			if key == "" {
				key = t.Name
			}
			normKey := r.normalizeIdent(key)
			newRef := tableRef{database: t.Schema, name: t.Name}
			// If an unaliased same-name from a different database already occupies this
			// key (rare: `FROM db1.orders, db2.orders` without aliases), mark as ambiguous
			// so isIndexed returns false rather than resolving to the wrong table.
			// Users should alias these tables to disambiguate.
			if existing, ok := tables[normKey]; ok && (existing.database != newRef.database || existing.name != newRef.name) {
				tables[normKey] = tableRef{}
			} else {
				tables[normKey] = newRef
			}
			return false
		})
	}
	return tables
}

// normalizeIdent lowercases identifiers when the DB is configured case-insensitive.
func (r *whereDisallowFuncOmniRule) normalizeIdent(name string) string {
	if r.dbMetadata != nil && r.dbMetadata.GetIsObjectCaseSensitive() {
		return name
	}
	return strings.ToLower(name)
}

// tableMetadata fetches the store-model table for the given ref, applying the
// cross-DB guard. Returns nil when not resolvable.
func (r *whereDisallowFuncOmniRule) tableMetadata(ref tableRef) *model.TableMetadata {
	if r.dbMetadata == nil {
		return nil
	}
	// Skip cross-database references; we only have metadata for the current DB.
	if ref.database != "" && !strings.EqualFold(ref.database, r.dbMetadata.DatabaseName()) {
		return nil
	}
	return r.dbMetadata.GetSchemaMetadata("").GetTable(ref.name)
}

// indexedColumns returns indexed column names for the given table reference.
func (r *whereDisallowFuncOmniRule) indexedColumns(ref tableRef) map[string]bool {
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
func (r *whereDisallowFuncOmniRule) allColumns(ref tableRef) map[string]bool {
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

// indexedByAlias maps alias (or table name if no alias) to the set of indexed columns on that table.
type indexedByAlias = map[string]map[string]bool

// scopeIndexed carries per-alias column metadata for the local (inner) scope
// and any outer scope inherited from enclosing queries. Unqualified refs first
// resolve against local (any table that declares the column); only when no
// local table declares the column — and the full local scope is resolvable —
// do we fall back to outer. If any local source has unknown columns (CTEs,
// derived tables, tables without metadata), we do NOT fall back: inner scope
// still owns the name, we just can't prove indexability, so we stay silent
// rather than misattribute to an outer indexed column.
type scopeIndexed struct {
	local           indexedByAlias  // per-alias indexed columns (local scope)
	localAll        indexedByAlias  // per-alias full column sets (local scope)
	localAliases    map[string]bool // every alias present in the local scope (including unresolved/derived)
	outer           indexedByAlias  // per-alias indexed columns (outer scope)
	localUnresolved bool            // true when the local scope has sources whose columns we can't enumerate
}

func (s scopeIndexed) empty() bool {
	return len(s.local) == 0 && len(s.outer) == 0
}

// resolveIndexedColumns builds a per-alias map of indexed columns.
func (r *whereDisallowFuncOmniRule) resolveIndexedColumns(tables tablesByAlias) indexedByAlias {
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
func (r *whereDisallowFuncOmniRule) resolveAllColumns(tables tablesByAlias) indexedByAlias {
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
//   - Unqualified: if any local table declares the column, local wins (flag only if
//     that local table has it indexed). If no local table declares the column, fall
//     back to outer indexed columns (supports correlated subqueries).
func (r *whereDisallowFuncOmniRule) isIndexed(ref *ast.ColumnRef, s scopeIndexed) bool {
	if ref == nil || s.empty() {
		return false
	}
	col := strings.ToLower(ref.Column)
	if ref.Table != "" {
		alias := r.normalizeIdent(ref.Table)
		// Local scope takes precedence: if the alias exists locally (even as a
		// derived table or CTE without index metadata), the ref resolves there.
		// Otherwise the qualified alias is definitively outer, even when other
		// local sources have unresolved metadata — collectTableNames records
		// every local alias (including derived tables) so this check is sound.
		if s.localAliases[alias] {
			return s.local[alias][col]
		}
		return s.outer[alias][col]
	}
	// Unqualified: does any local table declare this column?
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

func (r *whereDisallowFuncOmniRule) checkWhere(where ast.ExprNode, tables tablesByAlias) {
	r.checkWhereWithScopes(where, tables, nil, false)
}

func (r *whereDisallowFuncOmniRule) checkWhereWithScopes(where ast.ExprNode, local, outer tablesByAlias, localUnresolved bool) {
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
	// Combined table set for recursing into nested subqueries (each is a new scope
	// where current local+outer both become "outer").
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

func (r *whereDisallowFuncOmniRule) checkWhereNode(n ast.Node, indexed scopeIndexed) {
	switch expr := n.(type) {
	case *ast.FuncCallExpr:
		if col := r.findIndexedColumnInArgs(expr.Args, indexed); col != "" {
			r.addFunctionAdvice(strings.ToUpper(expr.Name), col, expr.Loc)
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
		if expr.Op == ast.UnaryMinus || expr.Op == ast.UnaryBitNot {
			if col := r.unwrapAndFindIndexedColumn(expr.Operand, indexed); col != "" {
				r.addCalculationAdvice(col, expr.Loc)
			}
		}
	default:
	}
}

func (r *whereDisallowFuncOmniRule) addFunctionAdvice(funcName, col string, loc ast.Loc) {
	r.AddAdviceAbsolute(&storepb.Advice{
		Status:        r.Level,
		Code:          code.StatementDisallowFunctionsAndCalculations.Int32(),
		Title:         r.Title,
		Content:       fmt.Sprintf("Function %q is applied to indexed column %q in the WHERE clause, which prevents index usage", funcName, col),
		StartPosition: common.ConvertANTLRLineToPosition(r.BaseLine + int(r.LocToLine(loc))),
	})
}

func (r *whereDisallowFuncOmniRule) addCalculationAdvice(col string, loc ast.Loc) {
	r.AddAdviceAbsolute(&storepb.Advice{
		Status:        r.Level,
		Code:          code.StatementDisallowFunctionsAndCalculations.Int32(),
		Title:         r.Title,
		Content:       fmt.Sprintf("Calculation is applied to indexed column %q in the WHERE clause, which prevents index usage", col),
		StartPosition: common.ConvertANTLRLineToPosition(r.BaseLine + int(r.LocToLine(loc))),
	})
}

// findIndexedColumnInArgs searches function arguments for an indexed ColumnRef,
// descending through passthrough wrappers (ParenExpr, CastExpr) so that
// `UPPER((name))` and `UPPER(CAST(name AS CHAR))` are still flagged.
// BinaryExpr/UnaryExpr (arithmetic) are NOT crossed — those are handled by the
// calculation advice path, so `ABS(id + 1)` is reported once as a calculation.
func (r *whereDisallowFuncOmniRule) findIndexedColumnInArgs(args []ast.ExprNode, indexed scopeIndexed) string {
	for _, arg := range args {
		if col := r.unwrapAndFindIndexedColumn(arg, indexed); col != "" {
			return col
		}
	}
	return ""
}

// unwrapAndFindIndexedColumn peels ParenExpr/CastExpr wrappers off expr and, if the
// inner expression is an indexed ColumnRef, returns its name. Returns "" otherwise.
func (r *whereDisallowFuncOmniRule) unwrapAndFindIndexedColumn(expr ast.ExprNode, indexed scopeIndexed) string {
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
		default:
			return ""
		}
	}
	return ""
}

func isArithmeticOp(op ast.BinaryOp) bool {
	switch op {
	case ast.BinOpAdd, ast.BinOpSub, ast.BinOpMul, ast.BinOpDiv, ast.BinOpMod,
		ast.BinOpDivInt, ast.BinOpBitAnd, ast.BinOpBitOr, ast.BinOpBitXor,
		ast.BinOpShiftLeft, ast.BinOpShiftRight:
		return true
	default:
		return false
	}
}
