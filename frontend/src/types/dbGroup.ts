import type { ConditionGroupExpr } from "@/plugins/cel";
import type { ComposedProject } from "@/types";
import type { DatabaseGroup } from "./proto/v1/database_group_service";

export interface ComposedDatabaseGroup extends DatabaseGroup {
  projectName: string;
  projectEntity: ComposedProject;
  simpleExpr: ConditionGroupExpr;
}
