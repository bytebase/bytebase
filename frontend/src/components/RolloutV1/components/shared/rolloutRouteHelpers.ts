import type { RouteLocationRaw } from "vue-router";
import {
  buildStageRoute as buildPlanDetailStageRoute,
  buildTaskDetailRoute as buildPlanDetailTaskRoute,
} from "@/router/dashboard/projectV1RouteHelpers";

/**
 * Build a route location for the task detail view.
 * Task name format: projects/{project}/plans/{plan}/rollout/stages/{stage}/tasks/{task}
 */
export const buildTaskDetailRoute = (taskName: string): RouteLocationRaw => {
  return buildPlanDetailTaskRoute(taskName);
};

/**
 * Build a route location for the stage view.
 * Stage name format: projects/{project}/plans/{plan}/rollout/stages/{stage}
 */
export const buildStageRoute = (stageName: string): RouteLocationRaw => {
  return buildPlanDetailStageRoute(stageName);
};
