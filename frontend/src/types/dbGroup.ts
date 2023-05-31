import { Environment } from "./proto/v1/environment_service";
import {
  DatabaseGroup,
  Project,
  SchemaGroup,
} from "./proto/v1/project_service";

export interface ComposedDatabaseGroup extends DatabaseGroup {
  databaseGroupName: string;
  projectName: string;
  project: Project;
  environmentName: string;
  environment: Environment;
}

export interface ComposedSchemaGroup extends SchemaGroup {
  databaseGroup: ComposedDatabaseGroup;
}
