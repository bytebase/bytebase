package pg

import (
	"context"

	"fmt"
	"strings"

	"github.com/pkg/errors"

	"github.com/bytebase/omni/pg/ast"
	"github.com/bytebase/omni/pg/catalog"
	omniparser "github.com/bytebase/omni/pg/parser"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/store/model"
)

// omniQuerySpanExtractor extracts query span information using the omni parser
// and catalog, replacing the ANTLR-based querySpanExtractor.
type omniQuerySpanExtractor struct {
	ctx             context.Context
	gCtx            base.GetQuerySpanContext
	defaultDatabase string
	searchPath      []string
	// metaCache is a lazy-load cache for database metadata.
	// Use getDatabaseMetadata() instead of accessing directly.
	metaCache     map[string]*model.DatabaseMetadata
	cat           *catalog.Catalog
	funcBodyCache map[uint32][]base.SourceColumnSet
	// funcOrigDefs stores the original (non-stubbed) function definitions keyed
	// by lowercase function name. Used by analyzeFunctionBody to get the real
	// body when it was stubbed during catalog loading.
	funcOrigDefs map[string]string
	// funcSourceColumns accumulates table-level access (column="") discovered inside
	// function bodies. Merged into the top-level QuerySpan.SourceColumns.
	funcSourceColumns base.SourceColumnSet
	// funcPredicateColumns accumulates columns used in WHERE/JOIN conditions
	// inside function bodies. Merged into the top-level QuerySpan.PredicateColumns.
	funcPredicateColumns base.SourceColumnSet
}

// newOmniQuerySpanExtractor creates a new omni-based query span extractor.
func newOmniQuerySpanExtractor(defaultDatabase string, searchPath []string, gCtx base.GetQuerySpanContext) *omniQuerySpanExtractor {
	if len(searchPath) == 0 {
		searchPath = []string{"public"}
	}
	return &omniQuerySpanExtractor{
		defaultDatabase:      defaultDatabase,
		searchPath:           searchPath,
		gCtx:                 gCtx,
		metaCache:            make(map[string]*model.DatabaseMetadata),
		funcBodyCache:        make(map[uint32][]base.SourceColumnSet),
		funcOrigDefs:         make(map[string]string),
		funcSourceColumns:    make(base.SourceColumnSet),
		funcPredicateColumns: make(base.SourceColumnSet),
	}
}

// getDatabaseMetadata returns cached database metadata, fetching it if not yet cached.
func (e *omniQuerySpanExtractor) getDatabaseMetadata(database string) (*model.DatabaseMetadata, error) {
	if meta, ok := e.metaCache[database]; ok {
		return meta, nil
	}
	_, meta, err := e.gCtx.GetDatabaseMetadataFunc(e.ctx, e.gCtx.InstanceID, database)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get database metadata for database: %s", database)
	}
	e.metaCache[database] = meta
	return meta, nil
}

// initCatalog initializes the omni catalog from the database metadata.
// It generates minimal DDL from the metadata proto and loads it into the catalog.
// We generate DDL directly here instead of using schema.GetDatabaseDefinition
// to avoid circular imports (schema/pg imports parser/pg).
func (e *omniQuerySpanExtractor) initCatalog() error {
	meta, err := e.getDatabaseMetadata(e.defaultDatabase)
	if err != nil {
		return errors.Wrapf(err, "failed to get database metadata for catalog init")
	}

	e.cat = catalog.New()
	e.cat.SetSearchPath(e.searchPath)

	if meta == nil {
		return nil
	}

	ddl := buildMinimalDDL(meta.GetProto())
	if ddl != "" {
		if _, err := e.cat.Exec(ddl, &catalog.ExecOptions{ContinueOnError: true}); err != nil {
			return errors.Wrapf(err, "failed to load schema DDL into catalog")
		}
	}

	// Store original (non-stubbed) function definitions for body analysis.
	for _, s := range meta.GetProto().GetSchemas() {
		for _, f := range s.GetFunctions() {
			if f.Definition != "" {
				origBody := extractFuncBodyFromDef(f.Definition)
				if origBody != "" {
					e.funcOrigDefs[strings.ToLower(f.Name)] = origBody
				}
			}
		}
	}

	return nil
}

// buildMinimalDDL generates CREATE SCHEMA / CREATE TABLE / CREATE VIEW / CREATE FUNCTION
// statements from metadata, sufficient for omni's AnalyzeSelectStmt to resolve columns.
func buildMinimalDDL(meta *storepb.DatabaseSchemaMetadata) string {
	if meta == nil {
		return ""
	}
	var b strings.Builder
	for _, s := range meta.Schemas {
		schemaName := s.Name
		if schemaName != "public" && schemaName != "pg_catalog" {
			fmt.Fprintf(&b, "CREATE SCHEMA IF NOT EXISTS %s;\n", quoteIdentifier(schemaName))
		}
		for _, t := range s.Tables {
			buildCreateTable(&b, schemaName, t)
		}
		for _, v := range s.Views {
			buildCreateView(&b, schemaName, v)
		}
		for _, v := range s.MaterializedViews {
			buildCreateMaterializedView(&b, schemaName, v)
		}
		for _, f := range s.Functions {
			if f.Definition != "" {
				// Load the function with a stubbed body to avoid type validation
				// failures. The catalog needs the function signature (name, params,
				// return type) to resolve SELECT * FROM func(), but the actual body
				// is analyzed separately via analyzeFunctionBody. Stubbing the body
				// avoids errors when table column types default to text but the
				// function declares int parameters.
				stubbed := stubFunctionBody(f.Definition)
				fmt.Fprintf(&b, "%s;\n", strings.TrimSuffix(strings.TrimSpace(stubbed), ";"))
			}
		}
	}
	return b.String()
}

