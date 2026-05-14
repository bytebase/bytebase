import { useMemo } from "react";
import type {
  PlanDetailPageSnapshot,
  PlanDetailPageState,
  PlanDetailPhase,
} from "./types";
import type { useEditingScopes } from "./useEditingScopes";
import type { usePhaseState } from "./usePhaseState";
import type { useSidebarMode } from "./useSidebarMode";

type EditingScopes = ReturnType<typeof useEditingScopes>;
type PhaseState = ReturnType<typeof usePhaseState>;
type Sidebar = ReturnType<typeof useSidebarMode>;

export function useDerivedPlanState(params: {
  snapshot: PlanDetailPageSnapshot;
  isEditing: boolean;
  isRefreshing: boolean;
  isRunningChecks: boolean;
  setIsRunningChecks: (running: boolean) => void;
  lastRefreshTime: number;
  phase: PhaseState;
  editing: EditingScopes;
  sidebar: Sidebar;
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
    lastRefreshTime,
    phase,
    editing,
    sidebar,
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
      lastRefreshTime,
      activePhases: phase.activePhases,
      routeName,
      routePhase,
      routeStageId,
      routeTaskId,
      selectedTaskName,
      pendingLeaveConfirm: editing.pendingLeaveConfirm,
      sidebarMode: sidebar.sidebarMode,
      containerWidth: sidebar.containerWidth,
      desktopSidebarWidth: sidebar.sidebarWidth,
      mobileSidebarOpen: sidebar.isMobileSidebarOpen,
      bypassLeaveGuardOnce: editing.bypassLeaveGuardOnce,
      patchState,
      refreshState,
      setEditing: editing.setEditing,
      setMobileSidebarOpen: sidebar.setMobileSidebarOpen,
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
      lastRefreshTime,
      patchState,
      phase,
      refreshState,
      resolveLeaveConfirm,
      routeName,
      routePhase,
      routeStageId,
      routeTaskId,
      selectedTaskName,
      sidebar,
      snapshot,
    ]
  );
}
