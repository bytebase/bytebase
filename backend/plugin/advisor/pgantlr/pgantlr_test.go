package pgantlr

import (
	"testing"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
)

func TestPostgreSQLANTLRRules(t *testing.T) {
	antlrRules := []advisor.SQLReviewRuleType{
		HelloWorldRule, // Test advisor to verify framework works
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
	// advisor.BuiltinRulePriorBackupCheck:        true,
}
