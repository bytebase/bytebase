package googlesql

import (
	"context"

	"github.com/bytebase/omni/googlesql/analysis"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

// Handlers bundles the registered engine handlers Register returns, so the
// engine package can re-export each with its own declaration for its tests
// (the differential corpus and the leak pins drive the exact functions
// production uses).
type Handlers struct {
	SplitSQL     base.SplitMultiSQLFunc
	GetQuerySpan base.GetQuerySpanFunc
}

// Register registers the splitter, query-validator, diagnose, query-span,
// statement-parser, and statement-type handlers for one GoogleSQL engine with
// the given dialect configuration.
func Register(engine storepb.Engine, cfg Config) Handlers {
	h := Handlers{
		SplitSQL: func(statement string) ([]base.Statement, error) {
			return SplitSQL(statement, cfg)
		},
		GetQuerySpan: func(ctx context.Context, gCtx base.GetQuerySpanContext, stmt base.Statement, database, _ string, _ bool) (*base.QuerySpan, error) {
			return getQuerySpan(ctx, cfg, database, gCtx, stmt.Text)
		},
	}
	base.Register(base.Engine{
		Split: h.SplitSQL,
		Parse: func(statement string) ([]base.ParsedStatement, error) {
			return ParseStatements(statement, cfg)
		},
		StatementTypes: GetStatementTypes,
		Diagnose: func(_ context.Context, _ base.DiagnoseContext, statement string) ([]base.Diagnostic, error) {
			return Diagnose(statement), nil
		},
		ValidateQuery: func(statement string) (bool, bool, error) {
			return analysis.ValidateQuery(statement, cfg.Dialect)
		},
		QuerySpan: h.GetQuerySpan,
	}, engine)
	return h
}
