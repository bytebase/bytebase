package pg

import (
	"context"

	"github.com/bytebase/omni/pg/analysis"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.RegisterDiagnoseFunc(storepb.Engine_POSTGRES, Diagnose)
	base.RegisterDiagnoseFunc(storepb.Engine_COCKROACHDB, Diagnose)
}

func Diagnose(_ context.Context, _ base.DiagnoseContext, statement string) ([]base.Diagnostic, error) {
	return base.DiagnoseSyntaxError(statement, convertDiagnoseError(analysis.Diagnose(statement))), nil
}

// convertDiagnoseError positions the error in the text omni parsed.
func convertDiagnoseError(err *analysis.DiagnoseError) *base.SyntaxError {
	if err == nil {
		return nil
	}
	if !err.ParseError {
		return base.SyntaxErrorAtOffset(err.Text, -1, err.Message)
	}
	return base.SyntaxErrorAtOffset(err.Text, err.Offset, err.Message)
}
