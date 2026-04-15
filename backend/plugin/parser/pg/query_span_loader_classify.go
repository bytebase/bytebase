package pg

import (
	"github.com/pkg/errors"

	"github.com/bytebase/omni/pg/catalog"
)

// fallbackReason classifies why analyzer analysis fell through to
// extractFallbackColumns / fallbackExtractLineage. It is a pure classification
// of the error from omni's AnalyzeSelectStmt, used by tests to assert which
// branch a given query takes.
//
// This is NOT a runtime gate, a metric, or a telemetry signal. It exists so
// that unit tests can distinguish "pseudo-induced degradation" from
// "undefined reference" from "analyzer does not support this feature", and so
// that debugging from logs carries a meaningful category.
type fallbackReason int

const (
	reasonNone fallbackReason = iota
	// reasonExpectedPseudoSemantic: analyzer failed because a pseudo-installed
	// object looks like text in place of its original user type. Common forms:
	// UndefinedFunction (overload signature no longer matches), DatatypeMismatch
	// (operator resolution against text-backed pseudo), AmbiguousFunction /
	// AmbiguousColumn (candidate pile from pseudo-widening).
	reasonExpectedPseudoSemantic
	// reasonUndefinedReference: analyzer could not find a relation, column,
	// schema, or object. This can be a loader bug (we should have installed
	// it) OR a legitimate user error (query typos, stale metadata). Without a
	// structured ErrorIdent on *catalog.Error we cannot programmatically
	// distinguish the two; humans triage from logs.
	reasonUndefinedReference
	// reasonAnalyzerUnsupported: non-*catalog.Error internal analyzer failure
	// (returned as fmt.Errorf from analyze.go code paths that hit an
	// unsupported PG feature or a defensive check).
	reasonAnalyzerUnsupported
)

// classifyAnalyzeError buckets an omni AnalyzeSelectStmt error into one of
// the fallbackReason categories. Non-nil errors always return a non-None
// reason; a nil error returns reasonNone.
func classifyAnalyzeError(err error) fallbackReason {
	if err == nil {
		return reasonNone
	}
	var cErr *catalog.Error
	if !errors.As(err, &cErr) {
		return reasonAnalyzerUnsupported
	}
	switch cErr.Code {
	case catalog.CodeUndefinedFunction,
		catalog.CodeAmbiguousFunction,
		catalog.CodeDatatypeMismatch,
		catalog.CodeFeatureNotSupported,
		catalog.CodeAmbiguousColumn:
		return reasonExpectedPseudoSemantic
	case catalog.CodeUndefinedTable,
		catalog.CodeUndefinedColumn,
		catalog.CodeUndefinedObject,
		catalog.CodeUndefinedSchema:
		return reasonUndefinedReference
	default:
		return reasonAnalyzerUnsupported
	}
}

// String returns a short lowercase label suitable for test assertions and
// log lines.
func (r fallbackReason) String() string {
	switch r {
	case reasonNone:
		return "none"
	case reasonExpectedPseudoSemantic:
		return "expected_pseudo_semantic"
	case reasonUndefinedReference:
		return "undefined_reference"
	case reasonAnalyzerUnsupported:
		return "analyzer_unsupported"
	default:
		return "unknown"
	}
}
