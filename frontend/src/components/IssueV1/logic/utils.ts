import { useDatabaseV1Store, useInstanceV1Store } from "@/store";
import {
  ComposedIssue,
  ComposedProject,
  unknownDatabase,
  UNKNOWN_ID,
} from "@/types";
import { Task, Task_Type } from "@/types/proto/v1/rollout_service";
import { extractDatabaseResourceName } from "@/utils";

export const extractCoreDatabaseInfoFromDatabaseCreateTask = (
  project: ComposedProject,
  task: Task
) => {
  const coreDatabaseInfo = (instance: string, databaseName: string) => {
    const name = `${instance}/databases/${databaseName}`;
    const maybeExistedDatabase = useDatabaseV1Store().getDatabaseByName(name);
    if (maybeExistedDatabase.uid !== String(UNKNOWN_ID)) {
      return maybeExistedDatabase;
    }

    const instanceEntity = useInstanceV1Store().getInstanceByName(instance);
    return {
      ...unknownDatabase(),
      name,
      uid: String(UNKNOWN_ID),
      databaseName,
      instance,
      instanceEntity,
      project: project.name,
      projectEntity: project,
    };
  };

  if (task.databaseCreate) {
    const databaseName = task.databaseCreate.database;
    const instance = task.target;
    return coreDatabaseInfo(instance, databaseName);
  }
  if (task.databaseRestoreRestore) {
    const db = extractDatabaseResourceName(
      task.databaseRestoreRestore.target || task.target
    );
    const databaseName = db.database;
    const instance = `instances/${db.instance}`;
    return coreDatabaseInfo(instance, databaseName);
  }
  if (task.type === Task_Type.DATABASE_RESTORE_CUTOVER) {
    const db = extractDatabaseResourceName(task.target);
    const databaseName = db.database;
    const instance = `instances/${db.instance}`;
    return coreDatabaseInfo(instance, databaseName);
  }

  return unknownDatabase();
};

export const databaseForTask = (issue: ComposedIssue, task: Task) => {
  if (
    task.type === Task_Type.DATABASE_CREATE ||
    task.type === Task_Type.DATABASE_RESTORE_RESTORE ||
    task.type === Task_Type.DATABASE_RESTORE_CUTOVER
  ) {
    // The database is not created yet.
    // extract database info from the task's and payload's properties.
    return extractCoreDatabaseInfoFromDatabaseCreateTask(
      issue.projectEntity,
      task
    );
  } else {
    if (
      task.databaseDataUpdate ||
      task.databaseSchemaUpdate ||
      task.databaseRestoreRestore ||
      task.type === Task_Type.DATABASE_SCHEMA_UPDATE_GHOST_CUTOVER ||
      task.type === Task_Type.DATABASE_SCHEMA_BASELINE
    ) {
      return useDatabaseV1Store().getDatabaseByName(task.target);
    }
  }
  return unknownDatabase();
};
