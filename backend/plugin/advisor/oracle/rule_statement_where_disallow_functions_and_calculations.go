// Package oracle is the advisor for oracle database.
package oracle

import (
	"context"
	"fmt"
	"strings"

	"github.com/bytebase/omni/oracle/ast"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	"github.com/bytebase/bytebase/backend/store/model"
)

var (
	_ advisor.Advisor = (*StatementWhereDisallowFunctionsAndCalculationsAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_ORACLE, storepb.SQLReviewRule_STATEMENT_WHERE_DISALLOW_FUNCTIONS_AND_CALCULATIONS, &StatementWhereDisallowFunctionsAndCalculationsAdvisor{})
}

// StatementWhereDisallowFunctionsAndCalculationsAdvisor is the Oracle advisor
// that flags functions or calculations applied to indexed columns in WHERE clauses.
type StatementWhereDisallowFunctionsAndCalculationsAdvisor struct{}

// Check implements advisor.Advisor.
func (*StatementWhereDisallowFunctionsAndCalculationsAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}
	// Oracle sync stores ONLY the connected schema's objects under a
	// SchemaMetadata{Name: ""} (see backend/plugin/db/oracle/sync.go:128-135
	// — d.databaseName is used as the DatabaseSchemaMetadata.Name and only
	// that owner's objects are populated). currentSchema must be "" to
	// match the anonymous container; cross-owner qualified refs (e.g.
	// `OTHER_OWNER.TECH_BOOK` where OTHER_OWNER != checkCtx.CurrentDatabase)
	// are NOT in our metadata and must return nil — see tableMetadata.
	rule := NewWhereDisallowFunctionsAndCalculationsRule(
		level,
		checkCtx.Rule.Type.String(),
		"",
		checkCtx.DBSchema,
		checkCtx.IsObjectCaseSensitive,
	)
	rule.currentDatabase = checkCtx.CurrentDatabase
	return RunOmniRules(checkCtx.ParsedStatements, []OmniRule{rule})
}

// ---- Types ---------------------------------------------------------------

type tableRef struct {
	schema string
	name   string
}

type tablesByAlias = map[string]tableRef
type indexedByAlias = map[string]map[string]bool

type scopeIndexed struct {
	local        indexedByAlias
	localAll     indexedByAlias
	localAliases map[string]bool
}

// WhereDisallowFunctionsAndCalculationsRule is the rule implementation.
type WhereDisallowFunctionsAndCalculationsRule struct {
	BaseRule
	currentSchema         string
	currentDatabase       string
	dbMetadata            *model.DatabaseMetadata
	isObjectCaseSensitive bool
	// cteStack mirrors the PG rule's CTE-shadowing handling. Pushed when
	// entering a Query_block with a Subquery_factoring_clause; popped on
	// exit. A FROM tableview_name matching any stacked entry is recorded
	// as opaque (`tableRef{}`) so it doesn't pick up base-table indexes.
	cteStack []map[string]bool
}

func (r *WhereDisallowFunctionsAndCalculationsRule) pushCTEs(names map[string]bool) {
	r.cteStack = append(r.cteStack, names)
}

func (r *WhereDisallowFunctionsAndCalculationsRule) popCTEs() {
	if n := len(r.cteStack); n > 0 {
		r.cteStack = r.cteStack[:n-1]
	}
}

func (r *WhereDisallowFunctionsAndCalculationsRule) cteVisible(name string) bool {
	for _, frame := range r.cteStack {
		if frame[name] {
			return true
		}
	}
	return false
}

// NewWhereDisallowFunctionsAndCalculationsRule constructs the rule.
func NewWhereDisallowFunctionsAndCalculationsRule(
	level storepb.Advice_Status,
	title string,
	currentSchema string,
	dbSchema *storepb.DatabaseSchemaMetadata,
	isObjectCaseSensitive bool,
) *WhereDisallowFunctionsAndCalculationsRule {
	r := &WhereDisallowFunctionsAndCalculationsRule{
		BaseRule:              NewBaseRule(level, title, 0),
		currentSchema:         currentSchema,
		isObjectCaseSensitive: isObjectCaseSensitive,
	}
	if dbSchema != nil {
		r.dbMetadata = model.NewDatabaseMetadata(dbSchema, nil, nil, storepb.Engine_ORACLE, isObjectCaseSensitive)
	}
	return r
}

// Name returns the rule name.
func (*WhereDisallowFunctionsAndCalculationsRule) Name() string {
	return "statement.where-disallow-functions-and-calculations"
}

