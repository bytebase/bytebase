import type { ConditionGroupExpr } from "@/plugins/cel";
import type { ComposedDatabase } from "@/types";
import type { Environment } from "./proto/v1/environment_service";
import type {
  DatabaseGroup,
  Project,
  SchemaGroup,
  SchemaGroup_Table,
} from "./proto/v1/project_service";

export interface ComposedDatabaseGroup extends DatabaseGroup {
  databaseGroupName: string;
  projectName: string;
  project: Project;
  environmentName: string;
  environment: Environment;
  simpleExpr: ConditionGroupExpr;
}

export interface ComposedSchemaGroup extends SchemaGroup {
  databaseGroup: ComposedDatabaseGroup;
}

export interface ComposedSchemaGroupTable extends SchemaGroup_Table {
  databaseEntity: ComposedDatabase;
}
