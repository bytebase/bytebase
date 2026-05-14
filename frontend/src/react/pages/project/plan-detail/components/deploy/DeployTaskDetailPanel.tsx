import { CalendarClock, EllipsisVertical, Undo2 } from "lucide-react";
import { useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { ReadonlyMonaco } from "@/react/components/monaco";
import { Button } from "@/react/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/react/components/ui/dropdown-menu";
import { Tooltip } from "@/react/components/ui/tooltip";
import { useEnvironmentV1Store } from "@/store";
import { getTimeForPbTimestampProtoEs } from "@/types";
import type { Task } from "@/types/proto-es/v1/rollout_service_pb";
import {
  Task_Status,
  TaskRun_Status,
} from "@/types/proto-es/v1/rollout_service_pb";
import { databaseForTask, humanizeTs } from "@/utils";
import {
  isReleaseBasedTask,
  releaseNameOfTaskV1,
} from "@/utils/v1/issue/rollout";
import { usePlanDetailContext } from "../../shell/PlanDetailContext";
import { DatabaseTarget } from "../PlanDetailChangesBranch";
import { PlanDetailRollbackSheet } from "../PlanDetailRollbackSheet";
import {
  type PlanDetailTaskRolloutAction,
  PlanDetailTaskRolloutActionPanel,
} from "../PlanDetailTaskRolloutActionPanel";
import { PlanDetailTaskRunDetail } from "../PlanDetailTaskRunDetail";
import { PlanDetailTaskRunTable } from "../PlanDetailTaskRunTable";
import { DeployReleaseInfoCard } from "./DeployReleaseInfoCard";
import { DeployTaskStatus } from "./DeployTaskStatus";
import { useDeployTaskActions } from "./taskActions";
import { useDeployTaskStatement } from "./useDeployTaskStatement";

export function DeployTaskDetailPanel({ task }: { task: Task }) {
  const { t } = useTranslation();
  const page = usePlanDetailContext();
  const environmentStore = useEnvironmentV1Store();
  const project = page.project;
  const taskRuns = useMemo(
    () =>
      page.taskRuns
        .filter((taskRun) => taskRun.name.startsWith(`${task.name}/taskRuns/`))
        .sort((left, right) => {
          const leftTime = left.createTime
            ? Number(left.createTime.seconds)
            : 0;
          const rightTime = right.createTime
            ? Number(right.createTime.seconds)
            : 0;
          return rightTime - leftTime;
        }),
    [page.taskRuns, task.name]
  );
  const latestTaskRun = taskRuns[0];
  const [taskActionOpen, setTaskActionOpen] = useState(false);
  const [taskAction, setTaskAction] = useState<
    PlanDetailTaskRolloutAction | undefined
  >(undefined);
  const [rollbackOpen, setRollbackOpen] = useState(false);
  const database = useMemo(
    () => databaseForTask(project, task, page.plan),
    [page.plan, project, task]
  );
  const stage = useMemo(
    () =>
      page.rollout?.stages.find((candidate) =>
        candidate.tasks.some((item) => item.name === task.name)
      ),
    [page.rollout?.stages, task.name]
  );
  const stageTitle = stage?.environment
    ? environmentStore.getEnvironmentByName(stage.environment).title
    : "";
  const scheduledTimeDisplay = useMemo(() => {
    const ts = getTimeForPbTimestampProtoEs(task.runTime, 0);
    if (!ts) return "";
    return humanizeTs(ts / 1000);
  }, [task.runTime]);
  const isReleaseTask = useMemo(() => isReleaseBasedTask(task), [task]);
  const {
    isLoading: isStatementLoading,
    isTruncated,
    statement,
  } = useDeployTaskStatement({
    enabled: !isReleaseTask,
    task,
  });
  const releaseName = useMemo(() => releaseNameOfTaskV1(task), [task]);
  const rollbackableTaskRun =
    latestTaskRun &&
    latestTaskRun.status === TaskRun_Status.DONE &&
    latestTaskRun.hasPriorBackup
      ? latestTaskRun
      : undefined;
  const { canCancel, canRun, canSkip } = useDeployTaskActions({ stage, task });
  const primaryActionLabel =
    task.status === Task_Status.FAILED ? t("common.retry") : t("common.run");

  return (
    <div className="space-y-4 p-4">
      <div className="flex flex-col gap-y-3">
        <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
          <div className="flex flex-row flex-wrap items-center gap-x-3 gap-y-2">
            <DeployTaskStatus size="large" status={task.status} />
            {stageTitle && (
              <span className="rounded-full bg-control-bg px-2 py-0.5 text-xs text-control">
                {stageTitle}
              </span>
            )}
            <div className="min-w-0 text-xl">
              <DatabaseTarget showEnvironment target={database.name} />
            </div>
            {scheduledTimeDisplay && task.status === Task_Status.PENDING && (
              <Tooltip
                content={
                  <div className="flex flex-col gap-y-1">
                    <div className="text-sm opacity-80">
                      {t("task.scheduled-time")}
                    </div>
                    <div className="whitespace-nowrap text-sm">
                      {scheduledTimeDisplay}
                    </div>
                  </div>
                }
              >
                <span className="inline-flex items-center gap-1 rounded-full bg-control-bg px-2 py-0.5 text-xs text-control">
                  <CalendarClock className="h-3.5 w-3.5 opacity-80" />
                  <span>{scheduledTimeDisplay}</span>
                </span>
              </Tooltip>
            )}
          </div>

          <div className="flex justify-start sm:justify-end">
            <div className="flex flex-wrap items-center gap-x-2 gap-y-2">
              {canRun && (
                <Button
                  onClick={() => {
                    setTaskAction("RUN");
                    setTaskActionOpen(true);
                  }}
                  size="sm"
                >
                  {primaryActionLabel}
                </Button>
              )}
              {(canSkip || canCancel) && (
                <DropdownMenu>
                  <DropdownMenuTrigger className="inline-flex h-8 items-center justify-center rounded-xs px-1 text-sm text-control outline-hidden hover:bg-control-bg focus-visible:ring-2 focus-visible:ring-accent">
                    <EllipsisVertical className="h-4 w-4" />
                  </DropdownMenuTrigger>
                  <DropdownMenuContent>
                    {canSkip && (
                      <DropdownMenuItem
                        onClick={() => {
                          setTaskAction("SKIP");
                          setTaskActionOpen(true);
                        }}
                      >
                        {t("common.skip")}
                      </DropdownMenuItem>
                    )}
                    {canCancel && (
                      <DropdownMenuItem
                        onClick={() => {
                          setTaskAction("CANCEL");
                          setTaskActionOpen(true);
                        }}
                      >
                        {t("common.cancel")}
                      </DropdownMenuItem>
                    )}
                  </DropdownMenuContent>
                </DropdownMenu>
              )}
              {rollbackableTaskRun && (
                <Button onClick={() => setRollbackOpen(true)} size="sm">
                  <Undo2 className="h-4 w-4" />
                  {t("common.rollback")}
                </Button>
              )}
            </div>
          </div>
        </div>
      </div>

      {latestTaskRun && (
        <div className="flex w-full flex-col gap-2">
          <span className="textlabel uppercase">
            {t("task-run.latest-logs")}
          </span>
          <PlanDetailTaskRunDetail
            databaseEngine={database.instanceResource?.engine}
            taskRun={latestTaskRun}
          />
        </div>
      )}

      {taskRuns.length > 0 && (
        <div className="space-y-2">
          <span className="textlabel uppercase">{t("task-run.history")}</span>
          <PlanDetailTaskRunTable
            databaseName={database.name}
            taskRuns={taskRuns}
          />
        </div>
      )}

      {!isReleaseTask && (
        <div className="flex min-h-0 flex-1 flex-col gap-2">
          <span className="textlabel uppercase">{t("common.statement")}</span>
          <div className="rounded-sm border bg-white p-3">
            {isStatementLoading ? (
              <div className="text-sm text-control-light">
                {t("common.loading")}
              </div>
            ) : statement ? (
              <>
                <ReadonlyMonaco
                  className="relative h-auto max-h-[420px] min-h-[160px]"
                  content={statement}
                  language="sql"
                />
                {isTruncated && (
                  <div className="mt-2 text-xs text-gray-500">
                    {t("rollout.task.statement-truncated-hint")}
                  </div>
                )}
              </>
            ) : (
              <div className="text-sm text-control-light">
                {t("common.no-data")}
              </div>
            )}
          </div>
        </div>
      )}

      {isReleaseTask && releaseName && (
        <DeployReleaseInfoCard
          className="w-full"
          compact
          releaseName={releaseName}
        />
      )}

      {stage && taskAction && (
        <PlanDetailTaskRolloutActionPanel
          action={taskAction}
          onConfirm={async () => {
            await page.refreshState();
          }}
          onOpenChange={(open) => {
            setTaskActionOpen(open);
            if (!open) {
              setTaskAction(undefined);
            }
          }}
          open={taskActionOpen}
          target={{ type: "tasks", stage, tasks: [task] }}
        />
      )}

      {rollbackableTaskRun && (
        <PlanDetailRollbackSheet
          items={[{ task, taskRun: rollbackableTaskRun }]}
          onOpenChange={setRollbackOpen}
          open={rollbackOpen}
          projectName={project.name}
          rolloutName={page.rollout?.name ?? ""}
        />
      )}
    </div>
  );
}
