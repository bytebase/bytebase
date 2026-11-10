import type { PlanDetailSliceCreator, SelectionSlice } from "./types";

export const createSelectionSlice: PlanDetailSliceCreator<SelectionSlice> = (
  set
) => ({
  routePhase: undefined,
  selectedStageId: undefined,
  selectedTaskName: undefined,
  setRouteSelection: (selection) =>
    set({
      routePhase: selection.phase,
      selectedStageId: selection.stageId,
      selectedTaskName: selection.taskName,
    }),
  displayedSpecId: undefined,
  setDisplayedSpecId: (displayedSpecId) => set({ displayedSpecId }),
});
