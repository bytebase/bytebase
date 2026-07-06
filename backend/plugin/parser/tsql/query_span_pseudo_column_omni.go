package tsql

import (
	"strings"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"

	"github.com/bytebase/omni/mssql/ast"
)

// pseudoColumnKind classifies T-SQL pseudo-column references the omni parser
// accepts as ColumnRef nodes.
type pseudoColumnKind int

const (
	pseudoColumnNone pseudoColumnKind = iota
	// pseudoColumnIdentity is $IDENTITY / IDENTITYCOL: the engine resolves it
	// to the table's single IDENTITY-property column at bind time.
	pseudoColumnIdentity
	// pseudoColumnGraph is $node_id / $edge_id / $from_id / $to_id on graph
	// tables. The backing engine columns are auto-generated and not present in
	// the synced catalog.
	pseudoColumnGraph
)

func classifyPseudoColumn(name string) pseudoColumnKind {
	switch strings.ToLower(name) {
	case "$identity", "identitycol":
		return pseudoColumnIdentity
	case "$node_id", "$edge_id", "$from_id", "$to_id":
		return pseudoColumnGraph
	default:
		// $ROWGUID / ROWGUIDCOL deliberately unhandled: the synced catalog has
		// no rowguid flag yet, so they keep today's fail-closed resolution
		// error until metadata support lands.
		return pseudoColumnNone
	}
}

// resolvePseudoColumnRef maps a pseudo-column reference to real column
// lineage. handled is false when v is not a (supported) pseudo-column and the
// ordinary resolution path should run.
//
//   - $IDENTITY / IDENTITYCOL map precisely to the in-scope table's column
//     with ColumnMetadata.IsIdentity, so masking rules on the identity column
//     apply to the pseudo-column reference too.
//   - Graph pseudo-columns get table-level lineage (the union of the table's
//     columns): $from_id/$to_id encode the identity of referenced node rows,
//     so empty lineage would fail open in scenarios where relationships
//     themselves are sensitive; over-masking is the safer direction.
//
// Both kinds require the reference to bind to exactly one candidate table in
// scope — the engine reports ambiguity errors itself, so mirroring that is
// fail-closed.
func (q *omniQuerySpanExtractor) resolvePseudoColumnRef(v *ast.ColumnRef) (base.QuerySpanResult, bool, error) {
	kind := classifyPseudoColumn(v.Column)
	if kind == pseudoColumnNone {
		return base.QuerySpanResult{}, false, nil
	}
	// Delimited identifiers are never pseudo-columns in T-SQL: [IDENTITYCOL]
	// and [$node_id] reference real columns with those names. omni strips the
	// delimiters from ColumnRef.Column, so check the source text — a
	// delimited column segment makes the ref end with ']' or '"', which a
	// bare pseudo-column or keyword never does.
	if q.columnSegmentIsDelimited(v) {
		return base.QuerySpanResult{}, false, nil
	}

	sources := q.pseudoColumnCandidateSources(v)
	switch kind {
	case pseudoColumnIdentity:
		r, err := q.resolveIdentityPseudoColumn(v, sources)
		return r, true, err
	case pseudoColumnGraph:
		r, err := q.resolveGraphPseudoColumn(v, sources)
		return r, true, err
	default:
		return base.QuerySpanResult{}, false, nil
	}
}

// columnSegmentIsDelimited reports whether the reference's column segment is
// bracket- or quote-delimited in the original source.
func (q *omniQuerySpanExtractor) columnSegmentIsDelimited(v *ast.ColumnRef) bool {
	end := v.Loc.End
	if end <= 0 || end > len(q.source) {
		return false
	}
	last := q.source[end-1]
	return last == ']' || last == '"'
}

