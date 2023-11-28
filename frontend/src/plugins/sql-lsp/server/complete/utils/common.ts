import type { SQLDialect } from "@/types";

export enum SortText {
  DATABASE = "0",
  TABLE = "1",
  // eslint-disable-next-line @typescript-eslint/no-duplicate-enum-values
  SUBQUERY = "1", // Same as TABLE
  COLUMN = "2",
  // eslint-disable-next-line @typescript-eslint/no-duplicate-enum-values
  ALIAS = "2", // Same as COLUMN
  KEYWORD = "3",
}

export const isDialectWithSchema = (dialect: SQLDialect) => {
  const DIALECTS_WITHOUT_SCHEMA: SQLDialect[] = ["MYSQL", "TIDB", "OCEANBASE"];
  if (DIALECTS_WITHOUT_SCHEMA.includes(dialect)) {
    return false;
  }
  return true;
};
