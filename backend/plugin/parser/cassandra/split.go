package cassandra

import (
	"github.com/antlr4-go/antlr/v4"
	"github.com/bytebase/parser/cql"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.RegisterSplitterFunc(storepb.Engine_CASSANDRA, SplitSQL)
}

// SplitSQL splits the input into multiple CQL statements using semicolon as delimiter.
func SplitSQL(statement string) ([]base.Statement, error) {
	lexer := cql.NewCqlLexer(antlr.NewInputStream(statement))
	stream := antlr.NewCommonTokenStream(lexer, antlr.TokenDefaultChannel)
	stream.Fill()

	return base.SplitSQLByLexer(stream, cql.CqlLexerSEMI)
}
