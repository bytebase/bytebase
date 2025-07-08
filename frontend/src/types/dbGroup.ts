import type { ConditionGroupExpr } from "@/plugins/cel";
import { UNKNOWN_ID, type ComposedProject } from "@/types";
import { extractDatabaseGroupName } from "@/utils";
import type { DatabaseGroup } from "./proto-es/v1/database_group_service_pb";

export interface ComposedDatabaseGroup extends DatabaseGroup {
  projectName: string;
  projectEntity: ComposedProject;
  simpleExpr: ConditionGroupExpr;
}

export const isValidDatabaseGroupName = (name: string): boolean => {
  if (typeof name !== "string") return false;
  const dbGroupName = extractDatabaseGroupName(name);
  return Boolean(dbGroupName) && dbGroupName !== String(UNKNOWN_ID);
};
