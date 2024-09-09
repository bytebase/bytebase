package mysql

import (
	"context"
	"strings"
	"unicode"

	"github.com/antlr4-go/antlr/v4"
	parser "github.com/bytebase/mysql-parser"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/proto/generated-go/store"
)

func init() {
	base.RegisterDiagnoseFunc(store.Engine_MYSQL, Diagnose)
	base.RegisterDiagnoseFunc(store.Engine_MARIADB, Diagnose)
	base.RegisterDiagnoseFunc(store.Engine_TIDB, Diagnose)
	base.RegisterDiagnoseFunc(store.Engine_OCEANBASE, Diagnose)
	base.RegisterDiagnoseFunc(store.Engine_CLICKHOUSE, Diagnose)
	base.RegisterDiagnoseFunc(store.Engine_STARROCKS, Diagnose)
	base.RegisterDiagnoseFunc(store.Engine_DORIS, Diagnose)
}

func Diagnose(_ context.Context, _ base.DiagnoseContext, statement string) ([]base.Diagnostic, error) {
	diagnostics := make([]base.Diagnostic, 0)
	syntaxError := parseMySQLStatement(statement)
	if syntaxError != nil {
		diagnostics = append(diagnostics, base.ConvertSyntaxErrorToDiagnostic(syntaxError))
	}

	return diagnostics, nil
}

func parseMySQLStatement(statement string) *base.SyntaxError {
	trimmedStatement := strings.TrimRightFunc(statement, unicode.IsSpace)
	if !strings.HasSuffix(trimmedStatement, ";") {
		// Add a semicolon to the end of the statement to allow users to omit the semicolon
		// for the last statement in the script.
		statement += ";"
	}
	inputStream := antlr.NewInputStream(statement)
	lexer := parser.NewMySQLLexer(inputStream)
	stream := antlr.NewCommonTokenStream(lexer, antlr.TokenDefaultChannel)

	p := parser.NewMySQLParser(stream)
	lexerErrorListener := &base.ParseErrorListener{
		BaseLine: 0,
	}
	lexer.RemoveErrorListeners()
	lexer.AddErrorListener(lexerErrorListener)

	parserErrorListener := &base.ParseErrorListener{
		BaseLine: 0,
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
