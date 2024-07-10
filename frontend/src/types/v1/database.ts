import { EMPTY_ID, UNKNOWN_ID } from "../const";
import { State } from "../proto/v1/common";
import { Database } from "../proto/v1/database_service";
import type { Environment } from "../proto/v1/environment_service";
import { emptyEnvironment, unknownEnvironment } from "./environment";
import {
  emptyInstanceResource,
  unknownInstanceResource,
  type ComposedInstanceResource,
} from "./instance";
import type { ComposedProject } from "./project";
import { emptyProject, unknownProject } from "./project";

export interface ComposedDatabase extends Database {
  /** related project entity */
  projectEntity: ComposedProject;
  /** extracted database name */
  databaseName: string;
  /** instance name. Format: instances/{instance} */
  instance: string;
  /** InstanceResource with more fields */
  instanceResource: ComposedInstanceResource;
  /** related environment entity composed by effectedEnvironment  */
  effectiveEnvironmentEntity: Environment;
}

export const emptyDatabase = (): ComposedDatabase => {
  const projectEntity = emptyProject();
  const instanceResource = emptyInstanceResource();
  const effectiveEnvironmentEntity = emptyEnvironment();
  const database = Database.fromJSON({
    name: `${instanceResource.name}/databases/${EMPTY_ID}`,
    uid: String(EMPTY_ID),
    syncState: State.ACTIVE,
    project: projectEntity.name,
    effectiveEnvironment: effectiveEnvironmentEntity.name,
  });
  return {
    ...database,
    databaseName: "",
    instance: instanceResource.name,
    instanceResource,
    projectEntity,
    effectiveEnvironmentEntity,
  };
};

export const unknownDatabase = (): ComposedDatabase => {
  const projectEntity = unknownProject();
  const instanceResource = unknownInstanceResource();
  const effectiveEnvironmentEntity = unknownEnvironment();
  const database = Database.fromJSON({
    name: `${instanceResource.name}/databases/${UNKNOWN_ID}`,
    uid: String(UNKNOWN_ID),
    syncState: State.ACTIVE,
    project: projectEntity.name,
    effectiveEnvironment: effectiveEnvironmentEntity.name,
  });
  return {
    ...database,
    databaseName: "<<Unknown database>>",
    instance: instanceResource.name,
    instanceResource,
    projectEntity,
    effectiveEnvironmentEntity,
  };
};
