import { type ComputedRef, computed, type Ref } from "vue";
import type { Rollout, Stage } from "@/types/proto-es/v1/rollout_service_pb";
import { isTaskRunning, isTaskUnfinished } from "../utils/taskStatus";

export interface UseStageSelectionReturn {
  selectedStage: ComputedRef<Stage | undefined>;
  isStageCreated: (stage: Stage) => boolean;
}

export const useStageSelection = (
  mergedStages: Ref<Stage[]>,
  routeStageId: Ref<string | undefined>,
  rollout: Ref<Rollout>
): UseStageSelectionReturn => {
  const isStageCreated = (stage: Stage): boolean => {
    // A stage is created if it exists in the actual rollout (not just preview)
    return rollout.value.stages.some(
      (s) => s.environment === stage.environment
    );
  };

  const hasRunningTasks = (stage: Stage): boolean => {
    return stage.tasks.some(isTaskRunning);
  };

  const hasUnfinishedTasks = (stage: Stage): boolean => {
    return stage.tasks.some(isTaskUnfinished);
  };

  const findSuitableStage = (stages: Stage[]): Stage | undefined => {
    const createdStages = stages.filter(isStageCreated);

    if (createdStages.length === 0) {
      return undefined;
    }

    // Auto-selection logic (only created stages)
    // Priority 1: First created stage with running tasks
    const runningStage = createdStages.find(hasRunningTasks);
    if (runningStage) {
      return runningStage;
    }

    // Priority 2: First created stage with unfinished tasks
    const unfinishedStage = createdStages.find(hasUnfinishedTasks);
    if (unfinishedStage) {
      return unfinishedStage;
    }

    // Priority 3: Last created stage
    return createdStages[createdStages.length - 1];
  };

  const selectedStage = computed(() => {
    const stages = mergedStages.value;
    if (!routeStageId.value) {
      return findSuitableStage(stages);
    }

    // Always follow route params if stageId is provided (allow both created and uncreated)
    if (routeStageId.value) {
      const stage = stages.find((s) => {
        const stageId = s.name.split("/").pop();
        return stageId === routeStageId.value;
      });
      if (stage) {
        return stage;
      }
    }

    return undefined;
  });

  return {
    selectedStage,
    isStageCreated,
  };
};
