import { useTranslation } from "react-i18next";
import { type Task, Task_Status } from "@/types/proto-es/v1/rollout_service_pb";
import { formatAbsoluteDateTime } from "@/utils/datetime";

// "Skipped by <user> · <time> · <reason>": who bypassed the task, when and why.
// Tasks skipped before the skipper was recorded show the reason alone.
export function DeployTaskSkippedReason({ task }: { task: Task }) {
  const { t } = useTranslation();
  if (task.status !== Task_Status.SKIPPED) {
    return null;
  }
  const skipTimeMs = task.skipTime
    ? Number(task.skipTime.seconds) * 1000
    : undefined;
  return (
    <div className="rounded-xs border border-control-border bg-control-bg px-3 py-2 text-sm">
      <span className="text-control-light">
        {task.skipper
          ? t("task.folded.skipped-by", {
              user: task.skipper.replace(/^users\//, ""),
            })
          : t("task.status.skipped")}
        {skipTimeMs ? ` · ${formatAbsoluteDateTime(skipTimeMs)}` : ""}
        {task.skipper ? " · " : ": "}
      </span>
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
