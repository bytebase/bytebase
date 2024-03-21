// Package advisor defines the interface for analyzing sql statements.
// The advisor could be syntax checker, index suggestion etc.
package advisor

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"sync"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/plugin/advisor/catalog"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

// Status is the advisor result status.
type Status string

const (
	// Success is the advisor status for successes.
	Success Status = "SUCCESS"
	// Warn is the advisor status for warnings.
	Warn Status = "WARN"
	// Error is the advisor status for errors.
	Error Status = "ERROR"

	// SyntaxErrorTitle is the error title for syntax error.
	SyntaxErrorTitle string = "Syntax error"
)

// NewStatusBySQLReviewRuleLevel returns status by SQLReviewRuleLevel.
func NewStatusBySQLReviewRuleLevel(level storepb.SQLReviewRuleLevel) (Status, error) {
	switch level {
	case storepb.SQLReviewRuleLevel_ERROR:
		return Error, nil
	case storepb.SQLReviewRuleLevel_WARNING:
		return Warn, nil
	}
	return "", errors.Errorf("unexpected rule level type: %s", level)
}

// GetPriority returns the priority of status.
func (s Status) GetPriority() int {
	switch s {
	case Success:
		return 0
	case Warn:
		return 1
	case Error:
		return 2
	}
	return 0
}

// Type is the type of advisor.
// nolint
type Type string

