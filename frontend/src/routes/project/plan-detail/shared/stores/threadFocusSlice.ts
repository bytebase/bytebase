import type { PlanDetailSliceCreator, ThreadFocusSlice } from "./types";

// "View in Statement" hands the target thread to the statement editor through
// the store: the timeline and the editor live in different phase sections and
// the editor may still be mounting when the request is made.
export const createThreadFocusSlice: PlanDetailSliceCreator<
  ThreadFocusSlice
> = (set) => ({
  threadFocus: undefined,
  requestThreadFocus: (request) =>
    set((state) => ({
      threadFocus: {
        ...request,
        nonce: (state.threadFocus?.nonce ?? 0) + 1,
      },
    })),
  clearThreadFocus: (nonce) =>
    set((state) =>
      state.threadFocus && state.threadFocus.nonce === nonce
        ? { threadFocus: undefined }
        : state
    ),
});
