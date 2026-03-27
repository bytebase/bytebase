package pg

import (
	"strings"

	"github.com/pkg/errors"

	"github.com/bytebase/omni/pg/ast"
	"github.com/bytebase/omni/pg/catalog"
	plpgsqlast "github.com/bytebase/omni/pg/plpgsql/ast"
	plpgsqlparser "github.com/bytebase/omni/pg/plpgsql/parser"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

// analyzeFunctionBody analyzes a user-defined function body and returns per-column
// source column sets. Results are cached by function OID on the extractor.
// A nil value in the cache means analysis is in progress (recursion sentinel).
// On parse errors, it degrades gracefully by returning empty sets.
func (e *omniQuerySpanExtractor) analyzeFunctionBody(proc *catalog.UserProc) []base.SourceColumnSet {
	// Check cache.
	if cached, ok := e.funcBodyCache[proc.OID]; ok {
		if cached == nil {
			// In-progress sentinel — recursive call. Return empty sets.
			return makeEmptySets(proc)
		}
		return cached
	}

	// Store nil sentinel to detect recursion.
	e.funcBodyCache[proc.OID] = nil

	// If the function body was stubbed during catalog loading (to avoid type
	// validation errors), use the original body from metadata instead.
	body := proc.Body
	if origBody, ok := e.funcOrigDefs[strings.ToLower(proc.Name)]; ok && origBody != "" {
		body = origBody
	}

	var result []base.SourceColumnSet
	var err error

	switch strings.ToLower(proc.Language) {
	case "plpgsql":
		result, err = e.analyzePLpgSQLBody(proc, body)
	case "sql":
		result, err = e.analyzeSQLBody(body)
	default:
		// Unsupported language (C, internal, etc.) — unknown lineage.
		result = makeEmptySets(proc)
	}

	if err != nil {
		// On error, cache empty sets so we don't retry.
		e.funcBodyCache[proc.OID] = makeEmptySets(proc)
		return e.funcBodyCache[proc.OID]
	}

	e.funcBodyCache[proc.OID] = result
	return result
}

func makeEmptySets(proc *catalog.UserProc) []base.SourceColumnSet {
	n := countOutputParams(proc)
	if n == 0 {
		n = 1 // scalar return
	}
	sets := make([]base.SourceColumnSet, n)
	for i := range sets {
		sets[i] = make(base.SourceColumnSet)
	}
	return sets
}

func countOutputParams(proc *catalog.UserProc) int {
	count := 0
	for _, mode := range proc.ArgModes {
		if mode == 'o' || mode == 't' || mode == 'b' {
			count++
		}
	}
	return count
}

func (e *omniQuerySpanExtractor) analyzeSQLBody(body string) ([]base.SourceColumnSet, error) {
	if body == "" {
		return nil, nil
	}

	stmts, err := ParsePg(body)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to parse SQL function body")
	}

	// Find the last SELECT statement (SQL functions return the result of the last statement).
	for i := len(stmts) - 1; i >= 0; i-- {
		selStmt, ok := stmts[i].AST.(*ast.SelectStmt)
		if !ok {
			continue
		}
		query, err := e.cat.AnalyzeSelectStmt(selStmt)
		if err != nil {
			// Fall back to parse-tree extraction when analysis fails
			// (e.g., nested function calls, parameter references).
			fallbackResults := e.extractFallbackColumns(selStmt)
			if len(fallbackResults) > 0 {
				result := make([]base.SourceColumnSet, len(fallbackResults))
				for j, fr := range fallbackResults {
					result[j] = fr.SourceColumns
				}
				return result, nil
			}
			return nil, errors.Wrapf(err, "failed to analyze SQL function body")
		}
		return e.extractFuncLineage(query), nil
	}

	return nil, nil
}

func (e *omniQuerySpanExtractor) extractFuncLineage(q *catalog.Query) []base.SourceColumnSet {
	var result []base.SourceColumnSet
	for _, te := range q.TargetList {
		if te.ResJunk {
			continue
		}
		colSet := make(base.SourceColumnSet)
		e.walkExpr(q, te.Expr, colSet)
		result = append(result, colSet)
	}
	return result
}

// variableScope tracks PL/pgSQL variable source columns with nested scoping.
type variableScope struct {
	vars   map[string]base.SourceColumnSet
	parent *variableScope
}

