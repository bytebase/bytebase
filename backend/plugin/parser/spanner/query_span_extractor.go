package spanner

import (
	"context"
	"sort"
	"strings"

	"github.com/bytebase/omni/googlesql/analysis"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/store/model"
)

// querySpanExtractor analyses a single Spanner (GoogleSQL) statement and
// produces a base.QuerySpan: the set of base tables it accesses, the per-output
// column lineage (expanded against catalog metadata when available), and its
// statement classification.
//
// The extractor delegates structural lineage to omni's analysis.GetQuerySpan
// (DialectSpanner), then applies the bytebase-specific post-processing the
// legacy plugin performed:
//   - the Spanner name model: named schemas under ONE database (a 2-part sch.t
//     is Schema=sch, Table=t; the db part of a 3-part db.sch.t is IGNORED — the
//     legacy resolver always used the session database), mapped onto
//     base.ColumnResource with Database always the default database;
//   - the user/system handling: a query reading ONLY system tables
//     (INFORMATION_SCHEMA / SPANNER_SYS) early-returns an EMPTY
//     SelectInfoSchema span without touching metadata, and a mixed user+system
//     query is rejected (base.MixUserSystemTablesError) — both exactly the
//     legacy extractor's behavior;
//   - expansion of each accessed table to its columns via the metadata getter so
//     SELECT * and bare column references resolve to physical columns.
//
// This mirrors the BigQuery cutover (and the proven Trino #20517) structurally;
// the deltas are the schema-model mapping, the SPANNER_SYS system set, and the
// system-only early return.
type querySpanExtractor struct {
	ctx context.Context

	defaultDatabase string

	gCtx base.GetQuerySpanContext

	// metaCache memoises database metadata lookups within a single span.
	metaCache map[string]*model.DatabaseMetadata
}

// newQuerySpanExtractor creates a new Spanner query span extractor. Its
// signature matches the legacy 3-arg form. The ignoreCaseSensitive flag is
// accepted for signature parity but unused — the legacy extractor ignored it
// too (its constructor took `_ bool`); schema names match case-sensitively and
// table names fall back to a case-folded match, exactly the legacy
// findTableSchema behavior.
func newQuerySpanExtractor(defaultDatabase string, gCtx base.GetQuerySpanContext, _ bool) *querySpanExtractor {
	return &querySpanExtractor{
		defaultDatabase: defaultDatabase,
		gCtx:            gCtx,
		metaCache:       make(map[string]*model.DatabaseMetadata),
	}
}

