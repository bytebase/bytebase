package mariadb

import (
	mariadbparser "github.com/bytebase/omni/mariadb/parser"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.RegisterSplitterFunc(storepb.Engine_MARIADB, SplitSQL)
}

// SplitSQL splits the given SQL statement into multiple SQL statements.
// It uses omni's lexical splitter which handles quotes, comments, compound
// statements (BEGIN...END, IF...END IF, etc.) and DELIMITER without
// requiring valid SQL.
func SplitSQL(statement string) ([]base.Statement, error) {
	segments := mariadbparser.Split(statement)

	result := make([]base.Statement, 0, len(segments))
	positionMapper := base.NewByteOffsetPositionMapper(statement)
	for _, seg := range segments {
		result = append(result, base.NewStatementFromRange(statement, positionMapper, seg.ByteStart, seg.ByteEnd, seg.Empty()))
	}
	return result, nil
}
