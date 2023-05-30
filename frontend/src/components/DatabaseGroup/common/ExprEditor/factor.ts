export const StringFactorList = [
  "resource.database_name",
  "resource.schema_name",
  "resource.table_name",
] as const;

export const FactorList = {
  DATABASE_GROUP: ["resource.database_name"],
  SCHEMA_GROUP: ["resource.table_name"],
};
