package advisor

// Code is the error code.
type Code int

// Application error codes.
const (
	Ok       Code = 0
	Internal Code = 1

	// db error
	DbStatementSyntaxError Code = 102

	// task check error
	EmptySchemaReviewPolicy Code = 401

	// 10001 ~ 10100 compatibility error code
	CompatibilityDropDatabase  Code = 10001
	CompatibilityRenameTable   Code = 10002
	CompatibilityDropTable     Code = 10003
	CompatibilityRenameColumn  Code = 10004
	CompatibilityDropColumn    Code = 10005
	CompatibilityAddPrimaryKey Code = 10006
	CompatibilityAddUniqueKey  Code = 10007
	CompatibilityAddForeignKey Code = 10008
	CompatibilityAddCheck      Code = 10009
	CompatibilityAlterCheck    Code = 10010
	CompatibilityAlterColumn   Code = 10011

	// 10101 ~ 10200 statement error code
	StatementNoWhere             Code = 10101
	StatementSelectAll           Code = 10102
	StatementLeadingWildcardLike Code = 10103

	// 10201 table naming advisor error code
	NamingTableConventionMismatch Code = 10201
	// 10202 column naming advisor error code
	NamingColumnConventionMismatch Code = 10202
	// 10203 index naming advisor error code
	NamingIndexConventionMismatch Code = 10203
	// 10204 unique key naming advisor error code
	NamingUKConventionMismatch Code = 10204
	// 10205 foreign key naming advisor error code
	NamingFKConventionMismatch Code = 10205

	// 10301 column rule advisor error code
	NoRequiredColumn Code = 10301
	ColumnCanNotNull Code = 10302

	NotInnoDBEngine Code = 10401

	// 10501 table rule advisor error code
	TableNoPK Code = 10501
)
