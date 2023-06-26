import { ConditionGroupExpr } from "@/plugins/cel";
import { Environment } from "./proto/v1/environment_service";
import {
  DatabaseGroup,
  Project,
  SchemaGroup,
  SchemaGroup_Table,
} from "./proto/v1/project_service";
import { ComposedDatabase } from "@/types";

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
