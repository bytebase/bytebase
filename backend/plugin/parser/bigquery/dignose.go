package bigquery

import (
	"context"
	"strings"
	"unicode"

	"github.com/antlr4-go/antlr/v4"
	parser "github.com/bytebase/parser/googlesql"

	"github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.RegisterDiagnoseFunc(store.Engine_BIGQUERY, Diagnose)
}

func Diagnose(_ context.Context, _ base.DiagnoseContext, statement string) ([]base.Diagnostic, error) {
	diagnostics := make([]base.Diagnostic, 0)
	syntaxError := parseGoogleSQLStatement(statement)
	if syntaxError != nil {
		diagnostics = append(diagnostics, base.ConvertSyntaxErrorToDiagnostic(syntaxError, statement))
	}

	return diagnostics, nil
}

func parseGoogleSQLStatement(statement string) *base.SyntaxError {
	trimmedStatement := strings.TrimRightFunc(statement, unicode.IsSpace)
	if len(trimmedStatement) > 0 && !strings.HasSuffix(trimmedStatement, ";") {
		// Add a semicolon to the end of the statement to allow users to omit the semicolon
		// for the last statement in the script.
		statement += ";"
	}
	inputStream := antlr.NewInputStream(statement)
	lexer := parser.NewGoogleSQLLexer(inputStream)
	stream := antlr.NewCommonTokenStream(lexer, antlr.TokenDefaultChannel)

	p := parser.NewGoogleSQLParser(stream)
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

	p.BuildParseTrees = false

	_ = p.Root()

	if lexerErrorListener.Err != nil {
		return lexerErrorListener.Err
	}

	if parserErrorListener.Err != nil {
		return parserErrorListener.Err
	}
	return nil
}