func buildCreateTable(b *strings.Builder, schema string, t *storepb.TableMetadata) {
	fmt.Fprintf(b, "CREATE TABLE %s.%s (\n", quoteIdentifier(schema), quoteIdentifier(t.Name))
	for i, c := range t.Columns {
		colType := c.Type
		if colType == "" {
			colType = "text"
		}
		fmt.Fprintf(b, "  %s %s", quoteIdentifier(c.Name), colType)
		if i < len(t.Columns)-1 {
			b.WriteString(",")
		}
		b.WriteString("\n")
	}
	b.WriteString(");\n")
}

func buildCreateView(b *strings.Builder, schema string, v *storepb.ViewMetadata) {
	if v.Definition == "" {
		return
	}
	def := strings.TrimSuffix(strings.TrimSpace(v.Definition), ";")
	fmt.Fprintf(b, "CREATE VIEW %s.%s AS %s;\n", quoteIdentifier(schema), quoteIdentifier(v.Name), def)
}

func buildCreateMaterializedView(b *strings.Builder, schema string, v *storepb.MaterializedViewMetadata) {
	if v.Definition == "" {
		return
	}
	def := strings.TrimSuffix(strings.TrimSpace(v.Definition), ";")
	fmt.Fprintf(b, "CREATE MATERIALIZED VIEW %s.%s AS %s;\n", quoteIdentifier(schema), quoteIdentifier(v.Name), def)
}

func quoteIdentifier(name string) string {
	return `"` + strings.ReplaceAll(name, `"`, `""`) + `"`
}

// extractFuncBodyFromDef extracts the dollar-quoted function body from a
// CREATE FUNCTION definition string.
func extractFuncBodyFromDef(definition string) string {
	// Find dollar-quoted body.
	tag, bodyStart, bodyEnd := findDollarQuotedBody(definition)
	if tag == "" {
		return ""
	}
	return definition[bodyStart : bodyEnd-len(tag)]
}

// findDollarQuotedBody locates the dollar-quoted body in a function definition.
// Returns the tag, the start of body content (after opening tag), and the end
// position (after closing tag).
func findDollarQuotedBody(definition string) (tag string, bodyStart int, bodyEnd int) {
	for i := 0; i < len(definition); i++ {
		if definition[i] != '$' {
			continue
		}
		// Find end of tag.
		tagEnd := i + 1
		for tagEnd < len(definition) && definition[tagEnd] != '$' {
			tagEnd++
		}
		if tagEnd >= len(definition) {
			continue
		}
		tag := definition[i : tagEnd+1]
		contentStart := tagEnd + 1
		// Find closing tag.
		closeIdx := strings.Index(definition[contentStart:], tag)
		if closeIdx >= 0 {
			return tag, contentStart, contentStart + closeIdx + len(tag)
		}
	}
	return "", 0, 0
}

// stubFunctionBody replaces the body of a CREATE FUNCTION statement with a
// minimal stub. This avoids catalog type validation failures when table column
// types are unknown (defaulting to text) while the function declares specific
// return types. The function signature is preserved so the catalog can resolve
// function calls.
func stubFunctionBody(definition string) string {
	tag, bodyStart, bodyEnd := findDollarQuotedBody(definition)
	if tag == "" {
		return definition
	}

	// Determine language for appropriate stub body.
	upper := strings.ToUpper(definition)
	lang := ""
	if langIdx := strings.Index(upper, "LANGUAGE"); langIdx >= 0 {
		rest := strings.TrimSpace(definition[langIdx+8:])
		parts := strings.Fields(rest)
		if len(parts) > 0 {
			lang = strings.ToLower(strings.TrimRight(parts[0], ";"))
		}
	}

	var stubBody string
	switch lang {
	case "plpgsql":
		stubBody = " BEGIN RETURN; END; "
	default:
		stubBody = " SELECT NULL "
	}

	return definition[:bodyStart] + stubBody + definition[bodyEnd-len(tag):]
}

