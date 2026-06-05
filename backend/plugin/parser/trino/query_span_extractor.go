package trino

import (
	"context"
	"strings"

	"github.com/bytebase/omni/trino/analysis"
	"github.com/bytebase/omni/trino/ast"
	"github.com/bytebase/omni/trino/parser"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/store/model"
)

// defaultTrinoSchema is the schema assumed for a table reference that omits the
// schema qualifier. The legacy plugin used "public" for the data-access-control
// resource format, and the query-span fixtures encode catalog.public.table; we
// keep that so the resource keys are unchanged across the cutover.
const defaultTrinoSchema = "public"

// querySpanExtractor analyses a single Trino statement and produces a
// base.QuerySpan: the set of physical columns it reads (expanded against
// catalog metadata when available), the predicate columns it filters on, and
// its statement classification.
//
// The extractor delegates table/column lineage to omni's analysis.GetQuerySpan,
// then applies bytebase-specific post-processing:
//   - default catalog/schema fill-in for bare table references (Trino's
//     catalog.schema.table → base.ColumnResource Database/Schema/Table),
//   - EXPLAIN unwrap so an EXPLAIN <query> still reports the tables it reads,
//   - expansion of each accessed table to its columns via the metadata getter,
//   - mapping omni's PredicateColumns onto the expanded source columns.
type querySpanExtractor struct {
	ctx context.Context

	defaultDatabase     string
	defaultSchema       string
	ignoreCaseSensitive bool

	gCtx base.GetQuerySpanContext

	// metaCache memoises database metadata lookups within a single span.
	metaCache map[string]*model.DatabaseMetadata
}

// newQuerySpanExtractor creates a new Trino query span extractor.
func newQuerySpanExtractor(defaultDatabase, defaultSchema string, gCtx base.GetQuerySpanContext, ignoreCaseSensitive bool) *querySpanExtractor {
	return &querySpanExtractor{
		defaultDatabase:     defaultDatabase,
		defaultSchema:       defaultSchema,
		gCtx:                gCtx,
		ignoreCaseSensitive: ignoreCaseSensitive,
		metaCache:           make(map[string]*model.DatabaseMetadata),
	}
}