func newVariableScope(parent *variableScope) *variableScope {
	return &variableScope{
		vars:   make(map[string]base.SourceColumnSet),
		parent: parent,
	}
}

func (s *variableScope) set(name string, sources base.SourceColumnSet) {
	s.vars[strings.ToLower(name)] = sources
}

func (s *variableScope) get(name string) (base.SourceColumnSet, bool) {
	lower := strings.ToLower(name)
	for scope := s; scope != nil; scope = scope.parent {
		if v, ok := scope.vars[lower]; ok {
			return v, true
		}
	}
	return nil, false
}

func (s *variableScope) merge(name string, sources base.SourceColumnSet) {
	lower := strings.ToLower(name)
	existing, ok := s.get(lower)
	if !ok {
		existing = make(base.SourceColumnSet)
	}
	merged := make(base.SourceColumnSet)
	for k, v := range existing {
		merged[k] = v
	}
	for k, v := range sources {
		merged[k] = v
	}
	s.vars[lower] = merged
}

// plpgsqlAnalyzer holds state for analyzing a single PL/pgSQL function body.
type plpgsqlAnalyzer struct {
	extractor *omniQuerySpanExtractor
	scope     *variableScope
	// returnResults collects per-column source sets from all RETURN QUERY paths.
	returnResults [][]base.SourceColumnSet
	// cteMap holds CTE definitions for resolving column refs through CTEs.
	cteMap map[string]*ast.SelectStmt
}

func (e *omniQuerySpanExtractor) analyzePLpgSQLBody(proc *catalog.UserProc, body string) ([]base.SourceColumnSet, error) {
	if body == "" {
		return makeEmptySets(proc), nil
	}

	block, err := plpgsqlparser.Parse(body)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to parse PL/pgSQL function body")
	}

	analyzer := &plpgsqlAnalyzer{
		extractor: e,
		scope:     newVariableScope(nil),
	}

	// Add function input parameters to scope so they can be substituted in
	// embedded SQL. Output/table params ('o', 't') are NOT added because their
	// names often collide with column names in the function body's queries
	// (e.g., RETURNS TABLE(a int, b int) where a, b are also column names).
	for i, name := range proc.ArgNames {
		if name == "" {
			continue
		}
		if i < len(proc.ArgModes) {
			mode := proc.ArgModes[i]
			if mode == 'o' || mode == 't' {
				continue
			}
		}
		analyzer.scope.set(name, make(base.SourceColumnSet))
	}

	analyzer.walkBlock(block)

	nCols := countOutputParams(proc)
	if nCols == 0 {
		nCols = 1
	}

	return analyzer.mergeReturnResults(nCols), nil
}

func (a *plpgsqlAnalyzer) walkBlock(block *plpgsqlast.PLBlock) {
	// Push scope for declarations.
	a.scope = newVariableScope(a.scope)
	defer func() { a.scope = a.scope.parent }()

	// Process declarations.
	for _, decl := range block.Declarations {
		switch d := decl.(type) {
		case *plpgsqlast.PLDeclare:
			a.scope.set(d.Name, make(base.SourceColumnSet))
		case *plpgsqlast.PLCursorDecl:
			a.scope.set(d.Name, make(base.SourceColumnSet))
		case *plpgsqlast.PLAliasDecl:
			// Alias references another variable — copy its sources.
			if sources, ok := a.scope.get(d.RefName); ok {
				a.scope.set(d.Name, sources)
			} else {
				a.scope.set(d.Name, make(base.SourceColumnSet))
			}
		default:
		}
	}

	// Process body statements.
	for _, stmt := range block.Body {
		a.walkStmt(stmt)
	}

	// Process exception handlers.
	for _, ex := range block.Exceptions {
		if ew, ok := ex.(*plpgsqlast.PLExceptionWhen); ok {
			for _, stmt := range ew.Body {
				a.walkStmt(stmt)
			}
		}
	}
}