// getQuerySpan extracts the query span for a single Spanner statement.
func (q *querySpanExtractor) getQuerySpan(ctx context.Context, statement string) (*base.QuerySpan, error) {
	q.ctx = ctx

	// query-span analysis expects exactly one non-empty statement (matching the
	// legacy behaviour, which parsed via ParseSpannerGoogleSQL and rejected != 1).
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
		return nil, errors.Errorf("expecting only one statement to get query span, but got %d", nonEmpty)
	}

	omniSpan, err := analysis.GetQuerySpan(single, analysis.DialectSpanner)
	if err != nil {
		return nil, err
	}

	// Reject a statement that reads from both user and system
	// (INFORMATION_SCHEMA / SPANNER_SYS) tables, matching the legacy
	// isMixedQuery → MixUserSystemTablesError.
	allSystems, mixed := classifyAccess(omniSpan)
	if mixed {
		return nil, base.MixUserSystemTablesError
	}
	// A query reading exclusively from system tables EARLY-RETURNS an EMPTY
	// SelectInfoSchema span — no source columns, no results, no metadata access.
	// This is the legacy spanner extractor's exact behavior (it returned
	// base.QuerySpan{Type: SelectInfoSchema} before resolving anything); system
	// metadata is never masked, and resolving these pseudo-tables would only risk
	// a spurious ResourceNotFoundError.
	if allSystems {
		return &base.QuerySpan{
			Type:          base.SelectInfoSchema,
			SourceColumns: base.SourceColumnSet{},
			Results:       []base.QuerySpanResult{},
		}, nil
	}

	queryType := mapQueryType(omniSpan.Type)
	// Legacy-spanner special case: a SET statement classifies as Select ("treat
	// SAFE SET as select"); omni classifies it Unknown.
	if queryType == base.QueryTypeUnknown && isSetStatement(single) {
		queryType = base.Select
	}

	// Top-level SourceColumns is the set of accessed tables at the TABLE level
	// (Column empty) — BigQuery's legacy span reports accessed tables, not
	// column-expanded source columns (unlike Trino). Uses the table name AS
	// WRITTEN in the query (the legacy accessTableListener recorded the written
	// identifier, e.g. lowercase `people`), with the default dataset filled in.
	accessTables := q.toTableResources(omniSpan)

	// Only a read-only query produces per-column results; DML/DDL/Explain/Unknown
	// return the accessed tables with no per-column lineage — exactly the legacy
	// behaviour (it returned an empty span for any non-Select). SelectInfoSchema is
	// a read so it is treated like Select here.
	if queryType != base.Select && queryType != base.SelectInfoSchema {
		return &base.QuerySpan{
			Type:          queryType,
			SourceColumns: base.SourceColumnSet{},
			Results:       []base.QuerySpanResult{},
		}, nil
	}

	// Resolve relation aliases (e.g. "g" in "FROM galleries g") back to physical
	// table names so aliased column references match the expanded source columns.
	aliasMap := q.buildAliasMap(omniSpan)

	// Expand each accessed table to its columns via metadata. orderedColumns
	// preserves AccessTables(FROM)-then-metadata column order for positional
	// SELECT * masking; fullSourceColumns is the membership set used for lineage
	// matching. The per-column resources use the METADATA-canonical table name
	// (e.g. `PEOPLE`), matching the legacy findTableSchema/PhysicalTable, which is
	// why result lineage carries metadata casing while the table-level
	// SourceColumns above carry the written casing.
	orderedColumns, fullSourceColumns, notFound := q.expandTablesToColumns(omniSpan)

	results := q.buildResults(omniSpan, fullSourceColumns, orderedColumns, aliasMap)

	span := &base.QuerySpan{
		Type:          queryType,
		SourceColumns: accessTables,
		// PredicateColumns is intentionally left empty: the legacy Spanner
		// extractor never populated predicate-column lineage (the recorded corpus
		// has predicatecolumns: [] on every case). omni DOES collect predicate
		// columns, but surfacing them would diverge from the legacy contract the
		// masking layer was calibrated against.
		Results: results,
	}
	if notFound != nil {
		span.NotFoundError = notFound
	}
	return span, nil
}

// classifyAccess reports (allSystems, mixed) over the span's access tables,
// using omni's per-table IsSystem flag (BigQuery: Schema == INFORMATION_SCHEMA).
// It reproduces the legacy isMixedQuery: mixed when both a system and a user
// table are present; allSystems when at least one system table is present and no
// user table is (the legacy `!hasUser && hasSystem`).
func classifyAccess(span *analysis.QuerySpan) (allSystems, mixed bool) {
	if span == nil {
		return false, false
	}
	hasSystem, hasUser := false, false
	for _, t := range span.AccessTables {
		if t.IsSystem {
			hasSystem = true
		} else {
			hasUser = true
		}
	}
	if hasSystem && hasUser {
		return false, true
	}
	return hasSystem && !hasUser, false
}

// toTableResources converts omni AccessTables into table-level
// base.ColumnResource keys (Column empty). The table and schema names are kept
// AS WRITTEN (matching the legacy accessTableListener); Database is ALWAYS the
// default database — the legacy spanner resolver silently ignored the db part
// of a 3-part db.schema.table reference and recorded the session database.
func (q *querySpanExtractor) toTableResources(span *analysis.QuerySpan) base.SourceColumnSet {
	out := base.SourceColumnSet{}
	if span == nil {
		return out
	}
	for _, t := range span.AccessTables {
		out[base.ColumnResource{
			Database: q.defaultDatabase,
			Schema:   t.Schema,
			Table:    t.Table,
		}] = true
	}
	// CTE references are not physical tables, but the legacy access-table listener
	// recorded every FROM table path — CTE references included — so union them into
	// the table-level SourceColumns.
	for _, t := range span.CTEReferences {
		out[base.ColumnResource{
			Database: q.defaultDatabase,
			Schema:   t.Schema,
			Table:    t.Table,
		}] = true
	}
	return out
}

