import type { ConditionGroupExpr } from "@/plugins/cel";
import type { ComposedDatabase } from "@/types";
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
  simpleExpr: ConditionGroupExpr;
}

export interface ComposedSchemaGroup extends SchemaGroup {
  databaseGroup: ComposedDatabaseGroup;
}

export interface ComposedSchemaGroupTable extends SchemaGroup_Table {
  databaseEntity: ComposedDatabase;
}
