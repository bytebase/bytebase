import { EMPTY_ID, UNKNOWN_ID } from "../const";
import { State } from "../proto/v1/common";
import { Database } from "../proto/v1/database_service";
import { Environment } from "../proto/v1/environment_service";
import { ComposedInstance, emptyInstance, unknownInstance } from "./instance";
import { ComposedProject, emptyProject, unknownProject } from "./project";

export interface ComposedDatabase extends Database {
  /** related project entity */
  projectEntity: ComposedProject;
  /** extracted database name */
  databaseName: string;
  /** instance name. Format: instances/{instance} */
  instance: string;
  /** related instance entity */
  instanceEntity: ComposedInstance;
  effectiveEnvironmentEntity: Environment;
}

export const emptyDatabase = (): ComposedDatabase => {
  const projectEntity = emptyProject();
  const instanceEntity = emptyInstance();
  const database = Database.fromJSON({
    name: `${instanceEntity.name}/databases/${EMPTY_ID}`,
    uid: String(EMPTY_ID),
    syncState: State.ACTIVE,
    project: projectEntity.name,
    effectiveEnvironment: instanceEntity.name,
  });
  return {
    ...database,
    databaseName: "",
    instance: instanceEntity.name,
    instanceEntity,
    projectEntity,
    effectiveEnvironmentEntity: instanceEntity.environmentEntity,
  };
};

export const unknownDatabase = (): ComposedDatabase => {
  const projectEntity = unknownProject();
  const instanceEntity = unknownInstance();
  const database = Database.fromJSON({
    name: `${instanceEntity.name}/databases/${UNKNOWN_ID}`,
    uid: String(UNKNOWN_ID),
    syncState: State.ACTIVE,
    project: projectEntity.name,
    effectiveEnvironment: instanceEntity.environmentEntity.name,
  });
  return {
    ...database,
    databaseName: "<<Unknown database>>",
    instance: instanceEntity.name,
    instanceEntity,
    projectEntity,
    effectiveEnvironmentEntity: instanceEntity.environmentEntity,
  };
};
