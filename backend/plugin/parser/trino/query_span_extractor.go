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

	// Expand each accessed table to its columns via metadata. If metadata is
	// unavailable (e.g. tests without a getter), fall back to the table-level
	// resource so callers still see what was read.
	fullSourceColumns, notFound := q.expandTablesToColumns(tables)

	results := q.buildResults(omniSpan, fullSourceColumns)
	predicateColumns := q.mapPredicateColumns(omniSpan, fullSourceColumns)

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
		db := t.Catalog
		if db == "" {
			db = q.defaultDatabase
		}
		schema := t.Schema
		if schema == "" {
			schema = q.defaultSchema
			if schema == "" {
				schema = defaultTrinoSchema
			}
		}
		out[base.ColumnResource{
			Database: db,
			Schema:   schema,
			Table:    t.Table,
		}] = true
	}
	return out
}

// expandTablesToColumns replaces each table-level resource with one resource per
// column from the table's metadata. Tables whose metadata can't be resolved are
// kept as table-level resources. The first ResourceNotFoundError encountered is
// returned (non-fatally) so the caller can attach it to the span, matching the
// legacy listener behaviour.
func (q *querySpanExtractor) expandTablesToColumns(tables base.SourceColumnSet) (base.SourceColumnSet, *base.ResourceNotFoundError) {
	out := make(base.SourceColumnSet)
	var firstNotFound *base.ResourceNotFoundError
	for resource := range tables {
		if resource.Column != "" {
			out[resource] = true
			continue
		}
		columns, err := q.tableColumns(resource.Database, resource.Schema, resource.Table)
		if err != nil {
			var notFound *base.ResourceNotFoundError
			if errors.As(err, &notFound) {
				if firstNotFound == nil {
					firstNotFound = notFound
				}
				// Keep the table-level resource so the read is still recorded.
				out[resource] = true
				continue
			}
			// Unexpected error: keep the table-level resource and move on.
			out[resource] = true
			continue
		}
		if len(columns) == 0 {
			out[resource] = true
			continue
		}
		for _, col := range columns {
			out[base.ColumnResource{
				Database: resource.Database,
				Schema:   resource.Schema,
				Table:    resource.Table,
				Column:   col,
			}] = true
		}
	}
	return out, firstNotFound
}

// buildResults maps omni's per-output-column results onto base.QuerySpanResult.
// Each result's SourceColumns is the subset of the expanded source columns that
// share the result's column name (best-effort; omni does not resolve a select
// item back to its owning relation).
func (*querySpanExtractor) buildResults(span *analysis.QuerySpan, fullSourceColumns base.SourceColumnSet) []base.QuerySpanResult {
	if span == nil {
		return []base.QuerySpanResult{}
	}
	results := make([]base.QuerySpanResult, 0, len(span.Results))
	for _, r := range span.Results {
		if r.Name == "*" {
			// SELECT * : expand to every source column, ordered arbitrarily.
			for col := range fullSourceColumns {
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
			addMatchingColumns(sourceColumns, fullSourceColumns, ref.Column, ref.Table)
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
// columns (matching on column name, and table when the ref carries one).
func (*querySpanExtractor) mapPredicateColumns(span *analysis.QuerySpan, fullSourceColumns base.SourceColumnSet) base.SourceColumnSet {
	out := make(base.SourceColumnSet)
	if span == nil {
		return out
	}
	for _, ref := range span.PredicateColumns {
		addMatchingColumns(out, fullSourceColumns, ref.Column, ref.Table)
	}
	return out
}

// addMatchingColumns adds every column in fullSourceColumns whose name matches
// refColumn (case-insensitively, and refTable when it is non-empty) to dst.
func addMatchingColumns(dst, fullSourceColumns base.SourceColumnSet, refColumn, refTable string) {
	for sc := range fullSourceColumns {
		if strings.EqualFold(sc.Column, refColumn) && (refTable == "" || strings.EqualFold(sc.Table, refTable)) {
			dst[sc] = true
		}
	}
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
