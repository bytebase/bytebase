package base

import (
	"github.com/antlr4-go/antlr/v4"

	"github.com/bytebase/bytebase/backend/common"
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
