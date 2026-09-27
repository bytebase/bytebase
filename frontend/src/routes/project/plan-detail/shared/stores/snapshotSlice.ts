import type {
  PlanDetailPageSnapshot,
  PlanDetailSliceCreator,
  SnapshotSlice,
} from "./types";

const buildDefaultSnapshot = (): PlanDetailPageSnapshot => ({
  taskRuns: [],
  planCheckRuns: [],
  isInitializing: true,
  isNotFound: false,
  isPermissionDenied: false,
});

export const createSnapshotSlice: PlanDetailSliceCreator<SnapshotSlice> = (
  set
) => ({
  snapshot: buildDefaultSnapshot(),
  setSnapshot: (snapshot) => set({ snapshot }),
  patchSnapshot: (patch) =>
    set((state) => ({ snapshot: { ...state.snapshot, ...patch } })),
});
