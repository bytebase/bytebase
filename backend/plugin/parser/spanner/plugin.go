// Package spanner registers the Spanner engine handlers, backed by the shared
// omni-based GoogleSQL implementation (backend/plugin/parser/googlesql) with the
// Spanner dialect configuration. See that package for the behavior contract and
// the masking notes; the differential corpus (test-data/query-span/standard.yaml,
// recorded from the legacy ANTLR resolver) and the leak-pin tests in this package
// pin the Spanner-specific behavior.
package spanner

import (
	"github.com/bytebase/omni/googlesql/analysis"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/googlesql"
)

// config carries the Spanner dialect deltas, each mirroring a documented legacy
// extractor behavior: named schemas under ONE database (the db part of a 3-part
// db.schema.table is ignored; schema names match case-sensitively), system-only
// queries (INFORMATION_SCHEMA / SPANNER_SYS) early-return an empty
// SelectInfoSchema span, SET classifies as Select, JOIN USING coalesced keys
// keep their metadata case, and split Text runs contiguously from the previous
// statement's end.
var config = googlesql.Config{
	Dialect:                analysis.DialectSpanner,
	IgnoreWrittenDatabase:  true,
	SchemaScopedMetadata:   true,
	SystemOnlyEmptySpan:    true,
	SetStatementIsSelect:   true,
	CanonicalBaseFieldName: true,
	ContiguousSplitText:    true,
}

var handlers = googlesql.Register(storepb.Engine_SPANNER, config)

// SplitSQL is the registered splitter, re-exported for the tests in this package.
var SplitSQL = handlers.SplitSQL

// GetQuerySpan is the registered query-span handler, re-exported for the tests
// in this package.
var GetQuerySpan = handlers.GetQuerySpan
