package pg

import (
	"testing"

	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/db"
)

func TestPostgreSQLRules(t *testing.T) {
	pgRules := []advisor.SQLReviewRuleType{
		advisor.PostgreSQLSchemaRuleColumnNotNull,
		advisor.PostgreSQLSchemaRuleRequiredColumn,
		advisor.PostgreSQLSchemaRuleColumnTypeDisallowList,
		advisor.PostgreSQLSchemaRuleCommentLength,
		advisor.PostgreSQLSchemaRuleIndexKeyNumberLimit,
		advisor.PostgreSQLSchemaRuleIndexNoDuplicateColumn,
		advisor.PostgreSQLSchemaRuleColumnNaming,
		advisor.PostgreSQLSchemaRuleFKNaming,
		advisor.PostgreSQLSchemaRuleIDXNaming,
		advisor.PostgreSQLSchemaRulePKNaming,
		advisor.PostgreSQLSchemaRuleUKNaming,
		advisor.PostgreSQLSchemaRuleTableNaming,
		advisor.PostgreSQLSchemaRuleSchemaBackwardCompatibility,
		advisor.PostgreSQLSchemaRuleStatementInsertRowLimit,
		advisor.PostgreSQLSchemaRuleStatementNoSelectAll,
		advisor.PostgreSQLSchemaRuleStatementNoLeadingWildcardLike,
		advisor.PostgreSQLSchemaRuleStatementRequireWhere,
		advisor.PostgreSQLSchemaRuleCharsetAllowlist,
		advisor.PostgreSQLSchemaRuleTableNoFK,
		advisor.PostgreSQLSchemaRuleTableRequirePK,
		advisor.PostgreSQLSchemaRuleColumnDisallowChangeType,
		advisor.PostgreSQLSchemaRuleTableDisallowPartition,
		advisor.PostgreSQLSchemaRuleIndexPrimaryKeyTypeAllowlist,
		advisor.PostgreSQLSchemaRuleColumnMaximumCharacterLength,
		advisor.PostgreSQLSchemaRuleStatementDisallowCommit,
		advisor.PostgreSQLSchemaRuleStatementDMLDryRun,
		advisor.PostgreSQLSchemaRuleStatementInsertMustSpecifyColumn,
		advisor.PostgreSQLSchemaRuleStatementInsertDisallowOrderByRand,
		advisor.PostgreSQLSchemaRuleTableDropNamingConvention,
		advisor.PostgreSQLSchemaRuleCollationAllowlist,
		advisor.PostgreSQLSchemaRuleIndexTotalNumberLimit,
		advisor.PostgreSQLSchemaRuleStatementAffectedRowLimit,
		advisor.PostgreSQLSchemaRuleStatementMergeAlterTable,
		advisor.PostgreSQLSchemaRuleColumnRequireDefault,
		advisor.PostgreSQLSchemaRuleStatementDisallowAddColumnWithDefault,
		advisor.PostgreSQLSchemaRuleCreateIndexConcurrently,
		advisor.PostgreSQLSchemaRuleStatementAddCheckNotValid,
		advisor.PostgreSQLSchemaRuleStatementDisallowAddNotNull,
	}

	for _, rule := range pgRules {
		advisor.RunSQLReviewRuleTest(t, rule, db.Postgres, true /* record */)
	}
}
