package redshift

import (
	"context"
	"strings"
	"unicode"

	lsp "github.com/bytebase/lsp-protocol"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.RegisterStatementRangesFunc(storepb.Engine_REDSHIFT, GetStatementRanges)
}

func GetStatementRanges(_ context.Context, _ base.StatementRangeContext, statement string) ([]base.Range, error) {
	ranges := getSQLStatementRangesUTF16Position([]byte(statement))
	return ranges, nil
}

func getSQLStatementRangesUTF16Position(content []byte) []lsp.Range {
	s := strings.TrimRightFunc(string(content), unicode.IsSpace)
	// Assuming the content is UTF-8 encoded.
	statements := strings.Split(s, ";")

	var ranges []lsp.Range
	// 0-based UTF-16 encoded line and character.
	line, character := 0, 0

	for i, statement := range statements {
		// Trim left space to provide more accurate range.
		statement = strings.TrimLeftFunc(statement, func(r rune) bool {
			if !unicode.IsSpace(r) {
				return false
			}
			if r == '\n' {
				line++
				character = 0
			} else {
				// Check rune utf16 length by BMP.
				if r <= 0xFFFF {
					character++
				} else {
					character += 2
				}
			}
			return true
		})

		// If the statement is empty, skip it.
		if statement == "" {
			continue
		}

		begin := lsp.Position{Line: uint32(line), Character: uint32(character)}
		for _, r := range statement {
			if r == '\n' {
				line++
				character = 0
			} else {
				// Check rune utf16 length by BMP.
				if r <= 0xFFFF {
					character++
				} else {
					character += 2
				}
			}
		}

		endLine, endCharacter := line, character
		// End is exclusive, so we check the next byte.
		if i == len(statements)-1 {
			// End of the content.
			endLine++
			endCharacter = 0
		} else {
			// Next byte is ';', include it.
			character++
			endCharacter++
			if nextStatement := statements[i+1]; len(nextStatement) > 0 && nextStatement[0] == '\n' {
				endLine++
				endCharacter = 0
			} else {
				endCharacter++
			}
		}
		end := lsp.Position{Line: uint32(endLine), Character: uint32(endCharacter)}
		ranges = append(ranges, lsp.Range{Start: begin, End: end})
	}
	return ranges
}
