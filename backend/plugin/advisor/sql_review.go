package advisor

import (
	"context"
	"database/sql"
	"encoding/json"
	"log"
	"regexp"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/plugin/advisor/catalog"
	"github.com/bytebase/bytebase/backend/plugin/advisor/db"
)

// How to add a SQL review rule:
//   1. Implement an advisor.(plugin/advisor/mysql or plugin/advisor/pg)
//   2. Register this advisor in map[db.Type][AdvisorType].(plugin/advisor.go)
//   3. Add advisor error code if needed(plugin/advisor/code.go).
//   4. Map SQLReviewRuleType to advisor.Type in getAdvisorTypeByRule(current file).

// SQLReviewRuleLevel is the error level for SQL review rule.
type SQLReviewRuleLevel string

// SQLReviewRuleType is the type of schema rule.
type SQLReviewRuleType string

const (
	// MySQL rule list.

	// MySQLSchemaRuleMySQLEngine require InnoDB as the storage engine.
	MySQLSchemaRuleMySQLEngine SQLReviewRuleType = "mysql.engine.mysql.use-innodb"

	// MySQLSchemaRuleTableNaming enforce the table name format.
	MySQLSchemaRuleTableNaming SQLReviewRuleType = "mysql.naming.table"
	// MySQLSchemaRuleColumnNaming enforce the column name format.
	MySQLSchemaRuleColumnNaming SQLReviewRuleType = "mysql.naming.column"
	// MySQLSchemaRuleUKNaming enforce the unique key name format.
	MySQLSchemaRuleUKNaming SQLReviewRuleType = "mysql.naming.index.uk"
	// MySQLSchemaRuleFKNaming enforce the foreign key name format.
	MySQLSchemaRuleFKNaming SQLReviewRuleType = "mysql.naming.index.fk"
	// MySQLSchemaRuleIDXNaming enforce the index name format.
	MySQLSchemaRuleIDXNaming SQLReviewRuleType = "mysql.naming.index.idx"
	// MySQLSchemaRuleAutoIncrementColumnNaming enforce the auto_increment column name format.
	MySQLSchemaRuleAutoIncrementColumnNaming SQLReviewRuleType = "mysql.naming.column.auto-increment"

	// MySQLSchemaRuleStatementNoSelectAll disallow 'SELECT *'.
	MySQLSchemaRuleStatementNoSelectAll SQLReviewRuleType = "mysql.statement.select.no-select-all"
	// MySQLSchemaRuleStatementRequireWhere require 'WHERE' clause.
	MySQLSchemaRuleStatementRequireWhere SQLReviewRuleType = "mysql.statement.where.require"
	// MySQLSchemaRuleStatementNoLeadingWildcardLike disallow leading '%' in LIKE, e.g. LIKE foo = '%x' is not allowed.
	MySQLSchemaRuleStatementNoLeadingWildcardLike SQLReviewRuleType = "mysql.statement.where.no-leading-wildcard-like"
	// MySQLSchemaRuleStatementDisallowCommit disallow using commit in the issue.
	MySQLSchemaRuleStatementDisallowCommit SQLReviewRuleType = "mysql.statement.disallow-commit"
	// MySQLSchemaRuleStatementDisallowLimit disallow the LIMIT clause in INSERT, DELETE and UPDATE statements.
	MySQLSchemaRuleStatementDisallowLimit SQLReviewRuleType = "mysql.statement.disallow-limit"
	// MySQLSchemaRuleStatementDisallowOrderBy disallow the ORDER BY clause in DELETE and UPDATE statements.
	MySQLSchemaRuleStatementDisallowOrderBy SQLReviewRuleType = "mysql.statement.disallow-order-by"
	// MySQLSchemaRuleStatementMergeAlterTable disallow redundant ALTER TABLE statements.
	MySQLSchemaRuleStatementMergeAlterTable SQLReviewRuleType = "mysql.statement.merge-alter-table"
	// MySQLSchemaRuleStatementInsertRowLimit enforce the insert row limit.
	MySQLSchemaRuleStatementInsertRowLimit SQLReviewRuleType = "mysql.statement.insert.row-limit"
	// MySQLSchemaRuleStatementInsertMustSpecifyColumn enforce the insert column specified.
	MySQLSchemaRuleStatementInsertMustSpecifyColumn SQLReviewRuleType = "mysql.statement.insert.must-specify-column"
	// MySQLSchemaRuleStatementInsertDisallowOrderByRand disallow the order by rand in the INSERT statement.
	MySQLSchemaRuleStatementInsertDisallowOrderByRand SQLReviewRuleType = "mysql.statement.insert.disallow-order-by-rand"
	// MySQLSchemaRuleStatementAffectedRowLimit enforce the UPDATE/DELETE affected row limit.
	MySQLSchemaRuleStatementAffectedRowLimit SQLReviewRuleType = "mysql.statement.affected-row-limit"
	// MySQLSchemaRuleStatementDMLDryRun dry run the dml.
	MySQLSchemaRuleStatementDMLDryRun SQLReviewRuleType = "mysql.statement.dml-dry-run"

	// MySQLSchemaRuleTableRequirePK require the table to have a primary key.
	MySQLSchemaRuleTableRequirePK SQLReviewRuleType = "mysql.table.require-pk"
	// MySQLSchemaRuleTableNoFK require the table disallow the foreign key.
	MySQLSchemaRuleTableNoFK SQLReviewRuleType = "mysql.table.no-foreign-key"
	// MySQLSchemaRuleTableDropNamingConvention require only the table following the naming convention can be deleted.
	MySQLSchemaRuleTableDropNamingConvention SQLReviewRuleType = "mysql.table.drop-naming-convention"
	// MySQLSchemaRuleTableCommentConvention enforce the table comment convention.
	MySQLSchemaRuleTableCommentConvention SQLReviewRuleType = "mysql.table.comment"
	// MySQLSchemaRuleTableDisallowPartition disallow the table partition.
	MySQLSchemaRuleTableDisallowPartition SQLReviewRuleType = "mysql.table.disallow-partition"

	// MySQLSchemaRuleRequiredColumn enforce the required columns in each table.
	MySQLSchemaRuleRequiredColumn SQLReviewRuleType = "mysql.column.required"
	// MySQLSchemaRuleColumnNotNull enforce the columns cannot have NULL value.
	MySQLSchemaRuleColumnNotNull SQLReviewRuleType = "mysql.column.no-null"
	// MySQLSchemaRuleColumnDisallowChangeType disallow change column type.
	MySQLSchemaRuleColumnDisallowChangeType SQLReviewRuleType = "mysql.column.disallow-change-type"
	// MySQLSchemaRuleColumnSetDefaultForNotNull require the not null column to set default value.
	MySQLSchemaRuleColumnSetDefaultForNotNull SQLReviewRuleType = "mysql.column.set-default-for-not-null"
	// MySQLSchemaRuleColumnDisallowChange disallow CHANGE COLUMN statement.
	MySQLSchemaRuleColumnDisallowChange SQLReviewRuleType = "mysql.column.disallow-change"
	// MySQLSchemaRuleColumnDisallowChangingOrder disallow changing column order.
	MySQLSchemaRuleColumnDisallowChangingOrder SQLReviewRuleType = "mysql.column.disallow-changing-order"
	// MySQLSchemaRuleColumnCommentConvention enforce the column comment convention.
	MySQLSchemaRuleColumnCommentConvention SQLReviewRuleType = "mysql.column.comment"
	// MySQLSchemaRuleColumnAutoIncrementMustInteger require the auto-increment column to be integer.
	MySQLSchemaRuleColumnAutoIncrementMustInteger SQLReviewRuleType = "mysql.column.auto-increment-must-integer"
	// MySQLSchemaRuleColumnTypeDisallowList enforce the column type disallow list.
	MySQLSchemaRuleColumnTypeDisallowList SQLReviewRuleType = "mysql.column.type-disallow-list"
	// MySQLSchemaRuleColumnDisallowSetCharset disallow set column charset.
	MySQLSchemaRuleColumnDisallowSetCharset SQLReviewRuleType = "mysql.column.disallow-set-charset"
	// MySQLSchemaRuleColumnMaximumCharacterLength enforce the maximum character length.
	MySQLSchemaRuleColumnMaximumCharacterLength SQLReviewRuleType = "mysql.column.maximum-character-length"
	// MySQLSchemaRuleColumnAutoIncrementInitialValue enforce the initial auto-increment value.
	MySQLSchemaRuleColumnAutoIncrementInitialValue SQLReviewRuleType = "mysql.column.auto-increment-initial-value"
	// MySQLSchemaRuleColumnAutoIncrementMustUnsigned enforce the auto-increment column to be unsigned.
	MySQLSchemaRuleColumnAutoIncrementMustUnsigned SQLReviewRuleType = "mysql.column.auto-increment-must-unsigned"
	// MySQLSchemaRuleCurrentTimeColumnCountLimit enforce the current column count limit.
	MySQLSchemaRuleCurrentTimeColumnCountLimit SQLReviewRuleType = "mysql.column.current-time-count-limit"
	// MySQLSchemaRuleColumnRequireDefault enforce the column default.
	MySQLSchemaRuleColumnRequireDefault SQLReviewRuleType = "mysql.column.require-default"

	// MySQLSchemaRuleSchemaBackwardCompatibility enforce the MySQL and TiDB support check whether the schema change is backward compatible.
	MySQLSchemaRuleSchemaBackwardCompatibility SQLReviewRuleType = "mysql.schema.backward-compatibility"

	// MySQLSchemaRuleDropEmptyDatabase enforce the MySQL and TiDB support check if the database is empty before users drop it.
	MySQLSchemaRuleDropEmptyDatabase SQLReviewRuleType = "mysql.database.drop-empty-database"

	// MySQLSchemaRuleIndexNoDuplicateColumn require the index no duplicate column.
	MySQLSchemaRuleIndexNoDuplicateColumn SQLReviewRuleType = "mysql.index.no-duplicate-column"
	// MySQLSchemaRuleIndexKeyNumberLimit enforce the index key number limit.
	MySQLSchemaRuleIndexKeyNumberLimit SQLReviewRuleType = "mysql.index.key-number-limit"
	// MySQLSchemaRuleIndexPKTypeLimit enforce the type restriction of columns in primary key.
	MySQLSchemaRuleIndexPKTypeLimit SQLReviewRuleType = "mysql.index.pk-type-limit"
	// MySQLSchemaRuleIndexTypeNoBlob enforce the type restriction of columns in index.
	MySQLSchemaRuleIndexTypeNoBlob SQLReviewRuleType = "mysql.index.type-no-blob"
	// MySQLSchemaRuleIndexTotalNumberLimit enforce the index total number limit.
	MySQLSchemaRuleIndexTotalNumberLimit SQLReviewRuleType = "mysql.index.total-number-limit"

	// MySQLSchemaRuleCharsetAllowlist enforce the charset allowlist.
	MySQLSchemaRuleCharsetAllowlist SQLReviewRuleType = "mysql.system.charset.allowlist"

	// MySQLSchemaRuleCollationAllowlist enforce the collation allowlist.
	MySQLSchemaRuleCollationAllowlist SQLReviewRuleType = "mysql.system.collation.allowlist"

	// TiDB rule list.

	// TiDBSchemaRuleTableNaming enforce the table name format.
	TiDBSchemaRuleTableNaming SQLReviewRuleType = "tidb.naming.table"
	// TiDBSchemaRuleColumnNaming enforce the column name format.
	TiDBSchemaRuleColumnNaming SQLReviewRuleType = "tidb.naming.column"
	// TiDBSchemaRuleUKNaming enforce the unique key name format.
	TiDBSchemaRuleUKNaming SQLReviewRuleType = "tidb.naming.index.uk"
	// TiDBSchemaRuleFKNaming enforce the foreign key name format.
	TiDBSchemaRuleFKNaming SQLReviewRuleType = "tidb.naming.index.fk"
	// TiDBSchemaRuleIDXNaming enforce the index name format.
	TiDBSchemaRuleIDXNaming SQLReviewRuleType = "tidb.naming.index.idx"
	// TiDBSchemaRuleAutoIncrementColumnNaming enforce the auto_increment column name format.
	TiDBSchemaRuleAutoIncrementColumnNaming SQLReviewRuleType = "tidb.naming.column.auto-increment"

	// TiDBSchemaRuleStatementNoSelectAll disallow 'SELECT *'.
	TiDBSchemaRuleStatementNoSelectAll SQLReviewRuleType = "tidb.statement.select.no-select-all"
	// TiDBSchemaRuleStatementRequireWhere require 'WHERE' clause.
	TiDBSchemaRuleStatementRequireWhere SQLReviewRuleType = "tidb.statement.where.require"
	// TiDBSchemaRuleStatementNoLeadingWildcardLike disallow leading '%' in LIKE, e.g. LIKE foo = '%x' is not allowed.
	TiDBSchemaRuleStatementNoLeadingWildcardLike SQLReviewRuleType = "tidb.statement.where.no-leading-wildcard-like"
	// TiDBSchemaRuleStatementDisallowCommit disallow using commit in the issue.
	TiDBSchemaRuleStatementDisallowCommit SQLReviewRuleType = "tidb.statement.disallow-commit"
	// TiDBSchemaRuleStatementDisallowLimit disallow the LIMIT clause in INSERT, DELETE and UPDATE statements.
	TiDBSchemaRuleStatementDisallowLimit SQLReviewRuleType = "tidb.statement.disallow-limit"
	// TiDBSchemaRuleStatementDisallowOrderBy disallow the ORDER BY clause in DELETE and UPDATE statements.
	TiDBSchemaRuleStatementDisallowOrderBy SQLReviewRuleType = "tidb.statement.disallow-order-by"
	// TiDBSchemaRuleStatementMergeAlterTable disallow redundant ALTER TABLE statements.
	TiDBSchemaRuleStatementMergeAlterTable SQLReviewRuleType = "tidb.statement.merge-alter-table"
	// TiDBSchemaRuleStatementInsertMustSpecifyColumn enforce the insert column specified.
	TiDBSchemaRuleStatementInsertMustSpecifyColumn SQLReviewRuleType = "tidb.statement.insert.must-specify-column"
	// TiDBSchemaRuleStatementInsertDisallowOrderByRand disallow the order by rand in the INSERT statement.
	TiDBSchemaRuleStatementInsertDisallowOrderByRand SQLReviewRuleType = "tidb.statement.insert.disallow-order-by-rand"
	// TiDBSchemaRuleStatementDMLDryRun dry run the dml.
	TiDBSchemaRuleStatementDMLDryRun SQLReviewRuleType = "tidb.statement.dml-dry-run"

	// TiDBSchemaRuleTableRequirePK require the table to have a primary key.
	TiDBSchemaRuleTableRequirePK SQLReviewRuleType = "tidb.table.require-pk"
	// TiDBSchemaRuleTableNoFK require the table disallow the foreign key.
	TiDBSchemaRuleTableNoFK SQLReviewRuleType = "tidb.table.no-foreign-key"
	// TiDBSchemaRuleTableDropNamingConvention require only the table following the naming convention can be deleted.
	TiDBSchemaRuleTableDropNamingConvention SQLReviewRuleType = "tidb.table.drop-naming-convention"
	// TiDBSchemaRuleTableCommentConvention enforce the table comment convention.
	TiDBSchemaRuleTableCommentConvention SQLReviewRuleType = "tidb.table.comment"
	// TiDBSchemaRuleTableDisallowPartition disallow the table partition.
	TiDBSchemaRuleTableDisallowPartition SQLReviewRuleType = "tidb.table.disallow-partition"

	// TiDBSchemaRuleRequiredColumn enforce the required columns in each table.
	TiDBSchemaRuleRequiredColumn SQLReviewRuleType = "tidb.column.required"
	// TiDBSchemaRuleColumnNotNull enforce the columns cannot have NULL value.
	TiDBSchemaRuleColumnNotNull SQLReviewRuleType = "tidb.column.no-null"
	// TiDBSchemaRuleColumnDisallowChangeType disallow change column type.
	TiDBSchemaRuleColumnDisallowChangeType SQLReviewRuleType = "tidb.column.disallow-change-type"
	// TiDBSchemaRuleColumnSetDefaultForNotNull require the not null column to set default value.
	TiDBSchemaRuleColumnSetDefaultForNotNull SQLReviewRuleType = "tidb.column.set-default-for-not-null"
	// TiDBSchemaRuleColumnDisallowChange disallow CHANGE COLUMN statement.
	TiDBSchemaRuleColumnDisallowChange SQLReviewRuleType = "tidb.column.disallow-change"
	// TiDBSchemaRuleColumnDisallowChangingOrder disallow changing column order.
	TiDBSchemaRuleColumnDisallowChangingOrder SQLReviewRuleType = "tidb.column.disallow-changing-order"
	// TiDBSchemaRuleColumnCommentConvention enforce the column comment convention.
	TiDBSchemaRuleColumnCommentConvention SQLReviewRuleType = "tidb.column.comment"
	// TiDBSchemaRuleColumnAutoIncrementMustInteger require the auto-increment column to be integer.
	TiDBSchemaRuleColumnAutoIncrementMustInteger SQLReviewRuleType = "tidb.column.auto-increment-must-integer"
	// TiDBSchemaRuleColumnTypeDisallowList enforce the column type disallow list.
	TiDBSchemaRuleColumnTypeDisallowList SQLReviewRuleType = "tidb.column.type-disallow-list"
	// TiDBSchemaRuleColumnDisallowSetCharset disallow set column charset.
	TiDBSchemaRuleColumnDisallowSetCharset SQLReviewRuleType = "tidb.column.disallow-set-charset"
	// TiDBSchemaRuleColumnMaximumCharacterLength enforce the maximum character length.
	TiDBSchemaRuleColumnMaximumCharacterLength SQLReviewRuleType = "tidb.column.maximum-character-length"
	// TiDBSchemaRuleColumnAutoIncrementInitialValue enforce the initial auto-increment value.
	TiDBSchemaRuleColumnAutoIncrementInitialValue SQLReviewRuleType = "tidb.column.auto-increment-initial-value"
	// TiDBSchemaRuleColumnAutoIncrementMustUnsigned enforce the auto-increment column to be unsigned.
	TiDBSchemaRuleColumnAutoIncrementMustUnsigned SQLReviewRuleType = "tidb.column.auto-increment-must-unsigned"
	// TiDBSchemaRuleCurrentTimeColumnCountLimit enforce the current column count limit.
	TiDBSchemaRuleCurrentTimeColumnCountLimit SQLReviewRuleType = "tidb.column.current-time-count-limit"
	// TiDBSchemaRuleColumnRequireDefault enforce the column default.
	TiDBSchemaRuleColumnRequireDefault SQLReviewRuleType = "tidb.column.require-default"

	// TiDBSchemaRuleSchemaBackwardCompatibility enforce the TiDB supports check whether the schema change is backward compatible.
	TiDBSchemaRuleSchemaBackwardCompatibility SQLReviewRuleType = "tidb.schema.backward-compatibility"

	// TiDBSchemaRuleDropEmptyDatabase enforce the TiDB supports check if the database is empty before users drop it.
	TiDBSchemaRuleDropEmptyDatabase SQLReviewRuleType = "tidb.database.drop-empty-database"

	// TiDBSchemaRuleIndexNoDuplicateColumn require the index no duplicate column.
	TiDBSchemaRuleIndexNoDuplicateColumn SQLReviewRuleType = "tidb.index.no-duplicate-column"
	// TiDBSchemaRuleIndexKeyNumberLimit enforce the index key number limit.
	TiDBSchemaRuleIndexKeyNumberLimit SQLReviewRuleType = "tidb.index.key-number-limit"
	// TiDBSchemaRuleIndexPKTypeLimit enforce the type restriction of columns in primary key.
	TiDBSchemaRuleIndexPKTypeLimit SQLReviewRuleType = "tidb.index.pk-type-limit"
	// TiDBSchemaRuleIndexTypeNoBlob enforce the type restriction of columns in index.
	TiDBSchemaRuleIndexTypeNoBlob SQLReviewRuleType = "tidb.index.type-no-blob"
	// TiDBSchemaRuleIndexTotalNumberLimit enforce the index total number limit.
	TiDBSchemaRuleIndexTotalNumberLimit SQLReviewRuleType = "tidb.index.total-number-limit"

	// TiDBSchemaRuleCharsetAllowlist enforce the charset allowlist.
	TiDBSchemaRuleCharsetAllowlist SQLReviewRuleType = "tidb.system.charset.allowlist"

	// TiDBSchemaRuleCollationAllowlist enforce the collation allowlist.
	TiDBSchemaRuleCollationAllowlist SQLReviewRuleType = "tidb.system.collation.allowlist"

	// PostgreSQL rule list.

	// PostgreSQLSchemaRuleTableNaming enforce the table name format.
	PostgreSQLSchemaRuleTableNaming SQLReviewRuleType = "pg.naming.table"
	// PostgreSQLSchemaRuleColumnNaming enforce the column name format.
	PostgreSQLSchemaRuleColumnNaming SQLReviewRuleType = "pg.naming.column"
	// PostgreSQLSchemaRulePKNaming enforce the primary key name format.
	PostgreSQLSchemaRulePKNaming SQLReviewRuleType = "pg.naming.index.pk"
	// PostgreSQLSchemaRuleUKNaming enforce the unique key name format.
	PostgreSQLSchemaRuleUKNaming SQLReviewRuleType = "pg.naming.index.uk"
	// PostgreSQLSchemaRuleFKNaming enforce the foreign key name format.
	PostgreSQLSchemaRuleFKNaming SQLReviewRuleType = "pg.naming.index.fk"
	// PostgreSQLSchemaRuleIDXNaming enforce the index name format.
	PostgreSQLSchemaRuleIDXNaming SQLReviewRuleType = "pg.naming.index.idx"

	// PostgreSQLSchemaRuleStatementNoSelectAll disallow 'SELECT *'.
	PostgreSQLSchemaRuleStatementNoSelectAll SQLReviewRuleType = "pg.statement.select.no-select-all"
	// PostgreSQLSchemaRuleStatementRequireWhere require 'WHERE' clause.
	PostgreSQLSchemaRuleStatementRequireWhere SQLReviewRuleType = "pg.statement.where.require"
	// PostgreSQLSchemaRuleStatementNoLeadingWildcardLike disallow leading '%' in LIKE, e.g. LIKE foo = '%x' is not allowed.
	PostgreSQLSchemaRuleStatementNoLeadingWildcardLike SQLReviewRuleType = "pg.statement.where.no-leading-wildcard-like"
	// PostgreSQLSchemaRuleStatementDisallowCommit disallow using commit in the issue.
	PostgreSQLSchemaRuleStatementDisallowCommit SQLReviewRuleType = "pg.statement.disallow-commit"
	// PostgreSQLSchemaRuleStatementMergeAlterTable disallow redundant ALTER TABLE statements.
	PostgreSQLSchemaRuleStatementMergeAlterTable SQLReviewRuleType = "pg.statement.merge-alter-table"
	// PostgreSQLSchemaRuleStatementInsertRowLimit enforce the insert row limit.
	PostgreSQLSchemaRuleStatementInsertRowLimit SQLReviewRuleType = "pg.statement.insert.row-limit"
	// PostgreSQLSchemaRuleStatementInsertMustSpecifyColumn enforce the insert column specified.
	PostgreSQLSchemaRuleStatementInsertMustSpecifyColumn SQLReviewRuleType = "pg.statement.insert.must-specify-column"
	// PostgreSQLSchemaRuleStatementInsertDisallowOrderByRand disallow the order by rand in the INSERT statement.
	PostgreSQLSchemaRuleStatementInsertDisallowOrderByRand SQLReviewRuleType = "pg.statement.insert.disallow-order-by-rand"
	// PostgreSQLSchemaRuleStatementAffectedRowLimit enforce the UPDATE/DELETE affected row limit.
	PostgreSQLSchemaRuleStatementAffectedRowLimit SQLReviewRuleType = "pg.statement.affected-row-limit"
	// PostgreSQLSchemaRuleStatementDMLDryRun dry run the dml.
	PostgreSQLSchemaRuleStatementDMLDryRun SQLReviewRuleType = "pg.statement.dml-dry-run"
	// PostgreSQLSchemaRuleStatementDisallowAddColumnWithDefault disallow to add column with DEFAULT.
	PostgreSQLSchemaRuleStatementDisallowAddColumnWithDefault SQLReviewRuleType = "pg.statement.disallow-add-column-with-default"
	// PostgreSQLSchemaRuleStatementAddCheckNotValid require add check constraints not valid.
	PostgreSQLSchemaRuleStatementAddCheckNotValid SQLReviewRuleType = "pg.statement.add-check-not-valid"
	// PostgreSQLSchemaRuleStatementDisallowAddNotNull disallow to add NOT NULL.
	PostgreSQLSchemaRuleStatementDisallowAddNotNull SQLReviewRuleType = "pg.statement.disallow-add-not-null"

	// PostgreSQLSchemaRuleTableRequirePK require the table to have a primary key.
	PostgreSQLSchemaRuleTableRequirePK SQLReviewRuleType = "pg.table.require-pk"
	// PostgreSQLSchemaRuleTableNoFK require the table disallow the foreign key.
	PostgreSQLSchemaRuleTableNoFK SQLReviewRuleType = "pg.table.no-foreign-key"
	// PostgreSQLSchemaRuleTableDropNamingConvention require only the table following the naming convention can be deleted.
	PostgreSQLSchemaRuleTableDropNamingConvention SQLReviewRuleType = "pg.table.drop-naming-convention"
	// PostgreSQLSchemaRuleTableDisallowPartition disallow the table partition.
	PostgreSQLSchemaRuleTableDisallowPartition SQLReviewRuleType = "pg.table.disallow-partition"

	// PostgreSQLSchemaRuleRequiredColumn enforce the required columns in each table.
	PostgreSQLSchemaRuleRequiredColumn SQLReviewRuleType = "pg.column.required"
	// PostgreSQLSchemaRuleColumnNotNull enforce the columns cannot have NULL value.
	PostgreSQLSchemaRuleColumnNotNull SQLReviewRuleType = "pg.column.no-null"
	// PostgreSQLSchemaRuleColumnDisallowChangeType disallow change column type.
	PostgreSQLSchemaRuleColumnDisallowChangeType SQLReviewRuleType = "pg.column.disallow-change-type"
	// PostgreSQLSchemaRuleColumnTypeDisallowList enforce the column type disallow list.
	PostgreSQLSchemaRuleColumnTypeDisallowList SQLReviewRuleType = "pg.column.type-disallow-list"
	// PostgreSQLSchemaRuleColumnMaximumCharacterLength enforce the maximum character length.
	PostgreSQLSchemaRuleColumnMaximumCharacterLength SQLReviewRuleType = "pg.column.maximum-character-length"
	// PostgreSQLSchemaRuleColumnRequireDefault enforce the column default.
	PostgreSQLSchemaRuleColumnRequireDefault SQLReviewRuleType = "pg.column.require-default"

	// PostgreSQLSchemaRuleSchemaBackwardCompatibility enforce the PostgreSQL supports check whether the schema change is backward compatible.
	PostgreSQLSchemaRuleSchemaBackwardCompatibility SQLReviewRuleType = "pg.schema.backward-compatibility"

	// PostgreSQLSchemaRuleIndexNoDuplicateColumn require the index no duplicate column.
	PostgreSQLSchemaRuleIndexNoDuplicateColumn SQLReviewRuleType = "pg.index.no-duplicate-column"
	// PostgreSQLSchemaRuleIndexKeyNumberLimit enforce the index key number limit.
	PostgreSQLSchemaRuleIndexKeyNumberLimit SQLReviewRuleType = "pg.index.key-number-limit"
	// PostgreSQLSchemaRuleIndexTotalNumberLimit enforce the index total number limit.
	PostgreSQLSchemaRuleIndexTotalNumberLimit SQLReviewRuleType = "pg.index.total-number-limit"
	// PostgreSQLSchemaRuleIndexPrimaryKeyTypeAllowlist enforce the primary key type allowlist.
	PostgreSQLSchemaRuleIndexPrimaryKeyTypeAllowlist SQLReviewRuleType = "pg.index.primary-key-type-allowlist"
	// PostgreSQLSchemaRuleCreateIndexConcurrently require creating indexes concurrently.
	PostgreSQLSchemaRuleCreateIndexConcurrently SQLReviewRuleType = "pg.index.create-concurrently"

	// PostgreSQLSchemaRuleCharsetAllowlist enforce the charset allowlist.
	PostgreSQLSchemaRuleCharsetAllowlist SQLReviewRuleType = "pg.system.charset.allowlist"

	// PostgreSQLSchemaRuleCollationAllowlist enforce the collation allowlist.
	PostgreSQLSchemaRuleCollationAllowlist SQLReviewRuleType = "pg.system.collation.allowlist"

	// PostgreSQLSchemaRuleCommentLength limit comment length.
	PostgreSQLSchemaRuleCommentLength SQLReviewRuleType = "pg.comment.length"

	// SchemaRuleLevelError is the error level of SQLReviewRuleLevel.
	SchemaRuleLevelError SQLReviewRuleLevel = "ERROR"
	// SchemaRuleLevelWarning is the warning level of SQLReviewRuleLevel.
	SchemaRuleLevelWarning SQLReviewRuleLevel = "WARNING"
	// SchemaRuleLevelDisabled is the disabled level of SQLReviewRuleLevel.
	SchemaRuleLevelDisabled SQLReviewRuleLevel = "DISABLED"

	// TableNameTemplateToken is the token for table name.
	TableNameTemplateToken = "{{table}}"
	// ColumnListTemplateToken is the token for column name list.
	ColumnListTemplateToken = "{{column_list}}"
	// ReferencingTableNameTemplateToken is the token for referencing table name.
	ReferencingTableNameTemplateToken = "{{referencing_table}}"
	// ReferencingColumnNameTemplateToken is the token for referencing column name.
	ReferencingColumnNameTemplateToken = "{{referencing_column}}"
	// ReferencedTableNameTemplateToken is the token for referenced table name.
	ReferencedTableNameTemplateToken = "{{referenced_table}}"
	// ReferencedColumnNameTemplateToken is the token for referenced column name.
	ReferencedColumnNameTemplateToken = "{{referenced_column}}"

	// defaultNameLengthLimit is the default length limit for naming rules.
	// PostgreSQL has it's own naming length limit, will auto slice the name to make sure its length <= 63
	// https://www.postgresql.org/docs/current/limits.html.
	// While MySQL does not enforce the limit, thus we use PostgreSQL's 63 as the default limit.
	defaultNameLengthLimit = 63
)

