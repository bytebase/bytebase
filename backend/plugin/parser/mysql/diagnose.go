package mysql

import (
	"context"
	"strings"
	"unicode"

	"github.com/antlr4-go/antlr/v4"
	parser "github.com/bytebase/parser/mysql"

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
	inputStream := antlr.NewInputStream(statement)
	lexer := parser.NewMySQLLexer(inputStream)
	stream := antlr.NewCommonTokenStream(lexer, antlr.TokenDefaultChannel)

	p := parser.NewMySQLParser(stream)
	startPosition := &storepb.Position{Line: 1}
	lexerErrorListener := &base.ParseErrorListener{
		Statement:     statement,
		StartPosition: startPosition,
	}
	lexer.RemoveErrorListeners()
	lexer.AddErrorListener(lexerErrorListener)

	parserErrorListener := &base.ParseErrorListener{
		Statement:     statement,
		StartPosition: startPosition,
	}
	p.RemoveErrorListeners()
	p.AddErrorListener(parserErrorListener)

	p.BuildParseTrees = false

	_ = p.Script()

	if lexerErrorListener.Err != nil {
		return lexerErrorListener.Err
	}

	if parserErrorListener.Err != nil {
		return parserErrorListener.Err
	}
	return nil
}
