import type { EditingSlice, PlanDetailSliceCreator } from "./types";

export const createEditingSlice: PlanDetailSliceCreator<EditingSlice> = (
  set
) => {
  let bypassOnce = false;
  return {
    editingScopes: {},
    setEditing: (scope, editing) =>
      set((state) => {
        const next = { ...state.editingScopes };
        if (editing) next[scope] = true;
        else delete next[scope];
        return { editingScopes: next };
      }),
    bypassLeaveGuardOnce: () => {
      bypassOnce = true;
    },
    isLeaveGuardBypassed: () => {
      if (bypassOnce) {
        bypassOnce = false;
        return true;
      }
      return false;
    },
    pendingLeaveTarget: null,
    setPendingLeaveTarget: (target) => set({ pendingLeaveTarget: target }),
    pendingLeaveConfirm: false,
    setPendingLeaveConfirm: (open) => set({ pendingLeaveConfirm: open }),
  };
};
