import {
  Task,
  TaskDatabaseCreatePayload,
  TaskDatabaseDataUpdatePayload,
  TaskDatabaseSchemaUpdateGhostSyncPayload,
  TaskDatabaseSchemaUpdatePayload,
  TaskDatabaseSchemaUpdateSDLPayload,
} from "@/types";

export const sheetIdOfTask = (task: Task) => {
  switch (task.type) {
    case "bb.task.database.create":
      return (
        ((task as Task).payload as TaskDatabaseCreatePayload).sheetId ||
        undefined
      );
    case "bb.task.database.schema.update":
      return (
        ((task as Task).payload as TaskDatabaseSchemaUpdatePayload).sheetId ||
        undefined
      );
    case "bb.task.database.schema.update-sdl":
      return (
        ((task as Task).payload as TaskDatabaseSchemaUpdateSDLPayload)
          .sheetId || undefined
      );
    case "bb.task.database.data.update":
      return (
        ((task as Task).payload as TaskDatabaseDataUpdatePayload).sheetId ||
        undefined
      );
    case "bb.task.database.schema.update.ghost.sync":
      return (
        ((task as Task).payload as TaskDatabaseSchemaUpdateGhostSyncPayload)
          .sheetId || undefined
      );
    default:
      return undefined;
  }
};
