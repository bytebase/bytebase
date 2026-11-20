package trino

import (
	"context"

	"github.com/bytebase/omni/trino/analysis"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.RegisterDiagnoseFunc(storepb.Engine_TRINO, Diagnose)
}

// Diagnose returns syntax diagnostics for the given Trino statement,
// delegating parse and stub filtering to omni's analysis package.
func Diagnose(_ context.Context, _ base.DiagnoseContext, statement string) ([]base.Diagnostic, error) {
	diags := analysis.Diagnose(statement)
	out := make([]base.Diagnostic, 0, len(diags))
	mapper := base.NewByteOffsetPositionMapper(statement)
	for _, d := range diags {
		syntaxErr := &base.SyntaxError{
			Position: mapper.Position(d.Loc.Start),
			Message:  d.Msg,
		}
		out = append(out, base.ConvertSyntaxErrorToDiagnostic(syntaxErr, statement))
	}
	return out, nil
}