var (
	// TemplateNamingTokens is the mapping for rule type to template token.
	TemplateNamingTokens = map[SQLReviewRuleType]map[string]bool{
		MySQLSchemaRuleIDXNaming: {
			TableNameTemplateToken:  true,
			ColumnListTemplateToken: true,
		},
		TiDBSchemaRuleIDXNaming: {
			TableNameTemplateToken:  true,
			ColumnListTemplateToken: true,
		},
		PostgreSQLSchemaRuleIDXNaming: {
			TableNameTemplateToken:  true,
			ColumnListTemplateToken: true,
		},
		PostgreSQLSchemaRulePKNaming: {
			TableNameTemplateToken:  true,
			ColumnListTemplateToken: true,
		},
		MySQLSchemaRuleUKNaming: {
			TableNameTemplateToken:  true,
			ColumnListTemplateToken: true,
		},
		TiDBSchemaRuleUKNaming: {
			TableNameTemplateToken:  true,
			ColumnListTemplateToken: true,
		},
		PostgreSQLSchemaRuleUKNaming: {
			TableNameTemplateToken:  true,
			ColumnListTemplateToken: true,
		},
		MySQLSchemaRuleFKNaming: {
			ReferencingTableNameTemplateToken:  true,
			ReferencingColumnNameTemplateToken: true,
			ReferencedTableNameTemplateToken:   true,
			ReferencedColumnNameTemplateToken:  true,
		},
		TiDBSchemaRuleFKNaming: {
			ReferencingTableNameTemplateToken:  true,
			ReferencingColumnNameTemplateToken: true,
			ReferencedTableNameTemplateToken:   true,
			ReferencedColumnNameTemplateToken:  true,
		},
		PostgreSQLSchemaRuleFKNaming: {
			ReferencingTableNameTemplateToken:  true,
			ReferencingColumnNameTemplateToken: true,
			ReferencedTableNameTemplateToken:   true,
			ReferencedColumnNameTemplateToken:  true,
		},
	}
)

