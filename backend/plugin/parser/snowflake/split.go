package snowflake

import (
	"github.com/antlr4-go/antlr/v4"
	parser "github.com/bytebase/parser/snowflake"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.RegisterSplitterFunc(storepb.Engine_SNOWFLAKE, SplitSQL)
}

// SplitSQL splits the given SQL statement into multiple SQL statements using ANTLR lexer.
func SplitSQL(statement string) ([]base.Statement, error) {
	lexer := parser.NewSnowflakeLexer(antlr.NewInputStream(statement))
	stream := antlr.NewCommonTokenStream(lexer, antlr.TokenDefaultChannel)
	stream.Fill()

	return base.SplitSQLByLexer(stream, parser.SnowflakeLexerSEMI)
}
