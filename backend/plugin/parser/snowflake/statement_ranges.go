package snowflake

import (
	"context"
	"strings"
	"unicode"

	"github.com/antlr4-go/antlr/v4"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"

	protocol "github.com/bytebase/lsp-protocol"
	parser "github.com/bytebase/snowsql-parser"

	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

func init() {
	base.RegisterStatementRangesFunc(storepb.Engine_SNOWFLAKE, GetStatementRanges)
}

func GetStatementRanges(_ context.Context, _ base.StatementRangeContext, statement string) ([]base.Range, error) {
	ranges := getSQLStatementRangesUTF16Position(statement)
	return ranges, nil
}

func getSQLStatementRangesUTF16Position(statement string) []base.Range {
	trimmedStatement := strings.TrimRightFunc(statement, unicode.IsSpace)
	inputStream := antlr.NewInputStream(trimmedStatement)
	lexer := parser.NewSnowflakeLexer(inputStream)
	stream := antlr.NewCommonTokenStream(lexer, antlr.TokenDefaultChannel)
	stream.Fill()

	var ranges []base.Range
	var buf []antlr.Token
	beginLine, beginUTF16CodePointOffset := 0, 0
	endLine, endUTF16CodePointOffset := 0, 0
	for _, token := range stream.GetAllTokens() {
		if token.GetTokenType() == parser.SnowflakeParserEOF {
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

		// Snowflake parser only uses SEMI as a statement terminator.
		if token.GetTokenType() != parser.SnowflakeParserSEMI {
			continue
		}
		if len(buf) == 1 {
			// Ignore single SEMI statement.
			buf = nil
			continue
		}

		ranges = append(ranges, base.Range{
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

		// Set the next begin position to the next token after SEMI.
		beginLine = endLine
		beginUTF16CodePointOffset = endUTF16CodePointOffset
	}

	// If there are remaining token in the buffer, it means the last statement does not end with SEMI.
	if len(buf) > 0 {
		ranges = append(ranges, base.Range{
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
