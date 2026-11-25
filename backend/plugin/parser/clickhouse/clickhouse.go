// Package clickhouse registers ClickHouse's parser functions, each backed by
// omni's ClickHouse packages.
package clickhouse

import (
	"context"

	chanalysis "github.com/bytebase/omni/clickhouse/analysis"
	chcompletion "github.com/bytebase/omni/clickhouse/completion"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.Register(base.Engine{
		Split:         splitSQL,
		Diagnose:      diagnose,
		Complete:      completion,
		ValidateQuery: chanalysis.ValidateQuery,
		Explain:       explainStatement,
		ResultLimit:   statementWithResultLimit,
	}, storepb.Engine_CLICKHOUSE)
}

func splitSQL(statement string) ([]base.Statement, error) {
	stmts, err := chanalysis.Split(statement)
	if err != nil {
		return nil, err
	}
	return base.OmniStatements(statement, stmts), nil
}

func diagnose(_ context.Context, _ base.DiagnoseContext, statement string) ([]base.Diagnostic, error) {
	var syntaxErr *base.SyntaxError
	if err := chanalysis.Diagnose(statement); err != nil {
		syntaxErr = base.SyntaxErrorAtOffset(statement, err.Offset, err.Message)
	}
	return base.DiagnoseSyntaxError(statement, syntaxErr), nil
}

func completion(ctx context.Context, cCtx base.CompletionContext, statement string, caretLine int, caretOffset int) ([]base.Candidate, error) {
	loadDB, listNames := base.CompletionMetadataLoaders(cCtx)
	candidates := chcompletion.CompleteSession(ctx, chcompletion.SessionConfig{
		DefaultDatabase:   cCtx.DefaultDatabase,
		LoadDatabase:      loadDB,
		ListDatabaseNames: listNames,
	}, statement, caretLine, caretOffset)
	return base.ConvertOmniCandidates(candidates, func(c chcompletion.Candidate) (string, chcompletion.CandidateType, string, string) {
		return c.Text, c.Type, c.Definition, c.Comment
	}, base.CompareCandidateByTypeText), nil
}

func explainStatement(statement string, format base.ExplainFormat) (string, error) {
	name := ""
	if format == base.ExplainFormatJSON {
		name = "JSON"
	}
	return chanalysis.ExplainStatement(statement, name), nil
}

// statementWithResultLimit implements base.ResultLimitFunc for ClickHouse,
// which takes no engineVersion.
func statementWithResultLimit(statement string, limit int, _ string) string {
	return chanalysis.WrapResultLimit(statement, limit)
}
