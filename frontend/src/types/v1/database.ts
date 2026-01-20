import { create } from "@bufbuild/protobuf";
import { extractDatabaseResourceName, isNullOrUndefined } from "@/utils";
import { UNKNOWN_ID } from "../const";
import { State } from "../proto-es/v1/common_pb";
import type { Database } from "../proto-es/v1/database_service_pb";
import { DatabaseSchema$ } from "../proto-es/v1/database_service_pb";
import { formatEnvironmentName, unknownEnvironment } from "./environment";
import { unknownInstance, unknownInstanceResource } from "./instance";
import { unknownProject } from "./project";

export const UNKNOWN_DATABASE_NAME = `${unknownInstance().name}/databases/${UNKNOWN_ID}`;

export const unknownDatabase = (): Database => {
  const instanceResource = unknownInstanceResource();
  return create(DatabaseSchema$, {
    name: `${instanceResource.name}/databases/${UNKNOWN_ID}`,
    state: State.ACTIVE,
    project: unknownProject().name,
    effectiveEnvironment: formatEnvironmentName(unknownEnvironment().id),
    instanceResource,
  });
};

export const isValidDatabaseName = (name: unknown): name is string => {
  if (typeof name !== "string") return false;
  const { instanceName, databaseName } = extractDatabaseResourceName(name);
  return (
    !isNullOrUndefined(instanceName) &&
    !isNullOrUndefined(databaseName) &&
    instanceName !== String(UNKNOWN_ID) &&
    databaseName !== String(UNKNOWN_ID)
  );
};
