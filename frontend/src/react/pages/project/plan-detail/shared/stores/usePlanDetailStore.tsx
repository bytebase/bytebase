import { createContext, type ReactNode, useContext, useState } from "react";
import { create, useStore } from "zustand";
import { createEditingSlice } from "./editingSlice";
import { createPhaseSlice } from "./phaseSlice";
import { createPollingSlice } from "./pollingSlice";
import { createSelectionSlice } from "./selectionSlice";
import { createSnapshotSlice } from "./snapshotSlice";
import type { PlanDetailStore } from "./types";

const createPlanDetailStore = () =>
  create<PlanDetailStore>()((...args) => ({
    ...createSnapshotSlice(...args),
    ...createPhaseSlice(...args),
    ...createEditingSlice(...args),
    ...createSelectionSlice(...args),
    ...createPollingSlice(...args),
  }));

type PlanDetailStoreApi = ReturnType<typeof createPlanDetailStore>;

const PlanDetailStoreContext = createContext<PlanDetailStoreApi | null>(null);

export const PlanDetailStoreProvider = ({
  children,
}: {
  children: ReactNode;
}) => {
  const [store] = useState(createPlanDetailStore);
  return (
    <PlanDetailStoreContext.Provider value={store}>
      {children}
    </PlanDetailStoreContext.Provider>
  );
};

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
