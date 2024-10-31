package bigquery

import (
	"strings"

	"github.com/antlr4-go/antlr/v4"
	parser "github.com/bytebase/google-sql-parser"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
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
			line, col := base.FirstDefaultChannelTokenPosition(buf)
			sqls = append(sqls, base.SingleSQL{
				Text:                 bufStr.String(),
				BaseLine:             buf[0].GetLine() - 1,
				LastLine:             buf[len(buf)-1].GetLine() - 1,
				LastColumn:           buf[len(buf)-1].GetColumn(),
				FirstStatementLine:   line,
				FirstStatementColumn: col,
				Empty:                empty,
			})
			buf = nil
			continue
		}
	}
	return sqls, nil
}
