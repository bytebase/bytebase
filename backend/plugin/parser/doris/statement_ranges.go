package doris

import (
	"context"

	"github.com/bytebase/omni/doris/analysis"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.RegisterStatementRangesFunc(storepb.Engine_DORIS, GetStatementRanges)
}

func GetStatementRanges(_ context.Context, _ base.StatementRangeContext, statement string) ([]base.Range, error) {
	rawRanges := analysis.StatementRanges(statement)
	if len(rawRanges) == 0 {
		return nil, nil
	}
	return base.MapByteRangesToLSP(statement, rawRanges, func(r analysis.Range) (int, int) {
		return r.Start, r.End
	}), nil
}
