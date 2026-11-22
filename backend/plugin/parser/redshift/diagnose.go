package redshift

import (
	"context"
	"fmt"

	"github.com/bytebase/omni/redshift/analysis"

	"github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.RegisterDiagnoseFunc(store.Engine_REDSHIFT, Diagnose)
}

func Diagnose(_ context.Context, _ base.DiagnoseContext, statement string) ([]base.Diagnostic, error) {
	diagnostics := make([]base.Diagnostic, 0)
	if diag := analysis.Diagnose(statement); diag != nil {
		position := ByteOffsetToRunePosition(statement, diag.Offset)
		syntaxErr := &base.SyntaxError{
			Position:   position,
			Message:    fmt.Sprintf("Syntax error at line %d: %s", position.Line, diag.Message),
			RawMessage: diag.Message,
		}
		diagnostics = append(diagnostics, base.ConvertSyntaxErrorToDiagnostic(syntaxErr, statement))
	}
	return diagnostics, nil
}
