package googlesql

import (
	"github.com/bytebase/omni/googlesql/parser"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

// SplitSQL splits the input into multiple SQL statements using the omni
// GoogleSQL splitter (block-aware: BEGIN/END, IF, LOOP, CASE bodies stay
// whole), re-attaching trailing ';' delimiters when present.
//
// When cfg.ContiguousSplitText is true (Spanner), each statement's Text runs
// contiguously from the end of the previous statement so inter-statement
// whitespace and comments attach to the following statement.
func SplitSQL(statement string, cfg Config) ([]base.Statement, error) {
	segs := parser.Split(statement)
	if !cfg.ContiguousSplitText {
		return base.BuildStatements(statement, segs, func(s parser.Segment) (int, int, bool) {
			return s.ByteStart, s.ByteEnd, s.Empty()
		}, true), nil
	}
	prevEnd := 0
	return base.BuildStatements(statement, segs, func(s parser.Segment) (int, int, bool) {
		start := prevEnd
		end := s.ByteEnd
		if end < len(statement) && statement[end] == ';' {
			end++
		}
		prevEnd = end
		return start, end, s.Empty()
	}, false), nil
}
