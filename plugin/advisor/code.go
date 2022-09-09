package advisor

// Code is the error code.
type Code int

// Application error codes.
const (
	Ok       Code = 0
	Internal Code = 1
	NotFound Code = 2

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

	// 201 ~ 299 statement error code.
	StatementSyntaxError         Code = 201
	StatementNoWhere             Code = 202
	StatementSelectAll           Code = 203
	StatementLeadingWildcardLike Code = 204
	StatementCreateTableAs       Code = 205

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

	// 401 ~ 499 column error code.
	NoRequiredColumn              Code = 401
	ColumnCanNotNull              Code = 402
	ChangeColumnType              Code = 403
	NotNullColumnWithNullDefault  Code = 404
	ColumnNotExists               Code = 405
	UseChangeColumnStatement      Code = 406
	ChangeColumnOrder             Code = 407
	NoColumnComment               Code = 408
	ColumnCommentTooLong          Code = 409
	AutoIncrementColumnNotInteger Code = 410
	DisabledColumnType            Code = 411

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

	// 701 ~ 799 database advisor error code.
	DatabaseNotEmpty   Code = 701
	NotCurrentDatabase Code = 702

	// 801 miss index error code.
	NotUseIndex                Code = 801
	IndexKeyNumberExceedsLimit Code = 802
	IndexPKType                Code = 803
	IndexTypeNoBlob            Code = 804
	IndexExists                Code = 805

	// 901 ~ 999 index error code.
	DuplicateColumnInIndex Code = 901

	// 1001 ~ 1099 charset error code.
	DisabledCharset Code = 1001
)

// Int returns the int type of code.
func (c Code) Int() int {
	return int(c)
}
