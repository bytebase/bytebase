import { Factor } from "@/plugins/cel";

export type ResourceType = "DATABASE_GROUP" | "SCHEMA_GROUP";

export const FactorList: Map<ResourceType, Factor[]> = new Map([
  ["DATABASE_GROUP", ["resource.database_name", "resource.instance_id"]],
  ["SCHEMA_GROUP", ["resource.table_name"]],
]);
