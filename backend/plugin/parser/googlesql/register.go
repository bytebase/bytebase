package googlesql

import (
	"context"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

// Register registers the splitter, diagnose, and query-span handlers for one
// GoogleSQL engine with the given dialect configuration, and returns the split
// and query-span functions so the engine package can re-export them for its
// tests (the differential corpus and the leak pins drive the exact functions
// production uses).
func Register(engine storepb.Engine, cfg Config) (base.SplitMultiSQLFunc, base.GetQuerySpanFunc) {
	split := func(statement string) ([]base.Statement, error) {
		return SplitSQL(statement, cfg)
	}
	querySpan := func(ctx context.Context, gCtx base.GetQuerySpanContext, stmt base.Statement, database, _ string, _ bool) (*base.QuerySpan, error) {
		return NewQuerySpanExtractor(cfg, database, gCtx).GetQuerySpan(ctx, stmt.Text)
	}
	base.RegisterSplitterFunc(engine, split)
	base.RegisterDiagnoseFunc(engine, func(_ context.Context, _ base.DiagnoseContext, statement string) ([]base.Diagnostic, error) {
		return Diagnose(statement), nil
	})
	base.RegisterGetQuerySpan(engine, querySpan)
	return split, querySpan
}
