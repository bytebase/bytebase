package tsql

import (
	"context"
	"errors"
	"strings"
	"unicode"

	mssqlomniparser "github.com/bytebase/omni/mssql/parser"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.RegisterDiagnoseFunc(storepb.Engine_MSSQL, Diagnose)
}

func Diagnose(_ context.Context, _ base.DiagnoseContext, statement string) ([]base.Diagnostic, error) {
	diagnostics := make([]base.Diagnostic, 0)
	syntaxError := parseTSQLStatement(statement)
	if syntaxError != nil {
		diagnostics = append(diagnostics, base.ConvertSyntaxErrorToDiagnostic(syntaxError, statement))
	}

	return diagnostics, nil
}

func parseTSQLStatement(statement string) *base.SyntaxError {
	trimmedStatement := strings.TrimRightFunc(statement, unicode.IsSpace)
	if len(trimmedStatement) > 0 && !strings.HasSuffix(trimmedStatement, ";") {
		// Add a semicolon to the end of the statement to allow users to omit the semicolon
		// for the last statement in the script.
		statement += ";"
	}

	_, err := ParseTSQLOmni(statement)
	if err == nil {
		return nil
	}

	var parseErr *mssqlomniparser.ParseError
	if !errors.As(err, &parseErr) {
		return &base.SyntaxError{
			Position: &storepb.Position{Line: 1, Column: 1},
			Message:  err.Error(),
		}
	}

	message := parseErr.Error()
	return &base.SyntaxError{
		Position:   ByteOffsetToRunePosition(statement, parseErr.Position),
		Message:    message,
		RawMessage: message,
	}
}
