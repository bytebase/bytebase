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
	diagnostics := make([]base.Diagnostic, 0)
	if syntaxError := convertDiagnoseError(analysis.Diagnose(statement)); syntaxError != nil {
		diagnostics = append(diagnostics, base.ConvertSyntaxErrorToDiagnostic(syntaxError, statement))
	}
	return diagnostics, nil
}

// convertDiagnoseError positions the error in the text omni parsed.
func convertDiagnoseError(err *analysis.DiagnoseError) *base.SyntaxError {
	if err == nil {
		return nil
	}
	if !err.ParseError {
		return &base.SyntaxError{
			Position: &storepb.Position{Line: 1, Column: 1},
			Message:  err.Message,
		}
	}
	return &base.SyntaxError{
		Position:   ByteOffsetToRunePosition(err.Text, err.Offset),
		Message:    err.Message,
		RawMessage: err.Message,
	}
}