// getQuerySpan extracts the query span for the given SQL statement.
func (e *omniQuerySpanExtractor) getQuerySpan(ctx context.Context, stmt string) (*base.QuerySpan, error) {
	e.ctx = ctx

	// Step 1: Parse with omni.
	omniStmts, err := ParsePg(stmt)
	if err != nil {
		// Convert omni parse error to *base.SyntaxError with position info
		// so the SQL service can populate DetailedError.SyntaxError.
		syntaxErr := &base.SyntaxError{
			Message:  err.Error(),
			Position: &storepb.Position{Line: 1, Column: 1},
		}
		var parseErr *omniparser.ParseError
		if errors.As(err, &parseErr) && parseErr.Position > 0 {
			syntaxErr.Position = ByteOffsetToRunePosition(stmt, parseErr.Position-1)
		}
		return nil, syntaxErr
	}
	if len(omniStmts) != 1 {
		return nil, errors.Errorf("expected exactly 1 statement, got %d", len(omniStmts))
	}

	// Step 2: Extract access tables (non-fatal on error — some statement
	// types like SET don't have table references).
	accessesMap := make(base.SourceColumnSet)
	accessTables, err := ExtractAccessTables(stmt, ExtractAccessTablesOption{
		DefaultDatabase:        e.defaultDatabase,
		DefaultSchema:          e.searchPath[0],
		GetDatabaseMetadata:    e.gCtx.GetDatabaseMetadataFunc,
		Ctx:                    ctx,
		InstanceID:             e.gCtx.InstanceID,
		SkipMetadataValidation: false,
	})
	if err == nil {
		for _, resource := range accessTables {
			accessesMap[resource] = true
		}
	}

	// Step 3: Check for mixed system/user tables.
	allSystems, mixed := isMixedQuery(accessesMap)
	if mixed {
		return nil, base.MixUserSystemTablesError
	}

	// Step 4: Classify query type.
	queryType, isExplainAnalyze := classifyQueryType(omniStmts[0].AST, allSystems)

	// Step 5: Return early for non-SELECT queries.
	if queryType != base.Select {
		return &base.QuerySpan{
			Type:          queryType,
			SourceColumns: base.SourceColumnSet{},
			Results:       []base.QuerySpanResult{},
		}, nil
	}

	// For EXPLAIN ANALYZE SELECT, return with source columns but no results.
	if isExplainAnalyze {
		return &base.QuerySpan{
			Type:          queryType,
			SourceColumns: accessesMap,
			Results:       []base.QuerySpanResult{},
		}, nil
	}

	// Step 6: Cast to SelectStmt, init catalog, analyze.
	selStmt, ok := omniStmts[0].AST.(*ast.SelectStmt)
	if !ok {
		// Not a SelectStmt (e.g., SET, SHOW classified as Select type).
		return &base.QuerySpan{
			Type:          base.Select,
			SourceColumns: accessesMap,
			Results:       []base.QuerySpanResult{},
		}, nil
	}

	if err := e.initCatalog(); err != nil {
		return nil, errors.Wrapf(err, "failed to initialize catalog")
	}

	query, err := e.cat.AnalyzeSelectStmt(selStmt)
	if err != nil {
		// Before falling back, try to handle user-defined table-returning functions
		// that omni's AnalyzeSelectStmt can't resolve (e.g., RETURNS TABLE functions
		// used as table sources: SELECT * FROM func()).
		if results := e.tryUserFuncTableSource(selStmt, accessesMap); results != nil {
			return &base.QuerySpan{
				Type:          base.Select,
				SourceColumns: accessesMap,
				Results:       results,
			}, nil
		}
		// Fail-open: return access tables with best-effort column names and lineage
		// when analysis fails (e.g., unsupported built-in functions).
		return &base.QuerySpan{
			Type:             base.Select,
			SourceColumns:    accessesMap,
			Results:          e.extractFallbackColumns(selStmt),
			PredicateColumns: e.funcPredicateColumns,
		}, nil
	}

	// Step 7: Extract lineage from analyzed query.
	// SourceColumns (table-level access) is already captured in accessesMap
	// from ExtractAccessTables above — no need to re-extract from the query.
	results := e.extractLineage(query, selStmt)

	// Merge columns discovered inside function bodies into the top-level sets.
	for col := range e.funcSourceColumns {
		accessesMap[col] = true
	}

	return &base.QuerySpan{
		Type:             base.Select,
		SourceColumns:    accessesMap,
		PredicateColumns: e.funcPredicateColumns,
		Results:          results,
	}, nil
}

// extractLineage extracts the result column lineage from an analyzed query.
// selStmt is the original parse tree, used to determine IsPlainField
// (which is true only for columns that came from * expansion).
func (e *omniQuerySpanExtractor) extractLineage(q *catalog.Query, selStmt *ast.SelectStmt) []base.QuerySpanResult {
	// Handle set operations (UNION/INTERSECT/EXCEPT).
	if q.SetOp != catalog.SetOpNone {
		return e.extractSetOpLineage(q, selStmt)
	}

	// Build a plain-field mask from the parse tree's target list.
	// IsPlainField = true only for columns expanded from SELECT * or SELECT t.*.
	plainMask := buildPlainFieldMask(selStmt, q)

	var results []base.QuerySpanResult
	idx := 0
	for _, te := range q.TargetList {
		if te.ResJunk {
			continue
		}
		sourceColSet := make(base.SourceColumnSet)
		e.walkExpr(q, te.Expr, sourceColSet)
		isPlain := idx < len(plainMask) && plainMask[idx]
		// Even if the parse tree says this came from *, the underlying
		// expression through CTEs/subqueries may not be a simple column ref.
		if isPlain {
			isPlain = isUltimatelyPlainColumn(q, te.Expr)
		}
		results = append(results, base.QuerySpanResult{
			Name:          te.ResName,
			SourceColumns: sourceColSet,
			IsPlainField:  isPlain,
		})
		idx++
	}
	return results
}

// isUltimatelyPlainColumn checks whether an expression, after resolving through
// CTEs and subqueries, is a simple column reference (VarExpr pointing to a
// physical table). Aggregates, functions, and other expressions return false.
func isUltimatelyPlainColumn(q *catalog.Query, expr catalog.AnalyzedExpr) bool {
	if expr == nil {
		return false
	}
	v, ok := expr.(*catalog.VarExpr)
	if !ok {
		return false
	}
	if v.RangeIdx < 0 || v.RangeIdx >= len(q.RangeTable) {
		return false
	}
	rte := q.RangeTable[v.RangeIdx]
	colIdx := int(v.AttNum - 1)

	switch rte.Kind {
	case catalog.RTERelation:
		return true
	case catalog.RTESubquery:
		if rte.Subquery != nil && colIdx >= 0 && colIdx < len(rte.Subquery.TargetList) {
			return isUltimatelyPlainColumn(rte.Subquery, rte.Subquery.TargetList[colIdx].Expr)
		}
	case catalog.RTECTE:
		if rte.CTEIndex >= 0 && rte.CTEIndex < len(q.CTEList) {
			cte := q.CTEList[rte.CTEIndex]
			if cte.Query != nil && colIdx >= 0 && colIdx < len(cte.Query.TargetList) {
				return isUltimatelyPlainColumn(cte.Query, cte.Query.TargetList[colIdx].Expr)
			}
		}
	default:
		return false
	}
	return false
}