// SQLReviewPolicy is the policy configuration for SQL review.
type SQLReviewPolicy struct {
	Name     string           `json:"name"`
	RuleList []*SQLReviewRule `json:"ruleList"`
}

// Validate validates the SQLReviewPolicy. It also validates the each review rule.
func (policy *SQLReviewPolicy) Validate() error {
	if policy.Name == "" || len(policy.RuleList) == 0 {
		return errors.Errorf("invalid payload, name or rule list cannot be empty")
	}
	for _, rule := range policy.RuleList {
		if err := rule.Validate(); err != nil {
			return err
		}
	}
	return nil
}

// String returns the marshal string value for SQL review policy.
func (policy *SQLReviewPolicy) String() (string, error) {
	s, err := json.Marshal(policy)
	if err != nil {
		return "", err
	}
	return string(s), nil
}

// SQLReviewRule is the rule for SQL review policy.
type SQLReviewRule struct {
	Type  SQLReviewRuleType  `json:"type"`
	Level SQLReviewRuleLevel `json:"level"`
	// Payload is the stringify value for XXXRulePayload (e.g. NamingRulePayload, StringArrayTypeRulePayload)
	// If the rule doesn't have any payload configuration, the payload would be "{}"
	Payload string `json:"payload"`
}

// Validate validates the SQL review rule.
func (rule *SQLReviewRule) Validate() error {
	// TODO(rebelice): add other SQL review rule validation.
	switch rule.Type {
	case MySQLSchemaRuleTableNaming, MySQLSchemaRuleColumnNaming, MySQLSchemaRuleAutoIncrementColumnNaming,
		PostgreSQLSchemaRuleTableNaming, PostgreSQLSchemaRuleColumnNaming,
		TiDBSchemaRuleTableNaming, TiDBSchemaRuleColumnNaming, TiDBSchemaRuleAutoIncrementColumnNaming:
		if _, _, err := UnamrshalNamingRulePayloadAsRegexp(rule.Payload); err != nil {
			return err
		}
	case MySQLSchemaRuleFKNaming, MySQLSchemaRuleIDXNaming, MySQLSchemaRuleUKNaming,
		TiDBSchemaRuleFKNaming, TiDBSchemaRuleIDXNaming, TiDBSchemaRuleUKNaming,
		PostgreSQLSchemaRuleFKNaming, PostgreSQLSchemaRuleIDXNaming, PostgreSQLSchemaRuleUKNaming, PostgreSQLSchemaRulePKNaming:
		if _, _, _, err := UnmarshalNamingRulePayloadAsTemplate(rule.Type, rule.Payload); err != nil {
			return err
		}
	case MySQLSchemaRuleRequiredColumn, PostgreSQLSchemaRuleRequiredColumn, TiDBSchemaRuleRequiredColumn:
		if _, err := UnmarshalRequiredColumnList(rule.Payload); err != nil {
			return err
		}
	case MySQLSchemaRuleColumnCommentConvention, MySQLSchemaRuleTableCommentConvention,
		TiDBSchemaRuleColumnCommentConvention, TiDBSchemaRuleTableCommentConvention:
		if _, err := UnmarshalCommentConventionRulePayload(rule.Payload); err != nil {
			return err
		}
	case MySQLSchemaRuleIndexKeyNumberLimit, MySQLSchemaRuleStatementInsertRowLimit, MySQLSchemaRuleIndexTotalNumberLimit,
		MySQLSchemaRuleColumnMaximumCharacterLength, MySQLSchemaRuleColumnAutoIncrementInitialValue, MySQLSchemaRuleStatementAffectedRowLimit,
		TiDBSchemaRuleIndexKeyNumberLimit, TiDBSchemaRuleIndexTotalNumberLimit,
		TiDBSchemaRuleColumnMaximumCharacterLength, TiDBSchemaRuleColumnAutoIncrementInitialValue,
		PostgreSQLSchemaRuleIndexKeyNumberLimit, PostgreSQLSchemaRuleStatementInsertRowLimit, PostgreSQLSchemaRuleIndexTotalNumberLimit,
		PostgreSQLSchemaRuleColumnMaximumCharacterLength, PostgreSQLSchemaRuleStatementAffectedRowLimit:
		if _, err := UnmarshalNumberTypeRulePayload(rule.Payload); err != nil {
			return err
		}
	case MySQLSchemaRuleColumnTypeDisallowList, MySQLSchemaRuleCharsetAllowlist, MySQLSchemaRuleCollationAllowlist,
		TiDBSchemaRuleColumnTypeDisallowList, TiDBSchemaRuleCharsetAllowlist, TiDBSchemaRuleCollationAllowlist,
		PostgreSQLSchemaRuleColumnTypeDisallowList, PostgreSQLSchemaRuleCharsetAllowlist, PostgreSQLSchemaRuleCollationAllowlist, PostgreSQLSchemaRuleIndexPrimaryKeyTypeAllowlist:
		if _, err := UnmarshalStringArrayTypeRulePayload(rule.Payload); err != nil {
			return err
		}
	}
	return nil
}

