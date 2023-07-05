export const NumberFactorList = [
  // Risk related factors
  "affected_rows",
  "level",
  "source",

  // Grant request issue related factors
  "request.row_limit",

  // Request query/export factors
  "expiration_days",
  "export_rows",
] as const;
export type NumberFactor = typeof NumberFactorList[number];

export const StringFactorList = [
  // Risk related factors
  "environment_id", // using `environment.resource_id`
  "project_id", // using `project.resource_id`
  "database_name",
  "db_engine",
  "sql_type",

  // Grant request issue related factors
  "resource.database",
  "resource.schema",
  "resource.table",
  "request.statement",
  "request.export_format",

  // Database/table group related factors
  "resource.environment_name", // using `environment.name`
  "resource.instance_id", // using `instance.resourceId`
  "resource.database_name",
  "resource.table_name",
] as const;
export type StringFactor = typeof StringFactorList[number];

export const TimestampFactorList = ["request.time"];
export type TimestampFactor = typeof TimestampFactorList[number];

export const HighLevelFactorList = ["level", "source"] as const;
export type HighLevelFactor = typeof HighLevelFactorList[number];

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
