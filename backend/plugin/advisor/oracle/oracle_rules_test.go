// Package oracle is the advisor for oracle database.
package oracle

import (
	"testing"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
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
		advisor.SchemaRuleStatementRequireWhereForSelect,
		advisor.SchemaRuleStatementRequireWhereForUpdateDelete,
		advisor.SchemaRuleStatementInsertMustSpecifyColumn,
		advisor.SchemaRuleIndexKeyNumberLimit,
		advisor.SchemaRuleColumnNotNull,
		advisor.SchemaRuleColumnRequireDefault,
		advisor.SchemaRuleAddNotNullColumnRequireDefault,
		advisor.SchemaRuleColumnMaximumVarcharLength,
		advisor.SchemaRuleTableNameNoKeyword,
		advisor.SchemaRuleIdentifierNoKeyword,
		advisor.SchemaRuleIdentifierCase,
		advisor.SchemaRuleStatementDisallowMixInDDL,
		advisor.SchemaRuleStatementDisallowMixInDML,
		advisor.SchemaRuleTableCommentConvention,
		advisor.SchemaRuleColumnCommentConvention,
	}

	for _, rule := range oracleRules {
		advisor.RunSQLReviewRuleTest(t, rule, storepb.Engine_ORACLE, false, false /* record */)
	}
}