// buildPlainFieldMask returns a boolean slice aligned with the analyzed
// query's non-junk TargetList. An entry is true if the column was produced
// by * expansion (SELECT * or SELECT t.*), false for explicit column refs.
func buildPlainFieldMask(selStmt *ast.SelectStmt, q *catalog.Query) []bool {
	// Collect non-junk target entries.
	var targets []*catalog.TargetEntry
	for _, te := range q.TargetList {
		if !te.ResJunk {
			targets = append(targets, te)
		}
	}
	nResults := len(targets)

	if selStmt == nil || selStmt.TargetList == nil {
		return make([]bool, nResults)
	}

	// Count parse-tree items (stars and non-stars).
	var parseTargets []*ast.ResTarget
	for _, item := range selStmt.TargetList.Items {
		if rt, ok := item.(*ast.ResTarget); ok {
			parseTargets = append(parseTargets, rt)
		}
	}

	// For each star, determine how many columns it expands to by
	// looking at the table it references and counting columns in that RTE.
	mask := make([]bool, nResults)
	pos := 0

	for _, rt := range parseTargets {
		if pos >= nResults {
			break
		}
		if isStarTarget(rt) {
			tableName := starTableName(rt)
			starCount := countStarColumnsFromRTE(q, targets, pos, tableName)
			for i := 0; i < starCount && pos < nResults; i++ {
				mask[pos] = true
				pos++
			}
		} else {
			mask[pos] = false
			pos++
		}
	}

	return mask
}

// isStarTarget checks if a ResTarget in the parse tree is a star expression
// (SELECT * or SELECT t.*).
func isStarTarget(rt *ast.ResTarget) bool {
	if rt == nil || rt.Val == nil {
		return false
	}
	cr, ok := rt.Val.(*ast.ColumnRef)
	if !ok || cr.Fields == nil || len(cr.Fields.Items) == 0 {
		return false
	}
	last := cr.Fields.Items[len(cr.Fields.Items)-1]
	_, isStar := last.(*ast.A_Star)
	return isStar
}

// starTableName extracts the table qualifier from a star target.
// For "SELECT *" returns "", for "SELECT t.*" returns "t".
func starTableName(rt *ast.ResTarget) string {
	cr, ok := rt.Val.(*ast.ColumnRef)
	if !ok || cr.Fields == nil {
		return ""
	}
	// For t.*, Fields = [String("t"), A_Star{}]
	if len(cr.Fields.Items) >= 2 {
		if s, ok := cr.Fields.Items[0].(*ast.String); ok {
			return s.Str
		}
	}
	return ""
}

// countStarColumnsFromRTE counts how many analyzed target entries starting
// at pos belong to a star expansion. For unqualified * (tableName=""), it
// counts entries that share the same contiguous RangeIdx sequence covering
// all non-join RTEs. For qualified t.* (tableName="t"), it counts entries
// whose RTE ERef matches the table name.
func countStarColumnsFromRTE(q *catalog.Query, targets []*catalog.TargetEntry, startPos int, tableName string) int {
	if startPos >= len(targets) {
		return 0
	}

	if tableName != "" {
		// Qualified star: count consecutive entries from the named RTE.
		count := 0
		for i := startPos; i < len(targets); i++ {
			v, ok := targets[i].Expr.(*catalog.VarExpr)
			if !ok {
				break
			}
			if v.RangeIdx < 0 || v.RangeIdx >= len(q.RangeTable) {
				break
			}
			rte := q.RangeTable[v.RangeIdx]
			if rte.ERef != tableName && rte.Alias != tableName {
				break
			}
			count++
		}
		if count == 0 {
			return 1
		}
		return count
	}

	// Unqualified star: expands all non-join RTEs in FROM order.
	// Count how many consecutive VarExprs follow the expected star expansion pattern.
	count := 0
	for i := startPos; i < len(targets); i++ {
		v, ok := targets[i].Expr.(*catalog.VarExpr)
		if !ok {
			break
		}
		if v.RangeIdx < 0 || v.RangeIdx >= len(q.RangeTable) {
			break
		}
		rte := q.RangeTable[v.RangeIdx]
		if rte.Kind == catalog.RTEJoin {
			break
		}
		// Check if this column's AttNum is sequential within its RTE.
		// Star expansion produces columns in order: attnum 1, 2, 3, ...
		if count > 0 {
			if prev, ok := targets[i-1].Expr.(*catalog.VarExpr); ok && v.RangeIdx == prev.RangeIdx && v.AttNum != prev.AttNum+1 {
				break
			}
		}
		count++
	}
	if count == 0 {
		return 1
	}
	return count
}

