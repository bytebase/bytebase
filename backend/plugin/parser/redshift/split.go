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
	return base.BuildStatements(statement, omniredshift.Split(statement), func(s omniredshift.Segment) (int, int, bool) {
		return s.ByteStart, s.ByteEnd, s.Empty()
	}, false), nil
}
