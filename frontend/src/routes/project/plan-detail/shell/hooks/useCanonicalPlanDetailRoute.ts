import { useEffect } from "react";
import { type RouteTarget, router } from "@/app/router";
import {
  getRouteQueryString,
  isPlanDetailPhase,
  isPlanDetailResourceRoute,
  PROJECT_V1_ROUTE_PLAN_DETAIL,
  PROJECT_V1_ROUTE_PLAN_DETAIL_SPEC_DETAIL,
  PROJECT_V1_ROUTE_PLAN_DETAIL_SPECS,
  PROJECT_V1_ROUTE_PLAN_ROLLOUT,
  PROJECT_V1_ROUTE_PLAN_ROLLOUT_STAGE,
  PROJECT_V1_ROUTE_PLAN_ROLLOUT_TASK,
} from "@/app/router/handles";
import {
  PLAN_DETAIL_SELECTION_QUERY_KEYS,
  stripPlanDetailSelectionQuery,
} from "@/app/router/planDetailRouteQuery";
import {
  buildPlanDeployRoute,
  buildPlanRolloutRoute,
} from "@/app/router/routeHelpers";
import { extractStageUID, extractTaskUID } from "@/utils/v1/issue/rollout";
import { isRolloutExpected } from "../../utils/phaseSummary";
import type { PlanDetailPageSnapshot } from "./types";

