package googlesql

import (
	"github.com/bytebase/omni/googlesql/parser"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

// SplitSQL splits the input into multiple SQL statements using the omni
// GoogleSQL splitter (block-aware: BEGIN/END, IF, LOOP, CASE bodies stay
// whole), then converts each Segment into a base.Statement with the position
// fields the legacy splitters produced. Positions are computed in a single O(n)
// pass via the shared base.ByteOffsetPositionMapper (segment offsets are
// queried in increasing order).
//
// Text conventions differ per dialect (Config.ContiguousSplitText): the legacy
// spanner parse-tree splitter ran each statement's Text contiguously from the
// END of the previous statement (inter-statement whitespace and comments attach
// to the FOLLOWING statement), while the legacy bigquery splitter started each
// statement's Text at its own first token. Both included the trailing ';',
// which omni's Segment.Text excludes (ByteEnd points AT it), so it is
// re-attached when present.
func SplitSQL(statement string, cfg Config) ([]base.Statement, error) {
	segs := parser.Split(statement)
	if len(segs) == 0 {
		return nil, nil
	}

	mapper := base.NewByteOffsetPositionMapper(statement)
	stmts := make([]base.Statement, 0, len(segs))
	prevEnd := 0
	for _, seg := range segs {
		end := seg.ByteEnd
		if end < len(statement) && statement[end] == ';' {
			end++
		}
		start := seg.ByteStart
		text := seg.Text
		if cfg.ContiguousSplitText {
			start = prevEnd
			text = statement[prevEnd:end]
		} else if end > seg.ByteEnd {
			text = statement[seg.ByteStart:end]
		}
		stmts = append(stmts, base.Statement{
			Text:  text,
			Empty: seg.Empty(),
			Start: mapper.Position(start),
			End:   mapper.Position(end),
			Range: &storepb.Range{
				Start: int32(start),
				End:   int32(end),
			},
		})
		prevEnd = end
	}
	return stmts, nil
}
