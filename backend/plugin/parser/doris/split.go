package doris

import (
	"github.com/antlr4-go/antlr/v4"
	parser "github.com/bytebase/parser/doris"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.RegisterSplitterFunc(storepb.Engine_STARROCKS, SplitSQL)
	base.RegisterSplitterFunc(storepb.Engine_DORIS, SplitSQL)
}

// SplitSQL splits the input into multiple SQL statements using semicolon as delimiter.
func SplitSQL(statement string) ([]base.Statement, error) {
	lexer := parser.NewDorisLexer(antlr.NewInputStream(statement))
	stream := antlr.NewCommonTokenStream(lexer, antlr.TokenDefaultChannel)
	stream.Fill()

	return base.SplitSQLByLexer(stream, parser.DorisLexerSEMICOLON)
}