export function useCanonicalPlanDetailRoute(params: {
  projectId: string;
  planId: string;
  routeHash?: string;
  routeName?: string;
  routeQuery: Record<string, unknown>;
  specId?: string;
  stageId?: string;
  taskId?: string;
  snapshot: PlanDetailPageSnapshot;
  isEditing: boolean;
  bypassLeaveGuardOnce: () => void;
}) {
  useEffect(() => {
    const pageKey = `${params.projectId}/${params.planId}`;
    if (
      params.snapshot.pageKey !== pageKey ||
      params.snapshot.isCreating ||
      params.planId.toLowerCase() === "create"
    ) {
      return;
    }
    const replace = (target: RouteTarget) => {
      if (params.isEditing) params.bypassLeaveGuardOnce();
      void router.replace(
        typeof target === "string" || !params.routeHash
          ? target
          : { ...target, hash: params.routeHash },
        { preventScrollReset: true }
      );
    };
    const base = { projectId: params.projectId, planId: params.planId };
    const cleanQuery = stripPlanDetailSelectionQuery(params.routeQuery);
    const hasSelectionQuery = Object.keys(params.routeQuery).some((key) =>
      PLAN_DETAIL_SELECTION_QUERY_KEYS.has(key)
    );

    // Resource paths own the selection. Remove contradictory or legacy
    // selection queries without adding a history entry.
    if (isPlanDetailResourceRoute(params.routeName) && hasSelectionQuery) {
      const target: RouteTarget =
        params.routeName === PROJECT_V1_ROUTE_PLAN_DETAIL_SPECS
          ? {
              name: PROJECT_V1_ROUTE_PLAN_DETAIL_SPECS,
              params: base,
              query: cleanQuery,
            }
          : params.routeName === PROJECT_V1_ROUTE_PLAN_DETAIL_SPEC_DETAIL
            ? {
                name: PROJECT_V1_ROUTE_PLAN_DETAIL_SPEC_DETAIL,
                params: { ...base, specId: params.specId },
                query: cleanQuery,
              }
            : params.routeName === PROJECT_V1_ROUTE_PLAN_ROLLOUT_TASK
              ? {
                  name: PROJECT_V1_ROUTE_PLAN_ROLLOUT_TASK,
                  params: {
                    ...base,
                    stageId: params.stageId,
                    taskId: params.taskId,
                  },
                  query: cleanQuery,
                }
              : params.routeName === PROJECT_V1_ROUTE_PLAN_ROLLOUT_STAGE
                ? {
                    name: PROJECT_V1_ROUTE_PLAN_ROLLOUT_STAGE,
                    params: { ...base, stageId: params.stageId },
                    query: cleanQuery,
                  }
                : {
                    name: PROJECT_V1_ROUTE_PLAN_ROLLOUT,
                    params: base,
                    query: cleanQuery,
                  };
      replace(target);
      return;
    }

    // Query-only resource selectors are old bookmarks. Wait for the snapshot,
    // validate once, then replace directly with the final canonical resource.
    // Eagerly trusting a stale stage would produce an intermediate wrong path
    // followed by a second correction after the rollout loads.
    if (params.routeName === PROJECT_V1_ROUTE_PLAN_DETAIL) {
      const querySpecId = getRouteQueryString(
        params.routeQuery.specId as never
      );
      const queryStageId = getRouteQueryString(
        params.routeQuery.stageId as never
      );
      const queryTaskId = getRouteQueryString(
        params.routeQuery.taskId as never
      );
      if (querySpecId || queryStageId || queryTaskId) {
        if (!params.snapshot.ready) return;

        if (querySpecId) {
          const specExists = params.snapshot.plan.specs.some(
            (spec) => spec.id === querySpecId
          );
          replace({
            name: specExists
              ? PROJECT_V1_ROUTE_PLAN_DETAIL_SPEC_DETAIL
              : PROJECT_V1_ROUTE_PLAN_DETAIL,
            params: specExists ? { ...base, specId: querySpecId } : base,
            query: cleanQuery,
          });
          return;
        }

        const rollout = params.snapshot.rollout;
        if (!rollout) {
          if (!isRolloutExpected(params.snapshot)) {
            replace({
              name: PROJECT_V1_ROUTE_PLAN_DETAIL,
              params: base,
              query: cleanQuery,
            });
          }
          return;
        }

        const queryStage = queryStageId
          ? rollout.stages.find(
              (stage) => extractStageUID(stage.name) === queryStageId
            )
          : undefined;
        if (queryTaskId) {
          const taskOwner = rollout.stages.find((stage) =>
            stage.tasks.some(
              (task) => extractTaskUID(task.name) === queryTaskId
            )
          );
          if (taskOwner) {
            replace({
              name: PROJECT_V1_ROUTE_PLAN_ROLLOUT_TASK,
              params: {
                ...base,
                stageId: extractStageUID(taskOwner.name),
                taskId: queryTaskId,
              },
              query: cleanQuery,
            });
            return;
          }
        }

        if (queryStage) {
          replace({
            name: PROJECT_V1_ROUTE_PLAN_ROLLOUT_STAGE,
            params: { ...base, stageId: queryStageId },
            query: cleanQuery,
          });
          return;
        }

        replace({
          name: PROJECT_V1_ROUTE_PLAN_ROLLOUT,
          params: base,
          query: cleanQuery,
        });
        return;
      }
      const phase = params.routeQuery.phase;
      if (phase !== undefined && !isPlanDetailPhase(phase)) {
        replace({
          name: PROJECT_V1_ROUTE_PLAN_DETAIL,
          params: base,
          query: cleanQuery,
        });
        return;
      }
    }

    if (!params.snapshot.ready) return;

    if (
      params.routeName === PROJECT_V1_ROUTE_PLAN_DETAIL_SPEC_DETAIL &&
      params.specId &&
      !params.snapshot.plan.specs.some((spec) => spec.id === params.specId)
    ) {
      replace({
        name: PROJECT_V1_ROUTE_PLAN_DETAIL,
        params: base,
        query: cleanQuery,
      });
      return;
    }

    if (
      (params.routeName === PROJECT_V1_ROUTE_PLAN_ROLLOUT ||
        params.routeName === PROJECT_V1_ROUTE_PLAN_ROLLOUT_STAGE ||
        params.routeName === PROJECT_V1_ROUTE_PLAN_ROLLOUT_TASK) &&
      !params.snapshot.rollout
    ) {
      if (!isRolloutExpected(params.snapshot)) {
        replace({
          name: PROJECT_V1_ROUTE_PLAN_DETAIL,
          params: base,
          query: cleanQuery,
        });
      }
      return;
    }

    const rollout = params.snapshot.rollout;
    if (!rollout) return;
    const stage = params.stageId
      ? rollout.stages.find(
          (item) => extractStageUID(item.name) === params.stageId
        )
      : undefined;

    if (params.routeName === PROJECT_V1_ROUTE_PLAN_ROLLOUT_STAGE && !stage) {
      replace(buildPlanRolloutRoute(params.projectId, params.planId));
      return;
    }

    if (params.routeName === PROJECT_V1_ROUTE_PLAN_ROLLOUT_TASK) {
      const taskOwner = params.taskId
        ? rollout.stages.find((item) =>
            item.tasks.some(
              (task) => extractTaskUID(task.name) === params.taskId
            )
          )
        : undefined;
      if (!taskOwner) {
        replace(
          stage
            ? buildPlanDeployRoute({
                ...base,
                stageId: extractStageUID(stage.name),
              })
            : buildPlanRolloutRoute(params.projectId, params.planId)
        );
        return;
      }
      const canonicalStageId = extractStageUID(taskOwner.name);
      if (canonicalStageId !== params.stageId) {
        replace({
          name: PROJECT_V1_ROUTE_PLAN_ROLLOUT_TASK,
          params: {
            ...base,
            stageId: canonicalStageId,
            taskId: params.taskId,
          },
          query: cleanQuery,
        });
      }
    }
  }, [
    params.bypassLeaveGuardOnce,
    params.isEditing,
    params.planId,
    params.projectId,
    params.routeHash,
    params.routeName,
    params.routeQuery,
    params.snapshot.isCreating,
    params.snapshot.issue,
    params.snapshot.pageKey,
    params.snapshot.plan,
    params.snapshot.ready,
    params.snapshot.rollout,
    params.specId,
    params.stageId,
    params.taskId,
  ]);
}