// NamingRulePayload is the payload for naming rule.
type NamingRulePayload struct {
	MaxLength int    `json:"maxLength"`
	Format    string `json:"format"`
}

// StringArrayTypeRulePayload is the payload for rules with string array value.
type StringArrayTypeRulePayload struct {
	List []string `json:"list"`
}

// RequiredColumnRulePayload is the payload for required column rule.
type RequiredColumnRulePayload struct {
	ColumnList []string `json:"columnList"`
}

// CommentConventionRulePayload is the payload for comment convention rule.
type CommentConventionRulePayload struct {
	Required  bool `json:"required"`
	MaxLength int  `json:"maxLength"`
}

// NumberTypeRulePayload is the number type payload.
type NumberTypeRulePayload struct {
	Number int `json:"number"`
}

// UnamrshalNamingRulePayloadAsRegexp will unmarshal payload to NamingRulePayload and compile it as regular expression.
func UnamrshalNamingRulePayloadAsRegexp(payload string) (*regexp.Regexp, int, error) {
	var nr NamingRulePayload
	if err := json.Unmarshal([]byte(payload), &nr); err != nil {
		return nil, 0, errors.Wrapf(err, "failed to unmarshal naming rule payload %q", payload)
	}

	format, err := regexp.Compile(nr.Format)
	if err != nil {
		return nil, 0, errors.Wrapf(err, "failed to compile regular expression \"%s\"", nr.Format)
	}

	// We need to be compatible with existed naming rules in the database. 0 means using the default length limit.
	maxLength := nr.MaxLength
	if maxLength == 0 {
		maxLength = defaultNameLengthLimit
	}

	return format, maxLength, nil
}

