import { useEffect } from "react";
import {
  getRouteQueryString,
  isPlanDetailPhase,
  isPlanDetailResourceRoute,
  PLAN_DETAIL_PHASE_CHANGES,
  PLAN_DETAIL_PHASE_DEPLOY,
  PROJECT_V1_ROUTE_PLAN_DETAIL_SPEC_DETAIL,
  PROJECT_V1_ROUTE_PLAN_DETAIL_SPECS,
  PROJECT_V1_ROUTE_PLAN_ROLLOUT,
  PROJECT_V1_ROUTE_PLAN_ROLLOUT_STAGE,
  PROJECT_V1_ROUTE_PLAN_ROLLOUT_TASK,
} from "@/app/router/handles";
import type { PlanDetailPhase } from "../../shared/stores/types";
import { usePlanDetailStore } from "../../shared/stores/usePlanDetailStore";

export function useRouteSelection(params: {
  routeName?: string;
  routeQuery: Record<string, unknown>;
  specId?: string;
  stageId?: string;
  taskId?: string;
}) {
  const setRouteSelection = usePlanDetailStore((s) => s.setRouteSelection);

  const rawQueryPhase = getRouteQueryString(params.routeQuery.phase as never);
  const queryPhase: PlanDetailPhase | undefined = isPlanDetailPhase(
    rawQueryPhase
  )
    ? rawQueryPhase
    : undefined;
  const isSpecDetailRoute =
    params.routeName === PROJECT_V1_ROUTE_PLAN_DETAIL_SPEC_DETAIL;
  const isSpecsRoute = params.routeName === PROJECT_V1_ROUTE_PLAN_DETAIL_SPECS;
  const isRolloutRoute = params.routeName === PROJECT_V1_ROUTE_PLAN_ROLLOUT;
  const isStageRoute = params.routeName === PROJECT_V1_ROUTE_PLAN_ROLLOUT_STAGE;
  const isTaskRoute = params.routeName === PROJECT_V1_ROUTE_PLAN_ROLLOUT_TASK;

  // A resource path owns the selection completely. Query selectors are a
  // read-only fallback only on the plan route, until canonicalization replaces
  // the old bookmark with its resource path.
  const isResourceRoute = isPlanDetailResourceRoute(params.routeName);
  const querySpecId = isResourceRoute
    ? undefined
    : getRouteQueryString(params.routeQuery.specId as never);
  const queryStageId = isResourceRoute
    ? undefined
    : getRouteQueryString(params.routeQuery.stageId as never);
  const queryTaskId = isResourceRoute
    ? undefined
    : getRouteQueryString(params.routeQuery.taskId as never);

  // The resource this URL selects, in precedence order. Both `phase` and the
  // disclosure identity below derive from this single discriminant, so the
  // spec → task → stage → rollout → plan ladder is written once.
  const kind =
    isSpecsRoute || isSpecDetailRoute || querySpecId
      ? "spec"
      : isTaskRoute || queryTaskId
        ? "task"
        : isStageRoute || queryStageId
          ? "stage"
          : isRolloutRoute
            ? "rollout"
            : "plan";
  // Apply the same precedence to the concrete selection fields. A legacy URL
  // can contain contradictory selectors; while canonicalization replaces it,
  // the store must still expose one coherent resource rather than a spec plus
  // stale stage/task state for an intermediate render.
  const specId =
    kind === "spec"
      ? isSpecDetailRoute
        ? params.specId
        : querySpecId
      : undefined;
  const stageId =
    kind === "stage" || kind === "task"
      ? isStageRoute || isTaskRoute
        ? params.stageId
        : queryStageId
      : undefined;
  const taskId =
    kind === "task" ? (isTaskRoute ? params.taskId : queryTaskId) : undefined;
  const phase =
    kind === "spec"
      ? PLAN_DETAIL_PHASE_CHANGES
      : kind === "plan"
        ? queryPhase
        : PLAN_DETAIL_PHASE_DEPLOY;

  // This semantic identity drives disclosure synchronization. It deliberately
  // excludes secondary state (line, task run, hash) and normalizes legacy query
  // selectors to the same identity as their canonical resource path. A task is
  // plan-unique, so its owning stage is not part of the task identity; correcting
  // a stale stage path must not create a second disclosure transition.
  const selectionKey =
    kind === "spec"
      ? `spec:${specId ?? ""}`
      : kind === "task"
        ? `task:${taskId ?? ""}`
        : kind === "stage"
          ? `stage:${stageId ?? ""}`
          : kind === "rollout"
            ? "rollout"
            : `plan:${phase ?? ""}`;

  useEffect(() => {
    setRouteSelection({
      phase,
      stageId,
      taskName: taskId,
      specId,
    });
  }, [setRouteSelection, phase, specId, stageId, taskId]);

  return { phase, selectionKey, specId, stageId, taskId };
}
