// Package spanner registers the Spanner engine handlers, backed by the shared
// omni-based GoogleSQL implementation (backend/plugin/parser/googlesql) with the
// Spanner dialect configuration. See that package for the behavior contract and
// the masking notes; the differential corpus (test-data/query-span/standard.yaml,
// recorded from the legacy ANTLR resolver) and the leak-pin tests in this package
// pin the Spanner-specific behavior.
package spanner

import (
	"context"

	"github.com/bytebase/omni/googlesql/analysis"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
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

func init() {
	base.RegisterSplitterFunc(storepb.Engine_SPANNER, SplitSQL)
	base.RegisterDiagnoseFunc(storepb.Engine_SPANNER, Diagnose)
	base.RegisterGetQuerySpan(storepb.Engine_SPANNER, GetQuerySpan)
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
