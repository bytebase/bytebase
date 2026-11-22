package trino

import (
	"context"

	"github.com/bytebase/omni/trino/analysis"
	"github.com/bytebase/omni/trino/parser"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.RegisterDiagnoseFunc(storepb.Engine_TRINO, Diagnose)
}

// Diagnose returns syntax diagnostics for the given Trino statement,
// delegating parse and stub filtering to omni's analysis package.
func Diagnose(_ context.Context, _ base.DiagnoseContext, statement string) ([]base.Diagnostic, error) {
	return base.ConvertOffsetDiagnostics(statement, analysis.Diagnose(statement), func(d parser.Diagnostic) (int, string) {
		return d.Loc.Start, d.Msg
	}), nil
}
