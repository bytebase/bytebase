package redshift

import (
	omniredshift "github.com/bytebase/omni/redshift"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.RegisterSplitterFunc(storepb.Engine_REDSHIFT, SplitSQL)
}

// SplitSQL splits the given SQL statement into multiple SQL statements.
func SplitSQL(statement string) ([]base.Statement, error) {
	segments := omniredshift.Split(statement)
	list := make([]base.Statement, 0, len(segments))
	positionMapper := base.NewByteOffsetPositionMapper(statement)
	for _, segment := range segments {
		list = append(list, base.Statement{
			Text:  segment.Text,
			Empty: segment.Empty(),
			Start: positionMapper.Position(segment.ByteStart),
			End:   positionMapper.Position(segment.ByteEnd),
			Range: &storepb.Range{
				Start: int32(segment.ByteStart),
				End:   int32(segment.ByteEnd),
			},
		})
	}
	return list, nil
}
