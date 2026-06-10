package spanner

import (
	"context"
	"strings"

	"github.com/bytebase/omni/googlesql/diagnostics"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.RegisterDiagnoseFunc(storepb.Engine_SPANNER, Diagnose)
}

// Diagnose returns syntax diagnostics for the given GoogleSQL (Spanner)
// statement, surfacing lex/parse errors from the omni parser via the omni
// diagnostics analyzer. Mirrors the BigQuery cutover (and Trino #20517).
//
// omni's Analyze emits a "<X> statement parsing is not yet supported" stub
// diagnostic for the handful of valid-but-not-yet-modeled statement forms (e.g.
// IMPORT MODULE, EXPORT METADATA) — recognized statement-leading keywords whose
// body the omni parser doesn't build yet. The legacy ANTLR parser parsed them
// without error, so flagging them would be a false positive on valid SQL; we
// suppress only those stubs and pass every genuine lex/parse error (including
// "unknown or unsupported statement …") through.
func Diagnose(_ context.Context, _ base.DiagnoseContext, statement string) ([]base.Diagnostic, error) {
	diags := diagnostics.Analyze(statement)
	out := make([]base.Diagnostic, 0, len(diags))
	mapper := base.NewByteOffsetPositionMapper(statement)
	for _, d := range diags {
		if isValidButUnimplementedStub(d.Message) {
			continue
		}
		syntaxErr := &base.SyntaxError{
			Position: mapper.Position(d.Range.Start.Offset),
			Message:  d.Message,
		}
		out = append(out, base.ConvertSyntaxErrorToDiagnostic(syntaxErr, statement))
	}
	return out, nil
}

// isValidButUnimplementedStub reports whether msg is an omni "not yet supported"
// stub for a statement form that is valid GoogleSQL (so it must not be flagged
// as a syntax error). The genuine-unknown case ("unknown or unsupported
// statement starting with …") and all lexer errors are intentionally NOT matched
// here.
func isValidButUnimplementedStub(msg string) bool {
	return strings.HasSuffix(msg, "statement parsing is not yet supported")
}