// expandTablesToColumns expands each accessed table into one resource per column
// from the table's metadata. It returns the columns both as an ordered slice
// (AccessTables/FROM order, then metadata column order) and as a set. The ordered
// slice drives positional SELECT * masking, where the order must match the
// executed result's column order; the set is used for membership tests.
//
// A table whose metadata can't be resolved is kept as a single table-level
// resource so the read is still recorded, and the first ResourceNotFoundError
// encountered is returned (non-fatally) for the caller to attach to the span.
// Each per-column resource uses the METADATA-canonical dataset/table/column
// names (the legacy findTableSchema returned a PhysicalTable keyed on the proto
// name), so result lineage carries metadata casing.
func (q *querySpanExtractor) expandTablesToColumns(span *analysis.QuerySpan) ([]base.ColumnResource, base.SourceColumnSet, *base.ResourceNotFoundError) {
	ordered := make([]base.ColumnResource, 0)
	set := make(base.SourceColumnSet)
	var firstNotFound *base.ResourceNotFoundError
	// add records a column both in the membership set (deduped, used for lineage
	// matching) and in the ordered slice (NOT deduped, so a self-join contributes
	// each instance's columns and the positional SELECT * masker stays aligned with
	// the executed output columns).
	add := func(res base.ColumnResource) {
		set[res] = true
		ordered = append(ordered, res)
	}
	if span == nil {
		return ordered, set, nil
	}
	for _, t := range span.AccessTables {
		if t.IsSystem {
			// Unreachable in practice: a system-only query early-returned in
			// getQuerySpan and a mixed user/system query was rejected. Kept as a
			// belt-and-braces guard — a system pseudo-table is never expanded
			// against metadata.
			add(base.ColumnResource{Database: q.defaultDatabase, Schema: t.Schema, Table: t.Table})
			continue
		}
		canonicalTable, columns, err := q.tableColumns(t.Schema, t.Table)
		if err != nil {
			var notFound *base.ResourceNotFoundError
			if errors.As(err, &notFound) && firstNotFound == nil {
				firstNotFound = notFound
			}
			// Keep the table-level resource (NotFound or any other error).
			add(base.ColumnResource{Database: q.defaultDatabase, Schema: t.Schema, Table: t.Table})
			continue
		}
		if len(columns) == 0 {
			add(base.ColumnResource{Database: q.defaultDatabase, Schema: t.Schema, Table: canonicalTable})
			continue
		}
		for _, col := range columns {
			add(base.ColumnResource{Database: q.defaultDatabase, Schema: t.Schema, Table: canonicalTable, Column: col})
		}
	}
	return ordered, set, firstNotFound
}

// buildResults maps omni's per-output columns onto base.QuerySpanResult. Each
// result's SourceColumns is the subset of the expanded source columns that share
// the result's referenced column name (best-effort; omni does not resolve a
// select item back to its owning relation, so aliasMap translates a relation
// alias on the reference back to its physical table first).
//
// For an unqualified "*" the expansion uses orderedColumns (FROM-table then
// metadata column order) rather than ranging the source-column map, so the
// per-result maskers, applied positionally against the executed result's column
// order, line up.
//
// omni's analysis resolves relation projections (CTE/derived/recursive bodies,
// set-operation merges with star markers, SELECT * EXCEPT/REPLACE, JOIN USING
// coalescing, UNNEST element lineage) before handing over; the one piece left
// to this metadata-aware consumer is enumerating a PHYSICAL table's columns
// (StarSegment.BaseTable), which omni cannot do catalog-free. Validated against
// the legacy-resolver differential corpus (test-data/query-span/standard.yaml,
// recorded from the legacy ANTLR resolver) plus the leak-pin unit tests for
// shapes the legacy resolver could not record.
func (q *querySpanExtractor) buildResults(span *analysis.QuerySpan, fullSourceColumns base.SourceColumnSet, orderedColumns []base.ColumnResource, aliasMap map[string][]string) []base.QuerySpanResult {
	if span == nil {
		return []base.QuerySpanResult{}
	}
	return q.expandColumnInfos(span.Results, fullSourceColumns, aliasMap)
}