// OnStatement checks omni query-bearing statements for functions or
// calculations applied to indexed columns in WHERE-like predicates.
func (r *WhereDisallowFunctionsAndCalculationsRule) OnStatement(node ast.Node) {
	switch n := node.(type) {
	case *ast.SelectStmt:
		r.checkOmniSelect(n, nil)
	case *ast.UpdateStmt:
		local := r.collectOmniDMLTargetTables(n.Target)
		r.checkOmniWhere(n.WhereClause, local)
		r.checkOmniNestedSubqueries(n, local)
	case *ast.DeleteStmt:
		local := r.collectOmniDMLTargetTables(n.Target)
		r.checkOmniWhere(n.WhereClause, local)
		r.checkOmniNestedSubqueries(n, local)
	case *ast.InsertStmt:
		if n.Select != nil {
			r.checkOmniSelect(n.Select, nil)
		}
		if subquery, ok := n.Subquery.(*ast.SelectStmt); ok {
			r.checkOmniSelect(subquery, nil)
			sourceTables := r.collectOmniTables(subquery.FromClause)
			for _, item := range listItems(n.MultiTable) {
				clause, ok := item.(*ast.InsertIntoClause)
				if !ok {
					continue
				}
				r.checkOmniWhere(clause.When, sourceTables)
			}
		}
		r.checkOmniNestedSubqueries(n, nil)
	case *ast.MergeStmt:
		local := tablesByAlias{}
		if n.Target != nil {
			ref := oracleTableRefFromObjectName(n.Target)
			local[""] = ref
			local[r.normalizeIdent(n.Target.Name)] = ref
			if n.TargetAlias != nil && n.TargetAlias.Name != "" {
				local[r.normalizeIdent(n.TargetAlias.Name)] = ref
			}
		}
		switch source := n.Source.(type) {
		case *ast.TableRef:
			if source.Name == nil {
				break
			}
			ref := oracleTableRefFromObjectName(source.Name)
			local[r.normalizeIdent(ref.name)] = ref
			if source.Alias != nil && source.Alias.Name != "" {
				local[r.normalizeIdent(source.Alias.Name)] = ref
			}
			if n.SourceAlias != nil && n.SourceAlias.Name != "" {
				local[r.normalizeIdent(n.SourceAlias.Name)] = ref
			}
		case *ast.SubqueryRef:
			if selectStmt, ok := source.Subquery.(*ast.SelectStmt); ok {
				r.checkOmniSelect(selectStmt, nil)
			}
			if source.Alias != nil && source.Alias.Name != "" {
				local[r.normalizeIdent(source.Alias.Name)] = tableRef{}
			}
			if n.SourceAlias != nil && n.SourceAlias.Name != "" {
				local[r.normalizeIdent(n.SourceAlias.Name)] = tableRef{}
			}
		default:
		}
		r.checkOmniWhere(n.On, local)
		for _, item := range listItems(n.Clauses) {
			clause, ok := item.(*ast.MergeClause)
			if !ok {
				continue
			}
			r.checkOmniWhere(clause.Condition, local)
			r.checkOmniWhere(clause.UpdateWhere, local)
			r.checkOmniWhere(clause.DeleteWhere, local)
			r.checkOmniWhere(clause.InsertWhere, local)
			r.checkOmniNestedSubqueries(clause, local)
		}
	case *ast.CreateTableStmt:
		if selectStmt, ok := n.AsQuery.(*ast.SelectStmt); ok {
			r.checkOmniSelect(selectStmt, nil)
		}
	case *ast.CreateViewStmt:
		if selectStmt, ok := n.Query.(*ast.SelectStmt); ok {
			r.checkOmniSelect(selectStmt, nil)
		}
	case *ast.PLSQLBlock:
		r.checkOmniPLSQLBlock(n)
	default:
	}
}

func (r *WhereDisallowFunctionsAndCalculationsRule) checkOmniPLSQLBlock(block *ast.PLSQLBlock) {
	omniWalkPLSQLBlockStatements(block, func(stmt ast.StmtNode) bool {
		return !r.checkOmniPLSQLQueryStatement(stmt)
	})
}

func (r *WhereDisallowFunctionsAndCalculationsRule) checkOmniPLSQLQueryStatement(stmt ast.StmtNode) bool {
	switch stmt.(type) {
	case *ast.SelectStmt, *ast.UpdateStmt, *ast.DeleteStmt, *ast.InsertStmt, *ast.MergeStmt, *ast.CreateTableStmt, *ast.CreateViewStmt:
		r.OnStatement(stmt)
		return true
	default:
		return false
	}
}

func (r *WhereDisallowFunctionsAndCalculationsRule) collectOmniDMLTargetTables(target ast.TableExpr) tablesByAlias {
	switch n := target.(type) {
	case *ast.TableRef:
		ref := oracleTableRefFromObjectName(n.Name)
		tables := tablesByAlias{"": ref}
		if n.Name != nil {
			tables[r.normalizeIdent(n.Name.Name)] = ref
		}
		if n.Alias != nil && n.Alias.Name != "" {
			tables[r.normalizeIdent(n.Alias.Name)] = ref
		}
		return tables
	case *ast.SubqueryRef:
		if selectStmt, ok := n.Subquery.(*ast.SelectStmt); ok {
			r.checkOmniSelect(selectStmt, nil)
		}
		if n.Alias != nil && n.Alias.Name != "" {
			return tablesByAlias{r.normalizeIdent(n.Alias.Name): tableRef{}}
		}
	default:
	}
	return nil
}

