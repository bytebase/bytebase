package tidb

import (
	"context"
	"unicode"
	"unicode/utf8"

	omnitidbparser "github.com/bytebase/omni/tidb/parser"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"

	lsp "github.com/bytebase/lsp-protocol"
)

func init() {
	base.RegisterStatementRangesFunc(storepb.Engine_TIDB, GetStatementRanges)
}

// GetStatementRanges returns LSP-compatible ranges (0-based line, UTF-16 character)
// for each top-level SQL statement in statement.
//
// Implementation note: this uses omni/tidb's Split, which is a pure lexical
// scanner and works on both valid and invalid SQL. This is required for LSP
// use cases — the caller (e.g. SQL editor Ctrl+Enter) asks for ranges while the
// user is mid-edit, so any grammar-level Parse would fail-fast and drop ranges
// for statements preceding a syntax error. Split is soft-fail by construction.
func GetStatementRanges(_ context.Context, _ base.StatementRangeContext, statement string) ([]base.Range, error) {
	segments := omnitidbparser.Split(statement)
	if len(segments) == 0 {
		return nil, nil
	}

	positions := buildBytePositionMap(statement)

	ranges := make([]base.Range, 0, len(segments))
	for _, seg := range segments {
		startOffset := min(seg.ByteStart+leadingWhitespaceBytes(statement, seg.ByteStart, seg.ByteEnd), seg.ByteEnd)

		endOffset := seg.ByteEnd + delimLenAt(statement, seg.ByteEnd)

		start, ok := positionAt(positions, startOffset)
		if !ok {
			continue
		}
		end, ok := positionAt(positions, endOffset)
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

func positionAt(positions []lsp.Position, byteOffset int) (lsp.Position, bool) {
	if byteOffset < 0 || byteOffset >= len(positions) {
		return lsp.Position{}, false
	}
	return positions[byteOffset], true
}

// leadingWhitespaceBytes returns the number of leading Unicode whitespace bytes
// within sql[start:end]. Mirrors the whitespace trim performed by the ANTLR-based
// statement-range helper so TiDB ranges line up with snowflake/doris/pg behavior.
func leadingWhitespaceBytes(sql string, start, end int) int {
	if start < 0 {
		start = 0
	}
	if end > len(sql) {
		end = len(sql)
	}
	i := start
	for i < end {
		r, size := utf8.DecodeRuneInString(sql[i:])
		if !unicode.IsSpace(r) {
			break
		}
		i += size
	}
	return i - start
}

// delimLenAt returns the length of the statement delimiter that sits at
// sql[byteEnd:], if any. For the common default delimiter (';') this is 1; for
// the last segment without a trailing delimiter it is 0. Custom DELIMITER
// directives are not fully tracked here — the range will simply stop at the
// start of the custom delimiter in that case, which is sufficient for cursor-
// contained-in-range checks used by BYT-9157.
func delimLenAt(sql string, byteEnd int) int {
	if byteEnd < 0 || byteEnd >= len(sql) {
		return 0
	}
	if sql[byteEnd] == ';' {
		return 1
	}
	return 0
}
