import { useRouter } from "vue-router";
import { PROJECT_V1_ROUTE_ROLLOUT_DETAIL_TASK_DETAIL } from "@/router/dashboard/projectV1";
import type { Task } from "@/types/proto-es/v1/rollout_service_pb";
import {
  extractPlanUIDFromRolloutName,
  extractProjectResourceName,
  extractStageNameFromTaskName,
  extractStageUID,
  extractTaskUID,
} from "@/utils";

export const useTaskNavigation = () => {
  const router = useRouter();

  const navigateToTaskDetail = (task: Task) => {
    // Task name format: projects/{project}/rollouts/{rollout}/stages/{stage}/tasks/{task}
    // Use utility functions for consistent parsing
    const projectName = extractProjectResourceName(task.name);
    const planId = extractPlanUIDFromRolloutName(task.name);

    // Extract stage ID from the task's name (Task → Stage relationship)
    const stageName = extractStageNameFromTaskName(task.name);
    const stageId = extractStageUID(stageName);

    const taskId = extractTaskUID(task.name);

    router.push({
      name: PROJECT_V1_ROUTE_ROLLOUT_DETAIL_TASK_DETAIL,
      params: {
        projectId: projectName,
        planId: planId || "-",
        stageId: stageId || "-",
        taskId: taskId || "-",
      },
    });
  };

  return {
    navigateToTaskDetail,
  };
};
