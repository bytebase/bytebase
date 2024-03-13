package mysql

import (
	"testing"

	"github.com/bytebase/bytebase/backend/plugin/advisor"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

func TestMySQLRules(t *testing.T) {
	mysqlRules := []advisor.SQLReviewRuleType{
		// advisor.SchemaRuleStatementWhereMaximumLogicalOperatorCount enforces maximum number of logical operators in the where clause.
		advisor.SchemaRuleStatementWhereMaximumLogicalOperatorCount,
	}

	for _, rule := range mysqlRules {
		advisor.RunSQLReviewRuleTest(t, rule, storepb.Engine_MYSQL, false /* record */)
	}
}
