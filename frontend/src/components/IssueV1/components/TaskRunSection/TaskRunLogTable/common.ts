import { t } from "@/plugins/i18n";
import { getDateForPbTimestampProtoEs } from "@/types";
import {
  TaskRunLogEntry_Type,
  type TaskRunLogEntry_CommandExecute,
  type TaskRunLogEntry_SchemaDump,
  type TaskRunLogEntry_TaskRunStatusUpdate,
  type TaskRunLogEntry_TransactionControl,
  type TaskRunLogEntry_DatabaseSync,
  type TaskRunLogEntry,
  type TaskRunLogEntry_PriorBackup,
  type TaskRunLogEntry_RetryInfo,
} from "@/types/proto-es/v1/rollout_service_pb";

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
  priorBackup?: TaskRunLogEntry_PriorBackup;
  retryInfo?: TaskRunLogEntry_RetryInfo;
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
  if (type === TaskRunLogEntry_Type.PRIOR_BACKUP) {
    return t("issue.task-run.task-run-log.entry-type.prior-backup");
  }
  if (type === TaskRunLogEntry_Type.RETRY_INFO) {
    return t("issue.task-run.task-run-log.entry-type.retry-info");
  }

  console.warn(
    `[displayTaskRunLogEntryType] should never reach this line: type=${TaskRunLogEntry_Type[type]}`
  );
  return "";
};

export const convertTaskRunLogEntryToFlattenLogEntries = (
  entry: TaskRunLogEntry,
  batch: number
): FlattenLogEntry[] => {
  const {
    type,
    taskRunStatusUpdate,
    schemaDump,
    commandExecute,
    transactionControl,
    databaseSync,
    deployId,
    priorBackup,
    retryInfo,
  } = entry;
  const flattenLogEntries: FlattenLogEntry[] = [];
  if (
    type === TaskRunLogEntry_Type.TASK_RUN_STATUS_UPDATE &&
    taskRunStatusUpdate
  ) {
    flattenLogEntries.push({
      batch,
      deployId,
      serial: 0,
      type: TaskRunLogEntry_Type.TASK_RUN_STATUS_UPDATE,
      startTime: getDateForPbTimestampProtoEs(entry.logTime),
      endTime: undefined,
      taskRunStatusUpdate,
    });
  }
  if (type === TaskRunLogEntry_Type.DATABASE_SYNC && databaseSync) {
    flattenLogEntries.push({
      batch,
      deployId,
      serial: 0,
      type: TaskRunLogEntry_Type.DATABASE_SYNC,
      startTime: getDateForPbTimestampProtoEs(databaseSync.startTime),
      endTime: getDateForPbTimestampProtoEs(databaseSync.endTime),
      databaseSync,
    });
  }
  if (type === TaskRunLogEntry_Type.TRANSACTION_CONTROL && transactionControl) {
    flattenLogEntries.push({
      batch,
      deployId,
      serial: 0,
      type: TaskRunLogEntry_Type.TRANSACTION_CONTROL,
      startTime: getDateForPbTimestampProtoEs(entry.logTime),
      endTime: undefined,
      transactionControl,
    });
  }
  if (type === TaskRunLogEntry_Type.SCHEMA_DUMP && schemaDump) {
    flattenLogEntries.push({
      batch,
      deployId,
      serial: 0,
      type: TaskRunLogEntry_Type.SCHEMA_DUMP,
      startTime: getDateForPbTimestampProtoEs(schemaDump.startTime),
      endTime: getDateForPbTimestampProtoEs(schemaDump.endTime),
      schemaDump,
    });
  }
  if (type === TaskRunLogEntry_Type.COMMAND_EXECUTE && commandExecute) {
    const { response, logTime: startTime } = commandExecute;
    commandExecute.commandIndexes.forEach((commandIndex, serial) => {
      let affectedRows = response?.affectedRows;
      if (
        commandExecute.commandIndexes.length ===
        response?.allAffectedRows.length
      ) {
        affectedRows = response?.allAffectedRows[serial] ?? affectedRows;
      }
      const endTime = response?.logTime;
      flattenLogEntries.push({
        batch,
        deployId,
        serial,
        type: TaskRunLogEntry_Type.COMMAND_EXECUTE,
        startTime: getDateForPbTimestampProtoEs(startTime),
        endTime: getDateForPbTimestampProtoEs(endTime),
        commandExecute: {
          raw: commandExecute,
          commandIndex,
          affectedRows,
        },
      });
    });
  }
  if (type === TaskRunLogEntry_Type.PRIOR_BACKUP && priorBackup) {
    flattenLogEntries.push({
      batch,
      deployId,
      serial: 0,
      type: TaskRunLogEntry_Type.PRIOR_BACKUP,
      startTime: getDateForPbTimestampProtoEs(priorBackup.startTime),
      endTime: getDateForPbTimestampProtoEs(priorBackup.endTime),
      priorBackup: priorBackup,
    });
  }
  if (type === TaskRunLogEntry_Type.RETRY_INFO && retryInfo) {
    flattenLogEntries.push({
      batch,
      deployId,
      serial: 0,
      type: TaskRunLogEntry_Type.RETRY_INFO,
      startTime: getDateForPbTimestampProtoEs(entry.logTime),
      endTime: undefined,
      retryInfo,
    });
  }
  return flattenLogEntries;
};
