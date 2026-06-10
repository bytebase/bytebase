package starrocks

import (
	"context"
	"strings"
	"unicode"
	"unicode/utf8"

	protocol "github.com/bytebase/lsp-protocol"
	"github.com/bytebase/omni/starrocks/parser"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.RegisterStatementRangesFunc(storepb.Engine_STARROCKS, GetStatementRanges)
}

// GetStatementRanges returns the UTF-16 line/character ranges of statements
// in the input, formatted for the LSP protocol. Empty (whitespace-only)
// segments are skipped, and ranges include the trailing semicolon when present
// to match the prior ANTLR-based behaviour.
func GetStatementRanges(_ context.Context, _ base.StatementRangeContext, statement string) ([]base.Range, error) {
	trimmed := strings.TrimRightFunc(statement, unicode.IsSpace)

	segs := parser.Split(trimmed)
	if len(segs) == 0 {
		// omni's splitter drops comment-only buffers entirely. The legacy
		// ANTLR helper still emitted a range for trailing comment tokens, so
		// editor features that count statement positions don't lose ground
		// to a file that contains only `-- note` or `/* ... */`. Emit a
		// single range covering the non-whitespace span when present.
		return commentOnlyRanges(trimmed), nil
	}

	type pos struct{ line, char uint32 }
	wanted := make(map[int]pos, len(segs)*3)
	for _, seg := range segs {
		wanted[seg.ByteStart] = pos{}
		wanted[seg.ByteEnd] = pos{}
		if seg.ByteEnd < len(trimmed) && trimmed[seg.ByteEnd] == ';' {
			wanted[seg.ByteEnd+1] = pos{}
		}
	}

	var line, char uint32
	for i := 0; i <= len(trimmed); {
		if _, need := wanted[i]; need {
			wanted[i] = pos{line, char}
		}
		if i == len(trimmed) {
			break
		}
		r, size := utf8.DecodeRuneInString(trimmed[i:])
		if r == '\n' {
			line++
			char = 0
		} else if r <= 0xFFFF {
			char++
		} else {
			char += 2 // UTF-16 surrogate pair
		}
		i += size
	}

	ranges := make([]base.Range, 0, len(segs))
	for _, seg := range segs {
		// Skip pure whitespace segments, but preserve comment-only segments
		// (the legacy ANTLR helper emits ranges for trailing comment tokens).
		if seg.Empty() && !hasNonWhitespace(trimmed[seg.ByteStart:seg.ByteEnd]) {
			continue
		}
		// Skip only leading whitespace — leading comments are kept inside the
		// range to match the prior ANTLR-helper behaviour, which includes
		// hidden-channel tokens but excludes pure whitespace.
		trimStart := seg.ByteStart
		for trimStart < seg.ByteEnd {
			c := trimmed[trimStart]
			if c == ' ' || c == '\t' || c == '\n' || c == '\r' {
				trimStart++
				continue
			}
			break
		}
		if _, ok := wanted[trimStart]; !ok {
			startPos := wanted[seg.ByteStart]
			l, c := startPos.line, startPos.char
			for i := seg.ByteStart; i < trimStart; {
				r, size := utf8.DecodeRuneInString(trimmed[i:])
				if r == '\n' {
					l++
					c = 0
				} else if r <= 0xFFFF {
					c++
				} else {
					c += 2
				}
				i += size
			}
			wanted[trimStart] = pos{l, c}
		}
		startPos, ok := wanted[trimStart]
		if !ok {
			// trimStart wasn't pre-collected; fall back to ByteStart.
			startPos = wanted[seg.ByteStart]
		}
		end := seg.ByteEnd
		if end < len(trimmed) && trimmed[end] == ';' {
			end++
		}
		endPos := wanted[end]
		ranges = append(ranges, base.Range{
			Start: protocol.Position{Line: startPos.line, Character: startPos.char},
			End:   protocol.Position{Line: endPos.line, Character: endPos.char},
		})
	}
	// Trailing comment-only content: omni's Split filters out comment-only
	// segments, so any non-whitespace bytes between the last segment's end
	// and the end of the trimmed input would otherwise be lost. The legacy
	// ANTLR helper emits a final range for these — replicate that.
	if len(segs) > 0 {
		last := segs[len(segs)-1]
		tailStart := last.ByteEnd
		if tailStart < len(trimmed) && trimmed[tailStart] == ';' {
			tailStart++
		}
		// Find first non-whitespace byte after tailStart.
		i := tailStart
		for i < len(trimmed) {
			c := trimmed[i]
			if c != ' ' && c != '\t' && c != '\n' && c != '\r' {
				break
			}
			i++
		}
		if i < len(trimmed) {
			// There's a trailing comment / non-whitespace chunk. Emit a range.
			if _, ok := wanted[i]; !ok {
				startPos := wanted[tailStart]
				l, c := startPos.line, startPos.char
				for j := tailStart; j < i; {
					r, size := utf8.DecodeRuneInString(trimmed[j:])
					if r == '\n' {
						l++
						c = 0
					} else if r <= 0xFFFF {
						c++
					} else {
						c += 2
					}
					j += size
				}
				wanted[i] = pos{l, c}
			}
			startPos := wanted[i]
			endPos := pos{startPos.line, startPos.char}
			for j := i; j < len(trimmed); {
				r, size := utf8.DecodeRuneInString(trimmed[j:])
				if r == '\n' {
					endPos.line++
					endPos.char = 0
				} else if r <= 0xFFFF {
					endPos.char++
				} else {
					endPos.char += 2
				}
				j += size
			}
			ranges = append(ranges, base.Range{
				Start: protocol.Position{Line: startPos.line, Character: startPos.char},
				End:   protocol.Position{Line: endPos.line, Character: endPos.char},
			})
		}
	}

	return ranges, nil
}

