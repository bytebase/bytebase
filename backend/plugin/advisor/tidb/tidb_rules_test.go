package tidb

import (
	"testing"

	"github.com/bytebase/bytebase/backend/plugin/advisor"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

func TestTiDBRules(t *testing.T) {
	tidbRules := []advisor.SQLReviewRuleType{
		advisor.SchemaRuleTableNaming,
	}

	for _, rule := range tidbRules {
		advisor.RunSQLReviewRuleTest(t, rule, storepb.Engine_TIDB, false /* record */)
	}
}
