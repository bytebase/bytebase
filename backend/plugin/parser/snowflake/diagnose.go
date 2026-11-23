package snowflake

import (
	"context"

	"github.com/bytebase/omni/snowflake/diagnostics"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.RegisterDiagnoseFunc(storepb.Engine_SNOWFLAKE, Diagnose)
}

// Diagnose returns syntax diagnostics for the given Snowflake statement. It
// surfaces genuine lex/parse errors from the omni best-effort parser.
func Diagnose(_ context.Context, _ base.DiagnoseContext, statement string) ([]base.Diagnostic, error) {
	return base.ConvertOffsetDiagnostics(statement, diagnostics.Analyze(statement), func(d diagnostics.Diagnostic) (int, string) {
		return d.Range.Start.Offset, d.Message
	}), nil
}
