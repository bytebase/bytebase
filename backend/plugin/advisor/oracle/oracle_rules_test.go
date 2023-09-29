// Package oracle is the advisor for oracle database.
package oracle

import (
	"testing"

	"github.com/bytebase/bytebase/backend/plugin/advisor"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
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
		advisor.SchemaRuleColumnNotNull,
		advisor.SchemaRuleColumnRequireDefault,
		advisor.SchemaRuleAddNotNullColumnRequireDefault,
		advisor.SchemaRuleColumnMaximumVarcharLength,
		advisor.SchemaRuleTableNameNoKeyword,
		advisor.SchemaRuleIdentifierNoKeyword,
		advisor.SchemaRuleIdentifierCase,
	}

	for _, rule := range oracleRules {
		advisor.RunSQLReviewRuleTest(t, rule, storepb.Engine_ORACLE, false /* record */)
	}
}
