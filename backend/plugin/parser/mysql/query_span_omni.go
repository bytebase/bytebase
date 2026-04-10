package mysql

import (
	"context"
	"fmt"
	"strings"

	"github.com/pkg/errors"

	"github.com/bytebase/omni/mysql/ast"
	"github.com/bytebase/omni/mysql/catalog"

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
	metaCache       map[string]*model.DatabaseMetadata
	cat             *catalog.Catalog

	ignoreCaseSensitive bool
}

func newOmniQuerySpanExtractor(defaultDatabase string, gCtx base.GetQuerySpanContext, ignoreCaseSensitive bool) *omniQuerySpanExtractor {
	return &omniQuerySpanExtractor{
		defaultDatabase:     defaultDatabase,
		gCtx:                gCtx,
		metaCache:           make(map[string]*model.DatabaseMetadata),
		ignoreCaseSensitive: ignoreCaseSensitive,
	}
}

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

func (e *omniQuerySpanExtractor) initCatalog() error {
	meta, err := e.getDatabaseMetadata(e.defaultDatabase)
	if err != nil {
		return errors.Wrapf(err, "failed to get database metadata for catalog init")
	}

	e.cat = catalog.New()
	e.cat.SetForeignKeyChecks(false)

	dbName := e.defaultDatabase
	initSQL := fmt.Sprintf("SET foreign_key_checks = 0;\nCREATE DATABASE IF NOT EXISTS `%s`;\nUSE `%s`;", dbName, dbName)
	if _, err := e.cat.Exec(initSQL, &catalog.ExecOptions{ContinueOnError: true}); err != nil {
		return errors.Wrap(err, "failed to initialize catalog")
	}

	if meta == nil {
		return nil
	}

	if e.gCtx.GetDatabaseDefinitionFunc != nil {
		schemaDDL, err := e.gCtx.GetDatabaseDefinitionFunc(meta.GetProto())
		if err != nil {
			return errors.Wrap(err, "failed to generate schema DDL")
		}
		if schemaDDL != "" {
			if _, err := e.cat.Exec(schemaDDL, &catalog.ExecOptions{ContinueOnError: true}); err != nil {
				return errors.Wrap(err, "failed to load schema DDL into catalog")
			}
		}
	}

	return nil
}

//nolint:nilerr // intentional fail-open: AnalyzeSelectStmt errors are swallowed for best-effort results
func (e *omniQuerySpanExtractor) getQuerySpan(ctx context.Context, stmt string) (*base.QuerySpan, error) {
	e.ctx = ctx

	// Step 1: Parse.
	parsed, err := ParseMySQLOmni(stmt)
	if err != nil {
		return nil, &base.SyntaxError{
			Message:  err.Error(),
			Position: &storepb.Position{Line: 1, Column: 1},
		}
	}
	if parsed == nil || len(parsed.Items) == 0 {
		return nil, errors.New("no statements parsed")
	}
	if len(parsed.Items) != 1 {
		return nil, errors.Errorf("expected exactly 1 statement, got %d", len(parsed.Items))
	}

	node := parsed.Items[0]

	// Step 2: Extract access tables from omni AST.
	accessesMap := e.extractAccessTablesFromAST(node)

	// Step 3: Check mixed system/user tables.
	allSystems, mixed := isMixedQuery(accessesMap, e.ignoreCaseSensitive)
	if mixed {
		return nil, base.MixUserSystemTablesError
	}

	// Step 4: Classify query type.
	queryType, isExplainAnalyze := classifyQueryTypeOmni(node, allSystems)

	// Step 5: Return early for non-SELECT queries.
	if queryType != base.Select {
		return &base.QuerySpan{
			Type:          queryType,
			SourceColumns: base.SourceColumnSet{},
			Results:       []base.QuerySpanResult{},
		}, nil
	}

	if isExplainAnalyze {
		return &base.QuerySpan{
			Type:          queryType,
			SourceColumns: accessesMap,
			Results:       []base.QuerySpanResult{},
		}, nil
	}

	// Step 6: Analyze SELECT.
	selStmt, ok := node.(*ast.SelectStmt)
	if !ok {
		return &base.QuerySpan{
			Type:          base.Select,
			SourceColumns: accessesMap,
			Results:       []base.QuerySpanResult{},
		}, nil
	}

	if err := e.initCatalog(); err != nil {
		return nil, errors.Wrapf(err, "failed to initialize catalog")
	}

	query, analyzeErr := e.cat.AnalyzeSelectStmt(selStmt)
	if analyzeErr != nil {
		// Fail-open: return access tables with best-effort column extraction
		// when analysis fails (e.g., unknown function names).
		return e.buildFallbackQuerySpan(selStmt, accessesMap), nil
	}

	// Expand merge views to see through to underlying tables.
	query = query.ExpandMergeViews(e.cat)

	// Step 7: Extract lineage.
	results := e.extractLineage(query, selStmt)

	return &base.QuerySpan{
		Type:          base.Select,
		SourceColumns: accessesMap,
		Results:       results,
	}, nil
}

