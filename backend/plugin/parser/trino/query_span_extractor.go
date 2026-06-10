package trino

import (
	"context"
	"strings"

	"github.com/bytebase/omni/trino/analysis"
	"github.com/bytebase/omni/trino/ast"
	"github.com/bytebase/omni/trino/catalog"
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

	omniSpan, err := analysis.GetQuerySpanWithCatalog(spanInput, q.buildSpanCatalog(ctx, spanInput))
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

// buildSpanCatalog constructs the omni Trino catalog the query-span analysis
// resolves against: views carry their defining query (so lineage through a view
// reaches the underlying base-table columns — masking config only attaches to
// table columns), and tables carry their column lists (so omni expands
// SELECT * to the exact projection). The session context is the extractor's
// default catalog/schema (with the same "public" fallback as
// resolveCatalogSchema, so view resolution and resource keys agree).
//
// Like the completion catalog, every catalog name is registered cheaply but
// full metadata is loaded only for the catalogs the statement needs (the
// default one and any named in the statement). Metadata fetches go through the
// extractor's cache, so the later expandTablesToColumns lookups reuse them.
// Returns nil — omni then behaves exactly catalog-less — when metadata is
// unavailable (e.g. tests without a getter).
func (q *querySpanExtractor) buildSpanCatalog(ctx context.Context, statement string) *catalog.Catalog {
	q.ctx = ctx
	if q.gCtx.GetDatabaseMetadataFunc == nil || q.gCtx.ListDatabaseNamesFunc == nil {
		return nil
	}
	names, err := q.gCtx.ListDatabaseNamesFunc(ctx, q.gCtx.InstanceID)
	if err != nil || len(names) == 0 {
		return nil
	}

	defaultDB := catalog.Normalize(q.defaultDatabase)
	cat := catalog.New()
	for _, dbName := range names {
		cat.EnsureCatalog(catalog.Normalize(dbName))
	}

	// Load catalogs transitively: the statement names some catalogs; the view
	// definitions loaded with them may name FURTHER catalogs (a view in the
	// default catalog defined over another catalog's table), which omni needs
	// loaded to resolve lineage through the view. Each round scans the text
	// corpus (statement + every loaded view definition) and loads any
	// newly-referenced catalog; it terminates because each catalog loads at
	// most once.
	corpus := strings.ToLower(statement)
	loaded := make(map[string]bool, len(names))
	for {
		progressed := false
		for _, dbName := range names {
			norm := catalog.Normalize(dbName)
			if loaded[norm] || !catalogNeeded(norm, defaultDB, corpus) {
				continue
			}
			loaded[norm] = true
			progressed = true
			meta, err := q.getDatabaseMetadata(dbName)
			if err != nil || meta == nil {
				continue
			}
			for _, def := range loadCatalogMetadata(cat, norm, meta) {
				corpus += "\n" + strings.ToLower(def)
			}
		}
		if !progressed {
			break
		}
	}

	if q.defaultDatabase != "" {
		cat.SetCurrentCatalog(defaultDB)
	}
	schema := q.defaultSchema
	if schema == "" {
		schema = defaultTrinoSchema
	}
	cat.SetCurrentSchema(catalog.Normalize(schema))
	return cat
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
func (q *querySpanExtractor) buildResults(span *analysis.QuerySpan, fullSourceColumns base.SourceColumnSet, orderedColumns []base.ColumnResource, aliasMap map[string][]string) []base.QuerySpanResult {
	if span == nil {
		return []base.QuerySpanResult{}
	}
	results := make([]base.QuerySpanResult, 0, len(span.Results))
	for _, r := range span.Results {
		// Unqualified star "*": expand to every source column, in result order.
		//
		// Best-effort boundary: omni records a star as a single "*" result over
		// the flat AccessTables and does not expose the resolved output relation,
		// so a star over a CTE or derived table that projects a subset/reordering
		// (e.g. WITH c AS (SELECT email FROM users) SELECT * FROM c) is
		// indistinguishable here from SELECT * over the base table and expands to
		// the base table's columns. This matches the prior ANTLR plugin (which
		// produced no columns for such stars) and is tracked as a follow-up; a
		// correct fix needs omni's analysis to expose the star's output columns.
		if r.Name == "*" {
			results = appendStarColumns(results, orderedColumns, nil)
			continue
		}
		// Qualified star "<rel>.*": omni names the result "<rel>.*" with a single
		// source ref that is the relation itself. Without this branch it would
		// fall through to the column-name match below, find no column literally
		// named "<rel>.*", and leave SourceColumns empty — the masker would then
		// treat those (possibly sensitive) columns as constants and return them
		// unmasked. Resolve the ref to the specific relation(s) it names (by
		// database/schema/table, so same-named tables in different schemas are
		// distinguished) and expand only those columns, preserving order. The
		// detection inspects the source-ref shape, not just the display name, so a
		// column merely aliased to a name ending in ".*" is not mistaken for a
		// star.
		if isQualifiedStar(r) {
			targets := q.qualifiedStarTargets(r.SourceColumns[0], span)
			results = appendStarColumns(results, orderedColumns, targets)
			continue
		}
		sourceColumns := base.SourceColumnSet{}
		for _, ref := range r.SourceColumns {
			addMatchingColumns(sourceColumns, fullSourceColumns, ref.Column, ref.Table, aliasMap)
		}
		results = append(results, base.QuerySpanResult{
			Name:          r.Name,
			SourceColumns: sourceColumns,
			// Plainness keys on the MAPPED physical set, not omni's raw ref
			// count: the catalog-aware resolver additively restates a plain
			// column as written + catalog-qualified refs, which dedupe back to
			// the one physical column here. Known drift: an expression over one
			// column repeated (phone || phone) also maps to a single physical
			// column and reads as plain — omni's resolver dedups refs, so the
			// distinction is unrecoverable here; inert for Trino, which is not
			// in EngineSupportQuerySpanPlainField.
			IsPlainField: len(sourceColumns) == 1,
		})
	}
	return results
}

// appendStarColumns appends one plain-field result per column in orderedColumns,
// in order (the positional masker maps each result to the executed result column
// at the same index). When targets is non-nil — a qualified star "<rel>.*" — only
// columns whose (database, schema, table) key is in targets are appended; a nil
// targets (unqualified "*") appends every column.
func appendStarColumns(results []base.QuerySpanResult, orderedColumns []base.ColumnResource, targets map[base.ColumnResource]bool) []base.QuerySpanResult {
	for _, col := range orderedColumns {
		if targets != nil {
			if !targets[base.ColumnResource{Database: col.Database, Schema: col.Schema, Table: col.Table}] {
				continue
			}
		}
		results = append(results, base.QuerySpanResult{
			Name:          col.Column,
			SourceColumns: base.SourceColumnSet{col: true},
			IsPlainField:  true,
		})
	}
	return results
}

// qualifiedStarResult reports whether r is a genuine qualified star "<rel>.*"
// (returning the relation qualifier), as opposed to an ordinary column that
// happens to be aliased to a name ending in ".*". omni represents a qualified
// star as a result whose Name is "<rel>.*" and whose single source ref is the
// relation name itself (Column == <rel>, no catalog/schema/table qualifier); a
// real column aliased to e.g. "u.*" instead carries the underlying column name
// in its source ref, so it is handled as a normal column and not star-expanded
// (which would misalign the positional masker and could leak).
func isQualifiedStar(r analysis.ColumnInfo) bool {
	qualifier, ok := qualifiedStarQualifier(r.Name)
	if !ok {
		return false
	}
	// omni represents a qualified star "<...>.<rel>.*" as a single source ref
	// whose Column is the relation name <rel> (any schema/catalog path written
	// before it lands in the ref's Table/Schema fields). A real column aliased to
	// a name ending in ".*" instead carries the underlying column name in Column,
	// which will not equal the qualifier, so it is handled as a normal column and
	// not star-expanded (avoiding a positional-masker misalignment/leak).
	//
	// Matching only on Column == qualifier (not on empty table/schema) is
	// deliberate: it keeps genuine table/schema/catalog-qualified stars like
	// public.users.* working. The sole residual ambiguity is the contrived case
	// of a column literally named like the qualifier and aliased to
	// "<qualifier>.*", which omni renders identically to a real star.
	if len(r.SourceColumns) != 1 {
		return false
	}
	return strings.EqualFold(r.SourceColumns[0].Column, qualifier)
}

// qualifiedStarTargets resolves the relation(s) named by a qualified-star source
// ref into a set of (database, schema, table) keys, so the star expands to
// exactly that relation's columns — distinguishing same-named tables in
// different schemas or catalogs. omni encodes the written qualifier in the ref
// with the parts shifted right: Column is the relation/alias, Table is the schema
// part, and Schema is the catalog part.
func (q *querySpanExtractor) qualifiedStarTargets(ref analysis.ColumnRef, span *analysis.QuerySpan) map[base.ColumnResource]bool {
	targets := make(map[base.ColumnResource]bool)
	// Alias-qualified star (u.*): collect every base table that carries the
	// alias. A reused/shadowed alias yields several, which over-includes (the
	// extra columns are ignored positionally) rather than risking a leak.
	aliasMatched := false
	for _, t := range span.AccessTables {
		if t.Alias != "" && strings.EqualFold(t.Alias, ref.Column) {
			db, schema := q.resolveCatalogSchema(t.Catalog, t.Schema)
			targets[base.ColumnResource{Database: db, Schema: schema, Table: t.Table}] = true
			aliasMatched = true
		}
	}
	if aliasMatched {
		return targets
	}
	// Table-name-qualified star (users.* / public.users.* / cat.sch.users.*):
	// ref.Column is the table, ref.Table the schema part, ref.Schema the catalog.
	// Only target a base table actually present in AccessTables, so a qualifier
	// naming a CTE/derived relation (which omni omits from AccessTables) does not
	// accidentally target an unrelated physical table of the same name. A
	// qualified star over a CTE/derived relation thus yields no columns here —
	// the same best-effort boundary as an unqualified star over such a relation
	// (omni does not expose the resolved output projection; tracked as a
	// follow-up).
	wantDB, wantSchema := q.resolveCatalogSchema(ref.Schema, ref.Table)
	for _, t := range span.AccessTables {
		db, schema := q.resolveCatalogSchema(t.Catalog, t.Schema)
		if strings.EqualFold(db, wantDB) && strings.EqualFold(schema, wantSchema) && strings.EqualFold(t.Table, ref.Column) {
			targets[base.ColumnResource{Database: db, Schema: schema, Table: t.Table}] = true
		}
	}
	return targets
}

// qualifiedStarQualifier returns the relation qualifier of a qualified-star
// result name like "u.*" or "catalog.schema.t.*" (the rightmost component before
// the trailing ".*"), and whether name is a qualified star. Column matching keys
// on the table/alias, so only the last component is returned.
func qualifiedStarQualifier(name string) (string, bool) {
	if !strings.HasSuffix(name, ".*") {
		return "", false
	}
	qualifier := strings.TrimSuffix(name, ".*")
	if i := strings.LastIndex(qualifier, "."); i >= 0 {
		qualifier = qualifier[i+1:]
	}
	if qualifier == "" {
		return "", false
	}
	return qualifier, true
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
