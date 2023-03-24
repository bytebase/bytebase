export const SQLTypeList = {
  DDL: [
    "CREATE",
    "CREATE_INDEX",
    "CREATE_VIEW",
    "CREATE_SEQUENCE",
    "CREATE_TABLE",
    "CREATE_SELECT",
    "CREATE_FUNCTION",
    "CREATE_PROCEDURE",
    "DROP",
    "DROP_INDEX",
    "DROP_VIEW",
    "DROP_TABLE",
    "DROP_FUNCTION",
    "DROP_PROCEDURE",
    "ALTER",
    "ALTER_INDEX",
    "ALTER_VIEW",
    "ALTER_TABLE",
    "ALTER_SEQUENCE",
    "ALTER_FUNCTION",
    "ALTER_PROCEDURE",
    "TRUNCATE",
    "RENAME",
  ] as const,
  DML: [
    "INSERT",
    "INSERT_SELECT",
    "REPLACE",
    "REPLACE_INTO",
    "UPDATE",
    "DELETE",
    "MERGE",
  ] as const,
} as const;

export type SQLTypeDDL = typeof SQLTypeList.DDL[number];
export type SQLTypeDML = typeof SQLTypeList.DML[number];
