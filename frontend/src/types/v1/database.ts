import { EMPTY_ID, UNKNOWN_ID } from "../const";
import { State } from "../proto/v1/common";
import { Database } from "../proto/v1/database_service";
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
}

export const emptyDatabase = (): ComposedDatabase => {
  const projectEntity = emptyProject();
  const instanceEntity = emptyInstance();
  const database = Database.fromJSON({
    name: `${instanceEntity.name}/databases/${EMPTY_ID}`,
    uid: String(EMPTY_ID),
    syncState: State.ACTIVE,
    project: projectEntity.name,
  });
  return {
    ...database,
    databaseName: "",
    instance: instanceEntity.name,
    instanceEntity,
    projectEntity,
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
  });
  return {
    ...database,
    databaseName: "<<Unknown database>>",
    instance: instanceEntity.name,
    instanceEntity,
    projectEntity,
  };
};