func (a *plpgsqlAnalyzer) walkStmt(node plpgsqlast.Node) {
	switch stmt := node.(type) {
	case *plpgsqlast.PLAssign:
		a.handleAssign(stmt)
	case *plpgsqlast.PLReturnQuery:
		a.handleReturnQuery(stmt)
	case *plpgsqlast.PLReturn:
		a.handleReturn(stmt)
	case *plpgsqlast.PLReturnNext:
		a.handleReturnNext(stmt)
	case *plpgsqlast.PLExecSQL:
		a.handleExecSQL(stmt)
	case *plpgsqlast.PLIf:
		a.handleIf(stmt)
	case *plpgsqlast.PLCase:
		a.handleCase(stmt)
	case *plpgsqlast.PLLoop:
		for _, s := range stmt.Body {
			a.walkStmt(s)
		}
	case *plpgsqlast.PLWhile:
		for _, s := range stmt.Body {
			a.walkStmt(s)
		}
	case *plpgsqlast.PLForI:
		for _, s := range stmt.Body {
			a.walkStmt(s)
		}
	case *plpgsqlast.PLForS:
		a.handleForS(stmt)
	case *plpgsqlast.PLForC:
		for _, s := range stmt.Body {
			a.walkStmt(s)
		}
	case *plpgsqlast.PLForDynS:
		for _, s := range stmt.Body {
			a.walkStmt(s)
		}
	case *plpgsqlast.PLForEachA:
		for _, s := range stmt.Body {
			a.walkStmt(s)
		}
	case *plpgsqlast.PLBlock:
		a.walkBlock(stmt)
	// Skip nodes that don't affect lineage:
	// PLDynExecute, PLRaise, PLNull, PLCommit, PLRollback, PLPerform,
	// PLCall, PLOpen, PLFetch, PLClose, PLGetDiag, PLAssert, PLExit
	default:
	}
}

func (a *plpgsqlAnalyzer) handleAssign(stmt *plpgsqlast.PLAssign) {
	varName := extractBaseVarName(stmt.Target)
	sources := a.analyzeEmbeddedExpr(stmt.Expr)
	a.scope.merge(varName, sources)
}

func (a *plpgsqlAnalyzer) handleReturnQuery(stmt *plpgsqlast.PLReturnQuery) {
	if stmt.Query == "" {
		return
	}
	results := a.analyzeEmbeddedSQL(stmt.Query)
	if len(results) > 0 {
		a.returnResults = append(a.returnResults, results)
	}
}

func (a *plpgsqlAnalyzer) handleReturn(stmt *plpgsqlast.PLReturn) {
	if stmt.Expr == "" {
		return
	}
	sources := a.analyzeEmbeddedExpr(stmt.Expr)
	a.returnResults = append(a.returnResults, []base.SourceColumnSet{sources})
}

func (a *plpgsqlAnalyzer) handleReturnNext(stmt *plpgsqlast.PLReturnNext) {
	if stmt.Expr == "" {
		return
	}
	sources := a.analyzeEmbeddedExpr(stmt.Expr)
	a.returnResults = append(a.returnResults, []base.SourceColumnSet{sources})
}

func (a *plpgsqlAnalyzer) handleExecSQL(stmt *plpgsqlast.PLExecSQL) {
	if len(stmt.Into) == 0 {
		return
	}
	results := a.analyzeEmbeddedSQL(stmt.SQLText)
	for i, varName := range stmt.Into {
		if i < len(results) {
			a.scope.merge(varName, results[i])
		}
	}
}

func (a *plpgsqlAnalyzer) handleForS(stmt *plpgsqlast.PLForS) {
	results := a.analyzeEmbeddedSQL(stmt.Query)
	merged := make(base.SourceColumnSet)
	for _, colSet := range results {
		for k, v := range colSet {
			merged[k] = v
		}
	}
	a.scope.merge(stmt.Var, merged)
	for _, s := range stmt.Body {
		a.walkStmt(s)
	}
}

func (a *plpgsqlAnalyzer) handleIf(stmt *plpgsqlast.PLIf) {
	for _, s := range stmt.ThenBody {
		a.walkStmt(s)
	}
	for _, elsif := range stmt.ElsIfs {
		if ei, ok := elsif.(*plpgsqlast.PLElsIf); ok {
			for _, s := range ei.Body {
				a.walkStmt(s)
			}
		}
	}
	for _, s := range stmt.ElseBody {
		a.walkStmt(s)
	}
}

func (a *plpgsqlAnalyzer) handleCase(stmt *plpgsqlast.PLCase) {
	for _, when := range stmt.Whens {
		if cw, ok := when.(*plpgsqlast.PLCaseWhen); ok {
			for _, s := range cw.Body {
				a.walkStmt(s)
			}
		}
	}
	for _, s := range stmt.ElseBody {
		a.walkStmt(s)
	}
}