func (r *WhereDisallowFunctionsAndCalculationsRule) checkOmniNestedSubqueries(node ast.Node, outer tablesByAlias) {
	omniWalk(node, func(n ast.Node) {
		switch subquery := n.(type) {
		case *ast.SubqueryExpr:
			if selectStmt, ok := subquery.Subquery.(*ast.SelectStmt); ok {
				r.checkOmniSelect(selectStmt, outer)
			}
		case *ast.ExistsExpr:
			if selectStmt, ok := subquery.Subquery.(*ast.SelectStmt); ok {
				r.checkOmniSelect(selectStmt, outer)
			}
		default:
		}
	})
}

func oracleTableRefFromObjectName(name *ast.ObjectName) tableRef {
	if name == nil {
		return tableRef{}
	}
	return tableRef{schema: name.Schema, name: name.Name}
}

func (r *WhereDisallowFunctionsAndCalculationsRule) checkOmniSelect(stmt *ast.SelectStmt, outer tablesByAlias) {
	if stmt == nil {
		return
	}
	if stmt.WithClause != nil {
		names := make(map[string]bool)
		r.pushCTEs(names)
		defer r.popCTEs()
		for _, item := range listItems(stmt.WithClause.CTEs) {
			cte, ok := item.(*ast.CTE)
			if !ok {
				continue
			}
			cteName := r.normalizeIdent(cte.Name)
			if r.omniQueryIsRecursiveCTE(cte.Query, cteName) {
				names[cteName] = true
			}
			if query, ok := cte.Query.(*ast.SelectStmt); ok {
				r.checkOmniSelect(query, outer)
			}
			names[cteName] = true
		}
	}
	if stmt.Larg != nil {
		r.checkOmniSelect(stmt.Larg, outer)
	}
	if stmt.Rarg != nil {
		r.checkOmniSelect(stmt.Rarg, outer)
	}
	local := r.collectOmniTables(stmt.FromClause)
	tables := mergeTableAliases(outer, local)
	r.checkOmniWhere(stmt.WhereClause, tables)
	if stmt.Hierarchical != nil {
		r.checkOmniWhere(stmt.Hierarchical.StartWith, tables)
		r.checkOmniWhere(stmt.Hierarchical.ConnectBy, tables)
	}
	for _, item := range listItems(stmt.FromClause) {
		r.checkOmniJoinPredicates(item, tables)
	}
	omniWalk(stmt, func(n ast.Node) {
		subquery, ok := n.(*ast.SubqueryExpr)
		if ok {
			if nested, ok := subquery.Subquery.(*ast.SelectStmt); ok {
				r.checkOmniSelect(nested, mergeTableAliases(outer, local))
			}
		}
	})
}

func (r *WhereDisallowFunctionsAndCalculationsRule) omniQueryIsRecursiveCTE(node ast.Node, tableName string) bool {
	selectStmt, ok := node.(*ast.SelectStmt)
	if !ok || (selectStmt.Larg == nil && selectStmt.Rarg == nil) {
		return false
	}
	found := false
	omniWalk(node, func(n ast.Node) {
		if found {
			return
		}
		table, ok := n.(*ast.TableRef)
		if !ok || table.Name == nil {
			return
		}
		found = r.normalizeIdent(table.Name.Name) == tableName
	})
	return found
}

func (r *WhereDisallowFunctionsAndCalculationsRule) checkOmniJoinPredicates(node ast.Node, tables tablesByAlias) {
	join, ok := node.(*ast.JoinClause)
	if !ok {
		return
	}
	r.checkOmniWhere(join.On, tables)
	if left, ok := join.Left.(ast.Node); ok {
		r.checkOmniJoinPredicates(left, tables)
	}
	if right, ok := join.Right.(ast.Node); ok {
		r.checkOmniJoinPredicates(right, tables)
	}
}

func (r *WhereDisallowFunctionsAndCalculationsRule) collectOmniTables(from *ast.List) tablesByAlias {
	tables := make(tablesByAlias)
	for _, item := range listItems(from) {
		switch n := item.(type) {
		case *ast.TableRef:
			ref := oracleTableRefFromObjectName(n.Name)
			if r.cteVisible(r.normalizeIdent(ref.name)) {
				tables[r.normalizeIdent(ref.name)] = tableRef{}
				if n.Alias != nil && n.Alias.Name != "" {
					tables[r.normalizeIdent(n.Alias.Name)] = tableRef{}
				}
				continue
			}
			if ref.name == "" {
				continue
			}
			tables[r.normalizeIdent(ref.name)] = ref
			if n.Alias != nil && n.Alias.Name != "" {
				tables[r.normalizeIdent(n.Alias.Name)] = ref
			}
			if len(tables) == 1 {
				tables[""] = ref
			}
		case *ast.JoinClause:
			for alias, ref := range r.collectOmniJoinTables(n) {
				tables[alias] = ref
			}
		default:
		}
	}
	return tables
}

