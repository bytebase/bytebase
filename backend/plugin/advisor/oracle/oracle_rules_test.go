package oracle

import (
	"testing"

	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/db"
)

func TestOracleRules(t *testing.T) {
	oracleRules := []advisor.SQLReviewRuleType{
		advisor.SchemaRuleTableRequirePK,
	}

	for _, rule := range oracleRules {
		advisor.RunSQLReviewRuleTest(t, rule, db.Oracle, false /* record */)
	}
}
