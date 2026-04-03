import { computed, type Ref, ref, watch } from "vue";
import type { Issue } from "@/types/proto-es/v1/issue_service_pb";
import type { Rollout } from "@/types/proto-es/v1/rollout_service_pb";

export type PhaseType = "changes" | "review" | "deploy";

export const useActivePhase = (
  issue: Ref<Issue | undefined>,
  rollout: Ref<Rollout | undefined>
) => {
  // The lifecycle-determined phase (always expanded by default)
  const currentPhase = computed<PhaseType>(() => {
    if (rollout.value) return "deploy";
    if (issue.value) return "review";
    return "changes";
  });

  // Set of expanded phases. The current lifecycle phase is always included.
  const manualExpanded = ref(new Set<PhaseType>());

  // When lifecycle transitions, reset manual expansions
  watch(currentPhase, () => {
    manualExpanded.value = new Set<PhaseType>();
  });

  const isExpanded = (phase: PhaseType): boolean => {
    if (phase === currentPhase.value) return true;
    return manualExpanded.value.has(phase);
  };

  const togglePhase = (phase: PhaseType) => {
    // The current lifecycle phase cannot be collapsed
    if (phase === currentPhase.value) return;

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
    if (phase === currentPhase.value) return;
    const next = new Set(manualExpanded.value);
    next.add(phase);
    manualExpanded.value = next;
  };

  const syncExpandedPhases = (phases: PhaseType[]) => {
    manualExpanded.value = new Set(
      phases.filter((phase) => phase !== currentPhase.value)
    );
  };

  return {
    currentPhase,
    isExpanded,
    togglePhase,
    expandPhase,
    syncExpandedPhases,
  };
};
