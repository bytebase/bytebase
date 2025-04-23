import type { ComposedRollout } from "@/types";
import type { Task } from "@/types/proto/api/v1alpha/rollout_service";

export const stageForTask = (rollout: ComposedRollout, task: Task) => {
  return rollout.stages.find((stage) =>
    Boolean(stage.tasks.find((t) => t.name === task.name))
  );
};
