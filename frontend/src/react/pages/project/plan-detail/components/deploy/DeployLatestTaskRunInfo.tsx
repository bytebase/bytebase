import type { Timestamp } from "@bufbuild/protobuf/wkt";
import {
  CheckCircle2,
  Circle,
  Clock3,
  LoaderCircle,
  User,
  XCircle,
} from "lucide-react";
import { useTranslation } from "react-i18next";
import { TaskRunLogViewer } from "@/react/components/task-run-log";
import { Tooltip } from "@/react/components/ui/tooltip";
import { getDateForPbTimestampProtoEs } from "@/types";
import { TaskRun_Status } from "@/types/proto-es/v1/rollout_service_pb";
import { formatAbsoluteDateTime, humanizeDate } from "@/utils";

export function DeployLatestTaskRunInfo({
  duration,
  executorEmail,
  status,
  taskRunName,
  updateTime,
}: {
  duration?: string;
  executorEmail?: string;
  status: TaskRun_Status;
  taskRunName?: string;
  updateTime?: Timestamp;
}) {
  const { t } = useTranslation();
  const updateDate = getDateForPbTimestampProtoEs(updateTime);
  const statusConfig = (() => {
    switch (status) {
      case TaskRun_Status.RUNNING:
        return {
          className: "text-blue-600",
          icon: LoaderCircle,
          label: t("task.status.running"),
          spinning: true,
        };
      case TaskRun_Status.DONE:
        return {
          className: "text-green-600",
          icon: CheckCircle2,
          label: t("task.status.done"),
          spinning: false,
        };
      case TaskRun_Status.FAILED:
        return {
          className: "text-red-600",
          icon: XCircle,
          label: t("task.status.failed"),
          spinning: false,
        };
      case TaskRun_Status.CANCELED:
        return {
          className: "text-gray-500",
          icon: Circle,
          label: t("task.status.canceled"),
          spinning: false,
        };
      default:
        return {
          className: "text-gray-500",
          icon: Circle,
          label: t("task.status.pending"),
          spinning: false,
        };
    }
  })();
  const StatusIcon = statusConfig.icon;

  return (
    <div className="space-y-2">
      <div className="flex flex-wrap items-center gap-x-2 gap-y-1 text-sm">
        <span className="shrink-0 font-medium text-gray-700">
          {t("task-run.latest")}
        </span>
        <span className="text-gray-300">·</span>
        <span
          className={`flex shrink-0 items-center gap-x-1 ${statusConfig.className}`}
        >
          <StatusIcon
            className={`h-4 w-4 ${statusConfig.spinning ? "animate-spin" : ""}`}
          />
          <span>{statusConfig.label}</span>
        </span>
        {updateDate && (
          <>
            <span className="text-gray-300">·</span>
            <Tooltip content={formatAbsoluteDateTime(updateDate.getTime())}>
              <span className="shrink-0 text-gray-500">
                {humanizeDate(updateDate)}
              </span>
            </Tooltip>
          </>
        )}
        {executorEmail && (
          <>
            <span className="text-gray-300">·</span>
            <Tooltip content={t("task.executed-by")}>
              <span className="flex shrink-0 items-center gap-x-1 text-gray-500">
                <User className="h-3.5 w-3.5" />
                <span className="truncate">{executorEmail}</span>
              </span>
            </Tooltip>
          </>
        )}
        {duration && (
          <>
            <span className="text-gray-300">·</span>
            <Tooltip content={t("common.duration")}>
              <span className="flex shrink-0 items-center gap-x-1 text-gray-500">
                <Clock3 className="h-3.5 w-3.5" />
                {duration}
              </span>
            </Tooltip>
          </>
        )}
      </div>

      {taskRunName && (
        <TaskRunLogViewer
          key={`${taskRunName}-${status}`}
          taskRunName={taskRunName}
        />
      )}
    </div>
  );
}
