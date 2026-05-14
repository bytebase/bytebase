import { Code, ConnectError } from "@connectrpc/connect";
import type { MutableRefObject } from "react";
import { useEffect } from "react";
import { router } from "@/router";
import {
  WORKSPACE_ROUTE_403,
  WORKSPACE_ROUTE_404,
} from "@/router/dashboard/workspaceRoutes";
import { unknownPlan } from "@/types/v1/issue/plan";
import type { PlanDetailStoreApi } from "../../shared/stores/usePlanDetailStore";
import { fetchPlanSnapshot } from "./fetchPlanSnapshot";
import type { PlanDetailPageSnapshot } from "./types";

export interface UseInitialFetchParams {
  projectId: string;
  planId: string;
  specId?: string;
  routeQueryRef: MutableRefObject<Record<string, unknown>>;
  storeApi: PlanDetailStoreApi;
  patchState: (patch: Partial<PlanDetailPageSnapshot>) => void;
}

export function useInitialFetch({
  projectId,
  planId,
  specId,
  routeQueryRef,
  storeApi,
  patchState,
}: UseInitialFetchParams): void {
  useEffect(() => {
    storeApi.setState({ editingScopes: {} });
    patchState({
      projectId,
      planId,
      specId,
      pageKey: `${projectId}/${planId}/${specId ?? ""}`,
      projectTitle: "",
      isCreating: planId.toLowerCase() === "create",
      isInitializing: true,
      plan: unknownPlan(),
      issue: undefined,
      rollout: undefined,
      planCheckRuns: [],
      taskRuns: [],
    });

    let canceled = false;

    const load = async () => {
      try {
        const patch = await fetchPlanSnapshot(
          projectId,
          planId,
          routeQueryRef.current
        );
        if (canceled) {
          return;
        }
        patchState({
          ...patch,
          isInitializing: false,
          specId,
        });
      } catch (error) {
        if (canceled) {
          return;
        }
        if (error instanceof ConnectError) {
          if (error.code === Code.NotFound) {
            void router.push({ name: WORKSPACE_ROUTE_404 });
          } else if (error.code === Code.PermissionDenied) {
            void router.push({ name: WORKSPACE_ROUTE_403 });
          }
          patchState({ isInitializing: false });
          return;
        }

        patchState({ isInitializing: false });
        throw error;
      }
    };

    void load();

    return () => {
      canceled = true;
    };
  }, [patchState, planId, projectId, routeQueryRef, specId, storeApi]);
}
