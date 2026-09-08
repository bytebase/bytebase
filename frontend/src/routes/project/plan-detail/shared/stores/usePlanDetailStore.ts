import { createContext, useContext } from "react";
import { create, useStore } from "zustand";
import { createEditingSlice } from "./editingSlice";
import { createPhaseSlice } from "./phaseSlice";
import {
  createPlacementSlice,
  defaultPlacementDeps,
  type PlacementDeps,
} from "./placementSlice";
import { createPollingSlice } from "./pollingSlice";
import { createSelectionSlice } from "./selectionSlice";
import { createSnapshotSlice } from "./snapshotSlice";
import { createThreadFocusSlice } from "./threadFocusSlice";
import type { PlanDetailStore } from "./types";

export const createPlanDetailStore = (
  placement: PlacementDeps = defaultPlacementDeps()
) =>
  create<PlanDetailStore>()((...args) => ({
    ...createSnapshotSlice(...args),
    ...createPhaseSlice(...args),
    ...createEditingSlice(...args),
    ...createSelectionSlice(...args),
    ...createPollingSlice(...args),
    ...createThreadFocusSlice(...args),
    ...createPlacementSlice(placement)(...args),
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
