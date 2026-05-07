import {
  ArrowUpRight,
  ChevronDown,
  ChevronRight,
  LoaderCircle,
  Play,
  SkipForward,
  X,
} from "lucide-react";
import { useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { ReadonlyMonaco } from "@/react/components/monaco";
import { Button } from "@/react/components/ui/button";
import { cn } from "@/react/lib/utils";
import { router } from "@/router";
import { getTimeForPbTimestampProtoEs } from "@/types";
import {
  type Stage,
  type Task,
  Task_Status,
  TaskRun_Status,
} from "@/types/proto-es/v1/rollout_service_pb";
import { humanizeTs } from "@/utils";
import {
  isReleaseBasedTask,
  releaseNameOfTaskV1,
} from "@/utils/v1/issue/rollout";
import { usePlanDetailContext } from "../../context/PlanDetailContext";
import { DatabaseTarget } from "../PlanDetailChangesBranch";
import { PlanDetailRollbackSheet } from "../PlanDetailRollbackSheet";
import {
  type PlanDetailTaskRolloutAction,
  PlanDetailTaskRolloutActionPanel,
} from "../PlanDetailTaskRolloutActionPanel";
import { DeployLatestTaskRunInfo } from "./DeployLatestTaskRunInfo";
import { DeployReleaseInfoCard } from "./DeployReleaseInfoCard";
import { DeployTaskStatus } from "./DeployTaskStatus";
import { useDeployTaskActions } from "./taskActions";
import { getTaskRunDuration } from "./taskRunUtils";
import { useDeployTaskStatement } from "./useDeployTaskStatement";

export function DeployTaskItem({
  active,
  isExpanded,
  isSelected,
  isSelectable,
  onToggleExpand,
  onToggleSelect,
  readonly,
  stageId,
  stage,
  task,
}: {
  active: boolean;
  isExpanded: boolean;
  isSelected: boolean;
  isSelectable: boolean;
  onToggleExpand: () => void;
  onToggleSelect: () => void;
  readonly: boolean;
  stageId: string;
  stage: Stage;
  task: Task;
}) {
  const { t } = useTranslation();
  const page = usePlanDetailContext();
  const taskRunsForTask = useMemo(
    () =>
      page.taskRuns.filter((run) =>
        run.name.startsWith(`${task.name}/taskRuns/`)
      ),
    [page.taskRuns, task.name]
  );
  const latestTaskRun = taskRunsForTask[taskRunsForTask.length - 1];
  const [action, setAction] = useState<
    PlanDetailTaskRolloutAction | undefined
  >();
  const [actionOpen, setActionOpen] = useState(false);
  const [rollbackOpen, setRollbackOpen] = useState(false);
  const isReleaseTask = isReleaseBasedTask(task);
  const {
    isLoading: isStatementLoading,
    isTruncated,
    statement,
  } = useDeployTaskStatement({
    enabled: isExpanded && !isReleaseTask,
    task,
  });
  const releaseName = releaseNameOfTaskV1(task);
  const { canCancel, canRun, canSkip } = useDeployTaskActions({
    stage,
    task,
  });
  const rollbackableTaskRun =
    latestTaskRun &&
    latestTaskRun.status === TaskRun_Status.DONE &&
    latestTaskRun.hasPriorBackup
      ? latestTaskRun
      : undefined;
  const scheduledTime =
    task.runTime && task.status === Task_Status.PENDING
      ? humanizeTs(getTimeForPbTimestampProtoEs(task.runTime, 0) / 1000)
      : "";
  const timingDisplay = latestTaskRun ? getTaskRunDuration(latestTaskRun) : "";
  const executorEmail =
    latestTaskRun?.creator.match(/users\/([^/]+)/)?.[1] ?? "";
  const collapsedContextInfo =
    task.status === Task_Status.DONE ? timingDisplay : "";
  const collapsedStatusText =
    latestTaskRun?.detail ||
    task.skippedReason ||
    (task.status === Task_Status.FAILED
      ? t("task.status.failed")
      : task.status === Task_Status.SKIPPED
        ? t("task.status.skipped")
        : "");
  const actionItems = [
    {
      key: "RUN" as const,
      label:
        task.status === Task_Status.FAILED
          ? t("common.retry")
          : t("common.run"),
      visible: canRun,
    },
    {
      key: "SKIP" as const,
      label: t("common.skip"),
      visible: canSkip,
    },
    {
      key: "CANCEL" as const,
      label: t("common.cancel"),
      visible: canCancel,
    },
  ].filter((item) => item.visible);
  const showHeaderActions =
    isExpanded && (actionItems.length > 0 || Boolean(rollbackableTaskRun));

  return (
    <>
      <div
        className={cn(
          "group relative rounded-lg border bg-white transition-all",
          active && "border-accent bg-accent/5"
        )}
      >
        <div
          className={
            isExpanded ? "space-y-3 py-4 pl-3 pr-4" : "py-2.5 pl-3 pr-4"
          }
        >
          <div className="flex items-center justify-between gap-x-3">
            <div className="flex min-w-0 flex-1 items-center gap-x-2">
              <input
                checked={isSelected}
                className="shrink-0 accent-accent"
                disabled={!isSelectable}
                onChange={() => onToggleSelect()}
                onClick={(event) => event.stopPropagation()}
                type="checkbox"
              />
              <DeployTaskStatus
                size={isExpanded ? "large" : "small"}
                status={task.status}
              />
              <div className="flex min-w-0 items-center gap-x-2">
                <div className="min-w-0">
                  <DatabaseTarget target={task.target} />
                </div>
                {isExpanded && !readonly && (
                  <button
                    className="flex shrink-0 items-center gap-x-1 text-xs text-accent transition-opacity hover:opacity-80"
                    onClick={() => {
                      void router.push({
                        query: {
                          phase: "deploy",
                          stageId,
                          taskId: task.name.split("/").pop(),
                        },
                      });
                    }}
                    type="button"
                  >
                    <ArrowUpRight className="h-4 w-4" />
                    <span>{t("common.view-details")}</span>
                  </button>
                )}
                {isExpanded && scheduledTime && (
                  <span className="flex shrink-0 items-center gap-x-1 rounded-full bg-blue-50 px-2 py-0.5 text-xs text-blue-600">
                    <LoaderCircle className="h-3 w-3 animate-spin" />
                    {scheduledTime}
                  </span>
                )}
              </div>
              {!isExpanded && (
                <div className="ml-auto shrink-0 text-xs text-gray-500">
                  {scheduledTime ? (
                    <span className="flex items-center gap-x-1 text-blue-600">
                      <LoaderCircle className="h-3 w-3 animate-spin" />
                      {scheduledTime}
                    </span>
                  ) : task.status === Task_Status.RUNNING && timingDisplay ? (
                    <span className="flex items-center gap-x-1 text-blue-600">
                      <LoaderCircle className="h-3 w-3 animate-spin" />
                      {timingDisplay}
                    </span>
                  ) : collapsedContextInfo ? (
                    <span>{collapsedContextInfo}</span>
                  ) : null}
                </div>
              )}
            </div>

            {showHeaderActions && (
              <div className="flex shrink-0 items-center gap-x-2">
                {actionItems.map((item) => (
                  <Button
                    key={item.key}
                    onClick={() => {
                      setAction(item.key);
                      setActionOpen(true);
                    }}
                    size="xs"
                    variant={item.key === "RUN" ? "default" : "outline"}
                  >
                    {item.key === "RUN" && <Play className="h-3 w-3" />}
                    {item.key === "SKIP" && <SkipForward className="h-3 w-3" />}
                    {item.key === "CANCEL" && <X className="h-3 w-3" />}
                    {item.label}
                  </Button>
                ))}
                {rollbackableTaskRun && (
                  <Button
                    onClick={() => setRollbackOpen(true)}
                    size="xs"
                    variant="outline"
                  >
                    {t("common.rollback")}
                  </Button>
                )}
              </div>
            )}

            {!readonly && (
              <button
                className={
                  isExpanded
                    ? "self-start rounded p-1 hover:bg-gray-100"
                    : "rounded p-1 hover:bg-gray-100"
                }
                onClick={onToggleExpand}
                type="button"
              >
                {isExpanded ? (
                  <ChevronDown className="h-4 w-4 text-gray-500" />
                ) : (
                  <ChevronRight className="h-4 w-4 text-gray-500" />
                )}
              </button>
            )}
          </div>

          {!isExpanded && collapsedStatusText && (
            <div className="mt-1 flex items-center gap-x-2 text-xs">
              {latestTaskRun?.createTime && (
                <span className="rounded-full border bg-gray-50 px-2 py-0.5 text-gray-500">
                  {humanizeTs(
                    getTimeForPbTimestampProtoEs(latestTaskRun.createTime, 0) /
                      1000
                  )}
                </span>
              )}
              <span
                className={cn(
                  "cursor-pointer truncate",
                  task.status === Task_Status.FAILED
                    ? "text-error"
                    : "italic text-gray-500"
                )}
                onClick={onToggleExpand}
              >
                {collapsedStatusText}
              </span>
            </div>
          )}

          {isExpanded && (
            <div className="space-y-3">
              {!isReleaseTask ? (
                <div>
                  <div className="mb-1 text-sm font-medium text-gray-700">
                    {t("common.statement")}
                  </div>
                  {isStatementLoading ? (
                    <div className="rounded-sm border p-3 text-sm text-control-light">
                      {t("common.loading")}
                    </div>
                  ) : statement ? (
                    <>
                      <ReadonlyMonaco
                        className={cn(
                          "relative max-h-64 min-h-[120px] overflow-hidden rounded border text-sm",
                          isTruncated && "rounded-b-none"
                        )}
                        content={statement}
                        language="sql"
                      />
                      {isTruncated && (
                        <div className="rounded-b border border-t-0 bg-gray-50 px-3 py-1.5 text-xs text-gray-500">
                          {t("rollout.task.statement-truncated-hint")}
                        </div>
                      )}
                    </>
                  ) : (
                    <div className="rounded-sm border p-3 text-sm text-control-light">
                      {t("common.no-data")}
                    </div>
                  )}
                </div>
              ) : releaseName ? (
                <DeployReleaseInfoCard compact releaseName={releaseName} />
              ) : null}

              {latestTaskRun && (
                <DeployLatestTaskRunInfo
                  duration={
                    task.status !== Task_Status.PENDING
                      ? timingDisplay
                      : undefined
                  }
                  executorEmail={executorEmail}
                  status={task.status}
                  taskRunName={latestTaskRun.name}
                  updateTime={latestTaskRun.updateTime}
                />
              )}
            </div>
          )}
        </div>
      </div>

      {action && (
        <PlanDetailTaskRolloutActionPanel
          action={action}
          onConfirm={async () => {
            await page.refreshState();
          }}
          onOpenChange={(open) => {
            setActionOpen(open);
            if (!open) {
              setAction(undefined);
            }
          }}
          open={actionOpen}
          target={{ type: "tasks", tasks: [task], stage }}
        />
      )}

      {rollbackableTaskRun && (
        <PlanDetailRollbackSheet
          items={[{ task, taskRun: rollbackableTaskRun }]}
          onOpenChange={setRollbackOpen}
          open={rollbackOpen}
          projectName={`projects/${page.projectId}`}
          rolloutName={page.rollout?.name ?? ""}
        />
      )}
    </>
  );
}
