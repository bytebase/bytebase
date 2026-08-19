package pg

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/bytebase/omni/pg/catalog"
)

// relationRef identifies a relation by schema and name, both lowercased.
//
// Case is folded on both sides of every comparison. A lookup that misses on
// letter case would drop a genuine read from the unresolved-columns check,
// which is the fail-open direction; folding can only keep an extra access,
// which at worst refuses a query.
//
// An empty schema is a wildcard that matches any schema for that name. It is
// used only when the analyzer bound a relation whose schema could not be
// recovered from the catalog.
type relationRef struct {
	schema string
	table  string
}

// relationSet is a set of relations a query reads.
type relationSet map[relationRef]bool

func (s relationSet) add(schema, table string) {
	if table == "" {
		return
	}
	s[relationRef{schema: strings.ToLower(schema), table: strings.ToLower(table)}] = true
}

func (s relationSet) has(schema, table string) bool {
	name := strings.ToLower(table)
	return s[relationRef{schema: strings.ToLower(schema), table: name}] || s[relationRef{table: name}]
}

// analyzedScope is what omni's scope resolution says a statement reads.
//
// The omni analyzer resolves every table reference against the CTEs in scope
// at that point, which is the one thing a name-based walk of the parse tree
// cannot do: a non-recursive CTE's own name is not visible inside its own body,
// WITH RECURSIVE makes it visible, an earlier sibling CTE is visible to a later
// one, and an inner WITH shadows an outer one. So `WITH t AS (SELECT * FROM t)
// SELECT * FROM t` reads the physical t once, in the CTE body, and the two
// other mentions of t are the CTE.
//
// The scope is built by walking the analyzed query, not the parse tree.
type analyzedScope struct {
	// relations holds every range table entry the analyzer bound to a real
	// relation, anywhere in the query.
	relations relationSet
	// cteNames holds every name a WITH clause binds, anywhere in the query.
	// It is not the decision — relations is — it bounds what a gap in the walk
	// can cost. See reads.
	cteNames map[string]bool
	// unknownExpr records that the walk met an expression type it does not
	// handle, so it cannot claim to have found every subquery. See reads.
	unknownExpr bool
}

// reads reports whether a table the access-table walk found is a relation this
// statement really reads.
//
// A nil scope means the caller has no analyzer output to consult and wants
// every access checked.
//
// The relation set decides: a name the analyzer bound to a real relation is a
// read, whatever else shares that name. The CTE-name set only limits the blast
// radius of a walk that misses a node type: an access is dropped only on
// positive evidence that the analyzer bound its name as a CTE. A missed node
// then costs a refused query (false positive) rather than an unchecked read
// (fail-open), and costs even that only when a CTE shares the name.
func (s *analyzedScope) reads(schema, table string) bool {
	if s == nil {
		return true
	}
	if s.unknownExpr {
		// The walk met a type it does not understand, so its evidence about what
		// this statement reads is incomplete. Check every access rather than
		// trust a walk that may have stepped over a subquery. On a healthy
		// snapshot this changes nothing; on a degraded one it refuses, which is
		// the direction this signal exists to fail in.
		return true
	}
	if s.relations.has(schema, table) {
		return true
	}
	return !s.cteNames[strings.ToLower(table)]
}

// scopeCollector walks an analyzed query and fills an analyzedScope.
//
// Every position that can hold a nested *catalog.Query is reached: the range
// table (subqueries and function arguments), the CTE list, both branches of a
// set operation, and every expression position that can hold a SubLinkExpr —
// the target list, the join tree's WHERE and ON conditions, HAVING, LIMIT,
// OFFSET, and window frame offsets.
//
// GROUP BY, ORDER BY and DISTINCT ON hold no expressions of their own: omni
// models them as *catalog.SortGroupClause, an index into the target list, so
// walking the target list covers them.
//
// A view's body is deliberately not expanded. A view is itself the relation the
// query reads, and the snapshot describes its columns directly; expanding it
// would widen the signal to relations the access-table walk never reports.
type scopeCollector struct {
	cat     *catalog.Catalog
	scope   *analyzedScope
	visited map[*catalog.Query]bool
}

