package cassandra

import (
	omnicassandra "github.com/bytebase/omni/cassandra"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.RegisterSplitterFunc(storepb.Engine_CASSANDRA, SplitSQL)
}

func SplitSQL(statement string) ([]base.Statement, error) {
	segments := omnicassandra.Split(statement)

	positionMapper := base.NewByteOffsetPositionMapper(statement)
	result := make([]base.Statement, 0, len(segments))
	for _, seg := range segments {
		result = append(result, base.Statement{
			Text:  seg.Text,
			Empty: seg.Empty,
			Start: positionMapper.Position(seg.ByteStart),
			End:   positionMapper.Position(seg.ByteEnd),
			Range: &storepb.Range{
				Start: int32(seg.ByteStart),
				End:   int32(seg.ByteEnd),
			},
		})
	}
	return result, nil
}
