import type { PlanDetailSliceCreator, SelectionSlice } from "./types";

export const createSelectionSlice: PlanDetailSliceCreator<SelectionSlice> = (
  set
) => ({
  routePhase: undefined,
  selectedSpecId: undefined,
  selectedStageId: undefined,
  selectedTaskName: undefined,
  setRouteSelection: (selection) =>
    set({
      routePhase: selection.phase,
      selectedSpecId: selection.specId,
      selectedStageId: selection.stageId,
      selectedTaskName: selection.taskName,
    }),
});
