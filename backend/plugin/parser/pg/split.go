package pg

import (
	omnipg "github.com/bytebase/omni/pg"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.RegisterSplitterFunc(storepb.Engine_POSTGRES, SplitSQL)
	base.RegisterSplitterFunc(storepb.Engine_COCKROACHDB, SplitSQL)
}

// SplitSQL splits the given SQL statement into multiple SQL statements.
// It uses omni's lexical splitter which handles quotes, dollar-quotes,
// comments, and BEGIN ATOMIC blocks without requiring valid SQL.
func SplitSQL(statement string) ([]base.Statement, error) {
	segments := omnipg.Split(statement)

	result := make([]base.Statement, 0, len(segments))
	for _, seg := range segments {
		result = append(result, base.Statement{
			Text:  seg.Text,
			Empty: seg.Empty(),
			Start: byteOffsetToRunePosition(statement, seg.ByteStart),
			End:   byteOffsetToRunePosition(statement, seg.ByteEnd),
			Range: &storepb.Range{
				Start: int32(seg.ByteStart),
				End:   int32(seg.ByteEnd),
			},
		})
	}
	return result, nil
}
