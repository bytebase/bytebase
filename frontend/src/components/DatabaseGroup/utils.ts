import { Factor } from "@/plugins/cel";

export type ResourceType = "DATABASE_GROUP" | "SCHEMA_GROUP";

export const StringFactorList = [
  "resource.database_name",
  "resource.schema_name",
  "resource.table_name",
  "resource.instance_id",
] as const;

export const FactorList: Map<ResourceType, Factor[]> = new Map([
  ["DATABASE_GROUP", ["resource.database_name", "resource.instance_id"]],
  ["SCHEMA_GROUP", ["resource.table_name"]],
]);
