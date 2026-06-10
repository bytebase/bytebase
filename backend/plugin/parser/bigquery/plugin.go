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
// keep their table-level resources, and set-operation star-merge names are
// upper-cased (the legacy BigQuery extractor's rendering).
var config = googlesql.Config{
	Dialect:                analysis.DialectBigQuery,
	UppercaseStarMergeName: true,
}

// SplitSQL and GetQuerySpan are the registered handlers, re-exported for the
// tests in this package.
var SplitSQL, GetQuerySpan = googlesql.Register(storepb.Engine_BIGQUERY, config)
