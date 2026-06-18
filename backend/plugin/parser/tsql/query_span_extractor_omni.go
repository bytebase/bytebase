package tsql

import (
	"context"
	"reflect"
	"strings"
	"unicode"

	"github.com/bytebase/omni/mssql/ast"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

// omniQuerySpanExtractor extracts query span information from omni MSSQL AST.
// It embeds querySpanExtractor so string-based resolution helpers
// (tsqlIsFieldSensitive, tsqlFindTableSchemaByParts, isIdentifierEqual,
// tsqlGetAllFieldsOfTableInFromOrOuterCTE) are reused directly.
type omniQuerySpanExtractor struct {
	*querySpanExtractor
	// source is the original SQL text, used to slice expression names via Loc.
	source string
}

func newOmniQuerySpanExtractor(defaultDatabase, defaultSchema string, gCtx base.GetQuerySpanContext, ignoreCaseSensitive bool) *omniQuerySpanExtractor {
	return &omniQuerySpanExtractor{
		querySpanExtractor: newQuerySpanExtractor(defaultDatabase, defaultSchema, gCtx, ignoreCaseSensitive),
	}
}

// getOmniQuerySpan is the public entry point. Mirrors the ANTLR getQuerySpan flow.
func (q *omniQuerySpanExtractor) getOmniQuerySpan(ctx context.Context, statement string) (*base.QuerySpan, error) {
	q.ctx = ctx
	q.source = statement

	stmts, err := ParseTSQLOmni(statement)
	if err != nil {
		return nil, err
	}
	if len(stmts) == 0 {
		return &base.QuerySpan{
			Type:          base.Select,
			SourceColumns: make(base.SourceColumnSet),
			Results:       []base.QuerySpanResult{},
		}, nil
	}
	if len(stmts) != 1 {
		return nil, errors.Errorf("expected exactly 1 statement, got %d", len(stmts))
	}

	root := stmts[0].AST
	if root == nil {
		// An empty statement. Match ANTLR's zero-AST behavior.
		return &base.QuerySpan{
			Type:          base.Select,
			SourceColumns: make(base.SourceColumnSet),
			Results:       []base.QuerySpanResult{},
		}, nil
	}
	if _, isGo := root.(*ast.GoStmt); isGo {
		// A bare "GO" batch separator. ANTLR discards it and returns an empty
		// Select span; match that.
		return &base.QuerySpan{
			Type:          base.Select,
			SourceColumns: make(base.SourceColumnSet),
			Results:       []base.QuerySpanResult{},
		}, nil
	}

	// DECLARE @t TABLE(...) populates gCtx.TempTables so subsequent statements
	// in the same session (the caller loops over split statements) can resolve
	// @t. Mirrors the legacy ANTLR tsqlSelectOnlyListener.EnterDeclare_statement
	// side-effect. Non-table DECLARE (plain scalar variables) is a no-op.
	if d, ok := root.(*ast.DeclareStmt); ok {
		populateOmniTempTables(d, q.gCtx.TempTables)
	}
	if c, ok := root.(*ast.CreateTableStmt); ok {
		populateOmniCreateTempTable(c, q.gCtx.TempTables)
	}

	accessTables := collectOmniAccessTables(root, q.defaultDatabase, q.defaultSchema)
	allSystems, mixed := isMixedQuery(accessTables, q.ignoreCaseSensitive)
	if mixed {
		return nil, base.MixUserSystemTablesError
	}

	queryType := classifyQueryType(root, allSystems)
	sel, isSelect := root.(*ast.SelectStmt)
	// classifyQueryType maps SET/DECLARE/SetOption to base.Select for historical
	// reasons; those have no result columns. Return early if the root isn't a
	// real SELECT.
	if queryType != base.Select || !isSelect {
		return &base.QuerySpan{
			Type:          queryType,
			SourceColumns: base.SourceColumnSet{},
			Results:       []base.QuerySpanResult{},
		}, nil
	}

	pseudo, err := q.extractFromSelectStmt(sel)
	if err != nil {
		var resourceNotFound *base.ResourceNotFoundError
		if errors.As(err, &resourceNotFound) {
			return &base.QuerySpan{
				Type:          base.Select,
				SourceColumns: accessTables,
				Results:       []base.QuerySpanResult{},
				NotFoundError: resourceNotFound,
			}, nil
		}
		return nil, err
	}

	return &base.QuerySpan{
		Type:             base.Select,
		SourceColumns:    accessTables,
		Results:          pseudo.GetQuerySpanResult(),
		PredicateColumns: q.predicateColumns,
	}, nil
}

// -------------------- SELECT processing --------------------

// extractFromSelectStmt processes a full SELECT statement (including set-ops,
// WITH clause) and returns the result column set as a PseudoTable.
func (q *omniQuerySpanExtractor) extractFromSelectStmt(sel *ast.SelectStmt) (*base.PseudoTable, error) {
	// WITH clause: populate CTE list BEFORE dispatching set-ops so that both
	// arms of a `WITH cte AS (...) SELECT ... FROM cte UNION SELECT ... FROM cte`
	// see the CTE in scope. The defer restores q.ctes on exit.
	ctesSnapshot := len(q.ctes)
	defer func() { q.ctes = q.ctes[:ctesSnapshot] }()
	if sel.WithClause != nil {
		if err := q.processWithClause(sel.WithClause); err != nil {
			return nil, err
		}
	}

	// Set operations: recurse into arms and union. The WITH clause above is
	// already in q.ctes, so both arms resolve CTE references correctly.
	if sel.Op != ast.SetOpNone && (sel.Larg != nil || sel.Rarg != nil) {
		return q.extractFromSetOp(sel)
	}

	// FROM clause — a SELECT without FROM is legal (e.g. SELECT 1).
	tableSourcesSnapshot := len(q.tableSourcesFrom)
	defer func() { q.tableSourcesFrom = q.tableSourcesFrom[:tableSourcesSnapshot] }()
	if sel.FromClause != nil {
		for _, item := range sel.FromClause.Items {
			ts, err := q.extractTableSource(item)
			if err != nil {
				return nil, err
			}
			q.tableSourcesFrom = append(q.tableSourcesFrom, ts...)
		}
	}

	// WHERE / HAVING predicate columns — resolved in THIS select's scope.
	// Subquery expressions encountered during the walk get cloned extractors;
	// their results merge back into q.predicateColumns. Errors (including
	// ResourceNotFoundError from a subquery referencing a missing table)
	// propagate up to getOmniQuerySpan which converts them into a span with
	// NotFoundError populated.
	if err := q.collectPredicatesInScope(sel.WhereClause); err != nil {
		return nil, err
	}
	if err := q.collectPredicatesInScope(sel.HavingClause); err != nil {
		return nil, err
	}

	// Target list.
	results, err := q.extractTargetList(sel.TargetList)
	if err != nil {
		return nil, err
	}
	q.populateSelectIntoTempTable(sel, results)

	return &base.PseudoTable{Columns: results}, nil
}

// collectPredicatesInScope walks a boolean expression in the extractor's
// current scope: plain ColumnRefs resolve against q.tableSourcesFrom +
// q.outerTableSources; subquery nodes trigger a scope clone, and the
// subquery's output-column sources plus its own predicate-column set merge
// back into q.predicateColumns.
//
// Errors from nested subquery extraction (ResourceNotFoundError or real
// parser errors) propagate so the caller can surface them via the top-level
// NotFoundError path instead of producing a silently-partial span.
func (q *omniQuerySpanExtractor) collectPredicatesInScope(expr ast.Node) error {
	if expr == nil {
		return nil
	}
	switch v := expr.(type) {
	case *ast.ColumnRef:
		r, err := q.tsqlIsFieldSensitive(v.Database, v.Schema, v.Table, v.Column)
		if err != nil {
			return errors.Wrapf(err, "failed to resolve predicate column %q", v.Column)
		}
		q.predicateColumns, _ = base.MergeSourceColumnSet(q.predicateColumns, r.SourceColumns)
		return nil
	case *ast.FullTextPredicate:
		if v.Columns != nil {
			for _, it := range v.Columns.Items {
				if cr, ok := it.(*ast.ColumnRef); ok {
					if err := q.collectPredicatesInScope(cr); err != nil {
						return err
					}
				}
			}
		}
		if err := q.collectPredicatesInScope(v.Value); err != nil {
			return err
		}
		return q.collectPredicatesInScope(v.LanguageTerm)
	case *ast.SubqueryExpr:
		return q.mergeSubqueryIntoPredicates(v.Query)
	case *ast.ExistsExpr:
		return q.mergeSubqueryIntoPredicates(v.Subquery)
	case *ast.SubqueryComparisonExpr:
		if err := q.collectPredicatesInScope(v.Left); err != nil {
			return err
		}
		return q.mergeSubqueryIntoPredicates(v.Subquery)
	case *ast.InExpr:
		if err := q.collectPredicatesInScope(v.Expr); err != nil {
			return err
		}
		if v.List != nil {
			for _, it := range v.List.Items {
				if err := q.collectPredicatesInScope(it); err != nil {
					return err
				}
			}
		}
		return q.collectPredicatesInScope(v.Subquery)
	case *ast.BinaryExpr:
		if err := q.collectPredicatesInScope(v.Left); err != nil {
			return err
		}
		return q.collectPredicatesInScope(v.Right)
	case *ast.UnaryExpr:
		return q.collectPredicatesInScope(v.Operand)
	case *ast.BetweenExpr:
		if err := q.collectPredicatesInScope(v.Expr); err != nil {
			return err
		}
		if err := q.collectPredicatesInScope(v.Low); err != nil {
			return err
		}
		return q.collectPredicatesInScope(v.High)
	case *ast.LikeExpr:
		if err := q.collectPredicatesInScope(v.Expr); err != nil {
			return err
		}
		if err := q.collectPredicatesInScope(v.Pattern); err != nil {
			return err
		}
		return q.collectPredicatesInScope(v.Escape)
	case *ast.IsExpr:
		return q.collectPredicatesInScope(v.Expr)
	case *ast.CaseExpr:
		if err := q.collectPredicatesInScope(v.Arg); err != nil {
			return err
		}
		if err := q.collectPredicatesInScope(v.ElseExpr); err != nil {
			return err
		}
		if v.WhenList != nil {
			for _, it := range v.WhenList.Items {
				if cw, ok := it.(*ast.CaseWhen); ok {
					if err := q.collectPredicatesInScope(cw.Condition); err != nil {
						return err
					}
					if err := q.collectPredicatesInScope(cw.Result); err != nil {
						return err
					}
				}
			}
		}
		return nil
	case *ast.IifExpr:
		if err := q.collectPredicatesInScope(v.Condition); err != nil {
			return err
		}
		if err := q.collectPredicatesInScope(v.TrueVal); err != nil {
			return err
		}
		return q.collectPredicatesInScope(v.FalseVal)
	case *ast.CoalesceExpr:
		if v.Args != nil {
			for _, it := range v.Args.Items {
				if err := q.collectPredicatesInScope(it); err != nil {
					return err
				}
			}
		}
		return nil
	case *ast.NullifExpr:
		if err := q.collectPredicatesInScope(v.Left); err != nil {
			return err
		}
		return q.collectPredicatesInScope(v.Right)
	case *ast.FuncCallExpr:
		if v.Args != nil {
			for _, it := range v.Args.Items {
				if err := q.collectPredicatesInScope(it); err != nil {
					return err
				}
			}
		}
		return nil
	case *ast.MethodCallExpr:
		if v.Args != nil {
			for _, it := range v.Args.Items {
				if err := q.collectPredicatesInScope(it); err != nil {
					return err
				}
			}
		}
		return nil
	case *ast.CastExpr:
		return q.collectPredicatesInScope(v.Expr)
	case *ast.ConvertExpr:
		if err := q.collectPredicatesInScope(v.Expr); err != nil {
			return err
		}
		return q.collectPredicatesInScope(v.Style)
	case *ast.TryCastExpr:
		return q.collectPredicatesInScope(v.Expr)
	case *ast.TryConvertExpr:
		if err := q.collectPredicatesInScope(v.Expr); err != nil {
			return err
		}
		return q.collectPredicatesInScope(v.Style)
	case *ast.ParenExpr:
		return q.collectPredicatesInScope(v.Expr)
	case *ast.CollateExpr:
		return q.collectPredicatesInScope(v.Expr)
	case *ast.AtTimeZoneExpr:
		if err := q.collectPredicatesInScope(v.Expr); err != nil {
			return err
		}
		return q.collectPredicatesInScope(v.TimeZone)
	default:
		// Literal, VariableRef, StarExpr, etc. — nothing to collect.
		return nil
	}
}

func (q *omniQuerySpanExtractor) mergeSubqueryIntoPredicates(body ast.Node) error {
	sel, ok := body.(*ast.SelectStmt)
	if !ok {
		return nil
	}
	clone := q.cloneForSubquery()
	pseudo, err := clone.extractFromSelectStmt(sel)
	if err != nil {
		return err
	}
	// Subquery output columns become outer-scope predicate columns.
	for _, c := range pseudo.GetQuerySpanResult() {
		q.predicateColumns, _ = base.MergeSourceColumnSet(q.predicateColumns, c.SourceColumns)
	}
	// And the clone's own predicate columns (from its WHERE etc.) propagate up.
	q.predicateColumns, _ = base.MergeSourceColumnSet(q.predicateColumns, clone.predicateColumns)
	return nil
}

func (q *omniQuerySpanExtractor) extractFromSetOp(sel *ast.SelectStmt) (*base.PseudoTable, error) {
	// Omni puts the WITH clause on the leftmost set-op arm (the parser attaches
	// the WITH to the first SELECT expression, not to the outer Op node). Lift
	// it into the current scope so the CTE is visible in BOTH arms, then
	// recurse into a copy of Larg with WithClause cleared to avoid double
	// processing.
	larg := sel.Larg
	if larg != nil && larg.WithClause != nil {
		if err := q.processWithClause(larg.WithClause); err != nil {
			return nil, err
		}
		cp := *larg
		cp.WithClause = nil
		larg = &cp
	}
	left, err := q.extractFromSelectStmt(larg)
	if err != nil {
		return nil, err
	}
	right, err := q.extractFromSelectStmt(sel.Rarg)
	if err != nil {
		return nil, err
	}
	merged, err := unionTableSources(left, right)
	if err != nil {
		return nil, err
	}
	return &base.PseudoTable{Columns: merged}, nil
}

func (q *omniQuerySpanExtractor) processWithClause(w *ast.WithClause) error {
	if w.CTEs == nil {
		return nil
	}
	for _, item := range w.CTEs.Items {
		cte, ok := item.(*ast.CommonTableExpr)
		if !ok {
			continue
		}
		body, ok := cte.Query.(*ast.SelectStmt)
		if !ok {
			return unsupportedNodeError("only SELECT CTE bodies are supported", cte.Query)
		}

		cteName := normIdent(cte.Name)
		cols, predicates, err := q.extractCTEBody(cteName, body)
		if err != nil {
			return err
		}
		if cte.Columns != nil && cte.Columns.Len() > 0 {
			if cte.Columns.Len() != len(cols) {
				return errors.Errorf("CTE %q column alias count %d does not match body column count %d", cte.Name, cte.Columns.Len(), len(cols))
			}
			for i, it := range cte.Columns.Items {
				if s, ok := it.(*ast.String); ok {
					cols[i].Name = normIdent(s.Str)
				}
			}
		}
		q.ctes = append(q.ctes, &base.PseudoTable{Name: cteName, Columns: cols})
		q.predicateColumns, _ = base.MergeSourceColumnSet(q.predicateColumns, predicates)
	}
	return nil
}

// extractCTEBody extracts the column set of a CTE body. For recursive CTEs
// (body is a set-op SelectStmt whose recursive arm references the CTE name),
// the extractor iterates until the column sources stabilize.
func (q *omniQuerySpanExtractor) extractCTEBody(cteName string, body *ast.SelectStmt) ([]base.QuerySpanResult, base.SourceColumnSet, error) {
	newClone := func() *omniQuerySpanExtractor {
		return &omniQuerySpanExtractor{
			querySpanExtractor: &querySpanExtractor{
				ctx:                 q.ctx,
				defaultDatabase:     q.defaultDatabase,
				defaultSchema:       q.defaultSchema,
				ignoreCaseSensitive: q.ignoreCaseSensitive,
				gCtx:                q.gCtx,
				ctes:                append([]*base.PseudoTable{}, q.ctes...),
				outerTableSources:   nil,
				predicateColumns:    make(base.SourceColumnSet),
				viewResolutionStack: cloneViewResolutionStack(q.viewResolutionStack),
			},
			source: q.source,
		}
	}

	// Non-set-op CTE: extract directly.
	if body.Op == ast.SetOpNone || (body.Larg == nil && body.Rarg == nil) {
		clone := newClone()
		pseudo, err := clone.extractFromSelectStmt(body)
		if err != nil {
			return nil, nil, err
		}
		return pseudo.GetQuerySpanResult(), clone.predicateColumns, nil
	}

	// Set-op CTE: extract anchor (Larg), register placeholder, then iterate Rarg
	// until the source column sets for the CTE columns stabilize.
	anchorClone := newClone()
	anchorPseudo, err := anchorClone.extractFromSelectStmt(body.Larg)
	if err != nil {
		return nil, nil, err
	}
	cols := append([]base.QuerySpanResult{}, anchorPseudo.GetQuerySpanResult()...)
	predicates := base.SourceColumnSet{}
	predicates, _ = base.MergeSourceColumnSet(predicates, anchorClone.predicateColumns)

	for iter := 0; iter < 16; iter++ {
		placeholder := &base.PseudoTable{Name: cteName, Columns: cols}
		clone := newClone()
		clone.ctes = append(clone.ctes, placeholder)
		rarg, rerr := clone.extractFromSelectStmt(body.Rarg)
		if rerr != nil {
			// Recursive arm failed to resolve (e.g. ResourceNotFoundError from
			// a missing table). Propagate — the top-level getOmniQuerySpan
			// converts ResourceNotFoundError into a span with NotFoundError
			// populated so callers don't see a silently-partial lineage.
			return nil, nil, rerr
		}
		rcols := rarg.GetQuerySpanResult()
		if len(rcols) != len(cols) {
			return nil, nil, errors.Errorf("CTE %q recursive arm returns %d columns, anchor returns %d", cteName, len(rcols), len(cols))
		}
		changed := false
		for i := range rcols {
			merged, anyChange := base.MergeSourceColumnSet(cols[i].SourceColumns, rcols[i].SourceColumns)
			cols[i].SourceColumns = merged
			if anyChange {
				changed = true
			}
		}
		predicates, _ = base.MergeSourceColumnSet(predicates, clone.predicateColumns)
		if !changed {
			break
		}
	}
	return cols, predicates, nil
}

// -------------------- FROM / table source --------------------

// extractTableSource returns the TableSource(s) produced by a single FROM item.
// A JoinClause contributes multiple sources (left + right flattened).
func (q *omniQuerySpanExtractor) extractTableSource(node ast.Node) ([]base.TableSource, error) {
	switch v := node.(type) {
	case *ast.JoinClause:
		return q.extractFromJoin(v)
	case *ast.TableRef:
		ts, err := q.resolveTableRef(v)
		if err != nil {
			return nil, err
		}
		return []base.TableSource{ts}, nil
	case *ast.AliasedTableRef:
		ts, err := q.resolveAliasedTableRef(v)
		if err != nil {
			return nil, err
		}
		return []base.TableSource{ts}, nil
	case *ast.SubqueryExpr:
		ts, err := q.resolveSubqueryAsTable(v, "")
		if err != nil {
			return nil, err
		}
		return []base.TableSource{ts}, nil
	case *ast.ValuesClause:
		ts, err := q.resolveValuesClause(v, "", nil)
		if err != nil {
			return nil, err
		}
		return []base.TableSource{ts}, nil
	case *ast.TableVarRef:
		return q.resolveTableVar(v, v.Alias)
	case *ast.TableVarMethodCallRef:
		return []base.TableSource{&base.PseudoTable{Name: v.Alias, Columns: xmlNodesColumns(v.Columns)}}, nil
	case *ast.PivotExpr:
		return nil, unsupportedTableSourceError(node)
	case *ast.UnpivotExpr:
		return nil, unsupportedTableSourceError(node)
	case *ast.FuncCallExpr:
		ts, err := q.resolveTableValuedFunction(v, "", nil)
		if err != nil {
			return nil, err
		}
		return []base.TableSource{ts}, nil
	default:
		return nil, unsupportedTableSourceError(node)
	}
}

func unsupportedTableSourceError(node ast.Node) error {
	return unsupportedNodeError("only full table name, supported TVF, derived table, values clause, and temp table in table source item are supported", node)
}

func unsupportedNodeError(message string, node ast.Node) error {
	typeName := "<nil>"
	if typ := reflect.TypeOf(node); typ != nil {
		typeName = typ.String()
	}
	return &base.TypeNotSupportedError{
		Err:  errors.New(message),
		Type: typeName,
	}
}

func (q *omniQuerySpanExtractor) extractFromJoin(j *ast.JoinClause) ([]base.TableSource, error) {
	left, err := q.extractTableSource(j.Left)
	if err != nil {
		return nil, err
	}
	if j.Type == ast.JoinCrossApply || j.Type == ast.JoinOuterApply {
		tableSourcesSnapshot := len(q.tableSourcesFrom)
		q.tableSourcesFrom = append(q.tableSourcesFrom, left...)
		defer func() { q.tableSourcesFrom = q.tableSourcesFrom[:tableSourcesSnapshot] }()
	}
	right, err := q.extractTableSource(j.Right)
	if err != nil {
		return nil, err
	}
	return append(left, right...), nil
}

func (q *omniQuerySpanExtractor) resolveTableRef(t *ast.TableRef) (base.TableSource, error) {
	ts, err := q.tsqlFindTableSchemaByParts(t.Server, t.Database, t.Schema, t.Object)
	if err != nil {
		return nil, err
	}
	if t.Alias != "" {
		ts = &base.PseudoTable{
			Name:    normIdent(t.Alias),
			Columns: ts.GetQuerySpanResult(),
		}
	}
	return ts, nil
}

func (q *omniQuerySpanExtractor) resolveAliasedTableRef(at *ast.AliasedTableRef) (base.TableSource, error) {
	var columnAliases []string
	if at.Columns != nil {
		for _, it := range at.Columns.Items {
			if s, ok := it.(*ast.String); ok {
				columnAliases = append(columnAliases, s.Str)
			}
		}
	}
	switch inner := at.Table.(type) {
	case *ast.TableRef:
		ts, err := q.tsqlFindTableSchemaByParts(inner.Server, inner.Database, inner.Schema, inner.Object)
		if err != nil {
			return nil, err
		}
		cols := append([]base.QuerySpanResult{}, ts.GetQuerySpanResult()...)
		if err := applyColumnAliases(cols, columnAliases); err != nil {
			return nil, err
		}
		name := inner.Object
		if at.Alias != "" {
			name = at.Alias
		}
		return &base.PseudoTable{Name: name, Columns: cols}, nil
	case *ast.SubqueryExpr:
		return q.resolveSubqueryAsTable(inner, at.Alias, columnAliases...)
	case *ast.ValuesClause:
		return q.resolveValuesClause(inner, at.Alias, columnAliases)
	case *ast.TableVarRef:
		inners, err := q.resolveTableVar(inner, at.Alias)
		if err != nil {
			return nil, err
		}
		if len(inners) == 1 {
			return inners[0], nil
		}
		return nil, unsupportedNodeError("aliased table variable must produce a single table source", at.Table)
	case *ast.FuncCallExpr:
		return q.resolveTableValuedFunction(inner, at.Alias, columnAliases)
	case *ast.TableVarMethodCallRef:
		cols := xmlNodesColumns(inner.Columns)
		if err := applyColumnAliases(cols, columnAliases); err != nil {
			return nil, err
		}
		return &base.PseudoTable{Name: normIdent(at.Alias), Columns: cols}, nil
	default:
		return nil, unsupportedNodeError("unsupported aliased table source item", at.Table)
	}
}

func (q *omniQuerySpanExtractor) resolveTableValuedFunction(fn *ast.FuncCallExpr, alias string, columnAliases []string) (base.TableSource, error) {
	if fn == nil || fn.Name == nil {
		return nil, unsupportedTableSourceError(fn)
	}
	var columnNames []string
	switch strings.ToUpper(fn.Name.Object) {
	case "STRING_SPLIT":
		columnNames = []string{"value"}
		hasOrdinal, err := stringSplitHasOrdinalColumn(fn.Args)
		if err != nil {
			return nil, err
		}
		if hasOrdinal {
			columnNames = append(columnNames, "ordinal")
		}
	case "OPENJSON":
		columnNames = []string{"key", "value", "type"}
	default:
		return nil, unsupportedTableSourceError(fn)
	}

	argSources, err := q.mergeListSources(fn.Args)
	if err != nil {
		return nil, err
	}
	columns := make([]base.QuerySpanResult, 0, len(columnNames))
	for _, name := range columnNames {
		columns = append(columns, base.QuerySpanResult{
			Name:          name,
			SourceColumns: argSources.SourceColumns,
		})
	}
	if strings.EqualFold(fn.Name.Object, "STRING_SPLIT") && len(columns) > 1 {
		columns[1].SourceColumns = make(base.SourceColumnSet)
	}
	if err := applyColumnAliases(columns, columnAliases); err != nil {
		return nil, err
	}
	return &base.PseudoTable{Name: normIdent(alias), Columns: columns}, nil
}

func stringSplitHasOrdinalColumn(args *ast.List) (bool, error) {
	if args == nil || args.Len() < 3 {
		return false, nil
	}
	arg := args.Items[2]
	for {
		paren, ok := arg.(*ast.ParenExpr)
		if !ok {
			break
		}
		arg = paren.Expr
	}
	literal, ok := arg.(*ast.Literal)
	if !ok {
		return false, unsupportedNodeError("STRING_SPLIT enable_ordinal must be a constant 0, 1, or NULL", arg)
	}
	switch literal.Type {
	case ast.LitInteger:
		switch literal.Ival {
		case 0:
			return false, nil
		case 1:
			return true, nil
		default:
			return false, unsupportedNodeError("STRING_SPLIT enable_ordinal must be 0 or 1", arg)
		}
	case ast.LitNull:
		return false, nil
	default:
		return false, unsupportedNodeError("STRING_SPLIT enable_ordinal must be a constant 0, 1, or NULL", arg)
	}
}

func (q *omniQuerySpanExtractor) resolveSubqueryAsTable(sq *ast.SubqueryExpr, alias string, columnAliases ...string) (base.TableSource, error) {
	body, ok := sq.Query.(*ast.SelectStmt)
	if !ok {
		return nil, unsupportedNodeError("only SELECT subqueries in table source are supported", sq.Query)
	}
	clone := q.cloneForSubquery()
	pseudo, err := clone.extractFromSelectStmt(body)
	if err != nil {
		return nil, err
	}
	cols := append([]base.QuerySpanResult{}, pseudo.GetQuerySpanResult()...)
	if err := applyColumnAliases(cols, columnAliases); err != nil {
		return nil, err
	}
	q.predicateColumns, _ = base.MergeSourceColumnSet(q.predicateColumns, clone.predicateColumns)
	return &base.PseudoTable{Name: normIdent(alias), Columns: cols}, nil
}

func (q *omniQuerySpanExtractor) resolveValuesClause(vc *ast.ValuesClause, alias string, columnAliases []string) (base.TableSource, error) {
	name := normIdent(alias)
	if vc.Rows == nil || vc.Rows.Len() == 0 {
		return &base.PseudoTable{Name: name}, nil
	}
	firstRow, ok := vc.Rows.Items[0].(*ast.List)
	if !ok {
		return &base.PseudoTable{Name: name}, nil
	}
	cols := make([]base.QuerySpanResult, 0, firstRow.Len())
	for rowIndex, item := range vc.Rows.Items {
		row, ok := item.(*ast.List)
		if !ok {
			continue
		}
		if row.Len() != firstRow.Len() {
			return nil, errors.Errorf("VALUES row %d has %d columns, first row has %d", rowIndex+1, row.Len(), firstRow.Len())
		}
		for i, it := range row.Items {
			expr, ok := it.(ast.ExprNode)
			if !ok {
				if rowIndex == 0 {
					cols = append(cols, base.QuerySpanResult{})
				}
				continue
			}
			r, err := q.resolveExpression(expr)
			if err != nil {
				return nil, err
			}
			if rowIndex == 0 {
				cols = append(cols, r)
				continue
			}
			cols[i].SourceColumns, _ = base.MergeSourceColumnSet(cols[i].SourceColumns, r.SourceColumns)
		}
	}
	if err := applyColumnAliases(cols, columnAliases); err != nil {
		return nil, err
	}
	return &base.PseudoTable{Name: name, Columns: cols}, nil
}

func (q *omniQuerySpanExtractor) resolveTableVar(tv *ast.TableVarRef, alias string) ([]base.TableSource, error) {
	if tv == nil {
		return nil, nil
	}
	name := tv.Name
	if temp, ok := q.findTempTable(name); ok {
		tableName := name
		if alias != "" {
			tableName = alias
		}
		cols := make([]base.QuerySpanResult, 0, len(temp.Columns))
		for _, c := range temp.Columns {
			cols = append(cols, base.QuerySpanResult{
				Name: c,
				SourceColumns: base.SourceColumnSet{
					base.ColumnResource{
						Server:   temp.Server,
						Database: temp.Database,
						Schema:   temp.Schema,
						Table:    temp.Name,
						Column:   c,
					}: true,
				},
				IsPlainField: true,
			})
		}
		return []base.TableSource{&base.PseudoTable{Name: tableName, Columns: cols}}, nil
	}
	return nil, &base.ResourceNotFoundError{
		Table: &name,
		Err:   errors.Errorf("temp table %s not found", name),
	}
}

// populateOmniTempTables mirrors the ANTLR tempTableColumnDefinitionListener:
// for each VariableDecl of table type, build a PhysicalTable keyed by the
// variable name (including the leading '@') and store it in the shared map.
func populateOmniTempTables(d *ast.DeclareStmt, out map[string]*base.PhysicalTable) {
	if d == nil || d.Variables == nil || out == nil {
		return
	}
	for _, it := range d.Variables.Items {
		v, ok := it.(*ast.VariableDecl)
		if !ok || !v.IsTable || v.TableDef == nil {
			continue
		}
		columns := make([]string, 0, v.TableDef.Len())
		for _, cdef := range v.TableDef.Items {
			if cd, ok := cdef.(*ast.ColumnDef); ok {
				columns = append(columns, cd.Name)
			}
		}
		out[v.Name] = &base.PhysicalTable{
			Name:    v.Name,
			Columns: columns,
		}
	}
}

func populateOmniCreateTempTable(c *ast.CreateTableStmt, out map[string]*base.PhysicalTable) {
	if c == nil || c.Name == nil || out == nil || !strings.HasPrefix(c.Name.Object, "#") {
		return
	}
	var columns []string
	if c.Columns != nil {
		columns = make([]string, 0, c.Columns.Len())
		for _, item := range c.Columns.Items {
			if cd, ok := item.(*ast.ColumnDef); ok {
				columns = append(columns, cd.Name)
			}
		}
	}
	out[c.Name.Object] = &base.PhysicalTable{
		Name:    c.Name.Object,
		Columns: columns,
	}
}

func (q *omniQuerySpanExtractor) populateSelectIntoTempTable(sel *ast.SelectStmt, results []base.QuerySpanResult) {
	if sel == nil || sel.IntoTable == nil || q.gCtx.TempTables == nil || !strings.HasPrefix(sel.IntoTable.Object, "#") {
		return
	}
	columns := make([]string, 0, len(results))
	for i, result := range results {
		name := result.Name
		if name == "" && sel.TargetList != nil && i < len(sel.TargetList.Items) {
			name = q.sliceName(sel.TargetList.Items[i])
		}
		columns = append(columns, name)
	}
	q.gCtx.TempTables[sel.IntoTable.Object] = &base.PhysicalTable{
		Name:    sel.IntoTable.Object,
		Columns: columns,
	}
}

func xmlNodesColumns(cols []string) []base.QuerySpanResult {
	out := make([]base.QuerySpanResult, 0, len(cols))
	for _, c := range cols {
		out = append(out, base.QuerySpanResult{Name: c})
	}
	return out
}

func applyColumnAliases(cols []base.QuerySpanResult, aliases []string) error {
	if len(aliases) == 0 {
		return nil
	}
	if len(aliases) != len(cols) {
		return errors.Errorf("column alias count %d does not match column count %d", len(aliases), len(cols))
	}
	for i, name := range aliases {
		cols[i].Name = normIdent(name)
	}
	return nil
}

func (q *omniQuerySpanExtractor) cloneForSubquery() *omniQuerySpanExtractor {
	clone := &omniQuerySpanExtractor{
		querySpanExtractor: &querySpanExtractor{
			ctx:                 q.ctx,
			defaultDatabase:     q.defaultDatabase,
			defaultSchema:       q.defaultSchema,
			ignoreCaseSensitive: q.ignoreCaseSensitive,
			gCtx:                q.gCtx,
			ctes:                append([]*base.PseudoTable{}, q.ctes...),
			outerTableSources:   append(append([]base.TableSource{}, q.outerTableSources...), q.tableSourcesFrom...),
			predicateColumns:    make(base.SourceColumnSet),
			viewResolutionStack: cloneViewResolutionStack(q.viewResolutionStack),
		},
		source: q.source,
	}
	return clone
}

// -------------------- Target list --------------------

func (q *omniQuerySpanExtractor) extractTargetList(list *ast.List) ([]base.QuerySpanResult, error) {
	if list == nil {
		return nil, nil
	}
	var results []base.QuerySpanResult
	for _, item := range list.Items {
		switch v := item.(type) {
		case *ast.ResTarget:
			if star, ok := v.Val.(*ast.StarExpr); ok {
				fields, err := q.expandStar(star)
				if err != nil {
					return nil, err
				}
				results = append(results, fields...)
				continue
			}
			r, err := q.resolveExpression(v.Val)
			if err != nil {
				return nil, err
			}
			if v.Name != "" {
				r.Name = normIdent(v.Name)
			}
			results = append(results, r)
		case *ast.StarExpr:
			fields, err := q.expandStar(v)
			if err != nil {
				return nil, err
			}
			results = append(results, fields...)
		case *ast.SelectAssign:
			// SELECT @v = expr — no result column produced; just fold the expr
			// for predicate/source tracking (omitted here since it doesn't emit a column).
			continue
		default:
			if expr, ok := item.(ast.ExprNode); ok {
				r, err := q.resolveExpression(expr)
				if err != nil {
					return nil, err
				}
				results = append(results, r)
				continue
			}
			return nil, unsupportedNodeError("unsupported SELECT target item", item)
		}
	}
	return results, nil
}

func (q *omniQuerySpanExtractor) expandStar(s *ast.StarExpr) ([]base.QuerySpanResult, error) {
	if s.Qualifier == "" {
		var out []base.QuerySpanResult
		for _, ts := range q.tableSourcesFrom {
			out = append(out, ts.GetQuerySpanResult()...)
		}
		return out, nil
	}
	return q.tsqlGetAllFieldsOfTableInFromOrOuterCTE("", "", s.Qualifier)
}

// -------------------- Expression resolver --------------------

// resolveExpression maps an omni ExprNode to a QuerySpanResult. This is the
// omni equivalent of the ANTLR getQuerySpanResultFromExpr.
func (q *omniQuerySpanExtractor) resolveExpression(expr ast.ExprNode) (base.QuerySpanResult, error) {
	return q.resolveExpressionNode(expr)
}

func (q *omniQuerySpanExtractor) resolveExpressionNode(n ast.Node) (base.QuerySpanResult, error) {
	if n == nil {
		return base.QuerySpanResult{SourceColumns: make(base.SourceColumnSet)}, nil
	}
	if r, handled, err := q.resolveSimpleExpressionNode(n); handled {
		return r, err
	}
	switch v := n.(type) {
	case *ast.ColumnRef:
		r, err := q.tsqlIsFieldSensitive(v.Database, v.Schema, v.Table, v.Column)
		if err != nil {
			return base.QuerySpanResult{}, errors.Wrapf(err, "failed to resolve column %q", v.Column)
		}
		r.IsPlainField = true
		return r, nil
	case *ast.AtTimeZoneExpr:
		r, err := q.mergeExprSources(v.Expr, v.TimeZone)
		r.Name = q.sliceName(n)
		return r, err
	case *ast.BinaryExpr:
		r, err := q.mergeExprSources(v.Left, v.Right)
		r.Name = q.sliceName(n)
		return r, err
	case *ast.UnaryExpr:
		r, err := q.resolveExpressionNode(v.Operand)
		if err != nil {
			return r, err
		}
		r.Name = q.sliceName(n)
		r.IsPlainField = false
		return r, nil
	case *ast.BetweenExpr:
		r, err := q.mergeExprSources(v.Expr, v.Low, v.High)
		r.Name = q.sliceName(n)
		return r, err
	case *ast.LikeExpr:
		r, err := q.mergeExprSources(v.Expr, v.Pattern, v.Escape)
		r.Name = q.sliceName(n)
		return r, err
	case *ast.IsExpr:
		r, err := q.resolveExpressionNode(v.Expr)
		if err != nil {
			return r, err
		}
		r.Name = q.sliceName(n)
		r.IsPlainField = false
		return r, nil
	case *ast.InExpr:
		exprs := []ast.ExprNode{v.Expr}
		if v.List != nil {
			for _, it := range v.List.Items {
				if e, ok := it.(ast.ExprNode); ok {
					exprs = append(exprs, e)
				}
			}
		}
		if v.Subquery != nil {
			exprs = append(exprs, v.Subquery)
		}
		r, err := q.mergeExprSources(exprs...)
		r.Name = q.sliceName(n)
		return r, err
	case *ast.CaseExpr:
		exprs := []ast.ExprNode{v.Arg, v.ElseExpr}
		if v.WhenList != nil {
			for _, it := range v.WhenList.Items {
				if cw, ok := it.(*ast.CaseWhen); ok {
					exprs = append(exprs, cw.Condition, cw.Result)
				}
			}
		}
		r, err := q.mergeExprSources(exprs...)
		r.Name = q.sliceName(n)
		return r, err
	case *ast.IifExpr:
		r, err := q.mergeExprSources(v.Condition, v.TrueVal, v.FalseVal)
		r.Name = q.sliceName(n)
		return r, err
	case *ast.CoalesceExpr:
		r, err := q.mergeListSources(v.Args)
		r.Name = q.sliceName(n)
		return r, err
	case *ast.NullifExpr:
		r, err := q.mergeExprSources(v.Left, v.Right)
		r.Name = q.sliceName(n)
		return r, err
	case *ast.FuncCallExpr:
		r, err := q.mergeListSources(v.Args)
		if err != nil {
			return r, err
		}
		// Also merge Over clause partition/order.
		if v.Over != nil {
			sub, err := q.mergeListSources(v.Over.PartitionBy)
			if err != nil {
				return r, err
			}
			r.SourceColumns, _ = base.MergeSourceColumnSet(r.SourceColumns, sub.SourceColumns)
			sub, err = q.mergeOrderBySources(v.Over.OrderBy)
			if err != nil {
				return r, err
			}
			r.SourceColumns, _ = base.MergeSourceColumnSet(r.SourceColumns, sub.SourceColumns)
		}
		if v.Within != nil {
			sub, err := q.mergeOrderBySources(v.Within)
			if err != nil {
				return r, err
			}
			r.SourceColumns, _ = base.MergeSourceColumnSet(r.SourceColumns, sub.SourceColumns)
		}
		r.Name = q.sliceName(n)
		return r, nil
	case *ast.MethodCallExpr:
		r, err := q.mergeListSources(v.Args)
		r.Name = q.sliceName(n)
		return r, err
	case *ast.CastExpr:
		r, err := q.resolveExpressionNode(v.Expr)
		if err != nil {
			return r, err
		}
		r.Name = q.sliceName(n)
		r.IsPlainField = false
		return r, nil
	case *ast.ConvertExpr:
		r, err := q.mergeExprSources(v.Expr, v.Style)
		r.Name = q.sliceName(n)
		return r, err
	case *ast.TryCastExpr:
		r, err := q.resolveExpressionNode(v.Expr)
		if err != nil {
			return r, err
		}
		r.Name = q.sliceName(n)
		r.IsPlainField = false
		return r, nil
	case *ast.TryConvertExpr:
		r, err := q.mergeExprSources(v.Expr, v.Style)
		r.Name = q.sliceName(n)
		return r, err
	case *ast.SubqueryExpr:
		return q.resolveSubqueryExpr(v)
	case *ast.SubqueryComparisonExpr:
		r, err := q.resolveExpressionNode(v.Left)
		if err != nil {
			return r, err
		}
		if body, ok := v.Subquery.(*ast.SelectStmt); ok {
			clone := q.cloneForSubquery()
			sub, subErr := clone.extractFromSelectStmt(body)
			if subErr != nil {
				return r, subErr
			}
			for _, sc := range sub.GetQuerySpanResult() {
				r.SourceColumns, _ = base.MergeSourceColumnSet(r.SourceColumns, sc.SourceColumns)
			}
			q.predicateColumns, _ = base.MergeSourceColumnSet(q.predicateColumns, clone.predicateColumns)
		}
		r.Name = q.sliceName(n)
		r.IsPlainField = false
		return r, nil
	case *ast.ExistsExpr:
		r := base.QuerySpanResult{
			Name:          q.sliceName(n),
			SourceColumns: make(base.SourceColumnSet),
		}
		if body, ok := v.Subquery.(*ast.SelectStmt); ok {
			clone := q.cloneForSubquery()
			if _, err := clone.extractFromSelectStmt(body); err != nil {
				return r, err
			}
			q.predicateColumns, _ = base.MergeSourceColumnSet(q.predicateColumns, clone.predicateColumns)
		}
		return r, nil
	case *ast.FullTextPredicate:
		r := base.QuerySpanResult{SourceColumns: make(base.SourceColumnSet)}
		if v.Columns != nil {
			for _, it := range v.Columns.Items {
				if cr, ok := it.(*ast.ColumnRef); ok {
					sub, err := q.tsqlIsFieldSensitive(cr.Database, cr.Schema, cr.Table, cr.Column)
					if err != nil {
						return r, errors.Wrapf(err, "failed to resolve full-text predicate column %q", cr.Column)
					}
					r.SourceColumns, _ = base.MergeSourceColumnSet(r.SourceColumns, sub.SourceColumns)
				}
			}
		}
		if v.Value != nil {
			sub, err := q.resolveExpressionNode(v.Value)
			if err != nil {
				return r, err
			}
			r.SourceColumns, _ = base.MergeSourceColumnSet(r.SourceColumns, sub.SourceColumns)
		}
		r.Name = q.sliceName(n)
		return r, nil
	case *ast.StarExpr:
		// Star in scalar context: treat as empty (already handled at target-list level).
		return base.QuerySpanResult{Name: q.sliceName(n), SourceColumns: make(base.SourceColumnSet)}, nil
	case *ast.GroupingSetsExpr, *ast.RollupExpr, *ast.CubeExpr:
		// Appear in GROUP BY; not typical targets. Fallback to empty.
		return base.QuerySpanResult{Name: q.sliceName(n), SourceColumns: make(base.SourceColumnSet)}, nil
	case *ast.ResTarget:
		r, err := q.resolveExpressionNode(v.Val)
		if err != nil {
			return r, err
		}
		if v.Name != "" {
			r.Name = v.Name
		}
		return r, nil
	case *ast.SelectAssign:
		return q.resolveExpressionNode(v.Value)
	case *ast.CaseWhen:
		return q.mergeExprSources(v.Condition, v.Result)
	case *ast.CurrentOfExpr:
		return base.QuerySpanResult{Name: q.sliceName(n), SourceColumns: make(base.SourceColumnSet)}, nil
	default:
		return base.QuerySpanResult{Name: q.sliceName(n), SourceColumns: make(base.SourceColumnSet)}, nil
	}
}

func (q *omniQuerySpanExtractor) resolveSimpleExpressionNode(n ast.Node) (base.QuerySpanResult, bool, error) {
	switch v := n.(type) {
	case *ast.Literal, *ast.VariableRef, *ast.Boolean:
		return base.QuerySpanResult{Name: q.sliceName(v), SourceColumns: make(base.SourceColumnSet)}, true, nil
	case *ast.ParenExpr:
		r, err := q.resolveExpressionNode(v.Expr)
		return r, true, err
	case *ast.CollateExpr:
		r, err := q.resolveExpressionNode(v.Expr)
		return r, true, err
	default:
		return base.QuerySpanResult{}, false, nil
	}
}

func (q *omniQuerySpanExtractor) resolveSubqueryExpr(sq *ast.SubqueryExpr) (base.QuerySpanResult, error) {
	r := base.QuerySpanResult{
		Name:          q.sliceName(sq),
		SourceColumns: make(base.SourceColumnSet),
	}
	body, ok := sq.Query.(*ast.SelectStmt)
	if !ok {
		return r, nil
	}
	clone := q.cloneForSubquery()
	pseudo, err := clone.extractFromSelectStmt(body)
	if err != nil {
		return r, err
	}
	// Scalar subquery contributes its first result column's source columns.
	results := pseudo.GetQuerySpanResult()
	for _, c := range results {
		r.SourceColumns, _ = base.MergeSourceColumnSet(r.SourceColumns, c.SourceColumns)
	}
	q.predicateColumns, _ = base.MergeSourceColumnSet(q.predicateColumns, clone.predicateColumns)
	return r, nil
}

func (q *omniQuerySpanExtractor) mergeExprSources(exprs ...ast.ExprNode) (base.QuerySpanResult, error) {
	r := base.QuerySpanResult{SourceColumns: make(base.SourceColumnSet)}
	for _, e := range exprs {
		if e == nil {
			continue
		}
		sub, err := q.resolveExpressionNode(e)
		if err != nil {
			return r, err
		}
		r.SourceColumns, _ = base.MergeSourceColumnSet(r.SourceColumns, sub.SourceColumns)
	}
	return r, nil
}

func (q *omniQuerySpanExtractor) mergeListSources(list *ast.List) (base.QuerySpanResult, error) {
	r := base.QuerySpanResult{SourceColumns: make(base.SourceColumnSet)}
	if list == nil {
		return r, nil
	}
	for _, it := range list.Items {
		if e, ok := it.(ast.ExprNode); ok {
			sub, err := q.resolveExpressionNode(e)
			if err != nil {
				return r, err
			}
			r.SourceColumns, _ = base.MergeSourceColumnSet(r.SourceColumns, sub.SourceColumns)
		}
	}
	return r, nil
}

func (q *omniQuerySpanExtractor) mergeOrderBySources(list *ast.List) (base.QuerySpanResult, error) {
	r := base.QuerySpanResult{SourceColumns: make(base.SourceColumnSet)}
	if list == nil {
		return r, nil
	}
	for _, it := range list.Items {
		var expr ast.Node
		switch v := it.(type) {
		case *ast.OrderByItem:
			expr = v.Expr
		case ast.ExprNode:
			expr = v
		default:
			return r, unsupportedNodeError("unsupported ORDER BY item", it)
		}
		sub, err := q.resolveExpressionNode(expr)
		if err != nil {
			return r, err
		}
		r.SourceColumns, _ = base.MergeSourceColumnSet(r.SourceColumns, sub.SourceColumns)
	}
	return r, nil
}

// -------------------- Helpers --------------------

// sliceName returns the whitespace-stripped source text covering a node's Loc.
// Mirrors ANTLR's ctx.GetText() convention for naming expression results.
// For *SubqueryExpr (and similar paren-wrapped nodes) the Loc spans the
// surrounding parens; we strip a matching leading `(` / trailing `)` pair to
// match the ANTLR subquery.getText() output.
func (q *omniQuerySpanExtractor) sliceName(n ast.Node) string {
	loc := omniNodeLoc(n)
	if loc.Start < 0 || loc.End < 0 || loc.Start >= len(q.source) {
		return ""
	}
	end := loc.End
	if end > len(q.source) {
		end = len(q.source)
	}
	text := stripSpaces(q.source[loc.Start:end])
	switch n.(type) {
	case *ast.SubqueryExpr, *ast.ExistsExpr, *ast.SubqueryComparisonExpr:
		if len(text) >= 2 && text[0] == '(' && text[len(text)-1] == ')' {
			text = text[1 : len(text)-1]
		}
	default:
	}
	return text
}

func stripSpaces(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		if !unicode.IsSpace(r) {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// normIdent lowercases a SQL-text-sourced identifier (table alias, column
// alias, CTE name, ResTarget name). Matches the legacy ANTLR
// NormalizeTSQLIdentifier behavior which lowercases unconditionally. Does not
// apply to identifiers sourced from metadata (physical table/column names);
// those are compared case-insensitively but stored as-is.
func normIdent(s string) string {
	return strings.ToLower(s)
}

// omniNodeLoc extracts the Loc field from an ast.Node via type switching.
// Not every node has a Loc (the Node interface doesn't expose it), so we
// enumerate the types we care about. Returns zero Loc otherwise.
func omniNodeLoc(n ast.Node) ast.Loc {
	if n == nil {
		return ast.NoLoc()
	}
	if loc, ok := omniExpressionNodeLoc(n); ok {
		return loc
	}
	switch v := n.(type) {
	case *ast.SelectStmt:
		return v.Loc
	case *ast.TableRef:
		return v.Loc
	case *ast.ResTarget:
		return v.Loc
	case *ast.SelectAssign:
		return v.Loc
	default:
		return ast.NoLoc()
	}
}

func omniExpressionNodeLoc(n ast.Node) (ast.Loc, bool) {
	switch v := n.(type) {
	case *ast.ColumnRef:
		return v.Loc, true
	case *ast.Literal:
		return v.Loc, true
	case *ast.VariableRef:
		return v.Loc, true
	case *ast.StarExpr:
		return v.Loc, true
	case *ast.BinaryExpr:
		return v.Loc, true
	case *ast.UnaryExpr:
		return v.Loc, true
	case *ast.BetweenExpr:
		return v.Loc, true
	case *ast.LikeExpr:
		return v.Loc, true
	case *ast.IsExpr:
		return v.Loc, true
	case *ast.InExpr:
		return v.Loc, true
	case *ast.CaseExpr:
		return v.Loc, true
	case *ast.CaseWhen:
		return v.Loc, true
	case *ast.IifExpr:
		return v.Loc, true
	case *ast.CoalesceExpr:
		return v.Loc, true
	case *ast.NullifExpr:
		return v.Loc, true
	case *ast.FuncCallExpr:
		return v.Loc, true
	case *ast.MethodCallExpr:
		return v.Loc, true
	case *ast.CastExpr:
		return v.Loc, true
	case *ast.ConvertExpr:
		return v.Loc, true
	case *ast.TryCastExpr:
		return v.Loc, true
	case *ast.TryConvertExpr:
		return v.Loc, true
	case *ast.SubqueryExpr:
		return v.Loc, true
	case *ast.SubqueryComparisonExpr:
		return v.Loc, true
	case *ast.ExistsExpr:
		return v.Loc, true
	case *ast.FullTextPredicate:
		return v.Loc, true
	case *ast.ParenExpr:
		return v.Loc, true
	case *ast.CollateExpr:
		return v.Loc, true
	case *ast.AtTimeZoneExpr:
		return v.Loc, true
	default:
		return ast.NoLoc(), false
	}
}

// -------------------- Access tables --------------------

// collectOmniAccessTables walks the AST and records every table-source
// position reference. Mirrors the ANTLR accessTableListener.
func collectOmniAccessTables(root ast.Node, currentDatabase, currentSchema string) base.SourceColumnSet {
	out := make(base.SourceColumnSet)
	walkAccessTables(root, currentDatabase, currentSchema, out)
	return out
}

func walkAccessTables(n ast.Node, db, schema string, out base.SourceColumnSet) {
	if n == nil {
		return
	}
	switch v := n.(type) {
	case *ast.SelectStmt:
		if v.WithClause != nil {
			walkAccessTables(v.WithClause, db, schema, out)
		}
		if v.FromClause != nil {
			for _, it := range v.FromClause.Items {
				walkAccessTables(it, db, schema, out)
			}
		}
		if v.WhereClause != nil {
			walkAccessTables(v.WhereClause, db, schema, out)
		}
		if v.HavingClause != nil {
			walkAccessTables(v.HavingClause, db, schema, out)
		}
		if v.TargetList != nil {
			for _, it := range v.TargetList.Items {
				walkAccessTables(it, db, schema, out)
			}
		}
		if v.Larg != nil {
			walkAccessTables(v.Larg, db, schema, out)
		}
		if v.Rarg != nil {
			walkAccessTables(v.Rarg, db, schema, out)
		}
	case *ast.WithClause:
		if v.CTEs != nil {
			for _, it := range v.CTEs.Items {
				if c, ok := it.(*ast.CommonTableExpr); ok {
					walkAccessTables(c.Query, db, schema, out)
				}
			}
		}
	case *ast.TableRef:
		addAccessTable(v, db, schema, out)
	case *ast.AliasedTableRef:
		walkAccessTables(v.Table, db, schema, out)
	case *ast.JoinClause:
		walkAccessTables(v.Left, db, schema, out)
		walkAccessTables(v.Right, db, schema, out)
		if v.Condition != nil {
			walkAccessTables(v.Condition, db, schema, out)
		}
	case *ast.SubqueryExpr:
		walkAccessTables(v.Query, db, schema, out)
	case *ast.PivotExpr:
		walkAccessTables(v.Source, db, schema, out)
	case *ast.UnpivotExpr:
		walkAccessTables(v.Source, db, schema, out)
	case *ast.ExistsExpr:
		walkAccessTables(v.Subquery, db, schema, out)
	case *ast.SubqueryComparisonExpr:
		walkAccessTables(v.Left, db, schema, out)
		walkAccessTables(v.Subquery, db, schema, out)
	case *ast.InExpr:
		walkAccessTables(v.Expr, db, schema, out)
		if v.List != nil {
			for _, it := range v.List.Items {
				walkAccessTables(it, db, schema, out)
			}
		}
		walkAccessTables(v.Subquery, db, schema, out)
	case *ast.BinaryExpr:
		walkAccessTables(v.Left, db, schema, out)
		walkAccessTables(v.Right, db, schema, out)
	case *ast.UnaryExpr:
		walkAccessTables(v.Operand, db, schema, out)
	case *ast.BetweenExpr:
		walkAccessTables(v.Expr, db, schema, out)
		walkAccessTables(v.Low, db, schema, out)
		walkAccessTables(v.High, db, schema, out)
	case *ast.LikeExpr:
		walkAccessTables(v.Expr, db, schema, out)
		walkAccessTables(v.Pattern, db, schema, out)
	case *ast.IsExpr:
		walkAccessTables(v.Expr, db, schema, out)
	case *ast.CaseExpr:
		walkAccessTables(v.Arg, db, schema, out)
		walkAccessTables(v.ElseExpr, db, schema, out)
		if v.WhenList != nil {
			for _, it := range v.WhenList.Items {
				walkAccessTables(it, db, schema, out)
			}
		}
	case *ast.CaseWhen:
		walkAccessTables(v.Condition, db, schema, out)
		walkAccessTables(v.Result, db, schema, out)
	case *ast.ParenExpr:
		walkAccessTables(v.Expr, db, schema, out)
	case *ast.CastExpr:
		walkAccessTables(v.Expr, db, schema, out)
	case *ast.ConvertExpr:
		walkAccessTables(v.Expr, db, schema, out)
	case *ast.TryCastExpr:
		walkAccessTables(v.Expr, db, schema, out)
	case *ast.TryConvertExpr:
		walkAccessTables(v.Expr, db, schema, out)
	case *ast.FuncCallExpr:
		if v.Args != nil {
			for _, it := range v.Args.Items {
				walkAccessTables(it, db, schema, out)
			}
		}
	case *ast.ResTarget:
		walkAccessTables(v.Val, db, schema, out)
	case *ast.ValuesClause:
		// No table access.
	case *ast.TableVarRef, *ast.TableVarMethodCallRef:
		// Temp-table var: no physical access.
	default:
	}
}

func addAccessTable(t *ast.TableRef, defDB, defSchema string, out base.SourceColumnSet) {
	if t == nil || strings.HasPrefix(t.Object, "#") {
		return
	}
	database := defDB
	if t.Database != "" {
		database = t.Database
	}
	schema := defSchema
	if t.Schema != "" {
		schema = t.Schema
	}
	out[base.ColumnResource{
		Server:   t.Server,
		Database: database,
		Schema:   schema,
		Table:    t.Object,
	}] = true
}
