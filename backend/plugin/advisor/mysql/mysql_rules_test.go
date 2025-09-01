package mysql

import (
	"testing"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
)

func TestMySQLRules(t *testing.T) {
	mysqlRules := []advisor.SQLReviewRuleType{
		// advisor.SchemaRuleMySQLEngine enforce the innodb engine.
		advisor.SchemaRuleMySQLEngine,

		// Naming related rules.
		// advisor.SchemaRuleTableNaming enforce the table name format.
		advisor.SchemaRuleTableNaming,
		// advisor.SchemaRuleColumnNaming enforce the column name format.
		advisor.SchemaRuleColumnNaming,
		// advisor.SchemaRuleUKNaming enforce the unique key name format.
		advisor.SchemaRuleUKNaming,
		// advisor.SchemaRuleFKNaming enforce the foreign key name format.
		advisor.SchemaRuleFKNaming,
		// advisor.SchemaRuleIDXNaming enforce the index name format.
		advisor.SchemaRuleIDXNaming,
		// advisor.SchemaRuleAutoIncrementColumnNaming enforce the auto_increment column name format.
		advisor.SchemaRuleAutoIncrementColumnNaming,
		// advisor.SchemaRuleIdentifierNoKeyword enforce the identifier no keyword.
		advisor.SchemaRuleIdentifierNoKeyword,

		// Statement related rules.
		// advisor.SchemaRuleStatementNoSelectAll disallow 'SELECT *'.
		advisor.SchemaRuleStatementNoSelectAll,
		// advisor.SchemaRuleStatementRequireWhereForSelect require 'WHERE' clause for SELECT statement.
		advisor.SchemaRuleStatementRequireWhereForSelect,
		// advisor.SchemaRuleStatementRequireWhereForUpdateDelete require 'WHERE' clause for UPDATE/DELETE statement.
		advisor.SchemaRuleStatementRequireWhereForUpdateDelete,
		// advisor.SchemaRuleStatementNoLeadingWildcardLike disallow leading '%' in LIKE, e.g. LIKE foo = '%x' is not allowed.
		advisor.SchemaRuleStatementNoLeadingWildcardLike,
		// advisor.SchemaRuleStatementDisallowCommit disallow using commit in the issue.
		advisor.SchemaRuleStatementDisallowCommit,
		// advisor.SchemaRuleStatementDisallowLimit disallow the LIMIT clause in INSERT, DELETE and UPDATE statements.
		advisor.SchemaRuleStatementDisallowLimit,
		// advisor.SchemaRuleStatementDisallowOrderBy disallow the ORDER BY clause in DELETE and UPDATE statements.
		advisor.SchemaRuleStatementDisallowOrderBy,
		// advisor.SchemaRuleStatementMergeAlterTable disallow redundant ALTER TABLE statements.
		advisor.SchemaRuleStatementMergeAlterTable,
		// advisor.SchemaRuleStatementInsertRowLimit enforce the insert row limit.
		advisor.SchemaRuleStatementInsertRowLimit,
		// advisor.SchemaRuleStatementInsertMustSpecifyColumn enforce the insert column specified.
		advisor.SchemaRuleStatementInsertMustSpecifyColumn,
		// advisor.SchemaRuleStatementInsertDisallowOrderByRand disallow the order by rand in the INSERT statement.
		advisor.SchemaRuleStatementInsertDisallowOrderByRand,
		// advisor.SchemaRuleStatementAffectedRowLimit enforce the UPDATE/DELETE affected row limit.
		advisor.SchemaRuleStatementAffectedRowLimit,
		// advisor.SchemaRuleStatementDMLDryRun dry run the dml.
		advisor.SchemaRuleStatementDMLDryRun,
		// advisor.SchemaRuleStatementNoEqualNull disallow the equal null.
		advisor.SchemaRuleStatementWhereNoEqualNull,
		// advisor.SchemaRuleStatementMaximumLimitValue enforce the maximum limit value.
		advisor.SchemaRuleStatementMaximumLimitValue,
		// advisor.SchemaRuleStatementMaximumJoinTableCount enforces maximum of tables in the joins.
		advisor.SchemaRuleStatementMaximumJoinTableCount,
		// advisor.SchemaRuleStatementWhereDisallowUsingFunction disallow using function in where clause.
		advisor.SchemaRuleStatementWhereDisallowFunctionsAndCalculations,
		// advisor.SchemaRuleStatementWhereMaximumLogicalOperatorCount enforces maximum number of logical operators in the where clause.
		advisor.SchemaRuleStatementWhereMaximumLogicalOperatorCount,
		// advisor.SchemaRuleStatementMaxExecutionTime enforce the maximum execution time.
		advisor.SchemaRuleStatementMaxExecutionTime,
		// advisor.SchemaRuleStatementRequireAlgorithmOption require the algorithm option in the alter table statement.
		advisor.SchemaRuleStatementRequireAlgorithmOption,
		// advisor.SchemaRuleStatementRequireLockOption require the lock option in the alter table statement.
		advisor.SchemaRuleStatementRequireLockOption,

		// Database related rules.
		// advisor.SchemaRuleDropEmptyDatabase enforce the MySQL support check if the database is empty before users drop it.
		advisor.SchemaRuleDropEmptyDatabase,

		// Table related rules.
		// advisor.SchemaRuleTableRequirePK require the table to have a primary key.
		advisor.SchemaRuleTableRequirePK,
		// advisor.SchemaRuleTableNoFK require the table disallow the foreign key.
		advisor.SchemaRuleTableNoFK,
		// advisor.SchemaRuleTableDropNamingConvention require only the table following the naming convention can be deleted.
		advisor.SchemaRuleTableDropNamingConvention,
		// advisor.SchemaRuleTableCommentConvention enforce the table comment convention.
		advisor.SchemaRuleTableCommentConvention,
		// advisor.SchemaRuleTableDisallowPartition disallow the table partition.
		advisor.SchemaRuleTableDisallowPartition,
		// advisor.SchemaRuleTableDisallowTrigger disallow the table trigger.
		advisor.SchemaRuleTableDisallowTrigger,
		// advisor.SchemaRuleTableNoDuplicateIndex require the table no duplicate index.
		advisor.SchemaRuleTableNoDuplicateIndex,
		// advisor.SchemaRuleTableDisallowSetCharset disallow set table charset when creating/altering table.
		advisor.SchemaRuleTableDisallowSetCharset,

		// Column related rules.
		// advisor.SchemaRuleRequiredColumn enforce the required columns in each table.
		advisor.SchemaRuleRequiredColumn,
		// advisor.SchemaRuleColumnNotNull enforce the columns cannot have NULL value.
		advisor.SchemaRuleColumnNotNull,
		// advisor.SchemaRuleColumnDisallowChangeType disallow change column type.
		advisor.SchemaRuleColumnDisallowChangeType,
		// advisor.SchemaRuleColumnSetDefaultForNotNull require the not null column to set default value.
		advisor.SchemaRuleColumnSetDefaultForNotNull,
		// advisor.SchemaRuleColumnDisallowChange disallow CHANGE COLUMN statement.
		advisor.SchemaRuleColumnDisallowChange,
		// advisor.SchemaRuleColumnDisallowChangingOrder disallow changing column order.
		advisor.SchemaRuleColumnDisallowChangingOrder,
		// advisor.SchemaRuleColumnDisallowDropIndex disallow drop index column.
		advisor.SchemaRuleColumnDisallowDropInIndex,
		// advisor.SchemaRuleColumnCommentConvention enforce the column comment convention.
		advisor.SchemaRuleColumnCommentConvention,
		// advisor.SchemaRuleColumnAutoIncrementMustInteger require the auto-increment column to be integer.
		advisor.SchemaRuleColumnAutoIncrementMustInteger,
		// advisor.SchemaRuleColumnTypeDisallowList enforce the column type disallow list.
		advisor.SchemaRuleColumnTypeDisallowList,
		// advisor.SchemaRuleColumnDisallowSetCharset disallow set column charset.
		advisor.SchemaRuleColumnDisallowSetCharset,
		// advisor.SchemaRuleColumnMaximumCharacterLength enforce the maximum character length.
		advisor.SchemaRuleColumnMaximumCharacterLength,
		// advisor.SchemaRuleColumnMaximumCharacterLength enforce the maximum varchar length.
		advisor.SchemaRuleColumnMaximumVarcharLength,
		// advisor.SchemaRuleColumnAutoIncrementInitialValue enforce the initial auto-increment value.
		advisor.SchemaRuleColumnAutoIncrementInitialValue,
		// advisor.SchemaRuleColumnAutoIncrementMustUnsigned enforce the auto-increment column to be unsigned.
		advisor.SchemaRuleColumnAutoIncrementMustUnsigned,
		// advisor.SchemaRuleColumnRequireDefault enforce the column default.
		advisor.SchemaRuleColumnRequireDefault,

		// Index related rules.
		// advisor.SchemaRuleIndexNoDuplicateColumn require the index no duplicate column.
		advisor.SchemaRuleIndexNoDuplicateColumn,
		// advisor.SchemaRuleIndexKeyNumberLimit enforce the index key number limit.
		advisor.SchemaRuleIndexKeyNumberLimit,
		// advisor.SchemaRuleIndexPKTypeLimit enforce the type restriction of columns in primary key.
		advisor.SchemaRuleIndexPKTypeLimit,
		// advisor.SchemaRuleIndexTypeNoBlob enforce the type restriction of columns in index.
		advisor.SchemaRuleIndexTypeNoBlob,
		// advisor.SchemaRuleIndexTotalNumberLimit enforce the index total number limit.
		advisor.SchemaRuleIndexTotalNumberLimit,
		// advisor.SchemaRuleIndexPrimaryKeyTypeAllowlist enforce the primary key type allowlist.
		advisor.SchemaRuleIndexPrimaryKeyTypeAllowlist,
		// advisor.SchemaRuleIndexTypeAllowList enforce the index type allowlist.
		advisor.SchemaRuleIndexTypeAllowList,

		// System related rules.
		// advisor.SchemaRuleSchemaBackwardCompatibility enforce the MySQL and TiDB support check whether the schema change is backward compatible.
		advisor.SchemaRuleSchemaBackwardCompatibility,
		// advisor.SchemaRuleCurrentTimeColumnCountLimit enforce the current column count limit.
		advisor.SchemaRuleCurrentTimeColumnCountLimit,
		// advisor.SchemaRuleCharsetAllowlist enforce the charset allowlist.
		advisor.SchemaRuleCharsetAllowlist,
		// advisor.SchemaRuleCollationAllowlist enforce the collation allowlist.
		advisor.SchemaRuleCollationAllowlist,
		// advisor.SchemaRuleDisallowProcedure enforce the disallow create procedure.
		advisor.SchemaRuleProcedureDisallowCreate,
		// advisor.SchemaRuleDisallowEvent enforce the disallow create event.
		advisor.SchemaRuleEventDisallowCreate,
		// advisor.SchemaRuleDisallowView enforce the disallow create view.
		advisor.SchemaRuleViewDisallowCreate,
		// advisor.SchemaRuleFunctionDisallowCreate enforce the disallow create function.
		advisor.SchemaRuleFunctionDisallowCreate,
		// advisor.SchemaRuleFunctionDisallowList enforce the function disallow list.
		advisor.SchemaRuleFunctionDisallowList,
		advisor.SchemaRuleStatementDisallowMixInDDL,
		advisor.SchemaRuleStatementDisallowMixInDML,
	}

	for _, rule := range mysqlRules {
		advisor.RunSQLReviewRuleTest(t, rule, storepb.Engine_MYSQL, false, false /* record */)
	}
}
