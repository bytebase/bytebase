package plsql

import (
	"testing"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func TestPLSQLSplitSQL(t *testing.T) {
	base.RunSplitTests(t, "test-data/test_split.yaml", base.SplitTestOptions{
		SplitFunc: SplitSQL,
	})
}
