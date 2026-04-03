import type {
  LocationQueryValue,
  RouteLocationNormalizedLoaded,
  RouteLocationRaw,
} from "vue-router";
import {
  extractPlanUID,
  extractPlanUIDFromRolloutName,
  extractProjectResourceName,
  extractStageNameFromTaskName,
  extractStageUID,
  extractTaskUID,
} from "@/utils";
import {
  PROJECT_V1_ROUTE_PLAN_DETAIL,
  PROJECT_V1_ROUTE_PLAN_DETAIL_SPEC_DETAIL,
} from "./projectV1";

export const PLAN_DETAIL_PHASE_DEPLOY = "deploy";

export const getRouteQueryString = (
  value?: LocationQueryValue | LocationQueryValue[]
): string | undefined => {
  if (typeof value === "string") {
    return value;
  }
  if (Array.isArray(value)) {
    return typeof value[0] === "string" ? value[0] : undefined;
  }
  return undefined;
};

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
}: BuildPlanDeployRouteParams): RouteLocationRaw => {
  return {
    name: PROJECT_V1_ROUTE_PLAN_DETAIL,
    params: { projectId, planId },
    query: {
      phase: PLAN_DETAIL_PHASE_DEPLOY,
      ...(stageId ? { stageId } : {}),
      ...(taskId ? { taskId } : {}),
    },
  };
};

export const buildSpecDetailRouteForCurrentPage = (
  currentRoute: Pick<RouteLocationNormalizedLoaded, "params" | "query">,
  specId: string
): RouteLocationRaw => {
  return {
    name: PROJECT_V1_ROUTE_PLAN_DETAIL_SPEC_DETAIL,
    params: {
      ...(currentRoute.params || {}),
      specId,
    },
    query: currentRoute.query || {},
  };
};

export const buildPlanDeployRouteFromPlanName = (
  planName: string,
  options?: Omit<BuildPlanDeployRouteParams, "projectId" | "planId">
): RouteLocationRaw => {
  return buildPlanDeployRoute({
    projectId: extractProjectResourceName(planName),
    planId: extractPlanUID(planName) || "_",
    ...options,
  });
};

export const buildPlanDeployRouteFromRolloutName = (
  rolloutName: string,
  options?: Omit<BuildPlanDeployRouteParams, "projectId" | "planId">
): RouteLocationRaw => {
  return buildPlanDeployRoute({
    projectId: extractProjectResourceName(rolloutName),
    planId: extractPlanUIDFromRolloutName(rolloutName) || "_",
    ...options,
  });
};

export const buildTaskDetailRoute = (taskName: string): RouteLocationRaw => {
  const stageName = extractStageNameFromTaskName(taskName);

  return buildPlanDeployRoute({
    projectId: extractProjectResourceName(taskName),
    planId: extractPlanUIDFromRolloutName(taskName) || "-",
    stageId: extractStageUID(stageName) || "-",
    taskId: extractTaskUID(taskName) || "-",
  });
};

export const buildStageRoute = (stageName: string): RouteLocationRaw => {
  return buildPlanDeployRoute({
    projectId: extractProjectResourceName(stageName),
    planId: extractPlanUIDFromRolloutName(stageName) || "_",
    stageId: extractStageUID(stageName) || "_",
  });
};