// expandColumnInfos expands a list of omni ColumnInfo (one query body's resolved
// projection) into base.QuerySpanResult. It is called for the top-level results
// and RECURSIVELY by buildSetOpMergeResults for each arm of a deferred set-op
// merge. Each ColumnInfo is one of: a star item (StarSegments, expanded + its
// EXCEPT/REPLACE applied), a per-position base-star set-op merge (StarMerge), a
// whole deferred set-op merge (SetOpMerge, both arms expanded then position-
// merged), or a concrete output column.
func (q *querySpanExtractor) expandColumnInfos(infos []analysis.ColumnInfo, fullSourceColumns base.SourceColumnSet, aliasMap map[string][]string) []base.QuerySpanResult {
	results := make([]base.QuerySpanResult, 0, len(infos))
	for _, r := range infos {
		switch {
		case r.StarSegments != nil:
			// A `*` / `rel.*` / CTE-or-derived star: omni resolved its ordered
			// expansion into segments (a base table to enumerate, or an
			// already-resolved concrete column). Expand them, then apply the star's
			// EXCEPT/REPLACE modifiers with the legacy name-collision last-wins dedup.
			results = append(results, q.expandStarSegments(r, fullSourceColumns, aliasMap)...)
		case r.SetOpMerge != nil:
			// A whole deferred set-operation merge whose arms carry un-enumerable
			// stars: expand each arm fully (recursively) and position-merge them,
			// reproducing the legacy "fully resolve each arm, then zip".
			results = append(results, q.buildSetOpMergeResults(r.SetOpMerge, fullSourceColumns, aliasMap)...)
		case r.StarMerge != nil:
			// A set-operation merge position whose other arm is a base-table star:
			// union the concrete arm's lineage with that table's Index-th column.
			results = append(results, q.buildStarMergeResult(r, fullSourceColumns, aliasMap))
		default:
			sourceColumns := base.SourceColumnSet{}
			for _, ref := range r.SourceColumns {
				addMatchingColumns(sourceColumns, fullSourceColumns, ref.Column, ref.Table, aliasMap)
			}
			name := resultName(r)
			// A base-table FIELD passthrough (the JOIN ... USING coalesced key)
			// keeps the field's METADATA case — the legacy resolver named it after
			// the left PhysicalTable's field, not the written/upper-cased token.
			if r.BaseFieldName && len(r.SourceColumns) > 0 {
				if canon := q.canonicalColumnCase(r.SourceColumns[0], r.Name); canon != "" {
					name = canon
				}
			}
			results = append(results, base.QuerySpanResult{
				Name:          name,
				SourceColumns: sourceColumns,
				// IsPlainField is carried from omni: a `SELECT *` / `rel.*` expansion
				// column over a base table is plain; an explicit select-list item, a
				// set-op merge, and a join-left column are not (omni already applied
				// these rules), matching base.PhysicalTable.GetQuerySpanResult vs the
				// rewrapped PseudoTable columns.
				IsPlainField: r.IsPlain,
			})
		}
	}
	return results
}

