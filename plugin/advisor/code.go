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
	NoRequiredColumn Code = 401
	ColumnCanNotNull Code = 402

	// 501 engine error code.
	NotInnoDBEngine Code = 501

	// 601 ~ 699 table rule advisor error code.
	TableNoPK                         Code = 601
	TableHasFK                        Code = 602
	TableDropNamingConventionMismatch Code = 603

	// 701 ~ 799 database advisor error code.
	DatabaseNotEmpty   Code = 701
	NotCurrentDatabase Code = 702

	// 801 miss index error code.
	NotUseIndex Code = 801
)

// Int returns the int type of code.
func (c Code) Int() int {
	return int(c)
}
