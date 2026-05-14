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
import { useDatabaseV1Store, useUserStore } from "@/store";
import { getDateForPbTimestampProtoEs } from "@/types";
import {
  type Stage,
  type TaskRun,
  TaskRun_Status,
} from "@/types/proto-es/v1/rollout_service_pb";
import { humanizeDate } from "@/utils";
import { extractTaskNameFromTaskRunName } from "@/utils/v1/issue/rollout";
import { usePlanDetailContext } from "../../shell/PlanDetailContext";
import { DatabaseTarget } from "../PlanDetailChangesBranch";
import { PlanDetailRollbackSheet } from "../PlanDetailRollbackSheet";
import { PlanDetailTaskRunDetail } from "../PlanDetailTaskRunDetail";
import { DeployTaskStatus } from "./DeployTaskStatus";
import { getTaskRunDuration, taskRunStatusToTaskStatus } from "./taskRunUtils";
import type { RollbackItem } from "./types";

const MAX_DISPLAY_ITEMS = 10;

export function DeployStageContentSidebar({ stage }: { stage: Stage }) {
  const { t } = useTranslation();
  const page = usePlanDetailContext();
  const userStore = useUserStore();
  const databaseStore = useDatabaseV1Store();
  const [rollbackOpen, setRollbackOpen] = useState(false);
  const [selectedTaskRun, setSelectedTaskRun] = useState<TaskRun | undefined>();
  const [creatorTitleByRun, setCreatorTitleByRun] = useState<
    Record<string, string>
  >({});

  const tasksByName = useMemo(
    () => new Map(stage.tasks.map((task) => [task.name, task])),
    [stage.tasks]
  );

  const selectedDatabaseEngine = useMemo(() => {
    if (!selectedTaskRun) return undefined;
    const target = tasksByName.get(
      extractTaskNameFromTaskRunName(selectedTaskRun.name)
    )?.target;
    return target
      ? databaseStore.getDatabaseByName(target).instanceResource?.engine
      : undefined;
  }, [databaseStore, selectedTaskRun, tasksByName]);

  const stageTaskRuns = useMemo(() => {
    return page.taskRuns.filter((taskRun) =>
      tasksByName.has(extractTaskNameFromTaskRunName(taskRun.name))
    );
  }, [page.taskRuns, tasksByName]);

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
                const task = tasksByName.get(
                  extractTaskNameFromTaskRunName(taskRun.name)
                );
                if (!task) return null;
                const creatorTitle = creatorTitleByRun[taskRun.name] ?? "";
                const duration = getTaskRunDuration(taskRun);
                const updateDate = taskRun.updateTime
                  ? getDateForPbTimestampProtoEs(taskRun.updateTime)
                  : undefined;
                return (
                  <button
                    key={taskRun.name}
                    className="flex w-full items-start gap-2 rounded-sm px-1 py-1 text-left hover:bg-gray-50"
                    onClick={() => setSelectedTaskRun(taskRun)}
                    type="button"
                  >
                    <div className="pt-0.5">
                      <DeployTaskStatus
                        size="small"
                        status={taskRunStatusToTaskStatus(taskRun.status)}
                      />
                    </div>
                    <div className="-mt-0.5 min-w-0 flex-1 px-1 py-0.5">
                      <DatabaseTarget target={task.target} />
                      {taskRun.status === TaskRun_Status.FAILED &&
                        taskRun.detail && (
                          <div className="mt-0.5 line-clamp-3 text-xs text-error">
                            {taskRun.detail}
                          </div>
                        )}
                      <div className="mt-0.5 flex items-center gap-1 text-xs text-gray-400">
                        <span>
                          {updateDate ? humanizeDate(updateDate) : "-"}
                        </span>
                        {creatorTitle && <span>· {creatorTitle}</span>}
                        {duration && <span>· {duration}</span>}
                      </div>
                    </div>
                  </button>
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

      <Sheet
        onOpenChange={(open) => !open && setSelectedTaskRun(undefined)}
        open={Boolean(selectedTaskRun)}
      >
        <SheetContent width="wide">
          <SheetHeader>
            <SheetTitle>{t("common.detail")}</SheetTitle>
          </SheetHeader>
          <SheetBody>
            {selectedTaskRun && (
              <PlanDetailTaskRunDetail
                databaseEngine={selectedDatabaseEngine}
                taskRun={selectedTaskRun}
              />
            )}
          </SheetBody>
        </SheetContent>
      </Sheet>
    </>
  );
}
