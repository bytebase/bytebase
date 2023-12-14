package base

import (
	"github.com/antlr4-go/antlr/v4"
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
func FirstDefaultChannelTokenPosition(tokens []antlr.Token) (int, int) {
	for _, token := range tokens {
		if token.GetChannel() == antlr.TokenDefaultChannel {
			// From antlr4, the line is ONE based, and the column is ZERO based.
			// So we should minus 1 for the line.
			return token.GetLine() - 1, token.GetColumn()
		}
	}
	// From antlr4, the line is ONE based, and the column is ZERO based.
	// So we should minus 1 for the line.
	return tokens[len(tokens)-1].GetLine() - 1, tokens[len(tokens)-1].GetColumn()
}

func IsEmpty(tokens []antlr.Token, semi int) bool {
	for _, token := range tokens {
		if token.GetChannel() == antlr.TokenDefaultChannel && token.GetTokenType() != semi && token.GetTokenType() != antlr.TokenEOF {
			return false
		}
	}
	return true
}
