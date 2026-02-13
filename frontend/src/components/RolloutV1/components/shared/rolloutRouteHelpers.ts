import type { RouteLocationRaw } from "vue-router";
import {
  PROJECT_V1_ROUTE_PLAN_ROLLOUT_STAGE,
  PROJECT_V1_ROUTE_PLAN_ROLLOUT_TASK,
} from "@/router/dashboard/projectV1";
import {
  extractPlanUIDFromRolloutName,
  extractProjectResourceName,
  extractStageNameFromTaskName,
  extractStageUID,
  extractTaskUID,
} from "@/utils";

/**
 * Build a route location for the task detail view.
 * Task name format: projects/{project}/plans/{plan}/rollout/stages/{stage}/tasks/{task}
 */
export const buildTaskDetailRoute = (taskName: string): RouteLocationRaw => {
  const projectName = extractProjectResourceName(taskName);
  const planId = extractPlanUIDFromRolloutName(taskName);
  const stageName = extractStageNameFromTaskName(taskName);
  const stageId = extractStageUID(stageName);
  const taskId = extractTaskUID(taskName);

  return {
    name: PROJECT_V1_ROUTE_PLAN_ROLLOUT_TASK,
    params: {
      projectId: projectName,
      planId: planId || "-",
      stageId: stageId || "-",
      taskId: taskId || "-",
    },
  };
};

/**
 * Build a route location for the stage view.
 * Stage name format: projects/{project}/plans/{plan}/rollout/stages/{stage}
 */
export const buildStageRoute = (stageName: string): RouteLocationRaw => {
  const projectName = extractProjectResourceName(stageName);
  const planId = extractPlanUIDFromRolloutName(stageName);
  const stageId = stageName.split("/").pop();

  return {
    name: PROJECT_V1_ROUTE_PLAN_ROLLOUT_STAGE,
    params: {
      projectId: projectName,
      planId: planId || "_",
      stageId: stageId || "_",
    },
  };
};