// UnmarshalNamingRulePayloadAsTemplate will unmarshal payload to NamingRulePayload and extract all the template keys.
// For example, "hard_code_{{table}}_{{column}}_end" will return
// "hard_code_{{table}}_{{column}}_end", ["{{table}}", "{{column}}"].
func UnmarshalNamingRulePayloadAsTemplate(ruleType SQLReviewRuleType, payload string) (string, []string, int, error) {
	var nr NamingRulePayload
	if err := json.Unmarshal([]byte(payload), &nr); err != nil {
		return "", nil, 0, errors.Wrapf(err, "failed to unmarshal naming rule payload %q", payload)
	}

	template := nr.Format
	keys, _ := parseTemplateTokens(template)

	for _, key := range keys {
		if _, ok := TemplateNamingTokens[ruleType][key]; !ok {
			return "", nil, 0, errors.Errorf("invalid template %s for rule %s", key, ruleType)
		}
	}

	// We need to be compatible with existed naming rules in the database. 0 means using the default length limit.
	maxLength := nr.MaxLength
	if maxLength == 0 {
		maxLength = defaultNameLengthLimit
	}

	return template, keys, maxLength, nil
}

// parseTemplateTokens parses the template and returns template tokens and their delimiters.
// For example, if the template is "{{DB_NAME}}_hello_{{LOCATION}}", then the tokens will be ["{{DB_NAME}}", "{{LOCATION}}"],
// and the delimiters will be ["_hello_"].
// The caller will usually replace the tokens with a normal string, or a regexp. In the latter case, it will be a problem
// if there are special regexp characters such as "$" in the delimiters. The caller should escape the delimiters in such cases.
func parseTemplateTokens(template string) ([]string, []string) {
	r := regexp.MustCompile(`{{[^{}]+}}`)
	tokens := r.FindAllString(template, -1)
	if len(tokens) > 0 {
		split := r.Split(template, -1)
		var delimiters []string
		for _, s := range split {
			if s != "" {
				delimiters = append(delimiters, s)
			}
		}
		return tokens, delimiters
	}
	return nil, nil
}