// extractSetOpLineage handles UNION/INTERSECT/EXCEPT by merging lineage from both branches.
func (e *omniQuerySpanExtractor) extractSetOpLineage(q *catalog.Query, selStmt *ast.SelectStmt) []base.QuerySpanResult {
	if q.LArg == nil || q.RArg == nil {
		return nil
	}
	lResults := e.extractLineage(q.LArg, selStmt.Larg)
	rResults := e.extractLineage(q.RArg, selStmt.Rarg)

	// For EXCEPT, output rows come only from the left branch — do not
	// merge right-side lineage into the result columns.
	includeRight := q.SetOp != catalog.SetOpExcept && q.SetOp != catalog.SetOpExceptAll

	var results []base.QuerySpanResult
	for i, lr := range lResults {
		merged := make(base.SourceColumnSet)
		for col := range lr.SourceColumns {
			merged[col] = true
		}
		if includeRight && i < len(rResults) {
			for col := range rResults[i].SourceColumns {
				merged[col] = true
			}
		}
		results = append(results, base.QuerySpanResult{
			Name:          lr.Name,
			SourceColumns: merged,
			IsPlainField:  false, // set-op results are never plain fields
		})
	}
	return results
}

// walkExpr recursively walks an analyzed expression tree and collects all
// source column references into the result set.
func (e *omniQuerySpanExtractor) walkExpr(q *catalog.Query, expr catalog.AnalyzedExpr, result base.SourceColumnSet) {
	if expr == nil {
		return
	}
	switch v := expr.(type) {
	case *catalog.VarExpr:
		e.resolveVar(q, v, result)
	case *catalog.FuncCallExpr:
		// Walk arguments first — they contribute to access tables.
		for _, arg := range v.Args {
			e.walkExpr(q, arg, result)
		}
		// Trace lineage through user-defined function body.
		if proc := e.cat.GetUserProcByOID(v.FuncOID); proc != nil {
			// Expression context: merge all output columns into one set.
			for _, colSet := range e.analyzeFunctionBody(proc) {
				for k, val := range colSet {
					result[k] = val
				}
			}
		}
	case *catalog.AggExpr:
		for _, arg := range v.Args {
			e.walkExpr(q, arg, result)
		}
	case *catalog.OpExpr:
		e.walkExpr(q, v.Left, result)
		e.walkExpr(q, v.Right, result)
	case *catalog.BoolExprQ:
		for _, arg := range v.Args {
			e.walkExpr(q, arg, result)
		}
	case *catalog.CaseExprQ:
		e.walkExpr(q, v.Arg, result)
		for _, w := range v.When {
			e.walkExpr(q, w.Condition, result)
			e.walkExpr(q, w.Result, result)
		}
		e.walkExpr(q, v.Default, result)
	case *catalog.CoalesceExprQ:
		for _, arg := range v.Args {
			e.walkExpr(q, arg, result)
		}
	case *catalog.SubLinkExpr:
		e.walkExpr(q, v.TestExpr, result)
		if v.SubQuery != nil {
			for _, te := range v.SubQuery.TargetList {
				if !te.ResJunk {
					e.walkExpr(v.SubQuery, te.Expr, result)
				}
			}
		}
	case *catalog.RelabelExpr:
		e.walkExpr(q, v.Arg, result)
	case *catalog.CoerceViaIOExpr:
		e.walkExpr(q, v.Arg, result)
	case *catalog.RowExprQ:
		for _, arg := range v.Args {
			e.walkExpr(q, arg, result)
		}
	case *catalog.WindowFuncExpr:
		for _, arg := range v.Args {
			e.walkExpr(q, arg, result)
		}
		e.walkExpr(q, v.AggFilter, result)
	case *catalog.NullIfExprQ:
		for _, arg := range v.Args {
			e.walkExpr(q, arg, result)
		}
	case *catalog.MinMaxExprQ:
		for _, arg := range v.Args {
			e.walkExpr(q, arg, result)
		}
	case *catalog.ArrayExprQ:
		for _, elem := range v.Elements {
			e.walkExpr(q, elem, result)
		}
	case *catalog.ScalarArrayOpExpr:
		e.walkExpr(q, v.Left, result)
		e.walkExpr(q, v.Right, result)
	case *catalog.CollateExprQ:
		e.walkExpr(q, v.Arg, result)
	case *catalog.NullTestExpr:
		e.walkExpr(q, v.Arg, result)
	case *catalog.BooleanTestExpr:
		e.walkExpr(q, v.Arg, result)
	case *catalog.DistinctExprQ:
		e.walkExpr(q, v.Left, result)
		e.walkExpr(q, v.Right, result)
	case *catalog.FieldSelectExprQ:
		e.walkExpr(q, v.Arg, result)
	// ConstExpr, SQLValueFuncExpr, CoerceToDomainValueExpr — no column refs.
	default:
	}
}