func (r *WhereDisallowFunctionsAndCalculationsRule) collectOmniJoinTables(join *ast.JoinClause) tablesByAlias {
	tables := make(tablesByAlias)
	for _, item := range []ast.TableExpr{join.Left, join.Right} {
		switch n := item.(type) {
		case *ast.TableRef:
			ref := oracleTableRefFromObjectName(n.Name)
			if r.cteVisible(r.normalizeIdent(ref.name)) {
				tables[r.normalizeIdent(ref.name)] = tableRef{}
				if n.Alias != nil && n.Alias.Name != "" {
					tables[r.normalizeIdent(n.Alias.Name)] = tableRef{}
				}
				continue
			}
			tables[r.normalizeIdent(ref.name)] = ref
			if n.Alias != nil && n.Alias.Name != "" {
				tables[r.normalizeIdent(n.Alias.Name)] = ref
			}
		case *ast.JoinClause:
			for alias, ref := range r.collectOmniJoinTables(n) {
				tables[alias] = ref
			}
		default:
		}
	}
	return tables
}

func mergeTableAliases(a, b tablesByAlias) tablesByAlias {
	merged := make(tablesByAlias)
	for k, v := range a {
		merged[k] = v
	}
	for k, v := range b {
		merged[k] = v
	}
	return merged
}

func (r *WhereDisallowFunctionsAndCalculationsRule) checkOmniWhere(expr ast.ExprNode, tables tablesByAlias) {
	if expr == nil || len(tables) == 0 {
		return
	}
	indexed := r.resolveIndexedColumns(tables)
	allCols := r.resolveAllColumns(tables)
	if len(indexed) == 0 {
		return
	}
	r.walkOmniWhere(expr, scopeIndexed{local: indexed, localAll: allCols, localAliases: map[string]bool{}}, tables)
}

func (r *WhereDisallowFunctionsAndCalculationsRule) walkOmniWhere(expr ast.ExprNode, scope scopeIndexed, tables tablesByAlias) {
	switch n := expr.(type) {
	case *ast.FuncCallExpr:
		if col := r.findIndexedColumnInOmniExprList(n.Args, scope); col != "" {
			r.addOmniFunctionAdvice(n.FuncName.Name, col, n.Loc)
		}
	case *ast.BinaryExpr:
		if n.Op != "=" && n.Op != "<>" && n.Op != "!=" && n.Op != "<" && n.Op != ">" && n.Op != "<=" && n.Op != ">=" {
			if col := r.findIndexedColumnInOmniExpr(n, scope); col != "" {
				r.addOmniCalculationAdvice(col, n.Loc)
			}
		}
	case *ast.UnaryExpr:
		if n.Op == "-" || n.Op == "+" {
			if col := r.findIndexedColumnInOmniExpr(n, scope); col != "" {
				r.addOmniCalculationAdvice(col, n.Loc)
			}
		}
	case *ast.SubqueryExpr:
		if selectStmt, ok := n.Subquery.(*ast.SelectStmt); ok {
			r.checkOmniSelect(selectStmt, tables)
		}
	case *ast.ExistsExpr:
		if selectStmt, ok := n.Subquery.(*ast.SelectStmt); ok {
			r.checkOmniSelect(selectStmt, tables)
		}
	default:
	}
	omniWalk(expr, func(child ast.Node) {
		if child == expr {
			return
		}
		if e, ok := child.(ast.ExprNode); ok {
			switch e.(type) {
			case *ast.FuncCallExpr, *ast.BinaryExpr, *ast.UnaryExpr, *ast.SubqueryExpr, *ast.ExistsExpr:
				r.walkOmniWhere(e, scope, tables)
			default:
			}
		}
	})
}

func (r *WhereDisallowFunctionsAndCalculationsRule) findIndexedColumnInOmniExprList(list *ast.List, scope scopeIndexed) string {
	for _, item := range listItems(list) {
		switch expr := item.(type) {
		case *ast.ColumnRef:
			qualifier := r.normalizeIdent(expr.Table)
			column := r.normalizeIdent(expr.Column)
			if r.isOmniIndexedColumn(qualifier, column, scope) {
				return column
			}
		case *ast.UnaryExpr:
			if expr.Op == "PRIOR" {
				if col := r.findDirectIndexedColumnInOmniExpr(expr.Operand, scope); col != "" {
					return col
				}
			}
		default:
		}
	}
	return ""
}

func (r *WhereDisallowFunctionsAndCalculationsRule) findDirectIndexedColumnInOmniExpr(expr ast.ExprNode, scope scopeIndexed) string {
	col, ok := expr.(*ast.ColumnRef)
	if !ok {
		return ""
	}
	qualifier := r.normalizeIdent(col.Table)
	column := r.normalizeIdent(col.Column)
	if r.isOmniIndexedColumn(qualifier, column, scope) {
		return column
	}
	return ""
}

func (r *WhereDisallowFunctionsAndCalculationsRule) findIndexedColumnInOmniExpr(expr ast.ExprNode, scope scopeIndexed) string {
	var found string
	omniWalk(expr, func(n ast.Node) {
		if found != "" {
			return
		}
		col, ok := n.(*ast.ColumnRef)
		if !ok {
			return
		}
		qualifier := r.normalizeIdent(col.Table)
		column := r.normalizeIdent(col.Column)
		if r.isOmniIndexedColumn(qualifier, column, scope) {
			found = column
		}
	})
	return found
}

func (*WhereDisallowFunctionsAndCalculationsRule) isOmniIndexedColumn(qualifier, column string, scope scopeIndexed) bool {
	if qualifier != "" {
		return scope.local[qualifier][column]
	}
	for alias, cols := range scope.local {
		if alias == "" {
			continue
		}
		if cols[column] {
			return true
		}
	}
	return scope.local[""][column]
}

