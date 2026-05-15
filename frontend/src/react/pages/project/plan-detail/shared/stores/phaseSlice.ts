import type {
  PhaseSlice,
  PlanDetailPhase,
  PlanDetailSliceCreator,
} from "./types";

const defaultActivePhases = (): Set<PlanDetailPhase> =>
  new Set(["changes", "review", "deploy"]);

export const createPhaseSlice: PlanDetailSliceCreator<PhaseSlice> = (set) => ({
  activePhases: defaultActivePhases(),
  togglePhase: (phase) =>
    set((state) => {
      const next = new Set(state.activePhases);
      if (next.has(phase)) next.delete(phase);
      else next.add(phase);
      return { activePhases: next };
    }),
  expandPhase: (phase) =>
    set((state) => {
      if (state.activePhases.has(phase)) return state;
      const next = new Set(state.activePhases);
      next.add(phase);
      return { activePhases: next };
    }),
  focusPhase: (phase) =>
    set((state) => {
      if (state.activePhases.size === 1 && state.activePhases.has(phase)) {
        return state;
      }
      return { activePhases: new Set([phase]) };
    }),
  collapsePhase: (phase) =>
    set((state) => {
      if (!state.activePhases.has(phase)) return state;
      const next = new Set(state.activePhases);
      next.delete(phase);
      return { activePhases: next };
    }),
});
