import { uniq } from "lodash-es";

const mysqlSQLTypeList = {
  DDL: [
    "CREATE_DATABASE",
    "CREATE_INDEX",
    "CREATE_TABLE",
    "CREATE_VIEW",
    "CREATE_SEQUENCE",
    "CREATE_PLACEMENT_POLICY",
    "DROP_INDEX",
    "DROP_TABLE",
    "DROP_SEQUENCE",
    "DROP_PLACEMENT_POLICY",
    "DROP_DATABASE",
    "ALTER_TABLE",
    "ALTER_SEQUENCE",
    "ALTER_PLACEMENT_POLICY",
    "TRUNCATE",
    "RENAME_TABLE",
  ] as const,
  DML: ["INSERT", "UPDATE", "DELETE"] as const,
} as const;

const pgSQLTypeList = {
  DDL: [
    "CREATE_DATABASE",
    "CREATE_SCHEMA",
    "CREATE_INDEX",
    "CREATE_TRIGGER",
    "CREATE_TYPE",
    "CREATE_EXTENSION",
    "CREATE_VIEW",
    "CREATE_SEQUENCE",
    "CREATE_TABLE",
    "CREATE_FUNCTION",
    "DROP_COLUMN",
    "DROP_CONSTRAINT",
    "DROP_DATABASE",
    "DROP_DEFAULT",
    "DROP_EXTENSION",
    "DROP_FUNCTION",
    "DROP_INDEX",
    "DROP_NOT_NULL",
    "DROP_SCHEMA",
    "DROP_SEQUENCE",
    "DROP_TABLE",
    "DROP_TRIGGER",
    "DROP_TYPE",
    "ALTER_COLUMN_TYPE",
    "ALTER_SEQUENCE",
    "ALTER_VIEW",
    "ALTER_TABLE",
    "ALTER_TYPE",
    "ALTER_TABLE_ADD_COLUMN_LIST",
    "ALTER_TABLE_ADD_CONSTRAINT",
    "RENAME_COLUMN",
    "RENAME_CONSTRAINT",
    "RENAME_INDEX",
    "RENAME_SCHEMA",
    "RENAME_VIEW",
    "RENAME_TABLE",
    "TRUNCATE",
    "RENAME",
  ] as const,
  DML: ["INSERT", "UPDATE", "DELETE"] as const,
} as const;

export const SQLTypeList = {
  DDL: uniq([...mysqlSQLTypeList.DDL, ...pgSQLTypeList.DDL].sort()),
  DML: uniq([...mysqlSQLTypeList.DML, ...pgSQLTypeList.DML].sort()),
} as const;

export type SQLTypeDDL = typeof SQLTypeList.DDL[number];
export type SQLTypeDML = typeof SQLTypeList.DML[number];
