import type { RouteTarget } from "@/react/router";
import {
  PLAN_DETAIL_PHASE_DEPLOY,
  PROJECT_V1_ROUTE_PLAN_DETAIL,
  PROJECT_V1_ROUTE_PLAN_DETAIL_SPEC_DETAIL,
} from "@/react/router/handles";
import { extractPlanUID } from "@/utils/v1/issue/plan";
import {
  extractPlanUIDFromRolloutName,
  extractStageNameFromTaskName,
  extractStageUID,
  extractTaskUID,
} from "@/utils/v1/issue/rollout";
import { extractProjectResourceName } from "@/utils/v1/project";

// Route-target builders ported vue-free from
// `@/router/dashboard/projectV1RouteHelpers`. Return the structural
// `RouteTarget` ({ name, params, query }) consumed by the React navigation
// shim's push/replace/resolve.

type BuildPlanDeployRouteParams = {
  projectId: string;
  planId: string;
  stageId?: string;
  taskId?: string;
};

export const buildPlanDeployRoute = ({
  projectId,
  planId,
  stageId,
  taskId,
}: BuildPlanDeployRouteParams): RouteTarget => ({
  name: PROJECT_V1_ROUTE_PLAN_DETAIL,
  params: { projectId, planId },
  query: {
    phase: PLAN_DETAIL_PHASE_DEPLOY,
    ...(stageId ? { stageId } : {}),
    ...(taskId ? { taskId } : {}),
  },
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
  query: (currentRoute.query || {}) as Record<string, string | undefined>,
});

export const buildPlanDeployRouteFromPlanName = (
  planName: string,
  options?: Omit<BuildPlanDeployRouteParams, "projectId" | "planId">
): RouteTarget =>
  buildPlanDeployRoute({
    projectId: extractProjectResourceName(planName),
    planId: extractPlanUID(planName) || "_",
    ...options,
  });

export const buildPlanDeployRouteFromRolloutName = (
  rolloutName: string,
  options?: Omit<BuildPlanDeployRouteParams, "projectId" | "planId">
): RouteTarget =>
  buildPlanDeployRoute({
    projectId: extractProjectResourceName(rolloutName),
    planId: extractPlanUIDFromRolloutName(rolloutName) || "_",
    ...options,
  });

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
