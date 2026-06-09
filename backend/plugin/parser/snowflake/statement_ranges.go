package snowflake

import (
	"context"
	"strings"
	"unicode"
	"unicode/utf8"

	protocol "github.com/bytebase/lsp-protocol"
	"github.com/bytebase/omni/snowflake/parser"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.RegisterStatementRangesFunc(storepb.Engine_SNOWFLAKE, GetStatementRanges)
}

// GetStatementRanges returns the UTF-16 line/character ranges of statements in
// the input, formatted for the LSP protocol. It is backed by the omni Snowflake
// splitter (parser.Split), the same splitter SplitSQL uses, so the emitted
// ranges line up byte-for-byte with the segments SplitSQL returns.
//
// Per segment the range:
//   - starts at the first non-whitespace byte (leading whitespace is trimmed,
//     but leading comments are kept inside the range — matching the prior
//     ANTLR helper, which folded hidden-channel comment tokens into the
//     following statement while excluding pure whitespace);
//   - ends just past the trailing ';' delimiter when present (omni's
//     Segment.ByteEnd points AT the ';', so we step over it), consistent with
//     SplitSQL re-attaching the delimiter.
//
// omni's splitter drops comment-only / lone-';' chunks entirely (zero
// segments). The legacy ANTLR helper still emitted a range for trailing
// comment tokens, so comment-only input and trailing comment tails are
// re-emitted here to keep editor features (which count statement positions)
// from losing ground — see commentOnlyRanges and the trailing-tail block.
func GetStatementRanges(_ context.Context, _ base.StatementRangeContext, statement string) ([]base.Range, error) {
	trimmed := strings.TrimRightFunc(statement, unicode.IsSpace)

	segs := parser.Split(trimmed)
	if len(segs) == 0 {
		// No SQL statements — typically comment-only input like `-- note` or
		// `/* ... */`, which the legacy ANTLR helper emitted a range for.
		return commentOnlyRanges(trimmed), nil
	}

	mapper := newUTF16Mapper(trimmed)

	ranges := make([]base.Range, 0, len(segs))
	for _, seg := range segs {
		// Skip pure-whitespace segments (omni filters comment-only chunks, so
		// any segment that reaches here with no non-whitespace byte is just
		// whitespace and carries no editor-meaningful position).
		if !hasNonWhitespace(trimmed[seg.ByteStart:seg.ByteEnd]) {
			continue
		}
		// Trim only leading whitespace — leading comments stay inside the
		// range to mirror the prior ANTLR helper.
		start := seg.ByteStart
		for start < seg.ByteEnd {
			c := trimmed[start]
			if c == ' ' || c == '\t' || c == '\n' || c == '\r' {
				start++
				continue
			}
			break
		}
		end := seg.ByteEnd
		if end < len(trimmed) && trimmed[end] == ';' {
			end++
		}
		ranges = append(ranges, base.Range{
			Start: mapper.position(start),
			End:   mapper.position(end),
		})
	}

	// Trailing comment-only content: omni's Split filters out comment-only
	// segments, so any non-whitespace bytes between the last segment's end and
	// the end of the trimmed input would otherwise be lost. The legacy ANTLR
	// helper emitted a final range for these — replicate that.
	last := segs[len(segs)-1]
	tailStart := last.ByteEnd
	if tailStart < len(trimmed) && trimmed[tailStart] == ';' {
		tailStart++
	}
	for tailStart < len(trimmed) {
		c := trimmed[tailStart]
		if c == ' ' || c == '\t' || c == '\n' || c == '\r' {
			tailStart++
			continue
		}
		break
	}
	if tailStart < len(trimmed) {
		ranges = append(ranges, base.Range{
			Start: mapper.position(tailStart),
			End:   mapper.position(len(trimmed)),
		})
	}

	return ranges, nil
}

// commentOnlyRanges emits a single Range covering the non-whitespace span of
// `trimmed`, or nil when nothing in `trimmed` looks like a comment. Used when
// the splitter finds zero segments — typically that means comment-only input
// like `-- note` or `/* ... */`, which the legacy ANTLR helper would have
// emitted a range for. Pure delimiters (e.g. a single `;`) are NOT turned into
// ranges; the legacy helper dropped single-terminator inputs.
func commentOnlyRanges(trimmed string) []base.Range {
	if !containsCommentMarker(trimmed) {
		return nil
	}
	start := 0
	for start < len(trimmed) {
		c := trimmed[start]
		if c == ' ' || c == '\t' || c == '\n' || c == '\r' {
			start++
			continue
		}
		break
	}
	if start >= len(trimmed) {
		return nil
	}
	mapper := newUTF16Mapper(trimmed)
	return []base.Range{{
		Start: mapper.position(start),
		End:   mapper.position(len(trimmed)),
	}}
}

// containsCommentMarker reports whether s contains one of the SQL comment
// introducers (`--`, `//`, `/*`). Anything inside a string literal could
// false-positive, but the splitter only routes us here when there are no
// segments at all — i.e. no string literals to worry about.
func containsCommentMarker(s string) bool {
	for i := 0; i < len(s); i++ {
		c := s[i]
		if i+1 < len(s) {
			if c == '-' && s[i+1] == '-' {
				return true
			}
			if c == '/' && (s[i+1] == '*' || s[i+1] == '/') {
				return true
			}
		}
	}
	return false
}

// hasNonWhitespace reports whether s contains any non-whitespace character.
func hasNonWhitespace(s string) bool {
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c != ' ' && c != '\t' && c != '\n' && c != '\r' {
			return true
		}
	}
	return false
}

// utf16Mapper maps byte offsets in a source string to UTF-16 LSP positions
// (0-based line, 0-based UTF-16 code-unit column) in a single O(n) precompute.
type utf16Mapper struct {
	positions []protocol.Position
}

func newUTF16Mapper(s string) *utf16Mapper {
	positions := make([]protocol.Position, len(s)+1)
	var line, character uint32
	for i := 0; i < len(s); {
		positions[i] = protocol.Position{Line: line, Character: character}
		r, size := utf8.DecodeRuneInString(s[i:])
		if r == '\n' {
			line++
			character = 0
		} else if r > 0xFFFF {
			character += 2 // UTF-16 surrogate pair
		} else {
			character++
		}
		i += size
	}
	positions[len(s)] = protocol.Position{Line: line, Character: character}
	return &utf16Mapper{positions: positions}
}

// position returns the UTF-16 position at byteOffset. byteOffset is clamped to
// [0, len(s)]; out-of-range offsets resolve to the nearest boundary.
func (m *utf16Mapper) position(byteOffset int) protocol.Position {
	if byteOffset < 0 {
		byteOffset = 0
	}
	if byteOffset >= len(m.positions) {
		byteOffset = len(m.positions) - 1
	}
	return m.positions[byteOffset]
}
