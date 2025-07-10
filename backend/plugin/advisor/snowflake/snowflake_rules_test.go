// Package snowflake is the advisor for snowflake database.
package snowflake

import (
	"testing"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
)

func TestSnowflakeRules(t *testing.T) {
	snowflakeRules := []advisor.SQLReviewRuleType{
		advisor.SchemaRuleTableNaming,
		advisor.SchemaRuleTableRequirePK,
		advisor.SchemaRuleTableNoFK,
		advisor.SchemaRuleColumnMaximumVarcharLength,
		advisor.SchemaRuleTableNameNoKeyword,
		advisor.SchemaRuleStatementRequireWhereForSelect,
		advisor.SchemaRuleStatementRequireWhereForUpdateDelete,
		advisor.SchemaRuleIdentifierNoKeyword,
		advisor.SchemaRuleRequiredColumn,
		advisor.SchemaRuleIdentifierCase,
		advisor.SchemaRuleColumnNotNull,
		advisor.SchemaRuleStatementNoSelectAll,
		advisor.SchemaRuleTableDropNamingConvention,
		advisor.SchemaRuleSchemaBackwardCompatibility,
	}

	for _, rule := range snowflakeRules {
		advisor.RunSQLReviewRuleTest(t, rule, storepb.Engine_SNOWFLAKE, false, false /* record */)
	}
}
