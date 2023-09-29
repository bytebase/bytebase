// Package mssql is the advisor for MSSQL database.
package mssql

import (
	"testing"

	"github.com/bytebase/bytebase/backend/plugin/advisor"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

func TestMSSQLRules(t *testing.T) {
	snowflakeRules := []advisor.SQLReviewRuleType{
		advisor.SchemaRuleStatementNoSelectAll,
		advisor.SchemaRuleTableNaming,
		advisor.SchemaRuleTableNameNoKeyword,
		advisor.SchemaRuleIdentifierNoKeyword,
		advisor.SchemaRuleStatementRequireWhere,
		advisor.SchemaRuleColumnMaximumVarcharLength,
		advisor.SchemaRuleTableDropNamingConvention,
		advisor.SchemaRuleTableRequirePK,
		advisor.SchemaRuleColumnNotNull,
		advisor.SchemaRuleTableNoFK,
		advisor.SchemaRuleSchemaBackwardCompatibility,
		advisor.SchemaRuleRequiredColumn,
	}

	for _, rule := range snowflakeRules {
		advisor.RunSQLReviewRuleTest(t, rule, storepb.Engine_MSSQL, false /* record */)
	}
}
