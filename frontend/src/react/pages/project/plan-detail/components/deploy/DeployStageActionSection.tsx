import { useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { Button } from "@/react/components/ui/button";
import {
  type Stage,
  Task_Status,
  TaskRun_Status,
} from "@/types/proto-es/v1/rollout_service_pb";
import { usePlanDetailContext } from "../../shell/PlanDetailContext";
import { PlanDetailRollbackSheet } from "../PlanDetailRollbackSheet";
import {
  type PlanDetailTaskRolloutAction,
  PlanDetailTaskRolloutActionPanel,
} from "../PlanDetailTaskRolloutActionPanel";
import type { RollbackItem } from "./types";

export function DeployStageActionSection({ stage }: { stage: Stage }) {
  const { t } = useTranslation();
  const page = usePlanDetailContext();
  const [actionOpen, setActionOpen] = useState(false);
  const [action, setAction] = useState<PlanDetailTaskRolloutAction | undefined>(
    undefined
  );
  const [rollbackOpen, setRollbackOpen] = useState(false);

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

  const actionItems = [
    {
      key: "RUN" as const,
      label: t("common.run"),
      visible: stage.tasks.some((task) =>
        [
          Task_Status.NOT_STARTED,
          Task_Status.CANCELED,
          Task_Status.FAILED,
        ].includes(task.status)
      ),
    },
    {
      key: "SKIP" as const,
      label: t("common.skip"),
      visible: stage.tasks.some((task) =>
        [
          Task_Status.NOT_STARTED,
          Task_Status.CANCELED,
          Task_Status.FAILED,
        ].includes(task.status)
      ),
    },
    {
      key: "CANCEL" as const,
      label: t("common.cancel"),
      visible: stage.tasks.some((task) =>
        [Task_Status.PENDING, Task_Status.RUNNING].includes(task.status)
      ),
    },
  ].filter((item) => item.visible);

  return (
    <>
      <div className="flex flex-wrap items-center gap-x-2 gap-y-2">
        {actionItems.map((item) => (
          <Button
            key={`${stage.name}-${item.key}`}
            onClick={() => {
              setAction(item.key);
              setActionOpen(true);
            }}
            size="xs"
            variant="outline"
          >
            {item.label}
          </Button>
        ))}
        {rollbackItems.length > 0 && (
          <Button
            onClick={() => setRollbackOpen(true)}
            size="xs"
            variant="outline"
          >
            {t("task-run.rollback.available", { count: rollbackItems.length })}
          </Button>
        )}
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
          target={{ type: "tasks", stage }}
        />
      )}

      {rollbackItems.length > 0 && (
        <PlanDetailRollbackSheet
          items={rollbackItems}
          onOpenChange={setRollbackOpen}
          open={rollbackOpen}
          projectName={`projects/${page.projectId}`}
          rolloutName={page.rollout?.name ?? ""}
        />
      )}
    </>
  );
}
