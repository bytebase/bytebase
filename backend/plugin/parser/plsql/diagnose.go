package plsql

import (
	"context"

	"github.com/bytebase/omni/oracle/analysis"

	"github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.RegisterDiagnoseFunc(store.Engine_ORACLE, Diagnose)
}

// Diagnose returns the script's first syntax error as a diagnostic.
func Diagnose(_ context.Context, _ base.DiagnoseContext, statement string) ([]base.Diagnostic, error) {
	return base.DiagnoseSyntaxError(statement, toSyntaxError(analysis.Diagnose(statement))), nil
}

// toSyntaxError places omni's error in the text omni parsed. An error without
// a position goes at the start.
func toSyntaxError(e *analysis.SyntaxError) *base.SyntaxError {
	if e == nil {
		return nil
	}
	return base.SyntaxErrorAtOffset(e.Text, e.Offset, e.Message)
}