// buildSetOpMergeResults expands a deferred set-operation merge: each arm is
// expanded fully (a base-table star against metadata, an EXCEPT/REPLACE star with
// its modifiers, a nested merge — recursively via expandColumnInfos), then the
// two expanded column lists are position-merged. This reproduces the legacy
// extractTableSourceFromQuerySetOperation: output NAME comes from the LEFT arm
// (the first-select-name rule), SourceColumns are the union of both arms at that
// position, and the merged column is never a plain field. omni emits this only
// when a per-position StarMerge cannot express the merge (a base-star UNION
// base-star, or an EXCEPT/REPLACE star arm in a set operation) — the arity is
// known only here, after metadata expansion.
//
// Width handling mirrors masking safety, not the legacy hard error: legacy
// rejects an unequal-width set operation outright (so the query never masks),
// while omni — best-effort over a possibly-already-rejected statement — merges up
// to the LONGER arm so no arm's column is dropped (a dropped sensitive column
// would under-attribute). The shorter arm simply contributes nothing past its
// length. Equal-width arms (the only valid set operations) merge exactly.
func (q *querySpanExtractor) buildSetOpMergeResults(m *analysis.SetOpMergeInfo, fullSourceColumns base.SourceColumnSet, aliasMap map[string][]string) []base.QuerySpanResult {
	left := q.expandColumnInfos(m.Left, fullSourceColumns, aliasMap)
	right := q.expandColumnInfos(m.Right, fullSourceColumns, aliasMap)
	if m.ByName {
		// BY NAME / CORRESPONDING aligns the arms by output column NAME, not
		// ordinal — an ordinal merge here mis-attributes whenever the arms'
		// column orders differ, and ignoring MatchColumns emits the wrong arity
		// (BY NAME ON (cols) outputs ONLY the listed columns), shifting the
		// positional masker off every real output column.
		return mergeExpandedByName(left, right, m.MatchColumns)
	}
	n := len(left)
	if len(right) > n {
		n = len(right)
	}
	merged := make([]base.QuerySpanResult, 0, n)
	for i := 0; i < n; i++ {
		sourceColumns := base.SourceColumnSet{}
		name := ""
		if i < len(left) {
			name = left[i].Name
			for sc := range left[i].SourceColumns {
				sourceColumns[sc] = true
			}
		}
		if i < len(right) {
			if name == "" {
				name = right[i].Name
			}
			for sc := range right[i].SourceColumns {
				sourceColumns[sc] = true
			}
		}
		merged = append(merged, base.QuerySpanResult{
			Name:          name,
			SourceColumns: sourceColumns,
			IsPlainField:  false,
		})
	}
	return merged
}

// mergeExpandedByName merges two expanded set-operation arms by output column
// NAME (case-insensitive — BigQuery column names are), mirroring the resolver's
// mergeProjectionsByName for arms that needed metadata expansion first. With
// matchColumns (BY NAME ON (cols)) the output is EXACTLY those columns in list
// order; otherwise the left arm's names in order plus right-only names appended
// (over-inclusion is masking-safe: a trailing extra never shifts earlier
// positions). Each output column's lineage is the union of BOTH arms'
// same-named columns.
func mergeExpandedByName(left, right []base.QuerySpanResult, matchColumns []string) []base.QuerySpanResult {
	namedSources := func(arm []base.QuerySpanResult, name string) base.SourceColumnSet {
		out := base.SourceColumnSet{}
		for _, r := range arm {
			if strings.EqualFold(r.Name, name) {
				for sc := range r.SourceColumns {
					out[sc] = true
				}
			}
		}
		return out
	}
	mergedFor := func(name string) base.QuerySpanResult {
		sources := namedSources(left, name)
		for sc := range namedSources(right, name) {
			sources[sc] = true
		}
		return base.QuerySpanResult{Name: name, SourceColumns: sources, IsPlainField: false}
	}
	if len(matchColumns) > 0 {
		out := make([]base.QuerySpanResult, 0, len(matchColumns))
		for _, name := range matchColumns {
			out = append(out, mergedFor(name))
		}
		return out
	}
	out := make([]base.QuerySpanResult, 0, len(left)+len(right))
	for _, l := range left {
		out = append(out, mergedFor(l.Name))
	}
	for _, r := range right {
		if !containsResultName(left, r.Name) {
			out = append(out, mergedFor(r.Name))
		}
	}
	return out
}

// containsResultName reports whether any result in list has the given output
// name under case-insensitive comparison.
func containsResultName(list []base.QuerySpanResult, name string) bool {
	for _, r := range list {
		if strings.EqualFold(r.Name, name) {
			return true
		}
	}
	return false
}

