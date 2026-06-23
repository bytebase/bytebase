package cassandra

import (
	"strings"

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
		empty := seg.Empty || isCommentOnly(seg.Text)
		result = append(result, base.Statement{
			Text:  seg.Text,
			Empty: empty,
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

func isCommentOnly(text string) bool {
	s := text
	for len(s) > 0 {
		if s[0] == ' ' || s[0] == '\t' || s[0] == '\n' || s[0] == '\r' {
			s = s[1:]
		} else if strings.HasPrefix(s, "--") {
			if idx := strings.IndexByte(s, '\n'); idx >= 0 {
				s = s[idx+1:]
			} else {
				return true
			}
		} else if strings.HasPrefix(s, "/*") {
			if idx := strings.Index(s, "*/"); idx >= 0 {
				s = s[idx+2:]
			} else {
				return true
			}
		} else {
			return false
		}
	}
	return true
}
