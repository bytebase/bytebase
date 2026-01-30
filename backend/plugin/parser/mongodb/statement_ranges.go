package mongodb

import (
	"context"

	lsp "github.com/bytebase/lsp-protocol"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.RegisterStatementRangesFunc(storepb.Engine_MONGODB, GetStatementRanges)
}

// GetStatementRanges returns the ranges of statements in the MongoDB shell script.
func GetStatementRanges(_ context.Context, _ base.StatementRangeContext, statement string) ([]base.Range, error) {
	raw := parseMongoShellRaw(statement)
	if raw == nil || raw.Tree == nil {
		return []base.Range{}, nil
	}

	// Build a mapping from character (rune) offset to LSP position.
	// ANTLR returns character offsets, not byte offsets.
	runePositions := buildRunePositionMap(statement)

	var ranges []base.Range

	for _, stmt := range raw.Tree.AllStatement() {
		if stmt == nil {
			continue
		}
		start := stmt.GetStart()
		stop := stmt.GetStop()
		if start == nil || stop == nil {
			continue
		}

		startOffset := start.GetStart()
		endOffset := stop.GetStop() + 1

		startPosition := getPositionByRuneOffset(startOffset, runePositions)
		endPosition := getPositionByRuneOffset(endOffset, runePositions)
		if startPosition == nil || endPosition == nil {
			continue
		}

		ranges = append(ranges, base.Range{
			Start: *startPosition,
			End:   *endPosition,
		})
	}

	return ranges, nil
}

// runePosition stores the LSP position for a given rune offset.
type runePosition struct {
	line      uint32
	character uint32 // UTF-16 code units
}

// buildRunePositionMap creates a mapping from rune offset to LSP position.
func buildRunePositionMap(statement string) []runePosition {
	runes := []rune(statement)
	positions := make([]runePosition, len(runes)+1) // +1 for end position

	var line uint32
	var character uint32

	for i, r := range runes {
		positions[i] = runePosition{line: line, character: character}

		if r == '\n' {
			line++
			character = 0
		} else {
			// Count UTF-16 code units
			if r > 0xFFFF {
				// Surrogate pair needed
				character += 2
			} else {
				character++
			}
		}
	}

	// Position after the last character
	positions[len(runes)] = runePosition{line: line, character: character}

	return positions
}

// getPositionByRuneOffset converts a rune (character) offset to an LSP Position.
func getPositionByRuneOffset(runeOffset int, positions []runePosition) *lsp.Position {
	if runeOffset < 0 || runeOffset >= len(positions) {
		return nil
	}
	pos := positions[runeOffset]
	return &lsp.Position{
		Line:      pos.line,
		Character: pos.character,
	}
}
