package redshift

import (
	"context"
	"unicode/utf8"

	lsp "github.com/bytebase/lsp-protocol"
	omniredshift "github.com/bytebase/omni/redshift"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.RegisterStatementRangesFunc(storepb.Engine_REDSHIFT, GetStatementRanges)
}

func GetStatementRanges(_ context.Context, _ base.StatementRangeContext, statement string) ([]base.Range, error) {
	omniRanges, err := omniredshift.StatementRanges(statement)
	if err == nil {
		ranges := make([]base.Range, 0, len(omniRanges))
		for _, r := range omniRanges {
			ranges = append(ranges, base.Range{
				Start: lsp.Position{
					Line:      uint32(r.Start.Line),
					Character: uint32(r.Start.Character),
				},
				End: lsp.Position{
					Line:      uint32(r.End.Line),
					Character: uint32(r.End.Character),
				},
			})
		}
		return ranges, nil
	}

	return getLexicalStatementRanges(statement)
}

func getLexicalStatementRanges(statement string) ([]base.Range, error) {
	statements, err := SplitSQL(statement)
	if err != nil {
		return nil, err
	}
	if len(statements) == 0 {
		return nil, nil
	}

	positions := buildUTF16PositionMap(statement)
	ranges := make([]base.Range, 0, len(statements))
	for _, stmt := range statements {
		if stmt.Empty || stmt.Range == nil {
			continue
		}
		start := int(stmt.Range.Start)
		end := int(stmt.Range.End)
		startPos, ok := positionAtByteOffset(positions, start)
		if !ok {
			continue
		}
		endPos, ok := positionAtByteOffset(positions, end)
		if !ok {
			continue
		}
		ranges = append(ranges, base.Range{
			Start: startPos,
			End:   endPos,
		})
	}
	return ranges, nil
}

func buildUTF16PositionMap(sql string) []lsp.Position {
	positions := make([]lsp.Position, len(sql)+1)
	var line, character uint32
	for i := 0; i < len(sql); {
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
	positions[len(sql)] = lsp.Position{Line: line, Character: character}
	return positions
}

func positionAtByteOffset(positions []lsp.Position, byteOffset int) (lsp.Position, bool) {
	if byteOffset < 0 || byteOffset >= len(positions) {
		return lsp.Position{}, false
	}
	return positions[byteOffset], true
}
