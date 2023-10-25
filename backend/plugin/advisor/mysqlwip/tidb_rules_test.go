package mysqlwip

import (
	"testing"

	"github.com/bytebase/bytebase/backend/plugin/advisor"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

func TestMySQLRules(t *testing.T) {
	tidbRules := []advisor.SQLReviewRuleType{
		// advisor.SchemaRuleTableNaming enforce the table name format.
		advisor.SchemaRuleTableNaming,
	}

	for _, rule := range tidbRules {
		advisor.RunSQLReviewRuleTest(t, rule, storepb.Engine_ENGINE_UNSPECIFIED, false /* record */)
	}
}
