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
	StatementSyntaxError             Code = 201
	StatementNoWhere                 Code = 202
	StatementSelectAll               Code = 203
	StatementLeadingWildcardLike     Code = 204
	StatementCreateTableAs           Code = 205
	StatementDisallowCommit          Code = 206
	StatementRedundantAlterTable     Code = 207
	StatementDMLDryRunFailed         Code = 208
	StatementAffectedRowExceedsLimit Code = 209
	StatementAddColumnWithDefault    Code = 210
	StatementAddCheckWithValidation  Code = 211
	StatementAddNotNull              Code = 212

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

	// 401 ~ 499 column error code.
	NoRequiredColumn                           Code = 401
	ColumnCannotNull                           Code = 402
	ChangeColumnType                           Code = 403
	NotNullColumnWithNoDefault                 Code = 404
	ColumnNotExists                            Code = 405
	UseChangeColumnStatement                   Code = 406
	ChangeColumnOrder                          Code = 407
	NoColumnComment                            Code = 408
	ColumnCommentTooLong                       Code = 409
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
	InvalidColumnDefault                    Code = 423

	// 501 engine error code.
	NotInnoDBEngine Code = 501

	// 601 ~ 699 table rule advisor error code.
	TableNoPK                         Code = 601
	TableHasFK                        Code = 602
	TableDropNamingConventionMismatch Code = 603
	TableNotExists                    Code = 604
	NoTableComment                    Code = 605
	TableCommentTooLong               Code = 606
	TableExists                       Code = 607
	CreateTablePartition              Code = 608
	TableIsReferencedByView           Code = 609

	// 701 ~ 799 database advisor error code.
	DatabaseNotEmpty   Code = 701
	NotCurrentDatabase Code = 702
	DatabaseIsDeleted  Code = 703

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
	CommentTooLong Code = 1301
)

// Int returns the int type of code.
func (c Code) Int() int {
	return int(c)
}
