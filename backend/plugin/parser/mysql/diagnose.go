package mysql

import (
	"context"
	"errors"
	"strings"
	"unicode"

	mysqlomniparser "github.com/bytebase/omni/mysql/parser"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.RegisterDiagnoseFunc(storepb.Engine_MYSQL, Diagnose)
	base.RegisterDiagnoseFunc(storepb.Engine_MARIADB, Diagnose)
	base.RegisterDiagnoseFunc(storepb.Engine_OCEANBASE, Diagnose)
	base.RegisterDiagnoseFunc(storepb.Engine_CLICKHOUSE, Diagnose)
}

func Diagnose(_ context.Context, _ base.DiagnoseContext, statement string) ([]base.Diagnostic, error) {
	diagnostics := make([]base.Diagnostic, 0)
	syntaxError := parseMySQLStatement(statement)
	if syntaxError != nil {
		diagnostics = append(diagnostics, base.ConvertSyntaxErrorToDiagnostic(syntaxError, statement))
	}

	return diagnostics, nil
}

func parseMySQLStatement(statement string) *base.SyntaxError {
	trimmedStatement := strings.TrimRightFunc(statement, unicode.IsSpace)
	if len(trimmedStatement) > 0 && !strings.HasSuffix(trimmedStatement, ";") {
		// Add a semicolon to the end of the statement to allow users to omit the semicolon
		// for the last statement in the script.
		statement += ";"
	}

	_, err := ParseMySQLOmni(statement)
	if err == nil {
		return nil
	}

	var parseErr *mysqlomniparser.ParseError
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
