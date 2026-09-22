import { useTranslation } from "react-i18next";
import { Alert } from "@/components/ui/alert";
import type { TaskRun } from "@/types/proto-es/v1/rollout_service_pb";
import { TaskRun_Status } from "@/types/proto-es/v1/rollout_service_pb";

// The engine's error for a failed run. The task run log below it cannot be
// relied on to carry the cause: several drivers (cassandra, databricks,
// elasticsearch, hive, redis, spanner and starrocks among them) never write a
// command-response entry, so their logs hold only the surrounding sync steps
// and a failure reads as "Database Sync Completed" with no cause anywhere.
//
// The log loads asynchronously, so gating this on the log's contents would
// flip the alert after first paint.
export function TaskRunErrorAlert({ taskRun }: Readonly<{ taskRun: TaskRun }>) {
  const { t } = useTranslation();

  if (taskRun.status !== TaskRun_Status.FAILED || !taskRun.detail) {
    return null;
  }

  return (
    <Alert
      variant="error"
      title={t("common.error")}
      description={
        <span className="whitespace-pre-wrap break-words font-mono text-xs">
          {taskRun.detail}
        </span>
      }
    />
  );
}
