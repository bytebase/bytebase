// Maps a Plan status reason to its tone. Kept pure and separate from the
// component so the mapping is unit-testable; the label text itself is rendered
// with literal t() calls in PlanStatusAction so the i18n usage scanner sees it.
import type { PlanStatusReason } from "./planLifecycleHeaderState";

export type PlanStatusTone = "neutral" | "error";

export function getPlanStatusTone(reason: PlanStatusReason): PlanStatusTone {
  switch (reason) {
    case "checking":
    case "in-review":
      return "neutral";
    case "rejected":
    case "checks-failing":
      return "error";
  }
}
