package trino

import (
	"github.com/bytebase/omni/trino/parser"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.RegisterSplitterFunc(storepb.Engine_TRINO, SplitSQL)
}

// SplitSQL splits the input into multiple SQL statements using the omni Trino
// splitter, re-attaching trailing ';' delimiters when present.
func SplitSQL(statement string) ([]base.Statement, error) {
	return base.BuildStatements(statement, parser.Split(statement), func(s parser.Segment) (int, int, bool) {
		return s.ByteStart, s.ByteEnd, s.Empty()
	}, true), nil
}
