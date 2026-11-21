package starrocks

import (
	"context"

	"github.com/bytebase/omni/starrocks/analysis"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.RegisterStatementRangesFunc(storepb.Engine_STARROCKS, GetStatementRanges)
}

func GetStatementRanges(_ context.Context, _ base.StatementRangeContext, statement string) ([]base.Range, error) {
	rawRanges := analysis.StatementRanges(statement)
	if len(rawRanges) == 0 {
		return nil, nil
	}
	return base.MapByteRangesToLSP(statement, rawRanges), nil
}
