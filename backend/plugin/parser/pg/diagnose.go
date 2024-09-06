package pg

import (
	"context"

	"github.com/antlr4-go/antlr/v4"
	parser "github.com/bytebase/postgresql-parser"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/proto/generated-go/store"
)

func init() {
	base.RegisterDiagnoseFunc(store.Engine_POSTGRES, Diagnose)
	base.RegisterDiagnoseFunc(store.Engine_REDSHIFT, Diagnose)
	base.RegisterDiagnoseFunc(store.Engine_RISINGWAVE, Diagnose)
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
func parsePostgreSQLStatement(sql string) *base.SyntaxError {
	lexer := parser.NewPostgreSQLLexer(antlr.NewInputStream(sql))
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
