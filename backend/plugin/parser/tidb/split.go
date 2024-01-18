package tidb

import (
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/plugin/parser/tokenizer"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

func init() {
	base.RegisterSplitterFunc(storepb.Engine_TIDB, SplitSQL)
}

// SplitSQL splits the given SQL statement into multiple SQL statements.
func SplitSQL(statement string) ([]base.SingleSQL, error) {
	t := tokenizer.NewTokenizer(statement)
	list, err := t.SplitTiDBMultiSQL()
	if err != nil {
		return nil, err
	}
	return list, nil
}

// SplitSQLKeepEmptyBlocks splits the given SQL statement into multiple SQL statements.
// TODO: remove SplitSQL, and rename this to SplitSQL.
func SplitSQLKeepEmptyBlocks(statement string) ([]base.SingleSQL, error) {
	t := tokenizer.NewTokenizer(statement, tokenizer.KeepEmptyBlocks())
	list, err := t.SplitTiDBMultiSQL()
	if err != nil {
		return nil, err
	}
	return list, nil
}
