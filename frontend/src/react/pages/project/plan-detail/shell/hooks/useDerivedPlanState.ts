import { useMemo } from "react";
import type {
  PlanDetailPageSnapshot,
  PlanDetailPageState,
  PlanDetailPhase,
} from "./types";
import type { useEditingScopes } from "./useEditingScopes";
import type { useLayoutMode } from "./useLayoutMode";
import type { usePhaseState } from "./usePhaseState";

type EditingScopes = ReturnType<typeof useEditingScopes>;
type Layout = ReturnType<typeof useLayoutMode>;
type PhaseState = ReturnType<typeof usePhaseState>;

export function useDerivedPlanState(params: {
  snapshot: PlanDetailPageSnapshot;
  isEditing: boolean;
  isRefreshing: boolean;
  isRunningChecks: boolean;
  setIsRunningChecks: (running: boolean) => void;
  phase: PhaseState;
  editing: EditingScopes;
  layout: Layout;
  routeName?: string;
  routePhase?: PlanDetailPhase;
  routeStageId?: string;
  routeTaskId?: string;
  patchState: (patch: Partial<PlanDetailPageSnapshot>) => void;
  refreshState: () => Promise<void>;
  closeTaskPanel: () => void;
  resolveLeaveConfirm: (confirmed: boolean) => void;
}): PlanDetailPageState {
  const {
    snapshot,
    isEditing,
    isRefreshing,
    isRunningChecks,
    setIsRunningChecks,
    phase,
    editing,
    layout,
    routeName,
    routePhase,
    routeStageId,
    routeTaskId,
    patchState,
    refreshState,
    closeTaskPanel,
    resolveLeaveConfirm,
  } = params;

  const selectedTaskName = useMemo(() => {
    if (!routeTaskId || !snapshot.rollout) {
      return undefined;
    }
    for (const stage of snapshot.rollout.stages) {
      const task = stage.tasks.find((item) =>
        item.name.endsWith(`/${routeTaskId}`)
      );
      if (task) {
        return task.name;
      }
    }
    return undefined;
  }, [routeTaskId, snapshot.rollout]);

  return useMemo(
    () => ({
      ...snapshot,
      isEditing,
      isRefreshing,
      isRunningChecks,
      setIsRunningChecks,
      activePhases: phase.activePhases,
      routeName,
      routePhase,
      routeStageId,
      routeTaskId,
      selectedTaskName,
      pendingLeaveConfirm: editing.pendingLeaveConfirm,
      layoutMode: layout.layoutMode,
      containerWidth: layout.containerWidth,
      bypassLeaveGuardOnce: editing.bypassLeaveGuardOnce,
      patchState,
      refreshState,
      setEditing: editing.setEditing,
      togglePhase: phase.togglePhase,
      expandPhase: phase.expandPhase,
      closeTaskPanel,
      resolveLeaveConfirm,
    }),
    [
      closeTaskPanel,
      editing,
      isEditing,
      isRefreshing,
      isRunningChecks,
      patchState,
      phase,
      refreshState,
      resolveLeaveConfirm,
      routeName,
      routePhase,
      routeStageId,
      routeTaskId,
      layout,
      selectedTaskName,
      snapshot,
    ]
  );
}