// expandStarSegments expands one star item's resolved segments (in order) into
// per-column results, then applies the star's EXCEPT/REPLACE modifiers. A base
// segment enumerates its physical table's columns via metadata (each a plain
// field per the segment's Plain); a concrete segment is emitted directly with its
// resolved lineage. EXCEPT/REPLACE then act on the expanded column list keyed by
// output name with last-wins on a name collision — the legacy starModify
// semantics (two FROM relations contributing a like-named column keep the later
// one before EXCEPT removes a name).
func (q *querySpanExtractor) expandStarSegments(r analysis.ColumnInfo, fullSourceColumns base.SourceColumnSet, aliasMap map[string][]string) []base.QuerySpanResult {
	var expanded []base.QuerySpanResult
	for _, seg := range r.StarSegments {
		if seg.BaseTable != nil {
			expanded = append(expanded, q.expandBaseTableColumns(*seg.BaseTable, seg.Plain, seg.ExceptColumns)...)
			continue
		}
		// A concrete segment (a CTE/derived/explicit column resolved by omni).
		sourceColumns := base.SourceColumnSet{}
		for _, ref := range seg.Sources {
			addMatchingColumns(sourceColumns, fullSourceColumns, ref.Column, ref.Table, aliasMap)
		}
		// Spanner legacy renders a base-table FIELD passthrough (the JOIN ...
		// USING coalesced key) in the field's METADATA case — the legacy resolver
		// named it after the left table's field, not the written USING token.
		name := seg.Name
		if seg.BaseFieldName && len(seg.Sources) > 0 {
			if canon := q.canonicalColumnCase(seg.Sources[0], name); canon != "" {
				name = canon
			}
		}
		expanded = append(expanded, base.QuerySpanResult{
			Name:          name,
			SourceColumns: sourceColumns,
			IsPlainField:  seg.Plain,
		})
	}
	return q.applyStarModifiers(expanded, r, fullSourceColumns, aliasMap)
}

// expandBaseTableColumns enumerates a physical base table's columns from metadata
// into per-column plain-field results, each lineage'd to that column. A table
// whose metadata cannot be resolved yields no columns (the access is still
// recorded at the table level by expandTablesToColumns / toTableResources, and a
// ResourceNotFoundError is surfaced there). except lists column names to SKIP
// (case-insensitive): a JOIN ... USING key is projected once as a coalesced
// concrete segment ahead of the side stars, so each side's star must not expand
// it again — re-expanding would shift every later position and misalign the
// positional masker.
func (q *querySpanExtractor) expandBaseTableColumns(rel analysis.ColumnRef, plain bool, except []string) []base.QuerySpanResult {
	canonicalTable, columns, err := q.tableColumns(rel.Schema, rel.Table)
	if err != nil || len(columns) == 0 {
		return nil
	}
	out := make([]base.QuerySpanResult, 0, len(columns))
	for _, col := range columns {
		if containsFold(except, col) {
			continue
		}
		res := base.ColumnResource{Database: q.defaultDatabase, Schema: rel.Schema, Table: canonicalTable, Column: col}
		out = append(out, base.QuerySpanResult{
			Name:          col,
			SourceColumns: base.SourceColumnSet{res: true},
			IsPlainField:  plain,
		})
	}
	return out
}

// canonicalColumnCase returns the metadata-canonical spelling of column in the
// table named by ref (case-folded match), or "" when the table or column cannot
// be resolved.
func (q *querySpanExtractor) canonicalColumnCase(ref analysis.ColumnRef, column string) string {
	_, columns, err := q.tableColumns(ref.Schema, ref.Table)
	if err != nil {
		return ""
	}
	for _, col := range columns {
		if strings.EqualFold(col, column) {
			return col
		}
	}
	return ""
}

