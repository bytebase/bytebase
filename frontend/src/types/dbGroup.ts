import { ConditionGroupExpr } from "@/plugins/cel";
import { Environment } from "./proto/v1/environment_service";
import { DatabaseGroup, SchemaGroup } from "./proto/v1/project_service";
import { ComposedProject } from "./v1";

export interface ComposedDatabaseGroup extends DatabaseGroup {
  databaseGroupName: string;
  projectName: string;
  project: ComposedProject;
  environmentName: string;
  environment: Environment;
  simpleExpr: ConditionGroupExpr;
}

export interface ComposedSchemaGroup extends SchemaGroup {
  databaseGroup: ComposedDatabaseGroup;
}
