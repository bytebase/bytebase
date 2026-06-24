import { useTranslation } from "react-i18next";
import { type Task, Task_Status } from "@/types/proto-es/v1/rollout_service_pb";

export function DeployTaskSkippedReason({ task }: { task: Task }) {
  const { t } = useTranslation();
  if (task.status !== Task_Status.SKIPPED) {
    return null;
  }
  return (
    <div className="rounded-xs border border-control-border bg-control-bg px-3 py-2 text-sm">
      <span className="text-control-light">{t("task.status.skipped")}: </span>
      {task.skippedReason ? (
        <span className="whitespace-pre-wrap break-words text-control">
          {task.skippedReason}
        </span>
      ) : (
        <span className="italic text-control-placeholder">
          {t("task.skipped-no-reason")}
        </span>
      )}
    </div>
  );
}