// applyStarModifiers applies a star item's EXCEPT/REPLACE modifiers to its
// expanded column list, reproducing the legacy starModify: results are keyed by
// output name (case-insensitive — BigQuery column names are), last-wins on a
// collision; an EXCEPT name drops its column entirely; a REPLACE (expr AS name)
// re-points the named column's lineage to the replacement expression's sources
// and clears its plain-field flag. Original order is preserved. When the star has
// no modifiers the expanded list is returned unchanged.
func (q *querySpanExtractor) applyStarModifiers(expanded []base.QuerySpanResult, r analysis.ColumnInfo, fullSourceColumns base.SourceColumnSet, aliasMap map[string][]string) []base.QuerySpanResult {
	if len(r.StarExcept) == 0 && len(r.StarReplace) == 0 {
		return expanded
	}
	type item struct {
		id    int
		field base.QuerySpanResult
	}
	// Key by output name; a name collision keeps the LAST occurrence (legacy
	// fieldItemMap overwrite), but the surviving entry keeps its FIRST ordinal so
	// the output order is stable.
	order := map[string]int{}
	byName := map[string]item{}
	next := 0
	for _, f := range expanded {
		key := strings.ToLower(f.Name)
		if _, ok := order[key]; !ok {
			order[key] = next
			next++
		}
		byName[key] = item{id: order[key], field: f}
	}
	for _, name := range r.StarExcept {
		delete(byName, strings.ToLower(name))
	}
	for _, rep := range r.StarReplace {
		key := strings.ToLower(rep.Name)
		if _, ok := byName[key]; !ok {
			continue
		}
		set := base.SourceColumnSet{}
		for _, ref := range rep.Sources {
			addMatchingColumns(set, fullSourceColumns, ref.Column, ref.Table, aliasMap)
		}
		byName[key] = item{
			id:    byName[key].id,
			field: base.QuerySpanResult{Name: rep.Name, SourceColumns: set, IsPlainField: false},
		}
	}
	items := make([]item, 0, len(byName))
	for _, it := range byName {
		items = append(items, it)
	}
	sort.Slice(items, func(i, j int) bool { return items[i].id < items[j].id })
	out := make([]base.QuerySpanResult, 0, len(items))
	for _, it := range items {
		out = append(out, it.field)
	}
	return out
}

// buildStarMergeResult builds a set-operation merge position whose other arm is a
// base-table star: the concrete arm's lineage (r.SourceColumns) is unioned with
// the star table's Index-th column (expanded from metadata). When the star arm is
// the LEFT one (StarMerge.LeftStar) the output name comes from that column (the
// legacy first-select-name rule); otherwise the concrete arm's name is kept. The
// merged column is never a plain field.
func (q *querySpanExtractor) buildStarMergeResult(r analysis.ColumnInfo, fullSourceColumns base.SourceColumnSet, aliasMap map[string][]string) base.QuerySpanResult {
	sourceColumns := base.SourceColumnSet{}
	for _, ref := range r.SourceColumns {
		addMatchingColumns(sourceColumns, fullSourceColumns, ref.Column, ref.Table, aliasMap)
	}
	// Spanner legacy naming: a concrete-arm output name is uppercased (the
	// legacy expression-name rendering), but a LEFT-star-derived name keeps the
	// star column's metadata case verbatim (the legacy first-arm rule passed the
	// PhysicalTable field name through unchanged — unlike the BigQuery legacy
	// extractor, which uppercased both).
	name := strings.ToUpper(r.Name)
	// A StarMerge always carries a bare single-base-table star (a USING-coalesced
	// or otherwise multi-segment star arm defers to SetOpMerge instead), so there
	// are no except columns to apply here.
	starCols := q.expandBaseTableColumns(r.StarMerge.Table, false, nil)
	if idx := r.StarMerge.Index; idx >= 0 && idx < len(starCols) {
		for sc := range starCols[idx].SourceColumns {
			sourceColumns[sc] = true
		}
		if r.StarMerge.LeftStar {
			name = starCols[idx].Name
		}
	}
	return base.QuerySpanResult{
		Name:          name,
		SourceColumns: sourceColumns,
		IsPlainField:  false,
	}
}

// resultName renders an explicit select-item's output column name to match the
// legacy extractor (extractTableSourceFromSelect): the name is UPPER-CASED, and
// for an expression that has no written name of its own omni leaves Name empty —
// in that case the legacy code used the name of the single bare column the
// expression's DFS surfaced (e.g. `ID+1` → "ID", `foo(bar(ID), NAME)` → "ID"),
// which is omni's first source column. A name-less expression with no column
// reference (e.g. a literal, or an opaque scalar subquery) stays "".
func resultName(r analysis.ColumnInfo) string {
	name := r.Name
	if name == "" && len(r.SourceColumns) > 0 {
		name = r.SourceColumns[0].Column
	}
	return strings.ToUpper(name)
}

