// Package oracle is the advisor for oracle database.
package oracle

import (
	"testing"

	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/db"
)

func TestOracleRules(t *testing.T) {
	oracleRules := []advisor.SQLReviewRuleType{
		advisor.SchemaRuleTableRequirePK,
		advisor.SchemaRuleTableNoFK,
		advisor.SchemaRuleTableNaming,
		advisor.SchemaRuleRequiredColumn,
		advisor.SchemaRuleColumnTypeDisallowList,
		advisor.SchemaRuleColumnMaximumCharacterLength,
		advisor.SchemaRuleStatementNoSelectAll,
		advisor.SchemaRuleStatementNoLeadingWildcardLike,
		advisor.SchemaRuleStatementRequireWhere,
		advisor.SchemaRuleStatementInsertMustSpecifyColumn,
		advisor.SchemaRuleIndexKeyNumberLimit,
	}

	for _, rule := range oracleRules {
		advisor.RunSQLReviewRuleTest(t, rule, db.Oracle, false /* record */)
	}
}