// resolveVar resolves a VarExpr to its ultimate source column(s).
func (e *omniQuerySpanExtractor) resolveVar(q *catalog.Query, v *catalog.VarExpr, result base.SourceColumnSet) {
	if v.RangeIdx < 0 || v.RangeIdx >= len(q.RangeTable) {
		return
	}
	rte := q.RangeTable[v.RangeIdx]
	colIdx := int(v.AttNum - 1) // AttNum is 1-based

	switch rte.Kind {
	case catalog.RTERelation:
		rel := e.cat.GetRelationByOID(rte.RelOID)
		if rel == nil || rel.Schema == nil {
			return
		}
		colName := ""
		if colIdx >= 0 && colIdx < len(rte.ColNames) {
			colName = rte.ColNames[colIdx]
		}
		result[base.ColumnResource{
			Database: e.defaultDatabase,
			Schema:   rel.Schema.Name,
			Table:    rel.Name,
			Column:   colName,
		}] = true

	case catalog.RTESubquery:
		if rte.Subquery == nil {
			return
		}
		if colIdx >= 0 && colIdx < len(rte.Subquery.TargetList) {
			te := rte.Subquery.TargetList[colIdx]
			e.walkExpr(rte.Subquery, te.Expr, result)
		}

	case catalog.RTECTE:
		if rte.CTEIndex >= 0 && rte.CTEIndex < len(q.CTEList) {
			cte := q.CTEList[rte.CTEIndex]
			if cte.Query != nil && colIdx >= 0 && colIdx < len(cte.Query.TargetList) {
				te := cte.Query.TargetList[colIdx]
				e.walkExpr(cte.Query, te.Expr, result)
			}
		}

	case catalog.RTEJoin:
		// After analysis, VarExprs typically point to base RTEs, not join RTEs.

	case catalog.RTEFunction:
		// Trace lineage through user-defined function body.
		if len(rte.FuncExprs) > 0 {
			if fc, ok := rte.FuncExprs[0].(*catalog.FuncCallExpr); ok {
				if proc := e.cat.GetUserProcByOID(fc.FuncOID); proc != nil {
					bodySets := e.analyzeFunctionBody(proc)
					if colIdx >= 0 && colIdx < len(bodySets) {
						for k, val := range bodySets[colIdx] {
							result[k] = val
						}
					}
				}
			}
		}

	default:
		// Unknown RTE kind — skip.
	}
}

// extractFallbackColumns attempts to extract column names and lineage from the parse tree
// when AnalyzeSelectStmt fails. It walks the AST to find column references and resolves
// them against tables in the FROM clause.
func (e *omniQuerySpanExtractor) extractFallbackColumns(selStmt *ast.SelectStmt) []base.QuerySpanResult {
	if selStmt == nil {
		return nil
	}

	// Build a helper analyzer to reuse column reference extraction.
	analyzer := &plpgsqlAnalyzer{
		extractor: e,
		scope:     newVariableScope(nil),
	}
	fromTables := analyzer.collectFromTables(selStmt)

	// Build CTE map so we can trace through CTEs to real tables.
	cteMap := collectCTEDefinitions(selStmt)
	analyzer.cteMap = cteMap

	// Check if SELECT has explicit target list (not *).
	if selStmt.TargetList != nil {
		var results []base.QuerySpanResult
		allStar := true
		for _, item := range selStmt.TargetList.Items {
			rt, ok := item.(*ast.ResTarget)
			if !ok {
				continue
			}
			if !isStarTarget(rt) {
				allStar = false
				name := rt.Name
				if name == "" {
					name = figureResTargetName(rt)
				}
				// Extract column references from the expression for lineage.
				colSet := make(base.SourceColumnSet)
				analyzer.extractColumnRefsFromExpr(rt.Val, fromTables, colSet)
				results = append(results, base.QuerySpanResult{
					Name:          name,
					SourceColumns: colSet,
				})
			}
		}
		if !allStar && len(results) > 0 {
			// Also extract WHERE clause columns for predicate tracking.
			if selStmt.WhereClause != nil {
				whereColSet := make(base.SourceColumnSet)
				analyzer.extractColumnRefsFromExpr(selStmt.WhereClause, fromTables, whereColSet)
				for k, v := range whereColSet {
					e.funcPredicateColumns[k] = v
				}
			}
			return results
		}
	}

	// For SELECT *, try to extract column names from FROM clause aliases.
	if selStmt.FromClause != nil {
		for _, item := range selStmt.FromClause.Items {
			if cols := extractColumnsFromFromItem(item); len(cols) > 0 {
				return cols
			}
		}
	}

	return nil
}

// figureResTargetName extracts a column name from a ResTarget's expression.
func figureResTargetName(rt *ast.ResTarget) string {
	if rt == nil || rt.Val == nil {
		return ""
	}
	if cr, ok := rt.Val.(*ast.ColumnRef); ok && cr.Fields != nil {
		// Use the last String field — for "ia.approver_emails", return "approver_emails".
		// For unqualified "col", return "col". Skip A_Star nodes.
		name := ""
		for _, f := range cr.Fields.Items {
			if s, ok := f.(*ast.String); ok {
				name = s.Str
			}
		}
		return name
	}
	if fc, ok := rt.Val.(*ast.FuncCall); ok && fc.Funcname != nil {
		// Use the last name part — for "pg_catalog.func", return "func".
		name := ""
		for _, f := range fc.Funcname.Items {
			if s, ok := f.(*ast.String); ok {
				name = s.Str
			}
		}
		return name
	}
	// SQL/JSON constructors.
	if _, ok := rt.Val.(*ast.JsonObjectConstructor); ok {
		return "json_object"
	}
	if _, ok := rt.Val.(*ast.JsonArrayConstructor); ok {
		return "json_array"
	}
	return ""
}

// extractColumnsFromFromItem extracts column names from a FROM clause item's alias.
func extractColumnsFromFromItem(item ast.Node) []base.QuerySpanResult {
	if item == nil {
		return nil
	}

	var alias *ast.Alias
	switch v := item.(type) {
	case *ast.RangeFunction:
		alias = v.Alias
	case *ast.RangeVar:
		alias = v.Alias
	case *ast.RangeSubselect:
		alias = v.Alias
	default:
		return nil
	}

	if alias == nil {
		return nil
	}

	// If alias has column names, use those.
	if alias.Colnames != nil && len(alias.Colnames.Items) > 0 {
		var results []base.QuerySpanResult
		for _, item := range alias.Colnames.Items {
			if s, ok := item.(*ast.String); ok {
				results = append(results, base.QuerySpanResult{
					Name:          s.Str,
					SourceColumns: base.SourceColumnSet{},
				})
			}
		}
		return results
	}

	// If it's a RangeFunction without explicit column names, use function name(s).
	if rf, ok := item.(*ast.RangeFunction); ok {
		return extractFuncColumnNames(rf, alias)
	}

	return nil
}

