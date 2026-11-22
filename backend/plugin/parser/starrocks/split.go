package starrocks

import (
	"github.com/bytebase/omni/starrocks/analysis"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.RegisterSplitterFunc(storepb.Engine_STARROCKS, SplitSQL)
}

// SplitSQL splits the input into multiple SQL statements using omni's analysis
// package, then attaches 1-based line and rune column positions.
func SplitSQL(statement string) ([]base.Statement, error) {
	return base.BuildStatements(statement, analysis.SplitStatements(statement), func(s analysis.Statement) (int, int, bool) {
		return s.Start, s.End, s.Empty
	}, false), nil
}
