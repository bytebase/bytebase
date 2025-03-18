package advisor

// Code is the error code.
type Code int

// Application error codes.
const (
	Ok          Code = 0
	Internal    Code = 1
	NotFound    Code = 2
	Unsupported Code = 3

	// 101 ~ 199 compatibility error code.
	CompatibilityDropDatabase  Code = 101
	CompatibilityRenameTable   Code = 102
	CompatibilityDropTable     Code = 103
	CompatibilityRenameColumn  Code = 104
	CompatibilityDropColumn    Code = 105
	CompatibilityAddPrimaryKey Code = 106
	CompatibilityAddUniqueKey  Code = 107
	CompatibilityAddForeignKey Code = 108
	CompatibilityAddCheck      Code = 109
	CompatibilityAlterCheck    Code = 110
	CompatibilityAlterColumn   Code = 111
	CompatibilityDropSchema    Code = 112

	// 201 ~ 299 statement error code.
	StatementSyntaxError                      Code = 201
	StatementNoWhere                          Code = 202
	StatementSelectAll                        Code = 203
	StatementLeadingWildcardLike              Code = 204
	StatementCreateTableAs                    Code = 205
	StatementDisallowCommit                   Code = 206
	StatementRedundantAlterTable              Code = 207
	StatementDMLDryRunFailed                  Code = 208
	StatementAffectedRowExceedsLimit          Code = 209
	StatementAddColumnWithDefault             Code = 210
	StatementAddCheckWithValidation           Code = 211
	StatementAddNotNull                       Code = 212
	StatementDisallowCascade                  Code = 213
	StatementCheckSelectFullTableScanFailed   Code = 214
	StatementHasTableFullScan                 Code = 215
	StatementCreateWithoutSchemaName          Code = 216
	StatementCheckSetRoleVariable             Code = 217
	StatementExplainQueryFailed               Code = 218
	StatementHasUsingFilesort                 Code = 219
	StatementHasUsingTemporary                Code = 220
	StatementWhereNoEqualNull                 Code = 221
	StatementExceedMaximumLimitValue          Code = 222
	StatementMaximumJoinTableCount            Code = 223
	StatementUnwantedQueryPlanLevel           Code = 224
	StatementWhereMaximumLogicalOperatorCount Code = 225
	StatementJoinColumnAttrsNotMatch          Code = 226
	StatementDisallowMixDDLDML                Code = 227
	// 228 is deprecated.
	StatementAddFKWithValidation              Code = 229
	StatementNonTransactional                 Code = 230
	StatementAddColumnWithPosition            Code = 231
	StatementOfflineDDL                       Code = 232
	StatementDisallowCrossDBQueries           Code = 233
	StatementDisallowFunctionsAndCalculations Code = 234
	StatementNoMaxExecutionTime               Code = 235
	StatementNoAlgorithmOption                Code = 236
	StatementNoLockOption                     Code = 237
	StatementObjectOwnerCheck                 Code = 238

	// 301 ～ 399 naming error code
	// 301 table naming advisor error code.
	NamingTableConventionMismatch Code = 301
	// 302 column naming advisor error code.
	NamingColumnConventionMismatch Code = 302
	// 303 index naming advisor error code.
	NamingIndexConventionMismatch Code = 303
	// 304 unique key naming advisor error code.
	NamingUKConventionMismatch Code = 304
	// 305 foreign key naming advisor error code.
	NamingFKConventionMismatch Code = 305
	// 306 primary key naming advisor error code.
	NamingPKConventionMismatch Code = 306
	// 307 auto_increment  column naming advisor error code.
	NamingAutoIncrementColumnConventionMismatch Code = 307
	// 308 name is keyword identifier advisor error code.
	NameIsKeywordIdentifier Code = 308
	// 309 naming case mismatch advisor error code.
	NamingCaseMismatch Code = 309
	// 310 not fully qualified object name error code.
	NamingNotFullyQualifiedName = 310

	// 401 ~ 499 column error code.
	NoRequiredColumn                           Code = 401
	ColumnCannotNull                           Code = 402
	ChangeColumnType                           Code = 403
	NotNullColumnWithNoDefault                 Code = 404
	ColumnNotExists                            Code = 405
	UseChangeColumnStatement                   Code = 406
	ChangeColumnOrder                          Code = 407
	AutoIncrementColumnNotInteger              Code = 410
	DisabledColumnType                         Code = 411
	ColumnExists                               Code = 412
	DropAllColumns                             Code = 413
	SetColumnCharset                           Code = 414
	CharLengthExceedsLimit                     Code = 415
	AutoIncrementColumnInitialValueNotMatch    Code = 416
	AutoIncrementColumnSigned                  Code = 417
	DefaultCurrentTimeColumnCountExceedsLimit  Code = 418
	OnUpdateCurrentTimeColumnCountExceedsLimit Code = 419
	NoDefault                                  Code = 420
	ColumnIsReferencedByView                   Code = 421
	VarcharLengthExceedsLimit                  Code = 422
	InvalidColumnDefault                       Code = 423
	DropIndexColumn                            Code = 424
	DropColumn                                 Code = 425

	// 501 engine error code.
	NotInnoDBEngine Code = 501

	// 601 ~ 699 table rule advisor error code.
	TableNoPK                         Code = 601
	TableHasFK                        Code = 602
	TableDropNamingConventionMismatch Code = 603
	TableNotExists                    Code = 604
	TableExists                       Code = 607
	CreateTablePartition              Code = 608
	TableIsReferencedByView           Code = 609
	CreateTableTrigger                Code = 610
	TotalTextLengthExceedsLimit       Code = 611
	DisallowSetCharset                Code = 612
	TableDisallowDDL                  Code = 613
	TableDisallowDML                  Code = 614
	TableExceedLimitSize              Code = 615
	NoCharset                         Code = 616
	NoCollation                       Code = 617

	// 701 ~ 799 database advisor error code.
	DatabaseNotEmpty   Code = 701
	NotCurrentDatabase Code = 702
	DatabaseIsDeleted  Code = 703
	DatabaseNotExists  Code = 704

	// 801 ~ 899 index error code.
	NotUseIndex                Code = 801
	IndexKeyNumberExceedsLimit Code = 802
	IndexPKType                Code = 803
	IndexTypeNoBlob            Code = 804
	IndexExists                Code = 805
	PrimaryKeyExists           Code = 806
	IndexEmptyKeys             Code = 807
	PrimaryKeyNotExists        Code = 808
	IndexNotExists             Code = 809
	IncorrectIndexName         Code = 810
	SpatialIndexKeyNullable    Code = 811
	DuplicateColumnInIndex     Code = 812
	IndexCountExceedsLimit     Code = 813
	CreateIndexUnconcurrently  Code = 814
	DuplicateIndexInTable      Code = 815
	IndexTypeNotAllowed        Code = 816
	RedundantIndex             Code = 817
	DropIndexUnconcurrently    Code = 818

	// 1001 ~ 1099 charset error code.
	DisabledCharset Code = 1001

	// 1101 ~ 1199 insert/update/delete error code.
	InsertTooManyRows      Code = 1101
	UpdateUseLimit         Code = 1102
	InsertUseLimit         Code = 1103
	UpdateUseOrderBy       Code = 1104
	DeleteUseOrderBy       Code = 1105
	DeleteUseLimit         Code = 1106
	InsertNotSpecifyColumn Code = 1107
	InsertUseOrderByRand   Code = 1108

	// 1201 ~ 1299 collation error code.
	DisabledCollation Code = 1201

	// 1301 ~ 1399 comment error code.
	CommentTooLong               Code = 1301
	CommentEmpty                 Code = 1032
	CommentMissingClassification Code = 1303

	// 1401 ~ 1499 procedure error code.
	DisallowCreateProcedure Code = 1401

	// 1501 ~ 1599 event error code.
	DisallowCreateEvent Code = 1501

	// 1601 ~ 1699 view error code.
	DisallowCreateView Code = 1601

	// 1701 ~ 1799 function error code.
	DisallowCreateFunction Code = 1701
	DisabledFunction       Code = 1702

	// 1801 ~ 1899 advice error code.
	// Advise enabling online migration for the issue.
	AdviseOnlineMigration Code = 1801
	// Advise using online migration to run the statement.
	AdviseOnlineMigrationForStatement Code = 1802
	// Advise not using online migration for the issue.
	AdviseNoOnlineMigration Code = 1803

	// 1901 ~ 1999 schema error code.
	SchemaNotExists Code = 1901

	// 2001 ~ 2099 builtin error code.
	BuiltinPriorBackupCheck Code = 2001
)

