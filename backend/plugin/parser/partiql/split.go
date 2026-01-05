package partiql

import (
	"github.com/antlr4-go/antlr/v4"
	parser "github.com/bytebase/parser/partiql"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.RegisterSplitterFunc(storepb.Engine_DYNAMODB, SplitSQL)
}

// SplitSQL splits the input into multiple SQL statements using semicolon as delimiter.
func SplitSQL(statement string) ([]base.Statement, error) {
	lexer := parser.NewPartiQLLexer(antlr.NewInputStream(statement))
	stream := antlr.NewCommonTokenStream(lexer, antlr.TokenDefaultChannel)
	stream.Fill()

	return base.SplitSQLByLexer(stream, parser.PartiQLLexerCOLON_SEMI)
}