// getQuerySpan extracts the query span for a single Trino statement.
func (q *querySpanExtractor) getQuerySpan(ctx context.Context, statement string) (*base.QuerySpan, error) {
	q.ctx = ctx

	// Split into top-level statements; query-span analysis expects exactly one
	// non-empty statement (matching the legacy behaviour).
	stmts, err := SplitSQL(statement)
	if err != nil {
		return nil, err
	}
	nonEmpty := 0
	var single string
	for _, s := range stmts {
		if s.Empty || strings.TrimSpace(s.Text) == "" {
			continue
		}
		nonEmpty++
		single = s.Text
	}
	if nonEmpty == 0 {
		return &base.QuerySpan{
			SourceColumns: base.SourceColumnSet{},
			Results:       []base.QuerySpanResult{},
		}, nil
	}
	if nonEmpty > 1 {
		return nil, errors.Errorf("expected exactly 1 statement, got %d", nonEmpty)
	}

	// Resolve the statement type up front so non-SELECT statements short-circuit
	// the way the legacy plugin did. EXPLAIN ANALYZE classifies as base.Select.
	file, errs := parser.Parse(single)
	if len(errs) > 0 || file == nil || len(file.Stmts) == 0 {
		return nil, errors.Errorf("failed to parse Trino statement: %s", single)
	}
	node := file.Stmts[0]
	queryType, _ := getQueryType(node)
	// Promote a user SELECT/EXPLAIN that references a system schema, matching the
	// legacy containsSystemSchema heuristic.
	switch queryType {
	case base.Select, base.Explain:
		if containsSystemSchema(single) {
			queryType = base.SelectInfoSchema
		}
	default:
	}

	// For statements that don't read columns into a result set (DML, DDL), the
	// legacy plugin returned a basic span: the accessed tables as source
	// columns, no per-column results. CREATE VIEW / EXPLAIN over a query still
	// report the underlying tables.
	readsResults := queryType == base.Select ||
		queryType == base.Explain ||
		queryType == base.SelectInfoSchema

	// Unwrap EXPLAIN so the inner query's tables are visible: omni's span walker
	// only descends into SELECT/set-op at the top level.
	spanInput := single
	if explain, ok := node.(*parser.ExplainStmt); ok && explain.Statement != nil {
		if loc, ok := nodeSpan(explain.Statement); ok && loc.Start >= 0 && loc.End <= len(single) && loc.Start < loc.End {
			spanInput = single[loc.Start:loc.End]
		}
	}

	omniSpan, err := analysis.GetQuerySpan(spanInput)
	if err != nil {
		return nil, err
	}

	tables := q.toTableResources(omniSpan)

	if !readsResults {
		// DML/DDL/unknown: report accessed tables as source columns (table-level,
		// no column expansion), no results — mirroring the legacy basic span.
		return &base.QuerySpan{
			Type:          queryType,
			SourceColumns: tables,
			Results:       []base.QuerySpanResult{},
		}, nil
	}

	// Resolve relation aliases (e.g. "u" in "FROM users u") back to physical
	// table names so aliased column references match the expanded source
	// columns.
	aliasMap := q.buildAliasMap(omniSpan)

	// Expand each accessed table to its columns via metadata. If metadata is
	// unavailable (e.g. tests without a getter), fall back to the table-level
	// resource so callers still see what was read. orderedColumns preserves
	// FROM-table-then-metadata column order for positional SELECT * masking.
	//
	// Trino system pseudo-catalogs (system.*) and information_schema are
	// detected and skipped per resolved table inside expandTablesToColumns (not
	// via a query-text substring scan), so a real maskable table is still
	// expanded even when the same statement mentions "system." in a literal.
	orderedColumns, fullSourceColumns, notFound := q.expandTablesToColumns(omniSpan)

	results := q.buildResults(omniSpan, fullSourceColumns, orderedColumns, aliasMap)
	predicateColumns := q.mapPredicateColumns(omniSpan, fullSourceColumns, aliasMap)

	span := &base.QuerySpan{
		Type:             queryType,
		SourceColumns:    fullSourceColumns,
		PredicateColumns: predicateColumns,
		Results:          results,
	}
	if notFound != nil {
		span.NotFoundError = notFound
	}
	return span, nil
}

// toTableResources converts omni AccessTables into table-level
// base.ColumnResource keys (Column empty), applying default catalog/schema.
func (q *querySpanExtractor) toTableResources(span *analysis.QuerySpan) base.SourceColumnSet {
	out := base.SourceColumnSet{}
	if span == nil {
		return out
	}
	for _, t := range span.AccessTables {
		db, schema := q.resolveCatalogSchema(t.Catalog, t.Schema)
		out[base.ColumnResource{
			Database: db,
			Schema:   schema,
			Table:    t.Table,
		}] = true
	}
	return out
}

// resolveCatalogSchema fills in the default database (catalog) and schema for a
// table reference that omitted them, mirroring Trino's session-qualified
// resolution. The schema falls back to defaultTrinoSchema ("public") when no
// default schema is configured, matching the legacy resource-key format.
func (q *querySpanExtractor) resolveCatalogSchema(catalog, schema string) (string, string) {
	db := catalog
	if db == "" {
		db = q.defaultDatabase
	}
	if schema == "" {
		schema = q.defaultSchema
		if schema == "" {
			schema = defaultTrinoSchema
		}
	}
	return db, schema
}