// Int returns the int type of code.
func (c Code) Int() int {
	return int(c)
}

// Int32 returns the int32 type of code.
func (c Code) Int32() int32 {
	return int32(c)
}

// Type is the type of advisor.
// nolint
type Type string

const (
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

	// MySQLWhereRequirementForSelect is an advisor type for MySQL WHERE clause requirement in SELECT statements.
	MySQLWhereRequirementForSelect Type = "bb.plugin.advisor.mysql.where.require.select"

	// MySQLWhereRequirementForUpdateDelete is an advisor type for MySQL WHERE clause requirement in UPDATE/DELETE statements.
	MySQLWhereRequirementForUpdateDelete Type = "bb.plugin.advisor.mysql.where.require.update-delete"

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

	// MySQLColumnDisallowDrop is an advisor type for MySQL disallow DROP COLUMN statement.
	MySQLColumnDisallowDrop Type = "bb.plugin.advisor.mysql.column.disallow-drop"

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

	// MySQLColumnRequireCharset is an advisor type for MySQL column require charset.
	MySQLColumnRequireCharset Type = "bb.plugin.advisor.mysql.column.require-charset"

	// MySQLColumnRequireCollation is an advisor type for MySQL column require collation.
	MySQLColumnRequireCollation Type = "bb.plugin.advisor.mysql.column.require-collation"

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

	// MySQLTableFieldsMaximumCount is an advisor type for limiting MySQL table size.
	MySQLTableLimitSize Type = "bb.plugin.advisor.mysql.table.limit-size"

	// MySQLTableRequireCharset is an advisor type for MySQL table require charset.
	MySQLTableRequireCharset Type = "bb.plugin.advisor.mysql.table.require-charset"

	// MySQLTableRequireCollation is an advisor type for MySQL table require collation.
	MySQLTableRequireCollation Type = "bb.plugin.advisor.mysql.table.require-collation"

	// MySQLTableDisallowDML is an advisor type for MySQL disallow DML.
	MySQLTableDisallowDML Type = "bb.plugin.advisor.mysql.table.disallow-dml"

	// MySQLTableDisallowDDL is an advisor type for MySQL disallow DDL.
	MySQLTableDisallowDDL Type = "bb.plugin.advisor.mysql.table.disallow-ddl"

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

	// MySQLStatementDisallowMixInDDL is the advisor for MySQL that checks no DML statements are mixed in the DDL statements.
	MySQLStatementDisallowMixInDDL Type = "bb.plugin.advisor.mysql.statement.disallow-mix-in-ddl"
	// MySQLStatementDisallowMixInDML is the advisor for MySQL that checks no DDL statements are mixed in the DML statements.
	MySQLStatementDisallowMixInDML Type = "bb.plugin.advisor.mysql.statement.disallow-mix-in-dml"

	// MySQLBuiltinPriorBackupCheck is an advisor type for MySQL prior backup check.
	MySQLBuiltinPriorBackupCheck Type = "bb.plugin.advisor.mysql.builtin.prior-backup-check"

	// MySQLStatementAddColumnWithoutPosition is an advisor type for MySQL checking no position in ADD COLUMN clause.
	MySQLStatementAddColumnWithoutPosition Type = "bb.plugin.advisor.mysql.statement.add-column-without-position"

	// MySQLStatementMaxExecutionTime is an advisor type for MySQL statement max execution time.
	MySQLStatementMaxExecutionTime Type = "bb.plugin.advisor.mysql.statement.max-execution-time"

	// MySQLStatementRequireAlgorithmOption is an advisor type for MySQL statement require algorithm option for online DDL.
	MySQLStatementRequireAlgorithmOption Type = "bb.plugin.advisor.mysql.statement.require-algorithm-option"

	// MySQLStatementRequireLockOption is an advisor type for MySQL statement require lock option for online DDL.
	MySQLStatementRequireLockOption Type = "bb.plugin.advisor.mysql.statement.require-lock-option"

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

	// MySQLDisallowOfflineDDL is an advisor type for MySQL disallow Offline DDL.
	MySQLDisallowOfflineDDL Type = "bb.plugin.advisor.mysql.disallow-offline-ddl"

	// PostgreSQL Advisor.

	// PostgreSQLSyntax is an advisor type for PostgreSQL syntax.
	PostgreSQLSyntax Type = "bb.plugin.advisor.postgresql.syntax"

	// PostgreSQLNamingFullyQualifiedObjectName is an advisor type for enforing full qualified object name.
	PostgreSQLNamingFullyQualifiedObjectName Type = "bb.plugin.advisor.postgresql.naming.fully-qualified"

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

	// PostgreSQLColumnCommentConvention is an advisor type for PostgreSQL column comment convention.
	PostgreSQLColumnCommentConvention Type = "bb.plugin.advisor.postgresql.column.comment"

	// PostgreSQLCommentConvention is an advisor type for PostgreSQL comment convention.
	PostgreSQLCommentConvention Type = "bb.plugin.advisor.postgresql.comment"

	// PostgreSQLTableRequirePK is an advisor type for PostgreSQL table require primary key.
	PostgreSQLTableRequirePK Type = "bb.plugin.advisor.postgresql.table.require-pk"

	// PostgreSQLNoLeadingWildcardLike is an advisor type for PostgreSQL no leading wildcard LIKE.
	PostgreSQLNoLeadingWildcardLike Type = "bb.plugin.advisor.postgresql.where.no-leading-wildcard-like"

	// PostgreSQLWhereRequirementForSelect is an advisor type for PostgreSQL WHERE clause requirement for SELECT statements.
	PostgreSQLWhereRequirementForSelect Type = "bb.plugin.advisor.postgresql.where.require.select"

	// PostgreSQLWhereRequirementForUpdateDelete is an advisor type for PostgreSQL WHERE clause requirement for UPDATE/DELETE statements.
	PostgreSQLWhereRequirementForUpdateDelete Type = "bb.plugin.advisor.postgresql.where.require.update-delete"

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

	// PostgreSQLIndexConcurrently is an advisor type for PostgreSQL to create or drop index concurrently.
	PostgreSQLIndexConcurrently Type = "bb.plugin.advisor.postgresql.index.create-concurrently"

	// PostgreSQLColumnTypeDisallowList is an advisor type for Postgresql column type disallow list.
	PostgreSQLColumnTypeDisallowList Type = "bb.plugin.advisor.postgresql.column.type-disallow-list"

	// PostgreSQLColumnDisallowChangingType is an advisor type for PostgreSQL disallow changing column type.
	PostgreSQLColumnDisallowChangingType Type = "bb.plugin.advisor.postgresql.column.disallow-changing-type"

	// PostgreSQLColumnMaximumCharacterLength is an advisor type for PostgreSQL maximum character length.
	PostgreSQLColumnMaximumCharacterLength Type = "bb.plugin.advisor.postgresql.column.maximum-character-length"

	// PostgreSQLRequireColumnDefault is an advisor type for PostgreSQL column default requirement.
	PostgreSQLRequireColumnDefault Type = "bb.plugin.advisor.postgresql.column.require-default"

	// PostgreSQLColumnDefaultDisallowVolatile is an advisor type for PostgreSQL column default disallow volatile.
	PostgreSQLColumnDefaultDisallowVolatile Type = "bb.plugin.advisor.postgresql.column.default-disallow-volatile"

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

	// PostgreSQLAddFKNotValid is a PostgreSQL advisor type requiring add foreign key not valid.
	PostgreSQLAddFKNotValid Type = "bb.plugin.advisor.postgresql.statement.add-foreign-key-not-valid"

	// PostgreSQLDisallowAddColumnWithDefault is an advisor type for PostgreSQL to disallow add column with default.
	PostgreSQLDisallowAddColumnWithDefault Type = "bb.plugin.advisor.postgresql.statement.disallow-add-column-with-default"

	// PostgreSQLDisallowAddNotNull is an advisor type for PostgreSQl to disallow add not null.
	PostgreSQLDisallowAddNotNull Type = "bb.plugin.advisor.postgresql.statement.disallow-add-not-null"

	// PostgreSQLNonTransactional is an advisor type for PostgreSQL to disallow non-transactional statements.
	PostgreSQLNonTransactional Type = "bb.plugin.advisor.postgresql.statement.non-transactional"

	// PostgreSQLTableDropNamingConvention is an advisor type for PostgreSQL table drop with naming convention.
	PostgreSQLTableDropNamingConvention Type = "bb.plugin.advisor.postgresql.table.drop-naming-convention"

	// PostgreSQLCollationAllowlist is an advisor type for PostgreSQL collation allowlist.
	PostgreSQLCollationAllowlist Type = "bb.plugin.advisor.postgresql.collation.allowlist"

	// PostgreSQLStatementDisallowRemoveTblCascade is an advisor type for PostgreSQL to disallow CASCADE when removing a table.
	PostgreSQLStatementDisallowRemoveTblCascade Type = "bb.plugin.advisor.postgresql.statement.disallow-rm-tbl-cascade"

	// PostgreSQLStatementDisallowOnDelCascade is an advisor type for PostgreSQL to disallow ON DELETE CASCADE clauses.
	PostgreSQLStatementDisallowOnDelCascade Type = "bb.plugin.advisor.postgresql.statement.disallow-on-del-cascade"

	// PostgreSQLStatementCreateSpecifySchema is an advisor type for PostgreSQL to specify schema when creating.
	PostgreSQLStatementCreateSpecifySchema Type = "bb.plugin.advisor.postgresql.statement.create-specify-schema"

	// PostgreSQLStatementCheckSetRoleVariable is an advisor type for PostgreSQL to check set role variable.
	PostgreSQLStatementCheckSetRoleVariable Type = "bb.plugin.advisor.postgresql.statement.check-set-role-variable"

	PostgreSQLStatementDisallowMixInDDL Type = "bb.plugin.advisor.postgresql.statement.disallow-mix-in-ddl"
	PostgreSQLStatementDisallowMixInDML Type = "bb.plugin.advisor.postgresql.statement.disallow-mix-in-dml"

	// PostgreSQLBuiltinPriorBackupCheck is an advisor type for PostgreSQL do prior backup check.
	PostgreSQLBuiltinPriorBackupCheck Type = "bb.plugin.advisor.postgresql.builtin.prior-backup-check"

	// PostgreSQLStatementObjectOwnerCheck is an advisor type for PostgreSQL do object owner check.
	PostgreSQLStatementObjectOwnerCheck Type = "bb.plugin.advisor.postgresql.statement.object-owner-check"

	// PostgreSQLStatementMaximumLimitValue is an advisor type for PostgreSQL statement maximum limit value.
	PostgreSQLStatementMaximumLimitValue Type = "bb.plugin.advisor.postgresql.statement.maximum-limit-value"

	// PostgreSQLTableCommentConvention is an advisor type for PostgreSQL table comment convention.
	PostgreSQLTableCommentConvention Type = "bb.plugin.advisor.postgresql.table.comment"

	// Oracle Advisor.

	// OracleSyntax is an advisor type for Oracle syntax.
	OracleSyntax Type = "bb.plugin.advisor.oracle.syntax"

	// OracleBuiltinPriorBackupCheck is an advisor type for Oracle prior backup check.
	OracleBuiltinPriorBackupCheck Type = "bb.plugin.advisor.oracle.builtin.prior-backup-check"

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

	// OracleWhereRequirementForSelect is an advisor type for Oracle WHERE clause requirement in SELECT statements.
	OracleWhereRequirementForSelect Type = "bb.plugin.advisor.oracle.where.require.select"

	// OracleWhereRequirementForUpdateDelete is an advisor type for Oracle WHERE clause requirement in UPDATE/DELETE statements.
	OracleWhereRequirementForUpdateDelete Type = "bb.plugin.advisor.oracle.where.require.update-delete"

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

	// OracleStatementDMLDryRun is an advisor type for Oracle DML dry run.
	OracleStatementDMLDryRun Type = "bb.plugin.advisor.oracle.statement.dml-dry-run"

	OracleStatementDisallowMixInDDL Type = "bb.plugin.advisor.oracle.statement.disallow-mix-in-ddl"
	OracleStatementDisallowMixInDML Type = "bb.plugin.advisor.oracle.statement.disallow-mix-in-dml"

	// OracleTableCommentConvention is an advisor type for Oracle table comment convention.
	OracleTableCommentConvention Type = "bb.plugin.advisor.oracle.table.comment"
	// OracleColumnCommentConvention is an advisor type for Oracle column comment convention.
	OracleColumnCommentConvention Type = "bb.plugin.advisor.oracle.column.comment"

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

	// SnowflakeWhereRequirementForSelect is an advisor type for Snowflake WHERE clause requirement for SELECT statements.
	SnowflakeWhereRequirementForSelect Type = "bb.plugin.advisor.snowflake.where.require.select"

	// SnowflakeWhereRequirementForUpdateDelete is an advisor type for Snowflake WHERE clause requirement for UPDATE/DELETE statements.
	SnowflakeWhereRequirementForUpdateDelete Type = "bb.plugin.advisor.snowflake.where.require.update-delete"

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

	// MSSQLBuiltinPriorBackupCheck is an advisor type for MSSQL prior backup check.
	MSSQLBuiltinPriorBackupCheck Type = "bb.plugin.advisor.mssql.builtin.prior-backup-check"

	// MSSQLNoSelectAll is an advisor type for MSSQL no select all.
	MSSQLNoSelectAll Type = "bb.plugin.advisor.mssql.select.no-select-all"

	// MSSQLNamingTableConvention is an advisor type for MSSQL table naming convention.
	MSSQLNamingTableConvention Type = "bb.plugin.advisor.mssql.naming.table"

	// MSSQLTableNamingNoKeyword is an advisor type for MSSQL table naming convention without keyword.
	MSSQLTableNamingNoKeyword Type = "bb.plugin.advisor.mssql.naming.table-no-keyword"

	// MSSQLIdentifierNamingNoKeyword is an advisor type for MSSQL identifier naming convention without keyword.
	MSSQLIdentifierNamingNoKeyword Type = "bb.plugin.advisor.mssql.naming.identifier-no-keyword"

	// MSSQLWhereRequirementForSelect is an advisor type for MySQL WHERE clause requirement in SELECT statements.
	MSSQLWhereRequirementForSelect Type = "bb.plugin.advisor.mssql.where.require.select"

	// MSSQLWhereRequirementForUpdateDelete is an advisor type for MySQL WHERE clause requirement in UPDATE/DELETE statements.
	MSSQLWhereRequirementForUpdateDelete Type = "bb.plugin.advisor.mssql.where.require.update-delete"

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

	// MSSQLColumnTypeDisallowList is an advisor type for MSSQL column type disallow list.
	MSSQLColumnTypeDisallowList Type = "bb.plugin.advisor.mssql.column.type-disallow-list"

	// MSSQLTableDisallowDDL is an advisor type for MSSQL disallow DDL for specific tables.
	MSSQLTableDisallowDDL = "bb.plugin.advisor.mssql.table.disallow-ddl"

	// MSSQLTableDisallowDML is an advisor type for MSSQL disallow DML for specific tables.
	MSSQLTableDisallowDML = "bb.plugin.advisor.mssql.table.disallow-dml"

	// MSSQLFuctionDisallowCreate restricts the creation of functions.
	MSSQLFunctionDisallowCreateOrAlter Type = "bb.plugin.advisor.mssql.function.disallow-create-or-alter"

	// MSSQLProcedureDisallowCreateOrAlter restricts the creation of procedures.
	MSSQLProcedureDisallowCreateOrAlter Type = "bb.plugin.advisor.mssql.procedure.disallow-create-or-alter"

	// MSSQLStatementDisallowCrossDBQueries prohibits cross database queries.
	MSSQLStatementDisallowCrossDBQueries Type = "bb.plugin.advisor.mssql.statement.disallow-cross-db-queries"

	// MSSQLStatementWhereDisallowFunctionsAndCalculations prohibit using functions or performing calculation in the where clause.
	MSSQLStatementWhereDisallowFunctionsAndCalculations Type = "bb.plugin.advisor.mssql.statement.disallow-functions-and-calculations"

	MSSQLIndexNotRedundant Type = "bb.plugin.advisor.mssql.index.not-redundant"

	MSSQLStatementDisallowMixInDDL Type = "bb.plugin.advisor.mssql.statement.disallow-mix-in-ddl"
	MSSQLStatementDisallowMixInDML Type = "bb.plugin.advisor.mssql.statement.disallow-mix-in-dml"
)
