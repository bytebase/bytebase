package base

import "github.com/antlr4-go/antlr/v4"

// Scanner is the wrapper of antlr.CommonTokenStream.
// It provides a common scanner for all parsers
// and more useful methods.
type Scanner struct {
	input  *antlr.CommonTokenStream
	index  int
	tokens []antlr.Token
	// tokenStack is used to store the index of the token for saving and restoring.
	tokenStack []int
}

// NewScanner creates a new Scanner.
func NewScanner(input *antlr.CommonTokenStream, fillInput bool) *Scanner {
	if fillInput {
		input.Fill()
	}
	return &Scanner{
		input:  input,
		index:  0,
		tokens: input.GetAllTokens(),
	}
}

// GetIndex returns the current index of the scanner.
func (s *Scanner) GetIndex() int {
	return s.index
}

// GetTokenChannel returns the channel of the current token.
func (s *Scanner) GetTokenChannel() int {
	return s.tokens[s.index].GetChannel()
}

// GetTokenText returns the text of the current token.
func (s *Scanner) GetTokenText() string {
	return s.tokens[s.index].GetText()
}

// GetPreviousTokenType returns the type of the previous token.
// If skipHidden is true, it will skip the hidden tokens.
// It does not change the current index of the scanner.
func (s *Scanner) GetPreviousTokenType(skipHidden bool) int {
	index := s.index
	for index > 0 {
		index--
		if s.tokens[index].GetChannel() == antlr.TokenDefaultChannel || !skipHidden {
			return s.tokens[index].GetTokenType()
		}
	}

	return antlr.TokenInvalidType
}

// GetPreviousTokenText returns the text of the previous tok
// If skipHidden is true, it will skip the hidden tokens.
// It does not change the current index of the scanner.
func (s *Scanner) GetPreviousTokenText(skipHidden bool) string {
	index := s.index
	for index > 0 {
		index--
		if s.tokens[index].GetChannel() == antlr.TokenDefaultChannel || !skipHidden {
			return s.tokens[index].GetText()
		}
	}

	return ""
}

// Push pushes the current index to the stack.
func (s *Scanner) Push() {
	s.tokenStack = append(s.tokenStack, s.index)
}

// PopAndRestore pops the index from the stack and restores the index.
func (s *Scanner) PopAndRestore() bool {
	if len(s.tokenStack) == 0 {
		return false
	}

	s.index = s.tokenStack[len(s.tokenStack)-1]
	s.tokenStack = s.tokenStack[:len(s.tokenStack)-1]
	return true
}

// Backward moves the index backward.
// If skipHidden is true, it will skip the hidden tokens.
func (s *Scanner) Backward(skipHidden bool) bool {
	for s.index > 0 {
		s.index--
		if s.tokens[s.index].GetChannel() == antlr.TokenDefaultChannel || !skipHidden {
			return true
		}
	}
	return false
}

// Forward moves the index forward.
// If skipHidden is true, it will skip the hidden tokens.
func (s *Scanner) Forward(skipHidden bool) bool {
	for s.index < len(s.tokens)-1 {
		s.index++
		if s.tokens[s.index].GetChannel() == antlr.TokenDefaultChannel || !skipHidden {
			return true
		}
	}
	return false
}

// GetTokenType returns the type of the current token.
func (s *Scanner) GetTokenType() int {
	return s.tokens[s.index].GetTokenType()
}

// SkipTokenSequence skips the token sequence.
// This method will ignore the hidden tokens for scanner, but not check for given token list.
// If the given token list is not fully matched:
//  1. It will skip the common prefix of the given token list and the current token list in the scanner.
//  2. Return false.
//
// If the given token list is fully matched:
//  1. It will skip the matched tokens in the scanner.
//  2. Return true.
func (s *Scanner) SkipTokenSequence(list []int) bool {
	if s.index >= len(s.tokens) {
		return false
	}

	for _, token := range list {
		if s.tokens[s.index].GetTokenType() != token {
			return false
		}

		// Skip to the next unhidden token.
		s.index++
		for s.index < len(s.tokens) && s.tokens[s.index].GetChannel() != antlr.TokenDefaultChannel {
			s.index++
		}

		if s.index >= len(s.tokens) {
			return false
		}
	}

	// Fully matched.
	return true
}

// IsTokenType returns whether the current token type is the given token type.
func (s *Scanner) IsTokenType(tokenType int) bool {
	return s.tokens[s.index].GetTokenType() == tokenType
}

// SeekIndex seeks the index of the scanner.
// If the index is out of range, it will do nothing.
func (s *Scanner) SeekIndex(index int) {
	if index >= 0 && index < len(s.tokens) {
		s.index = index
	}
}

// SeekPosition seeks the position of the scanner.
//  1. If the position is before the first token, it will do nothing and return false.
//
// The following cases will move the index to the token and return true:
//  2. If the position is in one of the tokens, it will move the index to the token.
//  3. If the position is between two tokens, it will move the index to the previous token.
//  4. If the position is after the last token, it will move the index to the last token.
func (s *Scanner) SeekPosition(line, column int) bool {
	if len(s.tokens) == 0 {
		return false
	}

	index := 0
	for index < len(s.tokens) {
		token := s.tokens[index]
		tokenLine := token.GetLine()
		if tokenLine >= line {
			tokenColumn := token.GetColumn()
			tokenLength := token.GetStop() - token.GetStart() + 1
			if tokenLine == line && tokenColumn <= column && column < tokenColumn+tokenLength {
				s.index = index
				break
			}

			if tokenLine > line || tokenColumn > column {
				if index == 0 {
					return false
				}

				s.index = index - 1
				break
			}
		}
		index++
	}

	if index == len(s.tokens) {
		s.index = index - 1
	}

	return true
}

// GetFollowingText returns the following text of the current token.
func (s *Scanner) GetFollowingText() string {
	stream := s.tokens[s.index].GetInputStream()
	return stream.GetText(s.tokens[s.index].GetStart(), stream.Size()-1)
}

// GetFollowingTextAfter returns the following text after the given position.
func (s *Scanner) GetFollowingTextAfter(pos int) string {
	stream := s.tokens[s.index].GetInputStream()
	return stream.GetText(pos, stream.Size()-1)
}
