package pg

import (
	"testing"

	"github.com/bytebase/bytebase/plugin/advisor"
	"github.com/bytebase/bytebase/plugin/advisor/db"
)

func TestPostgreSQLRules(t *testing.T) {
	pgRules := []advisor.SQLReviewRuleType{
		advisor.SchemaRuleColumnNotNull,
		advisor.SchemaRuleRequiredColumn,
		advisor.SchemaRuleColumnTypeDisallowList,
		advisor.SchemaRuleCommentLength,
		advisor.SchemaRuleIndexKeyNumberLimit,
		advisor.SchemaRuleIndexNoDuplicateColumn,
		advisor.SchemaRuleColumnNaming,
		advisor.SchemaRuleFKNaming,
		advisor.SchemaRuleIDXNaming,
		advisor.SchemaRulePKNaming,
		advisor.SchemaRuleUKNaming,
		advisor.SchemaRuleTableNaming,
		advisor.SchemaRuleSchemaBackwardCompatibility,
		advisor.SchemaRuleStatementInsertRowLimit,
		advisor.SchemaRuleStatementNoSelectAll,
		advisor.SchemaRuleStatementNoLeadingWildcardLike,
		advisor.SchemaRuleStatementRequireWhere,
		advisor.SchemaRuleCharsetAllowlist,
		advisor.SchemaRuleTableNoFK,
		advisor.SchemaRuleTableRequirePK,
		advisor.SchemaRuleColumnDisallowChangeType,
		advisor.SchemaRuleTableDisallowPartition,
	}

	for _, rule := range pgRules {
		advisor.RunSQLReviewRuleTest(t, rule, db.Postgres, false /* record */)
	}
}
