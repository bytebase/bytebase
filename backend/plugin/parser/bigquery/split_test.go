package bigquery

import (
	"testing"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func TestBigQuerySplitSQL(t *testing.T) {
	base.RunSplitTests(t, "test-data/test_split.yaml", base.SplitTestOptions{
		SplitFunc: SplitSQL,
	})
}
