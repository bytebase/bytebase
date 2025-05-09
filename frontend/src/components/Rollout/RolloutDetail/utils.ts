import { extractCoreDatabaseInfoFromDatabaseCreateTask } from "@/components/IssueV1";
import { mockDatabase } from "@/components/IssueV1/logic/utils";
import { useDatabaseV1Store } from "@/store";
import {
  isValidDatabaseName,
  unknownDatabase,
  type ComposedProject,
} from "@/types";
import { Task_Type, type Task } from "@/types/proto/v1/rollout_service";

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
        return mockDatabase(project, task.target);
      }
      return db;
    }
  }
  return unknownDatabase();
};
