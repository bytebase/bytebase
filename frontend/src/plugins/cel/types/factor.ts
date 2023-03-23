import { uniq, without } from "lodash-es";

export const NumberFactorList = [
  "insert_rows",
  "update_delete_rows",
  "risk",
  "source",
] as const;
export type NumberFactor = typeof NumberFactorList[number];

export const StringFactorList = [
  "environment",
  "project",
  "database_name",
  "db_engine",
  "sql_type",
] as const;
export type StringFactor = typeof StringFactorList[number];

export const HighLevelFactorList = ["risk", "source"] as const;
export type HighLevelFactor = typeof HighLevelFactorList[number];

export type Factor = NumberFactor | StringFactor | HighLevelFactor;

export const FactorList = {
  DDL: uniq([...HighLevelFactorList, ...StringFactorList]),
  DML: uniq([...HighLevelFactorList, ...NumberFactorList, ...StringFactorList]),
  CreateDatabase: without(
    [...HighLevelFactorList, ...StringFactorList],
    "sql_type"
  ),
};

export const isHighLevelFactor = (
  factor: string
): factor is HighLevelFactor => {
  return HighLevelFactorList.includes(factor as HighLevelFactor);
};
export const isStringFactor = (factor: string): factor is StringFactor => {
  return StringFactorList.includes(factor as StringFactor);
};
export const isNumberFactor = (factor: string): factor is NumberFactor => {
  return NumberFactorList.includes(factor as NumberFactor);
};
