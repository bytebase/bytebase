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
	return base.BuildStatements(statement, omnipg.Split(statement), func(s omnipg.Segment) (int, int, bool) {
		return s.ByteStart, s.ByteEnd, s.Empty()
	}, false), nil
}
