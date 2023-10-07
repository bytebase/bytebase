package tidb

import (
	"io"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/plugin/parser/tokenizer"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

func init() {
	base.RegisterSplitterFunc(storepb.Engine_TIDB, SplitSQL)
}

// SplitMultiSQLStream splits  multiSQL to stream.
func SplitMultiSQLStream(src io.Reader, f func(string) error) ([]base.SingleSQL, error) {
	t := tokenizer.NewStreamTokenizer(src, f)
	list, err := t.SplitTiDBMultiSQL()
	if err != nil {
		return nil, err
	}
	var results []base.SingleSQL
	for _, sql := range list {
		if sql.Empty {
			continue
		}
		results = append(results, sql)
	}
	return results, nil
}

// SplitSQL splits the given SQL statement into multiple SQL statements.
func SplitSQL(statement string) ([]base.SingleSQL, error) {
	t := tokenizer.NewTokenizer(statement)
	list, err := t.SplitTiDBMultiSQL()
	if err != nil {
		return nil, err
	}
	var results []base.SingleSQL
	for _, sql := range list {
		if sql.Empty {
			continue
		}
		results = append(results, sql)
	}
	return results, nil
}
