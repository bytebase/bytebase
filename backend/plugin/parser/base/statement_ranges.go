package base

import (
	"strings"
	"unicode"

	"github.com/antlr4-go/antlr/v4"

	protocol "github.com/bytebase/lsp-protocol"
)

// GetANTLRStatementRangesUTF16Position is a generic function to extract statement ranges from ANTLR token streams.
// It returns the ranges of statements in UTF-16 positions suitable for LSP protocol.
//
// Parameters:
//   - tokenStream: The ANTLR token stream to process
//   - eofTokenType: The EOF token type (e.g., parser.SnowflakeParserEOF)
//   - terminatorTokenType: The statement terminator token type (e.g., parser.SnowflakeParserSEMI)
//
// The function handles:
//   - UTF-16 surrogate pairs for characters outside the Basic Multilingual Plane (BMP)
//   - Trimming leading whitespace from statements for more accurate ranges
//   - Ignoring single terminator statements
//   - Handling statements that don't end with a terminator
func GetANTLRStatementRangesUTF16Position(tokenStream *antlr.CommonTokenStream, eofTokenType, terminatorTokenType int) []Range {
	var ranges []Range
	var buf []antlr.Token
	beginLine, beginUTF16CodePointOffset := 0, 0
	endLine, endUTF16CodePointOffset := 0, 0

	for _, token := range tokenStream.GetAllTokens() {
		if token.GetTokenType() == eofTokenType {
			break
		}
		tokenText := token.GetText()
		for _, r := range tokenText {
			if r == '\n' {
				endLine++
				endUTF16CodePointOffset = 0
			} else {
				if r <= 0xFFFF {
					endUTF16CodePointOffset++
				} else {
					// For non-BMP characters, we count as 2 UTF-16 code units.
					endUTF16CodePointOffset += 2
				}
			}
		}

		// Ignore heading spaces to provide more accurate range.
		if len(buf) == 0 && len(strings.TrimSpace(tokenText)) == 0 {
			beginLine = endLine
			beginUTF16CodePointOffset = endUTF16CodePointOffset
			continue
		}

		buf = append(buf, token)

		// Check if this is a statement terminator.
		if token.GetTokenType() != terminatorTokenType {
			continue
		}
		if len(buf) == 1 {
			// Ignore single terminator statement.
			buf = nil
			continue
		}

		ranges = append(ranges, Range{
			Start: protocol.Position{
				Line:      uint32(beginLine),
				Character: uint32(beginUTF16CodePointOffset),
			},
			End: protocol.Position{
				Line:      uint32(endLine),
				Character: uint32(endUTF16CodePointOffset),
			},
		})
		buf = nil

		// Set the next begin position to the next token after terminator.
		beginLine = endLine
		beginUTF16CodePointOffset = endUTF16CodePointOffset
	}

	// If there are remaining tokens in the buffer, it means the last statement does not end with terminator.
	if len(buf) > 0 {
		ranges = append(ranges, Range{
			Start: protocol.Position{
				Line:      uint32(beginLine),
				Character: uint32(beginUTF16CodePointOffset),
			},
			End: protocol.Position{
				Line:      uint32(endLine),
				Character: uint32(endUTF16CodePointOffset),
			},
		})
	}

	return ranges
}

// PrepareANTLRTokenStream is a helper function to prepare an ANTLR token stream from a statement.
// It trims trailing whitespace and creates the necessary lexer and token stream.
//
// Parameters:
//   - statement: The SQL statement to process
//   - createLexer: A function that creates an ANTLR lexer from an input stream
//
// Returns:
//   - A filled CommonTokenStream ready for processing
func PrepareANTLRTokenStream(statement string, createLexer func(antlr.CharStream) antlr.Lexer) *antlr.CommonTokenStream {
	trimmedStatement := strings.TrimRightFunc(statement, unicode.IsSpace)
	inputStream := antlr.NewInputStream(trimmedStatement)
	lexer := createLexer(inputStream)
	stream := antlr.NewCommonTokenStream(lexer, antlr.TokenDefaultChannel)
	stream.Fill()
	return stream
}
