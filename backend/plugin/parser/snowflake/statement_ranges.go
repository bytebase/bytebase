package snowflake

import (
	"context"

	"github.com/bytebase/omni/snowflake/analysis"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.RegisterStatementRangesFunc(storepb.Engine_SNOWFLAKE, GetStatementRanges)
}

// GetStatementRanges returns the UTF-16 line/character ranges of statements in
// the input, formatted for the LSP protocol.
func GetStatementRanges(_ context.Context, _ base.StatementRangeContext, statement string) ([]base.Range, error) {
	return base.MapByteRangesToLSP(statement, analysis.StatementRanges(statement)), nil
}
