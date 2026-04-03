package mysql

import (
	mysqlparser "github.com/bytebase/omni/mysql/parser"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.RegisterSplitterFunc(storepb.Engine_MYSQL, SplitSQL)
	base.RegisterSplitterFunc(storepb.Engine_MARIADB, SplitSQL)
	base.RegisterSplitterFunc(storepb.Engine_OCEANBASE, SplitSQL)
}

// SplitSQL splits the given SQL statement into multiple SQL statements.
// It uses omni's lexical splitter which handles quotes, comments, compound
// statements (BEGIN...END, IF...END IF, etc.) and DELIMITER without
// requiring valid SQL.
func SplitSQL(statement string) ([]base.Statement, error) {
	segments := mysqlparser.Split(statement)

	result := make([]base.Statement, 0, len(segments))
	for _, seg := range segments {
		// omni Segment excludes the trailing semicolon, but downstream
		// code expects it included. Extend the byte range to cover it.
		byteEnd := seg.ByteEnd
		if byteEnd < len(statement) && statement[byteEnd] == ';' {
			byteEnd++
		}
		text := statement[seg.ByteStart:byteEnd]

		result = append(result, base.Statement{
			Text:  text,
			Empty: seg.Empty(),
			Start: ByteOffsetToRunePosition(statement, seg.ByteStart),
			End:   ByteOffsetToRunePosition(statement, byteEnd),
			Range: &storepb.Range{
				Start: int32(seg.ByteStart),
				End:   int32(byteEnd),
			},
		})
	}
	return result, nil
}
