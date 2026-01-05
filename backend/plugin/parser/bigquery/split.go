package bigquery

import (
	"github.com/antlr4-go/antlr/v4"
	parser "github.com/bytebase/parser/googlesql"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.RegisterSplitterFunc(storepb.Engine_BIGQUERY, SplitSQL)
}

func SplitSQL(statement string) ([]base.Statement, error) {
	lexer := parser.NewGoogleSQLLexer(antlr.NewInputStream(statement))
	stream := antlr.NewCommonTokenStream(lexer, antlr.TokenDefaultChannel)
	stream.Fill()
	return base.SplitSQLByLexer(stream, parser.GoogleSQLLexerSEMI_SYMBOL)
}