func (a *plpgsqlAnalyzer) analyzeEmbeddedSQL(sql string) []base.SourceColumnSet {
	substituted := a.substituteVariables(sql)

	// Always collect access tables from embedded SQL for the top-level QuerySpan,
	// regardless of whether full analysis succeeds.
	a.collectAccessTablesFromSQL(substituted)

	stmts, err := ParsePg(substituted)
	if err != nil {
		return nil
	}
	if len(stmts) == 0 {
		return nil
	}

	selStmt, ok := stmts[0].AST.(*ast.SelectStmt)
	if !ok {
		return nil
	}

	query, err := a.extractor.cat.AnalyzeSelectStmt(selStmt)
	if err != nil {
		// When full analysis fails (e.g., type mismatches with aggregate functions),
		// fall back to extracting column references from the parse tree and resolving
		// them against known tables in the catalog.
		return a.fallbackExtractLineage(sql, selStmt)
	}

	// Also collect from the analyzed query for precise column-level tracking
	// (WHERE clause → predicate columns).
	a.collectQueryPredicateColumns(query)

	results := a.extractor.extractFuncLineage(query)

	// Map back variable references.
	origSelectItems := extractSelectItemNames(sql)
	for i, colSet := range results {
		if len(colSet) == 0 && i < len(origSelectItems) {
			varName := strings.TrimSpace(origSelectItems[i])
			if sources, ok := a.scope.get(varName); ok {
				results[i] = sources
			}
		}
	}

	return results
}

func extractSelectItemNames(sql string) []string {
	upper := strings.ToUpper(strings.TrimSpace(sql))
	idx := strings.Index(upper, "SELECT")
	if idx < 0 {
		return nil
	}
	rest := sql[idx+6:]

	fromIdx := -1
	depth := 0
	upperRest := strings.ToUpper(rest)
	for i := 0; i < len(upperRest); i++ {
		switch upperRest[i] {
		case '(':
			depth++
		case ')':
			depth--
		default:
			if depth == 0 && i+4 <= len(upperRest) && upperRest[i:i+4] == "FROM" {
				if (i == 0 || !isWordChar(rest[i-1])) && (i+4 >= len(rest) || !isWordChar(rest[i+4])) {
					fromIdx = i
				}
			}
		}
		if fromIdx >= 0 {
			break
		}
	}

	targetList := rest
	if fromIdx >= 0 {
		targetList = rest[:fromIdx]
	}

	var items []string
	depth = 0
	start := 0
	for i := 0; i < len(targetList); i++ {
		switch targetList[i] {
		case '(':
			depth++
		case ')':
			depth--
		case ',':
			if depth == 0 {
				items = append(items, strings.TrimSpace(targetList[start:i]))
				start = i + 1
			}
		default:
		}
	}
	items = append(items, strings.TrimSpace(targetList[start:]))

	return items
}

func (a *plpgsqlAnalyzer) analyzeEmbeddedExpr(expr string) base.SourceColumnSet {
	results := a.analyzeEmbeddedSQL("SELECT " + expr)
	merged := make(base.SourceColumnSet)
	for _, colSet := range results {
		for k, v := range colSet {
			merged[k] = v
		}
	}
	if len(merged) == 0 {
		varName := strings.TrimSpace(expr)
		if sources, ok := a.scope.get(varName); ok {
			for k, v := range sources {
				merged[k] = v
			}
		}
	}
	return merged
}

func (a *plpgsqlAnalyzer) substituteVariables(sql string) string {
	stmts, err := ParsePg(sql)
	if err == nil && len(stmts) > 0 {
		if selStmt, ok := stmts[0].AST.(*ast.SelectStmt); ok {
			if _, err := a.extractor.cat.AnalyzeSelectStmt(selStmt); err == nil {
				return sql
			}
		}
	}

	result := sql
	for scope := a.scope; scope != nil; scope = scope.parent {
		for varName := range scope.vars {
			result = replaceWholeWord(result, varName, "NULL::text")
		}
	}

	return result
}