// UnmarshalRequiredColumnList will unmarshal payload and parse the required column list.
func UnmarshalRequiredColumnList(payload string) ([]string, error) {
	stringArrayRulePayload, err := UnmarshalStringArrayTypeRulePayload(payload)
	if err != nil {
		return nil, err
	}
	if len(stringArrayRulePayload.List) != 0 {
		return stringArrayRulePayload.List, nil
	}

	// The RequiredColumnRulePayload is deprecated.
	// Just keep it to compatible with old data, and we can remove this later.
	columnRulePayload, err := unmarshalRequiredColumnRulePayload(payload)
	if err != nil {
		return nil, err
	}

	return columnRulePayload.ColumnList, nil
}

// unmarshalRequiredColumnRulePayload will unmarshal payload to RequiredColumnRulePayload.
func unmarshalRequiredColumnRulePayload(payload string) (*RequiredColumnRulePayload, error) {
	var rcr RequiredColumnRulePayload
	if err := json.Unmarshal([]byte(payload), &rcr); err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal required column rule payload %q", payload)
	}
	if len(rcr.ColumnList) == 0 {
		return nil, errors.Errorf("invalid required column rule payload, column list cannot be empty")
	}
	return &rcr, nil
}

// UnmarshalCommentConventionRulePayload will unmarshal payload to CommentConventionRulePayload.
func UnmarshalCommentConventionRulePayload(payload string) (*CommentConventionRulePayload, error) {
	var ccr CommentConventionRulePayload
	if err := json.Unmarshal([]byte(payload), &ccr); err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal comment convention rule payload %q", payload)
	}
	return &ccr, nil
}

// UnmarshalNumberTypeRulePayload will unmarshal payload to NumberTypeRulePayload.
func UnmarshalNumberTypeRulePayload(payload string) (*NumberTypeRulePayload, error) {
	var nlr NumberTypeRulePayload
	if err := json.Unmarshal([]byte(payload), &nlr); err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal number type rule payload %q", payload)
	}
	return &nlr, nil
}

// UnmarshalStringArrayTypeRulePayload will unmarshal payload to StringArrayTypeRulePayload.
func UnmarshalStringArrayTypeRulePayload(payload string) (*StringArrayTypeRulePayload, error) {
	var trr StringArrayTypeRulePayload
	if err := json.Unmarshal([]byte(payload), &trr); err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal string array rule payload %q", payload)
	}
	return &trr, nil
}

// SQLReviewCheckContext is the context for SQL review check.
type SQLReviewCheckContext struct {
	Charset   string
	Collation string
	DbType    db.Type
	Catalog   catalog.Catalog
	Driver    *sql.DB
	Context   context.Context
}

// SQLReviewCheck checks the statements with sql review rules.
func SQLReviewCheck(statements string, ruleList []*SQLReviewRule, checkContext SQLReviewCheckContext) ([]Advice, error) {
	var result []Advice

	finder := checkContext.Catalog.GetFinder()
	switch checkContext.DbType {
	case db.TiDB, db.MySQL, db.Postgres:
		if err := finder.WalkThrough(statements); err != nil {
			return convertWalkThroughErrorToAdvice(err)
		}
	}

	for _, rule := range ruleList {
		if rule.Level == SchemaRuleLevelDisabled {
			continue
		}

		advisorType, err := getAdvisorTypeByRule(rule.Type)
		if err != nil {
			log.Printf("not supported rule: %v. error:  %v\n", rule.Type, err)
			continue
		}

		adviceList, err := Check(
			checkContext.DbType,
			advisorType,
			Context{
				Charset:   checkContext.Charset,
				Collation: checkContext.Collation,
				Rule:      rule,
				Catalog:   finder,
				Driver:    checkContext.Driver,
				Context:   checkContext.Context,
			},
			statements,
		)
		if err != nil {
			return nil, errors.Wrap(err, "failed to check statement")
		}

		result = append(result, adviceList...)
	}

	// There may be multiple syntax errors, return one only.
	if len(result) > 0 && result[0].Title == SyntaxErrorTitle {
		return result[:1], nil
	}
	if len(result) == 0 {
		result = append(result, Advice{
			Status:  Success,
			Code:    Ok,
			Title:   "OK",
			Content: "",
		})
	}
	return result, nil
}

func convertWalkThroughErrorToAdvice(err error) ([]Advice, error) {
	walkThroughError, ok := err.(*catalog.WalkThroughError)
	if !ok {
		return nil, err
	}

	var res []Advice
	switch walkThroughError.Type {
	case catalog.ErrorTypeUnsupported:
		res = append(res, Advice{
			Status:  Error,
			Code:    Unsupported,
			Title:   walkThroughError.Content,
			Content: "",
			Line:    walkThroughError.Line,
		})
	case catalog.ErrorTypeParseError:
		res = append(res, Advice{
			Status:  Error,
			Code:    StatementSyntaxError,
			Title:   SyntaxErrorTitle,
			Content: walkThroughError.Content,
		})
	case catalog.ErrorTypeDeparseError:
		res = append(res, Advice{
			Status:  Error,
			Code:    Internal,
			Title:   "Internal error for walk-through",
			Content: walkThroughError.Content,
			Line:    walkThroughError.Line,
		})
	case catalog.ErrorTypeAccessOtherDatabase:
		res = append(res, Advice{
			Status:  Error,
			Code:    NotCurrentDatabase,
			Title:   "Access other database",
			Content: walkThroughError.Content,
			Line:    walkThroughError.Line,
		})
	case catalog.ErrorTypeDatabaseIsDeleted:
		res = append(res, Advice{
			Status:  Error,
			Code:    DatabaseIsDeleted,
			Title:   "Access deleted database",
			Content: walkThroughError.Content,
			Line:    walkThroughError.Line,
		})
	case catalog.ErrorTypeTableExists:
		res = append(res, Advice{
			Status:  Error,
			Code:    TableExists,
			Title:   "Table already exists",
			Content: walkThroughError.Content,
			Line:    walkThroughError.Line,
		})
	case catalog.ErrorTypeTableNotExists:
		res = append(res, Advice{
			Status:  Error,
			Code:    TableNotExists,
			Title:   "Table does not exist",
			Content: walkThroughError.Content,
			Line:    walkThroughError.Line,
		})
	case catalog.ErrorTypeColumnExists:
		res = append(res, Advice{
			Status:  Error,
			Code:    ColumnExists,
			Title:   "Column already exists",
			Content: walkThroughError.Content,
			Line:    walkThroughError.Line,
		})
	case catalog.ErrorTypeColumnNotExists:
		res = append(res, Advice{
			Status:  Error,
			Code:    ColumnNotExists,
			Title:   "Column does not exist",
			Content: walkThroughError.Content,
			Line:    walkThroughError.Line,
		})
	case catalog.ErrorTypeDropAllColumns:
		res = append(res, Advice{
			Status:  Error,
			Code:    DropAllColumns,
			Title:   "Drop all columns",
			Content: walkThroughError.Content,
			Line:    walkThroughError.Line,
		})
	case catalog.ErrorTypePrimaryKeyExists:
		res = append(res, Advice{
			Status:  Error,
			Code:    PrimaryKeyExists,
			Title:   "Primary key exists",
			Content: walkThroughError.Content,
			Line:    walkThroughError.Line,
		})
	case catalog.ErrorTypeIndexExists:
		res = append(res, Advice{
			Status:  Error,
			Code:    IndexExists,
			Title:   "Index exists",
			Content: walkThroughError.Content,
			Line:    walkThroughError.Line,
		})
	case catalog.ErrorTypeIndexEmptyKeys:
		res = append(res, Advice{
			Status:  Error,
			Code:    IndexEmptyKeys,
			Title:   "Index empty keys",
			Content: walkThroughError.Content,
			Line:    walkThroughError.Line,
		})
	case catalog.ErrorTypePrimaryKeyNotExists:
		res = append(res, Advice{
			Status:  Error,
			Code:    PrimaryKeyNotExists,
			Title:   "Primary key does not exist",
			Content: walkThroughError.Content,
			Line:    walkThroughError.Line,
		})
	case catalog.ErrorTypeIndexNotExists:
		res = append(res, Advice{
			Status:  Error,
			Code:    IndexNotExists,
			Title:   "Index does not exist",
			Content: walkThroughError.Content,
			Line:    walkThroughError.Line,
		})
	case catalog.ErrorTypeIncorrectIndexName:
		res = append(res, Advice{
			Status:  Error,
			Code:    IncorrectIndexName,
			Title:   "Incorrect index name",
			Content: walkThroughError.Content,
			Line:    walkThroughError.Line,
		})
	case catalog.ErrorTypeSpatialIndexKeyNullable:
		res = append(res, Advice{
			Status:  Error,
			Code:    SpatialIndexKeyNullable,
			Title:   "Spatial index key must be NOT NULL",
			Content: walkThroughError.Content,
			Line:    walkThroughError.Line,
		})
	default:
		res = append(res, Advice{
			Status:  Error,
			Code:    Internal,
			Title:   "Failed to walk-through",
			Content: walkThroughError.Content,
			Line:    walkThroughError.Line,
		})
	}

	return res, nil
}

