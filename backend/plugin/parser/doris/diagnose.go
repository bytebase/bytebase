package doris

import (
	"context"

	"github.com/bytebase/omni/doris/analysis"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.RegisterDiagnoseFunc(storepb.Engine_DORIS, Diagnose)
}

// Diagnose returns syntax diagnostics for the given Doris statement,
// delegating parse and stub filtering to omni's analysis package.
func Diagnose(_ context.Context, _ base.DiagnoseContext, statement string) ([]base.Diagnostic, error) {
	diags := analysis.Diagnose(statement)
	out := make([]base.Diagnostic, 0, len(diags))
	mapper := base.NewByteOffsetPositionMapper(statement)
	for _, d := range diags {
		syntaxErr := &base.SyntaxError{
			Position: mapper.Position(d.Offset),
			Message:  d.Message,
		}
		out = append(out, base.ConvertSyntaxErrorToDiagnostic(syntaxErr, statement))
	}
	return out, nil
}
