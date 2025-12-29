package trino

import (
	"testing"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func TestTrinoSplitSQL(t *testing.T) {
	base.RunSplitTests(t, "test-data/test_split.yaml", base.SplitTestOptions{
		SplitFunc:       SplitSQL,
		LexerSplitFunc:  splitByTokenizer,
		ParserSplitFunc: splitByParser,
	})
}
