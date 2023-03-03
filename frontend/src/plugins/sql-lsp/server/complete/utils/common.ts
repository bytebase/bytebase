import { SQLDialect } from "@/plugins/sql-lsp/types";

export enum SortText {
  DATABASE = "0",
  TABLE = "1",
  SUBQUERY = "1", // Same as TABLE
  COLUMN = "2",
  ALIAS = "2", // Same as COLUMN
  KEYWORD = "3",
}

export const isDialectWithSchema = (dialect: SQLDialect) => {
  const DIALECTS_WITHOUT_SCHEMA: SQLDialect[] = ["MYSQL", "TIDB"];
  if (DIALECTS_WITHOUT_SCHEMA.includes(dialect)) {
    return false;
  }
  return true;
};
