// Package snowflake is the advisor for snowflake database.
package snowflake

import (
	"testing"

	"github.com/bytebase/bytebase/backend/plugin/advisor"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

func TestSnowflakeRules(t *testing.T) {
	snowflakeRules := []advisor.SQLReviewRuleType{
		advisor.SchemaRuleTableNaming,
		advisor.SchemaRuleTableRequirePK,
		advisor.SchemaRuleTableNoFK,
		advisor.SchemaRuleColumnMaximumVarcharLength,
		advisor.SchemaRuleTableNameNoKeyword,
		advisor.SchemaRuleStatementRequireWhere,
		advisor.SchemaRuleIdentifierNoKeyword,
		advisor.SchemaRuleRequiredColumn,
		advisor.SchemaRuleIdentifierCase,
		advisor.SchemaRuleColumnNotNull,
		advisor.SchemaRuleStatementNoSelectAll,
		advisor.SchemaRuleTableDropNamingConvention,
		advisor.SchemaRuleSchemaBackwardCompatibility,
	}

	for _, rule := range snowflakeRules {
		advisor.RunSQLReviewRuleTest(t, rule, storepb.Engine_SNOWFLAKE, false /* record */)
	}
}