const (
	// Fake is a fake advisor type for testing.
	Fake Type = "bb.plugin.advisor.fake"

	// MySQL Advisor.

	// MySQLSyntax is an advisor type for MySQL syntax.
	MySQLSyntax Type = "bb.plugin.advisor.mysql.syntax"

	// MySQLUseInnoDB is an advisor type for MySQL InnoDB Engine.
	MySQLUseInnoDB Type = "bb.plugin.advisor.mysql.use-innodb"

	// MySQLOnlineMigration is an advisor type for MySQL using online migration to migrate large tables.
	MySQLOnlineMigration Type = "bb.plugin.advisor.mysql.online-migration"

	// MySQLMigrationCompatibility is an advisor type for MySQL migration compatibility.
	MySQLMigrationCompatibility Type = "bb.plugin.advisor.mysql.migration-compatibility"

	// MySQLWhereRequirement is an advisor type for MySQL WHERE clause requirement.
	MySQLWhereRequirement Type = "bb.plugin.advisor.mysql.where.require"

	// MySQLNoLeadingWildcardLike is an advisor type for MySQL no leading wildcard LIKE.
	MySQLNoLeadingWildcardLike Type = "bb.plugin.advisor.mysql.where.no-leading-wildcard-like"

	// MySQLNamingTableConvention is an advisor type for MySQL table naming convention.
	MySQLNamingTableConvention Type = "bb.plugin.advisor.mysql.naming.table"

	// MySQLNamingIndexConvention is an advisor type for MySQL index key naming convention.
	MySQLNamingIndexConvention Type = "bb.plugin.advisor.mysql.naming.index"

	// MySQLNamingUKConvention is an advisor type for MySQL unique key naming convention.
	MySQLNamingUKConvention Type = "bb.plugin.advisor.mysql.naming.uk"

	// MySQLNamingFKConvention is an advisor type for MySQL foreign key naming convention.
	MySQLNamingFKConvention Type = "bb.plugin.advisor.mysql.naming.fk"

	// MySQLNamingColumnConvention is an advisor type for MySQL column naming convention.
	MySQLNamingColumnConvention Type = "bb.plugin.advisor.mysql.naming.column"

	// MySQLNamingAutoIncrementColumnConvention is an advisor type for MySQL auto-increment naming convention.
	MySQLNamingAutoIncrementColumnConvention Type = "bb.plugin.advisor.mysql.naming.auto-increment-column"

	// MySQLIdentifierNamingNoKeyword is an advisor type for MySQL identifier naming convention without keyword.
	MySQLIdentifierNamingNoKeyword Type = "bb.plugin.advisor.mysql.naming.identifier-no-keyword"

	// MySQLColumnRequirement is an advisor type for MySQL column requirement.
	MySQLColumnRequirement Type = "bb.plugin.advisor.mysql.column.require"

	// MySQLColumnNoNull is an advisor type for MySQL column no NULL value.
	MySQLColumnNoNull Type = "bb.plugin.advisor.mysql.column.no-null"

	// MySQLColumnDisallowChangingType is an advisor type for MySQL disallow changing column type.
	MySQLColumnDisallowChangingType Type = "bb.plugin.advisor.mysql.column.disallow-changing-type"

	// MySQLColumnSetDefaultForNotNull is an advisor type for MySQL set default value for not null column.
	MySQLColumnSetDefaultForNotNull Type = "bb.plugin.advisor.mysql.column.set-default-for-not-null"

	// MySQLColumnDisallowChanging is an advisor type for MySQL disallow CHANGE COLUMN statement.
	MySQLColumnDisallowChanging Type = "bb.plugin.advisor.mysql.column.disallow-change"

	// MySQLColumnDisallowDropInIndex is an advisor type for MySQL disallow DROP COLUMN in index.
	MySQLColumnDisallowDropInIndex Type = "bb.plugin.advisor.mysql.column.disallow-drop-in-index"

	// MySQLColumnDisallowChangingOrder is an advisor type for MySQL disallow changing column order.
	MySQLColumnDisallowChangingOrder Type = "bb.plugin.advisor.mysql.column.disallow-changing-order"

	// MySQLColumnCommentConvention is an advisor type for MySQL column comment convention.
	MySQLColumnCommentConvention Type = "bb.plugin.advisor.mysql.column.comment"

	// MySQLAutoIncrementColumnMustInteger is an advisor type for auto-increment column.
	MySQLAutoIncrementColumnMustInteger Type = "bb.plugin.advisor.mysql.column.auto-increment-must-integer"

	// MySQLDisallowSetColumnCharset is an advisor type for MySQL disallow set column charset.
	MySQLDisallowSetColumnCharset Type = "bb.plugin.advisor.mysql.column.disallow-set-charset"

	// MySQLColumnTypeDisallowList is an advisor type for MySQL column type disallow list.
	MySQLColumnTypeDisallowList Type = "bb.plugin.advisor.mysql.column.type-disallow-list"

	// MySQLColumnMaximumCharacterLength is an advisor type for MySQL maximum character length.
	MySQLColumnMaximumCharacterLength Type = "bb.plugin.advisor.mysql.column.maximum-character-length"

	// MySQLColumnMaximumVarcharLength is an advisor type for MySQL maximum varchar length.
	MySQLColumnMaximumVarcharLength Type = "bb.plugin.advisor.mysql.column.maximum-varchar-length"

	// MySQLAutoIncrementColumnInitialValue is an advisor type for MySQL auto-increment column initial value.
	MySQLAutoIncrementColumnInitialValue Type = "bb.plugin.advisor.mysql.column.auto-increment-initial-value"

	// MySQLAutoIncrementColumnMustUnsigned is an advisor type for MySQL unsigned auto-increment column.
	MySQLAutoIncrementColumnMustUnsigned Type = "bb.plugin.advisor.mysql.column.auto-increment-must-unsigned"

	// MySQLCurrentTimeColumnCountLimit is an advisor type for MySQL current time column count limit.
	MySQLCurrentTimeColumnCountLimit Type = "bb.plugin.advisor.mysql.column.current-time-count-limit"

	// MySQLRequireColumnDefault is an advisor type for MySQL column default requirement.
	MySQLRequireColumnDefault Type = "bb.plugin.advisor.mysql.column.require-default"

	// MySQLNoSelectAll is an advisor type for MySQL no select all.
	MySQLNoSelectAll Type = "bb.plugin.advisor.mysql.select.no-select-all"

	// MySQLTableRequirePK is an advisor type for MySQL table require primary key.
	MySQLTableRequirePK Type = "bb.plugin.advisor.mysql.table.require-pk"

	// MySQLTableNoFK is an advisor type for MySQL table disallow foreign key.
	MySQLTableNoFK Type = "bb.plugin.advisor.mysql.table.no-foreign-key"

	// MySQLTableDropNamingConvention is an advisor type for MySQL table drop with naming convention.
	MySQLTableDropNamingConvention Type = "bb.plugin.advisor.mysql.table.drop-naming-convention"

	// MySQLTableCommentConvention is an advisor for MySQL table comment convention.
	MySQLTableCommentConvention Type = "bb.plugin.advisor.mysql.table.comment"

	// MySQLTableDisallowPartition is an advisor type for MySQL disallow table partition.
	MySQLTableDisallowPartition Type = "bb.plugin.advisor.mysql.table.disallow-partition"

	// MySQLTableDisallowTrigger is an advisor type for MySQL disallow table trigger.
	MySQLTableDisallowTrigger Type = "bb.plugin.advisor.mysql.table.disallow-trigger"

	// MySQLTableNoDuplicateIndex is an advisor type for MySQL no duplicate index.
	MySQLTableNoDuplicateIndex Type = "bb.plugin.advisor.mysql.table.no-duplicate-index"

	// MySQLTableTextFieldsTotalLength is an advisor type for MySQL table text fields total length.
	MySQLTableTextFieldsTotalLength Type = "bb.plugin.advisor.mysql.table.text-fields-total-length"

	// MySQLTableFieldsMaximumCount is an advisor type for MySQL table fields maximum count.
	MySQLTableDisallowSetCharset Type = "bb.plugin.advisor.mysql.table.disallow-set-charset"

	// MySQLDatabaseAllowDropIfEmpty is an advisor type for MySQL only allow drop empty database.
	MySQLDatabaseAllowDropIfEmpty Type = "bb.plugin.advisor.mysql.database.drop-empty-database"

	// MySQLIndexNoDuplicateColumn is an advisor type for MySQL no duplicate columns in index.
	MySQLIndexNoDuplicateColumn Type = "bb.plugin.advisor.mysql.index.no-duplicate-column"

	// MySQLIndexPKType is an advisor type for MySQL correct type of PK.
	MySQLIndexPKType Type = "bb.plugin.advisor.mysql.index.pk-type"

	// MySQLPrimaryKeyTypeAllowlist is an advisor type for MySQL primary key type allowlist.
	MySQLPrimaryKeyTypeAllowlist Type = "bb.plugin.advisor.mysql.index.primary-key-type-allowlist"

	// MySQLIndexKeyNumberLimit is an advisor type for MySQL index key number limit.
	MySQLIndexKeyNumberLimit Type = "bb.plugin.advisor.mysql.index.key-number-limit"

	// MySQLIndexTotalNumberLimit is an advisor type for MySQL index total number limit.
	MySQLIndexTotalNumberLimit Type = "bb.plugin.advisor.mysql.index.total-number-limit"

	// MySQLIndexTypeAllowList is an advisor type for MySQL index type allowlist.
	MySQLIndexTypeAllowList Type = "bb.plugin.advisor.mysql.index.type-allow-list"

	// MySQLCharsetAllowlist is an advisor type for MySQL charset allowlist.
	MySQLCharsetAllowlist Type = "bb.plugin.advisor.mysql.charset.allowlist"

	// MySQLCollationAllowlist is an advisor type for MySQL collation allowlist.
	MySQLCollationAllowlist Type = "bb.plugin.advisor.mysql.collation.allowlist"

	// MySQLIndexTypeNoBlob is an advisor type for MySQL index type no blob.
	MySQLIndexTypeNoBlob Type = "bb.plugin.advisor.mysql.index.type-no-blob"

	// MySQLStatementDisallowCommit is an advisor type for MySQL to disallow commit.
	MySQLStatementDisallowCommit Type = "bb.plugin.advisor.mysql.statement.disallow-commit"

	// MySQLStatementDisallowLimit is an advisor type for MySQL no LIMIT clause in INSERT/UPDATE/DELETE statement.
	MySQLStatementDisallowLimit Type = "bb.plugin.advisor.mysql.statement.disallow-limit"

	// MySQLStatementDisallowUsingFilesort is an advisor type for MySQL disallow using filesort in execution plan.
	MySQLStatementDisallowUsingFilesort Type = "bb.plugin.advisor.mysql.statement.disallow-using-filesort"

	// MySQLStatementDisallowUsingTemporary is an advisor type for MySQL disallow using temporary in execution plan.
	MySQLStatementDisallowUsingTemporary Type = "bb.plugin.advisor.mysql.statement.disallow-using-temporary"

	// MySQLInsertRowLimit is an advisor type for MySQL to limit INSERT rows.
	MySQLInsertRowLimit Type = "bb.plugin.advisor.mysql.insert.row-limit"

	// MySQLInsertMustSpecifyColumn is an advisor type for MySQL to enforce column specified.
	MySQLInsertMustSpecifyColumn Type = "bb.plugin.advisor.mysql.insert.must-specify-column"

	// MySQLInsertDisallowOrderByRand is an advisor type for MySQL to disallow order by rand in INSERT statements.
	MySQLInsertDisallowOrderByRand Type = "bb.plugin.advisor.mysql.insert.disallow-order-by-rand"

	// MySQLDisallowOrderBy is an advisor type for MySQL no ORDER BY clause in DELETE/UPDATE statement.
	MySQLDisallowOrderBy Type = "bb.plugin.advisor.mysql.statement.disallow-order-by"

	// MySQLMergeAlterTable is an advisor type for MySQL no redundant ALTER TABLE statements.
	MySQLMergeAlterTable Type = "bb.plugin.advisor.mysql.statement.merge-alter-table"

	// MySQLStatementAffectedRowLimit is an advisor type for MySQL UPDATE/DELETE affected row limit.
	MySQLStatementAffectedRowLimit Type = "bb.plugin.advisor.mysql.statement.affected-row-limit"

	// MySQLStatementDMLDryRun is an advisor type for MySQL DML dry run.
	MySQLStatementDMLDryRun Type = "bb.plugin.advisor.mysql.statement.dml-dry-run"

	// MySQLStatementSelectFullTableScan is an advisor type for checking MySQL select full table scan or not.
	MySQLStatementSelectFullTableScan Type = "bb.plugin.advisor.mysql.statement.select-full-table-scan"

	// MySQLStatementWhereNoEqualNull is an advisor type for checking MySQL no equal null in WHERE clause.
	MySQLStatementWhereNoEqualNull Type = "bb.plugin.advisor.mysql.statement.where.no-equal-null"

	// MySQLStatementWhereDisallowUsingFunction is an advisor type for checking MySQL disallow using function in WHERE clause.
	MySQLStatementWhereDisallowUsingFunction Type = "bb.plugin.advisor.mysql.statement.query.disallow-using-function"

	// MySQLStatementQueryMinumumPlanLevel is an advisor type for checking MySQL query minimum plan level.
	MySQLStatementQueryMinumumPlanLevel Type = "bb.plugin.advisor.mysql.statement.query.minimum-plan-level"

	// MySQLStatementWhereMaximumLogicalOperatorCount is an advisor type for checking MySQL statement maximum logical operator count in WHERE clause.
	MySQLStatementWhereMaximumLogicalOperatorCount Type = "bb.plugin.advisor.mysql.statement.where.maximum-logical-operator-count"

	// MySQLStatementMaximumLimitValue is an advisor type for MySQL statement maximum limit value.
	MySQLStatementMaximumLimitValue Type = "bb.plugin.advisor.mysql.statement.maximum-limit-value"

	// MySQLStatementMaximumJoinTableCount is an advisor type for MySQL statement maximum join table count.
	MySQLStatementMaximumJoinTableCount Type = "bb.plugin.advisor.mysql.statement.maximum-join-table-count"

	// MySQLStatementMaximumStatementsInTransaction is an advisor type for MySQL maximum statements in transaction.
	MySQLStatementMaximumStatementsInTransaction Type = "bb.plugin.advisor.mysql.statement.maximum-statements-in-transaction"

	// MySQLStatementJoinStrictColumnAttrs is an advisor type for MySQL statement strict column attrs(type, charset) in join.
	MySQLStatementJoinStrictColumnAttrs Type = "bb.plugin.advisor.mysql.statement.join-strict-column-attrs"

	// MySQLStatementDisallowMixDML is an advisor type for MySQL disallow mix DML on same table.
	MySQLStatementDisallowMixDML Type = "bb.plugin.advisor.mysql.statement.disallow-mix-dml"

	// MySQLStatementPriorBackupCheck is an advisor type for MySQL prior backup check.
	MySQLStatementPriorBackupCheck Type = "bb.plugin.advisor.mysql.statement.prior-backup-check"

	// MySQLProcedureDisallowCreate is an advisor type for MySQL disallow create procedure.
	MySQLProcedureDisallowCreate Type = "bb.plugin.advisor.mysql.procedure.disallow-create"

	// MySQLEventDisallowCreate is an advisor type for MySQL disallow create event.
	MySQLEventDisallowCreate Type = "bb.plugin.advisor.mysql.event.disallow-create"

	// MySQLViewDisallowCreate is an advisor type for MySQL disallow create view.
	MySQLViewDisallowCreate Type = "bb.plugin.advisor.mysql.view.disallow-create"

	// MySQLFunctionDisallowCreate is an advisor type for MySQL disallow create function.
	MySQLFunctionDisallowCreate Type = "bb.plugin.advisor.mysql.function.disallow-create"

	// MySQLFunctionDisallowedList is an advisor type for MySQL disallowed function list.
	MySQLFunctionDisallowedList Type = "bb.plugin.advisor.mysql.function.disallowed-list"

	// PostgreSQL Advisor.

	// PostgreSQLSyntax is an advisor type for PostgreSQL syntax.
	PostgreSQLSyntax Type = "bb.plugin.advisor.postgresql.syntax"

	// PostgreSQLNamingTableConvention is an advisor type for PostgreSQL table naming convention.
	PostgreSQLNamingTableConvention Type = "bb.plugin.advisor.postgresql.naming.table"

	// PostgreSQLNamingColumnConvention is an advisor type for PostgreSQL column naming convention.
	PostgreSQLNamingColumnConvention Type = "bb.plugin.advisor.postgresql.naming.column"

	// PostgreSQLNamingPKConvention is an advisor type for PostgreSQL primary key naming convention.
	PostgreSQLNamingPKConvention Type = "bb.plugin.advisor.postgresql.naming.pk"

	// PostgreSQLNamingIndexConvention is an advisor type for PostgreSQL index naming convention.
	PostgreSQLNamingIndexConvention Type = "bb.plugin.advisor.postgresql.naming.index"

	// PostgreSQLNamingUKConvention is an advisor type for PostgreSQL unique key naming convention.
	PostgreSQLNamingUKConvention Type = "bb.plugin.advisor.postgresql.naming.uk"

	// PostgreSQLNamingFKConvention is an advisor type for PostgreSQL foreign key naming convention.
	PostgreSQLNamingFKConvention Type = "bb.plugin.advisor.postgresql.naming.fk"

	// PostgreSQLColumnNoNull is an advisor type for PostgreSQL column no NULL value.
	PostgreSQLColumnNoNull Type = "bb.plugin.advisor.postgresql.column.no-null"

	// PostgreSQLColumnRequirement is an advisor type for PostgreSQL column requirement.
	PostgreSQLColumnRequirement Type = "bb.plugin.advisor.postgresql.column.require"

	// PostgreSQLCommentConvention is an advisor type for PostgreSQL comment convention.
	PostgreSQLCommentConvention Type = "bb.plugin.advisor.postgresql.comment"

	// PostgreSQLTableRequirePK is an advisor type for PostgreSQL table require primary key.
	PostgreSQLTableRequirePK Type = "bb.plugin.advisor.postgresql.table.require-pk"

	// PostgreSQLNoLeadingWildcardLike is an advisor type for PostgreSQL no leading wildcard LIKE.
	PostgreSQLNoLeadingWildcardLike Type = "bb.plugin.advisor.postgresql.where.no-leading-wildcard-like"

	// PostgreSQLWhereRequirement is an advisor type for PostgreSQL WHERE clause requirement.
	PostgreSQLWhereRequirement Type = "bb.plugin.advisor.postgresql.where.require"

	// PostgreSQLNoSelectAll is an advisor type for PostgreSQL no select all.
	PostgreSQLNoSelectAll Type = "bb.plugin.advisor.postgresql.select.no-select-all"

	// PostgreSQLMigrationCompatibility is an advisor type for PostgreSQL migration compatibility.
	PostgreSQLMigrationCompatibility Type = "bb.plugin.advisor.postgresql.migration-compatibility"

	// PostgreSQLTableNoFK is an advisor type for PostgreSQL table disallow foreign key.
	PostgreSQLTableNoFK Type = "bb.plugin.advisor.postgresql.table.no-foreign-key"

	// PostgreSQLTableDisallowPartition is an advisor type for PostgreSQL disallow table partition.
	PostgreSQLTableDisallowPartition Type = "bb.plugin.advisor.postgresql.table.disallow-partition"

	// PostgreSQLInsertRowLimit is an advisor type for PostgreSQL to limit INSERT rows.
	PostgreSQLInsertRowLimit Type = "bb.plugin.advisor.postgresql.insert.row-limit"

	// PostgreSQLInsertMustSpecifyColumn is an advisor type for PostgreSQL to enforce column specified.
	PostgreSQLInsertMustSpecifyColumn Type = "bb.plugin.advisor.postgresql.insert.must-specify-column"

	// PostgreSQLInsertDisallowOrderByRand is an advisor type for PostgreSQL to disallow order by rand in INSERT statements.
	PostgreSQLInsertDisallowOrderByRand Type = "bb.plugin.advisor.postgresql.insert.disallow-order-by-rand"

	// PostgreSQLIndexKeyNumberLimit is an advisor type for postgresql index key number limit.
	PostgreSQLIndexKeyNumberLimit Type = "bb.plugin.advisor.postgresql.index.key-number-limit"

	// PostgreSQLPrimaryKeyTypeAllowlist is an advisor type for postgresql primary key type allowlist.
	PostgreSQLPrimaryKeyTypeAllowlist Type = "bb.plugin.advisor.postgresql.index.primary-key-type-allowlist"

	// PostgreSQLIndexTotalNumberLimit is an advisor type for PostgreSQL index total number limit.
	PostgreSQLIndexTotalNumberLimit Type = "bb.plugin.advisor.postgresql.index.total-number-limit"

	// PostgreSQLEncodingAllowlist is an advisor type for PostgreSQL encoding allowlist.
	PostgreSQLEncodingAllowlist Type = "bb.plugin.advisor.postgresql.charset.allowlist"

	// PostgreSQLIndexNoDuplicateColumn is an advisor type for Postgresql no duplicate columns in index.
	PostgreSQLIndexNoDuplicateColumn Type = "bb.plugin.advisor.postgresql.index.no-duplicate-column"

	// PostgreSQLCreateIndexConcurrently is an advisor type for PostgreSQL to create index concurrently.
	PostgreSQLCreateIndexConcurrently Type = "bb.plugin.advisor.postgresql.index.create-concurrently"

	// PostgreSQLColumnTypeDisallowList is an advisor type for Postgresql column type disallow list.
	PostgreSQLColumnTypeDisallowList Type = "bb.plugin.advisor.postgresql.column.type-disallow-list"

	// PostgreSQLColumnDisallowChangingType is an advisor type for PostgreSQL disallow changing column type.
	PostgreSQLColumnDisallowChangingType Type = "bb.plugin.advisor.postgresql.column.disallow-changing-type"

	// PostgreSQLColumnMaximumCharacterLength is an advisor type for PostgreSQL maximum character length.
	PostgreSQLColumnMaximumCharacterLength Type = "bb.plugin.advisor.postgresql.column.maximum-character-length"

	// PostgreSQLRequireColumnDefault is an advisor type for PostgreSQL column default requirement.
	PostgreSQLRequireColumnDefault Type = "bb.plugin.advisor.postgresql.column.require-default"

	// PostgreSQLStatementDisallowCommit is an advisor type for PostgreSQL to disallow commit.
	PostgreSQLStatementDisallowCommit Type = "bb.plugin.advisor.postgresql.statement.disallow-commit"

	// PostgreSQLStatementDMLDryRun is an advisor type for PostgreSQL DML dry run.
	PostgreSQLStatementDMLDryRun Type = "bb.plugin.advisor.postgresql.statement.dml-dry-run"

	// PostgreSQLStatementAffectedRowLimit is an advisor type for PostgreSQL UPDATE/DELETE affected row limit.
	PostgreSQLStatementAffectedRowLimit Type = "bb.plugin.advisor.postgresql.statement.affected-row-limit"

	// PostgreSQLMergeAlterTable is an advisor type for PostgreSQL no redundant ALTER TABLE statements.
	PostgreSQLMergeAlterTable Type = "bb.plugin.advisor.postgresql.statement.merge-alter-table"

	// PostgreSQLAddCheckNotValid is an advisor type for PostgreSQL to add check not valid.
	PostgreSQLAddCheckNotValid Type = "bb.plugin.advisor.postgresql.statement.add-check-not-valid"

	// PostgreSQLDisallowAddColumnWithDefault is an advisor type for PostgreSQL to disallow add column with default.
	PostgreSQLDisallowAddColumnWithDefault Type = "bb.plugin.advisor.postgresql.statement.disallow-add-column-with-default"

	// PostgreSQLDisallowAddNotNull is an advisor type for PostgreSQl to disallow add not null.
	PostgreSQLDisallowAddNotNull Type = "bb.plugin.advisor.postgresql.statement.disallow-add-not-null"

	// PostgreSQLTableDropNamingConvention is an advisor type for PostgreSQL table drop with naming convention.
	PostgreSQLTableDropNamingConvention Type = "bb.plugin.advisor.postgresql.table.drop-naming-convention"

	// PostgreSQLCollationAllowlist is an advisor type for PostgreSQL collation allowlist.
	PostgreSQLCollationAllowlist Type = "bb.plugin.advisor.postgresql.collation.allowlist"

	// PostgreSQLStatementDisallowCascade is an advisor type for PostgreSQL to disallow cascade.
	PostgreSQLStatementDisallowCascade Type = "bb.plugin.advisor.postgresql.statement.disallow-cascade"

	// PostgreSQLStatementCreateSpecifySchema is an advisor type for PostgreSQL to specify schema when creating.
	PostgreSQLStatementCreateSpecifySchema Type = "bb.plugin.advisor.postgresql.statement.create-specify-schema"

	// PostgreSQLStatementCheckSetRoleVariable is an advisor type for PostgreSQL to check set role variable.
	PostgreSQLStatementCheckSetRoleVariable Type = "bb.plugin.advisor.postgresql.statement.check-set-role-variable"

	// PostgreSQLStatementDisallowMixDML is an advisor type for PostgreSQL disallow mix DML on same table.
	PostgreSQLStatementDisallowMixDML Type = "bb.plugin.advisor.postgresql.statement.disallow-mix-dml"

	// PostgreSQLStatementPriorBackupCheck is an advisor type for PostgreSQL do prior backup check.
	PostgreSQLStatementPriorBackupCheck Type = "bb.plugin.advisor.postgresql.statement.prior-backup-check"

	// Oracle Advisor.

	// OracleSyntax is an advisor type for Oracle syntax.
	OracleSyntax Type = "bb.plugin.advisor.oracle.syntax"

	// OracleTableRequirePK is an advisor type for Oracle table require primary key.
	OracleTableRequirePK Type = "bb.plugin.advisor.oracle.table.require-pk"

	// OracleTableNoFK is an advisor type for Oracle table disallow foreign key.
	OracleTableNoFK Type = "bb.plugin.advisor.oracle.table.no-foreign-key"

	// OracleNamingTableConvention is an advisor type for Oracle table naming convention.
	OracleNamingTableConvention Type = "bb.plugin.advisor.oracle.naming.table"

	// OracleColumnRequirement is an advisor type for Oracle column requirement.
	OracleColumnRequirement Type = "bb.plugin.advisor.oracle.column.require"

	// OracleColumnTypeDisallowList is an advisor type for Oracle column type disallow list.
	OracleColumnTypeDisallowList Type = "bb.plugin.advisor.oracle.column.type-disallow-list"

	// OracleColumnMaximumCharacterLength is an advisor type for Oracle maximum character length.
	OracleColumnMaximumCharacterLength Type = "bb.plugin.advisor.oracle.column.maximum-character-length"

	// OracleColumnMaximumVarcharLength is an advisor type for Oracle maximum varchar length.
	OracleColumnMaximumVarcharLength Type = "bb.plugin.advisor.oracle.column.maximum-varchar-length"

	// OracleNoSelectAll is an advisor type for Oracle no select all.
	OracleNoSelectAll Type = "bb.plugin.advisor.oracle.select.no-select-all"

	// OracleNoLeadingWildcardLike is an advisor type for Oracle no leading wildcard LIKE.
	OracleNoLeadingWildcardLike Type = "bb.plugin.advisor.oracle.where.no-leading-wildcard-like"

	// OracleWhereRequirement is an advisor type for Oracle WHERE clause requirement.
	OracleWhereRequirement Type = "bb.plugin.advisor.oracle.where.require"

	// OracleInsertMustSpecifyColumn is an advisor type for Oracle to enforce column specified.
	OracleInsertMustSpecifyColumn Type = "bb.plugin.advisor.oracle.insert.must-specify-column"

	// OracleIndexKeyNumberLimit is an advisor type for Oracle index key number limit.
	OracleIndexKeyNumberLimit Type = "bb.plugin.advisor.oracle.index.key-number-limit"

	// OracleColumnNoNull is an advisor type for Oracle column no NULL value.
	OracleColumnNoNull Type = "bb.plugin.advisor.oracle.column.no-null"

	// OracleRequireColumnDefault is an advisor type for Oracle column default requirement.
	OracleRequireColumnDefault Type = "bb.plugin.advisor.oracle.column.require-default"

	// OracleAddNotNullColumnRequireDefault is an advisor type for Oracle adding not null column requires default.
	OracleAddNotNullColumnRequireDefault Type = "bb.plugin.advisor.oracle.column.add-not-null-column-require-default"

	// OracleTableNamingNoKeyword is an advisor type for Oracle table naming convention without keyword.
	OracleTableNamingNoKeyword Type = "bb.plugin.advisor.oracle.naming.table-no-keyword"

	// OracleIdentifierNamingNoKeyword is an advisor type for Oracle identifier naming convention without keyword.
	OracleIdentifierNamingNoKeyword Type = "bb.plugin.advisor.oracle.naming.identifier-no-keyword"

	// OracleIdentifierCase is an advisor type for Oracle identifier case.
	OracleIdentifierCase Type = "bb.plugin.advisor.oracle.naming.identifier-case"

	// Snowflake Advisor.

	// SnowflakeSyntax is an advisor type for Snowflake syntax.
	SnowflakeSyntax Type = "bb.plugin.advisor.snowflake.syntax"

	// SnowflakeNamingTableConvention is an advisor type for Snowflake table naming convention.
	SnowflakeNamingTableConvention Type = "bb.plugin.advisor.snowflake.naming.table"

	// SnowflakeTableRequirePK is an advisor type for Snowflake table require primary key.
	SnowflakeTableRequirePK Type = "bb.plugin.advisor.snowflake.table.require-pk"

	// SnowflakeTableNoFK is an advisor type for Snowflake table disallow foreign key.
	SnowflakeTableNoFK Type = "bb.plugin.advisor.snowflake.table.no-foreign-key"

	// SnowflakeColumnMaximumVarcharLength is an advisor type for Snowflake maximum varchar length.
	SnowflakeColumnMaximumVarcharLength Type = "bb.plugin.advisor.snowflake.column.maximum-varchar-length"

	// SnowflakeTableNamingNoKeyword is an advisor type for Snowflake table naming convention without keyword.
	SnowflakeTableNamingNoKeyword Type = "bb.plugin.advisor.snowflake.naming.table-no-keyword"

	// SnowflakeWhereRequirement is an advisor type for Snowflake WHERE clause requirement.
	SnowflakeWhereRequirement Type = "bb.plugin.advisor.snowflake.where.require"

	// SnowflakeIdentifierNamingNoKeyword is an advisor type for Snowflake identifier naming convention without keyword.
	SnowflakeIdentifierNamingNoKeyword Type = "bb.plugin.advisor.snowflake.naming.identifier-no-keyword"

	// SnowflakeColumnRequirement is an advisor type for Snowflake column requirement.
	SnowflakeColumnRequirement Type = "bb.plugin.advisor.snowflake.column.require"

	// SnowflakeIdentifierCase is an advisor type for Snowflake identifier case.
	SnowflakeIdentifierCase Type = "bb.plugin.advisor.snowflake.naming.identifier-case"

	// SnowflakeColumnNoNull is an advisor type for Snowflake column no NULL value.
	SnowflakeColumnNoNull Type = "bb.plugin.advisor.snowflake.column.no-null"

	// SnowflakeNoSelectAll is an advisor type for Snowflake no select all.
	SnowflakeNoSelectAll Type = "bb.plugin.advisor.snowflake.select.no-select-all"

	// SnowflakeTableDropNamingConvention is an advisor type for Snowflake table drop with naming convention.
	SnowflakeTableDropNamingConvention Type = "bb.plugin.advisor.snowflake.table.drop-naming-convention"

	// SnowflakeMigrationCompatibility is an advisor type for Snowflake migration compatibility.
	SnowflakeMigrationCompatibility Type = "bb.plugin.advisor.snowflake.migration-compatibility"

	// MSSQL Advisor.

	// MSSQLSyntax is an advisor type for MSSQL syntax.
	MSSQLSyntax Type = "bb.plugin.advisor.mssql.syntax"

	// MSSQLNoSelectAll is an advisor type for MSSQL no select all.
	MSSQLNoSelectAll Type = "bb.plugin.advisor.mssql.select.no-select-all"

	// MSSQLNamingTableConvention is an advisor type for MSSQL table naming convention.
	MSSQLNamingTableConvention Type = "bb.plugin.advisor.mssql.naming.table"

	// MSSQLTableNamingNoKeyword is an advisor type for MSSQL table naming convention without keyword.
	MSSQLTableNamingNoKeyword Type = "bb.plugin.advisor.mssql.naming.table-no-keyword"

	// MSSQLIdentifierNamingNoKeyword is an advisor type for MSSQL identifier naming convention without keyword.
	MSSQLIdentifierNamingNoKeyword Type = "bb.plugin.advisor.mssql.naming.identifier-no-keyword"

	// MSSQLWhereRequirement is an advisor type for MSSQL WHERE clause requirement.
	MSSQLWhereRequirement Type = "bb.plugin.advisor.mssql.where.require"

	// MSSQLColumnMaximumVarcharLength is an advisor type for MSSQL maximum varchar length.
	MSSQLColumnMaximumVarcharLength Type = "bb.plugin.advisor.mssql.column.maximum-varchar-length"

	// MSSQLTableDropNamingConvention is an advisor type for MSSQL table drop with naming convention.
	MSSQLTableDropNamingConvention Type = "bb.plugin.advisor.mssql.table.drop-naming-convention"

	// MSSQLTableRequirePK is an advisor type for MSSQL table require primary key.
	MSSQLTableRequirePK Type = "bb.plugin.advisor.mssql.table.require-pk"

	// MSSQLColumnNoNull is an advisor type for MSSQL column no NULL value.
	MSSQLColumnNoNull Type = "bb.plugin.advisor.mssql.column.no-null"

	// MSSQLTableNoFK is an advisor type for MSSQL table disallow foreign key.
	MSSQLTableNoFK Type = "bb.plugin.advisor.mssql.table.no-foreign-key"

	// MSSQLMigrationCompatibility is an advisor type for MSSQL migration compatibility.
	MSSQLMigrationCompatibility Type = "bb.plugin.advisor.mssql.migration-compatibility"

	// MSSQLColumnRequirement is an advisor type for MSSQL column requirement.
	MSSQLColumnRequirement Type = "bb.plugin.advisor.mssql.column.require"
)

