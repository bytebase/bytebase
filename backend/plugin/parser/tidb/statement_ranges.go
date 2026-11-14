package tidb

import (
	"context"
	"unicode/utf8"

	"github.com/bytebase/omni/tidb/analysis"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"

	lsp "github.com/bytebase/lsp-protocol"
)

func init() {
	base.RegisterStatementRangesFunc(storepb.Engine_TIDB, GetStatementRanges)
}

// GetStatementRanges returns LSP-compatible ranges (0-based line, UTF-16 character)
// for each top-level SQL statement in statement. The ranges come from a lexical
// split, so the editor still gets them while the user is mid-edit.
func GetStatementRanges(_ context.Context, _ base.StatementRangeContext, statement string) ([]base.Range, error) {
	byteRanges := analysis.StatementRanges(statement)
	if byteRanges == nil {
		return nil, nil
	}

	positions := buildBytePositionMap(statement)

	ranges := make([]base.Range, 0, len(byteRanges))
	for _, r := range byteRanges {
		start, ok := positionAt(positions, r.Start)
		if !ok {
			continue
		}
		end, ok := positionAt(positions, r.End)
		if !ok {
			continue
		}
		ranges = append(ranges, base.Range{Start: start, End: end})
	}
	return ranges, nil
}

// buildBytePositionMap maps every byte offset in sql (0..len(sql) inclusive) to
// its LSP position (0-based line, 0-based UTF-16 character offset on that line).
// Surrogate pairs beyond the BMP count as two UTF-16 code units, matching LSP
// semantics.
func buildBytePositionMap(sql string) []lsp.Position {
	positions := make([]lsp.Position, len(sql)+1)
	var line, character uint32
	i := 0
	for i < len(sql) {
		positions[i] = lsp.Position{Line: line, Character: character}
		r, size := utf8.DecodeRuneInString(sql[i:])
		if r == '\n' {
			line++
			character = 0
		} else if r > 0xFFFF {
			character += 2
		} else {
			character++
		}
		i += size
	}
	positions[i] = lsp.Position{Line: line, Character: character}
	return positions
}

// positionAt returns the LSP Position at a rune-aligned byte offset. Offsets
// that land inside a multi-byte UTF-8 sequence return the zero Position, since
// buildBytePositionMap only fills rune boundaries. Out-of-bounds offsets return
// ok=false.
func positionAt(positions []lsp.Position, byteOffset int) (lsp.Position, bool) {
	if byteOffset < 0 || byteOffset >= len(positions) {
		return lsp.Position{}, false
	}
	return positions[byteOffset], true
}
