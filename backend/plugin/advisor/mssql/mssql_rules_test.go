// Package mssql is the advisor for MSSQL database.
package mssql

import (
	"testing"

	"github.com/bytebase/bytebase/backend/plugin/advisor"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

func TestMSSQLRules(t *testing.T) {
	mssqlRules := []advisor.SQLReviewRuleType{
		advisor.SchemaRuleStatementNoSelectAll,
		advisor.SchemaRuleTableNaming,
		advisor.SchemaRuleTableNameNoKeyword,
		advisor.SchemaRuleIdentifierNoKeyword,
		advisor.SchemaRuleStatementRequireWhereForSelect,
		advisor.SchemaRuleStatementRequireWhereForUpdateDelete,
		advisor.SchemaRuleColumnMaximumVarcharLength,
		advisor.SchemaRuleTableDropNamingConvention,
		advisor.SchemaRuleTableRequirePK,
		advisor.SchemaRuleColumnNotNull,
		advisor.SchemaRuleTableNoFK,
		advisor.SchemaRuleTableDisallowDDL,
		advisor.SchemaRuleTableDisallowDML,
		advisor.SchemaRuleSchemaBackwardCompatibility,
		advisor.SchemaRuleRequiredColumn,
		advisor.SchemaRuleColumnTypeDisallowList,
		advisor.SchemaRuleFunctionDisallowCreate,
		advisor.SchemaRuleProcedureDisallowCreate,
		advisor.SchemaRuleStatementDisallowCrossDBQueries,
		advisor.SchemaRuleStatementWhereDisallowFunctionsAndCaculations,
		advisor.SchemaRuleIndexNotRedundant,
		advisor.SchemaRuleStatementDisallowMixDDLDML,
	}

	for _, rule := range mssqlRules {
		_, needMockData := advisorNeedMockData[rule]
		advisor.RunSQLReviewRuleTest(t, rule, storepb.Engine_MSSQL, needMockData, false /* record */)
	}
}

// Add SQL review type here if you need metadata for test.
var advisorNeedMockData = map[advisor.SQLReviewRuleType]bool{
	advisor.SchemaRuleStatementDisallowCrossDBQueries: true,
	advisor.SchemaRuleSchemaBackwardCompatibility:     true,
	advisor.SchemaRuleIndexNotRedundant:               true,
}