// Advice is the result of an advisor.
type Advice struct {
	// Status is the SQL check result. Could be "SUCCESS", "WARN", "ERROR"
	Status Status `json:"status"`
	// Code is the SQL check error code.
	Code    Code   `json:"code"`
	Title   string `json:"title"`
	Content string `json:"content"`
	Line    int    `json:"line"`
	Column  int    `json:"column"`
	Details string `json:"details,omitempty"`
}

// SyntaxMode is the type of syntax mode.
type SyntaxMode int

const (
	// SyntaxModeNormal is the normal syntax mode.
	SyntaxModeNormal SyntaxMode = iota
	// SyntaxModeSDL is the SDL syntax mode.
	SyntaxModeSDL
)

// Context is the context for advisor.
type Context struct {
	Charset               string
	Collation             string
	DBSchema              *storepb.DatabaseSchemaMetadata
	SyntaxMode            SyntaxMode
	ChangeType            storepb.PlanCheckRunConfig_ChangeDatabaseType
	PreUpdateBackupDetail *storepb.PlanCheckRunConfig_PreUpdateBackupDetail

	// SQL review rule special fields.
	AST     any
	Rule    *storepb.SQLReviewRule
	Catalog *catalog.Finder
	Driver  *sql.DB
	Context context.Context

	// CurrentDatabase is the current database. Special for Snowflake.
	CurrentDatabase string
	// CurrentSchema is the current schema. Special for Oracle.
	CurrentSchema string
	// Statement is the original statement of AST, it is used for some PostgreSQL
	// advisors which need to check the token stream.
	Statements string
}

