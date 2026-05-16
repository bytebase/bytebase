import type { PlanDetailSliceCreator, PollingSlice } from "./types";

export const createPollingSlice: PlanDetailSliceCreator<PollingSlice> = (
  set
) => ({
  isRefreshing: false,
  isRunningChecks: false,
  pollTimerId: undefined,
  setRefreshing: (v) => set({ isRefreshing: v }),
  setRunningChecks: (v) => set({ isRunningChecks: v }),
  setPollTimerId: (id) => set({ pollTimerId: id }),
});
