import type { RouteTarget } from "@/app/router";
import {
  PROJECT_V1_ROUTE_PLAN_DETAIL,
  PROJECT_V1_ROUTE_PLAN_DETAIL_SPEC_DETAIL,
  PROJECT_V1_ROUTE_PLAN_ROLLOUT,
  PROJECT_V1_ROUTE_PLAN_ROLLOUT_STAGE,
  PROJECT_V1_ROUTE_PLAN_ROLLOUT_TASK,
} from "@/app/router/handles";
import { extractPlanUID } from "@/utils/v1/issue/plan";
import {
  extractPlanUIDFromRolloutName,
  extractStageNameFromTaskName,
  extractStageUID,
  extractTaskUID,
} from "@/utils/v1/issue/rollout";
import { extractProjectResourceName } from "@/utils/v1/project";

// Route-target builders return the structural `RouteTarget` consumed by the
// application router's push, replace, and resolve methods.

type BuildPlanDeployRouteSelection = { stageId: string; taskId?: string };

type BuildPlanDeployRouteParams = BuildPlanDeployRouteSelection & {
  projectId: string;
  planId: string;
};

export const buildPlanCreateRoute = (
  projectId: string,
  query: Record<string, string>
): RouteTarget => ({
  name: PROJECT_V1_ROUTE_PLAN_DETAIL,
  params: { projectId, planId: "create" },
  query,
});

export const buildPlanDeployRoute = ({
  projectId,
  planId,
  stageId,
  taskId,
}: BuildPlanDeployRouteParams): RouteTarget => {
  if (taskId) {
    return {
      name: PROJECT_V1_ROUTE_PLAN_ROLLOUT_TASK,
      params: { projectId, planId, stageId, taskId },
    };
  }
  return {
    name: PROJECT_V1_ROUTE_PLAN_ROLLOUT_STAGE,
    params: { projectId, planId, stageId },
  };
};

export const buildPlanRolloutRoute = (
  projectId: string,
  planId: string
): RouteTarget => ({
  name: PROJECT_V1_ROUTE_PLAN_ROLLOUT,
  params: { projectId, planId },
});

export const buildSpecDetailRouteForCurrentPage = (
  currentRoute: {
    params?: Record<string, string | string[] | undefined>;
    query?: Record<string, unknown>;
  },
  specId: string
): RouteTarget => ({
  name: PROJECT_V1_ROUTE_PLAN_DETAIL_SPEC_DETAIL,
  params: {
    ...(Object.fromEntries(
      Object.entries(currentRoute.params || {}).map(([k, v]) => [
        k,
        Array.isArray(v) ? (v[0] ?? "") : (v ?? ""),
      ])
    ) as Record<string, string>),
    specId,
  },
});

export const buildPlanRolloutRouteFromPlanName = (
  planName: string
): RouteTarget =>
  buildPlanRolloutRoute(
    extractProjectResourceName(planName),
    extractPlanUID(planName) || "_"
  );

export const buildPlanDeployRouteFromRolloutName = (
  rolloutName: string,
  options?: BuildPlanDeployRouteSelection
): RouteTarget => {
  const projectId = extractProjectResourceName(rolloutName);
  const planId = extractPlanUIDFromRolloutName(rolloutName) || "_";
  if (!options?.stageId) {
    return buildPlanRolloutRoute(projectId, planId);
  }
  return buildPlanDeployRoute({ projectId, planId, ...options });
};

export const buildTaskDetailRoute = (taskName: string): RouteTarget => {
  const stageName = extractStageNameFromTaskName(taskName);
  return buildPlanDeployRoute({
    projectId: extractProjectResourceName(taskName),
    planId: extractPlanUIDFromRolloutName(taskName) || "-",
    stageId: extractStageUID(stageName) || "-",
    taskId: extractTaskUID(taskName) || "-",
  });
};

export const buildStageRoute = (stageName: string): RouteTarget =>
  buildPlanDeployRoute({
    projectId: extractProjectResourceName(stageName),
    planId: extractPlanUIDFromRolloutName(stageName) || "_",
    stageId: extractStageUID(stageName) || "_",
  });
