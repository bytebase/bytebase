import { ref } from "vue";

export type PhaseType = "changes" | "review" | "deploy";

const DEFAULT_EXPANDED_PHASES: PhaseType[] = ["changes", "review", "deploy"];

export const useActivePhase = () => {
  const manualExpanded = ref(new Set<PhaseType>(DEFAULT_EXPANDED_PHASES));
  const isExpanded = (phase: PhaseType): boolean => {
    return manualExpanded.value.has(phase);
  };

  const togglePhase = (phase: PhaseType) => {
    const next = new Set(manualExpanded.value);
    if (next.has(phase)) {
      next.delete(phase);
    } else {
      next.add(phase);
    }
    manualExpanded.value = next;
  };

  // For programmatic expansion (e.g., route query ?section=deploy)
  const expandPhase = (phase: PhaseType) => {
    const next = new Set(manualExpanded.value);
    next.add(phase);
    manualExpanded.value = next;
  };

  const syncExpandedPhases = (phases: PhaseType[]) => {
    if (phases.length === 0) return;

    const next = new Set(manualExpanded.value);
    for (const phase of phases) {
      next.add(phase);
    }
    manualExpanded.value = next;
  };

  return {
    isExpanded,
    togglePhase,
    expandPhase,
    syncExpandedPhases,
  };
};
