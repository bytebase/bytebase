// Package snowflake is the advisor for snowflake database.
package snowflake

import (
	"testing"

	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/db"
)

func TestSnowflakeRules(t *testing.T) {
	snowflakeRules := []advisor.SQLReviewRuleType{
		advisor.SchemaRuleTableNaming,
		advisor.SchemaRuleTableRequirePK,
	}

	for _, rule := range snowflakeRules {
		advisor.RunSQLReviewRuleTest(t, rule, db.Snowflake, true /* record */)
	}
}