// expandTablesToColumns expands each accessed table into one resource per column
// from the table's metadata. It returns the columns both as an ordered slice
// (AccessTables/FROM order, then metadata column order) and as a set. The
// ordered slice drives positional SELECT * masking, where the order must match
// the executed result's column order; the set is used for membership tests.
//
// A table whose metadata can't be resolved is kept as a single table-level
// resource so the read is still recorded. The first ResourceNotFoundError
// encountered is returned (non-fatally) so the caller can attach it to the span,
// matching the legacy listener behaviour.
func (q *querySpanExtractor) expandTablesToColumns(span *analysis.QuerySpan) ([]base.ColumnResource, base.SourceColumnSet, *base.ResourceNotFoundError) {
	ordered := make([]base.ColumnResource, 0)
	set := make(base.SourceColumnSet)
	var firstNotFound *base.ResourceNotFoundError
	// add records a column both in the membership set (deduped, used for lineage
	// matching) and in the ordered slice (NOT deduped, so a self-join such as
	// "FROM users u1 JOIN users u2" contributes each instance's columns and the
	// positional SELECT * masker stays aligned with Trino's 2N output columns).
	add := func(res base.ColumnResource) {
		set[res] = true
		ordered = append(ordered, res)
	}
	if span == nil {
		return ordered, set, nil
	}
	for _, t := range span.AccessTables {
		db, schema := q.resolveCatalogSchema(t.Catalog, t.Schema)
		if isSystemSchemaTable(db, schema) {
			// Trino's "system" catalog (system.runtime/metadata/jdbc.*) and the
			// per-catalog information_schema are pseudo-objects, not
			// Bytebase-tracked databases. Resolving them via the metadata getter
			// would yield a ResourceNotFoundError that sql_service turns into a
			// hard "failed to mask data" rejection of an otherwise-successful
			// result. System metadata is never masked, so keep the table-level
			// resource and skip the lookup. Detection keys on the resolved
			// catalog/schema (not a query-text substring), so a literal
			// containing "system." cannot suppress masking of a real table.
			//
			// Best-effort boundary: a system table contributes one placeholder
			// here but expands to several columns in a real `SELECT *`. For the
			// rare `SELECT * FROM <system> JOIN <real>` shape the positional
			// masker can therefore misalign the trailing real columns. Fully
			// fixing it needs system-table column metadata (not available); such
			// queries are already classified SelectInfoSchema. Tracked as a
			// follow-up rather than blocking the cutover.
			add(base.ColumnResource{Database: db, Schema: schema, Table: t.Table})
			continue
		}
		columns, err := q.tableColumns(db, schema, t.Table)
		if err != nil {
			var notFound *base.ResourceNotFoundError
			if errors.As(err, &notFound) && firstNotFound == nil {
				firstNotFound = notFound
			}
			// Keep the table-level resource (NotFound or any other error).
			add(base.ColumnResource{Database: db, Schema: schema, Table: t.Table})
			continue
		}
		if len(columns) == 0 {
			add(base.ColumnResource{Database: db, Schema: schema, Table: t.Table})
			continue
		}
		for _, col := range columns {
			add(base.ColumnResource{Database: db, Schema: schema, Table: t.Table, Column: col})
		}
	}
	return ordered, set, firstNotFound
}

// isSystemSchemaTable reports whether a resolved table reference targets a Trino
// system pseudo-object: the built-in "system" catalog (system.runtime.*,
// system.metadata.*, system.jdbc.*) or the per-catalog "information_schema".
// These are not Bytebase-tracked databases and their metadata is never masked.
func isSystemSchemaTable(catalog, schema string) bool {
	return strings.EqualFold(catalog, "system") || strings.EqualFold(schema, "information_schema")
}