func replaceWholeWord(s, old, replacement string) string {
	lower := strings.ToLower(s)
	oldLower := strings.ToLower(old)
	var result strings.Builder
	i := 0
	for i < len(lower) {
		idx := strings.Index(lower[i:], oldLower)
		if idx < 0 {
			result.WriteString(s[i:])
			break
		}
		pos := i + idx
		if pos > 0 && isWordChar(s[pos-1]) {
			result.WriteString(s[i : pos+len(old)])
			i = pos + len(old)
			continue
		}
		end := pos + len(old)
		if end < len(s) && isWordChar(s[end]) {
			result.WriteString(s[i : pos+len(old)])
			i = pos + len(old)
			continue
		}
		result.WriteString(s[i:pos])
		result.WriteString(replacement)
		i = end
	}
	return result.String()
}

func isWordChar(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') || (b >= '0' && b <= '9') || b == '_'
}

func extractBaseVarName(target string) string {
	for i, c := range target {
		if c == '.' || c == '[' {
			return target[:i]
		}
	}
	return target
}

// collectAccessTablesFromSQL extracts all table/column references from embedded SQL
// and adds them to the extractor's funcSourceColumns. This works even when
// AnalyzeSelectStmt fails, because it uses the AST-level ExtractAccessTables.
func (a *plpgsqlAnalyzer) collectAccessTablesFromSQL(sql string) {
	accessTables, err := ExtractAccessTables(sql, ExtractAccessTablesOption{
		DefaultDatabase:        a.extractor.defaultDatabase,
		DefaultSchema:          a.extractor.searchPath[0],
		GetDatabaseMetadata:    a.extractor.gCtx.GetDatabaseMetadataFunc,
		Ctx:                    a.extractor.ctx,
		InstanceID:             a.extractor.gCtx.InstanceID,
		SkipMetadataValidation: false,
	})
	if err != nil {
		return
	}
	for _, resource := range accessTables {
		a.extractor.funcSourceColumns[resource] = true
	}
}

// collectQueryPredicateColumns walks WHERE/HAVING clauses in an analyzed query
// and collects predicate columns (column-level) into funcPredicateColumns.
// Note: funcSourceColumns only stores table-level access (via collectAccessTablesFromSQL),
// so we don't add column-level entries to it here.
func (a *plpgsqlAnalyzer) collectQueryPredicateColumns(q *catalog.Query) {
	if q == nil {
		return
	}

	// Handle set operations recursively.
	if q.SetOp != catalog.SetOpNone {
		a.collectQueryPredicateColumns(q.LArg)
		a.collectQueryPredicateColumns(q.RArg)
		return
	}

	// Collect from WHERE clause → predicate columns only.
	if q.JoinTree != nil && q.JoinTree.Quals != nil {
		a.extractor.walkExpr(q, q.JoinTree.Quals, a.extractor.funcPredicateColumns)
	}

	// Collect from HAVING clause → predicate columns only.
	if q.HavingQual != nil {
		a.extractor.walkExpr(q, q.HavingQual, a.extractor.funcPredicateColumns)
	}

	// Recurse into CTEs.
	for _, cte := range q.CTEList {
		if cte.Query != nil {
			a.collectQueryPredicateColumns(cte.Query)
		}
	}

	// Recurse into subqueries in FROM clause.
	for _, rte := range q.RangeTable {
		if rte.Kind == catalog.RTESubquery && rte.Subquery != nil {
			a.collectQueryPredicateColumns(rte.Subquery)
		}
	}
}

func (a *plpgsqlAnalyzer) mergeReturnResults(nCols int) []base.SourceColumnSet {
	result := make([]base.SourceColumnSet, nCols)
	for i := range result {
		result[i] = make(base.SourceColumnSet)
	}

	for _, returnCols := range a.returnResults {
		for i := 0; i < nCols && i < len(returnCols); i++ {
			for k, v := range returnCols[i] {
				result[i][k] = v
			}
		}
	}

	return result
}

