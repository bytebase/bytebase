package tidb

import (
	"context"

	"github.com/bytebase/omni/tidb/analysis"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.RegisterStatementRangesFunc(storepb.Engine_TIDB, GetStatementRanges)
}

// GetStatementRanges returns LSP-compatible ranges (0-based line, UTF-16 character)
// for each top-level SQL statement in statement. The ranges come from a lexical
// split, so the editor still gets them while the user is mid-edit.
func GetStatementRanges(_ context.Context, _ base.StatementRangeContext, statement string) ([]base.Range, error) {
	return base.MapByteRangesToLSP(statement, analysis.StatementRanges(statement), func(r analysis.ByteRange) (int, int) {
		return r.Start, r.End
	}), nil
}
