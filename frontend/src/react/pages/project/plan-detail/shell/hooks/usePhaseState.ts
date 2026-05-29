import { useCallback } from "react";
import type { PlanDetailPhase } from "../../shared/stores/types";
import { usePlanDetailStore } from "../../shared/stores/usePlanDetailStore";

export function usePhaseState() {
  const activePhases = usePlanDetailStore((s) => s.activePhases);
  const setActivePhases = usePlanDetailStore((s) => s.setActivePhases);
  const togglePhase = usePlanDetailStore((s) => s.togglePhase);
  const expandPhase = usePlanDetailStore((s) => s.expandPhase);

  const isActive = useCallback(
    (phase: PlanDetailPhase) => activePhases.has(phase),
    [activePhases]
  );

  return {
    activePhases,
    isActive,
    setActivePhases,
    togglePhase,
    expandPhase,
  };
}
