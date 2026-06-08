package redshift

import (
	"context"

	lsp "github.com/bytebase/lsp-protocol"
	omniredshift "github.com/bytebase/omni/redshift"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.RegisterStatementRangesFunc(storepb.Engine_REDSHIFT, GetStatementRanges)
}

func GetStatementRanges(_ context.Context, _ base.StatementRangeContext, statement string) ([]base.Range, error) {
	omniRanges, err := omniredshift.StatementRanges(statement)
	if err != nil {
		return nil, err
	}
	ranges := make([]base.Range, 0, len(omniRanges))
	for _, r := range omniRanges {
		ranges = append(ranges, base.Range{
			Start: lsp.Position{
				Line:      uint32(r.Start.Line),
				Character: uint32(r.Start.Character),
			},
			End: lsp.Position{
				Line:      uint32(r.End.Line),
				Character: uint32(r.End.Character),
			},
		})
	}
	return ranges, nil
}
