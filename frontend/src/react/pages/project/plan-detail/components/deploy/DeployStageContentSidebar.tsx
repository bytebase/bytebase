import { DatabaseBackup } from "lucide-react";
import { useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { Button } from "@/react/components/ui/button";
import {
  Sheet,
  SheetBody,
  SheetContent,
  SheetHeader,
  SheetTitle,
} from "@/react/components/ui/sheet";
import { router } from "@/router";
import { useUserStore } from "@/store";
import { getDateForPbTimestampProtoEs } from "@/types";
import {
  type Stage,
  Task_Status,
  TaskRun_Status,
} from "@/types/proto-es/v1/rollout_service_pb";
import { extractDatabaseResourceName, humanizeDate } from "@/utils";
import { usePlanDetailContext } from "../../context/PlanDetailContext";
import { PlanDetailRollbackSheet } from "../PlanDetailRollbackSheet";
import { PlanDetailTaskRunTable } from "../PlanDetailTaskRunTable";
import { DeployTaskStatus } from "./DeployTaskStatus";
import { getTaskRunDuration } from "./taskRunUtils";
import type { RollbackItem } from "./types";

const MAX_DISPLAY_ITEMS = 10;

export function DeployStageContentSidebar({ stage }: { stage: Stage }) {
  const { t } = useTranslation();
  const page = usePlanDetailContext();
  const userStore = useUserStore();
  const [rollbackOpen, setRollbackOpen] = useState(false);
  const [detailOpen, setDetailOpen] = useState(false);
  const [creatorTitleByRun, setCreatorTitleByRun] = useState<
    Record<string, string>
  >({});

  const stageTaskRuns = useMemo(() => {
    const taskNames = new Set(stage.tasks.map((task) => task.name));
    return page.taskRuns.filter((taskRun) => {
      const taskName = taskRun.name.replace(/\/taskRuns\/[^/]+$/, "");
      return taskNames.has(taskName);
    });
  }, [page.taskRuns, stage.tasks]);

  const sortedTaskRuns = useMemo(() => {
    return [...stageTaskRuns].sort((left, right) => {
      const leftTime = left.updateTime ? Number(left.updateTime.seconds) : 0;
      const rightTime = right.updateTime ? Number(right.updateTime.seconds) : 0;
      return rightTime - leftTime;
    });
  }, [stageTaskRuns]);

  const displayedTaskRuns = sortedTaskRuns.slice(0, MAX_DISPLAY_ITEMS);
  const displayedTaskRunsKey = displayedTaskRuns
    .map((taskRun) => `${taskRun.name}:${taskRun.creator}`)
    .join("|");

  useEffect(() => {
    const creators = displayedTaskRuns
      .map((taskRun) => taskRun.creator)
      .filter((creator): creator is string => Boolean(creator));
    if (creators.length === 0) {
      setCreatorTitleByRun({});
      return;
    }

    let canceled = false;
    void userStore.batchGetOrFetchUsers(creators).then(() => {
      if (canceled) return;
      const next = Object.fromEntries(
        displayedTaskRuns.map((taskRun) => {
          const creator = taskRun.creator;
          const user = creator
            ? userStore.getUserByIdentifier(creator)
            : undefined;
          return [
            taskRun.name,
            user?.title ||
              user?.email ||
              creator.match(/users\/([^/]+)/)?.[1] ||
              "",
          ];
        })
      );
      setCreatorTitleByRun(next);
    });

    return () => {
      canceled = true;
    };
  }, [displayedTaskRunsKey, userStore]);

  const rollbackItems = useMemo<RollbackItem[]>(() => {
    return stage.tasks
      .map((task) => {
        const taskRun = page.taskRuns.find(
          (run) =>
            run.name.startsWith(`${task.name}/taskRuns/`) &&
            run.status === TaskRun_Status.DONE &&
            run.hasPriorBackup
        );
        return taskRun ? { task, taskRun } : undefined;
      })
      .filter((item): item is RollbackItem => Boolean(item));
  }, [page.taskRuns, stage.tasks]);

  return (
    <>
      <div className="w-full px-4 py-2">
        {rollbackItems.length > 0 && (
          <div className="pb-2">
            <Button
              className="px-0 text-control hover:bg-transparent"
              onClick={() => setRollbackOpen(true)}
              size="sm"
              variant="ghost"
            >
              <DatabaseBackup className="h-4 w-4" />
              {t("task-run.rollback.available", {
                count: rollbackItems.length,
              })}
            </Button>
          </div>
        )}

        {stageTaskRuns.length > 0 && (
          <div>
            <div className="flex items-center justify-between px-3 py-2">
              <div className="flex items-center gap-2">
                <span className="text-sm font-medium text-gray-700">
                  {t("task.run-history")}
                </span>
              </div>
            </div>

            <div className="flex flex-col gap-y-2 overflow-y-auto px-3 pb-3 pt-1">
              {displayedTaskRuns.map((taskRun) => {
                const taskName = taskRun.name.replace(/\/taskRuns\/[^/]+$/, "");
                const task = stage.tasks.find((item) => item.name === taskName);
                const target = task?.target
                  ? extractDatabaseResourceName(task.target)
                  : undefined;
                const creatorTitle = creatorTitleByRun[taskRun.name] ?? "";
                const duration = getTaskRunDuration(taskRun);
                const updateDate = taskRun.updateTime
                  ? getDateForPbTimestampProtoEs(taskRun.updateTime)
                  : undefined;
                return (
                  <div key={taskRun.name} className="flex items-start gap-2">
                    <div className="pt-0.5">
                      <DeployTaskStatus
                        size="small"
                        status={
                          taskRun.status === TaskRun_Status.DONE
                            ? Task_Status.DONE
                            : taskRun.status === TaskRun_Status.FAILED
                              ? Task_Status.FAILED
                              : taskRun.status === TaskRun_Status.RUNNING
                                ? Task_Status.RUNNING
                                : taskRun.status === TaskRun_Status.PENDING ||
                                    taskRun.status === TaskRun_Status.AVAILABLE
                                  ? Task_Status.PENDING
                                  : Task_Status.CANCELED
                        }
                      />
                    </div>
                    <div className="-mt-0.5 min-w-0 flex-1 px-1 py-0.5">
                      <button
                        className="truncate text-left text-sm leading-4 text-gray-700 hover:underline"
                        onClick={() => {
                          if (!task) return;
                          void router.push({
                            query: {
                              phase: "deploy",
                              stageId: stage.name.split("/").pop(),
                              taskId: task.name.split("/").pop(),
                            },
                          });
                        }}
                        type="button"
                      >
                        {target?.instance ? (
                          <span className="text-gray-500">
                            {target.instance}
                            <span className="mx-0.5 text-gray-400">/</span>
                          </span>
                        ) : null}
                        <span>
                          {target?.databaseName ?? task?.target ?? ""}
                        </span>
                      </button>
                      {taskRun.status === TaskRun_Status.FAILED &&
                        taskRun.detail && (
                          <button
                            className="mt-0.5 line-clamp-3 text-left text-xs text-error hover:underline"
                            onClick={() => setDetailOpen(true)}
                            type="button"
                          >
                            {taskRun.detail}
                          </button>
                        )}
                      <div className="mt-0.5 flex items-center gap-1 text-xs text-gray-400">
                        <span>
                          {updateDate ? humanizeDate(updateDate) : "-"}
                        </span>
                        {creatorTitle && <span>· {creatorTitle}</span>}
                        {duration && <span>· {duration}</span>}
                      </div>
                    </div>
                  </div>
                );
              })}
              {sortedTaskRuns.length > MAX_DISPLAY_ITEMS && (
                <div className="text-left text-xs text-gray-400">
                  {t("task.only-showing-latest-n-runs", {
                    n: MAX_DISPLAY_ITEMS,
                  })}
                </div>
              )}
            </div>
          </div>
        )}
      </div>

      {rollbackItems.length > 0 && (
        <PlanDetailRollbackSheet
          items={rollbackItems}
          onOpenChange={setRollbackOpen}
          open={rollbackOpen}
          projectName={`projects/${page.projectId}`}
          rolloutName={page.rollout?.name ?? ""}
        />
      )}

      <Sheet onOpenChange={setDetailOpen} open={detailOpen}>
        <SheetContent width="wide">
          <SheetHeader>
            <SheetTitle>{t("common.detail")}</SheetTitle>
          </SheetHeader>
          <SheetBody>
            <PlanDetailTaskRunTable taskRuns={sortedTaskRuns} />
          </SheetBody>
        </SheetContent>
      </Sheet>
    </>
  );
}