// fallbackExtractLineage is called when AnalyzeSelectStmt fails for embedded SQL
// (e.g., due to type mismatches with aggregate functions). It uses the parse tree
// to extract column references and resolves them against known tables and variables.
func (a *plpgsqlAnalyzer) fallbackExtractLineage(origSQL string, selStmt *ast.SelectStmt) []base.SourceColumnSet {
	if selStmt == nil || selStmt.TargetList == nil {
		return nil
	}

	// Collect table references from the FROM clause.
	fromTables := a.collectFromTables(selStmt)

	var results []base.SourceColumnSet
	origItems := extractSelectItemNames(origSQL)

	for i, item := range selStmt.TargetList.Items {
		rt, ok := item.(*ast.ResTarget)
		if !ok {
			results = append(results, make(base.SourceColumnSet))
			continue
		}

		colSet := make(base.SourceColumnSet)

		// Try to extract column references from the expression.
		a.extractColumnRefsFromExpr(rt.Val, fromTables, colSet)

		// If still empty, try the original item name as a variable reference.
		if len(colSet) == 0 && i < len(origItems) {
			varName := strings.TrimSpace(origItems[i])
			// Strip wrapping parentheses for subquery expressions.
			varName = strings.TrimPrefix(strings.TrimSuffix(varName, ")"), "(")
			varName = strings.TrimSpace(varName)
			if sources, ok := a.scope.get(varName); ok {
				for k, v := range sources {
					colSet[k] = v
				}
			}
		}

		results = append(results, colSet)
	}

	// Extract column refs from WHERE clause for predicate tracking.
	if selStmt.WhereClause != nil {
		whereColSet := make(base.SourceColumnSet)
		a.extractColumnRefsFromExpr(selStmt.WhereClause, fromTables, whereColSet)
		for k, v := range whereColSet {
			a.extractor.funcPredicateColumns[k] = v
		}
	}

	return results
}

// collectFromTables collects table references from the FROM clause and maps
// table names/aliases to their schema-qualified names.
func (a *plpgsqlAnalyzer) collectFromTables(selStmt *ast.SelectStmt) map[string]base.ColumnResource {
	tables := make(map[string]base.ColumnResource)
	if selStmt.FromClause == nil {
		return tables
	}

	for _, item := range selStmt.FromClause.Items {
		a.collectTablesFromNode(item, tables)
	}
	return tables
}

func (a *plpgsqlAnalyzer) collectTablesFromNode(node ast.Node, tables map[string]base.ColumnResource) {
	switch v := node.(type) {
	case *ast.RangeVar:
		schema := "public"
		if v.Schemaname != "" {
			schema = v.Schemaname
		}
		resource := base.ColumnResource{
			Database: a.extractor.defaultDatabase,
			Schema:   schema,
			Table:    v.Relname,
		}
		tables[strings.ToLower(v.Relname)] = resource
		if v.Alias != nil && v.Alias.Aliasname != "" {
			tables[strings.ToLower(v.Alias.Aliasname)] = resource
		}
	case *ast.RangeSubselect:
		// For subquery aliases like (SELECT * FROM t) result, collect inner tables
		// and map them under the alias so column refs can be resolved.
		if v.Subquery != nil {
			if subSel, ok := v.Subquery.(*ast.SelectStmt); ok {
				innerTables := a.collectFromTables(subSel)
				// Add inner tables to the outer fromTables.
				for k, res := range innerTables {
					if _, exists := tables[k]; !exists {
						tables[k] = res
					}
				}
				// If the subquery has an alias, also map all inner tables'
				// columns as accessible through the alias.
				if v.Alias != nil && v.Alias.Aliasname != "" {
					alias := strings.ToLower(v.Alias.Aliasname)
					// Use the first inner table as the primary source for column lookup.
					for _, res := range innerTables {
						tables[alias] = res
						break
					}
				}
			}
		}
	case *ast.JoinExpr:
		if v.Larg != nil {
			a.collectTablesFromNode(v.Larg, tables)
		}
		if v.Rarg != nil {
			a.collectTablesFromNode(v.Rarg, tables)
		}
	default:
	}
}

