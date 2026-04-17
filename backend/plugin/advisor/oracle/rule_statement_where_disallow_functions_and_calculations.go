// Package oracle is the advisor for oracle database.
package oracle

import (
	"context"
	"fmt"
	"strings"

	"github.com/antlr4-go/antlr/v4"
	parser "github.com/bytebase/parser/plsql"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
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
	checker := NewGenericChecker([]Rule{rule})
	for _, stmt := range checkCtx.ParsedStatements {
		if stmt.AST == nil {
			continue
		}
		antlrAST, ok := base.GetANTLRAST(stmt.AST)
		if !ok {
			continue
		}
		rule.SetBaseLine(stmt.BaseLine())
		checker.SetBaseLine(stmt.BaseLine())
		rule.depth = 0
		antlr.ParseTreeWalkerDefault.Walk(checker, antlrAST.Tree)
	}
	return checker.GetAdviceList()
}

// ---- Types ---------------------------------------------------------------

type tableRef struct {
	schema string
	name   string
}

type tablesByAlias = map[string]tableRef
type indexedByAlias = map[string]map[string]bool

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

// WhereDisallowFunctionsAndCalculationsRule is the rule implementation.
type WhereDisallowFunctionsAndCalculationsRule struct {
	BaseRule
	currentSchema         string
	currentDatabase       string
	dbMetadata            *model.DatabaseMetadata
	isObjectCaseSensitive bool
	depth                 int
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
func (r *WhereDisallowFunctionsAndCalculationsRule) OnEnter(ctx antlr.ParserRuleContext, _ string) error {
	// Defensive reset: at the root of a freshly-walked tree, parent is nil.
	if ctx.GetParent() == nil {
		r.depth = 0
	}
	shouldDispatch := r.depth == 0
	switch c := ctx.(type) {
	case *parser.Select_statementContext:
		r.depth++
		if shouldDispatch {
			r.onSelectStmt(c)
		}
	case *parser.Update_statementContext:
		r.depth++
		if shouldDispatch {
			r.onUpdateStmt(c)
		}
	case *parser.Delete_statementContext:
		r.depth++
		if shouldDispatch {
			r.onDeleteStmt(c)
		}
	case *parser.Insert_statementContext:
		r.depth++
		if shouldDispatch {
			r.onInsertStmt(c)
		}
	case *parser.Merge_statementContext:
		r.depth++
		if shouldDispatch {
			r.onMergeStmt(c)
		}
	case *parser.Create_tableContext:
		r.depth++
		if shouldDispatch {
			r.onCreateTable(c)
		}
	case *parser.Create_viewContext:
		r.depth++
		if shouldDispatch {
			r.onCreateView(c)
		}
	case *parser.Create_materialized_viewContext:
		r.depth++
		if shouldDispatch {
			r.onCreateMaterializedView(c)
		}
	default:
	}
	return nil
}

// OnExit pops depth for the dispatch contexts pushed in OnEnter.
func (r *WhereDisallowFunctionsAndCalculationsRule) OnExit(ctx antlr.ParserRuleContext, _ string) error {
	switch ctx.(type) {
	case *parser.Select_statementContext,
		*parser.Update_statementContext,
		*parser.Delete_statementContext,
		*parser.Insert_statementContext,
		*parser.Merge_statementContext,
		*parser.Create_tableContext,
		*parser.Create_viewContext,
		*parser.Create_materialized_viewContext:
		if r.depth > 0 {
			r.depth--
		}
	default:
	}
	return nil
}

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
func (r *WhereDisallowFunctionsAndCalculationsRule) idExprText(ctx parser.IId_expressionContext) string {
	if ctx == nil {
		return ""
	}
	if reg := ctx.Regular_id(); reg != nil {
		return r.normalizeIdent(reg.GetText())
	}
	if d := ctx.DELIMITED_ID(); d != nil {
		return r.normalizeIdent(d.GetText())
	}
	return r.normalizeIdent(ctx.GetText())
}

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
func (r *WhereDisallowFunctionsAndCalculationsRule) collectTableNamesFromFrom(from parser.IFrom_clauseContext) tablesByAlias {
	tables := make(tablesByAlias)
	if from == nil {
		return tables
	}
	list := from.Table_ref_list()
	if list == nil {
		return tables
	}
	for _, tref := range list.AllTable_ref() {
		r.collectTableNamesFromTableRef(tref, tables)
	}
	return tables
}

func (r *WhereDisallowFunctionsAndCalculationsRule) collectTableNamesFromTableRef(tref parser.ITable_refContext, tables tablesByAlias) {
	if tref == nil {
		return
	}
	if aux := tref.Table_ref_aux(); aux != nil {
		r.collectTableNamesFromTableRefAux(aux, tables)
	}
	for _, jc := range tref.AllJoin_clause() {
		if jc == nil {
			continue
		}
		if aux := jc.Table_ref_aux(); aux != nil {
			r.collectTableNamesFromTableRefAux(aux, tables)
		}
	}
}

func (r *WhereDisallowFunctionsAndCalculationsRule) collectTableNamesFromTableRefAux(aux parser.ITable_ref_auxContext, tables tablesByAlias) {
	if aux == nil {
		return
	}
	internal := aux.Table_ref_aux_internal()
	alias := ""
	if ta := aux.Table_alias(); ta != nil {
		alias = r.tableAliasText(ta)
	}
	switch c := internal.(type) {
	case *parser.Table_ref_aux_internal_oneContext:
		// dml_table_expression_clause [pivot/unpivot]
		if dte := c.Dml_table_expression_clause(); dte != nil {
			r.collectTableNamesFromDml(dte, alias, tables)
		}
	case *parser.Table_ref_aux_internal_twoContext:
		// ( table_ref ) [subquery_operation_part]*
		// If the inner table_ref is a single base table (no joins, no
		// subquery), the outer alias points DIRECTLY to that table —
		// `FROM (orders) o` is equivalent to `FROM orders o`. Recording
		// the alias as opaque would erase the indexes and miss real
		// violations on `o.indexed_col`. Otherwise (joins / derived
		// tables / subquery_operation_part), the alias spans multiple
		// or unknown sources and must stay opaque.
		if inner := c.Table_ref(); inner != nil {
			if dte := singleBaseTableInRef(inner); dte != nil {
				r.collectTableNamesFromDml(dte, alias, tables)
			} else {
				if alias != "" {
					key := r.normalizeIdent(alias)
					if _, ok := tables[key]; !ok {
						tables[key] = tableRef{}
					}
				}
				r.collectTableNamesFromTableRef(inner, tables)
			}
		} else if alias != "" {
			key := r.normalizeIdent(alias)
			if _, ok := tables[key]; !ok {
				tables[key] = tableRef{}
			}
		}
	case *parser.Table_ref_aux_internal_threeContext:
		// ONLY ( dml_table_expression_clause )
		if dte := c.Dml_table_expression_clause(); dte != nil {
			r.collectTableNamesFromDml(dte, alias, tables)
		}
	default:
	}
}

// collectTableNamesFromDml records a dml_table_expression_clause (a table or
// a parenthesised SELECT) under the given alias (empty = use table name).
func (r *WhereDisallowFunctionsAndCalculationsRule) collectTableNamesFromDml(
	dte parser.IDml_table_expression_clauseContext,
	alias string,
	tables tablesByAlias,
) {
	dteCtx := dte
	if tvn := dteCtx.Tableview_name(); tvn != nil {
		schema, name := r.splitTableviewName(tvn)
		key := alias
		if key == "" {
			key = name
		}
		normKey := r.normalizeIdent(key)
		// CTE shadowing: an unqualified tableview matching a WITH-factoring
		// name binds to the CTE result, not the physical table.
		if schema == "" && r.cteVisible(name) {
			tables[normKey] = tableRef{}
			return
		}
		ref := tableRef{schema: schema, name: name}
		if existing, ok := tables[normKey]; ok && (existing.schema != ref.schema || existing.name != ref.name) {
			tables[normKey] = tableRef{}
		} else {
			tables[normKey] = ref
		}
		return
	}
	// A nested SELECT inside the dml-table-expression-clause (derived table).
	if dteCtx.Select_statement() != nil {
		key := r.normalizeIdent(alias)
		if key != "" {
			if _, ok := tables[key]; !ok {
				tables[key] = tableRef{}
			}
		}
	}
}

// splitTableviewName returns (schema, name) from a Tableview_name context.
func (r *WhereDisallowFunctionsAndCalculationsRule) splitTableviewName(tvn parser.ITableview_nameContext) (string, string) {
	if tvn == nil {
		return "", ""
	}
	first := ""
	if id := tvn.Identifier(); id != nil {
		first = r.idExprText(id.Id_expression())
	}
	if idExpr := tvn.Id_expression(); idExpr != nil {
		return first, r.idExprText(idExpr)
	}
	return "", first
}

// tableAliasText returns the canonical alias string from a Table_alias.
func (r *WhereDisallowFunctionsAndCalculationsRule) tableAliasText(ta parser.ITable_aliasContext) string {
	if ta == nil {
		return ""
	}
	if id := ta.Identifier(); id != nil {
		return r.idExprText(id.Id_expression())
	}
	if q := ta.Quoted_string(); q != nil {
		text := strings.Trim(q.GetText(), "'\"")
		return r.normalizeIdent(text)
	}
	return ""
}

// localHasUnresolvedSources reports whether any FROM entry hides columns we
// cannot enumerate (derived tables or tables without metadata).
func (r *WhereDisallowFunctionsAndCalculationsRule) localHasUnresolvedSources(from parser.IFrom_clauseContext, local tablesByAlias) bool {
	if from != nil {
		// If the FROM subtree contains a Subquery (derived table), sources are unresolved.
		if containsDerivedTable(from) {
			return true
		}
	}
	for _, ref := range local {
		if ref == (tableRef{}) {
			// Opaque alias (CTE-shadowed tableview, derived source, etc.) —
			// columns can't be enumerated, so the local scope is unresolved.
			return true
		}
		if r.tableMetadata(ref) == nil {
			return true
		}
	}
	return false
}

// singleBaseTableInRef returns the inner Dml_table_expression_clause iff the
// given Table_ref reduces to a single base table — no joins, no subquery
// expansion, no parenthesized join — recursing through nested parentheses
// (e.g. `((orders))`). Returns nil otherwise. Used to propagate the outer
// alias of `FROM (orders) o` directly to the underlying table so indexes
// resolve via `o.indexed_col`.
func singleBaseTableInRef(tr parser.ITable_refContext) parser.IDml_table_expression_clauseContext {
	if tr == nil {
		return nil
	}
	if joins := tr.AllJoin_clause(); len(joins) > 0 {
		return nil
	}
	aux := tr.Table_ref_aux()
	if aux == nil {
		return nil
	}
	switch internal := aux.Table_ref_aux_internal().(type) {
	case *parser.Table_ref_aux_internal_oneContext:
		if dte := internal.Dml_table_expression_clause(); dte != nil && dte.Tableview_name() != nil {
			return dte
		}
		return nil
	case *parser.Table_ref_aux_internal_twoContext:
		// Recurse — `((orders))` is still a single base table.
		return singleBaseTableInRef(internal.Table_ref())
	}
	return nil
}

// containsDerivedTable reports whether any FROM-clause node introduces a
// derived table. Only SOURCE positions are inspected — Join_on_part subtrees
// (JOIN ON predicates) are skipped because their subqueries are value
// expressions, not source producers; treating them as opaque sources sets
// localUnresolved=true on the enclosing query and suppresses the outer-scope
// fallback in isIndexedColumn, masking real correlated-ref violations.
// Mirrors PG's hasOpaqueSource. See spec §3 "Unresolved FROM-item inventory".
func containsDerivedTable(node antlr.Tree) bool {
	if node == nil {
		return false
	}
	if _, ok := node.(*parser.Join_on_partContext); ok {
		return false
	}
	if n, ok := node.(*parser.Table_ref_aux_internal_oneContext); ok {
		if dte, ok := n.Dml_table_expression_clause().(*parser.Dml_table_expression_clauseContext); ok && dte != nil && dte.Select_statement() != nil {
			return true
		}
	}
	for i := 0; i < node.GetChildCount(); i++ {
		if containsDerivedTable(node.GetChild(i)) {
			return true
		}
	}
	return false
}

// ---- Column reference resolution -----------------------------------------

// extractColumnRefFromGeneralElement returns (qualifier, column) if the
// general_element is a plain column reference (no function arguments on any
// part). Returns ("", "") when the ref is a function call or otherwise not a
// column.
func (r *WhereDisallowFunctionsAndCalculationsRule) extractColumnRefFromGeneralElement(ge parser.IGeneral_elementContext) (string, string) {
	if ge == nil {
		return "", ""
	}
	parts := ge.AllGeneral_element_part()
	if len(parts) == 0 {
		return "", ""
	}
	for _, p := range parts {
		if p.Function_argument() != nil {
			return "", ""
		}
	}
	names := make([]string, 0, len(parts))
	for _, p := range parts {
		names = append(names, r.idExprText(p.Id_expression()))
	}
	if len(names) == 1 {
		return "", names[0]
	}
	// a.b → (a, b). schema.table.col → (table, col).
	return names[len(names)-2], names[len(names)-1]
}

// extractColumnRefFromTableElement handles `table_element outer_join_sign`
// (the bare-column-with-(+) form). Returns (qualifier, column).
func (r *WhereDisallowFunctionsAndCalculationsRule) extractColumnRefFromTableElement(te parser.ITable_elementContext) (string, string) {
	if te == nil {
		return "", ""
	}
	ids := te.AllId_expression()
	if len(ids) == 0 {
		return "", ""
	}
	names := make([]string, 0, len(ids))
	for _, id := range ids {
		names = append(names, r.idExprText(id))
	}
	if len(names) == 1 {
		return "", names[0]
	}
	return names[len(names)-2], names[len(names)-1]
}

// isIndexedColumn reports whether a (qualifier, column) pair refers to an
// indexed column within scope s.
func (r *WhereDisallowFunctionsAndCalculationsRule) isIndexedColumn(qualifier, column string, s scopeIndexed) bool {
	if column == "" || s.empty() {
		return false
	}
	col := r.normalizeIdent(column)
	if qualifier != "" {
		alias := r.normalizeIdent(qualifier)
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

// ---- Function-name extraction --------------------------------------------

// functionCallName returns the upper-case function name applied, if the given
// general_element is a function call. Returns ("", false) otherwise.
func (r *WhereDisallowFunctionsAndCalculationsRule) functionCallName(ge parser.IGeneral_elementContext) (string, bool) {
	if ge == nil {
		return "", false
	}
	parts := ge.AllGeneral_element_part()
	if len(parts) == 0 {
		return "", false
	}
	// We treat a general_element as a function call when the LAST part has a
	// function_argument. The function name is the concatenation of all
	// id_expressions (so DBMS_RANDOM.VALUE(...) is "DBMS_RANDOM.VALUE").
	last := parts[len(parts)-1]
	if last.Function_argument() == nil {
		// Check if an earlier part had a function_argument — e.g. `foo(x).bar`
		// — which is "method access on a call". Treat as function form too.
		hasCall := false
		for _, p := range parts {
			if p.Function_argument() != nil {
				hasCall = true
				break
			}
		}
		if !hasCall {
			return "", false
		}
	}
	var names []string
	for _, p := range parts {
		names = append(names, r.idExprText(p.Id_expression()))
	}
	return strings.ToUpper(strings.Join(names, ".")), true
}

// extractStandardFunctionName returns the function keyword for a
// standard_function context (SUBSTR, TO_CHAR, COUNT, CAST, TRIM, etc.).
func (*WhereDisallowFunctionsAndCalculationsRule) extractStandardFunctionName(ctx *parser.Standard_functionContext) string {
	if ctx == nil {
		return ""
	}
	first := ctx.GetStart()
	if first == nil {
		return ""
	}
	return strings.ToUpper(first.GetText())
}

// ---- Core WHERE detection ------------------------------------------------

// checkQueryBlock inspects a Query_block with the given outer scope threaded in.
func (r *WhereDisallowFunctionsAndCalculationsRule) checkQueryBlock(qb parser.IQuery_blockContext, outer tablesByAlias) {
	if qb == nil {
		return
	}
	// CTE visibility per Oracle WITH semantics: push each factoring name
	// BEFORE walking its body, in declaration order. The body sees all
	// preceding siblings AND itself; later siblings remain invisible.
	//
	// This invariant satisfies both:
	//   - non-recursive WITH (preceding siblings)
	//   - recursive factoring (self-reference inside body resolves to
	//     opaque CTE, preventing false positives that would otherwise
	//     attach a same-named base table's indexes to CTE columns).
	//
	// Mirrors the PG handleWithClause logic (see that function's comment
	// for the full design history of why earlier oscillations between
	// push-all-first and push-after-body were both wrong).
	if sfc := qb.Subquery_factoring_clause(); sfc != nil {
		frame := make(map[string]bool)
		pushed := false
		for _, fe := range sfc.AllFactoring_element() {
			if fe == nil {
				continue
			}
			if qn := fe.Query_name(); qn != nil {
				if id := qn.Identifier(); id != nil {
					if text := id.GetText(); text != "" {
						frame[r.normalizeIdent(text)] = true
						if !pushed {
							r.pushCTEs(frame)
							pushed = true
						}
					}
				}
			}
			if sub := fe.Subquery(); sub != nil {
				r.checkSubqueryInScope(sub, nil)
			}
		}
		if pushed {
			defer r.popCTEs()
		}
	}
	from := qb.From_clause()
	local := r.collectTableNamesFromFrom(from)
	unresolved := r.localHasUnresolvedSources(from, local)
	combined := mergeAliases(outer, local)

	r.checkWhereClause(qb.Where_clause(), local, outer, unresolved)

	// ANSI JOIN ON predicates behave like WHERE predicates in the same scope.
	if from != nil {
		if list := from.Table_ref_list(); list != nil {
			for _, tref := range list.AllTable_ref() {
				if tref == nil {
					continue
				}
				for _, jc := range tref.AllJoin_clause() {
					if jc == nil {
						continue
					}
					for _, jop := range jc.AllJoin_on_part() {
						if jop == nil {
							continue
						}
						if cond := jop.Condition(); cond != nil {
							r.checkExpressionAsWhere(cond, local, outer, unresolved)
						}
					}
				}
			}
		}
	}

	// CONNECT BY / START WITH predicates behave like WHERE predicates.
	if hq := qb.Hierarchical_query_clause(); hq != nil {
		if sp := hq.Start_part(); sp != nil {
			if cond := sp.Condition(); cond != nil {
				r.checkExpressionAsWhere(cond, local, outer, unresolved)
			}
		}
		if cond := hq.Condition(); cond != nil {
			r.checkExpressionAsWhere(cond, local, outer, unresolved)
		}
	}

	// Recurse into subqueries inside SELECT list, FROM (derived tables), HAVING,
	// GROUP BY, ORDER BY — all with combined scope.
	r.recurseIntoNestedSubqueries(qb, combined)
}

func mergeAliases(outer, local tablesByAlias) tablesByAlias {
	combined := make(tablesByAlias, len(outer)+len(local))
	for a, r := range outer {
		combined[a] = r
	}
	for a, r := range local {
		combined[a] = r
	}
	return combined
}

// checkWhereClause is the entry point for the WHERE sub-tree of a query block.
func (r *WhereDisallowFunctionsAndCalculationsRule) checkWhereClause(where parser.IWhere_clauseContext, local, outer tablesByAlias, unresolved bool) {
	if where == nil {
		return
	}
	if expr := where.Expression(); expr != nil {
		r.checkExpressionAsWhere(expr, local, outer, unresolved)
	}
}

// checkExpressionAsWhere runs the function/calculation detector on an arbitrary
// expression tree, recursing into nested subqueries.
func (r *WhereDisallowFunctionsAndCalculationsRule) checkExpressionAsWhere(root antlr.Tree, local, outer tablesByAlias, unresolved bool) {
	localAliases := make(map[string]bool, len(local))
	for a := range local {
		localAliases[a] = true
	}
	s := scopeIndexed{
		local:           r.resolveIndexedColumns(local),
		localAll:        r.resolveAllColumns(local),
		localAliases:    localAliases,
		outer:           r.resolveIndexedColumns(outer),
		localUnresolved: unresolved,
	}
	combined := mergeAliases(outer, local)
	r.walkWhere(root, s, combined)
}

// walkWhere traverses the tree looking for nodes to check. Nested subqueries
// are re-entered via checkSubqueryInScope with the combined scope.
func (r *WhereDisallowFunctionsAndCalculationsRule) walkWhere(n antlr.Tree, s scopeIndexed, combined tablesByAlias) {
	if n == nil {
		return
	}
	// Re-enter into nested subqueries without running the detector on them.
	if sub, ok := n.(*parser.SubqueryContext); ok {
		r.checkSubqueryInScope(sub, combined)
		return
	}
	// General_element: may be a column ref (skip — handled via parent) or a
	// function call (detect).
	if ge, ok := n.(*parser.General_elementContext); ok {
		if name, isCall := r.functionCallName(ge); isCall {
			if col := r.findIndexedColumnInFunctionArgs(ge, s); col != "" {
				r.addFunctionAdvice(name, col, ge)
			}
			// Still recurse into the arguments — a nested function/calc on an
			// indexed column should ALSO be flagged (e.g. `ABS(id + 1)`).
			for _, part := range ge.AllGeneral_element_part() {
				if arg := part.Function_argument(); arg != nil {
					r.walkWhere(arg, s, combined)
				}
			}
			return
		}
		// Not a function call — the column ref case is reached via parent
		// nodes (concatenation / unary). Fall through.
	}
	// Standard_function: built-in (SUBSTR / COUNT / CAST / TRIM / …).
	if sf, ok := n.(*parser.Standard_functionContext); ok {
		name := r.extractStandardFunctionName(sf)
		// Special-case CAST: it's a transparent wrapper for the indexed-col
		// check but we still also flag CAST-on-indexed as a function.
		if col := r.findIndexedColumnInStandardFunction(sf, s); col != "" && name != "" {
			r.addFunctionAdvice(name, col, sf)
		}
		for i := 0; i < sf.GetChildCount(); i++ {
			r.walkWhere(sf.GetChild(i), s, combined)
		}
		return
	}
	// Concatenation: arithmetic ASTERISK / SOLIDUS / PLUS_SIGN / MINUS_SIGN.
	if c, ok := n.(*parser.ConcatenationContext); ok {
		if isArithmeticConcat(c) {
			if col := r.unwrapAndFindIndexedColumn(c.Concatenation(0), s); col != "" {
				r.addCalculationAdvice(col, c)
			} else if col := r.unwrapAndFindIndexedColumn(c.Concatenation(1), s); col != "" {
				r.addCalculationAdvice(col, c)
			}
		}
		for i := 0; i < c.GetChildCount(); i++ {
			r.walkWhere(c.GetChild(i), s, combined)
		}
		return
	}
	// Unary minus → calculation (NOT PRIOR, NOT CONNECT_BY_ROOT).
	if u, ok := n.(*parser.Unary_expressionContext); ok {
		if u.MINUS_SIGN() != nil {
			if inner := u.Unary_expression(); inner != nil {
				if col := r.unwrapAndFindIndexedColumn(inner, s); col != "" {
					r.addCalculationAdvice(col, u)
				}
			}
		}
		for i := 0; i < u.GetChildCount(); i++ {
			r.walkWhere(u.GetChild(i), s, combined)
		}
		return
	}

	for i := 0; i < n.GetChildCount(); i++ {
		r.walkWhere(n.GetChild(i), s, combined)
	}
}

// isArithmeticConcat reports whether the concatenation is an arithmetic op.
func isArithmeticConcat(c *parser.ConcatenationContext) bool {
	if c == nil || c.GetOp() == nil {
		return false
	}
	switch c.GetOp().GetTokenType() {
	case parser.PlSqlParserASTERISK,
		parser.PlSqlParserSOLIDUS,
		parser.PlSqlParserPLUS_SIGN,
		parser.PlSqlParserMINUS_SIGN:
		return true
	}
	return false
}

// unwrapAndFindIndexedColumn peels passthrough wrappers (parens, CAST) and,
// if the inner expression is an indexed column reference, returns its
// lower-cased column name. Returns "" otherwise.
func (r *WhereDisallowFunctionsAndCalculationsRule) unwrapAndFindIndexedColumn(n antlr.Tree, s scopeIndexed) string {
	for n != nil {
		switch c := n.(type) {
		case *parser.ConcatenationContext:
			// passthrough: only if not itself an arithmetic op
			if isArithmeticConcat(c) {
				return ""
			}
			if m := c.Model_expression(); m != nil {
				n = m
				continue
			}
			return ""
		case *parser.Model_expressionContext:
			if u := c.Unary_expression(); u != nil {
				n = u
				continue
			}
			return ""
		case *parser.Unary_expressionContext:
			if c.MINUS_SIGN() != nil || c.PRIOR() != nil || c.CONNECT_BY_ROOT() != nil || c.NEW() != nil || c.DISTINCT() != nil || c.ALL() != nil {
				return ""
			}
			if at := c.Atom(); at != nil {
				n = at
				continue
			}
			if sf := c.Standard_function(); sf != nil {
				if sfCtx, ok := sf.(*parser.Standard_functionContext); ok {
					// CAST unwrap: `CAST(col AS TYPE)` — the inside is transparent for column-ref extraction.
					return r.unwrapCastColumn(sfCtx, s)
				}
				return ""
			}
			return ""
		case *parser.AtomContext:
			// LEFT_PAREN expressions RIGHT_PAREN — passthrough if single.
			if ge := c.General_element(); ge != nil {
				qualifier, column := r.extractColumnRefFromGeneralElement(ge)
				if column != "" && r.isIndexedColumn(qualifier, column, s) {
					return r.normalizeIdent(column)
				}
				return ""
			}
			if te := c.Table_element(); te != nil {
				qualifier, column := r.extractColumnRefFromTableElement(te)
				if column != "" && r.isIndexedColumn(qualifier, column, s) {
					return r.normalizeIdent(column)
				}
				return ""
			}
			// (expressions) — unwrap if single expression.
			if exprs := c.Expressions(); exprs != nil {
				all := exprs.AllExpression()
				if len(all) == 1 {
					n = all[0]
					continue
				}
			}
			return ""
		case *parser.ExpressionContext:
			if le := c.Logical_expression(); le != nil {
				n = le
				continue
			}
			return ""
		case *parser.Logical_expressionContext:
			if ul := c.Unary_logical_expression(); ul != nil {
				n = ul
				continue
			}
			return ""
		case *parser.Unary_logical_expressionContext:
			if me := c.Multiset_expression(); me != nil {
				n = me
				continue
			}
			return ""
		case *parser.Multiset_expressionContext:
			if rel := c.Relational_expression(); rel != nil {
				n = rel
				continue
			}
			return ""
		case *parser.Relational_expressionContext:
			rels := c.AllRelational_expression()
			if len(rels) == 1 {
				n = rels[0]
				continue
			}
			if comp := c.Compound_expression(); comp != nil {
				n = comp
				continue
			}
			return ""
		case *parser.Compound_expressionContext:
			cats := c.AllConcatenation()
			if len(cats) >= 1 {
				n = cats[0]
				continue
			}
			return ""
		default:
			return ""
		}
	}
	return ""
}

// unwrapCastColumn returns the indexed-column name inside a `CAST(col AS T)`
// expression, or "" if the inside isn't a bare indexed column.
func (r *WhereDisallowFunctionsAndCalculationsRule) unwrapCastColumn(sf *parser.Standard_functionContext, s scopeIndexed) string {
	if sf == nil {
		return ""
	}
	of := sf.Other_function()
	if of == nil {
		return ""
	}
	ofCtx, ok := of.(*parser.Other_functionContext)
	if !ok {
		return ""
	}
	if ofCtx.CAST() == nil && ofCtx.XMLCAST() == nil {
		return ""
	}
	cats := ofCtx.AllConcatenation()
	if len(cats) == 0 {
		return ""
	}
	return r.unwrapAndFindIndexedColumn(cats[0], s)
}

// findIndexedColumnInFunctionArgs searches a function-call's argument list for
// an indexed column reference (unwrapping CAST / parens). Arithmetic is NOT
// crossed so `ABS(id + 1)` reports as a calculation via the inner walker.
func (r *WhereDisallowFunctionsAndCalculationsRule) findIndexedColumnInFunctionArgs(ge *parser.General_elementContext, s scopeIndexed) string {
	for _, part := range ge.AllGeneral_element_part() {
		arg := part.Function_argument()
		if arg == nil {
			continue
		}
		argCtx, ok := arg.(*parser.Function_argumentContext)
		if !ok {
			continue
		}
		for _, a := range argCtx.AllArgument() {
			if a == nil {
				continue
			}
			aCtx, ok := a.(*parser.ArgumentContext)
			if !ok {
				continue
			}
			if expr := aCtx.Expression(); expr != nil {
				if col := r.unwrapAndFindIndexedColumn(expr, s); col != "" {
					return col
				}
			}
		}
	}
	return ""
}

// findIndexedColumnInStandardFunction searches a built-in function's argument
// subtree for an indexed column reference.
func (r *WhereDisallowFunctionsAndCalculationsRule) findIndexedColumnInStandardFunction(sf *parser.Standard_functionContext, s scopeIndexed) string {
	if sf == nil {
		return ""
	}
	// Walk the function's children (NOT the root node itself — we want to
	// descend into its arguments, stopping only at further function calls /
	// arithmetic).
	for i := 0; i < sf.GetChildCount(); i++ {
		if col := r.findIndexedColumnInChildren(sf.GetChild(i), s); col != "" {
			return col
		}
	}
	return ""
}

// findIndexedColumnInChildren recursively searches for an indexed column
// reference without crossing into arithmetic concatenations (those spawn the
// calculation path) or further function calls.
func (r *WhereDisallowFunctionsAndCalculationsRule) findIndexedColumnInChildren(n antlr.Tree, s scopeIndexed) string {
	if n == nil {
		return ""
	}
	switch c := n.(type) {
	case *parser.ConcatenationContext:
		if isArithmeticConcat(c) {
			return ""
		}
	case *parser.General_elementContext:
		if _, isCall := r.functionCallName(c); isCall {
			return ""
		}
		qualifier, column := r.extractColumnRefFromGeneralElement(c)
		if column != "" && r.isIndexedColumn(qualifier, column, s) {
			return r.normalizeIdent(column)
		}
		return ""
	case *parser.Table_elementContext:
		qualifier, column := r.extractColumnRefFromTableElement(c)
		if column != "" && r.isIndexedColumn(qualifier, column, s) {
			return r.normalizeIdent(column)
		}
		return ""
	case *parser.Standard_functionContext:
		return ""
	default:
	}
	for i := 0; i < n.GetChildCount(); i++ {
		if col := r.findIndexedColumnInChildren(n.GetChild(i), s); col != "" {
			return col
		}
	}
	return ""
}

// ---- Subquery re-entry ---------------------------------------------------

// checkSubqueryInScope re-enters the recursive check on a SubqueryContext,
// threading the given outer scope for correlated refs.
func (r *WhereDisallowFunctionsAndCalculationsRule) checkSubqueryInScope(sub parser.ISubqueryContext, outer tablesByAlias) {
	if sub == nil {
		return
	}
	// Set-op: first element plus each subquery_operation_part.
	if base := sub.Subquery_basic_elements(); base != nil {
		r.checkSubqueryBasicElements(base, outer)
	}
	for _, op := range sub.AllSubquery_operation_part() {
		if op == nil {
			continue
		}
		if base := op.Subquery_basic_elements(); base != nil {
			r.checkSubqueryBasicElements(base, outer)
		}
	}
}

func (r *WhereDisallowFunctionsAndCalculationsRule) checkSubqueryBasicElements(be parser.ISubquery_basic_elementsContext, outer tablesByAlias) {
	if be == nil {
		return
	}
	if qb := be.Query_block(); qb != nil {
		r.checkQueryBlock(qb, outer)
		return
	}
	if inner := be.Subquery(); inner != nil {
		r.checkSubqueryInScope(inner, outer)
	}
}

// recurseIntoNestedSubqueries scans the non-WHERE parts of a query block for
// SubqueryContext nodes and dispatches each with the given outer scope.
func (r *WhereDisallowFunctionsAndCalculationsRule) recurseIntoNestedSubqueries(qb parser.IQuery_blockContext, outer tablesByAlias) {
	if qb == nil {
		return
	}
	// SELECT list
	if sl := qb.Selected_list(); sl != nil {
		r.recurseIntoSubqueriesIn(sl, outer)
	}
	// FROM (derived tables). Subqueries inside JOIN ON predicates are already
	// walked by checkQueryBlock via checkExpressionAsWhere; skip them here to
	// avoid double-flagging.
	if from := qb.From_clause(); from != nil {
		r.recurseIntoFromSubqueries(from, outer)
	}
	// GROUP BY + HAVING (HAVING is never itself flagged; nested subqueries
	// inside are still recursed).
	if gb := qb.Group_by_clause(); gb != nil {
		r.recurseIntoSubqueriesIn(gb, outer)
	}
	if ob := qb.Order_by_clause(); ob != nil {
		r.recurseIntoSubqueriesIn(ob, outer)
	}
}

func (r *WhereDisallowFunctionsAndCalculationsRule) recurseIntoSubqueriesIn(n antlr.Tree, outer tablesByAlias) {
	if n == nil {
		return
	}
	if sub, ok := n.(*parser.SubqueryContext); ok {
		r.checkSubqueryInScope(sub, outer)
		return
	}
	for i := 0; i < n.GetChildCount(); i++ {
		r.recurseIntoSubqueriesIn(n.GetChild(i), outer)
	}
}

// recurseIntoSubqueriesSkipWhere is like recurseIntoSubqueriesIn but skips
// Where_clauseContext subtrees. Callers that already dispatched the WHERE
// through checkWhereClause use this variant to avoid double-walking subqueries
// inside that WHERE (checkWhereClause → walkWhere re-enters SubqueryContext on
// its own).
func (r *WhereDisallowFunctionsAndCalculationsRule) recurseIntoSubqueriesSkipWhere(n antlr.Tree, outer tablesByAlias) {
	if n == nil {
		return
	}
	if _, ok := n.(*parser.Where_clauseContext); ok {
		return
	}
	if sub, ok := n.(*parser.SubqueryContext); ok {
		r.checkSubqueryInScope(sub, outer)
		return
	}
	for i := 0; i < n.GetChildCount(); i++ {
		r.recurseIntoSubqueriesSkipWhere(n.GetChild(i), outer)
	}
}

// recurseIntoFromSubqueries walks a FROM-clause subtree for derived-table
// subqueries but skips Join_on_part subtrees — JOIN ON conditions are
// dispatched separately by checkQueryBlock and would otherwise cause the
// same inner subquery to be checked twice.
func (r *WhereDisallowFunctionsAndCalculationsRule) recurseIntoFromSubqueries(n antlr.Tree, outer tablesByAlias) {
	if n == nil {
		return
	}
	if _, ok := n.(*parser.Join_on_partContext); ok {
		return
	}
	if sub, ok := n.(*parser.SubqueryContext); ok {
		r.checkSubqueryInScope(sub, outer)
		return
	}
	for i := 0; i < n.GetChildCount(); i++ {
		r.recurseIntoFromSubqueries(n.GetChild(i), outer)
	}
}

// ---- Top-level dispatchers -----------------------------------------------

// checkSelectOnly dispatches the top-level subquery of a Select_only_statement.
func (r *WhereDisallowFunctionsAndCalculationsRule) checkSelectOnly(sos parser.ISelect_only_statementContext) {
	if sos == nil {
		return
	}
	if sub := sos.Subquery(); sub != nil {
		r.checkSubqueryInScope(sub, nil)
	}
}

// onSelectStmt handles SELECT (including CTE / UNION / MINUS / INTERSECT).
func (r *WhereDisallowFunctionsAndCalculationsRule) onSelectStmt(ctx *parser.Select_statementContext) {
	if ctx == nil {
		return
	}
	r.checkSelectOnly(ctx.Select_only_statement())
}

// tablesUnresolved reports whether any concrete tableRef lacks metadata.
// Empty (placeholder) refs are skipped; use emptyIsUnresolved=true when an empty
// ref should itself count as unresolved (e.g. MERGE derived source).
func (r *WhereDisallowFunctionsAndCalculationsRule) tablesUnresolved(tables tablesByAlias, emptyIsUnresolved bool) bool {
	for _, ref := range tables {
		if ref == (tableRef{}) {
			if emptyIsUnresolved {
				return true
			}
			continue
		}
		if r.tableMetadata(ref) == nil {
			return true
		}
	}
	return false
}

// onUpdateStmt handles UPDATE ... SET ... WHERE ...
func (r *WhereDisallowFunctionsAndCalculationsRule) onUpdateStmt(ctx *parser.Update_statementContext) {
	if ctx == nil {
		return
	}
	tables := make(tablesByAlias)
	if gtr := ctx.General_table_ref(); gtr != nil {
		r.collectTableNamesFromGeneralTableRef(gtr, tables)
		r.dispatchTargetSubquery(gtr)
	}
	unresolved := r.tablesUnresolved(tables, false)
	if where := ctx.Where_clause(); where != nil {
		r.checkWhereClause(where, tables, nil, unresolved)
	}
	// Recurse into subqueries in SET expressions and WHERE subqueries (latter handled via walkWhere).
	if sc := ctx.Update_set_clause(); sc != nil {
		r.recurseIntoSubqueriesIn(sc, tables)
	}
	if ret := ctx.Static_returning_clause(); ret != nil {
		r.recurseIntoSubqueriesIn(ret, tables)
	}
}

// onDeleteStmt handles DELETE [FROM] … WHERE …
func (r *WhereDisallowFunctionsAndCalculationsRule) onDeleteStmt(ctx *parser.Delete_statementContext) {
	if ctx == nil {
		return
	}
	tables := make(tablesByAlias)
	if gtr := ctx.General_table_ref(); gtr != nil {
		r.collectTableNamesFromGeneralTableRef(gtr, tables)
		r.dispatchTargetSubquery(gtr)
	}
	unresolved := r.tablesUnresolved(tables, false)
	if where := ctx.Where_clause(); where != nil {
		r.checkWhereClause(where, tables, nil, unresolved)
	}
	if ret := ctx.Static_returning_clause(); ret != nil {
		r.recurseIntoSubqueriesIn(ret, tables)
	}
}

// dispatchTargetSubquery dispatches the inner SELECT when the UPDATE / DELETE
// target is a derived table (`UPDATE (SELECT … WHERE …) SET …`). The outer
// collector records only an opaque alias placeholder; the inner WHERE must
// still be checked independently.
func (r *WhereDisallowFunctionsAndCalculationsRule) dispatchTargetSubquery(gtr parser.IGeneral_table_refContext) {
	if gtr == nil {
		return
	}
	dte := gtr.Dml_table_expression_clause()
	if dte == nil {
		return
	}
	sel := dte.Select_statement()
	if sel == nil {
		return
	}
	r.checkSelectOnly(sel.Select_only_statement())
}

// onInsertStmt handles INSERT INTO … SELECT / VALUES …
func (r *WhereDisallowFunctionsAndCalculationsRule) onInsertStmt(ctx *parser.Insert_statementContext) {
	if ctx == nil {
		return
	}
	if sti := ctx.Single_table_insert(); sti != nil {
		if sel := sti.Select_statement(); sel != nil {
			r.checkSelectOnly(sel.Select_only_statement())
		}
		// VALUES (subquery) — walk only the values and returning trees, not
		// the full sti (which would double-walk a sibling SELECT source).
		if v := sti.Values_clause(); v != nil {
			r.recurseIntoSubqueriesIn(v, nil)
		}
		if ret := sti.Static_returning_clause(); ret != nil {
			r.recurseIntoSubqueriesIn(ret, nil)
		}
	}
	if mti := ctx.Multi_table_insert(); mti != nil {
		if sel := mti.Select_statement(); sel != nil {
			r.checkSelectOnly(sel.Select_only_statement())
		}
		// INSERT ALL WHEN <cond> THEN INTO ... — the WHEN predicate is a
		// per-row filter over the driving SELECT's columns. Build a scope from
		// that SELECT's FROM clause so indexed-column refs resolve.
		drivingTables := make(tablesByAlias)
		if sel := mti.Select_statement(); sel != nil {
			if sos := sel.Select_only_statement(); sos != nil {
				if sub := sos.Subquery(); sub != nil {
					if be := sub.Subquery_basic_elements(); be != nil {
						if qb := be.Query_block(); qb != nil {
							drivingTables = r.collectTableNamesFromFrom(qb.From_clause())
						}
					}
				}
			}
		}
		drivingUnresolved := r.tablesUnresolved(drivingTables, false)
		if cic := mti.Conditional_insert_clause(); cic != nil {
			for _, wp := range cic.AllConditional_insert_when_part() {
				if cond := wp.Condition(); cond != nil {
					r.checkExpressionAsWhere(cond, drivingTables, nil, drivingUnresolved)
				}
				// INTO branches may carry VALUES subqueries — recurse into
				// them. The WHEN Condition itself is already handled above
				// via checkExpressionAsWhere (which re-enters subqueries);
				// skip it here to avoid double-walking.
				for _, elt := range wp.AllMulti_table_element() {
					r.recurseIntoSubqueriesIn(elt, nil)
				}
			}
			if ep := cic.Conditional_insert_else_part(); ep != nil {
				r.recurseIntoSubqueriesIn(ep, nil)
			}
		}
		// Walk non-conditional multi-table-element children too (for INSERT
		// ALL without WHEN branches). Skip the driving SELECT (already
		// handled) and the Conditional_insert_clause (handled above, where
		// we split WHEN Condition from INTO branches).
		for i := 0; i < mti.GetChildCount(); i++ {
			child := mti.GetChild(i)
			if _, isSel := child.(parser.ISelect_statementContext); isSel {
				continue
			}
			if _, isCIC := child.(parser.IConditional_insert_clauseContext); isCIC {
				continue
			}
			r.recurseIntoSubqueriesIn(child, nil)
		}
	}
}

// onCreateTable handles CREATE TABLE ... AS subquery (CTAS).
func (r *WhereDisallowFunctionsAndCalculationsRule) onCreateTable(ctx *parser.Create_tableContext) {
	if ctx == nil {
		return
	}
	if sub := findFirstSubquery(ctx); sub != nil {
		r.checkSubqueryInScope(sub, nil)
	}
}

// onCreateView handles CREATE [OR REPLACE] VIEW v AS SELECT …
func (r *WhereDisallowFunctionsAndCalculationsRule) onCreateView(ctx *parser.Create_viewContext) {
	if ctx == nil {
		return
	}
	r.checkSelectOnly(ctx.Select_only_statement())
}

// onCreateMaterializedView handles CREATE MATERIALIZED VIEW mv AS SELECT …
func (r *WhereDisallowFunctionsAndCalculationsRule) onCreateMaterializedView(ctx *parser.Create_materialized_viewContext) {
	if ctx == nil {
		return
	}
	r.checkSelectOnly(ctx.Select_only_statement())
}

// onMergeStmt handles MERGE INTO target USING source ON cond WHEN …
func (r *WhereDisallowFunctionsAndCalculationsRule) onMergeStmt(ctx *parser.Merge_statementContext) {
	if ctx == nil {
		return
	}
	tables := make(tablesByAlias)
	for _, tv := range ctx.AllSelected_tableview() {
		r.collectTableNamesFromSelectedTableview(tv, tables)
		// MERGE source may be a subquery (`USING (SELECT … WHERE …)`). The
		// outer collector records only an opaque alias placeholder; run the
		// rule on the inner SELECT independently so its own WHERE is checked.
		if sel := tv.Select_statement(); sel != nil {
			r.checkSelectOnly(sel.Select_only_statement())
		}
	}
	unresolved := r.tablesUnresolved(tables, true)
	if cond := ctx.Condition(); cond != nil {
		r.checkExpressionAsWhere(cond, tables, nil, unresolved)
	}
	if mu := ctx.Merge_update_clause(); mu != nil {
		if where := mu.Where_clause(); where != nil {
			r.checkWhereClause(where, tables, nil, unresolved)
		}
		if mud := mu.Merge_update_delete_part(); mud != nil {
			if where := mud.Where_clause(); where != nil {
				r.checkWhereClause(where, tables, nil, unresolved)
			}
		}
		// SET col = (subquery) — recurse into any nested subqueries, skipping
		// the Where_clause subtrees already dispatched above.
		r.recurseIntoSubqueriesSkipWhere(mu, tables)
	}
	if mi := ctx.Merge_insert_clause(); mi != nil {
		if where := mi.Where_clause(); where != nil {
			r.checkWhereClause(where, tables, nil, unresolved)
		}
		// INSERT VALUES (subquery) or column-list subqueries — same recursion,
		// skipping the Where_clause subtree already dispatched above.
		r.recurseIntoSubqueriesSkipWhere(mi, tables)
	}
}

// collectTableNamesFromGeneralTableRef records the UPDATE / DELETE target.
func (r *WhereDisallowFunctionsAndCalculationsRule) collectTableNamesFromGeneralTableRef(gtr parser.IGeneral_table_refContext, tables tablesByAlias) {
	if gtr == nil {
		return
	}
	alias := ""
	if ta := gtr.Table_alias(); ta != nil {
		alias = r.tableAliasText(ta)
	}
	if dte := gtr.Dml_table_expression_clause(); dte != nil {
		r.collectTableNamesFromDml(dte, alias, tables)
	}
}

// collectTableNamesFromSelectedTableview records a MERGE target / source.
func (r *WhereDisallowFunctionsAndCalculationsRule) collectTableNamesFromSelectedTableview(tv parser.ISelected_tableviewContext, tables tablesByAlias) {
	if tv == nil {
		return
	}
	alias := ""
	if ta := tv.Table_alias(); ta != nil {
		alias = r.tableAliasText(ta)
	}
	if tvn := tv.Tableview_name(); tvn != nil {
		schema, name := r.splitTableviewName(tvn)
		key := alias
		if key == "" {
			key = name
		}
		normKey := r.normalizeIdent(key)
		// CTE shadowing — see collectTableNamesFromDml.
		if schema == "" && r.cteVisible(name) {
			tables[normKey] = tableRef{}
			return
		}
		ref := tableRef{schema: schema, name: name}
		if existing, ok := tables[normKey]; ok && (existing.schema != ref.schema || existing.name != ref.name) {
			tables[normKey] = tableRef{}
		} else {
			tables[normKey] = ref
		}
		return
	}
	if tv.Select_statement() != nil {
		key := r.normalizeIdent(alias)
		if key != "" {
			if _, ok := tables[key]; !ok {
				tables[key] = tableRef{}
			}
		}
	}
}

// findFirstSubquery does a DFS for the first SubqueryContext child in the tree.
func findFirstSubquery(n antlr.Tree) parser.ISubqueryContext {
	if n == nil {
		return nil
	}
	for i := 0; i < n.GetChildCount(); i++ {
		child := n.GetChild(i)
		if sub, ok := child.(*parser.SubqueryContext); ok {
			return sub
		}
		if sub := findFirstSubquery(child); sub != nil {
			return sub
		}
	}
	return nil
}

// ---- Advice builders ------------------------------------------------------

func (r *WhereDisallowFunctionsAndCalculationsRule) addFunctionAdvice(funcName, col string, ctx antlr.ParserRuleContext) {
	r.adviceList = append(r.adviceList, &storepb.Advice{
		Status:        r.level,
		Code:          code.StatementDisallowFunctionsAndCalculations.Int32(),
		Title:         r.title,
		Content:       fmt.Sprintf("Function %q is applied to indexed column %q in the WHERE clause, which prevents index usage", funcName, col),
		StartPosition: common.ConvertANTLRLineToPosition(r.baseLine + ctx.GetStart().GetLine()),
	})
}

func (r *WhereDisallowFunctionsAndCalculationsRule) addCalculationAdvice(col string, ctx antlr.ParserRuleContext) {
	r.adviceList = append(r.adviceList, &storepb.Advice{
		Status:        r.level,
		Code:          code.StatementDisallowFunctionsAndCalculations.Int32(),
		Title:         r.title,
		Content:       fmt.Sprintf("Calculation is applied to indexed column %q in the WHERE clause, which prevents index usage", col),
		StartPosition: common.ConvertANTLRLineToPosition(r.baseLine + ctx.GetStart().GetLine()),
	})
}