// commentOnlyRanges emits a single Range covering the non-whitespace span of
// `trimmed`, or nil when nothing in `trimmed` looks like a comment. Used
// when the splitter finds zero segments — typically that means comment-only
// input like `-- note` or `/* ... */`, which the legacy ANTLR helper would
// have emitted a range for. Pure delimiters (e.g. a single `;`) are NOT
// turned into ranges; the legacy helper dropped single-terminator inputs.
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
	var startLine, startChar uint32
	for i := 0; i < start; {
		r, size := utf8.DecodeRuneInString(trimmed[i:])
		if r == '\n' {
			startLine++
			startChar = 0
		} else if r <= 0xFFFF {
			startChar++
		} else {
			startChar += 2
		}
		i += size
	}
	endLine, endChar := startLine, startChar
	for i := start; i < len(trimmed); {
		r, size := utf8.DecodeRuneInString(trimmed[i:])
		if r == '\n' {
			endLine++
			endChar = 0
		} else if r <= 0xFFFF {
			endChar++
		} else {
			endChar += 2
		}
		i += size
	}
	return []base.Range{{
		Start: protocol.Position{Line: startLine, Character: startChar},
		End:   protocol.Position{Line: endLine, Character: endChar},
	}}
}

// containsCommentMarker reports whether s contains one of the SQL comment
// introducers (`--`, `/*`, `#`). Anything inside a string literal could
// false-positive, but the splitter only routes us here when there are
// no segments at all — i.e. no string literals to worry about.
func containsCommentMarker(s string) bool {
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c == '#' {
			return true
		}
		if i+1 < len(s) {
			if c == '-' && s[i+1] == '-' {
				return true
			}
			if c == '/' && s[i+1] == '*' {
				return true
			}
		}
	}
	return false
}

// hasNonWhitespace reports whether s contains any non-whitespace character.
// Used to detect comment-only segments (omni's Segment.Empty() treats them
// as empty since they contain no SQL tokens, but bytebase wants their ranges).
func hasNonWhitespace(s string) bool {
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c != ' ' && c != '\t' && c != '\n' && c != '\r' {
			return true
		}
	}
	return false
}
