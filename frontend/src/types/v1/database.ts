import { create } from "@bufbuild/protobuf";
import { extractDatabaseResourceName, isNullOrUndefined } from "@/utils";
import { EMPTY_ID, UNKNOWN_ID } from "../const";
import { State } from "../proto-es/v1/common_pb";
import type { Database } from "../proto-es/v1/database_service_pb";
import { DatabaseSchema$ } from "../proto-es/v1/database_service_pb";
import type { InstanceResource } from "../proto-es/v1/instance_service_pb";
import type { Project } from "../proto-es/v1/project_service_pb";
import type { Environment } from "../v1/environment";
import { formatEnvironmentName, unknownEnvironment } from "./environment";
import { unknownInstance, unknownInstanceResource } from "./instance";
import { unknownProject } from "./project";

export interface ComposedDatabase extends Database {
  /** related project entity */
  projectEntity: Project;
  /** extracted database name */
  databaseName: string;
  /** instance name. Format: instances/{instance} */
  instance: string;
  /** related environment entity composed by effectedEnvironment  */
  effectiveEnvironmentEntity: Environment;
  /** non-empty instanceResource field, should be filled by unknownInstanceResource() if needed */
  instanceResource: InstanceResource;
}

export const UNKNOWN_DATABASE_NAME = `${unknownInstance().name}/databases/${UNKNOWN_ID}`;

export const unknownDatabase = (): ComposedDatabase => {
  const projectEntity = unknownProject();
  const instanceResource = unknownInstanceResource();
  const effectiveEnvironmentEntity = unknownEnvironment();
  const database = create(DatabaseSchema$, {
    name: `${instanceResource.name}/databases/${UNKNOWN_ID}`,
    state: State.ACTIVE,
    project: projectEntity.name,
    effectiveEnvironment: formatEnvironmentName(effectiveEnvironmentEntity.id),
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

export const isValidDatabaseName = (name: unknown): name is string => {
  if (typeof name !== "string") return false;
  const { instanceName, databaseName } = extractDatabaseResourceName(name);
  return (
    !isNullOrUndefined(instanceName) &&
    !isNullOrUndefined(databaseName) &&
    instanceName !== String(UNKNOWN_ID) &&
    instanceName !== String(EMPTY_ID) &&
    databaseName !== String(UNKNOWN_ID)
  );
};