// extractColumnRefsFromExpr recursively walks an AST expression and resolves
// column references against known tables.
func (a *plpgsqlAnalyzer) extractColumnRefsFromExpr(node ast.Node, fromTables map[string]base.ColumnResource, result base.SourceColumnSet) {
	if node == nil {
		return
	}

	switch v := node.(type) {
	case *ast.ColumnRef:
		a.resolveColumnRef(v, fromTables, result)
	case *ast.FuncCall:
		if v.Args != nil {
			for _, arg := range v.Args.Items {
				a.extractColumnRefsFromExpr(arg, fromTables, result)
			}
		}
		// FILTER (WHERE ...) clause on aggregates → predicate columns.
		if v.AggFilter != nil {
			filterColSet := make(base.SourceColumnSet)
			a.extractColumnRefsFromExpr(v.AggFilter, fromTables, filterColSet)
			for k, val := range filterColSet {
				a.extractor.funcPredicateColumns[k] = val
			}
		}
	case *ast.A_Expr:
		a.extractColumnRefsFromExpr(v.Lexpr, fromTables, result)
		a.extractColumnRefsFromExpr(v.Rexpr, fromTables, result)
	case *ast.SubLink:
		if v.Subselect != nil {
			if subSel, ok := v.Subselect.(*ast.SelectStmt); ok {
				subTables := a.collectFromTables(subSel)
				for k, val := range fromTables {
					if _, exists := subTables[k]; !exists {
						subTables[k] = val
					}
				}
				if subSel.TargetList != nil {
					for _, item := range subSel.TargetList.Items {
						if rt, ok := item.(*ast.ResTarget); ok {
							a.extractColumnRefsFromExpr(rt.Val, subTables, result)
						}
					}
				}
				// Also collect from subquery WHERE clause for predicate tracking.
				if subSel.WhereClause != nil {
					whereColSet := make(base.SourceColumnSet)
					a.extractColumnRefsFromExpr(subSel.WhereClause, subTables, whereColSet)
					for k, val := range whereColSet {
						a.extractor.funcPredicateColumns[k] = val
					}
				}
			}
		}
	case *ast.TypeCast:
		a.extractColumnRefsFromExpr(v.Arg, fromTables, result)
	case *ast.BoolExpr:
		if v.Args != nil {
			for _, arg := range v.Args.Items {
				a.extractColumnRefsFromExpr(arg, fromTables, result)
			}
		}
	case *ast.CaseExpr:
		a.extractColumnRefsFromExpr(v.Arg, fromTables, result)
		if v.Args != nil {
			for _, w := range v.Args.Items {
				a.extractColumnRefsFromExpr(w, fromTables, result)
			}
		}
		a.extractColumnRefsFromExpr(v.Defresult, fromTables, result)
	case *ast.CaseWhen:
		a.extractColumnRefsFromExpr(v.Expr, fromTables, result)
		a.extractColumnRefsFromExpr(v.Result, fromTables, result)
	case *ast.CoalesceExpr:
		if v.Args != nil {
			for _, arg := range v.Args.Items {
				a.extractColumnRefsFromExpr(arg, fromTables, result)
			}
		}
	case *ast.NullTest:
		a.extractColumnRefsFromExpr(v.Arg, fromTables, result)
	case *ast.BooleanTest:
		a.extractColumnRefsFromExpr(v.Arg, fromTables, result)
	case *ast.A_Indirection:
		// Array subscript or field selection — the source is the base expression.
		a.extractColumnRefsFromExpr(v.Arg, fromTables, result)
		if v.Indirection != nil {
			for _, ind := range v.Indirection.Items {
				a.extractColumnRefsFromExpr(ind, fromTables, result)
			}
		}
	case *ast.JsonObjectConstructor:
		if v.Exprs != nil {
			for _, item := range v.Exprs.Items {
				if kv, ok := item.(*ast.JsonKeyValue); ok {
					if kv.Value != nil {
						a.extractColumnRefsFromExpr(kv.Value.RawExpr, fromTables, result)
					}
				}
			}
		}
	case *ast.JsonArrayConstructor:
		if v.Exprs != nil {
			for _, item := range v.Exprs.Items {
				if jve, ok := item.(*ast.JsonValueExpr); ok {
					a.extractColumnRefsFromExpr(jve.RawExpr, fromTables, result)
				}
			}
		}
	case *ast.JsonValueExpr:
		a.extractColumnRefsFromExpr(v.RawExpr, fromTables, result)
	case *ast.List:
		for _, item := range v.Items {
			a.extractColumnRefsFromExpr(item, fromTables, result)
		}
	default:
	}
}

