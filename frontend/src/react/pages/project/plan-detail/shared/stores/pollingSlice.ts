import type { PlanDetailSliceCreator, PollingSlice } from "./types";

export const createPollingSlice: PlanDetailSliceCreator<PollingSlice> = (
  set
) => ({
  isRefreshing: false,
  isRunningChecks: false,
  lastRefreshTime: 0,
  pollTimerId: undefined,
  setRefreshing: (v) => set({ isRefreshing: v }),
  setRunningChecks: (v) => set({ isRunningChecks: v }),
  setLastRefreshTime: (t) => set({ lastRefreshTime: t }),
  setPollTimerId: (id) => set({ pollTimerId: id }),
});
