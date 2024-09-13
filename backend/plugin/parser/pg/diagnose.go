package pg

import (
	"context"
	"strings"
	"unicode"

	"github.com/antlr4-go/antlr/v4"
	parser "github.com/bytebase/postgresql-parser"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/proto/generated-go/store"
)

func init() {
	base.RegisterDiagnoseFunc(store.Engine_POSTGRES, Diagnose)
	base.RegisterDiagnoseFunc(store.Engine_REDSHIFT, Diagnose)
	base.RegisterDiagnoseFunc(store.Engine_RISINGWAVE, Diagnose)
	base.RegisterDiagnoseFunc(store.Engine_COCKROACHDB, Diagnose)
}

func Diagnose(_ context.Context, _ base.DiagnoseContext, statement string) ([]base.Diagnostic, error) {
	diagnostics := make([]base.Diagnostic, 0)
	syntaxError := parsePostgreSQLStatement(statement)
	if syntaxError != nil {
		diagnostics = append(diagnostics, base.ConvertSyntaxErrorToDiagnostic(syntaxError))
	}

	return diagnostics, nil
}

// parsePostgreSQLStatement parses the given SQL and returns the ParseResult.
// Use the PostgreSQL parser based on antlr4.
func parsePostgreSQLStatement(statement string) *base.SyntaxError {
	trimmedStatement := strings.TrimRightFunc(statement, unicode.IsSpace)
	if !strings.HasSuffix(trimmedStatement, ";") {
		// Add a semicolon to the end of the statement to allow users to omit the semicolon
		// for the last statement in the script.
		statement += ";"
	}
	lexer := parser.NewPostgreSQLLexer(antlr.NewInputStream(statement))
	stream := antlr.NewCommonTokenStream(lexer, antlr.TokenDefaultChannel)
	p := parser.NewPostgreSQLParser(stream)
	lexerErrorListener := &base.ParseErrorListener{}
	lexer.RemoveErrorListeners()
	lexer.AddErrorListener(lexerErrorListener)

	parserErrorListener := &base.ParseErrorListener{
		BaseLine: 0,
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