// extractFuncColumnNames extracts column names from a RangeFunction.
// For functions like unnest(ARRAY, ARRAY), each argument produces a column
// named after the function.
func extractFuncColumnNames(rf *ast.RangeFunction, _ *ast.Alias) []base.QuerySpanResult {
	if rf.Functions == nil {
		return nil
	}

	var results []base.QuerySpanResult
	for _, funcItem := range rf.Functions.Items {
		funcList, ok := funcItem.(*ast.List)
		if !ok || len(funcList.Items) < 1 {
			continue
		}
		fc, ok := funcList.Items[0].(*ast.FuncCall)
		if !ok || fc.Funcname == nil {
			continue
		}
		// Get function name.
		funcName := ""
		for _, nameItem := range fc.Funcname.Items {
			if s, ok := nameItem.(*ast.String); ok {
				funcName = s.Str
			}
		}
		if funcName == "" {
			continue
		}
		// Determine number of columns: for set-returning functions like unnest,
		// each argument produces a column.
		nCols := 1
		if fc.Args != nil && len(fc.Args.Items) > 1 {
			nCols = len(fc.Args.Items)
		}
		for range nCols {
			results = append(results, base.QuerySpanResult{
				Name:          funcName,
				SourceColumns: base.SourceColumnSet{},
			})
		}
	}
	return results
}

// tryUserFuncTableSource handles the case where AnalyzeSelectStmt fails because
// the FROM clause contains a user-defined RETURNS TABLE function. The omni catalog
// cannot resolve these as table sources, so we look up the function manually,
// analyze its body for lineage, and construct the result.
// Returns nil if this case doesn't apply.
func (e *omniQuerySpanExtractor) tryUserFuncTableSource(selStmt *ast.SelectStmt, accessesMap base.SourceColumnSet) []base.QuerySpanResult {
	if selStmt == nil || selStmt.FromClause == nil {
		return nil
	}

	// Look for a single RangeFunction in the FROM clause.
	for _, item := range selStmt.FromClause.Items {
		rf, ok := item.(*ast.RangeFunction)
		if !ok || rf.Functions == nil {
			continue
		}

		// Extract function name from the RangeFunction.
		funcName := extractFuncNameFromRange(rf)
		if funcName == "" {
			continue
		}

		// Look up the function in the catalog.
		proc := e.lookupUserProcByName(funcName)
		if proc != nil {
			// Found in catalog — use it for lineage analysis.
			outNames := getOutputParamNames(proc)
			if len(outNames) == 0 {
				continue
			}
			bodySets := e.analyzeFunctionBody(proc)
			return e.buildFuncResults(outNames, bodySets, accessesMap)
		}

		// Catalog lookup failed — try metadata. This handles cases where
		// the metadata function name differs from the name in the DDL definition.
		if results := e.tryMetadataFuncLookup(funcName, accessesMap); results != nil {
			return results
		}
	}

	return nil
}

// extractFuncNameFromRange extracts the function name from a RangeFunction node.
func extractFuncNameFromRange(rf *ast.RangeFunction) string {
	if rf.Functions == nil {
		return ""
	}
	for _, funcItem := range rf.Functions.Items {
		funcList, ok := funcItem.(*ast.List)
		if !ok || len(funcList.Items) < 1 {
			continue
		}
		fc, ok := funcList.Items[0].(*ast.FuncCall)
		if !ok || fc.Funcname == nil {
			continue
		}
		for _, nameItem := range fc.Funcname.Items {
			if s, ok := nameItem.(*ast.String); ok {
				return s.Str
			}
		}
	}
	return ""
}

// lookupUserProcByName finds a user-defined function by name in the catalog,
// searching through schemas in the search path.
func (e *omniQuerySpanExtractor) lookupUserProcByName(name string) *catalog.UserProc {
	lowerName := strings.ToLower(name)
	for _, schemaName := range e.searchPath {
		for _, row := range e.cat.QueryPgProc(schemaName) {
			if strings.ToLower(row.ProName) == lowerName {
				return e.cat.GetUserProcByOID(row.OID)
			}
		}
	}
	return nil
}

// getOutputParamNames returns the names of output parameters (TABLE or OUT mode)
// for a user-defined function. Returns nil if the function has no output params.
func getOutputParamNames(proc *catalog.UserProc) []string {
	var names []string
	for i, mode := range proc.ArgModes {
		if mode == 't' || mode == 'o' {
			if i < len(proc.ArgNames) {
				names = append(names, proc.ArgNames[i])
			}
		}
	}
	return names
}

// buildFuncResults constructs QuerySpanResults from output parameter names and
// function body lineage analysis, and populates table-level access in accessesMap.
func (*omniQuerySpanExtractor) buildFuncResults(outNames []string, bodySets []base.SourceColumnSet, accessesMap base.SourceColumnSet) []base.QuerySpanResult {
	for _, colSet := range bodySets {
		for k, v := range colSet {
			accessesMap[base.ColumnResource{
				Database: k.Database,
				Schema:   k.Schema,
				Table:    k.Table,
			}] = v
		}
	}

	var results []base.QuerySpanResult
	for i, name := range outNames {
		sourceColumns := make(base.SourceColumnSet)
		if i < len(bodySets) {
			for k, v := range bodySets[i] {
				sourceColumns[k] = v
			}
		}
		results = append(results, base.QuerySpanResult{
			Name:          name,
			SourceColumns: sourceColumns,
		})
	}
	return results
}