// resolveColumnRef resolves a ColumnRef to a source column using known tables.
// If the table is a CTE, it traces through the CTE to find real source columns.
func (a *plpgsqlAnalyzer) resolveColumnRef(cr *ast.ColumnRef, fromTables map[string]base.ColumnResource, result base.SourceColumnSet) {
	if cr == nil || cr.Fields == nil || len(cr.Fields.Items) == 0 {
		return
	}

	var parts []string
	for _, f := range cr.Fields.Items {
		if s, ok := f.(*ast.String); ok {
			parts = append(parts, s.Str)
		}
	}

	switch len(parts) {
	case 1:
		colName := parts[0]
		if _, ok := a.scope.get(colName); ok {
			return
		}
		// Check if it's a column name in any FROM table.
		for _, resource := range fromTables {
			rel := a.extractor.cat.GetRelation(resource.Schema, resource.Table)
			if rel != nil && relationHasColumn(rel, colName) {
				result[base.ColumnResource{
					Database: resource.Database,
					Schema:   resource.Schema,
					Table:    resource.Table,
					Column:   colName,
				}] = true
				return
			}
		}
		// Check if it's a whole-row reference (table alias used as value, e.g., to_jsonb(t1)).
		if resource, ok := fromTables[strings.ToLower(colName)]; ok {
			rel := a.extractor.cat.GetRelation(resource.Schema, resource.Table)
			if rel != nil {
				for _, col := range rel.Columns {
					result[base.ColumnResource{
						Database: resource.Database,
						Schema:   resource.Schema,
						Table:    resource.Table,
						Column:   col.Name,
					}] = true
				}
				return
			}
		}
	case 2:
		tableName := strings.ToLower(parts[0])
		colName := parts[1]
		if resource, ok := fromTables[tableName]; ok {
			// Check if this table is actually a CTE — trace through to real sources.
			// The CTE name is the original table name (resource.Table), not the alias.
			if a.cteMap != nil {
				if cteSel, isCTE := a.cteMap[strings.ToLower(resource.Table)]; isCTE {
					a.resolveThroughCTE(cteSel, colName, fromTables, result)
					return
				}
			}
			result[base.ColumnResource{
				Database: resource.Database,
				Schema:   resource.Schema,
				Table:    resource.Table,
				Column:   colName,
			}] = true
		}
	default:
	}
}

// resolveThroughCTE traces a column reference through a CTE to find real source columns.
func (a *plpgsqlAnalyzer) resolveThroughCTE(cteSel *ast.SelectStmt, colName string, _ map[string]base.ColumnResource, result base.SourceColumnSet) {
	if cteSel == nil || cteSel.TargetList == nil {
		return
	}

	// Build FROM tables for the CTE's query.
	cteTables := a.collectFromTables(cteSel)
	// Include parent CTEs for nested CTE references.
	if a.cteMap != nil {
		for name := range a.cteMap {
			if _, exists := cteTables[name]; !exists {
				cteTables[name] = base.ColumnResource{
					Database: a.extractor.defaultDatabase,
					Schema:   "public",
					Table:    name,
				}
			}
		}
	}

	// Find the target entry matching the column name.
	for _, item := range cteSel.TargetList.Items {
		rt, ok := item.(*ast.ResTarget)
		if !ok {
			continue
		}
		// Match by alias first, then by expression name.
		rtName := rt.Name
		if rtName == "" {
			rtName = figureResTargetName(rt)
		}
		if !strings.EqualFold(rtName, colName) {
			continue
		}
		// Found matching column — extract source refs from its expression.
		a.extractColumnRefsFromExpr(rt.Val, cteTables, result)

		// Also collect predicate columns from the CTE's WHERE clause.
		if cteSel.WhereClause != nil {
			whereColSet := make(base.SourceColumnSet)
			a.extractColumnRefsFromExpr(cteSel.WhereClause, cteTables, whereColSet)
			for k, v := range whereColSet {
				a.extractor.funcPredicateColumns[k] = v
			}
		}
		return
	}
}

// collectCTEDefinitions extracts CTE names and their SelectStmt definitions from a query.
func collectCTEDefinitions(selStmt *ast.SelectStmt) map[string]*ast.SelectStmt {
	if selStmt == nil || selStmt.WithClause == nil {
		return nil
	}

	cteMap := make(map[string]*ast.SelectStmt)
	for _, item := range selStmt.WithClause.Ctes.Items {
		cte, ok := item.(*ast.CommonTableExpr)
		if !ok {
			continue
		}
		if sel, ok := cte.Ctequery.(*ast.SelectStmt); ok {
			cteMap[strings.ToLower(cte.Ctename)] = sel
		}
	}
	return cteMap
}

// relationHasColumn checks if a catalog relation has a column with the given name.
func relationHasColumn(rel *catalog.Relation, colName string) bool {
	lowerName := strings.ToLower(colName)
	for _, col := range rel.Columns {
		if strings.ToLower(col.Name) == lowerName {
			return true
		}
	}
	return false
}
