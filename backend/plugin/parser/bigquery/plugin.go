// Package bigquery registers the BigQuery engine handlers, backed by the shared
// omni-based GoogleSQL implementation (backend/plugin/parser/googlesql) with the
// BigQuery dialect configuration. See that package for the behavior contract and
// the masking notes; the differential corpus (test-data/query-span/standard.yaml,
// recorded from the legacy ANTLR resolver) and the leak-pin tests in this package
// pin the BigQuery-specific behavior.
package bigquery

import (
	"github.com/bytebase/omni/googlesql/analysis"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/googlesql"
)

// config carries the BigQuery dialect deltas: the written dataset is the
// DATABASE (default fill-in, single unnamed metadata schema), system queries
// keep their table-level resources, SET classifies as Select (the legacy
// listener's "treat SAFE SET as select"), and set-operation star-merge names
// are upper-cased (the legacy BigQuery extractor's rendering).
var config = googlesql.Config{
	Dialect:                analysis.DialectBigQuery,
	SetStatementIsSelect:   true,
	UppercaseStarMergeName: true,
}

var handlers = googlesql.Register(storepb.Engine_BIGQUERY, config)

// SplitSQL is the registered splitter, re-exported for the tests in this package.
var SplitSQL = handlers.SplitSQL

// GetQuerySpan is the registered query-span handler, re-exported for the tests
// in this package.
var GetQuerySpan = handlers.GetQuerySpan
