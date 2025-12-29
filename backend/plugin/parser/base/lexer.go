package base

import (
	"strings"

	"github.com/antlr4-go/antlr/v4"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

func GetDefaultChannelTokenType(tokens []antlr.Token, base int, offset int) int {
	current := base
	step := 1
	remaining := offset
	if offset < 0 {
		step = -1
		remaining = -offset
	}
	for remaining != 0 {
		current += step
		if current < 0 || current >= len(tokens) {
			return antlr.TokenEOF
		}

		if tokens[current].GetChannel() == antlr.TokenDefaultChannel {
			remaining--
		}
	}

	return tokens[current].GetTokenType()
}

// FirstDefaultChannelTokenPosition returns the first token position of the default channel.
// Both line and column are ZERO based.
func FirstDefaultChannelTokenPosition(tokens []antlr.Token) *common.ANTLRPosition {
	for _, token := range tokens {
		if token.GetChannel() == antlr.TokenDefaultChannel {
			return &common.ANTLRPosition{
				Line:   int32(token.GetLine()),
				Column: int32(token.GetColumn()),
			}
		}
	}
	return &common.ANTLRPosition{
		Line:   int32(tokens[len(tokens)-1].GetLine()),
		Column: int32(tokens[len(tokens)-1].GetColumn()),
	}
}

func IsEmpty(tokens []antlr.Token, semi int) bool {
	for _, token := range tokens {
		if token.GetChannel() == antlr.TokenDefaultChannel && token.GetTokenType() != semi && token.GetTokenType() != antlr.TokenEOF {
			return false
		}
	}
	return true
}

// SplitSQLByLexer is a grammar-free helper function that splits SQL statements using an ANTLR lexer.
// It works with any ANTLR-based lexer by accepting the token stream and semicolon token type as parameters.
//
// Parameters:
//   - stream: The ANTLR token stream (must be already filled with stream.Fill())
//   - semiTokenType: The token type value for semicolon in the specific grammar
//   - statement: The original SQL statement string (used for position conversion)
//
// Returns:
//   - A slice of Statement, each representing one statement with its text, position, and metadata
//
// Example usage:
//
//	lexer := parser.NewSnowflakeLexer(antlr.NewInputStream(statement))
//	stream := antlr.NewCommonTokenStream(lexer, antlr.TokenDefaultChannel)
//	stream.Fill()
//	return SplitSQLByLexer(stream, parser.SnowflakeLexerSEMI, statement)
func SplitSQLByLexer(stream *antlr.CommonTokenStream, semiTokenType int, statement string) ([]Statement, error) {
	tokens := stream.GetAllTokens()
	var buf []antlr.Token
	var sqls []Statement

	for i, token := range tokens {
		// Collect all tokens except the last one (EOF)
		if i < len(tokens)-1 {
			buf = append(buf, token)
		}

		// Split on semicolon or at the end of tokens
		if (token.GetTokenType() == semiTokenType || i == len(tokens)-1) && len(buf) > 0 {
			// Reconstruct the text from tokens
			bufStr := new(strings.Builder)
			empty := true

			for _, b := range buf {
				if _, err := bufStr.WriteString(b.GetText()); err != nil {
					return nil, err
				}
				// Check if there's any non-hidden token (actual code, not just comments/whitespace)
				if b.GetChannel() != antlr.TokenHiddenChannel {
					empty = false
				}
			}

			// Get the position of the first default channel token for Start position
			antlrPosition := FirstDefaultChannelTokenPosition(buf)

			// Use the last token in the buffer for End position (not EOF token when at end of stream)
			lastToken := buf[len(buf)-1]
			sqls = append(sqls, Statement{
				Text:     bufStr.String(),
				BaseLine: buf[0].GetLine() - 1, // BaseLine is the offset of the first token
				Range: &storepb.Range{
					Start: int32(buf[0].GetStart()),
					End:   int32(lastToken.GetStop() + 1),
				},
				End: common.ConvertANTLRTokenToExclusiveEndPosition(
					int32(lastToken.GetLine()),
					int32(lastToken.GetColumn()),
					lastToken.GetText(),
				),
				Start: common.ConvertANTLRPositionToPosition(antlrPosition, statement),
				Empty: empty,
			})

			buf = nil
			continue
		}
	}

	return sqls, nil
}
