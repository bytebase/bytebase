package pg

import (
	"testing"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
)

func TestPostgreSQLANTLRRules(t *testing.T) {
	antlrRules := []advisor.SQLReviewRuleType{
		advisor.BuiltinRulePriorBackupCheck,
		advisor.SchemaRuleCharsetAllowlist,
		advisor.SchemaRuleCollationAllowlist,
		advisor.SchemaRuleColumnCommentConvention,
		advisor.SchemaRuleColumnDefaultDisallowVolatile,
		advisor.SchemaRuleColumnDisallowChangeType,
		advisor.SchemaRuleColumnMaximumCharacterLength,
		advisor.SchemaRuleColumnNotNull,
		advisor.SchemaRuleColumnRequireDefault,
		advisor.SchemaRuleColumnTypeDisallowList,
		advisor.SchemaRuleCommentLength,
		advisor.SchemaRuleCreateIndexConcurrently,
		advisor.SchemaRuleFKNaming,
		advisor.SchemaRuleFullyQualifiedObjectName,
		advisor.SchemaRuleIDXNaming,
		advisor.SchemaRuleIndexKeyNumberLimit,
		advisor.SchemaRuleIndexNoDuplicateColumn,
		advisor.SchemaRuleIndexPrimaryKeyTypeAllowlist,
		advisor.SchemaRuleIndexTotalNumberLimit,
		advisor.SchemaRulePKNaming,
		advisor.SchemaRuleRequiredColumn,
		advisor.SchemaRuleStatementInsertDisallowOrderByRand,
		advisor.SchemaRuleStatementInsertMustSpecifyColumn,
		advisor.SchemaRuleStatementInsertRowLimit,
		advisor.SchemaRuleStatementMaximumLimitValue,
		advisor.SchemaRuleStatementMergeAlterTable,
		advisor.SchemaRuleStatementNoLeadingWildcardLike,
		advisor.SchemaRuleStatementNoSelectAll,
		advisor.SchemaRuleStatementObjectOwnerCheck,
		advisor.SchemaRuleStatementRequireWhereForSelect,
		advisor.SchemaRuleStatementRequireWhereForUpdateDelete,
		advisor.SchemaRuleColumnNaming,
		advisor.SchemaRuleSchemaBackwardCompatibility,
		advisor.SchemaRuleStatementAddCheckNotValid,
		advisor.SchemaRuleStatementAddFKNotValid,
		advisor.SchemaRuleStatementAffectedRowLimit,
		advisor.SchemaRuleStatementCheckSetRoleVariable,
		advisor.SchemaRuleStatementCreateSpecifySchema,
		advisor.SchemaRuleStatementDisallowAddColumnWithDefault,
		advisor.SchemaRuleStatementDisallowAddNotNull,
		advisor.SchemaRuleStatementDisallowCommit,
		advisor.SchemaRuleStatementDisallowMixInDDL,
		advisor.SchemaRuleStatementDisallowMixInDML,
		advisor.SchemaRuleStatementDisallowOnDelCascade,
		advisor.SchemaRuleStatementDisallowRemoveTblCascade,
		advisor.SchemaRuleStatementDMLDryRun,
		advisor.SchemaRuleStatementNonTransactional,
		advisor.SchemaRuleTableCommentConvention,
		advisor.SchemaRuleTableDisallowPartition,
		advisor.SchemaRuleTableDropNamingConvention,
		advisor.SchemaRuleTableNaming,
		advisor.SchemaRuleTableNoFK,
		advisor.SchemaRuleTableRequirePK,
		advisor.SchemaRuleUKNaming,
	}

	for _, rule := range antlrRules {
		RunANTLRAdvisorRuleTest(t, rule, storepb.Engine_POSTGRES, false /* record */)
	}
}
