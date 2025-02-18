import { extractCoreDatabaseInfoFromDatabaseCreateTask } from "@/components/IssueV1";
import {
  composeInstanceResourceForDatabase,
  useDatabaseV1Store,
  useEnvironmentV1Store,
} from "@/store";
import {
  isValidDatabaseName,
  unknownDatabase,
  unknownEnvironment,
  type ComposedProject,
} from "@/types";
import { State } from "@/types/proto/v1/common";
import { Task_Type, type Task } from "@/types/proto/v1/rollout_service";
import { extractDatabaseResourceName } from "@/utils";

export const databaseForTask = (project: ComposedProject, task: Task) => {
  if (task.type === Task_Type.DATABASE_CREATE) {
    // The database is not created yet.
    // extract database info from the task's and payload's properties.
    return extractCoreDatabaseInfoFromDatabaseCreateTask(project, task);
  } else {
    if (
      task.databaseDataUpdate ||
      task.databaseSchemaUpdate ||
      task.databaseDataExport ||
      task.type === Task_Type.DATABASE_SCHEMA_BASELINE
    ) {
      const db = useDatabaseV1Store().getDatabaseByName(task.target);
      if (!isValidDatabaseName(db.name)) {
        // Database not found, it's probably NOT_FOUND (maybe dropped actually)
        // Mock a database using all known resources
        db.project = project.name;
        db.projectEntity = project;

        db.name = task.target;
        const { instance, databaseName } = extractDatabaseResourceName(db.name);
        db.databaseName = databaseName;
        db.instance = instance;
        const ir = composeInstanceResourceForDatabase(instance, db);
        db.instanceResource = ir;
        db.environment = ir.environment;
        db.effectiveEnvironment = ir.environment;
        db.effectiveEnvironmentEntity =
          useEnvironmentV1Store().getEnvironmentByName(ir.environment) ??
          unknownEnvironment();
        db.syncState = State.DELETED;
      }
      return db;
    }
  }
  return unknownDatabase();
};
