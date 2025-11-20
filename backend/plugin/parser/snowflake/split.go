package snowflake

import (
	"strings"

	"github.com/antlr4-go/antlr/v4"
	parser "github.com/bytebase/parser/snowflake"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.RegisterSplitterFunc(storepb.Engine_SNOWFLAKE, SplitSQL)
}

// SplitSQL splits the given SQL statement into multiple SQL statements using ANTLR lexer.
func SplitSQL(statement string) ([]base.SingleSQL, error) {
	lexer := parser.NewSnowflakeLexer(antlr.NewInputStream(statement))
	stream := antlr.NewCommonTokenStream(lexer, antlr.TokenDefaultChannel)
	stream.Fill()

	tokens := stream.GetAllTokens()
	var buf []antlr.Token
	var sqls []base.SingleSQL

	for i, token := range tokens {
		if i < len(tokens)-1 {
			buf = append(buf, token)
		}
		if (token.GetTokenType() == parser.SnowflakeLexerSEMI || i == len(tokens)-1) && len(buf) > 0 {
			bufStr := new(strings.Builder)
			empty := true

			for _, b := range buf {
				if _, err := bufStr.WriteString(b.GetText()); err != nil {
					return nil, err
				}
				if b.GetChannel() != antlr.TokenHiddenChannel {
					empty = false
				}
			}
			antlrPosition := base.FirstDefaultChannelTokenPosition(buf)

			// For the End position, use the current token (semicolon or EOF)
			// instead of the last token in buf
			sqls = append(sqls, base.SingleSQL{
				Text:     bufStr.String(),
				BaseLine: buf[0].GetLine() - 1,
				End: common.ConvertANTLRPositionToPosition(&common.ANTLRPosition{
					Line:   int32(token.GetLine()),
					Column: int32(token.GetColumn()),
				}, statement),
				Start: common.ConvertANTLRPositionToPosition(antlrPosition, statement),
				Empty: empty,
			})
			buf = nil
			continue
		}
	}
	return sqls, nil
}
