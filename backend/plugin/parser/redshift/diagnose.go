package redshift

import (
	"context"

	"github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.RegisterDiagnoseFunc(store.Engine_REDSHIFT, Diagnose)
}

func Diagnose(_ context.Context, _ base.DiagnoseContext, statement string) ([]base.Diagnostic, error) {
	diagnostics := make([]base.Diagnostic, 0)
	syntaxError := parseRedshiftStatement(statement)
	if syntaxError != nil {
		diagnostics = append(diagnostics, base.ConvertSyntaxErrorToDiagnostic(syntaxError, statement))
	}

	return diagnostics, nil
}

// parseRedshiftStatement parses the given SQL and returns syntax errors if any.
func parseRedshiftStatement(statement string) *base.SyntaxError {
	stmt := base.Statement{
		Text:  statement,
		Start: &store.Position{Line: 1, Column: 1},
	}
	if _, err := ParseRedshiftOmni(statement); err != nil {
		syntaxErr, ok := convertOmniError(err, stmt).(*base.SyntaxError)
		if !ok {
			return &base.SyntaxError{
				Position: &store.Position{Line: 1, Column: 1},
				Message:  err.Error(),
			}
		}
		return syntaxErr
	}
	return nil
}
