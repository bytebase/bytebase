// Package googlesql holds the omni-backed implementation shared by the two
// GoogleSQL engine plugins (bigquery and spanner): the query-span extractor,
// the splitter, the diagnostics adapter, and the query-type mapping. One
// omni grammar serves both dialects; the per-engine packages register the
// base.* handlers with a Config carrying the few dialect deltas.
//
// Masking note: the query-span extractor is masking-critical (bytebase's
// masker applies per-output-column maskers positionally and fails OPEN, so
// under-attributed lineage returns sensitive columns unmasked). Keeping ONE
// implementation here — rather than a copy per engine — is deliberate: the
// behavior was validated against legacy-resolver differential corpora plus
// leak-pin tests in both engine packages, and a single code path cannot drift
// between engines.
package googlesql

import (
	"github.com/bytebase/omni/googlesql/analysis"
)

// Config carries the dialect deltas between the BigQuery and Spanner plugins.
// Every flag mirrors a documented legacy-extractor behavior difference; see the
// per-engine packages for the values and the corpus/leak-pin tests pinning them.
type Config struct {
	// Dialect selects omni's per-dialect analysis behavior (system schemas,
	// name-part bucketing, default join type).
	Dialect analysis.Dialect

	// IgnoreWrittenDatabase pins every resource's Database to the default
	// database. Spanner: the legacy resolver silently ignored the db part of a
	// 3-part db.schema.table and recorded the session database. BigQuery keeps
	// the written dataset (Database) with default fill-in.
	IgnoreWrittenDatabase bool

	// SchemaScopedMetadata looks tables up under the metadata schema named by
	// the reference (case-SENSITIVE, "" = default schema) within the default
	// database — the Spanner named-schema model. BigQuery instead resolves the
	// written dataset as the DATABASE and uses its single unnamed schema.
	SchemaScopedMetadata bool

	// SystemOnlyEmptySpan early-returns an EMPTY SelectInfoSchema span (no
	// source columns, no results, no metadata access) for a query reading ONLY
	// system tables — the legacy spanner extractor's exact behavior. BigQuery
	// instead keeps the system tables as table-level resources.
	SystemOnlyEmptySpan bool

	// SetStatementIsSelect classifies a SET statement as base.Select — BOTH
	// legacy queryTypeListeners' "treat SAFE SET as select" case (omni
	// classifies SET as Unknown, which the new-ACL access check would reject as
	// a disallowed query type).
	SetStatementIsSelect bool

	// UppercaseStarMergeName upper-cases a set-operation star-merge output name
	// (the legacy BigQuery extractor's rendering). The legacy spanner extractor
	// passed a LEFT-star-derived name through in the star column's metadata
	// case instead.
	UppercaseStarMergeName bool

	// CanonicalBaseFieldName renders a base-table FIELD passthrough (the JOIN
	// ... USING coalesced key, marked BaseFieldName by omni) in the field's
	// METADATA case — the legacy spanner resolver named it after the left
	// PhysicalTable's field, not the written USING token.
	CanonicalBaseFieldName bool

	// ContiguousSplitText makes each split statement's Text run contiguously
	// from the END of the previous statement (inter-statement whitespace and
	// comments attach to the FOLLOWING statement) — the legacy spanner
	// parse-tree splitter's convention. BigQuery starts each statement's Text
	// at its own first token.
	ContiguousSplitText bool
}
