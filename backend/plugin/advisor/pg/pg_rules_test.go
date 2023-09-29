package pg

import (
	"testing"

	"github.com/bytebase/bytebase/backend/plugin/advisor"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
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
		advisor.SchemaRuleIndexPrimaryKeyTypeAllowlist,
		advisor.SchemaRuleColumnMaximumCharacterLength,
		advisor.SchemaRuleStatementDisallowCommit,
		advisor.SchemaRuleStatementDMLDryRun,
		advisor.SchemaRuleStatementInsertMustSpecifyColumn,
		advisor.SchemaRuleStatementInsertDisallowOrderByRand,
		advisor.SchemaRuleTableDropNamingConvention,
		advisor.SchemaRuleCollationAllowlist,
		advisor.SchemaRuleIndexTotalNumberLimit,
		advisor.SchemaRuleStatementAffectedRowLimit,
		advisor.SchemaRuleStatementMergeAlterTable,
		advisor.SchemaRuleColumnRequireDefault,
		advisor.SchemaRuleStatementDisallowAddColumnWithDefault,
		advisor.SchemaRuleCreateIndexConcurrently,
		advisor.SchemaRuleStatementAddCheckNotValid,
		advisor.SchemaRuleStatementDisallowAddNotNull,
	}

	for _, rule := range pgRules {
		advisor.RunSQLReviewRuleTest(t, rule, storepb.Engine_POSTGRES, false /* record */)
	}
}
