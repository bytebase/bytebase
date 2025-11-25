package code

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
	StatementDisallowedInSDL                  Code = 239
	SDLDisallowColumnConstraint               Code = 240
	SDLRequireConstraintName                  Code = 241
	SDLRequireSchemaName                      Code = 242
	SDLRequireIndexName                       Code = 243
	SDLForeignKeyTableNotFound                Code = 244
	SDLForeignKeyColumnNotFound               Code = 245
	SDLForeignKeyTypeMismatch                 Code = 246
	SDLCheckConstraintInvalidColumn           Code = 247
	SDLCheckConstraintCrossTableReference     Code = 248
	SDLDuplicateTableName                     Code = 249
	SDLDuplicateIndexName                     Code = 250
	SDLDuplicateConstraintName                Code = 251
	SDLDuplicateColumnName                    Code = 252
	SDLMultiplePrimaryKey                     Code = 253
	SDLViewDependencyNotFound                 Code = 254
	SDLDropOperation                          Code = 255
	SDLReplaceOperation                       Code = 256

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
	AutoIncrementExists                        Code = 426
	OnUpdateColumnNotDatetimeOrTimestamp       Code = 427
	SetNullDefaultForNotNullColumn             Code = 428

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
	DatabaseNotEmpty       Code = 701
	NotCurrentDatabase     Code = 702
	DatabaseIsDeleted      Code = 703
	DatabaseNotExists      Code = 704
	ReferenceOtherDatabase Code = 705

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
	RelationExists  Code = 1902

	// 2001 ~ 2099 builtin error code.
	BuiltinPriorBackupCheck Code = 2001

	// 2101 ~ 2199 constraint error code.
	ConstraintNotExists Code = 2101

	// 2201 ~ 2299 view error code.
	ViewNotExists Code = 2201
	ViewExists    Code = 2202
)

// Int returns the int type of code.
func (c Code) Int() int {
	return int(c)
}

// Int32 returns the int32 type of code.
func (c Code) Int32() int32 {
	return int32(c)
}