func getAdvisorTypeByRule(ruleType SQLReviewRuleType) (Type, error) {
	switch ruleType {
	case MySQLSchemaRuleStatementRequireWhere:
		return MySQLWhereRequirement, nil
	case TiDBSchemaRuleStatementRequireWhere:
		return MySQLWhereRequirement, nil
	case PostgreSQLSchemaRuleStatementRequireWhere:
		return PostgreSQLWhereRequirement, nil
	case MySQLSchemaRuleStatementNoLeadingWildcardLike:
		return MySQLNoLeadingWildcardLike, nil
	case TiDBSchemaRuleStatementNoLeadingWildcardLike:
		return MySQLNoLeadingWildcardLike, nil
	case PostgreSQLSchemaRuleStatementNoLeadingWildcardLike:
		return PostgreSQLNoLeadingWildcardLike, nil
	case MySQLSchemaRuleStatementNoSelectAll:
		return MySQLNoSelectAll, nil
	case TiDBSchemaRuleStatementNoSelectAll:
		return MySQLNoSelectAll, nil
	case PostgreSQLSchemaRuleStatementNoSelectAll:
		return PostgreSQLNoSelectAll, nil
	case MySQLSchemaRuleSchemaBackwardCompatibility:
		return MySQLMigrationCompatibility, nil
	case TiDBSchemaRuleSchemaBackwardCompatibility:
		return MySQLMigrationCompatibility, nil
	case PostgreSQLSchemaRuleSchemaBackwardCompatibility:
		return PostgreSQLMigrationCompatibility, nil
	case MySQLSchemaRuleTableNaming:
		return MySQLNamingTableConvention, nil
	case TiDBSchemaRuleTableNaming:
		return MySQLNamingTableConvention, nil
	case PostgreSQLSchemaRuleTableNaming:
		return PostgreSQLNamingTableConvention, nil
	case MySQLSchemaRuleIDXNaming:
		return MySQLNamingIndexConvention, nil
	case TiDBSchemaRuleIDXNaming:
		return MySQLNamingIndexConvention, nil
	case PostgreSQLSchemaRuleIDXNaming:
		return PostgreSQLNamingIndexConvention, nil
	case PostgreSQLSchemaRulePKNaming:
		return PostgreSQLNamingPKConvention, nil
	case MySQLSchemaRuleUKNaming:
		return MySQLNamingUKConvention, nil
	case TiDBSchemaRuleUKNaming:
		return MySQLNamingUKConvention, nil
	case PostgreSQLSchemaRuleUKNaming:
		return PostgreSQLNamingUKConvention, nil
	case MySQLSchemaRuleFKNaming:
		return MySQLNamingFKConvention, nil
	case TiDBSchemaRuleFKNaming:
		return MySQLNamingFKConvention, nil
	case PostgreSQLSchemaRuleFKNaming:
		return PostgreSQLNamingFKConvention, nil
	case MySQLSchemaRuleColumnNaming:
		return MySQLNamingColumnConvention, nil
	case TiDBSchemaRuleColumnNaming:
		return MySQLNamingColumnConvention, nil
	case PostgreSQLSchemaRuleColumnNaming:
		return PostgreSQLNamingColumnConvention, nil
	case MySQLSchemaRuleAutoIncrementColumnNaming:
		return MySQLNamingAutoIncrementColumnConvention, nil
	case TiDBSchemaRuleAutoIncrementColumnNaming:
		return MySQLNamingAutoIncrementColumnConvention, nil
	case MySQLSchemaRuleRequiredColumn:
		return MySQLColumnRequirement, nil
	case TiDBSchemaRuleRequiredColumn:
		return MySQLColumnRequirement, nil
	case PostgreSQLSchemaRuleRequiredColumn:
		return PostgreSQLColumnRequirement, nil
	case MySQLSchemaRuleColumnNotNull:
		return MySQLColumnNoNull, nil
	case TiDBSchemaRuleColumnNotNull:
		return MySQLColumnNoNull, nil
	case PostgreSQLSchemaRuleColumnNotNull:
		return PostgreSQLColumnNoNull, nil
	case MySQLSchemaRuleColumnDisallowChangeType:
		return MySQLColumnDisallowChangingType, nil
	case TiDBSchemaRuleColumnDisallowChangeType:
		return MySQLColumnDisallowChangingType, nil
	case PostgreSQLSchemaRuleColumnDisallowChangeType:
		return PostgreSQLColumnDisallowChangingType, nil
	case MySQLSchemaRuleColumnSetDefaultForNotNull:
		return MySQLColumnSetDefaultForNotNull, nil
	case TiDBSchemaRuleColumnSetDefaultForNotNull:
		return MySQLColumnSetDefaultForNotNull, nil
	case MySQLSchemaRuleColumnDisallowChange:
		return MySQLColumnDisallowChanging, nil
	case TiDBSchemaRuleColumnDisallowChange:
		return MySQLColumnDisallowChanging, nil
	case MySQLSchemaRuleColumnDisallowChangingOrder:
		return MySQLColumnDisallowChangingOrder, nil
	case TiDBSchemaRuleColumnDisallowChangingOrder:
		return MySQLColumnDisallowChangingOrder, nil
	case MySQLSchemaRuleColumnCommentConvention:
		return MySQLColumnCommentConvention, nil
	case TiDBSchemaRuleColumnCommentConvention:
		return MySQLColumnCommentConvention, nil
	case MySQLSchemaRuleColumnAutoIncrementMustInteger:
		return MySQLAutoIncrementColumnMustInteger, nil
	case TiDBSchemaRuleColumnAutoIncrementMustInteger:
		return MySQLAutoIncrementColumnMustInteger, nil
	case MySQLSchemaRuleColumnTypeDisallowList:
		return MySQLColumnTypeRestriction, nil
	case TiDBSchemaRuleColumnTypeDisallowList:
		return MySQLColumnTypeRestriction, nil
	case PostgreSQLSchemaRuleColumnTypeDisallowList:
		return PostgreSQLColumnTypeDisallowList, nil
	case MySQLSchemaRuleColumnDisallowSetCharset:
		return MySQLDisallowSetColumnCharset, nil
	case TiDBSchemaRuleColumnDisallowSetCharset:
		return MySQLDisallowSetColumnCharset, nil
	case MySQLSchemaRuleColumnMaximumCharacterLength:
		return MySQLColumnMaximumCharacterLength, nil
	case TiDBSchemaRuleColumnMaximumCharacterLength:
		return MySQLColumnMaximumCharacterLength, nil
	case PostgreSQLSchemaRuleColumnMaximumCharacterLength:
		return PostgreSQLColumnMaximumCharacterLength, nil
	case MySQLSchemaRuleColumnAutoIncrementInitialValue:
		return MySQLAutoIncrementColumnInitialValue, nil
	case TiDBSchemaRuleColumnAutoIncrementInitialValue:
		return MySQLAutoIncrementColumnInitialValue, nil
	case MySQLSchemaRuleColumnAutoIncrementMustUnsigned:
		return MySQLAutoIncrementColumnMustUnsigned, nil
	case TiDBSchemaRuleColumnAutoIncrementMustUnsigned:
		return MySQLAutoIncrementColumnMustUnsigned, nil
	case MySQLSchemaRuleCurrentTimeColumnCountLimit:
		return MySQLCurrentTimeColumnCountLimit, nil
	case TiDBSchemaRuleCurrentTimeColumnCountLimit:
		return MySQLCurrentTimeColumnCountLimit, nil
	case MySQLSchemaRuleColumnRequireDefault:
		return MySQLRequireColumnDefault, nil
	case TiDBSchemaRuleColumnRequireDefault:
		return MySQLRequireColumnDefault, nil
	case PostgreSQLSchemaRuleColumnRequireDefault:
		return PostgreSQLRequireColumnDefault, nil
	case MySQLSchemaRuleTableRequirePK:
		return MySQLTableRequirePK, nil
	case TiDBSchemaRuleTableRequirePK:
		return MySQLTableRequirePK, nil
	case PostgreSQLSchemaRuleTableRequirePK:
		return PostgreSQLTableRequirePK, nil
	case MySQLSchemaRuleTableNoFK:
		return MySQLTableNoFK, nil
	case TiDBSchemaRuleTableNoFK:
		return MySQLTableNoFK, nil
	case PostgreSQLSchemaRuleTableNoFK:
		return PostgreSQLTableNoFK, nil
	case MySQLSchemaRuleTableDropNamingConvention:
		return MySQLTableDropNamingConvention, nil
	case TiDBSchemaRuleTableDropNamingConvention:
		return MySQLTableDropNamingConvention, nil
	case PostgreSQLSchemaRuleTableDropNamingConvention:
		return PostgreSQLTableDropNamingConvention, nil
	case MySQLSchemaRuleTableCommentConvention:
		return MySQLTableCommentConvention, nil
	case TiDBSchemaRuleTableCommentConvention:
		return MySQLTableCommentConvention, nil
	case MySQLSchemaRuleTableDisallowPartition:
		return MySQLTableDisallowPartition, nil
	case TiDBSchemaRuleTableDisallowPartition:
		return MySQLTableDisallowPartition, nil
	case PostgreSQLSchemaRuleTableDisallowPartition:
		return PostgreSQLTableDisallowPartition, nil
	case MySQLSchemaRuleMySQLEngine:
		return MySQLUseInnoDB, nil
	case MySQLSchemaRuleDropEmptyDatabase:
		return MySQLDatabaseAllowDropIfEmpty, nil
	case TiDBSchemaRuleDropEmptyDatabase:
		return MySQLDatabaseAllowDropIfEmpty, nil
	case MySQLSchemaRuleIndexNoDuplicateColumn:
		return MySQLIndexNoDuplicateColumn, nil
	case TiDBSchemaRuleIndexNoDuplicateColumn:
		return MySQLIndexNoDuplicateColumn, nil
	case PostgreSQLSchemaRuleIndexNoDuplicateColumn:
		return PostgreSQLIndexNoDuplicateColumn, nil
	case MySQLSchemaRuleIndexKeyNumberLimit:
		return MySQLIndexKeyNumberLimit, nil
	case TiDBSchemaRuleIndexKeyNumberLimit:
		return MySQLIndexKeyNumberLimit, nil
	case PostgreSQLSchemaRuleIndexKeyNumberLimit:
		return PostgreSQLIndexKeyNumberLimit, nil
	case MySQLSchemaRuleIndexTotalNumberLimit:
		return MySQLIndexTotalNumberLimit, nil
	case TiDBSchemaRuleIndexTotalNumberLimit:
		return MySQLIndexTotalNumberLimit, nil
	case PostgreSQLSchemaRuleIndexTotalNumberLimit:
		return PostgreSQLIndexTotalNumberLimit, nil
	case MySQLSchemaRuleStatementDisallowCommit:
		return MySQLStatementDisallowCommit, nil
	case TiDBSchemaRuleStatementDisallowCommit:
		return MySQLStatementDisallowCommit, nil
	case PostgreSQLSchemaRuleStatementDisallowCommit:
		return PostgreSQLStatementDisallowCommit, nil
	case MySQLSchemaRuleCharsetAllowlist:
		return MySQLCharsetAllowlist, nil
	case TiDBSchemaRuleCharsetAllowlist:
		return MySQLCharsetAllowlist, nil
	case PostgreSQLSchemaRuleCharsetAllowlist:
		return PostgreSQLEncodingAllowlist, nil
	case MySQLSchemaRuleCollationAllowlist:
		return MySQLCollationAllowlist, nil
	case TiDBSchemaRuleCollationAllowlist:
		return MySQLCollationAllowlist, nil
	case PostgreSQLSchemaRuleCollationAllowlist:
		return PostgreSQLCollationAllowlist, nil
	case MySQLSchemaRuleIndexPKTypeLimit:
		return MySQLIndexPKType, nil
	case TiDBSchemaRuleIndexPKTypeLimit:
		return MySQLIndexPKType, nil
	case MySQLSchemaRuleIndexTypeNoBlob:
		return MySQLIndexTypeNoBlob, nil
	case TiDBSchemaRuleIndexTypeNoBlob:
		return MySQLIndexTypeNoBlob, nil
	case PostgreSQLSchemaRuleIndexPrimaryKeyTypeAllowlist:
		return PostgreSQLPrimaryKeyTypeAllowlist, nil
	case PostgreSQLSchemaRuleCreateIndexConcurrently:
		return PostgreSQLCreateIndexConcurrently, nil
	case MySQLSchemaRuleStatementInsertRowLimit:
		return MySQLInsertRowLimit, nil
	case PostgreSQLSchemaRuleStatementInsertRowLimit:
		return PostgreSQLInsertRowLimit, nil
	case MySQLSchemaRuleStatementInsertMustSpecifyColumn:
		return MySQLInsertMustSpecifyColumn, nil
	case TiDBSchemaRuleStatementInsertMustSpecifyColumn:
		return MySQLInsertMustSpecifyColumn, nil
	case PostgreSQLSchemaRuleStatementInsertMustSpecifyColumn:
		return PostgreSQLInsertMustSpecifyColumn, nil
	case MySQLSchemaRuleStatementInsertDisallowOrderByRand:
		return MySQLInsertDisallowOrderByRand, nil
	case TiDBSchemaRuleStatementInsertDisallowOrderByRand:
		return MySQLInsertDisallowOrderByRand, nil
	case PostgreSQLSchemaRuleStatementInsertDisallowOrderByRand:
		return PostgreSQLInsertDisallowOrderByRand, nil
	case MySQLSchemaRuleStatementDisallowLimit:
		return MySQLDisallowLimit, nil
	case TiDBSchemaRuleStatementDisallowLimit:
		return MySQLDisallowLimit, nil
	case MySQLSchemaRuleStatementDisallowOrderBy:
		return MySQLDisallowOrderBy, nil
	case TiDBSchemaRuleStatementDisallowOrderBy:
		return MySQLDisallowOrderBy, nil
	case MySQLSchemaRuleStatementMergeAlterTable:
		return MySQLMergeAlterTable, nil
	case TiDBSchemaRuleStatementMergeAlterTable:
		return MySQLMergeAlterTable, nil
	case PostgreSQLSchemaRuleStatementMergeAlterTable:
		return PostgreSQLMergeAlterTable, nil
	case MySQLSchemaRuleStatementAffectedRowLimit:
		return MySQLStatementAffectedRowLimit, nil
	case PostgreSQLSchemaRuleStatementAffectedRowLimit:
		return PostgreSQLStatementAffectedRowLimit, nil
	case MySQLSchemaRuleStatementDMLDryRun:
		return MySQLStatementDMLDryRun, nil
	case TiDBSchemaRuleStatementDMLDryRun:
		return MySQLStatementDMLDryRun, nil
	case PostgreSQLSchemaRuleStatementDMLDryRun:
		return PostgreSQLStatementDMLDryRun, nil
	case PostgreSQLSchemaRuleStatementDisallowAddColumnWithDefault:
		return PostgreSQLDisallowAddColumnWithDefault, nil
	case PostgreSQLSchemaRuleStatementAddCheckNotValid:
		return PostgreSQLAddCheckNotValid, nil
	case PostgreSQLSchemaRuleStatementDisallowAddNotNull:
		return PostgreSQLDisallowAddNotNull, nil
	case PostgreSQLSchemaRuleCommentLength:
		return PostgreSQLCommentConvention, nil
	}
	return Fake, errors.Errorf("unknown SQL review rule type %v", ruleType)
}
