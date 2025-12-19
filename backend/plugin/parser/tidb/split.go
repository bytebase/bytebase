package tidb

import (
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/plugin/parser/tokenizer"
)

func init() {
	base.RegisterSplitterFunc(storepb.Engine_TIDB, SplitSQL)
}

// SplitSQL splits the given SQL statement into multiple SQL statements.
func SplitSQL(statement string) ([]base.Statement, error) {
	t := tokenizer.NewTokenizer(statement)
	list, err := t.SplitTiDBMultiSQL()
	if err != nil {
		return nil, err
	}
	return list, nil
}
