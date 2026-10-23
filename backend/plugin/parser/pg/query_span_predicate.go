package pg

import (
	"strings"

	"github.com/bytebase/omni/pg/ast"
	"github.com/bytebase/omni/pg/catalog"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

// collectPredicateColumns records the columns that decide which rows q and
// every query nested in it return: those read by WHERE, JOIN ON, HAVING, and
// aggregate FILTER conditions, LIMIT and OFFSET, and the arguments of
// set-returning functions. The database evaluates conditions on unmasked values, so these columns
// decide which rows come back even when their values are masked.
func (e *omniQuerySpanExtractor) collectPredicateColumns(q *catalog.Query) {
	e.collectPredicateColumnsInStack([]*catalog.Query{q}, make(map[*catalog.Query]bool))
}

// collectStatementPredicateColumns records the predicate columns of sel,
// analyzed as query, that reads accesses.
func (e *omniQuerySpanExtractor) collectStatementPredicateColumns(query *catalog.Query, sel *ast.SelectStmt, accesses base.SourceColumnSet) {
	e.collectPredicateColumns(query)
	// The analyzer leaves a set operation's LIMIT and OFFSET out of its tree,
	// so they are resolved by name.
	r := newPredicateNameResolver(sel, e.namedFunctionReturnsSet)
	limited := false
	ast.Inspect(sel, func(n ast.Node) bool {
		if v, ok := n.(*ast.SelectStmt); ok && v.Op != ast.SETOP_NONE {
			for _, limit := range []ast.Node{v.LimitCount, v.LimitOffset} {
				if limit != nil {
					r.addReferences(limit)
					limited = true
				}
			}
		}
		return true
	})
	if limited {
		e.recordPredicateNames(r, accesses)
	}
}

func (e *omniQuerySpanExtractor) collectPredicateColumnsInStack(queryStack []*catalog.Query, visited map[*catalog.Query]bool) {
	q := currentQuery(queryStack)
	if q == nil || visited[q] {
		return
	}
	visited[q] = true

	for _, cte := range q.CTEList {
		if cte != nil {
			e.collectPredicateColumnsInStack(appendQuery(queryStack, cte.Query), visited)
		}
	}
	e.collectConditionColumns(queryStack, q.LimitCount, visited)
	e.collectConditionColumns(queryStack, q.LimitOffset, visited)
	if q.SetOp != catalog.SetOpNone {
		// INTERSECT and EXCEPT keep a row by whether the other branch has it,
		// so their branches' output columns decide which rows come back.
		filters := q.SetOp == catalog.SetOpIntersect || q.SetOp == catalog.SetOpIntersectAll ||
			q.SetOp == catalog.SetOpExcept || q.SetOp == catalog.SetOpExceptAll
		for _, branch := range []*catalog.Query{q.LArg, q.RArg} {
			if filters && branch != nil {
				for _, te := range branch.TargetList {
					if te != nil && !te.ResJunk {
						e.walkExprWithVisited(appendQuery(queryStack, branch), te.Expr, e.predicateColumns, make(map[queryExprKey]bool))
					}
				}
			}
			e.collectPredicateColumnsInStack(appendQuery(queryStack, branch), visited)
		}
		return
	}
	for _, rte := range q.RangeTable {
		if rte == nil {
			continue
		}
		if rte.Kind == catalog.RTESubquery {
			e.collectPredicateColumnsInStack(appendQuery(queryStack, rte.Subquery), visited)
		}
		for _, expr := range rte.FuncExprs {
			// A function in FROM returns the rows its arguments and the
			// conditions in its body ask for.
			call, ok := expr.(*catalog.FuncCallExpr)
			if !ok {
				e.collectConditionColumns(queryStack, expr, visited)
				continue
			}
			for _, arg := range call.Args {
				e.collectConditionColumns(queryStack, arg, visited)
			}
			if proc := e.cat.GetUserProcByOID(call.FuncOID); proc != nil {
				e.analyzeFunctionBody(proc)
			}
		}
	}
	if q.JoinTree != nil {
		e.collectConditionColumns(queryStack, q.JoinTree.Quals, visited)
		for _, node := range q.JoinTree.FromList {
			e.collectJoinConditionColumns(queryStack, node, visited)
		}
	}
	e.collectConditionColumns(queryStack, q.HavingQual, visited)
	for _, te := range q.TargetList {
		if te != nil {
			e.collectNestedPredicateColumns(queryStack, te.Expr, visited)
		}
	}
}

func (e *omniQuerySpanExtractor) collectJoinConditionColumns(queryStack []*catalog.Query, node catalog.JoinNode, visited map[*catalog.Query]bool) {
	join, ok := node.(*catalog.JoinExprNode)
	if !ok || join == nil {
		return
	}
	// The analyzer turns USING and NATURAL joins into equality quals.
	e.collectConditionColumns(queryStack, join.Quals, visited)
	e.collectJoinConditionColumns(queryStack, join.Left, visited)
	e.collectJoinConditionColumns(queryStack, join.Right, visited)
}

// collectConditionColumns records every column condition reads, including the
// output columns of its subqueries, and the conditions nested in it.
func (e *omniQuerySpanExtractor) collectConditionColumns(queryStack []*catalog.Query, condition catalog.AnalyzedExpr, visited map[*catalog.Query]bool) {
	if condition == nil {
		return
	}
	e.walkExprWithVisited(queryStack, condition, e.predicateColumns, make(map[queryExprKey]bool))
	e.collectNestedPredicateColumns(queryStack, condition, visited)
}

// collectNestedPredicateColumns records the conditions nested in expr: the
// FILTER of an aggregate or window function, the arguments of a set-returning
// function, which decide how many rows it returns, and the conditions of its
// subqueries.
func (e *omniQuerySpanExtractor) collectNestedPredicateColumns(queryStack []*catalog.Query, expr catalog.AnalyzedExpr, visited map[*catalog.Query]bool) {
	catalog.WalkAnalyzedExpr(expr, func(n catalog.AnalyzedExpr) bool {
		var inputs []catalog.AnalyzedExpr
		switch v := n.(type) {
		case *catalog.SubLinkExpr:
			if v.SubQuery != nil {
				e.collectPredicateColumnsInStack(appendQuery(queryStack, v.SubQuery), visited)
			}
		case *catalog.AggExpr:
			inputs = []catalog.AnalyzedExpr{v.AggFilter}
		case *catalog.WindowFuncExpr:
			inputs = []catalog.AnalyzedExpr{v.AggFilter}
		case *catalog.FuncCallExpr:
			if e.returnsSet(v.FuncOID) {
				inputs = v.Args
			}
		default:
		}
		for _, input := range inputs {
			if input != nil {
				e.walkExprWithVisited(queryStack, input, e.predicateColumns, make(map[queryExprKey]bool))
			}
		}
		return true
	})
}

// namedFunctionReturnsSet reports whether a function of that name is known to
// return a set.
func (e *omniQuerySpanExtractor) namedFunctionReturnsSet(name string) bool {
	if proc := e.lookupUserProcByName(name); proc != nil {
		return proc.RetSet
	}
	for _, proc := range e.cat.LookupProcByName(strings.ToLower(name)) {
		if proc.RetSet {
			return true
		}
	}
	return false
}

// returnsSet reports whether the function is known to return a set. An
// unknown function, analyzed opaquely, is not.
func (e *omniQuerySpanExtractor) returnsSet(oid uint32) bool {
	if proc := e.cat.GetUserProcByOID(oid); proc != nil {
		return proc.RetSet
	}
	if proc := e.cat.LookupProcByOID(oid); proc != nil {
		return proc.RetSet
	}
	return false
}

// collectFallbackPredicateColumns stands in for collectPredicateColumns when
// analysis fails. Without scopes it over-approximates by name, so that an
// analysis failure never lets a condition on a masked column through.
func (e *omniQuerySpanExtractor) collectFallbackPredicateColumns(stmt ast.Node, accesses base.SourceColumnSet) {
	r := newPredicateNameResolver(stmt, e.namedFunctionReturnsSet)
	for _, name := range r.usingNames {
		r.addName(name)
	}
	for _, condition := range r.conditions {
		r.addReferences(condition)
	}
	// The joined columns are unknown without scopes, so a NATURAL JOIN reads
	// every column of both sides.
	for _, join := range r.naturalJoins {
		r.addJoinedRelation(join.Larg)
		r.addJoinedRelation(join.Rarg)
	}
	for _, setOperation := range r.filteringSetOperations {
		r.addSetOperationOutputs(setOperation)
	}
	e.recordPredicateNames(r, accesses)
}

func (e *omniQuerySpanExtractor) recordPredicateNames(r *predicateNameResolver, accesses base.SourceColumnSet) {
	for resource := range accesses {
		rel := e.cat.GetRelation(resource.Schema, resource.Table)
		if rel == nil {
			continue
		}
		table := strings.ToLower(resource.Table)
		for _, column := range rel.Columns {
			name := strings.ToLower(column.Name)
			if r.wholeRows[table] || r.anyTable[name] || r.ofTable[table][name] {
				e.predicateColumns[base.ColumnResource{
					Database: resource.Database,
					Schema:   resource.Schema,
					Table:    resource.Table,
					Column:   column.Name,
				}] = true
			}
		}
	}
}

// predicateNameResolver resolves the column references of conditions by name.
// A reference qualified by a table or its alias matches that table's column;
// any other reference matches the same-named column of every table, and an
// output name of a subquery or CTE also reads the columns that produce it.
type predicateNameResolver struct {
	// The WHERE, JOIN ON, HAVING, and FILTER conditions, LIMIT and OFFSET,
	// arguments of FROM and set-returning functions, USING column names,
	// NATURAL joins, and INTERSECT and EXCEPT operations of the statement, at
	// any depth.
	conditions             []ast.Node
	usingNames             []string
	naturalJoins           []*ast.JoinExpr
	filteringSetOperations []*ast.SelectStmt
	// projections maps an output name to the expressions that produce it. A
	// subquery or CTE name maps to its whole query, so a whole-row reference
	// to it reads every column.
	projections map[string][]ast.Node
	// relations maps a name or alias in FROM to the relations it may name; an
	// alias reused across scopes names several.
	relations map[string]map[string]bool
	// derived holds the names of subqueries and CTEs.
	derived map[string]bool

	anyTable  map[string]bool
	ofTable   map[string]map[string]bool
	wholeRows map[string]bool
}

func newPredicateNameResolver(stmt ast.Node, returnsSet func(name string) bool) *predicateNameResolver {
	r := &predicateNameResolver{
		projections: make(map[string][]ast.Node),
		relations:   make(map[string]map[string]bool),
		derived:     make(map[string]bool),
		anyTable:    make(map[string]bool),
		ofTable:     make(map[string]map[string]bool),
		wholeRows:   make(map[string]bool),
	}
	project := func(name string, nodes ...ast.Node) {
		if name != "" {
			r.projections[strings.ToLower(name)] = append(r.projections[strings.ToLower(name)], nodes...)
		}
	}
	ast.Inspect(stmt, func(n ast.Node) bool {
		switch v := n.(type) {
		case *ast.SelectStmt:
			r.conditions = append(r.conditions, v.WhereClause, v.HavingClause, v.LimitCount, v.LimitOffset)
			if v.Op == ast.SETOP_INTERSECT || v.Op == ast.SETOP_EXCEPT {
				r.filteringSetOperations = append(r.filteringSetOperations, v)
			}
			if v.TargetList != nil {
				for _, item := range v.TargetList.Items {
					if rt, ok := item.(*ast.ResTarget); ok {
						name := rt.Name
						if name == "" {
							name = figureResTargetName(rt)
						}
						project(name, rt.Val)
					}
				}
			}
		case *ast.JoinExpr:
			r.conditions = append(r.conditions, v.Quals)
			r.usingNames = append(r.usingNames, astStrings(v.UsingClause)...)
			if v.IsNatural {
				r.naturalJoins = append(r.naturalJoins, v)
			}
		case *ast.FuncCall:
			r.conditions = append(r.conditions, v.AggFilter)
			if names := astStrings(v.Funcname); len(names) > 0 && returnsSet(names[len(names)-1]) {
				r.conditions = append(r.conditions, v.Args)
			}
		case *ast.RangeFunction:
			// A function in FROM returns the rows its arguments ask for.
			r.conditions = append(r.conditions, v.Functions)
		case *ast.CommonTableExpr:
			r.derived[strings.ToLower(v.Ctename)] = true
			project(v.Ctename, v.Ctequery)
			for i, name := range astStrings(v.Aliascolnames) {
				project(name, outputsAt(v.Ctequery, i)...)
			}
		case *ast.RangeSubselect:
			if v.Alias != nil {
				r.derived[strings.ToLower(v.Alias.Aliasname)] = true
				project(v.Alias.Aliasname, v.Subquery)
				for i, name := range astStrings(v.Alias.Colnames) {
					project(name, outputsAt(v.Subquery, i)...)
				}
			}
		case *ast.RangeVar:
			relation := strings.ToLower(v.Relname)
			r.addRelationName(relation, relation)
			if v.Alias != nil {
				r.addRelationName(v.Alias.Aliasname, relation)
			}
		default:
		}
		return true
	})
	return r
}

func (r *predicateNameResolver) addRelationName(name, relation string) {
	name = strings.ToLower(name)
	if name == "" {
		return
	}
	if r.relations[name] == nil {
		r.relations[name] = make(map[string]bool)
	}
	r.relations[name][relation] = true
}

// relationsNamed returns the tables name may refer to, and the subqueries or
// CTEs it may refer to instead.
func (r *predicateNameResolver) relationsNamed(name string) (map[string]bool, []string) {
	name = strings.ToLower(name)
	tables := make(map[string]bool)
	var derived []string
	if r.derived[name] {
		derived = append(derived, name)
	}
	for relation := range r.relations[name] {
		if r.derived[relation] {
			derived = append(derived, relation)
		} else {
			tables[relation] = true
		}
	}
	return tables, derived
}

// addSetOperationOutputs adds the outputs of every branch of sel, which an
// INTERSECT or EXCEPT compares.
func (r *predicateNameResolver) addSetOperationOutputs(sel *ast.SelectStmt) {
	if sel == nil {
		return
	}
	if sel.Op != ast.SETOP_NONE {
		r.addSetOperationOutputs(sel.Larg)
		r.addSetOperationOutputs(sel.Rarg)
		return
	}
	if sel.TargetList == nil {
		return
	}
	for _, item := range sel.TargetList.Items {
		rt, ok := item.(*ast.ResTarget)
		if !ok {
			continue
		}
		if !isStarTarget(rt) {
			r.addReferences(rt.Val)
			continue
		}
		if sel.FromClause != nil {
			for _, from := range sel.FromClause.Items {
				r.addJoinedRelation(from)
			}
		}
	}
}

func (r *predicateNameResolver) addJoinedRelation(node ast.Node) {
	switch v := node.(type) {
	case *ast.RangeVar:
		if v.Alias != nil && v.Alias.Aliasname != "" {
			r.addWholeRow(v.Alias.Aliasname)
		} else {
			r.addWholeRow(v.Relname)
		}
	case *ast.RangeSubselect:
		r.addReferences(v.Subquery)
	case *ast.JoinExpr:
		r.addJoinedRelation(v.Larg)
		r.addJoinedRelation(v.Rarg)
	default:
	}
}

func (r *predicateNameResolver) addReferences(node ast.Node) {
	ast.Inspect(node, func(n ast.Node) bool {
		if ref, ok := n.(*ast.ColumnRef); ok {
			r.addReference(ref)
		}
		return true
	})
}

func (r *predicateNameResolver) addReference(ref *ast.ColumnRef) {
	if ref.Fields == nil {
		return
	}
	var fields []string
	star := false
	for _, field := range ref.Fields.Items {
		switch v := field.(type) {
		case *ast.String:
			fields = append(fields, v.Str)
		case *ast.A_Star:
			star = true
		default:
		}
	}
	if len(fields) == 0 {
		return
	}
	last := fields[len(fields)-1]
	// t.* reads every column of t; a bare t may be a column or a whole-row
	// reference.
	if star || len(fields) == 1 {
		r.addWholeRow(last)
		return
	}
	tables, derived := r.relationsNamed(fields[len(fields)-2])
	for table := range tables {
		if r.ofTable[table] == nil {
			r.ofTable[table] = make(map[string]bool)
		}
		r.ofTable[table][strings.ToLower(last)] = true
	}
	if len(derived) > 0 || len(tables) == 0 {
		r.addName(last)
	}
}

func (r *predicateNameResolver) addWholeRow(name string) {
	tables, derived := r.relationsNamed(name)
	for table := range tables {
		r.wholeRows[table] = true
	}
	for _, relation := range derived {
		r.addName(relation)
	}
	r.addName(name)
}

func (r *predicateNameResolver) addName(name string) {
	name = strings.ToLower(name)
	if name == "" || r.anyTable[name] {
		return
	}
	r.anyTable[name] = true
	for _, projection := range r.projections[name] {
		r.addReferences(projection)
	}
}

// outputsAt returns the expressions that produce the i-th output column of
// query, or query itself when they cannot be told apart.
func outputsAt(query ast.Node, i int) []ast.Node {
	sel, ok := query.(*ast.SelectStmt)
	if !ok || sel == nil {
		return []ast.Node{query}
	}
	if sel.Op != ast.SETOP_NONE {
		return append(outputsAt(sel.Larg, i), outputsAt(sel.Rarg, i)...)
	}
	if sel.TargetList == nil || i >= len(sel.TargetList.Items) {
		return []ast.Node{query}
	}
	for _, item := range sel.TargetList.Items[:i+1] {
		if rt, ok := item.(*ast.ResTarget); !ok || isStarTarget(rt) {
			return []ast.Node{query}
		}
	}
	return []ast.Node{sel.TargetList.Items[i].(*ast.ResTarget).Val}
}

func astStrings(list *ast.List) []string {
	if list == nil {
		return nil
	}
	var result []string
	for _, item := range list.Items {
		if s, ok := item.(*ast.String); ok {
			result = append(result, s.Str)
		}
	}
	return result
}
