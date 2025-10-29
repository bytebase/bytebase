package pgantlr

import (
	"testing"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
)

func TestPostgreSQLANTLRRules(t *testing.T) {
	antlrRules := []advisor.SQLReviewRuleType{
		HelloWorldRule,                                          // Test advisor to verify framework works
		advisor.BuiltinRulePriorBackupCheck,                     // Migrated from legacy
		advisor.SchemaRuleCharsetAllowlist,                      // Migrated from legacy
		advisor.SchemaRuleCollationAllowlist,                    // Migrated from legacy
		advisor.SchemaRuleColumnCommentConvention,               // Migrated from legacy
		advisor.SchemaRuleColumnDefaultDisallowVolatile,         // Migrated from legacy
		advisor.SchemaRuleColumnDisallowChangeType,              // Migrated from legacy
		advisor.SchemaRuleColumnMaximumCharacterLength,          // Migrated from legacy
		advisor.SchemaRuleColumnNotNull,                         // Migrated from legacy
		advisor.SchemaRuleColumnRequireDefault,                  // Migrated from legacy
		advisor.SchemaRuleColumnTypeDisallowList,                // Migrated from legacy
		advisor.SchemaRuleCommentLength,                         // Migrated from legacy
		advisor.SchemaRuleCreateIndexConcurrently,               // Migrated from legacy
		advisor.SchemaRuleFKNaming,                              // Migrated from legacy
		advisor.SchemaRuleFullyQualifiedObjectName,              // Migrated from legacy
		advisor.SchemaRuleIDXNaming,                             // Migrated from legacy
		advisor.SchemaRuleIndexKeyNumberLimit,                   // Migrated from legacy
		advisor.SchemaRuleIndexNoDuplicateColumn,                // Migrated from legacy
		advisor.SchemaRuleIndexPrimaryKeyTypeAllowlist,          // Migrated from legacy
		advisor.SchemaRuleIndexTotalNumberLimit,                 // Migrated from legacy
		advisor.SchemaRulePKNaming,                              // Migrated from legacy
		advisor.SchemaRuleRequiredColumn,                        // Migrated from legacy
		advisor.SchemaRuleStatementInsertDisallowOrderByRand,    // Migrated from legacy
		advisor.SchemaRuleStatementInsertMustSpecifyColumn,      // Migrated from legacy
		advisor.SchemaRuleStatementInsertRowLimit,               // Migrated from legacy
		advisor.SchemaRuleStatementMaximumLimitValue,            // Migrated from legacy
		advisor.SchemaRuleStatementMergeAlterTable,              // Migrated from legacy
		advisor.SchemaRuleStatementNoLeadingWildcardLike,        // Migrated from legacy
		advisor.SchemaRuleStatementNoSelectAll,                  // Migrated from legacy
		advisor.SchemaRuleStatementObjectOwnerCheck,             // Migrated from legacy
		advisor.SchemaRuleStatementRequireWhereForSelect,        // Migrated from legacy
		advisor.SchemaRuleStatementRequireWhereForUpdateDelete,  // Migrated from legacy
		advisor.SchemaRuleColumnNaming,                          // Migrated from legacy
		advisor.SchemaRuleSchemaBackwardCompatibility,           // Migrated from legacy
		advisor.SchemaRuleStatementAddCheckNotValid,             // Migrated from legacy
		advisor.SchemaRuleStatementAddFKNotValid,                // Migrated from legacy
		advisor.SchemaRuleStatementAffectedRowLimit,             // Migrated from legacy
		advisor.SchemaRuleStatementCheckSetRoleVariable,         // Migrated from legacy
		advisor.SchemaRuleStatementCreateSpecifySchema,          // Migrated from legacy
		advisor.SchemaRuleStatementDisallowAddColumnWithDefault, // Migrated from legacy
		advisor.SchemaRuleStatementDisallowAddNotNull,           // Migrated from legacy
		advisor.SchemaRuleStatementDisallowCommit,               // Migrated from legacy
		advisor.SchemaRuleStatementDisallowMixInDDL,             // Migrated from legacy
		advisor.SchemaRuleStatementDisallowMixInDML,             // Migrated from legacy
		advisor.SchemaRuleStatementDisallowOnDelCascade,         // Migrated from legacy
		advisor.SchemaRuleStatementDisallowRemoveTblCascade,     // Migrated from legacy
		advisor.SchemaRuleStatementDMLDryRun,                    // Migrated from legacy
		advisor.SchemaRuleStatementNonTransactional,             // Migrated from legacy
		advisor.SchemaRuleTableCommentConvention,                // Migrated from legacy
		advisor.SchemaRuleTableDisallowPartition,                // Migrated from legacy
		advisor.SchemaRuleTableDropNamingConvention,             // Migrated from legacy
		advisor.SchemaRuleTableNaming,                           // Migrated from legacy
		advisor.SchemaRuleTableNoFK,                             // Migrated from legacy
		advisor.SchemaRuleUKNaming,                              // Migrated from legacy
	}

	for _, rule := range antlrRules {
		needMetaData := advisorNeedMockData[rule]
		RunANTLRAdvisorRuleTest(t, rule, storepb.Engine_POSTGRES, needMetaData, false /* record */)
	}
}

// Add SQL review type here if you need metadata for test.
var advisorNeedMockData = map[advisor.SQLReviewRuleType]bool{
	advisor.BuiltinRulePriorBackupCheck:         true,
	advisor.SchemaRuleColumnNotNull:             true, // Needs metadata for PRIMARY KEY USING INDEX case
	advisor.SchemaRuleFullyQualifiedObjectName:  true, // Needs metadata for SELECT statement checks
	advisor.SchemaRuleIDXNaming:                 true, // Needs catalog for ALTER INDEX RENAME
	advisor.SchemaRuleIndexTotalNumberLimit:     true, // Needs catalog to count indexes
	advisor.SchemaRulePKNaming:                  true, // Needs catalog for PRIMARY KEY USING INDEX
	advisor.SchemaRuleStatementObjectOwnerCheck: true, // Needs catalog for ownership checks
	advisor.SchemaRuleTableRequirePK:            true, // Needs catalog for DROP CONSTRAINT/COLUMN checks
	advisor.SchemaRuleUKNaming:                  true, // Needs catalog for UNIQUE USING INDEX
}
