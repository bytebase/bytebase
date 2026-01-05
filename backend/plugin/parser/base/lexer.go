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
// IMPORTANT: This function requires that the lexer sends whitespace to the hidden channel
// (using `-> channel(HIDDEN)` in the grammar), NOT skipping it. This ensures GetAllTokens()
// returns all tokens including whitespace, allowing us to use token positions directly.
//
// Parameters:
//   - stream: The ANTLR token stream (must be already filled with stream.Fill())
//   - semiTokenType: The token type value for semicolon in the specific grammar
//
// Returns:
//   - A slice of Statement, each representing one statement with its text, position, and metadata
//
// Example usage:
//
//	lexer := parser.NewSnowflakeLexer(antlr.NewInputStream(statement))
//	stream := antlr.NewCommonTokenStream(lexer, antlr.TokenDefaultChannel)
//	stream.Fill()
//	return SplitSQLByLexer(stream, parser.SnowflakeLexerSEMI)
func SplitSQLByLexer(stream *antlr.CommonTokenStream, semiTokenType int) ([]Statement, error) {
	tokens := stream.GetAllTokens()
	var buf []antlr.Token
	var sqls []Statement
	byteOffset := 0

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

			stmtText := bufStr.String()
			stmtByteLength := len(stmtText)

			// Use the last token in the buffer for End position (not EOF token when at end of stream)
			lastToken := buf[len(buf)-1]
			// Use buf[0]'s position directly since GetAllTokens() includes hidden channel tokens (whitespace/comments)
			sqls = append(sqls, Statement{
				Text: stmtText,
				Range: &storepb.Range{
					Start: int32(byteOffset),
					End:   int32(byteOffset + stmtByteLength),
				},
				End: common.ConvertANTLRTokenToExclusiveEndPosition(
					int32(lastToken.GetLine()),
					int32(lastToken.GetColumn()),
					lastToken.GetText(),
				),
				Start: &storepb.Position{
					Line:   int32(buf[0].GetLine()),
					Column: int32(buf[0].GetColumn() + 1),
				},
				Empty: empty,
			})

			byteOffset += stmtByteLength
			buf = nil
			continue
		}
	}

	return sqls, nil
}

// CalculateLineAndColumn calculates the 0-based line number and 0-based column (character offset)
// for a given byte offset in the statement.
func CalculateLineAndColumn(statement string, byteOffset int) (line, column int) {
	if byteOffset > len(statement) {
		byteOffset = len(statement)
	}
	// Range over string iterates over runes (code points), not bytes
	for _, r := range statement[:byteOffset] {
		if r == '\n' {
			line++
			column = 0
		} else {
			column++
		}
	}
	return line, column
}
