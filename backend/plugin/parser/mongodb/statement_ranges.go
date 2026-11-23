package mongodb

import (
	"context"

	omnianalysis "github.com/bytebase/omni/analysis"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.RegisterStatementRangesFunc(storepb.Engine_MONGODB, GetStatementRanges)
}

// GetStatementRanges returns the ranges of statements in the MongoDB shell script.
// Uses best-effort parsing so that valid statements are returned even when other
// statements in the input have syntax errors (e.g. user is mid-typing).
func GetStatementRanges(_ context.Context, _ base.StatementRangeContext, statement string) ([]base.Range, error) {
	stmts, _ := parseMongoShellBestEffort(statement)
	if len(stmts) == 0 {
		return nil, nil
	}

	byteRanges := make([]omnianalysis.ByteRange, 0, len(stmts))
	for _, stmt := range stmts {
		if stmt.Range == nil {
			continue
		}
		byteRanges = append(byteRanges, omnianalysis.ByteRange{
			Start: int(stmt.Range.Start),
			End:   int(stmt.Range.End),
		})
	}
	return base.MapByteRangesToLSP(statement, byteRanges), nil
}
