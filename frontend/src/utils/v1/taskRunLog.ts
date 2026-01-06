import { t } from "@/plugins/i18n";
import { TaskRunLogEntry_Type } from "@/types/proto-es/v1/rollout_service_pb";

export const displayTaskRunLogEntryType = (type: TaskRunLogEntry_Type) => {
  if (type === TaskRunLogEntry_Type.COMMAND_EXECUTE) {
    return t("issue.task-run.task-run-log.entry-type.command-execute");
  }
  if (type === TaskRunLogEntry_Type.SCHEMA_DUMP) {
    return t("issue.task-run.task-run-log.entry-type.schema-dump");
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
  if (type === TaskRunLogEntry_Type.COMPUTE_DIFF) {
    return t("issue.task-run.task-run-log.entry-type.compute-diff");
  }
  if (type === TaskRunLogEntry_Type.RETRY_INFO) {
    return t("issue.task-run.task-run-log.entry-type.retry-info");
  }

  console.warn(
    `[displayTaskRunLogEntryType] should never reach this line: type=${TaskRunLogEntry_Type[type]}`
  );
  return "";
};