// collectAnalyzedScope returns what the analyzer resolved q to read.
func collectAnalyzedScope(cat *catalog.Catalog, q *catalog.Query) *analyzedScope {
	c := &scopeCollector{
		cat: cat,
		scope: &analyzedScope{
			relations: make(relationSet),
			cteNames:  make(map[string]bool),
		},
		visited: make(map[*catalog.Query]bool),
	}
	c.walkQuery(q)
	return c.scope
}

func (c *scopeCollector) walkQuery(q *catalog.Query) {
	// A recursive CTE's query refers to itself, so the walk needs a cycle guard.
	if q == nil || c.visited[q] {
		return
	}
	c.visited[q] = true

	c.walkCTEList(q.CTEList)
	c.walkRangeTable(q.RangeTable)
	c.walkJoinTree(q.JoinTree)
	c.walkTargetList(q.TargetList)
	c.walkWindowClause(q.WindowClause)
	c.walkExpr(q.HavingQual)
	c.walkExpr(q.LimitCount)
	c.walkExpr(q.LimitOffset)
	c.walkQuery(q.LArg)
	c.walkQuery(q.RArg)
}

func (c *scopeCollector) walkCTEList(list []*catalog.CommonTableExprQ) {
	for _, cte := range list {
		if cte == nil {
			continue
		}
		if cte.Name != "" {
			c.scope.cteNames[strings.ToLower(cte.Name)] = true
		}
		c.walkQuery(cte.Query)
	}
}

func (c *scopeCollector) walkRangeTable(rangeTable []*catalog.RangeTableEntry) {
	for _, rte := range rangeTable {
		c.walkRangeTableEntry(rte)
	}
}

func (c *scopeCollector) walkRangeTableEntry(rte *catalog.RangeTableEntry) {
	if rte == nil {
		return
	}
	switch rte.Kind {
	case catalog.RTERelation:
		c.collectRelation(rte)
		c.walkTablesample(rte.Tablesample)
	case catalog.RTESubquery:
		c.walkQuery(rte.Subquery)
	case catalog.RTEFunction:
		c.walkExprs(rte.FuncExprs)
	case catalog.RTECTE:
		// The CTE body is walked from the CTE list that binds it, which is this
		// query's or an enclosing one's — both are on the walk.
	case catalog.RTEJoin:
		// A join entry carries no relation of its own; its inputs are separate
		// range table entries.
	default:
	}
}

// collectRelation records the relation an RTERelation entry is bound to.
//
// The entry's SchemaName is only what the statement wrote, so it is empty for
// an unqualified reference. The catalog holds the schema the analyzer actually
// resolved the OID to, which is what the access-table walk reports too.
// walkTablesample walks a TABLESAMPLE clause. Its arguments and REPEATABLE seed
// are ordinary expressions and can hold a subquery.
func (c *scopeCollector) walkTablesample(ts *catalog.TablesampleClauseQ) {
	if ts == nil {
		return
	}
	c.walkExprs(ts.Args)
	c.walkExpr(ts.Repeatable)
}

func (c *scopeCollector) collectRelation(rte *catalog.RangeTableEntry) {
	schema := rte.SchemaName
	if c.cat != nil {
		if rel := c.cat.GetRelationByOID(rte.RelOID); rel != nil && rel.Schema != nil {
			schema = rel.Schema.Name
		}
	}
	c.scope.relations.add(schema, rte.RelName)
}

func (c *scopeCollector) walkJoinTree(tree *catalog.JoinTree) {
	if tree == nil {
		return
	}
	for _, node := range tree.FromList {
		c.walkJoinNode(node)
	}
	c.walkExpr(tree.Quals)
}

// walkJoinNode descends a join expression for its ON condition. The other
// JoinNode implementation, *catalog.RangeTableRef, is an index into the range
// table, which walkRangeTable covers in full.
func (c *scopeCollector) walkJoinNode(node catalog.JoinNode) {
	join, ok := node.(*catalog.JoinExprNode)
	if !ok || join == nil {
		return
	}
	c.walkJoinNode(join.Left)
	c.walkJoinNode(join.Right)
	c.walkExpr(join.Quals)
}

