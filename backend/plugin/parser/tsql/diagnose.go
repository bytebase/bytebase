package tsql

import (
	"context"

	"github.com/bytebase/omni/mssql/analysis"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.RegisterDiagnoseFunc(storepb.Engine_MSSQL, Diagnose)
}

// Diagnose returns the script's first syntax error as a diagnostic.
func Diagnose(_ context.Context, _ base.DiagnoseContext, statement string) ([]base.Diagnostic, error) {
	diagnostics := make([]base.Diagnostic, 0)
	if syntaxError := toSyntaxError(analysis.Diagnose(statement)); syntaxError != nil {
		diagnostics = append(diagnostics, base.ConvertSyntaxErrorToDiagnostic(syntaxError, statement))
	}
	return diagnostics, nil
}

// toSyntaxError places omni's error in the text omni parsed. An error without
// a position goes at the start.
func toSyntaxError(e *analysis.SyntaxError) *base.SyntaxError {
	if e == nil {
		return nil
	}
	if e.Offset < 0 {
		return &base.SyntaxError{
			Position: &storepb.Position{Line: 1, Column: 1},
			Message:  e.Message,
		}
	}
	return &base.SyntaxError{
		Position:   ByteOffsetToRunePosition(e.Text, e.Offset),
		Message:    e.Message,
		RawMessage: e.Message,
	}
}
