import { useEffect } from "react";
import { getRouteQueryString } from "@/router/dashboard/projectV1RouteHelpers";
import type { PlanDetailPhase } from "../../shared/stores/types";
import { usePlanDetailStore } from "../../shared/stores/usePlanDetailStore";

export function useRouteSelection(params: {
  routeQuery: Record<string, unknown>;
  specId?: string;
}) {
  const setRouteSelection = usePlanDetailStore((s) => s.setRouteSelection);

  const phase = getRouteQueryString(params.routeQuery.phase as never) as
    | PlanDetailPhase
    | undefined;
  const stageId = getRouteQueryString(params.routeQuery.stageId as never);
  const taskId = getRouteQueryString(params.routeQuery.taskId as never);

  useEffect(() => {
    setRouteSelection({
      phase,
      stageId,
      taskName: taskId,
      specId: params.specId,
    });
  }, [setRouteSelection, phase, stageId, taskId, params.specId]);

  return { phase, stageId, taskId };
}
