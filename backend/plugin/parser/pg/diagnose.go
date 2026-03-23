package pg

import (
	"context"
	"errors"
	"strings"
	"unicode"

	omniparser "github.com/bytebase/omni/pg/parser"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.RegisterDiagnoseFunc(storepb.Engine_POSTGRES, Diagnose)
	base.RegisterDiagnoseFunc(storepb.Engine_COCKROACHDB, Diagnose)
}

func Diagnose(_ context.Context, _ base.DiagnoseContext, statement string) ([]base.Diagnostic, error) {
	diagnostics := make([]base.Diagnostic, 0)
	syntaxError := parsePostgreSQLStatement(statement)
	if syntaxError != nil {
		diagnostics = append(diagnostics, base.ConvertSyntaxErrorToDiagnostic(syntaxError, statement))
	}

	return diagnostics, nil
}

// parsePostgreSQLStatement parses the given SQL and returns a SyntaxError if any.
func parsePostgreSQLStatement(statement string) *base.SyntaxError {
	trimmedStatement := strings.TrimRightFunc(statement, unicode.IsSpace)
	if len(trimmedStatement) > 0 && !strings.HasSuffix(trimmedStatement, ";") {
		// Add a semicolon to the end of the statement to allow users to omit the semicolon
		// for the last statement in the script.
		statement += ";"
	}

	_, err := ParsePg(statement)
	if err == nil {
		return nil
	}

	var parseErr *omniparser.ParseError
	if !errors.As(err, &parseErr) {
		return &base.SyntaxError{
			Position: &storepb.Position{Line: 1, Column: 1},
			Message:  err.Error(),
		}
	}

	pos := ByteOffsetToRunePosition(statement, parseErr.Position)
	return &base.SyntaxError{
		Position:   pos,
		Message:    parseErr.Message,
		RawMessage: parseErr.Message,
	}
}