// buildResults maps omni's per-output-column results onto base.QuerySpanResult.
// Each result's SourceColumns is the subset of the expanded source columns that
// share the result's column name (best-effort; omni does not resolve a select
// item back to its owning relation, so aliasMap translates a relation alias on
// the reference back to its physical table first).
//
// For SELECT *, the expansion uses orderedColumns (FROM-table then metadata
// column order) rather than ranging the source-column map: the masker applies
// per-result maskers positionally against the executed result's column order, so
// a nondeterministic order here could mask the wrong column and leak data.
func (*querySpanExtractor) buildResults(span *analysis.QuerySpan, fullSourceColumns base.SourceColumnSet, orderedColumns []base.ColumnResource, aliasMap map[string][]string) []base.QuerySpanResult {
	if span == nil {
		return []base.QuerySpanResult{}
	}
	results := make([]base.QuerySpanResult, 0, len(span.Results))
	for _, r := range span.Results {
		if r.Name == "*" {
			for _, col := range orderedColumns {
				results = append(results, base.QuerySpanResult{
					Name:          col.Column,
					SourceColumns: base.SourceColumnSet{col: true},
					IsPlainField:  true,
				})
			}
			continue
		}
		sourceColumns := base.SourceColumnSet{}
		for _, ref := range r.SourceColumns {
			addMatchingColumns(sourceColumns, fullSourceColumns, ref.Column, ref.Table, aliasMap)
		}
		results = append(results, base.QuerySpanResult{
			Name:          r.Name,
			SourceColumns: sourceColumns,
			IsPlainField:  len(r.SourceColumns) == 1,
		})
	}
	return results
}

// mapPredicateColumns maps omni's predicate column refs onto the expanded source
// columns (matching on column name, and table when the ref carries one; a
// relation alias on the ref is resolved via aliasMap).
func (*querySpanExtractor) mapPredicateColumns(span *analysis.QuerySpan, fullSourceColumns base.SourceColumnSet, aliasMap map[string][]string) base.SourceColumnSet {
	out := make(base.SourceColumnSet)
	if span == nil {
		return out
	}
	for _, ref := range span.PredicateColumns {
		addMatchingColumns(out, fullSourceColumns, ref.Column, ref.Table, aliasMap)
	}
	return out
}

// addMatchingColumns adds every column in fullSourceColumns whose name matches
// refColumn (case-insensitively, and whose table matches refTable when refTable
// is non-empty) to dst. refTable may be a relation alias; see tableMatches.
func addMatchingColumns(dst, fullSourceColumns base.SourceColumnSet, refColumn, refTable string, aliasMap map[string][]string) {
	for sc := range fullSourceColumns {
		if !strings.EqualFold(sc.Column, refColumn) {
			continue
		}
		if refTable == "" || tableMatches(sc.Table, refTable, aliasMap) {
			dst[sc] = true
		}
	}
}

// tableMatches reports whether the physical table scTable is named by refTable,
// either directly (its own name) or as a relation alias for it. Alias resolution
// is additive: the written refTable and every physical table the alias maps to
// are all accepted. omni's AccessTables is a flat, scope-less list, so an alias
// reused or shadowed across subqueries yields several candidates; accepting all
// of them over-includes (conservatively masks more) rather than risk an
// under-match that would leave a sensitive column unmasked.
func tableMatches(scTable, refTable string, aliasMap map[string][]string) bool {
	if strings.EqualFold(scTable, refTable) {
		return true
	}
	for _, phys := range aliasMap[strings.ToLower(refTable)] {
		if strings.EqualFold(scTable, phys) {
			return true
		}
	}
	return false
}

// buildAliasMap maps each relation alias (lower-cased) to the physical table
// names it stands for, so aliased column references resolve back to base tables
// during column matching. Because omni's AccessTables is flat (carries no SQL
// scope), an alias reused in different scopes maps to several tables; all are
// kept so matching can over-include rather than under-match. An alias equal to
// its own table name carries no information and is skipped.
func (*querySpanExtractor) buildAliasMap(span *analysis.QuerySpan) map[string][]string {
	if span == nil {
		return nil
	}
	out := make(map[string][]string)
	for _, t := range span.AccessTables {
		if t.Alias == "" || strings.EqualFold(t.Alias, t.Table) {
			continue
		}
		key := strings.ToLower(t.Alias)
		if !containsFold(out[key], t.Table) {
			out[key] = append(out[key], t.Table)
		}
	}
	return out
}