func (e *omniQuerySpanExtractor) buildFallbackQuerySpan(selStmt *ast.SelectStmt, accessesMap base.SourceColumnSet) *base.QuerySpan {
	return &base.QuerySpan{
		Type:          base.Select,
		SourceColumns: accessesMap,
		Results:       e.extractFallbackColumns(selStmt),
	}
}

func isMixedQuery(m base.SourceColumnSet, ignoreCaseSensitive bool) (bool, bool) {
	hasSystem, hasUser := false, false
	for table := range m {
		if isSystemResource(table, ignoreCaseSensitive) {
			hasSystem = true
		} else {
			hasUser = true
		}
	}
	if hasSystem && hasUser {
		return false, true
	}
	return !hasUser && hasSystem, false
}

var systemDatabases = map[string]bool{
	"information_schema": true,
	"performance_schema": true,
	"mysql":              true,
}

func isSystemResource(resource base.ColumnResource, ignoreCaseSensitive bool) bool {
	database := resource.Database
	if ignoreCaseSensitive {
		database = strings.ToLower(database)
	}
	return systemDatabases[database]
}

// classifyQueryTypeOmni determines the query type from an omni AST node.
func classifyQueryTypeOmni(node ast.Node, allSystems bool) (queryType base.QueryType, isExplainAnalyze bool) {
	switch n := node.(type) {
	case *ast.SelectStmt:
		if allSystems {
			return base.SelectInfoSchema, false
		}
		return base.Select, false
	case *ast.InsertStmt, *ast.UpdateStmt, *ast.DeleteStmt:
		return base.DML, false
	case *ast.CreateTableStmt, *ast.CreateViewStmt, *ast.CreateIndexStmt,
		*ast.AlterTableStmt, *ast.DropTableStmt, *ast.DropViewStmt, *ast.DropIndexStmt,
		*ast.RenameTableStmt, *ast.TruncateStmt, *ast.CreateDatabaseStmt,
		*ast.AlterDatabaseStmt, *ast.DropDatabaseStmt:
		return base.DDL, false
	case *ast.SetStmt:
		return base.Select, false
	case *ast.ShowStmt:
		return base.SelectInfoSchema, false
	case *ast.ExplainStmt:
		if n.Analyze {
			// Check what's inside the EXPLAIN ANALYZE.
			if n.Stmt != nil {
				switch n.Stmt.(type) {
				case *ast.SelectStmt:
					return base.Select, true
				case *ast.InsertStmt, *ast.UpdateStmt, *ast.DeleteStmt:
					return base.DML, true
				default:
					return base.Explain, true
				}
			}
			return base.Explain, true
		}
		return base.Explain, false
	default:
		return base.QueryTypeUnknown, false
	}
}

