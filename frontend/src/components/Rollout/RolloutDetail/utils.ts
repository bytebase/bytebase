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
  switch (task.type) {
    case Task_Type.DATABASE_CREATE:
      // The database is not created yet.
      // extract database info from the task's and payload's properties.
      return extractCoreDatabaseInfoFromDatabaseCreateTask(project, task);
    case Task_Type.DATABASE_SCHEMA_UPDATE:
    case Task_Type.DATABASE_SCHEMA_UPDATE_SDL:
    case Task_Type.DATABASE_SCHEMA_UPDATE_GHOST:
    case Task_Type.DATABASE_DATA_UPDATE:
    case Task_Type.DATABASE_EXPORT:
      const db = useDatabaseV1Store().getDatabaseByName(task.target);
      if (!isValidDatabaseName(db.name)) {
        return mockDatabase(project, task.target);
      }
      return db;
    default:
      return unknownDatabase();
  }
};
