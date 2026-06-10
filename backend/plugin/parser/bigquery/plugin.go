// Package bigquery registers the BigQuery engine handlers, backed by the shared
// omni-based GoogleSQL implementation (backend/plugin/parser/googlesql) with the
// BigQuery dialect configuration. See that package for the behavior contract and
// the masking notes; the differential corpus (test-data/query-span/standard.yaml,
// recorded from the legacy ANTLR resolver) and the leak-pin tests in this package
// pin the BigQuery-specific behavior.
package bigquery

import (
	"context"

	"github.com/bytebase/omni/googlesql/analysis"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
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

func init() {
	base.RegisterSplitterFunc(storepb.Engine_BIGQUERY, SplitSQL)
	base.RegisterDiagnoseFunc(storepb.Engine_BIGQUERY, Diagnose)
	base.RegisterGetQuerySpan(storepb.Engine_BIGQUERY, GetQuerySpan)
}

// SplitSQL splits the input into multiple SQL statements.
func SplitSQL(statement string) ([]base.Statement, error) {
	return googlesql.SplitSQL(statement, config)
}

// Diagnose returns syntax diagnostics for the given statement.
func Diagnose(_ context.Context, _ base.DiagnoseContext, statement string) ([]base.Diagnostic, error) {
	return googlesql.Diagnose(statement), nil
}

// GetQuerySpan returns the query span for the given statement.
func GetQuerySpan(
	ctx context.Context,
	gCtx base.GetQuerySpanContext,
	stmt base.Statement,
	database, _ string,
	_ bool,
) (*base.QuerySpan, error) {
	return googlesql.NewQuerySpanExtractor(config, database, gCtx).GetQuerySpan(ctx, stmt.Text)
}
