package trino

import (
	"context"
	"strings"

	"github.com/bytebase/omni/trino/parser"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.RegisterDiagnoseFunc(storepb.Engine_TRINO, Diagnose)
}

// Diagnose returns syntax diagnostics for the given Trino statement. It surfaces
// genuine lex/parse errors from the omni parser.
//
// The omni Trino parser emits a "<NAME> statement parsing is not yet supported"
// stub error from a handful of dispatch fallthroughs. Most of those fire only on
// genuinely malformed input — every valid CREATE/DROP/ALTER object keyword is
// implemented, so e.g. "CREATE TABLEE ..." (an unrecognized object) is a true
// SYNTAX_ERROR (confirmed against the Trino 481 oracle) and we must keep it. The
// only valid-Trino forms that still hit the stub are DESCRIBE INPUT / DESCRIBE
// OUTPUT (oracle: NOT_FOUND, i.e. they parse). We suppress ONLY those so the
// editor does not show a false positive on valid SQL; everything else passes
// through. When omni implements DESCRIBE INPUT/OUTPUT the stub stops firing and
// this filter becomes a no-op.
func Diagnose(_ context.Context, _ base.DiagnoseContext, statement string) ([]base.Diagnostic, error) {
	diags := parser.Diagnose(statement)
	out := make([]base.Diagnostic, 0, len(diags))
	mapper := base.NewByteOffsetPositionMapper(statement)
	for _, d := range diags {
		if isValidButUnimplementedStub(d.Msg) {
			continue
		}
		syntaxErr := &base.SyntaxError{
			Position: mapper.Position(d.Loc.Start),
			Message:  d.Msg,
		}
		out = append(out, base.ConvertSyntaxErrorToDiagnostic(syntaxErr, statement))
	}
	return out, nil
}

// isValidButUnimplementedStub reports whether msg is an omni "not yet supported"
// stub error for a statement form that is valid Trino (so it must not be flagged
// as a syntax error). Today that is only DESCRIBE INPUT / DESCRIBE OUTPUT; the
// CREATE/DROP/ALTER stubs fire exclusively on unrecognized object keywords,
// which are genuine syntax errors, so they are intentionally NOT matched here.
func isValidButUnimplementedStub(msg string) bool {
	return strings.HasPrefix(msg, "DESCRIBE ") &&
		strings.HasSuffix(msg, "statement parsing is not yet supported")
}
