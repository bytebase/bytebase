package mongodb

import (
	"context"
	"unicode/utf8"

	lsp "github.com/bytebase/lsp-protocol"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.RegisterStatementRangesFunc(storepb.Engine_MONGODB, GetStatementRanges)
}

// GetStatementRanges returns the ranges of statements in the MongoDB shell script.
// Returns best-effort results: if parsing fails (e.g. incomplete input during live typing),
// returns empty ranges rather than an error so LSP features degrade gracefully.
func GetStatementRanges(_ context.Context, _ base.StatementRangeContext, statement string) ([]base.Range, error) {
	stmts, _ := ParseMongoShell(statement)

	// Build byte offset to LSP position mapping.
	positions := buildBytePositionMap(statement)

	var ranges []base.Range
	for _, stmt := range stmts {
		if stmt.Range == nil {
			continue
		}
		startPos := getPositionByByteOffset(int(stmt.Range.Start), positions)
		endPos := getPositionByByteOffset(int(stmt.Range.End), positions)
		if startPos == nil || endPos == nil {
			continue
		}
		ranges = append(ranges, base.Range{
			Start: *startPos,
			End:   *endPos,
		})
	}

	return ranges, nil
}

// bytePositionEntry stores the LSP position for a given byte offset.
type bytePositionEntry struct {
	line      uint32
	character uint32 // UTF-16 code units
}

// buildBytePositionMap creates a mapping from byte offset to LSP position.
func buildBytePositionMap(statement string) map[int]bytePositionEntry {
	positions := make(map[int]bytePositionEntry, len(statement)+1)

	var line uint32
	var character uint32
	i := 0

	for i < len(statement) {
		positions[i] = bytePositionEntry{line: line, character: character}

		r, size := utf8.DecodeRuneInString(statement[i:])
		if r == '\n' {
			line++
			character = 0
		} else {
			// Count UTF-16 code units.
			if r > 0xFFFF {
				character += 2 // surrogate pair
			} else {
				character++
			}
		}
		i += size
	}
	// Position after the last character.
	positions[i] = bytePositionEntry{line: line, character: character}

	return positions
}

// getPositionByByteOffset converts a byte offset to an LSP Position.
func getPositionByByteOffset(byteOffset int, positions map[int]bytePositionEntry) *lsp.Position {
	pos, ok := positions[byteOffset]
	if !ok {
		return nil
	}
	return &lsp.Position{
		Line:      pos.line,
		Character: pos.character,
	}
}
