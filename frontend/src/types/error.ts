export enum GeneralErrorCode {
  OK = 0,
  INTERNAL = 1,
  NOT_AUTHORIZED = 2,
  INVALID = 3,
  NOT_FOUND = 4,
  CONFLICT = 5,
  NOT_IMPLEMENTED = 6,
}

export enum DBErrorCode {
  CONNECTION_ERROR = 101,
  SYNTAX_ERROR = 102,
  EXECUTION_ERROR = 103,
}

export enum MigrationErrorCode {
  MIGRATION_SCHEMA_MISSING = 201,
  MIGRAITON_ALREADY_APPLIED = 202,
  MGIRATION_OUT_OF_ORDER = 203,
  MIGRATION_BASELINE_MISSING = 204,
}

export type ErrorCode = GeneralErrorCode | DBErrorCode | MigrationErrorCode;