func (r *WhereDisallowFunctionsAndCalculationsRule) addOmniFunctionAdvice(funcName, col string, loc ast.Loc) {
	content := fmt.Sprintf("Function %q is applied to indexed column %q in the WHERE clause, which prevents index usage", funcName, col)
	line := r.locLine(loc)
	if r.hasOmniAdvice(content, line) {
		return
	}
	r.adviceList = append(r.adviceList, &storepb.Advice{
		Status:        r.level,
		Code:          code.StatementDisallowFunctionsAndCalculations.Int32(),
		Title:         r.title,
		Content:       content,
		StartPosition: common.ConvertANTLRLineToPosition(line),
	})
}

func (r *WhereDisallowFunctionsAndCalculationsRule) addOmniCalculationAdvice(col string, loc ast.Loc) {
	content := fmt.Sprintf("Calculation is applied to indexed column %q in the WHERE clause, which prevents index usage", col)
	line := r.locLine(loc)
	if r.hasOmniAdvice(content, line) {
		return
	}
	r.adviceList = append(r.adviceList, &storepb.Advice{
		Status:        r.level,
		Code:          code.StatementDisallowFunctionsAndCalculations.Int32(),
		Title:         r.title,
		Content:       content,
		StartPosition: common.ConvertANTLRLineToPosition(line),
	})
}

func (r *WhereDisallowFunctionsAndCalculationsRule) hasOmniAdvice(content string, line int) bool {
	for _, advice := range r.adviceList {
		if advice.Content == content && advice.GetStartPosition().GetLine() == int32(line) {
			return true
		}
	}
	return false
}

// OnEnter dispatches on top-level DML / DDL-with-query contexts. Inner work
// is done via manual recursion; the depth guard suppresses the walker's
// automatic re-entry into nested DML/SELECT contexts.
//
// Invariant: for every matching context type, OnEnter increments depth and
// OnExit decrements depth — UNCONDITIONALLY, regardless of whether a
// handler was dispatched. This keeps push/pop balanced even when a
// dispatch context nests another (e.g. Insert_statement contains a
// Select_statement source). The handler is called only when this is the
// outermost matching context (depth == 1 after increment).

// Defensive reset: at the root of a freshly-walked tree, parent is nil.

// OnExit pops depth for the dispatch contexts pushed in OnEnter.

// ---- Identifier / metadata helpers ---------------------------------------

// normalizeIdent returns the canonical lookup key for an identifier, matching
// the case mode of `model.NewDatabaseMetadata`:
//
//   - Case-sensitive (production Oracle, store.IsObjectCaseSensitive=true):
//     metadata stores names as Oracle's catalog returns them, which is
//     UPPER-case for unquoted identifiers and original-case for quoted
//     identifiers. The rule must therefore UPPER-case unquoted AST text and
//     strip quotes from quoted text without changing case. Lowercasing
//     unconditionally (the previous behavior) made every `tech_book` AST ref
//     miss against the `TECH_BOOK` metadata key, silently disabling the
//     rule in production.
//   - Case-insensitive (older test mode): metadata is folded to lower-case
//     by `normalizeNameByCaseSensitivity`, so the rule lower-cases too.
func (r *WhereDisallowFunctionsAndCalculationsRule) normalizeIdent(name string) string {
	if name == "" {
		return name
	}
	if strings.HasPrefix(name, "\"") && strings.HasSuffix(name, "\"") && len(name) >= 2 {
		inner := name[1 : len(name)-1]
		if r.isObjectCaseSensitive {
			return inner
		}
		return strings.ToLower(inner)
	}
	if r.isObjectCaseSensitive {
		return strings.ToUpper(name)
	}
	return strings.ToLower(name)
}

// idExprText returns the canonical lookup key for an Id_expression context.

// tableMetadata fetches the store-model table for the given ref.
//
// Oracle sync stores ONLY the connected schema's objects in
// SchemaMetadata{Name: ""} (see backend/plugin/db/oracle/sync.go). So:
//   - Bare ref (`TECH_BOOK`) → look up in the anonymous schema (current owner).
//   - Same-owner qualified ref (`<currentDatabase>.TECH_BOOK`) → same as bare;
//     the qualifier matches the connected schema's owner.
//   - Cross-owner qualified ref (`OTHER_OWNER.TECH_BOOK` with
//     OTHER_OWNER != currentDatabase) → return nil. We have no metadata for
//     other owners; treating it as the current owner's table would attach
//     the wrong indexes and emit incorrect advice.
func (r *WhereDisallowFunctionsAndCalculationsRule) tableMetadata(ref tableRef) *model.TableMetadata {
	if r.dbMetadata == nil {
		return nil
	}
	if ref.schema != "" && r.currentDatabase != "" &&
		r.normalizeIdent(ref.schema) != r.normalizeIdent(r.currentDatabase) {
		return nil
	}
	return r.dbMetadata.GetSchemaMetadata(r.currentSchema).GetTable(ref.name)
}