// Advisor is the interface for advisor.
type Advisor interface {
	Check(ctx Context, statement string) ([]Advice, error)
}

var (
	advisorMu sync.RWMutex
	advisors  = make(map[storepb.Engine]map[Type]Advisor)
)

// Register makes a advisor available by the provided id.
// If Register is called twice with the same name or if advisor is nil,
// it panics.
func Register(dbType storepb.Engine, advType Type, f Advisor) {
	advisorMu.Lock()
	defer advisorMu.Unlock()
	if f == nil {
		panic("advisor: Register advisor is nil")
	}
	dbAdvisors, ok := advisors[dbType]
	if !ok {
		advisors[dbType] = map[Type]Advisor{
			advType: f,
		}
	} else {
		if _, dup := dbAdvisors[advType]; dup {
			panic(fmt.Sprintf("advisor: Register called twice for advisor %v for %v", advType, dbType))
		}
		dbAdvisors[advType] = f
	}
}

// Check runs the advisor and returns the advices.
func Check(dbType storepb.Engine, advType Type, ctx Context, statement string) (adviceList []Advice, err error) {
	defer func() {
		if panicErr := recover(); panicErr != nil {
			panicErr, ok := panicErr.(error)
			if !ok {
				panicErr = errors.Errorf("%v", panicErr)
			}
			err = errors.Errorf("advisor check PANIC RECOVER, type: %v, err: %v", advType, panicErr)

			slog.Error("advisor check PANIC RECOVER", log.BBError(panicErr), log.BBStack("panic-stack"))
		}
	}()

	advisorMu.RLock()
	dbAdvisors, ok := advisors[dbType]
	defer advisorMu.RUnlock()
	if !ok {
		return nil, errors.Errorf("advisor: unknown db advisor type %v", dbType)
	}

	f, ok := dbAdvisors[advType]
	if !ok {
		return nil, errors.Errorf("advisor: unknown advisor %v for %v", advType, dbType)
	}

	return f.Check(ctx, statement)
}
