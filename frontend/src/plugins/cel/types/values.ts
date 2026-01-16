import { StatementType } from "@/types/proto-es/v1/common_pb";

// DDL statement types
const DDL_STATEMENT_TYPES = [
  StatementType.CREATE_DATABASE,
  StatementType.CREATE_TABLE,
  StatementType.CREATE_VIEW,
  StatementType.CREATE_INDEX,
  StatementType.CREATE_SEQUENCE,
  StatementType.CREATE_SCHEMA,
  StatementType.CREATE_FUNCTION,
  StatementType.CREATE_TRIGGER,
  StatementType.CREATE_PROCEDURE,
  StatementType.CREATE_EVENT,
  StatementType.CREATE_EXTENSION,
  StatementType.CREATE_TYPE,
  StatementType.DROP_DATABASE,
  StatementType.DROP_TABLE,
  StatementType.DROP_VIEW,
  StatementType.DROP_INDEX,
  StatementType.DROP_SEQUENCE,
  StatementType.DROP_SCHEMA,
  StatementType.DROP_FUNCTION,
  StatementType.DROP_TRIGGER,
  StatementType.DROP_PROCEDURE,
  StatementType.DROP_EVENT,
  StatementType.DROP_EXTENSION,
  StatementType.DROP_TYPE,
  StatementType.ALTER_DATABASE,
  StatementType.ALTER_TABLE,
  StatementType.ALTER_VIEW,
  StatementType.ALTER_SEQUENCE,
  StatementType.ALTER_EVENT,
  StatementType.ALTER_TYPE,
  StatementType.ALTER_INDEX,
  StatementType.TRUNCATE,
  StatementType.RENAME,
  StatementType.RENAME_INDEX,
  StatementType.RENAME_SCHEMA,
  StatementType.RENAME_SEQUENCE,
  StatementType.COMMENT,
] as const;

// DML statement types
const DML_STATEMENT_TYPES = [
  StatementType.INSERT,
  StatementType.UPDATE,
  StatementType.DELETE,
] as const;

// Helper to convert enum values to their string names for display/comparison
const getStatementTypeName = (type: StatementType): string => {
  return StatementType[type];
};

// Export lists with string names for backward compatibility with existing code
export const SQLTypeList = {
  DDL: DDL_STATEMENT_TYPES.map(getStatementTypeName),
  DML: DML_STATEMENT_TYPES.map(getStatementTypeName),
} as const;