// addMatchingColumns adds every column in fullSourceColumns whose name matches
// refColumn (case-insensitively — BigQuery column names are case-insensitive —
// and whose table matches refTable when refTable is non-empty) to dst. refTable
// may be a relation alias; see tableMatches.
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

// buildAliasMap maps each relation alias (lower-cased) to the metadata-canonical
// physical table names it stands for, so aliased column references resolve back to
// the expanded (metadata-cased) source columns during column matching. Because
// omni's AccessTables is flat (carries no SQL scope), an alias reused in different
// scopes maps to several tables; all are kept so matching can over-include rather
// than under-match. An alias equal to its own table name carries no information
// and is skipped.
func (q *querySpanExtractor) buildAliasMap(span *analysis.QuerySpan) map[string][]string {
	if span == nil {
		return nil
	}
	out := make(map[string][]string)
	for _, t := range span.AccessTables {
		if t.Alias == "" || strings.EqualFold(t.Alias, t.Table) {
			continue
		}
		table := t.Table
		if !t.IsSystem {
			if canonical, _, err := q.tableColumns(t.Schema, t.Table); err == nil && canonical != "" {
				table = canonical
			}
		}
		key := strings.ToLower(t.Alias)
		if !containsFold(out[key], table) {
			out[key] = append(out[key], table)
		}
	}
	return out
}

// containsFold reports whether list contains s under case-insensitive comparison.
func containsFold(list []string, s string) bool {
	for _, e := range list {
		if strings.EqualFold(e, s) {
			return true
		}
	}
	return false
}

// tableColumns returns the metadata-canonical table name and the column names of
// the given table or view in the given schema of the default database. Schema
// names match CASE-SENSITIVELY (the legacy findTableSchema looked the schema up
// by its written name; "" is the default schema) while table names fall back to
// a case-folded match — both exactly the legacy behavior.
func (q *querySpanExtractor) tableColumns(schema, table string) (string, []string, error) {
	metadata, err := q.getDatabaseMetadata(q.defaultDatabase)
	if err != nil {
		return "", nil, err
	}

	schemaMeta := metadata.GetSchemaMetadata(schema)
	if schemaMeta == nil {
		return "", nil, &base.ResourceNotFoundError{
			Database: &q.defaultDatabase,
			Schema:   &schema,
		}
	}

	// Table first.
	tableMeta := schemaMeta.GetTable(table)
	if tableMeta == nil {
		for _, name := range schemaMeta.ListTableNames() {
			if strings.EqualFold(name, table) {
				tableMeta = schemaMeta.GetTable(name)
				break
			}
		}
	}
	if tableMeta != nil {
		canonical := tableMeta.GetProto().GetName()
		var cols []string
		for _, col := range tableMeta.GetProto().GetColumns() {
			cols = append(cols, col.Name)
		}
		return canonical, cols, nil
	}

	// Then view.
	viewMeta := schemaMeta.GetView(table)
	if viewMeta == nil {
		for _, name := range schemaMeta.ListViewNames() {
			if strings.EqualFold(name, table) {
				viewMeta = schemaMeta.GetView(name)
				break
			}
		}
	}
	if viewMeta != nil {
		canonical := viewMeta.GetName()
		var cols []string
		for _, col := range viewMeta.GetColumns() {
			cols = append(cols, col.Name)
		}
		return canonical, cols, nil
	}

	return "", nil, &base.ResourceNotFoundError{
		Database: &q.defaultDatabase,
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
		return nil, errors.Wrapf(err, "failed to get database metadata for dataset: %s", database)
	}
	if meta == nil {
		empty := &model.DatabaseMetadata{}
		q.metaCache[database] = empty
		return empty, nil
	}
	q.metaCache[database] = meta
	return meta, nil
}