func (e *omniQuerySpanExtractor) extractLineage(q *catalog.Query, selStmt *ast.SelectStmt) []base.QuerySpanResult {
	if q.SetOp != catalog.SetOpNone {
		return e.extractSetOpLineage(q)
	}

	plainMask := buildPlainFieldMaskMySQL(selStmt, q)

	var results []base.QuerySpanResult
	idx := 0
	for _, te := range q.TargetList {
		if te.ResJunk {
			continue
		}
		sourceColSet := make(base.SourceColumnSet)
		e.walkExpr(q, te.Expr, sourceColSet)
		isPlain := idx < len(plainMask) && plainMask[idx]
		if isPlain {
			isPlain = isUltimatelyPlainColumnMySQL(q, te.Expr)
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

func (e *omniQuerySpanExtractor) extractSetOpLineage(q *catalog.Query) []base.QuerySpanResult {
	if q.LArg != nil {
		return e.extractLineage(q.LArg, nil)
	}
	var results []base.QuerySpanResult
	for _, te := range q.TargetList {
		if te.ResJunk {
			continue
		}
		results = append(results, base.QuerySpanResult{
			Name:          te.ResName,
			SourceColumns: make(base.SourceColumnSet),
		})
	}
	return results
}

func (e *omniQuerySpanExtractor) walkExpr(q *catalog.Query, expr catalog.AnalyzedExpr, result base.SourceColumnSet) {
	if expr == nil {
		return
	}
	switch v := expr.(type) {
	case *catalog.VarExprQ:
		e.resolveVar(q, v, result)
	case *catalog.FuncCallExprQ:
		for _, arg := range v.Args {
			e.walkExpr(q, arg, result)
		}
	case *catalog.OpExprQ:
		e.walkExpr(q, v.Left, result)
		e.walkExpr(q, v.Right, result)
	case *catalog.BoolExprQ:
		for _, arg := range v.Args {
			e.walkExpr(q, arg, result)
		}
	case *catalog.CaseExprQ:
		e.walkExpr(q, v.TestExpr, result)
		for _, w := range v.Args {
			e.walkExpr(q, w.Cond, result)
			e.walkExpr(q, w.Then, result)
		}
		e.walkExpr(q, v.Default, result)
	case *catalog.CoalesceExprQ:
		for _, arg := range v.Args {
			e.walkExpr(q, arg, result)
		}
	case *catalog.SubLinkExprQ:
		e.walkExpr(q, v.TestExpr, result)
		if v.Subquery != nil {
			for _, te := range v.Subquery.TargetList {
				if !te.ResJunk {
					e.walkExpr(v.Subquery, te.Expr, result)
				}
			}
		}
	case *catalog.CastExprQ:
		e.walkExpr(q, v.Arg, result)
	case *catalog.RowExprQ:
		for _, arg := range v.Args {
			e.walkExpr(q, arg, result)
		}
	case *catalog.NullTestExprQ:
		e.walkExpr(q, v.Arg, result)
	case *catalog.BetweenExprQ:
		e.walkExpr(q, v.Arg, result)
		e.walkExpr(q, v.Lower, result)
		e.walkExpr(q, v.Upper, result)
	case *catalog.InListExprQ:
		e.walkExpr(q, v.Arg, result)
		for _, item := range v.List {
			e.walkExpr(q, item, result)
		}
	// ConstExprQ — no column refs.
	default:
	}
}

func (e *omniQuerySpanExtractor) resolveVar(q *catalog.Query, v *catalog.VarExprQ, result base.SourceColumnSet) {
	if v.RangeIdx < 0 || v.RangeIdx >= len(q.RangeTable) {
		return
	}
	rte := q.RangeTable[v.RangeIdx]
	colIdx := v.AttNum - 1

	switch rte.Kind {
	case catalog.RTERelation:
		colName := ""
		if colIdx >= 0 && colIdx < len(rte.ColNames) {
			colName = rte.ColNames[colIdx]
		}
		dbName := rte.DBName
		if dbName == "" {
			dbName = e.defaultDatabase
		}
		result[base.ColumnResource{
			Database: dbName,
			Table:    rte.TableName,
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
		// For JOIN RTEs, the column name maps back to a physical column.
		// Look through all RTEs for a matching column name.
		if colIdx >= 0 && colIdx < len(rte.ColNames) {
			colName := rte.ColNames[colIdx]
			for ri, other := range q.RangeTable {
				if other.Kind == catalog.RTEJoin || ri == v.RangeIdx {
					continue
				}
				for ci, cn := range other.ColNames {
					if strings.EqualFold(cn, colName) {
						e.resolveVar(q, &catalog.VarExprQ{
							RangeIdx: ri,
							AttNum:   ci + 1,
						}, result)
						return
					}
				}
			}
		}

	default:
	}
}

// extractFallbackColumns provides best-effort column extraction from the parse tree
// when AnalyzeSelectStmt fails.
func (e *omniQuerySpanExtractor) extractFallbackColumns(selStmt *ast.SelectStmt) []base.QuerySpanResult {
	if selStmt == nil {
		return nil
	}

	// Build a table-reference map from the FROM clause for column resolution.
	fromTables := e.collectFromTablesSimple(selStmt)

	var results []base.QuerySpanResult
	for _, target := range selStmt.TargetList {
		// Handle *ast.StarExpr (bare SELECT *)
		if _, isStar := target.(*ast.StarExpr); isStar {
			for _, ref := range fromTables {
				meta, err := e.getDatabaseMetadata(ref.Database)
				if err != nil || meta == nil {
					continue
				}
				tableMeta := meta.GetSchemaMetadata("").GetTable(ref.Table)
				if tableMeta == nil {
					continue
				}
				for _, c := range tableMeta.GetProto().GetColumns() {
					sourceSet := make(base.SourceColumnSet)
					sourceSet[base.ColumnResource{Database: ref.Database, Table: ref.Table, Column: c.Name}] = true
					results = append(results, base.QuerySpanResult{
						Name: c.Name, SourceColumns: sourceSet, IsPlainField: true,
					})
				}
			}
			continue
		}

		// Unwrap ResTarget if present.
		var expr ast.ExprNode
		var aliasName string
		if rt, ok := target.(*ast.ResTarget); ok {
			expr = rt.Val
			aliasName = rt.Name
		} else {
			expr = target
		}
		if expr == nil {
			continue
		}

		col, isColRef := expr.(*ast.ColumnRef)
		// Handle SELECT t.*
		if isColRef && col.Star {
			tableName := col.Table
			if tableName == "" {
				// SELECT * — expand all tables.
				for _, ref := range fromTables {
					meta, err := e.getDatabaseMetadata(ref.Database)
					if err != nil || meta == nil {
						continue
					}
					tableMeta := meta.GetSchemaMetadata("").GetTable(ref.Table)
					if tableMeta == nil {
						continue
					}
					for _, c := range tableMeta.GetProto().GetColumns() {
						sourceSet := make(base.SourceColumnSet)
						sourceSet[base.ColumnResource{Database: ref.Database, Table: ref.Table, Column: c.Name}] = true
						results = append(results, base.QuerySpanResult{
							Name: c.Name, SourceColumns: sourceSet, IsPlainField: true,
						})
					}
				}
			} else {
				// SELECT t.* — expand specific table.
				for _, ref := range fromTables {
					if !strings.EqualFold(ref.Table, tableName) && !strings.EqualFold(ref.alias, tableName) {
						continue
					}
					meta, err := e.getDatabaseMetadata(ref.Database)
					if err != nil || meta == nil {
						continue
					}
					tableMeta := meta.GetSchemaMetadata("").GetTable(ref.Table)
					if tableMeta == nil {
						continue
					}
					for _, c := range tableMeta.GetProto().GetColumns() {
						sourceSet := make(base.SourceColumnSet)
						sourceSet[base.ColumnResource{Database: ref.Database, Table: ref.Table, Column: c.Name}] = true
						results = append(results, base.QuerySpanResult{
							Name: c.Name, SourceColumns: sourceSet, IsPlainField: true,
						})
					}
				}
			}
			continue
		}

		// Regular column or expression.
		name := aliasName
		sourceSet := make(base.SourceColumnSet)
		isPlain := false
		if isColRef {
			if name == "" {
				name = col.Column
			}
			e.resolveColumnRefFallback(col, fromTables, sourceSet)
			isPlain = len(sourceSet) > 0
		} else {
			// For expressions, walk the AST to find column refs.
			e.walkExprNodeForColumnRefs(expr, fromTables, sourceSet)
			if name == "" {
				// Use the expression text as the name (like MySQL does).
				if fc, ok := expr.(*ast.FuncCallExpr); ok {
					name = fc.Name
				}
			}
		}
		if name == "" {
			name = "?column?"
		}
		results = append(results, base.QuerySpanResult{
			Name:          name,
			SourceColumns: sourceSet,
			IsPlainField:  isPlain,
		})
	}
	return results
}

type fallbackTableRef struct {
	Database string
	Table    string
	alias    string
}

func (e *omniQuerySpanExtractor) collectFromTablesSimple(selStmt *ast.SelectStmt) []fallbackTableRef {
	var refs []fallbackTableRef
	for _, from := range selStmt.From {
		e.collectFromExprSimple(from, &refs)
	}
	return refs
}

func (e *omniQuerySpanExtractor) collectFromExprSimple(expr ast.TableExpr, refs *[]fallbackTableRef) {
	if expr == nil {
		return
	}
	switch v := expr.(type) {
	case *ast.TableRef:
		db := v.Schema
		if db == "" {
			db = e.defaultDatabase
		}
		*refs = append(*refs, fallbackTableRef{Database: db, Table: v.Name, alias: v.Alias})
	case *ast.JoinClause:
		e.collectFromExprSimple(v.Left, refs)
		e.collectFromExprSimple(v.Right, refs)
	default:
	}
}

func (e *omniQuerySpanExtractor) walkExprNodeForColumnRefs(expr ast.ExprNode, tables []fallbackTableRef, result base.SourceColumnSet) {
	if expr == nil {
		return
	}
	switch v := expr.(type) {
	case *ast.ColumnRef:
		e.resolveColumnRefFallback(v, tables, result)
	case *ast.FuncCallExpr:
		for _, arg := range v.Args {
			e.walkExprNodeForColumnRefs(arg, tables, result)
		}
	default:
	}
}

func (e *omniQuerySpanExtractor) resolveColumnRefFallback(col *ast.ColumnRef, tables []fallbackTableRef, result base.SourceColumnSet) {
	if col.Table != "" {
		// Qualified: find the table by name or alias.
		for _, ref := range tables {
			if strings.EqualFold(ref.Table, col.Table) || strings.EqualFold(ref.alias, col.Table) {
				result[base.ColumnResource{Database: ref.Database, Table: ref.Table, Column: col.Column}] = true
				return
			}
		}
	} else {
		// Unqualified: search all tables.
		for _, ref := range tables {
			meta, err := e.getDatabaseMetadata(ref.Database)
			if err != nil || meta == nil {
				continue
			}
			tableMeta := meta.GetSchemaMetadata("").GetTable(ref.Table)
			if tableMeta == nil {
				continue
			}
			for _, c := range tableMeta.GetProto().GetColumns() {
				if strings.EqualFold(c.Name, col.Column) {
					result[base.ColumnResource{Database: ref.Database, Table: ref.Table, Column: col.Column}] = true
					return
				}
			}
		}
	}
}

// extractAccessTablesFromAST walks the omni AST to find all table references.
func (e *omniQuerySpanExtractor) extractAccessTablesFromAST(node ast.Node) base.SourceColumnSet {
	result := make(base.SourceColumnSet)
	e.walkASTForTables(node, result)
	return result
}

func (e *omniQuerySpanExtractor) walkASTForTables(node ast.Node, result base.SourceColumnSet) {
	if node == nil {
		return
	}
	switch n := node.(type) {
	case *ast.SelectStmt:
		for _, from := range n.From {
			e.walkTableExprForTables(from, result)
		}
		e.walkExprNodeForTables(n.Where, result)
		for _, cte := range n.CTEs {
			if cte.Select != nil {
				e.walkASTForTables(cte.Select, result)
			}
		}
	case *ast.InsertStmt:
		if n.Table != nil {
			e.addTableRef(n.Table, result)
		}
		if n.Select != nil {
			e.walkASTForTables(n.Select, result)
		}
	case *ast.UpdateStmt:
		for _, t := range n.Tables {
			e.walkTableExprForTables(t, result)
		}
		e.walkExprNodeForTables(n.Where, result)
	case *ast.DeleteStmt:
		for _, t := range n.Tables {
			e.walkTableExprForTables(t, result)
		}
		for _, t := range n.Using {
			e.walkTableExprForTables(t, result)
		}
		e.walkExprNodeForTables(n.Where, result)
	case *ast.ExplainStmt:
		if n.Stmt != nil {
			e.walkASTForTables(n.Stmt, result)
		}
	default:
	}
}

func (e *omniQuerySpanExtractor) walkTableExprForTables(expr ast.TableExpr, result base.SourceColumnSet) {
	if expr == nil {
		return
	}
	switch v := expr.(type) {
	case *ast.TableRef:
		e.addTableRef(v, result)
	case *ast.JoinClause:
		e.walkTableExprForTables(v.Left, result)
		e.walkTableExprForTables(v.Right, result)
	case *ast.SubqueryExpr:
		if v.Select != nil {
			e.walkASTForTables(v.Select, result)
		}
	default:
	}
}

func (*omniQuerySpanExtractor) walkExprNodeForTables(_ ast.ExprNode, _ base.SourceColumnSet) {
	// WHERE clause doesn't introduce new table refs in the simple case.
}

func (e *omniQuerySpanExtractor) addTableRef(ref *ast.TableRef, result base.SourceColumnSet) {
	db := ref.Schema
	if db == "" {
		db = e.defaultDatabase
	}
	result[base.ColumnResource{
		Database: db,
		Table:    ref.Name,
	}] = true
}

func isUltimatelyPlainColumnMySQL(q *catalog.Query, expr catalog.AnalyzedExpr) bool {
	if expr == nil {
		return false
	}
	v, ok := expr.(*catalog.VarExprQ)
	if !ok {
		return false
	}
	if v.RangeIdx < 0 || v.RangeIdx >= len(q.RangeTable) {
		return false
	}
	rte := q.RangeTable[v.RangeIdx]
	colIdx := v.AttNum - 1

	switch rte.Kind {
	case catalog.RTERelation:
		return true
	case catalog.RTESubquery:
		if rte.Subquery != nil && colIdx >= 0 && colIdx < len(rte.Subquery.TargetList) {
			return isUltimatelyPlainColumnMySQL(rte.Subquery, rte.Subquery.TargetList[colIdx].Expr)
		}
	case catalog.RTECTE:
		if rte.CTEIndex >= 0 && rte.CTEIndex < len(q.CTEList) {
			cte := q.CTEList[rte.CTEIndex]
			if cte.Query != nil && colIdx >= 0 && colIdx < len(cte.Query.TargetList) {
				return isUltimatelyPlainColumnMySQL(cte.Query, cte.Query.TargetList[colIdx].Expr)
			}
		}
	default:
		return false
	}
	return false
}

func buildPlainFieldMaskMySQL(selStmt *ast.SelectStmt, q *catalog.Query) []bool {
	var targets []*catalog.TargetEntryQ
	for _, te := range q.TargetList {
		if !te.ResJunk {
			targets = append(targets, te)
		}
	}
	nResults := len(targets)

	if selStmt == nil || len(selStmt.TargetList) == 0 {
		return make([]bool, nResults)
	}

	mask := make([]bool, nResults)
	pos := 0

	for _, target := range selStmt.TargetList {
		if pos >= nResults {
			break
		}
		// Check if this is a star expression.
		isStar := false
		tableName := ""
		if rt, ok := target.(*ast.ResTarget); ok {
			if col, ok := rt.Val.(*ast.ColumnRef); ok && col.Star {
				isStar = true
				tableName = col.Table
			}
		}

		if isStar {
			starCount := countStarColumnsFromRTEMySQL(q, targets, pos, tableName)
			for i := 0; i < starCount && pos < nResults; i++ {
				mask[pos] = true
				pos++
			}
		} else {
			pos++
		}
	}

	return mask
}

// buildMinimalDDLMySQL generates CREATE TABLE / CREATE VIEW statements
// from metadata, sufficient for omni's AnalyzeSelectStmt to resolve columns.
func countStarColumnsFromRTEMySQL(q *catalog.Query, targets []*catalog.TargetEntryQ, startPos int, tableName string) int {
	if startPos >= len(targets) {
		return 0
	}

	if tableName != "" {
		count := 0
		for i := startPos; i < len(targets); i++ {
			v, ok := targets[i].Expr.(*catalog.VarExprQ)
			if !ok {
				break
			}
			if v.RangeIdx < 0 || v.RangeIdx >= len(q.RangeTable) {
				break
			}
			rte := q.RangeTable[v.RangeIdx]
			if !strings.EqualFold(rte.ERef, tableName) && !strings.EqualFold(rte.Alias, tableName) {
				break
			}
			count++
		}
		if count == 0 {
			return 1
		}
		return count
	}

	count := 0
	for i := startPos; i < len(targets); i++ {
		v, ok := targets[i].Expr.(*catalog.VarExprQ)
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
		count++
	}
	if count == 0 {
		return 1
	}
	return count
}
