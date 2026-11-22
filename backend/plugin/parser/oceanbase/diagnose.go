package oceanbase

import (
	"context"

	"github.com/bytebase/omni/oceanbase/analysis"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.RegisterDiagnoseFunc(storepb.Engine_OCEANBASE, Diagnose)
}

// Diagnose returns the script's first syntax error as a diagnostic.
func Diagnose(_ context.Context, _ base.DiagnoseContext, statement string) ([]base.Diagnostic, error) {
	var syntaxErr *base.SyntaxError
	if e := analysis.Diagnose(statement); e != nil {
		syntaxErr = base.SyntaxErrorAtOffset(e.Text, e.Offset, e.Message)
	}
	return base.DiagnoseSyntaxError(statement, syntaxErr), nil
}
