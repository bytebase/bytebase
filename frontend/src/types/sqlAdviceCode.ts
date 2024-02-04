/**
 * Backend reference: /plugin/advisor/code.go
 */

export enum SQLAdviceCodeGeneral {
  Ok = 0,
  Internal = 1,
  NotFound = 2,
  Unsupported = 3,
}

// 101 ~ 199 compatibility error code.
export enum SQLAdviceCodeCompatibility {
  DropDatabase = 101,
  RenameTable = 102,
  DropTable = 103,
  RenameColumn = 104,
  DropColumn = 105,
  AddPrimaryKey = 106,
  AddUniqueKey = 107,
  AddForeignKey = 108,
  AddCheck = 109,
  AlterCheck = 110,
  AlterColumn = 111,
}

// 201 ~ 299 statement error code.
export enum SQLAdviceCodeStatement {
  SyntaxError = 201,
  NoWhere = 202,
  SelectAll = 203,
  LeadingWildcardLike = 204,
  CreateTableAs = 205,
  DisallowCommit = 206,
  RedundantAlterTable = 207,
  DMLDryRunFailed = 208,
  AffectedRowExceedsLimit = 209,
  StatementAddColumnWithDefault = 210,
  StatementAddCheckWithValidation = 211,
  StatementAddNotNull = 212,
}

// 301 ～ 399 naming error code
export enum SQLAdviceCodeNaming {
  // 301 table naming advisor error code.
  TableConventionMismatch = 301,
  // 302 column naming advisor error code.
  ColumnConventionMismatch = 302,
  // 303 index naming advisor error code.
  IndexConventionMismatch = 303,
  // 304 unique key naming advisor error code.
  UKConventionMismatch = 304,
  // 305 foreign key naming advisor error code.
  FKConventionMismatch = 305,
  // 306 primary key naming advisor error code.
  PKConventionMismatch = 306,
  // 307 auto_increment  column naming advisor error code.
  AutoIncrementColumnConventionMismatch = 307,
}

// 401 ~ 499 column error code.
export enum SQLAdviceCodeColumn {
  NoRequiredColumn = 401,
  ColumnCannotNull = 402,
  ChangeColumnType = 403,
  NotNullColumnWithNoDefault = 404,
  ColumnNotExists = 405,
  UseChangeColumnStatement = 406,
  ChangeColumnOrder = 407,
  NoColumnComment = 408,
  ColumnCommentTooLong = 409,
  AutoIncrementColumnNotInteger = 410,
  DisabledColumnType = 411,
  ColumnExists = 412,
  DropAllColumns = 413,
  SetColumnCharset = 414,
  CharLengthExceedsLimit = 415,
  AutoIncrementColumnInitialValueNotMatch = 416,
  AutoIncrementColumnSigned = 417,
  DefaultCurrentTimeColumnCountExceedsLimit = 418,
  OnUpdateCurrentTimeColumnCountExceedsLimit = 419,
  NoDefault = 420,
}

// 501 engine error code.
export enum SQLAdviceCodeEngine {
  NotInnoDBEngine = 501,
}

// 601 ~ 699 table rule advisor error code.
export enum SQLAdviceCodeTable {
  TableNoPK = 601,
  TableHasFK = 602,
  TableDropNamingConventionMismatch = 603,
  TableNotExists = 604,
  NoTableComment = 605,
  TableCommentTooLong = 606,
  TableExists = 607,
  CreateTablePartition = 608,
  CreateTableTrigger = 610,
}

// 701 ~ 799 database advisor error code.
export enum SQLAdviceCodeDatabase {
  DatabaseNotEmpty = 701,
  NotCurrentDatabase = 702,
  DatabaseIsDeleted = 703,
}

// 801 ~ 899 index error code.
export enum SQLAdviceCodeIndex {
  NotUseIndex = 801,
  IndexKeyNumberExceedsLimit = 802,
  IndexPKType = 803,
  IndexTypeNoBlob = 804,
  IndexExists = 805,
  PrimaryKeyExists = 806,
  IndexEmptyKeys = 807,
  PrimaryKeyNotExists = 808,
  IndexNotExists = 809,
  IncorrectIndexName = 810,
  SpatialIndexKeyNullable = 811,
  DuplicateColumnInIndex = 812,
  IndexCountExceedsLimit = 813,
  CreateIndexUnconcurrently = 814,
}

// 1001 ~ 1099 charset error code.
export enum SQLAdviceCodeCharset {
  DisabledCharset = 1001,
}

// 1101 ~ 1199 insert/update/delete error code.
export enum SQLAdviceCodeDML {
  InsertTooManyRows = 1101,
  UpdateUseLimit = 1102,
  InsertUseLimit = 1103,
  UpdateUseOrderBy = 1104,
  DeleteUseOrderBy = 1105,
  DeleteUseLimit = 1106,
  InsertNotSpecifyColumn = 1107,
  InsertUseOrderByRand = 1108,
}

// 1201 ~ 1299 collation error code.
export enum SQLAdviceCodeCollation {
  DisabledCollation = 1201,
}

// 1301 ~ 1399 comment error code.
export enum SQLAdviceCodeComment {
  CommentTooLong = 1301,
}

export type SQLAdviceCode =
  | SQLAdviceCodeGeneral
  | SQLAdviceCodeCompatibility
  | SQLAdviceCodeStatement
  | SQLAdviceCodeNaming
  | SQLAdviceCodeColumn
  | SQLAdviceCodeEngine
  | SQLAdviceCodeTable
  | SQLAdviceCodeDatabase
  | SQLAdviceCodeIndex
  | SQLAdviceCodeCharset
  | SQLAdviceCodeDML
  | SQLAdviceCodeCollation
  | SQLAdviceCodeComment;
