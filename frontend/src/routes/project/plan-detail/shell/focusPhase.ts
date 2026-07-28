import type { PlanDetailPhase } from "../shared/stores/types";

export const planPhaseAnchorId = (phase: PlanDetailPhase): string =>
  `plan-phase-${phase}`;

// Expand a plan-detail phase and bring it into view. Header actions call this so
// the user lands on the section their action affects (e.g. deploy after a bypass
// or run). The phase section's anchor element is always mounted — even collapsed
// or "future" — so the scroll can run right after the expand; `scroll-mt` on the
// anchor keeps it clear of the sticky header.
export function focusPlanPhase(
  phase: PlanDetailPhase,
  expandPhase: (phase: PlanDetailPhase) => void
): void {
  expandPhase(phase);
  requestAnimationFrame(() => {
    document
      .getElementById(planPhaseAnchorId(phase))
      ?.scrollIntoView({ behavior: "smooth", block: "start" });
  });
}
