import { ArrowRight, Eye } from "lucide-react";
import { useTranslation } from "react-i18next";
import { Button } from "@/react/components/ui/button";
import { cn } from "@/react/lib/utils";
import { useEnvironmentV1Store } from "@/store";
import type { Rollout, Stage } from "@/types/proto-es/v1/rollout_service_pb";
import { Task_Status } from "@/types/proto-es/v1/rollout_service_pb";
import { getStageStatus } from "@/utils";
import { PlanDetailTabItem, PlanDetailTabStrip } from "../PlanDetailTabStrip";

function StageProgressCard({ stage }: { stage: Stage }) {
  const environmentStore = useEnvironmentV1Store();
  const env = environmentStore.getEnvironmentByName(stage.environment);
  const completed = stage.tasks.filter(
    (task) =>
      task.status === Task_Status.DONE || task.status === Task_Status.SKIPPED
  ).length;
  const stageStatus = getStageStatus(stage);
  const dotClass =
    stageStatus === Task_Status.DONE || stageStatus === Task_Status.SKIPPED
      ? "bg-success"
      : stageStatus === Task_Status.FAILED
        ? "bg-error"
        : stageStatus === Task_Status.RUNNING ||
            stageStatus === Task_Status.PENDING
          ? "bg-accent"
          : "bg-control-placeholder";

  return (
    <div className="flex items-center gap-2">
      <span className={cn("h-2 w-2 rounded-full", dotClass)} />
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