func (c *scopeCollector) walkTargetList(list []*catalog.TargetEntry) {
	for _, entry := range list {
		if entry == nil {
			continue
		}
		c.walkExpr(entry.Expr)
	}
}

func (c *scopeCollector) walkWindowClause(list []*catalog.WindowClauseQ) {
	for _, window := range list {
		if window == nil {
			continue
		}
		c.walkExpr(window.StartOffset)
		c.walkExpr(window.EndOffset)
	}
}

func (c *scopeCollector) walkExprs(exprs []catalog.AnalyzedExpr) {
	for _, expr := range exprs {
		c.walkExpr(expr)
	}
}

// walkExpr descends every child expression, and every subquery a SubLinkExpr
// carries.
//
// The type switch names the expression types that can carry a subquery in the
// pinned omni. It is NOT assumed exhaustive: omni adds types, and an earlier
// revision of this file enumerated them against a stale module directory and
// silently stopped checking ten statement shapes. Check the real list with
//
//	go list -m github.com/bytebase/omni
//	grep -rh 'func (.*) exprType() uint32' "$(go list -m -f '{{.Dir}}' github.com/bytebase/omni)"/pg/catalog/*.go
//
// The default arm marks the scope incomplete rather than trusting this list.
func (c *scopeCollector) walkExpr(expr catalog.AnalyzedExpr) {
	switch e := expr.(type) {
	case *catalog.SubLinkExpr:
		c.walkExpr(e.TestExpr)
		c.walkQuery(e.SubQuery)
	case *catalog.FuncCallExpr:
		c.walkExprs(e.Args)
	case *catalog.AggExpr:
		c.walkExprs(e.Args)
	case *catalog.WindowFuncExpr:
		c.walkExprs(e.Args)
		c.walkExpr(e.AggFilter)
	case *catalog.OpExpr:
		c.walkExpr(e.Left)
		c.walkExpr(e.Right)
	case *catalog.DistinctExprQ:
		c.walkExpr(e.Left)
		c.walkExpr(e.Right)
	case *catalog.ScalarArrayOpExpr:
		c.walkExpr(e.Left)
		c.walkExpr(e.Right)
	case *catalog.CaseExprQ:
		c.walkCaseExpr(e)
	case *catalog.BoolExprQ:
		c.walkExprs(e.Args)
	case *catalog.CoalesceExprQ:
		c.walkExprs(e.Args)
	case *catalog.NullIfExprQ:
		c.walkExprs(e.Args)
	case *catalog.MinMaxExprQ:
		c.walkExprs(e.Args)
	case *catalog.ArrayExprQ:
		c.walkExprs(e.Elements)
	case *catalog.RowExprQ:
		c.walkExprs(e.Args)
	case *catalog.RelabelExpr:
		c.walkExpr(e.Arg)
	case *catalog.CoerceViaIOExpr:
		c.walkExpr(e.Arg)
	case *catalog.CollateExprQ:
		c.walkExpr(e.Arg)
	case *catalog.NullTestExpr:
		c.walkExpr(e.Arg)
	case *catalog.BooleanTestExpr:
		c.walkExpr(e.Arg)
	case *catalog.FieldSelectExprQ:
		c.walkExpr(e.Arg)
	case *catalog.VarExpr, *catalog.ConstExpr, *catalog.SQLValueFuncExpr,
		*catalog.CoerceToDomainValueExpr:
		// Leaves: a column reference, a constant, CURRENT_DATE and friends, and
		// the VALUE keyword of a domain constraint carry no child expression.
	case nil:
	default:
		// An expression type omni added after this switch was written. If it can
		// carry a subquery, the reads inside it are no longer checked, so say so
		// rather than dropping them in silence.
		c.scope.unknownExpr = true
		slog.Debug("unhandled analyzed expression type in the relation walk",
			slog.String("type", fmt.Sprintf("%T", expr)))
	}
}

func (c *scopeCollector) walkCaseExpr(e *catalog.CaseExprQ) {
	c.walkExpr(e.Arg)
	for _, when := range e.When {
		if when == nil {
			continue
		}
		c.walkExpr(when.Condition)
		c.walkExpr(when.Result)
	}
	c.walkExpr(e.Default)
}