// pseudoColumnCandidateSources returns the in-scope table sources the
// reference may bind to, filtered by the optional table qualifier
// (t.$IDENTITY binds within t only). Inner scope is searched before outer,
// matching tsqlIsFieldSensitive.
func (q *omniQuerySpanExtractor) pseudoColumnCandidateSources(v *ast.ColumnRef) []base.TableSource {
	matches := func(ts base.TableSource) bool {
		if v.Table != "" && !q.isIdentifierEqual(v.Table, ts.GetTableName()) {
			return false
		}
		if v.Schema != "" && !q.isIdentifierEqual(v.Schema, ts.GetSchemaName()) {
			return false
		}
		if v.Database != "" && !q.isIdentifierEqual(v.Database, ts.GetDatabaseName()) {
			return false
		}
		return true
	}
	var result []base.TableSource
	for _, ts := range q.tableSourcesFrom {
		if matches(ts) {
			result = append(result, ts)
		}
	}
	if len(result) > 0 {
		return result
	}
	for i := len(q.outerTableSources) - 1; i >= 0; i-- {
		if matches(q.outerTableSources[i]) {
			result = append(result, q.outerTableSources[i])
		}
	}
	return result
}

func (q *omniQuerySpanExtractor) resolveIdentityPseudoColumn(v *ast.ColumnRef, sources []base.TableSource) (base.QuerySpanResult, error) {
	type match struct {
		column base.QuerySpanResult
	}
	var matches []match
	for _, ts := range sources {
		for _, column := range ts.GetQuerySpanResult() {
			if q.isIdentityColumnResult(column) {
				matches = append(matches, match{column: column})
				break // a table has at most one identity column
			}
		}
	}
	switch len(matches) {
	case 0:
		return base.QuerySpanResult{}, errors.Errorf("no identity column found in scope for %q", v.Column)
	case 1:
		r := matches[0].column
		r.IsPlainField = true
		return r, nil
	default:
		return base.QuerySpanResult{}, errors.Errorf("ambiguous %q: multiple tables in scope have an identity column", v.Column)
	}
}

// isIdentityColumnResult reports whether a table-source column is backed by
// exactly one physical column whose metadata has IsIdentity. The lookup goes
// through the column's source resource rather than the table source itself so
// aliased tables (wrapped as PseudoTable) resolve too.
func (q *omniQuerySpanExtractor) isIdentityColumnResult(column base.QuerySpanResult) bool {
	if !column.IsPlainField || len(column.SourceColumns) != 1 {
		return false
	}
	for resource := range column.SourceColumns {
		if resource.Database == "" || resource.Table == "" || resource.Column == "" {
			return false
		}
		_, databaseMeta, err := q.gCtx.GetDatabaseMetadataFunc(q.ctx, q.gCtx.InstanceID, resource.Database)
		if err != nil || databaseMeta == nil {
			return false
		}
		schemaMeta := databaseMeta.GetSchemaMetadata(resource.Schema)
		if schemaMeta == nil {
			return false
		}
		tableMeta := schemaMeta.GetTable(resource.Table)
		if tableMeta == nil {
			return false
		}
		for _, columnMeta := range tableMeta.GetProto().GetColumns() {
			if q.isIdentifierEqual(columnMeta.Name, resource.Column) {
				return columnMeta.IsIdentity
			}
		}
	}
	return false
}

func (*omniQuerySpanExtractor) resolveGraphPseudoColumn(v *ast.ColumnRef, sources []base.TableSource) (base.QuerySpanResult, error) {
	if len(sources) == 0 {
		return base.QuerySpanResult{}, errors.Errorf("no table in scope for graph pseudo-column %q", v.Column)
	}
	if len(sources) > 1 && v.Table == "" {
		return base.QuerySpanResult{}, errors.Errorf("ambiguous graph pseudo-column %q: qualify it with the table name", v.Column)
	}
	sourceColumns := make(base.SourceColumnSet)
	for _, column := range sources[0].GetQuerySpanResult() {
		sourceColumns, _ = base.MergeSourceColumnSet(sourceColumns, column.SourceColumns)
	}
	// Attribute-less graph tables (CREATE TABLE ... AS EDGE with no
	// user-defined columns) contribute no catalog columns, so the union is
	// empty — and empty SourceColumns downstream means NoneMasker, the exact
	// fail-open this table-level lineage is meant to prevent. Fail closed: a
	// table-only resource would not match column-level masking rules either.
	if len(sourceColumns) == 0 {
		return base.QuerySpanResult{}, errors.Errorf("cannot derive lineage for graph pseudo-column %q: table has no columns in the catalog", v.Column)
	}
	return base.QuerySpanResult{
		Name:          v.Column,
		SourceColumns: sourceColumns,
		IsPlainField:  false,
	}, nil
}
