import { createContext, useContext } from "react";
import { create, useStore } from "zustand";
import { createEditingSlice } from "./editingSlice";
import { createPhaseSlice } from "./phaseSlice";
import { createPollingSlice } from "./pollingSlice";
import { createSelectionSlice } from "./selectionSlice";
import { createSnapshotSlice } from "./snapshotSlice";
import type { PlanDetailStore } from "./types";

export const createPlanDetailStore = () =>
  create<PlanDetailStore>()((...args) => ({
    ...createSnapshotSlice(...args),
    ...createPhaseSlice(...args),
    ...createEditingSlice(...args),
    ...createSelectionSlice(...args),
    ...createPollingSlice(...args),
  }));

export type PlanDetailStoreApi = ReturnType<typeof createPlanDetailStore>;

export const PlanDetailStoreContext = createContext<PlanDetailStoreApi | null>(
  null
);

export function usePlanDetailStore<T>(selector: (s: PlanDetailStore) => T): T {
  const store = useContext(PlanDetailStoreContext);
  if (!store) throw new Error("PlanDetailStoreProvider missing");
  return useStore(store, selector);
}

export function usePlanDetailStoreApi(): PlanDetailStoreApi {
  const store = useContext(PlanDetailStoreContext);
  if (!store) throw new Error("PlanDetailStoreProvider missing");
  return store;
}
