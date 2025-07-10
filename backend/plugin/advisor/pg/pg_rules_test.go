package pg

import (
	"testing"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
)

func TestPostgreSQLRules(t *testing.T) {
	pgRules := []advisor.SQLReviewRuleType{
		advisor.BuiltinRulePriorBackupCheck,
		advisor.SchemaRuleColumnNotNull,
		advisor.SchemaRuleRequiredColumn,
		advisor.SchemaRuleColumnTypeDisallowList,
		advisor.SchemaRuleCommentLength,
		advisor.SchemaRuleIndexKeyNumberLimit,
		advisor.SchemaRuleIndexNoDuplicateColumn,
		advisor.SchemaRuleFullyQualifiedObjectName,
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
		advisor.SchemaRuleStatementNonTransactional,
		advisor.SchemaRuleStatementRequireWhereForSelect,
		advisor.SchemaRuleStatementRequireWhereForUpdateDelete,
		advisor.SchemaRuleCharsetAllowlist,
		advisor.SchemaRuleTableNoFK,
		advisor.SchemaRuleTableRequirePK,
		advisor.SchemaRuleColumnDisallowChangeType,
		advisor.SchemaRuleTableDisallowPartition,
		advisor.SchemaRuleIndexPrimaryKeyTypeAllowlist,
		advisor.SchemaRuleColumnMaximumCharacterLength,
		advisor.SchemaRuleStatementDisallowRemoveTblCascade,
		advisor.SchemaRuleStatementDisallowOnDelCascade,
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
		advisor.SchemaRuleColumnDefaultDisallowVolatile,
		advisor.SchemaRuleStatementDisallowAddColumnWithDefault,
		advisor.SchemaRuleCreateIndexConcurrently,
		advisor.SchemaRuleStatementAddCheckNotValid,
		advisor.SchemaRuleStatementAddFKNotValid,
		advisor.SchemaRuleStatementDisallowAddNotNull,
		advisor.SchemaRuleStatementCreateSpecifySchema,
		advisor.SchemaRuleStatementCheckSetRoleVariable,
		advisor.SchemaRuleStatementMaximumLimitValue,
		advisor.SchemaRuleTableCommentConvention,
		advisor.SchemaRuleColumnCommentConvention,
		advisor.SchemaRuleStatementDisallowMixInDDL,
		advisor.SchemaRuleStatementDisallowMixInDML,
	}

	for _, rule := range pgRules {
		_, needMetaData := advisorNeedMockData[rule]
		advisor.RunSQLReviewRuleTest(t, rule, storepb.Engine_POSTGRES, needMetaData, false /* record */)
	}
}

// Add SQL review type here if you need metadata for test.
var advisorNeedMockData = map[advisor.SQLReviewRuleType]bool{
	advisor.SchemaRuleFullyQualifiedObjectName: true,
	advisor.BuiltinRulePriorBackupCheck:        true,
}
