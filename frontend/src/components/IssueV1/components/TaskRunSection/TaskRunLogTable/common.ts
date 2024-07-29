import { t } from "@/plugins/i18n";
import {
  TaskRunLogEntry_Type,
  taskRunLogEntry_TypeToJSON,
  type TaskRunLogEntry_CommandExecute,
  type TaskRunLogEntry_SchemaDump,
  type TaskRunLogEntry_TaskRunStatusUpdate,
  type TaskRunLogEntry_TransactionControl,
  type TaskRunLogEntry_DatabaseSync,
} from "@/types/proto/v1/rollout_service";

export type FlattenLogEntry = {
  batch: number;
  serial: number;
  type: TaskRunLogEntry_Type;
  deployId: string;
  startTime?: Date;
  endTime?: Date;
  schemaDump?: TaskRunLogEntry_SchemaDump;
  commandExecute?: {
    raw: TaskRunLogEntry_CommandExecute;
    commandIndex: number;
    affectedRows?: number;
  };
  taskRunStatusUpdate?: TaskRunLogEntry_TaskRunStatusUpdate;
  transactionControl?: TaskRunLogEntry_TransactionControl;
  databaseSync?: TaskRunLogEntry_DatabaseSync;
};

export const displayTaskRunLogEntryType = (type: TaskRunLogEntry_Type) => {
  if (type === TaskRunLogEntry_Type.COMMAND_EXECUTE) {
    return t("issue.task-run.task-run-log.entry-type.command-execute");
  }
  if (type === TaskRunLogEntry_Type.SCHEMA_DUMP) {
    return t("issue.task-run.task-run-log.entry-type.schema-dump");
  }
  if (type === TaskRunLogEntry_Type.TASK_RUN_STATUS_UPDATE) {
    return t("issue.task-run.task-run-log.entry-type.task-run-status-update");
  }
  if (type === TaskRunLogEntry_Type.TRANSACTION_CONTROL) {
    return t("issue.task-run.task-run-log.entry-type.transaction-control");
  }
  if (type === TaskRunLogEntry_Type.DATABASE_SYNC) {
    return t("issue.task-run.task-run-log.entry-type.database-sync");
  }

  console.warn(
    `[displayTaskRunLogEntryType] should never reach this line: type=${taskRunLogEntry_TypeToJSON(type)}`
  );
  return "";
};
