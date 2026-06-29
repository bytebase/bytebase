import { ArrowRight, Eye } from "lucide-react";
import { useMemo } from "react";
import { useTranslation } from "react-i18next";
import { TaskStatusIcon } from "@/react/components/TaskStatusIcon";
import { Button } from "@/react/components/ui/button";
import { useAppStore } from "@/react/stores/app";
import type { Rollout, Stage } from "@/types/proto-es/v1/rollout_service_pb";
import { Task_Status } from "@/types/proto-es/v1/rollout_service_pb";
import { getStageStatus } from "@/utils";
import { PlanDetailTabItem, PlanDetailTabStrip } from "../PlanDetailTabStrip";

function StageProgressCard({ stage }: { stage: Stage }) {
  const environmentList = useAppStore((s) => s.environmentList);
  const env = useMemo(
    () => useAppStore.getState().getEnvironmentByName(stage.environment),
    [environmentList, stage.environment]
  );
  const completed = stage.tasks.filter(
    (task) =>
      task.status === Task_Status.DONE || task.status === Task_Status.SKIPPED
  ).length;
  const stageStatus = getStageStatus(stage);

  return (
    <div className="flex items-center gap-2">
      <TaskStatusIcon size="tiny" status={stageStatus} />
      <span className="text-sm font-medium text-main">{env.title}</span>
      <span className="text-xs text-control-light">
        ({completed}/{stage.tasks.length})
      </span>
    </div>
  );
}

export function DeployStageList({
  hasPendingTasks,
  onOpenPreview,
  onSelectStage,
  rollout,
  selectedStageId,
}: {
  hasPendingTasks: boolean;
  onOpenPreview: () => void;
  onSelectStage: (stage: Stage) => void;
  rollout: Rollout;
  selectedStageId?: string;
}) {
  const { t } = useTranslation();

  return (
    <PlanDetailTabStrip
      action={
        hasPendingTasks ? (
          <Button onClick={onOpenPreview} size="sm">
            <Eye className="h-4 w-4" />
            {t("rollout.pending-tasks-preview.action")}
          </Button>
        ) : undefined
      }
    >
      {rollout.stages.map((stage, index) => (
        <div key={stage.name} className="flex items-center">
          <PlanDetailTabItem
            onSelect={() => onSelectStage(stage)}
            selected={selectedStageId === stage.name}
          >
            <StageProgressCard stage={stage} />
          </PlanDetailTabItem>
          {index < rollout.stages.length - 1 && (
            <ArrowRight className="mx-2 h-4 w-4 shrink-0 text-gray-400" />
          )}
        </div>
      ))}
    </PlanDetailTabStrip>
  );
}
