package pg

import (
	"testing"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func TestPGSplitSQL(t *testing.T) {
	base.RunSplitTests(t, "test-data/test_split.yaml", base.SplitTestOptions{
		SplitFunc: SplitSQL,
	})
}
