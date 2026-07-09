import { Clock3, History, User } from "lucide-react";
import { useTranslation } from "react-i18next";
import { HumanizeTs } from "@/react/components/HumanizeTs";
import { TaskRunStatusIcon } from "@/react/components/TaskRunStatusIcon";
import { TaskRunLogViewer } from "@/react/components/task-run-log";
import { Button } from "@/react/components/ui/button";
import { Tooltip } from "@/react/components/ui/tooltip";
import { executorEmailOfTaskRun } from "@/react/lib/taskRun";
import { cn } from "@/react/lib/utils";
import { getDateForPbTimestampProtoEs } from "@/types";
import type { Engine } from "@/types/proto-es/v1/common_pb";
import { Engine as EngineEnum } from "@/types/proto-es/v1/common_pb";
import type { TaskRun } from "@/types/proto-es/v1/rollout_service_pb";
import { TaskRun_Status } from "@/types/proto-es/v1/rollout_service_pb";
import { stringifyTaskRunStatus } from "@/utils/v1/issue/rollout";
import { PlanDetailTaskRunSession } from "../PlanDetailTaskRunSession";

const STATUS_LABEL_CLASS: Partial<Record<TaskRun_Status, string>> = {
  [TaskRun_Status.RUNNING]: "text-info",
  [TaskRun_Status.DONE]: "text-success",
  [TaskRun_Status.FAILED]: "text-error",
};

export function DeployLatestTaskRunInfo({
  active = true,
  databaseEngine,
  duration,
  historyCount,
  onShowHistory,
  taskRun,
}: {
  // When false, the card's stage is hidden (kept mounted); pause the live log
  // and session polls below.
  active?: boolean;
  databaseEngine?: Engine;
  duration?: string;
  historyCount: number;
  onShowHistory: () => void;
  taskRun: TaskRun;
}) {
  const { t } = useTranslation();
  const updateDate = getDateForPbTimestampProtoEs(taskRun.updateTime);
  const executorEmail = executorEmailOfTaskRun(taskRun);
  const showSession =
    taskRun.status === TaskRun_Status.RUNNING &&
    databaseEngine === EngineEnum.POSTGRES;

  return (
    <div className="flex flex-col gap-2">
      <div className="flex flex-wrap items-center gap-x-2 gap-y-1 text-sm">
        <span className="shrink-0 font-medium text-control">
          {t("task-run.latest")}
        </span>
        <span className="text-control-placeholder">·</span>
        <span
          className={cn(
            "flex shrink-0 items-center gap-x-1",
            STATUS_LABEL_CLASS[taskRun.status] ?? "text-control-light"
          )}
        >
          <TaskRunStatusIcon size="tiny" status={taskRun.status} />
          <span>{stringifyTaskRunStatus(taskRun.status, t)}</span>
        </span>
        {updateDate && (
          <>
            <span className="text-control-placeholder">·</span>
            <HumanizeTs
              ts={updateDate.getTime() / 1000}
              className="shrink-0 text-control-light"
            />
          </>
        )}
        {executorEmail && (
          <>
            <span className="text-control-placeholder">·</span>
            <Tooltip content={t("task.executed-by")}>
              <span className="flex shrink-0 items-center gap-x-1 text-control-light">
                <User className="size-3.5" />
                <span className="truncate">{executorEmail}</span>
              </span>
            </Tooltip>
          </>
        )}
        {duration && (
          <>
            <span className="text-control-placeholder">·</span>
            <Tooltip content={t("common.duration")}>
              <span className="flex shrink-0 items-center gap-x-1 tabular-nums text-control-light">
                <Clock3 className="size-3.5" />
                {duration}
              </span>
            </Tooltip>
          </>
        )}
        {historyCount > 1 && (
          <span className="ml-auto flex shrink-0 items-center">
            <Button onClick={onShowHistory} size="xs" variant="ghost">
              <History className="size-3.5" />
              {t("task-run.history-with-count", { count: historyCount })}
            </Button>
          </span>
        )}
      </div>

      {/* Status is part of the key on purpose: a status flip (RUNNING→DONE)
          remounts the viewer for a fresh disclosure state on the new phase.
          The remount paints fully formed — useTaskRunLogData seeds from its
          caches during render — and the status prop still drives terminal
          cache freshness and the live poll while running. */}
      <TaskRunLogViewer
        active={active}
        key={`logs-${taskRun.name}-${taskRun.status}`}
        taskRunName={taskRun.name}
        taskRunStatus={taskRun.status}
      />

      {showSession && (
        <div className="flex flex-col gap-1">
          <div className="text-sm font-medium text-control">
            {t("issue.task-run.session")}
          </div>
          <PlanDetailTaskRunSession
            active={active}
            key={taskRun.name}
            taskRun={taskRun}
          />
        </div>
      )}
    </div>
  );
}
