package mongodb

import (
	"context"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.RegisterDiagnoseFunc(storepb.Engine_MONGODB, Diagnose)
}

// Diagnose performs syntax checking on a MongoDB shell script.
// Uses best-effort parsing to report all syntax errors, not just the first one.
func Diagnose(_ context.Context, _ base.DiagnoseContext, statement string) ([]base.Diagnostic, error) {
	_, parseErrors := ParseMongoShellBestEffort(statement)

	var diagnostics []base.Diagnostic
	for _, parseErr := range parseErrors {
		syntaxErr := &base.SyntaxError{
			Position: &storepb.Position{
				Line:   int32(parseErr.Line),
				Column: int32(parseErr.Column),
			},
			RawMessage: parseErr.Message,
			Message:    parseErr.Message,
		}
		diagnostics = append(diagnostics, base.ConvertSyntaxErrorToDiagnostic(syntaxErr, statement))
	}

	return diagnostics, nil
}
