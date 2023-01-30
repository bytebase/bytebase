package mysql

import (
	"testing"

	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/db"

	// Register pingcap parser driver.
	_ "github.com/pingcap/tidb/types/parser_driver"
)

func TestMySQLRules(t *testing.T) {
	mysqlRules := []advisor.SQLReviewRuleType{
		advisor.MySQLSchemaRuleMySQLEngine,
		advisor.MySQLSchemaRuleTableNaming,
		advisor.MySQLSchemaRuleColumnNaming,
		advisor.MySQLSchemaRuleUKNaming,
		advisor.MySQLSchemaRuleFKNaming,
		advisor.MySQLSchemaRuleIDXNaming,
		advisor.MySQLSchemaRuleAutoIncrementColumnNaming,
		advisor.MySQLSchemaRuleStatementNoSelectAll,
		advisor.MySQLSchemaRuleStatementRequireWhere,
		advisor.MySQLSchemaRuleStatementNoLeadingWildcardLike,
		advisor.MySQLSchemaRuleStatementDisallowCommit,
		advisor.MySQLSchemaRuleStatementDisallowLimit,
		advisor.MySQLSchemaRuleStatementDisallowOrderBy,
		advisor.MySQLSchemaRuleStatementMergeAlterTable,
		advisor.MySQLSchemaRuleStatementInsertRowLimit,
		advisor.MySQLSchemaRuleStatementInsertMustSpecifyColumn,
		advisor.MySQLSchemaRuleStatementInsertDisallowOrderByRand,
		advisor.MySQLSchemaRuleStatementAffectedRowLimit,
		advisor.MySQLSchemaRuleStatementDMLDryRun,
		advisor.MySQLSchemaRuleTableRequirePK,
		advisor.MySQLSchemaRuleTableNoFK,
		advisor.MySQLSchemaRuleTableDropNamingConvention,
		advisor.MySQLSchemaRuleTableCommentConvention,
		advisor.MySQLSchemaRuleTableDisallowPartition,
		advisor.MySQLSchemaRuleRequiredColumn,
		advisor.MySQLSchemaRuleColumnNotNull,
		advisor.MySQLSchemaRuleColumnDisallowChangeType,
		advisor.MySQLSchemaRuleColumnSetDefaultForNotNull,
		advisor.MySQLSchemaRuleColumnDisallowChange,
		advisor.MySQLSchemaRuleColumnDisallowChangingOrder,
		advisor.MySQLSchemaRuleColumnCommentConvention,
		advisor.MySQLSchemaRuleColumnAutoIncrementMustInteger,
		advisor.MySQLSchemaRuleColumnTypeDisallowList,
		advisor.MySQLSchemaRuleColumnDisallowSetCharset,
		advisor.MySQLSchemaRuleColumnMaximumCharacterLength,
		advisor.MySQLSchemaRuleColumnAutoIncrementInitialValue,
		advisor.MySQLSchemaRuleColumnAutoIncrementMustUnsigned,
		advisor.MySQLSchemaRuleCurrentTimeColumnCountLimit,
		advisor.MySQLSchemaRuleColumnRequireDefault,
		advisor.MySQLSchemaRuleSchemaBackwardCompatibility,
		advisor.MySQLSchemaRuleDropEmptyDatabase,
		advisor.MySQLSchemaRuleIndexNoDuplicateColumn,
		advisor.MySQLSchemaRuleIndexKeyNumberLimit,
		advisor.MySQLSchemaRuleIndexPKTypeLimit,
		advisor.MySQLSchemaRuleIndexTypeNoBlob,
		advisor.MySQLSchemaRuleIndexTotalNumberLimit,
		advisor.MySQLSchemaRuleCharsetAllowlist,
		advisor.MySQLSchemaRuleCollationAllowlist,
	}

	for _, rule := range mysqlRules {
		advisor.RunSQLReviewRuleTest(t, rule, db.MySQL, false /* record */)
	}
}
