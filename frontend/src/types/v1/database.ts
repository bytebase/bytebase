import { Database } from "../proto/v1/database_service";
import { Environment } from "../proto/v1/environment_service";
import { Instance } from "../proto/v1/instance_service";
import { ComposedProject } from "./project";

export interface ComposedInstance extends Instance {
  environmentEntity: Environment;
}

export interface ComposedDatabase extends Database {
  /** related project entity */
  projectEntity: ComposedProject;
  /** extracted database name */
  databaseName: string;
  /** instance name. Format: instances/{instance} */
  instance: string;
  /** related instance entity */
  instanceEntity: ComposedInstance;
}
