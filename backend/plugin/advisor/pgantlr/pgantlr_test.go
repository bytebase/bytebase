package pgantlr

import (
	"testing"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
)

func TestPostgreSQLANTLRRules(t *testing.T) {
	antlrRules := []advisor.SQLReviewRuleType{
		HelloWorldRule,                                       // Test advisor to verify framework works
		advisor.BuiltinRulePriorBackupCheck,                  // Migrated from legacy
		advisor.SchemaRuleCharsetAllowlist,                   // Migrated from legacy
		advisor.SchemaRuleCollationAllowlist,                 // Migrated from legacy
		advisor.SchemaRuleColumnCommentConvention,            // Migrated from legacy
		advisor.SchemaRuleColumnDefaultDisallowVolatile,      // Migrated from legacy
		advisor.SchemaRuleColumnDisallowChangeType,           // Migrated from legacy
		advisor.SchemaRuleColumnMaximumCharacterLength,       // Migrated from legacy
		advisor.SchemaRuleColumnNotNull,                      // Migrated from legacy
		advisor.SchemaRuleColumnRequireDefault,               // Migrated from legacy
		advisor.SchemaRuleColumnTypeDisallowList,             // Migrated from legacy
		advisor.SchemaRuleCommentLength,                      // Migrated from legacy
		advisor.SchemaRuleCreateIndexConcurrently,            // Migrated from legacy
		advisor.SchemaRuleIndexKeyNumberLimit,                // Migrated from legacy
		advisor.SchemaRuleIndexNoDuplicateColumn,             // Migrated from legacy
		advisor.SchemaRuleIndexPrimaryKeyTypeAllowlist,       // Migrated from legacy
		advisor.SchemaRuleIndexTotalNumberLimit,              // Migrated from legacy
		advisor.SchemaRuleRequiredColumn,                     // Migrated from legacy
		advisor.SchemaRuleStatementInsertDisallowOrderByRand, // Migrated from legacy
		advisor.SchemaRuleStatementInsertMustSpecifyColumn,   // Migrated from legacy
		advisor.SchemaRuleStatementInsertRowLimit,            // Migrated from legacy
		advisor.SchemaRuleColumnNaming,                       // Migrated from legacy
		// Add real rules here as you migrate them from legacy pg/ folder
		// Example:
		// advisor.SchemaRuleStatementDisallowCommit,
		// advisor.SchemaRuleStatementInsertMustSpecifyColumn,
		// advisor.SchemaRuleTableNaming,
		// etc.
	}

	for _, rule := range antlrRules {
		needMetaData := advisorNeedMockData[rule]
		RunANTLRAdvisorRuleTest(t, rule, storepb.Engine_POSTGRES, needMetaData, false /* record */)
	}
}

// Add SQL review type here if you need metadata for test.
var advisorNeedMockData = map[advisor.SQLReviewRuleType]bool{
	// advisor.SchemaRuleFullyQualifiedObjectName: true,
	advisor.BuiltinRulePriorBackupCheck:     true,
	advisor.SchemaRuleColumnNotNull:         true, // Needs metadata for PRIMARY KEY USING INDEX case
	advisor.SchemaRuleIndexTotalNumberLimit: true, // Needs catalog to count indexes
	// Other advisors don't need mock data
}
