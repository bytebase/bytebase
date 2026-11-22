package starrocks

import (
	"context"

	"github.com/bytebase/omni/starrocks/analysis"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.RegisterDiagnoseFunc(storepb.Engine_STARROCKS, Diagnose)
}

// Diagnose returns syntax diagnostics for the given StarRocks statement,
// delegating parse and stub filtering to omni's analysis package.
func Diagnose(_ context.Context, _ base.DiagnoseContext, statement string) ([]base.Diagnostic, error) {
	return base.ConvertOffsetDiagnostics(statement, analysis.Diagnose(statement), func(d analysis.SyntaxError) (int, string) {
		return d.Offset, d.Message
	}), nil
}
