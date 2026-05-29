import type {
  PhaseSlice,
  PlanDetailPhase,
  PlanDetailSliceCreator,
} from "./types";

const defaultActivePhases = (): Set<PlanDetailPhase> =>
  new Set(["changes", "review", "deploy"]);

const isSamePhaseSet = (
  a: Set<PlanDetailPhase>,
  b: Set<PlanDetailPhase>
): boolean => a.size === b.size && Array.from(a).every((phase) => b.has(phase));

export const createPhaseSlice: PlanDetailSliceCreator<PhaseSlice> = (set) => ({
  activePhases: defaultActivePhases(),
  setActivePhases: (phases) =>
    set((state) => {
      const next = new Set(phases);
      if (isSamePhaseSet(state.activePhases, next)) {
        return state;
      }
      return { activePhases: next };
    }),
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
});
