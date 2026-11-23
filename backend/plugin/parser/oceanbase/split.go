package oceanbase

import (
	omniparser "github.com/bytebase/omni/oceanbase/parser"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.RegisterSplitterFunc(storepb.Engine_OCEANBASE, SplitSQL)
}

// SplitSQL splits the given SQL statement into multiple SQL statements.
// It uses omni's lexical splitter which handles quotes, comments, compound
// statements (BEGIN...END, IF...END IF, etc.) and DELIMITER without
// requiring valid SQL.
func SplitSQL(statement string) ([]base.Statement, error) {
	return base.BuildStatements(statement, omniparser.Split(statement), func(s omniparser.Segment) (int, int, bool) {
		return s.ByteStart, s.ByteEnd, s.Empty()
	}, true), nil
}
