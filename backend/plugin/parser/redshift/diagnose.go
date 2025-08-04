package redshift

import (
	"context"
	"strings"
	"unicode"

	"github.com/antlr4-go/antlr/v4"
	parser "github.com/bytebase/parser/redshift"

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

// parseRedshiftStatement parses the given SQL and returns the ParseResult.
// Use the Redshift parser based on antlr4.
func parseRedshiftStatement(statement string) *base.SyntaxError {
	trimmedStatement := strings.TrimRightFunc(statement, unicode.IsSpace)
	if len(trimmedStatement) > 0 && !strings.HasSuffix(trimmedStatement, ";") {
		// Add a semicolon to the end of the statement to allow users to omit the semicolon
		// for the last statement in the script.
		statement += ";"
	}
	lexer := parser.NewRedshiftLexer(antlr.NewInputStream(statement))
	stream := antlr.NewCommonTokenStream(lexer, antlr.TokenDefaultChannel)
	p := parser.NewRedshiftParser(stream)
	lexerErrorListener := &base.ParseErrorListener{
		Statement: statement,
	}
	lexer.RemoveErrorListeners()
	lexer.AddErrorListener(lexerErrorListener)

	parserErrorListener := &base.ParseErrorListener{
		Statement: statement,
	}
	p.RemoveErrorListeners()
	p.AddErrorListener(parserErrorListener)

	p.BuildParseTrees = true

	p.Root()

	if lexerErrorListener.Err != nil {
		return lexerErrorListener.Err
	}

	if parserErrorListener.Err != nil {
		return parserErrorListener.Err
	}

	return nil
}
