import {
  CEL_ATTRIBUTE_LEVEL,
  CEL_ATTRIBUTE_REQUEST_EXPIRATION_DAYS,
  CEL_ATTRIBUTE_REQUEST_ROLE,
  CEL_ATTRIBUTE_REQUEST_TIME,
  CEL_ATTRIBUTE_RESOURCE_CLASSIFICATION_LEVEL,
  CEL_ATTRIBUTE_RESOURCE_COLUMN_NAME,
  CEL_ATTRIBUTE_RESOURCE_DATABASE,
  CEL_ATTRIBUTE_RESOURCE_DATABASE_NAME,
  CEL_ATTRIBUTE_RESOURCE_DB_ENGINE,
  CEL_ATTRIBUTE_RESOURCE_ENVIRONMENT_ID,
  CEL_ATTRIBUTE_RESOURCE_INSTANCE_ID,
  CEL_ATTRIBUTE_RESOURCE_PROJECT_ID,
  CEL_ATTRIBUTE_RESOURCE_SCHEMA_NAME,
  CEL_ATTRIBUTE_RESOURCE_TABLE_NAME,
  CEL_ATTRIBUTE_RISK_LEVEL,
  CEL_ATTRIBUTE_SOURCE,
  CEL_ATTRIBUTE_STATEMENT_AFFECTED_ROWS,
  CEL_ATTRIBUTE_STATEMENT_SQL_TYPE,
  CEL_ATTRIBUTE_STATEMENT_TABLE_ROWS,
  CEL_ATTRIBUTE_STATEMENT_TEXT,
} from "@/utils/cel-attributes";

export const NumberFactorList = [
  // Risk related factors
  CEL_ATTRIBUTE_STATEMENT_AFFECTED_ROWS,
  CEL_ATTRIBUTE_STATEMENT_TABLE_ROWS,

  // Request query/export factors
  CEL_ATTRIBUTE_REQUEST_EXPIRATION_DAYS,
] as const;
export type NumberFactor = (typeof NumberFactorList)[number];

export const StringFactorList = [
  // Risk related factors
  CEL_ATTRIBUTE_LEVEL,
  CEL_ATTRIBUTE_SOURCE,
  CEL_ATTRIBUTE_RISK_LEVEL,
  CEL_ATTRIBUTE_RESOURCE_ENVIRONMENT_ID,
  CEL_ATTRIBUTE_RESOURCE_PROJECT_ID,
  CEL_ATTRIBUTE_RESOURCE_DATABASE_NAME,
  CEL_ATTRIBUTE_RESOURCE_SCHEMA_NAME,
  CEL_ATTRIBUTE_RESOURCE_TABLE_NAME,
  CEL_ATTRIBUTE_RESOURCE_INSTANCE_ID,
  CEL_ATTRIBUTE_RESOURCE_DB_ENGINE,
  CEL_ATTRIBUTE_STATEMENT_SQL_TYPE,
  CEL_ATTRIBUTE_STATEMENT_TEXT,
  CEL_ATTRIBUTE_REQUEST_ROLE,

  // Grant request issue related factors
  CEL_ATTRIBUTE_RESOURCE_DATABASE,
  CEL_ATTRIBUTE_RESOURCE_SCHEMA_NAME,
  CEL_ATTRIBUTE_RESOURCE_TABLE_NAME,

  // Masking
  CEL_ATTRIBUTE_RESOURCE_COLUMN_NAME,
  // Masking rule
  CEL_ATTRIBUTE_RESOURCE_CLASSIFICATION_LEVEL,
] as const;
export type StringFactor = (typeof StringFactorList)[number];

export const TimestampFactorList = [CEL_ATTRIBUTE_REQUEST_TIME] as const;
export type TimestampFactor = (typeof TimestampFactorList)[number];

export const HighLevelFactorList = [
  CEL_ATTRIBUTE_LEVEL,
  CEL_ATTRIBUTE_SOURCE,
] as const;
export type HighLevelFactor = (typeof HighLevelFactorList)[number];

export type Factor =
  | NumberFactor
  | StringFactor
  | TimestampFactor
  | HighLevelFactor;

export const isNumberFactor = (factor: string): factor is NumberFactor => {
  return NumberFactorList.includes(factor as NumberFactor);
};

export const isStringFactor = (factor: string): factor is StringFactor => {
  return StringFactorList.includes(factor as StringFactor);
};

export const isTimestampFactor = (
  factor: string
): factor is TimestampFactor => {
  return TimestampFactorList.includes(factor as TimestampFactor);
};

export const isHighLevelFactor = (
  factor: string
): factor is HighLevelFactor => {
  return HighLevelFactorList.includes(factor as HighLevelFactor);
};
