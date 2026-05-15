import { DatabaseBackup } from "lucide-react";
import { useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { Button } from "@/react/components/ui/button";
import {
  type Stage,
  TaskRun_Status,
} from "@/types/proto-es/v1/rollout_service_pb";
import { usePlanDetailContext } from "../../shell/PlanDetailContext";
import { PlanDetailRollbackSheet } from "../PlanDetailRollbackSheet";
import type { RollbackItem } from "./types";

export function DeployStageRollbackSection({ stage }: { stage: Stage }) {
  const { t } = useTranslation();
  const page = usePlanDetailContext();
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

  if (rollbackItems.length === 0) {
    return null;
  }

  return (
    <>
      <div className="w-full px-4 pb-2">
        <Button onClick={() => setRollbackOpen(true)} size="sm" variant="ghost">
          <DatabaseBackup className="h-4 w-4" />
          {t("task-run.rollback.available", {
            count: rollbackItems.length,
          })}
        </Button>
      </div>

      <PlanDetailRollbackSheet
        items={rollbackItems}
        onOpenChange={setRollbackOpen}
        open={rollbackOpen}
        projectName={`projects/${page.projectId}`}
        rolloutName={page.rollout?.name ?? ""}
      />
    </>
  );
}