// indexedColumns returns indexed column names for the given table reference.
func (r *WhereDisallowFunctionsAndCalculationsRule) indexedColumns(ref tableRef) map[string]bool {
	table := r.tableMetadata(ref)
	if table == nil {
		return nil
	}
	cols := make(map[string]bool)
	for _, idx := range table.GetProto().GetIndexes() {
		for _, expr := range idx.GetExpressions() {
			cols[r.normalizeIdent(expr)] = true
		}
	}
	if len(cols) == 0 {
		return nil
	}
	return cols
}

// allColumns returns every column name defined for the given table reference.
// Keys use normalizeIdent so case-sensitive schemas distinguish quoted
// mixed-case identifiers; see §5 "Identifier normalization invariant".
func (r *WhereDisallowFunctionsAndCalculationsRule) allColumns(ref tableRef) map[string]bool {
	table := r.tableMetadata(ref)
	if table == nil {
		return nil
	}
	cols := make(map[string]bool)
	for _, c := range table.GetProto().GetColumns() {
		cols[r.normalizeIdent(c.GetName())] = true
	}
	if len(cols) == 0 {
		return nil
	}
	return cols
}

func (r *WhereDisallowFunctionsAndCalculationsRule) resolveIndexedColumns(tables tablesByAlias) indexedByAlias {
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

func (r *WhereDisallowFunctionsAndCalculationsRule) resolveAllColumns(tables tablesByAlias) indexedByAlias {
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

// ---- FROM-clause collection ----------------------------------------------

// collectTableNamesFromFrom walks a From_clause and returns aliases → tableRef.
// Derived tables and CTE references (when they can't be resolved to a real
// table) are recorded as empty tableRef{} placeholders so qualified refs can
// still be routed via alias lookup.

// dml_table_expression_clause [pivot/unpivot]

// ( table_ref ) [subquery_operation_part]*
// If the inner table_ref is a single base table (no joins, no
// subquery), the outer alias points DIRECTLY to that table —
// `FROM (orders) o` is equivalent to `FROM orders o`. Recording
// the alias as opaque would erase the indexes and miss real
// violations on `o.indexed_col`. Otherwise (joins / derived
// tables / subquery_operation_part), the alias spans multiple
// or unknown sources and must stay opaque.

// ONLY ( dml_table_expression_clause )

// collectTableNamesFromDml records a dml_table_expression_clause (a table or
// a parenthesised SELECT) under the given alias (empty = use table name).

// CTE shadowing: an unqualified tableview matching a WITH-factoring
// name binds to the CTE result, not the physical table.

// A nested SELECT inside the dml-table-expression-clause (derived table).

// splitTableviewName returns (schema, name) from a Tableview_name context.

// tableAliasText returns the canonical alias string from a Table_alias.

// localHasUnresolvedSources reports whether any FROM entry hides columns we
// cannot enumerate (derived tables or tables without metadata).

// If the FROM subtree contains a Subquery (derived table), sources are unresolved.

// Opaque alias (CTE-shadowed tableview, derived source, etc.) —
// columns can't be enumerated, so the local scope is unresolved.

// singleBaseTableInRef returns the inner Dml_table_expression_clause iff the
// given Table_ref reduces to a single base table — no joins, no subquery
// expansion, no parenthesized join — recursing through nested parentheses
// (e.g. `((orders))`). Returns nil otherwise. Used to propagate the outer
// alias of `FROM (orders) o` directly to the underlying table so indexes
// resolve via `o.indexed_col`.

// Recurse — `((orders))` is still a single base table.

// containsDerivedTable reports whether any FROM-clause node introduces a
// derived table. Only SOURCE positions are inspected — Join_on_part subtrees
// (JOIN ON predicates) are skipped because their subqueries are value
// expressions, not source producers; treating them as opaque sources sets
// localUnresolved=true on the enclosing query and suppresses the outer-scope
// fallback in isIndexedColumn, masking real correlated-ref violations.
// Mirrors PG's hasOpaqueSource. See spec §3 "Unresolved FROM-item inventory".

// elementReferencesName reports whether a factoring element is recursive —
// i.e. its body is a UNION ALL composition AND at least one branch
// references the element's own name as a FROM source.
//
// Why require UNION ALL (not just self-reference):
//   - Oracle's recursive subquery factoring REQUIRES a UNION ALL between an
//     anchor member and a recursive member; a single-branch self-shadow
//     (e.g. `WITH orders AS (SELECT * FROM orders …)`) is NON-recursive and
//     resolves the inner FROM to the REAL base table per SQL scope rules.
//   - Using UNION ALL as the gate means the function's "recursive" return
//     value matches Oracle's runtime binding: recursive members see the
//     CTE (opaque); non-recursive bodies see the base table (indexed).
//
// Walk rule for self-reference inside a branch: scan every descendant
// Dml_table_expression_clause; match on Tableview_name iff it has no
// schema qualifier and its normalized name equals `name`. Nested derived
// tables (dte with its own Select_statement) are opaque — their inner
// FROMs belong to a different scope and must not trigger a false match.

// Derived table: its inner SELECT is an opaque nested scope — do NOT
// descend. Its own Tableview_name IS in the element's FROM scope.

// ---- Column reference resolution -----------------------------------------

// extractColumnRefFromGeneralElement returns (qualifier, column) if the
// general_element is a plain column reference (no function arguments on any
// part). Returns ("", "") when the ref is a function call or otherwise not a
// column.

// a.b → (a, b). schema.table.col → (table, col).

// extractColumnRefFromTableElement handles `table_element outer_join_sign`
// (the bare-column-with-(+) form). Returns (qualifier, column).

// ---- Function-name extraction --------------------------------------------

// functionCallName returns the upper-case function name applied, if the given
// general_element is a function call. Returns ("", false) otherwise.

// We treat a general_element as a function call when the LAST part has a
// function_argument. The function name is the concatenation of all
// id_expressions (so DBMS_RANDOM.VALUE(...) is "DBMS_RANDOM.VALUE").

// Check if an earlier part had a function_argument — e.g. `foo(x).bar`
// — which is "method access on a call". Treat as function form too.

// extractStandardFunctionName returns the function keyword for a
// standard_function context (SUBSTR, TO_CHAR, COUNT, CAST, TRIM, etc.).

// ---- Core WHERE detection ------------------------------------------------

// checkQueryBlock inspects a Query_block with the given outer scope threaded in.

// CTE visibility matrix — per-factoring-element, matching the PG rule's
// handleWithClause but at element granularity because Oracle has no
// RECURSIVE keyword at the factoring-clause level (it is implicit —
// an element is recursive iff its body is a UNION ALL composition AND
// at least one branch references its own query_name in a FROM position,
// which matches Oracle's syntactic requirement for recursive subquery
// factoring):
//
//	| Element kind         | body sees self | preceding siblings | later siblings |
//	|----------------------|----------------|--------------------|----------------|
//	| Non-recursive        |       No       |         Yes        |        No      |
//	| Recursive            |      Yes       |         Yes        |        No      |
//
// Why per-element (not per-factoring-clause):
//   - Non-recursive body self-shadow (e.g. `WITH orders AS (SELECT * FROM
//     orders …)`, no UNION ALL) — inner `FROM orders` resolves to the
//     REAL base table per SQL scope rules; must push-AFTER so the
//     body-walk sees the base table and can emit `UPPER(indexed_col)`
//     advice. Otherwise false negative.
//   - Recursive body self-ref (e.g. `WITH orders AS (SELECT … FROM dual
//     UNION ALL SELECT … FROM orders WHERE UPPER(customer_name)=…)`) —
//     inner `FROM orders` resolves to the CTE itself; must push-BEFORE
//     so body-walk sees it as opaque and DOESN'T attach the same-named
//     base table's indexes. Otherwise false positive on recursive
//     members.
//
// Detection rule (elementReferencesName):
//   1. body has at least one Subquery_operation_part that is `UNION ALL`,
//   2. any descendant Dml_table_expression_clause's Tableview_name is a
//      single-component (unqualified) name whose normalized text equals
//      the element's own query_name.
// Derived-table subquery sources are treated as opaque (not descended).
//
// Prior commit 1ee832f7f0 collapsed push-before/push-after into an
// unconditional push-after, which fixed the non-recursive false negative
// but re-opened the recursive false positive — addressed by codex
// #3109425426.

// Push-BEFORE: body's self-reference resolves to the CTE
// (opaque), not the same-named base table.

// Push-AFTER: body's self-reference resolves to the REAL
// base table per non-recursive SQL scope rules.

// ANSI JOIN ON predicates behave like WHERE predicates in the same scope.

// CONNECT BY / START WITH predicates behave like WHERE predicates.

// Recurse into subqueries inside SELECT list, FROM (derived tables), HAVING,
// GROUP BY, ORDER BY — all with combined scope.

// checkWhereClause is the entry point for the WHERE sub-tree of a query block.

// checkExpressionAsWhere runs the function/calculation detector on an arbitrary
// expression tree, recursing into nested subqueries.

// walkWhere traverses the tree looking for nodes to check. Nested subqueries
// are re-entered via checkSubqueryInScope with the combined scope.

// Re-enter into nested subqueries without running the detector on them.

// General_element: may be a column ref (skip — handled via parent) or a
// function call (detect).

// Still recurse into the arguments — a nested function/calc on an
// indexed column should ALSO be flagged (e.g. `ABS(id + 1)`).

// Not a function call — the column ref case is reached via parent
// nodes (concatenation / unary). Fall through.

// Standard_function: built-in (SUBSTR / COUNT / CAST / TRIM / …).

// Special-case CAST: it's a transparent wrapper for the indexed-col
// check but we still also flag CAST-on-indexed as a function.

// Concatenation: arithmetic ASTERISK / SOLIDUS / PLUS_SIGN / MINUS_SIGN.

// Unary minus → calculation (NOT PRIOR, NOT CONNECT_BY_ROOT).

// isArithmeticConcat reports whether the concatenation is an arithmetic op.

// unwrapAndFindIndexedColumn peels passthrough wrappers (parens, CAST) and,
// if the inner expression is an indexed column reference, returns its
// lower-cased column name. Returns "" otherwise.

// passthrough: only if not itself an arithmetic op

// CAST unwrap: `CAST(col AS TYPE)` — the inside is transparent for column-ref extraction.

// LEFT_PAREN expressions RIGHT_PAREN — passthrough if single.

// (expressions) — unwrap if single expression.

// unwrapCastColumn returns the indexed-column name inside a `CAST(col AS T)`
// expression, or "" if the inside isn't a bare indexed column.

// findIndexedColumnInFunctionArgs searches a function-call's argument list for
// an indexed column reference (unwrapping CAST / parens). Arithmetic is NOT
// crossed so `ABS(id + 1)` reports as a calculation via the inner walker.

// findIndexedColumnInStandardFunction searches a built-in function's argument
// subtree for an indexed column reference.

// Walk the function's children (NOT the root node itself — we want to
// descend into its arguments, stopping only at further function calls /
// arithmetic).

// findIndexedColumnInChildren recursively searches for an indexed column
// reference without crossing into arithmetic concatenations (those spawn the
// calculation path) or further function calls.

// ---- Subquery re-entry ---------------------------------------------------

// checkSubqueryInScope re-enters the recursive check on a SubqueryContext,
// threading the given outer scope for correlated refs.

// Set-op: first element plus each subquery_operation_part.

// recurseIntoNestedSubqueries scans the non-WHERE parts of a query block for
// SubqueryContext nodes and dispatches each with the given outer scope.

// SELECT list

// FROM (derived tables). Subqueries inside JOIN ON predicates are already
// walked by checkQueryBlock via checkExpressionAsWhere; skip them here to
// avoid double-flagging.

// GROUP BY + HAVING (HAVING is never itself flagged; nested subqueries
// inside are still recursed).

// recurseIntoSubqueriesSkipWhere is like recurseIntoSubqueriesIn but skips
// Where_clauseContext subtrees. Callers that already dispatched the WHERE
// through checkWhereClause use this variant to avoid double-walking subqueries
// inside that WHERE (checkWhereClause → walkWhere re-enters SubqueryContext on
// its own).

// recurseIntoFromSubqueries walks a FROM-clause subtree for derived-table
// subqueries but skips Join_on_part subtrees — JOIN ON conditions are
// dispatched separately by checkQueryBlock and would otherwise cause the
// same inner subquery to be checked twice.

// ---- Top-level dispatchers -----------------------------------------------

// checkSelectOnly dispatches the top-level subquery of a Select_only_statement.

// onSelectStmt handles SELECT (including CTE / UNION / MINUS / INTERSECT).

// tablesUnresolved reports whether any concrete tableRef lacks metadata.
// Empty (placeholder) refs are skipped; use emptyIsUnresolved=true when an empty
// onUpdateStmt handles UPDATE ... SET ... WHERE ...

// Recurse into subqueries in SET expressions and WHERE subqueries (latter handled via walkWhere).

// onDeleteStmt handles DELETE [FROM] … WHERE …

// dispatchTargetSubquery dispatches the inner SELECT when the UPDATE / DELETE
// target is a derived table (`UPDATE (SELECT … WHERE …) SET …`). The outer
// collector records only an opaque alias placeholder; the inner WHERE must
// still be checked independently.

// onInsertStmt handles INSERT INTO … SELECT / VALUES …

// VALUES (subquery) — walk only the values and returning trees, not
// the full sti (which would double-walk a sibling SELECT source).

// INSERT ALL WHEN <cond> THEN INTO ... — the WHEN predicate is a
// per-row filter over the driving SELECT's columns. Build a scope from
// that SELECT's FROM clause so indexed-column refs resolve.

// INTO branches may carry VALUES subqueries — recurse into
// them. The WHEN Condition itself is already handled above
// via checkExpressionAsWhere (which re-enters subqueries);
// skip it here to avoid double-walking.

// Walk non-conditional multi-table-element children too (for INSERT
// ALL without WHEN branches). Skip the driving SELECT (already
// handled) and the Conditional_insert_clause (handled above, where
// we split WHEN Condition from INTO branches).

// onCreateTable handles CREATE TABLE ... AS subquery (CTAS).

// onCreateView handles CREATE [OR REPLACE] VIEW v AS SELECT …

// onCreateMaterializedView handles CREATE MATERIALIZED VIEW mv AS SELECT …

// onMergeStmt handles MERGE INTO target USING source ON cond WHEN …

// MERGE source may be a subquery (`USING (SELECT … WHERE …)`). The
// outer collector records only an opaque alias placeholder; run the
// rule on the inner SELECT independently so its own WHERE is checked.

// SET col = (subquery) — recurse into any nested subqueries, skipping
// the Where_clause subtrees already dispatched above.

// INSERT VALUES (subquery) or column-list subqueries — same recursion,
// skipping the Where_clause subtree already dispatched above.

// collectTableNamesFromGeneralTableRef records the UPDATE / DELETE target.

// collectTableNamesFromSelectedTableview records a MERGE target / source.

// CTE shadowing — see collectTableNamesFromDml.

// findFirstSubquery does a DFS for the first SubqueryContext child in the tree.

// ---- Advice builders ------------------------------------------------------
