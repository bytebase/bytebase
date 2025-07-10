package bigquery

import (
	"strings"

	"github.com/antlr4-go/antlr/v4"
	parser "github.com/bytebase/google-sql-parser"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.RegisterSplitterFunc(storepb.Engine_BIGQUERY, SplitSQL)
}

func SplitSQL(statement string) ([]base.SingleSQL, error) {
	lexer := parser.NewGoogleSQLLexer(antlr.NewInputStream(statement))
	stream := antlr.NewCommonTokenStream(lexer, antlr.TokenDefaultChannel)
	stream.Fill()

	tokens := stream.GetAllTokens()
	var buf []antlr.Token
	var sqls []base.SingleSQL
	for i, token := range tokens {
		if i < len(tokens)-1 {
			buf = append(buf, token)
		}
		if (token.GetTokenType() == parser.GoogleSQLLexerSEMI_SYMBOL || i == len(tokens)-1) && len(buf) > 0 {
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
			sqls = append(sqls, base.SingleSQL{
				Text:     bufStr.String(),
				BaseLine: common.ConvertANTLRLineToPositionLine(buf[0].GetLine()),
				End: common.ConvertANTLRPositionToPosition(&common.ANTLRPosition{
					Line:   int32(buf[len(buf)-1].GetLine()),
					Column: int32(buf[len(buf)-1].GetColumn()),
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