// containsFold reports whether list contains s under case-insensitive
// comparison.
func containsFold(list []string, s string) bool {
	for _, e := range list {
		if strings.EqualFold(e, s) {
			return true
		}
	}
	return false
}

// tableColumns returns the column names of the given table or view, honouring
// the case-sensitivity flag for object lookup.
func (q *querySpanExtractor) tableColumns(db, schema, table string) ([]string, error) {
	metadata, err := q.getDatabaseMetadata(db)
	if err != nil {
		return nil, err
	}

	schemaMeta := metadata.GetSchemaMetadata(schema)
	if schemaMeta == nil && q.ignoreCaseSensitive {
		for _, name := range metadata.ListSchemaNames() {
			if strings.EqualFold(name, schema) {
				schemaMeta = metadata.GetSchemaMetadata(name)
				break
			}
		}
	}
	if schemaMeta == nil {
		return nil, &base.ResourceNotFoundError{
			Database: &db,
			Schema:   &schema,
		}
	}

	// Table first.
	tableMeta := schemaMeta.GetTable(table)
	if tableMeta == nil && q.ignoreCaseSensitive {
		for _, name := range schemaMeta.ListTableNames() {
			if strings.EqualFold(name, table) {
				tableMeta = schemaMeta.GetTable(name)
				break
			}
		}
	}
	if tableMeta != nil {
		var cols []string
		for _, col := range tableMeta.GetProto().GetColumns() {
			cols = append(cols, col.Name)
		}
		return cols, nil
	}

	// Then view.
	viewMeta := schemaMeta.GetView(table)
	if viewMeta == nil && q.ignoreCaseSensitive {
		for _, name := range schemaMeta.ListViewNames() {
			if strings.EqualFold(name, table) {
				viewMeta = schemaMeta.GetView(name)
				break
			}
		}
	}
	if viewMeta != nil {
		var cols []string
		for _, col := range viewMeta.GetColumns() {
			cols = append(cols, col.Name)
		}
		return cols, nil
	}

	return nil, &base.ResourceNotFoundError{
		Database: &db,
		Schema:   &schema,
		Table:    &table,
	}
}

// getDatabaseMetadata fetches (and caches) metadata for the given database.
func (q *querySpanExtractor) getDatabaseMetadata(database string) (*model.DatabaseMetadata, error) {
	if database == "" {
		database = q.defaultDatabase
	}
	if meta, ok := q.metaCache[database]; ok {
		return meta, nil
	}
	if q.gCtx.GetDatabaseMetadataFunc == nil {
		return nil, &base.ResourceNotFoundError{Database: &database}
	}
	_, meta, err := q.gCtx.GetDatabaseMetadataFunc(q.ctx, q.gCtx.InstanceID, database)
	if err != nil {
		var notFound *base.ResourceNotFoundError
		if errors.As(err, &notFound) {
			return nil, err
		}
		return nil, errors.Wrapf(err, "failed to get database metadata for database: %s", database)
	}
	if meta == nil {
		empty := &model.DatabaseMetadata{}
		q.metaCache[database] = empty
		return empty, nil
	}
	q.metaCache[database] = meta
	return meta, nil
}

// nodeText returns the source substring covered by node n within statement, or
// "" when the node's location is unusable.
// nodeSpan returns the source byte range of an omni statement node. Most omni
// node types expose Span() ast.Loc; *parser.QueryStmt carries its range on a
// plain Loc field. (ast.NodeLoc only knows ast-package nodes — File/Identifier/
// QualifiedName — not the parser-package statement nodes, so it cannot be used
// to unwrap an EXPLAIN's inner statement.)
func nodeSpan(n ast.Node) (ast.Loc, bool) {
	if qs, ok := n.(*parser.QueryStmt); ok {
		return qs.Loc, true
	}
	if sp, ok := n.(interface{ Span() ast.Loc }); ok {
		return sp.Span(), true
	}
	return ast.Loc{}, false
}