// tryMetadataFuncLookup searches for a function by name in the database metadata
// (not the catalog). This handles cases where the catalog can't resolve the
// function — either because the metadata function name differs from the DDL name,
// or because RETURNS TABLE functions can't be loaded into the catalog.
func (e *omniQuerySpanExtractor) tryMetadataFuncLookup(funcName string, accessesMap base.SourceColumnSet) []base.QuerySpanResult {
	meta, err := e.getDatabaseMetadata(e.defaultDatabase)
	if err != nil || meta == nil {
		return nil
	}

	_, funcs := meta.SearchFunctions(e.searchPath, funcName)
	if len(funcs) == 0 {
		return nil
	}

	// Use the first matching function's definition.
	funcDef := funcs[0].Definition
	if funcDef == "" {
		return nil
	}

	// Parse the function definition to extract output param names and body.
	// We can't rely on the catalog for RETURNS TABLE functions because the
	// catalog may reject them due to type validation.
	outNames, body, lang := parseFuncDefForLineage(funcDef)
	if len(outNames) == 0 || body == "" {
		return nil
	}

	// Construct a synthetic UserProc for body analysis.
	syntheticProc := &catalog.UserProc{
		Name:     funcName,
		Language: lang,
		Body:     body,
	}
	// Set ArgModes and ArgNames for output params.
	for _, name := range outNames {
		syntheticProc.ArgModes = append(syntheticProc.ArgModes, 't')
		syntheticProc.ArgNames = append(syntheticProc.ArgNames, name)
	}
	// Also extract input params for variable substitution.
	inputNames := parseInputParamNames(funcDef)
	for _, name := range inputNames {
		syntheticProc.ArgModes = append([]byte{'i'}, syntheticProc.ArgModes...)
		syntheticProc.ArgNames = append([]string{name}, syntheticProc.ArgNames...)
	}

	// Use a temporary OID for caching.
	syntheticProc.OID = uint32(0xFFFF0000 + len(e.funcBodyCache))

	bodySets := e.analyzeFunctionBody(syntheticProc)
	return e.buildFuncResults(outNames, bodySets, accessesMap)
}

// parseFuncDefForLineage extracts output parameter names, function body, and
// language from a CREATE FUNCTION definition string.
func parseFuncDefForLineage(definition string) (outNames []string, body string, lang string) {
	upper := strings.ToUpper(definition)

	// Extract language.
	if langIdx := strings.Index(upper, "LANGUAGE"); langIdx >= 0 {
		rest := strings.TrimSpace(definition[langIdx+8:])
		parts := strings.Fields(rest)
		if len(parts) > 0 {
			lang = strings.ToLower(strings.TrimRight(parts[0], ";"))
		}
	}

	// Extract body.
	body = extractFuncBodyFromDef(definition)

	// Extract output param names from RETURNS TABLE(name type, ...).
	tableIdx := strings.Index(upper, "RETURNS TABLE")
	if tableIdx < 0 {
		tableIdx = strings.Index(upper, "RETURNS\n    TABLE")
	}
	if tableIdx >= 0 {
		rest := definition[tableIdx:]
		parenStart := strings.Index(rest, "(")
		if parenStart >= 0 {
			depth := 0
			parenEnd := -1
			for i := parenStart; i < len(rest); i++ {
				switch rest[i] {
				case '(':
					depth++
				case ')':
					depth--
					if depth == 0 {
						parenEnd = i
					}
				default:
				}
				if parenEnd >= 0 {
					break
				}
			}
			if parenEnd > parenStart {
				paramList := rest[parenStart+1 : parenEnd]
				for _, param := range strings.Split(paramList, ",") {
					parts := strings.Fields(strings.TrimSpace(param))
					if len(parts) >= 2 {
						outNames = append(outNames, parts[0])
					}
				}
			}
		}
	}

	return outNames, body, lang
}

// parseInputParamNames extracts input parameter names from a CREATE FUNCTION
// definition. It parses the parameter list before RETURNS.
func parseInputParamNames(definition string) []string {
	upper := strings.ToUpper(definition)

	// Find the parameter list: between FUNCTION name( and ) RETURNS.
	funcIdx := strings.Index(upper, "FUNCTION")
	if funcIdx < 0 {
		return nil
	}

	rest := definition[funcIdx:]
	parenStart := strings.Index(rest, "(")
	if parenStart < 0 {
		return nil
	}

	// Find the closing paren that ends the input params.
	returnsIdx := strings.Index(upper[funcIdx+parenStart:], "RETURNS")
	if returnsIdx < 0 {
		return nil
	}

	paramSection := rest[parenStart+1 : parenStart+returnsIdx]
	// Find the last ) before RETURNS.
	lastParen := strings.LastIndex(paramSection, ")")
	if lastParen >= 0 {
		paramSection = paramSection[:lastParen]
	}

	var names []string
	for _, param := range strings.Split(paramSection, ",") {
		parts := strings.Fields(strings.TrimSpace(param))
		if len(parts) >= 2 {
			names = append(names, parts[0])
		}
	}
	return names
}
