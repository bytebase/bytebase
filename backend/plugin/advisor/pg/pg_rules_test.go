package pg

import (
	"testing"

	"github.com/bytebase/bytebase/backend/plugin/advisor"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

func TestPostgreSQLRules(t *testing.T) {
	pgRules := []advisor.SQLReviewRuleType{
		advisor.SchemaRuleColumnNotNull,
		advisor.SchemaRuleRequiredColumn,
		advisor.SchemaRuleColumnTypeDisallowList,
		advisor.SchemaRuleCommentLength,
		advisor.SchemaRuleIndexKeyNumberLimit,
		advisor.SchemaRuleIndexNoDuplicateColumn,
		advisor.SchemaRuleFullyQualifiedObjectName,
		advisor.SchemaRuleColumnNaming,
		advisor.SchemaRuleFKNaming,
		advisor.SchemaRuleIDXNaming,
		advisor.SchemaRulePKNaming,
		advisor.SchemaRuleUKNaming,
		advisor.SchemaRuleTableNaming,
		advisor.SchemaRuleSchemaBackwardCompatibility,
		advisor.SchemaRuleStatementInsertRowLimit,
		advisor.SchemaRuleStatementNoSelectAll,
		advisor.SchemaRuleStatementNoLeadingWildcardLike,
		advisor.SchemaRuleStatementNonTransactional,
		advisor.SchemaRuleStatementRequireWhere,
		advisor.SchemaRuleCharsetAllowlist,
		advisor.SchemaRuleTableNoFK,
		advisor.SchemaRuleTableRequirePK,
		advisor.SchemaRuleColumnDisallowChangeType,
		advisor.SchemaRuleTableDisallowPartition,
		advisor.SchemaRuleIndexPrimaryKeyTypeAllowlist,
		advisor.SchemaRuleColumnMaximumCharacterLength,
		advisor.SchemaRuleStatementDisallowRemoveTblCascade,
		advisor.SchemaRuleStatementDisallowOnDelCascade,
		advisor.SchemaRuleStatementDisallowCommit,
		advisor.SchemaRuleStatementDMLDryRun,
		advisor.SchemaRuleStatementInsertMustSpecifyColumn,
		advisor.SchemaRuleStatementInsertDisallowOrderByRand,
		advisor.SchemaRuleTableDropNamingConvention,
		advisor.SchemaRuleCollationAllowlist,
		advisor.SchemaRuleIndexTotalNumberLimit,
		advisor.SchemaRuleStatementAffectedRowLimit,
		advisor.SchemaRuleStatementMergeAlterTable,
		advisor.SchemaRuleColumnRequireDefault,
		advisor.SchemaRuleStatementDisallowAddColumnWithDefault,
		advisor.SchemaRuleCreateIndexConcurrently,
		advisor.SchemaRuleStatementAddCheckNotValid,
		advisor.SchemaRuleStatementAddFKNotValid,
		advisor.SchemaRuleStatementDisallowAddNotNull,
		advisor.SchemaRuleStatementCreateSpecifySchema,
		advisor.SchemaRuleStatementCheckSetRoleVariable,
		advisor.SchemaRuleStatementMaximumLimitValue,
	}

	for _, rule := range pgRules {
		var dbSchemaMetadata *storepb.DatabaseSchemaMetadata
		if _, needMockData := advisorNeedMockData[rule]; needMockData {
			dbSchemaMetadata = getMockDBSChemaData()
		}
		advisor.RunSQLReviewRuleTest(t, rule, storepb.Engine_POSTGRES, dbSchemaMetadata, false /* record */)
	}
}

// Add SQL review type here if you need metadata for test.
var advisorNeedMockData = map[advisor.SQLReviewRuleType]bool{
	advisor.SchemaRuleFullyQualifiedObjectName: true,
}

func getMockDBSChemaData() *storepb.DatabaseSchemaMetadata {
	return &storepb.DatabaseSchemaMetadata{
		Name: "TEST_DB",
		Schemas: []*storepb.SchemaMetadata{
			{
				Name: "public",
				Tables: []*storepb.TableMetadata{
					{Name: "pokes"},
					{Name: "pokesv1"},
					{Name: "pokesv2"},
					{Name: "pokesv3"},
				},
				ExternalTables: []*storepb.ExternalTableMetadata{
					{Name: "pokesv4"},
					{Name: "pokesv5"},
					{Name: "pokesv6"},
					{Name: "pokesv7"},
				},
			},
		},
	}
}
