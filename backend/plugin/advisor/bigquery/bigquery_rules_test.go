package bigquery

import (
	"testing"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
)

func TestBigQueryRules(t *testing.T) {
	rules := []*storepb.SQLReviewRule{
		{Type: storepb.SQLReviewRule_STATEMENT_WHERE_REQUIRE_UPDATE_DELETE, Level: storepb.SQLReviewRule_WARNING},
	}

	for _, rule := range rules {
		advisor.RunSQLReviewRuleTest(t, rule, storepb.Engine_BIGQUERY, false /* record */)
	}
}
