package pg

import (
	"context"

	"fmt"
	"strings"

	"github.com/pkg/errors"

	"github.com/bytebase/omni/pg/ast"
	"github.com/bytebase/omni/pg/catalog"

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
	metaCache map[string]*model.DatabaseMetadata
	cat       *catalog.Catalog
}

// newOmniQuerySpanExtractor creates a new omni-based query span extractor.
func newOmniQuerySpanExtractor(defaultDatabase string, searchPath []string, gCtx base.GetQuerySpanContext) *omniQuerySpanExtractor {
	if len(searchPath) == 0 {
		searchPath = []string{"public"}
	}
	return &omniQuerySpanExtractor{
		defaultDatabase: defaultDatabase,
		searchPath:      searchPath,
		gCtx:            gCtx,
		metaCache:       make(map[string]*model.DatabaseMetadata),
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
				fmt.Fprintf(&b, "%s;\n", strings.TrimSuffix(strings.TrimSpace(f.Definition), ";"))
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

// getQuerySpan extracts the query span for the given SQL statement.
func (e *omniQuerySpanExtractor) getQuerySpan(ctx context.Context, stmt string) (*base.QuerySpan, error) {
	e.ctx = ctx

	// Step 1: Parse with omni.
	omniStmts, err := ParsePg(stmt)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to parse statement")
	}
	if len(omniStmts) != 1 {
		return nil, errors.Errorf("expected exactly 1 statement, got %d", len(omniStmts))
	}

	// Step 2: Extract access tables.
	accessTables, err := ExtractAccessTables(stmt, ExtractAccessTablesOption{
		DefaultDatabase:        e.defaultDatabase,
		DefaultSchema:          e.searchPath[0],
		GetDatabaseMetadata:    e.gCtx.GetDatabaseMetadataFunc,
		Ctx:                    ctx,
		InstanceID:             e.gCtx.InstanceID,
		SkipMetadataValidation: false,
	})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to extract access tables")
	}

	// Build access map.
	accessesMap := make(base.SourceColumnSet)
	for _, resource := range accessTables {
		accessesMap[resource] = true
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
		return nil, errors.Errorf("expected SelectStmt for SELECT query, got %T", omniStmts[0].AST)
	}

	if err := e.initCatalog(); err != nil {
		return nil, errors.Wrapf(err, "failed to initialize catalog")
	}

	query, err := e.cat.AnalyzeSelectStmt(selStmt)
	if err != nil {
		// Fallback: use legacy ANTLR-based extractor when omni analysis fails
		// (e.g., unsupported function signatures, complex syntax not yet handled).
		return e.fallbackToANTLR(ctx, stmt)
	}

	// Step 7: Extract lineage and source columns (stubs for now).
	results := e.extractLineage(query)
	sourceColumns := e.extractAllSourceColumns(query)

	// Merge extracted source columns with access tables.
	for col := range sourceColumns {
		accessesMap[col] = true
	}

	return &base.QuerySpan{
		Type:          base.Select,
		SourceColumns: accessesMap,
		Results:       results,
	}, nil
}

// extractLineage extracts the result column lineage from an analyzed query.
func (e *omniQuerySpanExtractor) extractLineage(q *catalog.Query) []base.QuerySpanResult {
	// Handle set operations (UNION/INTERSECT/EXCEPT).
	if q.SetOp != catalog.SetOpNone {
		return e.extractSetOpLineage(q)
	}

	var results []base.QuerySpanResult
	for _, te := range q.TargetList {
		if te.ResJunk {
			continue
		}
		_, isVar := te.Expr.(*catalog.VarExpr)
		sourceColSet := make(base.SourceColumnSet)
		e.walkExpr(q, te.Expr, sourceColSet)
		results = append(results, base.QuerySpanResult{
			Name:          te.ResName,
			SourceColumns: sourceColSet,
			IsPlainField:  isVar,
		})
	}
	return results
}

// extractSetOpLineage handles UNION/INTERSECT/EXCEPT by merging lineage from both branches.
func (e *omniQuerySpanExtractor) extractSetOpLineage(q *catalog.Query) []base.QuerySpanResult {
	if q.LArg == nil || q.RArg == nil {
		return nil
	}
	lResults := e.extractLineage(q.LArg)
	rResults := e.extractLineage(q.RArg)

	var results []base.QuerySpanResult
	for i, lr := range lResults {
		merged := make(base.SourceColumnSet)
		for col := range lr.SourceColumns {
			merged[col] = true
		}
		if i < len(rResults) {
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

// extractAllSourceColumns extracts all source columns referenced by the query
// (SELECT + WHERE + JOIN ON + HAVING).
func (e *omniQuerySpanExtractor) extractAllSourceColumns(q *catalog.Query) base.SourceColumnSet {
	result := make(base.SourceColumnSet)

	// Handle set operations.
	if q.SetOp != catalog.SetOpNone {
		if q.LArg != nil {
			for col := range e.extractAllSourceColumns(q.LArg) {
				result[col] = true
			}
		}
		if q.RArg != nil {
			for col := range e.extractAllSourceColumns(q.RArg) {
				result[col] = true
			}
		}
		return result
	}

	// Collect from all target entries.
	for _, te := range q.TargetList {
		e.walkExpr(q, te.Expr, result)
	}

	// Collect from WHERE clause and JOIN conditions.
	if q.JoinTree != nil {
		e.walkExpr(q, q.JoinTree.Quals, result)
		for _, fn := range q.JoinTree.FromList {
			e.walkJoinNodeExprs(q, fn, result)
		}
	}

	// Collect from HAVING clause.
	e.walkExpr(q, q.HavingQual, result)

	return result
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
		for _, arg := range v.Args {
			e.walkExpr(q, arg, result)
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
		// Function results have no further lineage to track.
	}
}

// walkJoinNodeExprs walks the JoinNode tree to collect column references
// from JOIN ON clauses.
func (e *omniQuerySpanExtractor) walkJoinNodeExprs(q *catalog.Query, node catalog.JoinNode, result base.SourceColumnSet) {
	if node == nil {
		return
	}
	switch n := node.(type) {
	case *catalog.JoinExprNode:
		e.walkExpr(q, n.Quals, result)
		e.walkJoinNodeExprs(q, n.Left, result)
		e.walkJoinNodeExprs(q, n.Right, result)
	case *catalog.RangeTableRef:
		// No expressions to walk.
	}
}

// fallbackToANTLR uses the legacy ANTLR-based extractor when omni analysis fails.
func (e *omniQuerySpanExtractor) fallbackToANTLR(ctx context.Context, stmt string) (*base.QuerySpan, error) {
	legacy := newQuerySpanExtractor(e.defaultDatabase, e.searchPath, e.gCtx)
	return legacy.getQuerySpan(ctx, stmt)
}
